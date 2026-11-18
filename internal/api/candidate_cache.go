package api

import (
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
)

// candidateMapCacheSize bounds the cached candidate maps; the oldest one is
// evicted when full, since a lobby is short-lived and regeneration is only
// slow.
const candidateMapCacheSize = 128

// candidateMapCache keeps the generated maps of seed candidates by game and
// candidate index, and makes concurrent requests for the same missing map
// share one generation.
type candidateMapCache struct {
	mu       sync.Mutex
	items    map[string]mapgen.MapData
	order    []string // keys of items, oldest first
	inflight map[string]*candidateMapCall
}

// candidateMapCall is a generation in progress; done is closed once data and
// err are set.
type candidateMapCall struct {
	done chan struct{}
	data mapgen.MapData
	err  error
}

func newCandidateMapCache() *candidateMapCache {
	return &candidateMapCache{
		items:    make(map[string]mapgen.MapData),
		inflight: make(map[string]*candidateMapCall),
	}
}

// get returns the map of key, generating it with generate when it is neither
// cached nor being generated. A failed generation is not cached.
func (c *candidateMapCache) get(key string, generate func() (mapgen.MapData, error)) (mapgen.MapData, error) {
	c.mu.Lock()
	if data, ok := c.items[key]; ok {
		c.mu.Unlock()
		return data, nil
	}
	if call, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		<-call.done
		return call.data, call.err
	}
	call := &candidateMapCall{done: make(chan struct{})}
	c.inflight[key] = call
	c.mu.Unlock()

	call.data, call.err = generate()

	c.mu.Lock()
	delete(c.inflight, key)
	if call.err == nil {
		c.putLocked(key, call.data)
	}
	c.mu.Unlock()
	close(call.done)
	return call.data, call.err
}

func (c *candidateMapCache) putLocked(key string, data mapgen.MapData) {
	if len(c.order) >= candidateMapCacheSize {
		delete(c.items, c.order[0])
		c.order = c.order[1:]
	}
	c.items[key] = data
	c.order = append(c.order, key)
}
