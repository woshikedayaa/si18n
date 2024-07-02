package si18n

import (
	"testing"
)

func TestLRUCacheUpdate(t *testing.T) {
	ks := map[string]string{
		"a.b.c.1": "abc1",
		"a.b.c.2": "abc2",
		"a.b.c.3": "abc3",
	}
	cache := newLRUCache(len(ks) * 2)
	for k, v := range ks {
		cache.Put(k, NewMessage(k, v))
	}
	for k, v := range ks {
		get, ok := cache.Get(k)
		equals(true, ok, t)
		equals(v, get.val, t)
	}
}

func TestLRUCacheGet(t *testing.T) {
	cache := newLRUCache(16)
	get, ok := cache.Get("empty")
	equals(false, ok, t)
	// equals("", get.val, t)
	cache.Put("key", NewMessage("key", "value"))
	get, ok = cache.Get("key")
	equals("value", get.val, t)
	equals(true, ok, t)
	get, ok = cache.Get("empty")
	equals(false, ok, t)
	//equals("", get.val, t)
	equals(1, cache.Len(), t)
}
