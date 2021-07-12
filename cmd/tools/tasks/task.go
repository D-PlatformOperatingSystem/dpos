// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tasks
package tasks

// Task
type Task interface {
	GetName() string
	Next() Task
	SetNext(t Task)

	Execute() error
}
