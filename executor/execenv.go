// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"bytes"

	"github.com/D-PlatformOperatingSystem/dpos/account"
	"github.com/D-PlatformOperatingSystem/dpos/client"
	"github.com/D-PlatformOperatingSystem/dpos/client/api"
	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	drivers "github.com/D-PlatformOperatingSystem/dpos/system/dapp"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/golang/protobuf/proto"
)

//    -> db
type executor struct {
	stateDB      dbm.KV
	localDB      dbm.KVDB
	coinsAccount *account.DB
	ctx          *executorCtx
	height       int64
	blocktime    int64
	//         ，
	difficulty uint64
	txs        []*types.Transaction
	api        client.QueueProtocolAPI
	gcli       types.DplatformOSClient
	execapi    api.ExecutorAPI
	receipts   []*types.ReceiptData
	//
	driverCache map[string]drivers.Driver
	//         ，        driver  ，    load
	currTxIdx  int
	currExecTx *types.Transaction
	currDriver drivers.Driver
	cfg        *types.DplatformOSConfig
	exec       *Executor
}

type executorCtx struct {
	stateHash  []byte
	height     int64
	blocktime  int64
	difficulty uint64
	parentHash []byte
	mainHash   []byte
	mainHeight int64
}

func newExecutor(ctx *executorCtx, exec *Executor, localdb dbm.KVDB, txs []*types.Transaction, receipts []*types.ReceiptData) *executor {
	client := exec.client
	types.AssertConfig(client)
	cfg := client.GetConfig()
	enableMVCC := exec.pluginEnable["mvcc"]
	opt := &StateDBOption{EnableMVCC: enableMVCC, Height: ctx.height}

	e := &executor{
		stateDB:      NewStateDB(client, ctx.stateHash, localdb, opt),
		localDB:      localdb,
		coinsAccount: account.NewCoinsAccount(cfg),
		height:       ctx.height,
		blocktime:    ctx.blocktime,
		difficulty:   ctx.difficulty,
		ctx:          ctx,
		txs:          txs,
		receipts:     receipts,
		api:          exec.qclient,
		gcli:         exec.grpccli,
		driverCache:  make(map[string]drivers.Driver),
		currTxIdx:    -1,
		cfg:          cfg,
		exec:         exec,
	}
	e.coinsAccount.SetDB(e.stateDB)
	return e
}

func (e *executor) enableMVCC(hash []byte) {
	e.stateDB.(*StateDB).enableMVCC(hash)
}

// AddMVCC convert key value to mvcc kv data
func AddMVCC(db dbm.KVDB, detail *types.BlockDetail) (kvlist []*types.KeyValue) {
	kvs := detail.KV
	hash := detail.Block.StateHash
	mvcc := dbm.NewSimpleMVCC(db)
	//
	kvlist, err := mvcc.AddMVCC(kvs, hash, detail.PrevStatusHash, detail.Block.Height)
	if err != nil {
		panic(err)
	}
	return kvlist
}

// DelMVCC convert key value to mvcc kv data
func DelMVCC(db dbm.KVDB, detail *types.BlockDetail) (kvlist []*types.KeyValue) {
	hash := detail.Block.StateHash
	mvcc := dbm.NewSimpleMVCC(db)
	kvlist, err := mvcc.DelMVCC(hash, detail.Block.Height, true)
	if err != nil {
		panic(err)
	}
	return kvlist
}

//         ：
//1.     ：   coin
//2.            ：              coin
func (e *executor) processFee(tx *types.Transaction) (*types.Receipt, error) {
	from := tx.From()
	accFrom := e.coinsAccount.LoadAccount(from)
	if accFrom.GetBalance()-tx.Fee >= 0 {
		copyfrom := *accFrom
		accFrom.Balance = accFrom.GetBalance() - tx.Fee
		receiptBalance := &types.ReceiptAccountTransfer{Prev: &copyfrom, Current: accFrom}
		set := e.coinsAccount.GetKVSet(accFrom)
		e.coinsAccount.SaveKVSet(set)
		return e.cutFeeReceipt(set, receiptBalance), nil
	}
	return nil, types.ErrNoBalance
}

func (e *executor) cutFeeReceipt(kvset []*types.KeyValue, receiptBalance proto.Message) *types.Receipt {
	feelog := &types.ReceiptLog{Ty: types.TyLogFee, Log: types.Encode(receiptBalance)}
	return &types.Receipt{
		Ty:   types.ExecPack,
		KV:   kvset,
		Logs: append([]*types.ReceiptLog{}, feelog),
	}
}

func (e *executor) getRealExecName(tx *types.Transaction, index int) []byte {
	exec := e.loadDriver(tx, index)
	realexec := exec.GetDriverName()
	var execer []byte
	if realexec != "none" {
		execer = []byte(realexec)
	} else {
		execer = tx.Execer
	}
	return execer
}

func (e *executor) checkTx(tx *types.Transaction, index int) error {

	if e.height > 0 && e.blocktime > 0 && tx.IsExpire(e.cfg, e.height, e.blocktime) {
		//
		return types.ErrTxExpire
	}
	if err := tx.Check(e.cfg, e.height, e.cfg.GetMinTxFeeRate(), e.cfg.GetMaxTxFee()); err != nil {
		return err
	}
	//
	//       name,
	if !types.IsAllowExecName(e.getRealExecName(tx, index), tx.Execer) {
		elog.Error("checkTx execNameNotAllow", "realname", string(e.getRealExecName(tx, index)), "exec", string(tx.Execer))
		return types.ErrExecNameNotAllow
	}
	return nil
}

func (e *executor) setEnv(exec drivers.Driver) {
	exec.SetAPI(e.api)
	//       coins account
	exec.SetCoinsAccount(e.coinsAccount)
	exec.SetStateDB(e.stateDB)
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime, e.difficulty)
	exec.SetBlockInfo(e.ctx.parentHash, e.ctx.mainHash, e.ctx.mainHeight)
	exec.SetExecutorAPI(e.api, e.gcli)
	e.execapi = exec.GetExecutorAPI()
	exec.SetTxs(e.txs)
	exec.SetReceipt(e.receipts)
}

func (e *executor) checkTxGroup(txgroup *types.Transactions, index int) error {

	if e.height > 0 && e.blocktime > 0 && txgroup.IsExpire(e.cfg, e.height, e.blocktime) {
		//
		return types.ErrTxExpire
	}
	if err := txgroup.Check(e.cfg, e.height, e.cfg.GetMinTxFeeRate(), e.cfg.GetMaxTxFee()); err != nil {
		return err
	}
	return nil
}

func (e *executor) execCheckTx(tx *types.Transaction, index int) error {
	//
	err := e.checkTx(tx, index)
	if err != nil {
		return err
	}
	//
	if err := address.CheckAddress(tx.To); err != nil {
		return err
	}
	var exec drivers.Driver

	//    none driver       TODO:        pool
	if types.Bytes2Str(tx.Execer) == "none" {
		exec = e.getNoneDriver()
		defer e.freeNoneDriver(exec)
	} else {
		exec = e.loadDriver(tx, index)
	}
	//
	if !exec.IsFree() && e.cfg.GetMinTxFeeRate() > 0 {
		from := tx.From()
		accFrom := e.coinsAccount.LoadAccount(from)

		//
		if accFrom.GetBalance() < tx.GetTxFee() {
			elog.Error("execCheckTx", "ispara", e.cfg.IsPara(), "exec", string(tx.Execer), "Balance", accFrom.GetBalance(), "TxFee", tx.GetTxFee())
			return types.ErrNoBalance
		}

		if accFrom.GetBalance() < e.cfg.GInt("MinBalanceTransfer") {
			elog.Error("execCheckTx", "ispara", e.cfg.IsPara(), "exec", string(tx.Execer), "nonce", tx.Nonce, "Balance", accFrom.GetBalance())
			return types.ErrBalanceLessThanTenTimesFee
		}

	}
	return exec.CheckTx(tx, index)
}

// Exec base exec func
func (e *executor) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	exec := e.loadDriver(tx, index)
	//to
	if err := drivers.CheckAddress(e.cfg, tx.GetRealToAddr(), e.height); err != nil {
		return nil, err
	}
	if e.localDB != nil && e.cfg.IsFork(e.height, "ForkLocalDBAccess") {
		e.localDB.(*LocalDB).DisableWrite()
		if exec.ExecutorOrder() != drivers.ExecLocalSameTime {
			e.localDB.(*LocalDB).DisableRead()
		}
		defer func() {
			e.localDB.(*LocalDB).EnableWrite()
			if exec.ExecutorOrder() != drivers.ExecLocalSameTime {
				e.localDB.(*LocalDB).EnableRead()
			}
		}()
	}
	//       CheckTx
	if err := exec.CheckTx(tx, index); err != nil {
		return nil, err
	}
	r, err := exec.Exec(tx, index)
	return r, err
}

func (e *executor) execLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriver(tx, index)
	return exec.ExecLocal(tx, r, index)
}

func (e *executor) execDelLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriver(tx, index)
	return exec.ExecDelLocal(tx, r, index)
}

func (e *executor) getNoneDriver() drivers.Driver {
	none := e.exec.noneDriverPool.Get().(drivers.Driver)
	e.setEnv(none)
	return none
}

func (e *executor) freeNoneDriver(none drivers.Driver) {
	e.exec.noneDriverPool.Put(none)
}

//   none
func (e *executor) loadNoneDriver() drivers.Driver {
	none, ok := e.driverCache["none"]
	var err error
	if !ok {
		none, err = drivers.LoadDriverWithClient(e.api, "none", 0)
		if err != nil {
			panic(err)
		}
		e.driverCache["none"] = none
	}
	return none
}

// loadDriver
//                  ，
//      tx，index，        ，    ，    ，
func (e *executor) loadDriver(tx *types.Transaction, index int) (c drivers.Driver) {

	//    index    ，
	if e.currExecTx == tx && e.currTxIdx == index {
		return e.currDriver
	}
	var err error
	name := types.Bytes2Str(tx.Execer)
	driver, ok := e.driverCache[name]
	isFork := e.cfg.IsFork(e.height, "ForkCacheDriver")

	if !ok {
		driver, err = drivers.LoadDriverWithClient(e.api, name, e.height)
		if err != nil {
			driver = e.loadNoneDriver()
		}
		e.driverCache[name] = driver
	}

	//fork  ，                  Allow  ，               allow
	//fork  ，            Allow
	if !ok || isFork {
		driver.SetEnv(e.height, 0, 0)
		err = driver.Allow(tx, index)
	}

	// allow    ，    none
	if err != nil {
		driver = e.loadNoneDriver()
		//fork  ，cache       allow   ，          ，                   none
		//fork  ，cache      Execer     driver
		//       fork       cache    ，            ，      allow   ，
		//  fork         cache
		//     ，cache                ，   driver    ，        allow
		if !isFork {
			e.driverCache[name] = driver
		}
	} else {
		driver.SetName(types.Bytes2Str(types.GetRealExecName(tx.Execer)))
		driver.SetCurrentExecName(name)
	}
	e.setEnv(driver)

	//     ，         ，        ，         index
	if e.currExecTx != tx && e.currTxIdx != index {
		e.currExecTx = tx
		e.currTxIdx = index
		e.currDriver = driver
	}
	return driver
}

func (e *executor) execTxGroup(txs []*types.Transaction, index int) ([]*types.Receipt, error) {
	txgroup := &types.Transactions{Txs: txs}
	err := e.checkTxGroup(txgroup, index)
	if err != nil {
		return nil, err
	}
	feelog, err := e.execFee(txs[0], index)
	if err != nil {
		return nil, err
	}
	//        ，        thread
	//        ，
	rollbackLog := copyReceipt(feelog)
	e.begin()
	receipts := make([]*types.Receipt, len(txs))
	for i := 1; i < len(txs); i++ {
		receipts[i] = &types.Receipt{Ty: types.ExecPack}
	}
	receipts[0], err = e.execTxOne(feelog, txs[0], index)
	if err != nil {
		//      ，
		if api.IsAPIEnvError(err) {
			return nil, err
		}
		//
		if e.cfg.IsFork(e.height, "ForkExecRollback") {
			e.rollback()
		}
		return receipts, nil
	}
	for i := 1; i < len(txs); i++ {
		//          ，
		receipts[i], err = e.execTxOne(receipts[i], txs[i], index+i)
		if err != nil {
			//reset other exec , and break!
			if api.IsAPIEnvError(err) {
				return nil, err
			}
			for k := 1; k < i; k++ {
				receipts[k] = &types.Receipt{Ty: types.ExecPack}
			}
			//  txs[0]
			if e.cfg.IsFork(e.height, "ForkResetTx0") {
				receipts[0] = rollbackLog
			}
			//
			e.rollback()
			return receipts, nil
		}
	}
	err = e.commit()
	if err != nil {
		return nil, err
	}
	return receipts, nil
}

func (e *executor) loadFlag(key []byte) (int64, error) {
	flag := &types.Int64{}
	flagBytes, err := e.localDB.Get(key)
	if err == nil {
		err = types.Decode(flagBytes, flag)
		if err != nil {
			return 0, err
		}
		return flag.GetData(), nil
	} else if err == types.ErrNotFound {
		return 0, nil
	}
	return 0, err
}

func (e *executor) execFee(tx *types.Transaction, index int) (*types.Receipt, error) {
	feelog := &types.Receipt{Ty: types.ExecPack}
	execer := string(tx.Execer)
	ex := e.loadDriver(tx, index)
	//         pubkey   ，            ,  checkTx
	//  checkTx
	if bytes.Equal(address.ExecPubKey(execer), tx.GetSignature().GetPubkey()) {
		err := ex.CheckTx(tx, index)
		if err != nil {
			return nil, err
		}
	}
	var err error
	//         0
	if !e.cfg.IsPara() && e.cfg.GetMinTxFeeRate() > 0 && !ex.IsFree() {
		feelog, err = e.processFee(tx)
		if err != nil {
			return nil, err
		}
	}
	return feelog, nil
}

func copyReceipt(feelog *types.Receipt) *types.Receipt {
	receipt := types.Receipt{}
	receipt = *feelog
	receipt.KV = make([]*types.KeyValue, len(feelog.KV))
	copy(receipt.KV, feelog.KV)
	receipt.Logs = make([]*types.ReceiptLog, len(feelog.Logs))
	copy(receipt.Logs, feelog.Logs)
	return &receipt
}

func (e *executor) execTxOne(feelog *types.Receipt, tx *types.Transaction, index int) (*types.Receipt, error) {
	//   pack   ，    index
	e.startTx()
	receipt, err := e.Exec(tx, index)
	if err != nil {
		elog.Error("exec tx error = ", "err", err, "exec", string(tx.Execer), "action", tx.ActionName())
		//add error log
		errlog := &types.ReceiptLog{Ty: types.TyLogErr, Log: []byte(err.Error())}
		feelog.Logs = append(feelog.Logs, errlog)
		return feelog, err
	}
	//    receipt，         ，
	//        :
	//1. statedb   Set  key       receipt.GetKV()
	//2. receipt.GetKV()    key,
	memkvset := e.stateDB.(*StateDB).GetSetKeys()
	err = e.checkKV(memkvset, receipt.GetKV())
	if err != nil {
		errlog := &types.ReceiptLog{Ty: types.TyLogErr, Log: []byte(err.Error())}
		feelog.Logs = append(feelog.Logs, errlog)
		return feelog, err
	}
	feelog, err = e.checkKeyAllow(feelog, tx, index, receipt.GetKV())
	if err != nil {
		return feelog, err
	}
	err = e.execLocalSameTime(tx, receipt, index)
	if err != nil {
		elog.Error("execLocalSameTime", "err", err)
		errlog := &types.ReceiptLog{Ty: types.TyLogErr, Log: []byte(err.Error())}
		feelog.Logs = append(feelog.Logs, errlog)
		return feelog, err
	}
	if receipt != nil {
		feelog.KV = append(feelog.KV, receipt.KV...)
		feelog.Logs = append(feelog.Logs, receipt.Logs...)
		feelog.Ty = receipt.Ty
	}
	if e.cfg.IsFork(e.height, "ForkStateDBSet") {
		for _, v := range feelog.KV {
			if err := e.stateDB.Set(v.Key, v.Value); err != nil {
				panic(err)
			}
		}
	}
	return feelog, nil
}

func (e *executor) checkKV(memset []string, kvs []*types.KeyValue) error {
	keys := make(map[string]bool)
	for _, kv := range kvs {
		k := kv.GetKey()
		keys[string(k)] = true
	}
	for _, key := range memset {
		if _, ok := keys[key]; !ok {
			elog.Error("err memset key", "key", key)
			//   receipt，
			return types.ErrNotAllowMemSetKey
		}
	}
	return nil
}

func (e *executor) checkKeyAllow(feelog *types.Receipt, tx *types.Transaction, index int, kvs []*types.KeyValue) (*types.Receipt, error) {
	for _, kv := range kvs {
		k := kv.GetKey()
		if !e.isAllowExec(k, tx, index) {
			elog.Error("err receipt key", "key", string(k), "tx.exec", string(tx.GetExecer()),
				"tx.action", tx.ActionName())
			//   receipt，
			errlog := &types.ReceiptLog{Ty: types.TyLogErr, Log: []byte(types.ErrNotAllowKey.Error())}
			feelog.Logs = append(feelog.Logs, errlog)
			return feelog, types.ErrNotAllowKey
		}
	}
	return feelog, nil
}

func (e *executor) begin() {
	if e.cfg.IsFork(e.height, "ForkExecRollback") {
		if e.stateDB != nil {
			e.stateDB.Begin()
		}
		if e.localDB != nil {
			e.localDB.Begin()
		}
	}
}

func (e *executor) commit() error {
	if e.cfg.IsFork(e.height, "ForkExecRollback") {
		if e.stateDB != nil {
			if err := e.stateDB.Commit(); err != nil {
				return err
			}
		}
		if e.localDB != nil {
			if err := e.localDB.Commit(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *executor) startTx() {
	if e.stateDB != nil {
		e.stateDB.(*StateDB).StartTx()
	}
	if e.localDB != nil {
		e.localDB.(*LocalDB).StartTx()
	}
}

func (e *executor) rollback() {
	if e.cfg.IsFork(e.height, "ForkExecRollback") {
		if e.stateDB != nil {
			e.stateDB.Rollback()
		}
		if e.localDB != nil {
			e.localDB.Rollback()
		}
	}
}

func (e *executor) execTx(exec *Executor, tx *types.Transaction, index int) (*types.Receipt, error) {
	if e.height == 0 { //genesis block
		receipt, err := e.Exec(tx, index)
		if err != nil {
			panic(err)
		}
		if err == nil && receipt == nil {
			panic("genesis block: executor not exist")
		}
		return receipt, nil
	}
	//      ：
	//1. mempool     ，
	//2.      ，         ，       ，
	err := e.checkTx(tx, index)
	if err != nil {
		elog.Error("execTx.checkTx ", "txhash", common.ToHex(tx.Hash()), "err", err)
		if e.cfg.IsPara() {
			panic(err)
		}
		return nil, err
	}
	//       (       )
	//       ，  receipt    pack
	//            error
	feelog, err := e.execFee(tx, index)
	if err != nil {
		return nil, err
	}
	//ignore err
	e.begin()
	feelog, err = e.execTxOne(feelog, tx, index)
	if err != nil {
		e.rollback()
		elog.Error("exec tx = ", "index", index, "execer", string(tx.Execer), "err", err)
	} else {
		err := e.commit()
		if err != nil {
			return nil, err
		}
	}

	if api.IsAPIEnvError(err) {
		return nil, err
	}
	return feelog, nil
}

//allowExec key
/*
      :
1.     :
              key
           exec key

2. friend     ,                  key
*/
func (e *executor) isAllowExec(key []byte, tx *types.Transaction, index int) bool {
	realExecer := e.getRealExecName(tx, index)
	return isAllowKeyWrite(e, key, realExecer, tx, index)
}

func (e *executor) isExecLocalSameTime(tx *types.Transaction, index int) bool {
	exec := e.loadDriver(tx, index)
	return exec.ExecutorOrder() == drivers.ExecLocalSameTime
}

func (e *executor) checkPrefix(execer []byte, kvs []*types.KeyValue) error {
	for i := 0; i < len(kvs); i++ {
		err := isAllowLocalKey(e.cfg, execer, kvs[i].Key)
		if err != nil {
			//      ， panic，
			panic(err)
			//return err
		}
	}
	return nil
}

func (e *executor) execLocalSameTime(tx *types.Transaction, receipt *types.Receipt, index int) error {
	if e.isExecLocalSameTime(tx, index) {
		var r = &types.ReceiptData{}
		if receipt != nil {
			r.Ty = receipt.Ty
			r.Logs = receipt.Logs
		}
		_, err := e.execLocalTx(tx, r, index)
		return err
	}
	return nil
}

func (e *executor) execLocalTx(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := e.execLocal(tx, r, index)
	if err == types.ErrActionNotSupport {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	memkvset := e.localDB.(*LocalDB).GetSetKeys()
	if kv != nil && kv.KV != nil {
		err := e.checkKV(memkvset, kv.KV)
		if err != nil {
			return nil, types.ErrNotAllowMemSetLocalKey
		}
		err = e.checkPrefix(tx.Execer, kv.KV)
		if err != nil {
			return nil, err
		}
		for _, kv := range kv.KV {
			err = e.localDB.Set(kv.Key, kv.Value)
			if err != nil {
				panic(err)
			}
		}
	} else {
		if len(memkvset) > 0 {
			return nil, types.ErrNotAllowMemSetLocalKey
		}
	}
	return kv, nil
}
