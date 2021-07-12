// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

//var
var (
	CliCmd        string                      //dplatformos cli
	CheckTimeout  int                         //  check
	autoTestItems = make(map[string]AutoTest) //     dapp
)

//AutoTest dapp  auto test   ，      dapp         ，
type AutoTest interface {
	GetName() string
	GetTestConfigType() reflect.Type
}

//Init                check
func Init(cliCmd string, checkTimeout int) {

	CliCmd = cliCmd
	CheckTimeout = checkTimeout
}

//RegisterAutoTest
func RegisterAutoTest(at AutoTest) {

	if at == nil || len(at.GetName()) == 0 {
		return
	}
	dapp := at.GetName()

	if _, ok := autoTestItems[dapp]; ok {
		panic("Register Duplicate Dapp, name = " + dapp)
	}
	autoTestItems[dapp] = at
}

//GetAutoTestConfig
func GetAutoTestConfig(dapp string) reflect.Type {

	if len(dapp) == 0 {

		return nil
	}

	if config, ok := autoTestItems[dapp]; ok {

		return config.GetTestConfigType()
	}

	return nil
}
