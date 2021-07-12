// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/golang/protobuf/proto"
)

//var
var (
	tempBlockKey     = []byte("TB:")
	lastTempBlockKey = []byte("LTB:")
)

//const
const (
	// After the node starts, it will wait for seconds for the fast download to start.
	// After the timeout, it will switch to the normal synchronization mode
	//waitTimeDownLoad
	waitTimeDownLoad = 120

	//          peer
	bestPeerCount = 2

	normalDownLoadMode = 0
	fastDownLoadMode   = 1
	chunkDownLoadMode  = 2
)

//DownLoadInfo blockchain    block
type DownLoadInfo struct {
	StartHeight int64
	EndHeight   int64
	Pids        []string
}

//ErrCountInfo  Failed to read a block when starting download. The maximum waiting time is 2 minutes and 120 seconds
type ErrCountInfo struct {
	Height int64
	Count  int64
}

//  temp block height    block
func calcHeightToTempBlockKey(height int64) []byte {
	return append(tempBlockKey, []byte(fmt.Sprintf("%012d", height))...)
}

//  last temp block height
func calcLastTempBlockHeightKey() []byte {
	return lastTempBlockKey
}

//GetDownloadSyncStatus
func (chain *BlockChain) GetDownloadSyncStatus() int {
	chain.downLoadModeLock.Lock()
	defer chain.downLoadModeLock.Unlock()
	return chain.downloadMode
}

//UpdateDownloadSyncStatus Update the synchronization mode of download block
func (chain *BlockChain) UpdateDownloadSyncStatus(mode int) {
	chain.downLoadModeLock.Lock()
	defer chain.downLoadModeLock.Unlock()
	chain.downloadMode = mode
}

//FastDownLoadBlocks Open the mode of fast download block
func (chain *BlockChain) FastDownLoadBlocks() {
	curHeight := chain.GetBlockHeight()
	lastTempHight := chain.GetLastTempBlockHeight()

	synlog.Info("FastDownLoadBlocks", "curHeight", curHeight, "lastTempHight", lastTempHight)

	//To start the fast download block mode, you need to execute the blocks that have been downloaded last time and temporarily stored in dB
	if lastTempHight != -1 && lastTempHight > curHeight {
		chain.ReadBlockToExec(lastTempHight, false)
	}
	//1: The number of best peers is satisfied, and the number of backward blocks is more than 5000. Fast synchronization is started
	//2: If the number of backward blocks is less than 5000, the normal synchronization mode will be started without fast synchronization
	//3: Start for two minutes, if you don't meet the conditions of fast download, you can exit directly and start the normal synchronization mode
	startTime := types.Now()

	for {
		curheight := chain.GetBlockHeight()
		peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()
		pids := chain.GetBestChainPids()
		//            batchsyncblocknum
		if pids != nil && peerMaxBlkHeight != -1 && curheight+batchsyncblocknum >= peerMaxBlkHeight {
			chain.UpdateDownloadSyncStatus(normalDownLoadMode)
			synlog.Info("FastDownLoadBlocks:quit!", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight)
			break
		} else if curheight+batchsyncblocknum < peerMaxBlkHeight && len(pids) >= bestPeerCount {
			synlog.Info("start download blocks!FastDownLoadBlocks", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight)
			go chain.ProcDownLoadBlocks(curheight, peerMaxBlkHeight, false, pids)
			go chain.ReadBlockToExec(peerMaxBlkHeight, true)
			break
		} else if types.Since(startTime) > waitTimeDownLoad*time.Second || chain.cfg.SingleMode {
			chain.UpdateDownloadSyncStatus(normalDownLoadMode)
			synlog.Info("FastDownLoadBlocks:waitTimeDownLoad:quit!", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight, "pids", pids)
			break
		} else {
			synlog.Info("FastDownLoadBlocks task sleep 1 second !")
			time.Sleep(time.Second)
		}
	}
}

//ReadBlockToExec            db  block
func (chain *BlockChain) ReadBlockToExec(height int64, isNewStart bool) {
	synlog.Info("ReadBlockToExec starting!!!", "height", height, "isNewStart", isNewStart)
	var waitCount ErrCountInfo
	waitCount.Height = 0
	waitCount.Count = 0
	cfg := chain.client.GetConfig()
	for {
		select {
		case <-chain.quit:
			return
		default:
		}
		curheight := chain.GetBlockHeight()
		peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()

		//                 batchsyncblocknum   block db
		if peerMaxBlkHeight > curheight+batchsyncblocknum && !chain.cfgBatchSync {
			atomic.CompareAndSwapInt32(&chain.isbatchsync, 1, 0)
		} else {
			atomic.CompareAndSwapInt32(&chain.isbatchsync, 0, 1)
		}
		if (curheight >= peerMaxBlkHeight && peerMaxBlkHeight != -1) || curheight >= height {
			chain.cancelDownLoadFlag(isNewStart)
			synlog.Info("ReadBlockToExec complete!", "curheight", curheight, "height", height, "peerMaxBlkHeight", peerMaxBlkHeight)
			break
		}
		block, err := chain.ReadBlockByHeight(curheight + 1)
		if err != nil {
			// downLoadTask     ，    block2  ，          download
			if isNewStart {
				if !chain.downLoadTask.InProgress() {
					if waitCount.Height == curheight+1 {
						waitCount.Count++
					} else {
						waitCount.Height = curheight + 1
						waitCount.Count = 1
					}
					if waitCount.Count >= 120 {
						chain.cancelDownLoadFlag(isNewStart)
						synlog.Error("ReadBlockToExec:ReadBlockByHeight:timeout", "height", curheight+1, "peerMaxBlkHeight", peerMaxBlkHeight, "err", err)
						break
					}
					time.Sleep(time.Second)
					continue
				} else {
					synlog.Info("ReadBlockToExec:ReadBlockByHeight", "height", curheight+1, "peerMaxBlkHeight", peerMaxBlkHeight, "err", err)
					time.Sleep(time.Second)
					continue
				}
			} else {
				chain.cancelDownLoadFlag(isNewStart)
				synlog.Error("ReadBlockToExec:ReadBlockByHeight", "height", curheight+1, "peerMaxBlkHeight", peerMaxBlkHeight, "err", err)
				break
			}
		}
		_, ismain, isorphan, err := chain.ProcessBlock(false, &types.BlockDetail{Block: block}, "download", true, -1)
		if err != nil {
			//
			if isNewStart && chain.downLoadTask.InProgress() {
				Err := chain.downLoadTask.Cancel()
				if Err != nil {
					synlog.Error("ReadBlockToExec:downLoadTask.Cancel!", "height", block.Height, "hash", common.ToHex(block.Hash(cfg)), "isNewStart", isNewStart, "err", Err)
				}
				chain.DefaultDownLoadInfo()
			}

			//                        ，
			chain.cancelDownLoadFlag(isNewStart)
			chain.blockStore.db.Delete(calcHeightToTempBlockKey(block.Height))

			synlog.Error("ReadBlockToExec:ProcessBlock:err!", "height", block.Height, "hash", common.ToHex(block.Hash(cfg)), "isNewStart", isNewStart, "err", err)
			break
		}
		synlog.Debug("ReadBlockToExec:ProcessBlock:success!", "height", block.Height, "ismain", ismain, "isorphan", isorphan, "hash", common.ToHex(block.Hash(cfg)))
	}
}

//CancelDownLoadFlag
func (chain *BlockChain) cancelDownLoadFlag(isNewStart bool) {
	if isNewStart {
		chain.UpdateDownloadSyncStatus(normalDownLoadMode)
	}
	chain.DelLastTempBlockHeight()
	synlog.Info("cancelFastDownLoadFlag", "isNewStart", isNewStart)
}

//ReadBlockByHeight                 block
func (chain *BlockChain) ReadBlockByHeight(height int64) (*types.Block, error) {
	blockByte, err := chain.blockStore.db.Get(calcHeightToTempBlockKey(height))
	if blockByte == nil || err != nil {
		return nil, types.ErrHeightNotExist
	}
	var block types.Block
	err = proto.Unmarshal(blockByte, &block)
	if err != nil {
		storeLog.Error("ReadBlockByHeight", "err", err)
		return nil, err
	}
	//
	err = chain.blockStore.db.Delete(calcHeightToTempBlockKey(height - 1))
	if err != nil {
		storeLog.Error("ReadBlockByHeight:Delete", "height", height, "err", err)
	}
	return &block, err
}

//WriteBlockToDbTemp      block
func (chain *BlockChain) WriteBlockToDbTemp(block *types.Block, lastHeightSave bool) error {
	if block == nil {
		panic("WriteBlockToDbTemp block is nil")
	}
	sync := true
	if atomic.LoadInt32(&chain.isbatchsync) == 0 {
		sync = false
	}
	beg := types.Now()
	defer func() {
		chainlog.Debug("WriteBlockToDbTemp", "height", block.Height, "sync", sync, "cost", types.Since(beg))
	}()
	newbatch := chain.blockStore.NewBatch(sync)

	blockByte, err := proto.Marshal(block)
	if err != nil {
		chainlog.Error("WriteBlockToDbTemp:Marshal", "height", block.Height)
	}
	newbatch.Set(calcHeightToTempBlockKey(block.Height), blockByte)
	if lastHeightSave {
		heightbytes := types.Encode(&types.Int64{Data: block.Height})
		newbatch.Set(calcLastTempBlockHeightKey(), heightbytes)
	}
	err = newbatch.Write()
	if err != nil {
		panic(err)
	}
	return nil
}

//GetLastTempBlockHeight                block
func (chain *BlockChain) GetLastTempBlockHeight() int64 {
	heightbytes, err := chain.blockStore.db.Get(calcLastTempBlockHeightKey())
	if heightbytes == nil || err != nil {
		chainlog.Error("GetLastTempBlockHeight", "err", err)
		return -1
	}

	var height types.Int64
	err = types.Decode(heightbytes, &height)
	if err != nil {
		chainlog.Error("GetLastTempBlockHeight:Decode", "err", err)
		return -1
	}
	return height.Data
}

//DelLastTempBlockHeight
func (chain *BlockChain) DelLastTempBlockHeight() {
	err := chain.blockStore.db.Delete(calcLastTempBlockHeightKey())
	if err != nil {
		synlog.Error("DelLastTempBlockHeight", "err", err)
	}
}

//ProcDownLoadBlocks     blocks
func (chain *BlockChain) ProcDownLoadBlocks(StartHeight int64, EndHeight int64, chunkDown bool, pids []string) {
	info := chain.GetDownLoadInfo()

	//      DownLoad           ，DownLoad    ， DownLoad
	if info.StartHeight != -1 || info.EndHeight != -1 {
		synlog.Info("ProcDownLoadBlocks", "pids", info.Pids, "StartHeight", info.StartHeight, "EndHeight", info.EndHeight)
	}

	chain.DefaultDownLoadInfo()
	chain.InitDownLoadInfo(StartHeight, EndHeight, pids)
	if chunkDown {
		chain.ReqDownLoadChunkBlocks()
	} else {
		chain.ReqDownLoadBlocks()
	}
}

//InitDownLoadInfo     DownLoad
func (chain *BlockChain) InitDownLoadInfo(StartHeight int64, EndHeight int64, pids []string) {
	chain.downLoadlock.Lock()
	defer chain.downLoadlock.Unlock()

	chain.downLoadInfo.StartHeight = StartHeight
	chain.downLoadInfo.EndHeight = EndHeight
	chain.downLoadInfo.Pids = pids
	synlog.Debug("InitDownLoadInfo begin", "StartHeight", StartHeight, "EndHeight", EndHeight, "pids", pids)

}

//DefaultDownLoadInfo  DownLoadInfo
func (chain *BlockChain) DefaultDownLoadInfo() {
	chain.downLoadlock.Lock()
	defer chain.downLoadlock.Unlock()

	chain.downLoadInfo.StartHeight = -1
	chain.downLoadInfo.EndHeight = -1
	chain.downLoadInfo.Pids = nil
	synlog.Debug("DefaultDownLoadInfo")
}

//GetDownLoadInfo   DownLoadInfo
func (chain *BlockChain) GetDownLoadInfo() *DownLoadInfo {
	chain.downLoadlock.Lock()
	defer chain.downLoadlock.Unlock()
	return chain.downLoadInfo
}

//UpdateDownLoadStartHeight   DownLoad     block
func (chain *BlockChain) UpdateDownLoadStartHeight(StartHeight int64) {
	chain.downLoadlock.Lock()
	defer chain.downLoadlock.Unlock()

	chain.downLoadInfo.StartHeight = StartHeight
	synlog.Debug("UpdateDownLoadStartHeight", "StartHeight", chain.downLoadInfo.StartHeight, "EndHeight", chain.downLoadInfo.EndHeight, "pids", len(chain.downLoadInfo.Pids))
}

//UpdateDownLoadPids   bestpeers
func (chain *BlockChain) UpdateDownLoadPids() {
	pids := chain.GetBestChainPids()

	chain.downLoadlock.Lock()
	defer chain.downLoadlock.Unlock()
	if len(pids) != 0 {
		chain.downLoadInfo.Pids = pids
		synlog.Info("UpdateDownLoadPids", "StartHeight", chain.downLoadInfo.StartHeight, "EndHeight", chain.downLoadInfo.EndHeight, "pids", len(chain.downLoadInfo.Pids))
	}
}

//ReqDownLoadBlocks   DownLoad   blocks
func (chain *BlockChain) ReqDownLoadBlocks() {
	info := chain.GetDownLoadInfo()
	if info.StartHeight != -1 && info.EndHeight != -1 && info.Pids != nil {
		synlog.Info("ReqDownLoadBlocks", "StartHeight", info.StartHeight, "EndHeight", info.EndHeight, "pids", len(info.Pids))
		err := chain.FetchBlock(info.StartHeight, info.EndHeight, info.Pids, true)
		if err != nil {
			synlog.Error("ReqDownLoadBlocks:FetchBlock", "err", err)
		}
	}
}

//DownLoadTimeOutProc
func (chain *BlockChain) DownLoadTimeOutProc(height int64) {
	info := chain.GetDownLoadInfo()
	synlog.Info("DownLoadTimeOutProc", "timeoutheight", height, "StartHeight", info.StartHeight, "EndHeight", info.EndHeight)

	//            pid    ，      peer   ，
	//                   ，         ，
	if info.Pids != nil {
		var exist bool
		for _, pid := range info.Pids {
			peerinfo := chain.GetPeerInfo(pid)
			if peerinfo != nil {
				exist = true
			}
		}
		if !exist {
			synlog.Info("DownLoadTimeOutProc:peer not exist!", "info.Pids", info.Pids, "GetPeers", chain.GetPeers())
			return
		}
	}
	if info.StartHeight != -1 && info.EndHeight != -1 && info.Pids != nil {
		//
		if info.StartHeight > height {
			chain.UpdateDownLoadStartHeight(height)
			info.StartHeight = height
		}
		synlog.Info("DownLoadTimeOutProc:FetchBlock", "StartHeight", info.StartHeight, "EndHeight", info.EndHeight, "pids", len(info.Pids))
		err := chain.FetchBlock(info.StartHeight, info.EndHeight, info.Pids, true)
		if err != nil {
			synlog.Error("DownLoadTimeOutProc:FetchBlock", "err", err)
		}
	}
}

// DownLoadBlocks
func (chain *BlockChain) DownLoadBlocks() {
	if !chain.cfg.DisableShard && chain.cfg.EnableFetchP2pstore {
		// 1.            chunkDownLoad
		chain.UpdateDownloadSyncStatus(chunkDownLoadMode) //      fastDownLoadMode
		if chain.GetDownloadSyncStatus() == chunkDownLoadMode {
			chain.ChunkDownLoadBlocks()
		}
	} else {
		// 2.            ,
		if chain.GetDownloadSyncStatus() == fastDownLoadMode {
			chain.FastDownLoadBlocks()
		}
	}
}

//ChunkDownLoadBlocks
func (chain *BlockChain) ChunkDownLoadBlocks() {
	curHeight := chain.GetBlockHeight()
	lastTempHight := chain.GetLastTempBlockHeight()

	synlog.Info("ChunkDownLoadBlocks", "curHeight", curHeight, "lastTempHight", lastTempHight)

	//                 db  blocks
	if lastTempHight != -1 && lastTempHight > curHeight {
		chain.ReadBlockToExec(lastTempHight, false)
	}
	//1：        MaxRollBlockNum   chunk  ，
	//2：           chunk  ，

	startTime := types.Now()

	for {
		curheight := chain.GetBlockHeight()
		peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()
		targetHeight := chain.calcSafetyChunkHeight(peerMaxBlkHeight) //     chunk
		pids := chain.GetBestChainPids()
		//            batchsyncblocknum
		if pids != nil && peerMaxBlkHeight != -1 && curheight+batchsyncblocknum >= peerMaxBlkHeight {
			synlog.Info("ChunkDownLoadBlocks:quit!", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight)
			chain.UpdateDownloadSyncStatus(normalDownLoadMode)
			break
		} else if curheight < targetHeight && pids != nil {
			synlog.Info("start download blocks!ChunkDownLoadBlocks", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight, "targetHeight", targetHeight)
			go chain.ProcDownLoadBlocks(curheight, targetHeight, true, pids)
			//   chunk
			go chain.ReadBlockToExec(targetHeight, true)
			break
		} else if types.Since(startTime) > waitTimeDownLoad*time.Second*3 || chain.cfg.SingleMode {
			synlog.Info("ChunkDownLoadBlocks:waitTimeDownLoad:quit!", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight, "pids", pids)
			chain.UpdateDownloadSyncStatus(normalDownLoadMode)
			break
		} else {
			synlog.Info("ChunkDownLoadBlocks task sleep 1 second !")
			time.Sleep(time.Second)
		}
	}
}

//ReqDownLoadChunkBlocks   DownLoad   blocks
func (chain *BlockChain) ReqDownLoadChunkBlocks() {
	info := chain.GetDownLoadInfo()
	if info.StartHeight != -1 && info.EndHeight != -1 && info.Pids != nil {
		synlog.Info("ReqDownLoadChunkBlocks", "StartHeight", info.StartHeight, "EndHeight", info.EndHeight, "pids", len(info.Pids))
		err := chain.FetchChunkBlock(info.StartHeight, info.EndHeight, info.Pids, true)
		if err != nil {
			synlog.Error("ReqDownLoadChunkBlocks:FetchBlock", "err", err)
		}
	}
}

//DownLoadChunkTimeOutProc
func (chain *BlockChain) DownLoadChunkTimeOutProc(height int64) {
	info := chain.GetDownLoadInfo()
	synlog.Info("DownLoadChunkTimeOutProc", "real chunkNum", height, "info.StartHeight", info.StartHeight, "info.EndHeight", info.EndHeight)
	//  TODO              ,
	if len(chain.GetPeers()) == 0 {
		synlog.Info("DownLoadChunkTimeOutProc:peers not exist!")
		return
	}
	if info.StartHeight != -1 && info.EndHeight != -1 && info.Pids != nil {
		//
		if info.StartHeight > height {
			chain.UpdateDownLoadStartHeight(height)
			info.StartHeight = height
		}
		synlog.Info("DownLoadChunkTimeOutProc:FetchChunkBlock", "StartHeight", info.StartHeight, "EndHeight", info.EndHeight, "pids", len(info.Pids))
		err := chain.FetchChunkBlock(info.StartHeight, info.EndHeight, info.Pids, true)
		if err != nil {
			synlog.Error("DownLoadChunkTimeOutProc:FetchChunkBlock", "err", err)
		}
	}
}
