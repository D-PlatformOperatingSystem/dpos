// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	core "github.com/libp2p/go-libp2p-core"
)

var (
	log                  = log15.New("module", "p2p.protocol.types")
	streamHandlerTypeMap = make(map[string]reflect.Type)
)

// RegisterStreamHandler   typeName,msgID,
func RegisterStreamHandler(typeName, msgID string, handler StreamHandler) {

	if handler == nil {
		panic("RegisterStreamHandler, handler is nil, msgId=" + msgID)
	}

	if _, exist := protocolTypeMap[typeName]; !exist {
		panic("RegisterStreamHandler, protocol type not exist, msgId=" + msgID)
	}

	typeID := formatHandlerTypeID(typeName, msgID)

	if _, dup := streamHandlerTypeMap[typeID]; dup {
		panic("addStreamHandler, handler is nil, typeID=" + typeID)
	}
	//streamHandlerTypeMap[typeID] = reflect.TypeOf(handler)
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() == reflect.Ptr {
		handlerType = handlerType.Elem()
	}
	streamHandlerTypeMap[typeID] = handlerType
}

// StreamHandler stream handler
type StreamHandler interface {
	// GetProtocol get protocol
	GetProtocol() IProtocol
	// SetProtocol        ,     protocol         ,  queue.client
	SetProtocol(protocol IProtocol)
	// VerifyRequest
	VerifyRequest(message types.Message, messageComm *types.MessageComm) bool
	//SignMessage
	SignProtoMessage(message types.Message, host core.Host) ([]byte, error)
	// Handle
	Handle(stream core.Stream)
	// ReuseStream   stream，
	ReuseStream() bool
}

// BaseStreamHandler base stream handler
type BaseStreamHandler struct {
	Protocol IProtocol
	child    StreamHandler
}

// SetProtocol set protocol
func (s *BaseStreamHandler) SetProtocol(protocol IProtocol) {
	s.Protocol = protocol
}

// Handle handle stream
func (s *BaseStreamHandler) Handle(core.Stream) {
}

// SignProtoMessage sign data
func (s *BaseStreamHandler) SignProtoMessage(message types.Message, host core.Host) ([]byte, error) {
	return SignProtoMessage(message, host)
}

// VerifyRequest verify data
func (s *BaseStreamHandler) VerifyRequest(message types.Message, messageComm *types.MessageComm) bool {
	//        ,      ,         true

	return AuthenticateMessage(message, messageComm)

}

// GetProtocol get protocol
func (s *BaseStreamHandler) GetProtocol() IProtocol {
	return s.Protocol
}

// ReuseStream   stream
func (s *BaseStreamHandler) ReuseStream() bool {
	return false
}

// HandleStream stream
func (s *BaseStreamHandler) HandleStream(stream core.Stream) {
	//log.Debug("BaseStreamHandler", "HandlerStream", stream.Conn().RemoteMultiaddr().String(), "proto", stream.Protocol())
	//TODO verify
	s.child.Handle(stream)
	//         stream，      ，      stream
	if s.child.ReuseStream() {
		return
	}
	CloseStream(stream)
}

func formatHandlerTypeID(protocolType, msgID string) string {
	return protocolType + "#" + msgID
}

func decodeHandlerTypeID(typeID string) (string, string) {

	arr := strings.Split(typeID, "#")
	return arr[0], arr[1]
}
