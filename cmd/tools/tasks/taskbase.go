// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"

var (
	mlog = log15.New("module", "task")
)

//TaskBase task base
type TaskBase struct {
	NextTask Task
}

//Execute
func (t *TaskBase) Execute() error {
	return nil
}

//SetNext set next
func (t *TaskBase) SetNext(task Task) {
	t.NextTask = task
}

//Next   next
func (t *TaskBase) Next() Task {
	return t.NextTask
}
