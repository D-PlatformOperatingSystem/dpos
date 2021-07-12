package manage

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/D-PlatformOperatingSystem/dpos/queue"

	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// manage peer info

const diffheightValue = 512

// PeerInfoManager peer info manager
type PeerInfoManager struct {
	peerInfo    sync.Map
	client      queue.Client
	host        host.Host
	blackcache  *TimeCache
	disConnFunc PruePeers
	done        chan struct{}
}

type peerStoreInfo struct {
	storeTime time.Duration
	peer      *types.Peer
}

// Add add peer info
func (p *PeerInfoManager) Add(pid string, info *types.Peer) {
	var storeInfo peerStoreInfo
	storeInfo.storeTime = time.Duration(time.Now().Unix())
	storeInfo.peer = info
	p.peerInfo.Store(pid, &storeInfo)
}

// Copy copy peer info
func (p *PeerInfoManager) Copy(dest *types.Peer, source *types.P2PPeerInfo) {
	dest.Addr = source.GetAddr()
	dest.Name = source.GetName()
	dest.Header = source.GetHeader()
	dest.Self = false
	dest.MempoolSize = source.GetMempoolSize()
	dest.Port = source.GetPort()
	dest.Version = source.GetVersion()
	dest.StoreDBVersion = source.GetStoreDBVersion()
	dest.LocalDBVersion = source.GetLocalDBVersion()
}

// GetPeerInfoInMin get peer info
func (p *PeerInfoManager) GetPeerInfoInMin(key string) *types.Peer {
	v, ok := p.peerInfo.Load(key)
	if !ok {
		return nil
	}
	info := v.(*peerStoreInfo)

	if time.Duration(time.Now().Unix())-info.storeTime > 60 {
		p.peerInfo.Delete(key)
		return nil
	}
	return info.peer
}

// FetchPeerInfosInMin fetch peer info
func (p *PeerInfoManager) FetchPeerInfosInMin() []*types.Peer {

	var peers []*types.Peer
	p.peerInfo.Range(func(key interface{}, value interface{}) bool {
		info := value.(*peerStoreInfo)
		if time.Duration(time.Now().Unix())-info.storeTime > 60 {
			p.peerInfo.Delete(key)
			return true
		}

		peers = append(peers, info.peer)
		return true
	})

	return peers
}

// Start monitor peer info
func (p *PeerInfoManager) Start() {
	for {
		select {
		case <-time.After(time.Minute):
			//      ，
			//	log.Debug("MonitorPeerInfos", "Num", len(p.FetchPeerInfosInMin()))
			msg := p.client.NewMessage("blockchain", types.EventGetLastHeader, nil)
			err := p.client.Send(msg, true)
			if err != nil {
				continue
			}
			resp, err := p.client.WaitTimeout(msg, time.Second*10)
			if err != nil {
				continue
			}
			header, ok := resp.GetData().(*types.Header)
			if !ok {
				continue
			}
			p.prue(header.GetHeight())

		case <-p.done:
			return

		}
	}
}
func (p *PeerInfoManager) prue(height int64) {
	p.peerInfo.Range(func(key interface{}, value interface{}) bool {
		info := value.(*peerStoreInfo)
		if time.Duration(time.Now().Unix())-info.storeTime > 60 {
			p.peerInfo.Delete(key)
			return true
		}
		//check blockheight,    512
		if info.peer.Header.GetHeight()+diffheightValue < height {
			//   Inbound   Outbound
			id, _ := peer.Decode(key.(string))
			conns := p.host.Network().ConnsToPeer(id)
			if len(conns) != 0 && conns[0].Stat().Direction == network.DirOutbound { //outbound
				//remove
				log.Debug("prue", "peer", key, "height", info.peer.Header.GetHeight(), "Direction", conns[0].Stat().Direction)
				p.peerInfo.Delete(key)
				//
				//if beBlack true        ，
				p.disConnFunc(id, true)
			}
		}

		return true
	})
}

// Close close peer info manager
func (p *PeerInfoManager) Close() {
	defer func() {
		if recover() != nil {
			log.Error("channel reclosed")
		}
	}()

	close(p.done)
}

//PruePeers close peer and put it into blacklist is beBlock is true.
type PruePeers func(pids peer.ID, beBlack bool)

// NewPeerInfoManager new peer info manager
func NewPeerInfoManager(host host.Host, cli queue.Client, timecache *TimeCache, callFunc PruePeers) *PeerInfoManager {
	peerInfoManage := &PeerInfoManager{done: make(chan struct{})}
	peerInfoManage.client = cli
	peerInfoManage.host = host
	peerInfoManage.blackcache = timecache
	peerInfoManage.disConnFunc = callFunc
	return peerInfoManage
}
