package protocol

import (
	_ "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/broadcast" //
	_ "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/download"  //
	_ "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/headers"   //
	_ "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/peer"      //
	prototypes "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/types"
)

// Init init p2p protocol
func Init(data *prototypes.P2PEnv) {
	manager := &prototypes.ProtocolManager{}
	manager.Init(data)
}
