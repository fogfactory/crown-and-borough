package engine

import (
	"slices"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// claimOrder is C N HHH CCC (specs/succession.md § Prétentions): the player's
// noble HHH, the heir, claims the fief titles of CCC, a noble of another
// player. It consumes a claim card from the player's noble hand. HHH must
// have been placed while CCC was married to one of the player's nobles; it
// must not be a bastard nor already hold a claim. Claims stack: when CCC dies
// they rank by age, then wife's family first. It costs no R.
type claimOrder struct{ order models.WinterOrder }

func (order claimOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	heirID, heirExists := resolution.noblesByCode[winterOrder.NobleCode]
	targetID, targetExists := resolution.noblesByCode[models.NobleCode(winterOrder.SpouseCode)]
	if !heirExists || !targetExists {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	heir, target := resolution.noblesByID[heirID], resolution.noblesByID[targetID]
	if heir == nil || target == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	if heir.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_not_owned")
		return
	}
	if target.OwnerID == playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "claim_on_own_noble")
		return
	}
	if heir.VoidsClaims() {
		resolution.rejectWinterOrder(playerID, winterOrder, "claim_by_bastard")
		return
	}
	if _, claiming := resolution.state.ClaimOf(heir.ID); claiming {
		resolution.rejectWinterOrder(playerID, winterOrder, "claim_already_staked")
		return
	}
	marriage, covered := resolution.state.MarriageCovering(target.ID, playerID, heir.PlacedTurn)
	if !covered {
		resolution.rejectWinterOrder(playerID, winterOrder, "claim_requires_marriage")
		return
	}
	spouseID := marriage.SpouseOf(target.ID)
	_, handIndex, inHand := resolution.state.NobleDeck.HandCard(playerID, models.NobleCardKindClaim, models.ClaimCardCode)
	if !inHand {
		resolution.rejectWinterOrder(playerID, winterOrder, "card_not_in_hand")
		return
	}
	if target.Has(models.DignityChevalierDEon) {
		resolution.discardHandCard(playerID, handIndex)
		resolution.loseClaimToMarriedEon(playerID, winterOrder, heir, target, marriage)
		return
	}
	resolution.consumeNobleCard(playerID, handIndex, heir.ID)
	resolution.state.Claims = append(resolution.state.Claims, models.Claim{
		Heir: heir.ID, Target: target.ID, Spouse: spouseID, Turn: resolution.state.Turn,
		WifeSide: resolution.wifeFamily(playerID, target, spouseID),
	})
	resolution.events = append(resolution.events, Event{
		Type:            EventTypeClaim,
		Phase:           winterPhase,
		OwnerID:         playerID,
		OrderID:         winterOrder.ID,
		NobleID:         heir.ID,
		NobleCode:       models.NobleCode(heir.Code),
		NobleName:       resolution.state.NobleDisplayName(*heir),
		SpouseNobleID:   target.ID,
		SpouseNobleCode: models.NobleCode(target.Code),
		SpouseNobleName: resolution.state.NobleDisplayName(*target),
		SpouseOwnerID:   target.OwnerID,
	})
}

// loseClaimToMarriedEon resolves a Claim played against a married chevalier
// d'Éon (specs/dames.md § Chevalier d'Éon): the target becomes a bastard, the
// Éon is unmasked, the marriage that justified the Claim is annulled, and the
// Claim itself is lost (its card was already consumed; no Claim is recorded).
func (ctx *resolutionContext) loseClaimToMarriedEon(playerID models.PlayerID, order models.WinterOrder, heir, target *models.Noble, marriage models.Marriage) {
	target.Dignities = append(target.Dignities, models.DignityBastard)
	ctx.events = append(ctx.events, Event{
		Type: EventTypeDignity, Phase: winterPhase, OwnerID: target.OwnerID, OrderID: order.ID,
		NobleID: target.ID, NobleCode: models.NobleCode(target.Code), NobleName: ctx.state.NobleDisplayName(*target),
		Dignity: models.DignityBastard, Reason: "claim_against_married_eon",
	})
	ctx.unmaskEon(target, winterPhase)
	ctx.state.Marriages = slices.DeleteFunc(ctx.state.Marriages, func(m models.Marriage) bool {
		return m.NobleA == marriage.NobleA && m.NobleB == marriage.NobleB && m.Turn == marriage.Turn
	})
	ctx.events = append(ctx.events, Event{
		Type: EventTypeClaim, Phase: winterPhase, OwnerID: playerID, OrderID: order.ID,
		NobleID: heir.ID, NobleCode: models.NobleCode(heir.Code), NobleName: ctx.state.NobleDisplayName(*heir),
		SpouseNobleID: target.ID, SpouseNobleCode: models.NobleCode(target.Code), SpouseNobleName: ctx.state.NobleDisplayName(*target),
		SpouseOwnerID: target.OwnerID, Reason: "claim_lost_married_eon",
	})
}

// wifeFamily reports whether the player owns the wife of the couple formed
// by target and the spouse with the given ID. A spouse who died is read from
// the removed nobles.
func (ctx *resolutionContext) wifeFamily(playerID models.PlayerID, target *models.Noble, spouseID models.NobleID) bool {
	if target.Sex == models.SexFemale {
		return target.OwnerID == playerID
	}
	if spouse := ctx.noblesByID[spouseID]; spouse != nil {
		return spouse.OwnerID == playerID
	}
	for _, removed := range ctx.state.RemovedNobles {
		if removed.ID == spouseID {
			return removed.OwnerID == playerID
		}
	}
	return false
}

// voidClaimOf cancels the claim the heir holds once a dignity forbids it
// (specs/succession.md § Bâtard).
func (ctx *resolutionContext) voidClaimOf(heirID models.NobleID) {
	ctx.state.ExpireClaims(func(claim models.Claim) (models.ClaimOutcome, bool) {
		return models.ClaimLapsed, claim.Heir == heirID
	})
	ctx.discardClaimCard(heirID)
}

// honourClaimOf records that the heir received what its claim bore on (a fief
// or the crown): settleClaimsOfDead then ends the claim as honoured.
func (ctx *resolutionContext) honourClaimOf(heirID models.NobleID) {
	if ctx.honouredHeirs == nil {
		ctx.honouredHeirs = make(map[models.NobleID]bool)
	}
	ctx.honouredHeirs[heirID] = true
}

// discardClaimCard sends the claim card played on the heir back to the
// discard pile once its claim is over.
func (ctx *resolutionContext) discardClaimCard(heirID models.NobleID) {
	deck := ctx.state.NobleDeck
	if deck == nil {
		return
	}
	for index, play := range deck.Played {
		if play.Noble != heirID {
			continue
		}
		if card, exists := deck.Card(play.Card); exists && card.Kind == models.NobleCardKindClaim {
			deck.Played = slices.Delete(deck.Played, index, index+1)
			deck.Discard = append(deck.Discard, play.Card)
			return
		}
	}
}

// settleClaimsOfDead ends the claims whose heir or target left play and those
// whose heir was served (a fief or the crown), after the dead nobles were
// removed and the indexes rebuilt: the first end as lapsed unless the heir
// was served, the second as honoured. Ended claims are kept in ExpiredClaims.
// A claim on a dead noble has been honoured by vacateFiefsOfMissingHolders
// first.
func (ctx *resolutionContext) settleClaimsOfDead() {
	honoured := ctx.honouredHeirs
	ctx.honouredHeirs = nil
	ctx.state.ExpireClaims(func(claim models.Claim) (models.ClaimOutcome, bool) {
		heirAlive := ctx.noblesByID[claim.Heir] != nil
		if !heirAlive || ctx.noblesByID[claim.Target] == nil || honoured[claim.Heir] {
			if heirAlive {
				// A living heir gets its card back; a dead one already released it.
				ctx.discardClaimCard(claim.Heir)
			}
			if honoured[claim.Heir] {
				return models.ClaimHonoured, true
			}
			return models.ClaimLapsed, true
		}
		return "", false
	})
}
