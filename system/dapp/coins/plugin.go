// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package coins    coins dapp
package coins

import (
	"github.com/D-PlatformOperatingSystem/dpos/pluginmgr"
	_ "github.com/D-PlatformOperatingSystem/dpos/system/dapp/coins/autotest" // register package
	"github.com/D-PlatformOperatingSystem/dpos/system/dapp/coins/executor"
	"github.com/D-PlatformOperatingSystem/dpos/system/dapp/coins/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.CoinsX,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
}
