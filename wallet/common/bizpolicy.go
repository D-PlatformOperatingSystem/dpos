// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	"github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// WalletBizPolicy
type WalletBizPolicy interface {
	// Init          ，
	Init(walletBiz WalletOperate, sub []byte)
	// OnAddBlockTx
	// block：
	// tx:
	// index:              ， 0
	// dbbatch:
	//                     ，        ，           ，    nil
	OnAddBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) *types.WalletTxDetail
	// OnDeleteBlockTx
	// block：
	// tx:
	// index:              ， 0
	// dbbatch:
	//                     ，        ，           ，    nil
	OnDeleteBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) *types.WalletTxDetail
	// SignTransaction        ，
	// key
	// req
	// needSysSign                 ，true    ，false        ，
	// signtx      ，
	// err
	SignTransaction(key crypto.PrivKey, req *types.ReqSignRawTx) (needSysSign bool, signtx string, err error)
	// OnCreateNewAccount             ，
	OnCreateNewAccount(acc *types.Account)
	// OnImportPrivateKey         ，
	OnImportPrivateKey(acc *types.Account)
	OnWalletLocked()
	OnWalletUnlocked(WalletUnLock *types.WalletUnLock)
	OnAddBlockFinish(block *types.BlockDetail)
	OnDeleteBlockFinish(block *types.BlockDetail)
	OnClose()
	OnSetQueueClient()
	Call(funName string, in types.Message) (ret types.Message, err error)
}
