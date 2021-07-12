// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"os"
	"testing"
	"time"

	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	_ "github.com/D-PlatformOperatingSystem/dpos/system"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportBlockProc(t *testing.T) {
	mockDOM := testnode.New("", nil)
	blockchain := mockDOM.GetBlockChain()
	chainlog.Debug("TestExportBlockProc begin --------------------")

	//  1030
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 1030
	var err error
	cfg := mockDOM.GetClient().GetConfig()
	for {
		_, err = addMainTx(cfg, mockDOM.GetGenesisKey(), mockDOM.GetAPI())
		require.NoError(t, err)

		curheight = blockchain.GetBlockHeight()
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}

	//  block  ,startheight 1
	title := mockDOM.GetCfg().Title
	driver := mockDOM.GetCfg().BlockChain.Driver
	dbPath := ""

	//
	err = blockchain.ExportBlock(title, dbPath, 10000)
	assert.Equal(t, err, types.ErrInvalidParam)

	err = blockchain.ExportBlock(title, dbPath, 0)
	require.NoError(t, err)
	//
	db := dbm.NewDB(title, driver, dbPath, 4)
	headertitle, err := db.Get([]byte("F:header:"))
	require.NoError(t, err)

	var fileHeader types.FileHeader
	err = types.Decode(headertitle, &fileHeader)
	require.NoError(t, err)
	assert.Equal(t, driver, fileHeader.Driver)
	assert.Equal(t, int64(0), fileHeader.GetStartHeight())
	assert.Equal(t, title, fileHeader.GetTitle())
	db.Close()
	mockDOM.Close()

	testImportBlockProc(t)

	//
	err = blockchain.ExportBlock(title, dbPath, -1)
	assert.Equal(t, err, types.ErrInvalidParam)

	err = blockchain.ExportBlock("test", dbPath, 0)
	assert.Equal(t, err, types.ErrInvalidParam)

	chainlog.Debug("TestExportBlock end --------------------")
}

func testImportBlockProc(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer mockDOM.Close()
	blockchain := mockDOM.GetBlockChain()
	chainlog.Debug("TestImportBlockProc begin --------------------")
	title := mockDOM.GetCfg().Title
	dbPath := ""
	driver := mockDOM.GetCfg().BlockChain.Driver

	//
	db := dbm.NewDB(title, driver, dbPath, 4)
	var endBlock types.EndBlock
	endHeight, err := db.Get([]byte("F:endblock:"))
	require.NoError(t, err)
	err = types.Decode(endHeight, &endBlock)
	require.NoError(t, err)
	db.Close()

	err = blockchain.ImportBlock(title, dbPath)
	require.NoError(t, err)
	curHeader, err := blockchain.ProcGetLastHeaderMsg()
	require.NoError(t, err)
	assert.Equal(t, curHeader.Height, endBlock.GetHeight())
	assert.Equal(t, curHeader.GetHash(), endBlock.GetHash())
	file := title + ".db"
	os.RemoveAll(file)

	chainlog.Debug("TestImportBlockProc end --------------------")
}
