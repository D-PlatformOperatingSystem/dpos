// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

//
//import (
//	prototypes "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/types"
//	core "github.com/libp2p/go-libp2p-core"
//)
//
//// broadcast v2
//type broadcastHandlerV2 struct {
//	prototypes.BaseStreamHandler
//}
//
//// ReuseStream   stream,
//func (b *broadcastHandlerV2) ReuseStream() bool {
//	return true
//}
//
//// Handle
//func (b *broadcastHandlerV2) Handle(stream core.Stream) {
//	protocol := b.GetProtocol().(*broadcastProtocol)
//	protocol.newStream <- stream
//}

//       ， broadcast v2    ，
/*func (protocol *broadcastProtocol) manageBroadcastPeers() {

	//         ，      ， 5         ,
	tick := time.NewTicker(time.Second * 5)
	timer := time.NewTimer(time.Minute * 10)

	for {
		select {
		//      ，
		case <-tick.C:
			morePeers := protocol.SubConfig.MaxBroadcastPeers - len(protocol.broadcastPeers)
			peers := protocol.GetConnsManager().FetchConnPeers()
			for i := 0; i < len(peers) && morePeers > 0; i++ {
				pid := peers[i]
				_, ok := protocol.broadcastPeers[pid]
				if !ok {
					protocol.addBroadcastPeer(pid, nil)
					morePeers--
				}
			}

		case <-timer.C:
			tick.Stop()
			timer.Stop()
			tick = time.NewTicker(time.Minute)

		//
		case stream := <-protocol.newStream:
			pid := stream.Conn().RemotePeer()
			//
			if len(protocol.broadcastPeers) >= protocol.SubConfig.MaxBroadcastPeers {
				_ = stream.Reset()
				break
			}
			pCancel, ok := protocol.broadcastPeers[pid]
			//            ,                ,           id  ，    id
			if ok {

				if strings.Compare(pid.String(), stream.Conn().LocalPeer().String()) > 0 {
					//
					pCancel()
					//               ，
					protocol.simultaneousPeers[pid] = struct{}{}
				} else {
					//            ，     stream
					_ = stream.Reset()
					break
				}
			}
			protocol.addBroadcastPeer(pid, stream)
		case pid := <-protocol.exitPeer:
			protocol.removeBroadcastPeer(pid)
		case pid := <-protocol.errPeer:
			//      tag ，
			protocol.Host.ConnManager().UpsertTag(pid, broadcastTag, func(oldVal int) int { return oldVal - 1 })
		case <-protocol.Ctx.Done():
			tick.Stop()
			timer.Stop()
			return
		}
	}
}*/

/* broadcast v2     ，
func (protocol *broadcastProtocol) broadcastV2(pid peer.ID, stream core.Stream, peerCtx context.Context, peerCancel context.CancelFunc) {

	sPid := pid.String()
	outgoing := protocol.ps.Sub(bcTopic, sPid)
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	log.Debug("broadcastV2Start", "pid", sPid, "addr", peerAddr)
	errChan := make(chan error, 2)
	defer func() {
		protocol.ps.Unsub(outgoing)
		err := <-errChan
		if err != nil {
			protocol.errPeer <- pid
		}
		protocol.exitPeer <- pid
		_ = stream.Reset()
		log.Debug("broadcastV2End", "pid", sPid, "addr", peerAddr)
	}()

	// handle broadcast recv
	go func() {
		data := &types.BroadCastData{}
		for {
			//
			err := prototypes.ReadStreamTimeout(data, stream, -1)
			if err != nil {
				log.Error("broadcastV2", "pid", sPid, "addr", peerAddr, "read stream err", err)
				errChan <- err
				peerCancel()
				return
			}
			_ = protocol.handleReceive(data, sPid, peerAddr, broadcastV2)
		}
	}()

	// handle broadcast send
	for {
		var err error
		select {
		case data := <-outgoing:
			sendData, doSend := protocol.handleSend(data, sPid, peerAddr)
			if !doSend {
				break //ignore send
			}
			err = prototypes.WriteStream(sendData, stream)
			if err != nil {
				errChan <- err
				log.Error("broadcastV2", "pid", sPid, "WriteStream err", err)
				return
			}

		case <-peerCtx.Done():
			errChan <- nil
			return
		}
	}
}
*/
