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

// removeFief deletes the fief identified by fiefID from state.
func (ctx *resolutionContext) removeFief(fiefID models.FiefID) {
	filtered := make([]models.Fief, 0, len(ctx.state.Fiefs))
	for _, fief := range ctx.state.Fiefs {
		if fief.ID != fiefID {
			filtered = append(filtered, fief)
		}
	}
	ctx.state.Fiefs = filtered
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

// fortificationBonus is the defensive bonus a territory's castle provides:
// none without a castle, the city bonus when the castle sits on a fief's
// capital (it replaces, rather than stacks with, the plain castle bonus), or
// the plain castle bonus otherwise (titres.md).
func (ctx *resolutionContext) fortificationBonus(territoryID models.TerritoryID) int {
	if !ctx.hasCastle(territoryID) {
		return 0
	}
	if ctx.fiefByCapital(territoryID) != nil {
		return ctx.balance.CityDefenseBonus
	}
	return ctx.balance.CastleDefenseBonus
}

// transferFiefOnCapitalCapture hands a fief entirely to the player who just
// took its capital: the fief becomes vacant (its titulaire's authority does
// not carry over) but keeps producing and scoring until dissolved (titres.md).
// Control is transitive in a fief (titres.md "Contrôle et occupation"), so
// the capture also flips OwnerID on every other member still held by the
// previous owner, clearing that member's capital status if it carried one
// and reporting one control_changed event per member, reason
// "fief_transferred". It is a no-op when territoryID is not a fief capital,
// or when the new owner already held the fief (an intervening ally stop, not
// a capture).
func (ctx *resolutionContext) transferFiefOnCapitalCapture(territoryID models.TerritoryID, newOwnerID models.PlayerID) {
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
		state := ctx.state.TerritoryStates[memberID]
		if state.OwnerID != nil && *state.OwnerID == newOwnerID {
			continue
		}
		memberPreviousOwnerID := models.PlayerID("")
		if state.OwnerID != nil {
			memberPreviousOwnerID = *state.OwnerID
		}
		ownerID := newOwnerID
		state.OwnerID = &ownerID
		ctx.state.TerritoryStates[memberID] = state
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
			NobleName:       noble.Name,
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
