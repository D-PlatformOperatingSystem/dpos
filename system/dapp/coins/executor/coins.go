// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package executor coins
package executor

/*
coins       exec。        。

        ：
EventTransfer ->
*/

// package none execer for unknow execer
// all none transaction exec ok, execept nofee
// nofee transaction will not pack into block

import (
	drivers "github.com/D-PlatformOperatingSystem/dpos/system/dapp"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// var clog = log.New("module", "execs.coins")
var driverName = "coins"

// Init defines a register function
func Init(name string, cfg *types.DplatformOSConfig, sub []byte) {
	if name != driverName {
		panic("system dapp can't be rename")
	}
	//     RegisterDappFork   Register dapp
	drivers.Register(cfg, driverName, newCoins, cfg.GetDappFork(driverName, "Enable"))
	InitExecType()
}

//InitExecType the initialization process is relatively heavyweight, lots of reflect, so it's global
func InitExecType() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Coins{}))
}

// GetName return name string
func GetName() string {
	return newCoins().GetName()
}

// Coins defines coins
type Coins struct {
	drivers.DriverBase
}

func newCoins() drivers.Driver {
	c := &Coins{}
	c.SetChild(c)
	c.SetExecutorType(types.LoadExecutorType(driverName))
	return c
}

// GetDriverName get drive name
func (c *Coins) GetDriverName() string {
	return driverName
}

// CheckTx check transaction amount
func (c *Coins) CheckTx(tx *types.Transaction, index int) error {
	ety := c.GetExecutorType()
	amount, err := ety.Amount(tx)
	if err != nil {
		return err
	}
	if amount < 0 {
		return types.ErrAmount
	}
	return nil
}

// IsFriend coins contract  the mining transaction that runs the ticket contract
func (c *Coins) IsFriend(myexec, writekey []byte, othertx *types.Transaction) bool {
	//step1
	if !c.AllowIsSame(myexec) {
		return false
	}
	//step2    othertx        (     ，        )
	types.AssertConfig(c.GetAPI())
	types := c.GetAPI().GetConfig()
	if othertx.ActionName() == "miner" {
		for _, exec := range types.GetMinerExecs() {
			if types.ExecName(exec) == string(othertx.Execer) {
				return true
			}
		}
	}

	return false
}

// CheckReceiptExecOk return true to check if receipt ty is ok
func (c *Coins) CheckReceiptExecOk() bool {
	return true
}
