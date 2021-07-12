// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"
	"time"

	"fmt"

	"strings"
	"sync"

	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
	"github.com/D-PlatformOperatingSystem/dpos/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var once sync.Once

func TestRollbackblock(t *testing.T) {
	once.Do(func() {
		types.RegFork("store-kvmvccmavl", func(cfg *types.DplatformOSConfig) {
			cfg.RegisterDappFork("store-kvmvccmavl", "ForkKvmvccmavl", 20*10000)
		})
	})
	str := types.GetDefaultCfgstring()
	new := strings.Replace(str, "Title=\"local\"", "Title=\"dplatformos\"", 1)
	cfg := types.NewDplatformOSConfig(new)
	mfg := cfg.GetModuleConfig()
	mfg.BlockChain.RollbackBlock = 0
	mockDOM := testnode.NewWithConfig(cfg, nil)
	chain := mockDOM.GetBlockChain()
	chain.Rollbackblock()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), kvCount)
	defer mockDOM.Close()

	//
	testMockSendTx(t, mockDOM)

	chain.Rollbackblock()
}

func TestNeedRollback(t *testing.T) {
	once.Do(func() {
		types.RegFork("store-kvmvccmavl", func(cfg *types.DplatformOSConfig) {
			cfg.RegisterDappFork("store-kvmvccmavl", "ForkKvmvccmavl", 20*10000)
		})
	})

	str := types.GetDefaultCfgstring()
	new := strings.Replace(str, "Title=\"local\"", "Title=\"dplatformos\"", 1)
	cfg := types.NewDplatformOSConfig(new)
	mockDOM := testnode.NewWithConfig(cfg, nil)
	chain := mockDOM.GetBlockChain()

	curHeight := int64(5)
	rollHeight := int64(5)
	ok := chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, false, ok)

	curHeight = int64(5)
	rollHeight = int64(6)
	ok = chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, false, ok)

	curHeight = int64(10 * 10000)
	rollHeight = int64(5 * 10000)
	ok = chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, true, ok)

	curHeight = int64(20*10000 + 1)
	rollHeight = int64(20 * 10000)
	ok = chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, true, ok)

	curHeight = int64(22 * 10000)
	rollHeight = int64(20 * 10000)
	ok = chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, false, ok)

	curHeight = int64(22 * 10000)
	rollHeight = int64(20*10000 + 1)
	ok = chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, true, ok)

}

func TestRollback(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mfg := cfg.GetModuleConfig()
	mfg.BlockChain.RollbackBlock = 2
	mockDOM := testnode.NewWithConfig(cfg, nil)
	chain := mockDOM.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), kvCount)
	defer mockDOM.Close()

	//
	testMockSendTx(t, mockDOM)

	chain.Rollback()
	require.Equal(t, int64(2), chain.GetBlockHeight())
}

func TestRollbackSave(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mfg := cfg.GetModuleConfig()
	mfg.BlockChain.RollbackBlock = 2
	mfg.BlockChain.RollbackSave = true
	mockDOM := testnode.NewWithConfig(cfg, nil)
	chain := mockDOM.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), kvCount)
	defer mockDOM.Close()

	//
	testMockSendTx(t, mockDOM)

	height := chain.GetBlockHeight()
	chain.Rollback()

	// check
	require.Equal(t, int64(2), chain.GetBlockHeight())
	for i := height; i > 2; i-- {
		key := []byte(fmt.Sprintf("TB:%012d", i))
		_, err := chain.GetDB().Get(key)
		assert.NoError(t, err)
	}
	value, err := chain.GetDB().Get([]byte("LTB:"))
	assert.NoError(t, err)
	assert.NotNil(t, value)
	h := &types.Int64{}
	err = types.Decode(value, h)
	assert.NoError(t, err)
	assert.Equal(t, height, h.Data)
}

func TestRollbackPara(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mfg := cfg.GetModuleConfig()
	mfg.BlockChain.RollbackBlock = 2
	mfg.BlockChain.IsParaChain = true
	mockDOM := testnode.NewWithConfig(cfg, nil)
	chain := mockDOM.GetBlockChain()
	defer mockDOM.Close()

	//
	testMockSendTx(t, mockDOM)

	chain.Rollback()
	require.Equal(t, int64(2), chain.GetBlockHeight())
}

func testMockSendTx(t *testing.T, mockDOM *testnode.DplatformOSMock) {
	cfg := mockDOM.GetClient().GetConfig()
	txs := util.GenCoinsTxs(cfg, mockDOM.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mockDOM.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mockDOM.WaitHeight(1)
	txs = util.GenCoinsTxs(cfg, mockDOM.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mockDOM.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mockDOM.WaitHeight(2)
	txs = util.GenNoneTxs(cfg, mockDOM.GetGenesisKey(), 1)
	for i := 0; i < len(txs); i++ {
		reply, err := mockDOM.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mockDOM.WaitHeight(3)
	txs = util.GenNoneTxs(cfg, mockDOM.GetGenesisKey(), 2)
	for i := 0; i < len(txs); i++ {
		reply, err := mockDOM.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mockDOM.WaitHeight(4)
	time.Sleep(time.Second)
}
