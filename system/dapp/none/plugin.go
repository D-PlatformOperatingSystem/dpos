// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package none    dapp，
package none

import (
	"github.com/D-PlatformOperatingSystem/dpos/pluginmgr"
	"github.com/D-PlatformOperatingSystem/dpos/system/dapp/none/executor"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "none",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
}
