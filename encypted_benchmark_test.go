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
 * encrypted_benchmark_test.go: Encrypted Storage Benchmark
 */

package goleveldb_encrypted

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
)

var keys [][]byte

var shortKey = []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf}
var longKey = []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf}

func initStrings(num, len int) {
	keys = make([][]byte, num)

	rand.Seed(int64(num + len))

	for i := 0; i < num; i += 1 {
		k := make([]byte, len)
		rand.Read(k)
		keys[i] = k
	}
}

func rangeEnd(len int) []byte {
	r := make([]byte, len+1)

	for i := 0; i <= len; i += 1 {
		r[i] = 0xff
	}

	return r
}

type bType int

var (
	fileStorage      bType = 1
	encryptedStorage bType = 2
)

func loadBench(db *leveldb.DB) {
	for i := 0; i < len(keys); i += 1 {
		db.Put(keys[i], []byte{'a'}, nil)
	}
	db.CompactRange(util.Range{Start: nil, Limit: nil})
}

func doBenchMark(b *testing.B, tp bType, num, len int, key []byte) {
	initStrings(num, len)
	for n := 0; n < b.N; n++ {
		dir, err := ioutil.TempDir("", "encrypted-leveldb-bench")
		if err != nil {
			b.Fatal(err.Error())
		}

		if tp == fileStorage {
			var db *leveldb.DB
			db, err = leveldb.OpenFile(dir, nil)
			if err != nil {
				b.Fatal(err.Error())
			}
			loadBench(db)
			db.Close()
		} else if tp == encryptedStorage {
			var edb *EncryptedDB
			edb, err = OpenAESEncryptedFile(dir, key, nil)
			if err != nil {
				b.Fatal(err.Error())
			}
			loadBench(edb.DB)
			edb.Close()
		}
		os.RemoveAll(dir)
	}
}

func Benchmark_Normal_100keys_8bytes(b *testing.B) {
	doBenchMark(b, fileStorage, 100, 8, shortKey)
}

func Benchmark_AES128_100keys_8bytes(b *testing.B) {
	doBenchMark(b, encryptedStorage, 100, 8, shortKey)
}

func Benchmark_AES256_100keys_8bytes(b *testing.B) {
	doBenchMark(b, encryptedStorage, 100, 8, longKey)
}

func Benchmark_Normal_10000keys_8bytes(b *testing.B) {
	doBenchMark(b, fileStorage, 10000, 8, shortKey)
}

func Benchmark_AES128_10000keys_8bytes(b *testing.B) {
	doBenchMark(b, encryptedStorage, 10000, 8, shortKey)
}

func Benchmark_AES256_10000keys_8bytes(b *testing.B) {
	doBenchMark(b, encryptedStorage, 1000, 8, longKey)
}

func Benchmark_Normal_100keys_32bytes(b *testing.B) {
	doBenchMark(b, fileStorage, 100, 32, shortKey)
}

func Benchmark_AES128_100keys_32bytes(b *testing.B) {
	doBenchMark(b, encryptedStorage, 100, 32, shortKey)
}

func Benchmark_AES256_100keys_32bytes(b *testing.B) {
	doBenchMark(b, encryptedStorage, 100, 32, longKey)
}

func Benchmark_Normal_10000keys_32bytes(b *testing.B) {
	doBenchMark(b, fileStorage, 10000, 32, shortKey)
}

func Benchmark_AES128_10000keys_32bytes(b *testing.B) {
	doBenchMark(b, encryptedStorage, 10000, 32, shortKey)
}

func Benchmark_AES256_10000keys_32bytes(b *testing.B) {
	doBenchMark(b, encryptedStorage, 10000, 32, longKey)
}
