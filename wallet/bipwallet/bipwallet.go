// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bipwallet
package bipwallet

import (
	"errors"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	bip32 "github.com/D-PlatformOperatingSystem/dpos/wallet/bipwallet/go-bip32"
	bip39 "github.com/D-PlatformOperatingSystem/dpos/wallet/bipwallet/go-bip39"
	bip44 "github.com/D-PlatformOperatingSystem/dpos/wallet/bipwallet/go-bip44"
	"github.com/D-PlatformOperatingSystem/dpos/wallet/bipwallet/transformer"
	_ "github.com/D-PlatformOperatingSystem/dpos/wallet/bipwallet/transformer/btcbase" //register
)

// https://github.com/satoshilabs/slips/blob/master/slip-0044.md
const (
	TypeBitcoin            uint32 = 0x80000000
	TypeLitecoin           uint32 = 0x80000002
	TypeEther              uint32 = 0x8000003c
	TypeEtherClassic       uint32 = 0x8000003d
	TypeFactomFactoids     uint32 = 0x80000083
	TypeFactomEntryCredits uint32 = 0x80000084
	TypeZcash              uint32 = 0x80000085
	TypeDpos               uint32 = 0x80003333
	TypeYcc                uint32 = 0x80003334
)

// CoinName
var CoinName = map[uint32]string{
	TypeEther:        "ETH",
	TypeEtherClassic: "ETC",
	TypeBitcoin:      "BTC",
	TypeLitecoin:     "LTC",
	TypeZcash:        "ZEC",
	TypeDpos:         "DOM",
	TypeYcc:          "YCC",
}

// coinNameType
var coinNameType = map[string]uint32{
	"ETH": TypeEther,
	"ETC": TypeEtherClassic,
	"BTC": TypeBitcoin,
	"LTC": TypeLitecoin,
	"ZEC": TypeZcash,
	"DOM": TypeDpos,
	"YCC": TypeYcc,
}

//GetSLIP0044CoinType       CoinType
func GetSLIP0044CoinType(name string) uint32 {
	name = strings.ToUpper(name)
	if ty, ok := coinNameType[name]; ok {
		return ty
	}
	log.Error("GetSLIP0044CoinType: " + name + " not exist.")
	return TypeDpos
}

// HDWallet   BIP-44   HD
type HDWallet struct {
	CoinType  uint32
	RootSeed  []byte
	MasterKey *bip32.Key
	KeyType   uint32
}

// NewKeyPair
func (w *HDWallet) NewKeyPair(index uint32) (priv, pub []byte, err error) {
	//bip44    32
	key, err := bip44.NewKeyFromMasterKey(w.MasterKey, w.CoinType, bip32.FirstHardenedChild, 0, index)
	if err != nil {
		return nil, nil, err
	}
	if w.KeyType == types.SECP256K1 {
		return key.Key, key.PublicKey().Key, err
	}

	edcrypto, err := crypto.New(crypto.GetName(int(w.KeyType)))
	if err != nil {
		return nil, nil, err
	}
	edkey, err := edcrypto.PrivKeyFromBytes(key.Key[:])
	if err != nil {
		return nil, nil, err
	}

	priv = edkey.Bytes()
	pub = edkey.PubKey().Bytes()

	return
}

// NewAddress
func (w *HDWallet) NewAddress(index uint32) (string, error) {
	if cointype, ok := CoinName[w.CoinType]; ok {
		_, pub, err := w.NewKeyPair(index)
		if err != nil {
			return "", err
		}

		trans, err := transformer.New(cointype)
		if err != nil {
			return "", err
		}
		addr, err := trans.PubKeyToAddress(pub)
		if err != nil {
			return "", err
		}
		return addr, nil
	}

	return "", errors.New("cointype no support to create address")

}

// PrivkeyToPub
func PrivkeyToPub(coinType, keyTy uint32, priv []byte) ([]byte, error) {
	if cointype, ok := CoinName[coinType]; ok {
		trans, err := transformer.New(cointype)
		if err != nil {
			return nil, err
		}
		pub, err := trans.PrivKeyToPub(keyTy, priv)
		if err != nil {
			return nil, err
		}

		return pub, nil

	}
	return nil, errors.New("cointype no support to create address")
}

// PubToAddress
func PubToAddress(pub []byte) (string, error) {
	return address.PubKeyToAddr(pub), nil
}

//NewMnemonicString       lang=0      ，lang=1      bitsize=[128,256]  bitsize%32=0
func NewMnemonicString(lang, bitsize int) (string, error) {
	entropy, err := bip39.NewEntropy(bitsize)
	if err != nil {
		return "", err
	}

	mnemonic, err := bip39.NewMnemonic(entropy, int32(lang))
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

// NewWalletFromMnemonic
func NewWalletFromMnemonic(coinType, keyType uint32, mnemonic string) (wallet *HDWallet, err error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	return NewWalletFromSeed(coinType, keyType, seed)
}

// NewWalletFromSeed
func NewWalletFromSeed(coinType, keyType uint32, seed []byte) (wallet *HDWallet, err error) {
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, err
	}
	return &HDWallet{
		CoinType:  coinType,
		KeyType:   keyType,
		RootSeed:  seed,
		MasterKey: masterKey}, nil
}
