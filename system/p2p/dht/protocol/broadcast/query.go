// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"bytes"
	"encoding/hex"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/merkle"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

func (protocol *broadcastProtocol) sendQueryData(query *types.P2PQueryData, p2pData *types.BroadCastData, pid string) bool {
	log.Debug("P2PSendQueryData", "pid", pid)
	p2pData.Value = &types.BroadCastData_Query{Query: query}
	return true
}

func (protocol *broadcastProtocol) sendQueryReply(rep *types.P2PBlockTxReply, p2pData *types.BroadCastData, pid string) bool {
	log.Debug("P2PSendQueryReply", "pid", pid)
	p2pData.Value = &types.BroadCastData_BlockRep{BlockRep: rep}
	return true
}

func (protocol *broadcastProtocol) recvQueryData(query *types.P2PQueryData, pid, peerAddr, version string) error {

	var reply interface{}
	if txReq := query.GetTxReq(); txReq != nil {

		txHash := hex.EncodeToString(txReq.TxHash)
		log.Debug("recvQueryTx", "txHash", txHash, "pid", pid)
		// mempool
		resp, err := protocol.QueryMempool(types.EventTxListByHash, &types.ReqTxHashList{Hashes: []string{string(txReq.TxHash)}})
		if err != nil {
			log.Error("recvQuery", "queryMempoolErr", err)
			return errQueryMempool
		}

		txList, _ := resp.(*types.ReplyTxList)
		//
		if len(txList.GetTxs()) != 1 || txList.GetTxs()[0] == nil {
			log.Error("recvQueryTx", "txHash", txHash, "err", "recvNilTxFromMempool")
			return errRecvMempool
		}
		p2pTx := &types.P2PTx{Tx: txList.Txs[0]}
		//           , ttl   1
		p2pTx.Route = &types.P2PRoute{TTL: 1}
		reply = p2pTx
		//              ，          ，
		removeIgnoreSendPeerAtomic(protocol.txSendFilter, txHash, pid)

	} else if blcReq := query.GetBlockTxReq(); blcReq != nil {

		log.Debug("recvQueryBlockTx", "hash", blcReq.BlockHash, "queryCount", len(blcReq.TxIndices), "pid", pid)
		blcHash, _ := common.FromHex(blcReq.BlockHash)
		if blcHash != nil {
			resp, err := protocol.QueryBlockChain(types.EventGetBlockByHashes, &types.ReqHashes{Hashes: [][]byte{blcHash}})
			if err != nil {
				log.Error("recvQueryBlockTx", "queryBlockChainErr", err)
				return errQueryBlockChain
			}
			blocks, ok := resp.(*types.BlockDetails)
			if !ok || len(blocks.Items) != 1 || blocks.Items[0].Block == nil {
				log.Error("recvQueryBlockTx", "blockHash", blcReq.BlockHash, "err", "blockNotExist")
				return errRecvBlockChain
			}
			block := blocks.Items[0].Block
			blockRep := &types.P2PBlockTxReply{BlockHash: blcReq.BlockHash}
			blockRep.TxIndices = blcReq.TxIndices
			for _, idx := range blcReq.TxIndices {
				blockRep.Txs = append(blockRep.Txs, block.Txs[idx])
			}
			//
			if len(blockRep.TxIndices) == 0 {
				blockRep.Txs = block.Txs
			}
			reply = blockRep
		}
	}

	if reply != nil {
		if err := protocol.sendPeer(reply, pid, version); err != nil {
			log.Error("recvQueryData", "pid", pid, "addr", peerAddr, "err", err)
			return errSendPeer
		}
	}
	return nil
}

func (protocol *broadcastProtocol) recvQueryReply(rep *types.P2PBlockTxReply, pid, peerAddr, version string) (err error) {

	log.Debug("recvQueryReply", "hash", rep.BlockHash, "queryTxsCount", len(rep.GetTxIndices()), "pid", pid)
	val, exist := protocol.ltBlockCache.Remove(rep.BlockHash)
	block, _ := val.(*types.Block)
	//not exist in cache or nil block
	if !exist || block == nil {
		log.Error("recvQueryReply", "hash", rep.BlockHash, "exist", exist, "isBlockNil", block == nil)
		return errLtBlockNotExist
	}
	for i, idx := range rep.TxIndices {
		block.Txs[idx] = rep.Txs[i]
	}

	//
	if len(rep.TxIndices) == 0 {
		block.Txs = rep.Txs
	}

	//   root hash
	if bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(protocol.BaseProtocol.ChainCfg, block.GetHeight(), block.Txs)) {
		log.Debug("recvQueryReply", "height", block.GetHeight())
		//   blockchain
		if err := protocol.postBlockChain(rep.BlockHash, pid, block); err != nil {
			log.Error("recvQueryReply", "height", block.GetHeight(), "send block to blockchain Error", err.Error())
			return errSendBlockChain
		}
		return nil
	}

	//          ，             ， txIndices         ,
	if len(rep.TxIndices) == 0 {
		log.Error("recvQueryReply", "height", block.GetHeight(), "hash", rep.BlockHash, "err", errBuildBlockFailed)
		return errBuildBlockFailed
	}

	log.Debug("recvQueryReply", "getBlockRetry", block.GetHeight(), "hash", rep.BlockHash)

	query := &types.P2PQueryData{
		Value: &types.P2PQueryData_BlockTxReq{
			BlockTxReq: &types.P2PBlockTxReq{
				BlockHash: rep.BlockHash,
				TxIndices: nil,
			},
		},
	}
	block.Txs = nil
	protocol.ltBlockCache.Add(rep.BlockHash, block, block.Size())
	//query peer
	if err = protocol.sendPeer(query, pid, version); err != nil {
		log.Error("recvQueryReply", "pid", pid, "addr", peerAddr, "err", err)
		protocol.ltBlockCache.Remove(rep.BlockHash)
		protocol.blockFilter.Remove(rep.BlockHash)
		return errSendPeer
	}
	return
}
