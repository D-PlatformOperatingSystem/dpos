package executor

import (
	"github.com/D-PlatformOperatingSystem/dpos/client"
	"github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//LocalDB      ，  localdb，         。
//     ，
//   get set      cache
//      list,    get set
type LocalDB struct {
	cache        map[string][]byte
	txcache      map[string][]byte
	keys         []string
	intx         bool
	hasbegin     bool
	kvs          []*types.KeyValue
	txid         *types.Int64
	client       queue.Client
	api          client.QueueProtocolAPI
	disableread  bool
	disablewrite bool
}

//NewLocalDB       LocalDB
func NewLocalDB(cli queue.Client, readOnly bool) db.KVDB {
	api, err := client.New(cli, nil)
	if err != nil {
		panic(err)
	}
	txid, err := api.LocalNew(readOnly)
	if err != nil {
		panic(err)
	}
	return &LocalDB{
		cache:  make(map[string][]byte),
		txid:   txid,
		client: cli,
		api:    api,
	}
}

//DisableRead     LocalDB
func (l *LocalDB) DisableRead() {
	l.disableread = true
}

//DisableWrite    LocalDB
func (l *LocalDB) DisableWrite() {
	l.disablewrite = true
}

//EnableRead     LocalDB
func (l *LocalDB) EnableRead() {
	l.disableread = false
}

//EnableWrite    LocalDB
func (l *LocalDB) EnableWrite() {
	l.disablewrite = false
}

func (l *LocalDB) resetTx() {
	l.intx = false
	l.txcache = nil
	l.keys = nil
	l.hasbegin = false
}

// StartTx reset state db keys
func (l *LocalDB) StartTx() {
	l.keys = nil
}

// GetSetKeys  get state db set keys
func (l *LocalDB) GetSetKeys() (keys []string) {
	return l.keys
}

//Begin
func (l *LocalDB) Begin() {
	l.intx = true
	l.keys = nil
	l.txcache = nil
	l.hasbegin = false
}

func (l *LocalDB) begin() {
	err := l.api.LocalBegin(l.txid)
	if err != nil {
		panic(err)
	}
}

//   save    ，      begin   ，
func (l *LocalDB) save() error {
	if l.kvs != nil {
		if !l.hasbegin {
			l.begin()
			l.hasbegin = true
		}
		param := &types.LocalDBSet{Txid: l.txid.Data}
		param.KV = l.kvs
		err := l.api.LocalSet(param)
		if err != nil {
			return err
		}
		l.kvs = nil
	}
	return nil
}

//Commit
func (l *LocalDB) Commit() error {
	for k, v := range l.txcache {
		l.cache[k] = v
	}
	err := l.save()
	if err != nil {
		return err
	}
	if l.hasbegin {
		err = l.api.LocalCommit(l.txid)
	}
	l.resetTx()
	return err
}

//Close
func (l *LocalDB) Close() error {
	l.cache = nil
	l.resetTx()
	err := l.api.LocalClose(l.txid)
	return err
}

//Rollback
func (l *LocalDB) Rollback() {
	if l.hasbegin {
		err := l.api.LocalRollback(l.txid)
		if err != nil {
			panic(err)
		}
	}
	l.resetTx()
}

//Get   key
func (l *LocalDB) Get(key []byte) ([]byte, error) {
	if l.disableread {
		return nil, types.ErrDisableRead
	}
	skey := string(key)
	if l.intx && l.txcache != nil {
		if value, ok := l.txcache[skey]; ok {
			return value, nil
		}
	}
	if value, ok := l.cache[skey]; ok {
		if value == nil {
			return nil, types.ErrNotFound
		}
		return value, nil
	}
	query := &types.LocalDBGet{Txid: l.txid.Data, Keys: [][]byte{key}}
	resp, err := l.api.LocalGet(query)
	if err != nil {
		panic(err) //no happen for ever
	}
	if nil == resp.Values {
		l.cache[skey] = nil
		return nil, types.ErrNotFound
	}
	value := resp.Values[0]
	if value == nil {
		l.cache[skey] = nil
		return nil, types.ErrNotFound
	}
	l.cache[skey] = value
	return value, nil
}

//Set   key
func (l *LocalDB) Set(key []byte, value []byte) error {
	if l.disablewrite {
		return types.ErrDisableWrite
	}
	skey := string(key)
	if l.intx {
		if l.txcache == nil {
			l.txcache = make(map[string][]byte)
		}
		l.keys = append(l.keys, skey)
		setmap(l.txcache, skey, value)
	} else {
		setmap(l.cache, skey, value)
	}
	l.kvs = append(l.kvs, &types.KeyValue{Key: key, Value: value})
	return nil
}

// List
func (l *LocalDB) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
	if l.disableread {
		return nil, types.ErrDisableRead
	}
	err := l.save()
	if err != nil {
		return nil, err
	}
	query := &types.LocalDBList{Txid: l.txid.Data, Prefix: prefix, Key: key, Count: count, Direction: direction}
	resp, err := l.api.LocalList(query)
	if err != nil {
		panic(err) //no happen for ever
	}
	values := resp.Values
	if values == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	return values, nil
}

// PrefixCount             key
func (l *LocalDB) PrefixCount(prefix []byte) (count int64) {
	panic("localdb not support PrefixCount")
}
