// Package protocol p2p protocol
package protocol

import (
	"context"

	"github.com/D-PlatformOperatingSystem/dpos/p2p"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	types2 "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	ds "github.com/ipfs/go-datastore"
	core "github.com/libp2p/go-libp2p-core"
	discovery "github.com/libp2p/go-libp2p-discovery"
	kbt "github.com/libp2p/go-libp2p-kbucket"
)

// all protocols
const (
	//p2pstore protocols
	FetchChunk        = "/dplatformos/fetch-chunk/" + types2.Version
	StoreChunk        = "/dplatformos/store-chunk/" + types2.Version
	GetHeader         = "/dplatformos/headers/" + types2.Version
	GetChunkRecord    = "/dplatformos/chunk-record/" + types2.Version
	BroadcastFullNode = "/dplatformos/full-node/" + types2.Version

	//sync protocols
	IsSync        = "/dplatformos/is-sync/" + types2.Version
	IsHealthy     = "/dplatformos/is-healthy/" + types2.Version
	GetLastHeader = "/dplatformos/last-header/" + types2.Version
)

// P2PEnv p2p
type P2PEnv struct {
	Ctx         context.Context
	ChainCfg    *types.DplatformOSConfig
	QueueClient queue.Client
	Host        core.Host
	P2PManager  *p2p.Manager
	SubConfig   *types2.P2PSubConfig
	DB          ds.Datastore
	*discovery.RoutingDiscovery

	RoutingTable *kbt.RoutingTable
}
