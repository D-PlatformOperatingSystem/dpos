// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"github.com/D-PlatformOperatingSystem/dpos/common/listmap"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

// SubConfig
type SubConfig struct {
	PoolCacheSize int64 `json:"poolCacheSize"`
	ProperFee     int64 `json:"properFee"`
}

//SimpleQueue       (        ，    )
type SimpleQueue struct {
	txList     *listmap.ListMap
	subConfig  SubConfig
	cacheBytes int64
}

//NewSimpleQueue
func NewSimpleQueue(subConfig SubConfig) *SimpleQueue {
	return &SimpleQueue{
		txList:    listmap.New(),
		subConfig: subConfig,
	}
}

//Exist
func (cache *SimpleQueue) Exist(hash string) bool {
	return cache.txList.Exist(hash)
}

//GetItem        key
func (cache *SimpleQueue) GetItem(hash string) (*Item, error) {
	item, err := cache.txList.GetItem(hash)
	if err != nil {
		return nil, err
	}
	return item.(*Item), nil
}

// Push    tx   SimpleQueue；  tx    SimpleQueue  Mempool       error
func (cache *SimpleQueue) Push(tx *Item) error {
	hash := tx.Value.Hash()
	if cache.Exist(string(hash)) {
		return types.ErrTxExist
	}
	if cache.txList.Size() >= int(cache.subConfig.PoolCacheSize) {
		return types.ErrMemFull
	}
	cache.txList.Push(string(hash), tx)
	cache.cacheBytes += int64(types.Size(tx.Value))
	return nil
}

// Remove
func (cache *SimpleQueue) Remove(hash string) error {
	item, err := cache.GetItem(hash)
	if err != nil {
		return err
	}
	cache.txList.Remove(hash)
	cache.cacheBytes -= int64(types.Size(item.Value))
	return nil
}

// Size
func (cache *SimpleQueue) Size() int {
	return cache.txList.Size()
}

// Walk
func (cache *SimpleQueue) Walk(count int, cb func(value *Item) bool) {
	i := 0
	cache.txList.Walk(func(item interface{}) bool {
		if !cb(item.(*Item)) {
			return false
		}
		i++
		return i != count
	})
}

// GetProperFee
func (cache *SimpleQueue) GetProperFee() int64 {
	return cache.subConfig.ProperFee
}

// GetCacheBytes
func (cache *SimpleQueue) GetCacheBytes() int64 {
	return cache.cacheBytes
}
