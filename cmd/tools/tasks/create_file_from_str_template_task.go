/*
 * Copyright D-Platform Corp. 2018 All Rights Reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package tasks

import (
	"fmt"
	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/util"
)

// CreateFileFromStrTemplateTask                    
type CreateFileFromStrTemplateTask struct {
	TaskBase
	SourceStr       string
	OutputFile      string
	BlockStrBegin   string // Block  SourceStr     
	BlockStrEnd     string
	ReplaceKeyPairs map[string]string
	fileContent     string
}

//GetName   name
func (c *CreateFileFromStrTemplateTask) GetName() string {
	return "CreateFileFromStrTemplateTask"
}

//Execute   
func (c *CreateFileFromStrTemplateTask) Execute() error {
	if len(c.BlockStrBegin) > 0 && len(c.BlockStrEnd) > 0 {
		c.SourceStr = fmt.Sprintf("%s%s%s", c.BlockStrBegin, c.SourceStr, c.BlockStrEnd)
	}
	c.fileContent = c.SourceStr
	for key, value := range c.ReplaceKeyPairs {
		c.fileContent = strings.Replace(c.fileContent, key, value, -1)
	}
	if err := util.MakeDir(c.OutputFile); err != nil {
		return err
	}
	util.DeleteFile(c.OutputFile)
	len, err := util.WriteStringToFile(c.OutputFile, c.fileContent)
	if err == nil {
		mlog.Info("Create file success.", "file", c.OutputFile, "file len", len)
	} else {
		mlog.Info("Create file falied.", "error", err)
	}
	return err
}
