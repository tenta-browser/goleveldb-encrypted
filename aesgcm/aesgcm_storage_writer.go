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
 * aesgcm_storage_reader.go: Implementation of storage.Reader for GoLevelDB
 *
 */

package aesgcm

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"os"
)

type aesgcmWriter struct {
	*bytes.Buffer
	fs     *aesgcmStorage
	fd     storage.FileDesc
	closed bool
	fp     *os.File
}

func newWriter(fp *os.File, fd storage.FileDesc, fs *aesgcmStorage) *aesgcmWriter {
	return &aesgcmWriter{
		Buffer: new(bytes.Buffer),
		fs:     fs,
		fd:     fd,
		closed: false,
		fp:     fp,
	}
}

func (w *aesgcmWriter) Close() error {
	err := w.Sync()
	if err != nil {
		return err
	}
	w.fs.mu.Lock()
	defer w.fs.mu.Unlock()
	if w.closed {
		return storage.ErrClosed
	}
	w.closed = true
	w.fs.open--
	err = w.fp.Close()
	if err != nil {
		w.fs.Log(fmt.Sprintf("close %s: %v", w.fd, err))
	}
	return err
}

func (w *aesgcmWriter) Sync() error {
	if err := w.fp.Truncate(0); err != nil {
		w.fs.Log(fmt.Sprintf("truncate %s: %v", w.fd, err))
		return err
	}

	if _, err := w.fp.Seek(0, 0); err != nil {
		w.fs.Log(fmt.Sprintf("seek %s: %v", w.fd, err))
		return err
	}

	nonce := make([]byte, w.fs.cyp.NonceSize())

	read, err := rand.Read(nonce)
	if err != nil {
		return err
	}
	if read != w.fs.cyp.NonceSize() {
		return errNonceUnavailable
	}

	crypt := w.fs.cyp.Seal(nil, nonce, w.Buffer.Bytes(), fdGenAD(w.fd))

	_, err = w.fp.Write(nonce)
	if err != nil {
		w.fs.Log(fmt.Sprintf("write %s: %v", w.fd, err))
		return err
	}

	_, err = w.fp.Write(crypt)
	if err != nil {
		w.fs.Log(fmt.Sprintf("write %s: %v", w.fd, err))
		return err
	}

	err = w.fp.Sync()
	if err != nil {
		w.fs.Log(fmt.Sprintf("sync %s: %v", w.fd, err))
		return err
	}

	if w.fd.Type == storage.TypeManifest {
		// Also sync parent directory if file type is manifest.
		// See: https://code.google.com/p/leveldb/issues/detail?id=190.
		if err := syncDir(w.fs.path); err != nil {
			w.fs.Log(fmt.Sprintf("syncDir: %v", err))
			return err
		}
	}

	return nil
}
