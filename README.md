GoLevelDB Encrypted Storage
==============

[![Go Report Card](https://goreportcard.com/badge/github.com/tenta-browser/goleveldb-encrypted)](https://goreportcard.com/report/github.com/tenta-browser/goleveldb-encrypted)
[![GoDoc](https://godoc.org/github.com/tenta-browser/goleveldb-encrypted?status.svg)](https://godoc.org/github.com/tenta-browser/goleveldb-encrypted)

GoLevelDB Encrypted Storage provides a strongly encrypted storage (data at rest) for [GoLevelDB](https://github.com/syndtr/goleveldb).

Contact: developer@tenta.io

Installation
============

1. `go get github.com/tenta-browser/goleveldb-encrypted`

Usage
=====

Since the storage engine can be manually instantiated (see the aesgcm package for the raw storage interface), but for most
use cases a wrapped is provided equivalent to the `OpenFile` wrapper in GoLevelDB. So simply replace a call to `OpenFile`
with a call to `OpenAESEncryptedFile`, and then use the database just like you normally would

```
db, err = OpenAESEncryptedFile(dir, key, nil)
defer db.Close()

db.Put([]byte("hello"), []byte("value"))
```

Security
========

This encryption engine is designed to be secure, but it's still under active development and we do not use it in production projects
yet. We'd be thrilled for everyone to test the heck out of it and endeavour to find problems with the implementation or security.

The entire contents of all data files are encrypted in AEAD mode using AES128 or AES256. Encryption mode is selected automatically based
on the key length, for AES128, use a 16 byte key and for AES256 a 32 byte key. The only files unencrypted are the `LOCK` file, which
exists only as a filesystem lock to prevent database corruption and the CURRENT file, which simply contains a pointer to the currently
active file (but no data).

File names are _not_ encrypted, however, they are simply numerically increasing sequence numbers, and we currently do not believe that
any meaningful information can be extracted from knowing the segment file numbers, however we will continue to evaluate this choice as
we develop this library.

An attacker will be able to estimate the total quantity of data (key length + value length) stored in the database. We do not believe that it will be practical to
determine the number of keys and values in the database, and we believe that the contents of the keys and values are strongly encrypted.

The current construction of the nonce has a very small chance of collision, if the database engine writes on the order of 2^32 file
segments. Given LevelDB's file write behavior this seems improbably even on extremely large and busy DB's, but we'll do further analysis
of the nonce implementation before declaring this code ready for production.

Performance
===========

GoLevelDB Encrypted Storage is still under active development and we do not use it in production yet. On small databases it runs somewhat
slower than the default file storage engine, as lots of time is spent encrypting housekeeping data compared to the default file storage.
On larger databases, the speed difference is small.

In addition, due to go's excellent AES support, there are minimal speed differences between 128 and 256 bit keys, so there's no
reason not to use longer keys.

On linux with go 1.11.4:

```
Benchmark_Normal_100keys_8bytes-4      	     300	 120535960 ns/op
Benchmark_AES128_100keys_8bytes-4      	     100	 207515379 ns/op
Benchmark_AES256_100keys_8bytes-4      	     100	 212134046 ns/op
Benchmark_Normal_10000keys_8bytes-4    	     100	 403099514 ns/op
Benchmark_AES128_10000keys_8bytes-4    	     100	 404127417 ns/op
Benchmark_AES256_10000keys_8bytes-4    	     100	 234569404 ns/op
Benchmark_Normal_100keys_32bytes-4     	     300	 108473730 ns/op
Benchmark_AES128_100keys_32bytes-4     	     100	 212959290 ns/op
Benchmark_AES256_100keys_32bytes-4     	     100	 220255819 ns/op
Benchmark_Normal_10000keys_32bytes-4   	     100	 434509217 ns/op
Benchmark_AES128_10000keys_32bytes-4   	     100	 474149648 ns/op
Benchmark_AES256_10000keys_32bytes-4   	     100	 443302136 ns/op

```

License
=======

This project contains some code from the GoLevelDB project, as this code is currently
package private.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

In addition, this entire repository may also be used under the BSD 3-clause
license available of the [GoLevelDB Project](https://github.com/syndtr/goleveldb/blob/master/LICENSE).

For any questions, please contact developer@tenta.io

Contributing
============

We welcome contributions, feedback and plain old complaining. Feel free to open
an issue or shoot us a message to developer@tenta.io. If you'd like to contribute,
please open a pull request and send us an email to sign a contributor agreement.

About Tenta
===========

This encryption library is brought to you by Team Tenta. Tenta is your [private, encrypted browser](https://tenta.com) that protects your data instead of selling. We're building a next-generation browser that combines all the privacy tools you need, including built-in OpenVPN. Everything is encrypted by default. That means your bookmarks, saved tabs, web history, web traffic, downloaded files, IP address and DNS. A truly incognito browser that's fast and easy.
