// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"time"

	lru "github.com/hashicorp/golang-lru"

	"strconv"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
)

var (
	bCoins   = []byte("coins")
	bToken   = []byte("token")
	withdraw = "withdraw"
	txCache  *lru.Cache
)

func init() {
	var err error
	txCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
}

//TxCacheGet      cache      ，
func TxCacheGet(tx *Transaction) (*TransactionCache, bool) {
	txc, ok := txCache.Get(tx)
	if !ok {
		return nil, ok
	}
	return txc.(*TransactionCache), ok
}

//TxCacheSet    cache
func TxCacheSet(tx *Transaction, txc *TransactionCache) {
	if txc == nil {
		txCache.Remove(tx)
		return
	}
	txCache.Add(tx, txc)
}

// CreateTxGroup      , feeRate      ,       GetProperFee
func CreateTxGroup(txs []*Transaction, feeRate int64) (*Transactions, error) {
	if len(txs) < 2 {
		return nil, ErrTxGroupCountLessThanTwo
	}
	txgroup := &Transactions{}
	txgroup.Txs = txs
	totalfee := int64(0)
	minfee := int64(0)
	header := txs[0].Hash()
	for i := len(txs) - 1; i >= 0; i-- {
		txs[i].GroupCount = int32(len(txs))
		totalfee += txs[i].GetFee()
		// Header Fee     GetRealFee  Size   ，Fee   0     ，size      ，header       common.Sha256Len+2
		//       Header     ， Next   ，      ，      ，txs[0].fee         ，
		txs[i].Header = header
		if i == 0 {
			// txs[0].fee       ，          fee，  >=check      fee，  size  10   ， 1000
			txs[i].Fee = 1 << 62
		} else {
			txs[i].Fee = 0
		}
		realfee, err := txs[i].GetRealFee(feeRate)
		if err != nil {
			return nil, err
		}
		minfee += realfee
		if i == 0 {
			if totalfee < minfee {
				totalfee = minfee
			}
			txs[0].Fee = totalfee
			header = txs[0].Hash()
		} else {
			txs[i].Fee = 0
			txs[i-1].Next = txs[i].Hash()
		}
	}
	for i := 0; i < len(txs); i++ {
		txs[i].Header = header
	}
	return txgroup, nil
}

//Tx          ，        。
//
func (txgroup *Transactions) Tx() *Transaction {
	if len(txgroup.GetTxs()) < 2 {
		return nil
	}
	headtx := txgroup.GetTxs()[0]
	//       tx
	copytx := *headtx
	data := Encode(txgroup)
	//  header       Hash
	copytx.Header = data
	return &copytx
}

//GetTxGroup
func (txgroup *Transactions) GetTxGroup() *Transactions {
	return txgroup
}

//SignN       n
func (txgroup *Transactions) SignN(n int, ty int32, priv crypto.PrivKey) error {
	if n >= len(txgroup.GetTxs()) {
		return ErrIndex
	}
	txgroup.GetTxs()[n].Sign(ty, priv)
	return nil
}

//CheckSign
func (txgroup *Transactions) CheckSign() bool {
	txs := txgroup.Txs
	for i := 0; i < len(txs); i++ {
		if !txs[i].checkSign() {
			return false
		}
	}
	return true
}

//RebuiltGroup
func (txgroup *Transactions) RebuiltGroup() {
	header := txgroup.Txs[0].Hash()
	for i := len(txgroup.Txs) - 1; i >= 0; i-- {
		txgroup.Txs[i].Header = header
		if i == 0 {
			header = txgroup.Txs[0].Hash()
		} else {
			txgroup.Txs[i-1].Next = txgroup.Txs[i].Hash()
		}
	}
	for i := 0; i < len(txgroup.Txs); i++ {
		txgroup.Txs[i].Header = header
	}
}

//SetExpire
func (txgroup *Transactions) SetExpire(cfg *DplatformOSConfig, n int, expire time.Duration) {
	if n >= len(txgroup.GetTxs()) {
		return
	}
	txgroup.GetTxs()[n].SetExpire(cfg, expire)
}

//IsExpire
func (txgroup *Transactions) IsExpire(cfg *DplatformOSConfig, height, blocktime int64) bool {
	txs := txgroup.Txs
	for i := 0; i < len(txs); i++ {
		if txs[i].isExpire(cfg, height, blocktime) {
			return true
		}
	}
	return false
}

//CheckWithFork  fork
func (txgroup *Transactions) CheckWithFork(cfg *DplatformOSConfig, checkFork, paraFork bool, height, minfee, maxFee int64) error {
	txs := txgroup.Txs
	if len(txs) < 2 {
		return ErrTxGroupCountLessThanTwo
	}
	para := make(map[string]bool)
	for i := 0; i < len(txs); i++ {
		if txs[i] == nil {
			return ErrTxGroupEmpty
		}
		err := txs[i].check(cfg, height, 0, maxFee)
		if err != nil {
			return err
		}
		if title, ok := GetParaExecTitleName(string(txs[i].Execer)); ok {
			para[title] = true
		}
	}
	//txgroup            ,     txgroup       tx
	//                            ，           fork
	if paraFork {
		if len(para) > 1 {
			tlog.Info("txgroup has multi para transaction")
			return ErrTxGroupParaCount
		}
		if len(para) > 0 {
			for _, tx := range txs {
				if !IsParaExecName(string(tx.Execer)) {
					tlog.Error("para txgroup has main chain transaction")
					return ErrTxGroupParaMainMixed
				}
			}
		}
	}

	for i := 1; i < len(txs); i++ {
		if txs[i].Fee != 0 {
			return ErrTxGroupFeeNotZero
		}
	}
	//  txs[0]
	totalfee := int64(0)
	for i := 0; i < len(txs); i++ {
		fee, err := txs[i].GetRealFee(minfee)
		if err != nil {
			return err
		}
		totalfee += fee
	}
	if txs[0].Fee < totalfee {
		return ErrTxFeeTooLow
	}
	if txs[0].Fee > maxFee && maxFee > 0 && checkFork {
		return ErrTxFeeTooHigh
	}
	//  hash
	for i := 0; i < len(txs); i++ {
		//          hash
		if i == 0 {
			if !bytes.Equal(txs[i].Hash(), txs[i].Header) {
				return ErrTxGroupHeader
			}
		} else {
			if !bytes.Equal(txs[0].Header, txs[i].Header) {
				return ErrTxGroupHeader
			}
		}
		//  group count
		if txs[i].GroupCount > MaxTxGroupSize {
			return ErrTxGroupCountBigThanMaxSize
		}
		if txs[i].GroupCount != int32(len(txs)) {
			return ErrTxGroupCount
		}
		//  next
		if i < len(txs)-1 {
			if !bytes.Equal(txs[i].Next, txs[i+1].Hash()) {
				return ErrTxGroupNext
			}
		} else {
			if txs[i].Next != nil {
				return ErrTxGroupNext
			}
		}
	}
	return nil
}

//Check height == 0    ，
func (txgroup *Transactions) Check(cfg *DplatformOSConfig, height, minfee, maxFee int64) error {
	paraFork := cfg.IsFork(height, "ForkTxGroupPara")
	checkFork := cfg.IsFork(height, "ForkBlockCheck")
	return txgroup.CheckWithFork(cfg, checkFork, paraFork, height, minfee, maxFee)
}

//TransactionCache
type TransactionCache struct {
	*Transaction
	txGroup *Transactions
	hash    []byte
	size    int
	signok  int   //init 0, ok 1, err 2
	checkok error //init 0, ok 1, err 2
	checked bool
	payload reflect.Value
	plname  string
	plerr   error
}

//NewTransactionCache new
func NewTransactionCache(tx *Transaction) *TransactionCache {
	return &TransactionCache{Transaction: tx}
}

//Hash   hash
func (tx *TransactionCache) Hash() []byte {
	if tx.hash == nil {
		tx.hash = tx.Transaction.Hash()
	}
	return tx.hash
}

//SetPayloadValue   payload  cache
func (tx *TransactionCache) SetPayloadValue(plname string, payload reflect.Value, plerr error) {
	tx.payload = payload
	tx.plerr = plerr
	tx.plname = plname
}

//GetPayloadValue   payload  cache
func (tx *TransactionCache) GetPayloadValue() (plname string, payload reflect.Value, plerr error) {
	if tx.plerr != nil || tx.plname != "" {
		return tx.plname, tx.payload, tx.plerr
	}
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		tx.SetPayloadValue("", reflect.ValueOf(nil), ErrExecNotFound)
		return "", reflect.ValueOf(nil), ErrExecNotFound
	}
	plname, payload, plerr = exec.DecodePayloadValue(tx.Tx())
	tx.SetPayloadValue(plname, payload, plerr)
	return
}

//Size
func (tx *TransactionCache) Size() int {
	if tx.size == 0 {
		tx.size = Size(tx.Tx())
	}
	return tx.size
}

//Tx      tx
func (tx *TransactionCache) Tx() *Transaction {
	return tx.Transaction
}

//Check
func (tx *TransactionCache) Check(cfg *DplatformOSConfig, height, minfee, maxFee int64) error {
	if !tx.checked {
		tx.checked = true
		txs, err := tx.GetTxGroup()
		if err != nil {
			tx.checkok = err
			return err
		}
		if txs == nil {
			tx.checkok = tx.check(cfg, height, minfee, maxFee)
		} else {
			tx.checkok = txs.Check(cfg, height, minfee, maxFee)
		}
	}
	return tx.checkok
}

//GetTotalFee
func (tx *TransactionCache) GetTotalFee(minFee int64) (int64, error) {
	txgroup, err := tx.GetTxGroup()
	if err != nil {
		tx.checkok = err
		return 0, err
	}
	var totalfee int64
	if txgroup == nil {
		return tx.GetRealFee(minFee)
	}
	txs := txgroup.Txs
	for i := 0; i < len(txs); i++ {
		fee, err := txs[i].GetRealFee(minFee)
		if err != nil {
			return 0, err
		}
		totalfee += fee
	}
	return totalfee, nil
}

//GetTxGroup
func (tx *TransactionCache) GetTxGroup() (*Transactions, error) {
	var err error
	if tx.txGroup == nil {
		tx.txGroup, err = tx.Transaction.GetTxGroup()
		if err != nil {
			return nil, err
		}
	}
	return tx.txGroup, nil
}

//CheckSign
func (tx *TransactionCache) CheckSign() bool {
	if tx.signok == 0 {
		tx.signok = 2
		group, err := tx.GetTxGroup()
		if err != nil {
			return false
		}
		if group == nil {
			// group，
			if ok := tx.checkSign(); ok {
				tx.signok = 1
			}
		} else {
			if ok := group.CheckSign(); ok {
				tx.signok = 1
			}
		}
	}
	return tx.signok == 1
}

//TxsToCache
func TxsToCache(txs []*Transaction) (caches []*TransactionCache) {
	caches = make([]*TransactionCache, len(txs))
	for i := 0; i < len(caches); i++ {
		caches[i] = NewTransactionCache(txs[i])
	}
	return caches
}

//CacheToTxs
func CacheToTxs(caches []*TransactionCache) (txs []*Transaction) {
	txs = make([]*Transaction, len(caches))
	for i := 0; i < len(caches); i++ {
		txs[i] = caches[i].Tx()
	}
	return txs
}

//HashSign hash      ，
func (tx *Transaction) HashSign() []byte {
	copytx := *tx
	copytx.Signature = nil
	data := Encode(&copytx)
	return common.Sha256(data)
}

//Tx
func (tx *Transaction) Tx() *Transaction {
	return tx
}

//GetTxGroup
func (tx *Transaction) GetTxGroup() (*Transactions, error) {
	if tx.GroupCount < 0 || tx.GroupCount == 1 || tx.GroupCount > 20 {
		return nil, ErrTxGroupCount
	}
	if tx.GroupCount > 0 {
		var txs Transactions
		err := Decode(tx.Header, &txs)
		if err != nil {
			return nil, err
		}
		return &txs, nil
	}
	if tx.Next != nil || tx.Header != nil {
		return nil, ErrNomalTx
	}
	return nil, nil
}

//Size
func (tx *Transaction) Size() int {
	return Size(tx)
}

//Sign
func (tx *Transaction) Sign(ty int32, priv crypto.PrivKey) {
	tx.Signature = nil
	data := Encode(tx)
	pub := priv.PubKey()
	sign := priv.Sign(data)
	tx.Signature = &Signature{
		Ty:        ty,
		Pubkey:    pub.Bytes(),
		Signature: sign.Bytes(),
	}
}

//CheckSign tx
func (tx *Transaction) CheckSign() bool {
	return tx.checkSign()
}

//txgroup
func (tx *Transaction) checkSign() bool {
	copytx := *tx
	copytx.Signature = nil
	data := Encode(&copytx)
	if tx.GetSignature() == nil {
		return false
	}
	return CheckSign(data, string(tx.Execer), tx.GetSignature())
}

//Check
func (tx *Transaction) Check(cfg *DplatformOSConfig, height, minfee, maxFee int64) error {
	group, err := tx.GetTxGroup()
	if err != nil {
		return err
	}
	if group == nil {
		return tx.check(cfg, height, minfee, maxFee)
	}
	return group.Check(cfg, height, minfee, maxFee)
}

func (tx *Transaction) check(cfg *DplatformOSConfig, height, minfee, maxFee int64) error {
	if minfee == 0 {
		return nil
	}
	//
	realFee, err := tx.GetRealFee(minfee)
	if err != nil {
		return err
	}
	//
	if tx.Fee < realFee {
		return ErrTxFeeTooLow
	}
	if tx.Fee > maxFee && maxFee > 0 && cfg.IsFork(height, "ForkBlockCheck") {
		return ErrTxFeeTooHigh
	}
	return nil
}

//SetExpire
func (tx *Transaction) SetExpire(cfg *DplatformOSConfig, expire time.Duration) {
	//Txheight
	if cfg.IsEnable("TxHeight") && int64(expire) > TxHeightFlag {
		tx.Expire = int64(expire)
		return
	}

	if int64(expire) > ExpireBound {
		if expire < time.Second*120 {
			expire = time.Second * 120
		}
		//
		tx.Expire = Now().Unix() + int64(expire/time.Second)
	} else {
		tx.Expire = int64(expire)
	}
}

//GetRealFee
func (tx *Transaction) GetRealFee(minFee int64) (int64, error) {
	txSize := Size(tx)
	//      ，
	if tx.Signature == nil {
		txSize += 300
	}
	if txSize > MaxTxSize {
		return 0, ErrTxMsgSizeTooBig
	}
	//
	realFee := int64(txSize/1000+1) * minFee
	return realFee, nil
}

//SetRealFee
func (tx *Transaction) SetRealFee(minFee int64) error {
	if tx.Fee == 0 {
		fee, err := tx.GetRealFee(minFee)
		if err != nil {
			return err
		}
		tx.Fee = fee
	}
	return nil
}

//ExpireBound
var ExpireBound int64 = 1000000000 //        ，  expireBound  height，  expireBound  blockTime

//IsExpire
func (tx *Transaction) IsExpire(cfg *DplatformOSConfig, height, blocktime int64) bool {
	group, _ := tx.GetTxGroup()
	if group == nil {
		return tx.isExpire(cfg, height, blocktime)
	}
	return group.IsExpire(cfg, height, blocktime)
}

//GetTxFee        ，
func (tx *Transaction) GetTxFee() int64 {
	group, _ := tx.GetTxGroup()
	if group == nil {
		return tx.Fee
	}
	return group.Txs[0].Fee
}

//From   from
func (tx *Transaction) From() string {
	return address.PubKeyToAddr(tx.GetSignature().GetPubkey())
}

//        ，    true，     false
func (tx *Transaction) isExpire(cfg *DplatformOSConfig, height, blocktime int64) bool {
	valid := tx.Expire
	// Expire 0，  false
	if valid == 0 {
		return false
	}
	if valid <= ExpireBound {
		//Expire  1e9， height valid > height      false else true
		return valid <= height
	}
	//EnableTxHeight     ,
	if txHeight := GetTxHeight(cfg, valid, height); txHeight > 0 {
		if txHeight-LowAllowPackHeight <= height && height <= txHeight+HighAllowPackHeight {
			return false
		}
		return true
	}
	// Expire  1e9， blockTime  valid > blocktime  false     else true
	return valid <= blocktime
}

//GetTxHeight
func GetTxHeight(cfg *DplatformOSConfig, valid int64, height int64) int64 {
	if cfg.IsPara() {
		return -1
	}
	if cfg.IsEnableFork(height, "ForkTxHeight", cfg.IsEnable("TxHeight")) && valid > TxHeightFlag {
		return valid - TxHeightFlag
	}
	return -1
}

//JSON Transaction      json
func (tx *Transaction) JSON() string {
	type transaction struct {
		Hash      string     `json:"hash,omitempty"`
		Execer    string     `json:"execer,omitempty"`
		Payload   string     `json:"payload,omitempty"`
		Signature *Signature `json:"signature,omitempty"`
		Fee       int64      `json:"fee,omitempty"`
		Expire    int64      `json:"expire,omitempty"`
		//   ID，    payload      ，
		Nonce int64 `json:"nonce,omitempty"`
		//     ，        ，
		To         string `json:"to,omitempty"`
		GroupCount int32  `json:"groupCount,omitempty"`
		Header     string `json:"header,omitempty"`
		Next       string `json:"next,omitempty"`
	}

	newtx := &transaction{}
	newtx.Hash = hex.EncodeToString(tx.Hash())
	newtx.Execer = string(tx.Execer)
	newtx.Payload = hex.EncodeToString(tx.Payload)
	newtx.Signature = tx.Signature
	newtx.Fee = tx.Fee
	newtx.Expire = tx.Expire
	newtx.Nonce = tx.Nonce
	newtx.To = tx.To
	newtx.GroupCount = tx.GroupCount
	newtx.Header = hex.EncodeToString(tx.Header)
	newtx.Next = hex.EncodeToString(tx.Next)
	data, err := json.MarshalIndent(newtx, "", "\t")
	if err != nil {
		return err.Error()
	}
	return string(data)
}

//Amount   tx payload  amount
func (tx *Transaction) Amount() (int64, error) {
	// TODO                 ，     0, nil
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return 0, nil
	}
	return exec.Amount(tx)
}

//Assets
func (tx *Transaction) Assets() ([]*Asset, error) {
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return nil, nil
	}
	return exec.GetAssets(tx)
}

//GetRealToAddr   tx payload  real to
func (tx *Transaction) GetRealToAddr() string {
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return tx.To
	}
	return exec.GetRealToAddr(tx)
}

//GetViewFromToAddr   tx payload  view from to
func (tx *Transaction) GetViewFromToAddr() (string, string) {
	exec := LoadExecutorType(string(tx.Execer))
	if exec == nil {
		return tx.From(), tx.To
	}
	return exec.GetViewFromToAddr(tx)
}

//ActionName   tx   Actionname
func (tx *Transaction) ActionName() string {
	execName := string(tx.Execer)
	exec := LoadExecutorType(execName)
	if exec == nil {
		//action name         ，
		realname := GetRealExecName(tx.Execer)
		exec = LoadExecutorType(string(realname))
		if exec == nil {
			return "unknown"
		}
	}
	return exec.ActionName(tx)
}

//IsWithdraw      withdraw  ，   from to   swap，
func (tx *Transaction) IsWithdraw() bool {
	if bytes.Equal(tx.GetExecer(), bCoins) || bytes.Equal(tx.GetExecer(), bToken) {
		if tx.ActionName() == withdraw {
			return true
		}
	}
	return false
}

// ParseExpire parse expire to int from during or height
func ParseExpire(expire string) (int64, error) {
	if len(expire) == 0 {
		return 0, ErrInvalidParam
	}
	if expire[0] == 'H' && expire[1] == ':' {
		txHeight, err := strconv.ParseInt(expire[2:], 10, 64)
		if err != nil {
			return 0, err
		}
		if txHeight <= 0 {
			//fmt.Printf("txHeight should be grate to 0")
			return 0, ErrHeightLessZero
		}
		if txHeight+TxHeightFlag < txHeight {
			return 0, ErrHeightOverflow
		}

		return txHeight + TxHeightFlag, nil
	}

	blockHeight, err := strconv.ParseInt(expire, 10, 64)
	if err == nil {
		return blockHeight, nil
	}

	expireTime, err := time.ParseDuration(expire)
	if err == nil {
		return int64(expireTime), nil
	}

	return 0, err
}

//CalcTxShortHash  txhash      ，    5
func CalcTxShortHash(hash []byte) string {
	if len(hash) >= 5 {
		return hex.EncodeToString(hash[0:5])
	}
	return ""
}

//TransactionSort
//    map             ,   title  ，     title   main
//  map  title    ，      map
func TransactionSort(rawtxs []*Transaction) []*Transaction {
	txMap := make(map[string]*Transactions)

	for _, tx := range rawtxs {
		title, isPara := GetParaExecTitleName(string(tx.Execer))
		if !isPara {
			title = MainChainName
		}
		if txMap[title] != nil {
			txMap[title].Txs = append(txMap[title].Txs, tx)
		} else {
			var temptxs Transactions
			temptxs.Txs = append(temptxs.Txs, tx)
			txMap[title] = &temptxs
		}
	}

	//    title  ，       map
	var newMp = make([]string, 0)
	for k := range txMap {
		newMp = append(newMp, k)
	}
	sort.Strings(newMp)

	var txs Transactions
	for _, v := range newMp {
		txs.Txs = append(txs.Txs, txMap[v].GetTxs()...)
	}
	return txs.GetTxs()
}

//Hash    hash   header  ，  tx group    ，
func (tx *Transaction) Hash() []byte {
	copytx := cloneTx(tx)
	copytx.Signature = nil
	copytx.Header = nil
	data := Encode(copytx)
	return common.Sha256(data)
}

//FullHash    fullhash
func (tx *Transaction) FullHash() []byte {
	copytx := tx.Clone()
	data := Encode(copytx)
	return common.Sha256(data)
}
