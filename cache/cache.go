package cache

import (
	"errors"

	"golang.org/x/net/context"
)

var ErrNotFound = errors.New("not found")

const contextKey = "cache"

type Cache interface {
	Get(key string, dest interface{}) error
	Put(key string, src interface{}) error
	Del(key string) error
}

func Get(ctx context.Context) Cache {
	c := ctx.Value(contextKey)
	if c == nil {
		panic("cache not present in context")
	}
	return c.(Cache)
}
