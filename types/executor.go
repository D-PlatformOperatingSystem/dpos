// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"reflect"
	"strings"
	"unicode"

	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/golang/protobuf/proto"
)

func init() {
	rand.Seed(Now().UnixNano())
}

// LogType
type LogType interface {
	Name() string
	Decode([]byte) (interface{}, error)
	JSON([]byte) ([]byte, error)
}

type logInfoType struct {
	ty     int64
	execer []byte
}

func newLogType(execer []byte, ty int64) LogType {
	return &logInfoType{ty: ty, execer: execer}
}

func (l *logInfoType) Name() string {
	return GetLogName(l.execer, l.ty)
}

func (l *logInfoType) Decode(data []byte) (interface{}, error) {
	return DecodeLog(l.execer, l.ty, data)
}

func (l *logInfoType) JSON(data []byte) ([]byte, error) {
	d, err := l.Decode(data)
	if err != nil {
		return nil, err
	}
	if msg, ok := d.(Message); ok {
		return PBToJSON(msg)
	}
	jsdata, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return jsdata, nil
}

var executorMap = map[string]ExecutorType{}

// RegistorExecutor
func RegistorExecutor(exec string, util ExecutorType) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if util.GetChild() == nil {
		panic("exec " + exec + " executorType child is nil")
	}
	if _, exist := executorMap[exec]; exist {
		panic("DupExecutorType")
	} else {
		executorMap[exec] = util
	}
}

// LoadExecutorType
func LoadExecutorType(execstr string) ExecutorType {
	//
	//
	realname := GetRealExecName([]byte(execstr))
	if exec, exist := executorMap[string(realname)]; exist {
		return exec
	}
	return nil
}

// CallExecNewTx
func CallExecNewTx(c *DplatformOSConfig, execName, action string, param interface{}) ([]byte, error) {
	exec := LoadExecutorType(execName)
	if exec == nil {
		tlog.Error("callExecNewTx", "Error", "exec not found")
		return nil, ErrNotSupport
	}
	// param is interface{type, var-nil}, check with nil always fail
	if reflect.ValueOf(param).IsNil() {
		tlog.Error("callExecNewTx", "Error", "param in nil")
		return nil, ErrInvalidParam
	}
	jsonStr, err := json.Marshal(param)
	if err != nil {
		tlog.Error("callExecNewTx", "Error", err)
		return nil, err
	}
	tx, err := exec.CreateTx(action, json.RawMessage(jsonStr))
	if err != nil {
		tlog.Error("callExecNewTx", "Error", err)
		return nil, err
	}
	return FormatTxEncode(c, execName, tx)
}

//CallCreateTransaction
func CallCreateTransaction(execName, action string, param Message) (*Transaction, error) {
	exec := LoadExecutorType(execName)
	if exec == nil {
		tlog.Error("CallCreateTx", "Error", "exec not found")
		return nil, ErrNotSupport
	}
	// param is interface{type, var-nil}, check with nil always fail
	if param == nil {
		tlog.Error("CallCreateTx", "Error", "param in nil")
		return nil, ErrInvalidParam
	}
	return exec.Create(action, param)
}

// CallCreateTx
func CallCreateTx(c *DplatformOSConfig, execName, action string, param Message) ([]byte, error) {
	tx, err := CallCreateTransaction(execName, action, param)
	if err != nil {
		return nil, err
	}
	return FormatTxEncode(c, execName, tx)
}

//CallCreateTxJSON create tx by json
func CallCreateTxJSON(c *DplatformOSConfig, execName, action string, param json.RawMessage) ([]byte, error) {
	exec := LoadExecutorType(execName)
	if exec == nil {
		execer := GetParaExecName([]byte(execName))
		//      ，   user.xxx
		if bytes.HasPrefix(execer, UserKey) {
			tx := &Transaction{Payload: param}
			return FormatTxEncode(c, execName, tx)
		}
		tlog.Error("CallCreateTxJSON", "Error", "exec not found")
		return nil, ErrExecNotFound
	}
	// param is interface{type, var-nil}, check with nil always fail
	if param == nil {
		tlog.Error("CallCreateTxJSON", "Error", "param in nil")
		return nil, ErrInvalidParam
	}
	tx, err := exec.CreateTx(action, param)
	if err != nil {
		tlog.Error("CallCreateTxJSON", "Error", err)
		return nil, err
	}
	return FormatTxEncode(c, execName, tx)
}

// CreateFormatTx
func CreateFormatTx(c *DplatformOSConfig, execName string, payload []byte) (*Transaction, error) {
	//  nonce,execer,to, fee    ,          transaction   ，   execer fee
	tx := &Transaction{Payload: payload}
	return FormatTx(c, execName, tx)
}

// FormatTx    tx
func FormatTx(c *DplatformOSConfig, execName string, tx *Transaction) (*Transaction, error) {
	//  nonce,execer,to, fee    ,          transaction   ，   execer fee
	tx.Nonce = rand.Int63()
	tx.Execer = []byte(execName)
	//   ，   to
	if c.IsPara() || tx.To == "" {
		tx.To = address.ExecAddress(string(tx.Execer))
	}
	var err error
	if tx.Fee == 0 {
		tx.Fee, err = tx.GetRealFee(c.GetMinTxFeeRate())
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

// FormatTxEncode         byte
func FormatTxEncode(c *DplatformOSConfig, execName string, tx *Transaction) ([]byte, error) {
	tx, err := FormatTx(c, execName, tx)
	if err != nil {
		return nil, err
	}
	txbyte := Encode(tx)
	return txbyte, nil
}

// LoadLog   log
func LoadLog(execer []byte, ty int64) LogType {
	loginfo := getLogType(execer, ty)
	if loginfo.Name == "LogReserved" {
		return nil
	}
	return newLogType(execer, ty)
}

// GetLogName     ,
func GetLogName(execer []byte, ty int64) string {
	t := getLogType(execer, ty)
	return t.Name
}

func getLogType(execer []byte, ty int64) *LogInfo {
	//          log
	if execer != nil {
		ety := LoadExecutorType(string(execer))
		if ety != nil {
			logmap := ety.GetLogMap()
			if logty, ok := logmap[ty]; ok {
				return logty
			}
		}
	}
	//    ，
	if logty, ok := SystemLog[ty]; ok {
		return logty
	}
	//
	return SystemLog[0]
}

// DecodeLog   log
func DecodeLog(execer []byte, ty int64, data []byte) (interface{}, error) {
	t := getLogType(execer, ty)
	if t.Name == "LogErr" || t.Name == "LogReserved" {
		msg := string(data)
		return msg, nil
	}
	pdata := reflect.New(t.Ty)
	if !pdata.CanInterface() {
		return nil, ErrDecode
	}
	msg, ok := pdata.Interface().(Message)
	if !ok {
		return nil, ErrDecode
	}
	err := Decode(data, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// ExecutorType
type ExecutorType interface {
	//       to addr
	GetRealToAddr(tx *Transaction) string
	GetCryptoDriver(ty int) (string, error)
	GetCryptoType(name string) (int, error)
	//      from   to
	GetViewFromToAddr(tx *Transaction) (string, string)
	ActionName(tx *Transaction) string
	//     create  ，createTx
	GetAction(action string) (Message, error)
	InitFuncList(list map[string]reflect.Method)
	Create(action string, message Message) (*Transaction, error)
	CreateTx(action string, message json.RawMessage) (*Transaction, error)
	CreateQuery(funcname string, message json.RawMessage) (Message, error)
	AssertCreate(createTx *CreateTx) (*Transaction, error)
	QueryToJSON(funcname string, message Message) ([]byte, error)
	Amount(tx *Transaction) (int64, error)
	DecodePayload(tx *Transaction) (Message, error)
	DecodePayloadValue(tx *Transaction) (string, reflect.Value, error)
	//write for executor
	GetPayload() Message
	GetChild() ExecutorType
	GetName() string
	//exec result of receipt log
	GetLogMap() map[int64]*LogInfo
	GetForks() *Forks
	IsFork(height int64, key string) bool
	//actionType -> name map
	GetTypeMap() map[string]int32
	GetValueTypeMap() map[string]reflect.Type
	//action function list map
	GetFuncMap() map[string]reflect.Method
	GetRPCFuncMap() map[string]reflect.Method
	GetExecFuncMap() map[string]reflect.Method
	CreateTransaction(action string, data Message) (*Transaction, error)
	// collect assets the tx deal with
	GetAssets(tx *Transaction) ([]*Asset, error)

	// about dplatformosConfig
	GetConfig() *DplatformOSConfig
	SetConfig(cfg *DplatformOSConfig)
}

// ExecTypeGet
type execTypeGet interface {
	GetTy() int32
}

// ExecTypeBase
type ExecTypeBase struct {
	child               ExecutorType
	childValue          reflect.Value
	actionFunList       map[string]reflect.Method
	execFuncList        map[string]reflect.Method
	actionListValueType map[string]reflect.Type
	rpclist             map[string]reflect.Method
	queryMap            map[string]reflect.Type
	forks               *Forks
	cfg                 *DplatformOSConfig
}

// GetChild
func (base *ExecTypeBase) GetChild() ExecutorType {
	return base.child
}

// SetChild
func (base *ExecTypeBase) SetChild(child ExecutorType) {
	base.child = child
	base.childValue = reflect.ValueOf(child)
	base.rpclist = ListMethod(child)
	base.actionListValueType = make(map[string]reflect.Type)
	base.actionFunList = make(map[string]reflect.Method)
	base.forks = child.GetForks()

	action := child.GetPayload()
	if action == nil {
		return
	}
	base.actionFunList = ListMethod(action)
	if _, ok := base.actionFunList["XXX_OneofWrappers"]; !ok {
		return
	}
	retval := base.actionFunList["XXX_OneofWrappers"].Func.Call([]reflect.Value{reflect.ValueOf(action)})
	if len(retval) != 1 {
		panic("err XXX_OneofWrappers")
	}
	list := ListType(retval[0].Interface().([]interface{}))

	for k, v := range list {
		data := strings.Split(k, "_")
		if len(data) != 2 {
			panic("name create " + k)
		}
		base.actionListValueType["Value_"+data[1]] = v
		field := v.Field(0)
		base.actionListValueType[field.Name] = field.Type.Elem()
		_, ok := v.FieldByName(data[1])
		if !ok {
			panic("no filed " + k)
		}
	}
	//check type map is all in value type list
	typelist := base.child.GetTypeMap()
	for k := range typelist {
		if _, ok := base.actionListValueType[k]; !ok {
			panic("value type not found " + k)
		}
		if _, ok := base.actionListValueType["Value_"+k]; !ok {
			panic("value type not found " + k)
		}
	}
}

// GetForks    fork
func (base *ExecTypeBase) GetForks() *Forks {
	return &Forks{}
}

// GetCryptoDriver    Crypto
func (base *ExecTypeBase) GetCryptoDriver(ty int) (string, error) {
	return "", ErrNotSupport
}

// GetCryptoType    Crypto
func (base *ExecTypeBase) GetCryptoType(name string) (int, error) {
	return 0, ErrNotSupport
}

// InitFuncList
func (base *ExecTypeBase) InitFuncList(list map[string]reflect.Method) {
	base.execFuncList = list
	actionList := base.GetFuncMap()
	for k, v := range actionList {
		base.execFuncList[k] = v
	}
	//     Query_   ,
	_, base.queryMap = BuildQueryType("Query_", base.execFuncList)
}

// GetRPCFuncMap    rpc
func (base *ExecTypeBase) GetRPCFuncMap() map[string]reflect.Method {
	return base.rpclist
}

// GetExecFuncMap
func (base *ExecTypeBase) GetExecFuncMap() map[string]reflect.Method {
	return base.execFuncList
}

// GetName    name
func (base *ExecTypeBase) GetName() string {
	return "typedriverbase"
}

// IsFork    fork
func (base *ExecTypeBase) IsFork(height int64, key string) bool {
	if base.GetForks() == nil {
		return false
	}
	return base.forks.IsFork(height, key)
}

// GetValueTypeMap
func (base *ExecTypeBase) GetValueTypeMap() map[string]reflect.Type {
	return base.actionListValueType
}

//GetRealToAddr      ToAddr
func (base *ExecTypeBase) GetRealToAddr(tx *Transaction) string {
	if !base.cfg.IsPara() {
		return tx.To
	}
	//
	_, v, err := base.child.DecodePayloadValue(tx)
	if err != nil {
		return tx.To
	}
	payload := v.Interface()
	if to, ok := getTo(payload); ok {
		return to
	}
	return tx.To
}

//  assert    ,genesis
func getTo(payload interface{}) (string, bool) {
	if ato, ok := payload.(*AssetsTransfer); ok {
		return ato.GetTo(), true
	}
	if ato, ok := payload.(*AssetsWithdraw); ok {
		return ato.GetTo(), true
	}
	if ato, ok := payload.(*AssetsTransferToExec); ok {
		return ato.GetTo(), true
	}
	return "", false
}

//IsAssetsTransfer
func IsAssetsTransfer(payload interface{}) bool {
	if _, ok := payload.(*AssetsTransfer); ok {
		return true
	}
	if _, ok := payload.(*AssetsWithdraw); ok {
		return true
	}
	if _, ok := payload.(*AssetsTransferToExec); ok {
		return true
	}
	return false
}

//Amounter
type Amounter interface {
	GetAmount() int64
}

//Amount   tx
func (base *ExecTypeBase) Amount(tx *Transaction) (int64, error) {
	_, v, err := base.child.DecodePayloadValue(tx)
	if err != nil {
		return 0, err
	}
	payload := v.Interface()
	//  assert
	if ato, ok := payload.(Amounter); ok {
		return ato.GetAmount(), nil
	}
	return 0, nil
}

//GetViewFromToAddr      FromAddr
func (base *ExecTypeBase) GetViewFromToAddr(tx *Transaction) (string, string) {
	return tx.From(), tx.To
}

//GetFuncMap
func (base *ExecTypeBase) GetFuncMap() map[string]reflect.Method {
	return base.actionFunList
}

//DecodePayload   tx    payload
func (base *ExecTypeBase) DecodePayload(tx *Transaction) (Message, error) {
	if base.child == nil {
		return nil, ErrActionNotSupport
	}
	payload := base.child.GetPayload()
	if payload == nil {
		return nil, ErrActionNotSupport
	}
	err := Decode(tx.GetPayload(), payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

//DecodePayloadValue   tx    payload  Value
func (base *ExecTypeBase) DecodePayloadValue(tx *Transaction) (string, reflect.Value, error) {
	name, value, err := base.decodePayloadValue(tx)
	return name, value, err
}

func (base *ExecTypeBase) decodePayloadValue(tx *Transaction) (string, reflect.Value, error) {
	if base.child == nil {
		return "", nilValue, ErrActionNotSupport
	}
	action, err := base.child.DecodePayload(tx)
	if err != nil || action == nil {
		tlog.Error("DecodePayload", "err", err, "exec", string(tx.Execer))
		return "", nilValue, err
	}
	name, ty, val, err := GetActionValue(action, base.child.GetFuncMap())
	if err != nil {
		return "", nilValue, err
	}
	typemap := base.child.GetTypeMap()
	//check types is ok
	if v, ok := typemap[name]; !ok || v != ty {
		tlog.Error("GetTypeMap is not ok")
		return "", nilValue, ErrActionNotSupport
	}
	return name, val, nil
}

//ActionName      payload action name
func (base *ExecTypeBase) ActionName(tx *Transaction) string {
	payload, err := base.child.DecodePayload(tx)
	if err != nil {
		return "unknown-err"
	}
	tm := base.child.GetTypeMap()
	if get, ok := payload.(execTypeGet); ok {
		ty := get.GetTy()
		for k, v := range tm {
			if v == ty {
				return lowcaseFirst(k)
			}
		}
	}
	return "unknown"
}

func lowcaseFirst(v string) string {
	if len(v) == 0 {
		return ""
	}
	change := []rune(v)
	if unicode.IsUpper(change[0]) {
		change[0] = unicode.ToLower(change[0])
		return string(change)
	}
	return v
}

//CreateQuery Query
func (base *ExecTypeBase) CreateQuery(funcname string, message json.RawMessage) (Message, error) {
	if _, ok := base.queryMap[funcname]; !ok {
		return nil, ErrActionNotSupport
	}
	ty := base.queryMap[funcname]
	p := reflect.New(ty.In(1).Elem())
	queryin := p.Interface()
	if in, ok := queryin.(proto.Message); ok {
		data, err := message.MarshalJSON()
		if err != nil {
			return nil, err
		}
		err = JSONToPB(data, in)
		if err != nil {
			return nil, err
		}
		return in, nil
	}
	return nil, ErrActionNotSupport
}

//QueryToJSON    json
func (base *ExecTypeBase) QueryToJSON(funcname string, message Message) ([]byte, error) {
	if _, ok := base.queryMap[funcname]; !ok {
		return nil, ErrActionNotSupport
	}
	return PBToJSON(message)
}

func (base *ExecTypeBase) callRPC(method reflect.Method, action string, msg interface{}) (tx *Transaction, err error) {
	valueret := method.Func.Call([]reflect.Value{base.childValue, reflect.ValueOf(action), reflect.ValueOf(msg)})
	if len(valueret) != 2 {
		return nil, ErrMethodNotFound
	}
	if !valueret[0].CanInterface() {
		return nil, ErrMethodNotFound
	}
	if !valueret[1].CanInterface() {
		return nil, ErrMethodNotFound
	}
	r1 := valueret[0].Interface()
	if r1 != nil {
		if r, ok := r1.(*Transaction); ok {
			tx = r
		} else {
			return nil, ErrMethodReturnType
		}
	}
	//  2
	r2 := valueret[1].Interface()
	err = nil
	if r2 != nil {
		if r, ok := r2.(error); ok {
			err = r
		} else {
			return nil, ErrMethodReturnType
		}
	}
	if tx == nil && err == nil {
		return nil, ErrActionNotSupport
	}
	return tx, err
}

//AssertCreate   assets
func (base *ExecTypeBase) AssertCreate(c *CreateTx) (*Transaction, error) {
	if c.ExecName != "" && !IsAllowExecName([]byte(c.ExecName), []byte(c.ExecName)) {
		tlog.Error("CreateTx", "Error", ErrExecNameNotMatch)
		return nil, ErrExecNameNotMatch
	}
	if c.Amount < 0 {
		return nil, ErrAmount
	}
	if c.IsWithdraw {
		p := &AssetsWithdraw{Cointoken: c.GetTokenSymbol(), Amount: c.Amount,
			Note: c.Note, ExecName: c.ExecName, To: c.To}
		return base.child.CreateTransaction("Withdraw", p)
	}
	if c.ExecName != "" {
		v := &AssetsTransferToExec{Cointoken: c.GetTokenSymbol(), Amount: c.Amount,
			Note: c.Note, ExecName: c.ExecName, To: c.To}
		return base.child.CreateTransaction("TransferToExec", v)
	}
	v := &AssetsTransfer{Cointoken: c.GetTokenSymbol(), Amount: c.Amount, Note: c.GetNote(), To: c.To}
	return base.child.CreateTransaction("Transfer", v)
}

//Create   tx
func (base *ExecTypeBase) Create(action string, msg Message) (*Transaction, error) {
	//    FuncList             RPC_{action}
	if msg == nil {
		return nil, ErrInvalidParam
	}
	if action == "" {
		action = "Default_Process"
	}
	funclist := base.GetRPCFuncMap()
	if method, ok := funclist["RPC_"+action]; ok {
		return base.callRPC(method, action, msg)
	}
	if _, ok := msg.(Message); !ok {
		return nil, ErrInvalidParam
	}
	typemap := base.child.GetTypeMap()
	if _, ok := typemap[action]; ok {
		ty1 := base.actionListValueType[action]
		ty2 := reflect.TypeOf(msg).Elem()
		if ty1 != ty2 {
			return nil, ErrInvalidParam
		}
		return base.CreateTransaction(action, msg.(Message))
	}
	tlog.Error(action + " ErrActionNotSupport")
	return nil, ErrActionNotSupport
}

//GetAction   action
func (base *ExecTypeBase) GetAction(action string) (Message, error) {
	typemap := base.child.GetTypeMap()
	if _, ok := typemap[action]; ok {
		tyvalue := reflect.New(base.actionListValueType[action])
		if !tyvalue.CanInterface() {
			tlog.Error(action + " tyvalue.CanInterface error")
			return nil, ErrActionNotSupport
		}
		data, ok := tyvalue.Interface().(Message)
		if !ok {
			tlog.Error(action + " tyvalue is not Message")
			return nil, ErrActionNotSupport
		}
		return data, nil
	}
	tlog.Error(action + " ErrActionNotSupport")
	return nil, ErrActionNotSupport
}

//CreateTx   json rpc
func (base *ExecTypeBase) CreateTx(action string, msg json.RawMessage) (*Transaction, error) {
	data, err := base.GetAction(action)
	if err != nil {
		return nil, err
	}
	b, err := msg.MarshalJSON()
	if err != nil {
		tlog.Error(action + " MarshalJSON  error")
		return nil, err
	}
	err = JSONToPB(b, data)
	if err != nil {
		tlog.Error(action + " jsontopb  error")
		return nil, err
	}
	return base.CreateTransaction(action, data)
}

//CreateTransaction   Transaction
func (base *ExecTypeBase) CreateTransaction(action string, data Message) (tx *Transaction, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrActionNotSupport
		}
	}()
	value := base.child.GetPayload()
	v := reflect.New(base.actionListValueType["Value_"+action])
	vn := reflect.Indirect(v)
	if vn.Kind() != reflect.Struct {
		tlog.Error("CreateTransaction vn not struct kind", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	field := vn.FieldByName(action)
	if !field.IsValid() || !field.CanSet() {
		tlog.Error("CreateTransaction vn filed can't set", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	field.Set(reflect.ValueOf(data))
	value2 := reflect.Indirect(reflect.ValueOf(value))
	if value2.Kind() != reflect.Struct {
		tlog.Error("CreateTransaction value2 not struct kind", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	field = value2.FieldByName("Value")
	if !field.IsValid() || !field.CanSet() {
		tlog.Error("CreateTransaction value filed can't set", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	field.Set(v)

	field = value2.FieldByName("Ty")
	if !field.IsValid() || !field.CanSet() {
		tlog.Error("CreateTransaction ty filed can't set", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	tymap := base.child.GetTypeMap()
	if tyid, ok := tymap[action]; ok {
		field.Set(reflect.ValueOf(tyid))
		tx := &Transaction{
			Payload: Encode(value),
		}
		return tx, nil
	}
	return nil, ErrActionNotSupport
}

// GetAssets
func (base *ExecTypeBase) GetAssets(tx *Transaction) ([]*Asset, error) {
	_, v, err := base.child.DecodePayloadValue(tx)
	if err != nil {
		return nil, err
	}
	payload := v.Interface()
	asset := &Asset{Exec: string(tx.Execer)}
	if a, ok := payload.(*AssetsTransfer); ok {
		asset.Symbol = a.Cointoken
	} else if a, ok := payload.(*AssetsWithdraw); ok {
		asset.Symbol = a.Cointoken
	} else if a, ok := payload.(*AssetsTransferToExec); ok {
		asset.Symbol = a.Cointoken
	} else {
		return nil, nil
	}
	amount, err := tx.Amount()
	if err != nil {
		return nil, nil
	}
	asset.Amount = amount
	return []*Asset{asset}, nil
}

//GetConfig ...
func (base *ExecTypeBase) GetConfig() *DplatformOSConfig {
	return base.cfg
}

//SetConfig ...
func (base *ExecTypeBase) SetConfig(cfg *DplatformOSConfig) {
	base.cfg = cfg
}
