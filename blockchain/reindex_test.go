// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"bytes"
	"testing"
	"time"

	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/common/version"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
	"github.com/D-PlatformOperatingSystem/dpos/util/testnode"
	"github.com/stretchr/testify/assert"
)

var (
	//  0   kv
	kvCount = 25
)

func TestReindex(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
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
	kvs1 := getAllKeys(db)
	version.SetLocalDBVersion("10000.0.0")
	chain.UpgradeChain()
	kvs2 := getAllKeys(db)
	assert.Equal(t, kvs1, kvs2)
}

func getAllKeys(db dbm.IteratorDB) (kvs []*types.KeyValue) {
	it := db.Iterator(nil, types.EmptyValue, false)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		key := copyBytes(it.Key())
		val := it.ValueCopy()
		//meta
		if string(key) == "LocalDBMeta" {
			continue
		}
		if bytes.HasPrefix(key, []byte("TotalFee")) {
			//println("--", string(key)[0:4], common.ToHex(key))
			totalFee := &types.TotalFee{}
			types.Decode(val, totalFee)
			//println("val", totalFee.String())
		}
		kvs = append(kvs, &types.KeyValue{Key: key, Value: val})
	}
	return kvs
}

func copyBytes(keys []byte) []byte {
	data := make([]byte, len(keys))
	copy(data, keys)
	return data
}

func TestUpgradeChain(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	cfg.GetModuleConfig().Store.LocalDBVersion = "0.0.0"
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

	version.SetLocalDBVersion("1.0.0")
	chain.UpgradeChain()
	ver, err := getUpgradeMeta(db)
	assert.Nil(t, err)
	assert.Equal(t, ver.GetVersion(), "1.0.0")

	version.SetLocalDBVersion("2.0.0")
	chain.UpgradeChain()

	ver, err = getUpgradeMeta(db)
	assert.Nil(t, err)
	assert.Equal(t, ver.GetVersion(), "2.0.0")
}

//GetUpgradeMeta   blockchain
func getUpgradeMeta(db dbm.DB) (*types.UpgradeMeta, error) {
	ver := types.UpgradeMeta{}
	version, err := db.Get(version.LocalDBMeta)
	if err != nil && err != dbm.ErrNotFoundInDb {
		return nil, err
	}
	if len(version) == 0 {
		return &types.UpgradeMeta{Version: "0.0.0"}, nil
	}
	err = types.Decode(version, &ver)
	if err != nil {
		return nil, err
	}
	return &ver, nil
}
