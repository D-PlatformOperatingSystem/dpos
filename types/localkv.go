// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"fmt"
)

//   key
var (
	LocalPrefix            = []byte("LODB")
	FlagTxQuickIndex       = []byte("FLAG:FlagTxQuickIndex")
	FlagKeyMVCC            = []byte("FLAG:keyMVCCFlag")
	TxHashPerfix           = []byte("TX:")
	TxShortHashPerfix      = []byte("STX:")
	TxAddrHash             = []byte("TxAddrHash:")
	TxAddrDirHash          = []byte("TxAddrDirHash:")
	AddrTxsCount           = []byte("AddrTxsCount:")
	ConsensusParaTxsPrefix = []byte("LODBP:Consensus:Para:")            //  para
	FlagReduceLocaldb      = []byte("FLAG:ReduceLocaldb")               //    localdb
	ReduceLocaldbHeight    = append(FlagReduceLocaldb, []byte(":H")...) //    localdb
)

// GetLocalDBKeyList   localdb key
func GetLocalDBKeyList() [][]byte {
	return [][]byte{
		FlagTxQuickIndex, FlagKeyMVCC, TxHashPerfix, TxShortHashPerfix, FlagReduceLocaldb,
	}
}

//CalcTxKey local db
func (c *DplatformOSConfig) CalcTxKey(hash []byte) []byte {
	if c.IsEnable("quickIndex") {
		return append(TxHashPerfix, hash...)
	}
	return hash
}

// CalcTxKeyValue   local db
func (c *DplatformOSConfig) CalcTxKeyValue(txr *TxResult) []byte {
	if c.IsEnable("reduceLocaldb") {
		txres := &TxResult{
			Height:     txr.GetHeight(),
			Index:      txr.GetIndex(),
			Blocktime:  txr.GetBlocktime(),
			ActionName: txr.GetActionName(),
		}
		return Encode(txres)
	}
	return Encode(txr)
}

//CalcTxShortKey local db
func CalcTxShortKey(hash []byte) []byte {
	return append(TxShortHashPerfix, hash[0:8]...)
}

//CalcTxAddrHashKey          hash  ，key=TxAddrHash:addr:height*100000 + index
//
func CalcTxAddrHashKey(addr string, heightindex string) []byte {
	return append(TxAddrHash, []byte(fmt.Sprintf("%s:%s", addr, heightindex))...)
}

//CalcTxAddrDirHashKey          hash  ，key=TxAddrHash:addr:flag:height*100000 + index
//
func CalcTxAddrDirHashKey(addr string, flag int32, heightindex string) []byte {
	return append(TxAddrDirHash, []byte(fmt.Sprintf("%s:%d:%s", addr, flag, heightindex))...)
}

//CalcAddrTxsCountKey            。add   ，del
func CalcAddrTxsCountKey(addr string) []byte {
	return append(AddrTxsCount, []byte(addr)...)
}

//StatisticFlag        key
func StatisticFlag() []byte {
	return []byte("Statistics:Flag")
}

//TotalFeeKey        key
func TotalFeeKey(hash []byte) []byte {
	key := []byte("TotalFeeKey:")
	return append(key, hash...)
}

//CalcLocalPrefix   localdb key
func CalcLocalPrefix(execer []byte) []byte {
	s := append([]byte("LODB-"), execer...)
	s = append(s, byte('-'))
	return s
}

//CalcStatePrefix   localdb key
func CalcStatePrefix(execer []byte) []byte {
	s := append([]byte("mavl-"), execer...)
	s = append(s, byte('-'))
	return s
}

//CalcRollbackKey      key
func CalcRollbackKey(execer []byte, hash []byte) []byte {
	prefix := CalcLocalPrefix(execer)
	key := append(prefix, []byte("rollback-")...)
	key = append(key, hash...)
	return key
}

//CalcConsensusParaTxsKey    localdb       title
func CalcConsensusParaTxsKey(key []byte) []byte {
	return append(ConsensusParaTxsPrefix, key...)
}

//CheckConsensusParaTxsKey   para               key
func CheckConsensusParaTxsKey(key []byte) bool {
	return bytes.HasPrefix(key, ConsensusParaTxsPrefix)
}
