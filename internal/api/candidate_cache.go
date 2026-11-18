package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
)

// candidateMapCacheSize bounds the cached candidate maps; the oldest one is
// evicted when full, since a lobby is short-lived and regeneration is only
// slow.
const candidateMapCacheSize = 128

// candidateMap is a generated candidate map, kept as the JSON it is served as
// with the ETag of that body.
type candidateMap struct {
	body []byte
	etag string
}

// candidateMapCache keeps the generated maps of seed candidates by game and
// candidate index, and makes concurrent requests for the same missing map
// share one generation.
type candidateMapCache struct {
	mu       sync.Mutex
	items    map[string]candidateMap
	order    []string // keys of items, oldest first
	inflight map[string]*candidateMapCall
}

// candidateMapCall is a generation in progress; done is closed once data and
// err are set.
type candidateMapCall struct {
	done chan struct{}
	data candidateMap
	err  error
}

func newCandidateMapCache() *candidateMapCache {
	return &candidateMapCache{
		items:    make(map[string]candidateMap),
		inflight: make(map[string]*candidateMapCall),
	}
}

// get returns the map of key, generating it with generate when it is neither
// cached nor being generated. A failed generation is not cached.
func (c *candidateMapCache) get(key string, generate func() (mapgen.MapData, error)) (candidateMap, error) {
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

	call.data, call.err = encodeCandidateMap(generate)

	c.mu.Lock()
	delete(c.inflight, key)
	if call.err == nil {
		c.putLocked(key, call.data)
	}
	c.mu.Unlock()
	close(call.done)
	return call.data, call.err
}

func (c *candidateMapCache) putLocked(key string, data candidateMap) {
	if len(c.order) >= candidateMapCacheSize {
		delete(c.items, c.order[0])
		c.order = c.order[1:]
	}
	c.items[key] = data
	c.order = append(c.order, key)
}

// encodeCandidateMap generates a map and marshals it once, for every request
// that follows.
func encodeCandidateMap(generate func() (mapgen.MapData, error)) (candidateMap, error) {
	data, err := generate()
	if err != nil {
		return candidateMap{}, err
	}
	body, err := json.Marshal(data)
	if err != nil {
		return candidateMap{}, err
	}
	sum := sha256.Sum256(body)
	return candidateMap{body: append(body, '\n'), etag: `"` + hex.EncodeToString(sum[:8]) + `"`}, nil
}
