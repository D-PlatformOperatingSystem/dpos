// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package solo

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof" //
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"google.golang.org/grpc"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/rpc/grpcclient"
	"github.com/decred/base58"
	b58 "github.com/mr-tron/base58"

	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
	"github.com/D-PlatformOperatingSystem/dpos/util/testnode"
	"github.com/stretchr/testify/assert"

	//      store,     plugin
	_ "github.com/D-PlatformOperatingSystem/dpos/system/dapp/init"
	_ "github.com/D-PlatformOperatingSystem/dpos/system/mempool/init"
	_ "github.com/D-PlatformOperatingSystem/dpos/system/store/init"
)

//   ： go test -cover
func TestSolo(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer mockDOM.Close()
	cfg := mockDOM.GetClient().GetConfig()
	txs := util.GenNoneTxs(cfg, mockDOM.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		mockDOM.GetAPI().SendTx(txs[i])
	}
	mockDOM.WaitHeight(1)
	txs = util.GenNoneTxs(cfg, mockDOM.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		mockDOM.GetAPI().SendTx(txs[i])
	}
	mockDOM.WaitHeight(2)
}

func BenchmarkSolo(b *testing.B) {
	cfg := testnode.GetDefaultConfig()
	subcfg := cfg.GetSubConfig()
	solocfg, err := types.ModifySubConfig(subcfg.Consensus["solo"], "waitTxMs", 1000)
	assert.Nil(b, err)
	subcfg.Consensus["solo"] = solocfg
	mockDOM := testnode.NewWithConfig(cfg, nil)
	defer mockDOM.Close()
	txs := util.GenCoinsTxs(cfg, mockDOM.GetGenesisKey(), int64(b.N))
	var last []byte
	var mu sync.Mutex
	b.ResetTimer()
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			for n := index; n < b.N; n += 10 {
				reply, err := mockDOM.GetAPI().SendTx(txs[n])
				if err != nil {
					assert.Nil(b, err)
				}
				mu.Lock()
				last = reply.GetMsg()
				mu.Unlock()
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	mockDOM.WaitTx(last)
}

var (
	tlog = log15.New("module", "test solo")
)

//mempool     10000tx/s
func BenchmarkSendTx(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	subcfg := cfg.GetSubConfig()
	cfg.GetModuleConfig().Exec.DisableAddrIndex = true
	solocfg, err := types.ModifySubConfig(subcfg.Consensus["solo"], "waitTxMs", 100)
	assert.Nil(b, err)
	solocfg, err = types.ModifySubConfig(solocfg, "benchMode", true)
	assert.Nil(b, err)
	subcfg.Consensus["solo"] = solocfg
	cfg.GetModuleConfig().RPC.JrpcBindAddr = "localhost:28803"
	cfg.GetModuleConfig().RPC.GrpcBindAddr = "localhost:28804"
	mockDOM := testnode.NewWithRPC(cfg, nil)
	log.SetLogLevel("error")
	defer mockDOM.Close()
	priv := mockDOM.GetGenesisKey()
	b.ResetTimer()
	b.Run("SendTx-Internal", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tx := util.CreateNoneTxWithTxHeight(cfg, priv, 0)
				mockDOM.GetAPI().SendTx(tx)
			}
		})
	})

	b.Run("SendTx-GRPC", func(b *testing.B) {
		gcli, _ := grpcclient.NewMainChainClient(cfg, "localhost:28804")
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tx := util.CreateNoneTxWithTxHeight(cfg, priv, 0)
				_, err := gcli.SendTransaction(context.Background(), tx)
				if err != nil {
					tlog.Error("sendtx grpc", "err", err)
				}
			}
		})
	})

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100
	defer http.DefaultClient.CloseIdleConnections()
	b.Run("SendTx-JSONRPC", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tx := util.CreateNoneTxWithTxHeight(cfg, priv, 0)
				poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"DplatformOS.SendTransaction","params":[{"data":"%v"}]}`,
					common.ToHex(types.Encode(tx)))

				resp, _ := http.Post("http://localhost:28803", "application/json", bytes.NewBufferString(poststr))
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
			}
		})
	})
}

//   10000            3500tx/s
func BenchmarkSoloNewBlock(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	cfg.GetModuleConfig().Exec.DisableAddrIndex = true
	cfg.GetModuleConfig().Mempool.DisableExecCheck = true
	cfg.GetModuleConfig().RPC.GrpcBindAddr = "localhost:28804"
	subcfg := cfg.GetSubConfig()
	solocfg, err := types.ModifySubConfig(subcfg.Consensus["solo"], "waitTxMs", 100)
	assert.Nil(b, err)
	solocfg, err = types.ModifySubConfig(solocfg, "benchMode", true)
	assert.Nil(b, err)
	subcfg.Consensus["solo"] = solocfg
	mockDOM := testnode.NewWithRPC(cfg, nil)
	defer mockDOM.Close()
	start := make(chan struct{})
	//pub := mockDOM.GetGenesisKey().PubKey().Bytes()
	var height int64
	for i := 0; i < 10; i++ {
		addr, _ := util.Genaddress()
		go func(addr string) {
			start <- struct{}{}
			conn, err := grpc.Dial("localhost:28804", grpc.WithInsecure())
			if err != nil {
				panic(err.Error())
			}
			defer conn.Close()
			gcli := types.NewDplatformOSClient(conn)
			for {
				tx := util.CreateNoneTxWithTxHeight(cfg, mockDOM.GetGenesisKey(), atomic.LoadInt64(&height))
				//
				//tx := util.CreateNoneTxWithTxHeight(cfg, nil, 0)
				//tx.Signature = &types.Signature{
				//	Ty: types.SECP256K1,
				//	Pubkey:pub,
				//}
				_, err := gcli.SendTransaction(context.Background(), tx)
				//_, err := mockDOM.GetAPI().SendTx(tx)
				if err != nil {
					if strings.Contains(err.Error(), "ErrChannelClosed") {
						return
					}
					tlog.Error("sendtx", "err", err.Error())
					time.Sleep(2 * time.Second)
					continue
				}
			}
		}(addr)
		<-start
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := mockDOM.WaitHeight(int64(i + 1))
		for err != nil {
			b.Log("SoloNewBlock", "waitblkerr", err)
			time.Sleep(time.Second / 10)
			err = mockDOM.WaitHeight(int64(i + 1))
		}
		atomic.AddInt64(&height, 1)
	}
}

//             4k
func BenchmarkTxSign(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	txBenchNum := 10000
	_, priv := util.Genaddress()
	txs := util.GenCoinsTxs(cfg, priv, int64(txBenchNum))

	start := make(chan struct{})
	wait := make(chan struct{})
	result := make(chan interface{}, txBenchNum)
	//
	for i := 0; i < 8; i++ {
		go func() {
			wait <- struct{}{}
			index := 0
			<-start
			for {
				//txs[index%txBenchNum].Sign(types.SECP256K1, priv)
				result <- txs[index%txBenchNum].CheckSign()
				index++
			}
		}()
		<-wait
	}
	b.ResetTimer()
	close(start)
	for i := 0; i < b.N; i++ {
		<-result
	}
}

//        , 80w /s
func BenchmarkMsgQueue(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	q := queue.New("channel")
	q.SetConfig(cfg)
	topicNum := 10
	topics := make([]string, topicNum)
	start := make(chan struct{})
	for i := 0; i < topicNum; i++ {
		topics[i] = fmt.Sprintf("bench-%d", i)
		go func(topic string) {
			start <- struct{}{}
			client := q.Client()
			client.Sub(topic)
			for range client.Recv() {
			}
		}(topics[i])
		<-start
	}
	_, priv := util.Genaddress()
	txs := util.GenCoinsTxs(cfg, priv, 1)
	sendCli := q.Client()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			msg := sendCli.NewMessage(topics[i%topicNum], int64(i), txs[0])
			err := sendCli.Send(msg, false)
			assert.Nil(b, err)
			i++
		}
	})
}

func BenchmarkTxHash(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	_, priv := util.Genaddress()
	txs := util.GenCoinsTxs(cfg, priv, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txs[0].Hash()
	}
}

func BenchmarkEncode(b *testing.B) {

	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	cfg := testnode.GetDefaultConfig()
	_, priv := util.Genaddress()
	txs := util.GenCoinsTxs(cfg, priv, 10000)

	block := &types.Block{}
	block.Txs = txs
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		types.Encode(block)
	}
}

func BenchmarkBase58(b *testing.B) {

	if testing.Short() {
		b.Skip("skipping in short mode.")
	}
	addr := "12oupcayRT7LvaC4qW4avxsTE7U41cKSio"
	buf := base58.Decode(addr)
	b.ResetTimer()
	b.Run("decred-base58", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			base58.Encode(buf)
		}
	})

	b.Run("mr-tron-base58", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b58.Encode(buf)
		}
	})
}
