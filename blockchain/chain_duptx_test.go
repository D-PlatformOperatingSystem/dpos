// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"
	"time"

	_ "github.com/D-PlatformOperatingSystem/dpos/system"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
	"github.com/D-PlatformOperatingSystem/dpos/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//  10   ，10     TxHeight，  size128
func TestCheckDupTxHashList01(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer func() {
		defer mockDOM.Close()
	}()
	cfg := mockDOM.GetClient().GetConfig()
	cfg.S("TxHeight", true)
	chainlog.Debug("TestCheckDupTxHashList01 begin --------------------")

	blockchain := mockDOM.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10
	var txs []*types.Transaction
	for {
		txlist, _, err := addTx(cfg, mockDOM.GetGenesisKey(), mockDOM.GetAPI())
		require.NoError(t, err)
		txs = append(txs, txlist...)
		curheight := blockchain.GetBlockHeight()
		chainlog.Debug("testCheckDupTxHashList01", "curheight", curheight, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	time.Sleep(time.Second)
	//
	duptxhashlist, err := checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), len(txs))
	//
	txs = util.GenCoinsTxs(cfg, mockDOM.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(cfg, mockDOM.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)
	chainlog.Debug("TestCheckDupTxHashList01 end --------------------")
}

//  10   ，10    TxHeight，  size128
func TestCheckDupTxHashList02(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer func() {
		defer mockDOM.Close()
	}()
	cfg := mockDOM.GetClient().GetConfig()
	cfg.S("TxHeight", true)
	chainlog.Debug("TestCheckDupTxHashList02 begin --------------------")
	blockchain := mockDOM.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10
	var txs []*types.Transaction
	for {
		txlist, _, err := addTxTxHeigt(cfg, mockDOM.GetGenesisKey(), mockDOM.GetAPI(), curheight)
		txs = append(txs, txlist...)
		require.NoError(t, err)
		curheight := blockchain.GetBlockHeight()
		chainlog.Debug("testCheckDupTxHashList02", "curheight", curheight, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	time.Sleep(time.Second)
	//
	duptxhashlist, err := checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), len(txs))

	//
	txs = util.GenCoinsTxs(cfg, mockDOM.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(cfg, mockDOM.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	chainlog.Debug("TestCheckDupTxHashList02 end --------------------")
}

//  130   ，130     TxHeight，
func TestCheckDupTxHashList03(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer func() {
		defer mockDOM.Close()
	}()
	cfg := mockDOM.GetClient().GetConfig()
	cfg.S("TxHeight", true)
	chainlog.Debug("TestCheckDupTxHashList03 begin --------------------")
	blockchain := mockDOM.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 130
	var txs []*types.Transaction
	for {
		txlist, _, err := addTx(cfg, mockDOM.GetGenesisKey(), mockDOM.GetAPI())
		txs = append(txs, txlist...)
		require.NoError(t, err)
		curheight := blockchain.GetBlockHeight()
		chainlog.Debug("testCheckDupTxHashList03", "curheight", curheight, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	time.Sleep(time.Second)
	//    ,  TxHeight，cache     db
	duptxhashlist, err := checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), len(txs))

	txs = util.GenCoinsTxs(cfg, mockDOM.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(cfg, mockDOM.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)
	chainlog.Debug("TestCheckDupTxHashList03 end --------------------")
}

//  130   ，130    TxHeight，
func TestCheckDupTxHashList04(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer func() {
		defer mockDOM.Close()
	}()
	cfg := mockDOM.GetClient().GetConfig()
	cfg.S("TxHeight", true)
	chainlog.Debug("TestCheckDupTxHashList04 begin --------------------")
	blockchain := mockDOM.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 130
	curheightForExpire := curheight
	var txs []*types.Transaction
	for {
		txlist, _, err := addTxTxHeigt(cfg, mockDOM.GetGenesisKey(), mockDOM.GetAPI(), curheightForExpire)
		txs = append(txs, txlist...)
		require.NoError(t, err)
		curheightForExpire = blockchain.GetBlockHeight()
		chainlog.Debug("testCheckDupTxHashList04", "curheight", curheightForExpire, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheightForExpire)
		require.NoError(t, err)
		if curheightForExpire >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	time.Sleep(time.Second)
	duptxhashlist, err := checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), len(txs))

	//
	txs = util.GenCoinsTxs(cfg, mockDOM.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(cfg, mockDOM.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	chainlog.Debug("TestCheckDupTxHashList04 end --------------------")
}

//  ：  10   ，10    TxHeight，TxHeight      size128
func TestCheckDupTxHashList05(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer func() {
		defer mockDOM.Close()
	}()
	cfg := mockDOM.GetClient().GetConfig()
	cfg.S("TxHeight", true)
	chainlog.Debug("TestCheckDupTxHashList05 begin --------------------")
	TxHeightOffset = 60
	//   TxHeight   TxHeight
	for i := 1; i < 10; i++ {
		_, _, err := addTxTxHeigt(cfg, mockDOM.GetGenesisKey(), mockDOM.GetAPI(), int64(i))
		require.EqualError(t, err, "ErrTxExpire")
		time.Sleep(sendTxWait)
	}
	chainlog.Debug("TestCheckDupTxHashList05 end --------------------")
}
