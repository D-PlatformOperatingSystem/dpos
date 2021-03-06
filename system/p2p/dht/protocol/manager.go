package protocol

import (
	"fmt"

	"github.com/D-PlatformOperatingSystem/dpos/queue"
)

//TODO

//Initializer is a initial function which any protocol should have.
type Initializer func(env *P2PEnv)

var (
	protocolInitializerArray []Initializer
)

//RegisterProtocolInitializer registers the initial function.
func RegisterProtocolInitializer(initializer Initializer) {
	protocolInitializerArray = append(protocolInitializerArray, initializer)
}

//InitAllProtocol initials all protocols.
func InitAllProtocol(env *P2PEnv) {
	for _, initializer := range protocolInitializerArray {
		initializer(env)
	}
}

// EventHandler handle dplatformos event
type EventHandler func(*queue.Message)

var (
	eventHandlerMap = make(map[int64]EventHandler)
)

// RegisterEventHandler registers a handler with an event ID.
func RegisterEventHandler(eventID int64, handler EventHandler) {
	if handler == nil {
		panic(fmt.Sprintf("addEventHandler, handler is nil, id=%d", eventID))
	}
	if _, dup := eventHandlerMap[eventID]; dup {
		panic(fmt.Sprintf("addEventHandler, duplicate handler, id=%d, len=%d", eventID, len(eventHandlerMap)))
	}
	eventHandlerMap[eventID] = handler
}

// GetEventHandler gets event handler by event ID.
func GetEventHandler(eventID int64) EventHandler {
	return eventHandlerMap[eventID]
}

// ClearEventHandler clear event handler map, plugin    p2p    ，       ，
func ClearEventHandler() {
	for k := range eventHandlerMap {
		delete(eventHandlerMap, k)
	}
}
