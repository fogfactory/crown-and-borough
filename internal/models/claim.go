package models

import (
	"fmt"
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

// ClaimsOn returns the claims staked on the noble, by priority: oldest first,
// the wife's family first within a winter, then in play order.
func (g *GameState) ClaimsOn(target NobleID) []Claim {
	var claims []Claim
	for _, claim := range g.Claims {
		if claim.Target == target {
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
// living noble of another player, bearing on a known spouse.
func (g *GameState) validateClaims(nobles, removedNobles map[NobleID]bool) error {
	owners := make(map[NobleID]PlayerID, len(g.Nobles))
	for _, noble := range g.Nobles {
		owners[noble.ID] = noble.OwnerID
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
		case heirs[claim.Heir]:
			return fmt.Errorf("models: claim: heir %q holds more than one claim", claim.Heir)
		case claim.Turn < 0 || claim.Turn > g.Turn:
			return fmt.Errorf("models: claim %q on %q: turn %d must be between 0 and %d", claim.Heir, claim.Target, claim.Turn, g.Turn)
		}
		heirs[claim.Heir] = true
	}
	return nil
}
