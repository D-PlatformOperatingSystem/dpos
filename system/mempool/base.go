// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

var mlog = log.New("module", "mempool.base")

//Mempool mempool
type Mempool struct {
	proxyMtx          sync.Mutex
	in                chan *queue.Message
	out               <-chan *queue.Message
	client            queue.Client
	header            *types.Header
	sync              bool
	cfg               *types.Mempool
	poolHeader        chan struct{}
	isclose           int32
	wg                sync.WaitGroup
	done              chan struct{}
	removeBlockTicket *time.Ticker
	cache             *txCache
}

//GetSync     mempool
func (mem *Mempool) getSync() bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.sync
}

//NewMempool   mempool
func NewMempool(cfg *types.Mempool) *Mempool {
	pool := &Mempool{}
	if cfg.MaxTxNumPerAccount == 0 {
		cfg.MaxTxNumPerAccount = maxTxNumPerAccount
	}
	if cfg.MaxTxLast == 0 {
		cfg.MaxTxLast = maxTxLast
	}
	if cfg.PoolCacheSize == 0 {
		cfg.PoolCacheSize = poolCacheSize
	}
	pool.in = make(chan *queue.Message)
	pool.out = make(<-chan *queue.Message)
	pool.done = make(chan struct{})
	pool.cfg = cfg
	pool.poolHeader = make(chan struct{}, 2)
	pool.removeBlockTicket = time.NewTicker(time.Minute)
	pool.cache = newCache(cfg.MaxTxNumPerAccount, cfg.MaxTxLast, cfg.PoolCacheSize)
	return pool
}

//Close   mempool
func (mem *Mempool) Close() {
	if mem.isClose() {
		return
	}
	atomic.StoreInt32(&mem.isclose, 1)
	close(mem.done)
	if mem.client != nil {
		mem.client.Close()
	}
	mem.removeBlockTicket.Stop()
	mlog.Info("mempool module closing")
	mem.wg.Wait()
	mlog.Info("mempool module closed")
}

//SetQueueClient    mempool
func (mem *Mempool) SetQueueClient(client queue.Client) {
	mem.client = client
	mem.client.Sub("mempool")
	mem.wg.Add(1)
	go mem.pollLastHeader()
	mem.wg.Add(1)
	go mem.checkSync()
	mem.wg.Add(1)
	go mem.removeBlockedTxs()

	mem.wg.Add(1)
	go mem.eventProcess()
}

// Size   mempool txCache
func (mem *Mempool) Size() int {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.Size()
}

// SetMinFee
func (mem *Mempool) SetMinFee(fee int64) {
	mem.proxyMtx.Lock()
	mem.cfg.MinTxFeeRate = fee
	mem.proxyMtx.Unlock()
}

//SetQueueCache
func (mem *Mempool) SetQueueCache(qcache QueueCache) {
	mem.cache.SetQueueCache(qcache)
}

// GetTxList  txCache        tx
func (mem *Mempool) getTxList(filterList *types.TxHashList) (txs []*types.Transaction) {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	count := filterList.GetCount()
	dupMap := make(map[string]bool)
	for i := 0; i < len(filterList.GetHashes()); i++ {
		dupMap[string(filterList.GetHashes()[i])] = true
	}
	return mem.filterTxList(count, dupMap, false)
}

func (mem *Mempool) filterTxList(count int64, dupMap map[string]bool, isAll bool) (txs []*types.Transaction) {
	height := mem.header.GetHeight()
	blocktime := mem.header.GetBlockTime()
	types.AssertConfig(mem.client)
	cfg := mem.client.GetConfig()
	mem.cache.Walk(int(count), func(tx *Item) bool {
		if len(dupMap) > 0 {
			if _, ok := dupMap[string(tx.Value.Hash())]; ok {
				return true
			}
		}
		if isExpired(cfg, tx, height, blocktime) && !isAll {
			return true
		}
		txs = append(txs, tx.Value)
		return true
	})
	return txs
}

// RemoveTxs  mempool     Hash txs
func (mem *Mempool) RemoveTxs(hashList *types.TxHashList) error {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	for _, hash := range hashList.Hashes {
		exist := mem.cache.Exist(string(hash))
		if exist {
			mem.cache.Remove(string(hash))
		}
	}
	return nil
}

// PushTx      mempool，     （error）
func (mem *Mempool) PushTx(tx *types.Transaction) error {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	err := mem.cache.Push(tx)
	return err
}

//  setHeader  mempool.header
func (mem *Mempool) setHeader(h *types.Header) {
	mem.proxyMtx.Lock()
	mem.header = h
	mem.proxyMtx.Unlock()
}

// GetHeader   Mempool.header
func (mem *Mempool) GetHeader() *types.Header {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.header
}

//IsClose     mempool
func (mem *Mempool) isClose() bool {
	return atomic.LoadInt32(&mem.isclose) == 1
}

// GetLastHeader   LastHeader height blockTime
func (mem *Mempool) GetLastHeader() (interface{}, error) {
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	msg := mem.client.NewMessage("blockchain", types.EventGetLastHeader, nil)
	err := mem.client.Send(msg, true)
	if err != nil {
		mlog.Error("blockchain closed", "err", err.Error())
		return nil, err
	}
	return mem.client.Wait(msg)
}

// GetAccTxs           （  ）
func (mem *Mempool) GetAccTxs(addrs *types.ReqAddrs) *types.TransactionDetails {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.GetAccTxs(addrs)
}

// TxNumOfAccount      mempool
func (mem *Mempool) TxNumOfAccount(addr string) int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return int64(mem.cache.TxNumOfAccount(addr))
}

// GetLatestTx          mempool
func (mem *Mempool) GetLatestTx() []*types.Transaction {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.GetLatestTx()
}

// GetTotalCacheBytes
func (mem *Mempool) GetTotalCacheBytes() int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.qcache.GetCacheBytes()
}

// pollLastHeader          LastHeader，       ，
func (mem *Mempool) pollLastHeader() {
	defer mem.wg.Done()
	defer func() {
		mlog.Info("pollLastHeader quit")
		mem.poolHeader <- struct{}{}
	}()
	for {
		if mem.isClose() {
			return
		}
		lastHeader, err := mem.GetLastHeader()
		if err != nil {
			mlog.Error(err.Error())
			time.Sleep(time.Second)
			continue
		}
		h := lastHeader.(*queue.Message).Data.(*types.Header)
		mem.setHeader(h)
		return
	}
}

func (mem *Mempool) removeExpired() {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	types.AssertConfig(mem.client)
	mem.cache.removeExpiredTx(mem.client.GetConfig(), mem.header.GetHeight(), mem.header.GetBlockTime())
}

// removeBlockedTxs   1
func (mem *Mempool) removeBlockedTxs() {
	defer mem.wg.Done()
	defer mlog.Info("RemoveBlockedTxs quit")
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	for {
		select {
		case <-mem.removeBlockTicket.C:
			if mem.isClose() {
				return
			}
			mem.removeExpired()
		case <-mem.done:
			return
		}
	}
}

// RemoveTxsOfBlock   mempool   Blockchain   tx
func (mem *Mempool) RemoveTxsOfBlock(block *types.Block) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	for _, tx := range block.Txs {
		hash := tx.Hash()
		exist := mem.cache.Exist(string(hash))
		if exist {
			mem.cache.Remove(string(hash))
		}
	}
	return true
}
func (mem *Mempool) getCacheFeeRate() int64 {
	if mem.cache.qcache == nil {
		return 0
	}
	feeRate := mem.cache.qcache.GetProperFee()

	//
	unitFee := mem.cfg.MinTxFeeRate
	if unitFee != 0 && feeRate%unitFee > 0 {
		feeRate = (feeRate/unitFee + 1) * unitFee
	}
	if feeRate > mem.cfg.MaxTxFeeRate {
		feeRate = mem.cfg.MaxTxFeeRate
	}
	return feeRate
}

// GetProperFeeRate
func (mem *Mempool) GetProperFeeRate(req *types.ReqProperFee) int64 {
	if req == nil || req.TxCount == 0 {
		req = &types.ReqProperFee{TxCount: 20}
	}
	if req.TxSize == 0 {
		req.TxSize = 10240
	}
	feeRate := mem.getCacheFeeRate()
	if mem.cfg.IsLevelFee {
		levelFeeRate := mem.getLevelFeeRate(mem.cfg.MinTxFeeRate, req.TxCount, req.TxSize)
		if levelFeeRate > feeRate {
			feeRate = levelFeeRate
		}
	}
	return feeRate
}

// getLevelFeeRate            ,       count, size
func (mem *Mempool) getLevelFeeRate(baseFeeRate int64, appendCount, appendSize int32) int64 {
	var feeRate int64
	sumByte := mem.GetTotalCacheBytes() + int64(appendSize)
	types.AssertConfig(mem.client)
	cfg := mem.client.GetConfig()
	maxTxNumber := cfg.GetP(mem.Height()).MaxTxNumber
	memSize := mem.Size()
	switch {
	case sumByte >= int64(types.MaxBlockSize/20) || int64(memSize+int(appendCount)) >= maxTxNumber/2:
		feeRate = 100 * baseFeeRate
	case sumByte >= int64(types.MaxBlockSize/100) || int64(memSize+int(appendCount)) >= maxTxNumber/10:
		feeRate = 10 * baseFeeRate
	default:
		feeRate = baseFeeRate
	}
	if feeRate > mem.cfg.MaxTxFeeRate {
		feeRate = mem.cfg.MaxTxFeeRate
	}
	return feeRate
}

// Mempool.DelBlock              mempool
func (mem *Mempool) delBlock(block *types.Block) {
	if len(block.Txs) <= 0 {
		return
	}
	blkTxs := block.Txs
	header := mem.GetHeader()
	types.AssertConfig(mem.client)
	cfg := mem.client.GetConfig()
	for i := 0; i < len(blkTxs); i++ {
		tx := blkTxs[i]
		//    ticket            ，  actionName miner
		if i == 0 && tx.ActionName() == types.MinerAction {
			continue
		}
		groupCount := int(tx.GetGroupCount())
		if groupCount > 1 && i+groupCount <= len(blkTxs) {
			group := types.Transactions{Txs: blkTxs[i : i+groupCount]}
			tx = group.Tx()
			i = i + groupCount - 1
		}
		err := tx.Check(cfg, header.GetHeight(), mem.cfg.MinTxFeeRate, mem.cfg.MaxTxFee)
		if err != nil {
			continue
		}
		if !mem.checkExpireValid(tx) {
			continue
		}
		err = mem.PushTx(tx)
		if err != nil {
			mlog.Error("mem", "push tx err", err)
		}
	}
}

// Height
func (mem *Mempool) Height() int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	if mem.header == nil {
		return -1
	}
	return mem.header.GetHeight()
}

// Wait wait mempool ready
func (mem *Mempool) Wait() {
	<-mem.poolHeader
	//wait sync
	<-mem.poolHeader
}

// SendTxToP2P  "p2p"
func (mem *Mempool) sendTxToP2P(tx *types.Transaction) {
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	msg := mem.client.NewMessage("p2p", types.EventTxBroadcast, tx)
	err := mem.client.Send(msg, false)
	if err != nil {
		mlog.Error("tx sent to p2p", "tx.Hash", common.ToHex(tx.Hash()))
		return
	}
	//mlog.Debug("tx sent to p2p", "tx.Hash", common.ToHex(tx.Hash()))
}

// Mempool.checkSync     mempool
func (mem *Mempool) checkSync() {
	defer func() {
		mlog.Info("getsync quit")
		mem.poolHeader <- struct{}{}
	}()
	defer mem.wg.Done()
	if mem.getSync() {
		return
	}
	if mem.cfg.ForceAccept {
		mem.setSync(true)
	}
	for {
		if mem.isClose() {
			return
		}
		if mem.client == nil {
			panic("client not bind message queue.")
		}
		msg := mem.client.NewMessage("blockchain", types.EventIsSync, nil)
		err := mem.client.Send(msg, true)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		resp, err := mem.client.Wait(msg)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		if resp.GetData().(*types.IsCaughtUp).GetIscaughtup() {
			mem.setSync(true)
			return
		}
		time.Sleep(time.Second)
		continue
	}
}

func (mem *Mempool) setSync(status bool) {
	mem.proxyMtx.Lock()
	mem.sync = status
	mem.proxyMtx.Unlock()
}

// getTxListByHash  qcache  SHashTxCache   hash   tx
func (mem *Mempool) getTxListByHash(hashList *types.ReqTxHashList) *types.ReplyTxList {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()

	var replyTxList types.ReplyTxList

	//   hash   tx
	if hashList.GetIsShortHash() {
		for _, sHash := range hashList.GetHashes() {
			tx := mem.cache.GetSHashTxCache(sHash)
			replyTxList.Txs = append(replyTxList.Txs, tx)
		}
		return &replyTxList
	}
	//  hash   tx
	for _, hash := range hashList.GetHashes() {
		tx := mem.cache.getTxByHash(hash)
		replyTxList.Txs = append(replyTxList.Txs, tx)
	}
	return &replyTxList
}
