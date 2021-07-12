// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"

	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
)

var blog = log.New("module", "db.gobadgerdb")

//GoBadgerDB db
type GoBadgerDB struct {
	BaseDB
	db *badger.DB
}

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoBadgerDB(name, dir, cache)
	}
	registerDBCreator(goBadgerDBBackendStr, dbCreator, false)
}

//NewGoBadgerDB new
func NewGoBadgerDB(name string, dir string, cache int) (*GoBadgerDB, error) {
	opts := badger.DefaultOptions(dir)
	if cache <= 128 {
		opts.ValueLogLoadingMode = options.FileIO
		//opts.MaxTableSize = int64(cache) << 18 // cache = 128, MaxTableSize = 32M
		opts.NumCompactors = 1
		opts.NumMemtables = 1
		opts.NumLevelZeroTables = 1
		opts.NumLevelZeroTablesStall = 2
		opts.TableLoadingMode = options.MemoryMap
		opts.ValueLogFileSize = 1 << 28 // 256M
	}

	db, err := badger.Open(opts)
	if err != nil {
		blog.Error("NewGoBadgerDB", "error", err)
		return nil, err
	}

	return &GoBadgerDB{db: db}, nil
}

//Get get
func (db *GoBadgerDB) Get(key []byte) ([]byte, error) {
	var val []byte
	err := db.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFoundInDb
			}
			blog.Error("Get", "txn.Get.error", err)
			return err

		}
		//xxxx
		val, err = item.ValueCopy(nil)
		if err != nil {
			blog.Error("Get", "item.Value.error", err)
			return err
		}

		//   leveldb
		if val == nil {
			val = make([]byte, 0)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return val, nil
}

//Set set
func (db *GoBadgerDB) Set(key []byte, value []byte) error {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, value)
		return err
	})

	if err != nil {
		blog.Error("Set", "error", err)
		return err
	}
	return nil
}

//SetSync
func (db *GoBadgerDB) SetSync(key []byte, value []byte) error {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, value)
		return err
	})

	if err != nil {
		blog.Error("SetSync", "error", err)
		return err
	}
	return nil
}

//Delete
func (db *GoBadgerDB) Delete(key []byte) error {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(key)
		return err
	})

	if err != nil {
		blog.Error("Delete", "error", err)
		return err
	}
	return nil
}

//DeleteSync
func (db *GoBadgerDB) DeleteSync(key []byte) error {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(key)
		return err
	})

	if err != nil {
		blog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

//DB db
func (db *GoBadgerDB) DB() *badger.DB {
	return db.db
}

//Close
func (db *GoBadgerDB) Close() {
	err := db.db.Close()
	if err != nil {
		return
	}
}

//Print
func (db *GoBadgerDB) Print() {
	// TODO: Returns statistics of the underlying DB
	err := db.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			v, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			blog.Info("Print", "key", string(k), "value", string(v))
			//blog.Info("Print", "key", string(item.Key()))
		}
		return nil
	})
	if err != nil {
		blog.Error("Print", err)
	}
}

//Stats ...
func (db *GoBadgerDB) Stats() map[string]string {
	//TODO
	return nil
}

//Iterator
func (db *GoBadgerDB) Iterator(start, end []byte, reverse bool) Iterator {
	txn := db.db.NewTransaction(false)
	opts := badger.DefaultIteratorOptions
	opts.Reverse = reverse
	it := txn.NewIterator(opts)
	if end == nil {
		end = bytesPrefix(start)
	}
	if bytes.Equal(end, types.EmptyValue) {
		end = nil
	}
	if reverse {
		it.Seek(end)
	} else {
		it.Seek(start)
	}
	return &goBadgerDBIt{it, itBase{start, end, reverse}, txn, nil}
}

type goBadgerDBIt struct {
	*badger.Iterator
	itBase
	txn *badger.Txn
	err error
}

//Next next
func (it *goBadgerDBIt) Next() bool {
	it.Iterator.Next()
	return it.Valid()
}

//Rewind ...
func (it *goBadgerDBIt) Rewind() bool {
	if it.reverse {
		it.Seek(it.end)
	} else {
		it.Seek(it.start)
	}
	return it.Valid()
}

//Seek
func (it *goBadgerDBIt) Seek(key []byte) bool {
	it.Iterator.Seek(key)
	return it.Valid()
}

//Close
func (it *goBadgerDBIt) Close() {
	it.Iterator.Close()
	it.txn.Discard()
}

//Valid
func (it *goBadgerDBIt) Valid() bool {
	return it.Iterator.Valid() && it.checkKey(it.Key())
}

func (it *goBadgerDBIt) Key() []byte {
	return it.Item().Key()
}

func (it *goBadgerDBIt) Value() []byte {
	value, err := it.Item().ValueCopy(nil)
	if err != nil {
		it.err = err
	}
	return value
}

func (it *goBadgerDBIt) ValueCopy() []byte {
	value, err := it.Item().ValueCopy(nil)
	if err != nil {
		it.err = err
	}
	return value
}

func (it *goBadgerDBIt) Error() error {
	return it.err
}

//GoBadgerDBBatch batch
type GoBadgerDBBatch struct {
	db    *GoBadgerDB
	batch *badger.Txn
	//wop   *opt.WriteOptions
	size int
	len  int
}

//NewBatch new
func (db *GoBadgerDB) NewBatch(sync bool) Batch {
	batch := db.db.NewTransaction(true)
	return &GoBadgerDBBatch{db, batch, 0, 0}
}

//Set set
func (mBatch *GoBadgerDBBatch) Set(key, value []byte) {
	err := mBatch.batch.Set(key, value)
	if err != nil {
		blog.Error("Set", "error", err)
	}
	mBatch.size += len(value)
	mBatch.size += len(key)
	mBatch.len += len(value)
}

//Delete
func (mBatch *GoBadgerDBBatch) Delete(key []byte) {
	err := mBatch.batch.Delete(key)
	if err != nil {
		blog.Error("Delete", "error", err)
	}
	mBatch.size += len(key)
	mBatch.len++
}

//Write
func (mBatch *GoBadgerDBBatch) Write() error {
	defer mBatch.batch.Discard()

	if err := mBatch.batch.Commit(); err != nil {
		blog.Error("Write", "error", err)
		return err
	}
	return nil
}

//ValueSize batch
func (mBatch *GoBadgerDBBatch) ValueSize() int {
	return mBatch.size
}

//ValueLen  batch
func (mBatch *GoBadgerDBBatch) ValueLen() int {
	return mBatch.len
}

//Reset
func (mBatch *GoBadgerDBBatch) Reset() {
	if nil != mBatch.db && nil != mBatch.db.db {
		mBatch.batch = mBatch.db.db.NewTransaction(true)
	}
	mBatch.size = 0
	mBatch.len = 0
}

// UpdateWriteSync ...
func (mBatch *GoBadgerDBBatch) UpdateWriteSync(sync bool) {
}
