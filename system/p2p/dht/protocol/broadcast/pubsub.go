// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"encoding/hex"
	"runtime"
	"time"

	prototypes "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/types"
	core "github.com/libp2p/go-libp2p-core"

	"github.com/D-PlatformOperatingSystem/dpos/p2p/utils"
	"github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/net"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/golang/snappy"
)

const (
	psTxTopic    = "tx/v1.0.0"
	psBlockTopic = "block/v1.0.0"
)

//   libp2p pubsub
type pubSub struct {
	*broadcastProtocol
}

// new pub sub
func newPubSub(b *broadcastProtocol) *pubSub {
	return &pubSub{b}
}

//       ，
func (p *pubSub) broadcast() {

	//TODO check net is sync

	txIncoming := make(chan net.SubMsg, 1024) //      ,
	txOutgoing := p.ps.Sub(psTxTopic)         //      ,
	//
	blockIncoming := make(chan net.SubMsg, 128)
	blockOutgoing := p.ps.Sub(psBlockTopic)

	// pub sub topic
	err := p.Pubsub.JoinAndSubTopic(psTxTopic, p.callback(txIncoming))
	if err != nil {
		log.Error("pubsub broadcast", "join tx topic err", err)
		return
	}
	err = p.Pubsub.JoinAndSubTopic(psBlockTopic, p.callback(blockIncoming))
	if err != nil {
		log.Error("pubsub broadcast", "join block topic err", err)
		return
	}

	//      topic    ，     ，
	for len(p.Pubsub.FetchTopicPeers(psTxTopic)) == 0 {
		time.Sleep(time.Second * 2)
		log.Warn("pub sub broadcast", "info", "no peers available")
	}

	//              ，
	//    ,           ，
	cpu := runtime.NumCPU()
	for i := 0; i < cpu; i++ {
		go p.handlePubMsg(psTxTopic, txOutgoing)
		go p.handleSubMsg(psTxTopic, txIncoming, p.txFilter)
	}

	//
	go p.handlePubMsg(psBlockTopic, blockOutgoing)
	go p.handleSubMsg(psBlockTopic, blockIncoming, p.blockFilter)
}

//
func (p *pubSub) handlePubMsg(topic string, out chan interface{}) {

	defer p.ps.Unsub(out)
	buf := make([]byte, 0)
	var err error
	for {
		select {
		case data, ok := <-out: //
			if !ok {
				return
			}
			msg := data.(types.Message)
			raw := p.encodeMsg(msg, &buf)
			if err != nil {
				log.Error("handlePubMsg", "topic", topic, "hash", p.getMsgHash(topic, msg), "err", err)
				break
			}

			err = p.Pubsub.Publish(topic, raw)
			if err != nil {
				log.Error("handlePubMsg", "publish err", err)
			}

		case <-p.Ctx.Done():
			return
		}
	}
}

//
func (p *pubSub) handleSubMsg(topic string, in chan net.SubMsg, filter *utils.Filterdata) {

	buf := make([]byte, 0)
	var err error
	var msg types.Message
	for {
		select {
		case data, ok := <-in: //
			if !ok {
				return
			}
			msg = p.newMsg(topic)
			err = p.decodeMsg(data.Data, &buf, msg)
			if err != nil {
				log.Error("handleSubMsg", "topic", topic, "decodeMsg err", err)
				break
			}
			hash := p.getMsgHash(topic, msg)
			//       ,
			if filter.Contains(hash) {
				break
			}

			//TODO        ，
			filter.Add(hash, struct{}{})

			//
			if topic == psTxTopic {
				err = p.postMempool(hash, msg.(*types.Transaction))
			} else {
				err = p.postBlockChain(hash, data.ReceivedFrom.String(), msg.(*types.Block))
			}

			if err != nil {
				log.Error("handleSubMsg", "topic", topic, "hash", hash, "post msg err", err)
			}

		case <-p.Ctx.Done():
			return
		}
	}

}

//
func (p *pubSub) getMsgHash(topic string, msg types.Message) string {
	if topic == psTxTopic {
		return hex.EncodeToString(msg.(*types.Transaction).Hash())
	}
	return hex.EncodeToString(msg.(*types.Block).Hash(p.ChainCfg))
}

//         s
func (p *pubSub) newMsg(topic string) types.Message {
	if topic == psTxTopic {
		return &types.Transaction{}
	}
	return &types.Block{}
}

//
func (p *pubSub) callback(out chan<- net.SubMsg) net.SubCallBack {
	return func(topic string, msg net.SubMsg) {
		out <- msg
	}
}

//        ，
func (p *pubSub) encodeMsg(msg types.Message, pbuf *[]byte) []byte {
	buf := *pbuf
	buf = buf[:cap(buf)]
	raw := types.Encode(msg)
	buf = snappy.Encode(buf, raw)
	*pbuf = buf
	//   raw          ，
	if cap(raw) >= len(buf) {
		raw = raw[:len(buf)]
	} else {
		raw = make([]byte, len(buf))
	}
	copy(raw, buf)
	return raw
}

//
func (p *pubSub) decodeMsg(raw []byte, pbuf *[]byte, msg types.Message) error {

	var err error
	buf := *pbuf
	buf = buf[:cap(buf)]
	buf, err = snappy.Decode(buf, raw)
	if err != nil {
		log.Error("pubSub decodeMsg", "snappy decode err", err)
		return errSnappyDecode
	}
	//      buf
	*pbuf = buf
	err = types.Decode(buf, msg)
	if err != nil {
		log.Error("pubSub decodeMsg", "pb decode err", err)
		return types.ErrDecode
	}

	return nil
}

// broadcast pub sub
type pubsubHandler struct {
	prototypes.BaseStreamHandler
}

// Handle,        ，
func (b *pubsubHandler) Handle(stream core.Stream) {
}
