// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"

	"github.com/D-PlatformOperatingSystem/dpos/blockchain"
	"github.com/D-PlatformOperatingSystem/dpos/util"
	"github.com/D-PlatformOperatingSystem/dpos/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestTryReduceLocalDB(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mockDOM := testnode.NewWithConfig(cfg, nil)
	//
	chain := mockDOM.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), kvCount)
	defer mockDOM.Close()

	blockchain.ReduceHeight = 0
	defer func() {
		blockchain.ReduceHeight = 10000
	}()

	var flagHeight int64
	count := 10
	for i := 0; i < 2; i++ {
		txs := util.GenCoinsTxs(cfg, mockDOM.GetGenesisKey(), int64(count))
		for j := 0; j < len(txs); j++ {
			reply, err := mockDOM.GetAPI().SendTx(txs[j])
			assert.Nil(t, err)
			assert.Equal(t, reply.IsOk, true)
			waitH := i*count + (j + 1)
			mockDOM.WaitHeight(int64(waitH))
		}
		flagHeight = chain.TryReduceLocalDB(flagHeight, int64(count))
		assert.Equal(t, flagHeight, int64((i+1)*count+1))
	}
}
