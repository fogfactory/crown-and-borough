package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type abundantHarvestCardDefinition struct{}

type abundantHarvestOrder struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

func (abundantHarvestCardDefinition) Kind() models.CardKind { return models.CardKindAbundantHarvest }

func (abundantHarvestCardDefinition) CanPlay(ctx *ExecutionContext, _ models.DeckOrder) (bool, string) {
	return ctx.season != models.SeasonWinter, "deck_order_out_of_season"
}

func (abundantHarvestCardDefinition) NewOrder(playerID models.PlayerID, order models.DeckOrder) ExecutableOrder {
	return abundantHarvestOrder{playerID: playerID, order: order}
}

func (order abundantHarvestOrder) Apply(ctx *ExecutionContext) {
	definition := abundantHarvestCardDefinition{}
	if applicable, reason := definition.CanPlay(ctx, order.order); !applicable {
		ctx.resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
		return
	}
	ctx.resolution.applyDeckCardOrder(order.playerID, order.order)
}
