package engine

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Resolve resolves supply and every current chain simultaneously. It validates
// and deep clones game before running its phases, so game is never mutated.
func Resolve(game *models.GameState, balance assetgen.Balance) (Resolution, error) {
	if game == nil {
		return Resolution{}, fmt.Errorf("engine: resolve: nil game state")
	}
	if err := game.Validate(); err != nil {
		return Resolution{}, fmt.Errorf("engine: resolve: invalid game state: %w", err)
	}
	if game.Season == models.SeasonWinter {
		return Resolution{}, fmt.Errorf("engine: resolve: winter state must use ResolveWinter")
	}
	state := cloneGameState(game)
	ctx := newResolutionContext(state, balance)

	resolveSupply(ctx)
	// Chains are attached before Resolve is called; this function only handles
	// the simultaneous resolution core.
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
