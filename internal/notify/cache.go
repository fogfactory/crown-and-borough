package notify

import (
	"sync"
	"time"
)

// cacheTTL is how long the notifier trusts what it read about a player: a
// changed setting, or a chat read again, takes effect within that delay.
const cacheTTL = time.Minute

// cachePruneSize is the number of entries above which expired ones are
// dropped on insert, so the cache stays bounded by the recently seen players.
const cachePruneSize = 256

// ttlCache remembers values for cacheTTL. It saves the database reads of
// events that follow each other closely, such as the messages of a
// conversation.
type ttlCache[K comparable, V any] struct {
	mu      sync.Mutex
	entries map[K]cacheEntry[V]
}

type cacheEntry[V any] struct {
	value   V
	expires time.Time
}

func (c *ttlCache[K, V]) get(key K, now time.Time) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || !now.Before(entry.expires) {
		var zero V
		return zero, false
	}
	return entry.value, true
}

func (c *ttlCache[K, V]) put(key K, value V, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[K]cacheEntry[V])
	}
	if len(c.entries) >= cachePruneSize {
		for existing, entry := range c.entries {
			if !now.Before(entry.expires) {
				delete(c.entries, existing)
			}
		}
	}
	c.entries[key] = cacheEntry[V]{value: value, expires: now.Add(cacheTTL)}
}

func (c *ttlCache[K, V]) remove(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}
