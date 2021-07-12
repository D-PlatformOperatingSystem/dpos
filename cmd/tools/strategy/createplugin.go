//Copyright D-Platform Corp. 2018 All Rights Reserved.
//Use of this source code is governed by a BSD-style
//license that can be found in the LICENSE file.

package strategy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/util"

	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/tasks"
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/types"
	"github.com/pkg/errors"
)

// createPluginStrategy         ，
type createPluginStrategy struct {
	strategyBasic

	gopath          string
	projName        string //
	execName        string //
	execNameFB      string
	className       string
	classTypeName   string
	classActionName string
	outRootPath     string //
	projectPath     string //        ,

}

func (c *createPluginStrategy) Run() error {
	fmt.Println("Begin run dplatformos create plugin project mode.")
	defer fmt.Println("Run dplatformos create plugin project mode finish.")
	if err := c.initMember(); err != nil {
		return err
	}
	return c.rumImpl()
}

func (c *createPluginStrategy) initMember() error {
	gopath := os.Getenv("GOPATH")
	if len(gopath) <= 0 {
		return errors.New("Can't find GOPATH")
	}
	c.gopath = gopath
	c.outRootPath = filepath.Join(gopath, "/src/github.com/D-PlatformOperatingSystem")
	c.projName, _ = c.getParam(types.KeyProjectName)
	c.execName, _ = c.getParam(types.KeyExecutorName)
	c.className, _ = c.getParam(types.KeyClassName)
	c.projectPath = fmt.Sprintf("%s/%s", c.outRootPath, c.projName)
	c.execNameFB, _ = util.MakeStringToUpper(c.execName, 0, 1)
	c.classTypeName = c.execNameFB + "Type"
	c.classActionName = c.execNameFB + "Action"
	return nil
}

func (c *createPluginStrategy) rumImpl() error {
	var err error
	tasks := c.buildTask()
	for _, task := range tasks {
		err = task.Execute()
		if err != nil {
			mlog.Error("Execute command failed.", "error", err, "taskname", task.GetName())
			break
		}
	}
	return err
}

func (c *createPluginStrategy) buildTask() []tasks.Task {
	//        gopath/src
	goprojpath := strings.Replace(c.projectPath, c.gopath+"/src/", "", -1)
	taskSlice := make([]tasks.Task, 0)
	taskSlice = append(taskSlice,
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftMainGo,
			OutputFile: fmt.Sprintf("%s/main.go", c.projectPath),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: c.projName,
				types.TagProjectPath: goprojpath,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftCfgToml,
			OutputFile: fmt.Sprintf("%s/%s.toml", c.projectPath, c.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: c.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:     CpftRunMain,
			BlockStrBegin: CpftRunmainBlock + "`",
			BlockStrEnd:   "`",
			OutputFile:    fmt.Sprintf("%s/%s.go", c.projectPath, c.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: c.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftMakefile,
			OutputFile: fmt.Sprintf("%s/Makefile", c.projectPath),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: c.projName,
				types.TagProjectPath: goprojpath,
				types.TagGoPath:      c.gopath,
				types.TagExecName:    c.execName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftTravisYml,
			OutputFile: fmt.Sprintf("%s/.travis.yml", c.projectPath),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: c.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftPluginToml,
			OutputFile: fmt.Sprintf("%s/plugin/plugin.toml", c.projectPath),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: c.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftCliMain,
			OutputFile: fmt.Sprintf("%s/cli/main.go", c.projectPath),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: c.projName,
				types.TagProjectPath: goprojpath,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftDappCommands,
			OutputFile: fmt.Sprintf("%s/plugin/dapp/%s/commands/cmd.go", c.projectPath, c.execName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: c.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftDappPlugin,
			OutputFile: fmt.Sprintf("%s/plugin/dapp/%s/plugin.go", c.projectPath, c.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: c.projName,
				types.TagExecNameFB:  c.execNameFB,
				types.TagProjectPath: goprojpath,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftDappExec,
			OutputFile: fmt.Sprintf("%s/plugin/dapp/%s/executor/%s.go", c.projectPath, c.projName, c.execName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: c.projName,
				types.TagExecName:    c.execName,
				types.TagClassName:   c.className,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:       CpftDappCreatepb,
			OutputFile:      fmt.Sprintf("%s/plugin/dapp/%s/proto/create_protobuf.sh", c.projectPath, c.projName),
			ReplaceKeyPairs: map[string]string{},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:       CpftDappMakefile,
			OutputFile:      fmt.Sprintf("%s/plugin/dapp/%s/proto/Makefile", c.projectPath, c.projName),
			ReplaceKeyPairs: map[string]string{},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftDappProto,
			OutputFile: fmt.Sprintf("%s/plugin/dapp/%s/proto/%s.proto", c.projectPath, c.projName, c.execName),
			ReplaceKeyPairs: map[string]string{
				types.TagActionName: c.classActionName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CpftDappTypefile,
			OutputFile: fmt.Sprintf("%s/plugin/dapp/%s/types/types.go", c.projectPath, c.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagExecNameFB:    c.execNameFB,
				types.TagExecName:      c.execName,
				types.TagClassTypeName: c.classTypeName,
				types.TagActionName:    c.classActionName,
			},
		},
		//       go
		&tasks.FormatDappSourceTask{
			OutputFolder: fmt.Sprintf("%s/%s/", c.outRootPath, c.projName),
		},
	)
	return taskSlice
}
