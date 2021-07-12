package peer

import (
	"fmt"
	"sync"

	core "github.com/libp2p/go-libp2p-core"

	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/net"
	prototypes "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/types"
	p2pty "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

type peerPubSub struct {
	*prototypes.BaseProtocol
	p2pCfg      *p2pty.P2PSubConfig
	mutex       sync.RWMutex
	pubsubOp    *net.PubSub
	topicMoudle sync.Map
}

// InitProtocol init protocol
func (p *peerPubSub) InitProtocol(env *prototypes.P2PEnv) {
	p.P2PEnv = env
	p.p2pCfg = env.SubConfig
	p.pubsubOp = env.Pubsub
	//
	prototypes.RegisterEventHandler(types.EventSubTopic, p.handleSubTopic)
	//    topic
	prototypes.RegisterEventHandler(types.EventFetchTopics, p.handleGetTopics)
	//
	prototypes.RegisterEventHandler(types.EventRemoveTopic, p.handleRemoveTopc)
	//
	prototypes.RegisterEventHandler(types.EventPubTopicMsg, p.handlePubMsg)
}

//    topic
func (p *peerPubSub) handleSubTopic(msg *queue.Message) {
	//           topic
	//  dplatformos
	subtopic := msg.GetData().(*types.SubTopic)
	topic := subtopic.GetTopic()
	//check topic
	moduleName := subtopic.GetModule()
	//   ，
	if !p.pubsubOp.HasTopic(topic) {
		err := p.pubsubOp.JoinAndSubTopic(topic, p.subCallBack) //  topic
		if err != nil {
			log.Error("peerPubSub", "err", err)
			msg.Reply(p.GetQueueClient().NewMessage("", types.EventSubTopic, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
			return
		}
	}

	var reply types.SubTopicReply
	reply.Status = true
	reply.Msg = fmt.Sprintf("subtopic %v success", topic)
	msg.Reply(p.GetQueueClient().NewMessage("", types.EventSubTopic, &types.Reply{IsOk: true, Msg: types.Encode(&reply)}))

	p.mutex.Lock()
	defer p.mutex.Unlock()

	moudles, ok := p.topicMoudle.Load(topic)
	if ok {
		moudles.(map[string]bool)[moduleName] = true
	} else {
		moudles := make(map[string]bool)
		moudles[moduleName] = true
		p.topicMoudle.Store(topic, moudles)
		return
	}
	p.topicMoudle.Store(topic, moudles)

	//
}

//
func (p *peerPubSub) subCallBack(topic string, msg net.SubMsg) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	moudles, ok := p.topicMoudle.Load(topic)
	if !ok {
		return
	}

	for moudleName := range moudles.(map[string]bool) {
		client := p.GetQueueClient()
		newmsg := client.NewMessage(moudleName, types.EventReceiveSubData, &types.TopicData{Topic: topic, From: core.PeerID(msg.From).String(), Data: msg.Data}) //       )
		client.Send(newmsg, false)
	}
}

//         topic
func (p *peerPubSub) handleGetTopics(msg *queue.Message) {
	_, ok := msg.GetData().(*types.FetchTopicList)
	if !ok {
		msg.Reply(p.GetQueueClient().NewMessage("", types.EventFetchTopics, &types.Reply{IsOk: false, Msg: []byte("need *types.FetchTopicList")}))
		return
	}
	//  topic
	topics := p.pubsubOp.GetTopics()
	var reply types.TopicList
	reply.Topics = topics
	msg.Reply(p.GetQueueClient().NewMessage("", types.EventFetchTopics, &types.Reply{IsOk: true, Msg: types.Encode(&reply)}))
}

//          topic
func (p *peerPubSub) handleRemoveTopc(msg *queue.Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	v, ok := msg.GetData().(*types.RemoveTopic)
	if !ok {

		msg.Reply(p.GetQueueClient().NewMessage("", types.EventRemoveTopic, &types.Reply{IsOk: false, Msg: []byte("need *types.RemoveTopic")}))
		return
	}

	vmdoules, ok := p.topicMoudle.Load(v.GetTopic())
	if !ok || len(vmdoules.(map[string]bool)) == 0 {
		msg.Reply(p.GetQueueClient().NewMessage("", types.EventRemoveTopic, &types.Reply{IsOk: false, Msg: []byte("this module no sub this topic")}))
		return
	}
	modules := vmdoules.(map[string]bool)
	delete(modules, v.GetModule()) //       module
	var reply types.RemoveTopicReply
	reply.Topic = v.GetTopic()
	reply.Status = true

	if len(modules) != 0 {
		msg.Reply(p.GetQueueClient().NewMessage("", types.EventRemoveTopic, &types.Reply{IsOk: true, Msg: types.Encode(&reply)}))
		return
	}

	p.pubsubOp.RemoveTopic(v.GetTopic())
	msg.Reply(p.GetQueueClient().NewMessage("", types.EventRemoveTopic, &types.Reply{IsOk: true, Msg: types.Encode(&reply)}))
}

//  Topic
func (p *peerPubSub) handlePubMsg(msg *queue.Message) {
	v, ok := msg.GetData().(*types.PublishTopicMsg)
	if !ok {
		msg.Reply(p.GetQueueClient().NewMessage("", types.EventPubTopicMsg, &types.Reply{IsOk: false, Msg: []byte("need *types.PublishTopicMsg")}))
		return
	}
	var isok = true
	var replyinfo = "push success"
	err := p.pubsubOp.Publish(v.GetTopic(), v.GetMsg())
	if err != nil {
		//publish msg failed
		isok = false
		replyinfo = err.Error()
	}
	msg.Reply(p.GetQueueClient().NewMessage("", types.EventPubTopicMsg, &types.Reply{IsOk: isok, Msg: []byte(replyinfo)}))
}
