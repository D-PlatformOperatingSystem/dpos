// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package init
package init

//
//      ，     ，              ，
import (
	//
	_ "github.com/D-PlatformOperatingSystem/dpos/system/crypto/ed25519"
	_ "github.com/D-PlatformOperatingSystem/dpos/system/crypto/secp256k1"
	_ "github.com/D-PlatformOperatingSystem/dpos/system/crypto/sm2"
)
