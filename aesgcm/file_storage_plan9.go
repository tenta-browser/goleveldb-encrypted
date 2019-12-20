// Originally from the version in goleveldb at
// https://github.com/syndtr/goleveldb/blob/master/leveldb/storage/file_storage_unix.go
// (note, the LICENSE file mentioned below refers to https://github.com/syndtr/goleveldb/blob/master/LICENSE)

// Copyright (c) 2012, Suryandaru Triandana <syndtr@gmail.com>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package aesgcm

import (
	"os"
)

type plan9FileLock struct {
	f *os.File
}

func (fl *plan9FileLock) release() error {
	return fl.f.Close()
}

func newFileLock(path string, readOnly bool) (fl fileLock, err error) {
	var (
		flag int
		perm os.FileMode
	)
	if readOnly {
		flag = os.O_RDONLY
	} else {
		flag = os.O_RDWR
		perm = os.ModeExclusive
	}
	f, err := os.OpenFile(path, flag, perm)
	if os.IsNotExist(err) {
		f, err = os.OpenFile(path, flag|os.O_CREATE, perm|0644)
	}
	if err != nil {
		return
	}
	fl = &plan9FileLock{f: f}
	return
}

func rename(oldpath, newpath string) error {
	if _, err := os.Stat(newpath); err == nil {
		if err := os.Remove(newpath); err != nil {
			return err
		}
	}

	return os.Rename(oldpath, newpath)
}

func syncDir(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return err
	}
	return nil
}
