// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found In the LICENSE file.

package mempool

import (
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//QueueCache
type QueueCache interface {
	Exist(hash string) bool
	GetItem(hash string) (*Item, error)
	Push(tx *Item) error
	Remove(hash string) error
	Size() int
	Walk(count int, cb func(tx *Item) bool)
	GetProperFee() int64
	GetCacheBytes() int64
}

// Item  Mempool
type Item struct {
	Value     *types.Transaction
	Priority  int64
	EnterTime int64
}

//TxCache     cache       ，     ，
type txCache struct {
	*AccountTxIndex
	*LastTxCache
	qcache   QueueCache
	totalFee int64
	*SHashTxCache
}

//NewTxCache init accountIndex and last cache
func newCache(maxTxPerAccount int64, sizeLast int64, poolCacheSize int64) *txCache {
	return &txCache{
		AccountTxIndex: NewAccountTxIndex(int(maxTxPerAccount)),
		LastTxCache:    NewLastTxCache(int(sizeLast)),
		SHashTxCache:   NewSHashTxCache(int(poolCacheSize)),
	}
}

//SetQueueCache set queue cache ,
func (cache *txCache) SetQueueCache(qcache QueueCache) {
	cache.qcache = qcache
}

//Remove   txCache   tx
func (cache *txCache) Remove(hash string) {
	item, err := cache.qcache.GetItem(hash)
	if err != nil {
		return
	}
	tx := item.Value
	err = cache.qcache.Remove(hash)
	if err != nil {
		mlog.Error("Remove", "cache Remove err", err)
	}
	cache.AccountTxIndex.Remove(tx)
	cache.LastTxCache.Remove(tx)
	cache.totalFee -= tx.Fee
	cache.SHashTxCache.Remove(tx)
}

//Exist
func (cache *txCache) Exist(hash string) bool {
	if cache.qcache == nil {
		return false
	}
	return cache.qcache.Exist(hash)
}

//Size cache tx num
func (cache *txCache) Size() int {
	if cache.qcache == nil {
		return 0
	}
	return cache.qcache.Size()
}

//TotalFee
func (cache *txCache) TotalFee() int64 {
	return cache.totalFee
}

//Walk iter all txs
func (cache *txCache) Walk(count int, cb func(tx *Item) bool) {
	if cache.qcache == nil {
		return
	}
	cache.qcache.Walk(count, cb)
}

//RemoveTxs
func (cache *txCache) RemoveTxs(txs []string) {
	for _, t := range txs {
		cache.Remove(t)
	}
}

//Push      cache
func (cache *txCache) Push(tx *types.Transaction) error {
	if !cache.AccountTxIndex.CanPush(tx) {
		return types.ErrManyTx
	}
	item := &Item{Value: tx, Priority: tx.Fee, EnterTime: types.Now().Unix()}
	err := cache.qcache.Push(item)
	if err != nil {
		return err
	}
	err = cache.AccountTxIndex.Push(tx)
	if err != nil {
		return err
	}
	cache.LastTxCache.Push(tx)
	cache.totalFee += tx.Fee
	cache.SHashTxCache.Push(tx)
	return nil
}

func (cache *txCache) removeExpiredTx(cfg *types.DplatformOSConfig, height, blocktime int64) {
	var txs []string
	cache.qcache.Walk(0, func(tx *Item) bool {
		if isExpired(cfg, tx, height, blocktime) {
			txs = append(txs, string(tx.Value.Hash()))
		}
		return true
	})
	cache.RemoveTxs(txs)
}

//
func isExpired(cfg *types.DplatformOSConfig, item *Item, height, blockTime int64) bool {
	if types.Now().Unix()-item.EnterTime >= mempoolExpiredInterval {
		return true
	}
	if item.Value.IsExpire(cfg, height, blockTime) {
		return true
	}
	return false
}

//getTxByHash     hash  tx
func (cache *txCache) getTxByHash(hash string) *types.Transaction {
	item, err := cache.qcache.GetItem(hash)
	if err != nil {
		return nil
	}
	return item.Value
}
