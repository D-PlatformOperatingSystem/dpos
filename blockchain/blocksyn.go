// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//var
var (
	BackBlockNum            int64 = 128                  //           blocks
	BackwardBlockNum        int64 = 16                   //             peer
	checkHeightNoIncSeconds int64 = 5 * 60               //               5
	checkBlockHashSeconds   int64 = 1 * 60               //1      tip hash peer      hash
	fetchPeerListSeconds    int64 = 5                    //5      peerlist
	MaxRollBlockNum         int64 = 10000                //    block
	ReduceHeight                  = MaxRollBlockNum      //
	SafetyReduceHeight            = ReduceHeight * 3 / 2 //
	//TODO
	batchsyncblocknum int64 = 5000 //    ，            5000  ，saveblock db

	synlog = chainlog.New("submodule", "syn")
)

//PeerInfo blockchain       peerinfo
type PeerInfo struct {
	Name       string
	ParentHash []byte
	Height     int64
	Hash       []byte
}

//PeerInfoList
type PeerInfoList []*PeerInfo

//Len
func (list PeerInfoList) Len() int {
	return len(list)
}

//Less
func (list PeerInfoList) Less(i, j int) bool {
	if list[i].Height < list[j].Height {
		return true
	} else if list[i].Height > list[j].Height {
		return false
	} else {
		return list[i].Name < list[j].Name
	}
}

//Swap
func (list PeerInfoList) Swap(i, j int) {
	temp := list[i]
	list[i] = list[j]
	list[j] = temp
}

//FaultPeerInfo
type FaultPeerInfo struct {
	Peer        *PeerInfo
	FaultHeight int64
	FaultHash   []byte
	ErrInfo     error
	ReqFlag     bool
}

//BestPeerInfo
type BestPeerInfo struct {
	Peer        *PeerInfo
	Height      int64
	Hash        []byte
	Td          *big.Int
	ReqFlag     bool
	IsBestChain bool
}

//BlockOnChain ...
//           ，
//
//
//BlockOnChain struct
type BlockOnChain struct {
	sync.RWMutex
	Height      int64
	OnChainTime int64
}

//initOnChainTimeout
func (chain *BlockChain) initOnChainTimeout() {
	chain.blockOnChain.Lock()
	defer chain.blockOnChain.Unlock()

	chain.blockOnChain.Height = -1
	chain.blockOnChain.OnChainTime = types.Now().Unix()
}

//OnChainTimeout
func (chain *BlockChain) OnChainTimeout(height int64) bool {
	chain.blockOnChain.Lock()
	defer chain.blockOnChain.Unlock()

	if chain.onChainTimeout == 0 {
		return false
	}

	curTime := types.Now().Unix()
	if chain.blockOnChain.Height != height {
		chain.blockOnChain.Height = height
		chain.blockOnChain.OnChainTime = curTime
		return false
	}
	if curTime-chain.blockOnChain.OnChainTime > chain.onChainTimeout {
		synlog.Debug("OnChainTimeout", "curTime", curTime, "blockOnChain", chain.blockOnChain)
		return true
	}
	return false
}

//SynRoutine
func (chain *BlockChain) SynRoutine() {
	//  peerlist    ，  1
	fetchPeerListTicker := time.NewTicker(time.Duration(fetchPeerListSeconds) * time.Second)

	// peer    block    ，  2s
	blockSynTicker := time.NewTicker(chain.blockSynInterVal * time.Second)

	//5      bestchain         ，                 ，
	//     peer       headers       ，     peer        blocks
	checkHeightNoIncreaseTicker := time.NewTicker(time.Duration(checkHeightNoIncSeconds) * time.Second)

	//    1       bestchain tiphash   peer      blockshash    。
	//                   ，   peer              headers
	//        block        ，          blocks      ，
	checkBlockHashTicker := time.NewTicker(time.Duration(checkBlockHashSeconds) * time.Second)

	//5          ，
	checkClockDriftTicker := time.NewTicker(300 * time.Second)

	//3          peer
	recoveryFaultPeerTicker := time.NewTicker(180 * time.Second)

	//2           ，
	checkBestChainTicker := time.NewTicker(120 * time.Second)

	//30s   peer    ChunkRecord
	chunkRecordSynTicker := time.NewTicker(30 * time.Second)

	//
	go chain.DownLoadBlocks()

	for {
		select {
		case <-chain.quit:
			//synlog.Info("quit SynRoutine!")
			return
		case <-blockSynTicker.C:
			//synlog.Info("blockSynTicker")
			if chain.GetDownloadSyncStatus() == normalDownLoadMode {
				go chain.SynBlocksFromPeers()
			}

		case <-fetchPeerListTicker.C:
			//synlog.Info("blockUpdateTicker")
			chain.tickerwg.Add(1)
			go chain.FetchPeerList()

		case <-checkHeightNoIncreaseTicker.C:
			//synlog.Info("CheckHeightNoIncrease")
			chain.tickerwg.Add(1)
			go chain.CheckHeightNoIncrease()

		case <-checkBlockHashTicker.C:
			//synlog.Info("checkBlockHashTicker")
			chain.tickerwg.Add(1)
			go chain.CheckTipBlockHash()

			//        ，         ，
		case <-checkClockDriftTicker.C:
			// ntp               go     ，    WaitGroup
			go chain.checkClockDrift()

			//      peer，         blockhash    ，    peer
		case <-recoveryFaultPeerTicker.C:
			chain.tickerwg.Add(1)
			go chain.RecoveryFaultPeer()

			//    peerlist            ，       blockhash
		case <-checkBestChainTicker.C:
			chain.tickerwg.Add(1)
			go chain.CheckBestChain(false)

			//    ChunkRecord
		case <-chunkRecordSynTicker.C:
			if !chain.cfg.DisableShard {
				go chain.ChunkRecordSync()
			}
		}
	}
}

/*
FetchBlock     ：
   P2P    EventFetchBlock(types.RequestGetBlock)，           ，
P2P         ，  blockchain     ， EventReply。
              ，P2P             ，
    EventAddBlocks(types.Blocks)   blockchain   ，
blockchain      EventReply
syncOrfork:true fork    ，       block
          :fasle       ，    128 block
*/
func (chain *BlockChain) FetchBlock(start int64, end int64, pid []string, syncOrfork bool) (err error) {
	if chain.client == nil {
		synlog.Error("FetchBlock chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}

	synlog.Debug("FetchBlock input", "StartHeight", start, "EndHeight", end, "pid", pid)
	blockcount := end - start
	if blockcount < 0 {
		return types.ErrStartBigThanEnd
	}
	var requestblock types.ReqBlocks
	requestblock.Start = start
	requestblock.IsDetail = false
	requestblock.Pid = pid

	//  block    128
	if blockcount >= chain.MaxFetchBlockNum {
		requestblock.End = start + chain.MaxFetchBlockNum - 1
	} else {
		requestblock.End = end
	}
	var cb func()
	var timeoutcb func(height int64)
	if syncOrfork {
		//        ，
		if requestblock.End < chain.downLoadInfo.EndHeight {
			cb = func() {
				chain.ReqDownLoadBlocks()
			}
			timeoutcb = func(height int64) {
				chain.DownLoadTimeOutProc(height)
			}
			chain.UpdateDownLoadStartHeight(requestblock.End + 1)
			//           bestpeer，       block
			if chain.GetDownloadSyncStatus() == fastDownLoadMode {
				chain.UpdateDownLoadPids()
			}
		} else { //   DownLoad block     ，  DownLoadInfo
			chain.DefaultDownLoadInfo()
		}
		err = chain.downLoadTask.Start(requestblock.Start, requestblock.End, cb, timeoutcb)
		if err != nil {
			return err
		}
	} else {
		if chain.GetPeerMaxBlkHeight()-requestblock.End > BackBlockNum {
			cb = func() {
				chain.SynBlocksFromPeers()
			}
		}
		err = chain.syncTask.Start(requestblock.Start, requestblock.End, cb, timeoutcb)
		if err != nil {
			return err
		}
	}

	synlog.Info("FetchBlock", "Start", requestblock.Start, "End", requestblock.End)
	msg := chain.client.NewMessage("p2p", types.EventFetchBlocks, &requestblock)
	Err := chain.client.Send(msg, true)
	if Err != nil {
		synlog.Error("FetchBlock", "client.Send err:", Err)
		return err
	}
	resp, err := chain.client.Wait(msg)
	if err != nil {
		synlog.Error("FetchBlock", "client.Wait err:", err)
		return err
	}
	return resp.Err()
}

//FetchPeerList  p2p    peerlist，    active       。
//        block    p2p
func (chain *BlockChain) FetchPeerList() {
	defer chain.tickerwg.Done()
	err := chain.fetchPeerList()
	if err != nil {
		synlog.Error("FetchPeerList.", "err", err)
	}
}

func (chain *BlockChain) fetchPeerList() error {
	if chain.client == nil {
		synlog.Error("fetchPeerList chain client not bind message queue.")
		return nil
	}
	msg := chain.client.NewMessage("p2p", types.EventPeerInfo, nil)
	Err := chain.client.SendTimeout(msg, true, 30*time.Second)
	if Err != nil {
		synlog.Error("fetchPeerList", "client.Send err:", Err)
		return Err
	}
	resp, err := chain.client.WaitTimeout(msg, 60*time.Second)
	if err != nil {
		synlog.Error("fetchPeerList", "client.Wait err:", err)
		return err
	}

	peerlist := resp.GetData().(*types.PeerList)
	if peerlist == nil {
		synlog.Error("fetchPeerList", "peerlist", "is nil")
		return types.ErrNoPeer
	}
	curheigt := chain.GetBlockHeight()

	var peerInfoList PeerInfoList
	for _, peer := range peerlist.Peers {
		//chainlog.Info("fetchPeerList", "peername:", peer.Name, "peerHeight:", peer.Header.Height)
		//          5

		if peer == nil || peer.Self || curheigt > peer.Header.Height+5 {
			continue
		}
		var peerInfo PeerInfo
		peerInfo.Name = peer.Name
		peerInfo.ParentHash = peer.Header.ParentHash
		peerInfo.Height = peer.Header.Height
		peerInfo.Hash = peer.Header.Hash
		peerInfoList = append(peerInfoList, &peerInfo)
	}
	//peerlist
	if len(peerInfoList) == 0 {
		return nil
	}
	//  height peer
	sort.Sort(peerInfoList)

	subInfoList := peerInfoList

	chain.peerMaxBlklock.Lock()
	chain.peerList = subInfoList
	chain.peerMaxBlklock.Unlock()

	//   peerlist  ，                 。
	if atomic.LoadInt32(&chain.firstcheckbestchain) == 0 {
		synlog.Info("fetchPeerList trigger first CheckBestChain")
		chain.CheckBestChain(true)
	}
	return nil
}

//GetRcvLastCastBlkHeight      block
func (chain *BlockChain) GetRcvLastCastBlkHeight() int64 {
	chain.castlock.Lock()
	defer chain.castlock.Unlock()
	return chain.rcvLastBlockHeight
}

//UpdateRcvCastBlkHeight      block
func (chain *BlockChain) UpdateRcvCastBlkHeight(height int64) {
	chain.castlock.Lock()
	defer chain.castlock.Unlock()
	chain.rcvLastBlockHeight = height
}

//GetsynBlkHeight        db block
func (chain *BlockChain) GetsynBlkHeight() int64 {
	chain.synBlocklock.Lock()
	defer chain.synBlocklock.Unlock()
	return chain.synBlockHeight
}

//UpdatesynBlkHeight        db block
func (chain *BlockChain) UpdatesynBlkHeight(height int64) {
	chain.synBlocklock.Lock()
	defer chain.synBlocklock.Unlock()
	chain.synBlockHeight = height
}

//GetPeerMaxBlkHeight   peerlist      block
func (chain *BlockChain) GetPeerMaxBlkHeight() int64 {
	chain.peerMaxBlklock.Lock()
	defer chain.peerMaxBlklock.Unlock()

	//  peerlist      ，peerlist           。
	if chain.peerList != nil {
		peerlen := len(chain.peerList)
		for i := peerlen - 1; i >= 0; i-- {
			if chain.peerList[i] != nil {
				ok := chain.IsFaultPeer(chain.peerList[i].Name)
				if !ok {
					return chain.peerList[i].Height
				}
			}
		}
		//     peer，           ，  peerlist    peer
		maxpeer := chain.peerList[peerlen-1]
		if maxpeer != nil {
			synlog.Debug("GetPeerMaxBlkHeight all peers are faultpeer maybe self on Side chain", "pid", maxpeer.Name, "Height", maxpeer.Height, "Hash", common.ToHex(maxpeer.Hash))
			return maxpeer.Height
		}
	}
	return -1
}

//GetPeerInfo   peerid  peerinfo
func (chain *BlockChain) GetPeerInfo(pid string) *PeerInfo {
	chain.peerMaxBlklock.Lock()
	defer chain.peerMaxBlklock.Unlock()

	//  peerinfo
	if chain.peerList != nil {
		for _, peer := range chain.peerList {
			if pid == peer.Name {
				return peer
			}
		}
	}
	return nil
}

//GetMaxPeerInfo   peerlist      peerinfo
func (chain *BlockChain) GetMaxPeerInfo() *PeerInfo {
	chain.peerMaxBlklock.Lock()
	defer chain.peerMaxBlklock.Unlock()

	//  peerlist      peer，peerlist           。
	if chain.peerList != nil {
		peerlen := len(chain.peerList)
		for i := peerlen - 1; i >= 0; i-- {
			if chain.peerList[i] != nil {
				ok := chain.IsFaultPeer(chain.peerList[i].Name)
				if !ok {
					return chain.peerList[i]
				}
			}
		}
		//     peer，           ，  peerlist    peer
		maxpeer := chain.peerList[peerlen-1]
		if maxpeer != nil {
			synlog.Debug("GetMaxPeerInfo all peers are faultpeer maybe self on Side chain", "pid", maxpeer.Name, "Height", maxpeer.Height, "Hash", common.ToHex(maxpeer.Hash))
			return maxpeer
		}
	}
	return nil
}

//GetPeers     peers
func (chain *BlockChain) GetPeers() PeerInfoList {
	chain.peerMaxBlklock.Lock()
	defer chain.peerMaxBlklock.Unlock()

	//  peerinfo
	var peers PeerInfoList

	if chain.peerList != nil {
		peers = append(peers, chain.peerList...)
	}
	return peers
}

//GetPeersMap   peers map
func (chain *BlockChain) GetPeersMap() map[string]bool {
	chain.peerMaxBlklock.Lock()
	defer chain.peerMaxBlklock.Unlock()
	peersmap := make(map[string]bool)

	if chain.peerList != nil {
		for _, peer := range chain.peerList {
			peersmap[peer.Name] = true
		}
	}
	return peersmap
}

//IsFaultPeer     pid     faultPeerList
func (chain *BlockChain) IsFaultPeer(pid string) bool {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	return chain.faultPeerList[pid] != nil
}

//IsErrExecBlock    block             。
func (chain *BlockChain) IsErrExecBlock(height int64, hash []byte) (bool, error) {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	//      peerlist，      peer
	for _, faultpeer := range chain.faultPeerList {
		if faultpeer.FaultHeight == height && bytes.Equal(hash, faultpeer.FaultHash) {
			return true, faultpeer.ErrInfo
		}
	}
	return false, nil
}

//GetFaultPeer     pid     faultPeerList
func (chain *BlockChain) GetFaultPeer(pid string) *FaultPeerInfo {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	return chain.faultPeerList[pid]
}

//RecoveryFaultPeer       peer  ，      peer    block    。
//    block     。        peer      ok
func (chain *BlockChain) RecoveryFaultPeer() {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	defer chain.tickerwg.Done()

	//      peerlist，      peer
	for pid, faultpeer := range chain.faultPeerList {
		//         ，          ，         ，          hash           。
		//          blockhash   ，hash
		blockhash, err := chain.blockStore.GetBlockHashByHeight(faultpeer.FaultHeight)
		if err == nil {
			if bytes.Equal(faultpeer.FaultHash, blockhash) {
				synlog.Debug("RecoveryFaultPeer ", "Height", faultpeer.FaultHeight, "FaultHash", common.ToHex(faultpeer.FaultHash), "pid", pid)
				delete(chain.faultPeerList, pid)
				continue
			}
		}

		err = chain.FetchBlockHeaders(faultpeer.FaultHeight, faultpeer.FaultHeight, pid)
		if err == nil {
			chain.faultPeerList[pid].ReqFlag = true
		}
		synlog.Debug("RecoveryFaultPeer", "pid", faultpeer.Peer.Name, "FaultHeight", faultpeer.FaultHeight, "FaultHash", common.ToHex(faultpeer.FaultHash), "Err", faultpeer.ErrInfo)
	}
}

//AddFaultPeer          FaultPeerList
func (chain *BlockChain) AddFaultPeer(faultpeer *FaultPeerInfo) {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	//         peerlist
	faultnode := chain.faultPeerList[faultpeer.Peer.Name]
	if faultnode != nil {
		synlog.Debug("AddFaultPeer old", "pid", faultnode.Peer.Name, "FaultHeight", faultnode.FaultHeight, "FaultHash", common.ToHex(faultnode.FaultHash), "Err", faultnode.ErrInfo)
	}
	chain.faultPeerList[faultpeer.Peer.Name] = faultpeer
	synlog.Debug("AddFaultPeer new", "pid", faultpeer.Peer.Name, "FaultHeight", faultpeer.FaultHeight, "FaultHash", common.ToHex(faultpeer.FaultHash), "Err", faultpeer.ErrInfo)
}

//RemoveFaultPeer  pid         ，  pid
func (chain *BlockChain) RemoveFaultPeer(pid string) {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()
	synlog.Debug("RemoveFaultPeer", "pid", pid)

	delete(chain.faultPeerList, pid)
}

//UpdateFaultPeer      peer
func (chain *BlockChain) UpdateFaultPeer(pid string, reqFlag bool) {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	faultpeer := chain.faultPeerList[pid]
	if faultpeer != nil {
		faultpeer.ReqFlag = reqFlag
	}
}

//RecordFaultPeer  blcok     ，    block  ，hash ，          peerid
func (chain *BlockChain) RecordFaultPeer(pid string, height int64, hash []byte, err error) {

	var faultnode FaultPeerInfo

	//  pid  peerinfo
	peerinfo := chain.GetPeerInfo(pid)
	if peerinfo == nil {
		synlog.Error("RecordFaultPeerNode GetPeerInfo is nil", "pid", pid)
		return
	}
	faultnode.Peer = peerinfo
	faultnode.FaultHeight = height
	faultnode.FaultHash = hash
	faultnode.ErrInfo = err
	faultnode.ReqFlag = false
	chain.AddFaultPeer(&faultnode)
}

//SynBlocksFromPeers blockSynSeconds          height     ，           peerlist      ，
func (chain *BlockChain) SynBlocksFromPeers() {

	curheight := chain.GetBlockHeight()
	RcvLastCastBlkHeight := chain.GetRcvLastCastBlkHeight()
	peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()

	//                 batchsyncblocknum   block db
	if peerMaxBlkHeight > curheight+batchsyncblocknum && !chain.cfgBatchSync {
		atomic.CompareAndSwapInt32(&chain.isbatchsync, 1, 0)
	} else if peerMaxBlkHeight >= 0 {
		atomic.CompareAndSwapInt32(&chain.isbatchsync, 0, 1)
	}

	//      ，
	if chain.syncTask.InProgress() {
		synlog.Info("chain syncTask InProgress")
		return
	}
	//            ，        。
	//
	if chain.downLoadTask.InProgress() {
		synlog.Info("chain downLoadTask InProgress")
		return
	}
	//  peers     .        block
	//    2          ，
	backWardThanTwo := curheight+1 < peerMaxBlkHeight
	backWardOne := curheight+1 == peerMaxBlkHeight && chain.OnChainTimeout(curheight)

	if backWardThanTwo || backWardOne {
		synlog.Info("SynBlocksFromPeers", "curheight", curheight, "LastCastBlkHeight", RcvLastCastBlkHeight, "peerMaxBlkHeight", peerMaxBlkHeight)
		pids := chain.GetBestChainPids()
		if pids != nil {
			recvChunk := chain.GetCurRecvChunkNum()
			curShouldChunk, _, _ := chain.CalcChunkInfo(curheight + 1)
			// TODO              FetchChunkBlock，            chunk，
			if !chain.cfg.DisableShard && chain.cfg.EnableFetchP2pstore &&
				curheight+MaxRollBlockNum < peerMaxBlkHeight && recvChunk >= curShouldChunk {
				err := chain.FetchChunkBlock(curheight+1, peerMaxBlkHeight, pids, false)
				if err != nil {
					synlog.Error("SynBlocksFromPeers FetchChunkBlock", "err", err)
				}
			} else {
				err := chain.FetchBlock(curheight+1, peerMaxBlkHeight, pids, false)
				if err != nil {
					synlog.Error("SynBlocksFromPeers FetchBlock", "err", err)
				}
			}
		} else {
			synlog.Info("SynBlocksFromPeers GetBestChainPids is nil")
		}
	}
}

//CheckHeightNoIncrease               ， peerlist              ，
//           ,        peer         blockheader
//  bestchain.Height -BackBlockNum -- bestchain.Height header
//                 block，           block        。
func (chain *BlockChain) CheckHeightNoIncrease() {
	defer chain.tickerwg.Done()

	//
	tipheight := chain.bestChain.Height()
	laststorheight := chain.blockStore.Height()

	if tipheight != laststorheight {
		synlog.Error("CheckHeightNoIncrease", "tipheight", tipheight, "laststorheight", laststorheight)
		return
	}
	//
	checkheight := chain.GetsynBlkHeight()

	//bestchain tip     ，           ，
	if tipheight != checkheight {
		chain.UpdatesynBlkHeight(tipheight)
		return
	}
	//           bestchain tip      。
	//        peer      peer       ，         ，
	//      peer    BackBlockNum headers
	maxpeer := chain.GetMaxPeerInfo()
	if maxpeer == nil {
		synlog.Error("CheckHeightNoIncrease GetMaxPeerInfo is nil")
		return
	}
	peermaxheight := maxpeer.Height
	pid := maxpeer.Name
	var err error
	if peermaxheight > tipheight && (peermaxheight-tipheight) > BackwardBlockNum && !chain.isBestChainPeer(pid) {
		//   peer    BackBlockNum blockheaders
		synlog.Debug("CheckHeightNoIncrease", "tipheight", tipheight, "pid", pid)
		if tipheight > BackBlockNum {
			err = chain.FetchBlockHeaders(tipheight-BackBlockNum, tipheight, pid)
		} else {
			err = chain.FetchBlockHeaders(0, tipheight, pid)
		}
		if err != nil {
			synlog.Error("CheckHeightNoIncrease FetchBlockHeaders", "err", err)
		}
	}
}

//FetchBlockHeaders    pid  start end   headers
func (chain *BlockChain) FetchBlockHeaders(start int64, end int64, pid string) (err error) {
	if chain.client == nil {
		synlog.Error("FetchBlockHeaders chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}

	chainlog.Debug("FetchBlockHeaders", "StartHeight", start, "EndHeight", end, "pid", pid)

	var requestblock types.ReqBlocks
	requestblock.Start = start
	requestblock.End = end
	requestblock.IsDetail = false
	requestblock.Pid = []string{pid}

	msg := chain.client.NewMessage("p2p", types.EventFetchBlockHeaders, &requestblock)
	Err := chain.client.Send(msg, true)
	if Err != nil {
		synlog.Error("FetchBlockHeaders", "client.Send err:", Err)
		return err
	}
	resp, err := chain.client.Wait(msg)
	if err != nil {
		synlog.Error("FetchBlockHeaders", "client.Wait err:", err)
		return err
	}
	return resp.Err()
}

//ProcBlockHeader   block header     ， tiphash   ，  peer   block
func (chain *BlockChain) ProcBlockHeader(headers *types.Headers, peerid string) error {

	//           peer    block header
	faultPeer := chain.GetFaultPeer(peerid)
	if faultPeer != nil && faultPeer.ReqFlag && faultPeer.FaultHeight == headers.Items[0].Height {
		//     block hash   ，    peer       ，  peer   peerlist
		if !bytes.Equal(headers.Items[0].Hash, faultPeer.FaultHash) {
			chain.RemoveFaultPeer(peerid)
		} else {
			chain.UpdateFaultPeer(peerid, false)
		}
		return nil
	}

	//
	bestchainPeer := chain.GetBestChainPeer(peerid)
	if bestchainPeer != nil && bestchainPeer.ReqFlag && headers.Items[0].Height == bestchainPeer.Height {
		chain.CheckBestChainProc(headers, peerid)
		return nil
	}

	//   tiphash      block header
	height := headers.Items[0].Height
	//  height       headers
	header, err := chain.blockStore.GetBlockHeaderByHeight(height)
	if err != nil {
		return err
	}
	//    hash
	if !bytes.Equal(headers.Items[0].Hash, header.Hash) {
		synlog.Info("ProcBlockHeader hash no equal", "height", height, "self hash", common.ToHex(header.Hash), "peer hash", common.ToHex(headers.Items[0].Hash))

		if height > BackBlockNum {
			err = chain.FetchBlockHeaders(height-BackBlockNum, height, peerid)
		} else if height != 0 {
			err = chain.FetchBlockHeaders(0, height, peerid)
		}
		if err != nil {
			synlog.Info("ProcBlockHeader FetchBlockHeaders", "err", err)
		}
	}
	return nil
}

//ProcBlockHeaders   headers     ，
func (chain *BlockChain) ProcBlockHeaders(headers *types.Headers, pid string) error {
	var ForkHeight int64 = -1
	var forkhash []byte
	var err error
	count := len(headers.Items)
	tipheight := chain.bestChain.Height()

	//
	for i := count - 1; i >= 0; i-- {
		exists := chain.bestChain.HaveBlock(headers.Items[i].Hash, headers.Items[i].Height)
		if exists {
			ForkHeight = headers.Items[i].Height
			forkhash = headers.Items[i].Hash
			break
		}
	}
	if ForkHeight == -1 {
		synlog.Error("ProcBlockHeaders do not find fork point ")
		synlog.Error("ProcBlockHeaders start headerinfo", "height", headers.Items[0].Height, "hash", common.ToHex(headers.Items[0].Hash))
		synlog.Error("ProcBlockHeaders end headerinfo", "height", headers.Items[count-1].Height, "hash", common.ToHex(headers.Items[count-1].Hash))

		//  5000 block       ，
		startheight := headers.Items[0].Height
		if tipheight > startheight && (tipheight-startheight) > MaxRollBlockNum {
			synlog.Error("ProcBlockHeaders Not Roll Back!", "selfheight", tipheight, "RollBackedhieght", startheight)
			return types.ErrNotRollBack
		}
		//          headers
		height := headers.Items[0].Height
		if height > BackBlockNum {
			err = chain.FetchBlockHeaders(height-BackBlockNum, height, pid)
		} else {
			err = chain.FetchBlockHeaders(0, height, pid)
		}
		if err != nil {
			synlog.Info("ProcBlockHeaders FetchBlockHeaders", "err", err)
		}
		return types.ErrContinueBack
	}
	synlog.Info("ProcBlockHeaders find fork point", "height", ForkHeight, "hash", common.ToHex(forkhash))

	//   pid   peer  ，
	peerinfo := chain.GetPeerInfo(pid)
	if peerinfo == nil {
		synlog.Error("ProcBlockHeaders GetPeerInfo is nil", "pid", pid)
		return types.ErrPeerInfoIsNil
	}

	//           block， pid
	peermaxheight := peerinfo.Height

	//              blcok
	if chain.downLoadTask.InProgress() {
		synlog.Info("ProcBlockHeaders downLoadTask.InProgress")
		return nil
	}
	//     block     fork
	//
	//
	if chain.GetDownloadSyncStatus() == normalDownLoadMode {
		if chain.syncTask.InProgress() {
			err = chain.syncTask.Cancel()
			synlog.Info("ProcBlockHeaders: cancel syncTask start fork process downLoadTask!", "err", err)
		}
		endHeight := peermaxheight
		if tipheight < peermaxheight {
			endHeight = tipheight + 1
		}
		go chain.ProcDownLoadBlocks(ForkHeight, endHeight, false, []string{pid})
	}
	return nil
}

//ProcAddBlockHeadersMsg    peer   headers
func (chain *BlockChain) ProcAddBlockHeadersMsg(headers *types.Headers, pid string) error {
	if headers == nil {
		return types.ErrInvalidParam
	}
	count := len(headers.Items)
	synlog.Debug("ProcAddBlockHeadersMsg", "count", count, "pid", pid)
	if count == 1 {
		return chain.ProcBlockHeader(headers, pid)
	}
	return chain.ProcBlockHeaders(headers, pid)

}

//CheckTipBlockHash               ， peerlist              ，
//           ,        peer         blockheader
//  bestchain.Height -BackBlockNum -- bestchain.Height header
//                 block，           block        。
func (chain *BlockChain) CheckTipBlockHash() {
	synlog.Debug("CheckTipBlockHash")
	defer chain.tickerwg.Done()

	//
	tipheight := chain.bestChain.Height()
	tiphash := chain.bestChain.Tip().hash
	laststorheight := chain.blockStore.Height()

	if tipheight != laststorheight {
		synlog.Error("CheckTipBlockHash", "tipheight", tipheight, "laststorheight", laststorheight)
		return
	}

	maxpeer := chain.GetMaxPeerInfo()
	if maxpeer == nil {
		synlog.Error("CheckTipBlockHash GetMaxPeerInfo is nil")
		return
	}
	peermaxheight := maxpeer.Height
	pid := maxpeer.Name
	peerhash := maxpeer.Hash
	var Err error
	//    peer tip block hash
	if peermaxheight > tipheight {
		//   peer   BackBlockNum blockheaders
		synlog.Debug("CheckTipBlockHash >", "peermaxheight", peermaxheight, "tipheight", tipheight)
		Err = chain.FetchBlockHeaders(tipheight, tipheight, pid)
	} else if peermaxheight == tipheight {
		//   tip block hash  ,        peer      headers，
		if !bytes.Equal(tiphash, peerhash) {
			if tipheight > BackBlockNum {
				synlog.Debug("CheckTipBlockHash ==", "peermaxheight", peermaxheight, "tipheight", tipheight)
				Err = chain.FetchBlockHeaders(tipheight-BackBlockNum, tipheight, pid)
			} else {
				synlog.Debug("CheckTipBlockHash !=", "peermaxheight", peermaxheight, "tipheight", tipheight)
				Err = chain.FetchBlockHeaders(1, tipheight, pid)
			}
		}
	} else {

		header, err := chain.blockStore.GetBlockHeaderByHeight(peermaxheight)
		if err != nil {
			return
		}
		if !bytes.Equal(header.Hash, peerhash) {
			if peermaxheight > BackBlockNum {
				synlog.Debug("CheckTipBlockHash<!=", "peermaxheight", peermaxheight, "tipheight", tipheight)
				Err = chain.FetchBlockHeaders(peermaxheight-BackBlockNum, peermaxheight, pid)
			} else {
				synlog.Debug("CheckTipBlockHash<!=", "peermaxheight", peermaxheight, "tipheight", tipheight)
				Err = chain.FetchBlockHeaders(1, peermaxheight, pid)
			}
		}
	}
	if Err != nil {
		synlog.Error("CheckTipBlockHash FetchBlockHeaders", "err", Err)
	}
}

//IsCaughtUp               ，
func (chain *BlockChain) IsCaughtUp() bool {

	height := chain.GetBlockHeight()

	//peerMaxBlklock.Lock()
	//defer peerMaxBlklock.Unlock()
	peers := chain.GetPeers()
	// peer       ，
	if peers == nil {
		synlog.Debug("IsCaughtUp has no peers")
		return chain.cfg.SingleMode
	}

	var maxPeerHeight int64 = -1
	peersNo := 0
	for _, peer := range peers {
		if peer != nil && maxPeerHeight < peer.Height {
			ok := chain.IsFaultPeer(peer.Name)
			if !ok {
				maxPeerHeight = peer.Height
			}
		}
		peersNo++
	}

	isCaughtUp := (height > 0 || types.Since(chain.startTime) > 60*time.Second) && (maxPeerHeight == 0 || (height >= maxPeerHeight && maxPeerHeight != -1))

	synlog.Debug("IsCaughtUp", "IsCaughtUp ", isCaughtUp, "height", height, "maxPeerHeight", maxPeerHeight, "peersNo", peersNo)
	return isCaughtUp
}

//GetNtpClockSyncStatus   ntp
func (chain *BlockChain) GetNtpClockSyncStatus() bool {
	chain.ntpClockSynclock.Lock()
	defer chain.ntpClockSynclock.Unlock()
	return chain.isNtpClockSync
}

//UpdateNtpClockSyncStatus     ntp
func (chain *BlockChain) UpdateNtpClockSyncStatus(Sync bool) {
	chain.ntpClockSynclock.Lock()
	defer chain.ntpClockSynclock.Unlock()
	chain.isNtpClockSync = Sync
}

//CheckBestChain             ,   peer       header
func (chain *BlockChain) CheckBestChain(isFirst bool) {
	if !isFirst {
		defer chain.tickerwg.Done()
	}
	peers := chain.GetPeers()
	// peer       ，
	if peers == nil {
		synlog.Debug("CheckBestChain has no peers")
		return
	}

	//
	atomic.CompareAndSwapInt32(&chain.firstcheckbestchain, 0, 1)

	tipheight := chain.bestChain.Height()

	chain.bestpeerlock.Lock()
	defer chain.bestpeerlock.Unlock()

	for _, peer := range peers {
		bestpeer := chain.bestChainPeerList[peer.Name]
		if bestpeer != nil {
			bestpeer.Peer = peer
			bestpeer.Height = tipheight
			bestpeer.Hash = nil
			bestpeer.Td = nil
			bestpeer.ReqFlag = true
		} else {
			if peer.Height < tipheight {
				continue
			}
			var newbestpeer BestPeerInfo
			newbestpeer.Peer = peer
			newbestpeer.Height = tipheight
			newbestpeer.Hash = nil
			newbestpeer.Td = nil
			newbestpeer.ReqFlag = true
			newbestpeer.IsBestChain = false
			chain.bestChainPeerList[peer.Name] = &newbestpeer
		}
		synlog.Debug("CheckBestChain FetchBlockHeaders", "height", tipheight, "pid", peer.Name)
		err := chain.FetchBlockHeaders(tipheight, tipheight, peer.Name)
		if err != nil {
			synlog.Error("CheckBestChain FetchBlockHeaders", "height", tipheight, "pid", peer.Name)
		}
	}
}

//GetBestChainPeer
func (chain *BlockChain) GetBestChainPeer(pid string) *BestPeerInfo {
	chain.bestpeerlock.Lock()
	defer chain.bestpeerlock.Unlock()
	return chain.bestChainPeerList[pid]
}

//isBestChainPeer   peer
func (chain *BlockChain) isBestChainPeer(pid string) bool {
	chain.bestpeerlock.Lock()
	defer chain.bestpeerlock.Unlock()
	peer := chain.bestChainPeerList[pid]
	if peer != nil && peer.IsBestChain {
		return true
	}
	return false
}

//GetBestChainPids             ,   peer       header
func (chain *BlockChain) GetBestChainPids() []string {
	var PeerPids []string
	chain.bestpeerlock.Lock()
	defer chain.bestpeerlock.Unlock()

	peersmap := chain.GetPeersMap()
	for key, value := range chain.bestChainPeerList {
		if !peersmap[value.Peer.Name] {
			delete(chain.bestChainPeerList, value.Peer.Name)
			synlog.Debug("GetBestChainPids:delete", "peer", value.Peer.Name)
			continue
		}
		if value.IsBestChain {
			ok := chain.IsFaultPeer(value.Peer.Name)
			if !ok {
				PeerPids = append(PeerPids, key)
			}
		}
	}
	synlog.Debug("GetBestChainPids", "pids", PeerPids)
	return PeerPids
}

//CheckBestChainProc
func (chain *BlockChain) CheckBestChainProc(headers *types.Headers, pid string) {

	//          blockhash
	blockhash, err := chain.blockStore.GetBlockHashByHeight(headers.Items[0].Height)
	if err != nil {
		synlog.Debug("CheckBestChainProc GetBlockHashByHeight", "Height", headers.Items[0].Height, "err", err)
		return
	}

	chain.bestpeerlock.Lock()
	defer chain.bestpeerlock.Unlock()

	bestchainpeer := chain.bestChainPeerList[pid]
	if bestchainpeer == nil {
		synlog.Debug("CheckBestChainProc bestChainPeerList is nil", "Height", headers.Items[0].Height, "pid", pid)
		return
	}
	//
	if bestchainpeer.Height == headers.Items[0].Height {
		bestchainpeer.Hash = headers.Items[0].Hash
		bestchainpeer.ReqFlag = false
		if bytes.Equal(headers.Items[0].Hash, blockhash) {
			bestchainpeer.IsBestChain = true
			synlog.Debug("CheckBestChainProc IsBestChain ", "Height", headers.Items[0].Height, "pid", pid)
		} else {
			bestchainpeer.IsBestChain = false
			synlog.Debug("CheckBestChainProc NotBestChain", "Height", headers.Items[0].Height, "pid", pid)
		}
	}
}

// ChunkRecordSync   chunkrecord
func (chain *BlockChain) ChunkRecordSync() {
	curheight := chain.GetBlockHeight()
	peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()
	recvChunk := chain.GetCurRecvChunkNum()

	curShouldChunk, _, _ := chain.CalcChunkInfo(curheight)
	targetChunk, _, _ := chain.CalcSafetyChunkInfo(peerMaxBlkHeight)
	if targetChunk < 0 ||
		curShouldChunk >= targetChunk || //              chunk
		recvChunk >= targetChunk {
		return
	}

	//
	if chain.chunkRecordTask.InProgress() {
		synlog.Info("chain chunkRecordTask InProgress")
		return
	}

	synlog.Info("ChunkRecordSync", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight,
		"recvChunk", recvChunk, "curShouldChunk", curShouldChunk)

	pids := chain.GetBestChainPids()
	if pids != nil {
		err := chain.FetchChunkRecords(recvChunk+1, targetChunk, pids)
		if err != nil {
			synlog.Error("ChunkRecordSync FetchChunkRecords", "err", err)
		}
	} else {
		synlog.Info("ChunkRecordSync GetBestChainPids is nil")
	}
}

//FetchChunkRecords    pid  start end   ChunkRecord,            blockHeight--->chunkhash
func (chain *BlockChain) FetchChunkRecords(start int64, end int64, pid []string) (err error) {
	if chain.client == nil {
		synlog.Error("FetchChunkRecords chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}

	synlog.Debug("FetchChunkRecords", "StartHeight", start, "EndHeight", end, "pid", pid)

	count := end - start
	if count < 0 {
		return types.ErrStartBigThanEnd
	}
	reqRec := &types.ReqChunkRecords{
		Start:    start,
		End:      end,
		IsDetail: false,
		Pid:      pid,
	}
	var cb func()
	if count >= int64(MaxReqChunkRecord) { //       MaxReqChunkRecord chunk record
		reqRec.End = reqRec.Start + int64(MaxReqChunkRecord) - 1
		cb = func() {
			chain.ChunkRecordSync()
		}
	}
	//                 chunk
	// TODO
	err = chain.chunkRecordTask.Start(reqRec.Start, reqRec.End, cb, nil)
	if err != nil {
		return err
	}

	msg := chain.client.NewMessage("p2p", types.EventGetChunkRecord, reqRec)
	err = chain.client.Send(msg, false)
	if err != nil {
		synlog.Error("FetchChunkRecords", "client.Send err:", err)
		return err
	}
	return err
}

/*
FetchChunkBlock     ：
   P2P    EventGetChunkBlock(types.RequestGetBlock)，           ，
P2P         ，  blockchain     ， EventReply。
              ，P2P             ，
    EventAddBlocks(types.Blocks)   blockchain   ，
blockchain      EventReply
*/
func (chain *BlockChain) FetchChunkBlock(startHeight, endHeight int64, pid []string, isDownLoad bool) (err error) {
	if chain.client == nil {
		synlog.Error("FetchChunkBlock chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}

	synlog.Debug("FetchChunkBlock input", "StartHeight", startHeight, "EndHeight", endHeight)

	blockcount := endHeight - startHeight
	if blockcount < 0 {
		return types.ErrStartBigThanEnd
	}
	chunkNum, _, end := chain.CalcChunkInfo(startHeight)

	var chunkhash []byte
	for i := 0; i < waitTimeDownLoad; i++ {
		chunkhash, err = chain.blockStore.getRecvChunkHash(chunkNum)
		if err != nil && isDownLoad { // downLoac
			time.Sleep(time.Second)
			continue
		}
		break
	}
	if err != nil {
		return ErrNoChunkInfoToDownLoad
	}

	//  chunk     block
	var requestblock types.ChunkInfoMsg
	requestblock.ChunkHash = chunkhash
	requestblock.Start = startHeight
	requestblock.End = endHeight
	if endHeight > end {
		requestblock.End = end
	}

	var cb func()
	var timeoutcb func(height int64)
	if isDownLoad {
		//        ，
		if requestblock.End < chain.downLoadInfo.EndHeight {
			cb = func() {
				chain.ReqDownLoadChunkBlocks()
			}
			timeoutcb = func(height int64) {
				chain.DownLoadChunkTimeOutProc(height)
			}
			chain.UpdateDownLoadStartHeight(requestblock.End + 1)
		} else { //   DownLoad block     ，  DownLoadInfo
			chain.DefaultDownLoadInfo()
		}
		// chunk           chunk
		// TODO
		err = chain.downLoadTask.Start(requestblock.Start, requestblock.End, cb, timeoutcb)
		if err != nil {
			return err
		}
	} else {
		if chain.GetPeerMaxBlkHeight()-requestblock.End > BackBlockNum {
			cb = func() {
				chain.SynBlocksFromPeers()
			}
		}
		// chunk           chunk
		// TODO
		err = chain.syncTask.Start(requestblock.Start, requestblock.End, cb, timeoutcb)
		if err != nil {
			return err
		}
	}
	synlog.Info("FetchChunkBlock", "chunkNum", chunkNum, "Start", requestblock.Start, "End", requestblock.End, "isDownLoad", isDownLoad, "chunkhash", common.ToHex(chunkhash))
	msg := chain.client.NewMessage("p2p", types.EventGetChunkBlock, &requestblock)
	err = chain.client.Send(msg, false)
	if err != nil {
		synlog.Error("FetchChunkBlock", "client.Send err:", err)
		return err
	}
	return err
}
