package mempool

import (
	"errors"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common"

	"github.com/D-PlatformOperatingSystem/dpos/util"

	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// CheckExpireValid          ，    false，     true
func (mem *Mempool) CheckExpireValid(msg *queue.Message) (bool, error) {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	if mem.header == nil {
		return false, types.ErrHeaderNotSet
	}
	tx := msg.GetData().(types.TxGroup).Tx()
	ok := mem.checkExpireValid(tx)
	if !ok {
		return ok, types.ErrTxExpire
	}
	return ok, nil
}

// checkTxListRemote
func (mem *Mempool) checkTxListRemote(txlist *types.ExecTxList) (*types.ReceiptCheckTxList, error) {
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	msg := mem.client.NewMessage("execs", types.EventCheckTx, txlist)
	err := mem.client.Send(msg, true)
	if err != nil {
		mlog.Error("execs closed", "err", err.Error())
		return nil, err
	}
	reply, err := mem.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	txList := reply.GetData().(*types.ReceiptCheckTxList)
	mem.client.FreeMessage(msg, reply)
	return txList, nil
}

func (mem *Mempool) checkExpireValid(tx *types.Transaction) bool {
	types.AssertConfig(mem.client)
	if tx.IsExpire(mem.client.GetConfig(), mem.header.GetHeight(), mem.header.GetBlockTime()) {
		return false
	}
	if tx.Expire > 1000000000 && tx.Expire < types.Now().Unix()+int64(time.Minute/time.Second) {
		return false
	}
	return true
}

// CheckTx
func (mem *Mempool) checkTx(msg *queue.Message) *queue.Message {
	tx := msg.GetData().(types.TxGroup).Tx()
	//
	if err := address.CheckAddress(tx.To); err != nil {
		msg.Data = types.ErrInvalidAddress
		return msg
	}
	//        mempool
	from := tx.From()
	if mem.TxNumOfAccount(from) >= mem.cfg.MaxTxNumPerAccount {
		msg.Data = types.ErrManyTx
		return msg
	}
	//
	valid, err := mem.CheckExpireValid(msg)
	if !valid {
		msg.Data = err
		return msg
	}
	return msg
}

// CheckTxs
func (mem *Mempool) checkTxs(msg *queue.Message) *queue.Message {
	//         nil
	if msg.GetData() == nil {
		msg.Data = types.ErrEmptyTx
		return msg
	}
	header := mem.GetHeader()
	txmsg := msg.GetData().(*types.Transaction)
	//
	tx := types.NewTransactionCache(txmsg)
	types.AssertConfig(mem.client)
	err := tx.Check(mem.client.GetConfig(), header.GetHeight(), mem.cfg.MinTxFeeRate, mem.cfg.MaxTxFee)
	if err != nil {
		msg.Data = err
		return msg
	}
	if mem.cfg.IsLevelFee {
		err = mem.checkLevelFee(tx)
		if err != nil {
			msg.Data = err
			return msg
		}
	}
	//                  ，            size
	//txmsg.ReCalcCacheHash()
	//  txgroup
	txs, err := tx.GetTxGroup()
	if err != nil {
		msg.Data = err
		return msg
	}
	msg.Data = tx
	//
	if txs == nil {
		return mem.checkTx(msg)
	}
	//txgroup
	for i := 0; i < len(txs.Txs); i++ {
		msgitem := mem.checkTx(&queue.Message{Data: txs.Txs[i]})
		if msgitem.Err() != nil {
			msg.Data = msgitem.Err()
			return msg
		}
	}
	return msg
}

// checkLevelFee
func (mem *Mempool) checkLevelFee(tx *types.TransactionCache) error {
	//  mempool
	feeRate := mem.getLevelFeeRate(mem.cfg.MinTxFeeRate, 0, 0)
	totalfee, err := tx.GetTotalFee(feeRate)
	if err != nil {
		return err
	}
	if tx.Fee < totalfee {
		return types.ErrTxFeeTooLow
	}
	return nil
}

//checkTxRemote           ，    Mempool，     goodChan，   Mempool     badChan
func (mem *Mempool) checkTxRemote(msg *queue.Message) *queue.Message {
	tx := msg.GetData().(types.TxGroup)
	lastheader := mem.GetHeader()

	//add check dup tx        /
	temtxlist := &types.ExecTxList{}
	txGroup, err := tx.GetTxGroup()
	if err != nil {
		msg.Data = err
		return msg
	}
	if txGroup == nil {
		temtxlist.Txs = append(temtxlist.Txs, tx.Tx())
	} else {
		temtxlist.Txs = append(temtxlist.Txs, txGroup.GetTxs()...)
	}
	temtxlist.Height = lastheader.Height
	newtxs, err := util.CheckDupTx(mem.client, temtxlist.Txs, temtxlist.Height)
	if err != nil {
		msg.Data = err
		return msg
	}
	if len(newtxs) != len(temtxlist.Txs) {
		msg.Data = types.ErrDupTx
		return msg
	}

	//exec            ，
	if !mem.cfg.DisableExecCheck {
		txlist := &types.ExecTxList{}
		txlist.Txs = append(txlist.Txs, tx.Tx())
		txlist.BlockTime = lastheader.BlockTime
		txlist.Height = lastheader.Height
		txlist.StateHash = lastheader.StateHash
		//       ，
		txlist.Difficulty = uint64(lastheader.Difficulty)
		txlist.IsMempool = true

		result, err := mem.checkTxListRemote(txlist)

		if err == nil && result.Errs[0] != "" {
			err = errors.New(result.Errs[0])
		}
		if err != nil {
			mlog.Error("checkTxRemote", "txHash", common.ToHex(tx.Tx().Hash()), "checkTxListRemoteErr", err)
			msg.Data = err
			return msg
		}
	}

	err = mem.PushTx(tx.Tx())
	if err != nil {
		mlog.Error("checkTxRemote", "push err", err)
		msg.Data = err
	}
	return msg
}
