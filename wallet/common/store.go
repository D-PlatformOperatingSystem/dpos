// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	"github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/common/version"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/golang/protobuf/proto"
)

var (
	storelog = log15.New("wallet", "store")
)

//AddrInfo   seed  index       ，
type AddrInfo struct {
	Index  uint32 `json:"index,omitempty"`
	Addr   string `json:"addr,omitempty"`
	Pubkey string `json:"pubkey,omitempty"`
}

// NewStore
func NewStore(db db.DB) *Store {
	return &Store{db: db, blkBatch: db.NewBatch(true)}
}

// Store           ，
type Store struct {
	db        db.DB
	blkBatch  db.Batch
	batchLock sync.Mutex
}

// Close
func (store *Store) Close() {
	store.db.Close()
}

// GetDB
func (store *Store) GetDB() db.DB {
	return store.db
}

// NewBatch
func (store *Store) NewBatch(sync bool) db.Batch {
	return store.db.NewBatch(sync)
}

// GetBlockBatch
func (store *Store) GetBlockBatch(sync bool) db.Batch {
	store.batchLock.Lock()
	store.blkBatch.Reset()
	store.blkBatch.UpdateWriteSync(sync)
	return store.blkBatch
}

//FreeBlockBatch free
func (store *Store) FreeBlockBatch() {
	store.batchLock.Unlock()
}

// Get
func (store *Store) Get(key []byte) ([]byte, error) {
	return store.db.Get(key)
}

// Set
func (store *Store) Set(key []byte, value []byte) (err error) {
	return store.db.Set(key, value)
}

// NewListHelper
func (store *Store) NewListHelper() *db.ListHelper {
	return db.NewListHelper(store.db)
}

// GetAccountByte     byte
func (store *Store) GetAccountByte(update bool, addr string, account *types.WalletAccountStore) ([]byte, error) {
	if len(addr) == 0 {
		storelog.Error("GetAccountByte addr is nil")
		return nil, types.ErrInvalidParam
	}
	if account == nil {
		storelog.Error("GetAccountByte account is nil")
		return nil, types.ErrInvalidParam
	}

	timestamp := fmt.Sprintf("%018d", types.Now().Unix())
	//          Accountkey
	if update {
		timestamp = account.TimeStamp
	}
	account.TimeStamp = timestamp

	accountbyte, err := proto.Marshal(account)
	if err != nil {
		storelog.Error("GetAccountByte", " proto.Marshal error", err)
		return nil, types.ErrMarshal
	}
	return accountbyte, nil
}

// SetWalletAccount
func (store *Store) SetWalletAccount(update bool, addr string, account *types.WalletAccountStore) error {
	accountbyte, err := store.GetAccountByte(update, addr, account)
	if err != nil {
		storelog.Error("SetWalletAccount", "GetAccountByte error", err)
		return err
	}
	//         ，Account，Addr，Label，
	newbatch := store.NewBatch(true)
	newbatch.Set(CalcAccountKey(account.TimeStamp, addr), accountbyte)
	newbatch.Set(CalcAddrKey(addr), accountbyte)
	newbatch.Set(CalcLabelKey(account.GetLabel()), accountbyte)
	return newbatch.Write()
}

// SetWalletAccountInBatch
func (store *Store) SetWalletAccountInBatch(update bool, addr string, account *types.WalletAccountStore, newbatch db.Batch) error {
	accountbyte, err := store.GetAccountByte(update, addr, account)
	if err != nil {
		storelog.Error("SetWalletAccount", "GetAccountByte error", err)
		return err
	}
	//         ，Account，Addr，Label，
	newbatch.Set(CalcAccountKey(account.TimeStamp, addr), accountbyte)
	newbatch.Set(CalcAddrKey(addr), accountbyte)
	newbatch.Set(CalcLabelKey(account.GetLabel()), accountbyte)
	return nil
}

// GetAccountByAddr
func (store *Store) GetAccountByAddr(addr string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(addr) == 0 {
		storelog.Error("GetAccountByAddr addr is empty")
		return nil, types.ErrInvalidParam
	}
	data, err := store.Get(CalcAddrKey(addr))
	if data == nil || err != nil {
		if err != db.ErrNotFoundInDb {
			storelog.Debug("GetAccountByAddr addr", "err", err)
		}
		return nil, types.ErrAddrNotExist
	}
	err = proto.Unmarshal(data, &account)
	if err != nil {
		storelog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &account, nil
}

// GetAccountByLabel
func (store *Store) GetAccountByLabel(label string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(label) == 0 {
		storelog.Error("GetAccountByLabel label is empty")
		return nil, types.ErrInvalidParam
	}
	data, err := store.Get(CalcLabelKey(label))
	if data == nil || err != nil {
		if err != db.ErrNotFoundInDb {
			storelog.Error("GetAccountByLabel label", "err", err)
		}
		return nil, types.ErrLabelNotExist
	}
	err = proto.Unmarshal(data, &account)
	if err != nil {
		storelog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &account, nil
}

// GetAccountByPrefix
func (store *Store) GetAccountByPrefix(addr string) ([]*types.WalletAccountStore, error) {
	if len(addr) == 0 {
		storelog.Error("GetAccountByPrefix addr is nil")
		return nil, types.ErrInvalidParam
	}
	list := store.NewListHelper()
	accbytes := list.PrefixScan([]byte(addr))
	if len(accbytes) == 0 {
		storelog.Debug("GetAccountByPrefix addr not exist")
		return nil, types.ErrAccountNotExist
	}
	WalletAccountStores := make([]*types.WalletAccountStore, len(accbytes))
	for index, accbyte := range accbytes {
		var walletaccount types.WalletAccountStore
		err := proto.Unmarshal(accbyte, &walletaccount)
		if err != nil {
			storelog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		WalletAccountStores[index] = &walletaccount
	}
	return WalletAccountStores, nil
}

//GetTxDetailByIter        key：height*100000+index             count
func (store *Store) GetTxDetailByIter(TxList *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	var txDetails types.WalletTxDetails
	if TxList == nil {
		storelog.Error("GetTxDetailByIter TxList is nil")
		return nil, types.ErrInvalidParam
	}

	var txbytes [][]byte
	//FromTx      ，
	//Direction == 0           count
	//Direction == 1           count
	if len(TxList.FromTx) == 0 {
		list := store.NewListHelper()
		if TxList.Direction == 0 {
			txbytes = list.IteratorScanFromLast(CalcTxKey(""), TxList.Count, db.ListDESC)
		} else {
			txbytes = list.IteratorScanFromFirst(CalcTxKey(""), TxList.Count, db.ListASC)
		}
		if len(txbytes) == 0 {
			storelog.Error("GetTxDetailByIter IteratorScanFromLast does not exist tx!")
			return nil, types.ErrTxNotExist
		}
	} else {
		list := store.NewListHelper()
		txbytes = list.IteratorScan(CalcTxKey(""), CalcTxKey(string(TxList.FromTx)), TxList.Count, TxList.Direction)
		if len(txbytes) == 0 {
			storelog.Error("GetTxDetailByIter IteratorScan does not exist tx!")
			return nil, types.ErrTxNotExist
		}
	}

	txDetails.TxDetails = make([]*types.WalletTxDetail, len(txbytes))
	for index, txdetailbyte := range txbytes {
		var txdetail types.WalletTxDetail
		err := proto.Unmarshal(txdetailbyte, &txdetail)
		if err != nil {
			storelog.Error("GetTxDetailByIter", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		txdetail.Txhash = txdetail.GetTx().Hash()
		txDetails.TxDetails[index] = &txdetail
	}
	return &txDetails, nil
}

// SetEncryptionFlag
func (store *Store) SetEncryptionFlag(batch db.Batch) error {
	var flag int64 = 1
	data, err := json.Marshal(flag)
	if err != nil {
		storelog.Error("SetEncryptionFlag marshal flag", "err", err)
		return types.ErrMarshal
	}

	batch.Set(CalcEncryptionFlag(), data)
	return nil
}

// GetEncryptionFlag
func (store *Store) GetEncryptionFlag() int64 {
	var flag int64
	data, err := store.Get(CalcEncryptionFlag())
	if data == nil || err != nil {
		data, err = store.Get(CalckeyEncryptionCompFlag())
		if data == nil || err != nil {
			return 0
		}
	}
	err = json.Unmarshal(data, &flag)
	if err != nil {
		storelog.Error("GetEncryptionFlag unmarshal", "err", err)
		return 0
	}
	return flag
}

// SetPasswordHash
func (store *Store) SetPasswordHash(password string, batch db.Batch) error {
	var WalletPwHash types.WalletPwHash
	//
	randstr := fmt.Sprintf("domchain:$@%s", crypto.CRandHex(16))
	WalletPwHash.Randstr = randstr

	//  password          hash
	pwhashstr := fmt.Sprintf("%s:%s", password, WalletPwHash.Randstr)
	pwhash := sha256.Sum256([]byte(pwhashstr))
	WalletPwHash.PwHash = pwhash[:]

	pwhashbytes, err := json.Marshal(WalletPwHash)
	if err != nil {
		storelog.Error("SetEncryptionFlag marshal flag", "err", err)
		return types.ErrMarshal
	}
	batch.Set(CalcPasswordHash(), pwhashbytes)
	return nil
}

// VerifyPasswordHash
func (store *Store) VerifyPasswordHash(password string) bool {
	var WalletPwHash types.WalletPwHash
	pwhashbytes, err := store.Get(CalcPasswordHash())
	if pwhashbytes == nil || err != nil {
		return false
	}
	err = json.Unmarshal(pwhashbytes, &WalletPwHash)
	if err != nil {
		storelog.Error("VerifyPasswordHash unmarshal", "err", err)
		return false
	}
	pwhashstr := fmt.Sprintf("%s:%s", password, WalletPwHash.Randstr)
	pwhash := sha256.Sum256([]byte(pwhashstr))
	Pwhash := pwhash[:]
	//        pwhash
	return bytes.Equal(WalletPwHash.GetPwHash(), Pwhash)
}

// DelAccountByLabel       ，
func (store *Store) DelAccountByLabel(label string) {
	err := store.GetDB().DeleteSync(CalcLabelKey(label))
	if err != nil {
		storelog.Error("DelAccountByLabel", "err", err)
	}
}

//SetWalletVersion
func (store *Store) SetWalletVersion(ver int64) error {
	data, err := json.Marshal(ver)
	if err != nil {
		storelog.Error("SetWalletVerKey marshal version", "err", err)
		return types.ErrMarshal
	}
	return store.GetDB().SetSync(version.WalletVerKey, data)
}

// GetWalletVersion   wallet
func (store *Store) GetWalletVersion() int64 {
	var ver int64
	data, err := store.Get(version.WalletVerKey)
	if data == nil || err != nil {
		return 0
	}
	err = json.Unmarshal(data, &ver)
	if err != nil {
		storelog.Error("GetWalletVersion unmarshal", "err", err)
		return 0
	}
	return ver
}

//HasSeed           seed
func (store *Store) HasSeed() (bool, error) {
	seed, err := store.Get(CalcWalletSeed())
	if len(seed) == 0 || err != nil {
		return false, types.ErrSeedExist
	}
	return true, nil
}

// GetAirDropIndex     index
func (store *Store) GetAirDropIndex() (string, error) {
	var airDrop AddrInfo
	data, err := store.Get(CalcAirDropIndex())
	if data == nil || err != nil {
		if err != db.ErrNotFoundInDb {
			storelog.Debug("GetAirDropIndex", "err", err)
		}
		return "", types.ErrAddrNotExist
	}
	err = json.Unmarshal(data, &airDrop)
	if err != nil {
		storelog.Error("GetWalletVersion unmarshal", "err", err)
		return "", err
	}
	storelog.Debug("GetAirDropIndex ", "airDrop", airDrop)
	return airDrop.Addr, nil
}

// SetAirDropIndex     index
func (store *Store) SetAirDropIndex(airDropIndex *AddrInfo) error {
	data, err := json.Marshal(airDropIndex)
	if err != nil {
		storelog.Error("SetAirDropIndex marshal", "err", err)
		return err
	}

	storelog.Debug("SetAirDropIndex ", "airDropIndex", airDropIndex)
	return store.GetDB().SetSync(CalcAirDropIndex(), data)
}
