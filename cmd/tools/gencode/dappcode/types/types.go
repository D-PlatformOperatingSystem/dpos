// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/gencode/base"
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/types"
)

func init() {

	base.RegisterCodeFile(typesCode{})
}

type typesCode struct {
	base.DappCodeFile
}

func (c typesCode) GetDirName() string {

	return "types"
}

func (c typesCode) GetFiles() map[string]string {

	return map[string]string{
		typesName: typesContent,
	}
}

func (c typesCode) GetDirReplaceTags() []string {
	return []string{types.TagExecName}
}

func (c typesCode) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagExecObject, types.TagClassName,
		types.TagActionIDText, types.TagTyLogActionType,
		types.TagLogMapText, types.TagTypeMapText}
}

var (
	typesName    = "${EXECNAME}.go"
	typesContent = `package types

import (
log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
"github.com/D-PlatformOperatingSystem/dpos/types"
"reflect"
)

/* 
 *         
 *   action      log  ，          
 *    action log   id   name      
*/


// action  id name，           
${ACTIONIDTEXT}

// log  id 
${TYLOGACTIONTYPE}

var (
    //${CLASSNAME}X        
	${CLASSNAME}X = "${EXECNAME}"
	//  actionMap
	actionMap = ${TYPEMAPTEXT}
	//  log id   log     ，       log  
	logMap = ${LOGMAPTEXT}
	tlog = log.New("module", "${EXECNAME}.types")
)

// init defines a register function
func init() {
    types.AllowUserExec = append(types.AllowUserExec, []byte(${CLASSNAME}X))
	//        
	types.RegFork(${CLASSNAME}X, InitFork)
	types.RegExec(${CLASSNAME}X, InitExecutor)
}

// InitFork defines register fork
func InitFork(cfg *types.DplatformOSConfig) {
	cfg.RegisterDappFork(${CLASSNAME}X, "Enable", 0)
}

// InitExecutor defines register executor
func InitExecutor(cfg *types.DplatformOSConfig) {
	types.RegistorExecutor(${CLASSNAME}X, NewType(cfg))
}

type ${EXECNAME}Type struct {
    types.ExecTypeBase
}

func NewType(cfg *types.DplatformOSConfig) *${EXECNAME}Type {
    c := &${EXECNAME}Type{}
    c.SetChild(c)
    c.SetConfig(cfg)
    return c
}

// GetPayload     action  
func (${EXEC_OBJECT} *${EXECNAME}Type) GetPayload() types.Message {
    return &${CLASSNAME}Action{}
}

// GeTypeMap     action id name  
func (${EXEC_OBJECT} *${EXECNAME}Type) GetTypeMap() map[string]int32 {
    return actionMap
}

// GetLogMap     log    
func (${EXEC_OBJECT} *${EXECNAME}Type) GetLogMap() map[int64]*types.LogInfo {
    return logMap
}

`
)
