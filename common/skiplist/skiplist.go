package skiplist

import (
	"fmt"
	"math/rand"
)

const maxLevel = 32
const prob = 0.35

// SkipValue       Value
type SkipValue struct {
	Score int64
	Value interface{}
}

//Compare Const
const (
	Big   = -1
	Small = 1
	Equal = 0
)

// Compare     ,
func (v *SkipValue) Compare(value *SkipValue) int {
	if v.Score > value.Score {
		return Big
	} else if v.Score == value.Score {
		return Equal
	}
	return Small
}

// skipListNode
type skipListNode struct {
	next  []*skipListNode
	prev  *skipListNode
	Value *SkipValue
}

// SkipList
type SkipList struct {
	header, tail *skipListNode
	findcount    int
	count        int
	level        int
}

// Iterator
type Iterator struct {
	list *SkipList
	node *skipListNode
}

// First        Value
func (sli *Iterator) First() *SkipValue {
	if sli.list.header.next[0] == nil {
		return nil
	}
	sli.node = sli.list.header.next[0]
	return sli.node.Value
}

// Last         Value
func (sli *Iterator) Last() *SkipValue {
	if sli.list.tail == nil {
		return nil
	}
	sli.node = sli.list.tail
	return sli.node.Value
}

// Prev
func (sli *Iterator) Prev() *Iterator {
	sli.node = sli.node.Prev()
	return sli
}

// Next
func (sli *Iterator) Next() *Iterator {
	sli.node = sli.node.Next()
	return sli
}

// Value         Value
func (sli *Iterator) Value() *SkipValue {
	return sli.node.Value
}

// Prev
func (node *skipListNode) Prev() *skipListNode {
	if node == nil || node.prev == nil {
		return nil
	}
	return node.prev
}

// Next
func (node *skipListNode) Next() *skipListNode {
	if node == nil || node.next[0] == nil {
		return nil
	}
	return node.next[0]
}

// Seek                            SkipValue
func (sli *Iterator) Seek(value *SkipValue) *SkipValue {
	x := sli.list.find(value)
	if x.next[0] == nil {
		return nil
	}
	sli.node = x.next[0]
	return sli.node.Value
}

func newskipListNode(level int, value *SkipValue) *skipListNode {
	node := &skipListNode{}
	node.next = make([]*skipListNode, level)
	node.Value = value
	return node
}

//NewSkipList
func NewSkipList(min *SkipValue) *SkipList {
	sl := &SkipList{}
	sl.level = 1
	sl.header = newskipListNode(maxLevel, min)
	return sl
}

func randomLevel() int {
	level := 1
	t := prob * 0xFFFF
	// #nosec
	for rand.Int()&0xFFFF < int(t) {
		level++
		if level == maxLevel {
			break
		}
	}
	return level
}

// GetIterator
func (sl *SkipList) GetIterator() *Iterator {
	it := &Iterator{}
	it.list = sl
	it.First()
	return it
}

// Len
func (sl *SkipList) Len() int {
	return sl.count
}

// Level
func (sl *SkipList) Level() int {
	return sl.level
}

func (sl *SkipList) find(value *SkipValue) *skipListNode {
	x := sl.header
	for i := sl.level - 1; i >= 0; i-- {
		for x.next[i] != nil && x.next[i].Value.Compare(value) < 0 {
			sl.findcount++
			x = x.next[i]
		}
	}
	return x
}

// FindCount
func (sl *SkipList) FindCount() int {
	return sl.findcount
}

// Find          SkipValue
func (sl *SkipList) Find(value *SkipValue) *SkipValue {
	x := sl.find(value)
	if x.next[0] != nil && x.next[0].Value.Compare(value) == 0 {
		return x.next[0].Value
	}
	return nil
}

// FindGreaterOrEqual                         SkipValue
func (sl *SkipList) FindGreaterOrEqual(value *SkipValue) *SkipValue {
	x := sl.find(value)
	if x.next[0] != nil {
		return x.next[0].Value
	}
	return nil
}

// Insert
func (sl *SkipList) Insert(value *SkipValue) int {
	var update [maxLevel]*skipListNode
	x := sl.header
	for i := sl.level - 1; i >= 0; i-- {
		for x.next[i] != nil && x.next[i].Value.Compare(value) <= 0 {
			x = x.next[i]
		}
		update[i] = x
	}
	//if x.next[0] != nil && x.next[0].Value.Compare(value) == 0 { //update
	//	x.next[0].Value = value
	//	return 0
	//}
	level := randomLevel()
	if level > sl.level {
		for i := sl.level; i < level; i++ {
			update[i] = sl.header
		}
		sl.level = level
	}
	x = newskipListNode(level, value)
	for i := 0; i < level; i++ {
		x.next[i] = update[i].next[i]
		update[i].next[i] = x
	}
	//
	if update[0] != sl.header {
		x.prev = update[0]
	}
	if x.next[0] != nil {
		x.next[0].prev = x
	} else {
		sl.tail = x
	}
	sl.count++
	return 1
}

// Delete
func (sl *SkipList) Delete(value *SkipValue) int {
	if value == nil {
		return 0
	}
	var update [maxLevel]*skipListNode
	x := sl.header
	for i := sl.level - 1; i >= 0; i-- {
		for x.next[i] != nil && x.next[i].Value.Compare(value) < 0 {
			x = x.next[i]
		}
		update[i] = x
	}
	if x.next[0] == nil || x.next[0].Value.Compare(value) != 0 { //not find
		return 0
	}
	x = x.next[0]
	for i := 0; i < sl.level; i++ {
		if update[i].next[i] == x {
			update[i].next[i] = x.next[i]
		}
	}
	if x.next[0] != nil {
		x.next[0].prev = x.prev
	} else {
		sl.tail = x.prev
	}
	for sl.level > 1 && sl.header.next[sl.level-1] == nil {
		sl.level--
	}
	sl.count--
	return 1
}

// Print
func (sl *SkipList) Print() {
	if sl.count > 0 {
		r := sl.header
		for i := sl.level - 1; i >= 0; i-- {
			e := r.next[i]
			//fmt.Print(i)
			for e != nil {
				fmt.Print(e.Value.Score)
				fmt.Print("    ")
				fmt.Print(e.Value)
				fmt.Println("")
				e = e.next[i]
			}
			fmt.Println()
		}
	} else {
		fmt.Println(" ")
	}
}

//Walk        SkipValue Value,  cb   false
func (sl *SkipList) Walk(cb func(value interface{}) bool) {
	for e := sl.header.Next(); e != nil; e = e.Next() {
		if cb == nil {
			return
		}
		if !cb(e.Value.Value) {
			return
		}
	}
}

//WalkS         SkipValue,  cb   false
func (sl *SkipList) WalkS(cb func(value interface{}) bool) {
	for e := sl.header.Next(); e != nil; e = e.Next() {
		if cb == nil {
			return
		}
		if !cb(e.Value) {
			return
		}
	}
}
