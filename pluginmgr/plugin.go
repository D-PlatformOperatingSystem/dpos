// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pluginmgr

import (
	"github.com/D-PlatformOperatingSystem/dpos/rpc/types"
	typ "github.com/D-PlatformOperatingSystem/dpos/types"
	wcom "github.com/D-PlatformOperatingSystem/dpos/wallet/common"
	"github.com/spf13/cobra"
)

// Plugin plugin module struct
type Plugin interface {
	//          ，       、
	GetName() string
	//
	GetExecutorName() string
	//
	InitExec(cfg *typ.DplatformOSConfig)
	InitWallet(wallet wcom.WalletOperate, sub map[string][]byte)
	AddCmd(rootCmd *cobra.Command)
	AddRPC(s types.RPCServer)
}
