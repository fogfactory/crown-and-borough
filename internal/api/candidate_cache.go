package api

import (
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
)

// candidateMapCacheSize bounds the cached candidate maps; the cache is reset
// when full, since a lobby is short-lived and regeneration is only slow.
const candidateMapCacheSize = 128

// candidateMapCache keeps the generated maps of seed candidates by game and
// candidate index.
type candidateMapCache struct {
	mu    sync.Mutex
	items map[string]mapgen.MapData
}

func newCandidateMapCache() *candidateMapCache {
	return &candidateMapCache{items: make(map[string]mapgen.MapData)}
}

func (c *candidateMapCache) get(key string) (mapgen.MapData, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, ok := c.items[key]
	return data, ok
}

func (c *candidateMapCache) put(key string, data mapgen.MapData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= candidateMapCacheSize {
		c.items = make(map[string]mapgen.MapData)
	}
	c.items[key] = data
}
