package blockchain

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sync/atomic"
	"testing"
	"time"

	bcMocks "github.com/D-PlatformOperatingSystem/dpos/blockchain/mocks"
	"github.com/D-PlatformOperatingSystem/dpos/client"
	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	"github.com/D-PlatformOperatingSystem/dpos/consensus"
	"github.com/D-PlatformOperatingSystem/dpos/executor"
	"github.com/D-PlatformOperatingSystem/dpos/mempool"
	"github.com/D-PlatformOperatingSystem/dpos/p2p"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/rpc"
	"github.com/D-PlatformOperatingSystem/dpos/rpc/jsonclient"
	rpctypes "github.com/D-PlatformOperatingSystem/dpos/rpc/types"
	"github.com/D-PlatformOperatingSystem/dpos/store"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
	"github.com/D-PlatformOperatingSystem/dpos/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var sendTxWait = time.Millisecond * 5

type DplatformOSMock struct {
	random  *rand.Rand
	q       queue.Queue
	client  queue.Client
	api     client.QueueProtocolAPI
	chain   *BlockChain
	mem     queue.Module
	cs      queue.Module
	exec    *executor.Executor
	wallet  queue.Module
	network queue.Module
	store   queue.Module
	rpc     *rpc.RPC
	cfg     *types.Config
	sub     *types.ConfigSubModule
	datadir string
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

//Close :
func (mock *DplatformOSMock) Close() {
	mock.closeNoLock()
}

func (mock *DplatformOSMock) closeNoLock() {
	mock.network.Close()
	mock.rpc.Close()
	mock.mem.Close()
	mock.exec.Close()
	mock.cs.Close()
	mock.wallet.Close()
	mock.chain.Close()
	mock.store.Close()
	mock.client.Close()
	err := os.RemoveAll(mock.datadir)
	if err != nil {
		return
	}
}

//GetClient :
func (mock *DplatformOSMock) GetClient() queue.Client {
	return mock.client
}

//GetBlockChain :
func (mock *DplatformOSMock) GetBlockChain() *BlockChain {
	return mock.chain
}

//GetGenesisKey :
func (mock *DplatformOSMock) GetGenesisKey() crypto.PrivKey {
	return util.TestPrivkeyList[1]
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

func (mock *DplatformOSMock) GetJSONC() *jsonclient.JSONClient {
	jsonc, err := jsonclient.NewJSONClient("http://" + mock.cfg.RPC.JrpcBindAddr + "/")
	if err != nil {
		return nil
	}
	return jsonc
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

func Test_procSubscribePush_pushSupport(t *testing.T) {
	chain, mockDOM := createBlockChainWithFalgSet(t, false, false)
	defer mockDOM.Close()
	subscribe := new(types.PushSubscribeReq)
	err := chain.procSubscribePush(subscribe)
	assert.Equal(t, types.ErrPushNotSupport, err)
}

func Test_procSubscribePush_nilParacheck(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	defer mockDOM.Close()
	err := chain.procSubscribePush(nil)
	assert.Equal(t, err, types.ErrInvalidParam)
}

func Test_addSubscriber_Paracheck(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	defer mockDOM.Close()
	subscribe := new(types.PushSubscribeReq)
	subscribe.LastSequence = 1
	err := chain.procSubscribePush(subscribe)
	assert.Equal(t, err, types.ErrInvalidParam)
}

func Test_addSubscriber_conflictPara(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	defer mockDOM.Close()
	subscribe := new(types.PushSubscribeReq)
	subscribe.LastSequence = 1
	err := chain.procSubscribePush(subscribe)
	assert.Equal(t, err, types.ErrInvalidParam)
}

func Test_addSubscriber_InvalidURL(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	defer mockDOM.Close()
	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = ""
	err := chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, types.ErrInvalidParam)
}

func Test_addSubscriber_InvalidType(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	defer mockDOM.Close()
	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.Type = int32(3)
	err := chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, types.ErrInvalidParam)
}

func Test_addSubscriber_inconsistentSeqHash(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	defer mockDOM.Close()
	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.LastSequence = 1
	err := chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, types.ErrInvalidParam)

	subscribe.LastSequence = 0
	subscribe.LastHeight = 1
	err = chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, types.ErrInvalidParam)
}

func Test_addSubscriber_Success(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	defer mockDOM.Close()
	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	key := calcPushKey(subscribe.Name)
	subInfo, err := chain.push.store.GetKey(key)
	assert.NotEqual(t, err, nil)
	assert.NotEqual(t, subInfo, nil)

	err = chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, nil)
	subInfo, err = chain.push.store.GetKey(key)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, subInfo, nil)

	var originSubInfo types.PushWithStatus
	err = types.Decode(subInfo, &originSubInfo)
	assert.Equal(t, err, nil)
	assert.Equal(t, originSubInfo.Push.URL, subscribe.URL)

	pushes, _ := chain.ProcListPush()
	assert.Equal(t, subscribe.Name, pushes.Pushes[0].Name)

	//    push，                push
	chainAnother := &BlockChain{
		isRecordBlockSequence: true,
		enablePushSubscribe:   true,
	}
	chainAnother.push = newpush(chain.blockStore, chain.blockStore, chain.client.GetConfig())
	recoverpushes, _ := chainAnother.ProcListPush()
	assert.Equal(t, subscribe.Name, recoverpushes.Pushes[0].Name)
}

func Test_addSubscriber_WithSeqHashHeight(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	defer mockDOM.Close()

	blockSeq, err := chain.blockStore.GetBlockSequence(5)
	assert.Equal(t, err, nil)
	header, err := chain.blockStore.GetBlockHeaderByHash(blockSeq.Hash)
	assert.Equal(t, err, nil)

	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.LastSequence = 5
	subscribe.LastHeight = header.Height
	subscribe.LastBlockHash = common.ToHex(blockSeq.Hash)
	key := calcPushKey(subscribe.Name)
	_, err = chain.push.store.GetKey(key)
	assert.NotEqual(t, err, nil)

	err = chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, nil)
	subInfo, err := chain.push.store.GetKey(key)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, subInfo, nil)

	var originSubInfo types.PushWithStatus
	err = types.Decode(subInfo, &originSubInfo)
	assert.Equal(t, err, nil)
	assert.Equal(t, originSubInfo.Push.URL, subscribe.URL)

	pushes, _ := chain.ProcListPush()
	assert.Equal(t, subscribe.Name, pushes.Pushes[0].Name)
}

func Test_PostBlockFail(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	ps := &bcMocks.PostService{}
	ps.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("timeout"))
	chain.push.postService = ps

	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.Type = PushBlock

	err := chain.push.addSubscriber(subscribe)
	time.Sleep(2 * time.Second)
	assert.Equal(t, err, nil)
	createBlocks(t, mockDOM, chain, 10)
	keyStr := string(calcPushKey(subscribe.Name))
	pushNotify := chain.push.tasks[keyStr]
	assert.Equal(t, pushNotify.subscribe.Name, subscribe.Name)
	assert.Equal(t, pushNotify.status, running)
	time.Sleep(1 * time.Second)
	createBlocks(t, mockDOM, chain, 1)

	assert.Greater(t, atomic.LoadInt32(&pushNotify.postFail2Sleep), int32(0))
	time.Sleep(1 * time.Second)

	lastSeq, _ := chain.ProcGetLastPushSeq(subscribe.Name)
	assert.Equal(t, lastSeq, int64(-1))

	mockDOM.Close()
}

func Test_GetLastPushSeqFailDue2RecordBlockSequence(t *testing.T) {
	chain, mockDOM := createBlockChainWithFalgSet(t, false, false)
	_, err := chain.ProcGetLastPushSeq("test")
	assert.Equal(t, types.ErrRecordBlockSequence, err)
	mockDOM.Close()
}

func Test_GetLastPushSeqFailDue2enablePushSubscribe(t *testing.T) {
	chain, mockDOM := createBlockChainWithFalgSet(t, true, false)
	_, err := chain.ProcGetLastPushSeq("test")
	assert.Equal(t, types.ErrPushNotSupport, err)
	mockDOM.Close()
}

func Test_GetLastPushSeqFailDue2NotSubscribed(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	_, err := chain.ProcGetLastPushSeq("test")
	assert.Equal(t, types.ErrPushNotSubscribed, err)
	mockDOM.Close()
}

func Test_PostDataFail(t *testing.T) {
	chain, mockDOM := createBlockChain(t)

	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.Type = PushBlock

	err := chain.push.addSubscriber(subscribe)
	time.Sleep(2 * time.Second)
	assert.Equal(t, err, nil)
	createBlocks(t, mockDOM, chain, 10)
	keyStr := string(calcPushKey(subscribe.Name))
	pushNotify := chain.push.tasks[keyStr]
	assert.Equal(t, pushNotify.subscribe.Name, subscribe.Name)
	assert.Equal(t, pushNotify.status, running)

	err = chain.push.postService.PostData(subscribe, []byte("1"), 1)
	assert.NotEqual(t, nil, err)

	mockDOM.Close()
}

func Test_PostBlockSuccess(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	ps := &bcMocks.PostService{}
	ps.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	chain.push.postService = ps

	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.Type = PushBlock

	err := chain.push.addSubscriber(subscribe)
	time.Sleep(2 * time.Second)
	assert.Equal(t, err, nil)
	createBlocks(t, mockDOM, chain, 10)
	keyStr := string(calcPushKey(subscribe.Name))
	pushNotify := chain.push.tasks[keyStr]
	assert.Equal(t, pushNotify.subscribe.Name, subscribe.Name)
	assert.Equal(t, pushNotify.status, running)
	time.Sleep(1 * time.Second)
	//     push，
	err = chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, nil)

	createBlocks(t, mockDOM, chain, 1)

	assert.Equal(t, atomic.LoadInt32(&pushNotify.postFail2Sleep), int32(0))
	time.Sleep(1 * time.Second)

	lastSeq, _ := chain.ProcGetLastPushSeq(subscribe.Name)
	assert.Greater(t, lastSeq, int64(21))

	mockDOM.Close()
}

func Test_PostBlockHeaderSuccess(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	ps := &bcMocks.PostService{}
	ps.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	chain.push.postService = ps

	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.Type = PushBlockHeader

	err := chain.push.addSubscriber(subscribe)
	time.Sleep(2 * time.Second)
	assert.Equal(t, err, nil)
	createBlocks(t, mockDOM, chain, 10)
	keyStr := string(calcPushKey(subscribe.Name))
	pushNotify := chain.push.tasks[keyStr]
	assert.Equal(t, pushNotify.subscribe.Name, subscribe.Name)
	assert.Equal(t, pushNotify.status, running)

	createBlocks(t, mockDOM, chain, 1)

	assert.Equal(t, atomic.LoadInt32(&pushNotify.postFail2Sleep), int32(0))
	time.Sleep(1 * time.Second)

	lastSeq, _ := chain.ProcGetLastPushSeq(subscribe.Name)
	assert.Greater(t, lastSeq, int64(21))

	mockDOM.Close()
}

func Test_PostTxReceipt(t *testing.T) {
	chain, mockDOM := createBlockChain(t)

	ps := &bcMocks.PostService{}
	ps.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	chain.push.postService = ps
	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.Type = PushTxReceipt
	subscribe.Contract = make(map[string]bool)
	subscribe.Contract["coins"] = true

	err := chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, nil)
	createBlocks(t, mockDOM, chain, 1)
	keyStr := string(calcPushKey(subscribe.Name))
	pushNotify := chain.push.tasks[keyStr]
	assert.Equal(t, pushNotify.subscribe.Name, subscribe.Name)

	assert.Equal(t, atomic.LoadInt32(&pushNotify.status), running)
	time.Sleep(2 * time.Second)
	assert.Equal(t, atomic.LoadInt32(&pushNotify.postFail2Sleep), int32(0))
	defer mockDOM.Close()
}

func Test_AddPush_reachMaxNum(t *testing.T) {
	chain, mockDOM := createBlockChain(t)

	ps := &bcMocks.PostService{}
	ps.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	chain.push.postService = ps

	for i := 0; i < maxPushSubscriber; i++ {
		subscribe := new(types.PushSubscribeReq)
		subscribe.Name = "push-test"
		subscribe.URL = "http://localhost"
		subscribe.Type = PushTxReceipt
		subscribe.Contract = make(map[string]bool)
		subscribe.Contract["coins"] = true
		subscribe.Name = "push-test-" + fmt.Sprintf("%d", i)
		err := chain.push.addSubscriber(subscribe)
		assert.Equal(t, err, nil)
	}
	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.Type = PushTxReceipt
	subscribe.Contract = make(map[string]bool)
	subscribe.Contract["coins"] = true
	subscribe.Name = "push-test-lastOne"
	err := chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, types.ErrTooManySeqCB)
	defer mockDOM.Close()
}

func Test_AddPush_PushNameShouldDiff(t *testing.T) {
	chain, mockDOM := createBlockChain(t)

	ps := &bcMocks.PostService{}
	ps.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	chain.push.postService = ps

	var pushNames []string
	for i := 0; i < 10; i++ {
		subscribe := new(types.PushSubscribeReq)
		subscribe.Name = "push-test"
		subscribe.URL = "http://localhost"
		subscribe.Type = PushTxReceipt
		subscribe.Contract = make(map[string]bool)
		subscribe.Contract["coins"] = true
		subscribe.Name = "push-test-" + fmt.Sprintf("%d", i)
		err := chain.push.addSubscriber(subscribe)
		pushNames = append(pushNames, subscribe.Name)
		assert.Equal(t, err, nil)
	}
	assert.Equal(t, len(chain.push.tasks), 10)
	//       name  url push
	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.Type = PushTxReceipt
	subscribe.Contract = make(map[string]bool)
	subscribe.Contract["coins"] = true
	subscribe.Name = "push-test-" + fmt.Sprintf("%d", 9)
	subscribe.URL = "http://localhost:28803"
	err := chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, types.ErrNotAllowModifyPush)

	//push
	chainAnother := &BlockChain{
		isRecordBlockSequence: true,
		enablePushSubscribe:   true,
	}
	chainAnother.push = newpush(chain.blockStore, chain.blockStore, chain.client.GetConfig())
	assert.Equal(t, 10, len(chainAnother.push.tasks))
	for _, name := range pushNames {
		assert.NotEqual(t, chainAnother.push.tasks[string(calcPushKey(name))], nil)
	}
	defer mockDOM.Close()
}

func Test_rmPushFailTask(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	chain.push.postFail2Sleep = int32(1)
	ps := &bcMocks.PostService{}
	ps.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("timeout"))
	chain.push.postService = ps

	createBlocks(t, mockDOM, chain, 10)
	var pushNames []string
	subCnt := 10
	for i := 0; i < subCnt; i++ {
		subscribe := new(types.PushSubscribeReq)
		subscribe.Name = "push-test"
		subscribe.URL = "http://localhost"
		subscribe.Type = PushTxReceipt
		subscribe.Contract = make(map[string]bool)
		subscribe.Contract["coins"] = true

		subscribe.Name = fmt.Sprintf("%d", i) + "-push-test-"
		err := chain.push.addSubscriber(subscribe)
		pushNames = append(pushNames, subscribe.Name)
		assert.Equal(t, err, nil)
	}
	chain.push.mu.Lock()
	assert.Equal(t, len(chain.push.tasks), subCnt)
	chain.push.mu.Unlock()
	createBlocks(t, mockDOM, chain, 10)
	time.Sleep(1 * time.Second)

	createBlocks(t, mockDOM, chain, 10)
	time.Sleep(1 * time.Second)
	closeChan := make(chan struct{})

	go func() {
		sleepCnt := 30
		for {
			chain.push.mu.Lock()
			if 0 == len(chain.push.tasks) {
				chain.push.mu.Unlock()
				close(closeChan)
				return
			}
			chain.push.mu.Unlock()
			sleepCnt--
			if sleepCnt <= 0 {
				close(closeChan)
				return
			}
			time.Sleep(time.Second)
		}
	}()

	<-closeChan
	fmt.Println("stoping Test_rmPushFailTask")
	chain.push.mu.Lock()
	assert.Equal(t, 0, len(chain.push.tasks))
	chain.push.mu.Unlock()

	defer mockDOM.Close()
}

//
func Test_ReactivePush(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	ps := &bcMocks.PostService{}
	ps.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	chain.push.postService = ps

	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.Type = PushBlock

	err := chain.push.addSubscriber(subscribe)
	time.Sleep(2 * time.Second)
	assert.Equal(t, err, nil)
	createBlocks(t, mockDOM, chain, 10)
	keyStr := string(calcPushKey(subscribe.Name))
	pushNotify := chain.push.tasks[keyStr]
	assert.Equal(t, pushNotify.subscribe.Name, subscribe.Name)
	assert.Equal(t, pushNotify.status, running)
	time.Sleep(1 * time.Second)

	createBlocks(t, mockDOM, chain, 1)

	assert.Equal(t, atomic.LoadInt32(&pushNotify.postFail2Sleep), int32(0))
	time.Sleep(1 * time.Second)

	lastSeq, _ := chain.ProcGetLastPushSeq(subscribe.Name)
	assert.Greater(t, lastSeq, int64(21))

	mockpsFail := &bcMocks.PostService{}
	mockpsFail.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("timeout"))
	chain.push.postService = mockpsFail
	chain.push.postFail2Sleep = int32(1)
	createBlocks(t, mockDOM, chain, 10)
	time.Sleep(4 * time.Second)
	assert.Equal(t, atomic.LoadInt32(&pushNotify.status), notRunning)
	lastSeq, _ = chain.ProcGetLastPushSeq(subscribe.Name)

	//
	chain.push.postService = ps
	err = chain.push.addSubscriber(subscribe)
	assert.Equal(t, err, nil)
	time.Sleep(1 * time.Second)
	chain.push.mu.Lock()
	pushNotify = chain.push.tasks[keyStr]
	chain.push.mu.Unlock()
	assert.Equal(t, atomic.LoadInt32(&pushNotify.status), running)
	lastSeqAfter, _ := chain.ProcGetLastPushSeq(subscribe.Name)
	assert.Greater(t, lastSeqAfter, lastSeq)

	mockDOM.Close()
}

//
func Test_RecoverPush(t *testing.T) {
	chain, mockDOM := createBlockChain(t)
	ps := &bcMocks.PostService{}
	ps.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	chain.push.postService = ps

	subscribe := new(types.PushSubscribeReq)
	subscribe.Name = "push-test"
	subscribe.URL = "http://localhost"
	subscribe.Type = PushBlock

	err := chain.push.addSubscriber(subscribe)
	time.Sleep(2 * time.Second)
	assert.Equal(t, err, nil)
	createBlocks(t, mockDOM, chain, 10)
	keyStr := string(calcPushKey(subscribe.Name))
	pushNotifyInfo := chain.push.tasks[keyStr]
	assert.Equal(t, pushNotifyInfo.subscribe.Name, subscribe.Name)
	assert.Equal(t, pushNotifyInfo.status, running)
	time.Sleep(1 * time.Second)

	createBlocks(t, mockDOM, chain, 1)

	assert.Equal(t, atomic.LoadInt32(&pushNotifyInfo.postFail2Sleep), int32(0))
	time.Sleep(1 * time.Second)

	lastSeq, _ := chain.ProcGetLastPushSeq(subscribe.Name)
	assert.Greater(t, lastSeq, int64(21))

	mockpsFail := &bcMocks.PostService{}
	mockpsFail.On("PostData", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("timeout"))
	chain.push.postService = mockpsFail
	chain.push.postFail2Sleep = int32(1)
	createBlocks(t, mockDOM, chain, 10)
	time.Sleep(3 * time.Second)
	assert.Equal(t, atomic.LoadInt32(&pushNotifyInfo.status), notRunning)
	chain.ProcGetLastPushSeq(subscribe.Name)

	//dplatformos push     ，       task ，
	chainAnother := &BlockChain{
		isRecordBlockSequence: true,
		enablePushSubscribe:   true,
	}
	chainAnother.push = newpush(chain.blockStore, chain.blockStore, chain.client.GetConfig())
	var nilInfo *pushNotify
	assert.Equal(t, chainAnother.push.tasks[string(calcPushKey(subscribe.Name))], nilInfo)

	mockDOM.Close()
}

//init work
func NewDplatformOSMock(cfgpath string, mockapi client.QueueProtocolAPI) *DplatformOSMock {
	cfg := types.NewDplatformOSConfig(types.GetDefaultCfgstring())
	return newWithConfigNoLock(cfg, mockapi)
}

func NewDplatformOSMockWithFlag(cfgpath string, mockapi client.QueueProtocolAPI, isRecordBlockSequence, enablePushSubscribe bool) *DplatformOSMock {
	cfg := types.NewDplatformOSConfig(types.GetDefaultCfgstring())
	cfg.GetModuleConfig().BlockChain.IsRecordBlockSequence = isRecordBlockSequence
	cfg.GetModuleConfig().BlockChain.EnablePushSubscribe = enablePushSubscribe
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

	mock.store = store.New(cfg)
	mock.store.SetQueueClient(q.Client())

	mock.chain = New(cfg)
	mock.chain.SetQueueClient(q.Client())

	mock.cs = consensus.New(cfg)
	mock.cs.SetQueueClient(q.Client())
	fmt.Print("init consensus " + mfg.Consensus.Name)

	mock.mem = mempool.New(cfg)
	mock.mem.SetQueueClient(q.Client())
	mock.mem.Wait()
	fmt.Print("init mempool")
	if mfg.P2P.Enable {
		mock.network = p2p.NewP2PMgr(cfg)
		mock.network.SetQueueClient(q.Client())
	} else {
		mock.network = &mockP2P{}
		mock.network.SetQueueClient(q.Client())
	}
	fmt.Print("init P2P")
	cli := q.Client()
	w := wallet.New(cfg)
	mock.client = q.Client()
	mock.wallet = w
	mock.wallet.SetQueueClient(cli)
	fmt.Print("init wallet")
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

func addTx(cfg *types.DplatformOSConfig, priv crypto.PrivKey, api client.QueueProtocolAPI) ([]*types.Transaction, string, error) {
	txs := util.GenCoinsTxs(cfg, priv, 1)
	hash := common.ToHex(txs[0].Hash())
	reply, err := api.SendTx(txs[0])
	if err != nil {
		return nil, hash, err
	}
	if !reply.GetIsOk() {
		return nil, hash, errors.New("sendtx unknow error")
	}
	return txs, hash, nil
}

func createBlocks(t *testing.T, mockDOM *DplatformOSMock, blockchain *BlockChain, number int64) {
	chainlog.Info("testProcAddBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + number

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}
	cfg := mockDOM.GetClient().GetConfig()
	for {
		_, _, err = addTx(cfg, mockDOM.GetGenesisKey(), mockDOM.GetAPI())
		require.NoError(t, err)
		curheight = blockchain.GetBlockHeight()
		chainlog.Info("testProcAddBlockMsg ", "curheight", curheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	chainlog.Info("testProcAddBlockMsg end --------------------")
}

func createBlockChain(t *testing.T) (*BlockChain, *DplatformOSMock) {
	mockDOM := NewDplatformOSMock("", nil)

	//cfg := mockDOM.GetClient().GetConfig()
	blockchain := mockDOM.GetBlockChain()
	//        10
	createBlocks(t, mockDOM, blockchain, 10)
	return blockchain, mockDOM
}

func createBlockChainWithFalgSet(t *testing.T, isRecordBlockSequence, enablePushSubscribe bool) (*BlockChain, *DplatformOSMock) {
	mockDOM := NewDplatformOSMockWithFlag("", nil, isRecordBlockSequence, enablePushSubscribe)

	//cfg := mockDOM.GetClient().GetConfig()
	blockchain := mockDOM.GetBlockChain()
	//        10
	createBlocks(t, mockDOM, blockchain, 10)
	return blockchain, mockDOM
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
		fmt.Print("import", "index", i, "addr", acc.(*types.WalletAccount).Acc.Addr)
	}
	req := &types.ReqAccountList{WithoutBalance: true}
	_, err = qAPI.ExecWalletFunc("wallet", "WalletGetAccountList", req)
	if err != nil {
		panic(err)
	}
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
