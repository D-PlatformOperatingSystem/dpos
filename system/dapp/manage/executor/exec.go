// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	"github.com/D-PlatformOperatingSystem/dpos/system/dapp"
	mty "github.com/D-PlatformOperatingSystem/dpos/system/dapp/manage/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

func (c *Manage) checkAddress(addr string) error {
	if dapp.IsDriverAddress(addr, c.GetHeight()) {
		return nil
	}
	return address.CheckAddress(addr)
}

func (c *Manage) checkTxToAddress(tx *types.Transaction, index int) error {
	return c.checkAddress(tx.GetRealToAddr())
}

// Exec_Modify modify exec
func (c *Manage) Exec_Modify(manageAction *types.ModifyConfig, tx *types.Transaction, index int) (*types.Receipt, error) {
	clog.Info("manage.Exec", "start index", index)
	//         To
	types.AssertConfig(c.GetAPI())
	cfg := c.GetAPI().GetConfig()
	if cfg.IsDappFork(c.GetHeight(), mty.ManageX, "ForkManageExec") {
		if err := c.checkTxToAddress(tx, index); err != nil {
			return nil, err
		}
	}
	action := NewAction(c, tx)
	return action.modifyConfig(manageAction)

}
