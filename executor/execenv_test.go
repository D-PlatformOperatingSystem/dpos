// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"
	"time"

	drivers "github.com/D-PlatformOperatingSystem/dpos/system/dapp"

	"strings"

	_ "github.com/D-PlatformOperatingSystem/dpos/system"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
	"github.com/stretchr/testify/assert"
)

func TestLoadDriverFork(t *testing.T) {
	str := types.GetDefaultCfgstring()
	new := strings.Replace(str, "Title=\"local\"", "Title=\"dplatformos\"", 1)
	exec, _ := initEnv(new)
	cfg := exec.client.GetConfig()
	execInit(cfg)
	drivers.Register(cfg, "notAllow", newAllowApp, 0)
	var txs []*types.Transaction
	addr, _ := util.Genaddress()
	genkey := util.TestPrivkeyList[0]
	tx := util.CreateCoinsTx(cfg, genkey, addr, types.Coin)
	txs = append(txs, tx)
	tx.Execer = []byte("notAllow")
	tx1 := *tx
	tx1.Execer = []byte("user.p.para.notAllow")
	tx2 := *tx
	tx2.Execer = []byte("user.p.test.notAllow")
	// local fork   0,     fork
	//types.SetTitleOnlyForTest("dplatformos")
	t.Log("get fork value", cfg.GetFork("ForkCacheDriver"), cfg.GetTitle())
	cases := []struct {
		height     int64
		driverName string
		tx         *types.Transaction
		index      int
	}{
		{cfg.GetFork("ForkCacheDriver") - 1, "notAllow", tx, 0},
		//           none
		{cfg.GetFork("ForkCacheDriver") - 1, "none", &tx1, 0},
		// index > 0 allow,     fork  ，      allow,           ， none
		{cfg.GetFork("ForkCacheDriver") - 1, "none", &tx1, 1},
		//     allow     none
		{cfg.GetFork("ForkCacheDriver"), "none", &tx2, 0},
		// fork        , index>0   allow
		{cfg.GetFork("ForkCacheDriver"), "notAllow", &tx2, 1},
	}
	ctx := &executorCtx{
		stateHash:  nil,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: nil,
	}
	execute := newExecutor(ctx, exec, nil, txs, nil)
	for idx, c := range cases {
		if execute.height != c.height {
			execute.driverCache = make(map[string]drivers.Driver)
			execute.height = c.height
		}
		driver := execute.loadDriver(c.tx, c.index)
		assert.Equal(t, c.driverName, driver.GetDriverName(), "case index=%d", idx)
	}
}

type notAllowApp struct {
	*drivers.DriverBase
}

func newAllowApp() drivers.Driver {
	app := &notAllowApp{DriverBase: &drivers.DriverBase{}}
	app.SetChild(app)
	return app
}

func (app *notAllowApp) GetDriverName() string {
	return "notAllow"
}

func (app *notAllowApp) Allow(tx *types.Transaction, index int) error {

	if app.GetHeight() == 0 {
		return types.ErrActionNotSupport
	}
	//              action  allow  ，  0
	if index > 0 || string(tx.Execer) == "notAllow" {
		return nil
	}
	return types.ErrActionNotSupport
}
