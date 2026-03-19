package cacheutil

import "testing"

func TestBoundedCache(t *testing.T) {
	t.Parallel()

	cache := NewBoundedCache[int](2)
	cache.Put("a", 1)
	cache.Put("b", 2)

	if got, ok := cache.Get("a"); !ok || got != 1 {
		t.Fatalf("Get(a) = %v, %v", got, ok)
	}

	cache.Put("c", 3)
	if _, ok := cache.Get("a"); ok {
		t.Fatal("expected oldest item to be evicted")
	}

	cache.Put("b", 20)
	if got, ok := cache.Get("b"); !ok || got != 20 {
		t.Fatalf("Get(b) = %v, %v", got, ok)
	}

	if cache.Len() != 2 {
		t.Fatalf("Len() = %d", cache.Len())
	}
}
