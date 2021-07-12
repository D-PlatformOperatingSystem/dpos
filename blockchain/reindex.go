// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"strconv"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/common/version"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//UpgradeChain   localdb
func (chain *BlockChain) UpgradeChain() {
	meta, err := chain.blockStore.GetUpgradeMeta()
	if err != nil {
		panic(err)
	}
	curheight := chain.GetBlockHeight()
	if curheight == -1 {
		meta = &types.UpgradeMeta{
			Version: version.GetLocalDBVersion(),
		}
		err = chain.blockStore.SetUpgradeMeta(meta)
		if err != nil {
			panic(err)
		}
		return
	}
	start := meta.Height
	v1, _, _ := getLocalDBVersion()
	if chain.needReIndex(meta) {
		//        index，   del all keys
		//reindex     ，         meta
		if v1 == 1 {
			if !meta.Starting {
				chainlog.Info("begin del all keys")
				chain.blockStore.delAllKeys()
				chainlog.Info("end del all keys")
			}
			chain.reIndex(start, curheight)
		} else if v1 == 2 {
			//      isRecordBlockSequence  ，    seq
			var isSeq bool
			lastSequence, err := chain.blockStore.LoadBlockLastSequence()
			if err != nil || lastSequence < 0 {
				isSeq = false
			} else {
				curheight = lastSequence
				isSeq = true
			}
			chain.reIndexForTable(start, curheight, isSeq)
		}
		meta := &types.UpgradeMeta{
			Starting: false,
			Version:  version.GetLocalDBVersion(),
			Height:   0,
		}
		err = chain.blockStore.SetUpgradeMeta(meta)
		if err != nil {
			panic(err)
		}
	}
}

func (chain *BlockChain) reIndex(start, end int64) {
	for i := start; i <= end; i++ {
		err := chain.reIndexOne(i)
		if err != nil {
			panic(err)
		}
	}
}

func (chain *BlockChain) needReIndex(meta *types.UpgradeMeta) bool {
	if meta.Starting { //  index
		return true
	}
	v1 := meta.Version
	v2 := version.GetLocalDBVersion()
	v1arr := strings.Split(v1, ".")
	v2arr := strings.Split(v2, ".")
	if len(v1arr) != 3 || len(v2arr) != 3 {
		panic("upgrade meta version error")
	}
	//
	v1Value, err := strconv.Atoi(v1arr[0])
	if err != nil {
		panic("upgrade meta strconv.Atoi error")
	}
	v2Value, err := strconv.Atoi(v2arr[0])
	if err != nil {
		panic("upgrade meta strconv.Atoi error")
	}
	if v1arr[0] != v2arr[0] && v2Value > v1Value {
		return true
	}
	return false
}

func (chain *BlockChain) reIndexOne(height int64) error {
	newbatch := chain.blockStore.NewBatch(false)
	blockdetail, err := chain.GetBlock(height)
	if err != nil {
		chainlog.Error("reindexone.GetBlock", "err", err)
		panic(err)
	}
	if height%1000 == 0 {
		chainlog.Info("reindex -> ", "height", height, "lastheight", chain.GetBlockHeight())
	}
	//  tx   db (newbatch, blockdetail)
	err = chain.blockStore.AddTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("reIndexOne indexTxs:", "height", blockdetail.Block.Height, "err", err)
		panic(err)
	}
	meta := &types.UpgradeMeta{
		Starting: true,
		Version:  version.GetLocalDBVersion(),
		Height:   height + 1,
	}
	newbatch.Set(version.LocalDBMeta, types.Encode(meta))
	return newbatch.Write()
}

func (chain *BlockChain) reIndexForTable(start, end int64, isSeq bool) {
	for i := start; i <= end; i++ {
		err := chain.reIndexForTableOne(i, end, isSeq)
		if err != nil {
			panic(err)
		}
	}
	chainlog.Info("reindex:reIndexForTable:complete")
}

//  table     block header body  paratx  ，
//       bodyPrefix, headerPrefix, heightToHeaderPrefix,key
//       seq  height   block
func (chain *BlockChain) reIndexForTableOne(index int64, lastindex int64, isSeq bool) error {
	newbatch := chain.blockStore.NewBatch(false)
	var blockdetail *types.BlockDetail
	var err error
	blockOptType := types.AddBlock

	if isSeq {
		blockdetail, blockOptType, err = chain.blockStore.loadBlockBySequenceOld(index)
	} else {
		blockdetail, err = chain.blockStore.loadBlockByHeightOld(index)
	}
	if err != nil {
		chainlog.Error("reindex:reIndexForTable:load Block Err", "index", index, "isSeq", isSeq, "err", err)
		panic(err)
	}
	height := blockdetail.Block.GetHeight()
	hash := blockdetail.Block.Hash(chain.client.GetConfig())
	curHeight := chain.GetBlockHeight()

	//   add        ，del
	if blockOptType == types.AddBlock {
		if index%1000 == 0 {
			chainlog.Info("reindex -> ", "index", index, "lastindex", lastindex, "isSeq", isSeq)
		}

		saveReceipt := true
		//   localdb,        blockReceipt
		if chain.client.GetConfig().IsEnable("reduceLocaldb") && curHeight-SafetyReduceHeight > height {
			saveReceipt = false
		}
		//  table    header body  paratx
		err = chain.blockStore.saveBlockForTable(newbatch, blockdetail, true, saveReceipt)
		if err != nil {
			chainlog.Error("reindex:reIndexForTable", "height", height, "isSeq", isSeq, "err", err)
			panic(err)
		}

		//        header body key
		newbatch.Delete(calcHashToBlockHeaderKey(hash))
		newbatch.Delete(calcHashToBlockBodyKey(hash))
		newbatch.Delete(calcHeightToBlockHeaderKey(height))

		//   localdb
		if chain.client.GetConfig().IsEnable("reduceLocaldb") && curHeight-SafetyReduceHeight > height {
			chain.reduceIndexTx(newbatch, blockdetail.GetBlock())
			newbatch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data: height}))
		}
	} else {
		parakvs, _ := delParaTxTable(chain.blockStore.db, height)
		for _, kv := range parakvs {
			if len(kv.GetKey()) != 0 && kv.GetValue() == nil {
				newbatch.Delete(kv.GetKey())
			}
		}
		//   localdb，      ，    tx   ，         ，           tx
		if chain.client.GetConfig().IsEnable("reduceLocaldb") && curHeight-SafetyReduceHeight > height {
			chain.deleteTx(newbatch, blockdetail.GetBlock())
		}
	}

	meta := &types.UpgradeMeta{
		Starting: true,
		Version:  version.GetLocalDBVersion(),
		Height:   index + 1,
	}
	newbatch.Set(version.LocalDBMeta, types.Encode(meta))

	return newbatch.Write()
}

//       V1,V2,V3
func getLocalDBVersion() (int, int, int) {

	version := version.GetLocalDBVersion()
	vArr := strings.Split(version, ".")
	if len(vArr) != 3 {
		panic("getLocalDBVersion version error")
	}
	v1, _ := strconv.Atoi(vArr[0])
	v2, _ := strconv.Atoi(vArr[1])
	v3, _ := strconv.Atoi(vArr[2])

	return v1, v2, v3
}
