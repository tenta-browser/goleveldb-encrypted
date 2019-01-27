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
 * encrypted.go: Encrypted Storage Wrapper
 */

package goleveldb_encrypted

import (
	"github.com/jwriteclub/goleveldb-encrypted/aesgcm"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"io"
)

type EncryptedDB struct {
	*leveldb.DB
	scloser io.Closer
}

func (e *EncryptedDB) Close() {
	e.DB.Close()
	e.scloser.Close()
}

func OpenAESEncryptedFile(path string, key []byte, opt *opt.Options) (db *EncryptedDB, err error) {
	stor, err := aesgcm.OpenEncryptedFile(path, key, opt.GetReadOnly())
	if err != nil {
		return
	}
	ldb, err := leveldb.Open(stor, opt)
	if err != nil {
		stor.Close()
	} else {
		db = &EncryptedDB{
			DB:      ldb,
			scloser: stor,
		}
	}
	return
}
