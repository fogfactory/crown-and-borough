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
	fief := ctx.fiefContaining(territoryID)
	if fief == nil {
		return false
	}
	if ctx.taxedFiefsThisTurn[fief.ID] || ctx.pendingTaxWindowFiefs[fief.ID] {
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
// capital is inert, like any other unanchored infrastructure (#215, #193).
func (ctx *resolutionContext) fortificationBonus(territoryID models.TerritoryID) int {
	if !ctx.hasCastle(territoryID) && !ctx.hasFortifiedVillage(territoryID) {
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

// transferFiefOnCapitalCapture hands a fief entirely to the player who just
// took its capital: the fief becomes vacant (its titulaire's authority does
// not carry over) but keeps producing and scoring until dissolved (titres.md).
// Control is transitive in a fief (titres.md "Contrôle et occupation"), so
// the capture also hands every other member still held by the previous owner
// to the new one in control, the control pass's working view of who controls
// what, clearing that member's capital status if it carried one and reporting
// one control_changed event per member, reason "fief_transferred". It is a
// no-op when territoryID is not a fief capital, or when the new owner already
// held the fief (an intervening ally stop, not a capture).
func (ctx *resolutionContext) transferFiefOnCapitalCapture(territoryID models.TerritoryID, newOwnerID models.PlayerID, control map[models.TerritoryID]models.PlayerID) {
	fief := ctx.fiefByCapital(territoryID)
	if fief == nil || fief.OwnerID == newOwnerID {
		return
	}
	previousOwnerID := fief.OwnerID
	fief.OwnerID = newOwnerID
	fief.HolderNobleID = nil
	ctx.events = append(ctx.events, Event{
		Type:            EventTypeFiefConquered,
		Phase:           5,
		TerritoryID:     fief.CapitalTerritoryID,
		FiefID:          fief.ID,
		FiefTitle:       fief.Title,
		FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
		PreviousOwnerID: previousOwnerID,
		OwnerID:         newOwnerID,
	})
	for _, memberID := range fief.Territories {
		if memberID == fief.CapitalTerritoryID {
			continue
		}
		memberPreviousOwnerID, controlled := control[memberID]
		if controlled && memberPreviousOwnerID == newOwnerID {
			continue
		}
		ownerID := newOwnerID
		control[memberID] = ownerID
		ctx.clearCapitalOnControlLoss(memberPreviousOwnerID, memberID)
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeControlChanged,
			Phase:           5,
			TerritoryID:     memberID,
			PreviousOwnerID: memberPreviousOwnerID,
			OwnerID:         ownerID,
			Reason:          "fief_transferred",
		})
	}
}

// vacateFiefsOfMissingHolders clears the titulaire of every fief whose noble
// no longer exists (death, e.g. from plague): the fief stays with its owner,
// vacant, until it is attributed again or dissolved (titres.md). Call after
// the caller has removed the dead nobles and rebuilt indexes.
func (ctx *resolutionContext) vacateFiefsOfMissingHolders() {
	for i := range ctx.state.Fiefs {
		fief := &ctx.state.Fiefs[i]
		if fief.HolderNobleID == nil {
			continue
		}
		if ctx.noblesByID[*fief.HolderNobleID] != nil {
			continue
		}
		fief.HolderNobleID = nil
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeFiefVacated,
			Phase:           phaseForSeason(ctx.state.Season),
			TerritoryID:     fief.CapitalTerritoryID,
			FiefID:          fief.ID,
			FiefTitle:       fief.Title,
			FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
			OwnerID:         fief.OwnerID,
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
// never dissolved for lack of attribution any more (titres.md "Perte et
// vacance d'un fief"): when its owner still has a free noble, the fief is
// attributed by default to the one whose trigram sorts first, with a warning
// event telling the player to take back manual attribution next turn.
// Without any free noble, the fief simply stays vacant: it keeps producing
// and scoring until attributed or dissolved by its capital's castle falling
// (dissolveFiefOnCapitalCastleLoss, unaffected by this function).
func (ctx *resolutionContext) resolveVacantFiefsAtWinterEnd() {
	for i := range ctx.state.Fiefs {
		fief := &ctx.state.Fiefs[i]
		if fief.HolderNobleID != nil {
			continue
		}
		nobleID, exists := ctx.smallestFreeNoble(fief.OwnerID)
		if !exists {
			continue
		}
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

// smallestFreeNoble returns the id of playerID's free noble whose trigram
// sorts first, for the deterministic default fief attribution above. It is
// the same eligibility as the T A order (assignFiefOrder): owned by
// playerID, NobleStatusFree (so implicitly alive and uncaptured). Holding
// another fief's title is not disqualifying: a noble may hold several
// (titres.md "Constitution d'un fief").
func (ctx *resolutionContext) smallestFreeNoble(playerID models.PlayerID) (models.NobleID, bool) {
	var best *models.Noble
	for i := range ctx.state.Nobles {
		noble := &ctx.state.Nobles[i]
		if noble.OwnerID != playerID || noble.Status != models.NobleStatusFree {
			continue
		}
		if best == nil || noble.Code < best.Code {
			best = noble
		}
	}
	if best == nil {
		return "", false
	}
	return best.ID, true
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
