// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/D-PlatformOperatingSystem/dpos/types"

	"github.com/D-PlatformOperatingSystem/dpos/wallet/bipwallet"
)

func TestBipwallet(t *testing.T) {
	/*

	    TypeEther:        "ETH",
		TypeEtherClassic: "ETC",
		TypeBitcoin:      "BTC",
		TypeLitecoin:     "LTC",
		TypeZayedcoin:    "ZEC",
		TypeDpos:          "DOM",
		TypeYcc:          "YCC",
	*/
	//bitsize=128   12       ，bitsize+32=160    15       ，bitszie=256   24
	mnem, err := bipwallet.NewMnemonicString(0, 160)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("   :", mnem)
	//    ，      wallet
	wallet, err := bipwallet.NewWalletFromMnemonic(bipwallet.TypeEther, types.SECP256K1,
		"wish address cram damp very indicate regret sound figure scheme review scout")
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}
	var index uint32
	//      Key pair
	priv, pub, err := wallet.NewKeyPair(index)
	fmt.Println("privkey:", hex.EncodeToString(priv))
	fmt.Println("pubkey:", hex.EncodeToString(pub))
	//
	address, err := wallet.NewAddress(index)
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}

	fmt.Println("address:", address)
	address, err = bipwallet.PubToAddress(pub)
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}

	fmt.Println("PubToAddress:", address)

	pub, err = bipwallet.PrivkeyToPub(bipwallet.TypeEther, types.SECP256K1, priv)
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}

	fmt.Println("PrivToPub:", hex.EncodeToString(pub))

}
