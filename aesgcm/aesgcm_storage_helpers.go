/**
 * GoLevelDB Encrypted Storage
 *
 *    Copyright 2019 Tenta, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * For any questions, please contact developer@tenta.io
 *
 * aesgcm_storage_file_test.go: Version of file_storage_test.go for encrypted storage
 *
 * This file contains mostly code originally from
 * https://github.com/syndtr/goleveldb/blob/master/leveldb/storage
 * licensed under a BSD license which bears the following copyright
 *
 * "Copyright (c) 2012, Suryandaru Triandana <syndtr@gmail.com>
 * All rights reservefs."
 *
 * "Use of this source code is governed by a BSD-style license that can be
 * found in the LICENSE file."
 *
 */

package aesgcm

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type fileLock interface {
	release() error
}

type aesgcmStorageLock struct {
	fs *aesgcmStorage
}

func (lock *aesgcmStorageLock) Unlock() {
	if lock.fs != nil {
		lock.fs.mu.Lock()
		defer lock.fs.mu.Unlock()
		if lock.fs.slock == lock {
			lock.fs.slock = nil
		}
	}
}

type int64Slice []int64

func (p int64Slice) Len() int           { return len(p) }
func (p int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (fs *aesgcmStorage) setMeta(fd storage.FileDesc) error {
	writeFileSynced := func(filename string, data []byte, perm os.FileMode) error {
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
		if err != nil {
			return err
		}
		n, err := f.Write(data)
		if err == nil && n < len(data) {
			err = io.ErrShortWrite
		}
		if err1 := f.Sync(); err == nil {
			err = err1
		}
		if err1 := f.Close(); err == nil {
			err = err1
		}
		return err
	}

	content := fsGenName(fd) + "\n"
	// Check and backup old CURRENT file.
	currentPath := filepath.Join(fs.path, "CURRENT")
	if _, err := os.Stat(currentPath); err == nil {
		b, err := ioutil.ReadFile(currentPath)
		if err != nil {
			fs.Log(fmt.Sprintf("backup CURRENT: %v", err))
			return err
		}
		if string(b) == content {
			// Content not changed, do nothing.
			return nil
		}
		if err := writeFileSynced(currentPath+".bak", b, 0644); err != nil {
			fs.Log(fmt.Sprintf("backup CURRENT: %v", err))
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	path := fmt.Sprintf("%s.%d", filepath.Join(fs.path, "CURRENT"), fd.Num)
	if err := writeFileSynced(path, []byte(content), 0644); err != nil {
		fs.Log(fmt.Sprintf("create CURRENT.%d: %v", fd.Num, err))
		return err
	}
	// Replace CURRENT file.
	if err := rename(path, currentPath); err != nil {
		fs.Log(fmt.Sprintf("rename CURRENT.%d: %v", fd.Num, err))
		return err
	}
	// Sync root directory.
	if err := syncDir(fs.path); err != nil {
		fs.Log(fmt.Sprintf("syncDir: %v", err))
		return err
	}
	return nil
}

func (fs *aesgcmStorage) SetMeta(fd storage.FileDesc) error {
	if !storage.FileDescOk(fd) {
		return storage.ErrInvalidFile
	}
	if fs.readOnly {
		return errReadOnly
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.open < 0 {
		return storage.ErrClosed
	}
	return fs.setMeta(fd)
}

func isCorrupted(err error) bool {
	switch err.(type) {
	case *storage.ErrCorrupted:
		return true
	}
	return false
}

func (fs *aesgcmStorage) GetMeta() (storage.FileDesc, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.open < 0 {
		return storage.FileDesc{}, storage.ErrClosed
	}
	dir, err := os.Open(fs.path)
	if err != nil {
		return storage.FileDesc{}, err
	}
	names, err := dir.Readdirnames(0)
	// Close the dir first before checking for Readdirnames error.
	if ce := dir.Close(); ce != nil {
		fs.Log(fmt.Sprintf("close dir: %v", ce))
	}
	if err != nil {
		return storage.FileDesc{}, err
	}
	// Try this in order:
	// - CURRENT.[0-9]+ ('pending rename' file, descending order)
	// - CURRENT
	// - CURRENT.bak
	//
	// Skip corrupted file or file that point to a missing target file.
	type currentFile struct {
		name string
		fd   storage.FileDesc
	}
	tryCurrent := func(name string) (*currentFile, error) {
		b, err := ioutil.ReadFile(filepath.Join(fs.path, name))
		if err != nil {
			if os.IsNotExist(err) {
				err = os.ErrNotExist
			}
			return nil, err
		}
		var fd storage.FileDesc
		if len(b) < 1 || b[len(b)-1] != '\n' || !fsParseNamePtr(string(b[:len(b)-1]), &fd) {
			fs.Log(fmt.Sprintf("%s: corrupted content: %q", name, b))
			err := &storage.ErrCorrupted{Err: errCorruptedCurrent}
			return nil, err
		}
		if _, err := os.Stat(filepath.Join(fs.path, fsGenName(fd))); err != nil {
			if os.IsNotExist(err) {
				fs.Log(fmt.Sprintf("%s: missing target file: %s", name, fd))
				err = os.ErrNotExist
			}
			return nil, err
		}
		return &currentFile{name: name, fd: fd}, nil
	}
	tryCurrents := func(names []string) (*currentFile, error) {
		var (
			cur *currentFile
			// Last corruption error.
			lastCerr error
		)
		for _, name := range names {
			var err error
			cur, err = tryCurrent(name)
			if err == nil {
				break
			} else if err == os.ErrNotExist {
				// Fallback to the next file.
			} else if isCorrupted(err) {
				lastCerr = err
				// Fallback to the next file.
			} else {
				// In case the error is due to permission, etc.
				return nil, err
			}
		}
		if cur == nil {
			err := os.ErrNotExist
			if lastCerr != nil {
				err = lastCerr
			}
			return nil, err
		}
		return cur, nil
	}

	// Try 'pending rename' files.
	var nums []int64
	for _, name := range names {
		if strings.HasPrefix(name, "CURRENT.") && name != "CURRENT.bak" {
			i, err := strconv.ParseInt(name[8:], 10, 64)
			if err == nil {
				nums = append(nums, i)
			}
		}
	}
	var (
		pendCur   *currentFile
		pendErr   = os.ErrNotExist
		pendNames []string
	)
	if len(nums) > 0 {
		sort.Sort(sort.Reverse(int64Slice(nums)))
		pendNames = make([]string, len(nums))
		for i, num := range nums {
			pendNames[i] = fmt.Sprintf("CURRENT.%d", num)
		}
		pendCur, pendErr = tryCurrents(pendNames)
		if pendErr != nil && pendErr != os.ErrNotExist && !isCorrupted(pendErr) {
			return storage.FileDesc{}, pendErr
		}
	}

	// Try CURRENT and CURRENT.bak.
	curCur, curErr := tryCurrents([]string{"CURRENT", "CURRENT.bak"})
	if curErr != nil && curErr != os.ErrNotExist && !isCorrupted(curErr) {
		return storage.FileDesc{}, curErr
	}

	// pendCur takes precedence, but guards against obsolete pendCur.
	if pendCur != nil && (curCur == nil || pendCur.fd.Num > curCur.fd.Num) {
		curCur = pendCur
	}

	if curCur != nil {
		// Restore CURRENT file to proper state.
		if !fs.readOnly && (curCur.name != "CURRENT" || len(pendNames) != 0) {
			// Ignore setMeta errors, however don't delete obsolete files if we
			// catch error.
			if err := fs.setMeta(curCur.fd); err == nil {
				// Remove 'pending rename' files.
				for _, name := range pendNames {
					if err := os.Remove(filepath.Join(fs.path, name)); err != nil {
						fs.Log(fmt.Sprintf("remove %s: %v", name, err))
					}
				}
			}
		}
		return curCur.fd, nil
	}

	// Nothing found.
	if isCorrupted(pendErr) {
		return storage.FileDesc{}, pendErr
	}
	return storage.FileDesc{}, curErr
}

func (fs *aesgcmStorage) List(ft storage.FileType) (fds []storage.FileDesc, err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.open < 0 {
		return nil, storage.ErrClosed
	}
	dir, err := os.Open(fs.path)
	if err != nil {
		return
	}
	names, err := dir.Readdirnames(0)
	// Close the dir first before checking for Readdirnames error.
	if cerr := dir.Close(); cerr != nil {
		fs.Log(fmt.Sprintf("close dir: %v", cerr))
	}
	if err == nil {
		for _, name := range names {
			if fd, ok := fsParseName(name); ok && fd.Type&ft != 0 {
				fds = append(fds, fd)
			}
		}
	}
	return
}

func fsGenName(fd storage.FileDesc) string {
	switch fd.Type {
	case storage.TypeManifest:
		return fmt.Sprintf("MANIFEST-%06d", fd.Num)
	case storage.TypeJournal:
		return fmt.Sprintf("%06d.log", fd.Num)
	case storage.TypeTable:
		return fmt.Sprintf("%06d.ldb", fd.Num)
	case storage.TypeTemp:
		return fmt.Sprintf("%06d.tmp", fd.Num)
	default:
		panic("invalid file type")
	}
}

func fsParseName(name string) (fd storage.FileDesc, ok bool) {
	var tail string
	_, err := fmt.Sscanf(name, "%d.%s", &fd.Num, &tail)
	if err == nil {
		switch tail {
		case "log":
			fd.Type = storage.TypeJournal
		case "ldb":
			fd.Type = storage.TypeTable
		case "tmp":
			fd.Type = storage.TypeTemp
		default:
			return
		}
		return fd, true
	}
	n, _ := fmt.Sscanf(name, "MANIFEST-%d%s", &fd.Num, &tail)
	if n == 1 {
		fd.Type = storage.TypeManifest
		return fd, true
	}
	return
}

func fsParseNamePtr(name string, fd *storage.FileDesc) bool {
	_fd, ok := fsParseName(name)
	if fd != nil {
		*fd = _fd
	}
	return ok
}

func (fs *aesgcmStorage) Lock() (storage.Locker, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.open < 0 {
		return nil, storage.ErrClosed
	}
	if fs.readOnly {
		return &aesgcmStorageLock{}, nil
	}
	if fs.slock != nil {
		return nil, storage.ErrLocked
	}
	fs.slock = &aesgcmStorageLock{fs: fs}
	return fs.slock, nil
}

func (fs *aesgcmStorage) Remove(fd storage.FileDesc) error {
	if !storage.FileDescOk(fd) {
		return storage.ErrInvalidFile
	}
	if fs.readOnly {
		return errReadOnly
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.open < 0 {
		return storage.ErrClosed
	}
	err := os.Remove(filepath.Join(fs.path, fsGenName(fd)))
	if err != nil {
		fs.Log(fmt.Sprintf("remove %s: %v", fd, err))
	}
	return err
}

func (fs *aesgcmStorage) Rename(oldfd, newfd storage.FileDesc) error {
	if !storage.FileDescOk(oldfd) || !storage.FileDescOk(newfd) {
		return storage.ErrInvalidFile
	}
	if oldfd == newfd {
		return nil
	}
	if fs.readOnly {
		return errReadOnly
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.open < 0 {
		return storage.ErrClosed
	}
	return rename(filepath.Join(fs.path, fsGenName(oldfd)), filepath.Join(fs.path, fsGenName(newfd)))
}
