package peer

import (
	"time"

	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"

	pubsub "github.com/libp2p/go-libp2p-pubsub"

	l "github.com/D-PlatformOperatingSystem/dpos/common/log"
	"github.com/D-PlatformOperatingSystem/dpos/queue"

	prototypes "github.com/D-PlatformOperatingSystem/dpos/system/p2p/dht/protocol/types"

	"github.com/D-PlatformOperatingSystem/dpos/types"

	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	l.SetLogLevel("err")
}

func newTestpeerPubSubWithQueue(q queue.Queue) *peerPubSub {
	env := newTestEnv(q)
	protocol := &peerPubSub{}
	protocol.BaseProtocol = new(prototypes.BaseProtocol)
	prototypes.ClearEventHandler()
	protocol.InitProtocol(env)

	return protocol
}

func newTestPubProtocol(q queue.Queue) *peerPubSub {

	return newTestpeerPubSubWithQueue(q)
}

func TestPeerPubSubInitProtocol(t *testing.T) {
	q := queue.New("test")
	protocol := newTestPubProtocol(q)
	assert.NotNil(t, protocol)

}

//    topic
func testHandleSubTopicEvent(protocol *peerPubSub, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleSubTopic(msg)
}

//    topic

func testHandleRemoveTopicEvent(protocol *peerPubSub, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleRemoveTopc(msg)
}

//    topiclist

func testHandleGetTopicsEvent(protocol *peerPubSub, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleGetTopics(msg)

}

//  pubmsg
func testHandlerPubMsg(protocol *peerPubSub, msg *queue.Message) {
	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handlePubMsg(msg)
}
func testSubTopic(t *testing.T, protocol *peerPubSub) {

	msgs := make([]*queue.Message, 0)
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "blockchain",
		Topic:  "bzTest",
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "blockchain",
		Topic:  "bzTest",
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "mempool",
		Topic:  "bzTest",
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "rpc",
		Topic:  "rtopic",
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "rpc",
		Topic:  "rtopic",
	}))

	//
	for _, msg := range msgs {
		testHandleSubTopicEvent(protocol, msg)
		replyMsg, err := protocol.GetQueueClient().Wait(msg)
		assert.Nil(t, err)

		subReply, ok := replyMsg.GetData().(*types.Reply)
		if ok {
			t.Log("subReply status", subReply.IsOk)
			if subReply.IsOk {
				var reply types.SubTopicReply
				types.Decode(subReply.GetMsg(), &reply)
				assert.NotNil(t, reply)
				t.Log("reply", reply.GetMsg())
			} else {
				//
				t.Log("subfailed Reply ", string(subReply.GetMsg()))
			}

		}

	}

}

func testPushMsg(t *testing.T, protocol *peerPubSub) {
	pubTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventPubTopicMsg, &types.PublishTopicMsg{Topic: "bzTest", Msg: []byte("one two tree four")})
	testHandlerPubMsg(protocol, pubTopicMsg)
	resp, err := protocol.GetQueueClient().WaitTimeout(pubTopicMsg, time.Second*10)
	assert.Nil(t, err)
	rpy := resp.GetData().(*types.Reply)
	t.Log("bzTest isok", rpy.IsOk, "msg", string(rpy.GetMsg()))
	assert.True(t, rpy.IsOk)
	//      topic
	pubTopicMsg = protocol.QueueClient.NewMessage("p2p", types.EventPubTopicMsg, &types.PublishTopicMsg{Topic: "bzTest2", Msg: []byte("one two tree four")})
	testHandlerPubMsg(protocol, pubTopicMsg)
	resp, err = protocol.GetQueueClient().WaitTimeout(pubTopicMsg, time.Second*10)
	assert.Nil(t, err)
	rpy = resp.GetData().(*types.Reply)
	t.Log("bzTest2 isok", rpy.IsOk, "msg", string(rpy.GetMsg()))
	assert.False(t, rpy.IsOk)
	errPubTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventPubTopicMsg, &types.FetchTopicList{})
	testHandlerPubMsg(protocol, errPubTopicMsg)

}

//    topiclist
func testFetchTopics(t *testing.T, protocol *peerPubSub) []string {
	//  topiclist
	fetchTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventFetchTopics, &types.FetchTopicList{})
	testHandleGetTopicsEvent(protocol, fetchTopicMsg)
	resp, err := protocol.GetQueueClient().WaitTimeout(fetchTopicMsg, time.Second*10)
	assert.Nil(t, err)
	//  topiclist
	assert.True(t, resp.GetData().(*types.Reply).GetIsOk())
	var topiclist types.TopicList
	err = types.Decode(resp.GetData().(*types.Reply).GetMsg(), &topiclist)
	assert.Nil(t, err)

	t.Log("topiclist", topiclist.GetTopics())

	errFetchTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventFetchTopics, &types.PublishTopicMsg{})
	testHandleGetTopicsEvent(protocol, errFetchTopicMsg)
	return topiclist.GetTopics()
}

//

func testSendTopicData(t *testing.T, protocol *peerPubSub) {
	//         ,  mempool,blockchain       hello,world 1
	//protocol.msgChan <- &types.TopicData{Topic: "bzTest", From: "123435555", Data: []byte("hello,world 1")}
	msg := &pubsub.Message{Message: &pubsub_pb.Message{Data: []byte("hello,world 1")}, ReceivedFrom: "123435555"}
	protocol.subCallBack("bzTest", msg)

}

//       module topic
func testRemoveModuleTopic(t *testing.T, protocol *peerPubSub, topic, module string) {
	//  topic
	removetopic := protocol.QueueClient.NewMessage("p2p", types.EventRemoveTopic, &types.RemoveTopic{
		Topic:  topic,
		Module: module,
	})
	//    mempool    hello,world 2
	testHandleRemoveTopicEvent(protocol, removetopic) //  blockchain
	msg := &pubsub.Message{Message: &pubsub_pb.Message{Data: []byte("hello,world 2")}, ReceivedFrom: "123435555"}
	protocol.subCallBack("bzTest", msg)
	//protocol.msgChan <- &types.TopicData{Topic: "bzTest", From: "123435555", Data: []byte("hello,world 2")}

	errRemovetopic := protocol.QueueClient.NewMessage("p2p", types.EventRemoveTopic, &types.FetchTopicList{})
	testHandleRemoveTopicEvent(protocol, errRemovetopic) //  blockchain

	errRemovetopic = protocol.QueueClient.NewMessage("p2p", types.EventRemoveTopic, &types.RemoveTopic{Topic: "haha",
		Module: module})
	testHandleRemoveTopicEvent(protocol, errRemovetopic) //  blockchain

}

func testBlockRecvSubData(t *testing.T, q queue.Queue) {
	client := q.Client()
	client.Sub("blockchain")
	go func() {
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventReceiveSubData:
				if req, ok := msg.GetData().(*types.TopicData); ok {
					t.Log("blockchain Recv from", req.GetFrom(), "topic:", req.GetTopic(), "data", string(req.GetData()))
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}

			}

		}
	}()

}

func testMempoolRecvSubData(t *testing.T, q queue.Queue) {
	client := q.Client()
	client.Sub("mempool")
	go func() {
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventReceiveSubData:
				if req, ok := msg.GetData().(*types.TopicData); ok {
					t.Log("mempool Recv", req.GetFrom(), "topic:", req.GetTopic(), "data", string(req.GetData()))
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}

			}

		}
	}()

}
func TestPubSub(t *testing.T) {
	q := queue.New("test")
	testBlockRecvSubData(t, q)
	testMempoolRecvSubData(t, q)
	protocol := newTestPubProtocol(q)
	testSubTopic(t, protocol) //  topic

	topics := testFetchTopics(t, protocol) //  topic list
	assert.Equal(t, len(topics), 2)
	testSendTopicData(t, protocol) //  chan

	testPushMsg(t, protocol)                                   //
	testRemoveModuleTopic(t, protocol, "bzTest", "blockchain") //        topic
	topics = testFetchTopics(t, protocol)                      //  topic list
	assert.Equal(t, len(topics), 2)
	//--------
	testRemoveModuleTopic(t, protocol, "rtopic", "rpc") //        topic
	topics = testFetchTopics(t, protocol)
	//t.Log("after Remove rtopic", topics)
	assert.Equal(t, 1, len(topics))
	testRemoveModuleTopic(t, protocol, "bzTest", "mempool") //        topic
	topics = testFetchTopics(t, protocol)
	//t.Log("after Remove bzTest", topics)
	assert.Equal(t, 0, len(topics))

	time.Sleep(time.Second)
}
