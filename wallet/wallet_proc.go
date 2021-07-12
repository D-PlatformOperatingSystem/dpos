// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/D-PlatformOperatingSystem/dpos/account"
	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	cty "github.com/D-PlatformOperatingSystem/dpos/system/dapp/coins/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/wallet/bipwallet"
	wcom "github.com/D-PlatformOperatingSystem/dpos/wallet/common"
	"github.com/golang/protobuf/proto"
)

// ProcSignRawTx
//input:
//type ReqSignRawTx struct {
//	Addr    string
//	Privkey string
//	TxHex   string
//	Expire  string
//}
//output:
//string
//
func (wallet *Wallet) ProcSignRawTx(unsigned *types.ReqSignRawTx) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	index := unsigned.Index

	//Privacy    ，     plugin    ，            ，     scan             ，     scaning  ，
	//        ，   nil
	//if ok, err := wallet.IsRescanUtxosFlagScaning(); ok || err != nil {
	//	return "", err types.ErrNotSupport
	//}

	var key crypto.PrivKey
	if unsigned.GetAddr() != "" {
		ok, err := wallet.checkWalletStatus()
		if !ok {
			return "", err
		}
		key, err = wallet.getPrivKeyByAddr(unsigned.GetAddr())
		if err != nil {
			return "", err
		}
	} else if unsigned.GetPrivkey() != "" {
		keyByte, err := common.FromHex(unsigned.GetPrivkey())
		if err != nil {
			return "", err
		}
		if len(keyByte) == 0 {
			return "", types.ErrPrivateKeyLen
		}
		cr, err := crypto.New(types.GetSignName("", wallet.SignType))
		if err != nil {
			return "", err
		}
		key, err = cr.PrivKeyFromBytes(keyByte)
		if err != nil {
			return "", err
		}
	} else {
		return "", types.ErrNoPrivKeyOrAddr
	}

	txByteData, err := common.FromHex(unsigned.GetTxHex())
	if err != nil {
		return "", err
	}
	var tx types.Transaction
	err = types.Decode(txByteData, &tx)
	if err != nil {
		return "", err
	}

	if unsigned.NewToAddr != "" {
		tx.To = unsigned.NewToAddr
	}
	if unsigned.Fee != 0 {
		tx.Fee = unsigned.Fee
	} else {
		//get proper fee if not set
		proper, err := wallet.api.GetProperFee(nil)
		if err != nil {
			return "", err
		}
		fee, err := tx.GetRealFee(proper.ProperFee)
		if err != nil {
			return "", err
		}
		tx.Fee = fee
	}

	expire, err := types.ParseExpire(unsigned.GetExpire())
	if err != nil {
		return "", err
	}
	types.AssertConfig(wallet.client)
	cfg := wallet.client.GetConfig()
	tx.SetExpire(cfg, time.Duration(expire))
	if policy, ok := wcom.PolicyContainer[string(cfg.GetParaExec(tx.Execer))]; ok {
		//
		needSysSign, signtx, err := policy.SignTransaction(key, unsigned)
		if !needSysSign {
			return signtx, err
		}
	}

	group, err := tx.GetTxGroup()
	if err != nil {
		return "", err
	}
	if group == nil {
		tx.Sign(int32(wallet.SignType), key)
		txHex := types.Encode(&tx)
		signedTx := hex.EncodeToString(txHex)
		return signedTx, nil
	}
	if int(index) > len(group.GetTxs()) {
		return "", types.ErrIndex
	}
	if index <= 0 {
		//
		group.SetExpire(cfg, 0, time.Duration(expire))
		group.RebuiltGroup()
		for i := range group.Txs {
			err := group.SignN(i, int32(wallet.SignType), key)
			if err != nil {
				return "", err
			}
		}
		grouptx := group.Tx()
		txHex := types.Encode(grouptx)
		signedTx := hex.EncodeToString(txHex)
		return signedTx, nil
	}
	index--
	err = group.SignN(int(index), int32(wallet.SignType), key)
	if err != nil {
		return "", err
	}
	grouptx := group.Tx()
	txHex := types.Encode(grouptx)
	signedTx := hex.EncodeToString(txHex)
	return signedTx, nil
}

// ProcGetAccount
func (wallet *Wallet) ProcGetAccount(req *types.ReqGetAccount) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	accStore, err := wallet.walletStore.GetAccountByLabel(req.GetLabel())
	if err != nil {
		return nil, err
	}

	accs, err := wallet.accountdb.LoadAccounts(wallet.api, []string{accStore.GetAddr()})
	if err != nil {
		return nil, err
	}
	return &types.WalletAccount{Label: accStore.GetLabel(), Acc: accs[0]}, nil

}

// ProcGetAccountList
//output:
//type WalletAccounts struct {
//	Wallets []*WalletAccount
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//
func (wallet *Wallet) ProcGetAccountList(req *types.ReqAccountList) (*types.WalletAccounts, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	//  Account
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Info("ProcGetAccountList", "GetAccountByPrefix:err", err)
		return nil, err
	}
	if req.WithoutBalance {
		return makeAccountWithoutBalance(WalletAccStores)
	}

	addrs := make([]string, len(WalletAccStores))
	for index, AccStore := range WalletAccStores {
		if len(AccStore.Addr) != 0 {
			addrs[index] = AccStore.Addr
		}
	}
	//                account
	accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcGetAccountList", "LoadAccounts:err", err)
		return nil, err
	}

	//
	if len(WalletAccStores) != len(accounts) {
		walletlog.Error("ProcGetAccountList err!", "AccStores)", len(WalletAccStores), "accounts", len(accounts))
	}

	var WalletAccounts types.WalletAccounts
	WalletAccounts.Wallets = make([]*types.WalletAccount, len(WalletAccStores))

	for index, Account := range accounts {
		var WalletAccount types.WalletAccount
		//            account
		if len(Account.Addr) == 0 {
			Account.Addr = addrs[index]
		}
		WalletAccount.Acc = Account
		WalletAccount.Label = WalletAccStores[index].GetLabel()
		WalletAccounts.Wallets[index] = &WalletAccount
	}
	return &WalletAccounts, nil
}

func makeAccountWithoutBalance(accountStores []*types.WalletAccountStore) (*types.WalletAccounts, error) {
	var WalletAccounts types.WalletAccounts
	WalletAccounts.Wallets = make([]*types.WalletAccount, len(accountStores))

	for index, account := range accountStores {
		var WalletAccount types.WalletAccount
		//            account
		if len(account.Addr) == 0 {
			continue
		}
		WalletAccount.Acc = &types.Account{Addr: account.Addr}
		WalletAccount.Label = account.GetLabel()
		WalletAccounts.Wallets[index] = &WalletAccount
	}
	return &WalletAccounts, nil
}

// ProcCreateNewAccount
//input:
//type ReqNewAccount struct {
//	Label string
//output:
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//type Account struct {
//	Currency int32
//	Balance  int64
//	Frozen   int64
//	Addr     string
//
func (wallet *Wallet) ProcCreateNewAccount(Label *types.ReqNewAccount) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
	if !ok {
		return nil, err
	}

	if Label == nil || len(Label.GetLabel()) == 0 {
		walletlog.Error("ProcCreateNewAccount Label is nil")
		return nil, types.ErrInvalidParam
	}

	//    label
	WalletAccStores, err := wallet.walletStore.GetAccountByLabel(Label.GetLabel())
	if WalletAccStores != nil && err == nil {
		walletlog.Error("ProcCreateNewAccount Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
	}

	var Account types.Account
	var walletAccount types.WalletAccount
	var WalletAccStore types.WalletAccountStore
	var addr string
	var privkeybyte []byte

	cointype := wallet.GetCoinType()
	//  seed    ,           seed    seed
	seed, err := wallet.getSeed(wallet.Password)
	if err != nil {
		walletlog.Error("ProcCreateNewAccount", "getSeed err", err)
		return nil, err
	}

	for {
		privkeyhex, err := GetPrivkeyBySeed(wallet.walletStore.GetDB(), seed, 0, wallet.SignType, wallet.CoinType)
		if err != nil {
			walletlog.Error("ProcCreateNewAccount", "GetPrivkeyBySeed err", err)
			return nil, err
		}
		privkeybyte, err = common.FromHex(privkeyhex)
		if err != nil || len(privkeybyte) == 0 {
			walletlog.Error("ProcCreateNewAccount", "FromHex err", err)
			return nil, err
		}

		pub, err := bipwallet.PrivkeyToPub(cointype, uint32(wallet.SignType), privkeybyte)
		if err != nil {
			seedlog.Error("ProcCreateNewAccount PrivkeyToPub", "err", err)
			return nil, types.ErrPrivkeyToPub
		}
		addr, err = bipwallet.PubToAddress(pub)
		if err != nil {
			seedlog.Error("ProcCreateNewAccount PubToAddress", "err", err)
			return nil, types.ErrPrivkeyToPub
		}
		//                 ，             ，
		//             ，         index
		account, err := wallet.walletStore.GetAccountByAddr(addr)
		if account == nil || err != nil {
			break
		}
	}

	Account.Addr = addr
	Account.Currency = 0
	Account.Balance = 0
	Account.Frozen = 0

	walletAccount.Acc = &Account
	walletAccount.Label = Label.GetLabel()

	//     password      aes cbc
	Encrypted := wcom.CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	WalletAccStore.Privkey = common.ToHex(Encrypted)
	WalletAccStore.Label = Label.GetLabel()
	WalletAccStore.Addr = addr

	//       wallet
	err = wallet.walletStore.SetWalletAccount(false, Account.Addr, &WalletAccStore)
	if err != nil {
		return nil, err
	}

	//            account
	addrs := make([]string, 1)
	addrs[0] = addr
	accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil {
		walletlog.Error("ProcCreateNewAccount", "LoadAccounts err", err)
		return nil, err
	}
	//
	if len(accounts[0].Addr) == 0 {
		accounts[0].Addr = addr
	}
	walletAccount.Acc = accounts[0]

	// blockchain    Account.Addr
	for _, policy := range wcom.PolicyContainer {
		policy.OnCreateNewAccount(walletAccount.Acc)
	}

	return &walletAccount, nil
}

// ProcWalletTxList
//input:
//type ReqWalletTransactionList struct {
//	FromTx []byte
//	Count  int32
//output:
//type WalletTxDetails struct {
//	TxDetails []*WalletTxDetail
//type WalletTxDetail struct {
//	Tx      *Transaction
//	Receipt *ReceiptData
//	Height  int64
//	Index   int64
//
func (wallet *Wallet) ProcWalletTxList(TxList *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if TxList == nil {
		walletlog.Error("ProcWalletTxList TxList is nil!")
		return nil, types.ErrInvalidParam
	}
	if TxList.GetDirection() != 0 && TxList.GetDirection() != 1 {
		walletlog.Error("ProcWalletTxList Direction err!")
		return nil, types.ErrInvalidParam
	}
	//   10
	if TxList.Count == 0 {
		TxList.Count = 10
	}
	if int64(TxList.Count) > types.MaxBlockCountPerTime {
		return nil, types.ErrMaxCountPerTime
	}

	WalletTxDetails, err := wallet.walletStore.GetTxDetailByIter(TxList)
	if err != nil {
		walletlog.Error("ProcWalletTxList", "GetTxDetailByIter err", err)
		return nil, err
	}
	return WalletTxDetails, nil
}

// ProcImportPrivKey
//input:
//type ReqWalletImportPrivKey struct {
//	Privkey string
//	Label   string
//output:
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//    ，
func (wallet *Wallet) ProcImportPrivKey(PrivKey *types.ReqWalletImportPrivkey) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	walletaccount, err := wallet.procImportPrivKey(PrivKey)
	return walletaccount, err
}

func (wallet *Wallet) procImportPrivKey(PrivKey *types.ReqWalletImportPrivkey) (*types.WalletAccount, error) {
	ok, err := wallet.checkWalletStatus()
	if !ok {
		return nil, err
	}

	if PrivKey == nil || len(PrivKey.GetLabel()) == 0 || len(PrivKey.GetPrivkey()) == 0 {
		walletlog.Error("ProcImportPrivKey input parameter is nil!")
		return nil, types.ErrInvalidParam
	}

	//  label
	Account, err := wallet.walletStore.GetAccountByLabel(PrivKey.GetLabel())
	if Account != nil && err == nil {
		walletlog.Error("ProcImportPrivKey Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
	}

	cointype := wallet.GetCoinType()

	privkeybyte, err := common.FromHex(PrivKey.Privkey)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("ProcImportPrivKey", "FromHex err", err)
		return nil, types.ErrFromHex
	}

	pub, err := bipwallet.PrivkeyToPub(cointype, uint32(wallet.SignType), privkeybyte)
	if err != nil {
		seedlog.Error("ProcImportPrivKey PrivkeyToPub", "err", err)
		return nil, types.ErrPrivkeyToPub
	}

	addr, err := bipwallet.PubToAddress(pub)
	if err != nil {
		seedlog.Error("ProcImportPrivKey PrivkeyToPub", "err", err)
		return nil, types.ErrPrivkeyToPub
	}

	//
	Encryptered := wcom.CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)
	Encrypteredstr := common.ToHex(Encryptered)
	//  PrivKey   addr
	Account, err = wallet.walletStore.GetAccountByAddr(addr)
	if Account != nil && err == nil {
		if Account.Privkey == Encrypteredstr {
			walletlog.Error("ProcImportPrivKey Privkey is exist in wallet!")
			return nil, types.ErrPrivkeyExist
		}
		walletlog.Error("ProcImportPrivKey!", "Account.Privkey", Account.Privkey, "input Privkey", PrivKey.Privkey)
		return nil, types.ErrPrivkey
	}

	var walletaccount types.WalletAccount
	var WalletAccStore types.WalletAccountStore
	WalletAccStore.Privkey = Encrypteredstr //
	WalletAccStore.Label = PrivKey.GetLabel()
	WalletAccStore.Addr = addr
	//  Addr:label+privkey+addr
	err = wallet.walletStore.SetWalletAccount(false, addr, &WalletAccStore)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "SetWalletAccount err", err)
		return nil, err
	}

	//            account
	addrs := make([]string, 1)
	addrs[0] = addr
	accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil {
		walletlog.Error("ProcImportPrivKey", "LoadAccounts err", err)
		return nil, err
	}
	//
	if len(accounts[0].Addr) == 0 {
		accounts[0].Addr = addr
	}
	walletaccount.Acc = accounts[0]
	walletaccount.Label = PrivKey.Label

	for _, policy := range wcom.PolicyContainer {
		policy.OnImportPrivateKey(accounts[0])
	}
	return &walletaccount, nil
}

// ProcSendToAddress
//input:
//type ReqWalletSendToAddress struct {
//	From   string
//	To     string
//	Amount int64
//	Note   string
//output:
//type ReplyHash struct {
//	Hashe []byte
//           ，    hash
func (wallet *Wallet) ProcSendToAddress(SendToAddress *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if SendToAddress == nil {
		walletlog.Error("ProcSendToAddress input para is nil")
		return nil, types.ErrInvalidParam
	}
	if len(SendToAddress.From) == 0 || len(SendToAddress.To) == 0 {
		walletlog.Error("ProcSendToAddress input para From or To is nil!")
		return nil, types.ErrInvalidParam
	}

	ok, err := wallet.isTransfer(SendToAddress.GetTo())
	if !ok {
		return nil, err
	}

	//  from      account  ，
	addrs := make([]string, 1)
	addrs[0] = SendToAddress.GetFrom()
	var accounts []*types.Account
	var tokenAccounts []*types.Account
	accounts, err = wallet.accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcSendToAddress", "LoadAccounts err", err)
		return nil, err
	}
	Balance := accounts[0].Balance
	amount := SendToAddress.GetAmount()
	//amount      0
	if amount < 0 {
		return nil, types.ErrAmount
	}
	if !SendToAddress.IsToken {
		if Balance-amount < wallet.FeeAmount {
			return nil, types.ErrInsufficientBalance
		}
	} else {
		//   token  ，       coin     fee，         token
		if Balance < wallet.FeeAmount {
			return nil, types.ErrInsufficientBalance
		}
		if nil == wallet.accTokenMap[SendToAddress.TokenSymbol] {
			tokenAccDB, err := account.NewAccountDB(wallet.api.GetConfig(), "token", SendToAddress.TokenSymbol, nil)
			if err != nil {
				return nil, err
			}
			wallet.accTokenMap[SendToAddress.TokenSymbol] = tokenAccDB
		}
		tokenAccDB := wallet.accTokenMap[SendToAddress.TokenSymbol]
		tokenAccounts, err = tokenAccDB.LoadAccounts(wallet.api, addrs)
		if err != nil || len(tokenAccounts) == 0 {
			walletlog.Error("ProcSendToAddress", "Load Token Accounts err", err)
			return nil, err
		}
		tokenBalance := tokenAccounts[0].Balance
		if tokenBalance < amount {
			return nil, types.ErrInsufficientTokenBal
		}
	}
	addrto := SendToAddress.GetTo()
	note := SendToAddress.GetNote()
	priv, err := wallet.getPrivKeyByAddr(addrs[0])
	if err != nil {
		return nil, err
	}
	return wallet.sendToAddress(priv, addrto, amount, note, SendToAddress.IsToken, SendToAddress.TokenSymbol)
}

// ProcWalletSetFee
//type ReqWalletSetFee struct {
//	Amount int64
//
func (wallet *Wallet) ProcWalletSetFee(WalletSetFee *types.ReqWalletSetFee) error {
	if WalletSetFee.Amount < wallet.minFee {
		walletlog.Error("ProcWalletSetFee err!", "Amount", WalletSetFee.Amount, "MinFee", wallet.minFee)
		return types.ErrInvalidParam
	}
	err := wallet.walletStore.SetFeeAmount(WalletSetFee.Amount)
	if err == nil {
		walletlog.Debug("ProcWalletSetFee success!")
		wallet.mtx.Lock()
		wallet.FeeAmount = WalletSetFee.Amount
		wallet.mtx.Unlock()
	}
	return err
}

// ProcWalletSetLabel
//input:
//type ReqWalletSetLabel struct {
//	Addr  string
//	Label string
//output:
//type WalletAccount struct {
//	Acc   *Account
//	Label string
//
func (wallet *Wallet) ProcWalletSetLabel(SetLabel *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	if SetLabel == nil || len(SetLabel.Addr) == 0 || len(SetLabel.Label) == 0 {
		walletlog.Error("ProcWalletSetLabel input parameter is nil!")
		return nil, types.ErrInvalidParam
	}
	//  label
	Account, err := wallet.walletStore.GetAccountByLabel(SetLabel.GetLabel())
	if Account != nil && err == nil {
		walletlog.Error("ProcWalletSetLabel Label is exist in wallet!")
		return nil, types.ErrLabelHasUsed
	}
	//               ,    label
	Account, err = wallet.walletStore.GetAccountByAddr(SetLabel.Addr)
	if err == nil && Account != nil {
		oldLabel := Account.Label
		Account.Label = SetLabel.GetLabel()
		err := wallet.walletStore.SetWalletAccount(true, SetLabel.Addr, Account)
		if err == nil {
			//  label            label db
			wallet.walletStore.DelAccountByLabel(oldLabel)

			//              account
			addrs := make([]string, 1)
			addrs[0] = SetLabel.Addr
			accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
			if err != nil || len(accounts) == 0 {
				walletlog.Error("ProcWalletSetLabel", "LoadAccounts err", err)
				return nil, err
			}
			var walletAccount types.WalletAccount
			walletAccount.Acc = accounts[0]
			walletAccount.Label = SetLabel.GetLabel()
			return &walletAccount, err
		}
	}
	return nil, err
}

// ProcMergeBalance
//input:
//type ReqWalletMergeBalance struct {
//	To string
//output:
//type ReplyHashes struct {
//	Hashes [][]byte
//     balance
func (wallet *Wallet) ProcMergeBalance(MergeBalance *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
	if !ok {
		return nil, err
	}

	if len(MergeBalance.GetTo()) == 0 {
		walletlog.Error("ProcMergeBalance input para is nil!")
		return nil, types.ErrInvalidParam
	}

	//
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Error("ProcMergeBalance", "GetAccountByPrefix err", err)
		return nil, err
	}

	addrs := make([]string, len(WalletAccStores))
	for index, AccStore := range WalletAccStores {
		if len(AccStore.Addr) != 0 {
			addrs[index] = AccStore.Addr
		}
	}
	//              account
	accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
	if err != nil || len(accounts) == 0 {
		walletlog.Error("ProcMergeBalance", "LoadAccounts err", err)
		return nil, err
	}

	//
	if len(WalletAccStores) != len(accounts) {
		walletlog.Error("ProcMergeBalance", "AccStores", len(WalletAccStores), "accounts", len(accounts))
	}
	//  privkey    pubkey        addr
	cr, err := crypto.New(types.GetSignName("", wallet.SignType))
	if err != nil {
		walletlog.Error("ProcMergeBalance", "err", err)
		return nil, err
	}

	addrto := MergeBalance.GetTo()
	note := "MergeBalance"

	var ReplyHashes types.ReplyHashes

	types.AssertConfig(wallet.client)
	cfg := wallet.client.GetConfig()
	for index, Account := range accounts {
		Privkey := WalletAccStores[index].Privkey
		//
		prikeybyte, err := common.FromHex(Privkey)
		if err != nil || len(prikeybyte) == 0 {
			walletlog.Error("ProcMergeBalance", "FromHex err", err, "index", index)
			continue
		}

		privkey := wcom.CBCDecrypterPrivkey([]byte(wallet.Password), prikeybyte)
		priv, err := cr.PrivKeyFromBytes(privkey)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "PrivKeyFromBytes err", err, "index", index)
			continue
		}
		//   to
		if Account.Addr == addrto {
			continue
		}
		//       ，
		amount := Account.GetBalance()
		if amount < wallet.FeeAmount {
			continue
		}
		amount = amount - wallet.FeeAmount
		v := &cty.CoinsAction_Transfer{
			Transfer: &types.AssetsTransfer{Amount: amount, Note: []byte(note)},
		}
		if cfg.IsPara() {
			v.Transfer.To = MergeBalance.GetTo()
		}
		transfer := &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
		//
		exec := []byte("coins")
		toAddr := addrto
		if cfg.IsPara() {
			exec = []byte(cfg.GetTitle() + "coins")
			toAddr = address.ExecAddress(string(exec))
		}
		tx := &types.Transaction{Execer: exec, Payload: types.Encode(transfer), Fee: wallet.FeeAmount, To: toAddr, Nonce: wallet.random.Int63()}
		tx.SetExpire(cfg, time.Second*120)
		tx.Sign(int32(wallet.SignType), priv)
		//walletlog.Info("ProcMergeBalance", "tx.Nonce", tx.Nonce, "tx", tx, "index", index)

		//       mempool
		msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
		err = wallet.client.Send(msg, true)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "Send tx err", err, "index", index)
			continue
		}
		resp, err := wallet.client.Wait(msg)
		if err != nil {
			walletlog.Error("ProcMergeBalance", "Send tx err", err, "index", index)
			continue
		}
		//     mempool    ，
		reply := resp.GetData().(*types.Reply)
		if !reply.GetIsOk() {
			walletlog.Error("ProcMergeBalance", "Send tx err", string(reply.GetMsg()), "index", index)
			continue
		}
		ReplyHashes.Hashes = append(ReplyHashes.Hashes, tx.Hash())
	}
	return &ReplyHashes, nil
}

// ProcWalletSetPasswd
//input:
//type ReqWalletSetPasswd struct {
//	Oldpass string
//	Newpass string
//
func (wallet *Wallet) ProcWalletSetPasswd(Passwd *types.ReqWalletSetPasswd) error {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	isok, err := wallet.checkWalletStatus()
	if !isok && err == types.ErrSaveSeedFirst {
		return err
	}
	//
	if !isValidPassWord(Passwd.NewPass) {
		return types.ErrInvalidPassWord
	}
	//        ，       ，
	tempislock := atomic.LoadInt32(&wallet.isWalletLocked)
	//wallet.isWalletLocked = false
	atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 1, 0)

	defer func() {
		//wallet.isWalletLocked = tempislock
		atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, tempislock)
	}()

	//           oldpass
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		isok := wallet.walletStore.VerifyPasswordHash(Passwd.OldPass)
		if !isok {
			walletlog.Error("ProcWalletSetPasswd Verify Oldpasswd fail!")
			return types.ErrVerifyOldpasswdFail
		}
	}

	if len(wallet.Password) != 0 && Passwd.OldPass != wallet.Password {
		walletlog.Error("ProcWalletSetPasswd Oldpass err!")
		return types.ErrVerifyOldpasswdFail
	}

	//        passwdhash
	newBatch := wallet.walletStore.NewBatch(true)
	err = wallet.walletStore.SetPasswordHash(Passwd.NewPass, newBatch)
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "SetPasswordHash err", err)
		return err
	}
	//
	err = wallet.walletStore.SetEncryptionFlag(newBatch)
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "SetEncryptionFlag err", err)
		return err
	}
	//  old    seed             seed
	seed, err := wallet.getSeed(Passwd.OldPass)
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd", "getSeed err", err)
		return err
	}
	ok, err := SaveSeedInBatch(wallet.walletStore.GetDB(), seed, Passwd.NewPass, newBatch)
	if !ok {
		walletlog.Error("ProcWalletSetPasswd", "SaveSeed err", err)
		return err
	}

	//                  ,  Account
	WalletAccStores, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(WalletAccStores) == 0 {
		walletlog.Error("ProcWalletSetPasswd", "GetAccountByPrefix:err", err)
	}

	for _, AccStore := range WalletAccStores {
		//  old Password
		storekey, err := common.FromHex(AccStore.GetPrivkey())
		if err != nil || len(storekey) == 0 {
			walletlog.Info("ProcWalletSetPasswd", "addr", AccStore.Addr, "FromHex err", err)
			continue
		}
		Decrypter := wcom.CBCDecrypterPrivkey([]byte(Passwd.OldPass), storekey)

		//
		Encrypter := wcom.CBCEncrypterPrivkey([]byte(Passwd.NewPass), Decrypter)
		AccStore.Privkey = common.ToHex(Encrypter)
		err = wallet.walletStore.SetWalletAccountInBatch(true, AccStore.Addr, AccStore, newBatch)
		if err != nil {
			walletlog.Info("ProcWalletSetPasswd", "addr", AccStore.Addr, "SetWalletAccount err", err)
		}
	}

	err = newBatch.Write()
	if err != nil {
		walletlog.Error("ProcWalletSetPasswd newBatch.Write", "err", err)
		return err
	}
	wallet.Password = Passwd.NewPass
	wallet.EncryptFlag = 1
	return nil
}

//ProcWalletLock
func (wallet *Wallet) ProcWalletLock() error {
	//         seed
	has, err := wallet.walletStore.HasSeed()
	if !has || err != nil {
		return types.ErrSaveSeedFirst
	}

	atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, 1)
	for _, policy := range wcom.PolicyContainer {
		policy.OnWalletLocked()
	}
	return nil
}

// ProcWalletUnLock
//input:
//type WalletUnLock struct {
//	Passwd  string
//	Timeout int64
//    Timeout  ，
func (wallet *Wallet) ProcWalletUnLock(WalletUnLock *types.WalletUnLock) error {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	//         seed
	has, err := wallet.walletStore.HasSeed()
	if !has || err != nil {
		return types.ErrSaveSeedFirst
	}
	//           passwd
	if len(wallet.Password) == 0 && wallet.EncryptFlag == 1 {
		isok := wallet.walletStore.VerifyPasswordHash(WalletUnLock.Passwd)
		if !isok {
			walletlog.Error("ProcWalletUnLock Verify Oldpasswd fail!")
			return types.ErrVerifyOldpasswdFail
		}
	}
	//       password
	if len(wallet.Password) != 0 && WalletUnLock.Passwd != wallet.Password {
		return types.ErrInputPassword
	}
	//            ,
	wallet.Password = WalletUnLock.Passwd
	//
	if !WalletUnLock.WalletOrTicket {
		//wallet.isTicketLocked = false
		atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 1, 0)
		if WalletUnLock.Timeout != 0 {
			wallet.resetTimeout(WalletUnLock.Timeout)
		}
	}
	for _, policy := range wcom.PolicyContainer {
		policy.OnWalletUnlocked(WalletUnLock)
	}
	return nil

}

//      ，
func (wallet *Wallet) resetTimeout(Timeout int64) {
	if wallet.timeout == nil {
		wallet.timeout = time.AfterFunc(time.Second*time.Duration(Timeout), func() {
			//wallet.isWalletLocked = true
			atomic.CompareAndSwapInt32(&wallet.isWalletLocked, 0, 1)
		})
	} else {
		wallet.timeout.Reset(time.Second * time.Duration(Timeout))
	}
}

//ProcWalletAddBlock wallet    blockchain   addblock  ，         tx    db
func (wallet *Wallet) ProcWalletAddBlock(block *types.BlockDetail) {
	if block == nil {
		walletlog.Error("ProcWalletAddBlock input para is nil!")
		return
	}
	//walletlog.Error("ProcWalletAddBlock", "height", block.GetBlock().GetHeight())
	types.AssertConfig(wallet.client)
	cfg := wallet.client.GetConfig()
	txlen := len(block.Block.GetTxs())
	newbatch := wallet.walletStore.GetBlockBatch(true)
	defer wallet.walletStore.FreeBlockBatch()
	for index := 0; index < txlen; index++ {
		tx := block.Block.Txs[index]
		execer := string(cfg.GetParaExec(tx.Execer))
		//
		if policy, ok := wcom.PolicyContainer[execer]; ok {
			wtxdetail := policy.OnAddBlockTx(block, tx, int32(index), newbatch)
			if wtxdetail == nil {
				continue
			}
			if len(wtxdetail.Fromaddr) > 0 {
				txdetailbyte, err := proto.Marshal(wtxdetail)
				if err != nil {
					walletlog.Error("ProcWalletAddBlock", "Marshal txdetail error", err, "Height", block.Block.Height, "index", index)
					continue
				}
				blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
				heightstr := fmt.Sprintf("%018d", blockheight)
				key := wcom.CalcTxKey(heightstr)
				newbatch.Set(key, txdetailbyte)
			}

		} else { //
			// TODO:         ，            ，
			//  from
			pubkey := block.Block.Txs[index].Signature.GetPubkey()
			addr := address.PubKeyToAddress(pubkey)
			param := &buildStoreWalletTxDetailParam{
				tokenname:  "",
				block:      block,
				tx:         tx,
				index:      index,
				newbatch:   newbatch,
				isprivacy:  false,
				addDelType: AddTx,
				//utxos:      nil,
			}
			//from addr
			fromaddress := addr.String()
			param.senderRecver = fromaddress
			if len(fromaddress) != 0 && wallet.AddrInWallet(fromaddress) {
				param.sendRecvFlag = sendTx
				wallet.buildAndStoreWalletTxDetail(param)
				walletlog.Debug("ProcWalletAddBlock", "fromaddress", fromaddress)
				continue
			}
			//toaddr            ，     para
			toaddr := tx.GetRealToAddr()
			if len(toaddr) != 0 && wallet.AddrInWallet(toaddr) {
				param.sendRecvFlag = recvTx
				wallet.buildAndStoreWalletTxDetail(param)
				walletlog.Debug("ProcWalletAddBlock", "toaddr", toaddr)
				continue
			}
		}
	}
	err := newbatch.Write()
	if err != nil {
		walletlog.Error("ProcWalletAddBlock newbatch.Write", "err", err)
		atomic.CompareAndSwapInt32(&wallet.fatalFailureFlag, 0, 1)
	}

	for _, policy := range wcom.PolicyContainer {
		policy.OnAddBlockFinish(block)
	}
}

//
type buildStoreWalletTxDetailParam struct {
	tokenname    string
	block        *types.BlockDetail
	tx           *types.Transaction
	index        int
	newbatch     dbm.Batch
	senderRecver string
	isprivacy    bool
	addDelType   int32
	sendRecvFlag int32
	//utxos        []*types.UTXO
}

func (wallet *Wallet) buildAndStoreWalletTxDetail(param *buildStoreWalletTxDetailParam) {
	blockheight := param.block.Block.Height*maxTxNumPerBlock + int64(param.index)
	heightstr := fmt.Sprintf("%018d", blockheight)
	walletlog.Debug("buildAndStoreWalletTxDetail", "heightstr", heightstr, "addDelType", param.addDelType)
	if AddTx == param.addDelType {
		var txdetail types.WalletTxDetail
		var Err error
		key := wcom.CalcTxKey(heightstr)
		txdetail.Tx = param.tx
		txdetail.Height = param.block.Block.Height
		txdetail.Index = int64(param.index)
		txdetail.Receipt = param.block.Receipts[param.index]
		txdetail.Blocktime = param.block.Block.BlockTime
		txdetail.Txhash = param.tx.Hash()
		txdetail.ActionName = txdetail.Tx.ActionName()
		txdetail.Amount, Err = param.tx.Amount()
		if Err != nil {
			walletlog.Error("buildAndStoreWalletTxDetail Amount err", "Height", param.block.Block.Height, "index", param.index)
		}
		txdetail.Fromaddr = param.senderRecver
		//txdetail.Spendrecv = param.utxos

		txdetailbyte, err := proto.Marshal(&txdetail)
		if err != nil {
			walletlog.Error("buildAndStoreWalletTxDetail Marshal txdetail err", "Height", param.block.Block.Height, "index", param.index)
			return
		}
		param.newbatch.Set(key, txdetailbyte)
	} else {
		param.newbatch.Delete(wcom.CalcTxKey(heightstr))
	}
}

//ProcWalletDelBlock wallet    blockchain   delblock  ，         tx  db
func (wallet *Wallet) ProcWalletDelBlock(block *types.BlockDetail) {
	if block == nil {
		walletlog.Error("ProcWalletDelBlock input para is nil!")
		return
	}
	//walletlog.Error("ProcWalletDelBlock", "height", block.GetBlock().GetHeight())
	types.AssertConfig(wallet.client)
	cfg := wallet.client.GetConfig()
	txlen := len(block.Block.GetTxs())
	newbatch := wallet.walletStore.GetBlockBatch(true)
	defer wallet.walletStore.FreeBlockBatch()
	for index := txlen - 1; index >= 0; index-- {
		blockheight := block.Block.Height*maxTxNumPerBlock + int64(index)
		heightstr := fmt.Sprintf("%018d", blockheight)
		tx := block.Block.Txs[index]

		execer := string(cfg.GetParaExec(tx.Execer))
		//
		if policy, ok := wcom.PolicyContainer[execer]; ok {
			wtxdetail := policy.OnDeleteBlockTx(block, tx, int32(index), newbatch)
			if wtxdetail == nil {
				continue
			}
			if len(wtxdetail.Fromaddr) > 0 {
				newbatch.Delete(wcom.CalcTxKey(heightstr))
			}

		} else { //
			// TODO:                      ，
			//  from
			pubkey := tx.Signature.GetPubkey()
			addr := address.PubKeyToAddress(pubkey)
			fromaddress := addr.String()
			if len(fromaddress) != 0 && wallet.AddrInWallet(fromaddress) {
				newbatch.Delete(wcom.CalcTxKey(heightstr))
				continue
			}
			//toaddr
			toaddr := tx.GetRealToAddr()
			if len(toaddr) != 0 && wallet.AddrInWallet(toaddr) {
				newbatch.Delete(wcom.CalcTxKey(heightstr))
			}
		}
	}
	err := newbatch.Write()
	if err != nil {
		walletlog.Error("ProcWalletDelBlock newbatch.Write", "err", err)
	}
	for _, policy := range wcom.PolicyContainer {
		policy.OnDeleteBlockFinish(block)
	}
}

// GetTxDetailByHashs
func (wallet *Wallet) GetTxDetailByHashs(ReqHashes *types.ReqHashes) {
	//  txhashs     txdetail
	msg := wallet.client.NewMessage("blockchain", types.EventGetTransactionByHash, ReqHashes)
	err := wallet.client.Send(msg, true)
	if err != nil {
		walletlog.Error("GetTxDetailByHashs Send EventGetTransactionByHash", "err", err)
		return
	}
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		walletlog.Error("GetTxDetailByHashs EventGetTransactionByHash", "err", err)
		return
	}
	TxDetails := resp.GetData().(*types.TransactionDetails)
	if TxDetails == nil {
		walletlog.Info("GetTxDetailByHashs TransactionDetails is nil")
		return
	}

	//                   wallet db
	newbatch := wallet.walletStore.NewBatch(true)
	for _, txdetal := range TxDetails.Txs {
		height := txdetal.GetHeight()
		txindex := txdetal.GetIndex()

		blockheight := height*maxTxNumPerBlock + txindex
		heightstr := fmt.Sprintf("%018d", blockheight)
		var txdetail types.WalletTxDetail
		txdetail.Tx = txdetal.GetTx()
		txdetail.Height = txdetal.GetHeight()
		txdetail.Index = txdetal.GetIndex()
		txdetail.Receipt = txdetal.GetReceipt()
		txdetail.Blocktime = txdetal.GetBlocktime()
		txdetail.Amount = txdetal.GetAmount()
		txdetail.Fromaddr = txdetal.GetFromaddr()
		txdetail.ActionName = txdetal.GetTx().ActionName()

		txdetailbyte, err := proto.Marshal(&txdetail)
		if err != nil {
			walletlog.Error("GetTxDetailByHashs Marshal txdetail err", "Height", height, "index", txindex)
			return
		}
		newbatch.Set(wcom.CalcTxKey(heightstr), txdetailbyte)
	}
	err = newbatch.Write()
	if err != nil {
		walletlog.Error("GetTxDetailByHashs newbatch.Write", "err", err)
	}
}

//       seed  ,
func (wallet *Wallet) genSeed(lang int32) (*types.ReplySeed, error) {
	seed, err := CreateSeed("", lang)
	if err != nil {
		walletlog.Error("genSeed", "CreateSeed err", err)
		return nil, err
	}
	var ReplySeed types.ReplySeed
	ReplySeed.Seed = seed
	return &ReplySeed, nil
}

// GenSeed
func (wallet *Wallet) GenSeed(lang int32) (*types.ReplySeed, error) {
	return wallet.genSeed(lang)
}

//GetSeed   seed  ,
func (wallet *Wallet) GetSeed(password string) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	return wallet.getSeed(password)
}

//  seed  ,
func (wallet *Wallet) getSeed(password string) (string, error) {
	ok, err := wallet.checkWalletStatus()
	if !ok {
		return "", err
	}

	seed, err := GetSeed(wallet.walletStore.GetDB(), password)
	if err != nil {
		walletlog.Error("getSeed", "GetSeed err", err)
		return "", err
	}
	return seed, nil
}

// SaveSeed
func (wallet *Wallet) SaveSeed(password string, seed string) (bool, error) {
	return wallet.saveSeed(password, seed)
}

//  seed       ,          ,          seed
func (wallet *Wallet) saveSeed(password string, seed string) (bool, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	//              seed，              ，          seed
	exit, err := wallet.walletStore.HasSeed()
	if exit && err == nil {
		return false, types.ErrSeedExist
	}
	//     ，seed       12
	if len(password) == 0 || len(seed) == 0 {
		return false, types.ErrInvalidParam
	}

	//
	if !isValidPassWord(password) {
		return false, types.ErrInvalidPassWord
	}

	seedarry := strings.Fields(seed)
	curseedlen := len(seedarry)
	if curseedlen < SaveSeedLong {
		walletlog.Error("saveSeed VeriySeedwordnum", "curseedlen", curseedlen, "SaveSeedLong", SaveSeedLong)
		return false, types.ErrSeedWordNum
	}

	var newseed string
	for index, seedstr := range seedarry {
		if index != curseedlen-1 {
			newseed += seedstr + " "
		} else {
			newseed += seedstr
		}
	}

	//  seed           ，     seed
	have, err := VerifySeed(newseed, wallet.SignType, wallet.CoinType)
	if !have {
		walletlog.Error("saveSeed VerifySeed", "err", err)
		return false, types.ErrSeedWord
	}
	//    seed password
	newBatch := wallet.walletStore.NewBatch(true)
	err = wallet.walletStore.SetPasswordHash(password, newBatch)
	if err != nil {
		walletlog.Error("saveSeed", "SetPasswordHash err", err)
		return false, err
	}
	//
	err = wallet.walletStore.SetEncryptionFlag(newBatch)
	if err != nil {
		walletlog.Error("saveSeed", "SetEncryptionFlag err", err)
		return false, err
	}

	ok, err := SaveSeedInBatch(wallet.walletStore.GetDB(), newseed, password, newBatch)
	if !ok {
		walletlog.Error("saveSeed", "SaveSeed err", err)
		return false, err
	}

	err = newBatch.Write()
	if err != nil {
		walletlog.Error("saveSeed newBatch.Write", "err", err)
		return false, err
	}
	wallet.Password = password
	wallet.EncryptFlag = 1
	return true, nil
}

//ProcDumpPrivkey
func (wallet *Wallet) ProcDumpPrivkey(addr string) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
	if !ok {
		return "", err
	}
	if len(addr) == 0 {
		walletlog.Error("ProcDumpPrivkey input para is nil!")
		return "", types.ErrInvalidParam
	}

	priv, err := wallet.getPrivKeyByAddr(addr)
	if err != nil {
		return "", err
	}
	return common.ToHex(priv.Bytes()), nil
	//return strings.ToUpper(common.ToHex(priv.Bytes())), nil
}

//                 ，
func (wallet *Wallet) setFatalFailure(reportErrEvent *types.ReportErrEvent) {

	walletlog.Error("setFatalFailure", "reportErrEvent", reportErrEvent.String())
	if reportErrEvent.Error == "ErrDataBaseDamage" {
		atomic.StoreInt32(&wallet.fatalFailureFlag, 1)
	}
}

func (wallet *Wallet) getFatalFailure() int32 {
	return atomic.LoadInt32(&wallet.fatalFailureFlag)
}

//       ,     8-30   。     +
func isValidPassWord(password string) bool {
	pwLen := len(password)
	if pwLen < 8 || pwLen > 30 {
		return false
	}

	var char bool
	var digit bool
	for _, s := range password {
		if unicode.IsLetter(s) {
			char = true
		} else if unicode.IsDigit(s) {
			digit = true
		} else {
			return false
		}
	}
	return char && digit
}

// CreateNewAccountByIndex   index      ，        。
func (wallet *Wallet) createNewAccountByIndex(index uint32) (string, error) {
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
	if !ok {
		return "", err
	}

	if !isValidIndex(index) {
		walletlog.Error("createNewAccountByIndex index err", "index", index)
		return "", types.ErrInvalidParam
	}

	//          ，
	airDropAddr, err := wallet.walletStore.GetAirDropIndex()
	if airDropAddr != "" && err == nil {
		priv, err := wallet.getPrivKeyByAddr(airDropAddr)
		if err != nil {
			return "", err
		}
		return common.ToHex(priv.Bytes()), nil
	}

	var addr string
	var privkeybyte []byte
	var HexPubkey string
	var isUsed bool

	cointype := wallet.GetCoinType()

	//  seed    ,           seed    seed
	seed, err := wallet.getSeed(wallet.Password)
	if err != nil {
		walletlog.Error("createNewAccountByIndex", "getSeed err", err)
		return "", err
	}

	//     index      ，       ，
	privkeyhex, err := GetPrivkeyBySeed(wallet.walletStore.GetDB(), seed, index, wallet.SignType, wallet.CoinType)
	if err != nil {
		walletlog.Error("createNewAccountByIndex", "GetPrivkeyBySeed err", err)
		return "", err
	}
	privkeybyte, err = common.FromHex(privkeyhex)
	if err != nil || len(privkeybyte) == 0 {
		walletlog.Error("createNewAccountByIndex", "FromHex err", err)
		return "", err
	}

	pub, err := bipwallet.PrivkeyToPub(cointype, uint32(wallet.SignType), privkeybyte)
	if err != nil {
		seedlog.Error("createNewAccountByIndex PrivkeyToPub", "err", err)
		return "", types.ErrPrivkeyToPub
	}

	HexPubkey = hex.EncodeToString(pub)

	addr, err = bipwallet.PubToAddress(pub)
	if err != nil {
		seedlog.Error("createNewAccountByIndex PubToAddress", "err", err)
		return "", types.ErrPrivkeyToPub
	}
	//                 ，              ，
	//          ,
	account, err := wallet.walletStore.GetAccountByAddr(addr)
	if account != nil && err == nil {
		isUsed = true
	}

	//
	if !isUsed {
		Account := types.Account{
			Addr:     addr,
			Currency: 0,
			Balance:  0,
			Frozen:   0,
		}
		//    label
		Label := "airdropaddr"
		for {
			i := 0
			WalletAccStores, err := wallet.walletStore.GetAccountByLabel(Label)
			if WalletAccStores != nil && err == nil {
				walletlog.Debug("createNewAccountByIndex Label is exist in wallet!", "WalletAccStores", WalletAccStores)
				i++
				Label = Label + fmt.Sprintf("%d", i)
			} else {
				break
			}
		}

		walletAccount := types.WalletAccount{
			Acc:   &Account,
			Label: Label,
		}

		//     password      aes cbc
		Encrypted := wcom.CBCEncrypterPrivkey([]byte(wallet.Password), privkeybyte)

		var WalletAccStore types.WalletAccountStore
		WalletAccStore.Privkey = common.ToHex(Encrypted)
		WalletAccStore.Label = Label
		WalletAccStore.Addr = addr

		//       wallet
		err = wallet.walletStore.SetWalletAccount(false, Account.Addr, &WalletAccStore)
		if err != nil {
			return "", err
		}

		//            account
		addrs := make([]string, 1)
		addrs[0] = addr
		accounts, err := wallet.accountdb.LoadAccounts(wallet.api, addrs)
		if err != nil {
			walletlog.Error("createNewAccountByIndex", "LoadAccounts err", err)
			return "", err
		}
		//
		if len(accounts[0].Addr) == 0 {
			accounts[0].Addr = addr
		}
		walletAccount.Acc = accounts[0]

		// blockchain    Account.Addr
		for _, policy := range wcom.PolicyContainer {
			policy.OnCreateNewAccount(walletAccount.Acc)
		}
	}
	//
	airfrop := &wcom.AddrInfo{
		Index:  index,
		Addr:   addr,
		Pubkey: HexPubkey,
	}
	err = wallet.walletStore.SetAirDropIndex(airfrop)
	if err != nil {
		walletlog.Error("createNewAccountByIndex", "SetAirDropIndex err", err)
	}
	return privkeyhex, nil
}

//isValidIndex  index
func isValidIndex(index uint32) bool {
	if types.AirDropMinIndex <= index && index <= types.AirDropMaxIndex {
		return true
	}
	return false
}

//ProcDumpPrivkeysFile
func (wallet *Wallet) ProcDumpPrivkeysFile(fileName, passwd string) error {
	_, err := os.Stat(fileName)
	if err == nil {
		walletlog.Error("ProcDumpPrivkeysFile file already exists!", "fileName", fileName)
		return types.ErrFileExists
	}

	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
	if !ok {
		return err
	}

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		walletlog.Error("ProcDumpPrivkeysFile create file error!", "fileName", fileName, "err", err)
		return err
	}
	defer f.Close()

	accounts, err := wallet.walletStore.GetAccountByPrefix("Account")
	if err != nil || len(accounts) == 0 {
		walletlog.Info("ProcDumpPrivkeysFile GetWalletAccounts", "GetAccountByPrefix:err", err)
		return err
	}

	for i, acc := range accounts {
		priv, err := wallet.getPrivKeyByAddr(acc.Addr)
		if err != nil {
			walletlog.Info("getPrivKeyByAddr", acc.Addr, err)
			continue
		}

		privkey := common.ToHex(priv.Bytes())
		content := privkey + "& *.prickey.+.label.* &" + acc.Label

		Encrypter, err := AesgcmEncrypter([]byte(passwd), []byte(content))
		if err != nil {
			walletlog.Error("ProcDumpPrivkeysFile AesgcmEncrypter fileContent error!", "fileName", fileName, "err", err)
			continue
		}

		f.WriteString(string(Encrypter))

		if i < len(accounts)-1 {
			f.WriteString("&ffzm.&**&")
		}
	}

	return nil
}

// ProcImportPrivkeysFile
//input:
//type  struct {
//	fileName string
//  passwd   string
//    ，
func (wallet *Wallet) ProcImportPrivkeysFile(fileName, passwd string) error {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		walletlog.Error("ProcImportPrivkeysFile file is not exist!", "fileName", fileName)
		return err
	}

	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()

	ok, err := wallet.checkWalletStatus()
	if !ok {
		return err
	}

	f, err := os.Open(fileName)
	if err != nil {
		walletlog.Error("ProcImportPrivkeysFile Open file error", "fileName", fileName, "err", err)
		return err
	}
	defer f.Close()

	fileContent, err := ioutil.ReadAll(f)
	if err != nil {
		walletlog.Error("ProcImportPrivkeysFile read file error", "fileName", fileName, "err", err)
		return err
	}
	accounts := strings.Split(string(fileContent), "&ffzm.&**&")
	for _, value := range accounts {
		Decrypter, err := AesgcmDecrypter([]byte(passwd), []byte(value))
		if err != nil {
			walletlog.Error("ProcImportPrivkeysFile AesgcmDecrypter fileContent error", "fileName", fileName, "err", err)
			return types.ErrVerifyOldpasswdFail
		}

		acc := strings.Split(string(Decrypter), "& *.prickey.+.label.* &")
		if len(acc) != 2 {
			walletlog.Error("ProcImportPrivkeysFile len(acc) != 2, File format error.", "Decrypter", string(Decrypter), "len", len(acc))
			continue
		}
		privKey := acc[0]
		label := acc[1]

		//  label
		Account, err := wallet.walletStore.GetAccountByLabel(label)
		if Account != nil && err == nil {
			walletlog.Info("ProcImportPrivKey Label is exist in wallet, label = label + _2!")
			label = label + "_2"
		}

		PrivKey := &types.ReqWalletImportPrivkey{
			Privkey: privKey,
			Label:   label,
		}

		_, err = wallet.procImportPrivKey(PrivKey)
		if err == types.ErrPrivkeyExist {
			//            ?
			// wallet.ProcWalletSetLabel()
		} else if err != nil {
			walletlog.Info("ProcImportPrivKey procImportPrivKey error")
			return err
		}
	}

	return nil
}
