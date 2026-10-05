package engine

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// nobleCount returns the living nobles owned by the player: free, hostage or
// held in a dungeon. Dead and removed nobles are no longer in state.Nobles.
func (resolution *resolutionContext) nobleCount(playerID models.PlayerID) int {
	count := 0
	for _, noble := range resolution.state.Nobles {
		if noble.OwnerID == playerID {
			count++
		}
	}
	return count
}

// nobleLimit returns the player's current noble cap: the base balance value
// raised by the bonus of every noble of the player, whatever its status or
// marriage (each bastard adds one), then clamped to the balance maximum.
func (resolution *resolutionContext) nobleLimit(playerID models.PlayerID) int {
	limit := resolution.balance.NobleLimit
	for _, noble := range resolution.state.Nobles {
		if noble.OwnerID == playerID {
			limit += noble.NobleLimitBonus()
		}
	}
	return min(limit, resolution.balance.NobleLimitMax)
}

// recruitNobleOrder is R N XXX YYY: it plays the noble card XXX from the
// player's hand to make the noble appear on the castle or village YYY
// (specs/succession.md § Deck de nobles). It costs no R.
type recruitNobleOrder struct{ order models.WinterOrder }

func (order recruitNobleOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	if !resolution.territoryExists(winterOrder.TerritoryID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_territory")
		return
	}
	if !resolution.controlsTerritory(playerID, winterOrder.TerritoryID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "territory_not_controlled")
		return
	}
	if !resolution.hasSettlement(winterOrder.TerritoryID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_requires_settlement")
		return
	}
	army := resolution.currentArmyAt(winterOrder.TerritoryID)
	if army == nil || army.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_requires_owned_army")
		return
	}
	if resolution.nobleCount(playerID) >= resolution.nobleLimit(playerID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_limit_reached")
		return
	}
	card, handIndex, inHand := resolution.state.NobleDeck.HandCard(playerID, models.NobleCardKindNoble, winterOrder.CardCode)
	if !inHand {
		resolution.rejectWinterOrder(playerID, winterOrder, "card_not_in_hand")
		return
	}
	territory := resolution.territoriesByID[winterOrder.TerritoryID]
	noble := models.Noble{
		ID:               nextNobleID(resolution.state.Nobles, resolution.state.RemovedNobles),
		Code:             card.Code,
		Name:             fmt.Sprintf("%s de %s", card.Name, territory.Name),
		Sex:              card.Sex,
		OwnerID:          playerID,
		LocationID:       winterOrder.TerritoryID,
		Status:           models.NobleStatusFree,
		LastEmissionTurn: 0,
	}
	resolution.consumeNobleCard(playerID, handIndex, noble.ID)
	resolution.state.Nobles = append(resolution.state.Nobles, noble)
	resolution.rebuildIndexes()
	resolution.events = append(resolution.events, Event{
		Type:        EventTypeRecruit,
		Phase:       winterPhase,
		OwnerID:     playerID,
		OrderID:     winterOrder.ID,
		TerritoryID: winterOrder.TerritoryID,
		NobleID:     noble.ID,
		NobleCode:   models.NobleCode(noble.Code),
		NobleName:   resolution.state.NobleDisplayName(noble),
	})
}
