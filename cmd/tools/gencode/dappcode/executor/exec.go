// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/gencode/base"
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/types"
)

func init() {
	base.RegisterCodeFile(execCode{})
	base.RegisterCodeFile(execLocalCode{})
	base.RegisterCodeFile(execDelLocalCode{})
}

type execCode struct {
	executorCodeFile
}

func (execCode) GetFiles() map[string]string {

	return map[string]string{
		execName: execContent,
	}
}

func (execCode) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagClassName, types.TagExecFileContent, types.TagExecObject}
}

type execLocalCode struct {
	executorCodeFile
}

func (execLocalCode) GetFiles() map[string]string {

	return map[string]string{
		execLocalName: execLocalContent,
	}
}

func (execLocalCode) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagExecLocalFileContent, types.TagExecObject}
}

type execDelLocalCode struct {
	executorCodeFile
}

func (execDelLocalCode) GetFiles() map[string]string {

	return map[string]string{
		execDelName: execDelContent,
	}
}

func (execDelLocalCode) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagExecDelLocalFileContent, types.TagExecObject}
}

var (
	execName    = "exec.go"
	execContent = `package executor

import (
	${EXECNAME}types "${IMPORTPATH}/${EXECNAME}/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

/*
 *            
 *       （statedb）       （log）
*/

${EXECFILECONTENT}`

	execLocalName    = "exec_local.go"
	execLocalContent = `package executor

import (
	${EXECNAME}types "${IMPORTPATH}/${EXECNAME}/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

/*
 *             ，     
 *      ，    (localDB),       ，   
*/

${EXECLOCALFILECONTENT}

//      ，        localdb kv，   exec-local   kv    
func (${EXEC_OBJECT} *${EXECNAME}) addAutoRollBack(tx *types.Transaction, kv []*types.KeyValue) *types.LocalDBSet {

	dbSet := &types.LocalDBSet{}
	dbSet.KV = ${EXEC_OBJECT}.AddRollbackKV(tx, tx.Execer, kv)
	return dbSet
}
`

	execDelName    = "exec_del_local.go"
	execDelContent = `package executor

import (
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

/* 
 *                 
*/

// ExecDelLocal localdb kv        
func (${EXEC_OBJECT} *${EXECNAME}) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kvs, err := ${EXEC_OBJECT}.DelRollbackKV(tx, tx.Execer)
	if err != nil {
		return nil, err
	}
	dbSet := &types.LocalDBSet{}
	dbSet.KV = append(dbSet.KV, kvs...)
	return dbSet, nil
}
`
)
