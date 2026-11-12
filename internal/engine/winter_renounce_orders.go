package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// renounceClaimOrder is T C HHH (specs/titres.md § Conquête d'un fief par un
// noble, "Renonciation"): the player gives up the claim their heir HHH holds.
// The claim ends as renounced and its card goes back to the discard pile, so a
// later C N of the same winter, in sheet order, may stake a new claim. It is
// free.
type renounceClaimOrder struct{ order models.WinterOrder }

func (order renounceClaimOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	heirID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	heir := resolution.noblesByID[heirID]
	if !exists || heir == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	if heir.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_not_owned")
		return
	}
	claim, claiming := resolution.state.RenounceClaim(heirID)
	if !claiming {
		resolution.rejectWinterOrder(playerID, winterOrder, "no_claim_to_renounce")
		return
	}
	resolution.discardClaimCard(heirID)
	event := Event{
		Type:    EventTypeTitleRenounced,
		Phase:   winterPhase,
		OwnerID: playerID,
		OrderID: winterOrder.ID,
		Reason:  "claim",
	}
	resolution.setNoble(&event, heir)
	if target := resolution.noblesByID[claim.Target]; target != nil {
		event.SpouseNobleName = resolution.state.NobleDisplayName(*target)
	}
	resolution.events = append(resolution.events, event)
}

// renounceTitleOrder is T D NNN XXX: the player's noble NNN gives up the
// exiled or usurped title it carries on the fief whose capital is XXX. The
// engine determines which. An exiled title is abandoned: the usurper becomes
// the official titulaire and no restoration is possible any more. A usurped
// title is given back: the exile recovers it and the fief returns to the
// exile's owner, as when the usurper dies. It is free.
type renounceTitleOrder struct{ order models.WinterOrder }

func (order renounceTitleOrder) Apply(ctx *ExecutionContext) {
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
	fief := resolution.fiefByCapital(winterOrder.TerritoryID)
	if fief == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_not_found")
		return
	}
	kind, previousOwnerID := resolution.state.RenounceContestedTitle(fief, nobleID)
	if kind == "" {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_not_contested")
		return
	}
	if kind == models.TitleRenouncedUsurped {
		resolution.clearFiefCapitals(previousOwnerID, fief)
	}
	resolution.reportTitleRenounced(fief, previousOwnerID, playerID, winterOrder, string(kind), noble)
}

func (ctx *resolutionContext) reportTitleRenounced(fief *models.Fief, previousOwnerID, playerID models.PlayerID, order models.WinterOrder, reason string, renouncer *models.Noble) {
	event := ctx.fiefTitleEvent(EventTypeTitleRenounced, fief, previousOwnerID, reason, winterPhase)
	event.OrderID = order.ID
	event.OwnerID = fief.OwnerID
	ctx.setNoble(&event, renouncer)
	ctx.events = append(ctx.events, event)
}
