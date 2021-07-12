package blockchain

import (
	"fmt"

	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/common/db/table"
	"github.com/D-PlatformOperatingSystem/dpos/common/merkle"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

var (
	chainParaTxPrefix  = []byte("CHAIN-paratx")
	chainBodyPrefix    = []byte("CHAIN-body")
	chainHeaderPrefix  = []byte("CHAIN-header")
	chainReceiptPrefix = []byte("CHAIN-receipt")
)

func calcHeightHashKey(height int64, hash []byte) []byte {
	return append([]byte(fmt.Sprintf("%012d", height)), hash...)
}
func calcHeightParaKey(height int64) []byte {
	return []byte(fmt.Sprintf("%012d", height))
}

func calcHeightTitleKey(height int64, title string) []byte {
	return append([]byte(fmt.Sprintf("%012d", height)), []byte(title)...)
}

/*
table  body
data:  block body
index: hash
*/
var bodyOpt = &table.Option{
	Prefix:  "CHAIN-body",
	Name:    "body",
	Primary: "heighthash",
	Index:   []string{"hash"},
}

//NewBodyTable
func NewBodyTable(kvdb dbm.KV) *table.Table {
	rowmeta := NewBodyRow()
	table, err := table.NewTable(rowmeta, kvdb, bodyOpt)
	if err != nil {
		panic(err)
	}
	return table
}

//BodyRow table meta
type BodyRow struct {
	*types.BlockBody
}

//NewBodyRow     meta
func NewBodyRow() *BodyRow {
	return &BodyRow{BlockBody: &types.BlockBody{}}
}

//CreateRow
func (body *BodyRow) CreateRow() *table.Row {
	return &table.Row{Data: &types.BlockBody{}}
}

//SetPayload
func (body *BodyRow) SetPayload(data types.Message) error {
	if blockbody, ok := data.(*types.BlockBody); ok {
		body.BlockBody = blockbody
		return nil
	}
	return types.ErrTypeAsset
}

//Get        key
func (body *BodyRow) Get(key string) ([]byte, error) {
	if key == "heighthash" {
		return calcHeightHashKey(body.Height, body.Hash), nil
	} else if key == "hash" {
		return body.Hash, nil
	}
	return nil, types.ErrNotFound
}

//saveBlockBodyTable   block body
func saveBlockBodyTable(db dbm.DB, body *types.BlockBody) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewBodyTable(kvdb)

	err := table.Replace(body)
	if err != nil {
		return nil, err
	}

	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

//     index     blockbody
//      ：height+hash；indexName="",prefix=nil,primaryKey=calcHeightHashKey
//  index  ：hash; indexName="hash",prefix=BodyRow.Get(indexName),primaryKey=nil
func getBodyByIndex(db dbm.DB, indexName string, prefix []byte, primaryKey []byte) (*types.BlockBody, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewBodyTable(kvdb)

	rows, err := table.ListIndex(indexName, prefix, primaryKey, 0, dbm.ListASC)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		panic("getBodyByIndex")
	}
	body, ok := rows[0].Data.(*types.BlockBody)
	if !ok {
		return nil, types.ErrDecode
	}
	return body, nil
}

//delBlockBodyTable   block Body
func delBlockBodyTable(db dbm.DB, height int64, hash []byte) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewBodyTable(kvdb)

	err := table.Del(calcHeightHashKey(height, hash))
	if err != nil {
		return nil, err
	}

	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

/*
table  header
data:  block header
primary:heighthash
index: hash
*/
var headerOpt = &table.Option{
	Prefix:  "CHAIN-header",
	Name:    "header",
	Primary: "heighthash",
	Index:   []string{"hash"},
}

//HeaderRow table meta
type HeaderRow struct {
	*types.Header
}

//NewHeaderRow     meta
func NewHeaderRow() *HeaderRow {
	return &HeaderRow{Header: &types.Header{}}
}

//NewHeaderTable
func NewHeaderTable(kvdb dbm.KV) *table.Table {
	rowmeta := NewHeaderRow()
	table, err := table.NewTable(rowmeta, kvdb, headerOpt)
	if err != nil {
		panic(err)
	}
	return table
}

//CreateRow
func (header *HeaderRow) CreateRow() *table.Row {
	return &table.Row{Data: &types.Header{}}
}

//SetPayload
func (header *HeaderRow) SetPayload(data types.Message) error {
	if blockheader, ok := data.(*types.Header); ok {
		header.Header = blockheader
		return nil
	}
	return types.ErrTypeAsset
}

//Get        key
func (header *HeaderRow) Get(key string) ([]byte, error) {
	if key == "heighthash" {
		return calcHeightHashKey(header.Height, header.Hash), nil
	} else if key == "hash" {
		return header.Hash, nil
	}
	return nil, types.ErrNotFound
}

//saveHeaderTable   block header
func saveHeaderTable(db dbm.DB, header *types.Header) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewHeaderTable(kvdb)

	err := table.Replace(header)
	if err != nil {
		return nil, err
	}

	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

//     index     blockheader
//      ：height+hash；indexName="",prefix=nil,primaryKey=calcHeightHashKey
//  index  ：hash; indexName="hash",prefix=HeaderRow.Get(indexName),primaryKey=nil
func getHeaderByIndex(db dbm.DB, indexName string, prefix []byte, primaryKey []byte) (*types.Header, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewHeaderTable(kvdb)

	rows, err := table.ListIndex(indexName, prefix, primaryKey, 0, dbm.ListASC)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		panic("getHeaderByIndex")
	}
	header, ok := rows[0].Data.(*types.Header)
	if !ok {
		return nil, types.ErrDecode
	}
	return header, nil
}

/*
table  paratx
data:  types.HeightPara
primary:heighttitle
index: height,title
*/
var paratxOpt = &table.Option{
	Prefix:  "CHAIN-paratx",
	Name:    "paratx",
	Primary: "heighttitle",
	Index:   []string{"height", "title"},
}

//ParaTxRow table meta
type ParaTxRow struct {
	*types.HeightPara
}

//NewParaTxRow     meta
func NewParaTxRow() *ParaTxRow {
	return &ParaTxRow{HeightPara: &types.HeightPara{}}
}

//NewParaTxTable
func NewParaTxTable(kvdb dbm.KV) *table.Table {
	rowmeta := NewParaTxRow()
	table, err := table.NewTable(rowmeta, kvdb, paratxOpt)
	if err != nil {
		panic(err)
	}
	return table
}

//CreateRow
func (paratx *ParaTxRow) CreateRow() *table.Row {
	return &table.Row{Data: &types.HeightPara{}}
}

//SetPayload
func (paratx *ParaTxRow) SetPayload(data types.Message) error {
	if heightPara, ok := data.(*types.HeightPara); ok {
		paratx.HeightPara = heightPara
		return nil
	}
	return types.ErrTypeAsset
}

//Get        key
func (paratx *ParaTxRow) Get(key string) ([]byte, error) {
	if key == "heighttitle" {
		return calcHeightTitleKey(paratx.Height, paratx.Title), nil
	} else if key == "height" {
		return calcHeightParaKey(paratx.Height), nil
	} else if key == "title" {
		return []byte(paratx.Title), nil
	}
	return nil, types.ErrNotFound
}

//saveParaTxTable
func saveParaTxTable(cfg *types.DplatformOSConfig, db dbm.DB, height int64, hash []byte, txs []*types.Transaction) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewParaTxTable(kvdb)
	if !cfg.IsFork(height, "ForkRootHash") {
		for _, tx := range txs {
			exec := string(tx.Execer)
			if types.IsParaExecName(exec) {
				if title, ok := types.GetParaExecTitleName(exec); ok {
					var paratx = &types.HeightPara{Height: height, Title: title, Hash: hash}
					err := table.Replace(paratx)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	} else {
		//                 roothash
		_, childhashs := merkle.CalcMultiLayerMerkleInfo(cfg, height, txs)
		for i, childhash := range childhashs {
			var paratx = &types.HeightPara{
				Height:         height,
				Hash:           hash,
				Title:          childhash.Title,
				ChildHash:      childhash.ChildHash,
				StartIndex:     childhash.StartIndex,
				ChildHashIndex: uint32(i),
				TxCount:        childhash.GetTxCount(),
			}
			err := table.Replace(paratx)
			if err != nil {
				return nil, err
			}
		}

	}
	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

//delParaTxTable
//
func delParaTxTable(db dbm.DB, height int64) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewParaTxTable(kvdb)
	rows, _ := table.ListIndex("height", calcHeightParaKey(height), nil, 0, dbm.ListASC)
	for _, row := range rows {
		err := table.Del(row.Primary)
		if err != nil {
			return nil, err
		}
	}
	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

//     index     ParaTx
//  height    ：indexName="height",prefix=HeaderRow.Get(indexName)
//  title  ;     indexName="title",prefix=HeaderRow.Get(indexName)
func getParaTxByIndex(db dbm.DB, indexName string, prefix []byte, primaryKey []byte, count, direction int32) (*types.HeightParas, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewParaTxTable(kvdb)

	rows, err := table.ListIndex(indexName, prefix, primaryKey, count, direction)
	if err != nil {
		return nil, err
	}
	var rep types.HeightParas
	for _, row := range rows {
		r, ok := row.Data.(*types.HeightPara)
		if !ok {
			return nil, types.ErrDecode
		}
		rep.Items = append(rep.Items, r)
	}
	return &rep, nil
}

/*
table  receipt
data:  block receipt
index: hash
*/
var receiptOpt = &table.Option{
	Prefix:  "CHAIN-receipt",
	Name:    "receipt",
	Primary: "heighthash",
	Index:   []string{"hash"},
}

//NewReceiptTable
func NewReceiptTable(kvdb dbm.KV) *table.Table {
	rowmeta := NewReceiptRow()
	table, err := table.NewTable(rowmeta, kvdb, receiptOpt)
	if err != nil {
		panic(err)
	}
	return table
}

//ReceiptRow table meta
type ReceiptRow struct {
	*types.BlockReceipt
}

//NewReceiptRow     meta
func NewReceiptRow() *ReceiptRow {
	return &ReceiptRow{BlockReceipt: &types.BlockReceipt{}}
}

//CreateRow
func (recpt *ReceiptRow) CreateRow() *table.Row {
	return &table.Row{Data: &types.BlockReceipt{}}
}

//SetPayload
func (recpt *ReceiptRow) SetPayload(data types.Message) error {
	if blockReceipt, ok := data.(*types.BlockReceipt); ok {
		recpt.BlockReceipt = blockReceipt
		return nil
	}
	return types.ErrTypeAsset
}

//Get        key
func (recpt *ReceiptRow) Get(key string) ([]byte, error) {
	if key == "heighthash" {
		return calcHeightHashKey(recpt.Height, recpt.Hash), nil
	} else if key == "hash" {
		return recpt.Hash, nil
	}
	return nil, types.ErrNotFound
}

//saveBlockReceiptTable   block receipt
func saveBlockReceiptTable(db dbm.DB, recpt *types.BlockReceipt) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewReceiptTable(kvdb)

	err := table.Replace(recpt)
	if err != nil {
		return nil, err
	}

	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

//     index     blockReceipt
//      ：height+hash；indexName="",prefix=nil,primaryKey=calcHeightHashKey
//  index  ：hash; indexName="hash",prefix=ReceiptRow.Get(indexName),primaryKey=nil
func getReceiptByIndex(db dbm.DB, indexName string, prefix []byte, primaryKey []byte) (*types.BlockReceipt, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewReceiptTable(kvdb)

	rows, err := table.ListIndex(indexName, prefix, primaryKey, 0, dbm.ListASC)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		panic("getReceiptByIndex")
	}
	recpt, ok := rows[0].Data.(*types.BlockReceipt)
	if !ok {
		return nil, types.ErrDecode
	}
	return recpt, nil
}

//delBlockReceiptTable   block receipt
func delBlockReceiptTable(db dbm.DB, height int64, hash []byte) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewReceiptTable(kvdb)

	err := table.Del(calcHeightHashKey(height, hash))
	if err != nil {
		return nil, err
	}

	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}
