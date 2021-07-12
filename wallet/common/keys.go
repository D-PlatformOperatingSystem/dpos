// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import "fmt"

const (
	keyAccount            = "Account"
	keyAddr               = "Addr"
	keyLabel              = "Label"
	keyTx                 = "Tx"
	keyEncryptionFlag     = "Encryption"
	keyEncryptionCompFlag = "EncryptionFlag" //                    ，             ，
	keyPasswordHash       = "PasswordHash"
	keyWalletSeed         = "walletseed"
	keyAirDropIndex       = "AirDropIndex" //    seed
)

// CalcAccountKey     Account     list，
func CalcAccountKey(timestamp string, addr string) []byte {
	return []byte(fmt.Sprintf("%s:%s:%s", keyAccount, timestamp, addr))
}

// CalcAddrKey   addr    Account
func CalcAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyAddr, addr))
}

// CalcLabelKey   label  Account
func CalcLabelKey(label string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyLabel, label))
}

// CalcTxKey   height*100000+index   Tx
//key:Tx:height*100000+index
func CalcTxKey(key string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyTx, key))
}

// CalcEncryptionFlag     Key
func CalcEncryptionFlag() []byte {
	return []byte(keyEncryptionFlag)
}

// CalckeyEncryptionCompFlag       Key
func CalckeyEncryptionCompFlag() []byte {
	return []byte(keyEncryptionCompFlag)
}

// CalcPasswordHash   hash Key
func CalcPasswordHash() []byte {
	return []byte(keyPasswordHash)
}

// CalcWalletSeed   Seed Key
func CalcWalletSeed() []byte {
	return []byte(keyWalletSeed)
}

// CalcAirDropIndex     Seed  index
func CalcAirDropIndex() []byte {
	return []byte(keyAirDropIndex)
}
