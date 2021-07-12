// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"
	"time"

	"github.com/D-PlatformOperatingSystem/dpos/common/address"
	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/common/version"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
	"github.com/D-PlatformOperatingSystem/dpos/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestNeedReExec(t *testing.T) {
	defer func() {
		r := recover()
		if r == "not support upgrade store to greater than 2.0.0" {
			assert.Equal(t, r, "not support upgrade store to greater than 2.0.0")
		} else if r == "not support degrade the program" {
			assert.Equal(t, r, "not support degrade the program")
		}
	}()
	cfg := types.NewDplatformOSConfig(types.GetDefaultCfgstring())
	mockDOM := testnode.NewWithConfig(cfg, nil)

	//
	chain := mockDOM.GetBlockChain()
	//
	testcase := []*types.UpgradeMeta{
		{Starting: true},
		{Starting: false, Version: "1.0.0"},
		{Starting: false, Version: "1.0.0"},
		{Starting: false, Version: "2.0.0"},
		{Starting: false, Version: "2.0.0"},
	}
	//
	vsCase := []string{
		"1.0.0",
		"1.0.0",
		"2.0.0",
		"3.0.0",
		"1.0.0",
	}
	result := []bool{
		true,
		false,
		true,
		false,
		false,
	}
	for i, c := range testcase {
		version.SetStoreDBVersion(vsCase[i])
		res := chain.NeedReExec(c)
		assert.Equal(t, res, result[i])
	}
}

func GetAddrTxsCount(db dbm.DB, addr string) (int64, error) {
	count := types.Int64{}
	TxsCount, err := db.Get(types.CalcAddrTxsCountKey(addr))
	if err != nil && err != types.ErrNotFound {
		return 0, err
	}
	if len(TxsCount) == 0 {
		return 0, nil
	}
	err = types.Decode(TxsCount, &count)
	if err != nil {
		return 0, err
	}
	return count.Data, nil
}

func TestUpgradeStore(t *testing.T) {
	cfg := types.NewDplatformOSConfig(types.GetDefaultCfgstring())
	cfg.GetModuleConfig().BlockChain.EnableReExecLocal = true
	mockDOM := testnode.NewWithConfig(cfg, nil)
	//
	chain := mockDOM.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), kvCount)
	defer mockDOM.Close()
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
	addr := address.PubKeyToAddress(mockDOM.GetGenesisKey().PubKey().Bytes()).String()
	count1, err := GetAddrTxsCount(db, addr)
	assert.NoError(t, err)
	version.SetStoreDBVersion("2.0.0")
	chain.UpgradeStore()
	count2, err := GetAddrTxsCount(db, addr)
	assert.NoError(t, err)
	assert.Equal(t, count1, count2)
}
