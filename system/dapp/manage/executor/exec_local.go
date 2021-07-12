// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	pty "github.com/D-PlatformOperatingSystem/dpos/system/dapp/manage/types"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// ExecLocal_Modify defines execlocal modify func
func (c *Manage) ExecLocal_Modify(transfer *types.ModifyConfig, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == pty.ManageActionModifyConfig {
			var receipt types.ReceiptConfig
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //     ，
			}
			key := receipt.Current.Key
			set.KV = append(set.KV, &types.KeyValue{Key: localKey(key), Value: types.Encode(receipt.Current)})
			clog.Debug("ExecLocal to savelogs", "config ", key, "value", receipt.Current)
		}
	}
	return set, nil
}
