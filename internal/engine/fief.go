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
// It is a no-op when territoryID is not a fief capital, or when the new owner
// already held the fief (an intervening ally stop, not a capture).
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

// dissolveVacantFiefs dissolves every fief still vacant at the end of winter:
// its territories simply become controlled outside any fief (titres.md).
func (ctx *resolutionContext) dissolveVacantFiefs() {
	for _, fief := range append([]models.Fief(nil), ctx.state.Fiefs...) {
		if fief.HolderNobleID != nil {
			continue
		}
		ctx.dissolveFief(fief, "vacant_at_winter_end")
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
