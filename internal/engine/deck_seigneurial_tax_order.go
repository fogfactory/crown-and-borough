package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// seigneurialTaxCardDefinition is the "P TX XXX" order (titres.md "Taxe
// seigneuriale", #189): XXX targets a fief's capital, by exception to the
// usual "TER is a region's seed village" rule, and the player must control
// that fief. Its effect (doubling the fief's territorial income, widening
// revolt eligibility) is applied by applySeigneurialTax in
// season_effects.go, alongside the other bonus cards.
type seigneurialTaxCardDefinition struct{}

type seigneurialTaxOrder struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

func (seigneurialTaxCardDefinition) Kind() models.CardKind { return models.CardKindSeigneurialTax }

func (seigneurialTaxCardDefinition) CanPlay(ctx *ExecutionContext, order models.DeckOrder) (bool, string) {
	if ctx.season == models.SeasonWinter {
		return false, "deck_order_out_of_season"
	}
	if ctx.resolution.isTitheOrder(order) {
		return titheRejection(ctx.resolution, ctx.playerID, order)
	}
	if order.TargetNobleID != "" {
		if issuer := ctx.resolution.noblesByID[order.TargetNobleID]; issuer == nil || issuer.OwnerID != ctx.playerID {
			return false, "tax_requires_own_issuer"
		}
	}
	fief := ctx.resolution.fiefByCapital(order.TargetTerritoryID)
	if fief != nil && order.TargetNobleID != "" && fief.HolderNobleID != nil && *fief.HolderNobleID != order.TargetNobleID {
		return false, "seigneurial_tax_requires_fief_holder"
	}
	if fief == nil {
		return false, "seigneurial_tax_requires_fief_capital"
	}
	if fief.OwnerID != ctx.playerID {
		return false, "seigneurial_tax_requires_fief_owner"
	}
	return true, ""
}

func (seigneurialTaxCardDefinition) NewOrder(playerID models.PlayerID, order models.DeckOrder) ExecutableOrder {
	return seigneurialTaxOrder{playerID: playerID, order: order}
}

func (order seigneurialTaxOrder) Apply(ctx *ExecutionContext) {
	definition := seigneurialTaxCardDefinition{}
	if ctx.resolution.isTitheOrder(order.order) {
		if applicable, reason := definition.CanPlay(ctx, order.order); !applicable {
			ctx.resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
			return
		}
		ctx.resolution.applyDeckCardOrder(order.playerID, order.order)
		return
	}
	if applicable, reason := definition.CanPlay(ctx, order.order); !applicable {
		ctx.resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
		return
	}
	ctx.resolution.applyDeckCardOrder(order.playerID, order.order)
}
