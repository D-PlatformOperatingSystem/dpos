package mempool

import (
	"github.com/D-PlatformOperatingSystem/dpos/common/listmap"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//LastTxCache     cache
type LastTxCache struct {
	max int
	l   *listmap.ListMap
}

//NewLastTxCache        cache
func NewLastTxCache(size int) *LastTxCache {
	return &LastTxCache{
		max: size,
		l:   listmap.New(),
	}
}

//GetLatestTx          txCache
func (cache *LastTxCache) GetLatestTx() (txs []*types.Transaction) {
	cache.l.Walk(func(v interface{}) bool {
		txs = append(txs, v.(*types.Transaction))
		return true
	})
	return txs
}

//Remove remove tx of last cache
func (cache *LastTxCache) Remove(tx *types.Transaction) {
	cache.l.Remove(string(tx.Hash()))
}

//Push tx into LastTxCache
func (cache *LastTxCache) Push(tx *types.Transaction) {
	if cache.l.Size() >= cache.max {
		v := cache.l.GetTop()
		if v != nil {
			cache.Remove(v.(*types.Transaction))
		}
	}
	cache.l.Push(string(tx.Hash()), tx)
}
