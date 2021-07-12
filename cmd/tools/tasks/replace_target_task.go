// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/types"
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/util"
)

// ReplaceTargetTask
//           ：
// ${PROJECTNAME}
// ${CLASSNAME}
// ${ACTIONNAME}
// ${EXECNAME}
type ReplaceTargetTask struct {
	TaskBase
	OutputPath  string
	ProjectName string
	ClassName   string
	ActionName  string
	ExecName    string
}

//GetName   name
func (r *ReplaceTargetTask) GetName() string {
	return "ReplaceTargetTask"
}

// Execute
// 1.      output
// 2.        ，
// 3.              ，         ，
// 4.
func (r *ReplaceTargetTask) Execute() error {
	mlog.Info("Execute replace target task.")
	err := filepath.Walk(r.OutputPath, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if err := r.replaceTarget(path); err != nil {
			mlog.Error("replaceTarget error", "error", err, "path", path)
			return err
		}
		return nil
	})
	return err
}

func (r *ReplaceTargetTask) replaceTarget(file string) error {
	replacePairs := []struct {
		src string
		dst string
	}{
		{src: types.TagProjectName, dst: r.ProjectName},
		{src: types.TagClassName, dst: r.ClassName},
		{src: types.TagActionName, dst: r.ActionName},
		{src: types.TagExecName, dst: r.ExecName},
	}
	bcontent, err := util.ReadFile(file)
	if err != nil {
		return err
	}
	content := string(bcontent)
	for _, pair := range replacePairs {
		content = strings.Replace(content, pair.src, pair.dst, -1)
	}
	util.DeleteFile(file)
	_, err = util.WriteStringToFile(file, content)
	return err
}
