package pokecache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	cache := NewCache(2 * time.Second)
	key := "test-key"
	val := []byte("test-value")

	cache.Add(key, val)

	if result, found := cache.Get(key); !found || string(result) != "test-value" {
		t.Fatalf("expected to find key %s with value %s", key, val)
	}

	time.Sleep(3 * time.Second)

	if _, found := cache.Get(key); found {
		t.Fatalf("expected key %s to be reaped from the cache", key)
	}
}
