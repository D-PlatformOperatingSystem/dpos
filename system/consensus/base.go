// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package consensus

import (
	"bytes"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/D-PlatformOperatingSystem/dpos/client"
	"github.com/D-PlatformOperatingSystem/dpos/common"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/common/merkle"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
)

var tlog = log.New("module", "consensus")

var (
	zeroHash [32]byte
)

var randgen *rand.Rand

func init() {
	randgen = rand.New(rand.NewSource(types.Now().UnixNano()))
	QueryData.Register("base", &BaseClient{})
}

//Miner
type Miner interface {
	CreateGenesisTx() []*types.Transaction
	GetGenesisBlockTime() int64
	CreateBlock()
	AddBlock(b *types.Block) error
	CheckBlock(parent *types.Block, current *types.BlockDetail) error
	ProcEvent(msg *queue.Message) bool
	CmpBestBlock(newBlock *types.Block, cmpBlock *types.Block) bool
}

//BaseClient ...
type BaseClient struct {
	client       queue.Client
	api          client.QueueProtocolAPI
	minerStart   int32
	isclosed     int32
	once         sync.Once
	Cfg          *types.Consensus
	currentBlock *types.Block
	mulock       sync.Mutex
	child        Miner
	minerstartCB func()
	isCaughtUp   int32
}

//NewBaseClient ...
func NewBaseClient(cfg *types.Consensus) *BaseClient {
	var flag int32
	if cfg.Minerstart {
		flag = 1
	}
	client := &BaseClient{minerStart: flag, isCaughtUp: 0}
	client.Cfg = cfg
	log.Info("Enter consensus " + cfg.Name)
	return client
}

//GetGenesisBlockTime
func (bc *BaseClient) GetGenesisBlockTime() int64 {
	return bc.Cfg.GenesisBlockTime
}

//SetChild ...
func (bc *BaseClient) SetChild(c Miner) {
	bc.child = c
}

//GetAPI   api
func (bc *BaseClient) GetAPI() client.QueueProtocolAPI {
	return bc.api
}

//SetAPI ...
func (bc *BaseClient) SetAPI(api client.QueueProtocolAPI) {
	bc.api = api
}

//AddBlock        ，
func (bc *BaseClient) AddBlock(b *types.Block) error {
	return nil
}

//InitClient
func (bc *BaseClient) InitClient(c queue.Client, minerstartCB func()) {
	log.Info("Enter SetQueueClient method of consensus")
	bc.client = c
	bc.minerstartCB = minerstartCB
	var err error
	bc.api, err = client.New(c, nil)
	if err != nil {
		panic(err)
	}
	bc.InitMiner()
}

//GetQueueClient
func (bc *BaseClient) GetQueueClient() queue.Client {
	return bc.client
}

//RandInt64
func (bc *BaseClient) RandInt64() int64 {
	return randgen.Int63()
}

//InitMiner
func (bc *BaseClient) InitMiner() {
	bc.once.Do(bc.minerstartCB)
}

//Wait wait for ready
func (bc *BaseClient) Wait() {}

//SetQueueClient
func (bc *BaseClient) SetQueueClient(c queue.Client) {
	bc.InitClient(c, func() {
		//call init block
		bc.InitBlock()
	})
	go bc.EventLoop()
	go bc.child.CreateBlock()
}

//InitBlock change init block
func (bc *BaseClient) InitBlock() {
	cfg := bc.client.GetConfig()
	block, err := bc.RequestLastBlock()
	if err != nil {
		panic(err)
	}
	if block == nil {
		//
		newblock := &types.Block{}
		newblock.Height = 0
		newblock.BlockTime = bc.child.GetGenesisBlockTime()
		// TODO:              nil，    ？
		newblock.ParentHash = zeroHash[:]
		tx := bc.child.CreateGenesisTx()
		newblock.Txs = tx
		newblock.TxHash = merkle.CalcMerkleRoot(cfg, newblock.Height, newblock.Txs)
		if newblock.Height == 0 {
			types.AssertConfig(bc.client)
			newblock.Difficulty = cfg.GetP(0).PowLimitBits
		}
		err := bc.WriteBlock(zeroHash[:], newblock)
		if err != nil {
			panic(err)
		}
	} else {
		bc.SetCurrentBlock(block)
	}
}

//Close
func (bc *BaseClient) Close() {
	atomic.StoreInt32(&bc.minerStart, 0)
	atomic.StoreInt32(&bc.isclosed, 1)
	bc.client.Close()
	log.Info("consensus base closed")
}

//IsClosed
func (bc *BaseClient) IsClosed() bool {
	return atomic.LoadInt32(&bc.isclosed) == 1
}

//CheckTxDup
func (bc *BaseClient) CheckTxDup(txs []*types.Transaction) (transactions []*types.Transaction) {
	cacheTxs := types.TxsToCache(txs)
	var err error
	cacheTxs, err = util.CheckTxDup(bc.client, cacheTxs, 0)
	if err != nil {
		return txs
	}
	return types.CacheToTxs(cacheTxs)
}

//IsMining
func (bc *BaseClient) IsMining() bool {
	return atomic.LoadInt32(&bc.minerStart) == 1
}

//IsCaughtUp
func (bc *BaseClient) IsCaughtUp() bool {
	if bc.client == nil {
		panic("bc not bind message queue.")
	}
	msg := bc.client.NewMessage("blockchain", types.EventIsSync, nil)
	err := bc.client.Send(msg, true)
	if err != nil {
		return false
	}
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return false
	}
	return resp.GetData().(*types.IsCaughtUp).GetIscaughtup()
}

//ExecConsensus
func (bc *BaseClient) ExecConsensus(data *types.ChainExecutor) (types.Message, error) {
	param, err := QueryData.Decode(data.Driver, data.FuncName, data.Param)
	if err != nil {
		return nil, err
	}
	return QueryData.Call(data.Driver, data.FuncName, param)
}

//EventLoop
func (bc *BaseClient) EventLoop() {
	//   blockchain  ，
	bc.client.Sub("consensus")
	go func() {
		for msg := range bc.client.Recv() {
			tlog.Debug("consensus recv", "msg", msg)
			if msg.Ty == types.EventConsensusQuery {
				exec := msg.GetData().(*types.ChainExecutor)
				param, err := QueryData.Decode(exec.Driver, exec.FuncName, exec.Param)
				if err != nil {
					msg.Reply(bc.api.NewMessage("", 0, err))
					continue
				}
				reply, err := QueryData.Call(exec.Driver, exec.FuncName, param)
				if err != nil {
					msg.Reply(bc.api.NewMessage("", 0, err))
				} else {
					msg.Reply(bc.api.NewMessage("", 0, reply))
				}
			} else if msg.Ty == types.EventAddBlock {
				block := msg.GetData().(*types.BlockDetail).Block
				bc.SetCurrentBlock(block)
				bc.child.AddBlock(block)
			} else if msg.Ty == types.EventCheckBlock {
				block := msg.GetData().(*types.BlockDetail)
				err := bc.CheckBlock(block)
				msg.ReplyErr("EventCheckBlock", err)
			} else if msg.Ty == types.EventMinerStart {
				if !atomic.CompareAndSwapInt32(&bc.minerStart, 0, 1) {
					msg.ReplyErr("EventMinerStart", types.ErrMinerIsStared)
				} else {
					bc.InitMiner()
					msg.ReplyErr("EventMinerStart", nil)
				}
			} else if msg.Ty == types.EventMinerStop {
				if !atomic.CompareAndSwapInt32(&bc.minerStart, 1, 0) {
					msg.ReplyErr("EventMinerStop", types.ErrMinerNotStared)
				} else {
					msg.ReplyErr("EventMinerStop", nil)
				}
			} else if msg.Ty == types.EventDelBlock {
				block := msg.GetData().(*types.BlockDetail).Block
				bc.UpdateCurrentBlock(block)
			} else if msg.Ty == types.EventCmpBestBlock {
				var reply types.Reply
				cmpBlock := msg.GetData().(*types.CmpBlock)
				reply.IsOk = bc.CmpBestBlock(cmpBlock.Block, cmpBlock.CmpHash)
				msg.Reply(bc.api.NewMessage("", 0, &reply))
			} else {
				if !bc.child.ProcEvent(msg) {
					msg.ReplyErr("BaseClient.EventLoop() ", types.ErrActionNotSupport)
				}
			}
		}
	}()
}

//CheckBlock
func (bc *BaseClient) CheckBlock(block *types.BlockDetail) error {
	//check parent
	if block.Block.Height <= 0 { //genesis block not check
		return nil
	}
	parent, err := bc.RequestBlock(block.Block.Height - 1)
	if err != nil {
		return err
	}
	//check base info
	if parent.Height+1 != block.Block.Height {
		return types.ErrBlockHeight
	}
	types.AssertConfig(bc.client)
	cfg := bc.client.GetConfig()
	if cfg.IsFork(block.Block.Height, "ForkCheckBlockTime") && parent.BlockTime > block.Block.BlockTime {
		return types.ErrBlockTime
	}
	//check parent hash
	if string(block.Block.GetParentHash()) != string(parent.Hash(cfg)) {
		return types.ErrParentHash
	}
	//check block size and tx count
	if cfg.IsFork(block.Block.Height, "ForkBlockCheck") {
		if block.Block.Size() > types.MaxBlockSize {
			return types.ErrBlockSize
		}
		if int64(len(block.Block.Txs)) > cfg.GetP(block.Block.Height).MaxTxNumber {
			return types.ErrManyTx
		}
	}
	//check by drivers
	err = bc.child.CheckBlock(parent, block)
	return err
}

//RequestTx Mempool
func (bc *BaseClient) RequestTx(listSize int, txHashList [][]byte) []*types.Transaction {
	if bc.client == nil {
		panic("bc not bind message queue.")
	}
	msg := bc.client.NewMessage("mempool", types.EventTxList, &types.TxHashList{Hashes: txHashList, Count: int64(listSize)})
	err := bc.client.Send(msg, true)
	if err != nil {
		return nil
	}
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return nil
	}
	return resp.GetData().(*types.ReplyTxList).GetTxs()
}

//RequestBlock
func (bc *BaseClient) RequestBlock(start int64) (*types.Block, error) {
	if bc.client == nil {
		panic("bc not bind message queue.")
	}
	reqblock := &types.ReqBlocks{Start: start, End: start, IsDetail: false, Pid: []string{""}}
	msg := bc.client.NewMessage("blockchain", types.EventGetBlocks, reqblock)
	err := bc.client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	blocks := resp.GetData().(*types.BlockDetails)
	return blocks.Items[0].Block, nil
}

//RequestLastBlock      block blockchain
func (bc *BaseClient) RequestLastBlock() (*types.Block, error) {
	if bc.client == nil {
		panic("client not bind message queue.")
	}
	msg := bc.client.NewMessage("blockchain", types.EventGetLastBlock, nil)
	err := bc.client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	block := resp.GetData().(*types.Block)
	return block, nil
}

//del mempool
func (bc *BaseClient) delMempoolTx(deltx []*types.Transaction) error {
	hashList := buildHashList(deltx)
	msg := bc.client.NewMessage("mempool", types.EventDelTxList, hashList)
	err := bc.client.Send(msg, true)
	if err != nil {
		return err
	}
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return err
	}
	if resp.GetData().(*types.Reply).GetIsOk() {
		return nil
	}
	return errors.New(string(resp.GetData().(*types.Reply).GetMsg()))
}

func buildHashList(deltx []*types.Transaction) *types.TxHashList {
	list := &types.TxHashList{}
	for i := 0; i < len(deltx); i++ {
		list.Hashes = append(list.Hashes, deltx[i].Hash())
	}
	return list
}

//WriteBlock  blockchain
func (bc *BaseClient) WriteBlock(prev []byte, block *types.Block) error {
	//  block         mempool
	rawtxs := make([]*types.Transaction, len(block.Txs))
	copy(rawtxs, block.Txs)

	blockdetail := &types.BlockDetail{Block: block}
	msg := bc.client.NewMessage("blockchain", types.EventAddBlockDetail, blockdetail)
	err := bc.client.Send(msg, true)
	if err != nil {
		return err
	}
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return err
	}
	blockdetail = resp.GetData().(*types.BlockDetail)
	// mempool
	deltx := diffTx(rawtxs, blockdetail.Block.Txs)
	if len(deltx) > 0 {
		err := bc.delMempoolTx(deltx)
		if err != nil {
			return err
		}
	}
	if blockdetail != nil {
		bc.SetCurrentBlock(blockdetail.Block)
	} else {
		return errors.New("block detail is nil")
	}
	return nil
}

// PreExecBlock      ,   raft, tendermint   , errReturn
func (bc *BaseClient) PreExecBlock(block *types.Block, errReturn bool) *types.Block {
	lastBlock, err := bc.RequestBlock(block.Height - 1)
	if err != nil {
		log.Error("PreExecBlock RequestBlock fail", "err", err)
		return nil
	}
	blockdetail, deltx, err := util.PreExecBlock(bc.client, lastBlock.StateHash, block, errReturn, false, true)
	if err != nil {
		log.Error("util.PreExecBlock fail", "err", err)
		return nil
	}
	if len(deltx) > 0 {
		err := bc.delMempoolTx(deltx)
		if err != nil {
			log.Error("PreExecBlock delMempoolTx fail", "err", err)
			return nil
		}
	}
	return blockdetail.Block
}

func diffTx(tx1, tx2 []*types.Transaction) (deltx []*types.Transaction) {
	txlist2 := make(map[string]bool)
	for _, tx := range tx2 {
		txlist2[string(tx.Hash())] = true
	}
	for _, tx := range tx1 {
		hash := string(tx.Hash())
		if _, ok := txlist2[hash]; !ok {
			deltx = append(deltx, tx)
		}
	}
	return deltx
}

//SetCurrentBlock
func (bc *BaseClient) SetCurrentBlock(b *types.Block) {
	bc.mulock.Lock()
	bc.currentBlock = b
	bc.mulock.Unlock()
}

//UpdateCurrentBlock
func (bc *BaseClient) UpdateCurrentBlock(b *types.Block) {
	bc.mulock.Lock()
	defer bc.mulock.Unlock()
	block, err := bc.RequestLastBlock()
	if err != nil {
		log.Error("UpdateCurrentBlock", "RequestLastBlock", err)
		return
	}
	bc.currentBlock = block
}

//GetCurrentBlock
func (bc *BaseClient) GetCurrentBlock() (b *types.Block) {
	bc.mulock.Lock()
	defer bc.mulock.Unlock()
	return bc.currentBlock
}

//GetCurrentHeight
func (bc *BaseClient) GetCurrentHeight() int64 {
	bc.mulock.Lock()
	start := bc.currentBlock.Height
	bc.mulock.Unlock()
	return start
}

//Lock
func (bc *BaseClient) Lock() {
	bc.mulock.Lock()
}

//Unlock
func (bc *BaseClient) Unlock() {
	bc.mulock.Unlock()
}

//ConsensusTicketMiner ...
func (bc *BaseClient) ConsensusTicketMiner(iscaughtup *types.IsCaughtUp) {
	if !atomic.CompareAndSwapInt32(&bc.isCaughtUp, 0, 1) {
		log.Info("ConsensusTicketMiner", "isCaughtUp", bc.isCaughtUp)
	} else {
		log.Info("ConsensusTicketMiner", "isCaughtUp", bc.isCaughtUp)
	}
}

//AddTxsToBlock
func (bc *BaseClient) AddTxsToBlock(block *types.Block, txs []*types.Transaction) []*types.Transaction {
	size := block.Size()
	max := types.MaxBlockSize - 100000 //  100K  ，
	currentCount := int64(len(block.Txs))
	types.AssertConfig(bc.client)
	cfg := bc.client.GetConfig()
	maxTx := cfg.GetP(block.Height).MaxTxNumber
	addedTx := make([]*types.Transaction, 0, len(txs))
	for i := 0; i < len(txs); i++ {
		txGroup, err := txs[i].GetTxGroup()
		if err != nil {
			continue
		}
		if txGroup == nil {
			currentCount++
			if currentCount > maxTx {
				return addedTx
			}
			size += txs[i].Size()
			if size > max {
				return addedTx
			}
			addedTx = append(addedTx, txs[i])
			block.Txs = append(block.Txs, txs[i])
		} else {
			currentCount += int64(len(txGroup.Txs))
			if currentCount > maxTx {
				return addedTx
			}
			for i := 0; i < len(txGroup.Txs); i++ {
				size += txGroup.Txs[i].Size()
			}
			if size > max {
				return addedTx
			}
			addedTx = append(addedTx, txGroup.Txs...)
			block.Txs = append(block.Txs, txGroup.Txs...)
		}
	}
	return addedTx
}

//CheckTxExpire    tx        ，        tx  ，    ticket     updateBlock
func (bc *BaseClient) CheckTxExpire(txs []*types.Transaction, height int64, blocktime int64) (transactions []*types.Transaction) {
	var txlist types.Transactions
	var hasTxExpire bool

	types.AssertConfig(bc.client)
	cfg := bc.client.GetConfig()
	for i := 0; i < len(txs); i++ {

		groupCount := txs[i].GroupCount
		if groupCount == 0 {
			if isExpire(cfg, txs[i:i+1], height, blocktime) {
				txs[i] = nil
				hasTxExpire = true
			}
			continue
		}

		//  GroupCount
		if i+int(groupCount) > len(txs) {
			continue
		}

		//
		grouptxs := txs[i : i+int(groupCount)]
		if isExpire(cfg, grouptxs, height, blocktime) {
			for j := i; j < i+int(groupCount); j++ {
				txs[j] = nil
				hasTxExpire = true
			}
		}
		i = i + int(groupCount) - 1
	}

	//
	if hasTxExpire {
		for _, tx := range txs {
			if tx != nil {
				txlist.Txs = append(txlist.Txs, tx)
			}
		}
		return txlist.GetTxs()
	}

	return txs
}

//          ，
func isExpire(cfg *types.DplatformOSConfig, txs []*types.Transaction, height int64, blocktime int64) bool {
	for _, tx := range txs {
		if height > 0 && blocktime > 0 && tx.IsExpire(cfg, height, blocktime) {
			log.Debug("isExpire", "height", height, "blocktime", blocktime, "hash", common.ToHex(tx.Hash()), "Expire", tx.Expire)
			return true
		}
	}
	return false
}

//CmpBestBlock
//height,BlockTime,ParentHash
//    newBlock        true，    false
func (bc *BaseClient) CmpBestBlock(newBlock *types.Block, cmpHash []byte) bool {

	cfg := bc.client.GetConfig()
	curBlock := bc.GetCurrentBlock()

	//             ,
	if bytes.Equal(cmpHash, curBlock.Hash(cfg)) {
		if curBlock.GetHeight() == newBlock.GetHeight() && curBlock.BlockTime == newBlock.BlockTime && bytes.Equal(newBlock.GetParentHash(), curBlock.GetParentHash()) {
			return bc.child.CmpBestBlock(newBlock, curBlock)
		}
		return false
	}
	//             ，   blockchain    cmpHash   block
	block, err := bc.ReqBlockByHash(cmpHash)
	if err != nil {
		log.Error("CmpBestBlock:RequestBlockByHash", "Hash", common.ToHex(cmpHash), "err", err)
		return false
	}

	if block.GetHeight() == newBlock.GetHeight() && block.BlockTime == newBlock.BlockTime && bytes.Equal(block.GetParentHash(), newBlock.GetParentHash()) {
		return bc.child.CmpBestBlock(newBlock, block)
	}
	return false
}

//ReqBlockByHash     hash
func (bc *BaseClient) ReqBlockByHash(hash []byte) (*types.Block, error) {
	if bc.client == nil {
		panic("bc not bind message queue.")
	}
	hashes := types.ReqHashes{}
	hashes.Hashes = append(hashes.Hashes, hash)
	msg := bc.client.NewMessage("blockchain", types.EventGetBlockByHashes, &hashes)
	err := bc.client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	blocks := resp.GetData().(*types.BlockDetails)
	if len(blocks.Items) == 1 && blocks.Items[0] != nil {
		return blocks.Items[0].Block, nil
	}
	return nil, types.ErrHashNotExist
}
