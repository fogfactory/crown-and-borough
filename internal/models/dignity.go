package models

import "slices"

// Dignity is a permanent distinction a noble carries (specs/succession.md
// § Bâtard, specs/dames.md § Dignités). The bastard is the only one so far.
type Dignity string

const (
	DignityBastard Dignity = "bastard"
)

// DignityBastardCardCode is the code of the bastard dignity card in D N XXX CCC.
const DignityBastardCardCode = "BAS"

// DignityEffect declares everything a dignity changes in the rules. Every
// rule that depends on a dignity (noble cap, succession, titles, kingship,
// marriage, capture, card targeting) reads it through the Noble helpers
// below instead of testing a dignity by name: a new dignity adds an entry to
// dignityEffects rather than a scattered test (specs/succession.md § Bâtard).
type DignityEffect struct {
	// CardCode is the code of the dignity card that confers it.
	CardCode string
	// NobleLimitBonus raises the owner's noble cap while a noble carries it.
	NobleLimitBonus int
	// LastInSuccession places the carrier after every other noble in the line
	// of succession and restricts its titles to the last of the lineage.
	LastInSuccession bool
	// CannotBeKing forbids the carrier to become king.
	CannotBeKing bool
	// MarriageIsNotAlliance makes the carrier's marriage a plain record with
	// no alliance weight, category, score sharing or density bonus.
	MarriageIsNotAlliance bool
	// CapturedToDungeon sends the carrier straight to the dungeon when it is
	// captured in combat instead of making it a hostage.
	CapturedToDungeon bool
	// VoidsClaims forbids the carrier to claim titles and cancels the claim
	// it already holds as heir.
	VoidsClaims bool
	// TargetPredicate is the targeting predicate special cards use to
	// recognise the carrier.
	TargetPredicate string
}

// dignityEffects is the single dignity registry.
var dignityEffects = map[Dignity]DignityEffect{
	DignityBastard: {
		CardCode:              DignityBastardCardCode,
		NobleLimitBonus:       1,
		LastInSuccession:      true,
		CannotBeKing:          true,
		MarriageIsNotAlliance: true,
		CapturedToDungeon:     true,
		VoidsClaims:           true,
		TargetPredicate:       "is_bastard",
	},
}

// IsValid reports whether the dignity is a known value.
func (d Dignity) IsValid() bool {
	_, exists := dignityEffects[d]
	return exists
}

// Effect returns the declared effect of the dignity.
func (d Dignity) Effect() DignityEffect { return dignityEffects[d] }

// DignityForCardCode resolves the dignity a dignity card code confers.
func DignityForCardCode(code string) (Dignity, bool) {
	for dignity, effect := range dignityEffects {
		if effect.CardCode == code {
			return dignity, true
		}
	}
	return "", false
}

// Has reports whether the noble carries the dignity.
func (n Noble) Has(dignity Dignity) bool { return slices.Contains(n.Dignities, dignity) }

// IsBastard is the is_bastard targeting predicate.
func (n Noble) IsBastard() bool { return n.Has(DignityBastard) }

func (n Noble) anyDignity(test func(DignityEffect) bool) bool {
	for _, dignity := range n.Dignities {
		if test(dignityEffects[dignity]) {
			return true
		}
	}
	return false
}

// NobleLimitBonus is the noble cap bonus the noble's dignities grant its
// owner (before the noble_limit_max clamp).
func (n Noble) NobleLimitBonus() int {
	bonus := 0
	for _, dignity := range n.Dignities {
		bonus += dignityEffects[dignity].NobleLimitBonus
	}
	return bonus
}

// LastInSuccession reports whether a dignity places the noble last in the
// line of succession.
func (n Noble) LastInSuccession() bool {
	return n.anyDignity(func(e DignityEffect) bool { return e.LastInSuccession })
}

// CanBeKing reports whether no dignity forbids kingship. Kingship itself is
// not implemented yet; this is the hook the crown rules will query.
func (n Noble) CanBeKing() bool {
	return !n.anyDignity(func(e DignityEffect) bool { return e.CannotBeKing })
}

// MarriageIsAlliance reports whether a marriage of this noble counts as an
// alliance. Alliance weights are not implemented yet; this is the hook they
// will query.
func (n Noble) MarriageIsAlliance() bool {
	return !n.anyDignity(func(e DignityEffect) bool { return e.MarriageIsNotAlliance })
}

// CapturedToDungeon reports whether the noble goes straight to the dungeon
// when captured in combat.
func (n Noble) CapturedToDungeon() bool {
	return n.anyDignity(func(e DignityEffect) bool { return e.CapturedToDungeon })
}

// VoidsClaims reports whether the noble can no longer be the heir of a claim.
func (n Noble) VoidsClaims() bool {
	return n.anyDignity(func(e DignityEffect) bool { return e.VoidsClaims })
}
