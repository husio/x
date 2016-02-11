package cache

import "testing"

func TestLocalCache(t *testing.T) {
	c := newLocalCache(2)

	must(c.Put("a", 1))
	must(c.Put("b", 2))

	var val int
	must(c.Get("a", &val))
	if val != 1 {
		t.Errorf("want 1, got %d", val)
	}
	must(c.Get("b", &val))
	if val != 2 {
		t.Errorf("want 2, got %d", val)
	}

	// setting thrid should throw 'a' out of cache
	must(c.Put("c", 3))
	must(c.Get("c", &val))
	if val != 3 {
		t.Errorf("want 3, got %d", val)
	}

	if err := c.Get("a", &val); err != ErrNotFound {
		t.Errorf("want error, got %v", err)
	}

	// getting b should prevent from expiring
	must(c.Get("b", &val))
	// putting new value should expire 'c' - 'b' was refreshed
	must(c.Put("d", 4))

	if err := c.Get("c", &val); err != ErrNotFound {
		t.Errorf("want error, got %v", err)
	}
	must(c.Get("b", &val))
	must(c.Get("d", &val))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
