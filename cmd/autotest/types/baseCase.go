// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"errors"

	"github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	dplatformosType "github.com/D-PlatformOperatingSystem/dpos/system/dapp/commands/types"
)

//CaseFunc interface for testCase
type CaseFunc interface {
	//    id
	GetID() string
	//
	GetCmd() string
	//
	GetDep() []string
	//
	GetRepeat() int
	//          ，
	GetBaseCase() *BaseCase
	//                 ，
	SetDependData(interface{})
	//      ，
	SendCommand(packID string) (PackFunc, error)
}

//PackFunc interface for check testCase result
type PackFunc interface {
	//  id
	GetPackID() string
	//  id
	SetPackID(id string)
	//
	GetBaseCase() *BaseCase
	//
	GetTxHash() string
	//      ，json
	GetTxReceipt() string
	//          ，
	GetBasePack() *BaseCasePack
	//  log15
	SetLogger(fLog log15.Logger, tLog log15.Logger)
	//  check
	GetCheckHandlerMap() interface{}
	//
	GetDependData() interface{}
	//    check
	CheckResult(interface{}) (bool, bool)
}

//BaseCase base test case
type BaseCase struct {
	ID        string   `toml:"id"`
	Command   string   `toml:"command"`
	Dep       []string `toml:"dep,omitempty"`
	CheckItem []string `toml:"checkItem,omitempty"`
	Repeat    int      `toml:"repeat,omitempty"`
	Fail      bool     `toml:"fail,omitempty"`
}

//check item handler
//  autotest    ，handlerfunc    json map  ，      dplatformos TxDetailResult

//CheckHandlerFuncDiscard   func
type CheckHandlerFuncDiscard func(map[string]interface{}) bool

//CheckHandlerMapDiscard   map
type CheckHandlerMapDiscard map[string]CheckHandlerFuncDiscard

//

//CheckHandlerParamType
type CheckHandlerParamType *dplatformosType.TxDetailResult

//CheckHandlerFunc   func
type CheckHandlerFunc func(CheckHandlerParamType) bool

//CheckHandlerMap   map
type CheckHandlerMap map[string]CheckHandlerFunc

//BaseCasePack pack testCase with some check info
type BaseCasePack struct {
	TCase      CaseFunc
	CheckTimes int
	TxHash     string
	TxReceipt  string
	PackID     string
	FLog       log15.Logger
	TLog       log15.Logger
}

//DefaultSend default send command implementation, only for transaction type case
func DefaultSend(testCase CaseFunc, testPack PackFunc, packID string) (PackFunc, error) {

	baseCase := testCase.GetBaseCase()
	txHash, bSuccess := SendTxCommand(baseCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := testPack.GetBasePack()
	pack.TxHash = txHash
	pack.TCase = testCase

	pack.PackID = packID
	pack.CheckTimes = 0
	return testPack, nil
}

//SendCommand interface CaseFunc implementing by BaseCase
func (t *BaseCase) SendCommand(packID string) (PackFunc, error) {
	return nil, nil
}

//GetID   id
func (t *BaseCase) GetID() string {

	return t.ID
}

//GetCmd   cmd
func (t *BaseCase) GetCmd() string {

	return t.Command
}

//GetDep   dep
func (t *BaseCase) GetDep() []string {

	return t.Dep
}

//GetRepeat   repeat
func (t *BaseCase) GetRepeat() int {

	return t.Repeat
}

//GetBaseCase
func (t *BaseCase) GetBaseCase() *BaseCase {

	return t
}

//SetDependData
func (t *BaseCase) SetDependData(interface{}) {

}

//interface PackFunc implementing by BaseCasePack

//GetPackID   pack id
func (pack *BaseCasePack) GetPackID() string {

	return pack.PackID
}

//SetPackID   pack id
func (pack *BaseCasePack) SetPackID(id string) {

	pack.PackID = id
}

//GetBaseCase
func (pack *BaseCasePack) GetBaseCase() *BaseCase {

	return pack.TCase.GetBaseCase()
}

//GetTxHash     hash
func (pack *BaseCasePack) GetTxHash() string {

	return pack.TxHash
}

//GetTxReceipt
func (pack *BaseCasePack) GetTxReceipt() string {

	return pack.TxReceipt
}

//SetLogger
func (pack *BaseCasePack) SetLogger(fLog log15.Logger, tLog log15.Logger) {

	pack.FLog = fLog
	pack.TLog = tLog
}

//GetBasePack     pack
func (pack *BaseCasePack) GetBasePack() *BaseCasePack {

	return pack
}

//GetDependData
func (pack *BaseCasePack) GetDependData() interface{} {

	return nil
}

//GetCheckHandlerMap   map
func (pack *BaseCasePack) GetCheckHandlerMap() interface{} {

	//return make(map[string]CheckHandlerFunc, 1)
	return nil
}

//CheckResult
func (pack *BaseCasePack) CheckResult(handlerMap interface{}) (bCheck bool, bSuccess bool) {

	bCheck = false
	bSuccess = false

	tCase := pack.TCase.GetBaseCase()
	txInfo, bReady := GetTxInfo(pack.TxHash)

	if !bReady && (txInfo != "tx not exist\n" || pack.CheckTimes >= CheckTimeout) {

		pack.TxReceipt = txInfo
		pack.FLog.Error("CheckTimeout", "TestID", pack.PackID, "ErrInfo", txInfo)
		pack.TxReceipt = txInfo
		return true, false
	}

	if bReady {

		bCheck = true
		var tyname string
		var jsonMap map[string]interface{}
		var txRecp dplatformosType.TxDetailResult
		pack.TxReceipt = txInfo
		pack.FLog.Info("TxReceiptJson", "TestID", pack.PackID)
		//hack, for pretty json log
		pack.FLog.Info("PrettyJsonLogFormat", "TxReceipt", []byte(txInfo))
		//
		err := json.Unmarshal([]byte(txInfo), &jsonMap)
		err1 := json.Unmarshal([]byte(txInfo), &txRecp)

		if err != nil || err1 != nil {

			pack.FLog.Error("UnMarshalFailed", "TestID", pack.PackID, "jsonStr", txInfo, "ErrInfo", err.Error())
			return true, false
		}

		tyname, bSuccess = GetTxRecpTyname(jsonMap)
		pack.FLog.Info("CheckItemResult", "TestID", pack.PackID, "RecpTyname", tyname)

		if !bSuccess {

			logArr := jsonMap["receipt"].(map[string]interface{})["logs"].([]interface{})
			logErrInfo := ""
			for _, log := range logArr {

				logMap := log.(map[string]interface{})

				if logMap["tyName"].(string) == "LogErr" {

					logErrInfo = logMap["log"].(string)
					break
				}
			}
			pack.FLog.Error("ExecPack", "TestID", pack.PackID,
				"LogErrInfo", logErrInfo)

		} else {

			//      autotest map
			if funcMap, ok := handlerMap.(CheckHandlerMapDiscard); ok {

				for _, item := range tCase.CheckItem {

					checkHandler, exist := funcMap[item]
					if exist {

						itemRes := checkHandler(jsonMap)
						bSuccess = bSuccess && itemRes
						pack.FLog.Info("CheckItemResult", "TestID", pack.PackID, "Item", item, "Passed", itemRes)
					}
				}

			} else if funcMap, ok := handlerMap.(CheckHandlerMap); ok { //

				for _, item := range tCase.CheckItem {

					checkHandler, exist := funcMap[item]
					if exist {

						itemRes := checkHandler(&txRecp)
						bSuccess = bSuccess && itemRes
						pack.FLog.Info("CheckItemResult", "TestID", pack.PackID, "Item", item, "Passed", itemRes)
					}
				}
			}
		}
	}

	pack.CheckTimes++
	return bCheck, bSuccess
}
