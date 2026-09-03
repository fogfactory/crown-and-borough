package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type fairWeatherCardDefinition struct{}

type fairWeatherOrder struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

func (fairWeatherCardDefinition) Kind() models.CardKind { return models.CardKindFairWeather }

func (fairWeatherCardDefinition) CanPlay(ctx *ExecutionContext, _ models.DeckOrder) (bool, string) {
	return ctx.season != models.SeasonWinter, "deck_order_out_of_season"
}

func (fairWeatherCardDefinition) NewOrder(playerID models.PlayerID, order models.DeckOrder) ExecutableOrder {
	return fairWeatherOrder{playerID: playerID, order: order}
}

func (order fairWeatherOrder) Apply(ctx *ExecutionContext) {
	definition := fairWeatherCardDefinition{}
	if applicable, reason := definition.CanPlay(ctx, order.order); !applicable {
		ctx.resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
		return
	}
	ctx.resolution.applyDeckCardOrder(order.playerID, order.order)
}
