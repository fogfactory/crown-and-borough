package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
)

// candidateMapCacheSize bounds the cached candidate maps; the oldest one is
// evicted when full, since a lobby is short-lived and regeneration is only
// slow.
const candidateMapCacheSize = 128

// candidatePreview is the map contract plus the territories holding the
// start castles.
type candidatePreview struct {
	mapgen.MapData
	Castles []string `json:"starts"`
}

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

	// Settle the call even if the generator panics, so waiters never hang.
	call.err = errors.New("candidate map generation did not complete")
	defer func() {
		c.mu.Lock()
		delete(c.inflight, key)
		if call.err == nil {
			c.putLocked(key, call.data)
		}
		c.mu.Unlock()
		close(call.done)
	}()
	call.data, call.err = encodeCandidateMap(generate)
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
	// The preview shows where the castles stand, as a sorted set: nothing says
	// which house gets which start, so the draw stays secret.
	starts := append([]string(nil), data.Starts...)
	sort.Strings(starts)
	body, err := json.Marshal(candidatePreview{MapData: data, Castles: starts})
	if err != nil {
		return candidateMap{}, err
	}
	sum := sha256.Sum256(body)
	return candidateMap{body: append(body, '\n'), etag: `"` + hex.EncodeToString(sum[:8]) + `"`}, nil
}
