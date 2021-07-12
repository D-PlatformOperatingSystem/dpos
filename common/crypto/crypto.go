// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package crypto    、
package crypto

import (
	"errors"
	"fmt"
	"sync"
)

//ErrNotSupportAggr
var ErrNotSupportAggr = errors.New("AggregateCrypto not support")

//PrivKey
type PrivKey interface {
	Bytes() []byte
	Sign(msg []byte) Signature
	PubKey() PubKey
	Equals(PrivKey) bool
}

//Signature
type Signature interface {
	Bytes() []byte
	IsZero() bool
	String() string
	Equals(Signature) bool
}

//PubKey
type PubKey interface {
	Bytes() []byte
	KeyString() string
	VerifyBytes(msg []byte, sig Signature) bool
	Equals(PubKey) bool
}

//Crypto
type Crypto interface {
	GenKey() (PrivKey, error)
	SignatureFromBytes([]byte) (Signature, error)
	PrivKeyFromBytes([]byte) (PrivKey, error)
	PubKeyFromBytes([]byte) (PubKey, error)
}

//AggregateCrypto
type AggregateCrypto interface {
	Aggregate(sigs []Signature) (Signature, error)
	AggregatePublic(pubs []PubKey) (PubKey, error)
	VerifyAggregatedOne(pubs []PubKey, m []byte, sig Signature) error
	VerifyAggregatedN(pubs []PubKey, ms [][]byte, sig Signature) error
}

//ToAggregate               ，
func ToAggregate(c Crypto) (AggregateCrypto, error) {
	if aggr, ok := c.(AggregateCrypto); ok {
		return aggr, nil
	}
	return nil, ErrNotSupportAggr
}

var (
	drivers     = make(map[string]Crypto)
	driversCGO  = make(map[string]Crypto)
	driversType = make(map[string]int)
	driverMutex sync.Mutex
)

//Register       ，         cgo
func Register(name string, driver Crypto, isCGO bool) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	d := drivers
	if isCGO {
		d = driversCGO
	}
	if _, dup := d[name]; dup {
		panic("crypto: Register called twice for driver " + name)
	}
	d[name] = driver
}

//RegisterType
func RegisterType(name string, ty int) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	for n, t := range driversType {
		//      cgo  ，  name，ty     ，
		//      ，
		//            ，           ty  name
		if (n == name && t != ty) || (n != name && t == ty) {
			panic(fmt.Sprintf("crypto: Register Conflict, exist=(%s,%d), register=(%s, %d)", n, t, name, ty))
		}
	}
	driversType[name] = ty
}

//GetName   name
func GetName(ty int) string {
	for name, t := range driversType {
		if t == ty {
			return name
		}
	}
	return "unknown"
}

//GetType   type
func GetType(name string) int {
	if ty, ok := driversType[name]; ok {
		return ty
	}
	return 0
}

//New new
func New(name string) (c Crypto, err error) {

	//         cgo
	c, ok := driversCGO[name]
	if ok {
		return c, nil
	}
	//   cgo,
	c, ok = drivers[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
	}
	return c, err
}

//CertSignature
type CertSignature struct {
	Signature []byte
	Cert      []byte
}
