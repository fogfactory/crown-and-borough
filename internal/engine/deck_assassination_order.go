package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// AssassinationReason is carried by the event of an assassination.
const AssassinationReason = "assassination"

// assassinationCardDefinition is the assassination card (specs/ordres-speciaux.md
// § Assassinat, #414): "P AS HHH NNN" has the player's free noble HHH, the
// commissioner, kill the noble NNN of another player, wherever they are. NNN
// dies without a roll, through the normal death path (fiefs, claims, the chain
// emitted this turn, the crown). The commissioner and the player who played
// the card are recorded on the lineage entry of the victim
// (models.RemovedNoble); who knows the link is derived from them and from the
// inquiry ledger (models.GameState.KnowsAssassination). The card is consumed when it is played and the death applies
// on the spot, before supply and the army orders; in winter the card is
// consumed before the hands are refilled and the death applies at stage 9,
// after the trial verdicts (specs/hiver.md).
type assassinationCardDefinition struct{}

type assassinationOrder struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

func (assassinationCardDefinition) Kind() models.CardKind { return models.CardKindAssassination }

func (assassinationCardDefinition) CanPlay(ctx *ExecutionContext, order models.DeckOrder) (bool, string) {
	// Playable in every season, winter included (specs/ordres-speciaux.md).
	commissioner := ctx.resolution.noblesByID[order.CommissionerNobleID]
	if commissioner == nil || commissioner.OwnerID != ctx.playerID || commissioner.Status != models.NobleStatusFree {
		return false, "assassination_commissioner_invalid"
	}
	victim := ctx.resolution.noblesByID[order.TargetNobleID]
	if victim == nil {
		return false, "assassination_unknown_noble"
	}
	if victim.OwnerID == ctx.playerID {
		return false, "assassination_own_noble"
	}
	return true, ""
}

func (assassinationCardDefinition) NewOrder(playerID models.PlayerID, order models.DeckOrder) ExecutableOrder {
	return assassinationOrder{playerID: playerID, order: order}
}

func (order assassinationOrder) Apply(ctx *ExecutionContext) {
	definition := assassinationCardDefinition{}
	if applicable, reason := definition.CanPlay(ctx, order.order); !applicable {
		ctx.resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
		return
	}
	resolution := ctx.resolution
	if !resolution.applyDeckCardOrder(order.playerID, order.order) {
		return
	}
	resolution.assassinate(order.playerID, order.order)
}

// assassinate kills the victim and records the commissioner. The event names
// the commissioner and the victim's owner: the report projection reveals the
// commissioner to the card's player only (api.visibleSeasonEffects).
func (ctx *resolutionContext) assassinate(playerID models.PlayerID, order models.DeckOrder) {
	victim := *ctx.noblesByID[order.TargetNobleID]
	commissioner := *ctx.noblesByID[order.CommissionerNobleID]
	ctx.events = append(ctx.events, Event{
		Type: EventTypeAssassination, Phase: phaseForSeason(ctx.state.Season), OwnerID: playerID, OrderID: order.ID,
		CardKind: models.CardKindAssassination, Reason: AssassinationReason,
		NobleID: victim.ID, NobleCode: models.NobleCode(victim.Code), NobleName: ctx.state.NobleDisplayName(victim),
		PreviousOwnerID: victim.OwnerID, TerritoryID: victim.LocationID,
		CommissionerCode: models.NobleCode(commissioner.Code), CommissionerName: ctx.state.NobleDisplayName(commissioner),
		Season: ctx.state.Season, Year: ctx.state.Year(),
	})
	ctx.removeNobleInAction(victim, models.DeathCauseAssassination)
	for index := range ctx.state.RemovedNobles {
		if ctx.state.RemovedNobles[index].ID == victim.ID {
			ctx.state.RemovedNobles[index].Commissioner = commissioner.ID
			ctx.state.RemovedNobles[index].Author = playerID
			// A prior inquiry on the victim or on the commissioner teaches
			// the link now; the author already reads it in the event.
			for _, other := range sortedPlayerIDs(ctx.state.Players) {
				if other != playerID && ctx.state.KnowsAssassination(other, ctx.state.RemovedNobles[index]) {
					ctx.emitAssassinationRevealed(other, ctx.state.RemovedNobles[index])
				}
			}
		}
	}
}
