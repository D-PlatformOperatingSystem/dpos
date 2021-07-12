package mempool

import (
	"github.com/D-PlatformOperatingSystem/dpos/common/listmap"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//AccountTxIndex
type AccountTxIndex struct {
	maxperaccount int
	accMap        map[string]*listmap.ListMap
}

//NewAccountTxIndex
func NewAccountTxIndex(maxperaccount int) *AccountTxIndex {
	return &AccountTxIndex{
		maxperaccount: maxperaccount,
		accMap:        make(map[string]*listmap.ListMap),
	}
}

// TxNumOfAccount      Mempool
func (cache *AccountTxIndex) TxNumOfAccount(addr string) int {
	if _, ok := cache.accMap[addr]; ok {
		return cache.accMap[addr].Size()
	}
	return 0
}

// GetAccTxs           （  ）
func (cache *AccountTxIndex) GetAccTxs(addrs *types.ReqAddrs) *types.TransactionDetails {
	res := &types.TransactionDetails{}
	for _, addr := range addrs.Addrs {
		if value, ok := cache.accMap[addr]; ok {
			value.Walk(func(val interface{}) bool {
				v := val.(*types.Transaction)
				txAmount, err := v.Amount()
				if err != nil {
					txAmount = 0
				}
				res.Txs = append(res.Txs,
					&types.TransactionDetail{
						Tx:         v,
						Amount:     txAmount,
						Fromaddr:   addr,
						ActionName: v.ActionName(),
					})
				return true
			})
		}
	}
	return res
}

//Remove
func (cache *AccountTxIndex) Remove(tx *types.Transaction) {
	addr := tx.From()
	if lm, ok := cache.accMap[addr]; ok {
		lm.Remove(string(tx.Hash()))
		if lm.Size() == 0 {
			delete(cache.accMap, addr)
		}
	}
}

// Push push transaction to AccountTxIndex
func (cache *AccountTxIndex) Push(tx *types.Transaction) error {
	addr := tx.From()
	_, ok := cache.accMap[addr]
	if !ok {
		cache.accMap[addr] = listmap.New()
	}
	if cache.accMap[addr].Size() >= cache.maxperaccount {
		return types.ErrManyTx
	}
	cache.accMap[addr].Push(string(tx.Hash()), tx)
	return nil
}

//CanPush     push   account index
func (cache *AccountTxIndex) CanPush(tx *types.Transaction) bool {
	if item, ok := cache.accMap[tx.From()]; ok {
		return item.Size() < cache.maxperaccount
	}
	return true
}
