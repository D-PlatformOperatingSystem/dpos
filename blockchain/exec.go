// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
)

//
func execBlock(client queue.Client, prevStateRoot []byte, block *types.Block, errReturn bool, sync bool) (*types.BlockDetail, []*types.Transaction, error) {
	return util.ExecBlock(client, prevStateRoot, block, errReturn, sync, true)
}

//
func execBlockUpgrade(client queue.Client, prevStateRoot []byte, block *types.Block, sync bool) error {
	return util.ExecBlockUpgrade(client, prevStateRoot, block, sync)
}
