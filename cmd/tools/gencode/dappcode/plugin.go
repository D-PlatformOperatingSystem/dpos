// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dappcode

import (
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/gencode/base"
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/types"
)

func init() {

	base.RegisterCodeFile(pluginCodeFile{})
}

type pluginCodeFile struct {
	base.DappCodeFile
}

func (c pluginCodeFile) GetFiles() map[string]string {

	return map[string]string{
		pluginName: pluginContent,
	}
}

func (c pluginCodeFile) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagClassName}
}

var (
	pluginName    = "plugin.go"
	pluginContent = `
package types

import (
	"${IMPORTPATH}/${EXECNAME}/commands"
	${EXECNAME}types "${IMPORTPATH}/${EXECNAME}/types"
	"${IMPORTPATH}/${EXECNAME}/executor"
	"${IMPORTPATH}/${EXECNAME}/rpc"
	"github.com/D-PlatformOperatingSystem/dpos/pluginmgr"
)

/*  
 *    dapp     
*/

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     ${EXECNAME}types.${CLASSNAME}X,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.Cmd,
		RPC:      rpc.Init,
	})
}`
)
