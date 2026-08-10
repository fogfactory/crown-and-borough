package orders

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// AssignChain validates current reception conditions and atomically replaces
// the receiving army's chain. Callers must reject a chain returned by
// ParseChain when that call returned any ParseError before calling AssignChain.
// Static non-adjacency diagnostics are deferrable: reception keeps the chain
// and P1.4 invalidates the offending order at execution, breaking the chain
// there. Any other validation or reception error leaves the state, its chain
// ID counter, and the emitter's capacity unchanged.
func AssignChain(game *models.GameState, chain models.Chain) error {
	if game == nil {
		return assignmentError(ErrInvalidChain, "game state is nil")
	}
	if err := game.Validate(); err != nil {
		return assignmentError(ErrInvalidChain, fmt.Sprintf("game state is invalid: %v", err))
	}
	if validationErrors := ValidateChain(game, chain); len(validationErrors) != 0 {
		for _, validationError := range validationErrors {
			if validationError.Deferrable() {
				continue
			}
			return assignmentError(ErrInvalidChain, fmt.Sprintf("chain validation failed: %s", validationError))
		}
	}

	indexes := indexGame(game)
	noble := indexes.noblesByID[chain.NobleID]
	if noble == nil {
		return assignmentError(ErrInvalidChain, fmt.Sprintf("emitting noble %q does not exist", nobleReference(indexes, chain.NobleID)))
	}
	if noble.Status != models.NobleStatusFree {
		return assignmentError(ErrNoblePrisoner, fmt.Sprintf("noble %q is a prisoner", nobleReference(indexes, noble.ID)))
	}
	if noble.LastEmissionTurn == game.Turn {
		return assignmentError(ErrEmissionCapacity, fmt.Sprintf("noble %q has already emitted in turn %d", nobleReference(indexes, noble.ID), game.Turn))
	}

	positionID := chain.Orders[0].PositionID
	army := armyAtTerritory(game, indexes, positionID)
	if army == nil {
		return assignmentError(ErrNoArmyOnPosition, fmt.Sprintf("no army occupies receiving position %q", territoryReference(indexes, positionID)))
	}
	if army.OwnerID != noble.OwnerID {
		return assignmentError(ErrArmyNotOwned, fmt.Sprintf("army at receiving position %q belongs to %q, not emitting noble owner %q", territoryReference(indexes, positionID), army.OwnerID, noble.OwnerID))
	}
	for _, existing := range game.Chains {
		if existing.PendingDisperse != nil && existing.PendingDisperse.ArmyID == army.ID && existing.ArmyID != army.ID {
			return assignmentError(ErrInvalidChain, fmt.Sprintf("army at receiving position %q executes pending dispersion for chain %q", territoryReference(indexes, positionID), existing.ID))
		}
	}

	for orderIndex, order := range chain.Orders {
		if orderIndex == 0 && (order.Type == models.OrderTypeHostage || order.Type == models.OrderTypeDungeon) {
			if err := validateImmediatePrisonerTarget(indexes, army, order); err != nil {
				return err
			}
		}
	}

	// All failure paths above return before this point. Build the replacement
	// first, then make the small state update without any fallible operations.
	received := cloneChain(chain)
	receivedID := models.ChainID(fmt.Sprintf("C%d", game.NextChainID))
	for _, existing := range game.Chains {
		if existing.ID == receivedID {
			return assignmentError(ErrInvalidChain, fmt.Sprintf("next chain id %q is already in use", receivedID))
		}
	}
	received.ID = receivedID
	received.ArmyID = army.ID
	received.CurrentIndex = 0
	for i := range received.Orders {
		received.Orders[i].ArmyID = army.ID
	}

	replacement := make([]models.Chain, 0, len(game.Chains)+1)
	if army.ChainID != nil {
		for _, existing := range game.Chains {
			if existing.ID != *army.ChainID {
				replacement = append(replacement, existing)
			}
		}
	} else {
		replacement = append(replacement, game.Chains...)
	}
	replacement = append(replacement, received)

	game.Chains = replacement
	game.NextChainID++
	army.ChainID = &receivedID
	noble.LastEmissionTurn = game.Turn
	return nil
}

func validateImmediatePrisonerTarget(indexes gameIndexes, army *models.Army, order models.Order) error {
	target := indexes.noblesByID[order.NobleTargetIDs[0]]
	if target == nil || target.Status == models.NobleStatusFree || target.LocationID != army.TerritoryID || target.OwnerID == army.OwnerID {
		return assignmentError(ErrNobleNotPrisoner, fmt.Sprintf("noble target %q is not a prisoner held by army at %q", nobleReference(indexes, order.NobleTargetIDs[0]), territoryReference(indexes, army.TerritoryID)))
	}
	return nil
}
