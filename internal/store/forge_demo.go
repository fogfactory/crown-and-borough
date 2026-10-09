//go:build demo

package store

import "github.com/fogfactory/crown-and-borough/internal/models"

// Forge mutates the state of every game held by the memory store, then checks
// the result is still valid. It exists only in demo builds (-tags demo): the
// forged-state scenarios of cmd/server use it to start a game in a situation
// that would take many turns to reach (scripts/demo.sh).
func (s *MemoryStore) Forge(fn func(*models.GameState)) error {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	for _, game := range s.games {
		game.mu.Lock()
		fn(game.state)
		err := game.state.Validate()
		game.mu.Unlock()
		if err != nil {
			return err
		}
	}
	return nil
}
