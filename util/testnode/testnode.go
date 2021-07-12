// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package testnode            ，           。

package testnode

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/p2p"

	"github.com/D-PlatformOperatingSystem/dpos/account"
	"github.com/D-PlatformOperatingSystem/dpos/blockchain"
	"github.com/D-PlatformOperatingSystem/dpos/client"
	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	"github.com/D-PlatformOperatingSystem/dpos/common/limits"
	"github.com/D-PlatformOperatingSystem/dpos/common/log"
	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/consensus"
	"github.com/D-PlatformOperatingSystem/dpos/executor"
	"github.com/D-PlatformOperatingSystem/dpos/mempool"
	"github.com/D-PlatformOperatingSystem/dpos/pluginmgr"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/rpc"
	"github.com/D-PlatformOperatingSystem/dpos/rpc/jsonclient"
	rpctypes "github.com/D-PlatformOperatingSystem/dpos/rpc/types"
	"github.com/D-PlatformOperatingSystem/dpos/store"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
	"github.com/D-PlatformOperatingSystem/dpos/wallet"
)

func init() {
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	log.SetLogLevel("info")
}

//      dplatformos
var lognode = log15.New("module", "lognode")

//DplatformOSMock :
type DplatformOSMock struct {
	random   *rand.Rand
	q        queue.Queue
	client   queue.Client
	api      client.QueueProtocolAPI
	chain    *blockchain.BlockChain
	mem      queue.Module
	cs       queue.Module
	exec     *executor.Executor
	wallet   queue.Module
	network  queue.Module
	store    queue.Module
	rpc      *rpc.RPC
	cfg      *types.Config
	sub      *types.ConfigSubModule
	datadir  string
	lastsend []byte
	mu       sync.Mutex
}

//GetDefaultConfig :
func GetDefaultConfig() *types.DplatformOSConfig {
	return types.NewDplatformOSConfig(types.GetDefaultCfgstring())
}

//NewWithConfig :
func NewWithConfig(cfg *types.DplatformOSConfig, mockapi client.QueueProtocolAPI) *DplatformOSMock {
	return newWithConfig(cfg, mockapi)
}

// NewWithRPC           rpc
func NewWithRPC(cfg *types.DplatformOSConfig, mockapi client.QueueProtocolAPI) *DplatformOSMock {
	mock := newWithConfigNoLock(cfg, mockapi)
	mock.rpc.Close()
	server := rpc.New(cfg)
	server.SetAPI(mock.api)
	server.SetQueueClient(mock.q.Client())
	mock.rpc = server
	return mock
}

func newWithConfig(cfg *types.DplatformOSConfig, mockapi client.QueueProtocolAPI) *DplatformOSMock {
	return newWithConfigNoLock(cfg, mockapi)
}

func newWithConfigNoLock(cfg *types.DplatformOSConfig, mockapi client.QueueProtocolAPI) *DplatformOSMock {
	mfg := cfg.GetModuleConfig()
	sub := cfg.GetSubConfig()
	q := queue.New("channel")
	q.SetConfig(cfg)
	types.Debug = false
	datadir := util.ResetDatadir(mfg, "$TEMP/")
	mock := &DplatformOSMock{cfg: mfg, sub: sub, q: q, datadir: datadir}
	mock.random = rand.New(rand.NewSource(types.Now().UnixNano()))

	mock.exec = executor.New(cfg)
	mock.exec.SetQueueClient(q.Client())
	lognode.Info("init exec")

	mock.store = store.New(cfg)
	mock.store.SetQueueClient(q.Client())
	lognode.Info("init store")

	mock.chain = blockchain.New(cfg)
	mock.chain.SetQueueClient(q.Client())
	lognode.Info("init blockchain")

	mock.cs = consensus.New(cfg)
	mock.cs.SetQueueClient(q.Client())
	lognode.Info("init consensus " + mfg.Consensus.Name)

	mock.mem = mempool.New(cfg)
	mock.mem.SetQueueClient(q.Client())
	mock.mem.Wait()
	lognode.Info("init mempool")
	if mfg.P2P.Enable {
		mock.network = p2p.NewP2PMgr(cfg)
		mock.network.SetQueueClient(q.Client())
	} else {
		mock.network = &mockP2P{}
		mock.network.SetQueueClient(q.Client())
	}
	lognode.Info("init P2P")
	cli := q.Client()
	w := wallet.New(cfg)
	mock.client = q.Client()
	mock.wallet = w
	mock.wallet.SetQueueClient(cli)
	lognode.Info("init wallet")
	if mockapi == nil {
		var err error
		mockapi, err = client.New(q.Client(), nil)
		if err != nil {
			return nil
		}
		newWalletRealize(mockapi)
	}
	mock.api = mockapi
	server := rpc.New(cfg)
	server.SetAPI(mock.api)
	server.SetQueueClientNoListen(q.Client())
	mock.rpc = server
	return mock
}

//New :
func New(cfgpath string, mockapi client.QueueProtocolAPI) *DplatformOSMock {
	var cfg *types.DplatformOSConfig
	if cfgpath == "" || cfgpath == "--notset--" || cfgpath == "--free--" {
		cfg = types.NewDplatformOSConfig(types.GetDefaultCfgstring())
		if cfgpath == "--free--" {
			setFee(cfg.GetModuleConfig(), 0)
			cfg.SetMinFee(0)
		}
	} else {
		cfg = types.NewDplatformOSConfig(types.ReadFile(cfgpath))
	}
	return newWithConfig(cfg, mockapi)
}

//Listen :
func (mock *DplatformOSMock) Listen() {
	pluginmgr.AddRPC(mock.rpc)
	var portgrpc, portjsonrpc int
	for {
		portgrpc, portjsonrpc = mock.rpc.Listen()
		if portgrpc != 0 && portjsonrpc != 0 {
			break
		}
	}

	if strings.HasSuffix(mock.cfg.RPC.JrpcBindAddr, ":0") {
		l := len(mock.cfg.RPC.JrpcBindAddr)
		mock.cfg.RPC.JrpcBindAddr = mock.cfg.RPC.JrpcBindAddr[0:l-2] + ":" + fmt.Sprint(portjsonrpc)
	}
	if strings.HasSuffix(mock.cfg.RPC.GrpcBindAddr, ":0") {
		l := len(mock.cfg.RPC.GrpcBindAddr)
		mock.cfg.RPC.GrpcBindAddr = mock.cfg.RPC.GrpcBindAddr[0:l-2] + ":" + fmt.Sprint(portgrpc)
	}
}

//ModifyParaClient modify para config
func ModifyParaClient(cfg *types.DplatformOSConfig, gaddr string) {
	sub := cfg.GetSubConfig()
	if sub.Consensus["para"] != nil {
		data, err := types.ModifySubConfig(sub.Consensus["para"], "ParaRemoteGrpcClient", gaddr)
		if err != nil {
			panic(err)
		}
		sub.Consensus["para"] = data
		cfg.S("config.consensus.sub.para.ParaRemoteGrpcClient", gaddr)
	}
}

//GetBlockChain :
func (mock *DplatformOSMock) GetBlockChain() *blockchain.BlockChain {
	return mock.chain
}

func setFee(cfg *types.Config, fee int64) {
	cfg.Mempool.MinTxFeeRate = fee
	cfg.Wallet.MinFee = fee
}

//GetJSONC :
func (mock *DplatformOSMock) GetJSONC() *jsonclient.JSONClient {
	jsonc, err := jsonclient.NewJSONClient("http://" + mock.cfg.RPC.JrpcBindAddr + "/")
	if err != nil {
		return nil
	}
	return jsonc
}

//SendAndSign :
func (mock *DplatformOSMock) SendAndSign(priv crypto.PrivKey, hextx string) ([]byte, error) {
	txbytes, err := common.FromHex(hextx)
	if err != nil {
		return nil, err
	}
	tx := &types.Transaction{}
	err = types.Decode(txbytes, tx)
	if err != nil {
		return nil, err
	}
	tx.Fee = 1e6
	tx.Sign(types.SECP256K1, priv)
	reply, err := mock.api.SendTx(tx)
	if err != nil {
		return nil, err
	}
	return reply.GetMsg(), nil
}

//SendAndSignNonce       nonce   nonce
func (mock *DplatformOSMock) SendAndSignNonce(priv crypto.PrivKey, hextx string, nonce int64) ([]byte, error) {
	txbytes, err := common.FromHex(hextx)
	if err != nil {
		return nil, err
	}
	tx := &types.Transaction{}
	err = types.Decode(txbytes, tx)
	if err != nil {
		return nil, err
	}
	tx.Nonce = nonce
	tx.Fee = 1e6
	tx.Sign(types.SECP256K1, priv)
	reply, err := mock.api.SendTx(tx)
	if err != nil {
		return nil, err
	}
	return reply.GetMsg(), nil
}

func newWalletRealize(qAPI client.QueueProtocolAPI) {
	seed := &types.SaveSeedByPw{
		Seed:   "subject hamster apple parent vital can adult chapter fork business humor pen tiger void elephant",
		Passwd: "123456dpos",
	}
	reply, err := qAPI.ExecWalletFunc("wallet", "SaveSeed", seed)
	if !reply.(*types.Reply).IsOk && err != nil {
		panic(err)
	}
	reply, err = qAPI.ExecWalletFunc("wallet", "WalletUnLock", &types.WalletUnLock{Passwd: "123456dpos"})
	if !reply.(*types.Reply).IsOk && err != nil {
		panic(err)
	}
	for i, priv := range util.TestPrivkeyHex {
		privkey := &types.ReqWalletImportPrivkey{Privkey: priv, Label: fmt.Sprintf("label%d", i)}
		acc, err := qAPI.ExecWalletFunc("wallet", "WalletImportPrivkey", privkey)
		if err != nil {
			panic(err)
		}
		lognode.Info("import", "index", i, "addr", acc.(*types.WalletAccount).Acc.Addr)
	}
	req := &types.ReqAccountList{WithoutBalance: true}
	_, err = qAPI.ExecWalletFunc("wallet", "WalletGetAccountList", req)
	if err != nil {
		panic(err)
	}
}

//GetAPI :
func (mock *DplatformOSMock) GetAPI() client.QueueProtocolAPI {
	return mock.api
}

//GetRPC :
func (mock *DplatformOSMock) GetRPC() *rpc.RPC {
	return mock.rpc
}

//GetCfg :
func (mock *DplatformOSMock) GetCfg() *types.Config {
	return mock.cfg
}

//GetLastSendTx :
func (mock *DplatformOSMock) GetLastSendTx() []byte {
	return mock.lastsend
}

//Close :
func (mock *DplatformOSMock) Close() {
	mock.closeNoLock()
}

func (mock *DplatformOSMock) closeNoLock() {
	lognode.Info("network close")
	mock.network.Close()
	lognode.Info("network close")
	mock.rpc.Close()
	lognode.Info("rpc close")
	mock.mem.Close()
	lognode.Info("mem close")
	mock.exec.Close()
	lognode.Info("exec close")
	mock.cs.Close()
	lognode.Info("cs close")
	mock.wallet.Close()
	lognode.Info("wallet close")
	mock.chain.Close()
	lognode.Info("chain close")
	mock.store.Close()
	lognode.Info("store close")
	mock.client.Close()
	err := os.RemoveAll(mock.datadir)
	if err != nil {
		return
	}
}

//WaitHeight :
func (mock *DplatformOSMock) WaitHeight(height int64) error {
	for {
		header, err := mock.api.GetLastHeader()
		if err != nil {
			return err
		}
		if header.Height >= height {
			break
		}
		time.Sleep(time.Second / 10)
	}
	return nil
}

//WaitTx :
func (mock *DplatformOSMock) WaitTx(hash []byte) (*rpctypes.TransactionDetail, error) {
	if hash == nil {
		return nil, nil
	}
	for {
		param := &types.ReqHash{Hash: hash}
		_, err := mock.api.QueryTx(param)
		if err != nil {
			time.Sleep(time.Second / 10)
			continue
		}
		var testResult rpctypes.TransactionDetail
		data := rpctypes.QueryParm{
			Hash: common.ToHex(hash),
		}
		err = mock.GetJSONC().Call("DplatformOS.QueryTransaction", data, &testResult)
		return &testResult, err
	}
}

//SendHot :
func (mock *DplatformOSMock) SendHot() error {
	types.AssertConfig(mock.client)
	tx := util.CreateCoinsTx(mock.client.GetConfig(), mock.GetGenesisKey(), mock.GetHotAddress(), 10000*types.Coin)
	mock.SendTx(tx)
	return mock.Wait()
}

//SendTx :
func (mock *DplatformOSMock) SendTx(tx *types.Transaction) []byte {
	reply, err := mock.GetAPI().SendTx(tx)
	if err != nil {
		panic(err)
	}
	mock.SetLastSend(reply.GetMsg())
	return reply.GetMsg()
}

//SetLastSend :
func (mock *DplatformOSMock) SetLastSend(hash []byte) {
	mock.mu.Lock()
	mock.lastsend = hash
	mock.mu.Unlock()
}

//SendTxRPC :
func (mock *DplatformOSMock) SendTxRPC(tx *types.Transaction) []byte {
	var txhash string
	hextx := common.ToHex(types.Encode(tx))
	err := mock.GetJSONC().Call("DplatformOS.SendTransaction", &rpctypes.RawParm{Data: hextx}, &txhash)
	if err != nil {
		panic(err)
	}
	hash, err := common.FromHex(txhash)
	if err != nil {
		panic(err)
	}
	mock.lastsend = hash
	return hash
}

//Wait :
func (mock *DplatformOSMock) Wait() error {
	if mock.lastsend == nil {
		return nil
	}
	_, err := mock.WaitTx(mock.lastsend)
	return err
}

//GetAccount :
func (mock *DplatformOSMock) GetAccount(stateHash []byte, addr string) *types.Account {
	statedb := executor.NewStateDB(mock.client, stateHash, nil, nil)
	types.AssertConfig(mock.client)
	acc := account.NewCoinsAccount(mock.client.GetConfig())
	acc.SetDB(statedb)
	return acc.LoadAccount(addr)
}

//GetExecAccount :get execer account info
func (mock *DplatformOSMock) GetExecAccount(stateHash []byte, execer, addr string) *types.Account {
	statedb := executor.NewStateDB(mock.client, stateHash, nil, nil)
	types.AssertConfig(mock.client)
	acc := account.NewCoinsAccount(mock.client.GetConfig())
	acc.SetDB(statedb)
	return acc.LoadExecAccount(addr, address.ExecAddress(execer))
}

//GetBlock :
func (mock *DplatformOSMock) GetBlock(height int64) *types.Block {
	blocks, err := mock.api.GetBlocks(&types.ReqBlocks{Start: height, End: height})
	if err != nil {
		panic(err)
	}
	return blocks.Items[0].Block
}

//GetLastBlock :
func (mock *DplatformOSMock) GetLastBlock() *types.Block {
	header, err := mock.api.GetLastHeader()
	if err != nil {
		panic(err)
	}
	return mock.GetBlock(header.Height)
}

//GetClient :
func (mock *DplatformOSMock) GetClient() queue.Client {
	return mock.client
}

//GetHotKey :
func (mock *DplatformOSMock) GetHotKey() crypto.PrivKey {
	return util.TestPrivkeyList[0]
}

//GetHotAddress :
func (mock *DplatformOSMock) GetHotAddress() string {
	return address.PubKeyToAddress(mock.GetHotKey().PubKey().Bytes()).String()
}

//GetGenesisKey :
func (mock *DplatformOSMock) GetGenesisKey() crypto.PrivKey {
	return util.TestPrivkeyList[1]
}

//GetGenesisAddress :
func (mock *DplatformOSMock) GetGenesisAddress() string {
	return address.PubKeyToAddress(mock.GetGenesisKey().PubKey().Bytes()).String()
}

type mockP2P struct {
}

//SetQueueClient :
func (m *mockP2P) SetQueueClient(client queue.Client) {
	go func() {
		p2pKey := "p2p"
		client.Sub(p2pKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventPeerInfo:
				msg.Reply(client.NewMessage(p2pKey, types.EventPeerList, &types.PeerList{}))
			case types.EventGetNetInfo:
				msg.Reply(client.NewMessage(p2pKey, types.EventPeerList, &types.NodeNetInfo{}))
			case types.EventTxBroadcast, types.EventBlockBroadcast:
				client.FreeMessage(msg)
			default:
				msg.ReplyErr("p2p->Do not support "+types.GetEventName(int(msg.Ty)), types.ErrNotSupport)
			}
		}
	}()
}

//Wait for ready
func (m *mockP2P) Wait() {}

//Close :
func (m *mockP2P) Close() {
}
