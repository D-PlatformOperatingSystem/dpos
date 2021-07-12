// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"sync/atomic"
	"time"

	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// ReduceChain   chain
func (chain *BlockChain) ReduceChain() {
	height := chain.GetBlockHeight()
	cfg := chain.client.GetConfig()
	if cfg.IsEnable("reduceLocaldb") {
		//   localdb
		chain.InitReduceLocalDB(height)
		chain.reducewg.Add(1)
		go chain.ReduceLocalDB()
	} else {
		flagHeight, _ := chain.blockStore.loadFlag(types.ReduceLocaldbHeight) //    reduceLocaldb，
		if flagHeight != 0 {
			panic("toml config disable reduce localdb, but database enable reduce localdb")
		}
	}
}

// InitReduceLocalDB
func (chain *BlockChain) InitReduceLocalDB(height int64) {
	flag, err := chain.blockStore.loadFlag(types.FlagReduceLocaldb)
	if err != nil {
		panic(err)
	}
	flagHeight, err := chain.blockStore.loadFlag(types.ReduceLocaldbHeight)
	if err != nil {
		panic(err)
	}
	if flag == 0 {
		endHeight := height - SafetyReduceHeight //
		if endHeight > flagHeight {
			chainlog.Info("start reduceLocaldb", "start height", flagHeight, "end height", endHeight)
			chain.walkOver(flagHeight, endHeight, false, chain.reduceBodyInit,
				func(batch dbm.Batch, height int64) {
					height++
					batch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data: height}))
				})
			// CompactRange
			chainlog.Info("reduceLocaldb start compact db")
			err = chain.blockStore.db.CompactRange(nil, nil)
			chainlog.Info("reduceLocaldb end compact db", "error", err)
		}
		chain.blockStore.saveReduceLocaldbFlag()
	}
}

// walkOver walk over
func (chain *BlockChain) walkOver(start, end int64, sync bool, fn func(batch dbm.Batch, height int64),
	fnflag func(batch dbm.Batch, height int64)) {
	//
	const batchDataSize = 1024 * 1024 * 10
	newbatch := chain.blockStore.NewBatch(sync)
	for i := start; i <= end; i++ {
		fn(newbatch, i)
		if newbatch.ValueSize() > batchDataSize {
			//
			fnflag(newbatch, i)
			dbm.MustWrite(newbatch)
			newbatch.Reset()
			chainlog.Info("reduceLocaldb", "height", i)
		}
	}
	if newbatch.ValueSize() > 0 {
		//
		fnflag(newbatch, end)
		dbm.MustWrite(newbatch)
		newbatch.Reset()
		chainlog.Info("reduceLocaldb end", "height", end)
	}
}

// reduceBodyInit  body  receipt    ； TxHashPerfix key TxResult  receipt tx
func (chain *BlockChain) reduceBodyInit(batch dbm.Batch, height int64) {
	blockDetail, err := chain.blockStore.LoadBlockByHeight(height)
	if err == nil {
		cfg := chain.client.GetConfig()
		kvs, err := delBlockReceiptTable(chain.blockStore.db, height, blockDetail.Block.Hash(cfg))
		if err != nil {
			chainlog.Debug("reduceBody delBlockReceiptTable", "height", height, "error", err)
			return
		}
		for _, kv := range kvs {
			if kv.GetValue() == nil {
				batch.Delete(kv.GetKey())
			}
		}
		chain.reduceIndexTx(batch, blockDetail.GetBlock())
	}
}

// reduceIndexTx        hash-TX
func (chain *BlockChain) reduceIndexTx(batch dbm.Batch, block *types.Block) {
	cfg := chain.client.GetConfig()
	Txs := block.GetTxs()
	for index, tx := range Txs {
		hash := tx.Hash()
		//     quickIndex      hash     ，    ，
		if cfg.IsEnable("quickIndex") {
			batch.Delete(hash)
		}

		//           txresult,          txresult
		txresult := &types.TxResult{
			Height:     block.Height,
			Index:      int32(index),
			Blocktime:  block.BlockTime,
			ActionName: tx.ActionName(),
		}
		batch.Set(cfg.CalcTxKey(hash), cfg.CalcTxKeyValue(txresult))
	}
}

func (chain *BlockChain) deleteTx(batch dbm.Batch, block *types.Block) {
	cfg := chain.client.GetConfig()
	Txs := block.GetTxs()
	for _, tx := range Txs {
		batch.Delete(cfg.CalcTxKey(tx.Hash()))
	}
}

// reduceReceipts   receipts
func reduceReceipts(src *types.BlockBody) []*types.ReceiptData {
	dst := src.Clone()
	for i := 0; i < len(dst.Receipts); i++ {
		for j := 0; j < len(dst.Receipts[i].Logs); j++ {
			if dst.Receipts[i].Logs[j] != nil {
				if dst.Receipts[i].Logs[j].Ty == types.TyLogErr { //
					continue
				}
				dst.Receipts[i].Logs[j].Log = nil
			}
		}
	}
	return dst.Receipts
}

// ReduceLocalDB     localdb
func (chain *BlockChain) ReduceLocalDB() {
	defer chain.reducewg.Done()

	flagHeight, err := chain.blockStore.loadFlag(types.ReduceLocaldbHeight)
	if err != nil {
		panic(err)
	}
	if flagHeight < 0 {
		flagHeight = 0
	}
	// 10s          reduce localdb
	checkTicker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-chain.quit:
			return
		case <-checkTicker.C:
			flagHeight = chain.TryReduceLocalDB(flagHeight, 100)
		}
	}
}

//TryReduceLocalDB TryReduce try reduce
func (chain *BlockChain) TryReduceLocalDB(flagHeight int64, rangeHeight int64) (newHeight int64) {
	if rangeHeight <= 0 {
		rangeHeight = 100
	}
	height := chain.GetBlockHeight()
	safetyHeight := height - ReduceHeight
	if safetyHeight/rangeHeight > flagHeight/rangeHeight { //   rangeHeight
		sync := true
		if atomic.LoadInt32(&chain.isbatchsync) == 0 {
			sync = false
		}
		chain.walkOver(flagHeight, safetyHeight, sync, chain.reduceBody,
			func(batch dbm.Batch, height int64) {
				//           ，
				height++
				batch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data: height}))
			})
		flagHeight = safetyHeight + 1
		chainlog.Debug("reduceLocaldb ticker", "current height", flagHeight)
		return flagHeight
	}
	return flagHeight
}

// reduceBody  body  receipt    ；
func (chain *BlockChain) reduceBody(batch dbm.Batch, height int64) {
	hash, err := chain.blockStore.GetBlockHashByHeight(height)
	if err == nil {
		kvs, err := delBlockReceiptTable(chain.blockStore.db, height, hash)
		if err != nil {
			chainlog.Debug("reduceBody delBlockReceiptTable", "height", height, "error", err)
			return
		}
		for _, kv := range kvs {
			if kv.GetValue() == nil {
				batch.Delete(kv.GetKey())
			}
		}
	}
}
