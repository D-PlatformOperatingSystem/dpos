// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types_test

import (
	"testing"

	"strings"

	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	_ "github.com/D-PlatformOperatingSystem/dpos/system"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/stretchr/testify/assert"
)

//how to create transafer for para
func TestCallCreateTxPara(t *testing.T) {
	str := types.ReadFile("testdata/guodun2.toml")
	new := strings.Replace(str, "Title=\"user.p.guodun2.\"", "Title=\"user.p.sto.\"", 1)
	cfg := types.NewDplatformOSConfig(new)

	req := &types.CreateTx{
		To:          "184wj4nsgVxKyz2NhM3Yb5RK5Ap6AFRFq2",
		Amount:      10,
		Fee:         1,
		Note:        []byte("12312"),
		IsWithdraw:  false,
		IsToken:     false,
		TokenSymbol: "",
		ExecName:    cfg.ExecName("coins"),
	}
	assert.True(t, cfg.IsPara())
	tx, err := types.CallCreateTransaction("coins", "", req)
	assert.Nil(t, err)
	tx, err = types.FormatTx(cfg, "coins", tx)
	assert.Nil(t, err)
	assert.Equal(t, "coins", string(tx.Execer))
	assert.Equal(t, address.ExecAddress("coins"), tx.To)
	tx, err = types.FormatTx(cfg, cfg.ExecName("coins"), tx)
	assert.Nil(t, err)
	assert.Equal(t, "user.p.sto.coins", string(tx.Execer))
	assert.Equal(t, address.ExecAddress("user.p.sto.coins"), tx.To)
}

func TestExecName(t *testing.T) {
	cfg := types.NewDplatformOSConfig(types.GetDefaultCfgstring())
	assert.Equal(t, cfg.ExecName("coins"), "coins")
	cfg.SetTitleOnlyForTest("user.p.sto.")
	assert.Equal(t, cfg.ExecName("coins"), "user.p.sto.coins")
	//# exec      #
	assert.Equal(t, cfg.ExecName("#coins"), "coins")
}
