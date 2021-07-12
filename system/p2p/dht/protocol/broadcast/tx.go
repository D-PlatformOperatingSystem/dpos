// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"encoding/hex"

	"github.com/D-PlatformOperatingSystem/dpos/types"
)

func (protocol *broadcastProtocol) sendTx(tx *types.P2PTx, p2pData *types.BroadCastData, pid string) (doSend bool) {

	txHash := hex.EncodeToString(tx.Tx.Hash())
	ttl := tx.GetRoute().GetTTL()
	isLightSend := ttl >= protocol.p2pCfg.LightTxTTL

	//      ,           Tx
	if addIgnoreSendPeerAtomic(protocol.txSendFilter, txHash, pid) {
		return false
	}

	//log.Debug("P2PSendTx", "txHash", txHash, "ttl", ttl, "peerAddr", peerAddr)
	//     ttl,     
	if ttl > protocol.p2pCfg.MaxTTL { //        
		return false
	}

	//    ttl     
	if isLightSend {
		p2pData.Value = &types.BroadCastData_LtTx{ //     ttl,     
			LtTx: &types.LightTx{
				TxHash: tx.Tx.Hash(),
				Route:  tx.GetRoute(),
			},
		}
	} else {
		p2pData.Value = &types.BroadCastData_Tx{Tx: tx} //  Tx  
	}
	return true
}

func (protocol *broadcastProtocol) recvTx(tx *types.P2PTx, pid string) (err error) {
	if tx.GetTx() == nil {
		return
	}
	txHash := hex.EncodeToString(tx.GetTx().Hash())
	//   id       ,       
	addIgnoreSendPeerAtomic(protocol.txSendFilter, txHash, pid)
	//      
	if protocol.txFilter.AddWithCheckAtomic(txHash, true) {
		return
	}
	//log.Debug("recvTx", "tx", txHash, "ttl", tx.GetRoute().GetTTL(), "peerAddr", peerAddr)
	//             ,  route    
	if tx.GetRoute() == nil {
		tx.Route = &types.P2PRoute{TTL: 1}
	}
	protocol.txFilter.Add(txHash, tx.GetRoute())
	return protocol.postMempool(txHash, tx.GetTx())

}

func (protocol *broadcastProtocol) recvLtTx(tx *types.LightTx, pid, peerAddr, version string) (err error) {

	txHash := hex.EncodeToString(tx.TxHash)
	//   id       ,       
	addIgnoreSendPeerAtomic(protocol.txSendFilter, txHash, pid)
	//               ,       
	if protocol.txFilter.Contains(txHash) {
		return nil
	}

	//log.Debug("recvLtTx", "txHash", txHash, "ttl", tx.GetRoute().GetTTL(), "peerAddr", peerAddr)
	//     ,                
	query := &types.P2PQueryData{}
	query.Value = &types.P2PQueryData_TxReq{
		TxReq: &types.P2PTxReq{
			TxHash: tx.TxHash,
		},
	}
	//        
	if protocol.sendPeer(query, pid, version) != nil {
		log.Error("recvLtTx", "pid", pid, "addr", peerAddr, "err", err)
		return errSendPeer
	}

	return nil
}
