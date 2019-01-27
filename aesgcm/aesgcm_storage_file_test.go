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
 * This file contains some code originally from
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
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var testKey = []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf}

var cases = []struct {
	oldName []string
	name    string
	ftype   storage.FileType
	num     int64
}{
	{nil, "000100.log", storage.TypeJournal, 100},
	{nil, "000000.log", storage.TypeJournal, 0},
	{nil, "MANIFEST-000002", storage.TypeManifest, 2},
	{nil, "MANIFEST-000007", storage.TypeManifest, 7},
	{nil, "9223372036854775807.log", storage.TypeJournal, 9223372036854775807},
	{nil, "000100.tmp", storage.TypeTemp, 100},
}

var invalidCases = []string{
	"",
	"foo",
	"foo-dx-100.log",
	".log",
	"",
	"manifest",
	"CURREN",
	"CURRENTX",
	"MANIFES",
	"MANIFEST",
	"MANIFEST-",
	"XMANIFEST-3",
	"MANIFEST-3x",
	"LOC",
	"LOCKx",
	"LO",
	"LOGx",
	"18446744073709551616.log",
	"184467440737095516150.log",
	"100",
	"100.",
	"100.lop",
}

func tempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "aesgcm-storage-")
	if err != nil {
		t.Fatal(t)
	}
	t.Log("Using temp-dir:", dir)
	return dir
}

func TestFileStorage_CreateFileName(t *testing.T) {
	for _, c := range cases {
		if name := fsGenName(storage.FileDesc{c.ftype, c.num}); name != c.name {
			t.Errorf("invalid filename got '%s', want '%s'", name, c.name)
		}
	}
}

func TestFileStorage_MetaSetGet(t *testing.T) {
	temp := tempDir(t)
	fs, err := OpenEncryptedFile(temp, testKey, false)
	if err != nil {
		t.Fatal("OpenFile: got error: ", err)
	}

	for i := 0; i < 10; i++ {
		num := rand.Int63()
		fd := storage.FileDesc{Type: storage.TypeManifest, Num: num}
		w, err := fs.Create(fd)
		if err != nil {
			t.Fatalf("Create(%d): got error: %v", i, err)
		}
		w.Write([]byte("TEST"))
		w.Close()
		if err := fs.SetMeta(fd); err != nil {
			t.Fatalf("SetMeta(%d): got error: %v", i, err)
		}
		rfd, err := fs.GetMeta()
		if err != nil {
			t.Fatalf("GetMeta(%d): got error: %v", i, err)
		}
		if fd != rfd {
			t.Fatalf("Invalid meta (%d): got '%s', want '%s'", i, rfd, fd)
		}
	}
	os.RemoveAll(temp)
}

func TestFileStorage_Meta(t *testing.T) {
	type current struct {
		num      int64
		backup   bool
		current  bool
		manifest bool
		corrupt  bool
	}
	type testCase struct {
		currents []current
		notExist bool
		corrupt  bool
		expect   int64
	}
	cases := []testCase{
		{
			currents: []current{
				{num: 2, backup: true, manifest: true},
				{num: 1, current: true},
			},
			expect: 2,
		},
		{
			currents: []current{
				{num: 2, backup: true, manifest: true},
				{num: 1, current: true, manifest: true},
			},
			expect: 1,
		},
		{
			currents: []current{
				{num: 2, manifest: true},
				{num: 3, manifest: true},
				{num: 4, current: true, manifest: true},
			},
			expect: 4,
		},
		{
			currents: []current{
				{num: 2, manifest: true},
				{num: 3, manifest: true},
				{num: 4, current: true, manifest: true, corrupt: true},
			},
			expect: 3,
		},
		{
			currents: []current{
				{num: 2, manifest: true},
				{num: 3, manifest: true},
				{num: 5, current: true, manifest: true, corrupt: true},
				{num: 4, backup: true, manifest: true},
			},
			expect: 4,
		},
		{
			currents: []current{
				{num: 4, manifest: true},
				{num: 3, manifest: true},
				{num: 2, current: true, manifest: true},
			},
			expect: 4,
		},
		{
			currents: []current{
				{num: 4, manifest: true, corrupt: true},
				{num: 3, manifest: true},
				{num: 2, current: true, manifest: true},
			},
			expect: 3,
		},
		{
			currents: []current{
				{num: 4, manifest: true, corrupt: true},
				{num: 3, manifest: true, corrupt: true},
				{num: 2, current: true, manifest: true},
			},
			expect: 2,
		},
		{
			currents: []current{
				{num: 4},
				{num: 3, manifest: true},
				{num: 2, current: true, manifest: true},
			},
			expect: 3,
		},
		{
			currents: []current{
				{num: 4},
				{num: 3, manifest: true},
				{num: 6, current: true},
				{num: 5, backup: true, manifest: true},
			},
			expect: 5,
		},
		{
			currents: []current{
				{num: 4},
				{num: 3},
				{num: 6, current: true},
				{num: 5, backup: true},
			},
			notExist: true,
		},
		{
			currents: []current{
				{num: 4, corrupt: true},
				{num: 3},
				{num: 6, current: true},
				{num: 5, backup: true},
			},
			corrupt: true,
		},
	}
	for i, tc := range cases {
		t.Logf("Test-%d", i)
		temp := tempDir(t)
		fs, err := OpenEncryptedFile(temp, testKey, false)
		if err != nil {
			t.Fatal("OpenFile: got error: ", err)
		}
		for _, cur := range tc.currents {
			var curName string
			switch {
			case cur.current:
				curName = "CURRENT"
			case cur.backup:
				curName = "CURRENT.bak"
			default:
				curName = fmt.Sprintf("CURRENT.%d", cur.num)
			}
			fd := storage.FileDesc{Type: storage.TypeManifest, Num: cur.num}
			content := fmt.Sprintf("%s\n", fsGenName(fd))
			if cur.corrupt {
				content = content[:len(content)-1-rand.Intn(3)]
			}
			if err := ioutil.WriteFile(filepath.Join(temp, curName), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
			if cur.manifest {
				w, err := fs.Create(fd)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := w.Write([]byte("TEST")); err != nil {
					t.Fatal(err)
				}
				w.Close()
			}
		}
		ret, err := fs.GetMeta()
		if tc.notExist {
			if err != os.ErrNotExist {
				t.Fatalf("expect ErrNotExist, got: %v", err)
			}
		} else if tc.corrupt {
			if !isCorrupted(err) {
				t.Fatalf("expect ErrCorrupted, got: %v", err)
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			if ret.Type != storage.TypeManifest {
				t.Fatalf("expecting manifest, got: %s", ret.Type)
			}
			if ret.Num != tc.expect {
				t.Fatalf("invalid num, expect=%d got=%d", tc.expect, ret.Num)
			}
			fis, err := ioutil.ReadDir(temp)
			if err != nil {
				t.Fatal(err)
			}
			for _, fi := range fis {
				if strings.HasPrefix(fi.Name(), "CURRENT") {
					switch fi.Name() {
					case "CURRENT", "CURRENT.bak":
					default:
						t.Fatalf("found rouge CURRENT file: %s", fi.Name())
					}
				}
				t.Logf("-> %s", fi.Name())
			}
		}
		os.RemoveAll(temp)
	}
}

func TestFileStorage_ParseFileName(t *testing.T) {
	for _, c := range cases {
		for _, name := range append([]string{c.name}, c.oldName...) {
			fd, ok := fsParseName(name)
			if !ok {
				t.Errorf("cannot parse filename '%s'", name)
				continue
			}
			if fd.Type != c.ftype {
				t.Errorf("filename '%s' invalid type got '%d', want '%d'", name, fd.Type, c.ftype)
			}
			if fd.Num != c.num {
				t.Errorf("filename '%s' invalid number got '%d', want '%d'", name, fd.Num, c.num)
			}
		}
	}
}

func TestFileStorage_InvalidFileName(t *testing.T) {
	for _, name := range invalidCases {
		if fsParseNamePtr(name, nil) {
			t.Errorf("filename '%s' should be invalid", name)
		}
	}
}

func TestFileStorage_Locking(t *testing.T) {
	temp := tempDir(t)
	defer os.RemoveAll(temp)

	p1, err := OpenEncryptedFile(temp, testKey, false)
	if err != nil {
		t.Fatal("OpenFile(1): got error: ", err)
	}

	p2, err := OpenEncryptedFile(temp, testKey, false)
	if err != nil {
		t.Logf("OpenFile(2): got error: %s (expected)", err)
	} else {
		p2.Close()
		p1.Close()
		t.Fatal("OpenFile(2): expect error")
	}

	p1.Close()

	p3, err := OpenEncryptedFile(temp, testKey, false)
	if err != nil {
		t.Fatal("OpenFile(3): got error: ", err)
	}
	defer p3.Close()

	l, err := p3.Lock()
	if err != nil {
		t.Fatal("storage lock failed(1): ", err)
	}
	_, err = p3.Lock()
	if err == nil {
		t.Fatal("expect error for second storage lock attempt")
	} else {
		t.Logf("storage lock got error: %s (expected)", err)
	}
	l.Unlock()
	_, err = p3.Lock()
	if err != nil {
		t.Fatal("storage lock failed(2): ", err)
	}
}

func TestFileStorage_ReadOnlyLocking(t *testing.T) {
	temp := tempDir(t)
	defer os.RemoveAll(temp)

	p1, err := OpenEncryptedFile(temp, testKey, false)
	if err != nil {
		t.Fatal("OpenFile(1): got error: ", err)
	}

	_, err = OpenEncryptedFile(temp, testKey, true)
	if err != nil {
		t.Logf("OpenFile(2): got error: %s (expected)", err)
	} else {
		t.Fatal("OpenFile(2): expect error")
	}

	p1.Close()

	p3, err := OpenEncryptedFile(temp, testKey, true)
	if err != nil {
		t.Fatal("OpenFile(3): got error: ", err)
	}

	p4, err := OpenEncryptedFile(temp, testKey, true)
	if err != nil {
		t.Fatal("OpenFile(4): got error: ", err)
	}

	_, err = OpenEncryptedFile(temp, testKey, false)
	if err != nil {
		t.Logf("OpenFile(5): got error: %s (expected)", err)
	} else {
		t.Fatal("OpenFile(2): expect error")
	}

	p3.Close()
	p4.Close()
}
