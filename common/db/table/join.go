package table

import (
	"errors"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
)

var tablelog = log15.New("module", "db.table")

/*
      :

                 ，                 。

table1:

[gameId, status]

table2:
[txhash, gameId, addr]

        ,

         : join,   : addr & status        key,           :

join_table2_table1:
//table2 primary key
//table1 primary key
//addr_status      index
[txhash, gameId, addr_status]

  join   ：
table2    table1  primary key

    :
table1          : join_table2_table1    addr & status
table2            : join_table2_table1    addr & status

  :

table1     gameId    status ->    join_table2_table1      gameId    addr & status
table2     txhash     addr ->    join_table2_table1       txhash     addr & status

   join_table2_table1

table2         addr & status   index
*/

//JoinTable               ，
//    : LeftTable
//   : RightTable
type JoinTable struct {
	left  *Table
	right *Table
	*Table
	Fk         string
	leftIndex  []string
	rightIndex []string
}

//NewJoinTable     JoinTable
func NewJoinTable(left *Table, right *Table, indexes []string) (*JoinTable, error) {
	if left.kvdb != right.kvdb {
		return nil, errors.New("jointable: kvdb must same")
	}
	if _, ok := left.kvdb.(db.KVDB); !ok {
		return nil, errors.New("jointable: kvdb must be db.KVDB")
	}
	if left.opt.Prefix != right.opt.Prefix {
		return nil, errors.New("jointable: left and right table prefix must same")
	}
	fk := right.opt.Primary
	if !left.canGet(fk) {
		return nil, errors.New("jointable: left must has right primary index")
	}
	join := &JoinTable{left: left, right: right, Fk: fk}
	for _, index := range indexes {
		joinindex := strings.Split(index, joinsep)
		if len(joinindex) != 2 {
			return nil, errors.New("jointable: index config error")
		}
		if joinindex[0] != "" && !left.canGet(joinindex[0]) {
			return nil, errors.New("jointable: left table can not get: " + joinindex[0])
		}
		if joinindex[0] != "" {
			join.leftIndex = append(join.leftIndex, joinindex[0])
		}
		if joinindex[1] == "" || !right.canGet(joinindex[1]) {
			return nil, errors.New("jointable: right table can not get: " + joinindex[1])
		}
		if joinindex[1] != "" {
			join.rightIndex = append(join.rightIndex, joinindex[1])
		}
	}
	opt := &Option{
		Join:    true,
		Prefix:  left.opt.Prefix,
		Name:    left.opt.Name + joinsep + right.opt.Name,
		Primary: left.opt.Primary,
		Index:   indexes,
	}
	mytable, err := NewTable(&JoinMeta{
		left:  left.meta,
		right: right.meta}, left.kvdb, opt)
	if err != nil {
		return nil, err
	}
	join.Table = mytable
	return join, nil
}

//GetLeft get left table
func (join *JoinTable) GetLeft() *Table {
	return join.left
}

//GetRight get right table
func (join *JoinTable) GetRight() *Table {
	return join.right
}

//GetTable get table by name
func (join *JoinTable) GetTable(name string) (*Table, error) {
	if join.left.opt.Name == name {
		return join.left, nil
	}
	if join.right.opt.Name == name {
		return join.right, nil
	}
	return nil, types.ErrNotFound
}

//MustGetTable if name not exist, panic
func (join *JoinTable) MustGetTable(name string) *Table {
	table, err := join.GetTable(name)
	if err != nil {
		panic(err)
	}
	return table
}

//GetData rewrite get data of jointable
func (join *JoinTable) GetData(primaryKey []byte) (*Row, error) {
	leftrow, err := join.left.GetData(primaryKey)
	if err != nil {
		return nil, err
	}
	rightprimary, err := join.left.index(leftrow, join.Fk)
	if err != nil {
		return nil, err
	}
	rightrow, err := join.right.GetData(rightprimary)
	if err != nil {
		return nil, err
	}
	rowjoin := join.meta.CreateRow()
	rowjoin.Ty = None
	rowjoin.Primary = leftrow.Primary
	rowjoin.Data.(*JoinData).Left = leftrow.Data
	rowjoin.Data.(*JoinData).Right = rightrow.Data
	return rowjoin, nil
}

//ListIndex   jointable
func (join *JoinTable) ListIndex(indexName string, prefix []byte, primaryKey []byte, count, direction int32) (rows []*Row, err error) {
	if !strings.Contains(indexName, joinsep) || !join.canGet(indexName) {
		return nil, errors.New("joinable query: indexName must be join index")
	}
	query := &Query{table: join, kvdb: join.left.kvdb.(db.KVDB)}
	return query.ListIndex(indexName, prefix, primaryKey, count, direction)
}

//Save      save   ，    Save left,right table
//  save jointable
//  update    ，  del, add,          update
//  update     del + add
//left modify: del index, add new index (query right by primary) (check in cache)
//right modify: query all primary in left, include in cache, del index, add new index
//TODO:
func (join *JoinTable) Save() (kvs []*types.KeyValue, err error) {
	for _, row := range join.left.rows {
		if row.Ty == None {
			continue
		}
		err := join.saveLeft(row)
		if err != nil {
			return nil, err
		}
	}
	for _, row := range join.right.rows {
		if row.Ty == None {
			continue
		}
		err := join.saveRight(row)
		if err != nil {
			return nil, err
		}
	}
	joinkvs, err := join.Table.Save()
	if err != nil {
		return nil, err
	}
	kvs = append(kvs, joinkvs...)
	leftkvs, err := join.left.Save()
	if err != nil {
		return nil, err
	}
	kvs = append(kvs, leftkvs...)
	rightkvs, err := join.right.Save()
	if err != nil {
		return nil, err
	}
	kvs = append(kvs, rightkvs...)
	return util.DelDupKey(kvs), nil
}

func (join *JoinTable) isLeftModify(row *Row) bool {
	oldrow := &Row{Data: row.old}
	for _, index := range join.leftIndex {
		_, _, ismodify, err := join.left.getModify(row, oldrow, index)
		if ismodify {
			return true
		}
		if err != nil {
			tablelog.Error("isLeftModify", "err", err)
		}
	}
	return false
}

func (join *JoinTable) isRightModify(row *Row) bool {
	oldrow := &Row{Data: row.old}
	for _, index := range join.rightIndex {
		_, _, ismodify, err := join.right.getModify(row, oldrow, index)
		if ismodify {
			return true
		}
		if err != nil {
			tablelog.Error("isLeftModify", "err", err)
		}
	}
	return false
}

func (join *JoinTable) saveLeft(row *Row) error {
	if row.Ty == Update && !join.isLeftModify(row) {
		return nil
	}
	olddata := &JoinData{}
	rowjoin := join.meta.CreateRow()
	rowjoin.Ty = row.Ty
	rowjoin.Primary = row.Primary
	rowjoin.Data.(*JoinData).Left = row.Data
	olddata.Left = row.old
	rightprimary, err := join.left.index(row, join.Fk)
	if err != nil {
		return err
	}
	rightrow, incache, err := join.right.findRow(rightprimary)
	if err != nil {
		return err
	}
	if incache && rightrow.Ty == Update {
		olddata.Right = rightrow.old
	} else {
		olddata.Right = rightrow.Data
	}
	//    left    ,      (  right     ， right   )
	if row.Ty == Update {
		rowjoin.old = olddata
	}
	rowjoin.Data.(*JoinData).Right = rightrow.Data
	join.addRowCache(rowjoin)
	return nil
}

func (join *JoinTable) saveRight(row *Row) error {
	if row.Ty == Update && !join.isRightModify(row) {
		return nil
	}
	indexName := join.right.opt.Primary
	indexValue := row.Primary
	q := join.left.GetQuery(join.left.kvdb.(db.KVDB))
	rows, err := q.ListIndex(indexName, indexValue, nil, 0, db.ListDESC)
	if err != nil && err != types.ErrNotFound {
		return err
	}
	rows, err = join.left.mergeCache(rows, indexName, indexValue)
	if err != nil {
		return err
	}
	for _, onerow := range rows {
		olddata := &JoinData{Right: row.old, Left: onerow.Data}
		if onerow.Ty == Update {
			olddata.Left = onerow.old
		}
		rowjoin := join.meta.CreateRow()
		rowjoin.Ty = row.Ty
		rowjoin.Primary = onerow.Primary
		if row.Ty == Update {
			rowjoin.old = olddata
		}
		rowjoin.Data.(*JoinData).Right = row.Data
		rowjoin.Data.(*JoinData).Left = onerow.Data
		join.addRowCache(rowjoin)
	}
	return nil
}

//JoinData  left   right
type JoinData struct {
	Left  types.Message
	Right types.Message
}

//Reset data
func (msg *JoinData) Reset() {
	msg.Left.Reset()
	msg.Right.Reset()
}

//ProtoMessage data
func (msg *JoinData) ProtoMessage() {
	msg.Left.ProtoMessage()
	msg.Right.ProtoMessage()
}

//String string
func (msg *JoinData) String() string {
	return msg.Left.String() + msg.Right.String()
}

//JoinMeta left right      meta
type JoinMeta struct {
	left  RowMeta
	right RowMeta
	data  *JoinData
}

//CreateRow create a meta struct
func (tx *JoinMeta) CreateRow() *Row {
	return &Row{Data: &JoinData{}}
}

//SetPayload
func (tx *JoinMeta) SetPayload(data types.Message) error {
	if txdata, ok := data.(*JoinData); ok {
		tx.data = txdata
		if tx.data.Left != nil && tx.data.Right != nil {
			err := tx.left.SetPayload(tx.data.Left)
			if err != nil {
				return err
			}
			err = tx.right.SetPayload(tx.data.Right)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return types.ErrTypeAsset
}

//Get   indexName    indexValue
func (tx *JoinMeta) Get(key string) ([]byte, error) {
	indexs := strings.Split(key, joinsep)
	//  primary
	if len(indexs) <= 1 {
		return tx.left.Get(key)
	}
	var leftvalue []byte
	var err error
	if indexs[0] != "" {
		leftvalue, err = tx.left.Get(indexs[0])
		if err != nil {
			return nil, err
		}
	}
	rightvalue, err := tx.right.Get(indexs[1])
	if err != nil {
		return nil, err
	}
	return JoinKey(leftvalue, rightvalue), nil
}

//JoinKey   left   right key      key
func JoinKey(leftvalue, rightvalue []byte) []byte {
	return types.Encode(&types.KeyValue{Key: leftvalue, Value: rightvalue})
}
