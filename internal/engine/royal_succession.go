package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// settleAfterDeaths applies the aftermath of deaths on the titles, once the
// dead nobles were removed and the indexes rebuilt: fiefs pass to the heir of a
// claim or fall vacant, the crown of a dead king passes to the heir of a claim
// on the royal couple, religious titles of the dead end, and the claims left
// without a living heir or target are settled.
func (ctx *resolutionContext) settleAfterDeaths() {
	ctx.state.RefreshQueen()
	ctx.vacateFiefsOfMissingHolders()
	ctx.passCrownOfDeadKing()
	ctx.state.DropReligiousTitlesOfMissingNobles()
	ctx.settleClaimsOfDead()
	ctx.state.RefreshQueen()
	ctx.returnOrphanFiefsToCrown()
}

// passCrownOfDeadKing crowns the heir of a claim on the dead king or his queen
// (specs/politique.md § Claim sur le couple royal): no election, no anointing,
// and the heir may hold any rank in his line. Without a king who left play, or
// without an eligible heir, nothing happens: DropReligiousTitlesOfMissingNobles
// then leaves the throne vacant.
func (ctx *resolutionContext) passCrownOfDeadKing() {
	heirID, crowned := ctx.state.PassCrownOfMissingKing()
	if !crowned {
		return
	}
	ctx.honourClaimOf(heirID)
	heir := ctx.noblesByID[heirID]
	ctx.events = append(ctx.events, Event{
		Type:      EventTypeInvestiture,
		Phase:     phaseForSeason(ctx.state.Season),
		OwnerID:   heir.OwnerID,
		NobleID:   heir.ID,
		NobleCode: models.NobleCode(heir.Code),
		NobleName: ctx.state.NobleDisplayName(*heir),
		Reason:    "claim",
		Election:  &ElectionResult{Kind: models.ElectionKing, Candidates: []ElectionCandidate{}},
	})
}
