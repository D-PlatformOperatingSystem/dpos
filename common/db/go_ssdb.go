// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"

	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/syndtr/goleveldb/leveldb/util"

	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/types"
)

var dlog = log.New("module", "db.ssdb")
var sdbBench = &SsdbBench{}

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoSSDB(name, dir, cache)
	}
	registerDBCreator(ssDBBackendStr, dbCreator, false)
}

//SsdbBench ...
type SsdbBench struct {
	//
	writeCount int
	//
	writeNum int
	//
	writeTime time.Duration
	readCount int
	readNum   int
	readTime  time.Duration
}

//SsdbNode
type SsdbNode struct {
	ip   string
	port int
}

//GoSSDB db
type GoSSDB struct {
	BaseDB
	pool  *SDBPool
	nodes []*SsdbNode
}

func (bench *SsdbBench) write(num int, cost time.Duration) {
	bench.writeCount++
	bench.writeNum += num
	bench.writeTime += cost
}

func (bench *SsdbBench) read(num int, cost time.Duration) {
	bench.readCount++
	bench.readNum += num
	bench.readTime += cost
}

func (bench *SsdbBench) String() string {
	return fmt.Sprintf("SSDBBenchmark[(ReadTimes=%v, ReadRecordNum=%v, ReadCostTime=%v;) (WriteTimes=%v, WriteRecordNum=%v, WriteCostTime=%v)",
		bench.readCount, bench.readNum, bench.readTime, bench.writeCount, bench.writeNum, bench.writeTime)
}

func printSsdbBenchmark() {
	tick := time.Tick(time.Minute * 5)
	for {
		<-tick
		dlog.Info(sdbBench.String())
	}
}

// url pattern: ip:port,ip:port
func parseSsdbNode(url string) (nodes []*SsdbNode) {
	hosts := strings.Split(url, ",")
	if hosts == nil {
		dlog.Error("invalid ssdb url")
		return nil
	}
	for _, host := range hosts {
		parts := strings.Split(host, ":")
		if parts == nil || len(parts) != 2 {
			dlog.Error("invalid ssd url", "part", host)
			continue
		}
		ip := parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			dlog.Error("invalid ssd url host port", "port", parts[1])
			continue
		}
		nodes = append(nodes, &SsdbNode{ip, port})
	}
	return nodes
}

//NewGoSSDB new
func NewGoSSDB(name string, dir string, cache int) (*GoSSDB, error) {
	database := &GoSSDB{}
	database.nodes = parseSsdbNode(dir)

	if database.nodes == nil {
		dlog.Error("no valid ssdb instance exists, exit!")
		return nil, types.ErrDataBaseDamage
	}
	var err error
	database.pool, err = NewSDBPool(database.nodes)
	if err != nil {
		dlog.Error("connect to ssdb error!", "ssdb", database.nodes[0])
		return nil, types.ErrDataBaseDamage
	}

	go printSsdbBenchmark()

	return database, nil
}

//Get get
func (db *GoSSDB) Get(key []byte) ([]byte, error) {
	start := time.Now()

	value, err := db.pool.get().Get(string(key))
	if err != nil {
		//dlog.Error("Get value error", "error", err, "key", key, "keyhex", hex.EncodeToString(key), "keystr", string(key))
		return nil, err
	}
	if value == nil {
		return nil, ErrNotFoundInDb
	}

	sdbBench.read(1, time.Since(start))
	return value.Bytes(), nil
}

//Set
func (db *GoSSDB) Set(key []byte, value []byte) error {
	start := time.Now()

	err := db.pool.get().Set(string(key), value)
	if err != nil {
		dlog.Error("Set", "error", err)
		return err
	}
	sdbBench.write(1, time.Since(start))
	return nil
}

//SetSync
func (db *GoSSDB) SetSync(key []byte, value []byte) error {
	return db.Set(key, value)
}

//Delete
func (db *GoSSDB) Delete(key []byte) error {
	start := time.Now()

	err := db.pool.get().Del(string(key))
	if err != nil {
		dlog.Error("Delete", "error", err)
		return err
	}
	sdbBench.write(1, time.Since(start))
	return nil
}

//DeleteSync
func (db *GoSSDB) DeleteSync(key []byte) error {
	return db.Delete(key)
}

//Close
func (db *GoSSDB) Close() {
	db.pool.close()
}

//Print
func (db *GoSSDB) Print() {
}

//Stats ...
func (db *GoSSDB) Stats() map[string]string {
	return make(map[string]string)
}

//Iterator
func (db *GoSSDB) Iterator(itbeg []byte, itend []byte, reverse bool) Iterator {
	start := time.Now()

	var (
		keys  []string
		err   error
		begin string
		end   string
	)
	if itend == nil {
		itend = bytesPrefix(itbeg)
	}
	if bytes.Equal(itend, types.EmptyValue) {
		itend = nil
	}
	limit := util.Range{Start: itbeg, Limit: itend}
	if reverse {
		begin = string(limit.Limit)
		end = string(itbeg)
		keys, err = db.pool.get().Rkeys(begin, end, IteratorPageSize)
	} else {
		begin = string(itbeg)
		end = string(limit.Limit)
		keys, err = db.pool.get().Keys(begin, end, IteratorPageSize)
	}

	it := newSSDBIt(begin, end, itbeg, itend, []string{}, reverse, db)
	if err != nil {
		dlog.Error("get iterator error", "error", err, "keys", keys)
		return it
	}
	if len(keys) > 0 {
		it.keys = keys

		//                ，
		if len(it.keys) == IteratorPageSize {
			it.nextPage = true
			it.tmpEnd = it.keys[IteratorPageSize-1]
		}
	}

	sdbBench.read(len(keys), time.Since(start))
	return it
}

//   ssdb           ，      KEY；
//        KEY    ，        ，    1024 KEY；
// Next
type ssDBIt struct {
	itBase
	db      *GoSSDB
	keys    []string
	index   int
	reverse bool
	//
	begin string
	//
	end string

	//         ，
	//
	tmpEnd   string
	nextPage bool

	//        （ 0  ）
	pageNo int
}

func newSSDBIt(begin, end string, prefix, itend []byte, keys []string, reverse bool, db *GoSSDB) *ssDBIt {
	return &ssDBIt{
		itBase:  itBase{prefix, itend, reverse},
		index:   -1,
		keys:    keys,
		reverse: reverse,
		db:      db,
	}
}

func (dbit *ssDBIt) Close() {
	dbit.keys = []string{}
}

//
func (dbit *ssDBIt) cacheNextPage(begin string) bool {
	if dbit.initKeys(begin, dbit.end) {
		dbit.index = 0
		dbit.pageNo++
		return true
	}
	return false

}

func (dbit *ssDBIt) initKeys(begin, end string) bool {
	var (
		keys []string
		err  error
	)
	if dbit.reverse {
		keys, err = dbit.db.pool.get().Rkeys(begin, end, IteratorPageSize)
	} else {
		keys, err = dbit.db.pool.get().Keys(begin, end, IteratorPageSize)
	}
	if err != nil {
		dlog.Error("get iterator next page error", "error", err, "begin", begin, "end", dbit.end, "reverse", dbit.reverse)
		return false
	}

	if len(keys) > 0 {
		//      keys，   index
		dbit.keys = keys

		//                ，
		if len(keys) == IteratorPageSize {
			dbit.nextPage = true
			dbit.tmpEnd = dbit.keys[IteratorPageSize-1]
		} else {
			dbit.nextPage = false
		}
		return true
	}
	return false

}

func (dbit *ssDBIt) Next() bool {
	if len(dbit.keys) > dbit.index+1 {
		dbit.index++
		return true
	}
	//         ，
	if dbit.nextPage {
		return dbit.cacheNextPage(dbit.tmpEnd)
	}
	return false

}

func (dbit *ssDBIt) checkKeyCmp(key1, key2 string, reverse bool) bool {
	if reverse {
		return strings.Compare(key1, key2) < 0
	}
	return strings.Compare(key1, key2) > 0
}

func (dbit *ssDBIt) findInPage(key string) int {
	pos := -1
	for i, v := range dbit.keys {
		if i < dbit.index {
			continue
		}
		if dbit.checkKeyCmp(key, v, dbit.reverse) {
			continue
		} else {
			pos = i
			break
		}
	}
	return pos
}

func (dbit *ssDBIt) Seek(key []byte) bool {
	keyStr := string(key)
	pos := dbit.findInPage(keyStr)

	//          ，
	for pos == -1 && dbit.nextPage {
		if dbit.cacheNextPage(dbit.tmpEnd) {
			pos = dbit.findInPage(keyStr)
		} else {
			break
		}
	}

	dbit.index = pos
	return dbit.Valid()
}

func (dbit *ssDBIt) Rewind() bool {
	//      Rewind        ，        else  ；
	//           ，     else
	if dbit.pageNo == 0 {
		dbit.index = 0
		return true
	}

	//       N     ，Rewind
	if dbit.initKeys(dbit.begin, dbit.end) {
		dbit.index = 0
		dbit.pageNo = 0
		return true
	}
	return false

}

func (dbit *ssDBIt) Key() []byte {
	if dbit.index >= 0 && dbit.index < len(dbit.keys) {
		return []byte(dbit.keys[dbit.index])
	}
	return nil

}
func (dbit *ssDBIt) Value() []byte {
	key := dbit.keys[dbit.index]
	value, err := dbit.db.Get([]byte(key))

	if err != nil {
		dlog.Error("get iterator value error", "key", key, "error", err)
		return nil
	}
	return value
}

func (dbit *ssDBIt) Error() error {
	return nil
}

func (dbit *ssDBIt) ValueCopy() []byte {
	v := dbit.Value()
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func (dbit *ssDBIt) Valid() bool {
	start := time.Now()
	if dbit.index < 0 {
		return false
	}
	if len(dbit.keys) <= dbit.index {
		return false
	}
	key := dbit.keys[dbit.index]
	sdbBench.read(1, time.Since(start))
	return dbit.checkKey([]byte(key))
}

type ssDBBatch struct {
	db       *GoSSDB
	batchset map[string][]byte
	batchdel map[string]bool
	size     int
}

//NewBatch new
func (db *GoSSDB) NewBatch(sync bool) Batch {
	return &ssDBBatch{db: db, batchset: make(map[string][]byte), batchdel: make(map[string]bool)}
}

func (db *ssDBBatch) Set(key, value []byte) {
	db.batchset[string(key)] = value
	delete(db.batchdel, string(key))
	db.size += len(value)
	db.size += len(key)
}

func (db *ssDBBatch) Delete(key []byte) {
	db.batchset[string(key)] = []byte{}
	delete(db.batchset, string(key))
	db.batchdel[string(key)] = true
	db.size += len(key)
}

//           ，  ssdb                  ；
//            （   KEY     VALUE    ）；
//          ；
//           ，               （      ）；
func (db *ssDBBatch) Write() error {
	start := time.Now()

	if len(db.batchset) > 0 {
		err := db.db.pool.get().MultiSet(db.batchset)
		if err != nil {
			dlog.Error("Write (multi_set)", "error", err)
			return err
		}
	}

	if len(db.batchdel) > 0 {
		var dkeys []string
		for k := range db.batchdel {
			dkeys = append(dkeys, k)
		}
		err := db.db.pool.get().MultiDel(dkeys...)
		if err != nil {
			dlog.Error("Write (multi_del)", "error", err)
			return err
		}
	}

	sdbBench.write(len(db.batchset)+len(db.batchdel), time.Since(start))
	return nil
}

func (db *ssDBBatch) ValueSize() int {
	return db.size
}

//ValueLen  batch
func (db *ssDBBatch) ValueLen() int {
	return len(db.batchset)
}

func (db *ssDBBatch) Reset() {
	db.batchset = make(map[string][]byte)
	db.batchdel = make(map[string]bool)
	db.size = 0
}

func (db *ssDBBatch) UpdateWriteSync(sync bool) {
}
