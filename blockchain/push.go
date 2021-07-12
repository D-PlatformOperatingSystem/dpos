package blockchain

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

const (
	notRunning               = int32(1)
	running                  = int32(2)
	pushBlockMaxSeq          = 10
	pushTxReceiptMaxSeq      = 100
	pushMaxSize              = 1 * 1024 * 1024
	maxPushSubscriber        = int(100)
	subscribeStatusActive    = int32(1)
	subscribeStatusNotActive = int32(2)
	postFail2Sleep           = int32(60) //      ，sleep
	chanBufCap               = int(10)
)

// Push types ID
const (
	PushBlock       = int32(0)
	PushBlockHeader = int32(1)
	PushTxReceipt   = int32(2)
)

// CommonStore    store
//      ，      db.KVDB
//       ，  store,
type CommonStore interface {
	SetSync(key, value []byte) error
	Set(key, value []byte) error
	GetKey(key []byte) ([]byte, error)
	PrefixCount(prefix []byte) int64
	List(prefix []byte) ([][]byte, error)
}

//SequenceStore ...
type SequenceStore interface {
	LoadBlockLastSequence() (int64, error)
	// seqUpdateChan -> block sequence
	GetBlockSequence(seq int64) (*types.BlockSequence, error)
	// hash -> block header
	GetBlockHeaderByHash(hash []byte) (*types.Header, error)
	// seqUpdateChan -> block, size
	LoadBlockBySequence(seq int64) (*types.BlockDetail, int, error)
	// get last header
	LastHeader() *types.Header
	// hash -> seqUpdateChan
	GetSequenceByHash(hash []byte) (int64, error)
}

//PostService ...
type PostService interface {
	PostData(subscribe *types.PushSubscribeReq, postdata []byte, seq int64) (err error)
}

//                    goroute，       subscriber           ，
//    ，                  ，             ，     cpu   ，
//                ，    cach
//TODO：                        ，
//pushNotify push Notify
type pushNotify struct {
	subscribe      *types.PushSubscribeReq
	seqUpdateChan  chan int64
	closechan      chan struct{}
	status         int32
	postFail2Sleep int32
}

//Push ...
type Push struct {
	store          CommonStore
	sequenceStore  SequenceStore
	tasks          map[string]*pushNotify
	mu             sync.Mutex
	postService    PostService
	cfg            *types.DplatformOSConfig
	postFail2Sleep int32
	postwg         *sync.WaitGroup
}

//PushClient ...
type PushClient struct {
	client *http.Client
}

// PushType ...
type PushType int32

func (pushType PushType) string() string {
	return []string{"PushBlock", "PushBlockHeader", "PushTxReceipt", "NotSupported"}[pushType]
}

//PostData ...
func (pushClient *PushClient) PostData(subscribe *types.PushSubscribeReq, postdata []byte, seq int64) (err error) {
	//post data in body
	chainlog.Info("postData begin", "seq", seq, "subscribe name", subscribe.Name)
	var buf bytes.Buffer
	g := gzip.NewWriter(&buf)
	if _, err = g.Write(postdata); err != nil {
		return err
	}
	if err = g.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", subscribe.URL, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := pushClient.client.Do(req)
	if err != nil {
		chainlog.Info("postData", "Do err", err)
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return err
	}
	if string(body) != "ok" && string(body) != "OK" {
		chainlog.Error("postData fail", "name:", subscribe.Name, "URL", subscribe.URL,
			"Contract:", subscribe.Contract, "body", string(body))
		_ = resp.Body.Close()
		return types.ErrPushSeqPostData
	}
	chainlog.Debug("postData success", "name", subscribe.Name, "URL", subscribe.URL,
		"Contract:", subscribe.Contract, "updateSeq", seq)
	return resp.Body.Close()
}

//ProcAddBlockSeqCB   seq callback
func (chain *BlockChain) procSubscribePush(subscribe *types.PushSubscribeReq) error {
	if !chain.enablePushSubscribe {
		chainlog.Error("Push is not enabled for subscribed")
		return types.ErrPushNotSupport
	}

	if !chain.isRecordBlockSequence {
		chainlog.Error("procSubscribePush not support sequence")
		return types.ErrRecordBlockSequence
	}

	if subscribe == nil {
		chainlog.Error("procSubscribePush para is null")
		return types.ErrInvalidParam
	}

	if chain.client.GetConfig().IsEnable("reduceLocaldb") && subscribe.Type == PushTxReceipt {
		chainlog.Error("Tx receipts are reduced on this node")
		return types.ErrTxReceiptReduced
	}
	return chain.push.addSubscriber(subscribe)
}

//ProcListPush
func (chain *BlockChain) ProcListPush() (*types.PushSubscribes, error) {
	if !chain.isRecordBlockSequence {
		return nil, types.ErrRecordBlockSequence
	}
	if !chain.enablePushSubscribe {
		return nil, types.ErrPushNotSupport
	}

	values, err := chain.push.store.List(pushPrefix)
	if err != nil {
		return nil, err
	}
	var listSeqCBs types.PushSubscribes
	for _, value := range values {
		var onePush types.PushWithStatus
		err := types.Decode(value, &onePush)
		if err != nil {
			return nil, err
		}
		listSeqCBs.Pushes = append(listSeqCBs.Pushes, onePush.Push)
	}
	return &listSeqCBs, nil
}

// ProcGetLastPushSeq Seq     0   ，                  -1
func (chain *BlockChain) ProcGetLastPushSeq(name string) (int64, error) {
	if !chain.isRecordBlockSequence {
		return -1, types.ErrRecordBlockSequence
	}
	if !chain.enablePushSubscribe {
		return -1, types.ErrPushNotSupport
	}

	lastSeqbytes, err := chain.push.store.GetKey(calcLastPushSeqNumKey(name))
	if lastSeqbytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("getSeqCBLastNum", "error", err)
		}
		return -1, types.ErrPushNotSubscribed
	}
	n, err := decodeHeight(lastSeqbytes)
	if err != nil {
		return -1, err
	}
	storeLog.Error("getSeqCBLastNum", "name", name, "num", n)

	return n, nil
}

func newpush(commonStore CommonStore, seqStore SequenceStore, cfg *types.DplatformOSConfig) *Push {
	tasks := make(map[string]*pushNotify)

	pushClient := &PushClient{
		client: &http.Client{Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}},
	}
	service := &Push{store: commonStore,
		sequenceStore:  seqStore,
		tasks:          tasks,
		postService:    pushClient,
		cfg:            cfg,
		postFail2Sleep: postFail2Sleep,
		postwg:         &sync.WaitGroup{},
	}
	service.init()

	return service
}

//   :       seq
func (push *Push) init() {
	var subscribes []*types.PushSubscribeReq
	values, err := push.store.List(pushPrefix)
	if err != nil && err != dbm.ErrNotFoundInDb {
		chainlog.Error("Push init", "err", err)
		return
	}
	if 0 == len(values) {
		return
	}
	for _, value := range values {
		var pushWithStatus types.PushWithStatus
		err := types.Decode(value, &pushWithStatus)
		if err != nil {
			chainlog.Error("Push init", "Failed to decode subscribe due to err:", err)
			return
		}
		if pushWithStatus.Status == subscribeStatusActive {
			subscribes = append(subscribes, pushWithStatus.Push)
		}

	}
	for _, subscribe := range subscribes {
		push.addTask(subscribe)
	}
}

// Close ...
func (push *Push) Close() {
	push.mu.Lock()
	for _, task := range push.tasks {
		close(task.closechan)
	}
	push.mu.Unlock()
	push.postwg.Wait()
}

func (push *Push) addSubscriber(subscribe *types.PushSubscribeReq) error {
	if subscribe == nil {
		chainlog.Error("addSubscriber input para is null")
		return types.ErrInvalidParam
	}

	if subscribe.Type != PushBlock && subscribe.Type != PushBlockHeader && subscribe.Type != PushTxReceipt {
		chainlog.Error("addSubscriber input type is error", "type", subscribe.Type)
		return types.ErrInvalidParam
	}

	//             ，        ，
	if subscribe.LastBlockHash != "" || subscribe.LastSequence != 0 || subscribe.LastHeight != 0 {
		if subscribe.LastBlockHash == "" || subscribe.LastSequence == 0 || subscribe.LastHeight == 0 {
			chainlog.Error("addSubscriber ErrInvalidParam", "seqUpdateChan", subscribe.LastSequence, "height", subscribe.LastHeight, "hash", subscribe.LastBlockHash)
			return types.ErrInvalidParam
		}
	}

	//              ，             ，
	if exist, subscribeInDB := push.hasSubscriberExist(subscribe); exist {
		if subscribeInDB.URL != subscribe.URL || subscribeInDB.Type != subscribe.Type {
			return types.ErrNotAllowModifyPush
		}
		//          push  ，
		return push.check2ResumePush(subscribeInDB)
	}

	push.mu.Lock()
	if len(push.tasks) >= maxPushSubscriber {
		chainlog.Error("addSubscriber too many push subscriber")
		push.mu.Unlock()
		return types.ErrTooManySeqCB
	}
	push.mu.Unlock()

	//
	if subscribe.LastSequence > 0 {
		sequence, err := push.sequenceStore.GetBlockSequence(subscribe.LastSequence)
		if err != nil {
			chainlog.Error("addSubscriber continue-seqUpdateChan-push", "load-1", err)
			return err
		}

		//    ，
		//     ，      hash，      ；    hash
		reloadHash := common.ToHex(sequence.Hash)
		if subscribe.LastBlockHash == reloadHash {
			//    last seqUpdateChan，     0
			err = push.setLastPushSeq(subscribe.Name, subscribe.LastSequence)
			if err != nil {
				chainlog.Error("addSubscriber", "setLastPushSeq", err)
				return err
			}
			return push.persisAndStart(subscribe)
		}
	}

	return push.persisAndStart(subscribe)
}

func (push *Push) hasSubscriberExist(subscribe *types.PushSubscribeReq) (bool, *types.PushSubscribeReq) {
	value, err := push.store.GetKey(calcPushKey(subscribe.Name))
	if err == nil {
		var pushWithStatus types.PushWithStatus
		err = types.Decode(value, &pushWithStatus)
		return err == nil, pushWithStatus.Push
	}
	return false, nil
}

func (push *Push) subscriberCount() int64 {
	return push.store.PrefixCount(pushPrefix)
}

//
func (push *Push) persisAndStart(subscribe *types.PushSubscribeReq) error {
	if len(subscribe.Name) > 128 || len(subscribe.URL) > 1024 || len(subscribe.URL) == 0 {
		storeLog.Error("Invalid para to persisAndStart due to wrong length", "len(subscribe.Name)=", len(subscribe.Name),
			"len(subscribe.URL)=", len(subscribe.URL), "len(subscribe.Contract)=", len(subscribe.Contract))
		return types.ErrInvalidParam
	}
	key := calcPushKey(subscribe.Name)
	storeLog.Info("persisAndStart", "key", string(key), "subscribe", subscribe)
	push.addTask(subscribe)

	pushWithStatus := &types.PushWithStatus{
		Push:   subscribe,
		Status: subscribeStatusActive,
	}

	return push.store.SetSync(key, types.Encode(pushWithStatus))
}

func (push *Push) check2ResumePush(subscribe *types.PushSubscribeReq) error {
	if len(subscribe.Name) > 128 || len(subscribe.URL) > 1024 || len(subscribe.Contract) > 128 {
		storeLog.Error("Invalid para to persisAndStart due to wrong length", "len(subscribe.Name)=", len(subscribe.Name),
			"len(subscribe.URL)=", len(subscribe.URL), "len(subscribe.Contract)=", len(subscribe.Contract))
		return types.ErrInvalidParam
	}
	push.mu.Lock()
	defer push.mu.Unlock()

	keyStr := string(calcPushKey(subscribe.Name))
	storeLog.Info("check2ResumePush", "key", keyStr, "subscribe", subscribe)

	notify := push.tasks[keyStr]
	//
	if nil == notify {
		push.tasks[keyStr] = &pushNotify{
			subscribe:     subscribe,
			seqUpdateChan: make(chan int64, chanBufCap),
			closechan:     make(chan struct{}),
			status:        notRunning,
		}
		push.runTask(push.tasks[keyStr])
		storeLog.Info("check2ResumePush new pushNotify created")
		return nil
	}

	if running == atomic.LoadInt32(&notify.status) {
		storeLog.Info("Is already in state:running", "postFail2Sleep", atomic.LoadInt32(&notify.postFail2Sleep))
		atomic.StoreInt32(&notify.postFail2Sleep, 0)
		return nil
	}
	storeLog.Info("check2ResumePush to resume a push", "name", subscribe.Name)

	push.runTask(push.tasks[keyStr])
	return nil
}

//  add   push ,     seq
func (push *Push) updateLastSeq(name string) {
	last, err := push.sequenceStore.LoadBlockLastSequence()
	if err != nil {
		chainlog.Error("LoadBlockLastSequence", "err", err)
		return
	}

	notify := push.tasks[string(calcPushKey(name))]
	notify.seqUpdateChan <- last
	chainlog.Debug("updateLastSeq", "last", last, "notify.seqUpdateChan", len(notify.seqUpdateChan))
}

// addTask   name    task,
func (push *Push) addTask(subscribe *types.PushSubscribeReq) {
	push.mu.Lock()
	defer push.mu.Unlock()
	keyStr := string(calcPushKey(subscribe.Name))
	push.tasks[keyStr] = &pushNotify{
		subscribe:      subscribe,
		seqUpdateChan:  make(chan int64, chanBufCap),
		closechan:      make(chan struct{}),
		status:         notRunning,
		postFail2Sleep: 0,
	}

	push.runTask(push.tasks[keyStr])
}

func trigeRun(run chan struct{}, sleep time.Duration, name string) {
	chainlog.Info("trigeRun", name, "name", "sleep", sleep, "run len", len(run))
	if sleep > 0 {
		time.Sleep(sleep)
	}
	go func() {
		run <- struct{}{}
	}()
}

func (push *Push) runTask(input *pushNotify) {
	//  goroutine
	push.updateLastSeq(input.subscribe.Name)

	go func(in *pushNotify) {
		var lastesBlockSeq int64
		var continueFailCount int32
		var err error

		subscribe := in.subscribe
		lastProcessedseq := push.getLastPushSeq(subscribe)

		atomic.StoreInt32(&in.status, running)

		runChan := make(chan struct{}, 10)
		pushMaxSeq := pushBlockMaxSeq
		if subscribe.Type == PushTxReceipt {
			pushMaxSeq = pushTxReceiptMaxSeq
		}

		chainlog.Debug("start push with info", "subscribe name", subscribe.Name, "Type", PushType(subscribe.Type).string())
		for {
			select {
			case <-runChan:
				if atomic.LoadInt32(&input.postFail2Sleep) > 0 {
					if postFail2SleepNew := atomic.AddInt32(&input.postFail2Sleep, -1); postFail2SleepNew > 0 {
						chainlog.Debug("wait another ticker for post fail", "postFail2Sleep", postFail2SleepNew, "name", in.subscribe.Name)
						trigeRun(runChan, time.Second, subscribe.Name)
						continue
					}
				}

			case lastestSeq := <-in.seqUpdateChan:
				chainlog.Debug("runTask recv:", "lastestSeq", lastestSeq, "subscribe name", subscribe.Name, "Type", PushType(subscribe.Type).string())
				//               ，    ，     sleep
				if atomic.LoadInt32(&input.postFail2Sleep) > 0 {
					if postFail2SleepNew := atomic.AddInt32(&input.postFail2Sleep, -1); postFail2SleepNew > 0 {
						chainlog.Debug("wait another ticker for post fail", "postFail2Sleep", postFail2SleepNew, "name", in.subscribe.Name)
						trigeRun(runChan, time.Second, subscribe.Name)
						continue
					}
				}
				//       sequence,                 ，         chan     sequence
				if lastesBlockSeq, err = push.sequenceStore.LoadBlockLastSequence(); err != nil {
					chainlog.Error("LoadBlockLastSequence", "err", err)
					return
				}

				//       ，      ，
				if lastProcessedseq >= lastesBlockSeq {
					continue
				}
				chainlog.Debug("another new block", "subscribe name", subscribe.Name, "Type", PushType(subscribe.Type).string(),
					"last push sequence", lastProcessedseq, "lastest sequence", lastesBlockSeq,
					"time second", time.Now().Second())
				//         ，              ，
				seqCount := pushMaxSeq
				if seqCount > int(lastesBlockSeq-lastProcessedseq) {
					seqCount = int(lastesBlockSeq - lastProcessedseq)
				}

				data, updateSeq, err := push.getPushData(subscribe, lastProcessedseq+1, seqCount, pushMaxSize)
				if err != nil {
					chainlog.Error("getPushData", "err", err, "seqCurrent", lastProcessedseq+1, "maxSeq", seqCount,
						"Name", subscribe.Name, "pushType:", PushType(subscribe.Type).string())
					continue
				}

				if data != nil {
					err = push.postService.PostData(subscribe, data, updateSeq)
					if err != nil {
						continueFailCount++
						chainlog.Error("postdata failed", "err", err, "lastProcessedseq", lastProcessedseq,
							"Name", subscribe.Name, "pushType:", PushType(subscribe.Type).string(), "continueFailCount", continueFailCount)
						if continueFailCount >= 3 {
							atomic.StoreInt32(&in.status, notRunning)
							chainlog.Error("postdata failed exceed 3 times", "Name", subscribe.Name, "in.status", atomic.LoadInt32(&in.status))

							pushWithStatus := &types.PushWithStatus{
								Push:   subscribe,
								Status: subscribeStatusNotActive,
							}

							key := calcPushKey(subscribe.Name)
							push.mu.Lock()
							delete(push.tasks, string(key))
							push.mu.Unlock()
							_ = push.store.SetSync(key, types.Encode(pushWithStatus))
							push.postwg.Done()
							return
						}
						//sleep 60s，  1s，  60 ，      ，
						atomic.StoreInt32(&input.postFail2Sleep, push.postFail2Sleep)
						trigeRun(runChan, time.Second, subscribe.Name)
						continue
					}
					_ = push.setLastPushSeq(subscribe.Name, updateSeq)
				}
				continueFailCount = 0
				lastProcessedseq = updateSeq
			case <-in.closechan:
				push.postwg.Done()
				chainlog.Info("getPushData", "push task closed for subscribe", subscribe.Name)
				return
			}
		}

	}(input)
	push.postwg.Add(1)
}

// UpdateSeq sequence
func (push *Push) UpdateSeq(seq int64) {
	push.mu.Lock()
	defer push.mu.Unlock()
	for _, notify := range push.tasks {
		//   seq（    block，    lock，        channel   ）
		if len(notify.seqUpdateChan) < chanBufCap {
			chainlog.Info("new block Update Seq notified", "subscribe", notify.subscribe.Name, "current sequence", seq, "length", len(notify.seqUpdateChan))
			notify.seqUpdateChan <- seq
		}
		chainlog.Info("new block UpdateSeq", "subscribe", notify.subscribe.Name, "current sequence", seq, "length", len(notify.seqUpdateChan))
	}
}

func (push *Push) getPushData(subscribe *types.PushSubscribeReq, startSeq int64, seqCount, maxSize int) ([]byte, int64, error) {
	if subscribe.Type == PushBlock {
		return push.getBlockSeqs(subscribe.Encode, startSeq, seqCount, maxSize)
	} else if subscribe.Type == PushBlockHeader {
		return push.getHeaderSeqs(subscribe.Encode, startSeq, seqCount, maxSize)
	}
	return push.getTxReceipts(subscribe, startSeq, seqCount, maxSize)
}

func (push *Push) getTxReceipts(subscribe *types.PushSubscribeReq, startSeq int64, seqCount, maxSize int) ([]byte, int64, error) {
	txReceipts := &types.TxReceipts4Subscribe{}
	totalSize := 0
	actualIterCount := 0
	for i := startSeq; i < startSeq+int64(seqCount); i++ {
		chainlog.Info("getTxReceipts", "startSeq:", i)
		seqdata, err := push.sequenceStore.GetBlockSequence(i)
		if err != nil {
			return nil, -1, err
		}
		detail, _, err := push.sequenceStore.LoadBlockBySequence(i)
		if err != nil {
			return nil, -1, err
		}

		txReceiptsPerBlk := &types.TxReceipts4SubscribePerBlk{}
		chainlog.Info("getTxReceipts", "height:", detail.Block.Height, "tx numbers:", len(detail.Block.Txs), "Receipts numbers:", len(detail.Receipts))
		for txIndex, tx := range detail.Block.Txs {
			if subscribe.Contract[string(tx.Execer)] {
				chainlog.Info("getTxReceipts", "txIndex:", txIndex)
				txReceiptsPerBlk.Tx = append(txReceiptsPerBlk.Tx, tx)
				txReceiptsPerBlk.ReceiptData = append(txReceiptsPerBlk.ReceiptData, detail.Receipts[txIndex])
				//txReceiptsPerBlk.KV = append(txReceiptsPerBlk.KV, detail.KV[txIndex])
			}
		}
		if len(txReceiptsPerBlk.Tx) > 0 {
			txReceiptsPerBlk.Height = detail.Block.Height
			txReceiptsPerBlk.BlockHash = detail.Block.Hash(push.cfg)
			txReceiptsPerBlk.ParentHash = detail.Block.ParentHash
			txReceiptsPerBlk.PreviousHash = []byte{}
			txReceiptsPerBlk.AddDelType = int32(seqdata.Type)
			txReceiptsPerBlk.SeqNum = i
		}
		size := types.Size(txReceiptsPerBlk)
		if len(txReceiptsPerBlk.Tx) > 0 && totalSize+size < maxSize {
			txReceipts.TxReceipts = append(txReceipts.TxReceipts, txReceiptsPerBlk)
			totalSize += size
			chainlog.Debug("get Tx Receipts subscribed for pushing", "Name", subscribe.Name, "contract:", subscribe.Contract,
				"height=", txReceiptsPerBlk.Height)
		} else if totalSize+size > maxSize {
			break
		}
		actualIterCount++
	}

	updateSeq := startSeq + int64(actualIterCount) - 1
	if len(txReceipts.TxReceipts) == 0 {
		return nil, updateSeq, nil
	}
	chainlog.Info("getTxReceipts", "updateSeq", updateSeq, "actualIterCount", actualIterCount)

	var postdata []byte
	var err error
	if subscribe.Encode == "json" {
		postdata, err = types.PBToJSON(txReceipts)
		if err != nil {
			return nil, -1, err
		}
	} else {
		postdata = types.Encode(txReceipts)
	}

	return postdata, updateSeq, nil
}

func (push *Push) getBlockDataBySeq(seq int64) (*types.BlockSeq, int, error) {
	seqdata, err := push.sequenceStore.GetBlockSequence(seq)
	if err != nil {
		return nil, 0, err
	}
	detail, blockSize, err := push.sequenceStore.LoadBlockBySequence(seq)
	if err != nil {
		return nil, 0, err
	}
	return &types.BlockSeq{Num: seq, Seq: seqdata, Detail: detail}, blockSize, nil
}

func (push *Push) getBlockSeqs(encode string, seq int64, seqCount, maxSize int) ([]byte, int64, error) {
	seqs := &types.BlockSeqs{}
	totalSize := 0
	for i := 0; i < seqCount; i++ {
		seq, size, err := push.getBlockDataBySeq(seq + int64(i))
		if err != nil {
			return nil, -1, err
		}
		if totalSize == 0 || totalSize+size < maxSize {
			seqs.Seqs = append(seqs.Seqs, seq)
			totalSize += size
		} else {
			break
		}
	}
	updateSeq := seqs.Seqs[0].Num + int64(len(seqs.Seqs)) - 1

	var postdata []byte
	var err error
	if encode == "json" {
		postdata, err = types.PBToJSON(seqs)
		if err != nil {
			return nil, -1, err
		}
	} else {
		postdata = types.Encode(seqs)
	}
	return postdata, updateSeq, nil
}

func (push *Push) getHeaderSeqs(encode string, seq int64, seqCount, maxSize int) ([]byte, int64, error) {
	seqs := &types.HeaderSeqs{}
	totalSize := 0
	for i := 0; i < seqCount; i++ {
		seq, size, err := push.getHeaderDataBySeq(seq + int64(i))
		if err != nil {
			return nil, -1, err
		}
		if totalSize == 0 || totalSize+size < maxSize {
			seqs.Seqs = append(seqs.Seqs, seq)
			totalSize += size
		} else {
			break
		}
	}
	updateSeq := seqs.Seqs[0].Num + int64(len(seqs.Seqs)) - 1

	var postdata []byte
	var err error

	if encode == "json" {
		postdata, err = types.PBToJSON(seqs)
		if err != nil {
			return nil, -1, err
		}
	} else {
		postdata = types.Encode(seqs)
	}
	return postdata, updateSeq, nil
}

func (push *Push) getHeaderDataBySeq(seq int64) (*types.HeaderSeq, int, error) {
	seqdata, err := push.sequenceStore.GetBlockSequence(seq)
	if err != nil {
		return nil, 0, err
	}
	header, err := push.sequenceStore.GetBlockHeaderByHash(seqdata.Hash)
	if err != nil {
		return nil, 0, err
	}
	return &types.HeaderSeq{Num: seq, Seq: seqdata, Header: header}, header.Size(), nil
}

// GetLastPushSeq Seq     0   ，                  -1
func (push *Push) getLastPushSeq(subscribe *types.PushSubscribeReq) int64 {
	seqbytes, err := push.store.GetKey(calcLastPushSeqNumKey(subscribe.Name))
	if seqbytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("getLastPushSeq", "error", err)
		}
		return -1
	}
	n, err := decodeHeight(seqbytes)
	if err != nil {
		return -1
	}
	chainlog.Info("getLastPushSeq", "name", subscribe.Name,
		"Contract:", subscribe.Contract, "num", n)

	return n
}

func (push *Push) setLastPushSeq(name string, num int64) error {
	return push.store.SetSync(calcLastPushSeqNumKey(name), types.Encode(&types.Int64{Data: num}))
}
