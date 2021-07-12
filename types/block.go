// Copyright D-Platform Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"runtime"
	"sync"

	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/crypto"
	proto "github.com/golang/protobuf/proto"
)

// Hash   block hash
func (block *Block) Hash(cfg *DplatformOSConfig) []byte {
	if cfg.IsFork(block.Height, "ForkBlockHash") {
		return block.HashNew()
	}
	return block.HashOld()
}

//HashByForkHeight hash        fork      hash
func (block *Block) HashByForkHeight(forkheight int64) []byte {
	if block.Height >= forkheight {
		return block.HashNew()
	}
	return block.HashOld()
}

//HashNew     Hash
func (block *Block) HashNew() []byte {
	data, err := proto.Marshal(block.getHeaderHashNew())
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}

//HashOld     hash
func (block *Block) HashOld() []byte {
	data, err := proto.Marshal(block.getHeaderHashOld())
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}

// Size   block Size
func (block *Block) Size() int {
	return Size(block)
}

// GetHeader   block Header
func (block *Block) GetHeader(cfg *DplatformOSConfig) *Header {
	head := &Header{}
	head.Version = block.Version
	head.ParentHash = block.ParentHash
	head.TxHash = block.TxHash
	head.BlockTime = block.BlockTime
	head.Height = block.Height
	head.Difficulty = block.Difficulty
	head.StateHash = block.StateHash
	head.TxCount = int64(len(block.Txs))
	head.Hash = block.Hash(cfg)
	return head
}

func (block *Block) getHeaderHashOld() *Header {
	head := &Header{}
	head.Version = block.Version
	head.ParentHash = block.ParentHash
	head.TxHash = block.TxHash
	head.BlockTime = block.BlockTime
	head.Height = block.Height
	return head
}

func (block *Block) getHeaderHashNew() *Header {
	head := &Header{}
	head.Version = block.Version
	head.ParentHash = block.ParentHash
	head.TxHash = block.TxHash
	head.BlockTime = block.BlockTime
	head.Height = block.Height
	head.Difficulty = block.Difficulty
	head.StateHash = block.StateHash
	head.TxCount = int64(len(block.Txs))
	return head
}

// VerifySignature           ,
func VerifySignature(cfg *DplatformOSConfig, block *Block, txs []*Transaction) bool {
	//
	if !block.verifySignature(cfg) {
		return false
	}
	//
	return verifyTxsSignature(txs)
}

// CheckSign   block   ,
func (block *Block) CheckSign(cfg *DplatformOSConfig) bool {
	return VerifySignature(cfg, block, block.Txs)
}

func (block *Block) verifySignature(cfg *DplatformOSConfig) bool {
	if block.GetSignature() == nil {
		return true
	}
	hash := block.Hash(cfg)
	return CheckSign(hash, "", block.GetSignature())
}

func verifyTxsSignature(txs []*Transaction) bool {

	//          ，
	if len(txs) == 0 {
		return true
	}
	done := make(chan struct{})
	defer close(done)
	taskes := gen(done, txs)
	cpuNum := runtime.NumCPU()
	// Start a fixed number of goroutines to read and digest files.
	c := make(chan result) // HLc
	var wg sync.WaitGroup
	wg.Add(cpuNum)
	for i := 0; i < cpuNum; i++ {
		go func() {
			checksign(done, taskes, c) // HLc
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c) // HLc
	}()
	// End of pipeline. OMIT
	for r := range c {
		if !r.isok {
			return false
		}
	}
	return true
}

func gen(done <-chan struct{}, task []*Transaction) <-chan *Transaction {
	ch := make(chan *Transaction)
	go func() {
		defer func() {
			close(ch)
		}()
		for i := 0; i < len(task); i++ {
			select {
			case ch <- task[i]:
			case <-done:
				return
			}
		}
	}()
	return ch
}

type result struct {
	isok bool
}

func check(data *Transaction) bool {
	return data.CheckSign()
}

func checksign(done <-chan struct{}, taskes <-chan *Transaction, c chan<- result) {
	for task := range taskes {
		select {
		case c <- result{check(task)}:
		case <-done:
			return
		}
	}
}

// CheckSign
func CheckSign(data []byte, execer string, sign *Signature) bool {
	//GetDefaultSign:       ，
	c, err := crypto.New(GetSignName(execer, int(sign.Ty)))
	if err != nil {
		return false
	}
	pub, err := c.PubKeyFromBytes(sign.Pubkey)
	if err != nil {
		return false
	}
	signbytes, err := c.SignatureFromBytes(sign.Signature)
	if err != nil {
		return false
	}
	return pub.VerifyBytes(data, signbytes)
}

//FilterParaTxsByTitle     title
//1，
//2,          ，
//            proof    ，
//               ，     roothash。    roothash proof
func (blockDetail *BlockDetail) FilterParaTxsByTitle(cfg *DplatformOSConfig, title string) *ParaTxDetail {
	var paraTx ParaTxDetail
	paraTx.Header = blockDetail.Block.GetHeader(cfg)

	for i := 0; i < len(blockDetail.Block.Txs); i++ {
		tx := blockDetail.Block.Txs[i]
		if IsSpecificParaExecName(title, string(tx.Execer)) {

			//       para  ，
			if tx.GroupCount >= 2 {
				txDetails, endIdx := blockDetail.filterParaTxGroup(tx, i)
				paraTx.TxDetails = append(paraTx.TxDetails, txDetails...)
				i = endIdx - 1
				continue
			}

			//  para
			var txDetail TxDetail
			txDetail.Tx = tx
			txDetail.Receipt = blockDetail.Receipts[i]
			txDetail.Index = uint32(i)
			paraTx.TxDetails = append(paraTx.TxDetails, &txDetail)

		}
	}
	return &paraTx
}

//filterParaTxGroup   para
func (blockDetail *BlockDetail) filterParaTxGroup(tx *Transaction, index int) ([]*TxDetail, int) {
	var headIdx int
	var txDetails []*TxDetail

	for i := index; i >= 0; i-- {
		if bytes.Equal(tx.Header, blockDetail.Block.Txs[i].Hash()) {
			headIdx = i
			break
		}
	}

	endIdx := headIdx + int(tx.GroupCount)
	for i := headIdx; i < endIdx; i++ {
		var txDetail TxDetail
		txDetail.Tx = blockDetail.Block.Txs[i]
		txDetail.Receipt = blockDetail.Receipts[i]
		txDetail.Index = uint32(i)
		txDetails = append(txDetails, &txDetail)
	}
	return txDetails, endIdx
}

// Size   blockDetail Size
func (blockDetail *BlockDetail) Size() int {
	return Size(blockDetail)
}

// Size   header Size
func (header *Header) Size() int {
	return Size(header)
}

// Size   paraTxDetail Size
func (paraTxDetail *ParaTxDetail) Size() int {
	return Size(paraTxDetail)
}
