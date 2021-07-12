// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"container/list"
	"math/big"
	"sync/atomic"

	"github.com/D-PlatformOperatingSystem/dpos/client/api"
	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/difficulty"
	"github.com/D-PlatformOperatingSystem/dpos/types"
	"github.com/D-PlatformOperatingSystem/dpos/util"
)

//ProcessBlock          blockdetail，peer     block，   peer     block
//      peer     block
//       Receipts   ,        Receipts
//       ：    ，      ，  err
func (b *BlockChain) ProcessBlock(broadcast bool, block *types.BlockDetail, pid string, addBlock bool, sequence int64) (*types.BlockDetail, bool, bool, error) {
	chainlog.Debug("ProcessBlock:Processing", "height", block.Block.Height, "blockHash", common.ToHex(block.Block.Hash(b.client.GetConfig())))

	//blockchain close      block
	if atomic.LoadInt32(&b.isclosed) == 1 {
		return nil, false, false, types.ErrIsClosed
	}
	cfg := b.client.GetConfig()
	if block.Block.Height > 0 {
		var lastBlockHash []byte
		if addBlock {
			lastBlockHash = block.Block.GetParentHash()
		} else {
			lastBlockHash = block.Block.Hash(cfg)
		}
		if pid == "self" && !bytes.Equal(lastBlockHash, b.bestChain.Tip().hash) {
			chainlog.Error("addBlockDetail parent hash no match", "err", types.ErrBlockHashNoMatch,
				"bestHash", common.ToHex(b.bestChain.Tip().hash), "blockHash", common.ToHex(lastBlockHash),
				"addBlock", addBlock, "height", block.Block.Height)
			return nil, false, false, types.ErrBlockHashNoMatch
		}
	}
	blockHash := block.Block.Hash(cfg)

	//           block  ,       block
	if !addBlock {
		if b.isParaChain {
			return b.ProcessDelParaChainBlock(broadcast, block, pid, sequence)
		}
		return nil, false, false, types.ErrNotSupport
	}
	//   block
	//   block    ，           ，
	//  block  peer       peerlist
	exists := b.blockExists(blockHash)
	if exists {
		is, err := b.IsErrExecBlock(block.Block.Height, blockHash)
		if is {
			b.RecordFaultPeer(pid, block.Block.Height, blockHash, err)
		}
		chainlog.Debug("ProcessBlock already have block", "blockHash", common.ToHex(blockHash))
		return nil, false, false, types.ErrBlockExist
	}

	//
	exists = b.orphanPool.IsKnownOrphan(blockHash)
	if exists {
		//           ，
		//                ，
		//                ，
		if b.blockExists(block.Block.GetParentHash()) {
			b.orphanPool.RemoveOrphanBlockByHash(blockHash)
			chainlog.Debug("ProcessBlock:maybe Accept Orphan Block", "blockHash", common.ToHex(blockHash))
		} else {
			chainlog.Debug("ProcessBlock already have block(orphan)", "blockHash", common.ToHex(blockHash))
			return nil, false, false, types.ErrBlockExist
		}
	}

	//   block  block    ，        block
	//   0
	var prevHashExists bool
	prevHash := block.Block.GetParentHash()
	if 0 == block.Block.GetHeight() {
		if bytes.Equal(prevHash, make([]byte, sha256Len)) {
			prevHashExists = true
		}
	} else {
		prevHashExists = b.blockExists(prevHash)
	}
	if !prevHashExists {
		chainlog.Debug("ProcessBlock:AddOrphanBlock", "height", block.Block.GetHeight(), "blockHash", common.ToHex(blockHash), "prevHash", common.ToHex(prevHash))
		b.orphanPool.AddOrphanBlock(broadcast, block.Block, pid, sequence)
		return nil, false, true, nil
	}

	//             block
	return b.maybeAddBestChain(broadcast, block, pid, sequence)
}

//            block
func (b *BlockChain) maybeAddBestChain(broadcast bool, block *types.BlockDetail, pid string, sequence int64) (*types.BlockDetail, bool, bool, error) {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

	blockHash := block.Block.Hash(b.client.GetConfig())
	exists := b.blockExists(blockHash)
	if exists {
		return nil, false, false, types.ErrBlockExist
	}
	chainlog.Debug("maybeAddBestChain", "height", block.Block.GetHeight(), "blockHash", common.ToHex(blockHash))
	blockdetail, isMainChain, err := b.maybeAcceptBlock(broadcast, block, pid, sequence)

	if err != nil {
		return nil, false, false, err
	}
	//     blockHash
	err = b.orphanPool.ProcessOrphans(blockHash, b)
	if err != nil {
		return nil, false, false, err
	}
	return blockdetail, isMainChain, false, nil
}

//  block      index
func (b *BlockChain) blockExists(hash []byte) bool {
	// Check block index first (could be main chain or side chain blocks).
	if b.index.HaveBlock(hash) {
		return true
	}

	//           ，  hash  blockheader，      false。
	blockheader, err := b.blockStore.GetBlockHeaderByHash(hash)
	if blockheader == nil || err != nil {
		return false
	}
	//block       ，          。       false
	height, err := b.blockStore.GetHeightByBlockHash(hash)
	if err != nil {
		return false
	}
	return height != -1
}

//      block
func (b *BlockChain) maybeAcceptBlock(broadcast bool, block *types.BlockDetail, pid string, sequence int64) (*types.BlockDetail, bool, error) {
	//      block Parent block    index
	prevHash := block.Block.GetParentHash()
	prevNode := b.index.LookupNode(prevHash)
	if prevNode == nil {
		chainlog.Debug("maybeAcceptBlock", "previous block is unknown", common.ToHex(prevHash))
		return nil, false, types.ErrParentBlockNoExist
	}

	blockHeight := block.Block.GetHeight()
	if blockHeight != prevNode.height+1 {
		chainlog.Debug("maybeAcceptBlock height err", "blockHeight", blockHeight, "prevHeight", prevNode.height)
		return nil, false, types.ErrBlockHeightNoMatch
	}

	//  block   db ，    blockchain     ，     saveblock   hash
	sync := true
	if atomic.LoadInt32(&b.isbatchsync) == 0 {
		sync = false
	}

	err := b.blockStore.dbMaybeStoreBlock(block, sync)
	if err != nil {
		if err == types.ErrDataBaseDamage {
			chainlog.Error("dbMaybeStoreBlock newbatch.Write", "err", err)
			go util.ReportErrEventToFront(chainlog, b.client, "blockchain", "wallet", types.ErrDataBaseDamage)
		}
		return nil, false, err
	}
	//     node       index
	cfg := b.client.GetConfig()
	newNode := newBlockNode(cfg, broadcast, block.Block, pid, sequence)
	if prevNode != nil {
		newNode.parent = prevNode
	}
	b.index.AddNode(newNode)

	//   block
	var isMainChain bool
	block, isMainChain, err = b.connectBestChain(newNode, block)
	if err != nil {
		return nil, false, err
	}

	return block, isMainChain, nil
}

// block
func (b *BlockChain) connectBestChain(node *blockNode, block *types.BlockDetail) (*types.BlockDetail, bool, error) {

	enBestBlockCmp := b.client.GetConfig().GetModuleConfig().Consensus.EnableBestBlockCmp
	parentHash := block.Block.GetParentHash()
	tip := b.bestChain.Tip()
	cfg := b.client.GetConfig()

	//   block      ,tip       block    .
	if bytes.Equal(parentHash, tip.hash) {
		var err error
		block, err = b.connectBlock(node, block)
		if err != nil {
			return nil, false, err
		}
		return block, true, nil
	}
	chainlog.Debug("connectBestChain", "parentHash", common.ToHex(parentHash), "bestChain.Tip().hash", common.ToHex(tip.hash))

	//   tip   block   tipid
	tiptd, Err := b.blockStore.GetTdByBlockHash(tip.hash)
	if tiptd == nil || Err != nil {
		chainlog.Error("connectBestChain tiptd is not exits!", "height", tip.height, "b.bestChain.Tip().hash", common.ToHex(tip.hash))
		return nil, false, Err
	}
	parenttd, Err := b.blockStore.GetTdByBlockHash(parentHash)
	if parenttd == nil || Err != nil {
		chainlog.Error("connectBestChain parenttd is not exits!", "height", block.Block.Height, "parentHash", common.ToHex(parentHash), "block.Block.hash", common.ToHex(block.Block.Hash(cfg)))
		return nil, false, types.ErrParentTdNoExist
	}
	blocktd := new(big.Int).Add(node.Difficulty, parenttd)

	chainlog.Debug("connectBestChain tip:", "hash", common.ToHex(tip.hash), "height", tip.height, "TD", difficulty.BigToCompact(tiptd))
	chainlog.Debug("connectBestChain node:", "hash", common.ToHex(node.hash), "height", node.height, "TD", difficulty.BigToCompact(blocktd))

	//
	//     ，    ，                       ，
	iSideChain := blocktd.Cmp(tiptd) <= 0
	if enBestBlockCmp && blocktd.Cmp(tiptd) == 0 && node.height == tip.height && util.CmpBestBlock(b.client, block.Block, tip.hash) {
		iSideChain = false
	}
	if iSideChain {
		fork := b.bestChain.FindFork(node)
		if fork != nil && bytes.Equal(parentHash, fork.hash) {
			chainlog.Info("connectBestChain FORK:", "Block hash", common.ToHex(node.hash), "fork.height", fork.height, "fork.hash", common.ToHex(fork.hash))
		} else {
			chainlog.Info("connectBestChain extends a side chain:", "Block hash", common.ToHex(node.hash), "fork.height", fork.height, "fork.hash", common.ToHex(fork.hash))
		}
		return nil, false, nil
	}

	//print
	chainlog.Debug("connectBestChain tip", "height", tip.height, "hash", common.ToHex(tip.hash))
	chainlog.Debug("connectBestChain node", "height", node.height, "hash", common.ToHex(node.hash), "parentHash", common.ToHex(parentHash))
	chainlog.Debug("connectBestChain block", "height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)))

	//        block node
	detachNodes, attachNodes := b.getReorganizeNodes(node)

	// Reorganize the chain.
	err := b.reorganizeChain(detachNodes, attachNodes)
	if err != nil {
		return nil, false, err
	}

	return nil, true, nil
}

//  block         ，   bestchain tip
func (b *BlockChain) connectBlock(node *blockNode, blockdetail *types.BlockDetail) (*types.BlockDetail, error) {
	//blockchain close      block
	if atomic.LoadInt32(&b.isclosed) == 1 {
		return nil, types.ErrIsClosed
	}

	// Make sure it's extending the end of the best chain.
	parentHash := blockdetail.Block.GetParentHash()
	if !bytes.Equal(parentHash, b.bestChain.Tip().hash) {
		chainlog.Error("connectBlock hash err", "height", blockdetail.Block.Height, "Tip.height", b.bestChain.Tip().height)
		return nil, types.ErrBlockHashNoMatch
	}

	sync := true
	if atomic.LoadInt32(&b.isbatchsync) == 0 {
		sync = false
	}

	var err error
	var lastSequence int64

	block := blockdetail.Block
	prevStateHash := b.bestChain.Tip().statehash
	errReturn := (node.pid != "self")
	blockdetail, _, err = execBlock(b.client, prevStateHash, block, errReturn, sync)
	if err != nil {
		//       block  ,            ，      ，
		//                     ，   index
		ok := IsRecordFaultErr(err)

		if node.pid == "download" || (!ok && node.pid == "self") {
			//       block  api  queue          block index    ，
			//            ，           block
			//                           block
			chainlog.Debug("connectBlock DelNode!", "height", block.Height, "node.hash", common.ToHex(node.hash), "err", err)
			b.index.DelNode(node.hash)
		} else {
			b.RecordFaultPeer(node.pid, block.Height, node.hash, err)
		}
		chainlog.Error("connectBlock ExecBlock is err!", "height", block.Height, "err", err)
		return nil, err
	}
	cfg := b.client.GetConfig()
	//   node
	if node.pid == "self" {
		prevhash := node.hash
		node.statehash = blockdetail.Block.GetStateHash()
		node.hash = blockdetail.Block.Hash(cfg)
		b.index.UpdateNode(prevhash, node)
	}

	//         block
	newbatch := b.blockStore.batch
	newbatch.Reset()
	newbatch.UpdateWriteSync(sync)
	//  tx   db
	beg := types.Now()
	err = b.blockStore.AddTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("connectBlock indexTxs:", "height", block.Height, "err", err)
		return nil, err
	}
	txCost := types.Since(beg)
	beg = types.Now()
	//  block   db
	lastSequence, err = b.blockStore.SaveBlock(newbatch, blockdetail, node.sequence)
	if err != nil {
		chainlog.Error("connectBlock SaveBlock:", "height", block.Height, "err", err)
		return nil, err
	}
	saveBlkCost := types.Since(beg)
	//cache new add block
	beg = types.Now()
	b.cache.CacheBlock(blockdetail)
	b.txCache.Add(blockdetail.Block)
	cacheCost := types.Since(beg)

	//  block     db
	difficulty := difficulty.CalcWork(block.Difficulty)
	var blocktd *big.Int
	if block.Height == 0 {
		blocktd = difficulty
	} else {
		parenttd, err := b.blockStore.GetTdByBlockHash(parentHash)
		if err != nil {
			chainlog.Error("connectBlock GetTdByBlockHash", "height", block.Height, "parentHash", common.ToHex(parentHash))
			return nil, err
		}
		blocktd = new(big.Int).Add(difficulty, parenttd)
	}

	err = b.blockStore.SaveTdByBlockHash(newbatch, blockdetail.Block.Hash(cfg), blocktd)
	if err != nil {
		chainlog.Error("connectBlock SaveTdByBlockHash:", "height", block.Height, "err", err)
		return nil, err
	}
	beg = types.Now()
	err = newbatch.Write()
	if err != nil {
		chainlog.Error("connectBlock newbatch.Write", "err", err)
		panic(err)
	}
	writeCost := types.Since(beg)
	chainlog.Debug("ConnectBlock", "execLocal", txCost, "saveBlk", saveBlkCost, "cacheBlk", cacheCost, "writeBatch", writeCost)
	chainlog.Debug("connectBlock info", "height", block.Height, "batchsync", sync, "hash", common.ToHex(blockdetail.Block.Hash(cfg)))

	//         header
	b.blockStore.UpdateHeight2(blockdetail.GetBlock().GetHeight())
	b.blockStore.UpdateLastBlock2(blockdetail.Block)

	//    best chain tip
	b.bestChain.SetTip(node)

	b.query.updateStateHash(blockdetail.GetBlock().GetStateHash())

	err = b.SendAddBlockEvent(blockdetail)
	if err != nil {
		chainlog.Debug("connectBlock SendAddBlockEvent", "err", err)
	}
	//    block     ，
	b.syncTask.Done(blockdetail.Block.GetHeight())

	//   block
	if node.broadcast {
		if blockdetail.Block.BlockTime-types.Now().Unix() > FutureBlockDelayTime {
			//  block   futureblocks
			b.futureBlocks.Add(string(blockdetail.Block.Hash(cfg)), blockdetail)
			chainlog.Debug("connectBlock futureBlocks.Add", "height", block.Height, "hash", common.ToHex(blockdetail.Block.Hash(cfg)), "blocktime", blockdetail.Block.BlockTime, "curtime", types.Now().Unix())
		} else {
			b.SendBlockBroadcast(blockdetail)
		}
	}

	//         isRecordBlockSequence   enablePushSubscribe
	if b.isRecordBlockSequence && b.enablePushSubscribe {
		b.push.UpdateSeq(lastSequence)
		chainlog.Debug("isRecordBlockSequence", "lastSequence", lastSequence, "height", block.Height)
	}
	return blockdetail, nil
}

//      blocks
func (b *BlockChain) disconnectBlock(node *blockNode, blockdetail *types.BlockDetail, sequence int64) error {
	var lastSequence int64
	//     best chain tip
	if !bytes.Equal(node.hash, b.bestChain.Tip().hash) {
		chainlog.Error("disconnectBlock:", "height", blockdetail.Block.Height, "node.hash", common.ToHex(node.hash), "bestChain.top.hash", common.ToHex(b.bestChain.Tip().hash))
		return types.ErrBlockHashNoMatch
	}

	//    block
	newbatch := b.blockStore.NewBatch(true)

	// db   tx
	err := b.blockStore.DelTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("disconnectBlock DelTxs:", "height", blockdetail.Block.Height, "err", err)
		return err
	}

	// db   block
	lastSequence, err = b.blockStore.DelBlock(newbatch, blockdetail, sequence)
	if err != nil {
		chainlog.Error("disconnectBlock DelBlock:", "height", blockdetail.Block.Height, "err", err)
		return err
	}
	err = newbatch.Write()
	if err != nil {
		chainlog.Error("disconnectBlock newbatch.Write", "err", err)
		panic(err)
	}
	//        header
	b.blockStore.UpdateHeight()
	b.blockStore.UpdateLastBlock(blockdetail.Block.ParentHash)

	//      tip  ，        tip
	b.bestChain.DelTip(node)

	//    ，mempool     block
	err = b.SendDelBlockEvent(blockdetail)
	if err != nil {
		chainlog.Error("disconnectBlock SendDelBlockEvent", "err", err)
	}
	b.query.updateStateHash(node.parent.statehash)

	//  node       tip
	newtipnode := b.bestChain.Tip()

	//      block
	b.DelCacheBlock(blockdetail.Block.Height, node.hash)

	if newtipnode != node.parent {
		chainlog.Error("disconnectBlock newtipnode err:", "newtipnode.height", newtipnode.height, "node.parent.height", node.parent.height)
	}
	if !bytes.Equal(blockdetail.Block.GetParentHash(), b.bestChain.Tip().hash) {
		chainlog.Error("disconnectBlock", "newtipnode.height", newtipnode.height, "node.parent.height", node.parent.height)
		chainlog.Error("disconnectBlock", "newtipnode.hash", common.ToHex(newtipnode.hash), "delblock.parent.hash", common.ToHex(blockdetail.Block.GetParentHash()))
	}

	chainlog.Debug("disconnectBlock success", "newtipnode.height", newtipnode.height, "node.parent.height", node.parent.height)
	chainlog.Debug("disconnectBlock success", "newtipnode.hash", common.ToHex(newtipnode.hash), "delblock.parent.hash", common.ToHex(blockdetail.Block.GetParentHash()))

	//         isRecordBlockSequence   enablePushSubscribe
	if b.isRecordBlockSequence && b.enablePushSubscribe {
		b.push.UpdateSeq(lastSequence)
		chainlog.Debug("isRecordBlockSequence", "lastSequence", lastSequence, "height", blockdetail.Block.Height)
	}

	return nil
}

//    blockchain
func (b *BlockChain) getReorganizeNodes(node *blockNode) (*list.List, *list.List) {
	attachNodes := list.New()
	detachNodes := list.New()

	//         ，       block index push attachNodes
	forkNode := b.bestChain.FindFork(node)
	for n := node; n != nil && n != forkNode; n = n.parent {
		attachNodes.PushFront(n)
	}

	//         ，       block bestchain push attachNodes
	for n := b.bestChain.Tip(); n != nil && n != forkNode; n = n.parent {
		detachNodes.PushBack(n)
	}

	return detachNodes, attachNodes
}

//LoadBlockByHash   hash
func (b *BlockChain) LoadBlockByHash(hash []byte) (block *types.BlockDetail, err error) {

	//
	block = b.cache.GetCacheBlock(hash)
	if block != nil {
		return block, err
	}

	//
	block, _ = b.blockStore.GetActiveBlock(string(hash))
	if block != nil {
		return block, err
	}

	//
	block, err = b.blockStore.LoadBlockByHash(hash)

	//
	if block != nil {
		mainHash, _ := b.blockStore.GetBlockHashByHeight(block.Block.GetHeight())
		if mainHash != nil && bytes.Equal(mainHash, hash) {
			b.blockStore.AddActiveBlock(string(hash), block)
		}
	}
	return block, err
}

//  blockchain
func (b *BlockChain) reorganizeChain(detachNodes, attachNodes *list.List) error {
	detachBlocks := make([]*types.BlockDetail, 0, detachNodes.Len())
	attachBlocks := make([]*types.BlockDetail, 0, attachNodes.Len())

	//  node  blockhash  block   db
	cfg := b.client.GetConfig()
	for e := detachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		block, err := b.LoadBlockByHash(n.hash)

		//      blocks
		if block != nil && err == nil {
			detachBlocks = append(detachBlocks, block)
			chainlog.Debug("reorganizeChain detachBlocks ", "height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)))
		} else {
			chainlog.Error("reorganizeChain detachBlocks fail", "height", n.height, "hash", common.ToHex(n.hash), "err", err)
			return err
		}
	}

	for e := attachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		block, err := b.LoadBlockByHash(n.hash)

		//      db blocks
		if block != nil && err == nil {
			attachBlocks = append(attachBlocks, block)
			chainlog.Debug("reorganizeChain attachBlocks ", "height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)))
		} else {
			chainlog.Error("reorganizeChain attachBlocks fail", "height", n.height, "hash", common.ToHex(n.hash), "err", err)
			return err
		}
	}

	// Disconnect blocks from the main chain.
	for i, e := 0, detachNodes.Front(); e != nil; i, e = i+1, e.Next() {
		n := e.Value.(*blockNode)
		block := detachBlocks[i]

		// Update the database and chain state.
		err := b.disconnectBlock(n, block, n.sequence)
		if err != nil {
			return err
		}
	}

	// Connect the new best chain blocks.
	for i, e := 0, attachNodes.Front(); e != nil; i, e = i+1, e.Next() {
		n := e.Value.(*blockNode)
		block := attachBlocks[i]

		// Update the database and chain state.
		_, err := b.connectBlock(n, block)
		if err != nil {
			return err
		}
	}

	// Log the point where the chain forked and old and new best chain
	// heads.
	if attachNodes.Front() != nil {
		firstAttachNode := attachNodes.Front().Value.(*blockNode)
		chainlog.Debug("REORGANIZE: Chain forks at hash", "hash", common.ToHex(firstAttachNode.parent.hash), "height", firstAttachNode.parent.height)

	}
	if detachNodes.Front() != nil {
		firstDetachNode := detachNodes.Front().Value.(*blockNode)
		chainlog.Debug("REORGANIZE: Old best chain head was hash", "hash", common.ToHex(firstDetachNode.hash), "height", firstDetachNode.parent.height)

	}
	if attachNodes.Back() != nil {
		lastAttachNode := attachNodes.Back().Value.(*blockNode)
		chainlog.Debug("REORGANIZE: New best chain head is hash", "hash", common.ToHex(lastAttachNode.hash), "height", lastAttachNode.parent.height)
	}
	return nil
}

//ProcessDelParaChainBlock     best chain tip      ，
func (b *BlockChain) ProcessDelParaChainBlock(broadcast bool, blockdetail *types.BlockDetail, pid string, sequence int64) (*types.BlockDetail, bool, bool, error) {

	//     tip
	tipnode := b.bestChain.Tip()
	blockHash := blockdetail.Block.Hash(b.client.GetConfig())

	if !bytes.Equal(blockHash, b.bestChain.Tip().hash) {
		chainlog.Error("ProcessDelParaChainBlock:", "delblockheight", blockdetail.Block.Height, "delblockHash", common.ToHex(blockHash), "bestChain.top.hash", common.ToHex(b.bestChain.Tip().hash))
		return nil, false, false, types.ErrBlockHashNoMatch
	}
	err := b.disconnectBlock(tipnode, blockdetail, sequence)
	if err != nil {
		return nil, false, false, err
	}
	//                       ，
	//              disconnectBlock
	//        index
	b.index.DelNode(blockHash)

	return nil, true, false, nil
}

// IsRecordFaultErr
func IsRecordFaultErr(err error) bool {
	return err != types.ErrFutureBlock && !api.IsGrpcError(err) && !api.IsQueueError(err)
}
