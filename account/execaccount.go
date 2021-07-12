// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"github.com/D-PlatformOperatingSystem/dpos/client"
	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// LoadExecAccount Load exec account from address and exec
func (acc *DB) LoadExecAccount(addr, execaddr string) *types.Account {
	value, err := acc.db.Get(acc.execAccountKey(addr, execaddr))
	if err != nil {
		return &types.Account{Addr: addr}
	}
	var acc1 types.Account
	err = types.Decode(value, &acc1)
	if err != nil {
		panic(err) //
	}
	return &acc1
}

// LoadExecAccountQueue load exec account from statedb
func (acc *DB) LoadExecAccountQueue(api client.QueueProtocolAPI, addr, execaddr string) (*types.Account, error) {
	header, err := api.GetLastHeader()
	if err != nil {
		return nil, err
	}
	return acc.LoadExecAccountHistoryQueue(api, addr, execaddr, header.GetStateHash())
}

// SaveExecAccount save exec account data to db
func (acc *DB) SaveExecAccount(execaddr string, acc1 *types.Account) {
	set := acc.GetExecKVSet(execaddr, acc1)
	for i := 0; i < len(set); i++ {
		err := acc.db.Set(set[i].GetKey(), set[i].Value)
		if err != nil {
			panic(err)
		}
	}
}

// GetExecKVSet               kv
func (acc *DB) GetExecKVSet(execaddr string, acc1 *types.Account) (kvset []*types.KeyValue) {
	value := types.Encode(acc1)
	kvset = append(kvset, &types.KeyValue{
		Key:   acc.execAccountKey(acc1.Addr, execaddr),
		Value: value,
	})
	return kvset
}

func (acc *DB) execAccountKey(address, execaddr string) (key []byte) {
	key = make([]byte, 0, len(acc.execAccountKeyPerfix)+len(execaddr)+len(address)+1)
	key = append(key, acc.execAccountKeyPerfix...)
	key = append(key, []byte(execaddr)...)
	key = append(key, []byte(":")...)
	key = append(key, []byte(address)...)
	return key
}

// TransferToExec transfer coins from address to exec address
func (acc *DB) TransferToExec(from, to string, amount int64) (*types.Receipt, error) {
	receipt, err := acc.Transfer(from, to, amount)
	if err != nil {
		return nil, err
	}
	receipt2, err := acc.ExecDeposit(from, to, amount)
	if err != nil {
		//
		panic(err)
	}
	return acc.mergeReceipt(receipt, receipt2), nil
}

// TransferWithdraw
func (acc *DB) TransferWithdraw(from, to string, amount int64) (*types.Receipt, error) {
	//
	if err := acc.CheckTransfer(to, from, amount); err != nil {
		return nil, err
	}
	receipt, err := acc.ExecWithdraw(to, from, amount)
	if err != nil {
		return nil, err
	}
	//    transfer
	receipt2, err := acc.Transfer(to, from, amount)
	if err != nil {
		panic(err) // withdraw
	}
	return acc.mergeReceipt(receipt, receipt2), nil
}

//ExecFrozen       ，      Deposit     ，
func (acc *DB) ExecFrozen(addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadExecAccount(addr, execaddr)
	if acc1.Balance-amount < 0 {
		alog.Error("ExecFrozen", "balance", acc1.Balance, "amount", amount)
		return nil, types.ErrNoBalance
	}
	copyacc := *acc1
	acc1.Balance -= amount
	acc1.Frozen += amount
	receiptBalance := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyacc,
		Current:  acc1,
	}
	acc.SaveExecAccount(execaddr, acc1)
	ty := int32(types.TyLogExecFrozen)
	return acc.execReceipt(ty, acc1, receiptBalance), nil
}

// ExecActive
func (acc *DB) ExecActive(addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadExecAccount(addr, execaddr)
	if acc1.Frozen-amount < 0 {
		return nil, types.ErrNoBalance
	}
	copyacc := *acc1
	acc1.Balance += amount
	acc1.Frozen -= amount
	receiptBalance := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyacc,
		Current:  acc1,
	}
	acc.SaveExecAccount(execaddr, acc1)
	ty := int32(types.TyLogExecActive)
	return acc.execReceipt(ty, acc1, receiptBalance), nil
}

// ExecTransfer
func (acc *DB) ExecTransfer(from, to, execaddr string, amount int64) (*types.Receipt, error) {
	if from == to {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	accFrom := acc.LoadExecAccount(from, execaddr)
	accTo := acc.LoadExecAccount(to, execaddr)

	if accFrom.GetBalance()-amount < 0 {
		return nil, types.ErrNoBalance
	}
	copyaccFrom := *accFrom
	copyaccTo := *accTo

	accFrom.Balance -= amount
	accTo.Balance += amount

	receiptBalanceFrom := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyaccFrom,
		Current:  accFrom,
	}
	receiptBalanceTo := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyaccTo,
		Current:  accTo,
	}

	acc.SaveExecAccount(execaddr, accFrom)
	acc.SaveExecAccount(execaddr, accTo)
	return acc.execReceipt2(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
}

// ExecTransferFrozen            ，
func (acc *DB) ExecTransferFrozen(from, to, execaddr string, amount int64) (*types.Receipt, error) {
	if from == to {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	accFrom := acc.LoadExecAccount(from, execaddr)
	accTo := acc.LoadExecAccount(to, execaddr)
	b := accFrom.GetFrozen() - amount
	if b < 0 {
		return nil, types.ErrNoBalance
	}
	copyaccFrom := *accFrom
	copyaccTo := *accTo

	accFrom.Frozen -= amount
	accTo.Balance += amount

	receiptBalanceFrom := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyaccFrom,
		Current:  accFrom,
	}
	receiptBalanceTo := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyaccTo,
		Current:  accTo,
	}

	acc.SaveExecAccount(execaddr, accFrom)
	acc.SaveExecAccount(execaddr, accTo)
	return acc.execReceipt2(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
}

// ExecAddress
func (acc *DB) ExecAddress(name string) string {
	return address.ExecAddress(name)
}

// ExecDepositFrozen     coins      ，
func (acc *DB) ExecDepositFrozen(addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	//issue coins to exec addr
	receipt1, err := acc.ExecIssueCoins(execaddr, amount)
	if err != nil {
		return nil, err
	}
	receipt2, err := acc.execDepositFrozen(addr, execaddr, amount)
	if err != nil {
		return nil, err
	}
	return acc.mergeReceipt(receipt1, receipt2), nil
}

// ExecIssueCoins   coins
func (acc *DB) ExecIssueCoins(execaddr string, amount int64) (*types.Receipt, error) {
	cfg := acc.cfg
	//
	allow := false
	for _, exec := range cfg.GetMinerExecs() {
		//
		if acc.ExecAddress(cfg.ExecName(exec)) == execaddr {
			allow = true
			break
		}
		//
		if cfg.GetFundAddr() == execaddr {
			allow = true
			break
		}
	}
	if !allow {
		return nil, types.ErrNotAllowDeposit
	}
	receipt, err := acc.depositBalance(execaddr, amount)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (acc *DB) execDepositFrozen(addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadExecAccount(addr, execaddr)
	copyacc := *acc1
	acc1.Frozen += amount
	receiptBalance := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyacc,
		Current:  acc1,
	}
	acc.SaveExecAccount(execaddr, acc1)
	ty := int32(types.TyLogExecDeposit)
	return acc.execReceipt(ty, acc1, receiptBalance), nil
}

// ExecDeposit     addr execaddr
func (acc *DB) ExecDeposit(addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadExecAccount(addr, execaddr)
	copyacc := *acc1
	acc1.Balance += amount
	receiptBalance := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyacc,
		Current:  acc1,
	}
	//alog.Debug("execDeposit", "addr", addr, "execaddr", execaddr, "account", acc)
	acc.SaveExecAccount(execaddr, acc1)
	ty := int32(types.TyLogExecDeposit)
	return acc.execReceipt(ty, acc1, receiptBalance), nil
}

// ExecWithdraw
func (acc *DB) ExecWithdraw(execaddr, addr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadExecAccount(addr, execaddr)
	if acc1.Balance-amount < 0 {
		return nil, types.ErrNoBalance
	}
	copyacc := *acc1
	acc1.Balance -= amount
	receiptBalance := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyacc,
		Current:  acc1,
	}
	acc.SaveExecAccount(execaddr, acc1)
	ty := int32(types.TyLogExecWithdraw)
	return acc.execReceipt(ty, acc1, receiptBalance), nil
}

func (acc *DB) execReceipt(ty int32, acc1 *types.Account, r *types.ReceiptExecAccountTransfer) *types.Receipt {
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(r),
	}
	kv := acc.GetExecKVSet(r.ExecAddr, acc1)
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1},
	}
}

func (acc *DB) execReceipt2(acc1, acc2 *types.Account, r1, r2 *types.ReceiptExecAccountTransfer) *types.Receipt {
	ty := int32(types.TyLogExecTransfer)
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(r1),
	}
	log2 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(r2),
	}
	kv := acc.GetExecKVSet(r1.ExecAddr, acc1)
	kv = append(kv, acc.GetExecKVSet(r2.ExecAddr, acc2)...)
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1, log2},
	}
}

func (acc *DB) mergeReceipt(receipt, receipt2 *types.Receipt) *types.Receipt {
	receipt.Logs = append(receipt.Logs, receipt2.Logs...)
	receipt.KV = append(receipt.KV, receipt2.KV...)
	return receipt
}

// LoadExecAccountHistoryQueue      statehash,
func (acc *DB) LoadExecAccountHistoryQueue(api client.QueueProtocolAPI, addr, execaddr string, stateHash []byte) (*types.Account, error) {
	get := types.StoreGet{StateHash: stateHash}
	get.Keys = append(get.Keys, acc.execAccountKey(addr, execaddr))
	values, err := api.StoreGet(&get)
	if err != nil {
		return nil, err
	}
	if len(values.Values) <= 0 {
		return nil, types.ErrNotFound
	}
	value := values.Values[0]
	if value == nil {
		return &types.Account{Addr: addr}, nil
	}

	var acc1 types.Account
	err = types.Decode(value, &acc1)
	if err != nil {
		return nil, err
	}

	return &acc1, nil
}
