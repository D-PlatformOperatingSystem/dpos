package skiplist

import (
	"container/list"

	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//Scorer      Value  Score
type Scorer interface {
	GetScore() int64
	Hash() []byte
	// score
	Compare(Scorer) int
	//
	ByteSize() int64
}

// Queue skiplist         score      ，score
type Queue struct {
	txMap      map[string]*list.Element
	txList     *SkipList
	maxsize    int64
	cacheBytes int64
}

// NewQueue
func NewQueue(maxsize int64) *Queue {
	return &Queue{
		txMap:   make(map[string]*list.Element),
		txList:  NewSkipList(&SkipValue{Score: -1, Value: nil}),
		maxsize: maxsize,
	}
}

/*
       Score    ，           Score
*/
func (cache *Queue) insertSkipValue(item Scorer) *list.Element {
	skvalue := cache.CreateSkipValue(item)
	value := cache.txList.Find(skvalue)
	var txlist *list.List
	if value == nil {
		txlist = list.New()
		skvalue.Value = txlist
		cache.txList.Insert(skvalue)
	} else {
		txlist = value.Value.(*list.List)
	}
	return txlist.PushBack(item)
}

func (cache *Queue) deleteSkipValue(item *list.Element) error {
	if item == nil {
		return nil
	}
	skvalue := cache.CreateSkipValue(item.Value.(Scorer))
	value := cache.txList.Find(skvalue)
	var txlist *list.List
	if value == nil {
		return types.ErrNotFound
	}
	txlist = value.Value.(*list.List)
	txlist.Remove(item)
	if txlist.Len() == 0 {
		cache.txList.Delete(value)
	}
	return nil
}

//CreateSkipValue          score  Value
func (cache *Queue) CreateSkipValue(item Scorer) *SkipValue {
	skvalue := &SkipValue{Score: item.GetScore()}
	return skvalue
}

//MaxSize    cache
func (cache *Queue) MaxSize() int64 {
	return cache.maxsize
}

//Exist
func (cache *Queue) Exist(hash string) bool {
	_, exists := cache.txMap[hash]
	return exists
}

//GetItem        key
func (cache *Queue) GetItem(hash string) (Scorer, error) {
	if k, exist := cache.txMap[hash]; exist {
		return k.Value.(Scorer), nil
	}
	return nil, types.ErrNotFound
}

//GetCacheBytes get cache byte size
func (cache *Queue) GetCacheBytes() int64 {
	return cache.cacheBytes
}

//Insert Scorer item to queue
func (cache *Queue) Insert(hash string, item Scorer) {
	cache.cacheBytes += item.ByteSize()
	cache.txMap[hash] = cache.insertSkipValue(item)
}

// Push item     ，                ，          ，        ，
func (cache *Queue) Push(item Scorer) error {
	hash := item.Hash()
	if cache.Exist(string(hash)) {
		return types.ErrTxExist
	}
	sv := cache.CreateSkipValue(item)
	if int64(cache.Size()) >= cache.maxsize {
		tail := cache.Last()
		lasthash := string(tail.Hash())
		cmp := sv.Compare(cache.CreateSkipValue(tail))
		if cmp == Big || (cmp == Equal && item.Compare(tail) == Big) {
			err := cache.Remove(lasthash)
			if err != nil {
				return err
			}
		} else {
			return types.ErrMemFull
		}
	}
	cache.Insert(string(hash), item)
	return nil
}

// Remove
func (cache *Queue) Remove(hash string) error {
	elm, ok := cache.txMap[hash]
	if !ok {
		return types.ErrNotFound
	}
	//  txMap    ，
	delete(cache.txMap, hash)
	err := cache.deleteSkipValue(elm)
	if err != nil {
		println("queue_data_crash")
		return err
	}
	cache.cacheBytes -= elm.Value.(Scorer).ByteSize()
	return nil
}

// Size
func (cache *Queue) Size() int {
	return len(cache.txMap)
}

//Last
func (cache *Queue) Last() Scorer {
	if cache.Size() == 0 {
		return nil
	}
	tailqueue := cache.txList.GetIterator().Last()
	tail := tailqueue.Value.(*list.List).Back().Value.(Scorer)
	return tail
}

//First
func (cache *Queue) First() Scorer {
	if cache.Size() == 0 {
		return nil
	}
	tailqueue := cache.txList.GetIterator().First()
	tail := tailqueue.Value.(*list.List).Front().Value.(Scorer)
	return tail
}

// Walk
func (cache *Queue) Walk(count int, cb func(value Scorer) bool) {
	i := 0
	cache.txList.Walk(func(item interface{}) bool {
		l := item.(*list.List)
		for e := l.Front(); e != nil; e = e.Next() {
			if !cb(e.Value.(Scorer)) {
				return false
			}
			i++
			if i == count {
				return false
			}
		}
		return true
	})
}
