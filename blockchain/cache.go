// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"container/list"
	"sync"

	"github.com/D-PlatformOperatingSystem/dpos/types"
	lru "github.com/hashicorp/golang-lru"
)

//BlockCache
type BlockCache struct {
	cache      map[int64]*list.Element
	cacheHash  map[string]*list.Element
	cacheSize  int64
	cachelock  sync.Mutex
	cacheQueue *list.List
	maxHeight  int64 //      cache
	sysPm      *types.DplatformOSConfig
}

//NewBlockCache new
func NewBlockCache(param *types.DplatformOSConfig, defCacheSize int64) *BlockCache {
	return &BlockCache{
		cache:      make(map[int64]*list.Element),
		cacheHash:  make(map[string]*list.Element),
		cacheSize:  defCacheSize,
		cacheQueue: list.New(),
		maxHeight:  0,
		sysPm:      param,
	}
}

//CheckcacheBlock  cache     block
func (chain *BlockCache) CheckcacheBlock(height int64) (block *types.BlockDetail) {
	chain.cachelock.Lock()
	defer chain.cachelock.Unlock()

	elem, ok := chain.cache[height]
	if ok {
		// Already exists. Move to back of cacheQueue.
		chain.cacheQueue.MoveToBack(elem)
		return elem.Value.(*types.BlockDetail)
	}
	return nil
}

//GetCacheBlock     ，cache    128
func (chain *BlockCache) GetCacheBlock(hash []byte) (block *types.BlockDetail) {
	chain.cachelock.Lock()
	defer chain.cachelock.Unlock()
	elem, ok := chain.cacheHash[string(hash)]
	if ok {
		return elem.Value.(*types.BlockDetail)
	}
	return nil
}

//CacheBlock   block cache ，
func (chain *BlockCache) CacheBlock(blockdetail *types.BlockDetail) {
	chain.cachelock.Lock()
	defer chain.cachelock.Unlock()
	if chain.maxHeight > 0 && blockdetail.Block.Height != chain.maxHeight+1 {
		chainlog.Error("...cacheBlock not continue...")
		if types.Debug {
			panic("...cacheBlock not continue...")
		}
	}
	chain.maxHeight = blockdetail.Block.Height
	if len(blockdetail.Receipts) == 0 && len(blockdetail.Block.Txs) != 0 {
		chainlog.Debug("cacheBlock  Receipts == 0", "height", blockdetail.Block.GetHeight())
	}
	chain.addCacheBlock(blockdetail)

	// Maybe expire an item.
	if int64(chain.cacheQueue.Len()) > chain.cacheSize {
		blockdetail := chain.cacheQueue.Remove(chain.cacheQueue.Front()).(*types.BlockDetail)
		chain.delCacheBlock(blockdetail)
	}
}

//DelBlockFromCache   block cache ，
func (chain *BlockCache) DelBlockFromCache(height int64) {
	chain.cachelock.Lock()
	defer chain.cachelock.Unlock()
	if chain.maxHeight > 0 && height != chain.maxHeight {
		chainlog.Error("...del cacheBlock not continue...")
		if types.Debug {
			panic("...del cacheBlock not continue...")
		}
	}
	chain.maxHeight = height - 1
	elem, ok := chain.cache[height]
	if ok {
		blockdetail := chain.cacheQueue.Remove(elem).(*types.BlockDetail)
		chain.delCacheBlock(blockdetail)
	}
}

func (chain *BlockCache) addCacheBlock(blockdetail *types.BlockDetail) {
	// Create entry in cache and append to cacheQueue.
	elem := chain.cacheQueue.PushBack(blockdetail)
	chain.cache[blockdetail.Block.Height] = elem
	chain.cacheHash[string(blockdetail.Block.Hash(chain.sysPm))] = elem
}

func (chain *BlockCache) delCacheBlock(blockdetail *types.BlockDetail) {
	delete(chain.cache, blockdetail.Block.Height)
	delete(chain.cacheHash, string(blockdetail.Block.Hash(chain.sysPm)))
}

// cache tx

//             hash
type cacheTx struct {
	height   int64 //      cache
	txHashes []string
}

//TxCache   hash
type TxCache struct {
	capacity int
	cacheTxs map[string]bool
	data     *lru.Cache
	lock     *sync.RWMutex
}

//NewTxCache new
func NewTxCache(defCacheSize int) *TxCache {
	cache := &TxCache{cacheTxs: make(map[string]bool), capacity: defCacheSize, lock: &sync.RWMutex{}}
	var err error
	cache.data, err = lru.New(defCacheSize)
	if err != nil {
		panic(err)
	}
	return cache

}

// Add :              hash
func (c *TxCache) Add(block *types.Block) bool {

	c.lock.Lock()
	defer c.lock.Unlock()

	//
	if txcache, exist := c.data.Peek(block.GetHeight()); exist {
		for _, tx := range txcache.(cacheTx).txHashes {
			delete(c.cacheTxs, tx)
		}
		c.data.Remove(block.GetHeight())
	}

	//      ,
	if c.data.Len() >= c.capacity {
		_, v, ok := c.data.RemoveOldest()
		if !ok {
			chainlog.Error("TxCache.Add RemoveOldest fail ...", "len", c.data.Len(), "capacity", c.capacity)
			return false
		}
		for _, tx := range v.(cacheTx).txHashes {
			delete(c.cacheTxs, tx)
		}
	}

	//        hash
	var txs cacheTx
	txs.height = block.GetHeight()
	for _, tx := range block.Txs {
		txhash := string(tx.Hash())
		txs.txHashes = append(txs.txHashes, txhash)
		c.cacheTxs[txhash] = true
	}
	c.data.Add(block.GetHeight(), txs)
	return true
}

// Del :             hash
func (c *TxCache) Del(height int64) bool {

	c.lock.Lock()
	defer c.lock.Unlock()

	//
	if txcache, exist := c.data.Peek(height); exist {
		for _, tx := range txcache.(cacheTx).txHashes {
			delete(c.cacheTxs, tx)
		}
		return c.data.Remove(height)
	}
	return true
}

//HasCacheTx
func (c *TxCache) HasCacheTx(hash []byte) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.cacheTxs[string(hash)]
	return ok
}
