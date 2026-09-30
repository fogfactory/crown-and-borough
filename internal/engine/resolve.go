package engine

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Resolve resolves supply and every current chain simultaneously. It validates
// and deep clones game before running its phases, so game is never mutated.
func Resolve(game *models.GameState, balance assetgen.Balance) (Resolution, error) {
	return ResolveWithDeckOrders(game, balance, nil)
}

func ResolveWithDeckOrders(game *models.GameState, balance assetgen.Balance, deckOrders map[models.PlayerID][]models.DeckOrder) (Resolution, error) {
	return resolveFromControl(game, balance, deckOrders, nil)
}

// resolveFromControl is ResolveWithDeckOrders starting from an explicit
// control snapshot instead of the one derived from game (nil derives it, which
// is what every production caller wants). The adjudication corpus uses it to
// replay states that no derivation can produce, such as an empty castle still
// held by a player who has left, and so check that the phases resolve them
// exactly as they did when control was stored.
func resolveFromControl(game *models.GameState, balance assetgen.Balance, deckOrders map[models.PlayerID][]models.DeckOrder, startControl map[models.TerritoryID]models.PlayerID) (Resolution, error) {
	if game == nil {
		return Resolution{}, fmt.Errorf("engine: resolve: nil game state")
	}
	if err := game.Validate(); err != nil {
		return Resolution{}, fmt.Errorf("engine: resolve: invalid game state: %w", err)
	}
	if game.Season == models.SeasonWinter {
		return Resolution{}, fmt.Errorf("engine: resolve: winter state must use ResolveWinter")
	}
	if err := validateActionDeckOrders(game, balance, deckOrders); err != nil {
		return Resolution{}, err
	}
	if err := validateStoredChains(game); err != nil {
		return Resolution{}, err
	}
	state := cloneGameState(game)
	ctx := newResolutionContext(state, balance)
	if startControl != nil {
		ctx.startControl = startControl
	}
	revealCurrentAugury(ctx)
	markPendingTaxWindowFiefs(ctx, deckOrders)
	resolveDeckOrders(ctx, deckOrders)
	resolveSeasonEffects(ctx)

	// Chains are attached before Resolve is called; this function only handles
	// the simultaneous resolution core.
	enumerateIntentions(ctx)
	cancelAttackedOriginPeaceful(ctx)
	calculateSupports(ctx)
	if err := resolveContests(ctx); err != nil {
		return Resolution{}, err
	}
	if err := executeMovementsAndRetreats(ctx); err != nil {
		return Resolution{}, err
	}
	resolveRevoltCombats(ctx)
	progressChainsAndControl(ctx)
	// Ravitaillement (territory income, rations, mills, famine) resolves last,
	// on the armies' post-combat, post-movement, post-control positions: an
	// army that flees into a source is fed there this same turn, a village
	// captured this turn already feeds from the moment it changes hands
	// (#208). Its famine penalty is never applied this same turn (an army
	// famished here fights at full strength until next turn, see
	// models.Army.Starving); the auto-pillage it can trigger may dissolve a
	// fief capital's castle, so what that dissolution unanchors is reported as
	// abandoned right after, against the control the turn settled on.
	resolveSupply(ctx)
	ctx.emitAbandonedControl(ctx.settledControl)
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

// validateStoredChains checks that every stored chain passes the static
// validation it was received under. Execution relies on it and only checks
// world conditions, so a chain that does not is a corrupted state.
func validateStoredChains(game *models.GameState) error {
	for _, chain := range game.Chains {
		if validationErrors := orders.ValidateChain(game, chain); len(validationErrors) != 0 {
			return fmt.Errorf("engine: resolve: chain %q is statically invalid: %w", chain.ID, validationErrors[0])
		}
	}
	return nil
}
