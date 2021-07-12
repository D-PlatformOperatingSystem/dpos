package executor_test

import (
	"testing"

	dbm "github.com/D-PlatformOperatingSystem/dpos/common/db"
	"github.com/D-PlatformOperatingSystem/dpos/executor"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestLocalDBGet(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer mockDOM.Close()
	db := executor.NewLocalDB(mockDOM.GetClient(), false)
	defer db.(*executor.LocalDB).Close()
	testDBGet(t, db)
}

func TestLocalDBGetReadOnly(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer mockDOM.Close()
	db := executor.NewLocalDB(mockDOM.GetClient(), true)
	defer db.(*executor.LocalDB).Close()
	testDBGet(t, db)
}

func TestLocalDBEnable(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer mockDOM.Close()
	db := executor.NewLocalDB(mockDOM.GetClient(), false)
	ldb := db.(*executor.LocalDB)
	defer ldb.Close()
	_, err := ldb.Get([]byte("hello"))
	assert.Equal(t, err, types.ErrNotFound)
	ldb.DisableRead()
	_, err = ldb.Get([]byte("hello"))

	assert.Equal(t, err, types.ErrDisableRead)
	_, err = ldb.List(nil, nil, 0, 0)
	assert.Equal(t, err, types.ErrDisableRead)
	ldb.EnableRead()
	_, err = ldb.Get([]byte("hello"))
	assert.Equal(t, err, types.ErrNotFound)
	_, err = ldb.List(nil, nil, 0, 0)
	assert.Equal(t, err, nil)
	ldb.DisableWrite()
	err = ldb.Set([]byte("hello"), nil)
	assert.Equal(t, err, types.ErrDisableWrite)
	ldb.EnableWrite()
	err = ldb.Set([]byte("hello"), nil)
	assert.Equal(t, err, nil)

}

func BenchmarkLocalDBGet(b *testing.B) {
	mockDOM := testnode.New("", nil)
	defer mockDOM.Close()
	db := executor.NewLocalDB(mockDOM.GetClient(), false)
	defer db.(*executor.LocalDB).Close()

	err := db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(b, err)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		v, err := db.Get([]byte("k1"))
		assert.Nil(b, err)
		assert.Equal(b, v, []byte("v1"))
	}
}

func TestLocalDBTxGet(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer mockDOM.Close()
	db := executor.NewLocalDB(mockDOM.GetClient(), false)
	testTxGet(t, db)
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

func TestLocalDB(t *testing.T) {
	mockDOM := testnode.New("", nil)
	defer mockDOM.Close()
	db := executor.NewLocalDB(mockDOM.GetClient(), false)
	defer db.(*executor.LocalDB).Close()
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

	//beigin and rollback not imp
	db.Begin()
	err = db.Set([]byte("k2"), []byte("v2"))
	assert.Nil(t, err)
	db.Rollback()
	_, err = db.Get([]byte("k2"))
	assert.Equal(t, err, types.ErrNotFound)
	err = db.Set([]byte("k2"), []byte("v2"))
	assert.Nil(t, err)
	//get
	v, err = db.Get([]byte("k2"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v2"))
	//list
	values, err := db.List([]byte("k"), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(values), 2)
	assert.Equal(t, string(values[0]), "v2")
	assert.Equal(t, string(values[1]), "v11")
	err = db.Commit()
	assert.Nil(t, err)
	//get
	v, err = db.Get([]byte("k2"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v2"))
	//list
	values, err = db.List([]byte("k"), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(values), 2)
	assert.Equal(t, string(values[0]), "v2")
	assert.Equal(t, string(values[1]), "v11")
}
