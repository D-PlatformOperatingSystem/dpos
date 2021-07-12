// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"strings"

	"fmt"

	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/common/version"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// Upgrade   localDB storeDB
func (chain *BlockChain) Upgrade() {
	chainlog.Info("chain upgrade start")
	chain.UpgradeChain()
	chainlog.Info("storedb upgrade start")
	chain.UpgradeStore()
	chainlog.Info("upgrade all dapp")
	chain.UpgradePlugin()
	chainlog.Info("chain reduce start")
	chain.ReduceChain()
}

// UpgradePlugin
func (chain *BlockChain) UpgradePlugin() {
	msg := chain.client.NewMessage("execs", types.EventUpgrade, nil)
	err := chain.client.Send(msg, true)
	if err != nil {
		panic(err)
	}
	resp, err := chain.client.Wait(msg)
	if err != nil {
		panic(err)
	}
	if resp == nil {
		return
	}
	kv := resp.GetData().(*types.LocalDBSet)
	if kv == nil || len(kv.KV) == 0 {
		return
	}

	chain.blockStore.mustSaveKvset(kv)
}

//UpgradeStore   storedb
func (chain *BlockChain) UpgradeStore() {
	meta, err := chain.blockStore.GetStoreUpgradeMeta()
	if err != nil {
		panic(err)
	}
	curheight := chain.GetBlockHeight()
	if curheight == -1 {
		meta = &types.UpgradeMeta{
			Version: version.GetStoreDBVersion(),
		}
		err = chain.blockStore.SetStoreUpgradeMeta(meta)
		if err != nil {
			panic(err)
		}
	}
	if chain.NeedReExec(meta) {
		//        index，   del all keys
		if !meta.Starting && chain.cfg.EnableReExecLocal {
			chainlog.Info("begin del all keys")
			chain.blockStore.delAllKeys()
			chainlog.Info("end del all keys")
		}
		start := meta.Height
		//reExecBlock     ，         meta
		chain.ReExecBlock(start, curheight)
		meta := &types.UpgradeMeta{
			Starting: false,
			Version:  version.GetStoreDBVersion(),
			Height:   0,
		}
		err = chain.blockStore.SetStoreUpgradeMeta(meta)
		if err != nil {
			panic(err)
		}
	}
}

// ReExecBlock
func (chain *BlockChain) ReExecBlock(startHeight, curHeight int64) {
	var prevStateHash []byte
	if startHeight > 0 {
		blockdetail, err := chain.GetBlock(startHeight - 1)
		if err != nil {
			panic(fmt.Sprintf("get height=%d err, this not allow fail", startHeight-1))
		}
		prevStateHash = blockdetail.Block.StateHash
	}

	for i := startHeight; i <= curHeight; i++ {
		blockdetail, err := chain.GetBlock(i)
		if err != nil {
			panic(fmt.Sprintf("get height=%d err, this not allow fail", i))
		}
		block := blockdetail.Block
		err = execBlockUpgrade(chain.client, prevStateHash, block, false)
		if err != nil {
			panic(fmt.Sprintf("execBlockEx height=%d err=%s, this not allow fail", i, err.Error()))
		}

		if chain.cfg.EnableReExecLocal {
			//   tx   db
			newbatch := chain.blockStore.NewBatch(false)
			err = chain.blockStore.AddTxs(newbatch, blockdetail)
			if err != nil {
				panic(fmt.Sprintf("execBlockEx connectBlock readd Txs fail height=%d err=%s, this not allow fail", i, err.Error()))
			}
			dbm.MustWrite(newbatch)
		}

		prevStateHash = block.StateHash
		//
		err = chain.upgradeMeta(i)
		if err != nil {
			panic(err)
		}
	}
}

// NeedReExec
func (chain *BlockChain) NeedReExec(meta *types.UpgradeMeta) bool {
	if meta.Starting { //
		return true
	}
	v1 := meta.Version
	v2 := version.GetStoreDBVersion()
	v1arr := strings.Split(v1, ".") //
	v2arr := strings.Split(v2, ".") //
	if len(v1arr) != 3 || len(v2arr) != 3 {
		panic("upgrade store meta version error")
	}
	if v2arr[0] > "2" {
		chainlog.Info("NeedReExec", "version program", v1, "version DB", v2)
		panic("not support upgrade store to greater than 2.0.0")
	}
	if v1arr[0] > v2arr[0] { //
		chainlog.Info("NeedReExec", "version program", v1, "version DB", v2)
		panic("not support degrade the program")
	}
	return v1arr[0] != v2arr[0]
}

func (chain *BlockChain) upgradeMeta(height int64) error {
	meta := &types.UpgradeMeta{
		Starting: true,
		Version:  version.GetStoreDBVersion(),
		Height:   height + 1,
	}
	return chain.blockStore.SetStoreUpgradeMeta(meta)
}
