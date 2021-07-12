// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"encoding/hex"
	"errors"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	cty "github.com/D-PlatformOperatingSystem/dpos/system/dapp/coins/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

func init() {
	rand.Seed(types.Now().UnixNano())
}

// GetBalance
func (wallet *Wallet) GetBalance(addr string, execer string) (*types.Account, error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}
	return wallet.getBalance(addr, execer)
}

func (wallet *Wallet) getBalance(addr string, execer string) (*types.Account, error) {
	reqbalance := &types.ReqBalance{Addresses: []string{addr}, Execer: execer}
	reply, err := wallet.queryBalance(reqbalance)
	if err != nil {
		return nil, err
	}
	return reply[0], nil
}

// GetAllPrivKeys
func (wallet *Wallet) GetAllPrivKeys() ([]crypto.PrivKey, error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	return wallet.getAllPrivKeys()
}

func (wallet *Wallet) getAllPrivKeys() ([]crypto.PrivKey, error) {
	accounts, err := wallet.getWalletAccounts()
	if err != nil {
		return nil, err
	}

	ok, err := wallet.checkWalletStatus()
	if !ok && err != types.ErrOnlyTicketUnLocked {
		return nil, err
	}
	var privs []crypto.PrivKey
	for _, acc := range accounts {
		priv, err := wallet.getPrivKeyByAddr(acc.Addr)
		if err != nil {
			return nil, err
		}
		privs = append(privs, priv)
	}
	return privs, nil
}

// GetHeight
func (wallet *Wallet) GetHeight() int64 {
	if !wallet.isInited() {
		return 0
	}
	msg := wallet.client.NewMessage("blockchain", types.EventGetBlockHeight, nil)
	err := wallet.client.Send(msg, true)
	if err != nil {
		return 0
	}
	replyHeight, err := wallet.client.Wait(msg)
	h := replyHeight.GetData().(*types.ReplyBlockHeight).Height
	walletlog.Debug("getheight = ", "height", h)
	if err != nil {
		return 0
	}
	return h
}

func (wallet *Wallet) sendTransactionWait(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (err error) {
	hash, err := wallet.sendTransaction(payload, execer, priv, to)
	if err != nil {
		return err
	}
	txinfo := wallet.waitTx(hash)
	if txinfo.Receipt.Ty != types.ExecOk {
		return errors.New("sendTransactionWait error")
	}
	return nil
}

// SendTransaction
func (wallet *Wallet) SendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	return wallet.sendTransaction(payload, execer, priv, to)
}

func (wallet *Wallet) sendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error) {
	if to == "" {
		to = address.ExecAddress(string(execer))
	}
	tx := &types.Transaction{Execer: execer, Payload: types.Encode(payload), Fee: wallet.minFee, To: to}
	tx.Nonce = rand.Int63()
	proper, err := wallet.api.GetProperFee(nil)
	if err != nil {
		return nil, err
	}
	fee, err := tx.GetRealFee(proper.ProperFee)
	if err != nil {
		return nil, err
	}
	tx.Fee = fee
	tx.SetExpire(wallet.client.GetConfig(), time.Second*120)
	tx.Sign(int32(wallet.SignType), priv)
	reply, err := wallet.sendTx(tx)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		walletlog.Info("wallet sendTransaction", "err", string(reply.GetMsg()))
		return nil, errors.New(string(reply.GetMsg()))
	}
	return tx.Hash(), nil
}

func (wallet *Wallet) sendTx(tx *types.Transaction) (*types.Reply, error) {
	if wallet.client == nil {
		panic("client not bind message queue.")
	}
	return wallet.api.SendTx(tx)
}

// WaitTx
func (wallet *Wallet) WaitTx(hash []byte) *types.TransactionDetail {
	return wallet.waitTx(hash)
}

func (wallet *Wallet) waitTx(hash []byte) *types.TransactionDetail {
	i := 0
	for {
		if atomic.LoadInt32(&wallet.isclosed) == 1 {
			return nil
		}
		i++
		if i%100 == 0 {
			walletlog.Error("wait transaction timeout", "hash", hex.EncodeToString(hash))
			return nil
		}
		res, err := wallet.queryTx(hash)
		if err != nil {
			time.Sleep(time.Second)
		}
		if res != nil {
			return res
		}
	}
}

// WaitTxs
func (wallet *Wallet) WaitTxs(hashes [][]byte) (ret []*types.TransactionDetail) {
	return wallet.waitTxs(hashes)
}

func (wallet *Wallet) waitTxs(hashes [][]byte) (ret []*types.TransactionDetail) {
	for _, hash := range hashes {
		result := wallet.waitTx(hash)
		ret = append(ret, result)
	}
	return ret
}

func (wallet *Wallet) queryTx(hash []byte) (*types.TransactionDetail, error) {
	msg := wallet.client.NewMessage("blockchain", types.EventQueryTx, &types.ReqHash{Hash: hash})
	err := wallet.client.Send(msg, true)
	if err != nil {
		walletlog.Error("QueryTx", "Error", err.Error())
		return nil, err
	}
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.TransactionDetail), nil
}

// SendToAddress
func (wallet *Wallet) SendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	return wallet.sendToAddress(priv, addrto, amount, note, Istoken, tokenSymbol)
}

func (wallet *Wallet) createSendToAddress(addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.Transaction, error) {
	var tx *types.Transaction
	var isWithdraw = false
	if amount < 0 {
		amount = -amount
		isWithdraw = true
	}
	create := &types.CreateTx{
		To:          addrto,
		Amount:      amount,
		Note:        []byte(note),
		IsWithdraw:  isWithdraw,
		IsToken:     Istoken,
		TokenSymbol: tokenSymbol,
	}

	exec := cty.CoinsX
	//    ，token        ,     ，token
	//      ，
	if create.IsToken {
		exec = "token"
	}
	ety := types.LoadExecutorType(exec)
	if ety == nil {
		return nil, types.ErrActionNotSupport
	}
	tx, err := ety.AssertCreate(create)
	if err != nil {
		return nil, err
	}
	cfg := wallet.client.GetConfig()
	tx.SetExpire(cfg, time.Second*120)
	proper, err := wallet.api.GetProperFee(nil)
	if err != nil {
		return nil, err
	}
	fee, err := tx.GetRealFee(proper.ProperFee)
	if err != nil {
		return nil, err
	}
	tx.Fee = fee
	if tx.To == "" {
		tx.To = addrto
	}
	if len(tx.Execer) == 0 {
		tx.Execer = []byte(exec)
	}

	if cfg.IsPara() {
		tx.Execer = []byte(cfg.GetTitle() + string(tx.Execer))
		tx.To = address.ExecAddress(string(tx.Execer))
	}

	tx.Nonce = rand.Int63()
	return tx, nil
}

func (wallet *Wallet) sendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error) {
	tx, err := wallet.createSendToAddress(addrto, amount, note, Istoken, tokenSymbol)
	if err != nil {
		return nil, err
	}
	tx.Sign(int32(wallet.SignType), priv)

	reply, err := wallet.api.SendTx(tx)
	if err != nil {
		return nil, err
	}
	if !reply.GetIsOk() {
		return nil, errors.New(string(reply.GetMsg()))
	}
	var hash types.ReplyHash
	hash.Hash = tx.Hash()
	return &hash, nil
}

func (wallet *Wallet) queryBalance(in *types.ReqBalance) ([]*types.Account, error) {

	switch in.GetExecer() {
	case "coins":
		addrs := in.GetAddresses()
		var exaddrs []string
		for _, addr := range addrs {
			if err := address.CheckAddress(addr); err != nil {
				addr = address.ExecAddress(addr)
			}
			exaddrs = append(exaddrs, addr)
		}
		accounts, err := wallet.accountdb.LoadAccounts(wallet.api, exaddrs)
		if err != nil {
			walletlog.Error("GetBalance", "err", err.Error())
			return nil, err
		}
		return accounts, nil
	default:
		execaddress := address.ExecAddress(in.GetExecer())
		addrs := in.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {
			acc, err := wallet.accountdb.LoadExecAccountQueue(wallet.api, addr, execaddress)
			if err != nil {
				walletlog.Error("GetBalance", "err", err.Error())
				return nil, err
			}
			accounts = append(accounts, acc)
		}
		return accounts, nil
	}
}

func (wallet *Wallet) getMinerColdAddr(addr string) ([]string, error) {
	reqaddr := &types.ReqString{Data: addr}
	//ticket pos33       miner
	consensus := wallet.client.GetConfig().GetModuleConfig().Consensus.Name
	req := types.ChainExecutor{
		Driver:   consensus,
		FuncName: "MinerSourceList",
		Param:    types.Encode(reqaddr),
	}

	msg := wallet.client.NewMessage("exec", types.EventBlockChainQuery, &req)
	err := wallet.client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	reply := resp.GetData().(types.Message).(*types.ReplyStrings)
	return reply.Datas, nil
}

// IsCaughtUp
func (wallet *Wallet) IsCaughtUp() bool {
	if !wallet.isInited() {
		return false
	}
	if wallet.client == nil {
		panic("wallet client not bind message queue.")
	}
	reply, err := wallet.api.IsSync()
	if err != nil {
		return false
	}
	return reply.IsOk
}
