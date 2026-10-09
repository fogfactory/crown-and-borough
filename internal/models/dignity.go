package models

import "slices"

// Dignity is a permanent distinction a noble carries (specs/succession.md
// § Bâtard, specs/dames.md § Dignités): the bastard, open to any noble, and
// the dignities of the ladies.
type Dignity string

const (
	DignityBastard    Dignity = "bastard"
	DignityDArc       Dignity = "d_arc"
	DignityCastellan  Dignity = "castellan"
	DignityAbbess     Dignity = "abbess"
	DignityHerbalist  Dignity = "herbalist"
	DignityAstrologer Dignity = "astrologer"
	// The dignities below are hidden: only the carrier's owner knows them.
	DignityChevalierDEon Dignity = "chevalier_d_eon"
	DignityCorrespondent Dignity = "correspondent"
	DignitySpy           Dignity = "spy"
	DignityWitch         Dignity = "witch"
)

// DignityCardinal is carried by a cardinal that obtained the title with a
// cardinal card of the noble deck; a purchased cardinal has no card and does
// not carry it. The card lies on the bishop from the moment it is played, the
// title itself follows at the investiture; the dignity is lost, with the card
// going back to the discard pile, when the title ends.
const DignityCardinal Dignity = "cardinal"

// Codes of the dignity cards in D N XXX CCC.
const (
	DignityBastardCardCode    = "BAS"
	DignityDArcCardCode       = "ARC"
	DignityCastellanCardCode  = "CTL"
	DignityAbbessCardCode     = "ABB"
	DignityHerbalistCardCode  = "HRB"
	DignityAstrologerCardCode = "AST"
	DignityEonCardCode        = "EON"
	DignityCorrespondentCode  = "COR"
	DignitySpyCardCode        = "ESP"
	DignityWitchCardCode      = "SOR"
	DignityCardinalCardCode   = "CAR"
)

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
	// FemaleOnly restricts the dignity to female nobles (specs/dames.md
	// § Dignités).
	FemaleOnly bool
	// RequiresUnmarried forbids the dignity card on a married noble.
	RequiresUnmarried bool
	// MarriageBlocked forbids the carrier to marry after the nomination.
	MarriageBlocked bool
	// Hidden makes the dignity known to the carrier's owner only.
	Hidden bool
	// ArmyForceBonus is added to the force of an army the free carrier
	// commands, on top of the noble command bonus.
	ArmyForceBonus int
	// PlagueImmune protects the troops and nobles on the carrier's territory
	// and the adjacent ones from plague.
	PlagueImmune bool
	// RationDiscount lowers the ration cost of the armies standing on the
	// carrier's territory, down to zero.
	RationDiscount int
	// CalamityForecast is the number of upcoming calamities the owner of the
	// free carrier sees in winter.
	CalamityForecast int
	// SeesFiefOrders lets the owner of the carrier, standing in a castle, and
	// whoever holds her hostage there, know the orders given on its fief.
	SeesFiefOrders bool
	// NeedsRegion makes the card name the region (bishopric) the carrier is
	// attached to, for good.
	NeedsRegion bool
	// SeesAbbeyOrders reveals the order chains started in the carrier's region
	// while she is in it.
	SeesAbbeyOrders bool
	// SeesHostOrders reveals to the owner every order the player holding the
	// carrier hostage gives.
	SeesHostOrders bool
	// SeesHostHand reveals to the owner the whole hand of the player holding
	// the carrier hostage.
	SeesHostHand bool
	// RivalConsumption is the extra ration cost per turn of every army that is
	// not the owner's in the carrier's region.
	RivalConsumption int
	// ExemptFromDirectTrial keeps the carrier out of reach of the direct trial:
	// she can only be tried once excommunicated (specs/dames.md § Carte de
	// procès).
	ExemptFromDirectTrial bool
	// ChangesSexToMale turns the carrier into a male noble when it is played
	// (the chevalier d'Éon).
	ChangesSexToMale bool
}

// dignityEffects is the single dignity registry.
var dignityEffects = map[Dignity]DignityEffect{
	DignityCardinal: {CardCode: DignityCardinalCardCode},
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
	DignityDArc: {
		CardCode: DignityDArcCardCode, FemaleOnly: true, RequiresUnmarried: true,
		MarriageBlocked: true, ArmyForceBonus: 1,
	},
	DignityCastellan: {CardCode: DignityCastellanCardCode, FemaleOnly: true, SeesFiefOrders: true},
	DignityAbbess: {
		CardCode: DignityAbbessCardCode, FemaleOnly: true, RequiresUnmarried: true,
		MarriageBlocked: true, NeedsRegion: true, SeesAbbeyOrders: true, ExemptFromDirectTrial: true,
	},
	DignityHerbalist: {
		CardCode: DignityHerbalistCardCode, FemaleOnly: true, RequiresUnmarried: true,
		MarriageBlocked: true, PlagueImmune: true, RationDiscount: 2,
	},
	DignityAstrologer: {CardCode: DignityAstrologerCardCode, FemaleOnly: true, CalamityForecast: 3},
	DignityChevalierDEon: {
		CardCode: DignityEonCardCode, FemaleOnly: true, RequiresUnmarried: true,
		Hidden: true, ChangesSexToMale: true,
	},
	DignityCorrespondent: {CardCode: DignityCorrespondentCode, FemaleOnly: true, Hidden: true, SeesHostOrders: true},
	DignitySpy:           {CardCode: DignitySpyCardCode, FemaleOnly: true, Hidden: true, SeesHostHand: true},
	DignityWitch:         {CardCode: DignityWitchCardCode, FemaleOnly: true, Hidden: true, RivalConsumption: 1},
}

// LadyDignities lists the dignities of the ladies, in deck order.
var LadyDignities = []Dignity{
	DignityDArc, DignityCastellan, DignityAbbess, DignityHerbalist, DignityAstrologer,
	DignityChevalierDEon, DignityCorrespondent, DignitySpy, DignityWitch,
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
// alliance (see engine.AllianceWeight).
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

// MarriageBlockedByDignity reports whether a dignity forbids the noble to
// marry (the Bloqué dignities).
func (n Noble) MarriageBlockedByDignity() bool {
	return n.anyDignity(func(e DignityEffect) bool { return e.MarriageBlocked })
}

// HasHiddenDignity reports whether the noble carries a hidden dignity.
func (n Noble) HasHiddenDignity() bool {
	return n.anyDignity(func(e DignityEffect) bool { return e.Hidden })
}

// ArmyForceBonus is the force the noble adds to an army it commands through
// its dignities, only while it is free: a prisoner loses its bonuses.
func (n Noble) ArmyForceBonus() int {
	if n.Status != NobleStatusFree {
		return 0
	}
	bonus := 0
	for _, dignity := range n.Dignities {
		bonus += dignityEffects[dignity].ArmyForceBonus
	}
	return bonus
}

// DignityActive reports whether the noble's dignities still apply: a prisoner
// (dungeon) loses them, a hostage keeps them (specs/dames.md § Dignités).
func (n Noble) DignityActive() bool { return n.Status != NobleStatusDungeon }

func (n Noble) activeDignity(test func(DignityEffect) bool) bool {
	return n.DignityActive() && n.anyDignity(test)
}

// ProtectsFromPlague reports whether the noble shields its surroundings from
// plague.
func (n Noble) ProtectsFromPlague() bool {
	return n.activeDignity(func(e DignityEffect) bool { return e.PlagueImmune })
}

// RationDiscount is the ration cost the noble spares the armies on its
// territory.
func (n Noble) RationDiscount() int {
	if !n.DignityActive() {
		return 0
	}
	discount := 0
	for _, dignity := range n.Dignities {
		discount += dignityEffects[dignity].RationDiscount
	}
	return discount
}

// RivalConsumption is the extra ration cost per turn the noble's dignities
// impose on rival armies in its region.
func (n Noble) RivalConsumption() int {
	if !n.DignityActive() {
		return 0
	}
	extra := 0
	for _, dignity := range n.Dignities {
		extra += dignityEffects[dignity].RivalConsumption
	}
	return extra
}

// CalamityForecast is the number of upcoming calamities the noble lets the
// player it serves see.
func (n Noble) CalamityForecast() int {
	if !n.DignityActive() {
		return 0
	}
	forecast := 0
	for _, dignity := range n.Dignities {
		forecast = max(forecast, dignityEffects[dignity].CalamityForecast)
	}
	return forecast
}

// SeesFiefOrders, SeesAbbeyOrders, SeesHostOrders and SeesHostHand report the
// information powers of the noble's active dignities.
func (n Noble) SeesFiefOrders() bool {
	return n.activeDignity(func(e DignityEffect) bool { return e.SeesFiefOrders })
}
func (n Noble) SeesAbbeyOrders() bool {
	return n.activeDignity(func(e DignityEffect) bool { return e.SeesAbbeyOrders })
}
func (n Noble) SeesHostOrders() bool {
	return n.activeDignity(func(e DignityEffect) bool { return e.SeesHostOrders })
}
func (n Noble) SeesHostHand() bool {
	return n.activeDignity(func(e DignityEffect) bool { return e.SeesHostHand })
}

// TrialDignity returns the dignity of the ladies that makes the noble liable
// to the direct trial, hidden ones included: a trial reveals it. It reports
// false for a noble who cannot be tried directly: not a lady, married, without
// a dignity of the ladies, or exempt from the direct trial (the Abbess,
// specs/dames.md § Carte de procès). A chevalier d'Éon is a lady under her
// public male identity.
func (n Noble) TrialDignity(married bool) (Dignity, bool) {
	if married || (n.Sex != SexFemale && n.SecretSex != SexFemale) {
		return "", false
	}
	var found Dignity
	for _, dignity := range n.Dignities {
		effect := dignityEffects[dignity]
		if effect.ExemptFromDirectTrial {
			return "", false
		}
		if effect.FemaleOnly && found == "" {
			found = dignity
		}
	}
	return found, found != ""
}

// CanReceive returns why a dignity cannot be played on the noble, or
// "" when it can. married tells whether the noble is married.
func (d Dignity) CanReceive(noble Noble, married bool) string {
	effect := d.Effect()
	switch {
	case effect.FemaleOnly && noble.Sex != SexFemale:
		return "dignity_female_only"
	case effect.RequiresUnmarried && married:
		return "noble_married"
	case noble.Has(d):
		return "noble_already_" + string(d)
	case effect.FemaleOnly && noble.anyDignity(func(held DignityEffect) bool { return held.FemaleOnly }):
		// The dignities of the ladies do not stack: one per noble.
		return "dignity_exclusive"
	}
	return ""
}
