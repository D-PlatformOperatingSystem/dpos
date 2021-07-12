// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types
package types

// P2PSubConfig p2p
type P2PSubConfig struct {
	// P2P
	Port int32 `protobuf:"varint,1,opt,name=port" json:"port,omitempty"`
	//
	Seeds []string `protobuf:"bytes,2,rep,name=seeds" json:"seeds,omitempty"`
	//           ttl
	LightTxTTL int32 `protobuf:"varint,3,opt,name=lightTxTTL" json:"lightTxTTL,omitempty"`
	//     ttl, ttl
	MaxTTL int32 `protobuf:"varint,4,opt,name=maxTTL" json:"maxTTL,omitempty"`
	// p2p    ,      /   /

	Channel int32 `protobuf:"varint,5,opt,name=channel" json:"channel,omitempty"`
	//            ,  KB,
	MinLtBlockSize int32 `protobuf:"varint,6,opt,name=minLtBlockSize" json:"minLtBlockSize,omitempty"`
	//  dht
	MaxConnectNum int32 `protobuf:"varint,7,opt,name=maxConnectNum" json:"maxConnectNum,omitempty"`
	//
	BootStraps []string `protobuf:"bytes,8,rep,name=bootStraps" json:"bootStraps,omitempty"`
	//           ,   M
	LtBlockCacheSize int32 `protobuf:"varint,9,opt,name=ltBlockCacheSize" json:"ltBlockCacheSize,omitempty"`

	//          ，         ，        ，NAT

	RelayHop            bool   `protobuf:"varint,10,opt,name=relayHop" json:"relayHop,omitempty"`
	DisableFindLANPeers bool   `protobuf:"varint,11,opt,name=disableFindLANPeers" json:"disableFindLANPeers,omitempty"`
	DHTDataDriver       string `protobuf:"bytes,12,opt,name=DHTDataDriver" json:"DHTDataDriver,omitempty"`
	DHTDataPath         string `protobuf:"bytes,13,opt,name=DHTDataPath" json:"DHTDataPath,omitempty"`
	DHTDataCache        int32  `protobuf:"varint,14,opt,name=DHTDataCache" json:"DHTDataCache,omitempty"`
	//
	IsFullNode bool `protobuf:"varint,15,opt,name=isFullNode" json:"isFullNode,omitempty"`
	//
	MaxBroadcastPeers int `protobuf:"varint,16,opt,name=maxBroadcastPeers" json:"maxBroadcastPeers,omitempty"`
	//pub sub
	DisablePubSubMsgSign bool `protobuf:"varint,17,opt,name=disablePubSubMsgSign" json:"disablePubSubMsgSign,omitempty"`
	//        ，     NAT     ，RelayEnable=true，            。
	RelayEnable bool `protobuf:"varint,18,opt,name=relayEnable" json:"relayEnable,omitempty"`
	//
	RelayNodeAddr []string `protobuf:"varint,19,opt,name=relayNodeAddr" json:"relayNodeAddr,omitempty"`
}
