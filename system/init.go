// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package system        
package system

import (
	_ "github.com/D-PlatformOperatingSystem/dpos/system/consensus/init" //register consensus init package
	_ "github.com/D-PlatformOperatingSystem/dpos/system/crypto/init"
	_ "github.com/D-PlatformOperatingSystem/dpos/system/dapp/init"
	_ "github.com/D-PlatformOperatingSystem/dpos/system/mempool/init"
	_ "github.com/D-PlatformOperatingSystem/dpos/system/p2p/init" // init p2p plugin
	_ "github.com/D-PlatformOperatingSystem/dpos/system/store/init"
)
