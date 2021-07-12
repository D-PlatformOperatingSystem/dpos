// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package btcbase

import (
	"github.com/D-PlatformOperatingSystem/dpos/wallet/bipwallet/transformer"
)

//
var coinPrefix = map[string][]byte{
	"BTC":  {0x00},
	"BCH":  {0x00},
	"DOM":  {0x00},
	"YCC":  {0x00},
	"LTC":  {0x30},
	"ZEC":  {0x1c, 0xb8},
	"USDT": {0x00},
}

func init() {
	//
	for name, prefix := range coinPrefix {
		transformer.Register(name, &btcBaseTransformer{prefix})
	}
}
