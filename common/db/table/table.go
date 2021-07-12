// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//Package table       kv
package table

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
	"github.com/golang/protobuf/proto"
)

//    :
/*
  :
save:
    :
tableprefix + tablename + Primary -> data

index:
tableprefix + tablemetaname + index + primary -> primary

read:
list by Primary ->
list by index

  index         primary list
   table    （   primary key）

del:
   primaryKey + index
*/

//
//
//primary key auto  del      primary key
const (
	None = iota
	Add
	Update
	Del
)

//meta key
const meta = sep + "m" + sep
const data = sep + "d" + sep

//RowMeta
type RowMeta interface {
	CreateRow() *Row
	SetPayload(types.Message) error
	Get(key string) ([]byte, error)
}

//Row
type Row struct {
	Ty      int
	Primary []byte
	Data    types.Message
	old     types.Message
}

func encodeInt64(p int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, p)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeInt64(p []byte) (int64, error) {
	buf := bytes.NewBuffer(p)
	var i int64
	err := binary.Read(buf, binary.LittleEndian, &i)
	if err != nil {
		return 0, err
	}
	return i, nil
}

//Encode row
func (row *Row) Encode() ([]byte, error) {
	b, err := encodeInt64(int64(len(row.Primary)))
	if err != nil {
		return nil, err
	}
	b = append(b, row.Primary...)
	b = append(b, types.Encode(row.Data)...)
	return b, nil
}

//DecodeRow from data
func DecodeRow(data []byte) ([]byte, []byte, error) {
	if len(data) <= 8 {
		return nil, nil, types.ErrDecode
	}
	l, err := decodeInt64(data[:8])
	if err != nil {
		return nil, nil, err
	}
	if len(data) < int(l)+8 {
		return nil, nil, types.ErrDecode
	}
	return data[8 : int(l)+8], data[int(l)+8:], nil
}

//Table      ,      primary key, index key
type Table struct {
	meta       RowMeta
	rows       []*Row
	rowmap     map[string]*Row
	kvdb       db.KV
	opt        *Option
	autoinc    *Count
	dataprefix string
	metaprefix string
}

//Option table
type Option struct {
	Prefix  string
	Name    string
	Primary string
	Join    bool
	Index   []string
}

const sep = "-"
const joinsep = "#"

//NewTable
//primary    : auto,
//index    nil
func NewTable(rowmeta RowMeta, kvdb db.KV, opt *Option) (*Table, error) {
	if len(opt.Index) > 16 {
		return nil, ErrTooManyIndex
	}
	for _, index := range opt.Index {
		if strings.Contains(index, sep) || index == "primary" {
			return nil, ErrIndexKey
		}
		if !opt.Join && strings.Contains(index, joinsep) {
			return nil, ErrIndexKey
		}
	}
	if opt.Primary == "" {
		opt.Primary = "auto"
	}
	if _, err := getPrimaryKey(rowmeta, opt.Primary); err != nil {
		return nil, err
	}
	//     "-"
	if strings.Contains(opt.Name, sep) {
		return nil, ErrTablePrefixOrTableName
	}
	// jointable     "#"
	if !opt.Join && strings.Contains(opt.Name, joinsep) {
		return nil, ErrTablePrefixOrTableName
	}
	dataprefix := opt.Prefix + sep + opt.Name + data
	metaprefix := opt.Prefix + sep + opt.Name + meta
	count := NewCount(opt.Prefix, opt.Name+sep+"autoinc"+sep, kvdb)
	return &Table{
		meta:       rowmeta,
		kvdb:       kvdb,
		rowmap:     make(map[string]*Row),
		opt:        opt,
		autoinc:    count,
		dataprefix: dataprefix,
		metaprefix: metaprefix}, nil
}

func getPrimaryKey(meta RowMeta, primary string) ([]byte, error) {
	if primary == "" {
		return nil, ErrEmptyPrimaryKey
	}
	if strings.Contains(primary, sep) {
		return nil, ErrPrimaryKey
	}
	if primary != "auto" {
		key, err := meta.Get(primary)
		return key, err
	}
	return nil, nil
}

func (table *Table) addRowCache(row *Row) {
	primary := string(row.Primary)
	if row.Ty == Del {
		delete(table.rowmap, primary)
	} else if row.Ty == Add || row.Ty == Update {
		table.rowmap[primary] = row
	}
	table.rows = append(table.rows, row)
}

func (table *Table) delRowCache(row *Row) {
	row.Ty = None
	primary := string(row.Primary)
	delete(table.rowmap, primary)
}

func (table *Table) mergeCache(rows []*Row, indexName string, indexValue []byte) ([]*Row, error) {
	replaced := make(map[string]bool)
	for i, row := range rows {
		if cacherow, ok := table.rowmap[string(row.Primary)]; ok {
			rows[i] = cacherow
			replaced[string(row.Primary)] = true
		}
	}
	//add not in db but in cache rows
	for _, row := range table.rowmap {
		if _, ok := replaced[string(row.Primary)]; ok {
			continue
		}
		v, err := table.index(row, indexName)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(v, indexValue) {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (table *Table) findRow(primary []byte) (*Row, bool, error) {
	if row, ok := table.rowmap[string(primary)]; ok {
		return row, true, nil
	}
	row, err := table.GetData(primary)
	return row, false, err
}

func (table *Table) hasIndex(name string) bool {
	for _, index := range table.opt.Index {
		if index == name {
			return true
		}
	}
	return false
}

func (table *Table) canGet(name string) bool {
	row := table.meta.CreateRow()
	err := table.meta.SetPayload(row.Data)
	if err != nil {
		return false
	}
	_, err = table.meta.Get(name)
	return err == nil
}

func (table *Table) checkIndex(data types.Message) error {
	err := table.meta.SetPayload(data)
	if err != nil {
		return err
	}
	if _, err := getPrimaryKey(table.meta, table.opt.Primary); err != nil {
		return err
	}
	for i := 0; i < len(table.opt.Index); i++ {
		_, err := table.meta.Get(table.opt.Index[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (table *Table) getPrimaryAuto() ([]byte, error) {
	i, err := table.autoinc.Inc()
	if err != nil {
		return nil, err
	}
	return []byte(pad(i)), nil
}

//primaryKey
//1. auto     ,    。
//2.   auto
func (table *Table) primaryKey(data types.Message) (primaryKey []byte, err error) {
	if table.opt.Primary == "auto" {
		primaryKey, err = table.getPrimaryAuto()
		if err != nil {
			return nil, err
		}
	} else {
		primaryKey, err = table.getPrimaryFromData(data)
	}
	return
}

func (table *Table) getPrimaryFromData(data types.Message) (primaryKey []byte, err error) {
	err = table.meta.SetPayload(data)
	if err != nil {
		return nil, err
	}
	primaryKey, err = getPrimaryKey(table.meta, table.opt.Primary)
	if err != nil {
		return nil, err
	}
	return
}

//ListIndex  list table index
func (table *Table) ListIndex(indexName string, prefix []byte, primaryKey []byte, count, direction int32) (rows []*Row, err error) {
	kvdb, ok := table.kvdb.(db.KVDB)
	if !ok {
		return nil, errors.New("list only support KVDB interface")
	}
	query := &Query{table: table, kvdb: kvdb}
	return query.ListIndex(indexName, prefix, primaryKey, count, direction)
}

//Replace       ，
func (table *Table) Replace(data types.Message) error {
	if err := table.checkIndex(data); err != nil {
		return err
	}
	primaryKey, err := table.primaryKey(data)
	if err != nil {
		return err
	}
	//   auto   ，
	if table.opt.Primary == "auto" {
		table.addRowCache(&Row{Data: data, Primary: primaryKey, Ty: Add})
		return nil
	}
	//       ,
	//TODO:       ，          index
	row, incache, err := table.findRow(primaryKey)
	if err == types.ErrNotFound {
		table.addRowCache(&Row{Data: data, Primary: primaryKey, Ty: Add})
		return nil
	}
	//update or add
	if incache {
		row.Data = data
		return nil
	}
	//
	table.addRowCache(&Row{Data: data, Primary: primaryKey, Ty: Update, old: row.Data})
	return nil
}

//Add
func (table *Table) Add(data types.Message) error {
	if err := table.checkIndex(data); err != nil {
		return err
	}
	primaryKey, err := table.primaryKey(data)
	if err != nil {
		return err
	}
	//find in cache + db
	_, _, err = table.findRow(primaryKey)
	if err != types.ErrNotFound {
		return ErrDupPrimaryKey
	}
	//  cache      ，
	table.addRowCache(&Row{Data: data, Primary: primaryKey, Ty: Add})
	return nil
}

//Update
func (table *Table) Update(primaryKey []byte, newdata types.Message) (err error) {
	if err := table.checkIndex(newdata); err != nil {
		return err
	}
	p1, err := table.getPrimaryFromData(newdata)
	if err != nil {
		return err
	}
	if !bytes.Equal(p1, primaryKey) {
		return types.ErrInvalidParam
	}
	row, incache, err := table.findRow(primaryKey)
	//
	if err != nil {
		return err
	}
	//update and add
	if incache {
		row.Data = newdata
		return nil
	}
	table.addRowCache(&Row{Data: newdata, Primary: primaryKey, Ty: Update, old: row.Data})
	return nil
}

//Del         (      )
func (table *Table) Del(primaryKey []byte) error {
	row, incache, err := table.findRow(primaryKey)
	if err != nil {
		return err
	}
	if incache {
		rowty := row.Ty
		table.delRowCache(row)
		if rowty == Add {
			return nil
		}
	}
	//copy row
	delrow := *row
	delrow.Ty = Del
	table.addRowCache(&delrow)
	return nil
}

//DelRow
func (table *Table) DelRow(data types.Message) error {
	primaryKey, err := table.primaryKey(data)
	if err != nil {
		return err
	}
	return table.Del(primaryKey)
}

//getDataKey data key
func (table *Table) getDataKey(primaryKey []byte) []byte {
	return append([]byte(table.dataprefix), primaryKey...)
}

//GetIndexKey data key
func (table *Table) getIndexKey(indexName string, index, primaryKey []byte) []byte {
	key := table.indexPrefix(indexName)
	key = append(key, index...)
	key = append(key, []byte(sep)...)
	key = append(key, primaryKey...)
	return key
}

func (table *Table) primaryPrefix() []byte {
	return []byte(table.dataprefix)
}

func (table *Table) indexPrefix(indexName string) []byte {
	key := append([]byte(table.metaprefix), []byte(indexName+sep)...)
	return key
}

func (table *Table) index(row *Row, indexName string) ([]byte, error) {
	err := table.meta.SetPayload(row.Data)
	if err != nil {
		return nil, err
	}
	return table.meta.Get(indexName)
}

func (table *Table) getData(primaryKey []byte) ([]byte, error) {
	key := table.getDataKey(primaryKey)
	value, err := table.kvdb.Get(key)
	if err != nil {
		return nil, err
	}
	return value, nil
}

//GetData
func (table *Table) GetData(primaryKey []byte) (*Row, error) {
	value, err := table.getData(primaryKey)
	if err != nil {
		return nil, err
	}
	return table.getRow(value)
}

func (table *Table) getRow(value []byte) (*Row, error) {
	primary, data, err := DecodeRow(value)
	if err != nil {
		return nil, err
	}
	row := table.meta.CreateRow()
	row.Primary = primary
	err = types.Decode(data, row.Data)
	if err != nil {
		return nil, err
	}
	return row, nil
}

//Save
func (table *Table) Save() (kvs []*types.KeyValue, err error) {
	for _, row := range table.rows {
		kvlist, err := table.saveRow(row)
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, kvlist...)
	}
	kvlist, err := table.autoinc.Save()
	if err != nil {
		return nil, err
	}
	kvs = append(kvs, kvlist...)
	//del cache
	table.rowmap = make(map[string]*Row)
	table.rows = nil
	return util.DelDupKey(kvs), nil
}

func pad(i int64) string {
	return fmt.Sprintf("%020d", i)
}

func (table *Table) saveRow(row *Row) (kvs []*types.KeyValue, err error) {
	if row.Ty == Del {
		return table.delRow(row)
	} else if row.Ty == Add {
		return table.addRow(row)
	} else if row.Ty == Update {
		return table.updateRow(row)
	} else if row.Ty == None {
		return nil, nil
	}
	return nil, errors.New("save table unknow action")
}

func (table *Table) delRow(row *Row) (kvs []*types.KeyValue, err error) {
	if !table.opt.Join {
		deldata := &types.KeyValue{Key: table.getDataKey(row.Primary)}
		kvs = append(kvs, deldata)
	}
	for _, index := range table.opt.Index {
		indexkey, err := table.index(row, index)
		if err != nil {
			return nil, err
		}
		delindex := &types.KeyValue{Key: table.getIndexKey(index, indexkey, row.Primary)}
		kvs = append(kvs, delindex)
	}
	return kvs, nil
}

func (table *Table) addRow(row *Row) (kvs []*types.KeyValue, err error) {
	if !table.opt.Join {
		data, err := row.Encode()
		if err != nil {
			return nil, err
		}
		adddata := &types.KeyValue{Key: table.getDataKey(row.Primary), Value: data}
		kvs = append(kvs, adddata)
	}
	for _, index := range table.opt.Index {
		indexkey, err := table.index(row, index)
		if err != nil {
			return nil, err
		}
		addindex := &types.KeyValue{Key: table.getIndexKey(index, indexkey, row.Primary), Value: row.Primary}
		kvs = append(kvs, addindex)
	}
	return kvs, nil
}

func (table *Table) updateRow(row *Row) (kvs []*types.KeyValue, err error) {
	if proto.Equal(row.Data, row.old) {
		return nil, nil
	}
	if !table.opt.Join {
		data, err := row.Encode()
		if err != nil {
			return nil, err
		}
		adddata := &types.KeyValue{Key: table.getDataKey(row.Primary), Value: data}
		kvs = append(kvs, adddata)
	}
	oldrow := &Row{Data: row.old}
	for _, index := range table.opt.Index {
		indexkey, oldkey, ismodify, err := table.getModify(row, oldrow, index)
		if err != nil {
			return nil, err
		}
		if !ismodify {
			continue
		}
		//del old
		delindex := &types.KeyValue{Key: table.getIndexKey(index, oldkey, row.Primary)}
		kvs = append(kvs, delindex)
		//add new
		addindex := &types.KeyValue{Key: table.getIndexKey(index, indexkey, row.Primary), Value: row.Primary}
		kvs = append(kvs, addindex)
	}
	return kvs, nil
}

func (table *Table) getModify(row, oldrow *Row, index string) ([]byte, []byte, bool, error) {
	if oldrow.Data == nil {
		return nil, nil, false, ErrNilValue
	}
	indexkey, err := table.index(row, index)
	if err != nil {
		return nil, nil, false, err
	}
	oldkey, err := table.index(oldrow, index)
	if err != nil {
		return nil, nil, false, err
	}
	if bytes.Equal(indexkey, oldkey) {
		return indexkey, oldkey, false, nil
	}
	return indexkey, oldkey, true, nil
}

//GetQuery       (     kvdb  nil)
func (table *Table) GetQuery(kvdb db.KVDB) *Query {
	if kvdb == nil {
		var ok bool
		kvdb, ok = table.kvdb.(db.KVDB)
		if !ok {
			return nil
		}
	}
	return &Query{table: table, kvdb: kvdb}
}

func (table *Table) getMeta() RowMeta {
	return table.meta
}

//GetMeta   meta
func (table *Table) GetMeta() RowMeta {
	return table.getMeta()
}

func (table *Table) getOpt() *Option {
	return table.opt
}
