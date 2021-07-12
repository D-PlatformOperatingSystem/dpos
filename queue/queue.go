// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package queue dplatformos
package queue

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/types"

	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
)

//    ：
//
//  ：topic

//1.     ：
//1.1   topic        （       ）      ，         .
//1.2               channel
var qlog = log.New("module", "queue")

const (
	defaultChanBuffer    = 64
	defaultLowChanBuffer = 40960
)

//
var (
	ErrIsQueueClosed    = errors.New("ErrIsQueueClosed")
	ErrQueueTimeout     = errors.New("ErrQueueTimeout")
	ErrQueueChannelFull = errors.New("ErrQueueChannelFull")
)

// DisableLog disable log
func DisableLog() {
	qlog.SetHandler(log.DiscardHandler())
}

type chanSub struct {
	high    chan *Message
	low     chan *Message
	isClose int32
}

// Queue only one obj in project
// Queue only generate Client and start、Close operate,
// if you send massage or receive massage on Queue, please use Client.
type Queue interface {
	Close()
	Start()
	Client() Client
	Name() string
	SetConfig(cfg *types.DplatformOSConfig)
	GetConfig() *types.DplatformOSConfig
}

type queue struct {
	chanSubs  map[string]*chanSub
	mu        sync.Mutex
	done      chan struct{}
	interrupt chan struct{}
	callback  chan *Message
	isClose   int32
	name      string
	cfg       *types.DplatformOSConfig
	msgPool   *sync.Pool
}

// New new queue struct
func New(name string) Queue {
	q := &queue{
		chanSubs:  make(map[string]*chanSub),
		name:      name,
		done:      make(chan struct{}, 1),
		interrupt: make(chan struct{}, 1),
		callback:  make(chan *Message, 1024),
	}
	q.msgPool = &sync.Pool{
		New: func() interface{} {
			return &Message{
				chReply: make(chan *Message, 1),
			}
		},
	}
	go func() {
		for {
			select {
			case <-q.done:
				qlog.Info("closing dplatformos callback")
				return
			case msg := <-q.callback:
				if msg.callback != nil {
					msg.callback(msg)
				}
			}
		}
	}()
	return q
}

// GetConfig return the queue DplatformOSConfig
func (q *queue) GetConfig() *types.DplatformOSConfig {
	return q.cfg
}

// Name return the queue name
func (q *queue) SetConfig(cfg *types.DplatformOSConfig) {
	if cfg == nil {
		panic("set config is nil")
	}
	if q.cfg != nil {
		panic("do not reset queue config")
	}
	q.cfg = cfg
}

// Name return the queue name
func (q *queue) Name() string {
	return q.name
}

// Start
func (q *queue) Start() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// Block until a signal is received.
	select {
	case <-q.done:
		qlog.Info("closing dplatformos done")
		//atomic.StoreInt32(&q.isClose, 1)
		break
	case <-q.interrupt:
		qlog.Info("closing dplatformos")
		//atomic.StoreInt32(&q.isClose, 1)
		break
	case s := <-c:
		qlog.Info("Got signal:", s)
		//atomic.StoreInt32(&q.isClose, 1)
		break
	}
}

func (q *queue) isClosed() bool {
	return atomic.LoadInt32(&q.isClose) == 1
}

// Close
func (q *queue) Close() {
	if q.isClosed() {
		return
	}
	q.mu.Lock()
	for topic, ch := range q.chanSubs {
		if ch.isClose == 0 {
			ch.high <- &Message{}
			ch.low <- &Message{}
			q.chanSubs[topic] = &chanSub{isClose: 1}
		}
	}
	q.mu.Unlock()
	q.done <- struct{}{}
	close(q.done)
	atomic.StoreInt32(&q.isClose, 1)
	qlog.Info("queue module closed")
}

func (q *queue) chanSub(topic string) *chanSub {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.chanSubs[topic]
	if !ok {
		q.chanSubs[topic] = &chanSub{
			high:    make(chan *Message, defaultChanBuffer),
			low:     make(chan *Message, defaultLowChanBuffer),
			isClose: 0,
		}
	}
	return q.chanSubs[topic]
}

func (q *queue) closeTopic(topic string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	sub, ok := q.chanSubs[topic]
	if !ok {
		return
	}
	if sub.isClose == 0 {
		sub.high <- &Message{}
		sub.low <- &Message{}
	}
	q.chanSubs[topic] = &chanSub{isClose: 1}
}

func (q *queue) send(msg *Message, timeout time.Duration) (err error) {
	if q.isClosed() {
		return types.ErrChannelClosed
	}
	sub := q.chanSub(msg.Topic)
	if sub.isClose == 1 {
		return types.ErrChannelClosed
	}
	if timeout == -1 {
		sub.high <- msg
		return nil
	}
	defer func() {
		res := recover()
		if res != nil {
			err = res.(error)
		}
	}()
	if timeout == 0 {
		select {
		case sub.high <- msg:
			return nil
		default:
			qlog.Error("send chainfull", "msg", msg, "topic", msg.Topic, "sub", sub)
			return ErrQueueChannelFull
		}
	}
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case sub.high <- msg:
	case <-t.C:
		qlog.Error("send timeout", "msg", msg, "topic", msg.Topic, "sub", sub)
		return ErrQueueTimeout
	}
	return nil
}

func (q *queue) sendAsyn(msg *Message) error {
	if q.isClosed() {
		return types.ErrChannelClosed
	}
	sub := q.chanSub(msg.Topic)
	if sub.isClose == 1 {
		return types.ErrChannelClosed
	}
	select {
	case sub.low <- msg:
		return nil
	default:
		qlog.Error("send asyn err", "msg", msg, "err", ErrQueueChannelFull)
		return ErrQueueChannelFull
	}
}

func (q *queue) sendLowTimeout(msg *Message, timeout time.Duration) error {
	if q.isClosed() {
		return types.ErrChannelClosed
	}
	sub := q.chanSub(msg.Topic)
	if sub.isClose == 1 {
		return types.ErrChannelClosed
	}
	if timeout == -1 {
		sub.low <- msg
		return nil
	}
	if timeout == 0 {
		return q.sendAsyn(msg)
	}
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case sub.low <- msg:
		return nil
	case <-t.C:
		qlog.Error("send asyn timeout", "msg", msg)
		return ErrQueueTimeout
	}
}

// Client new client
func (q *queue) Client() Client {
	return newClient(q)
}

// Message message struct
type Message struct {
	Topic    string
	Ty       int64
	ID       int64
	Data     interface{}
	chReply  chan *Message
	callback func(msg *Message)
}

// NewMessage new message
func NewMessage(id int64, topic string, ty int64, data interface{}) (msg *Message) {
	msg = &Message{}
	msg.ID = id
	msg.Ty = ty
	msg.Data = data
	msg.Topic = topic
	msg.chReply = make(chan *Message, 1)
	return msg
}

// NewMessageCallback reply block
func NewMessageCallback(id int64, topic string, ty int64, data interface{}, callback func(msg *Message)) (msg *Message) {
	msg = &Message{}
	msg.ID = id
	msg.Ty = ty
	msg.Data = data
	msg.Topic = topic
	msg.callback = callback
	return msg
}

// GetData get message data
func (msg *Message) GetData() interface{} {
	if _, ok := msg.Data.(error); ok {
		return nil
	}
	return msg.Data
}

// Err if err return error msg, or return nil
func (msg *Message) Err() error {
	if err, ok := msg.Data.(error); ok {
		return err
	}
	return nil
}

// Reply reply message to reply chan
func (msg *Message) Reply(replyMsg *Message) {
	if msg.chReply == nil {
		qlog.Debug("reply a empty chreply", "msg", msg)
		return
	}
	msg.chReply <- replyMsg
}

// String print the message information
func (msg *Message) String() string {
	return fmt.Sprintf("{topic:%s, Ty:%s, Id:%d, Err:%v, Ch:%v}", msg.Topic,
		types.GetEventName(int(msg.Ty)), msg.ID, msg.Err(), msg.chReply != nil)
}

// ReplyErr reply error
func (msg *Message) ReplyErr(title string, err error) {
	var reply types.Reply
	if err != nil {
		qlog.Error(title, "reply.err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	} else {
		qlog.Debug(title, "success", "ok")
		reply.IsOk = true
	}
	id := atomic.AddInt64(&gid, 1)
	msg.Reply(NewMessage(id, "", types.EventReply, &reply))
}
