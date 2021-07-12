// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types    dplatformos     、  、
package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/types/jsonpb"
	"github.com/golang/protobuf/proto"

	//   system crypto
	_ "github.com/D-PlatformOperatingSystem/dpos/system/crypto/init"
)

var tlog = log.New("module", "types")

// Size1Kshiftlen tx    1k
const Size1Kshiftlen uint = 10

// Message   proto.Message
type Message proto.Message

//TxGroup       ，Transactions   Transaction
type TxGroup interface {
	Tx() *Transaction
	GetTxGroup() (*Transactions, error)
	CheckSign() bool
}

//ExecName     name
func (c *DplatformOSConfig) ExecName(name string) string {
	if len(name) > 1 && name[0] == '#' {
		return name[1:]
	}
	if IsParaExecName(name) {
		return name
	}
	if c.IsPara() {
		return c.GetTitle() + name
	}
	return name
}

//IsAllowExecName    allow   ->   GetRealExecName
//name     3    100
func IsAllowExecName(name []byte, execer []byte) bool {
	// name
	if len(name) > address.MaxExecNameLength || len(execer) > address.MaxExecNameLength {
		return false
	}
	if len(name) < 3 || len(execer) < 3 {
		return false
	}
	// name      "-"
	if bytes.Contains(name, slash) || bytes.Contains(name, sharp) {
		return false
	}
	if !bytes.Equal(name, execer) && !bytes.Equal(name, GetRealExecName(execer)) {
		return false
	}
	if bytes.HasPrefix(name, UserKey) {
		return true
	}
	for i := range AllowUserExec {
		if bytes.Equal(AllowUserExec[i], name) {
			return true
		}
	}
	return false
}

var bytesExec = []byte("exec-")
var commonPrefix = []byte("mavl-")

//GetExecKey       key
func GetExecKey(key []byte) (string, bool) {
	n := 0
	start := 0
	end := 0
	for i := len(commonPrefix); i < len(key); i++ {
		if key[i] == '-' {
			n = n + 1
			if n == 2 {
				start = i + 1
			}
			if n == 3 {
				end = i
				break
			}
		}
	}
	if start > 0 && end > 0 {
		if bytes.Equal(key[start:end+1], bytesExec) {
			//find addr
			start = end + 1
			for k := end; k < len(key); k++ {
				if key[k] == ':' { //end+1
					end = k
					return string(key[start:end]), true
				}
			}
		}
	}
	return "", false
}

//FindExecer
func FindExecer(key []byte) (execer []byte, err error) {
	if !bytes.HasPrefix(key, commonPrefix) {
		return nil, ErrMavlKeyNotStartWithMavl
	}
	for i := len(commonPrefix); i < len(key); i++ {
		if key[i] == '-' {
			return key[len(commonPrefix):i], nil
		}
	}
	return nil, ErrNoExecerInMavlKey
}

//GetParaExec
func (c *DplatformOSConfig) GetParaExec(execer []byte) []byte {
	//
	if !c.IsPara() {
		return execer
	}
	//
	if !strings.HasPrefix(string(execer), c.GetTitle()) {
		return execer
	}
	return execer[len(c.GetTitle()):]
}

//GetParaExecName
func GetParaExecName(execer []byte) []byte {
	if !bytes.HasPrefix(execer, ParaKey) {
		return execer
	}
	count := 0
	for i := 0; i < len(execer); i++ {
		if execer[i] == '.' {
			count++
		}
		if count == 3 && i < (len(execer)-1) {
			newexec := execer[i+1:]
			return newexec
		}
	}
	return execer
}

//GetRealExecName          name
func GetRealExecName(execer []byte) []byte {
	//      ，
	execer = GetParaExecName(execer)
	//
	if bytes.HasPrefix(execer, ParaKey) {
		return execer
	}
	if bytes.HasPrefix(execer, UserKey) {
		//  user.p.    ,   user.
		count := 0
		index := 0
		for i := 0; i < len(execer); i++ {
			if execer[i] == '.' {
				count++
			}
			index = i
			if count == 2 {
				index--
				break
			}
		}
		e := execer[len(UserKey) : index+1]
		if len(e) > 0 {
			return e
		}
	}
	return execer
}

//Encode
func Encode(data proto.Message) []byte {
	b, err := proto.Marshal(data)
	if err != nil {
		panic(err)
	}
	return b
}

//Size
func Size(data proto.Message) int {
	return proto.Size(data)
}

//Decode
func Decode(data []byte, msg proto.Message) error {
	return proto.Unmarshal(data, msg)
}

//JSONToPB  JSON     protobuffer
func JSONToPB(data []byte, msg proto.Message) error {
	return jsonpb.Unmarshal(bytes.NewReader(data), msg)
}

//JSONToPBUTF8     utf8     bytes
func JSONToPBUTF8(data []byte, msg proto.Message) error {
	decode := &jsonpb.Unmarshaler{EnableUTF8BytesToString: true}
	return decode.Unmarshal(bytes.NewReader(data), msg)
}

//Hash         hash
func (leafnode *LeafNode) Hash() []byte {
	data, err := proto.Marshal(leafnode)
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}

var sha256Len = 32

//Hash         hash
func (innernode *InnerNode) Hash() []byte {
	rightHash := innernode.RightHash
	leftHash := innernode.LeftHash
	hashLen := sha256Len
	if len(innernode.RightHash) > hashLen {
		innernode.RightHash = innernode.RightHash[len(innernode.RightHash)-hashLen:]
	}
	if len(innernode.LeftHash) > hashLen {
		innernode.LeftHash = innernode.LeftHash[len(innernode.LeftHash)-hashLen:]
	}
	data, err := proto.Marshal(innernode)
	if err != nil {
		panic(err)
	}
	innernode.RightHash = rightHash
	innernode.LeftHash = leftHash
	return common.Sha256(data)
}

//NewErrReceipt  new    Receipt
func NewErrReceipt(err error) *Receipt {
	berr := err.Error()
	errlog := &ReceiptLog{Ty: TyLogErr, Log: []byte(berr)}
	return &Receipt{Ty: ExecErr, KV: nil, Logs: []*ReceiptLog{errlog}}
}

//CheckAmount
func CheckAmount(amount int64) bool {
	if amount <= 0 || amount >= MaxCoin {
		return false
	}
	return true
}

//GetEventName      name    id
func GetEventName(event int) string {
	name, ok := eventName[event]
	if ok {
		return name
	}
	return "unknow-event"
}

//GetSignName
func GetSignName(execer string, signType int) string {
	//
	if execer != "" {
		exec := LoadExecutorType(execer)
		if exec != nil {
			name, err := exec.GetCryptoDriver(signType)
			if err == nil {
				return name
			}
		}
	}
	//
	return crypto.GetName(signType)
}

//GetSignType
func GetSignType(execer string, name string) int {
	//
	if execer != "" {
		exec := LoadExecutorType(execer)
		if exec != nil {
			ty, err := exec.GetCryptoType(name)
			if err == nil {
				return ty
			}
		}
	}
	//
	return crypto.GetType(name)
}

// ConfigPrefix     key
var ConfigPrefix = "mavl-config-"

// ConfigKey      bug，     key     ，
// mavl-config–{key}  key     -
func ConfigKey(key string) string {
	return fmt.Sprintf("%s-%s", ConfigPrefix, key)
}

// ManagePrefix            key
var ManagePrefix = "mavl-"

//ManageKey        key
func ManageKey(key string) string {
	return fmt.Sprintf("%s-%s", ManagePrefix+"manage", key)
}

//ManaeKeyWithHeigh        key
func (c *DplatformOSConfig) ManaeKeyWithHeigh(key string, height int64) string {
	if c.IsFork(height, "ForkExecKey") {
		return ManageKey(key)
	}
	return ConfigKey(key)
}

//ReceiptDataResult
type ReceiptDataResult struct {
	Ty     int32               `json:"ty"`
	TyName string              `json:"tyname"`
	Logs   []*ReceiptLogResult `json:"logs"`
}

//ReceiptLogResult   log
type ReceiptLogResult struct {
	Ty     int32       `json:"ty"`
	TyName string      `json:"tyname"`
	Log    interface{} `json:"log"`
	RawLog string      `json:"rawlog"`
}

//DecodeReceiptLog
func (r *ReceiptData) DecodeReceiptLog(execer []byte) (*ReceiptDataResult, error) {
	result := &ReceiptDataResult{Ty: r.GetTy()}
	switch r.Ty {
	case 0:
		result.TyName = "ExecErr"
	case 1:
		result.TyName = "ExecPack"
	case 2:
		result.TyName = "ExecOk"
	default:
		return nil, ErrLogType
	}

	logs := r.GetLogs()
	for _, l := range logs {
		var lTy string
		var logIns interface{}
		lLog, err := hex.DecodeString(common.ToHex(l.GetLog())[2:])
		if err != nil {
			return nil, err
		}

		logType := LoadLog(execer, int64(l.Ty))
		if logType == nil {
			//tlog.Error("DecodeReceiptLog:", "Faile to decodeLog with type value logtype", l.Ty)
			return nil, ErrLogType
		}

		logIns, _ = logType.Decode(lLog)
		lTy = logType.Name()

		result.Logs = append(result.Logs, &ReceiptLogResult{Ty: l.Ty, TyName: lTy, Log: logIns, RawLog: common.ToHex(l.GetLog())})
	}
	return result, nil
}

//OutputReceiptDetails
func (r *ReceiptData) OutputReceiptDetails(execer []byte, logger log.Logger) {
	rds, err := r.DecodeReceiptLog(execer)
	if err == nil {
		logger.Debug("receipt decode", "receipt data", rds)
		for _, rdl := range rds.Logs {
			logger.Debug("receipt log", "log", rdl)
		}
	} else {
		logger.Error("decodelogerr", "err", err)
	}
}

//IterateRangeByStateHash
func (t *ReplyGetTotalCoins) IterateRangeByStateHash(key, value []byte) bool {
	tlog.Debug("ReplyGetTotalCoins.IterateRangeByStateHash", "key", string(key))
	var acc Account
	err := Decode(value, &acc)
	if err != nil {
		tlog.Error("ReplyGetTotalCoins.IterateRangeByStateHash", "err", err)
		return true
	}
	//tlog.Info("acc:", "value", acc)
	if t.Num >= t.Count {
		t.NextKey = key
		return true
	}
	t.Num++
	t.Amount += acc.Balance
	return false
}

// GetTxTimeInterval
func GetTxTimeInterval() time.Duration {
	return time.Second * 120
}

// ParaCrossTx
type ParaCrossTx interface {
	IsParaCrossTx() bool
}

// PBToJSON
func PBToJSON(r Message) ([]byte, error) {
	encode := &jsonpb.Marshaler{EmitDefaults: true}
	var buf bytes.Buffer
	if err := encode.Marshal(&buf, r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PBToJSONUTF8
func PBToJSONUTF8(r Message) ([]byte, error) {
	encode := &jsonpb.Marshaler{EmitDefaults: true, EnableUTF8BytesToString: true}
	var buf bytes.Buffer
	if err := encode.Marshal(&buf, r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

//MustPBToJSON panic when error
func MustPBToJSON(req Message) []byte {
	data, err := PBToJSON(req)
	if err != nil {
		panic(err)
	}
	return data
}

// MustDecode
func MustDecode(data []byte, v interface{}) {
	if data == nil {
		return
	}
	err := json.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}
}

// AddItem   item
func (t *ReplyGetExecBalance) AddItem(execAddr, value []byte) {
	var acc Account
	err := Decode(value, &acc)
	if err != nil {
		tlog.Error("ReplyGetExecBalance.AddItem", "err", err)
		return
	}
	tlog.Info("acc:", "value", acc)
	t.Amount += acc.Balance
	t.Amount += acc.Frozen

	t.AmountActive += acc.Balance
	t.AmountFrozen += acc.Frozen

	item := &ExecBalanceItem{ExecAddr: execAddr, Frozen: acc.Frozen, Active: acc.Balance}
	t.Items = append(t.Items, item)
}

//Clone
func Clone(data proto.Message) proto.Message {
	return proto.Clone(data)
}

//Clone
func (sig *Signature) Clone() *Signature {
	if sig == nil {
		return nil
	}
	return &Signature{
		Ty:        sig.Ty,
		Pubkey:    sig.Pubkey,
		Signature: sig.Signature,
	}
}

//       tmp := *tx           proto         size
//proto buffer         ，       ，         bug
func cloneTx(tx *Transaction) *Transaction {
	copytx := &Transaction{}
	copytx.Execer = tx.Execer
	copytx.Payload = tx.Payload
	copytx.Signature = tx.Signature
	copytx.Fee = tx.Fee
	copytx.Expire = tx.Expire
	copytx.Nonce = tx.Nonce
	copytx.To = tx.To
	copytx.GroupCount = tx.GroupCount
	copytx.Header = tx.Header
	copytx.Next = tx.Next
	return copytx
}

//Clone copytx := proto.Clone(tx).(*Transaction) too slow
func (tx *Transaction) Clone() *Transaction {
	if tx == nil {
		return nil
	}
	tmp := cloneTx(tx)
	tmp.Signature = tx.Signature.Clone()
	return tmp
}

//Clone    ： BlockDetail
func (b *BlockDetail) Clone() *BlockDetail {
	if b == nil {
		return nil
	}
	return &BlockDetail{
		Block:          b.Block.Clone(),
		Receipts:       cloneReceipts(b.Receipts),
		KV:             cloneKVList(b.KV),
		PrevStatusHash: b.PrevStatusHash,
	}
}

//Clone    ReceiptData
func (r *ReceiptData) Clone() *ReceiptData {
	if r == nil {
		return nil
	}
	return &ReceiptData{
		Ty:   r.Ty,
		Logs: cloneReceiptLogs(r.Logs),
	}
}

//Clone     receiptLog
func (r *ReceiptLog) Clone() *ReceiptLog {
	if r == nil {
		return nil
	}
	return &ReceiptLog{
		Ty:  r.Ty,
		Log: r.Log,
	}
}

//Clone KeyValue
func (kv *KeyValue) Clone() *KeyValue {
	if kv == nil {
		return nil
	}
	return &KeyValue{
		Key:   kv.Key,
		Value: kv.Value,
	}
}

//Clone Block    (   types.Message      )
func (b *Block) Clone() *Block {
	if b == nil {
		return nil
	}
	return &Block{
		Version:    b.Version,
		ParentHash: b.ParentHash,
		TxHash:     b.TxHash,
		StateHash:  b.StateHash,
		Height:     b.Height,
		BlockTime:  b.BlockTime,
		Difficulty: b.Difficulty,
		MainHash:   b.MainHash,
		MainHeight: b.MainHeight,
		Signature:  b.Signature.Clone(),
		Txs:        cloneTxs(b.Txs),
	}
}

//Clone BlockBody    (   types.Message      )
func (b *BlockBody) Clone() *BlockBody {
	if b == nil {
		return nil
	}
	return &BlockBody{
		Txs:        cloneTxs(b.Txs),
		Receipts:   cloneReceipts(b.Receipts),
		MainHash:   b.MainHash,
		MainHeight: b.MainHeight,
		Hash:       b.Hash,
		Height:     b.Height,
	}
}

//cloneReceipts
func cloneReceipts(b []*ReceiptData) []*ReceiptData {
	if b == nil {
		return nil
	}
	rs := make([]*ReceiptData, len(b))
	for i := 0; i < len(b); i++ {
		rs[i] = b[i].Clone()
	}
	return rs
}

//cloneReceiptLogs     ReceiptLogs
func cloneReceiptLogs(b []*ReceiptLog) []*ReceiptLog {
	if b == nil {
		return nil
	}
	rs := make([]*ReceiptLog, len(b))
	for i := 0; i < len(b); i++ {
		rs[i] = b[i].Clone()
	}
	return rs
}

//cloneTxs     txs
func cloneTxs(b []*Transaction) []*Transaction {
	if b == nil {
		return nil
	}
	txs := make([]*Transaction, len(b))
	for i := 0; i < len(b); i++ {
		txs[i] = b[i].Clone()
	}
	return txs
}

//cloneKVList   kv
func cloneKVList(b []*KeyValue) []*KeyValue {
	if b == nil {
		return nil
	}
	kv := make([]*KeyValue, len(b))
	for i := 0; i < len(b); i++ {
		kv[i] = b[i].Clone()
	}
	return kv
}

// Bytes2Str
//         ，    35 ，        byte      ，
//     b        string   ，          ，
func Bytes2Str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Str2Bytes
//         ，    13 ，         byte      ，
//             byte  ，         string，  panic
func Str2Bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

//Hash    hash
func (hashes *ReplyHashes) Hash() []byte {
	data, err := proto.Marshal(hashes)
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}
