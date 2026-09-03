package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type revoltCardDefinition struct{}

type revoltOrder struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

func (revoltCardDefinition) Kind() models.CardKind { return models.CardKindRevolt }

func (revoltCardDefinition) CanPlay(ctx *ExecutionContext, order models.DeckOrder) (bool, string) {
	if ctx.season == models.SeasonWinter {
		return false, "deck_order_out_of_season"
	}
	if !ctx.resolution.hasActiveCalamity(order.RegionSeed, models.CardKindFamine) {
		return false, "revolt_requires_famine"
	}
	return true, ""
}

func (revoltCardDefinition) NewOrder(playerID models.PlayerID, order models.DeckOrder) ExecutableOrder {
	return revoltOrder{playerID: playerID, order: order}
}

func (order revoltOrder) Apply(ctx *ExecutionContext) {
	definition := revoltCardDefinition{}
	if applicable, reason := definition.CanPlay(ctx, order.order); !applicable {
		ctx.resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
		return
	}
	ctx.resolution.applyDeckCardOrder(order.playerID, order.order)
}
