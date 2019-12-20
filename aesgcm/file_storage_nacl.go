// Originally from the version in goleveldb at
// https://github.com/syndtr/goleveldb/blob/master/leveldb/storage/file_storage_unix.go
// (note, the LICENSE file mentioned below refers to https://github.com/syndtr/goleveldb/blob/master/LICENSE)

// Copyright (c) 2012, Suryandaru Triandana <syndtr@gmail.com>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build nacl

package aesgcm

import (
	"os"
	"syscall"
)

func newFileLock(path string, readOnly bool) (fl fileLock, err error) {
	return nil, syscall.ENOTSUP
}

func setFileLock(f *os.File, readOnly, lock bool) error {
	return syscall.ENOTSUP
}

func rename(oldpath, newpath string) error {
	return syscall.ENOTSUP
}

func isErrInvalid(err error) bool {
	return false
}

func syncDir(name string) error {
	return syscall.ENOTSUP
}
