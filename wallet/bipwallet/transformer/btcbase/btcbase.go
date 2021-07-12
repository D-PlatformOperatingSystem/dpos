// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package btcbase
//         ：BTC、BCH、LTC、ZEC、USDT、 DOM
package btcbase

import (
	"crypto/sha256"
	"fmt"

	"github.com/D-PlatformOperatingSystem/dpos/common/address"

	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	"github.com/mr-tron/base58/base58"
	"golang.org/x/crypto/ripemd160"
)

// btcBaseTransformer
type btcBaseTransformer struct {
	prefix []byte //
}

// Base58ToByte base58
func (t btcBaseTransformer) Base58ToByte(str string) (bin []byte, err error) {
	bin, err = base58.Decode(str)
	return
}

// ByteToBase58      base58
func (t btcBaseTransformer) ByteToBase58(bin []byte) (str string) {
	str = base58.Encode(bin)
	return
}

//TODO:           ，

// PrivKeyToPub 32
func (t btcBaseTransformer) PrivKeyToPub(keyTy uint32, priv []byte) (pub []byte, err error) {
	if len(priv) != 32 && len(priv) != 64 {
		return nil, fmt.Errorf("invalid priv key byte")
	}
	//pub = secp256k1.PubkeyFromSeckey(priv)

	edcrypto, err := crypto.New(crypto.GetName(int(keyTy)))
	if err != nil {
		return nil, err
	}
	edkey, err := edcrypto.PrivKeyFromBytes(priv[:])
	if err != nil {
		return nil, err
	}
	return edkey.PubKey().Bytes(), nil

}

//checksum: first four bytes of double-SHA256.
func checksum(input []byte) (cksum [4]byte) {
	h := sha256.New()
	_, err := h.Write(input)
	if err != nil {
		return
	}
	intermediateHash := h.Sum(nil)
	h.Reset()
	_, err = h.Write(intermediateHash)
	if err != nil {
		return
	}
	finalHash := h.Sum(nil)
	copy(cksum[:], finalHash[:])
	return
}

// PubKeyToAddress              ，  base58
//（                    ，      ）
func (t btcBaseTransformer) PubKeyToAddress(pub []byte) (addr string, err error) {
	if len(pub) != 33 && len(pub) != 65 { //
		//return "", fmt.Errorf("invalid public key byte:%v", len(pub))
		return address.PubKeyToAddr(pub), nil
	}

	sha256h := sha256.New()
	_, err = sha256h.Write(pub)
	if err != nil {
		return "", err
	}
	//160hash
	ripemd160h := ripemd160.New()
	_, err = ripemd160h.Write(sha256h.Sum([]byte("")))
	if err != nil {
		return "", err
	}
	//
	hash160res := append(t.prefix, ripemd160h.Sum([]byte(""))...)

	//
	cksum := checksum(hash160res)
	address := append(hash160res, cksum[:]...)

	//    base58
	addr = base58.Encode(address)
	return
}
