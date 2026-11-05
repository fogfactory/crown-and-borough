package models

import (
	"fmt"
	"slices"
	"sort"
)

// Claim records a pretension (specs/succession.md § Prétentions): the heir, a
// noble of one family placed while Target and Spouse were married, claims the
// fief titles of Target, a noble of the other family. Target and Spouse are
// the couple the claim bears on. When Target dies, the fief titles it holds
// pass to the heir. Turn is the absolute GameState.Turn of the winter that
// played the claim; WifeSide tells that the heir belongs to the wife's family.
// Claims stack: they rank by Turn, then wife's family first, then play order.
type Claim struct {
	Heir     NobleID `json:"heir"`
	Target   NobleID `json:"target"`
	Spouse   NobleID `json:"spouse"`
	Turn     int     `json:"turn"`
	WifeSide bool    `json:"wifeSide,omitempty"`
}

// ClaimOutcome tells how a claim ended.
type ClaimOutcome string

const (
	// ClaimHonoured: the heir received a fief or the crown of the target.
	ClaimHonoured ClaimOutcome = "honoured"
	// ClaimLapsed: the claim ended without effect (a dignity voided it, the
	// heir or the target died, another heir was served first).
	ClaimLapsed ClaimOutcome = "lapsed"
)

// ExpiredClaim is a claim that ended, kept like an ended marriage so the
// lineage still tells it (specs/succession.md § Prétentions). EndedTurn is
// the absolute GameState.Turn it ended on.
type ExpiredClaim struct {
	Claim
	EndedTurn int          `json:"endedTurn"`
	Outcome   ClaimOutcome `json:"outcome"`
}

// ExpireClaims moves to ExpiredClaims every claim for which outcomeOf returns
// true, with the outcome it gives, ended on the current turn.
func (g *GameState) ExpireClaims(outcomeOf func(Claim) (ClaimOutcome, bool)) {
	kept := g.Claims[:0:0]
	for _, claim := range g.Claims {
		if outcome, ended := outcomeOf(claim); ended {
			g.ExpiredClaims = append(g.ExpiredClaims, ExpiredClaim{Claim: claim, EndedTurn: g.Turn, Outcome: outcome})
		} else {
			kept = append(kept, claim)
		}
	}
	g.Claims = kept
}

// ClaimsOn returns the claims staked on the noble, by priority: oldest first,
// the wife's family first within a winter, then in play order.
func (g *GameState) ClaimsOn(target NobleID) []Claim {
	return g.claimsOnAny(target)
}

// claimsOnAny returns the claims staked on any of the nobles, ranked like
// ClaimsOn.
func (g *GameState) claimsOnAny(targets ...NobleID) []Claim {
	var claims []Claim
	for _, claim := range g.Claims {
		if slices.Contains(targets, claim.Target) {
			claims = append(claims, claim)
		}
	}
	sort.SliceStable(claims, func(i, j int) bool {
		if claims[i].Turn != claims[j].Turn {
			return claims[i].Turn < claims[j].Turn
		}
		return claims[i].WifeSide && !claims[j].WifeSide
	})
	return claims
}

// CrownHeirOf returns the noble who inherits the crown of the dead king
// (specs/politique.md § Claim sur le couple royal): the first living heir, in
// the usual claim ranking, of a claim on the king or on his queen (the spouse
// of any marriage of his that the pope did not dissolve) who can be king. An
// heir who cannot (a woman, a bastard, the pope) is skipped for the crown but
// keeps his claim on the fiefs. Call before the claims are settled.
func (g *GameState) CrownHeirOf(kingID NobleID) (NobleID, bool) {
	targets := []NobleID{kingID}
	for _, marriage := range g.Marriages {
		if !marriage.Dissolved && (marriage.NobleA == kingID || marriage.NobleB == kingID) {
			targets = append(targets, marriage.SpouseOf(kingID))
		}
	}
	for _, claim := range g.claimsOnAny(targets...) {
		for _, heir := range g.Nobles {
			if heir.ID == claim.Heir && heir.Sex.CanHoldReligiousOrRoyalTitle() && heir.CanBeKing() && !g.IsPope(heir.ID) {
				return heir.ID, true
			}
		}
	}
	return "", false
}

// PassCrownOfMissingKing crowns the heir of the king who left play, if the
// crown is held by a noble missing from Nobles and a claim on the royal couple
// names an eligible heir, and returns him. Otherwise it changes nothing: the
// throne stays vacant once DropReligiousTitlesOfMissingNobles runs. Call after
// the dead were removed and before the claims are settled.
func (g *GameState) PassCrownOfMissingKing() (NobleID, bool) {
	if g.King == nil || slices.ContainsFunc(g.Nobles, func(noble Noble) bool { return noble.ID == *g.King }) {
		return "", false
	}
	heir, found := g.CrownHeirOf(*g.King)
	if found {
		g.King = &heir
	}
	return heir, found
}

func hasClaim(claims []Claim, heir NobleID) bool {
	for _, claim := range claims {
		if claim.Heir == heir {
			return true
		}
	}
	return false
}

// ClaimOf returns the claim the heir holds.
func (g *GameState) ClaimOf(heir NobleID) (Claim, bool) {
	for _, claim := range g.Claims {
		if claim.Heir == heir {
			return claim, true
		}
	}
	return Claim{}, false
}

// MarriageCovering returns a marriage of target with a noble of the owner
// that was active on the given turn: concluded on or before it and not ended
// by a death before it. A marriage ends the turn a spouse died, and that turn
// still counts.
func (g *GameState) MarriageCovering(target NobleID, owner PlayerID, turn int) (Marriage, bool) {
	owners := make(map[NobleID]PlayerID, len(g.Nobles)+len(g.RemovedNobles))
	deaths := make(map[NobleID]int, len(g.RemovedNobles))
	for _, noble := range g.Nobles {
		owners[noble.ID] = noble.OwnerID
	}
	for _, removed := range g.RemovedNobles {
		owners[removed.ID] = removed.OwnerID
		deaths[removed.ID] = removed.Turn
	}
	for _, marriage := range g.Marriages {
		spouse := marriage.NobleA
		if spouse == target {
			spouse = marriage.NobleB
		} else if marriage.NobleB != target {
			continue
		}
		if owners[spouse] != owner || marriage.Turn > turn {
			continue
		}
		if marriage.Dissolved && marriage.DissolvedTurn < turn {
			continue
		}
		if died, dead := deaths[spouse]; dead && died < turn {
			continue
		}
		return marriage, true
	}
	return Marriage{}, false
}

// SpouseOf returns the other spouse of the marriage.
func (m Marriage) SpouseOf(nobleID NobleID) NobleID {
	if m.NobleA == nobleID {
		return m.NobleB
	}
	return m.NobleA
}

// validateClaims checks every claim: an heir holds at most one claim, on a
// living noble of another player, bearing on a known spouse whose house is the
// heir's (an heir is born of the alliance between the target and the spouse).
func (g *GameState) validateClaims(nobles, removedNobles map[NobleID]bool) error {
	owners := make(map[NobleID]PlayerID, len(g.Nobles))
	for _, noble := range g.Nobles {
		owners[noble.ID] = noble.OwnerID
	}
	spouseOwners := make(map[NobleID]PlayerID, len(g.Nobles)+len(g.RemovedNobles))
	for _, removed := range g.RemovedNobles {
		spouseOwners[removed.ID] = removed.OwnerID
	}
	for _, noble := range g.Nobles {
		spouseOwners[noble.ID] = noble.OwnerID
	}
	heirs := make(map[NobleID]bool, len(g.Claims))
	for _, claim := range g.Claims {
		heirOwner, heirAlive := owners[claim.Heir]
		targetOwner, targetAlive := owners[claim.Target]
		switch {
		case !heirAlive:
			return fmt.Errorf("models: claim: unknown heir %q", claim.Heir)
		case !targetAlive:
			return fmt.Errorf("models: claim: unknown target %q", claim.Target)
		case !nobles[claim.Spouse] && !removedNobles[claim.Spouse]:
			return fmt.Errorf("models: claim %q on %q: unknown spouse %q", claim.Heir, claim.Target, claim.Spouse)
		case heirOwner == targetOwner:
			return fmt.Errorf("models: claim %q on %q: heir and target belong to the same player", claim.Heir, claim.Target)
		case spouseOwners[claim.Spouse] != heirOwner:
			return fmt.Errorf("models: claim %q on %q: the heir must belong to the house of the spouse %q", claim.Heir, claim.Target, claim.Spouse)
		case heirs[claim.Heir]:
			return fmt.Errorf("models: claim: heir %q holds more than one claim", claim.Heir)
		case claim.Turn < 0 || claim.Turn > g.Turn:
			return fmt.Errorf("models: claim %q on %q: turn %d must be between 0 and %d", claim.Heir, claim.Target, claim.Turn, g.Turn)
		}
		heirs[claim.Heir] = true
	}
	return nil
}
