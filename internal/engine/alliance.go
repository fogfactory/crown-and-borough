package engine

import (
	"slices"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// nobleAllianceWeight is the succession rank plus the title rank of a living
// noble (specs/succession.md § Poids d'alliance). It reports false when the
// noble is not alive.
func nobleAllianceWeight(state *models.GameState, balance assetgen.AllianceBalance, nobleID models.NobleID) (int, bool) {
	var owner models.PlayerID
	found := false
	for _, noble := range state.Nobles {
		if noble.ID == nobleID {
			owner, found = noble.OwnerID, true
			break
		}
	}
	if !found {
		return 0, false
	}
	weight := 0
	ranks := balance.SuccessionRanks
	for position, noble := range state.SuccessionLine(owner) {
		if noble.ID == nobleID && len(ranks) > 0 {
			weight = ranks[min(position, len(ranks)-1)]
			break
		}
	}
	if fief := state.HighestHeldFief(nobleID); fief != nil {
		weight += balance.TitleRanks[fief.Title]
	}
	return weight, true
}

// isAlliance reports whether the marriage is an active alliance: both spouses
// alive and neither carrying a dignity that voids the alliance (the bastard).
func isAlliance(state *models.GameState, marriage models.Marriage) bool {
	if !marriage.Active(state) {
		return false
	}
	for _, noble := range state.Nobles {
		if (noble.ID == marriage.NobleA || noble.ID == marriage.NobleB) && !noble.MarriageIsAlliance() {
			return false
		}
	}
	return true
}

// marriageHouses returns the owners of the two spouses.
func marriageHouses(state *models.GameState, marriage models.Marriage) (models.PlayerID, models.PlayerID) {
	var a, b models.PlayerID
	for _, noble := range state.Nobles {
		switch noble.ID {
		case marriage.NobleA:
			a = noble.OwnerID
		case marriage.NobleB:
			b = noble.OwnerID
		}
	}
	return a, b
}

// AllianceWeight is the alliance weight of a marriage: the weaker spouse's
// weight plus DensityBonus per other alliance between the same two players,
// uncapped. It reports false when the marriage is not an active alliance
// (ended by a death, or involving a bastard).
func AllianceWeight(state *models.GameState, balance assetgen.Balance, marriage models.Marriage) (int, bool) {
	if state == nil || !isAlliance(state, marriage) {
		return 0, false
	}
	weightA, _ := nobleAllianceWeight(state, balance.Alliance, marriage.NobleA)
	weightB, _ := nobleAllianceWeight(state, balance.Alliance, marriage.NobleB)
	weight := min(weightA, weightB)

	houseA, houseB := marriageHouses(state, marriage)
	for _, other := range state.Marriages {
		if (other.NobleA == marriage.NobleA && other.NobleB == marriage.NobleB) || !isAlliance(state, other) {
			continue
		}
		otherA, otherB := marriageHouses(state, other)
		if (otherA == houseA && otherB == houseB) || (otherA == houseB && otherB == houseA) {
			weight += balance.Alliance.DensityBonus
		}
	}
	return weight, true
}

// AllianceCategory ranks a marriage by alliance weight.
type AllianceCategory string

const (
	AllianceHead      AllianceCategory = "head"
	AllianceMixed     AllianceCategory = "mixed"
	AllianceSecondary AllianceCategory = "secondary"
)

// MarriageCategory classifies an active alliance by its current weight. It is
// derived from the state on every call, so deaths, executions, assassinations,
// claims and title changes reclassify the marriage without any stored value.
// It reports false when the marriage is not an active alliance.
func MarriageCategory(state *models.GameState, balance assetgen.Balance, marriage models.Marriage) (AllianceCategory, bool) {
	weight, ok := AllianceWeight(state, balance, marriage)
	if !ok {
		return "", false
	}
	switch {
	case weight >= balance.Alliance.HeadMinWeight:
		return AllianceHead, true
	case weight >= balance.Alliance.MixedMinWeight:
		return AllianceMixed, true
	default:
		return AllianceSecondary, true
	}
}

// activeHeads assigns each house its active head (specs/succession.md § Tête
// active). Head-category alliances are taken by descending alliance weight,
// ties going to the oldest marriage, then to the first recorded; one is kept
// only if neither of its houses already has a head, so a marriage is a head
// for both houses or for neither. Every other head-weight marriage counts as
// mixed for both. It is derived from the state on every call, so deaths,
// executions and title changes reassign heads with no stored value.
func activeHeads(state *models.GameState, balance assetgen.Balance) map[models.PlayerID]models.Marriage {
	type candidate struct {
		marriage models.Marriage
		weight   int
	}
	var candidates []candidate
	for _, marriage := range state.Marriages {
		houseA, houseB := marriageHouses(state, marriage)
		if houseA == houseB {
			continue
		}
		if category, ok := MarriageCategory(state, balance, marriage); !ok || category != AllianceHead {
			continue
		}
		weight, _ := AllianceWeight(state, balance, marriage)
		candidates = append(candidates, candidate{marriage, weight})
	}
	slices.SortStableFunc(candidates, func(a, b candidate) int {
		if a.weight != b.weight {
			return b.weight - a.weight
		}
		return a.marriage.Turn - b.marriage.Turn
	})
	heads := map[models.PlayerID]models.Marriage{}
	for _, c := range candidates {
		houseA, houseB := marriageHouses(state, c.marriage)
		if _, taken := heads[houseA]; taken {
			continue
		}
		if _, taken := heads[houseB]; taken {
			continue
		}
		heads[houseA], heads[houseB] = c.marriage, c.marriage
	}
	return heads
}

// ActiveHeadMarriage returns the player's active head marriage, if any.
func ActiveHeadMarriage(state *models.GameState, balance assetgen.Balance, player models.PlayerID) (models.Marriage, bool) {
	if state == nil {
		return models.Marriage{}, false
	}
	marriage, found := activeHeads(state, balance)[player]
	return marriage, found
}

// EffectiveMarriageCategory is the category a marriage has, the same for both
// houses: a head-weight marriage that is not the active head of its houses
// counts as mixed. It reports false when the marriage is not an active
// alliance or does not involve the player.
func EffectiveMarriageCategory(state *models.GameState, balance assetgen.Balance, player models.PlayerID, marriage models.Marriage) (AllianceCategory, bool) {
	houseA, houseB := marriageHouses(state, marriage)
	if houseA != player && houseB != player {
		return "", false
	}
	category, ok := MarriageCategory(state, balance, marriage)
	if !ok {
		return "", false
	}
	if category != AllianceHead {
		return category, true
	}
	if active, found := activeHeads(state, balance)[player]; found && active.NobleA == marriage.NobleA && active.NobleB == marriage.NobleB {
		return AllianceHead, true
	}
	return AllianceMixed, true
}
