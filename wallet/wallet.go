// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package wallet wallet dplatformos
package wallet

import (
	"errors"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/account"
	"github.com/D-PlatformOperatingSystem/dpos/client"
	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	clog "github.com/D-PlatformOperatingSystem/dpos/common/log"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/wallet/bipwallet"
	wcom "github.com/D-PlatformOperatingSystem/dpos/wallet/common"
)

var (
	//minFee           int64
	maxTxNumPerBlock int64 = types.MaxTxsPerBlock
	// MaxTxHashsPerTime
	MaxTxHashsPerTime int64 = 100
	walletlog               = log.New("module", "wallet")
	//accountdb         *account.DB
	//accTokenMap = make(map[string]*account.DB)
)

func init() {
	wcom.QueryData.Register("wallet", &Wallet{})
}

const (
	// AddTx
	AddTx int32 = 20001
	// DelTx
	DelTx int32 = 20002
	//
	sendTx int32 = 30001
	recvTx int32 = 30002
)

// Wallet
type Wallet struct {
	client queue.Client
	//           ,   api  client
	api                client.QueueProtocolAPI
	mtx                sync.Mutex
	timeout            *time.Timer
	mineStatusReporter wcom.MineStatusReport
	isclosed           int32
	isWalletLocked     int32
	fatalFailureFlag   int32
	Password           string
	FeeAmount          int64
	EncryptFlag        int64
	wg                 *sync.WaitGroup
	walletStore        *walletStore
	random             *rand.Rand
	cfg                *types.Wallet
	done               chan struct{}
	rescanwg           *sync.WaitGroup
	lastHeader         *types.Header
	initFlag           uint32 //               ，   0，
	SignType           int    // SignType      1；secp256k1，2：ed25519，3：sm2
	CoinType           uint32 // CoinType      dpos:0x80003333,ycc:0x80003334

	minFee      int64
	accountdb   *account.DB
	accTokenMap map[string]*account.DB
}

// SetLogLevel
func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

// DisableLog
func DisableLog() {
	walletlog.SetHandler(log.DiscardHandler())
	storelog.SetHandler(log.DiscardHandler())
}

// New
func New(cfg *types.DplatformOSConfig) *Wallet {
	mcfg := cfg.GetModuleConfig().Wallet
	//walletStore
	//accountdb = account.NewCoinsAccount()
	walletStoreDB := dbm.NewDB("wallet", mcfg.Driver, mcfg.DbPath, mcfg.DbCache)
	//walletStore := NewStore(walletStoreDB)
	walletStore := newStore(walletStoreDB)
	//minFee = cfg.MinFee
	signType := types.GetSignType("", mcfg.SignType)
	if signType == types.Invalid {
		signType = types.SECP256K1
	}

	wallet := &Wallet{
		walletStore:      walletStore,
		isWalletLocked:   1,
		fatalFailureFlag: 0,
		wg:               &sync.WaitGroup{},
		FeeAmount:        walletStore.GetFeeAmount(mcfg.MinFee),
		EncryptFlag:      walletStore.GetEncryptionFlag(),
		done:             make(chan struct{}),
		cfg:              mcfg,
		rescanwg:         &sync.WaitGroup{},
		initFlag:         0,
		SignType:         signType,
		CoinType:         bipwallet.GetSLIP0044CoinType(mcfg.CoinType),
		minFee:           mcfg.MinFee,
		accountdb:        account.NewCoinsAccount(cfg),
		accTokenMap:      make(map[string]*account.DB),
	}
	wallet.random = rand.New(rand.NewSource(types.Now().UnixNano()))
	wcom.QueryData.SetThis("wallet", reflect.ValueOf(wallet))
	return wallet
}

//Wait for wallet ready
func (wallet *Wallet) Wait() {}

// RegisterMineStatusReporter
func (wallet *Wallet) RegisterMineStatusReporter(reporter wcom.MineStatusReport) error {
	if reporter == nil {
		return types.ErrInvalidParam
	}
	if wallet.mineStatusReporter != nil {
		return errors.New("ReporterIsExisted")
	}
	consensus := wallet.client.GetConfig().GetModuleConfig().Consensus.Name

	if !isConflict(consensus, reporter.PolicyName()) {
		wallet.mineStatusReporter = reporter
	}
	return nil
}

//   policy Consensus    ，      reporter
func isConflict(curConsensus string, policy string) bool {
	walletlog.Info("isConflict", "curConsensus", curConsensus, "policy", policy)

	return curConsensus != policy
}

// GetConfig
func (wallet *Wallet) GetConfig() *types.Wallet {
	return wallet.cfg
}

// GetAPI     API
func (wallet *Wallet) GetAPI() client.QueueProtocolAPI {
	return wallet.api
}

// GetDBStore
func (wallet *Wallet) GetDBStore() dbm.DB {
	return wallet.walletStore.GetDB()
}

// GetSignType
func (wallet *Wallet) GetSignType() int {
	return wallet.SignType
}

// GetCoinType
func (wallet *Wallet) GetCoinType() uint32 {
	return wallet.CoinType
}

// GetPassword
func (wallet *Wallet) GetPassword() string {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	return wallet.Password
}

// Nonce
func (wallet *Wallet) Nonce() int64 {
	return wallet.random.Int63()
}

// AddWaitGroup
func (wallet *Wallet) AddWaitGroup(delta int) {
	wallet.wg.Add(delta)
}

// WaitGroupDone
func (wallet *Wallet) WaitGroupDone() {
	wallet.wg.Done()
}

// GetBlockHeight
func (wallet *Wallet) GetBlockHeight() int64 {
	return wallet.GetHeight()
}

// GetRandom
func (wallet *Wallet) GetRandom() *rand.Rand {
	return wallet.random
}

// GetWalletDone
func (wallet *Wallet) GetWalletDone() chan struct{} {
	return wallet.done
}

// GetLastHeader
func (wallet *Wallet) GetLastHeader() *types.Header {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	return wallet.lastHeader
}

// GetWaitGroup
func (wallet *Wallet) GetWaitGroup() *sync.WaitGroup {
	return wallet.wg
}

// GetAccountByLabel
func (wallet *Wallet) GetAccountByLabel(label string) (*types.WalletAccountStore, error) {
	return wallet.walletStore.GetAccountByLabel(label)
}

// IsRescanUtxosFlagScaning       UTXO
func (wallet *Wallet) IsRescanUtxosFlagScaning() (bool, error) {
	in := &types.ReqNil{}
	flag := false
	for _, policy := range wcom.PolicyContainer {
		out, err := policy.Call("GetUTXOScaningFlag", in)
		if err != nil {
			if err.Error() == types.ErrNotSupport.Error() {
				continue
			}
			return flag, err
		}
		reply, ok := out.(*types.Reply)
		if !ok {
			err = types.ErrTypeAsset
			return flag, err
		}
		flag = reply.IsOk
		return flag, err
	}

	return flag, nil
}

// Close
func (wallet *Wallet) Close() {
	//
	//set close flag to isclosed == 1
	atomic.StoreInt32(&wallet.isclosed, 1)
	for _, policy := range wcom.PolicyContainer {
		policy.OnClose()
	}
	close(wallet.done)
	wallet.client.Close()
	wallet.wg.Wait()
	//
	wallet.walletStore.Close()
	walletlog.Info("wallet module closed")
}

// IsClose
func (wallet *Wallet) IsClose() bool {
	return atomic.LoadInt32(&wallet.isclosed) == 1
}

// IsWalletLocked
func (wallet *Wallet) IsWalletLocked() bool {
	return atomic.LoadInt32(&wallet.isWalletLocked) != 0
}

// SetQueueClient
func (wallet *Wallet) SetQueueClient(cli queue.Client) {
	var err error
	wallet.client = cli
	wallet.client.Sub("wallet")
	wallet.api, err = client.New(cli, nil)
	if err != nil {
		panic("SetQueueClient client.New err")
	}
	sub := cli.GetConfig().GetSubConfig().Wallet
	//   client    Init
	wcom.Init(wallet, sub)
	wallet.wg.Add(1)
	go wallet.ProcRecvMsg()
	for _, policy := range wcom.PolicyContainer {
		policy.OnSetQueueClient()
	}
	wallet.setInited(true)
}

// GetAccountByAddr
func (wallet *Wallet) GetAccountByAddr(addr string) (*types.WalletAccountStore, error) {
	return wallet.walletStore.GetAccountByAddr(addr)
}

// SetWalletAccount
func (wallet *Wallet) SetWalletAccount(update bool, addr string, account *types.WalletAccountStore) error {
	return wallet.walletStore.SetWalletAccount(update, addr, account)
}

// GetPrivKeyByAddr
func (wallet *Wallet) GetPrivKeyByAddr(addr string) (crypto.PrivKey, error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	return wallet.getPrivKeyByAddr(addr)
}

func (wallet *Wallet) getPrivKeyByAddr(addr string) (crypto.PrivKey, error) {
	//
	Accountstor, err := wallet.walletStore.GetAccountByAddr(addr)
	if err != nil {
		walletlog.Error("getPrivKeyByAddr", "GetAccountByAddr err:", err)
		return nil, err
	}

	//  password
	prikeybyte, err := common.FromHex(Accountstor.GetPrivkey())
	if err != nil || len(prikeybyte) == 0 {
		walletlog.Error("getPrivKeyByAddr", "FromHex err", err)
		return nil, err
	}

	privkey := wcom.CBCDecrypterPrivkey([]byte(wallet.Password), prikeybyte)
	//  privkey    pubkey        addr
	cr, err := crypto.New(types.GetSignName("", wallet.SignType))
	if err != nil {
		walletlog.Error("getPrivKeyByAddr", "err", err)
		return nil, err
	}
	priv, err := cr.PrivKeyFromBytes(privkey)
	if err != nil {
		walletlog.Error("getPrivKeyByAddr", "PrivKeyFromBytes err", err)
		return nil, err
	}
	return priv, nil
}

// AddrInWallet
func (wallet *Wallet) AddrInWallet(addr string) bool {
	if !wallet.isInited() {
		return false
	}
	if len(addr) == 0 {
		return false
	}
	acc, err := wallet.walletStore.GetAccountByAddr(addr)
	if err == nil && acc != nil {
		return true
	}
	return false
}

//IsTransfer                ，         seed
func (wallet *Wallet) IsTransfer(addr string) (bool, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	return wallet.isTransfer(addr)
}

//isTransfer                ，         seed
func (wallet *Wallet) isTransfer(addr string) (bool, error) {

	ok, err := wallet.checkWalletStatus()
	//           ErrSaveSeedFirst
	if ok || err == types.ErrSaveSeedFirst {
		return ok, err
	}
	//      ，       ,    addr
	//     ticket
	if !wallet.isTicketLocked() {
		consensus := wallet.client.GetConfig().GetModuleConfig().Consensus.Name

		if addr == address.ExecAddress(consensus) {
			return true, nil
		}
	}
	return ok, err
}

//CheckWalletStatus         ,    ，seed
func (wallet *Wallet) CheckWalletStatus() (bool, error) {
	if !wallet.isInited() {
		return false, types.ErrNotInited
	}

	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	return wallet.checkWalletStatus()
}

//CheckWalletStatus         ,    ，seed
func (wallet *Wallet) checkWalletStatus() (bool, error) {
	//     ，ticket    ，      ticket
	if wallet.IsWalletLocked() && !wallet.isTicketLocked() {
		return false, types.ErrOnlyTicketUnLocked
	} else if wallet.IsWalletLocked() {
		return false, types.ErrWalletIsLocked
	}

	//         seed
	has, err := wallet.walletStore.HasSeed()
	if !has || err != nil {
		return false, types.ErrSaveSeedFirst
	}
	return true, nil
}

func (wallet *Wallet) isTicketLocked() bool {
	locked := true
	if wallet.mineStatusReporter != nil {
		locked = wallet.mineStatusReporter.IsTicketLocked()
	}
	return locked
}

func (wallet *Wallet) isAutoMinning() bool {
	autoMining := false
	if wallet.mineStatusReporter != nil {
		autoMining = wallet.mineStatusReporter.IsAutoMining()
	}
	return autoMining
}

// GetWalletStatus
func (wallet *Wallet) GetWalletStatus() *types.WalletStatus {
	var err error
	s := &types.WalletStatus{}
	s.IsWalletLock = wallet.IsWalletLocked()
	s.IsHasSeed, err = wallet.walletStore.HasSeed()
	s.IsAutoMining = wallet.isAutoMinning()
	s.IsTicketLock = wallet.isTicketLocked()
	if err != nil {
		walletlog.Debug("GetWalletStatus HasSeed ", "err", err)
	}
	walletlog.Debug("GetWalletStatus", "walletstatus", s)
	return s
}

// GetWalletAccounts
//output:
//type WalletAccountStore struct {
//	Privkey   string  //      hex
//	Label     string
//	Addr      string
//	TimeStamp string
//             ，
func (wallet *Wallet) getWalletAccounts() ([]*types.WalletAccountStore, error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}

	//  Account
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Info("GetWalletAccounts", "GetAccountByPrefix:err", err)
		return nil, err
	}
	return WalletAccStores, err
}

//GetWalletAccounts
func (wallet *Wallet) GetWalletAccounts() ([]*types.WalletAccountStore, error) {
	if !wallet.isInited() {
		return nil, types.ErrNotInited
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	return wallet.getWalletAccounts()
}

func (wallet *Wallet) updateLastHeader(block *types.BlockDetail, mode int) error {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	header, err := wallet.api.GetLastHeader()
	if err != nil {
		return err
	}
	if block != nil {
		if mode == 1 && block.Block.Height > header.Height {
			wallet.lastHeader = &types.Header{
				BlockTime: block.Block.BlockTime,
				Height:    block.Block.Height,
				StateHash: block.Block.StateHash,
			}
		} else if mode == -1 && wallet.lastHeader != nil && wallet.lastHeader.Height == block.Block.Height {
			wallet.lastHeader = header
		}
	}
	if block == nil || wallet.lastHeader == nil {
		wallet.lastHeader = header
	}
	return nil
}

func (wallet *Wallet) setInited(flag bool) {
	if flag && !wallet.isInited() {
		atomic.StoreUint32(&wallet.initFlag, 1)
	}
}

func (wallet *Wallet) isInited() bool {
	return atomic.LoadUint32(&wallet.initFlag) != 0
}
