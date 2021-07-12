// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"syscall"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// Rollbackblock chain Rollbackblock
func (chain *BlockChain) Rollbackblock() {
	tipnode := chain.bestChain.Tip()
	if chain.cfg.RollbackBlock > 0 {
		if chain.NeedRollback(tipnode.height, chain.cfg.RollbackBlock) {
			chainlog.Info("chain rollback start")
			chain.Rollback()
			chainlog.Info("chain rollback end")
		}
		syscall.Exit(0)
	}
}

// NeedRollback need Rollback
func (chain *BlockChain) NeedRollback(curHeight, rollHeight int64) bool {
	if curHeight <= rollHeight {
		chainlog.Info("curHeight is small than rollback height, no need rollback")
		return false
	}
	cfg := chain.client.GetConfig()
	kvmvccMavlFork := cfg.GetDappFork("store-kvmvccmavl", "ForkKvmvccmavl")
	if curHeight >= kvmvccMavlFork+10000 && rollHeight <= kvmvccMavlFork {
		chainlog.Info("because ForkKvmvccmavl", "current height", curHeight, "not support rollback to", rollHeight)
		return false
	}
	return true
}

// Rollback chain Rollback
func (chain *BlockChain) Rollback() {
	cfg := chain.client.GetConfig()
	//     tip
	tipnode := chain.bestChain.Tip()
	startHeight := tipnode.height
	for i := startHeight; i > chain.cfg.RollbackBlock; i-- {
		blockdetail, err := chain.blockStore.LoadBlockByHeight(i)
		if err != nil {
			panic(fmt.Sprintln("rollback LoadBlockByHeight err :", err))
		}
		if chain.cfg.RollbackSave { //
			lastHeightSave := false
			if i == startHeight {
				lastHeightSave = true
			}
			err = chain.WriteBlockToDbTemp(blockdetail.Block, lastHeightSave)
			if err != nil {
				panic(fmt.Sprintln("rollback WriteBlockToDbTemp fail", "height", blockdetail.Block.Height, "error ", err))
			}
		}
		sequence := int64(-1)
		if chain.isParaChain {
			//       seq
			sequence, err = chain.ProcGetMainSeqByHash(blockdetail.Block.Hash(cfg))
			if err != nil {
				chainlog.Error("chain rollback get main seq fail", "height: ", i, "err", err, "hash", common.ToHex(blockdetail.Block.Hash(cfg)))
			}
		}
		err = chain.disBlock(blockdetail, sequence)
		if err != nil {
			panic(fmt.Sprintln("rollback block fail ", "height", blockdetail.Block.Height, "blockHash:", common.ToHex(blockdetail.Block.Hash(cfg))))
		}
		//   storedb
		chain.sendDelStore(blockdetail.Block.StateHash, blockdetail.Block.Height)
		chainlog.Info("chain rollback ", "height: ", i, "blockheight", blockdetail.Block.Height, "hash", common.ToHex(blockdetail.Block.Hash(cfg)), "state hash", common.ToHex(blockdetail.Block.StateHash))
	}
}

//   blocks
func (chain *BlockChain) disBlock(blockdetail *types.BlockDetail, sequence int64) error {
	var lastSequence int64
	cfg := chain.client.GetConfig()

	//    block
	newbatch := chain.blockStore.NewBatch(true)

	// db   tx
	err := chain.blockStore.DelTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("disBlock DelTxs:", "height", blockdetail.Block.Height, "err", err)
		return err
	}

	// db   block
	lastSequence, err = chain.blockStore.DelBlock(newbatch, blockdetail, sequence)
	if err != nil {
		chainlog.Error("disBlock DelBlock:", "height", blockdetail.Block.Height, "err", err)
		return err
	}
	db.MustWrite(newbatch)

	//        header
	chain.blockStore.UpdateHeight()
	chain.blockStore.UpdateLastBlock(blockdetail.Block.ParentHash)

	//    ，mempool     block
	err = chain.SendDelBlockEvent(blockdetail)
	if err != nil {
		chainlog.Error("disBlock SendDelBlockEvent", "err", err)
	}

	//      block
	chain.DelCacheBlock(blockdetail.Block.Height, blockdetail.Block.Hash(cfg))

	//         isRecordBlockSequence   enablePushSubscribe
	if chain.isRecordBlockSequence && chain.enablePushSubscribe {
		chain.push.UpdateSeq(lastSequence)
		chainlog.Debug("isRecordBlockSequence", "lastSequence", lastSequence, "height", blockdetail.Block.Height)
	}

	return nil
}

//   store    ，    kvmvcc
func (chain *BlockChain) sendDelStore(hash []byte, height int64) {
	storeDel := &types.StoreDel{StateHash: hash, Height: height}
	msg := chain.client.NewMessage("store", types.EventStoreDel, storeDel)
	err := chain.client.Send(msg, true)
	if err != nil {
		chainlog.Debug("sendDelStoreEvent -->>store", "err", err)
	}
	_, err = chain.client.Wait(msg)
	if err != nil {
		panic(fmt.Sprintln("sendDelStore", err))
	}
}
