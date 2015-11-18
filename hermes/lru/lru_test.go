package lru

import (
	"fmt"
	"testing"
)

func TestLRUGetSet(t *testing.T) {
	lru := NewLRU(0)
	lru.Set("a", "first")
	lru.Set("b", "second")

	if v, ok := lru.Get("a"); !ok || v.(string) != "first" {
		t.Errorf("expected \"first\", got %v", v)
	}
	if v, ok := lru.Get("b"); !ok || v.(string) != "second" {
		t.Errorf("expected \"second\", got %v", v)
	}
	if v, ok := lru.Get("x"); ok {
		t.Errorf("unexpected value: %v", v)
	}
}

func TestLRUExpiration(t *testing.T) {
	lru := NewLRU(2)
	lru.Set("a", "first")
	lru.Set("b", "second")
	lru.Set("c", "third")

	if v, ok := lru.Get("a"); ok {
		t.Errorf("unexpected value: %v", v)
	}
	if v, ok := lru.Get("b"); !ok || v.(string) != "second" {
		t.Errorf("expected \"second\", got %v", v)
	}
	if v, ok := lru.Get("c"); !ok || v.(string) != "third" {
		t.Errorf("expected \"first\", got %v", v)
	}

	lru.Set("a", "first")
	lru.Set("b", "second")

	if v, ok := lru.Get("a"); !ok || v.(string) != "first" {
		t.Errorf("expected \"first\", got %v", v)
	}
	if v, ok := lru.Get("b"); !ok || v.(string) != "second" {
		t.Errorf("expected \"second\", got %v", v)
	}
	if v, ok := lru.Get("c"); ok {
		t.Errorf("unexpected value: %v", v)
	}

	lru.Get("a") // get push to the top of cache
	lru.Set("c", "third")

	if v, ok := lru.Get("a"); !ok || v.(string) != "first" {
		t.Errorf("expected \"first\", got %v", v)
	}
	if v, ok := lru.Get("b"); ok {
		t.Errorf("unexpected value: %v", v)
	}
	if v, ok := lru.Get("c"); !ok || v.(string) != "third" {
		t.Errorf("expected \"second\", got %v", v)
	}
}

func TestLRUPop(t *testing.T) {
	lru := NewLRU(2)
	lru.Set("a", "first")
	lru.Set("b", "second")

	if v, ok := lru.Pop("a"); !ok || v.(string) != "first" {
		t.Errorf("expected \"first\", got %v", v)
	}
	if v, ok := lru.Pop("a"); ok {
		t.Errorf("unexpected value: %v", v)
	}

	if v, ok := lru.Pop("b"); !ok || v.(string) != "second" {
		t.Errorf("expected \"second\", got %v", v)
	}
	if v, ok := lru.Pop("b"); ok {
		t.Errorf("unexpected value: %v", v)
	}
}

func BenchmarkSet_Size0(b *testing.B)     { benchSet(b, 0) }
func BenchmarkSet_Size10(b *testing.B)    { benchSet(b, 10) }
func BenchmarkSet_Size100(b *testing.B)   { benchSet(b, 500) }
func BenchmarkSet_Size10000(b *testing.B) { benchSet(b, 10000) }

func benchSet(b *testing.B, size int) {
	lru := NewLRU(size)

	keys := make([]string, size+10)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		lru.Set(key, "foobar")
	}
}
