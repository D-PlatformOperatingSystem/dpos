package db

import (
	"sync"
)

// LocalDB local db for store key value in local
type LocalDB struct {
	txcache  DB
	cache    DB
	maindb   DB
	intx     bool
	mu       sync.RWMutex
	readOnly bool
}

func newMemDB() DB {
	memdb, err := NewGoMemDB("", "", 0)
	if err != nil {
		panic(err)
	}
	return memdb
}

// NewLocalDB new local db
func NewLocalDB(maindb DB, readOnly bool) KVDB {
	if readOnly {
		//       memdb，      ，     localdb，  memdb
		return &LocalDB{
			maindb:   maindb,
			readOnly: true,
		}
	}
	return &LocalDB{
		cache:   newMemDB(),
		txcache: newMemDB(),
		maindb:  maindb,
	}
}

// Get get value from local db
func (l *LocalDB) Get(key []byte) ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	value, err := l.get(key)
	if isdeleted(value) {
		//       (          emptyvalue)
		return nil, ErrNotFoundInDb
	}
	return value, err
}

func (l *LocalDB) get(key []byte) ([]byte, error) {
	if l.intx && l.txcache != nil {
		if value, err := l.txcache.Get(key); err == nil {
			return value, nil
		}
	}
	if l.cache != nil {
		if value, err := l.cache.Get(key); err == nil {
			return value, nil
		}
	}
	value, err := l.maindb.Get(key)
	if err != nil {
		return nil, err
	}
	if l.cache != nil {
		err = l.cache.Set(key, value)
		if err != nil {
			panic(err)
		}
	}
	return value, nil
}

// Set set key value to local db
func (l *LocalDB) Set(key []byte, value []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.readOnly {
		panic("set local db in read only mode")
	}
	if l.intx {
		if l.txcache == nil {
			l.txcache = newMemDB()
		}
		setdb2(l.txcache, key, value)
	} else if l.cache != nil {
		setdb2(l.cache, key, value)
	}
	return nil
}

// List            ，set   cache         list
func (l *LocalDB) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	dblist := make([]IteratorDB, 0)
	if l.txcache != nil {
		dblist = append(dblist, l.txcache)
	}
	if l.cache != nil {
		dblist = append(dblist, l.cache)
	}
	if l.maindb != nil {
		dblist = append(dblist, l.maindb)
	}
	mergedb := NewMergedIteratorDB(dblist)
	it := NewListHelper(mergedb)
	return it.List(prefix, key, count, direction), nil
}

// PrefixCount             key
func (l *LocalDB) PrefixCount(prefix []byte) (count int64) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	dblist := make([]IteratorDB, 0)
	if l.txcache != nil {
		dblist = append(dblist, l.txcache)
	}
	if l.cache != nil {
		dblist = append(dblist, l.cache)
	}
	if l.maindb != nil {
		dblist = append(dblist, l.maindb)
	}
	mergedb := NewMergedIteratorDB(dblist)
	it := NewListHelper(mergedb)
	return it.PrefixCount(prefix)
}

//Begin
func (l *LocalDB) Begin() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.intx = true
	l.txcache = nil
}

// Rollback reset tx
func (l *LocalDB) Rollback() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.resetTx()
}

// Commit canche tx
func (l *LocalDB) Commit() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.txcache == nil {
		l.resetTx()
		return nil
	}
	it := l.txcache.Iterator(nil, nil, false)
	for it.Next() {
		err := l.cache.Set(it.Key(), it.Value())
		if err != nil {
			panic(err)
		}
	}
	l.resetTx()
	return nil
}

func (l *LocalDB) resetTx() {
	l.intx = false
	l.txcache = nil
}

func setdb2(d DB, key []byte, value []byte) {
	//value == nil     key，  key
	err := d.Set(key, value)
	if err != nil {
		panic(err)
	}
}
