// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/gencode/base"
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/types"
)

func init() {

	base.RegisterCodeFile(executorCodeFile{})
}

type executorCodeFile struct {
	base.DappCodeFile
}

func (c executorCodeFile) GetDirName() string {

	return "executor"
}

func (c executorCodeFile) GetFiles() map[string]string {

	return map[string]string{
		executorName: executorContent,
		kvName:       kvContent,
	}
}

func (c executorCodeFile) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagExecObject, types.TagImportPath, types.TagClassName}
}

var (
	executorName    = "${EXECNAME}.go"
	executorContent = `package executor

import (
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	${EXECNAME}types "${IMPORTPATH}/${EXECNAME}/types"
	drivers "github.com/D-PlatformOperatingSystem/dpos/system/dapp"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

/* 
 *        
 *         
*/ 


var (
	//  
	elog = log.New("module", "${EXECNAME}.executor")
)

var driverName = ${EXECNAME}types.${CLASSNAME}X

// Init register dapp
func Init(name string, cfg *types.DplatformOSConfig, sub []byte) {
	drivers.Register(cfg, GetName(), new${CLASSNAME}, cfg.GetDappFork(driverName, "Enable"))
    InitExecType()
}

// InitExecType Init Exec Type
func InitExecType() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&${EXECNAME}{}))
}

type ${EXECNAME} struct {
	drivers.DriverBase
}

func new${CLASSNAME}() drivers.Driver {
	t := &${EXECNAME}{}
	t.SetChild(t)
	t.SetExecutorType(types.LoadExecutorType(driverName))
	return t
}

// GetName get driver name
func GetName() string {
	return new${CLASSNAME}().GetName()
}

func (${EXEC_OBJECT} *${EXECNAME}) GetDriverName() string {
	return driverName
}

// CheckTx            ，     
func (${EXEC_OBJECT} *${EXECNAME}) CheckTx(tx *types.Transaction, index int) error {
	// implement code
	return nil
}

`

	kvName    = "kv.go"
	kvContent = `package executor

/*
 *       kv   ，key           
 *  key = keyPrefix + userKey
 *          ，  ’-‘      
*/

var (
	//KeyPrefixStateDB state db key    
	KeyPrefixStateDB = "mavl-${EXECNAME}-"
	//KeyPrefixLocalDB local db key    
	KeyPrefixLocalDB = "LODB-${EXECNAME}-"
)
`
)
