// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package init      dapp
package init

import (
	_ "github.com/D-PlatformOperatingSystem/dpos/system/dapp/coins"  // register coins package
	_ "github.com/D-PlatformOperatingSystem/dpos/system/dapp/manage" // register manage package
	_ "github.com/D-PlatformOperatingSystem/dpos/system/dapp/none"   // register none package
)
