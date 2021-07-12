package listmap

import (
	"container/list"

	"github.com/D-PlatformOperatingSystem/dpos/types"
)

//ListMap list   map
type ListMap struct {
	m map[string]*list.Element
	l *list.List
}

//New
func New() *ListMap {
	return &ListMap{
		m: make(map[string]*list.Element),
		l: list.New(),
	}
}

//Size     item
func (lm *ListMap) Size() int {
	return len(lm.m)
}

//Exist
func (lm *ListMap) Exist(key string) bool {
	_, ok := lm.m[key]
	return ok
}

//GetItem   key      item
func (lm *ListMap) GetItem(key string) (interface{}, error) {
	item, ok := lm.m[key]
	if !ok {
		return nil, types.ErrNotFound
	}
	return item.Value, nil
}

//Push
func (lm *ListMap) Push(key string, value interface{}) {
	if elm, ok := lm.m[key]; ok {
		elm.Value = value
		return
	}
	elm := lm.l.PushBack(value)
	lm.m[key] = elm
}

//GetTop
func (lm *ListMap) GetTop() interface{} {
	elm := lm.l.Front()
	if elm == nil {
		return nil
	}
	return elm.Value
}

//Remove     key
func (lm *ListMap) Remove(key string) interface{} {
	if elm, ok := lm.m[key]; ok {
		value := lm.l.Remove(elm)
		delete(lm.m, key)
		return value
	}
	return nil
}

//Walk       ，  cb   false
func (lm *ListMap) Walk(cb func(value interface{}) bool) {
	for e := lm.l.Front(); e != nil; e = e.Next() {
		if cb == nil {
			return
		}
		if !cb(e.Value) {
			return
		}
	}
}
