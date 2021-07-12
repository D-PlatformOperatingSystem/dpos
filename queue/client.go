// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"unsafe"

	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//             ，            。
//         client
//         ：
// client := queue.Client()
// client.Sub("topicname")
// for msg := range client.Recv() {
//     process(msg)
// }
// process

var gid int64

// Client        ，             client
type Client interface {
	Send(msg *Message, waitReply bool) (err error) //
	SendTimeout(msg *Message, waitReply bool, timeout time.Duration) (err error)
	Wait(msg *Message) (*Message, error)                               //
	WaitTimeout(msg *Message, timeout time.Duration) (*Message, error) //
	Recv() chan *Message
	Reply(msg *Message)
	Sub(topic string) //
	Close()
	CloseQueue() (*types.Reply, error)
	NewMessage(topic string, ty int64, data interface{}) (msg *Message)
	FreeMessage(msg ...*Message) //  msg，
	GetConfig() *types.DplatformOSConfig
}

// Module be used for module interface
type Module interface {
	SetQueueClient(client Client)
	//wait for ready
	Wait()
	Close()
}

type client struct {
	q          *queue
	recv       chan *Message
	done       chan struct{}
	wg         *sync.WaitGroup
	topic      unsafe.Pointer
	isClosed   int32
	isCloseing int32
}

func newClient(q *queue) Client {
	client := &client{}
	client.q = q
	client.recv = make(chan *Message, 5)
	client.done = make(chan struct{}, 1)
	client.wg = &sync.WaitGroup{}
	return client
}

// GetConfig return the queue DplatformOSConfig
func (client *client) GetConfig() *types.DplatformOSConfig {
	types.AssertConfig(client.q)
	cfg := client.q.GetConfig()
	if cfg == nil {
		panic("DplatformOSConfig is nil")
	}
	return cfg
}

// Send     ,msg    ,waitReply
//1.     send          ，
//2.               response
func (client *client) Send(msg *Message, waitReply bool) (err error) {
	timeout := time.Duration(-1)
	err = client.SendTimeout(msg, waitReply, timeout)
	if err == ErrQueueTimeout {
		panic(err)
	}
	return err
}

// SendTimeout     ， msg    ,waitReply       ， timeout
func (client *client) SendTimeout(msg *Message, waitReply bool, timeout time.Duration) (err error) {
	if client.isClose() {
		return ErrIsQueueClosed
	}
	if !waitReply {
		//msg.chReply = nil
		return client.q.sendLowTimeout(msg, timeout)
	}
	return client.q.send(msg, timeout)
}

//
//1. SendAsyn
//2. Send

// NewMessage      topic     ty     data
func (client *client) NewMessage(topic string, ty int64, data interface{}) (msg *Message) {
	id := atomic.AddInt64(&gid, 1)
	msg = client.q.msgPool.Get().(*Message)
	msg.ID = id
	msg.Ty = ty
	msg.Data = data
	msg.Topic = topic
	return
}

// FreeMessage   msg
func (client *client) FreeMessage(msgs ...*Message) {
	for _, msg := range msgs {
		if msg == nil || msg.chReply == nil {
			continue
		}
		msg.Data = nil
		client.q.msgPool.Put(msg)
	}
}

func (client *client) Reply(msg *Message) {
	if msg.chReply != nil {
		msg.Reply(msg)
		return
	}
	if msg.callback != nil {
		client.q.callback <- msg
	}
}

// WaitTimeout      msg    timeout
func (client *client) WaitTimeout(msg *Message, timeout time.Duration) (*Message, error) {
	if msg.chReply == nil {
		return &Message{}, errors.New("empty wait channel")
	}

	var t <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		t = timer.C
	}
	select {
	case msg = <-msg.chReply:
		return msg, msg.Err()
	case <-client.done:
		return &Message{}, ErrIsQueueClosed
	case <-t:
		return &Message{}, ErrQueueTimeout
	}
}

// Wait
func (client *client) Wait(msg *Message) (*Message, error) {
	timeout := time.Duration(-1)
	msg, err := client.WaitTimeout(msg, timeout)
	if err == ErrQueueTimeout {
		panic(err)
	}
	return msg, err
}

// Recv
func (client *client) Recv() chan *Message {
	return client.recv
}

func (client *client) getTopic() string {
	return *(*string)(atomic.LoadPointer(&client.topic))
}

func (client *client) setTopic(topic string) {
	// #nosec
	atomic.StorePointer(&client.topic, unsafe.Pointer(&topic))
}

func (client *client) isClose() bool {
	return atomic.LoadInt32(&client.isClosed) == 1
}

func (client *client) isInClose() bool {
	return atomic.LoadInt32(&client.isCloseing) == 1
}

// Close   client
func (client *client) Close() {
	if atomic.LoadInt32(&client.isClosed) == 1 || atomic.LoadPointer(&client.topic) == nil {
		return
	}
	topic := client.getTopic()
	client.q.closeTopic(topic)
	close(client.done)
	atomic.StoreInt32(&client.isCloseing, 1)
	client.wg.Wait()
	atomic.StoreInt32(&client.isClosed, 1)
	close(client.Recv())
	for msg := range client.Recv() {
		msg.Reply(client.NewMessage(msg.Topic, msg.Ty, types.ErrChannelClosed))
	}
}

// CloseQueue
func (client *client) CloseQueue() (*types.Reply, error) {
	//	client.q.Close()
	if client.q.isClosed() {
		return &types.Reply{IsOk: true}, nil
	}
	qlog.Debug("queue", "msg", "closing dplatformos")
	client.q.interrupt <- struct{}{}
	//	close(client.q.interupt)
	return &types.Reply{IsOk: true}, nil
}

func (client *client) isEnd(data *Message, ok bool) bool {
	if !ok {
		return true
	}
	if data.Data == nil && data.ID == 0 && data.Ty == 0 {
		return true
	}
	if atomic.LoadInt32(&client.isClosed) == 1 {
		return true
	}
	return false
}

// Sub
func (client *client) Sub(topic string) {
	//
	if client.isInClose() || client.isClose() {
		return
	}
	client.wg.Add(1)
	client.setTopic(topic)
	sub := client.q.chanSub(topic)
	go func() {
		defer func() {
			client.wg.Done()
		}()
		for {
			select {
			case data, ok := <-sub.high:
				if client.isEnd(data, ok) {
					qlog.Info("unsub1", "topic", topic)
					return
				}
				client.Recv() <- data
			default:
				select {
				case data, ok := <-sub.high:
					if client.isEnd(data, ok) {
						qlog.Info("unsub2", "topic", topic)
						return
					}
					client.Recv() <- data
				case data, ok := <-sub.low:
					if client.isEnd(data, ok) {
						qlog.Info("unsub3", "topic", topic)
						return
					}
					client.Recv() <- data
				case <-client.done:
					qlog.Error("unsub4", "topic", topic)
					return
				}
			}
		}
	}()
}
