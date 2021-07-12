// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/common/difficulty"
	"github.com/D-PlatformOperatingSystem/dpos/common/utils"
	"github.com/D-PlatformOperatingSystem/dpos/common/version"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/golang/protobuf/proto"
)

//var
var (
	blockLastHeight       = []byte("blockLastHeight")
	bodyPrefix            = []byte("Body:")
	LastSequence          = []byte("LastSequence")
	headerPrefix          = []byte("Header:")
	heightToHeaderPrefix  = []byte("HH:")
	hashPrefix            = []byte("Hash:")
	tdPrefix              = []byte("TD:")
	heightToHashKeyPrefix = []byte("Height:")
	seqToHashKey          = []byte("Seq:")
	HashToSeqPrefix       = []byte("HashToSeq:")
	pushPrefix            = []byte("push2subscribe:")
	lastSeqNumPrefix      = []byte("lastSeqNumPrefix:")
	paraSeqToHashKey      = []byte("ParaSeq:")
	HashToParaSeqPrefix   = []byte("HashToParaSeq:")
	LastParaSequence      = []byte("LastParaSequence")
	// chunk
	BodyHashToChunk    = []byte("BodyHashToChunk:")
	ChunkNumToHash     = []byte("ChunkNumToHash:")
	ChunkHashToNum     = []byte("ChunkHashToNum:")
	RecvChunkNumToHash = []byte("RecvChunkNumToHash:")
	MaxSerialChunkNum  = []byte("MaxSilChunkNum:")
	ToDeleteChunkSign  = []byte("ToDelChunkSign:")
	storeLog           = chainlog.New("submodule", "store")
)

//GetLocalDBKeyList
//  chainBody chainHeader     key。chainParaTx      0
//bodyPrefix，headerPrefix，heightToHeaderPrefix   key
func GetLocalDBKeyList() [][]byte {
	return [][]byte{
		blockLastHeight, bodyPrefix, LastSequence, headerPrefix, heightToHeaderPrefix,
		hashPrefix, tdPrefix, heightToHashKeyPrefix, seqToHashKey, HashToSeqPrefix,
		pushPrefix, lastSeqNumPrefix, tempBlockKey, lastTempBlockKey, LastParaSequence,
		chainParaTxPrefix, chainBodyPrefix, chainHeaderPrefix, chainReceiptPrefix,
		BodyHashToChunk, ChunkNumToHash, ChunkHashToNum, RecvChunkNumToHash,
		MaxSerialChunkNum, ToDeleteChunkSign,
	}
}

//  block hash   blockbody
func calcHashToBlockBodyKey(hash []byte) []byte {
	return append(bodyPrefix, hash...)
}

func calcPushKey(name string) []byte {
	return []byte(string(pushPrefix) + name)
}

func calcLastPushSeqNumKey(name string) []byte {
	return []byte(string(lastSeqNumPrefix) + name)
}

//  block hash   header
func calcHashToBlockHeaderKey(hash []byte) []byte {
	return append(headerPrefix, hash...)
}

func calcHeightToBlockHeaderKey(height int64) []byte {
	return append(heightToHeaderPrefix, []byte(fmt.Sprintf("%012d", height))...)
}

//  block hash   block height
func calcHashToHeightKey(hash []byte) []byte {
	return append(hashPrefix, hash...)
}

//  block hash   block   TD
func calcHashToTdKey(hash []byte) []byte {
	return append(tdPrefix, hash...)
}

//  block height    block  hash
func calcHeightToHashKey(height int64) []byte {
	return append(heightToHashKeyPrefix, []byte(fmt.Sprintf("%v", height))...)
}

//  block        block hash,KEY=Seq:sequence
func calcSequenceToHashKey(sequence int64, isPara bool) []byte {
	if isPara {
		return append(paraSeqToHashKey, []byte(fmt.Sprintf("%v", sequence))...)
	}
	return append(seqToHashKey, []byte(fmt.Sprintf("%v", sequence))...)
}

//  block hash   seq   ，KEY=Seq:sequence，      addblock  ，  delblock       seq hash
func calcHashToSequenceKey(hash []byte, isPara bool) []byte {
	if isPara {
		return append(HashToParaSeqPrefix, hash...)
	}
	return append(HashToSeqPrefix, hash...)
}

func calcLastSeqKey(isPara bool) []byte {
	if isPara {
		return LastParaSequence
	}
	return LastSequence
}

//  block hash   seq   ，KEY=Seq:sequence，      addblock  ，  delblock       seq hash
func calcHashToMainSequenceKey(hash []byte) []byte {
	return append(HashToSeqPrefix, hash...)
}

//  block        block hash,KEY=MainSeq:sequence
func calcMainSequenceToHashKey(sequence int64) []byte {
	return append(seqToHashKey, []byte(fmt.Sprintf("%v", sequence))...)
}

//        blockhash--->chunkhash
func calcBlockHashToChunkHash(hash []byte) []byte {
	return append(BodyHashToChunk, hash...)
}

//        chunkNum--->chunkhash
func calcChunkNumToHash(chunkNum int64) []byte {
	return append(ChunkNumToHash, []byte(fmt.Sprintf("%012d", chunkNum))...)
}

//        chunkhash--->chunkNum
func calcChunkHashToNum(hash []byte) []byte {
	return append(ChunkHashToNum, hash...)
}

//        chunkNum--->chunkhash
func calcRecvChunkNumToHash(chunkNum int64) []byte {
	return append(RecvChunkNumToHash, []byte(fmt.Sprintf("%012d", chunkNum))...)
}

//    chunkNum--->MaxPeerHeight        chunk            ，
func calcToDeleteChunkSign(chunkNum int64) []byte {
	return append(ToDeleteChunkSign, []byte(fmt.Sprintf("%012d", chunkNum))...)
}

//BlockStore
type BlockStore struct {
	db             dbm.DB
	client         queue.Client
	height         int64
	lastBlock      *types.Block
	lastheaderlock sync.Mutex
	saveSequence   bool
	isParaChain    bool
	batch          dbm.Batch

	//       block，
	activeBlocks *utils.SpaceLimitCache
}

//NewBlockStore new
func NewBlockStore(chain *BlockChain, db dbm.DB, client queue.Client) *BlockStore {
	height, err := LoadBlockStoreHeight(db)
	if err != nil {
		chainlog.Info("init::LoadBlockStoreHeight::database may be crash", "err", err.Error())
		if err != types.ErrHeightNotExist {
			panic(err)
		}
	}
	blockStore := &BlockStore{
		height: height,
		db:     db,
		client: client,
	}
	if chain != nil {
		blockStore.saveSequence = chain.isRecordBlockSequence
		blockStore.isParaChain = chain.isParaChain
	}
	cfg := chain.client.GetConfig()
	if height == -1 {
		chainlog.Info("load block height error, may be init database", "height", height)
		if cfg.IsEnable("quickIndex") {
			blockStore.saveQuickIndexFlag()
		}
	} else {
		blockdetail, err := blockStore.LoadBlockByHeight(height)
		if err != nil {
			chainlog.Error("init::LoadBlockByHeight::database may be crash")
			panic(err)
		}
		blockStore.lastBlock = blockdetail.GetBlock()
		flag, err := blockStore.loadFlag(types.FlagTxQuickIndex)
		if err != nil {
			panic(err)
		}
		if cfg.IsEnable("quickIndex") {
			if flag == 0 {
				blockStore.initQuickIndex(height)
			}
		} else {
			if flag != 0 {
				panic("toml config disable tx quick index, but database enable quick index")
			}
		}
	}
	blockStore.batch = db.NewBatch(true)

	//
	maxActiveBlockNum := maxActiveBlocks
	maxActiveBlockSize := maxActiveBlocksCacheSize
	if cfg.GetModuleConfig().BlockChain.MaxActiveBlockNum > 0 {
		maxActiveBlockNum = cfg.GetModuleConfig().BlockChain.MaxActiveBlockNum
	}
	if cfg.GetModuleConfig().BlockChain.MaxActiveBlockSize > 0 {
		maxActiveBlockSize = cfg.GetModuleConfig().BlockChain.MaxActiveBlockSize
	}
	blockStore.activeBlocks = utils.NewSpaceLimitCache(maxActiveBlockNum, maxActiveBlockSize*1024*1024)

	return blockStore
}

//  :
//           quickIndex
//    ，
//1.   hash      TX:hash
//2.      Tx:hash      8   index
//3. 2000       ，
//4.        ,  quickIndex
func (bs *BlockStore) initQuickIndex(height int64) {
	storeLog.Info("quickIndex upgrade start", "current height", height)
	batch := bs.db.NewBatch(true)
	var maxsize = 100 * 1024 * 1024
	var count = 0
	cfg := bs.client.GetConfig()
	for i := int64(0); i <= height; i++ {
		blockdetail, err := bs.LoadBlockByHeight(i)
		if err != nil {
			panic(err)
		}
		for _, tx := range blockdetail.Block.Txs {
			hash := tx.Hash()
			txresult, err := bs.db.Get(hash)
			if err != nil {
				panic(err)
			}
			count += len(txresult)
			batch.Set(cfg.CalcTxKey(hash), txresult)
			batch.Set(types.CalcTxShortKey(hash), []byte("1"))
		}
		if count > maxsize {
			storeLog.Info("initQuickIndex", "height", i)
			err := batch.Write()
			if err != nil {
				panic(err)
			}
			batch.Reset()
			count = 0
		}
	}
	if count > 0 {
		err := batch.Write()
		if err != nil {
			panic(err)
		}
		storeLog.Info("initQuickIndex", "height", height)
		batch.Reset()
	}
	bs.saveQuickIndexFlag()
}

// store    :  block       ，
//   store

// SetSync store
func (bs *BlockStore) SetSync(key, value []byte) error {
	return bs.db.SetSync(key, value)
}

// Set store
func (bs *BlockStore) Set(key, value []byte) error {
	return bs.db.Set(key, value)
}

// GetKey store    ， Get
func (bs *BlockStore) GetKey(key []byte) ([]byte, error) {
	value, err := bs.db.Get(key)
	if err != nil && err != dbm.ErrNotFoundInDb {
		return nil, types.ErrNotFound

	}
	return value, err
}

// PrefixCount store
func (bs *BlockStore) PrefixCount(prefix []byte) int64 {
	counts := dbm.NewListHelper(bs.db).PrefixCount(prefix)
	return counts
}

// List store
func (bs *BlockStore) List(prefix []byte) ([][]byte, error) {
	values := dbm.NewListHelper(bs.db).PrefixScan(prefix)
	if values == nil {
		return nil, types.ErrNotFound
	}
	return values, nil
}

func (bs *BlockStore) delAllKeys() {
	var allkeys [][]byte
	allkeys = append(allkeys, GetLocalDBKeyList()...)
	allkeys = append(allkeys, version.GetLocalDBKeyList()...)
	allkeys = append(allkeys, types.GetLocalDBKeyList()...)
	var lastkey []byte
	isvalid := true
	for isvalid {
		lastkey, isvalid = bs.delKeys(lastkey, allkeys)
	}
}

func (bs *BlockStore) delKeys(seek []byte, allkeys [][]byte) ([]byte, bool) {
	it := bs.db.Iterator(seek, types.EmptyValue, false)
	defer it.Close()
	i := 0
	count := 0
	var lastkey []byte
	for it.Rewind(); it.Valid(); it.Next() {
		key := it.Key()
		lastkey = key
		if it.Error() != nil {
			panic(it.Error())
		}
		has := false
		for _, prefix := range allkeys {
			if bytes.HasPrefix(key, prefix) {
				has = true
				break
			}
		}
		if !has {
			i++
			if i > 0 && i%10000 == 0 {
				chainlog.Info("del key count", "count", i)
			}
			err := bs.db.Delete(key)
			if err != nil {
				panic(err)
			}
		}
		count++
		if count == 1000000 {
			break
		}
	}
	return lastkey, it.Valid()
}

func (bs *BlockStore) saveQuickIndexFlag() {
	kv := types.FlagKV(types.FlagTxQuickIndex, 1)
	err := bs.db.Set(kv.Key, kv.Value)
	if err != nil {
		panic(err)
	}
}

func (bs *BlockStore) loadFlag(key []byte) (int64, error) {
	flag := &types.Int64{}
	flagBytes, err := bs.db.Get(key)
	if err == nil {
		err = types.Decode(flagBytes, flag)
		if err != nil {
			return 0, err
		}
		return flag.GetData(), nil
	} else if err == types.ErrNotFound || err == dbm.ErrNotFoundInDb {
		return 0, nil
	}
	return 0, err
}

//HasTx
func (bs *BlockStore) HasTx(key []byte) (bool, error) {
	cfg := bs.client.GetConfig()
	if cfg.IsEnable("quickIndex") {
		if _, err := bs.db.Get(types.CalcTxShortKey(key)); err != nil {
			if err == dbm.ErrNotFoundInDb {
				return false, nil
			}
			return false, err
		}
		//   hash       ，      hash      。
		//   hash  ，  hash
		//return true, nil
	}
	if _, err := bs.db.Get(cfg.CalcTxKey(key)); err != nil {
		if err == dbm.ErrNotFoundInDb {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

//Height   BlockStore     block
func (bs *BlockStore) Height() int64 {
	return atomic.LoadInt64(&bs.height)
}

//UpdateHeight   db  block   BlockStore.Height
func (bs *BlockStore) UpdateHeight() {
	height, err := LoadBlockStoreHeight(bs.db)
	if err != nil && err != types.ErrHeightNotExist {
		storeLog.Error("UpdateHeight", "LoadBlockStoreHeight err", err)
		return
	}
	atomic.StoreInt64(&bs.height, height)
	storeLog.Debug("UpdateHeight", "curblockheight", height)
}

//UpdateHeight2      block   BlockStore.Height
func (bs *BlockStore) UpdateHeight2(height int64) {
	atomic.StoreInt64(&bs.height, height)
	storeLog.Debug("UpdateHeight2", "curblockheight", height)
}

//LastHeader   BlockStore     blockheader
func (bs *BlockStore) LastHeader() *types.Header {
	bs.lastheaderlock.Lock()
	defer bs.lastheaderlock.Unlock()

	//   lastBlock  lastheader
	var blockheader = types.Header{}
	if bs.lastBlock != nil {
		blockheader.Version = bs.lastBlock.Version
		blockheader.ParentHash = bs.lastBlock.ParentHash
		blockheader.TxHash = bs.lastBlock.TxHash
		blockheader.StateHash = bs.lastBlock.StateHash
		blockheader.Height = bs.lastBlock.Height
		blockheader.BlockTime = bs.lastBlock.BlockTime
		blockheader.Signature = bs.lastBlock.Signature
		blockheader.Difficulty = bs.lastBlock.Difficulty

		blockheader.Hash = bs.lastBlock.Hash(bs.client.GetConfig())
		blockheader.TxCount = int64(len(bs.lastBlock.Txs))
	}
	return &blockheader
}

//UpdateLastBlock   LastBlock
func (bs *BlockStore) UpdateLastBlock(hash []byte) {
	blockdetail, err := bs.LoadBlockByHash(hash)
	if err != nil {
		storeLog.Error("UpdateLastBlock", "hash", common.ToHex(hash), "error", err)
		return
	}
	bs.lastheaderlock.Lock()
	defer bs.lastheaderlock.Unlock()
	if blockdetail != nil {
		bs.lastBlock = blockdetail.Block
	}
	storeLog.Debug("UpdateLastBlock", "UpdateLastBlock", blockdetail.Block.Height, "LastHederhash", common.ToHex(blockdetail.Block.Hash(bs.client.GetConfig())))
}

//UpdateLastBlock2   LastBlock
func (bs *BlockStore) UpdateLastBlock2(block *types.Block) {
	bs.lastheaderlock.Lock()
	defer bs.lastheaderlock.Unlock()
	bs.lastBlock = block
	storeLog.Debug("UpdateLastBlock", "UpdateLastBlock", block.Height, "LastHederhash", common.ToHex(block.Hash(bs.client.GetConfig())))
}

//LastBlock      block
func (bs *BlockStore) LastBlock() *types.Block {
	bs.lastheaderlock.Lock()
	defer bs.lastheaderlock.Unlock()
	if bs.lastBlock != nil {
		return bs.lastBlock
	}
	return nil
}

//Get get
func (bs *BlockStore) Get(keys *types.LocalDBGet) *types.LocalReplyValue {
	var reply types.LocalReplyValue
	for i := 0; i < len(keys.Keys); i++ {
		key := keys.Keys[i]
		value, err := bs.db.Get(key)
		if err != nil && err != types.ErrNotFound {
			storeLog.Error("Get", "error", err)
		}
		reply.Values = append(reply.Values, value)
	}
	return &reply
}

//LoadBlockByHeight   height    BlockDetail
//    height+hash     header body
//            block
//            localdb        ，
//        height
//            loadBlockByIndex  block
func (bs *BlockStore) LoadBlockByHeight(height int64) (*types.BlockDetail, error) {
	hash, err := bs.GetBlockHashByHeight(height)
	if err != nil {
		return nil, err
	}

	block, err := bs.loadBlockByIndex("", calcHeightHashKey(height, hash), nil)
	if block == nil && err != nil {
		return bs.loadBlockByHashOld(hash)
	}
	return block, nil
}

//LoadBlockByHash   hash  BlockDetail
func (bs *BlockStore) LoadBlockByHash(hash []byte) (*types.BlockDetail, error) {
	block, _, err := bs.loadBlockByHash(hash)
	return block, err
}

func (bs *BlockStore) loadBlockByHash(hash []byte) (*types.BlockDetail, int, error) {
	var blockSize int

	//  hash    blockdetail
	blockdetail, err := bs.loadBlockByIndex("hash", hash, nil)
	if err != nil {
		storeLog.Error("loadBlockByHash:loadBlockByIndex", "hash", common.ToHex(hash), "err", err)
		return nil, blockSize, err
	}
	blockSize = blockdetail.Size()
	return blockdetail, blockSize, nil
}

//SaveBlock     blocks   db    ,      sequence
func (bs *BlockStore) SaveBlock(storeBatch dbm.Batch, blockdetail *types.BlockDetail, sequence int64) (int64, error) {
	cfg := bs.client.GetConfig()
	var lastSequence int64 = -1
	height := blockdetail.Block.Height
	if len(blockdetail.Receipts) == 0 && len(blockdetail.Block.Txs) != 0 {
		storeLog.Error("SaveBlock Receipts is nil ", "height", height)
	}
	hash := blockdetail.Block.Hash(cfg)
	//    body header
	err := bs.saveBlockForTable(storeBatch, blockdetail, true, true)
	if err != nil {
		storeLog.Error("SaveBlock:saveBlockForTable", "height", height, "hash", common.ToHex(hash), "error", err)
		return lastSequence, err
	}
	//     block
	heightbytes := types.Encode(&types.Int64{Data: height})
	storeBatch.Set(blockLastHeight, heightbytes)

	//  block hash height     ，    hash  block
	storeBatch.Set(calcHashToHeightKey(hash), heightbytes)

	//  block height block hash     ，    height  block
	storeBatch.Set(calcHeightToHashKey(height), hash)

	if bs.saveSequence || bs.isParaChain {
		//    block     type add
		lastSequence, err = bs.saveBlockSequence(storeBatch, hash, height, types.AddBlock, sequence)
		if err != nil {
			storeLog.Error("SaveBlock SaveBlockSequence", "height", height, "hash", common.ToHex(hash), "error", err)
			return lastSequence, err
		}
	}
	storeLog.Debug("SaveBlock success", "blockheight", height, "hash", common.ToHex(hash))

	return lastSequence, nil
}

//BlockdetailToBlockBody get block detail
func (bs *BlockStore) BlockdetailToBlockBody(blockdetail *types.BlockDetail) *types.BlockBody {
	cfg := bs.client.GetConfig()
	height := blockdetail.Block.Height
	hash := blockdetail.Block.Hash(cfg)
	var blockbody types.BlockBody
	blockbody.Txs = blockdetail.Block.Txs
	blockbody.Receipts = blockdetail.Receipts
	blockbody.MainHash = hash
	blockbody.MainHeight = height
	blockbody.Hash = hash
	blockbody.Height = height
	if bs.isParaChain {
		blockbody.MainHash = blockdetail.Block.MainHash
		blockbody.MainHeight = blockdetail.Block.MainHeight
	}
	return &blockbody
}

//DelBlock   block   db
func (bs *BlockStore) DelBlock(storeBatch dbm.Batch, blockdetail *types.BlockDetail, sequence int64) (int64, error) {
	var lastSequence int64 = -1
	height := blockdetail.Block.Height
	hash := blockdetail.Block.Hash(bs.client.GetConfig())

	//     block
	bytes := types.Encode(&types.Int64{Data: height - 1})
	storeBatch.Set(blockLastHeight, bytes)

	//  block hash height
	storeBatch.Delete(calcHashToHeightKey(hash))

	//  block height block hash     ，    height  block
	storeBatch.Delete(calcHeightToHashKey(height))

	if bs.saveSequence || bs.isParaChain {
		//    block     type del
		lastSequence, err := bs.saveBlockSequence(storeBatch, hash, height, types.DelBlock, sequence)
		if err != nil {
			storeLog.Error("DelBlock SaveBlockSequence", "height", height, "hash", common.ToHex(hash), "error", err)
			return lastSequence, err
		}
	}
	//                 ，
	if !bs.isParaChain {
		parakvs, _ := delParaTxTable(bs.db, height)
		for _, kv := range parakvs {
			if len(kv.GetKey()) != 0 && kv.GetValue() == nil {
				storeBatch.Delete(kv.GetKey())
			}
		}
	}
	storeLog.Debug("DelBlock success", "blockheight", height, "hash", common.ToHex(hash))
	return lastSequence, nil
}

//GetTx   tx hash  db      tx
func (bs *BlockStore) GetTx(hash []byte) (*types.TxResult, error) {
	if len(hash) == 0 {
		err := errors.New("input hash is null")
		return nil, err
	}
	cfg := bs.client.GetConfig()
	rawBytes, err := bs.db.Get(cfg.CalcTxKey(hash))
	if rawBytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetTx", "hash", common.ToHex(hash), "err", err)
		}
		err = errors.New("tx not exist")
		return nil, err
	}

	var txResult types.TxResult
	err = proto.Unmarshal(rawBytes, &txResult)
	if err != nil {
		return nil, err
	}
	return bs.getRealTxResult(&txResult), nil
}

func (bs *BlockStore) getRealTxResult(txr *types.TxResult) *types.TxResult {
	cfg := bs.client.GetConfig()
	if !cfg.IsEnable("reduceLocaldb") {
		return txr
	}

	var blockinfo *types.BlockDetail
	var err error
	var exist bool

	//             ，
	hash, err := bs.GetBlockHashByHeight(txr.Height)
	if err != nil {
		chainlog.Error("getRealTxResult GetBlockHashByHeight", "height", txr.Height, "error", err)
		return txr
	}

	blockinfo, exist = bs.GetActiveBlock(string(hash))
	if !exist {
		//        localdb     block   tx      receipt
		blockinfo, err = bs.LoadBlockByHash(hash)
		if err != nil {
			chainlog.Error("getRealTxResult LoadBlockByHash", "height", txr.Height, "hash", common.ToHex(hash), "error", err)
			return txr
		}

		//
		bs.AddActiveBlock(string(hash), blockinfo)
	}

	if int(txr.Index) < len(blockinfo.Block.Txs) {
		txr.Tx = blockinfo.Block.Txs[txr.Index]
	}
	if int(txr.Index) < len(blockinfo.Receipts) {
		txr.Receiptdate = blockinfo.Receipts[txr.Index]
	}

	return txr
}

//AddTxs       tx   db
func (bs *BlockStore) AddTxs(storeBatch dbm.Batch, blockDetail *types.BlockDetail) error {
	kv, err := bs.getLocalKV(blockDetail)
	if err != nil {
		storeLog.Error("indexTxs getLocalKV err", "Height", blockDetail.Block.Height, "err", err)
		return err
	}
	//storelog.Info("add txs kv num", "n", len(kv.KV))
	for i := 0; i < len(kv.KV); i++ {
		if kv.KV[i].Value == nil {
			storeBatch.Delete(kv.KV[i].Key)
		} else {
			storeBatch.Set(kv.KV[i].Key, kv.KV[i].Value)
		}
	}
	return nil
}

//DelTxs       tx   db
func (bs *BlockStore) DelTxs(storeBatch dbm.Batch, blockDetail *types.BlockDetail) error {
	//  key:addr:flag:height ,value:txhash
	//flag :0-->from,1--> to
	//height=height*10000+index
	kv, err := bs.getDelLocalKV(blockDetail)
	if err != nil {
		storeLog.Error("indexTxs getLocalKV err", "Height", blockDetail.Block.Height, "err", err)
		return err
	}
	for i := 0; i < len(kv.KV); i++ {
		if kv.KV[i].Value == nil {
			storeBatch.Delete(kv.KV[i].Key)
		} else {
			storeBatch.Set(kv.KV[i].Key, kv.KV[i].Value)
		}
	}

	return nil
}

//GetHeightByBlockHash  db        hash   block
func (bs *BlockStore) GetHeightByBlockHash(hash []byte) (int64, error) {

	heightbytes, err := bs.db.Get(calcHashToHeightKey(hash))
	if heightbytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetHeightByBlockHash", "error", err)
		}
		return -1, types.ErrHashNotExist
	}
	return decodeHeight(heightbytes)
}

func decodeHeight(heightbytes []byte) (int64, error) {
	var height types.Int64
	err := types.Decode(heightbytes, &height)
	if err != nil {
		//may be old database format json...
		err = json.Unmarshal(heightbytes, &height.Data)
		if err != nil {
			storeLog.Error("GetHeightByBlockHash Could not unmarshal height bytes", "error", err)
			return -1, types.ErrUnmarshal
		}
	}
	return height.Data, nil
}

//GetBlockHashByHeight  db        height   blockhash
func (bs *BlockStore) GetBlockHashByHeight(height int64) ([]byte, error) {

	hash, err := bs.db.Get(calcHeightToHashKey(height))
	if hash == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetBlockHashByHeight", "error", err)
		}
		return nil, types.ErrHeightNotExist
	}
	return hash, nil
}

//GetBlockHeaderByHeight   blockheight  blockheader
//        ，            ，
func (bs *BlockStore) GetBlockHeaderByHeight(height int64) (*types.Header, error) {
	return bs.loadHeaderByIndex(height)
}

//GetBlockHeaderByHash   blockhash  blockheader
func (bs *BlockStore) GetBlockHeaderByHash(hash []byte) (*types.Header, error) {
	//  table     ，hash   table
	header, err := getHeaderByIndex(bs.db, "hash", hash, nil)
	if header == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetBlockHerderByHash:getHeaderByIndex ", "err", err)
		}
		return nil, types.ErrHashNotExist
	}
	return header, nil
}

func (bs *BlockStore) getLocalKV(detail *types.BlockDetail) (*types.LocalDBSet, error) {
	if bs.client == nil {
		panic("client not bind message queue.")
	}
	msg := bs.client.NewMessage("execs", types.EventAddBlock, detail)
	err := bs.client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := bs.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	kv := resp.GetData().(*types.LocalDBSet)
	return kv, nil
}

func (bs *BlockStore) getDelLocalKV(detail *types.BlockDetail) (*types.LocalDBSet, error) {
	if bs.client == nil {
		panic("client not bind message queue.")
	}
	msg := bs.client.NewMessage("execs", types.EventDelBlock, detail)
	err := bs.client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := bs.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	localDBSet := resp.GetData().(*types.LocalDBSet)
	return localDBSet, nil
}

//GetTdByBlockHash  db        blockhash   block   td
func (bs *BlockStore) GetTdByBlockHash(hash []byte) (*big.Int, error) {

	blocktd, err := bs.db.Get(calcHashToTdKey(hash))
	if blocktd == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetTdByBlockHash ", "error", err)
		}
		return nil, types.ErrHashNotExist
	}
	td := new(big.Int)
	return td.SetBytes(blocktd), nil
}

//SaveTdByBlockHash   block hash       db
func (bs *BlockStore) SaveTdByBlockHash(storeBatch dbm.Batch, hash []byte, td *big.Int) error {
	if td == nil {
		return types.ErrInvalidParam
	}

	storeBatch.Set(calcHashToTdKey(hash), td.Bytes())
	return nil
}

//NewBatch new
func (bs *BlockStore) NewBatch(sync bool) dbm.Batch {
	storeBatch := bs.db.NewBatch(sync)
	return storeBatch
}

//LoadBlockStoreHeight
func LoadBlockStoreHeight(db dbm.DB) (int64, error) {
	bytes, err := db.Get(blockLastHeight)
	if bytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("LoadBlockStoreHeight", "error", err)
		}
		return -1, types.ErrHeightNotExist
	}
	return decodeHeight(bytes)
}

//     block      db ，           。     chain        block
func (bs *BlockStore) dbMaybeStoreBlock(blockdetail *types.BlockDetail, sync bool) error {
	if blockdetail == nil {
		return types.ErrInvalidParam
	}
	height := blockdetail.Block.GetHeight()
	hash := blockdetail.Block.Hash(bs.client.GetConfig())
	storeBatch := bs.batch
	storeBatch.Reset()
	storeBatch.UpdateWriteSync(sync)
	//Save block header body  table
	err := bs.saveBlockForTable(storeBatch, blockdetail, false, true)
	if err != nil {
		chainlog.Error("dbMaybeStoreBlock:saveBlockForTable", "height", height, "hash", common.ToHex(hash), "err", err)
		return err
	}
	//  block     db
	parentHash := blockdetail.Block.ParentHash

	//        big.int
	difficulty := difficulty.CalcWork(blockdetail.Block.Difficulty)

	var blocktd *big.Int
	if height == 0 {
		blocktd = difficulty
	} else {
		parenttd, err := bs.GetTdByBlockHash(parentHash)
		if err != nil {
			chainlog.Error("dbMaybeStoreBlock GetTdByBlockHash", "height", height, "parentHash", common.ToHex(parentHash))
			return err
		}
		blocktd = new(big.Int).Add(difficulty, parenttd)
	}

	err = bs.SaveTdByBlockHash(storeBatch, blockdetail.Block.Hash(bs.client.GetConfig()), blocktd)
	if err != nil {
		chainlog.Error("dbMaybeStoreBlock SaveTdByBlockHash:", "height", height, "hash", common.ToHex(hash), "err", err)
		return err
	}

	err = storeBatch.Write()
	if err != nil {
		chainlog.Error("dbMaybeStoreBlock storeBatch.Write:", "err", err)
		panic(err)
	}
	return nil
}

//LoadBlockLastSequence        block
func (bs *BlockStore) LoadBlockLastSequence() (int64, error) {
	lastKey := calcLastSeqKey(bs.isParaChain)
	bytes, err := bs.db.Get(lastKey)
	if bytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("LoadBlockLastSequence", "error", err)
		}
		return -1, types.ErrHeightNotExist
	}
	return decodeHeight(bytes)
}

//LoadBlockLastMainSequence        block
func (bs *BlockStore) LoadBlockLastMainSequence() (int64, error) {
	bytes, err := bs.db.Get(LastSequence)
	if bytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("LoadBlockLastMainSequence", "error", err)
		}
		return -1, types.ErrHeightNotExist
	}
	return decodeHeight(bytes)
}

//SaveBlockSequence   block          blockchain
//        ，      1   block hash ，   isRecordBlockSequence
//     isParaChain ，        sequence
//
// seq->hash, hash->seq, last_seq
//
func (bs *BlockStore) saveBlockSequence(storeBatch dbm.Batch, hash []byte, height int64, Type int64, sequence int64) (int64, error) {
	var newSequence int64
	if bs.saveSequence {
		Sequence, err := bs.LoadBlockLastSequence()
		if err != nil {
			storeLog.Error("SaveBlockSequence", "LoadBlockLastSequence err", err)
			if err != types.ErrHeightNotExist {
				panic(err)
			}
		}

		newSequence = Sequence + 1
		//  isRecordBlockSequence     0      ，     0
		if newSequence == 0 && height != 0 {
			storeLog.Error("isRecordBlockSequence is true must Synchronizing data from zero block", "height", height, "seq", newSequence)
			panic(errors.New("isRecordBlockSequence is true must Synchronizing data from zero block"))
		}

		// seq->hash
		var blockSequence types.BlockSequence
		blockSequence.Hash = hash
		blockSequence.Type = Type
		BlockSequenceByte, err := proto.Marshal(&blockSequence)
		if err != nil {
			storeLog.Error("SaveBlockSequence Marshal BlockSequence", "hash", common.ToHex(hash), "error", err)
			return newSequence, err
		}
		storeBatch.Set(calcSequenceToHashKey(newSequence, bs.isParaChain), BlockSequenceByte)

		sequenceBytes := types.Encode(&types.Int64{Data: newSequence})
		// hash->seq    add block  hash seq
		if Type == types.AddBlock {
			storeBatch.Set(calcHashToSequenceKey(hash, bs.isParaChain), sequenceBytes)
		}

		//   last seq
		storeBatch.Set(calcLastSeqKey(bs.isParaChain), sequenceBytes)
	}

	if !bs.isParaChain {
		return newSequence, nil
	}

	mainSeq := sequence
	var blockSequence types.BlockSequence
	blockSequence.Hash = hash
	blockSequence.Type = Type
	BlockSequenceByte, err := proto.Marshal(&blockSequence)
	if err != nil {
		storeLog.Error("SaveBlockSequence Marshal BlockSequence", "hash", common.ToHex(hash), "error", err)
		return newSequence, err
	}
	storeBatch.Set(calcMainSequenceToHashKey(mainSeq), BlockSequenceByte)

	// hash->seq    add block  hash seq
	sequenceBytes := types.Encode(&types.Int64{Data: mainSeq})
	if Type == types.AddBlock {
		storeBatch.Set(calcHashToMainSequenceKey(hash), sequenceBytes)
	}
	storeBatch.Set(LastSequence, sequenceBytes)

	return newSequence, nil
}

//LoadBlockBySequence   seq    BlockDetail
func (bs *BlockStore) LoadBlockBySequence(Sequence int64) (*types.BlockDetail, int, error) {
	//    Sequence        blockhash      db
	BlockSequence, err := bs.GetBlockSequence(Sequence)
	if err != nil {
		return nil, 0, err
	}
	return bs.loadBlockByHash(BlockSequence.Hash)
}

//LoadBlockByMainSequence   main seq    BlockDetail
func (bs *BlockStore) LoadBlockByMainSequence(sequence int64) (*types.BlockDetail, int, error) {
	//    Sequence        blockhash      db
	BlockSequence, err := bs.GetBlockByMainSequence(sequence)
	if err != nil {
		return nil, 0, err
	}
	return bs.loadBlockByHash(BlockSequence.Hash)
}

//GetBlockSequence  db        Sequence   block
func (bs *BlockStore) GetBlockSequence(Sequence int64) (*types.BlockSequence, error) {
	var blockSeq types.BlockSequence
	blockSeqByte, err := bs.db.Get(calcSequenceToHashKey(Sequence, bs.isParaChain))
	if blockSeqByte == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetBlockSequence", "error", err)
		}
		return nil, types.ErrHeightNotExist
	}

	err = proto.Unmarshal(blockSeqByte, &blockSeq)
	if err != nil {
		storeLog.Error("GetBlockSequence", "err", err)
		return nil, err
	}
	return &blockSeq, nil
}

//GetBlockByMainSequence  db        Sequence   block
func (bs *BlockStore) GetBlockByMainSequence(sequence int64) (*types.BlockSequence, error) {
	var blockSeq types.BlockSequence
	blockSeqByte, err := bs.db.Get(calcMainSequenceToHashKey(sequence))
	if blockSeqByte == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetBlockByMainSequence", "error", err)
		}
		return nil, types.ErrHeightNotExist
	}

	err = proto.Unmarshal(blockSeqByte, &blockSeq)
	if err != nil {
		storeLog.Error("GetBlockByMainSequence", "err", err)
		return nil, err
	}
	return &blockSeq, nil
}

//GetSequenceByHash   block       seq
func (bs *BlockStore) GetSequenceByHash(hash []byte) (int64, error) {
	var seq types.Int64
	seqbytes, err := bs.db.Get(calcHashToSequenceKey(hash, bs.isParaChain))
	if seqbytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetSequenceByHash", "error", err)
		}
		return -1, types.ErrHashNotExist
	}

	err = types.Decode(seqbytes, &seq)
	if err != nil {
		storeLog.Error("GetSequenceByHash  types.Decode", "error", err)
		return -1, types.ErrUnmarshal
	}
	return seq.Data, nil
}

//GetMainSequenceByHash   block       seq，    parachain
func (bs *BlockStore) GetMainSequenceByHash(hash []byte) (int64, error) {
	var seq types.Int64
	seqbytes, err := bs.db.Get(calcHashToMainSequenceKey(hash))
	if seqbytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetMainSequenceByHash", "error", err)
		}
		return -1, types.ErrHashNotExist
	}

	err = types.Decode(seqbytes, &seq)
	if err != nil {
		storeLog.Error("GetMainSequenceByHash  types.Decode", "error", err)
		return -1, types.ErrUnmarshal
	}
	return seq.Data, nil
}

//GetDbVersion   blockchain
func (bs *BlockStore) GetDbVersion() int64 {
	ver := types.Int64{}
	version, err := bs.db.Get(version.BlockChainVerKey)
	if err != nil && err != types.ErrNotFound {
		storeLog.Info("GetDbVersion", "err", err)
		return 0
	}
	if len(version) == 0 {
		storeLog.Info("GetDbVersion len(version)==0")
		return 0
	}
	err = types.Decode(version, &ver)
	if err != nil {
		storeLog.Info("GetDbVersion", "types.Decode err", err)
		return 0
	}
	storeLog.Info("GetDbVersion", "blockchain db version", ver.Data)
	return ver.Data
}

//SetDbVersion   blockchain
func (bs *BlockStore) SetDbVersion(versionNo int64) error {
	ver := types.Int64{Data: versionNo}
	verByte := types.Encode(&ver)

	storeLog.Info("SetDbVersion", "blcokchain db version", versionNo)

	return bs.db.SetSync(version.BlockChainVerKey, verByte)
}

//GetUpgradeMeta   blockchain
func (bs *BlockStore) GetUpgradeMeta() (*types.UpgradeMeta, error) {
	ver := types.UpgradeMeta{}
	version, err := bs.db.Get(version.LocalDBMeta)
	if err != nil && err != dbm.ErrNotFoundInDb {
		return nil, err
	}
	if len(version) == 0 {
		return &types.UpgradeMeta{Version: "0.0.0"}, nil
	}
	err = types.Decode(version, &ver)
	if err != nil {
		return nil, err
	}
	storeLog.Info("GetUpgradeMeta", "blockchain db version", ver)
	return &ver, nil
}

//SetUpgradeMeta   blockchain
func (bs *BlockStore) SetUpgradeMeta(meta *types.UpgradeMeta) error {
	verByte := types.Encode(meta)
	storeLog.Info("SetUpgradeMeta", "meta", meta)
	return bs.db.SetSync(version.LocalDBMeta, verByte)
}

//GetStoreUpgradeMeta     blockchain  Store
func (bs *BlockStore) GetStoreUpgradeMeta() (*types.UpgradeMeta, error) {
	ver := types.UpgradeMeta{}
	version, err := bs.db.Get(version.StoreDBMeta)
	if err != nil && err != dbm.ErrNotFoundInDb {
		return nil, err
	}
	if len(version) == 0 {
		return &types.UpgradeMeta{Version: "0.0.0"}, nil
	}
	err = types.Decode(version, &ver)
	if err != nil {
		return nil, err
	}
	storeLog.Info("GetStoreUpgradeMeta", "blockchain db version", ver)
	return &ver, nil
}

//SetStoreUpgradeMeta   blockchain  Store
func (bs *BlockStore) SetStoreUpgradeMeta(meta *types.UpgradeMeta) error {
	verByte := types.Encode(meta)
	storeLog.Debug("SetStoreUpgradeMeta", "meta", meta)
	return bs.db.SetSync(version.StoreDBMeta, verByte)
}

const (
	seqStatusOk = iota
	seqStatusNeedCreate
)

//CheckSequenceStatus
func (bs *BlockStore) CheckSequenceStatus(recordSequence bool) int {
	lastHeight := bs.Height()
	lastSequence, err := bs.LoadBlockLastSequence()
	if err != nil {
		if err != types.ErrHeightNotExist {
			storeLog.Error("CheckSequenceStatus", "LoadBlockLastSequence err", err)
			panic(err)
		}
	}
	//  isRecordBlockSequence
	if recordSequence {
		//    isRecordBlockSequence
		if lastSequence == -1 && lastHeight != -1 {
			storeLog.Error("CheckSequenceStatus", "lastHeight", lastHeight, "lastSequence", lastSequence)
			return seqStatusNeedCreate
		}
		//lastSequence       lastheight
		if lastHeight > lastSequence {
			storeLog.Error("CheckSequenceStatus", "lastHeight", lastHeight, "lastSequence", lastSequence)
			return seqStatusNeedCreate
		}
		return seqStatusOk
	}
	//   isRecordBlockSequence
	if lastSequence != -1 {
		storeLog.Error("CheckSequenceStatus", "lastSequence", lastSequence)
		panic("can not disable isRecordBlockSequence")
	}
	return seqStatusOk
}

//CreateSequences       sequence
func (bs *BlockStore) CreateSequences(batchSize int64) {
	lastSeq, err := bs.LoadBlockLastSequence()
	if err != nil {
		if err != types.ErrHeightNotExist {
			storeLog.Error("CreateSequences LoadBlockLastSequence", "error", err)
			panic("CreateSequences LoadBlockLastSequence" + err.Error())
		}
	}
	storeLog.Info("CreateSequences LoadBlockLastSequence", "start", lastSeq)

	newBatch := bs.NewBatch(true)
	lastHeight := bs.Height()

	for i := lastSeq + 1; i <= lastHeight; i++ {
		seq := i
		header, err := bs.GetBlockHeaderByHeight(i)
		if err != nil {
			storeLog.Error("CreateSequences GetBlockHeaderByHeight", "height", i, "error", err)
			panic("CreateSequences GetBlockHeaderByHeight" + err.Error())
		}

		// seq->hash
		var blockSequence types.BlockSequence
		blockSequence.Hash = header.Hash
		blockSequence.Type = types.AddBlock
		BlockSequenceByte, err := proto.Marshal(&blockSequence)
		if err != nil {
			storeLog.Error("CreateSequences Marshal BlockSequence", "height", i, "hash", common.ToHex(header.Hash), "error", err)
			panic("CreateSequences Marshal BlockSequence" + err.Error())
		}
		newBatch.Set(calcSequenceToHashKey(seq, bs.isParaChain), BlockSequenceByte)

		// hash -> seq
		sequenceBytes := types.Encode(&types.Int64{Data: seq})
		newBatch.Set(calcHashToSequenceKey(header.Hash, bs.isParaChain), sequenceBytes)

		if i-lastSeq == batchSize {
			storeLog.Info("CreateSequences ", "height", i)
			newBatch.Set(calcLastSeqKey(bs.isParaChain), types.Encode(&types.Int64{Data: i}))
			err = newBatch.Write()
			if err != nil {
				storeLog.Error("CreateSequences newBatch.Write", "error", err)
				panic("CreateSequences newBatch.Write" + err.Error())
			}
			lastSeq = i
			newBatch.Reset()
		}
	}
	// last seq
	newBatch.Set(calcLastSeqKey(bs.isParaChain), types.Encode(&types.Int64{Data: lastHeight}))
	err = newBatch.Write()
	if err != nil {
		storeLog.Error("CreateSequences newBatch.Write", "error", err)
		panic("CreateSequences newBatch.Write" + err.Error())
	}
	storeLog.Info("CreateSequences done")
}

//SetConsensusPara   kv    , value     delete
func (bs *BlockStore) SetConsensusPara(kvs *types.LocalDBSet) error {
	var isSync bool
	if kvs.GetTxid() != 0 {
		isSync = true
	}
	batch := bs.db.NewBatch(isSync)
	for i := 0; i < len(kvs.KV); i++ {
		if types.CheckConsensusParaTxsKey(kvs.KV[i].Key) {
			if kvs.KV[i].Value == nil {
				batch.Delete(kvs.KV[i].Key)
			} else {
				batch.Set(kvs.KV[i].Key, kvs.KV[i].Value)
			}
		} else {
			storeLog.Error("Set:CheckConsensusParaTxsKey:fail", "key", string(kvs.KV[i].Key))
		}
	}
	err := batch.Write()
	if err != nil {
		panic(err)
	}
	return err
}

//saveBlockForTable  block header body  paratx   table  ，
func (bs *BlockStore) saveBlockForTable(storeBatch dbm.Batch, blockdetail *types.BlockDetail, isBestChain, isSaveReceipt bool) error {

	height := blockdetail.Block.Height
	cfg := bs.client.GetConfig()
	hash := blockdetail.Block.Hash(cfg)

	// Save blockbody
	blockbody := bs.BlockdetailToBlockBody(blockdetail)
	//    blockbody    receipt         ,
	//   body    Receipts   ，     ReceiptTable
	blockbody.Receipts = reduceReceipts(blockbody)
	bodykvs, err := saveBlockBodyTable(bs.db, blockbody)
	if err != nil {
		storeLog.Error("SaveBlockForTable:saveBlockBodyTable", "height", height, "hash", common.ToHex(hash), "err", err)
		return err
	}
	for _, kv := range bodykvs {
		storeBatch.Set(kv.GetKey(), kv.GetValue())
	}

	// Save blockReceipt
	if isSaveReceipt {
		blockReceipt := &types.BlockReceipt{
			Receipts: blockdetail.Receipts,
			Hash:     hash,
			Height:   height,
		}
		recptkvs, err := saveBlockReceiptTable(bs.db, blockReceipt)
		if err != nil {
			storeLog.Error("SaveBlockForTable:saveBlockReceiptTable", "height", height, "hash", common.ToHex(hash), "err", err)
			return err
		}
		for _, kv := range recptkvs {
			storeBatch.Set(kv.GetKey(), kv.GetValue())
		}
	}

	// Save blockheader
	var blockheader types.Header
	blockheader.Version = blockdetail.Block.Version
	blockheader.ParentHash = blockdetail.Block.ParentHash
	blockheader.TxHash = blockdetail.Block.TxHash
	blockheader.StateHash = blockdetail.Block.StateHash
	blockheader.Height = blockdetail.Block.Height
	blockheader.BlockTime = blockdetail.Block.BlockTime
	blockheader.Signature = blockdetail.Block.Signature
	blockheader.Difficulty = blockdetail.Block.Difficulty
	blockheader.Hash = hash
	blockheader.TxCount = int64(len(blockdetail.Block.Txs))

	headerkvs, err := saveHeaderTable(bs.db, &blockheader)
	if err != nil {
		storeLog.Error("SaveBlock:saveHeaderTable", "height", height, "hash", common.ToHex(hash), "err", err)
		return err
	}
	for _, kv := range headerkvs {
		storeBatch.Set(kv.GetKey(), kv.GetValue())
	}

	//
	if isBestChain && !bs.isParaChain {
		paratxkvs, err := saveParaTxTable(cfg, bs.db, height, hash, blockdetail.Block.Txs)
		if err != nil {
			storeLog.Error("SaveBlock:saveParaTxTable", "height", height, "hash", common.ToHex(hash), "err", err)
			return err
		}
		for _, kv := range paratxkvs {
			storeBatch.Set(kv.GetKey(), kv.GetValue())
		}
	}
	return nil
}

//loadBlockByIndex       BlockDetail
func (bs *BlockStore) loadBlockByIndex(indexName string, prefix []byte, primaryKey []byte) (*types.BlockDetail, error) {
	cfg := bs.client.GetConfig()
	//  header
	blockheader, err := getHeaderByIndex(bs.db, indexName, prefix, primaryKey)
	if blockheader == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("loadBlockByIndex:getHeaderByIndex", "indexName", indexName, "prefix", prefix, "primaryKey", primaryKey, "err", err)
		}
		return nil, types.ErrHashNotExist
	}

	//  body
	blockbody, err := bs.multiGetBody(blockheader, indexName, prefix, primaryKey)
	if blockbody == nil || err != nil {
		return nil, err
	}

	blockreceipt := blockbody.Receipts
	//             ReceiptTable      receipt  ,               receipt
	if !cfg.IsEnable("reduceLocaldb") || bs.Height() < blockheader.Height+ReduceHeight {
		receipt, err := getReceiptByIndex(bs.db, indexName, prefix, primaryKey)
		if receipt != nil {
			blockreceipt = receipt.Receipts
		} else {
			storeLog.Error("loadBlockByIndex:getReceiptByIndex", "indexName", indexName, "prefix", prefix, "primaryKey", primaryKey, "err", err)
			if !cfg.IsEnable("reduceLocaldb") {
				return nil, types.ErrHashNotExist
			}
		}
	}

	var blockdetail types.BlockDetail
	var block types.Block

	block.Version = blockheader.Version
	block.ParentHash = blockheader.ParentHash
	block.TxHash = blockheader.TxHash
	block.StateHash = blockheader.StateHash
	block.Height = blockheader.Height
	block.BlockTime = blockheader.BlockTime
	block.Signature = blockheader.Signature
	block.Difficulty = blockheader.Difficulty
	block.Txs = blockbody.Txs
	block.MainHeight = blockbody.MainHeight
	block.MainHash = blockbody.MainHash

	blockdetail.Receipts = blockreceipt
	blockdetail.Block = &block
	return &blockdetail, nil
}

//loadHeaderByIndex
func (bs *BlockStore) loadHeaderByIndex(height int64) (*types.Header, error) {

	hash, err := bs.GetBlockHashByHeight(height)
	if err != nil {
		return nil, err
	}

	header, err := getHeaderByIndex(bs.db, "", calcHeightHashKey(height, hash), nil)
	if header == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("loadHeaderByHeight:getHeaderByIndex", "error", err)
		}
		return nil, types.ErrHashNotExist
	}
	return header, nil
}

//loadBlockByHashOld    db        key   block
func (bs *BlockStore) loadBlockByHashOld(hash []byte) (*types.BlockDetail, error) {
	var blockdetail types.BlockDetail
	var blockheader types.Header
	var blockbody types.BlockBody
	var block types.Block

	//  hash  blockheader
	header, err := bs.db.Get(calcHashToBlockHeaderKey(hash))
	if header == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("loadBlockByHashOld:calcHashToBlockHeaderKey", "hash", common.ToHex(hash), "err", err)
		}
		return nil, types.ErrHashNotExist
	}
	err = proto.Unmarshal(header, &blockheader)
	if err != nil {
		storeLog.Error("loadBlockByHashOld", "err", err)
		return nil, err
	}

	//  hash  blockbody
	body, err := bs.db.Get(calcHashToBlockBodyKey(hash))
	if body == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("loadBlockByHashOld:calcHashToBlockBodyKey ", "err", err)
		}
		return nil, types.ErrHashNotExist
	}
	err = proto.Unmarshal(body, &blockbody)
	if err != nil {
		storeLog.Error("loadBlockByHashOld", "err", err)
		return nil, err
	}

	block.Version = blockheader.Version
	block.ParentHash = blockheader.ParentHash
	block.TxHash = blockheader.TxHash
	block.StateHash = blockheader.StateHash
	block.Height = blockheader.Height
	block.BlockTime = blockheader.BlockTime
	block.Signature = blockheader.Signature
	block.Difficulty = blockheader.Difficulty
	block.Txs = blockbody.Txs
	block.MainHeight = blockbody.MainHeight
	block.MainHash = blockbody.MainHash
	blockdetail.Receipts = blockbody.Receipts
	blockdetail.Block = &block

	return &blockdetail, nil
}

//loadBlockByHeightOld       2.0.0          block
func (bs *BlockStore) loadBlockByHeightOld(height int64) (*types.BlockDetail, error) {
	hash, err := bs.GetBlockHashByHeight(height)
	if err != nil {
		return nil, err
	}
	return bs.loadBlockByHashOld(hash)
}

//getBlockHeaderByHeightOld    db        key   header
func (bs *BlockStore) getBlockHeaderByHeightOld(height int64) (*types.Header, error) {
	blockheader, err := bs.db.Get(calcHeightToBlockHeaderKey(height))
	if err != nil {
		var hash []byte
		hash, err = bs.GetBlockHashByHeight(height)
		if err != nil {
			return nil, err
		}
		blockheader, err = bs.db.Get(calcHashToBlockHeaderKey(hash))
	}
	if blockheader == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("getBlockHeaderByHeightOld:calcHashToBlockHeaderKey", "error", err)
		}
		return nil, types.ErrHashNotExist
	}
	var header types.Header
	err = proto.Unmarshal(blockheader, &header)
	if err != nil {
		storeLog.Error("getBlockHeaderByHeightOld", "Could not unmarshal blockheader:", blockheader)
		return nil, err
	}
	return &header, nil
}

//loadBlockBySequenceOld   seq    BlockDetail  ,v2
//         ，      ，
func (bs *BlockStore) loadBlockBySequenceOld(Sequence int64) (*types.BlockDetail, int64, error) {

	blockSeq, err := bs.GetBlockSequence(Sequence)
	if err != nil {
		return nil, 0, err
	}

	block, err := bs.loadBlockByHashOld(blockSeq.Hash)
	if err != nil {
		block, err = bs.LoadBlockByHash(blockSeq.Hash)
		if err != nil {
			storeLog.Error("getBlockHeaderByHeightOld", "Sequence", Sequence, "type", blockSeq.Type, "hash", common.ToHex(blockSeq.Hash), "err", err)
			panic(err)
		}
	}
	return block, blockSeq.GetType(), nil
}

func (bs *BlockStore) saveReduceLocaldbFlag() {
	kv := types.FlagKV(types.FlagReduceLocaldb, 1)
	err := bs.db.Set(kv.Key, kv.Value)
	if err != nil {
		panic(err)
	}
}

func (bs *BlockStore) mustSaveKvset(kv *types.LocalDBSet) {
	batch := bs.NewBatch(false)
	for i := 0; i < len(kv.KV); i++ {
		if kv.KV[i].Value == nil {
			batch.Delete(kv.KV[i].Key)
		} else {
			batch.Set(kv.KV[i].Key, kv.KV[i].Value)
		}
	}
	dbm.MustWrite(batch)
}

func (bs *BlockStore) multiGetBody(blockheader *types.Header, indexName string, prefix []byte, primaryKey []byte) (*types.BlockBody, error) {
	cfg := bs.client.GetConfig()
	chainCfg := cfg.GetModuleConfig().BlockChain

	//  body
	//var blockbody *types.BlockBody
	//if chainCfg.EnableIfDelLocalChunk { // 6.6  ，
	//	chunkNum, _, _ := calcChunkInfo(chainCfg, blockheader.Height)
	//	if chunkNum <= bs.GetMaxSerialChunkNum() { //
	//		bodys, err := bs.getBodyFromP2Pstore(blockheader.Hash, blockheader.Height, blockheader.Height)
	//		if bodys == nil || len(bodys.Items) == 0 || err != nil {
	//			if err != dbm.ErrNotFoundInDb {
	//				storeLog.Error("multiGetBody:getBodyFromP2Pstore", "chunkNum", chunkNum, "height", blockheader.Height,
	//					"hash", common.ToHex(blockheader.Hash), "err", err)
	//			}
	//			return nil, types.ErrHashNotExist
	//		}
	//		blockbody = bodys.Items[0]
	//		storeLog.Info("multiGetBody", "chunkNum", chunkNum, "height", blockheader.Height,
	//			"hash", common.ToHex(blockheader.Hash))
	//		return blockbody, nil
	//	}
	//
	//	storeLog.Info("multiGetBody", "chunkNum", chunkNum, "height", blockheader.Height,
	//		"hash", common.ToHex(blockheader.Hash))
	//	blockbody, err := getBodyByIndex(bs.db, indexName, prefix, primaryKey)
	//	if blockbody == nil || err != nil {
	//		if err != dbm.ErrNotFoundInDb {
	//			storeLog.Error("multiGetBody:getBodyByIndex", "indexName", indexName, "prefix", prefix, "primaryKey", primaryKey, "err", err)
	//		}
	//		return nil, types.ErrHashNotExist
	//	}
	//	return blockbody, nil
	//}

	blockbody, err := getBodyByIndex(bs.db, indexName, prefix, primaryKey)
	if blockbody == nil || err != nil {
		if !chainCfg.DisableShard && chainCfg.EnableFetchP2pstore {
			bodys, err := bs.getBodyFromP2Pstore(blockheader.Hash, blockheader.Height, blockheader.Height)
			if bodys == nil || len(bodys.Items) == 0 || err != nil {
				if err != dbm.ErrNotFoundInDb {
					storeLog.Error("multiGetBody:getBodyFromP2Pstore", "indexName", indexName, "prefix", prefix,
						"primaryKey", primaryKey, "err", err, "hash", common.ToHex(blockheader.Hash))
				}
				return nil, types.ErrHashNotExist
			}
			blockbody = bodys.Items[0]
			return blockbody, nil
		}

		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("multiGetBody:getBodyByIndex", "indexName", indexName, "prefix", prefix, "primaryKey", primaryKey, "err", err)
		}
		return nil, types.ErrHashNotExist
	}
	return blockbody, nil
}

func (bs *BlockStore) getBodyFromP2Pstore(hash []byte, start, end int64) (*types.BlockBodys, error) {
	stime := time.Now()
	defer func() {
		etime := time.Now()
		storeLog.Info("getBodyFromP2Pstore", "start", start, "end", end, "cost time", etime.Sub(stime))
	}()
	value, err := bs.db.Get(calcBlockHashToChunkHash(hash))
	if value == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("getBodyFromP2Pstore:calcBlockHashToChunkHash", "hash", common.ToHex(hash), "chunkhash", common.ToHex(value), "err", err)
		}
		return nil, types.ErrHashNotExist
	}
	if bs.client == nil {
		storeLog.Error("getBodyFromP2Pstore: chain client not bind message queue.")
		return nil, types.ErrClientNotBindQueue
	}
	req := &types.ChunkInfoMsg{
		ChunkHash: value,
		Start:     start,
		End:       end,
	}
	msg := bs.client.NewMessage("p2p", types.EventGetChunkBlockBody, req)
	err = bs.client.Send(msg, true)
	if err != nil {
		storeLog.Error("EventGetChunkBlockBody", "chunk hash", common.ToHex(value), "hash", common.ToHex(hash), "err", err)
		return nil, err
	}
	//           3
	resp, err := bs.client.WaitTimeout(msg, time.Minute*3)
	if err != nil {
		storeLog.Error("EventGetChunkBlockBody", "client.Wait err:", err)
		return nil, err
	}
	bodys, ok := resp.Data.(*types.BlockBodys)
	if !ok {
		err = types.ErrNotFound
		storeLog.Error("EventGetChunkBlockBody", "client.Wait err:", err)
		return nil, err
	}
	return bodys, nil
}

func (bs *BlockStore) getCurChunkNum(prefix []byte) int64 {
	it := bs.db.Iterator(prefix, nil, true)
	defer it.Close()
	height := int64(-1)
	var err error
	if it.Rewind() && it.Valid() {
		height, err = strconv.ParseInt(string(bytes.TrimPrefix(it.Key(), prefix)), 10, 64)
		if err != nil {
			return -1
		}
	}
	return height
}

func (bs *BlockStore) getRecvChunkHash(chunkNum int64) ([]byte, error) {
	value, err := bs.GetKey(calcRecvChunkNumToHash(chunkNum))
	if err != nil {
		storeLog.Error("getRecvChunkHash", "chunkNum", chunkNum, "error", err)
		return nil, err
	}
	var chunk types.ChunkInfo
	err = types.Decode(value, &chunk)
	if err != nil {
		storeLog.Error("getRecvChunkHash", "chunkNum", chunkNum, "decode err:", err)
		return nil, err
	}
	return chunk.ChunkHash, err
}

//GetMaxSerialChunkNum get max serial chunk num
func (bs *BlockStore) GetMaxSerialChunkNum() int64 {
	value, err := bs.db.Get(MaxSerialChunkNum)
	if err != nil {
		return -1
	}
	chunkNum := &types.Int64{}
	err = types.Decode(value, chunkNum)
	if err != nil {
		return -1
	}
	return chunkNum.Data
}

//SetMaxSerialChunkNum set max serial chunk num
func (bs *BlockStore) SetMaxSerialChunkNum(chunkNum int64) error {
	data := &types.Int64{
		Data: chunkNum,
	}
	err := bs.db.Set(MaxSerialChunkNum, types.Encode(data))
	if err != nil {
		return err
	}
	return nil
}

// GetActiveBlock :
func (bs *BlockStore) GetActiveBlock(hash string) (*types.BlockDetail, bool) {
	block := bs.activeBlocks.Get(hash)
	if block != nil {
		return block.(*types.BlockDetail), true
	}
	return nil, false
}

// AddActiveBlock :
func (bs *BlockStore) AddActiveBlock(hash string, block *types.BlockDetail) bool {
	return bs.activeBlocks.Add(hash, block, block.Size())
}

// RemoveActiveBlock :
func (bs *BlockStore) RemoveActiveBlock(hash string) bool {
	_, ok := bs.activeBlocks.Remove(hash)
	return ok
}
