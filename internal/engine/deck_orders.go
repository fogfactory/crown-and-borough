package engine

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

type CardDefinition interface {
	Kind() models.CardKind
	CanPlay(*ExecutionContext, models.DeckOrder) (bool, string)
	NewOrder(models.PlayerID, models.DeckOrder) ExecutableOrder
}

var cardDefinitions = map[models.CardKind]CardDefinition{
	models.CardKindFairWeather:     fairWeatherCardDefinition{},
	models.CardKindAbundantHarvest: abundantHarvestCardDefinition{},
	models.CardKindRevolt:          revoltCardDefinition{},
}

type deckOrderIntent struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

func newExecutableDeckOrder(playerID models.PlayerID, order models.DeckOrder) ExecutableOrder {
	definition := cardDefinitions[order.Kind]
	if definition == nil || order.Type != models.DeckOrderTypePlay {
		return nil
	}
	return definition.NewOrder(playerID, order)
}

func validateActionDeckOrders(game *models.GameState, balance assetgen.Balance, deckOrders map[models.PlayerID][]models.DeckOrder) error {
	players := make(map[models.PlayerID]bool, len(game.Players))
	for _, player := range game.Players {
		players[player.ID] = true
	}
	hands := make(map[models.PlayerID][]models.SpecialCardID)
	if game.SpecialDeck != nil {
		for playerID, hand := range game.SpecialDeck.Hands {
			hands[playerID] = append([]models.SpecialCardID(nil), hand...)
		}
	}
	validationContext := newResolutionContext(game, balance)
	for _, playerID := range sortedDeckPlayerIDs(deckOrders) {
		if !players[playerID] {
			return fmt.Errorf("engine: resolve: unknown player %q", playerID)
		}
		for _, order := range deckOrders[playerID] {
			if order.Type != models.DeckOrderTypePlay {
				return fmt.Errorf("engine: resolve: deck order %q is not valid in an action season", order.Type)
			}
			definition := cardDefinitions[order.Kind]
			if definition == nil {
				return fmt.Errorf("engine: resolve: invalid deck kind %q", order.Kind)
			}
			if playable, reason := definition.CanPlay(&ExecutionContext{resolution: validationContext, playerID: playerID, season: game.Season}, order); !playable {
				return fmt.Errorf("engine: resolve: deck order %q rejected: %s", order.Kind, reason)
			}
			index := -1
			for handIndex, cardID := range hands[playerID] {
				if cardKind(game.SpecialDeck, cardID) == order.Kind {
					index = handIndex
					break
				}
			}
			if index < 0 {
				return fmt.Errorf("engine: resolve: player %q has no card of kind %q", playerID, order.Kind)
			}
			hands[playerID] = append(hands[playerID][:index], hands[playerID][index+1:]...)
		}
	}
	return nil
}

func sortedDeckPlayerIDs(deckOrders map[models.PlayerID][]models.DeckOrder) []models.PlayerID {
	playerIDs := make([]models.PlayerID, 0, len(deckOrders))
	for playerID := range deckOrders {
		playerIDs = append(playerIDs, playerID)
	}
	sort.Slice(playerIDs, func(i, j int) bool { return playerIDs[i] < playerIDs[j] })
	return playerIDs
}

func resolveDeckOrders(ctx *resolutionContext, deckOrders map[models.PlayerID][]models.DeckOrder) {
	for _, playerID := range sortedDeckPlayerIDs(deckOrders) {
		for _, order := range deckOrders[playerID] {
			executable := newExecutableDeckOrder(playerID, order)
			if executable == nil {
				ctx.rejectDeckOrderReason(playerID, order, "invalid_deck_order")
				continue
			}
			executable.Apply(&ExecutionContext{resolution: ctx, playerID: playerID, season: ctx.state.Season})
		}
	}
}

func (ctx *resolutionContext) applyDeckCardOrder(playerID models.PlayerID, order models.DeckOrder) {
	if !ctx.consumeDeckCard(playerID, order.Kind) {
		ctx.rejectDeckOrder(playerID, order)
		return
	}
	ctx.deckIntents = append(ctx.deckIntents, deckOrderIntent{playerID: playerID, order: order})
}

func (ctx *resolutionContext) consumeDeckCard(playerID models.PlayerID, kind models.CardKind) bool {
	if ctx.state.SpecialDeck == nil {
		return false
	}
	hand := ctx.state.SpecialDeck.Hands[playerID]
	for index, cardID := range hand {
		for _, card := range ctx.state.SpecialDeck.Cards {
			if card.ID != cardID || card.Kind != kind {
				continue
			}
			ctx.state.SpecialDeck.Hands[playerID] = append(hand[:index], hand[index+1:]...)
			ctx.state.SpecialDeck.Discard = append(ctx.state.SpecialDeck.Discard, cardID)
			return true
		}
	}
	return false
}

func (ctx *resolutionContext) hasActiveCalamity(regionSeed models.TerritoryID, kind models.CardKind) bool {
	augury, exists := ctx.state.Auguries[ctx.state.Year()]
	if !exists {
		return false
	}
	for _, calamity := range augury.Calamities {
		if calamity.Kind == kind && calamity.RegionSeed == regionSeed && calamity.Season == ctx.state.Season {
			return true
		}
	}
	return false
}

func (ctx *resolutionContext) rejectDeckOrder(playerID models.PlayerID, order models.DeckOrder) {
	ctx.rejectDeckOrderReason(playerID, order, "no_card_for_kind")
}

func (ctx *resolutionContext) rejectDeckOrderReason(playerID models.PlayerID, order models.DeckOrder, reason string) {
	ctx.events = append(ctx.events, Event{
		Type:    EventTypeRejected,
		Phase:   winterPhase,
		OwnerID: playerID,
		OrderID: order.ID,
		Reason:  reason,
	})
}
