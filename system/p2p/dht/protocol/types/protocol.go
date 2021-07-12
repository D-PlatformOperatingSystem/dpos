// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types protocol and stream register	`
package types

import (
	"context"
	"reflect"
	"time"

	"github.com/libp2p/go-libp2p-core/metrics"

	ds "github.com/ipfs/go-datastore"
	discovery "github.com/libp2p/go-libp2p-discovery"
	kbt "github.com/libp2p/go-libp2p-kbucket"

	"github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/net"

	"github.com/D-PlatformOperatingSystem/dpos/p2p"

	"github.com/D-PlatformOperatingSystem/dpos/queue"
	p2pty "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	protocolTypeMap = make(map[string]reflect.Type)
)

// IProtocol protocol interface
type IProtocol interface {
	InitProtocol(*P2PEnv)
	GetP2PEnv() *P2PEnv
}

// RegisterProtocol       
func RegisterProtocol(typeName string, proto IProtocol) {

	if proto == nil {
		panic("RegisterProtocol, protocol is nil, msgId=" + typeName)
	}
	if _, dup := protocolTypeMap[typeName]; dup {
		panic("RegisterProtocol, protocol is nil, msgId=" + typeName)
	}

	protoType := reflect.TypeOf(proto)

	if protoType.Kind() == reflect.Ptr {
		protoType = protoType.Elem()
	}
	protocolTypeMap[typeName] = protoType
}

func init() {
	RegisterProtocol("BaseProtocol", &BaseProtocol{})
}

// ProtocolManager     
type ProtocolManager struct {
	protoMap map[string]IProtocol
}

// P2PEnv p2p      
type P2PEnv struct {
	ChainCfg        *types.DplatformOSConfig
	QueueClient     queue.Client
	Host            core.Host
	ConnManager     IConnManager
	PeerInfoManager IPeerInfoManager
	Discovery       *net.Discovery
	P2PManager      *p2p.Manager
	SubConfig       *p2pty.P2PSubConfig
	Pubsub          *net.PubSub
	Ctx             context.Context
	DB              ds.Datastore
	RoutingTable    *kbt.RoutingTable
	*discovery.RoutingDiscovery
}

// RoutingTabler routing table interface
type RoutingTabler interface {
	RoutingTable() *kbt.RoutingTable
}

// IConnManager connection manager interface
type IConnManager interface {
	FetchConnPeers() []peer.ID
	BoundSize() (in int, out int)
	IsNeighbors(id peer.ID) bool
	GetLatencyByPeer(pids []peer.ID) map[string]time.Duration
	GetNetRate() metrics.Stats
	BandTrackerByProtocol() *types.NetProtocolInfos
	RateCaculate(ratebytes float64) string
}

// IPeerInfoManager peer info manager interface
type IPeerInfoManager interface {
	Copy(dest *types.Peer, source *types.P2PPeerInfo)
	Add(pid string, info *types.Peer)
	FetchPeerInfosInMin() []*types.Peer
	GetPeerInfoInMin(key string) *types.Peer
}

// BaseProtocol store public data
type BaseProtocol struct {
	*P2PEnv
}

// Init    
func (p *ProtocolManager) Init(env *P2PEnv) {
	p.protoMap = make(map[string]IProtocol)
	//  P2P          protocol  
	for id, protocolType := range protocolTypeMap {
		log.Debug("InitProtocolManager", "protoTy id", id)
		protoVal := reflect.New(protocolType)
		baseValue := protoVal.Elem().FieldByName("BaseProtocol")
		//      ,     BaseProtocol  
		if baseValue != reflect.ValueOf(nil) && baseValue.Kind() == reflect.Ptr {
			baseValue.Set(reflect.ValueOf(&BaseProtocol{})) // baseprotocal     
		}

		protocol := protoVal.Interface().(IProtocol)
		protocol.InitProtocol(env)
		p.protoMap[id] = protocol

	}

	//  P2P          handler  
	for id, handlerType := range streamHandlerTypeMap {
		log.Debug("InitProtocolManager", "stream handler id", id)
		handlerValue := reflect.New(handlerType)
		baseValue := handlerValue.Elem().FieldByName("BaseStreamHandler")
		//      ,     BaseStreamHandler  
		if baseValue != reflect.ValueOf(nil) && baseValue.Kind() == reflect.Ptr {
			baseValue.Set(reflect.ValueOf(&BaseStreamHandler{}))
		}

		newHandler := handlerValue.Interface().(StreamHandler)
		protoID, msgID := decodeHandlerTypeID(id)
		proto := p.protoMap[protoID]
		newHandler.SetProtocol(proto)
		var baseHandler BaseStreamHandler
		baseHandler.child = newHandler
		baseHandler.SetProtocol(p.protoMap[protoID])
		env.Host.SetStreamHandler(core.ProtocolID(msgID), baseHandler.HandleStream)
	}

}

// InitProtocol      
func (base *BaseProtocol) InitProtocol(data *P2PEnv) {

	base.P2PEnv = data
}

// NewMessageCommon new msg common struct
func (base *BaseProtocol) NewMessageCommon(msgID, pid string, nodePubkey []byte, gossip bool) *types.MessageComm {
	return &types.MessageComm{Version: "",
		NodeId:     pid,
		NodePubKey: nodePubkey,
		Timestamp:  time.Now().Unix(),
		Id:         msgID,
		Gossip:     gossip}

}

// GetP2PEnv get p2p env
func (base *BaseProtocol) GetP2PEnv() *P2PEnv {
	return base.P2PEnv
}

// GetChainCfg get chain cfg
func (base *BaseProtocol) GetChainCfg() *types.DplatformOSConfig {

	return base.ChainCfg

}

// GetQueueClient get dplatformos msg queue client
func (base *BaseProtocol) GetQueueClient() queue.Client {

	return base.QueueClient
}

// GetHost get local host
func (base *BaseProtocol) GetHost() core.Host {

	return base.Host

}

// GetConnsManager get connection manager
func (base *BaseProtocol) GetConnsManager() IConnManager {
	return base.ConnManager

}

// GetPeerInfoManager get peer info manager
func (base *BaseProtocol) GetPeerInfoManager() IPeerInfoManager {
	return base.PeerInfoManager
}

// QueryMempool query mempool
func (base *BaseProtocol) QueryMempool(ty int64, req interface{}) (interface{}, error) {
	return base.QueryModule("mempool", ty, req)
}

// QueryBlockChain query blockchain
func (base *BaseProtocol) QueryBlockChain(ty int64, req interface{}) (interface{}, error) {
	return base.QueryModule("blockchain", ty, req)
}

// QueryModule query msg queue module
func (base *BaseProtocol) QueryModule(module string, msgTy int64, req interface{}) (interface{}, error) {

	client := base.GetQueueClient()
	msg := client.NewMessage(module, msgTy, req)
	err := client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		return nil, err
	}
	return resp.GetData(), nil
}
