// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"

	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var mlog = log.New("module", "db.memdb")

// memdb

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoMemDB(name, dir, cache)
	}
	registerDBCreator(memDBBackendStr, dbCreator, false)
}

//GoMemDB db
type GoMemDB struct {
	BaseDB
	db *memdb.DB
}

//NewGoMemDB new
func NewGoMemDB(name string, dir string, cache int) (*GoMemDB, error) {
	return &GoMemDB{
		db: memdb.New(comparer.DefaultComparer, 0),
	}, nil
}

//Get get
func (db *GoMemDB) Get(key []byte) ([]byte, error) {
	v, err := db.db.Get(key)
	if err != nil {
		return nil, ErrNotFoundInDb
	}
	return cloneByte(v), nil
}

//Set set
func (db *GoMemDB) Set(key []byte, value []byte) error {
	err := db.db.Put(key, value)
	if err != nil {
		llog.Error("Set", "error", err)
		return err
	}
	return nil
}

//SetSync
func (db *GoMemDB) SetSync(key []byte, value []byte) error {
	err := db.db.Put(key, value)
	if err != nil {
		llog.Error("SetSync", "error", err)
		return err
	}
	return nil
}

//Delete
func (db *GoMemDB) Delete(key []byte) error {
	err := db.db.Delete(key)
	if err != nil {
		llog.Error("Delete", "error", err)
		return err
	}
	return nil
}

//DeleteSync
func (db *GoMemDB) DeleteSync(key []byte) error {
	err := db.db.Delete(key)
	if err != nil {
		llog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

//DB db
func (db *GoMemDB) DB() *memdb.DB {
	return db.db
}

//Close
func (db *GoMemDB) Close() {
}

//Print
func (db *GoMemDB) Print() {
	it := db.db.NewIterator(nil)
	for it.Next() {
		mlog.Info("Print", "key", string(it.Key()), "value", string(it.Value()))
	}
}

//Stats ...
func (db *GoMemDB) Stats() map[string]string {
	//TODO
	return nil
}

//Iterator
func (db *GoMemDB) Iterator(start []byte, end []byte, reverse bool) Iterator {
	if end == nil {
		end = bytesPrefix(start)
	}
	if bytes.Equal(end, types.EmptyValue) {
		end = nil
	}
	r := &util.Range{Start: start, Limit: end}
	it := db.db.NewIterator(r)
	base := itBase{start, end, reverse}
	return &goLevelDBIt{it, base}
}

type kv struct{ k, v []byte }
type memBatch struct {
	db     *GoMemDB
	writes []kv
	size   int
	len    int
}

//NewBatch new
func (db *GoMemDB) NewBatch(sync bool) Batch {
	return &memBatch{db: db}
}

func (b *memBatch) Set(key, value []byte) {
	b.writes = append(b.writes, kv{cloneByte(key), cloneByte(value)})
	b.size += len(value)
	b.size += len(key)
	b.len += len(value)
}

func (b *memBatch) Delete(key []byte) {
	b.writes = append(b.writes, kv{cloneByte(key), nil})
	b.size += len(key)
	b.len++
}

func (b *memBatch) Write() error {
	var err error
	for _, kv := range b.writes {
		if kv.v == nil {
			err = b.db.Delete(kv.k)
		} else {
			err = b.db.Set(kv.k, kv.v)
		}
	}
	return err
}

func (b *memBatch) ValueSize() int {
	return b.size
}

//ValueLen  batch
func (b *memBatch) ValueLen() int {
	return b.len
}

func (b *memBatch) Reset() {
	//  batch   buf，    db reset，     db
	b.writes = b.writes[:0]
	b.size = 0
	b.len = 0
}

func (b *memBatch) UpdateWriteSync(sync bool) {
}
