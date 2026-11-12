package engine

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// fiefByCapital returns the fief whose capital is territoryID, or nil.
func (ctx *resolutionContext) fiefByCapital(territoryID models.TerritoryID) *models.Fief {
	for i := range ctx.state.Fiefs {
		if ctx.state.Fiefs[i].CapitalTerritoryID == territoryID {
			return &ctx.state.Fiefs[i]
		}
	}
	return nil
}

// fiefContaining returns the fief that already lists territoryID among its
// territories (capital or not), or nil.
func (ctx *resolutionContext) fiefContaining(territoryID models.TerritoryID) *models.Fief {
	for i := range ctx.state.Fiefs {
		for _, member := range ctx.state.Fiefs[i].Territories {
			if member == territoryID {
				return &ctx.state.Fiefs[i]
			}
		}
	}
	return nil
}

// removeFief deletes the fief identified by fiefID from state, along with any
// persisted tax window pointing at it (titres.md "Taxe seigneuriale"): a
// dissolved fief can no longer widen revolt eligibility for territories that
// are not even members of a fief any more.
func (ctx *resolutionContext) removeFief(fiefID models.FiefID) {
	filtered := make([]models.Fief, 0, len(ctx.state.Fiefs))
	for _, fief := range ctx.state.Fiefs {
		if fief.ID != fiefID {
			filtered = append(filtered, fief)
		}
	}
	ctx.state.Fiefs = filtered
	if len(ctx.state.TaxedFiefs) == 0 {
		return
	}
	filteredTax := make([]models.TaxedFief, 0, len(ctx.state.TaxedFiefs))
	for _, taxed := range ctx.state.TaxedFiefs {
		if taxed.FiefID != fiefID {
			filteredTax = append(filteredTax, taxed)
		}
	}
	ctx.state.TaxedFiefs = filteredTax
}

// fiefTaxWindowActive reports whether fiefID's persisted tax window
// (state.TaxedFiefs) still covers the current turn: the turn the tax was
// played, or the turn right after (titres.md "Taxe seigneuriale"). It does
// not see a tax played this same turn -- that is folded in separately via
// taxedFiefsThisTurn (once applied) or pendingTaxWindowFiefs (while still
// validating a submission), both checked by revoltEligibleByTax below.
func (ctx *resolutionContext) fiefTaxWindowActive(fiefID models.FiefID) bool {
	for _, taxed := range ctx.state.TaxedFiefs {
		if taxed.FiefID == fiefID && (taxed.Turn == ctx.state.Turn || taxed.Turn == ctx.state.Turn-1) {
			return true
		}
	}
	return false
}

// revoltEligibleByTax reports whether territoryID's fief was taxed recently
// enough to allow Révolte on it independently of famine: the persisted
// window from a previous turn, a tax already applied this turn, or a tax
// merely co-submitted this turn while a submission is still being validated
// (titres.md "Taxe seigneuriale").
func (ctx *resolutionContext) revoltEligibleByTax(territoryID models.TerritoryID) bool {
	if ctx.revoltEligibleByTithe(territoryID) {
		return true
	}
	fief := ctx.fiefContaining(territoryID)
	if fief == nil {
		return false
	}
	if ctx.taxedFiefsThisTurn[fief.ID] || ctx.royalTaxedFiefs[fief.ID] != "" || ctx.pendingTaxWindowFiefs[fief.ID] {
		return true
	}
	return ctx.fiefTaxWindowActive(fief.ID)
}

// pruneTaxedFiefs drops every persisted tax window entry that can no longer
// cover the current or next turn, so state.TaxedFiefs stays bounded instead
// of growing for the whole game (titres.md "Taxe seigneuriale").
func pruneTaxedFiefs(taxedFiefs []models.TaxedFief, currentTurn int) []models.TaxedFief {
	filtered := make([]models.TaxedFief, 0, len(taxedFiefs))
	for _, taxed := range taxedFiefs {
		if taxed.Turn >= currentTurn-1 {
			filtered = append(filtered, taxed)
		}
	}
	return filtered
}

// nextFiefID allocates the next unused sequential fief id. Like nextNobleID
// and nextInfrastructureID, this is an internal detail: the API and reports
// address a fief by its capital's trigram instead (titres.md, #194).
func nextFiefID(fiefs []models.Fief) models.FiefID {
	next := 1
	for _, fief := range fiefs {
		value := string(fief.ID)
		if len(value) < 2 || value[0] != 'F' {
			continue
		}
		var sequence int
		if _, err := fmt.Sscanf(value[1:], "%d", &sequence); err == nil && sequence >= next {
			next = sequence + 1
		}
	}
	return models.FiefID(fmt.Sprintf("F%d", next))
}

// fortificationBonus is the defensive bonus a territory's castle or fortified
// village provides: none without either, the city bonus when a castle sits on
// a fief's capital (it replaces, rather than stacks with, the plain castle
// bonus -- a fortified village can never itself be a fief capital, which
// always requires an actual castle), the plain castle bonus when the castle
// or fortified village is anchored some other way (a non-capital fief member,
// or a player's own capital) or currently held by an army, and none at all
// otherwise: an empty castle or fortified village outside every fief and
// capital is inert, like an unanchored supply depot (#215, #193).
func (ctx *resolutionContext) fortificationBonus(territoryID models.TerritoryID) int {
	if !ctx.hasLiveCastle(territoryID) && !ctx.hasFortifiedVillage(territoryID) {
		return 0
	}
	if ctx.fiefByCapital(territoryID) != nil {
		return ctx.balance.CityDefenseBonus
	}
	if !ctx.territoryAnchored(territoryID) && ctx.currentArmyAt(territoryID) == nil {
		return 0
	}
	return ctx.balance.CastleDefenseBonus
}

// captureFiefCapital applies the arrival of a player's army on the capital of
// a fief owned by somebody else (titres.md "Conquête d'un fief par un noble")
// and reports whether the fief changed hands. Control of the capital is
// derived from the fief's owner, so nothing is written when it returns false:
// the army then merely occupies the capital.
//
// The owner of the exiled noble of the fief retakes it with a mere army
// (restoration, no noble needed): the fief comes back to them entirely, the
// exile is titulaire again and the usurper loses everything. Otherwise an
// army needs a free noble of its owner on the capital to conquer: the fief,
// with all its territories, passes to the conqueror (the capital's own
// control, the other members through control by the control pass's working
// view of who controls what). A held fief gives its first free noble, in line
// of succession, as usurper, and its titulaire becomes the exile; a fief
// already usurped drops its usurper and keeps its exile; a vacant fief passes
// vacant, with neither usurper nor exile. One control_changed event per
// transferred member, reason "fief_transferred", is reported.
func (ctx *resolutionContext) captureFiefCapital(fief *models.Fief, army *models.Army, control map[models.TerritoryID]models.PlayerID) bool {
	if fief == nil || fief.CapitalTerritoryID != army.TerritoryID || fief.OwnerID == army.OwnerID {
		return false
	}
	previousOwnerID := fief.OwnerID
	if fief.ExiledNobleID != nil {
		if exile := ctx.noblesByID[*fief.ExiledNobleID]; exile != nil && exile.OwnerID == army.OwnerID {
			usurperID := fief.HolderNobleID
			ctx.state.RestoreExile(fief)
			event := ctx.fiefTitleEvent(EventTypeFiefConquered, fief, previousOwnerID, "restoration", 5)
			ctx.setNoble(&event, exile)
			if usurperID != nil {
				ctx.setExiledNoble(&event, ctx.noblesByID[*usurperID])
			}
			ctx.events = append(ctx.events, event)
			ctx.transferFiefMembers(fief, army.OwnerID, control)
			return true
		}
	}
	free := ctx.state.FreeNoblesAt(army.OwnerID, army.TerritoryID)
	if len(free) == 0 {
		return false
	}
	reason := ""
	if fief.HolderNobleID != nil {
		if !fief.Usurped {
			if holder := ctx.noblesByID[*fief.HolderNobleID]; holder != nil {
				exiledID := holder.ID
				fief.ExiledNobleID = &exiledID
				fief.Usurped = true
			}
		}
		if fief.Usurped {
			// A fallen usurper loses everything; the original exile stays.
			usurperID := free[0].ID
			fief.HolderNobleID = &usurperID
			reason = "usurpation"
		} else {
			fief.HolderNobleID = nil
		}
	}
	fief.OwnerID = army.OwnerID
	fief.Crown = false
	event := ctx.fiefTitleEvent(EventTypeFiefConquered, fief, previousOwnerID, reason, 5)
	if reason != "" {
		ctx.setNoble(&event, ctx.noblesByID[*fief.HolderNobleID])
		ctx.setExiledNoble(&event, ctx.noblesByID[*fief.ExiledNobleID])
	}
	ctx.events = append(ctx.events, event)
	ctx.transferFiefMembers(fief, army.OwnerID, control)
	return true
}

// fiefTitleEvent is the base of the events a fief changing hands or titulaire
// reports: the fief, its territories, and the previous and new owners.
func (ctx *resolutionContext) fiefTitleEvent(eventType EventType, fief *models.Fief, previousOwnerID models.PlayerID, reason string, phase int) Event {
	return Event{
		Type:            eventType,
		Phase:           phase,
		TerritoryID:     fief.CapitalTerritoryID,
		FiefID:          fief.ID,
		FiefTitle:       fief.Title,
		FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
		PreviousOwnerID: previousOwnerID,
		OwnerID:         fief.OwnerID,
		Reason:          reason,
	}
}

// setNoble names the noble an event is about.
func (ctx *resolutionContext) setNoble(event *Event, noble *models.Noble) {
	if noble == nil {
		return
	}
	event.NobleID = noble.ID
	event.NobleCode = models.NobleCode(noble.Code)
	event.NobleName = ctx.state.NobleDisplayName(*noble)
}

// setExiledNoble names the exiled noble (or the fallen usurper) an event
// reports next to its main noble.
func (ctx *resolutionContext) setExiledNoble(event *Event, noble *models.Noble) {
	if noble == nil {
		return
	}
	event.ExiledNobleID = noble.ID
	event.ExiledNobleCode = models.NobleCode(noble.Code)
	event.ExiledNobleName = ctx.state.NobleDisplayName(*noble)
}

// transferFiefMembers hands every member of the fief, but its capital, still
// held by another player in the control pass's working view to the fief's new
// owner, clearing that member's capital status if it carried one.
func (ctx *resolutionContext) transferFiefMembers(fief *models.Fief, newOwnerID models.PlayerID, control map[models.TerritoryID]models.PlayerID) {
	for _, memberID := range fief.Territories {
		if memberID == fief.CapitalTerritoryID {
			continue
		}
		memberPreviousOwnerID, controlled := control[memberID]
		if controlled && memberPreviousOwnerID == newOwnerID {
			continue
		}
		control[memberID] = newOwnerID
		ctx.clearCapitalOnControlLoss(memberPreviousOwnerID, memberID)
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeControlChanged,
			Phase:           5,
			TerritoryID:     memberID,
			PreviousOwnerID: memberPreviousOwnerID,
			OwnerID:         newOwnerID,
			Reason:          "fief_transferred",
		})
	}
}

// claimHeirOf returns the living heir of the first claim staked on the dead
// noble, in the order the claims were played, or nil when none stands.
func (ctx *resolutionContext) claimHeirOf(deadID models.NobleID) *models.Noble {
	for _, claim := range ctx.state.ClaimsOn(deadID) {
		if heir := ctx.noblesByID[claim.Heir]; heir != nil {
			return heir
		}
	}
	return nil
}

// passFiefToHeir hands a fief whose titulaire died to the heir that claimed
// its title (specs/succession.md § Prétentions), without any succession rank
// check. When an army of a player other than the heir's house stands on the
// fief's capital with a free noble (the garrison of the previous owner
// included), that noble takes the fief as usurper, the fief passes to the
// army's owner like a conquered one, and the heir only receives the exiled
// title (titres.md "Conquête d'un fief par un noble"). Otherwise the heir
// becomes the titulaire, the fief changes owner with all its territories, and
// an army without a noble on the capital merely occupies it.
func (ctx *resolutionContext) passFiefToHeir(fief *models.Fief, heir models.Noble) {
	previousOwnerID := fief.OwnerID
	heirID := heir.ID
	ctx.honourClaimOf(heirID)
	event := Event{
		Type:            EventTypeFiefConquered,
		Phase:           phaseForSeason(ctx.state.Season),
		TerritoryID:     fief.CapitalTerritoryID,
		FiefID:          fief.ID,
		FiefTitle:       fief.Title,
		FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
		PreviousOwnerID: previousOwnerID,
		Reason:          "claim",
	}
	if usurper := ctx.claimUsurperAt(fief.CapitalTerritoryID, heir); usurper != nil {
		usurperID := usurper.ID
		fief.OwnerID = usurper.OwnerID
		fief.HolderNobleID = &usurperID
		fief.ExiledNobleID = &heirID
		fief.Usurped = true
		event.Reason = "claim_usurpation"
		ctx.setNoble(&event, usurper)
		ctx.setExiledNoble(&event, &heir)
	} else {
		fief.OwnerID = heir.OwnerID
		fief.HolderNobleID = &heirID
		ctx.setNoble(&event, &heir)
	}
	fief.Crown = false
	event.OwnerID = fief.OwnerID
	if fief.OwnerID != previousOwnerID {
		for _, memberID := range fief.Territories {
			ctx.clearCapitalOnControlLoss(previousOwnerID, memberID)
		}
	}
	ctx.events = append(ctx.events, event)
}

// claimUsurperAt returns the free noble that usurps a fief whose titulaire
// died when the claim of heir fires (GameState.ClaimUsurperAt), garrison of
// the previous owner included.
func (ctx *resolutionContext) claimUsurperAt(capitalID models.TerritoryID, heir models.Noble) *models.Noble {
	return ctx.state.ClaimUsurperAt(capitalID, heir.OwnerID)
}

// vacateFiefsOfMissingHolders settles the titles of the fiefs whose titulaire
// or exiled noble no longer exists (death, e.g. from plague), through
// GameState.SettleFiefTitlesOfMissingNobles: a fief whose titulaire was the
// target of a claim passes to the heir (passFiefToHeir), a usurped fief
// returns to its exile when the usurper dies, becomes the usurper's own when
// the exile dies (or goes to the heir of a claim on the exile as new exile),
// and any other fief stays with its owner, vacant, until it is attributed
// again or dissolved (titres.md). Claims on a usurper do not apply to the
// usurped fief. Call after the caller has removed the dead nobles and rebuilt
// indexes.
func (ctx *resolutionContext) vacateFiefsOfMissingHolders() {
	phase := phaseForSeason(ctx.state.Season)
	settlements := ctx.state.SettleFiefTitlesOfMissingNobles(func(_ *models.Fief, missing models.NobleID) *models.Noble {
		return ctx.claimHeirOf(missing)
	})
	for _, settlement := range settlements {
		fief := settlement.Fief
		switch settlement.Outcome {
		case models.FiefTitleClaimed:
			ctx.passFiefToHeir(fief, *ctx.noblesByID[settlement.Heir])
		case models.FiefTitleVacated:
			ctx.events = append(ctx.events, ctx.fiefTitleEvent(EventTypeFiefVacated, fief, "", "", phase))
		case models.FiefTitleRestored:
			if fief.OwnerID != settlement.PreviousOwner {
				for _, memberID := range fief.Territories {
					ctx.clearCapitalOnControlLoss(settlement.PreviousOwner, memberID)
				}
			}
			event := ctx.fiefTitleEvent(EventTypeFiefConquered, fief, settlement.PreviousOwner, "usurper_died", phase)
			ctx.setNoble(&event, ctx.noblesByID[*fief.HolderNobleID])
			ctx.events = append(ctx.events, event)
		case models.FiefTitleLegitimised:
			event := ctx.fiefTitleEvent(EventTypeFiefTitleSettled, fief, "", "usurper_legitimised", phase)
			ctx.setNoble(&event, ctx.noblesByID[*fief.HolderNobleID])
			ctx.events = append(ctx.events, event)
		case models.FiefTitleReexiled:
			ctx.honourClaimOf(settlement.Heir)
			event := ctx.fiefTitleEvent(EventTypeFiefTitleSettled, fief, "", "exile_replaced", phase)
			ctx.setNoble(&event, ctx.noblesByID[settlement.Heir])
			ctx.events = append(ctx.events, event)
		}
	}
}

// returnOrphanFiefsToCrown applies GameState.ReturnOrphanFiefsToCrown and its
// aftermath: the former owner loses the capital designation of any territory
// of the fief, and one fief_conquered event with the reason "crown" reports it.
// Call once claims are settled and the queen refreshed, and again after a
// coronation, which brings the vacant fiefs back that no crown player could
// take.
func (ctx *resolutionContext) returnOrphanFiefsToCrown() {
	for _, reversion := range ctx.state.ReturnOrphanFiefsToCrown() {
		fief := reversion.Fief
		for _, memberID := range fief.Territories {
			ctx.clearCapitalOnControlLoss(reversion.PreviousOwner, memberID)
		}
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeFiefConquered,
			Phase:           phaseForSeason(ctx.state.Season),
			TerritoryID:     fief.CapitalTerritoryID,
			FiefID:          fief.ID,
			FiefTitle:       fief.Title,
			FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
			PreviousOwnerID: reversion.PreviousOwner,
			OwnerID:         fief.OwnerID,
			Reason:          "crown",
		})
	}
}

// dissolveFiefOnCapitalCastleLoss dissolves the fief anchored on territoryID
// immediately when its capital's castle disappears (pillage or automatic
// famine pillage): titres.md deliberately chose immediate dissolution over
// suspending the city bonus. A no-op when territoryID is not a fief capital.
func (ctx *resolutionContext) dissolveFiefOnCapitalCastleLoss(territoryID models.TerritoryID) {
	fief := ctx.fiefByCapital(territoryID)
	if fief == nil {
		return
	}
	ctx.dissolveFief(*fief, "capital_castle_lost")
}

// resolveVacantFiefsAtWinterEnd runs at the end of every winter, after winter
// orders (including a same-turn T A) and before stock conservation. A fief is
// never dissolved for lack of attribution (titres.md "Perte et vacance d'un
// fief"): it is attributed by default to the first noble of its owner's line
// of succession (succession.md § Lignée), whatever that noble's status
// (a hostage or prisoner is not skipped), with a warning event telling the
// player to take back manual attribution next turn. An owner without any
// living noble leaves the fief vacant: it keeps producing and scoring until
// attributed or dissolved by its capital's castle falling
// (dissolveFiefOnCapitalCastleLoss, unaffected by this function). A crown
// fief is skipped: only a W G order (or its owner's T A) attributes it.
func (ctx *resolutionContext) resolveVacantFiefsAtWinterEnd() {
	for i := range ctx.state.Fiefs {
		fief := &ctx.state.Fiefs[i]
		if fief.HolderNobleID != nil || fief.Crown {
			continue
		}
		line := ctx.state.SuccessionLine(fief.OwnerID)
		if len(line) == 0 {
			continue
		}
		nobleID := line[0].ID
		noble := ctx.noblesByID[nobleID]
		fief.HolderNobleID = &nobleID
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeFiefAutoAssigned,
			Phase:           winterPhase,
			OwnerID:         fief.OwnerID,
			TerritoryID:     fief.CapitalTerritoryID,
			FiefID:          fief.ID,
			FiefTitle:       fief.Title,
			FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
			NobleID:         nobleID,
			NobleCode:       models.NobleCode(noble.Code),
			NobleName:       ctx.state.NobleDisplayName(*noble),
			Reason:          "fief_auto_assigned_default_holder",
		})
	}
}

func (ctx *resolutionContext) dissolveFief(fief models.Fief, reason string) {
	ctx.removeFief(fief.ID)
	ctx.events = append(ctx.events, Event{
		Type:            EventTypeFiefDissolved,
		Phase:           phaseForSeason(ctx.state.Season),
		TerritoryID:     fief.CapitalTerritoryID,
		FiefID:          fief.ID,
		FiefTitle:       fief.Title,
		FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
		OwnerID:         fief.OwnerID,
		Reason:          reason,
	})
}
