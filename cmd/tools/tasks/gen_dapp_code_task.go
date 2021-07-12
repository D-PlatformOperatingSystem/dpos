// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/gencode"
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/types"
	util2 "github.com/D-PlatformOperatingSystem/dpos/util"

	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/util"
)

// GenDappCodeTask       pb.go        ，
type GenDappCodeTask struct {
	TaskBase
	DappName     string
	DappDir      string
	ProtoFile    string
	PackagePath  string
	replacePairs map[string]string
}

//GetName   name
func (c *GenDappCodeTask) GetName() string {
	return "GenDappCodeTask"
}

//Execute
func (c *GenDappCodeTask) Execute() error {
	mlog.Info("Execute generate dapp code task.")

	c.replacePairs = make(map[string]string)
	pbContext, err := util.ReadFile(c.ProtoFile)
	if err != nil {
		mlog.Error("ReadProtoFile", "Err", err.Error(), "proto", c.ProtoFile)
		return fmt.Errorf("ReadProtoFileErr:%s", err.Error())
	}

	pbContent := string(pbContext)

	if err = c.calcReplacePairs(pbContent); err != nil {
		mlog.Error("CalcReplacePairs", "Err", err.Error())
		return fmt.Errorf("CalcReplacePairsErr:%s", err.Error())
	}

	if err = c.genDappCode(); err != nil {
		return fmt.Errorf("GenDappCodeErr:%s", err.Error())
	}

	return err
}

func (c *GenDappCodeTask) calcReplacePairs(pbContent string) error {

	dapp := strings.ToLower(c.DappName)
	className, _ := util2.MakeStringToUpper(dapp, 0, 1)
	c.replacePairs[types.TagExecName] = dapp
	c.replacePairs[types.TagExecObject] = dapp[:1]
	c.replacePairs[types.TagClassName] = className
	c.replacePairs[types.TagImportPath] = c.PackagePath

	pbAppend := gencode.ProtoFileAppendService
	if strings.Contains(pbContent, "service") {
		pbAppend = ""
	}

	c.replacePairs[types.TagProtoFileContent] = pbContent
	c.replacePairs[types.TagProtoFileAppend] = pbAppend

	actionName := className + "Action"
	actionInfos, err := readDappActionFromProto(pbContent, actionName)

	if err != nil {
		return fmt.Errorf("ReadProtoActionErr:%s", err.Error())
	}

	//exec
	c.replacePairs[types.TagExecFileContent] = formatExecContent(actionInfos, dapp)
	c.replacePairs[types.TagExecLocalFileContent] = formatExecLocalContent(actionInfos, dapp)
	c.replacePairs[types.TagExecDelLocalFileContent] = formatExecDelLocalContent(actionInfos, dapp)

	//types
	c.replacePairs[types.TagTyLogActionType] = buildActionLogTypeText(actionInfos, className)
	c.replacePairs[types.TagActionIDText] = buildActionIDText(actionInfos, className)
	c.replacePairs[types.TagLogMapText] = buildLogMapText()
	c.replacePairs[types.TagTypeMapText] = buildTypeMapText(actionInfos, className)

	return nil

}

func (c *GenDappCodeTask) genDappCode() error {

	codeTypes := gencode.GetCodeFilesWithType("dapp")

	for _, code := range codeTypes {

		dirName := code.GetDirName()
		for _, tag := range code.GetDirReplaceTags() {
			dirName = strings.Replace(dirName, tag, c.replacePairs[tag], -1)
		}
		dirPath := filepath.Join(c.DappDir, dirName)
		err := os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			mlog.Error("MakeCodeDir", "Err", err.Error(), "DirPath", dirPath)
			return err
		}
		files := code.GetFiles()
		tags := code.GetFileReplaceTags()

		for name, content := range files {

			for _, tag := range tags {
				name = strings.Replace(name, tag, c.replacePairs[tag], -1)
				content = strings.Replace(content, tag, c.replacePairs[tag], -1)
			}

			_, err = util.WriteStringToFile(filepath.Join(dirPath, name), content)

			if err != nil {
				mlog.Error("GenNewCodeFile", "Err", err.Error(), "CodeFile", filepath.Join(dirPath, name))
				return err
			}
		}

	}

	return nil
}
