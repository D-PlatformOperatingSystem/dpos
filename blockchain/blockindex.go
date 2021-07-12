// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"container/list"
	"math/big"
	"sync"

	"github.com/D-PlatformOperatingSystem/dpos/common/difficulty"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

type blockNode struct {
	parent     *blockNode
	hash       []byte
	Difficulty *big.Int
	height     int64
	statehash  []byte
	broadcast  bool
	pid        string
	sequence   int64
	BlockTime  int64
}

type blockIndex struct {
	sync.RWMutex
	index      map[string]*list.Element
	cacheQueue *list.List
}

const (
	indexCacheLimit = 102400 //     index   blocknode
)

func initBlockNode(cfg *types.DplatformOSConfig, node *blockNode, block *types.Block, broadcast bool, pid string, sequence int64) {
	*node = blockNode{
		hash:       block.Hash(cfg),
		Difficulty: difficulty.CalcWork(block.Difficulty),
		height:     block.Height,
		statehash:  block.GetStateHash(),
		broadcast:  broadcast,
		pid:        pid,
		sequence:   sequence,
		BlockTime:  block.BlockTime,
	}
}

func newBlockNode(cfg *types.DplatformOSConfig, broadcast bool, block *types.Block, pid string, sequence int64) *blockNode {
	var node blockNode
	initBlockNode(cfg, &node, block, broadcast, pid, sequence)
	return &node
}

//       blocknode
func newBlockNodeByHeader(broadcast bool, header *types.Header, pid string, sequence int64) *blockNode {
	node := &blockNode{
		hash:       header.Hash,
		Difficulty: difficulty.CalcWork(header.Difficulty),
		height:     header.Height,
		statehash:  header.GetStateHash(),
		broadcast:  broadcast,
		pid:        pid,
		sequence:   sequence,
		BlockTime:  header.BlockTime,
	}
	return node
}

var sha256Len = 32

func newPreGenBlockNode() *blockNode {
	node := &blockNode{
		hash:       make([]byte, sha256Len),
		Difficulty: big.NewInt(-1),
		height:     -1,
		statehash:  make([]byte, sha256Len),
		broadcast:  false,
		pid:        "self",
	}
	return node
}

func (node *blockNode) Ancestor(height int64) *blockNode {
	if height < 0 || height > node.height {
		return nil
	}

	n := node
	for ; n != nil && n.height != height; n = n.parent {
		// Intentionally left blank
	}

	return n
}
func (node *blockNode) RelativeAncestor(distance int64) *blockNode {
	return node.Ancestor(node.height - distance)
}

func newBlockIndex() *blockIndex {
	return &blockIndex{
		index:      make(map[string]*list.Element),
		cacheQueue: list.New(),
	}
}

func (bi *blockIndex) HaveBlock(hash []byte) bool {
	bi.RLock()
	defer bi.RUnlock()
	_, hasBlock := bi.index[string(hash)]

	return hasBlock
}

func (bi *blockIndex) LookupNode(hash []byte) *blockNode {
	bi.Lock()
	defer bi.Unlock()

	elem, ok := bi.index[string(hash)]
	if ok {
		return elem.Value.(*blockNode)
	}

	return nil
}

func (bi *blockIndex) AddNode(node *blockNode) {
	bi.Lock()
	defer bi.Unlock()

	// Create entry in cache and append to cacheQueue.
	elem := bi.cacheQueue.PushBack(node)
	bi.index[string(node.hash)] = elem

	// Maybe expire an item.
	if int64(bi.cacheQueue.Len()) > indexCacheLimit {
		delnode := bi.cacheQueue.Remove(bi.cacheQueue.Front()).(*blockNode)
		//      parent      ，    parent
		delnode.parent = nil
		delete(bi.index, string(delnode.hash))
	}
}

func (bi *blockIndex) UpdateNode(hash []byte, node *blockNode) bool {
	bi.Lock()
	defer bi.Unlock()
	elem, ok := bi.index[string(hash)]
	if !ok {
		return false
	}
	elem.Value = node
	delete(bi.index, string(hash))
	bi.index[string(node.hash)] = elem
	return true
}

//  index
func (bi *blockIndex) DelNode(hash []byte) {
	bi.Lock()
	defer bi.Unlock()

	elem, ok := bi.index[string(hash)]
	if ok {
		delnode := bi.cacheQueue.Remove(elem).(*blockNode)
		delnode.parent = nil
		delete(bi.index, string(hash))
	}
}
