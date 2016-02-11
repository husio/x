package cache

import (
	"container/list"
	"encoding/json"
	"sync"

	"golang.org/x/net/context"
)

type localCache struct {
	mu      sync.Mutex
	maxsize int
	idx     map[string]*item
	order   *list.List
}

type item struct {
	key string
	val []byte
	el  *list.Element
}

func WithLocalCache(ctx context.Context, maxsize int) context.Context {
	c := newLocalCache(maxsize)
	return context.WithValue(ctx, contextKey, c)
}

func newLocalCache(maxsize int) *localCache {
	return &localCache{
		idx:     make(map[string]*item, maxsize*2),
		maxsize: maxsize,
		order:   list.New(),
	}
}

func (c *localCache) Get(key string, dest interface{}) error {
	c.mu.Lock()
	it, ok := c.idx[key]
	if ok {
		it.el = c.order.PushFront(it.el)
	}
	c.mu.Unlock()

	if !ok {
		return ErrNotFound
	}
	return json.Unmarshal(it.val, dest)
}

func (c *localCache) Put(key string, src interface{}) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return nil
	}
	it := &item{key: key, val: raw}
	c.mu.Lock()
	c.idx[it.key] = it
	it.el = c.order.PushFront(it)
	for len(c.idx) > c.maxsize {
		last := c.order.Back().Value.(*item)
		c.order.Remove(last.el)
		delete(c.idx, last.key)
	}
	c.mu.Unlock()
	return nil
}

func (c *localCache) Del(key string) error {
	c.mu.Lock()
	if it, ok := c.idx[key]; ok {
		delete(c.idx, key)
		c.order.Remove(it.el)
	}
	c.mu.Unlock()
	return nil
}
