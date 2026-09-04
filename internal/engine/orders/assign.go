package orders

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/i18n"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// AssignChain validates current reception conditions and atomically replaces
// the receiving army's chain. Callers must reject a chain returned by
// ParseChain when that call returned any ParseError before calling AssignChain.
// Any validation or reception error leaves the state, its chain ID counter,
// and the emitter's capacity unchanged.
func AssignChain(game *models.GameState, chain models.Chain) error {
	if game == nil {
		return assignmentError(ErrInvalidChain, i18n.AssignmentGameNil)
	}
	if err := game.Validate(); err != nil {
		return assignmentError(ErrInvalidChain, i18n.AssignmentInvalidState, err)
	}
	if validationErrors := ValidateChain(game, chain); len(validationErrors) != 0 {
		return assignmentError(ErrInvalidChain, i18n.AssignmentChainValidation, validationErrors[0])
	}

	indexes := indexGame(game)
	noble := indexes.noblesByID[chain.NobleID]
	if noble == nil {
		return assignmentError(ErrInvalidChain, i18n.AssignmentNobleUnknown, nobleReference(indexes, chain.NobleID))
	}
	if noble.Status == models.NobleStatusDungeon {
		return assignmentError(ErrNoblePrisoner, i18n.AssignmentNobleDungeon, nobleReference(indexes, noble.ID))
	}
	if noble.LastEmissionTurn == game.Turn {
		return assignmentError(ErrEmissionCapacity, i18n.AssignmentEmissionCapacity, nobleReference(indexes, noble.ID), game.Turn)
	}

	positionID := chain.Orders[0].PositionID
	army := armyAtTerritory(game, indexes, positionID)
	if army == nil {
		return assignmentError(ErrNoArmyOnPosition, i18n.AssignmentNoArmy, territoryReference(indexes, positionID))
	}
	if err := receivingArmyOwnershipError(indexes, positionID, army, noble.OwnerID); err != nil {
		return err
	}
	for _, existing := range game.Chains {
		if existing.PendingDisperse != nil && existing.PendingDisperse.ArmyID == army.ID && existing.ArmyID != army.ID {
			return assignmentError(ErrInvalidChain, i18n.AssignmentPendingDisperse, territoryReference(indexes, positionID), existing.ID)
		}
	}

	// All failure paths above return before this point. Build the replacement
	// first, then make the small state update without any fallible operations.
	received := cloneChain(chain)
	receivedID := models.ChainID(fmt.Sprintf("C%d", game.NextChainID))
	for _, existing := range game.Chains {
		if existing.ID == receivedID {
			return assignmentError(ErrInvalidChain, i18n.AssignmentChainIDInUse, receivedID)
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
