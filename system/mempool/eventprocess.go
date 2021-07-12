package mempool

import (
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

func (mem *Mempool) reply() {
	defer mlog.Info("piple line quit")
	defer mem.wg.Done()
	for m := range mem.out {
		if m.Err() != nil {
			m.Reply(mem.client.NewMessage("rpc", types.EventReply,
				&types.Reply{IsOk: false, Msg: []byte(m.Err().Error())}))
		} else { //TODO, rpc p2p        ， rpc      ，p2p
			mem.sendTxToP2P(m.GetData().(types.TxGroup).Tx())
			m.Reply(mem.client.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: true, Msg: nil}))
		}
	}
}

func (mem *Mempool) pipeLine() <-chan *queue.Message {
	//check sign
	step1 := func(data *queue.Message) *queue.Message {
		if data.Err() != nil {
			return data
		}
		return mem.checkSign(data)
	}
	chs := make([]<-chan *queue.Message, processNum)
	for i := 0; i < processNum; i++ {
		chs[i] = step(mem.done, mem.in, step1)
	}
	out1 := merge(mem.done, chs)

	//checktx remote
	step2 := func(data *queue.Message) *queue.Message {
		if data.Err() != nil {
			return data
		}
		return mem.checkTxRemote(data)
	}
	chs2 := make([]<-chan *queue.Message, processNum)
	for i := 0; i < processNum; i++ {
		chs2[i] = step(mem.done, out1, step2)
	}
	return merge(mem.done, chs2)
}

//
func (mem *Mempool) eventProcess() {
	defer mem.wg.Done()
	defer close(mem.in)
	//event process
	mem.out = mem.pipeLine()
	mlog.Info("mempool piple line start")
	mem.wg.Add(1)
	go mem.reply()

	for msg := range mem.client.Recv() {
		//NOTE:     msg        ，          msg      ，
		msgName := types.GetEventName(int(msg.Ty))
		mlog.Debug("mempool recv", "msgid", msg.ID, "msg", msgName)
		beg := types.Now()
		switch msg.Ty {
		case types.EventTx:
			mem.eventTx(msg)
		case types.EventGetMempool:
			//     EventGetMempool：  mempool
			mem.eventGetMempool(msg)
		case types.EventTxList:
			//     EventTxList：  mempool
			mem.eventTxList(msg)
		case types.EventDelTxList:
			//     EventDelTxList：  mempool       ，       mempool
			mem.eventDelTxList(msg)
		case types.EventAddBlock:
			//     EventAddBlock：           mempool
			mem.eventAddBlock(msg)
		case types.EventGetMempoolSize:
			//     EventGetMempoolSize：  mempool
			mem.eventGetMempoolSize(msg)
		case types.EventGetLastMempool:
			//     EventGetLastMempool：         mempool
			mem.eventGetLastMempool(msg)
		case types.EventDelBlock:
			//     ，           mempool
			mem.eventDelBlock(msg)
		case types.EventGetAddrTxs:
			//   mempool     （ ）
			mem.eventGetAddrTxs(msg)
		case types.EventGetProperFee:
			//
			mem.eventGetProperFee(msg)
			//     EventTxListByHash：  hash     tx
		case types.EventTxListByHash:
			mem.eventTxListByHash(msg)
		case types.EventCheckTxsExist:
			mem.eventCheckTxsExist(msg)
		default:
		}
		mlog.Debug("mempool", "cost", types.Since(beg), "msg", msgName)
	}
}

//EventTx        mempool
func (mem *Mempool) eventTx(msg *queue.Message) {
	if !mem.getSync() {
		msg.Reply(mem.client.NewMessage("", types.EventReply, &types.Reply{Msg: []byte(types.ErrNotSync.Error())}))
		mlog.Debug("wrong tx", "err", types.ErrNotSync.Error())
	} else {
		checkedMsg := mem.checkTxs(msg)
		select {
		case mem.in <- checkedMsg:
		case <-mem.done:
		}
	}
}

// EventGetMempool   Mempool
func (mem *Mempool) eventGetMempool(msg *queue.Message) {
	var isAll bool
	if msg.GetData() == nil {
		isAll = false
	} else {
		isAll = msg.GetData().(*types.ReqGetMempool).GetIsAll()
	}
	msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
		&types.ReplyTxList{Txs: mem.filterTxList(0, nil, isAll)}))
}

// EventDelTxList   Mempool       ，       Mempool
func (mem *Mempool) eventDelTxList(msg *queue.Message) {
	hashList := msg.GetData().(*types.TxHashList)
	if len(hashList.GetHashes()) == 0 {
		msg.ReplyErr("EventDelTxList", types.ErrSize)
	} else {
		err := mem.RemoveTxs(hashList)
		msg.ReplyErr("EventDelTxList", err)
	}
}

// EventTxList   mempool
func (mem *Mempool) eventTxList(msg *queue.Message) {
	hashList := msg.GetData().(*types.TxHashList)
	if hashList.Count <= 0 {
		msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, types.ErrSize))
		mlog.Error("not an valid size", "msg", msg)
	} else {
		txList := mem.getTxList(hashList)
		msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, &types.ReplyTxList{Txs: txList}))
	}
}

// EventAddBlock            mempool
func (mem *Mempool) eventAddBlock(msg *queue.Message) {
	block := msg.GetData().(*types.BlockDetail).Block
	if block.Height > mem.Height() || (block.Height == 0 && mem.Height() == 0) {
		header := &types.Header{}
		header.BlockTime = block.BlockTime
		header.Height = block.Height
		header.StateHash = block.StateHash
		mem.setHeader(header)
	}
	mem.RemoveTxsOfBlock(block)
}

// EventGetMempoolSize   mempool
func (mem *Mempool) eventGetMempoolSize(msg *queue.Message) {
	memSize := int64(mem.Size())
	msg.Reply(mem.client.NewMessage("rpc", types.EventMempoolSize,
		&types.MempoolSize{Size: memSize}))
}

// EventGetLastMempool          mempool
func (mem *Mempool) eventGetLastMempool(msg *queue.Message) {
	txList := mem.GetLatestTx()
	msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
		&types.ReplyTxList{Txs: txList}))
}

// EventDelBlock     ，           mempool
func (mem *Mempool) eventDelBlock(msg *queue.Message) {
	block := msg.GetData().(*types.BlockDetail).Block
	if block.Height != mem.GetHeader().GetHeight() {
		return
	}
	lastHeader, err := mem.GetLastHeader()
	if err != nil {
		mlog.Error(err.Error())
		return
	}
	h := lastHeader.(*queue.Message).Data.(*types.Header)
	mem.setHeader(h)
	mem.delBlock(block)
}

// eventGetAddrTxs   mempool     （ ）
func (mem *Mempool) eventGetAddrTxs(msg *queue.Message) {
	addrs := msg.GetData().(*types.ReqAddrs)
	txlist := mem.GetAccTxs(addrs)
	msg.Reply(mem.client.NewMessage("", types.EventReplyAddrTxs, txlist))
}

// eventGetProperFee
func (mem *Mempool) eventGetProperFee(msg *queue.Message) {
	req, _ := msg.GetData().(*types.ReqProperFee)
	properFee := mem.GetProperFeeRate(req)
	msg.Reply(mem.client.NewMessage("rpc", types.EventReplyProperFee,
		&types.ReplyProperFee{ProperFee: properFee}))
}

func (mem *Mempool) checkSign(data *queue.Message) *queue.Message {
	tx, ok := data.GetData().(types.TxGroup)
	if ok && tx.CheckSign() {
		return data
	}
	mlog.Error("wrong tx", "err", types.ErrSign)
	data.Data = types.ErrSign
	return data
}

// eventTxListByHash   hash  tx
func (mem *Mempool) eventTxListByHash(msg *queue.Message) {
	shashList := msg.GetData().(*types.ReqTxHashList)
	replytxList := mem.getTxListByHash(shashList)
	msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, replytxList))
}

// eventCheckTxsExist
func (mem *Mempool) eventCheckTxsExist(msg *queue.Message) {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	reply := &types.ReplyCheckTxsExist{}
	//      mempool       ，
	if mem.sync {
		hashes := msg.GetData().(*types.ReqCheckTxsExist).TxHashes
		reply.ExistFlags = make([]bool, len(hashes))
		for i, hash := range hashes {
			reply.ExistFlags[i] = mem.cache.Exist(types.Bytes2Str(hash))
			if reply.ExistFlags[i] {
				reply.ExistCount++
			}
		}
	}
	msg.Reply(mem.client.NewMessage("", types.EventReply, reply))
}
