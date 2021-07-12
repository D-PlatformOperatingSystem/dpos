// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"
)

var slash = []byte("-")
var sharp = []byte("#")

//Debug
var Debug = false

//LogErr log
type LogErr []byte

//LogReserved LogReserved
type LogReserved []byte

//LogInfo loginfo
type LogInfo struct {
	Ty   reflect.Type
	Name string
}

//UserKeyX
const (
	UserKeyX = "user."
	ParaKeyX = "user.p."
	NoneX    = "none"
)

//DefaultCoinsSymbol
const (
	DefaultCoinsSymbol = "dpos"
)

//UserKeyX           byte
var (
	UserKey    = []byte(UserKeyX)
	ParaKey    = []byte(ParaKeyX)
	ExecerNone = []byte(NoneX)
)

//
const (
	InputPrecision        float64 = 1e4
	Multiple1E4           int64   = 1e4
	DOM                           = "DOM"
	TxGroupMaxCount               = 20
	MinerAction                   = "miner"
	Int1E4                int64   = 10000
	Float1E4              float64 = 10000.0
	AirDropMinIndex       uint32  = 100000000         //     seed        ，  index
	AirDropMaxIndex       uint32  = 101000000         //     seed        ，  index
	MaxBlockCountPerTime  int64   = 1000              //          block     1000
	MaxBlockSizePerTime           = 100 * 1024 * 1024 //          block   size100M
	AddBlock              int64   = 1
	DelBlock              int64   = 2
	MainChainName                 = "main"
	MaxHeaderCountPerTime int64   = 10000 //          header     10000

)

//ty = 1 -> secp256k1
//ty = 2 -> ed25519
//ty = 3 -> sm2
//ty = 4 -> onetimeed25519
//ty = 5 -> RingBaseonED25519
//ty = 1+offset(1<<8) ->auth_ecdsa
//ty = 2+offset(1<<8) -> auth_sm2
const (
	Invalid   = 0
	SECP256K1 = 1
	ED25519   = 2
	SM2       = 3
)

//log type
const (
	TyLogReserved = 0
	TyLogErr      = 1
	TyLogFee      = 2
	//TyLogTransfer coins
	TyLogTransfer        = 3
	TyLogGenesis         = 4
	TyLogDeposit         = 5
	TyLogExecTransfer    = 6
	TyLogExecWithdraw    = 7
	TyLogExecDeposit     = 8
	TyLogExecFrozen      = 9
	TyLogExecActive      = 10
	TyLogGenesisTransfer = 11
	TyLogGenesisDeposit  = 12
	TyLogRollback        = 13
	TyLogMint            = 14
	TyLogBurn            = 15
)

//SystemLog   log
var SystemLog = map[int64]*LogInfo{
	TyLogReserved:        {reflect.TypeOf(LogReserved{}), "LogReserved"},
	TyLogErr:             {reflect.TypeOf(LogErr{}), "LogErr"},
	TyLogFee:             {reflect.TypeOf(ReceiptAccountTransfer{}), "LogFee"},
	TyLogTransfer:        {reflect.TypeOf(ReceiptAccountTransfer{}), "LogTransfer"},
	TyLogDeposit:         {reflect.TypeOf(ReceiptAccountTransfer{}), "LogDeposit"},
	TyLogExecTransfer:    {reflect.TypeOf(ReceiptExecAccountTransfer{}), "LogExecTransfer"},
	TyLogExecWithdraw:    {reflect.TypeOf(ReceiptExecAccountTransfer{}), "LogExecWithdraw"},
	TyLogExecDeposit:     {reflect.TypeOf(ReceiptExecAccountTransfer{}), "LogExecDeposit"},
	TyLogExecFrozen:      {reflect.TypeOf(ReceiptExecAccountTransfer{}), "LogExecFrozen"},
	TyLogExecActive:      {reflect.TypeOf(ReceiptExecAccountTransfer{}), "LogExecActive"},
	TyLogGenesisTransfer: {reflect.TypeOf(ReceiptAccountTransfer{}), "LogGenesisTransfer"},
	TyLogGenesisDeposit:  {reflect.TypeOf(ReceiptAccountTransfer{}), "LogGenesisDeposit"},
	TyLogRollback:        {reflect.TypeOf(LocalDBSet{}), "LogRollback"},
	TyLogMint:            {reflect.TypeOf(ReceiptAccountMint{}), "LogMint"},
	TyLogBurn:            {reflect.TypeOf(ReceiptAccountBurn{}), "LogBurn"},
}

//exec type
const (
	ExecErr  = 0
	ExecPack = 1
	ExecOk   = 2
)

// TODO
//func init() {
//	S("TxHeight", false)
//}

//flag:

//TxHeight
//    :
//               ，
//

//TxHeightFlag             TxHeight
var TxHeightFlag int64 = 1 << 62

//HighAllowPackHeight eg: current Height is 10000
//TxHeight is  10010
//=> Height <= TxHeight + HighAllowPackHeight
//=> Height >= TxHeight - LowAllowPackHeight
//            : 10010 - 100 = 9910 , 10010 + 200 =  10210 (9910,10210)
//
//  ，           .
//       :
//    ，         ，          (9910,10210)。
//           ，      9910 - currentHeight
var HighAllowPackHeight int64 = 90

//LowAllowPackHeight      low
var LowAllowPackHeight int64 = 30

//MaxAllowPackInterval
var MaxAllowPackInterval int64 = 5000
