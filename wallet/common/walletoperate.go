// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package common
package common

import (
	"math/rand"
	"reflect"
	"sync"

	"github.com/D-PlatformOperatingSystem/dpos/client"
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	"github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

var (
	// QueryData
	QueryData = types.NewQueryData("On_")
	// PolicyContainer
	PolicyContainer = make(map[string]WalletBizPolicy)
)

// Init
func Init(wallet WalletOperate, sub map[string][]byte) {
	for k, v := range PolicyContainer {
		v.Init(wallet, sub[k])
	}
}

// RegisterPolicy
func RegisterPolicy(key string, policy WalletBizPolicy) {
	if _, existed := PolicyContainer[key]; existed {
		panic("RegisterPolicy dup")
	}
	PolicyContainer[key] = policy
	QueryData.Register(key, policy)
	QueryData.SetThis(key, reflect.ValueOf(policy))
}

// WalletOperate
type WalletOperate interface {
	RegisterMineStatusReporter(reporter MineStatusReport) error

	GetAPI() client.QueueProtocolAPI
	GetDBStore() db.DB
	GetSignType() int
	GetCoinType() uint32
	GetPassword() string
	GetBlockHeight() int64
	GetRandom() *rand.Rand
	GetWalletDone() chan struct{}
	GetLastHeader() *types.Header
	GetTxDetailByHashs(ReqHashes *types.ReqHashes)
	GetWaitGroup() *sync.WaitGroup
	GetAllPrivKeys() ([]crypto.PrivKey, error)
	GetWalletAccounts() ([]*types.WalletAccountStore, error)
	GetPrivKeyByAddr(addr string) (crypto.PrivKey, error)
	GetConfig() *types.Wallet
	GetBalance(addr string, execer string) (*types.Account, error)

	IsWalletLocked() bool
	IsClose() bool
	IsCaughtUp() bool
	AddrInWallet(addr string) bool

	CheckWalletStatus() (bool, error)
	Nonce() int64

	WaitTx(hash []byte) *types.TransactionDetail
	WaitTxs(hashes [][]byte) (ret []*types.TransactionDetail)
	SendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error)
	SendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error)
}
