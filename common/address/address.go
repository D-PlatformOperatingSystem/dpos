// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package address
package address

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"unsafe"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/decred/base58"
	lru "github.com/hashicorp/golang-lru"
)

var addrSeed = []byte("DOM seed bytes for public key")
var addressCache *lru.Cache
var pubkey2AddrCache *lru.Cache
var checkAddressCache *lru.Cache
var multisignCache *lru.Cache
var multiCheckAddressCache *lru.Cache
var execPubKeyCache *lru.Cache

// ErrCheckVersion :
var ErrCheckVersion = errors.New("check version error")

//ErrCheckChecksum :
var ErrCheckChecksum = errors.New("Address Checksum error")

//ErrAddressChecksum :
var ErrAddressChecksum = errors.New("address checksum error")

//MaxExecNameLength
const MaxExecNameLength = 100

//NormalVer
const NormalVer byte = 0

//MultiSignVer
const MultiSignVer byte = 5

func init() {
	var err error
	multisignCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	pubkey2AddrCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	addressCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	checkAddressCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	multiCheckAddressCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	execPubKeyCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
}

//ExecAddress       ，   cache
func ExecAddress(name string) string {
	if value, ok := addressCache.Get(name); ok {
		return value.(string)
	}
	addr := GetExecAddress(name)
	addrstr := addr.String()
	addressCache.Add(name, addrstr)
	return addrstr
}

//MultiSignAddress create a multi sign address
func MultiSignAddress(pubkey []byte) string {
	skey := *(*string)(unsafe.Pointer(&pubkey))
	if value, ok := multisignCache.Get(skey); ok {
		return value.(string)
	}
	addr := HashToAddress(MultiSignVer, pubkey)
	addrstr := addr.String()
	multisignCache.Add(string(pubkey), addrstr)
	return addrstr
}

//ExecPubKey
func ExecPubKey(name string) []byte {
	if len(name) > MaxExecNameLength {
		panic("name too long")
	}
	if value, ok := execPubKeyCache.Get(name); ok {
		return value.([]byte)
	}
	var bname [200]byte
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := common.Sha2Sum(buf)
	execPubKeyCache.Add(name, hash)
	return hash[:]
}

//GetExecAddress
func GetExecAddress(name string) *Address {
	hash := ExecPubKey(name)
	addr := PubKeyToAddress(hash[:])
	return addr
}

//PubKeyToAddress
func PubKeyToAddress(in []byte) *Address {
	return HashToAddress(NormalVer, in)
}

//PubKeyToAddr
func PubKeyToAddr(in []byte) string {
	instr := *(*string)(unsafe.Pointer(&in))
	if value, ok := pubkey2AddrCache.Get(instr); ok {
		return value.(string)
	}
	addr := HashToAddress(NormalVer, in).String()
	pubkey2AddrCache.Add(string(in), addr)
	return addr
}

//HashToAddress hash32 to address
func HashToAddress(version byte, in []byte) *Address {
	a := new(Address)
	a.Pubkey = make([]byte, len(in))
	copy(a.Pubkey[:], in[:])
	a.Version = version
	a.SetBytes(common.Rimp160(in))
	return a
}

func checksum(input []byte) (cksum [4]byte) {
	h := sha256.Sum256(input)
	h2 := sha256.Sum256(h[:])
	copy(cksum[:], h2[:4])
	return
}

func checkAddress(ver byte, addr string) (e error) {

	dec := base58.Decode(addr)
	if dec == nil {
		e = errors.New("Cannot decode b58 string '" + addr + "'")
		checkAddressCache.Add(addr, e)
		return
	}
	if len(dec) < 25 {
		e = errors.New("Address too short " + hex.EncodeToString(dec))
		checkAddressCache.Add(addr, e)
		return
	}
	//version
	if dec[0] != ver {
		e = ErrCheckVersion
		return
	}
	//         (     ，        )
	if len(dec) == 25 {
		sh := common.Sha2Sum(dec[0:21])
		if !bytes.Equal(sh[:4], dec[21:25]) {
			e = ErrCheckChecksum
			return
		}
	}
	var cksum [4]byte
	copy(cksum[:], dec[len(dec)-4:])
	//    :
	if checksum(dec[:len(dec)-4]) != cksum {
		e = ErrAddressChecksum
	}
	return e
}

//CheckMultiSignAddress
func CheckMultiSignAddress(addr string) (e error) {
	if value, ok := multiCheckAddressCache.Get(addr); ok {
		if value == nil {
			return nil
		}
		return value.(error)
	}
	e = checkAddress(MultiSignVer, addr)
	multiCheckAddressCache.Add(addr, e)
	return
}

//CheckAddress
func CheckAddress(addr string) (e error) {
	if value, ok := checkAddressCache.Get(addr); ok {
		if value == nil {
			return nil
		}
		return value.(error)
	}
	e = checkAddress(NormalVer, addr)
	checkAddressCache.Add(addr, e)
	return
}

//NewAddrFromString new
func NewAddrFromString(hs string) (a *Address, e error) {
	dec := base58.Decode(hs)
	if dec == nil {
		e = errors.New("Cannot decode b58 string '" + hs + "'")
		return
	}
	if len(dec) < 25 {
		e = errors.New("Address too short " + hex.EncodeToString(dec))
		return
	}
	if len(dec) == 25 {
		sh := common.Sha2Sum(dec[0:21])
		if !bytes.Equal(sh[:4], dec[21:25]) {
			e = ErrCheckChecksum
		} else {
			a = new(Address)
			a.Version = dec[0]
			copy(a.Hash160[:], dec[1:21])
			a.Checksum = make([]byte, 4)
			copy(a.Checksum, dec[21:25])
			a.Enc58str = hs
		}
	}
	return
}

//Address
type Address struct {
	Version  byte
	Hash160  [20]byte // For a stealth address: it's HASH160
	Checksum []byte   // Unused for a stealth address
	Pubkey   []byte   // Unused for a stealth address
	Enc58str string
}

//SetBytes      bytes
func (a *Address) SetBytes(b []byte) {
	copy(a.Hash160[:], b)
}

func (a *Address) String() string {
	if a.Enc58str == "" {
		var ad [25]byte
		ad[0] = a.Version
		copy(ad[1:21], a.Hash160[:])
		if a.Checksum == nil {
			sh := common.Sha2Sum(ad[0:21])
			a.Checksum = make([]byte, 4)
			copy(a.Checksum, sh[:4])
		}
		copy(ad[21:25], a.Checksum[:])
		a.Enc58str = base58.Encode(ad[:])
	}
	return a.Enc58str
}
