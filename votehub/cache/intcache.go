package cache

import (
	"sync"

	"golang.org/x/net/context"
)

type IntegerCache struct {
	mu  sync.Mutex
	mem map[string]int64
}

func WithIntCache(ctx context.Context) context.Context {
	c := &IntegerCache{
		mem: make(map[string]int64),
	}
	return context.WithValue(ctx, "cache:int", c)
}

func IntCache(ctx context.Context) *IntegerCache {
	c := ctx.Value("cache:int")
	if c == nil {
		panic("IntCache not present in context")
	}
	return c.(*IntegerCache)
}

func (c *IntegerCache) Put(key string, val int64) {
	c.mu.Lock()
	c.mem[key] = val
	c.mu.Unlock()
}

func (c *IntegerCache) Get(key string) (int64, bool) {
	c.mu.Lock()
	val, ok := c.mem[key]
	c.mu.Unlock()
	if !ok {
		return 0, false
	}
	return val, true
}

func (c *IntegerCache) Del(key string) {
	c.mu.Lock()
	delete(c.mem, key)
	c.mu.Unlock()
}

func (c *IntegerCache) Len() int {
	c.mu.Lock()
	size := len(c.mem)
	c.mu.Unlock()
	return size
}
