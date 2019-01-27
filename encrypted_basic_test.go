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
 * encrypted_basic_test.go: Basic test of the encrypted storage using the wrapper function
 */

package goleveldb_encrypted

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"testing"
)

var testKey = []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf}

func tempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "encrypted-leveldb")
	if err != nil {
		t.Fatal(t)
	}
	t.Log("Using temp-dir:", dir)
	return dir
}

var basicTestData = []struct {
	key, value string
}{
	{"hello", "world"},
	{"fizz", "buzz"},
	{"foo", "bar"},
}

func TestOpenAESEncryptedFile(t *testing.T) {
	d := tempDir(t)

	db, e := OpenAESEncryptedFile(d, testKey, nil)
	//db, e := leveldb.OpenFile(d, nil)
	if e != nil {
		t.Logf("%s", e.Error())
		t.Fail()
	}

	for _, i := range basicTestData {
		e := db.Put([]byte(i.key), []byte(i.value), nil)
		if e != nil {
			t.Logf("%s", e.Error())
			t.Fail()
		}
	}

	db.Close()

	db2, e := OpenAESEncryptedFile(d, testKey, nil)

	if e != nil {
		println(e.Error())
		t.Logf("%s", e.Error())
		t.Fail()
		return
	}

	for _, i := range basicTestData {
		val, e := db2.Get([]byte(i.key), nil)
		if e != nil {
			t.Logf("%s", e.Error())
			t.Fail()
		}
		if !bytes.Equal(val, []byte(i.value)) {
			t.Logf("expected %s, got %s", i.value, val)
			t.Fail()
		}
	}

	db2.Close()

	os.RemoveAll(d)
}

func TestOpenAESEncryptedFile_Fuzz(t *testing.T) {
	d := tempDir(t)

	db, e := OpenAESEncryptedFile(d, testKey, nil)

	if e != nil {
		t.Logf("Could not create DB: %s", e.Error())
		t.Fail()
		return
	}

	keys := make([]string, 1000)

	for i := 0; i < 1000; i += 1 {
		k := make([]byte, 8)
		rand.Read(k)
		s := fmt.Sprintf("%016X", k)
		h := hmac.New(sha512.New512_256, testKey)
		v := h.Sum([]byte(s))

		db.Put([]byte(s), v, nil)

		keys[i] = s
	}

	db.Close()

	sort.Strings(keys)

	db2, e := OpenAESEncryptedFile(d, testKey, nil)

	if e != nil {
		t.Logf("Could not reopen DB: %s", e.Error())
		t.Fail()
		return
	}

	for _, k := range keys {
		h, e := db2.Has([]byte(k), nil)
		if !h {
			t.Logf("Missing key %s", k)
			t.Fail()
		}
		if e != nil {
			t.Logf("Has error: %s", e.Error())
			t.Fail()
		}
	}

	iter := db2.NewIterator(&util.Range{Start: []byte{0x0}, Limit: []byte{'~'}}, nil)

	x := 0

	for iter.Next() {
		k := keys[x]
		h := hmac.New(sha512.New512_256, testKey)
		v := h.Sum([]byte(k))

		if !bytes.Equal(iter.Key(), []byte(k)) {
			t.Logf("Out of order key: expected %s, actual %s", k, string(iter.Key()))
			t.Fail()
		}

		if !bytes.Equal(iter.Value(), v) {
			t.Logf("Invalid data: expected %016X, actual %016X", v, iter.Value())
			t.Fail()
		}
		x += 1
	}

	db2.Put([]byte("hello"), []byte("world"), nil)

	for i := 0; i < 1000; i += 1 {
		db2.Delete([]byte(keys[i]), nil)
	}

	db2.Close()

	db3, e := OpenAESEncryptedFile(d, testKey, nil)

	for _, k := range keys {
		h, e := db3.Has([]byte(k), nil)
		if h {
			t.Logf("Still has key %s", k)
			t.Fail()
		}
		if e != nil {
			t.Logf("Has error: %s", e.Error())
			t.Fail()
		}
	}

	h, e := db3.Has([]byte("hello"), nil)

	if !h {
		t.Logf("Missing key hello")
		t.Fail()
	}

	os.RemoveAll(d)
}
