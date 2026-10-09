package store

import "github.com/fogfactory/crown-and-borough/internal/models"

// DemoForge mutates the state of every game of the store (temporary demo helper, never committed).
func (s *MemoryStore) DemoForge(fn func(*models.GameState)) error {
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
