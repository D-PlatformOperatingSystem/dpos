// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"fmt"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/merkle"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//ProcGetTransactionByAddr
//    key:addr:flag:height ,value:txhash
//key=addr :
//key=addr:1 :      from
//key=addr:2 :      to
func (chain *BlockChain) ProcGetTransactionByAddr(addr *types.ReqAddr) (*types.ReplyTxInfos, error) {
	if addr == nil || len(addr.Addr) == 0 {
		return nil, types.ErrInvalidParam
	}
	//   10
	if addr.Count == 0 {
		addr.Count = 10
	}

	if int64(addr.Count) > types.MaxBlockCountPerTime {
		return nil, types.ErrMaxCountPerTime
	}
	//
	curheigt := chain.GetBlockHeight()
	if addr.GetHeight() > curheigt || addr.GetHeight() < -1 {
		chainlog.Error("ProcGetTransactionByAddr Height err")
		return nil, types.ErrInvalidParam
	}
	if addr.GetDirection() != 0 && addr.GetDirection() != 1 {
		chainlog.Error("ProcGetTransactionByAddr Direction err")
		return nil, types.ErrInvalidParam
	}
	if addr.GetIndex() < 0 || addr.GetIndex() > types.MaxTxsPerBlock {
		chainlog.Error("ProcGetTransactionByAddr Index err")
		return nil, types.ErrInvalidParam
	}
	//   drivers--> main
	//     ：  --> GetTxsByAddr
	//     ：  --> interface{}
	cfg := chain.client.GetConfig()
	txinfos, err := chain.query.Query(cfg.ExecName("coins"), "GetTxsByAddr", addr)
	if err != nil {
		chainlog.Info("ProcGetTransactionByAddr does not exist tx!", "addr", addr, "err", err)
		return nil, err
	}
	return txinfos.(*types.ReplyTxInfos), nil
}

//ProcGetTransactionByHashes
//type TransactionDetails struct {
//	Txs []*Transaction
//}
//  hashs
func (chain *BlockChain) ProcGetTransactionByHashes(hashs [][]byte) (TxDetails *types.TransactionDetails, err error) {
	if int64(len(hashs)) > types.MaxBlockCountPerTime {
		return nil, types.ErrMaxCountPerTime
	}
	var txDetails types.TransactionDetails
	for _, txhash := range hashs {
		txresult, err := chain.GetTxResultFromDb(txhash)
		var txDetail types.TransactionDetail
		if err == nil && txresult != nil {
			setTxDetailFromTxResult(&txDetail, txresult)

			//chainlog.Debug("ProcGetTransactionByHashes", "txDetail", txDetail.String())
			txDetails.Txs = append(txDetails.Txs, &txDetail)
		} else {
			txDetails.Txs = append(txDetails.Txs, &txDetail) //
			chainlog.Debug("ProcGetTransactionByHashes hash no exit", "txhash", common.ToHex(txhash))
		}
	}
	return &txDetails, nil
}

//getTxHashProofs     txindex txs  proof ，  ：index 0
func getTxHashProofs(Txs []*types.Transaction, index int32) [][]byte {
	txlen := len(Txs)
	leaves := make([][]byte, txlen)

	for index, tx := range Txs {
		leaves[index] = tx.Hash()
	}

	proofs := merkle.GetMerkleBranch(leaves, uint32(index))
	chainlog.Debug("getTransactionDetail", "index", index, "proofs", proofs)

	return proofs
}

//GetTxResultFromDb   txhash  txindex db   tx
//type TxResult struct {
//	Height int64
//	Index  int32
//	Tx     *types.Transaction
//  Receiptdate *ReceiptData
//}
func (chain *BlockChain) GetTxResultFromDb(txhash []byte) (tx *types.TxResult, err error) {
	txinfo, err := chain.blockStore.GetTx(txhash)
	if err != nil {
		return nil, err
	}
	return txinfo, nil
}

//HasTx
func (chain *BlockChain) HasTx(txhash []byte, onlyquerycache bool) (has bool, err error) {
	has = chain.txCache.HasCacheTx(txhash)
	if has {
		return true, nil
	}
	if onlyquerycache {
		return has, nil
	}
	return chain.blockStore.HasTx(txhash)
}

//GetDuplicateTxHashList
func (chain *BlockChain) GetDuplicateTxHashList(txhashlist *types.TxHashList) (duptxhashlist *types.TxHashList, err error) {
	var dupTxHashList types.TxHashList
	onlyquerycache := false
	if txhashlist.Count == -1 {
		onlyquerycache = true
	}
	if txhashlist.Expire != nil && len(txhashlist.Expire) != len(txhashlist.Hashes) {
		return nil, types.ErrInvalidParam
	}
	cfg := chain.client.GetConfig()
	for i, txhash := range txhashlist.Hashes {
		expire := int64(0)
		if txhashlist.Expire != nil {
			expire = txhashlist.Expire[i]
		}
		txHeight := types.GetTxHeight(cfg, expire, txhashlist.Count)
		// txHeight > 0     ，       cache
		if txHeight > 0 {
			onlyquerycache = true
		}
		has, err := chain.HasTx(txhash, onlyquerycache)
		if err == nil && has {
			dupTxHashList.Hashes = append(dupTxHashList.Hashes, txhash)
		}
	}
	return &dupTxHashList, nil
}

/*ProcQueryTxMsg     ：
EventQueryTx(types.ReqHash) : rpc     blockchain       EventQueryTx(types.ReqHash)    ，
         ，     EventTransactionDetail(types.TransactionDetail)
   ：
type ReqHash struct {Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`}
type TransactionDetail struct {Hashs [][]byte `protobuf:"bytes,1,rep,name=hashs,proto3" json:"hashs,omitempty"}
*/
func (chain *BlockChain) ProcQueryTxMsg(txhash []byte) (proof *types.TransactionDetail, err error) {
	txresult, err := chain.GetTxResultFromDb(txhash)
	if err != nil {
		return nil, err
	}
	block, err := chain.GetBlock(txresult.Height)
	if err != nil {
		return nil, err
	}

	var txDetail types.TransactionDetail
	cfg := chain.client.GetConfig()
	height := block.Block.GetHeight()
	if chain.isParaChain {
		height = block.Block.GetMainHeight()
	}
	//    tx txlist  proof,    ForkRootHash  proof
	if !cfg.IsFork(height, "ForkRootHash") {
		proofs := getTxHashProofs(block.Block.Txs, txresult.Index)
		txDetail.Proofs = proofs
	} else {
		txproofs := chain.getMultiLayerProofs(txresult.Height, block.Block.Hash(cfg), block.Block.Txs, txresult.Index)
		txDetail.TxProofs = txproofs
		txDetail.FullHash = block.Block.Txs[txresult.Index].FullHash()
	}

	setTxDetailFromTxResult(&txDetail, txresult)
	return &txDetail, nil
}

func setTxDetailFromTxResult(TransactionDetail *types.TransactionDetail, txresult *types.TxResult) {
	TransactionDetail.Receipt = txresult.Receiptdate
	TransactionDetail.Tx = txresult.GetTx()
	TransactionDetail.Height = txresult.GetHeight()
	TransactionDetail.Index = int64(txresult.GetIndex())
	TransactionDetail.Blocktime = txresult.GetBlocktime()

	//  Amount
	amount, err := txresult.GetTx().Amount()
	if err != nil {
		// return nil, err
		amount = 0
	}
	TransactionDetail.Amount = amount
	assets, err := txresult.GetTx().Assets()
	if err != nil {
		assets = nil
	}
	TransactionDetail.Assets = assets
	TransactionDetail.ActionName = txresult.GetTx().ActionName()

	//  from
	TransactionDetail.Fromaddr = txresult.GetTx().From()
}

//ProcGetAddrOverview   addrOverview
//type  AddrOverview {
//	int64 reciver = 1;
//	int64 balance = 2;
//	int64 txCount = 3;}
func (chain *BlockChain) ProcGetAddrOverview(addr *types.ReqAddr) (*types.AddrOverview, error) {
	cfg := chain.client.GetConfig()
	if addr == nil || len(addr.Addr) == 0 {
		chainlog.Error("ProcGetAddrOverview input err!")
		return nil, types.ErrInvalidParam
	}
	chainlog.Debug("ProcGetAddrOverview", "Addr", addr.GetAddr())

	var addrOverview types.AddrOverview

	//     reciver
	amount, err := chain.query.Query(cfg.ExecName("coins"), "GetAddrReciver", addr)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "GetAddrReciver err", err)
		addrOverview.Reciver = 0
	} else {
		addrOverview.Reciver = amount.(*types.Int64).GetData()
	}
	if err != nil && err != types.ErrEmpty {
		return nil, err
	}

	var reqkey types.ReqKey

	//       PrefixCount        ，executor/localdb.go PrefixCount
	//           localdb，        GetAddrTxsCount
	reqkey.Key = []byte(fmt.Sprintf("AddrTxsCount:%s", addr.Addr))
	count, err := chain.query.Query(cfg.ExecName("coins"), "GetAddrTxsCount", &reqkey)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "GetAddrTxsCount err", err)
		addrOverview.TxCount = 0
	} else {
		addrOverview.TxCount = count.(*types.Int64).GetData()
	}
	chainlog.Debug("GetAddrTxsCount", "TxCount ", addrOverview.TxCount)

	return &addrOverview, nil
}

//getTxFullHashProofs     txinde txs  proof, ForkRootHash    fullhash
func getTxFullHashProofs(Txs []*types.Transaction, index int32) [][]byte {
	txlen := len(Txs)
	leaves := make([][]byte, txlen)

	for index, tx := range Txs {
		leaves[index] = tx.FullHash()
	}

	proofs := merkle.GetMerkleBranch(leaves, uint32(index))

	return proofs
}

//               ,  fullhash
func (chain *BlockChain) getMultiLayerProofs(height int64, blockHash []byte, Txs []*types.Transaction, index int32) []*types.TxProof {
	var proofs []*types.TxProof

	//             merkle ，  FullHash
	if chain.isParaChain {
		txProofs := getTxFullHashProofs(Txs, index)
		proof := types.TxProof{Proofs: txProofs, Index: uint32(index)}
		proofs = append(proofs, &proof)
		return proofs
	}

	//
	title, haveParaTx := types.GetParaExecTitleName(string(Txs[index].Execer))
	if !haveParaTx {
		title = types.MainChainName
	}

	replyparaTxs, err := chain.LoadParaTxByHeight(height, "", 0, 1)
	if err != nil || 0 == len(replyparaTxs.Items) {
		filterlog.Error("getMultiLayerProofs", "height", height, "blockHash", common.ToHex(blockHash), "err", err)
		return nil
	}

	//            index ，         index
	var startIndex int32
	var childIndex uint32
	var hashes [][]byte
	var exist bool
	var childHash []byte
	var txCount int32

	for _, paratx := range replyparaTxs.Items {
		if title == paratx.Title && bytes.Equal(blockHash, paratx.GetHash()) {
			startIndex = paratx.GetStartIndex()
			childIndex = paratx.ChildHashIndex
			childHash = paratx.ChildHash
			txCount = paratx.GetTxCount()
			exist = true
		}
		hashes = append(hashes, paratx.ChildHash)
	}
	//       ,
	if len(replyparaTxs.Items) == 1 && startIndex == 0 && childIndex == 0 {
		txProofs := getTxFullHashProofs(Txs, index)
		proof := types.TxProof{Proofs: txProofs, Index: uint32(index)}
		proofs = append(proofs, &proof)
		return proofs
	}

	endindex := startIndex + txCount
	//
	if !exist || startIndex > index || endindex > int32(len(Txs)) || childIndex >= uint32(len(replyparaTxs.Items)) {
		filterlog.Error("getMultiLayerProofs", "height", height, "blockHash", common.ToHex(blockHash), "exist", exist, "replyparaTxs", replyparaTxs)
		filterlog.Error("getMultiLayerProofs", "startIndex", startIndex, "index", index, "childIndex", childIndex)
		return nil
	}

	//           ,          hash
	var childHashes [][]byte
	for i := startIndex; i < endindex; i++ {
		childHashes = append(childHashes, Txs[i].FullHash())
	}
	txOnChildIndex := index - startIndex

	//
	txOnChildBranch := merkle.GetMerkleBranch(childHashes, uint32(txOnChildIndex))
	txproof := types.TxProof{Proofs: txOnChildBranch, Index: uint32(txOnChildIndex), RootHash: childHash}
	proofs = append(proofs, &txproof)

	//
	childBranch := merkle.GetMerkleBranch(hashes, childIndex)
	childproof := types.TxProof{Proofs: childBranch, Index: childIndex}
	proofs = append(proofs, &childproof)
	return proofs
}
