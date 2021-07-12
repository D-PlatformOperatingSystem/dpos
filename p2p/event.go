// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

var (
	log = log15.New("module", "p2p.manage")
)

/**        p2p     ,     ,
 *            p2p,
 */
func (mgr *Manager) handleSysEvent() {
	mgr.Client.Sub("p2p")
	log.Debug("Manager handleSysEvent start")
	for msg := range mgr.Client.Recv() {
		switch msg.Ty {

		case types.EventTxBroadcast, types.EventBlockBroadcast: //
			mgr.pub2All(msg)
		case types.EventFetchBlocks, types.EventGetMempool, types.EventFetchBlockHeaders:
			mgr.pub2P2P(msg, mgr.p2pCfg.Types[0])
		case types.EventPeerInfo:
			//
			p2pTy := mgr.p2pCfg.Types[0]
			req, _ := msg.Data.(*types.P2PGetPeerReq)
			for _, ty := range mgr.p2pCfg.Types {
				if ty == req.GetP2PType() {
					p2pTy = req.GetP2PType()
				}
			}
			mgr.pub2P2P(msg, p2pTy)

		case types.EventGetNetInfo:
			//
			p2pTy := mgr.p2pCfg.Types[0]
			req, _ := msg.Data.(*types.P2PGetNetInfoReq)
			for _, ty := range mgr.p2pCfg.Types {
				if ty == req.GetP2PType() {
					p2pTy = req.GetP2PType()
				}
			}
			mgr.pub2P2P(msg, p2pTy)

		case types.EventPubTopicMsg, types.EventFetchTopics, types.EventRemoveTopic, types.EventSubTopic:
			p2pTy := mgr.p2pCfg.Types[0]
			mgr.pub2P2P(msg, p2pTy)

		default:
			mgr.pub2P2P(msg, "dht")
			//log.Warn("unknown msgtype", "msg", msg)
			//msg.Reply(mgr.Client.NewMessage("", msg.Ty, types.Reply{Msg: []byte("unknown msgtype")}))
			//continue
		}
	}
	log.Debug("Manager handleSysEvent stop")
}

//   p2p         ,            p2p    ,
func (mgr *Manager) handleP2PSub() {

	//mgr.subChan = mgr.PubSub.Sub("p2p")
	log.Debug("Manager handleP2PSub start")
	//for msg := range mgr.subChan {
	//
	//}

}

// PubBroadCast       p2p    ,
func (mgr *Manager) PubBroadCast(hash string, data interface{}, eventTy int) error {

	exist, _ := mgr.broadcastFilter.ContainsOrAdd(hash, true)
	// eventTy,   =1,   =54
	//log.Debug("PubBroadCast", "eventTy", eventTy, "hash", hash, "exist", exist)
	if exist {
		return nil
	}
	var err error
	if eventTy == types.EventTx {
		//        ，          ，
		err = mgr.Client.Send(mgr.Client.NewMessage("mempool", types.EventTx, data), true)
	} else if eventTy == types.EventBroadcastAddBlock {
		err = mgr.Client.Send(mgr.Client.NewMessage("blockchain", types.EventBroadcastAddBlock, data), true)
	}
	if err != nil {
		log.Error("PubBroadCast", "eventTy", eventTy, "sendMsgErr", err)
	}
	return err
}

//
func (mgr *Manager) pub2All(msg *queue.Message) {

	for _, ty := range mgr.p2pCfg.Types {
		mgr.PubSub.Pub(msg, ty)
	}

}

//
func (mgr *Manager) pub2P2P(msg *queue.Message, p2pType string) {

	mgr.PubSub.Pub(msg, p2pType)
}
