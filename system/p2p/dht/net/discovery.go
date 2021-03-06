package net

import (
	"context"
	"fmt"

	"github.com/D-PlatformOperatingSystem/dpos/types"

	protocol "github.com/libp2p/go-libp2p-core/protocol"

	p2pty "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/types"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	kbt "github.com/libp2p/go-libp2p-kbucket"

	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

var (
	log = log15.New("module", "p2p.dht")
)

const (
	// Deprecated       ，    ，TODO
	classicDhtProtoID = "/ipfs/kad/%s/1.0.0/%d"
	dhtProtoID        = "/%s-%d/kad/1.0.0" //title-channel/kad/1.0.0
)

// Discovery dht discovery
type Discovery struct {
	kademliaDHT      *dht.IpfsDHT
	RoutingDiscovery *discovery.RoutingDiscovery
	mdnsService      *mdns
	ctx              context.Context
	subCfg           *p2pty.P2PSubConfig
	bootstrapnodes   []peer.AddrInfo
	host             host.Host
}

// InitDhtDiscovery init dht discovery
func InitDhtDiscovery(ctx context.Context, host host.Host, peersInfo []peer.AddrInfo, chainCfg *types.DplatformOSConfig, subCfg *p2pty.P2PSubConfig) *Discovery {

	// Make the DHT,   ID       。
	//     DHTProto        IPFS  ，dhtproto=/ipfs/kad/1.0.0
	d := new(Discovery)
	opt := opts.Protocols(protocol.ID(fmt.Sprintf(dhtProtoID, chainCfg.GetTitle(), subCfg.Channel)),
		protocol.ID(fmt.Sprintf(classicDhtProtoID, chainCfg.GetTitle(), subCfg.Channel)))
	kademliaDHT, err := dht.New(ctx, host, opt)
	if err != nil {
		panic(err)
	}
	d.kademliaDHT = kademliaDHT
	d.ctx = ctx
	d.bootstrapnodes = peersInfo
	d.subCfg = subCfg
	d.host = host
	return d

}

//Start  the dht
func (d *Discovery) Start() {
	//      ，  addrbook
	initInnerPeers(d.host, d.bootstrapnodes, d.subCfg)
	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	if err := d.kademliaDHT.Bootstrap(d.ctx); err != nil {
		//panic(err)
		log.Error("Bootstrap", "err", err.Error())
	}
	d.RoutingDiscovery = discovery.NewRoutingDiscovery(d.kademliaDHT)
}

//Close close the dht
func (d *Discovery) Close() error {
	if d.kademliaDHT != nil {
		return d.kademliaDHT.Close()
	}
	d.CloseFindLANPeers()
	return nil
}

// FindLANPeers
func (d *Discovery) FindLANPeers(host host.Host, serviceTag string) (<-chan peer.AddrInfo, error) {
	mdns, err := initMDNS(d.ctx, host, serviceTag)
	if err != nil {
		return nil, err
	}
	d.mdnsService = mdns
	return d.mdnsService.PeerChan(), nil
}

// CloseFindLANPeers close peers
func (d *Discovery) CloseFindLANPeers() {
	if d.mdnsService != nil {
		d.mdnsService.Service.Close()
	}
}

// ListPeers routingTable
func (d *Discovery) ListPeers() []peer.ID {
	if d.kademliaDHT == nil {
		return nil
	}
	return d.kademliaDHT.RoutingTable().ListPeers()
}

// RoutingTableSize routingTable size
func (d *Discovery) RoutingTableSize() int {
	if d.kademliaDHT == nil {
		return 0
	}
	return d.kademliaDHT.RoutingTable().Size()
}

// FindLocalPeer   pid     DHT   peer
func (d *Discovery) FindLocalPeer(pid peer.ID) peer.AddrInfo {
	if d.kademliaDHT == nil {
		return peer.AddrInfo{}
	}
	return d.kademliaDHT.FindLocal(pid)
}

// FindLocalPeers find local peers
func (d *Discovery) FindLocalPeers(pids []peer.ID) []peer.AddrInfo {
	var addrinfos []peer.AddrInfo
	for _, pid := range pids {
		addrinfos = append(addrinfos, d.FindLocalPeer(pid))
	}
	return addrinfos
}

// Update update peer
func (d *Discovery) Update(pid peer.ID) error {
	_, err := d.kademliaDHT.RoutingTable().Update(pid)
	return err
}

// FindNearestPeers find nearest peers
func (d *Discovery) FindNearestPeers(pid peer.ID, count int) []peer.ID {
	if d.kademliaDHT == nil {
		return nil
	}

	return d.kademliaDHT.RoutingTable().NearestPeers(kbt.ConvertPeerID(pid), count)
}

// Remove remove peer
func (d *Discovery) Remove(pid peer.ID) {
	if d.kademliaDHT == nil {
		return
	}
	d.kademliaDHT.RoutingTable().Remove(pid)

}

// RoutingTable get routing table
func (d *Discovery) RoutingTable() *kbt.RoutingTable {
	return d.kademliaDHT.RoutingTable()
}
