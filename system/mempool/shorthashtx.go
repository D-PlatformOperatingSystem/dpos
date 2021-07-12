package mempool

import (
	"github.com/D-PlatformOperatingSystem/dpos/common"
	"github.com/D-PlatformOperatingSystem/dpos/common/listmap"
	log "github.com/D-PlatformOperatingSystem/dpos/common/log/log15"
	"github.com/D-PlatformOperatingSystem/dpos/types"
)

var shashlog = log.New("module", "mempool.shash")

//SHashTxCache   shorthash
type SHashTxCache struct {
	max int
	l   *listmap.ListMap
}

//NewSHashTxCache      hash   cache
func NewSHashTxCache(size int) *SHashTxCache {
	return &SHashTxCache{
		max: size,
		l:   listmap.New(),
	}
}

//GetSHashTxCache   shorthash   tx
func (cache *SHashTxCache) GetSHashTxCache(sHash string) *types.Transaction {
	tx, err := cache.l.GetItem(sHash)
	if err != nil {
		return nil
	}
	return tx.(*types.Transaction)

}

//Remove remove tx of SHashTxCache
func (cache *SHashTxCache) Remove(tx *types.Transaction) {
	txhash := tx.Hash()
	cache.l.Remove(types.CalcTxShortHash(txhash))
	//shashlog.Debug("SHashTxCache:Remove", "shash", types.CalcTxShortHash(txhash), "txhash", common.ToHex(txhash))
}

//Push tx into SHashTxCache
func (cache *SHashTxCache) Push(tx *types.Transaction) {
	shash := types.CalcTxShortHash(tx.Hash())

	if cache.Exist(shash) {
		shashlog.Error("SHashTxCache:Push:Exist", "oldhash", common.ToHex(cache.GetSHashTxCache(shash).Hash()), "newhash", common.ToHex(tx.Hash()))
		return
	}
	if cache.l.Size() >= cache.max {
		shashlog.Error("SHashTxCache:Push:ErrMemFull", "cache.l.Size()", cache.l.Size(), "cache.max", cache.max)
		return
	}
	cache.l.Push(shash, tx)
	//shashlog.Debug("SHashTxCache:Push", "shash", shash, "txhash", common.ToHex(tx.Hash()))
}

//Exist
func (cache *SHashTxCache) Exist(shash string) bool {
	return cache.l.Exist(shash)
}
