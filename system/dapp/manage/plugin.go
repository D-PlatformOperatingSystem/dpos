// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package manage manage
// 1.
// 2.
// 3. （  ）
package manage

import (
	"github.com/D-PlatformOperatingSystem/dpos/pluginmgr"
	"github.com/D-PlatformOperatingSystem/dpos/system/dapp/manage/commands"
	"github.com/D-PlatformOperatingSystem/dpos/system/dapp/manage/executor"
	"github.com/D-PlatformOperatingSystem/dpos/system/dapp/manage/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.ManageX,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.ConfigCmd,
		RPC:      nil,
	})
}
