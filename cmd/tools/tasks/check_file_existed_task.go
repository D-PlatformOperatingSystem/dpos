// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"github.com/D-PlatformOperatingSystem/dpos/cmd/tools/util"
)

// CheckFileExistedTask
type CheckFileExistedTask struct {
	TaskBase
	FileName string
}

//GetName   name
func (c *CheckFileExistedTask) GetName() string {
	return "CheckFileExistedTask"
}

//Execute
func (c *CheckFileExistedTask) Execute() error {
	mlog.Info("Execute file existed task.", "file", c.FileName)
	_, err := util.CheckFileExists(c.FileName)
	return err
}
