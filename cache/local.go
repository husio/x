package cache

import (
	"encoding/json"
	"sync"

	"golang.org/x/net/context"
)

type localCache struct {
	mu  sync.Mutex
	mem map[string][]byte
}

func WithLocalCache(ctx context.Context) context.Context {
	c := &localCache{
		mem: make(map[string][]byte),
	}
	return context.WithValue(ctx, contextKey, c)
}

func (c *localCache) Get(key string, dest interface{}) error {
	c.mu.Lock()
	raw, ok := c.mem[key]
	c.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	return json.Unmarshal(raw, dest)
}

func (c *localCache) Put(key string, src interface{}) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return nil
	}
	c.mu.Lock()
	c.mem[key] = raw
	c.mu.Unlock()
	return nil
}

func (c *localCache) Del(key string) error {
	c.mu.Lock()
	delete(c.mem, key)
	c.mu.Unlock()
	return nil
}
