package lru

import "container/list"

type LRU struct {
	idx   map[string]*item
	order *list.List
	size  int
}

type item struct {
	key   string
	value interface{}
	el    *list.Element
}

func NewLRU(size int) *LRU {
	return &LRU{
		idx:   make(map[string]*item),
		order: list.New(),
		size:  size,
	}
}

func (lru *LRU) Get(key string) (interface{}, bool) {
	it, ok := lru.idx[key]
	if !ok {
		return nil, false
	}
	lru.order.MoveToFront(it.el)
	return it.value, true
}

func (lru *LRU) Set(key string, value interface{}) {
	if it, ok := lru.idx[key]; ok {
		it.value = value
		lru.order.MoveToFront(it.el)
		return
	}

	it := &item{
		key:   key,
		value: value,
	}
	it.el = lru.order.PushFront(it)
	lru.idx[key] = it

	// expire old
	for lru.size > 0 && len(lru.idx) > lru.size {
		it := lru.order.Remove(lru.order.Back()).(*item)
		delete(lru.idx, it.key)
	}
}

func (lru *LRU) Pop(key string) (interface{}, bool) {
	it, ok := lru.idx[key]
	if !ok {
		return nil, false
	}

	lru.order.Remove(it.el)
	delete(lru.idx, it.key)

	return it.value, true
}

func (lru *LRU) Len() int {
	return len(lru.idx)
}
