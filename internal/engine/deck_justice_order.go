package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Reasons carried by the event of a played justice card.
const (
	JusticeExecutedReason = "justice_executed"
	JusticeNoEffectReason = "justice_no_effect"
)

// Reasons for which a justice play is rejected, the card being kept.
const (
	JusticeUnknownNobleReason = "justice_unknown_noble"
	JusticeKingReason         = "justice_king"
)

// justiceCardDefinition is the justice card (specs/ordres-speciaux.md
// § Justice, #415): "P JU HHH" targets any living noble of any player, the
// player's own included. If HHH commissioned at least one assassination,
// whoever the victim, they are executed (normal death: fiefs, claims, cards,
// the crown); otherwise the card is consumed with no effect, which only the
// player who played it learns. The card is consumed when played and resolves
// after the assassinations of the turn (deckOrderRank); on the winter sheet it
// is reserved at stage 8 and applied at stage 9 (winterEventCards). The King
// is never a target.
type justiceCardDefinition struct{}

type justiceOrder struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

func (justiceCardDefinition) Kind() models.CardKind { return models.CardKindJustice }

func (justiceCardDefinition) CanPlay(ctx *ExecutionContext, order models.DeckOrder) (bool, string) {
	if ctx.resolution.noblesByID[order.TargetNobleID] == nil {
		return false, JusticeUnknownNobleReason
	}
	if ctx.resolution.state.IsKing(order.TargetNobleID) {
		// The King is never tried, by card or by cardinals.
		return false, JusticeKingReason
	}
	return true, ""
}

func (justiceCardDefinition) NewOrder(playerID models.PlayerID, order models.DeckOrder) ExecutableOrder {
	return justiceOrder{playerID: playerID, order: order}
}

func (justiceCardDefinition) applyWinterEffect(ctx *resolutionContext, playerID models.PlayerID, order models.DeckOrder) {
	ctx.resolveJustice(playerID, order)
}

func (order justiceOrder) Apply(ctx *ExecutionContext) {
	definition := justiceCardDefinition{}
	if applicable, reason := definition.CanPlay(ctx, order.order); !applicable {
		ctx.resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
		return
	}
	resolution := ctx.resolution
	if !resolution.applyDeckCardOrder(order.playerID, order.order) {
		return
	}
	resolution.resolveJustice(order.playerID, order.order)
}

// resolveJustice executes the target when they commissioned an assassination.
// The event is public when the execution takes place, with the player who
// played the card, and private to that player otherwise
// (api.visibleSeasonEffects).
func (ctx *resolutionContext) resolveJustice(playerID models.PlayerID, order models.DeckOrder) {
	target := *ctx.noblesByID[order.TargetNobleID]
	event := Event{
		Type: EventTypeJustice, Phase: phaseForSeason(ctx.state.Season), OwnerID: playerID, OrderID: order.ID,
		CardKind: models.CardKindJustice, Reason: JusticeNoEffectReason,
		NobleID: target.ID, NobleCode: models.NobleCode(target.Code), NobleName: ctx.state.NobleDisplayName(target),
		PreviousOwnerID: target.OwnerID, TerritoryID: target.LocationID,
		Season: ctx.state.Season, Year: ctx.state.Year(),
	}
	if !ctx.state.IsCommissioner(target.ID) {
		ctx.events = append(ctx.events, event)
		return
	}
	event.Reason = JusticeExecutedReason
	ctx.events = append(ctx.events, event)
	ctx.removeNobleInAction(target, models.DeathCauseExecution)
}
