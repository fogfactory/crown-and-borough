package engine

import (
	"slices"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// buyCardinalOrder is N C NNN: the player pays religion.cardinal_cost R to
// promote its bishop NNN to cardinal (specs/religieux.md § Nomination et achat
// d'un cardinal). It is a management order (stage 2): the payment is taken
// here, the title is conferred at the investiture. The purchase cap counts the
// purchased cardinals in play plus the purchases already pending; cardinals
// obtained by card have their own cap, the number of cardinal cards.
type buyCardinalOrder struct{ order models.WinterOrder }

func (order buyCardinalOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	nobleID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	noble := resolution.noblesByID[nobleID]
	if !exists || noble == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	if noble.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_not_owned")
		return
	}
	// An excommunicated noble holds no title (stage 1 strips it), so a
	// purchase on it fails here without any payment.
	if !resolution.state.IsBishop(noble.ID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "cardinal_requires_bishop")
		return
	}
	if resolution.hasCardinalPending(noble.ID) || resolution.state.IsCardinal(noble.ID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "already_cardinal")
		return
	}
	if PurchasedCardinals(resolution.state)+resolution.pendingPurchases() >= CardinalPurchaseCap(resolution.state, resolution.balance) {
		resolution.rejectWinterOrder(playerID, winterOrder, "cardinal_cap_reached")
		return
	}
	payFrom := noble.LocationID
	if capitalID, _, hasCapital := resolution.capitalTerritory(playerID); hasCapital {
		payFrom = capitalID
	}
	spent, paid := resolution.payWinterCost(playerID, payFrom, resolution.balance.Religion.CardinalCost)
	if !paid {
		resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
		return
	}
	resolution.pendingCardinals = append(resolution.pendingCardinals, pendingCardinal{noble: noble.ID})
	resolution.events = append(resolution.events, Event{
		Type:          EventTypeCardinalPurchase,
		Phase:         winterPhase,
		OwnerID:       playerID,
		OrderID:       winterOrder.ID,
		NobleID:       noble.ID,
		NobleCode:     models.NobleCode(noble.Code),
		NobleName:     resolution.state.NobleDisplayName(*noble),
		ResourceSpent: spent,
	})
}

// pendingCardinal is a bishop whose cardinal title was bought this winter and
// is conferred at the investiture.
type pendingCardinal struct {
	noble models.NobleID
}

func (ctx *resolutionContext) hasCardinalPending(id models.NobleID) bool {
	return slices.ContainsFunc(ctx.pendingCardinals, func(pending pendingCardinal) bool { return pending.noble == id })
}

func (ctx *resolutionContext) pendingPurchases() int { return len(ctx.pendingCardinals) }

// playCardinalCard is D N XXX CAR: it plays a cardinal card of the noble deck
// on one of the player's own bishops (specs/religieux.md § Nomination et
// achat d'un cardinal). It is free; the number of cardinal cards in the deck
// is its cap. Like any dignity card it is played at any time: the noble is a
// cardinal at once, so it votes and counts in the elections of the same winter
// (only elected and bought titles wait for the investiture).
func (ctx *resolutionContext) playCardinalCard(playerID models.PlayerID, order models.WinterOrder, noble *models.Noble, handIndex int) {
	switch {
	case noble.OwnerID != playerID:
		ctx.rejectWinterOrder(playerID, order, "noble_not_owned")
		return
	case !ctx.state.IsBishop(noble.ID):
		ctx.rejectWinterOrder(playerID, order, "cardinal_requires_bishop")
		return
	case ctx.state.IsCardinal(noble.ID) || ctx.hasCardinalPending(noble.ID):
		ctx.rejectWinterOrder(playerID, order, "already_cardinal")
		return
	}
	ctx.consumeNobleCard(playerID, handIndex, noble.ID)
	noble.Dignities = append(noble.Dignities, models.DignityCardinal)
	ctx.state.Cardinals = append(ctx.state.Cardinals, noble.ID)
	ctx.events = append(ctx.events, Event{
		Type:      EventTypeDignity,
		Phase:     winterPhase,
		OwnerID:   playerID,
		OrderID:   order.ID,
		NobleID:   noble.ID,
		NobleCode: models.NobleCode(noble.Code),
		NobleName: ctx.state.NobleDisplayName(*noble),
		Dignity:   models.DignityCardinal,
	})
}

// excommunicate excommunicates a noble and takes the religious titles it holds.
// A cardinal obtained by card gives its card back to the discard pile; a
// purchased one frees its place under the purchase cap.
func (ctx *resolutionContext) excommunicate(excommunication models.Excommunication) {
	ctx.state.Excommunicate(excommunication)
	if noble := ctx.noblesByID[excommunication.Noble]; noble != nil && noble.Has(models.DignityCardinal) {
		ctx.removeDignity(noble, models.DignityCardinal)
	}
}

// investPurchasedCardinals confers the cardinal titles bought this winter. A
// noble that lost its bishopric or was excommunicated since gets nothing and
// the payment is not refunded (the stage order makes this case rare).
func (ctx *resolutionContext) investPurchasedCardinals() {
	pending := ctx.pendingCardinals
	ctx.pendingCardinals = nil
	for _, promotion := range pending {
		noble := ctx.noblesByID[promotion.noble]
		if noble == nil {
			continue
		}
		_, excommunicated := ctx.state.ExcommunicationOf(noble.ID)
		if excommunicated || !ctx.state.IsBishop(noble.ID) || ctx.state.IsCardinal(noble.ID) {
			continue
		}
		ctx.state.Cardinals = append(ctx.state.Cardinals, noble.ID)
		ctx.events = append(ctx.events, Event{
			Type:      EventTypeInvestiture,
			Phase:     winterPhase,
			OwnerID:   noble.OwnerID,
			NobleID:   noble.ID,
			NobleCode: models.NobleCode(noble.Code),
			NobleName: ctx.state.NobleDisplayName(*noble),
			Reason:    "cardinal",
		})
	}
}
