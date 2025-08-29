package pokecache

import (
	"testing"
	"time"
)

func TestCacheAddGetAndExpiry(t *testing.T) {
	cache := NewCache(50 * time.Millisecond)

	key := "foo"
	value := []byte("bar")

	cache.Add(key, value)

	// Retrieve immediately
	got, ok := cache.Get(key)
	if !ok {
		t.Fatalf("expected key %s to exist", key)
	}
	if string(got) != "bar" {
		t.Errorf("expected value %q, got %q", "bar", string(got))
	}

	// Wait for it to expire
	time.Sleep(60 * time.Millisecond)

	_, ok = cache.Get(key)
	if ok {
		t.Errorf("expected key %s to be expired", key)
	}
}
