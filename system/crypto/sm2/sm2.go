// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sm2
package sm2

import (
	"bytes"
	"errors"
	"fmt"

	"crypto/elliptic"
	"encoding/asn1"
	"math/big"

	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	"github.com/tjfoc/gmsm/sm2"
)

//const
const (
	SM2PrivateKeyLength    = 32
	SM2PublicKeyLength     = 65
	SM2PublicKeyCompressed = 33
)

//Driver
type Driver struct{}

//GenKey
func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := [SM2PrivateKeyLength]byte{}
	copy(privKeyBytes[:], crypto.CRandBytes(SM2PrivateKeyLength))
	priv, _ := privKeyFromBytes(sm2.P256Sm2(), privKeyBytes[:])
	copy(privKeyBytes[:], SerializePrivateKey(priv))
	return PrivKeySM2(privKeyBytes), nil
}

//PrivKeyFromBytes
func (d Driver) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	if len(b) != SM2PrivateKeyLength {
		return nil, errors.New("invalid priv key byte")
	}
	privKeyBytes := new([SM2PrivateKeyLength]byte)
	copy(privKeyBytes[:], b[:SM2PrivateKeyLength])

	priv, _ := privKeyFromBytes(sm2.P256Sm2(), privKeyBytes[:])

	copy(privKeyBytes[:], SerializePrivateKey(priv))
	return PrivKeySM2(*privKeyBytes), nil
}

//PubKeyFromBytes
func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	if len(b) != SM2PublicKeyLength && len(b) != SM2PublicKeyCompressed {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([SM2PublicKeyLength]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeySM2(*pubKeyBytes), nil
}

//SignatureFromBytes
func (d Driver) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	var certSignature crypto.CertSignature
	_, err = asn1.Unmarshal(b, &certSignature)
	if err != nil {
		return SignatureSM2(b), nil
	}

	if len(certSignature.Cert) == 0 {
		return SignatureSM2(b), nil
	}

	return SignatureSM2(certSignature.Signature), nil
}

//PrivKeySM2
type PrivKeySM2 [SM2PrivateKeyLength]byte

//Bytes
func (privKey PrivKeySM2) Bytes() []byte {
	s := make([]byte, SM2PrivateKeyLength)
	copy(s, privKey[:])
	return s
}

//Sign
func (privKey PrivKeySM2) Sign(msg []byte) crypto.Signature {
	priv, _ := privKeyFromBytes(sm2.P256Sm2(), privKey[:])
	r, s, err := sm2.Sign(priv, crypto.Sm3Hash(msg))
	if err != nil {
		return nil
	}
	//sm2   LowS
	//s = ToLowS(pub, s)
	return SignatureSM2(Serialize(r, s))
}

//PubKey
func (privKey PrivKeySM2) PubKey() crypto.PubKey {
	_, pub := privKeyFromBytes(sm2.P256Sm2(), privKey[:])
	var pubSM2 PubKeySM2
	copy(pubSM2[:], sm2.Compress(pub))
	return pubSM2
}

//Equals
func (privKey PrivKeySM2) Equals(other crypto.PrivKey) bool {
	if otherSecp, ok := other.(PrivKeySM2); ok {
		return bytes.Equal(privKey[:], otherSecp[:])
	}

	return false
}

func (privKey PrivKeySM2) String() string {
	return fmt.Sprintf("PrivKeySM2{*****}")
}

//PubKeySM2
type PubKeySM2 [SM2PublicKeyLength]byte

//Bytes
func (pubKey PubKeySM2) Bytes() []byte {
	length := SM2PublicKeyLength
	if pubKey.isCompressed() {
		length = SM2PublicKeyCompressed
	}
	s := make([]byte, length)
	copy(s, pubKey[0:length])
	return s
}

func (pubKey PubKeySM2) isCompressed() bool {
	return pubKey[0] != pubkeyUncompressed
}

//VerifyBytes
func (pubKey PubKeySM2) VerifyBytes(msg []byte, sig crypto.Signature) bool {
	if wrap, ok := sig.(SignatureS); ok {
		sig = wrap.Signature
	}
	sigSM2, ok := sig.(SignatureSM2)
	if !ok {
		fmt.Printf("convert failed\n")
		return false
	}
	var pub *sm2.PublicKey
	if pubKey.isCompressed() {
		pub = sm2.Decompress(pubKey[0:SM2PublicKeyCompressed])
	} else {
		var err error
		pub, err = parsePubKey(pubKey[:], sm2.P256Sm2())
		if err != nil {
			fmt.Printf("parse pubkey failed\n")
			return false
		}
	}
	r, s, err := Deserialize(sigSM2)
	if err != nil {
		fmt.Printf("unmarshal sign failed")
		return false
	}
	//       ecdsa   ，-s     ，     LowS
	//fmt.Printf("verify:%x, r:%d, s:%d\n", crypto.Sm3Hash(msg), r, s)
	//lowS := IsLowS(s)
	//if !lowS {
	//	fmt.Printf("lowS check failed")
	//	return false
	//}

	return sm2.Verify(pub, crypto.Sm3Hash(msg), r, s)
}

func (pubKey PubKeySM2) String() string {
	return fmt.Sprintf("PubKeySM2{%X}", pubKey[:])
}

//KeyString Must return the full bytes in hex.
// Used for map keying, etc.
func (pubKey PubKeySM2) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

//Equals
func (pubKey PubKeySM2) Equals(other crypto.PubKey) bool {
	if otherSecp, ok := other.(PubKeySM2); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	}
	return false
}

//SignatureSM2
type SignatureSM2 []byte

//SignatureS
type SignatureS struct {
	crypto.Signature
}

//Bytes
func (sig SignatureSM2) Bytes() []byte {
	s := make([]byte, len(sig))
	copy(s, sig[:])
	return s
}

//IsZero    0
func (sig SignatureSM2) IsZero() bool { return len(sig) == 0 }

func (sig SignatureSM2) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)

}

//Equals
func (sig SignatureSM2) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureSM2); ok {
		return bytes.Equal(sig[:], otherEd[:])
	}
	return false
}

//const
const (
	Name = "sm2"
	ID   = 3
)

func init() {
	crypto.Register(Name, &Driver{}, false)
	crypto.RegisterType(Name, ID)
}

func privKeyFromBytes(curve elliptic.Curve, pk []byte) (*sm2.PrivateKey, *sm2.PublicKey) {
	x, y := curve.ScalarBaseMult(pk)

	priv := &sm2.PrivateKey{
		PublicKey: sm2.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(pk),
	}

	return priv, &priv.PublicKey
}
