// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package blockchain        ，
*/
package blockchain

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	lru "github.com/hashicorp/golang-lru"
)

//var
var (
	//cache    block
	zeroHash             [32]byte
	InitBlockNum         int64 = 10240 //       db index bestchain      blocknode ， blockNodeCacheLimit
	chainlog                   = log.New("module", "blockchain")
	FutureBlockDelayTime int64 = 1
)

const (
	maxFutureBlocks = 256

	maxActiveBlocks = 1024

	//              ， 100M
	maxActiveBlocksCacheSize = 100

	defaultChunkBlockNum = 1
)

//BlockChain
type BlockChain struct {
	client  queue.Client
	cache   *BlockCache
	txCache *TxCache

	//        db
	blockStore *BlockStore
	push       *Push
	//cache    block
	cfg             *types.BlockChain
	syncTask        *Task
	downLoadTask    *Task
	chunkRecordTask *Task

	query *Query

	//          block  ,      active
	rcvLastBlockHeight int64

	//          block  ,      active ,
	synBlockHeight int64

	//  peer   block  ,      active
	peerList PeerInfoList
	recvwg   *sync.WaitGroup
	tickerwg *sync.WaitGroup
	reducewg *sync.WaitGroup

	synblock            chan struct{}
	quit                chan struct{}
	isclosed            int32
	runcount            int32
	isbatchsync         int32
	firstcheckbestchain int32 //

	//
	orphanPool *OrphanPool
	//        blocknode
	index *blockIndex
	//
	bestChain *chainView

	chainLock sync.RWMutex
	//blockchain
	startTime time.Time

	//
	isCaughtUp bool

	//  block       ，         。
	//              ，      .
	cfgBatchSync bool

	//        peer
	// ExecBlock          peerid          hash
	faultPeerList map[string]*FaultPeerInfo

	bestChainPeerList map[string]*BestPeerInfo

	//  futureblocks
	futureBlocks *lru.Cache // future blocks are broadcast later processing

	//downLoad block info
	downLoadInfo *DownLoadInfo
	downloadMode int //         ，        db，     goroutine   block

	isRecordBlockSequence bool //    add  del block   ，  blcokchain
	enablePushSubscribe   bool //
	isParaChain           bool //      。       Sequence
	isStrongConsistency   bool
	//lock
	synBlocklock     sync.Mutex
	peerMaxBlklock   sync.Mutex
	castlock         sync.Mutex
	ntpClockSynclock sync.Mutex
	faultpeerlock    sync.Mutex
	bestpeerlock     sync.Mutex
	downLoadlock     sync.Mutex
	downLoadModeLock sync.Mutex
	isNtpClockSync   bool //ntp

	//cfg
	MaxFetchBlockNum int64 //        block
	TimeoutSeconds   int64
	blockSynInterVal time.Duration
	DefCacheSize     int64
	DefTxCacheSize   int

	failed int32

	blockOnChain   *BlockOnChain
	onChainTimeout int64

	//
	maxSerialChunkNum int64
}

//New new
func New(cfg *types.DplatformOSConfig) *BlockChain {
	mcfg := cfg.GetModuleConfig().BlockChain
	futureBlocks, err := lru.New(maxFutureBlocks)
	if err != nil {
		panic("when New BlockChain lru.New return err")
	}

	defCacheSize := int64(128)
	if mcfg.DefCacheSize > 0 {
		defCacheSize = mcfg.DefCacheSize
	}
	if atomic.LoadInt64(&mcfg.ChunkblockNum) == 0 {
		atomic.StoreInt64(&mcfg.ChunkblockNum, defaultChunkBlockNum)
	}

	// 	   AllowPackHeight
	InitAllowPackHeight(mcfg)
	txCacheSize := int(types.HighAllowPackHeight + types.LowAllowPackHeight + 1)

	blockchain := &BlockChain{
		cache:              NewBlockCache(cfg, defCacheSize),
		txCache:            NewTxCache(txCacheSize),
		DefCacheSize:       defCacheSize,
		DefTxCacheSize:     txCacheSize,
		rcvLastBlockHeight: -1,
		synBlockHeight:     -1,
		maxSerialChunkNum:  -1,
		peerList:           nil,
		cfg:                mcfg,
		recvwg:             &sync.WaitGroup{},
		tickerwg:           &sync.WaitGroup{},
		reducewg:           &sync.WaitGroup{},

		syncTask:        newTask(300 * time.Second), //             ，    task
		downLoadTask:    newTask(300 * time.Second),
		chunkRecordTask: newTask(120 * time.Second),

		quit:                make(chan struct{}),
		synblock:            make(chan struct{}, 1),
		orphanPool:          NewOrphanPool(cfg),
		index:               newBlockIndex(),
		isCaughtUp:          false,
		isbatchsync:         1,
		firstcheckbestchain: 0,
		cfgBatchSync:        mcfg.Batchsync,
		faultPeerList:       make(map[string]*FaultPeerInfo),
		bestChainPeerList:   make(map[string]*BestPeerInfo),
		futureBlocks:        futureBlocks,
		downLoadInfo:        &DownLoadInfo{},
		isNtpClockSync:      true,
		MaxFetchBlockNum:    128 * 6, //        block
		TimeoutSeconds:      2,
		downloadMode:        fastDownLoadMode,
		blockOnChain:        &BlockOnChain{},
		onChainTimeout:      0,
	}
	blockchain.initConfig(cfg)
	return blockchain
}

func (chain *BlockChain) initConfig(cfg *types.DplatformOSConfig) {
	mcfg := cfg.GetModuleConfig().BlockChain
	//if cfg.IsEnable("TxHeight") && chain.DefCacheSize <= (types.LowAllowPackHeight+types.HighAllowPackHeight+1) {
	//	panic("when Enable TxHeight DefCacheSize must big than types.LowAllowPackHeight + types.HighAllowPackHeight")
	//}

	if mcfg.MaxFetchBlockNum > 0 {
		chain.MaxFetchBlockNum = mcfg.MaxFetchBlockNum
	}

	if mcfg.TimeoutSeconds > 0 {
		chain.TimeoutSeconds = mcfg.TimeoutSeconds
	}
	chain.blockSynInterVal = time.Duration(chain.TimeoutSeconds)
	chain.isStrongConsistency = mcfg.IsStrongConsistency
	chain.isRecordBlockSequence = mcfg.IsRecordBlockSequence
	chain.enablePushSubscribe = mcfg.EnablePushSubscribe
	chain.isParaChain = mcfg.IsParaChain
	cfg.S("quickIndex", mcfg.EnableTxQuickIndex)
	cfg.S("reduceLocaldb", mcfg.EnableReduceLocaldb)

	if mcfg.OnChainTimeout > 0 {
		chain.onChainTimeout = mcfg.OnChainTimeout
	}
	chain.initOnChainTimeout()
}

//Close
func (chain *BlockChain) Close() {
	//          ，
	atomic.StoreInt32(&chain.isclosed, 1)

	//
	close(chain.quit)

	//
	for atomic.LoadInt32(&chain.runcount) > 0 {
		time.Sleep(time.Microsecond)
	}
	chain.client.Close()
	//wait for recvwg quit:
	chainlog.Info("blockchain wait for recvwg quit")
	chain.recvwg.Wait()

	//wait for tickerwg quit:
	chainlog.Info("blockchain wait for tickerwg quit")
	chain.tickerwg.Wait()

	//wait for reducewg quit:
	chainlog.Info("blockchain wait for reducewg quit")
	chain.reducewg.Wait()

	if chain.push != nil {
		chainlog.Info("blockchain wait for push quit")
		chain.push.Close()
	}

	//
	chain.blockStore.db.Close()
	chainlog.Info("blockchain module closed")
}

//SetQueueClient
func (chain *BlockChain) SetQueueClient(client queue.Client) {
	chain.client = client
	chain.client.Sub("blockchain")

	blockStoreDB := dbm.NewDB("blockchain", chain.cfg.Driver, chain.cfg.DbPath, chain.cfg.DbCache)
	blockStore := NewBlockStore(chain, blockStoreDB, client)
	chain.blockStore = blockStore
	stateHash := chain.getStateHash()
	chain.query = NewQuery(blockStoreDB, chain.client, stateHash)
	if chain.enablePushSubscribe && chain.isRecordBlockSequence {
		chain.push = newpush(chain.blockStore, chain.blockStore, chain.client.GetConfig())
		chainlog.Info("chain push is setup")
	}

	//startTime
	chain.startTime = types.Now()
	//       chunk
	chain.maxSerialChunkNum = chain.blockStore.GetMaxSerialChunkNum()

	//recv      ，        lastblock
	chain.recvwg.Add(1)
	//   blockchian
	chain.InitBlockChain()
	go chain.ProcRecvMsg()
}

//Wait for ready
func (chain *BlockChain) Wait() {}

//GetStore only used for test
func (chain *BlockChain) GetStore() *BlockStore {
	return chain.blockStore
}

//GetOrphanPool
func (chain *BlockChain) GetOrphanPool() *OrphanPool {
	return chain.orphanPool
}

//InitBlockChain
func (chain *BlockChain) InitBlockChain() {
	//isRecordBlockSequence
	seqStatus := chain.blockStore.CheckSequenceStatus(chain.isRecordBlockSequence)
	if seqStatus == seqStatusNeedCreate {
		chain.blockStore.CreateSequences(100000)
	}

	//      128 block   cache
	curheight := chain.GetBlockHeight()
	if chain.client.GetConfig().IsEnable("TxHeight") {
		chain.InitCache(curheight)
	}
	//         10240      index bestview
	beg := types.Now()
	chain.InitIndexAndBestView()
	chainlog.Info("InitIndexAndBestView", "cost", types.Since(beg))

	//             ，  blockchain
	curdbver := chain.blockStore.GetDbVersion()
	if curdbver == 0 && curheight == -1 {
		curdbver = 1
		err := chain.blockStore.SetDbVersion(curdbver)
		//               types.S("dbversion", curdbver)
		if err != nil {
			curdbver = 0
			chainlog.Error("InitIndexAndBestView SetDbVersion ", "err", err)
		}
	}
	cfg := chain.client.GetConfig()
	cfg.S("dbversion", curdbver)
	if !chain.cfg.IsParaChain && chain.cfg.RollbackBlock <= 0 {
		//     /  block
		go chain.SynRoutine()

		//     futureblock
		go chain.UpdateRoutine()
	}

	if !chain.cfg.DisableShard {
		chain.tickerwg.Add(1)
		go chain.chunkProcessRoutine()
	}

	//     DownLoadInfo
	chain.DefaultDownLoadInfo()
}

func (chain *BlockChain) getStateHash() []byte {
	blockhight := chain.GetBlockHeight()
	blockdetail, err := chain.GetBlock(blockhight)
	if err != nil {
		return zeroHash[:]
	}
	if blockdetail != nil {
		return blockdetail.GetBlock().GetStateHash()
	}
	return zeroHash[:]
}

//SendAddBlockEvent blockchain   add block db    mempool  consense
func (chain *BlockChain) SendAddBlockEvent(block *types.BlockDetail) (err error) {
	if chain.client == nil {
		chainlog.Error("SendAddBlockEvent: chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}
	if block == nil {
		chainlog.Error("SendAddBlockEvent block is null")
		return types.ErrInvalidParam
	}
	chainlog.Debug("SendAddBlockEvent", "Height", block.Block.Height)

	chainlog.Debug("SendAddBlockEvent -->>mempool")
	msg := chain.client.NewMessage("mempool", types.EventAddBlock, block)
	//          ，                   ，   mempool
	Err := chain.client.Send(msg, true)
	if Err != nil {
		chainlog.Error("SendAddBlockEvent -->>mempool", "err", Err)
	}
	chainlog.Debug("SendAddBlockEvent -->>consensus")

	msg = chain.client.NewMessage("consensus", types.EventAddBlock, block)
	Err = chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Error("SendAddBlockEvent -->>consensus", "err", Err)
	}
	chainlog.Debug("SendAddBlockEvent -->>wallet", "height", block.GetBlock().GetHeight())
	msg = chain.client.NewMessage("wallet", types.EventAddBlock, block)
	Err = chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Error("SendAddBlockEvent -->>wallet", "err", Err)
	}
	return nil
}

//SendBlockBroadcast blockchain     block
func (chain *BlockChain) SendBlockBroadcast(block *types.BlockDetail) {
	cfg := chain.client.GetConfig()
	if chain.client == nil {
		chainlog.Error("SendBlockBroadcast: chain client not bind message queue.")
		return
	}
	if block == nil {
		chainlog.Error("SendBlockBroadcast block is null")
		return
	}
	chainlog.Debug("SendBlockBroadcast", "Height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)))

	msg := chain.client.NewMessage("p2p", types.EventBlockBroadcast, block.Block)
	err := chain.client.Send(msg, false)
	if err != nil {
		chainlog.Error("SendBlockBroadcast", "Height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)), "err", err)
	}
}

//GetBlockHeight
func (chain *BlockChain) GetBlockHeight() int64 {
	return chain.blockStore.Height()
}

//GetBlock          block，        ，       db
func (chain *BlockChain) GetBlock(height int64) (block *types.BlockDetail, err error) {

	cfg := chain.client.GetConfig()

	//             ，     add    block
	blockdetail := chain.cache.CheckcacheBlock(height)
	if blockdetail != nil {
		if len(blockdetail.Receipts) == 0 && len(blockdetail.Block.Txs) != 0 {
			chainlog.Debug("GetBlock  CheckcacheBlock Receipts ==0", "height", height)
		}
		return blockdetail, nil
	}

	//
	hash, err := chain.blockStore.GetBlockHashByHeight(height)
	if err != nil {
		chainlog.Error("GetBlock GetBlockHashByHeight", "height", height, "error", err)
		return nil, err
	}
	block, exist := chain.blockStore.GetActiveBlock(string(hash))
	if exist {
		return block, nil
	}

	// blockstore db   block height  block
	blockinfo, err := chain.blockStore.LoadBlockByHeight(height)
	if blockinfo != nil {
		if len(blockinfo.Receipts) == 0 && len(blockinfo.Block.Txs) != 0 {
			chainlog.Debug("GetBlock  LoadBlock Receipts ==0", "height", height)
		}

		//
		chain.blockStore.AddActiveBlock(string(blockinfo.Block.Hash(cfg)), blockinfo)
		return blockinfo, nil
	}
	return nil, err

}

//SendDelBlockEvent blockchain    del block db    mempool  consense  wallet
func (chain *BlockChain) SendDelBlockEvent(block *types.BlockDetail) (err error) {
	if chain.client == nil {
		chainlog.Error("SendDelBlockEvent: chain client not bind message queue.")
		err := types.ErrClientNotBindQueue
		return err
	}
	if block == nil {
		chainlog.Error("SendDelBlockEvent block is null")
		return nil
	}

	chainlog.Debug("SendDelBlockEvent -->>mempool&consensus&wallet", "height", block.GetBlock().GetHeight())

	msg := chain.client.NewMessage("consensus", types.EventDelBlock, block)
	Err := chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Debug("SendDelBlockEvent -->>consensus", "err", err)
	}
	msg = chain.client.NewMessage("mempool", types.EventDelBlock, block)
	Err = chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Debug("SendDelBlockEvent -->>mempool", "err", err)
	}
	msg = chain.client.NewMessage("wallet", types.EventDelBlock, block)
	Err = chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Debug("SendDelBlockEvent -->>wallet", "err", err)
	}
	return nil
}

//GetDB   DB
func (chain *BlockChain) GetDB() dbm.DB {
	return chain.blockStore.db
}

//InitCache
func (chain *BlockChain) InitCache(height int64) {
	if height < 0 {
		return
	}
	for i := height - chain.DefCacheSize; i <= height; i++ {
		if i < 0 {
			i = 0
		}
		blockdetail, err := chain.GetBlock(i)
		if err != nil {
			panic(err)
		}
		chain.cache.CacheBlock(blockdetail)
		chain.txCache.Add(blockdetail.Block)
	}
}

//InitIndexAndBestView                  128 block node   index bestchain
//             block  ，.........todo
func (chain *BlockChain) InitIndexAndBestView() {
	//  lastblocks    ,  bestviewtip
	var node *blockNode
	var prevNode *blockNode
	var height int64
	var initflag = false
	curheight := chain.blockStore.height
	if curheight == -1 {
		node = newPreGenBlockNode()
		node.parent = nil
		chain.bestChain = newChainView(node)
		chain.index.AddNode(node)
		return
	}
	if curheight >= InitBlockNum {
		height = curheight - InitBlockNum
	} else {
		height = 0
	}
	for ; height <= curheight; height++ {
		header, err := chain.blockStore.GetBlockHeaderByHeight(height)
		if header == nil {
			chainlog.Error("InitIndexAndBestView GetBlockHeaderByHeight", "height", height, "err", err)
			//    localdb 2.0.0
			header, err = chain.blockStore.getBlockHeaderByHeightOld(height)
			if header == nil {
				chainlog.Error("InitIndexAndBestView getBlockHeaderByHeightOld", "height", height, "err", err)
				panic("InitIndexAndBestView fail!")
			}
		}

		newNode := newBlockNodeByHeader(false, header, "self", -1)
		newNode.parent = prevNode
		prevNode = newNode

		chain.index.AddNode(newNode)
		if !initflag {
			chain.bestChain = newChainView(newNode)
			initflag = true
		} else {
			chain.bestChain.SetTip(newNode)
		}
	}

}

//UpdateRoutine       futureblock
func (chain *BlockChain) UpdateRoutine() {
	//1       futureblock，futureblock time            block
	futureblockTicker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-chain.quit:
			//chainlog.Info("UpdateRoutine quit")
			return
		case <-futureblockTicker.C:
			chain.ProcFutureBlocks()
		}
	}
}

//ProcFutureBlocks       futureblocks， futureblock block  time           block
func (chain *BlockChain) ProcFutureBlocks() {
	cfg := chain.client.GetConfig()
	for _, hash := range chain.futureBlocks.Keys() {
		if block, exist := chain.futureBlocks.Peek(hash); exist {
			if block != nil {
				blockdetail := block.(*types.BlockDetail)
				//block           ，   block，    block futureblocks
				if types.Now().Unix() > blockdetail.Block.BlockTime {
					chain.SendBlockBroadcast(blockdetail)
					chain.futureBlocks.Remove(hash)
					chainlog.Debug("ProcFutureBlocks Remove", "height", blockdetail.Block.Height, "hash", common.ToHex(blockdetail.Block.Hash(cfg)), "blocktime", blockdetail.Block.BlockTime, "curtime", types.Now().Unix())
				}
			}
		}
	}
}

//SetValueByKey   kv  blockchain
func (chain *BlockChain) SetValueByKey(kvs *types.LocalDBSet) error {
	return chain.blockStore.SetConsensusPara(kvs)
}

//GetValueByKey   key  blockchain      value
func (chain *BlockChain) GetValueByKey(keys *types.LocalDBGet) *types.LocalReplyValue {
	return chain.blockStore.Get(keys)
}

//DelCacheBlock
func (chain *BlockChain) DelCacheBlock(height int64, hash []byte) {

	chain.cache.DelBlockFromCache(height)
	chain.txCache.Del(height)
	chain.blockStore.RemoveActiveBlock(string(hash))
}

//InitAllowPackHeight       LowAllowPackHeight  HighAllowPackHeight
func InitAllowPackHeight(mcfg *types.BlockChain) {
	if mcfg.HighAllowPackHeight != 0 && mcfg.LowAllowPackHeight != 0 {
		if mcfg.HighAllowPackHeight+mcfg.LowAllowPackHeight < types.MaxAllowPackInterval {
			types.HighAllowPackHeight = mcfg.HighAllowPackHeight
			types.LowAllowPackHeight = mcfg.LowAllowPackHeight
		} else {
			panic("when Enable TxHeight HighAllowPackHeight + LowAllowPackHeight must less than types.MaxAllowPackInterval")
		}
	}
	chainlog.Debug("InitAllowPackHeight", "mcfg.HighAllowPackHeight", mcfg.HighAllowPackHeight, "mcfg.LowAllowPackHeight", mcfg.LowAllowPackHeight)
	chainlog.Debug("InitAllowPackHeight", "types.HighAllowPackHeight", types.HighAllowPackHeight, "types.LowAllowPackHeight", types.LowAllowPackHeight)
}
