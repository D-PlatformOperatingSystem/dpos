// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//const
const (
	ENDN = '\n'
	ENDR = '\r'

	OK       = "ok"
	NotFound = "not_found"

	ReadTimeOut  = 3
	WriteTimeOut = 3

	ReadBufSize = 8 * 1024

	IteratorPageSize = 10240

	PooledSize = 3
)

//SDBClient ...
type SDBClient struct {
	sock     *net.TCPConn
	timeZero time.Time
	mu       sync.Mutex
}

//SDBPool SDB
type SDBPool struct {
	clients []*SDBClient
	round   *RoundInt
}

//RoundInt ...
type RoundInt struct {
	round int
	index int
}

func (val *RoundInt) incr() int {
	val.index++
	if val.index < val.round {
		return val.index
	}
	val.index = 0
	return val.index
}

func (pool *SDBPool) get() *SDBClient {
	return pool.clients[pool.round.incr()]
}
func (pool *SDBPool) close() {
	for _, v := range pool.clients {
		err := v.Close()
		dlog.Error("ssdb close ", "error", err)
	}
}

//NewSDBPool new
func NewSDBPool(nodes []*SsdbNode) (pool *SDBPool, err error) {
	dbpool := &SDBPool{}
	for i := 0; i < PooledSize; i++ {
		for _, v := range nodes {
			db, err := Connect(v.ip, v.port)
			if err != nil {
				dlog.Error("connect to ssdb error!", "ssdb", v)
				return dbpool, types.ErrDataBaseDamage
			}
			dbpool.clients = append(dbpool.clients, db)
		}
	}
	dbpool.round = &RoundInt{round: PooledSize * len(nodes)}
	return dbpool, nil
}

//Connect
func Connect(ip string, port int) (*SDBClient, error) {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, err
	}
	sock, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	var c SDBClient
	c.sock = sock
	c.timeZero = time.Time{}

	return &c, nil
}

//Get      key
//  key
//        Value,
//            ，       nil
func (c *SDBClient) Get(key string) (*Value, error) {
	resp, err := c.Do("get", key)
	if err != nil {
		return nil, newErrorf(err, "Get %s error", key)
	}

	if len(resp) == 0 {
		return nil, newError("ssdb response error")
	}

	if len(resp) == 2 && resp[0] == OK {
		return toValue(resp[1]), nil
	}

	return nil, makeError(resp, key)
}

//Set      key
//  key
//  val     value  ,val        ，          ，         Encoding
//  ttl   ，       ，
//     err，     ，       nil
func (c *SDBClient) Set(key string, val []byte) (err error) {
	var resp []string
	resp, err = c.Do("set", key, val)
	if err != nil {
		return newErrorf(err, "Set %s error", key)
	}
	if len(resp) > 0 && resp[0] == OK {
		return nil
	}
	return makeError(resp, key)
}

//Del      key
//  key      key
//     err，     ，       nil
func (c *SDBClient) Del(key string) error {
	resp, err := c.Do("del", key)
	if err != nil {
		return newErrorf(err, "Del %s error", key)
	}

	//response looks like s: [ok 1]
	if len(resp) > 0 && resp[0] == OK {
		return nil
	}
	return makeError(resp, key)
}

//MultiSet        key-value.
//     key-value
//     err，     ，       nil
func (c *SDBClient) MultiSet(kvs map[string][]byte) (err error) {

	args := []interface{}{"multi_set"}

	for k, v := range kvs {
		args = append(args, k)
		args = append(args, v)
	}
	resp, err := c.Do(args...)

	if err != nil {
		return newErrorf(err, "MultiSet %s error", kvs)
	}

	if len(resp) > 0 && resp[0] == OK {
		return nil
	}
	return makeError(resp, kvs)
}

//MultiDel        key         .
//  key，     key，
//     err，     ，       nil
func (c *SDBClient) MultiDel(key ...string) (err error) {
	if len(key) == 0 {
		return nil
	}
	args := []interface{}{"multi_del"}
	for _, v := range key {
		args = append(args, v)
	}
	resp, err := c.Do(args...)
	if err != nil {
		return newErrorf(err, "MultiDel %s error", key)
	}

	if len(resp) > 0 && resp[0] == OK {
		return nil
	}
	return makeError(resp, key)
}

//MultiGet        key         .
//  key，     key，
//     err，     ，       nil
func (c *SDBClient) MultiGet(key ...string) (vals []*Value, err error) {
	if len(key) == 0 {
		return nil, nil
	}
	data := []interface{}{"multi_get"}
	for _, k := range key {
		data = append(data, k)
	}
	resp, err := c.Do(data...)

	if err != nil {
		return nil, newErrorf(err, "MultiGet %s error", key)
	}

	size := len(resp)
	if size > 0 && resp[0] == OK {
		for i := 1; i < size && i+1 < size; i += 2 {
			vals = append(vals, toValue(resp[i+1]))
		}
		return vals, nil
	}
	return nil, makeError(resp, key)
}

//Keys        (key_start, key_end]   key   .("", ""]       .
//  keyStart int       key(   ),        -inf.
//  keyEnd int       key(  ),        +inf.
//  limit int           .
//          key    .
//     err，     ，       nil
func (c *SDBClient) Keys(keyStart, keyEnd string, limit int64) ([]string, error) {

	resp, err := c.Do("keys", keyStart, keyEnd, limit)

	if err != nil {
		return nil, newErrorf(err, "Keys [%s,%s] %d error", keyStart, keyEnd, limit)
	}
	if len(resp) > 0 && resp[0] == OK {
		return resp[1:], nil
	}
	return nil, makeError(resp, keyStart, keyEnd, limit)
}

//Rkeys        (key_start, key_end]   key   .("", ""]       .
//  keyStart int       key(   ),        -inf.
//  keyEnd int       key(  ),        +inf.
//  limit int           .
//          key    .
//     err，     ，       nil
func (c *SDBClient) Rkeys(keyStart, keyEnd string, limit int64) ([]string, error) {

	resp, err := c.Do("rkeys", keyStart, keyEnd, limit)

	if err != nil {
		return nil, newErrorf(err, "Rkeys [%s,%s] %d error", keyStart, keyEnd, limit)
	}
	if len(resp) > 0 && resp[0] == OK {
		return resp[1:], nil
	}
	return nil, makeError(resp, keyStart, keyEnd, limit)
}

//Do do
func (c *SDBClient) Do(args ...interface{}) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//dlog.Warn("begin to send", "value", fmt.Sprintf("%v", args))
	err := c.send(args)
	if err != nil {
		//dlog.Error("send error", "value", fmt.Sprintf("%v", err))
		return nil, err
	}
	resp, err := c.recv()
	//dlog.Warn("begin to return", "value", fmt.Sprintf("%v", resp))
	return resp, err
}

func (c *SDBClient) send(args []interface{}) error {
	var packetBuf bytes.Buffer
	var err error
	for _, arg := range args {
		switch arg := arg.(type) {
		case string:
			if _, err = packetBuf.Write(strconv.AppendInt(nil, int64(len(arg)), 10)); err != nil {
				return err
			}
			if err = packetBuf.WriteByte(ENDN); err != nil {
				return err
			}
			if _, err = packetBuf.WriteString(arg); err != nil {
				return err
			}
		case []string:
			for _, a := range arg {
				if _, err = packetBuf.Write(strconv.AppendInt(nil, int64(len(a)), 10)); err != nil {
					return err
				}
				if err = packetBuf.WriteByte(ENDN); err != nil {
					return err
				}
				if _, err = packetBuf.WriteString(a); err != nil {
					return err
				}
				if err = packetBuf.WriteByte(ENDN); err != nil {
					return err
				}
			}
			continue
		case []byte:
			if _, err = packetBuf.Write(strconv.AppendInt(nil, int64(len(arg)), 10)); err != nil {
				return err
			}
			if err = packetBuf.WriteByte(ENDN); err != nil {
				return err
			}
			if _, err = packetBuf.Write(arg); err != nil {
				return err
			}
		case int64:
			bs := strconv.AppendInt(nil, arg, 10)
			if _, err = packetBuf.Write(strconv.AppendInt(nil, int64(len(bs)), 10)); err != nil {
				return err
			}
			if err = packetBuf.WriteByte(ENDN); err != nil {
				return err
			}
			if _, err = packetBuf.Write(bs); err != nil {
				return err
			}
		case nil:
			if err = packetBuf.WriteByte(0); err != nil {
				return err
			}
			if err = packetBuf.WriteByte(ENDN); err != nil {
				return err
			}
			if _, err = packetBuf.WriteString(""); err != nil {
				return err
			}
		default:
			return fmt.Errorf("bad arguments type")
		}
		if err = packetBuf.WriteByte(ENDN); err != nil {
			return err
		}
	}
	if err = packetBuf.WriteByte(ENDN); err != nil {
		return err
	}
	if err = c.sock.SetWriteDeadline(time.Now().Add(time.Second * WriteTimeOut)); err != nil {
		return err
	}
	for _, err = packetBuf.WriteTo(c.sock); packetBuf.Len() > 0; {
		if err != nil {
			packetBuf.Reset()
			return newErrorf(err, "client socket write error")
		}
	}
	//
	if err := c.sock.SetWriteDeadline(c.timeZero); err != nil {
		return err
	}
	packetBuf.Reset()
	return nil
}
func (c *SDBClient) recv() (resp []string, err error) {
	packetBuf := []byte{}
	//        ，
	if err = c.sock.SetReadDeadline(time.Now().Add(time.Second * ReadTimeOut)); err != nil {
		return nil, err
	}
	//     ，    ，    ，    ，    ，
	readBuf := make([]byte, ReadBufSize)
	for {
		bufSize, err := c.sock.Read(readBuf)
		if err != nil {
			return nil, newErrorf(err, "client socket read error")
		}
		if bufSize < 1 {
			continue
		}
		packetBuf = append(packetBuf, readBuf[:bufSize]...)

		for {
			rsp, n := c.parse(packetBuf)
			if n == -1 {
				break
			} else if n == -2 {
				return nil, newErrorf(err, "parse error")
			} else {
				resp = append(resp, rsp)
				packetBuf = packetBuf[n+1:]
			}
		}
	}
}

func (c *SDBClient) parse(buf []byte) (resp string, size int) {
	n := bytes.IndexByte(buf, ENDN)
	size = -1
	if n != -1 {
		if n == 0 || n == 1 && buf[0] == ENDR { //  ，
			size = -2
			return
		}
		//     ，
		blockSize := ToNum(buf[:n])
		bufSize := len(buf)

		if n+blockSize < bufSize {
			resp = string(buf[n+1 : blockSize+n+1])
			for i := blockSize + n + 1; i < bufSize; i++ {
				if buf[i] == ENDN {
					size = i
					return
				}
			}
		}
	}
	return
}

// Close The Client Connection
func (c *SDBClient) Close() error {
	return c.sock.Close()
}

//         ，
func makeError(resp []string, errKey ...interface{}) error {
	if len(resp) < 1 {
		return newError("ssdb response error")
	}
	//           ，            exists
	if resp[0] == NotFound {
		return ErrNotFoundInDb
	}
	if len(errKey) > 0 {
		return newError("access ssdb error, code is %v, parameter is %v", resp, errKey)
	}
	return newError("access ssdb error, code is %v", resp)

}

//Value    ，      string
type Value struct {
	val []byte
}

//   string
func (v *Value) String() string {
	return string(v.val)
}

//Bytes    []byte
func (v *Value) Bytes() []byte {
	return v.val
}

func toValue(val interface{}) *Value {
	if val == nil {
		return nil
	}
	if v, ok := val.(string); ok {
		return &Value{val: []byte(v)}
	} else if v, ok := val.([]byte); ok {
		return &Value{val: v}
	} else {
		dlog.Error("unsupported value type", "value", val)
		return nil
	}
}

var (
	byt              = []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	maxByteSize byte = 57
	minByteSize byte = 48
)

//ToNum []byte -> int
func ToNum(bs []byte) int {
	re := 0
	for _, v := range bs {
		if v > maxByteSize || v < minByteSize {
			return re
		}
		re = re*10 + byt[v]
	}
	return re
}

var (
	//FormatString
	FormatString = "%v\nthe trace error is\n%s"
)

//
func newError(format string, p ...interface{}) error {
	return fmt.Errorf(format, p...)
}

//
//
func newErrorf(err error, format string, p ...interface{}) error {
	return fmt.Errorf(FormatString, fmt.Sprintf(format, p...), err)
}
