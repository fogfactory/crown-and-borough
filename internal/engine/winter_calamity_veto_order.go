package engine

import (
	"slices"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// calamityVetoOrder is V C XXX I (specs/dames.md § Astrologue): the
// astrologue XXX of the player strikes the forecast calamity I from the draw
// pile; the following calamity takes its place. Each astrologue acts once per
// winter.
type calamityVetoOrder struct{ order models.WinterOrder }

func (order calamityVetoOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	nobleID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	if !exists {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	noble := resolution.noblesByID[nobleID]
	// Only the owner commands the astrologue: a captor profits from the
	// forecast but cannot strike calamities.
	if noble == nil || noble.OwnerID != playerID || noble.CalamityForecast() == 0 {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_not_astrologer")
		return
	}
	if resolution.calamityVetoes[noble.ID] {
		resolution.rejectWinterOrder(playerID, winterOrder, "calamity_veto_already_used")
		return
	}
	deck := resolution.state.SpecialDeck
	if deck == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "calamity_not_forecast")
		return
	}
	if len(winterOrder.Indices) != 1 {
		resolution.rejectWinterOrder(playerID, winterOrder, "calamity_not_forecast")
		return
	}
	forecast := CalamityForecast(resolution.state, noble.CalamityForecast())
	struck := make([]models.SpecialCardID, 0, len(winterOrder.Indices))
	for _, index := range winterOrder.Indices {
		if index < 1 || index > len(forecast) {
			resolution.rejectWinterOrder(playerID, winterOrder, "calamity_not_forecast")
			return
		}
		struck = append(struck, forecast[index-1].CardID)
	}
	if resolution.calamityVetoes == nil {
		resolution.calamityVetoes = map[models.NobleID]bool{}
	}
	resolution.calamityVetoes[noble.ID] = true
	deck.DrawPile = slices.DeleteFunc(deck.DrawPile, func(id models.SpecialCardID) bool { return slices.Contains(struck, id) })
	deck.Discard = append(deck.Discard, struck...)
	resolution.events = append(resolution.events, Event{
		Type: EventTypeCalamityVeto, Phase: winterPhase, OwnerID: playerID,
		OrderID: winterOrder.ID, NobleID: noble.ID,
	})
}

// ForecastedCalamity is one calamity card of the draw pile an astrologer sees.
type ForecastedCalamity struct {
	CardID models.SpecialCardID
	Kind   models.CardKind
}

// CalamityForecast lists the next count calamity cards of the special deck
// draw pile, in draw order.
func CalamityForecast(state *models.GameState, count int) []ForecastedCalamity {
	if state.SpecialDeck == nil || count <= 0 {
		return nil
	}
	kinds := make(map[models.SpecialCardID]models.CardKind, len(state.SpecialDeck.Cards))
	for _, card := range state.SpecialDeck.Cards {
		kinds[card.ID] = card.Kind
	}
	var forecast []ForecastedCalamity
	for _, cardID := range state.SpecialDeck.DrawPile {
		if kind := kinds[cardID]; kind.IsCalamity() {
			forecast = append(forecast, ForecastedCalamity{CardID: cardID, Kind: kind})
			if len(forecast) == count {
				break
			}
		}
	}
	return forecast
}
