// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"context"
	"sync/atomic"

	prototypes "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
)

type broadcastHandler struct {
	*prototypes.BaseStreamHandler
}

// Handle
func (handler *broadcastHandler) Handle(stream core.Stream) {

	protocol := handler.GetProtocol().(*broadcastProtocol)
	pid := stream.Conn().RemotePeer()
	sPid := pid.Pretty()
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	var data types.MessageBroadCast
	err := prototypes.ReadStream(&data, stream)
	if err != nil {
		log.Error("Handle", "pid", pid.Pretty(), "addr", peerAddr, "err", err)
		return
	}
	_ = protocol.handleReceive(data.Message, sPid, peerAddr, broadcastV1)
	sendNonBlocking(protocol.peerV1, pid)
}

// VerifyRequest verify request
func (handler *broadcastHandler) VerifyRequest(types.Message, *types.MessageComm) bool {
	return true
}

//       ，
func (protocol *broadcastProtocol) addBroadcastPeer(id peer.ID) {
	//         ，
	pCtx, pCancel := context.WithCancel(protocol.Ctx)
	protocol.broadcastPeers[id] = pCancel
	atomic.AddInt32(&protocol.peerV1Num, 1)
	protocol.Host.ConnManager().Protect(id, broadcastTag)
	go protocol.broadcastV1(pCtx, id)
}

//
func (protocol *broadcastProtocol) removeBroadcastPeer(id peer.ID) {
	protocol.Host.ConnManager().Unprotect(id, broadcastTag)
	delete(protocol.broadcastPeers, id)
	atomic.AddInt32(&protocol.peerV1Num, -1)
}

//
func (protocol *broadcastProtocol) handleClassicBroadcast() {

	for {

		select {
		case pid := <-protocol.peerV1:
			//
			if len(protocol.broadcastPeers) >= protocol.p2pCfg.MaxBroadcastPeers {
				break
			}
			_, ok := protocol.broadcastPeers[pid]
			//
			if ok {
				break
			}
			protocol.addBroadcastPeer(pid)

		case pid := <-protocol.exitPeer:
			protocol.removeBroadcastPeer(pid)
		case pid := <-protocol.errPeer:
			//      tag ，
			protocol.Host.ConnManager().UpsertTag(pid, broadcastTag, func(oldVal int) int { return oldVal - 1 })
		case <-protocol.Ctx.Done():
			return

		}
	}
}

//TODO             ，
func (protocol *broadcastProtocol) broadcastV1(peerCtx context.Context, pid peer.ID) {

	var stream core.Stream
	var err error
	outgoing := protocol.ps.Sub(bcTopic)
	sPid := pid.String()
	log.Debug("broadcastV1Start", "pid", sPid)
	defer func() {
		protocol.ps.Unsub(outgoing)
		sendNonBlocking(protocol.exitPeer, pid)
		if stream != nil {
			_ = stream.Reset()
		}
		if err != nil {
			sendNonBlocking(protocol.errPeer, pid)
		}
		log.Debug("broadcastV1End", "pid", sPid)
	}()

	for {
		select {
		case data := <-outgoing:
			sendData, doSend := protocol.handleSend(data, sPid)
			if !doSend {
				break //ignore send
			}
			//    MessageBroadCast
			broadData := &types.MessageBroadCast{
				Message: sendData}

			stream, err = prototypes.NewStream(protocol.Host, pid, broadcastV1)
			if err != nil {
				log.Error("broadcastV1", "pid", sPid, "NewStreamErr", err)
				return
			}

			err = prototypes.WriteStream(broadData, stream)
			if err != nil {
				log.Error("broadcastV1", "pid", sPid, "WriteStream err", err)
				return
			}
			err = prototypes.CloseStream(stream)
			if err != nil {
				log.Error("broadcastV1", "pid", sPid, "CloseStream err", err)
				return
			}
		case <-peerCtx.Done():
			return

		}
	}

}

//             ，
func sendNonBlocking(ch chan peer.ID, id peer.ID) {
	select {
	case ch <- id:
	default:
	}
}
