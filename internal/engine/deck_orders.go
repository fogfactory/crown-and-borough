package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

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
