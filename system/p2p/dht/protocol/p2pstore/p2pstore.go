package p2pstore

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol"
	types2 "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

const maxConcurrency = 10

var log = log15.New("module", "protocol.p2pstore")

//Protocol ...
type Protocol struct {
	*protocol.P2PEnv //

	//cache the notify message when other peers notify this node to store chunk.
	notifying sync.Map
	//send the message in <notifying> to this queue if chunk does not exist.
	notifyingQueue chan *types.ChunkInfoMsg
	//chunks that full node can provide without checking.
	chunkWhiteList sync.Map

	//          ，
	healthyRoutingTable *kb.RoutingTable

	//      chunk    ，
	localChunkInfo      map[string]LocalChunkInfo
	localChunkInfoMutex sync.RWMutex

	concurrency int64
}

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

//InitProtocol initials the protocol.
func InitProtocol(env *protocol.P2PEnv) {
	p := &Protocol{
		P2PEnv:              env,
		healthyRoutingTable: kb.NewRoutingTable(dht.KValue, kb.ConvertPeerID(env.Host.ID()), time.Minute, env.Host.Peerstore()),
		notifyingQueue:      make(chan *types.ChunkInfoMsg, 1024),
	}
	//routing table       healthy routing table
	rm := p.RoutingTable.PeerRemoved
	p.RoutingTable.PeerRemoved = func(id peer.ID) {
		rm(id)
		p.healthyRoutingTable.Remove(id)
	}
	p.initLocalChunkInfoMap()

	//  p2p    ，
	p.Host.SetStreamHandler(protocol.FetchChunk, protocol.HandlerWithClose(p.handleStreamFetchChunk)) //    ，
	p.Host.SetStreamHandler(protocol.StoreChunk, protocol.HandlerWithAuth(p.handleStreamStoreChunks))
	p.Host.SetStreamHandler(protocol.GetHeader, protocol.HandlerWithAuthAndSign(p.handleStreamGetHeader))
	p.Host.SetStreamHandler(protocol.GetChunkRecord, protocol.HandlerWithAuthAndSign(p.handleStreamGetChunkRecord))
	//    eventHandler，    blockchain
	protocol.RegisterEventHandler(types.EventNotifyStoreChunk, protocol.EventHandlerWithRecover(p.handleEventNotifyStoreChunk))
	protocol.RegisterEventHandler(types.EventGetChunkBlock, protocol.EventHandlerWithRecover(p.handleEventGetChunkBlock))
	protocol.RegisterEventHandler(types.EventGetChunkBlockBody, protocol.EventHandlerWithRecover(p.handleEventGetChunkBlockBody))
	protocol.RegisterEventHandler(types.EventGetChunkRecord, protocol.EventHandlerWithRecover(p.handleEventGetChunkRecord))

	go p.syncRoutine()
	go func() {
		for i := 0; i < 3; i++ { //           healthy routing table
			p.updateHealthyRoutingTable()
			time.Sleep(time.Second * 1)
		}

		ticker1 := time.NewTicker(time.Minute)
		ticker2 := time.NewTicker(types2.RefreshInterval)
		ticker3 := time.NewTicker(types2.CheckHealthyInterval)
		ticker4 := time.NewTicker(time.Hour)

		for {
			select {
			case <-ticker1.C:
				p.updateChunkWhiteList()
			case <-ticker2.C:
				p.republish()
			case <-ticker3.C:
				p.updateHealthyRoutingTable()
				p.advertiseFullNode()
			case <-ticker4.C:
				//debug info
				p.localChunkInfoMutex.Lock()
				log.Info("debugLocalChunk", "local chunk hash len", len(p.localChunkInfo))
				p.localChunkInfoMutex.Unlock()
				p.debugFullNode()
				log.Info("debug healthy peers", "======== amount", p.healthyRoutingTable.Size())
				for _, pid := range p.healthyRoutingTable.ListPeers() {
					log.Info("LatencyEWMA", "pid", pid, "maddr", p.Host.Peerstore().Addrs(pid), "latency", p.Host.Peerstore().LatencyEWMA(pid))
				}
				log.Info("debug length", "notifying msg len", len(p.notifyingQueue))
			case <-p.Ctx.Done():
				return
			}
		}
	}()
}

func (p *Protocol) syncRoutine() {
	for {
		select {
		case <-p.Ctx.Done():
			return
		case info := <-p.notifyingQueue:
			p.syncChunk(info)
			p.notifying.Delete(hex.EncodeToString(info.ChunkHash))
		}
	}
}

func (p *Protocol) syncChunk(info *types.ChunkInfoMsg) {
	//     p2pStore，
	if err := p.updateChunk(info); err == nil {
		return
	}

	var bodys *types.BlockBodys
	bodys, _ = p.getChunkFromBlockchain(info)
	if bodys == nil {
		//blockchain      ，
		bodys, _, _ = p.mustFetchChunk(p.Ctx, info, true)
	}
	if bodys == nil {
		log.Error("syncChunk error", "chunkhash", hex.EncodeToString(info.ChunkHash), "start", info.Start)
		return
	}
	if err := p.addChunkBlock(info, bodys); err != nil {
		log.Error("syncChunk", "addChunkBlock error", err)
		return
	}
	log.Info("syncChunk", "chunkhash", hex.EncodeToString(info.ChunkHash), "start", info.Start)
}

func (p *Protocol) updateChunkWhiteList() {
	p.chunkWhiteList.Range(func(k, v interface{}) bool {
		t := v.(time.Time)
		if time.Since(t) > time.Minute*10 {
			p.chunkWhiteList.Delete(k)
		}
		return true
	})
}

func (p *Protocol) advertiseFullNode(opts ...discovery.Option) {
	if !p.SubConfig.IsFullNode {
		return
	}
	_, err := p.Advertise(p.Ctx, protocol.BroadcastFullNode, opts...)
	if err != nil {
		log.Error("advertiseFullNode", "error", err)
	}
}

//TODO
//debug info, to delete soon
func (p *Protocol) debugFullNode() {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	peerInfos, err := p.FindPeers(ctx, protocol.BroadcastFullNode)
	if err != nil {
		log.Error("debugFullNode", "FindPeers error", err)
		return
	}
	var count int
	for peerInfo := range peerInfos {
		count++
		log.Info("debugFullNode", "ID", peerInfo.ID, "maddrs", peerInfo.Addrs)
	}
	log.Info("debugFullNode", "total count", count)
}
