package engine

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Resolve resolves every current chain simultaneously. It validates and deep
// clones game before running the five P1.4 phases, so game is never mutated.
func Resolve(game *models.GameState) (Resolution, error) {
	if game == nil {
		return Resolution{}, fmt.Errorf("engine: resolve: nil game state")
	}
	if err := game.Validate(); err != nil {
		return Resolution{}, fmt.Errorf("engine: resolve: invalid game state: %w", err)
	}
	state := cloneGameState(game)
	ctx := newResolutionContext(state)

	// P1.5 supply will run before this phase; P2.3 can prepare delivered chains
	// before Resolve is called without changing this simultaneous core.
	enumerateIntentions(ctx)
	calculateSupports(ctx)
	if err := resolveContests(ctx); err != nil {
		return Resolution{}, err
	}
	if err := executeMovementsAndRetreats(ctx); err != nil {
		return Resolution{}, err
	}
	progressChainsAndControl(ctx)
	if err := ctx.rebuildOccupancy(); err != nil {
		return Resolution{}, err
	}
	if err := state.Validate(); err != nil {
		return Resolution{}, fmt.Errorf("engine: resolve: invalid result: %w", err)
	}
	return Resolution{
		State:  state,
		Events: append([]Event(nil), ctx.events...),
	}, nil
}
