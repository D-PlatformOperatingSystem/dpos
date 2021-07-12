package protocol

import (
	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"

	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/types"
)

var (
	log = log15.New("module", "p2p.protocol")
)

// HandleEvent handle p2p event
func HandleEvent(msg *queue.Message) {

	if eventHander, ok := types.GetEventHandler(msg.Ty); ok {
		//log.Debug("HandleEvent", "msgTy", msg.Ty)
		eventHander(msg)
	} else if eventHandler := GetEventHandler(msg.Ty); eventHandler != nil {
		//log.Debug("HandleEvent2", "msgTy", msg.Ty)
		eventHandler(msg)
	} else {

		log.Error("HandleEvent", "unknown msgTy", msg.Ty)
	}
}
