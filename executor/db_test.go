// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"

	"strings"

	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/queue"
	"github.com/D-PlatformOperatingSystem/dpos/store"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/stretchr/testify/assert"
)

func newStateDbForTest(height int64, cfg *types.DplatformOSConfig) dbm.KV {
	q := queue.New("channel")
	q.SetConfig(cfg)
	return NewStateDB(q.Client(), nil, nil, &StateDBOption{Height: height})
}
func TestStateDBGet(t *testing.T) {
	cfg := types.NewDplatformOSConfig(types.GetDefaultCfgstring())
	db := newStateDbForTest(0, cfg)
	testDBGet(t, db)
}

func testDBGet(t *testing.T, db dbm.KV) {
	err := db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(t, err)
	v, err := db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	err = db.Set([]byte("k1"), []byte("v11"))
	assert.Nil(t, err)
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v11"))

	stateDb := db.(*StateDB)
	vs, err := stateDb.BatchGet([][]byte{[]byte("k1")})
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{[]byte("v11")}, vs)
}

func TestStateDBTxGetOld(t *testing.T) {
	str := types.GetDefaultCfgstring()
	new := strings.Replace(str, "Title=\"local\"", "Title=\"dplatformos\"", 1)
	cfg := types.NewDplatformOSConfig(new)

	q := queue.New("channel")
	q.SetConfig(cfg)
	// store
	s := store.New(cfg)
	s.SetQueueClient(q.Client())
	// exec
	db := NewStateDB(q.Client(), nil, nil, &StateDBOption{Height: cfg.GetFork("ForkExecRollback") - 1})
	defer func() {
		s.Close()
		q.Close()
	}()

	db.Begin()
	err := db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(t, err)
	v, err := db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	db.Rollback()
	v, err = db.Get([]byte("k1"))
	assert.Equal(t, err, types.ErrNotFound)
	assert.Equal(t, v, []byte(nil))

	db.Begin()
	err = db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(t, err)
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	db.Commit()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	err = db.Set([]byte("k1"), []byte("v11"))
	assert.Nil(t, err)

	db.Begin()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	//fork    bug，
	assert.Equal(t, v, []byte("v1"))

	db.Begin()
	db.Rollback()
	db.Commit()
}

func testTxGet(t *testing.T, db dbm.KV) {
	//
	db.Begin()
	err := db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(t, err)
	v, err := db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	db.Commit()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	//  transaction set，  set  ，  rollback
	err = db.Set([]byte("k1"), []byte("v11"))
	assert.Nil(t, err)

	db.Begin()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v11"))

	err = db.Set([]byte("k1"), []byte("v12"))
	assert.Nil(t, err)
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v12"))

	db.Rollback()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v11"))
}

func TestStateDBTxGet(t *testing.T) {
	cfg := types.NewDplatformOSConfig(types.GetDefaultCfgstring())
	db := newStateDbForTest(cfg.GetFork("ForkExecRollback"), cfg)
	testTxGet(t, db)
}
