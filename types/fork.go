// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"strings"
)

/*
MaxHeight   forks        ，      ，     fork，           fork   ，    fork    MaxHeight
              ，           fork
*/
const MaxHeight = 10000000000000000

//Forks fork
type Forks struct {
	forks map[string]int64
}

func checkKey(key string) {
	if strings.Contains(key, ".") {
		panic("name must not have dot")
	}
}

//SetFork   fork
func (f *Forks) SetFork(key string, height int64) {
	checkKey(key)
	f.setFork(key, height)
}

//ReplaceFork   fork
func (f *Forks) ReplaceFork(key string, height int64) {
	checkKey(key)
	f.replaceFork(key, height)
}

//SetDappFork   dapp fork
func (f *Forks) SetDappFork(dapp, key string, height int64) {
	checkKey(key)
	f.setFork(dapp+"."+key, height)
}

//ReplaceDappFork   dapp fork
func (f *Forks) ReplaceDappFork(dapp, key string, height int64) {
	checkKey(key)
	f.replaceFork(dapp+"."+key, height)
}

func (f *Forks) replaceFork(key string, height int64) {
	if f.forks == nil {
		f.forks = make(map[string]int64)
	}
	if _, ok := f.forks[key]; !ok {
		panic("replace a not exist key " + " " + key)
	}
	f.forks[key] = height
}

func (f *Forks) setFork(key string, height int64) {
	if f.forks == nil {
		f.forks = make(map[string]int64)
	}
	f.forks[key] = height
}

// GetFork      ，  fork   0
func (f *Forks) GetFork(key string) int64 {
	height, ok := f.forks[key]
	if !ok {
		tlog.Error("get fork key not exisit -> " + key)
		return MaxHeight
	}
	return height
}

// HasFork fork
func (f *Forks) HasFork(key string) bool {
	_, ok := f.forks[key]
	return ok
}

// GetDappFork   dapp fork
func (f *Forks) GetDappFork(app string, key string) int64 {
	return f.GetFork(app + "." + key)
}

// SetAllFork     fork
func (f *Forks) SetAllFork(height int64) {
	for k := range f.forks {
		f.forks[k] = height
	}
}

// GetAll     fork
func (f *Forks) GetAll() map[string]int64 {
	return f.forks
}

// IsFork   fork
func (f *Forks) IsFork(height int64, fork string) bool {
	ifork := f.GetFork(fork)
	if height == -1 || height >= ifork {
		return true
	}
	return false
}

// IsDappFork   dapp fork
func (f *Forks) IsDappFork(height int64, dapp, fork string) bool {
	return f.IsFork(height, dapp+"."+fork)
}

// SetTestNetFork dom test net fork
func (f *Forks) SetTestNetFork() {
	f.SetFork("ForkChainParamV1", 0)
	f.SetFork("ForkChainParamV2", 0)
	f.SetFork("ForkMemPoolParamV1", 0)
	f.SetFork("ForkCheckTxDup", 0)
	f.SetFork("ForkBlockHash", 0)
	f.SetFork("ForkMinerTime", 0)
	f.SetFork("ForkTransferExec", 0)
	f.SetFork("ForkExecKey", 0)
	f.SetFork("ForkWithdraw", 0)
	f.SetFork("ForkTxGroup", 0)
	f.SetFork("ForkResetTx0", 0)
	f.SetFork("ForkExecRollback", 0)
	f.SetFork("ForkTxHeight", 0)
	f.SetFork("ForkCheckBlockTime", 0)
	f.SetFork("ForkMultiSignAddress", 0)
	f.SetFork("ForkStateDBSet", 0)
	f.SetFork("ForkBlockCheck", 0)
	f.SetFork("ForkLocalDBAccess", 0)
	f.SetFork("ForkTxGroupPara", 0)
	f.SetFork("ForkBase58AddressCheck", 0)
	//  fork      ，    user.p.x.exec driver，        0  ，
	f.SetFork("ForkEnableParaRegExec", 0)
	f.SetFork("ForkCacheDriver", 0)
	f.SetFork("ForkTicketFundAddrV1", 0)
	f.SetFork("ForkRootHash", 0)

}

func (f *Forks) setLocalFork() {
	f.SetAllFork(0)
	f.ReplaceFork("ForkBlockHash", 1)
	f.ReplaceFork("ForkRootHash", 1)
}

//paraName not used currently
func (f *Forks) setForkForParaZero() {
	f.SetAllFork(0)
	f.ReplaceFork("ForkBlockHash", 1)
	f.ReplaceFork("ForkRootHash", 1)
}

// IsFork      fork
func (c *DplatformOSConfig) IsFork(height int64, fork string) bool {
	return c.forks.IsFork(height, fork)
}

// IsDappFork   dapp fork
func (c *DplatformOSConfig) IsDappFork(height int64, dapp, fork string) bool {
	return c.forks.IsDappFork(height, dapp, fork)
}

// GetDappFork   dapp fork
func (c *DplatformOSConfig) GetDappFork(dapp, fork string) int64 {
	return c.forks.GetDappFork(dapp, fork)
}

// SetDappFork   dapp fork
func (c *DplatformOSConfig) SetDappFork(dapp, fork string, height int64) {
	if c.needSetForkZero() {
		height = 0
		if fork == "ForkBlockHash" {
			height = 1
		}
	}
	c.forks.SetDappFork(dapp, fork, height)
}

// RegisterDappFork   dapp fork
func (c *DplatformOSConfig) RegisterDappFork(dapp, fork string, height int64) {
	if c.needSetForkZero() {
		height = 0
		if fork == "ForkBlockHash" {
			height = 1
		}
	}
	c.forks.SetDappFork(dapp, fork, height)
}

// GetFork     fork
func (c *DplatformOSConfig) GetFork(fork string) int64 {
	return c.forks.GetFork(fork)
}

// HasFork      fork
func (c *DplatformOSConfig) HasFork(fork string) bool {
	return c.forks.HasFork(fork)
}

// IsEnableFork      fork
func (c *DplatformOSConfig) IsEnableFork(height int64, fork string, enable bool) bool {
	if !enable {
		return false
	}
	return c.IsFork(height, fork)
}

//fork     ：
//   fork         ，   fork     -1; forks   toml
func (c *DplatformOSConfig) initForkConfig(forks *ForkList) {
	dplatformosfork := c.forks.GetAll()
	if dplatformosfork == nil {
		panic("dplatformos fork not init")
	}
	//    dplatformosfork  system
	s := ""
	for k := range dplatformosfork {
		if !strings.Contains(k, ".") {
			if _, ok := forks.System[k]; !ok {
				s += "system fork " + k + " not config\n"
			}
		}
	}
	for k := range dplatformosfork {
		forkname := strings.Split(k, ".")
		if len(forkname) == 1 {
			continue
		}
		if len(forkname) > 2 {
			panic("fork name has too many dot")
		}
		exec := forkname[0]
		name := forkname[1]
		if forks.Sub != nil {
			//       ，    enable
			if _, ok := forks.Sub[exec]; !ok {
				continue
			}
			if _, ok := forks.Sub[exec][name]; ok {
				continue
			}
		}
		s += "exec " + exec + " name " + name + " not config\n"
	}
	//         ，
	for k, v := range forks.System {
		if v == -1 {
			v = MaxHeight
		}
		if !c.HasFork(k) {
			s += "system fork not exist : " + k + "\n"
		}
		//   toml         fork             fork
		c.forks.SetFork(k, v)
	}
	//  allow exec    ，
	AllowUserExec = [][]byte{ExecerNone}
	for dapp, forklist := range forks.Sub {
		AllowUserExec = append(AllowUserExec, []byte(dapp))
		for k, v := range forklist {
			if v == -1 {
				v = MaxHeight
			}
			if !c.HasFork(dapp + "." + k) {
				s += "exec fork not exist : exec = " + dapp + " key = " + k + "\n"
			}
			//   toml         fork             fork
			c.forks.SetDappFork(dapp, k, v)
		}
	}
	if c.enableCheckFork {
		if len(s) > 0 {
			panic(s)
		}
	}
}
