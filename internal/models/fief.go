package models

import "strings"

// FiefID identifies a fief. It is an internal engine detail used for
// uniqueness and the test corpus only: the API and reports address a fief by
// its capital territory's trigram instead (titres.md, #194).
type FiefID string

// Fief is a group of at least FiefMinTerritories territories constituted by a
// T F winter order. Territories[0] is always the capital: the territory
// whose castle became a city and anchors the fief. HolderNobleID is nil when
// the fief is vacant (no titulaire), which happens when its noble dies or
// when a fief is conquered; a vacant fief still counts for score until it is
// dissolved (titres.md).
type Fief struct {
	ID                 FiefID        `json:"id"`
	Title              FiefTitle     `json:"title"`
	CapitalTerritoryID TerritoryID   `json:"capitalTerritory"`
	Territories        []TerritoryID `json:"territories"`
	OwnerID            PlayerID      `json:"owner"`
	HolderNobleID      *NobleID      `json:"holderNoble,omitempty"`
	// Crown marks a crown fief: a vacant fief returned to the crown player
	// when its player had no noble left (politique.md "Héritage d'un fief sans
	// héritier"). It is never attributed by default and is granted by a W G
	// order. Only a vacant fief can be a crown fief: any attribution, or a
	// conquest, clears the mark.
	Crown bool `json:"crown,omitempty"`
	// ExiledNobleID is the legitimate titulaire a conquest or a claim ousted
	// (titres.md "Conquête d'un fief par un noble"). The exiled title has no
	// game effect: the noble is deliberately not a HolderNobleID. Its owner
	// recovers the fief by retaking the capital, or when the usurper dies.
	ExiledNobleID *NobleID `json:"exiledNoble,omitempty"`
	// Usurped marks HolderNobleID as a usurper: the noble who took the fief by
	// conquest or claim over ExiledNobleID. It implies an exile.
	Usurped bool `json:"usurped,omitempty"`
}

// FreeNoblesAt returns the free nobles of the player standing on the
// territory, in the order of the player's line of succession. A hostage or a
// prisoner carried there is left out.
func (g *GameState) FreeNoblesAt(owner PlayerID, territoryID TerritoryID) []Noble {
	var free []Noble
	for _, noble := range g.SuccessionLine(owner) {
		if noble.Status == NobleStatusFree && noble.LocationID == territoryID {
			free = append(free, noble)
		}
	}
	return free
}

// ClaimUsurperAt returns the free noble that usurps a fief whose titulaire
// died when a claim fires for an heir of heirOwner's house: the first free
// noble, in line of succession, of the player's army standing on the fief's
// capital (the engine keeps one army per territory, so the first army found
// there is the only one). A rebel army, an army of the heir's own house, or an army without a
// free noble yields nil, and the heir takes the title.
func (g *GameState) ClaimUsurperAt(capitalID TerritoryID, heirOwner PlayerID) *Noble {
	for _, army := range g.Armies {
		if army.TerritoryID != capitalID {
			continue
		}
		if army.OwnerID == NeutralPlayerID || army.OwnerID == heirOwner {
			return nil
		}
		if free := g.FreeNoblesAt(army.OwnerID, capitalID); len(free) > 0 {
			return &free[0]
		}
		return nil
	}
	return nil
}

// SettleFiefTitlesOfMissingNobles applies the death of the nobles that are no
// longer in play to the usurpations and the plain titles. For every fief whose
// holder or exiled noble is missing, holder and exile are judged together:
//   - the usurper is missing and the exile lives: the exile recovers the title
//     and the fief returns to the exile's owner (restored);
//   - the exile is missing and the usurper lives: heirOf may name the living
//     heir of a claim on the exile, who becomes the new exile (reexiled);
//     otherwise the usurper becomes the official holder (legitimised);
//   - both are missing: the fief is vacant and keeps its owner (vacated);
//   - an ordinary holder is missing: heirOf may name the heir that takes the
//     fief (claimed: the holder is left in place for the caller to serve the
//     heir), otherwise the fief is vacant (vacated).
//
// heirOf receives the fief and the missing noble the claims bear on and returns
// the living heir to serve, or nil. Call once the dead nobles were removed.
func (g *GameState) SettleFiefTitlesOfMissingNobles(heirOf func(fief *Fief, missing NobleID) *Noble) []FiefTitleSettlement {
	living := make(map[NobleID]bool, len(g.Nobles))
	for _, noble := range g.Nobles {
		living[noble.ID] = true
	}
	var settlements []FiefTitleSettlement
	for i := range g.Fiefs {
		fief := &g.Fiefs[i]
		holderMissing := fief.HolderNobleID != nil && !living[*fief.HolderNobleID]
		exileMissing := fief.ExiledNobleID != nil && !living[*fief.ExiledNobleID]
		if !holderMissing && !exileMissing {
			continue
		}
		switch {
		case fief.Usurped && holderMissing && exileMissing:
			fief.HolderNobleID, fief.ExiledNobleID, fief.Usurped = nil, nil, false
			settlements = append(settlements, FiefTitleSettlement{Fief: fief, Outcome: FiefTitleVacated})
		case fief.Usurped && holderMissing:
			previousOwner := g.RestoreExile(fief)
			settlements = append(settlements, FiefTitleSettlement{Fief: fief, Outcome: FiefTitleRestored, PreviousOwner: previousOwner})
		case fief.Usurped && exileMissing:
			missing := *fief.ExiledNobleID
			// The usurper cannot also be the exile: a claim of its own is ignored.
			if heir := heirOf(fief, missing); heir != nil && heir.ID != *fief.HolderNobleID {
				heirID := heir.ID
				fief.ExiledNobleID = &heirID
				settlements = append(settlements, FiefTitleSettlement{Fief: fief, Outcome: FiefTitleReexiled, Heir: heirID})
			} else {
				fief.ExiledNobleID, fief.Usurped = nil, false
				settlements = append(settlements, FiefTitleSettlement{Fief: fief, Outcome: FiefTitleLegitimised})
			}
		default:
			// An ordinary holder died. Defensive: a stale exile without a
			// usurper is dropped with it.
			fief.ExiledNobleID, fief.Usurped = nil, false
			if !holderMissing {
				continue
			}
			missing := *fief.HolderNobleID
			if heir := heirOf(fief, missing); heir != nil {
				settlements = append(settlements, FiefTitleSettlement{Fief: fief, Outcome: FiefTitleClaimed, Heir: heir.ID})
				continue
			}
			fief.HolderNobleID = nil
			settlements = append(settlements, FiefTitleSettlement{Fief: fief, Outcome: FiefTitleVacated})
		}
	}
	return settlements
}

// RestoreExile gives the fief back to the exiled noble: the fief returns to
// the exile's owner, the exile becomes its official holder, and the usurper,
// who loses everything, is dropped. It returns the previous owner. The caller
// checks that the fief has an exile (and a usurper) and that the exile is
// alive.
func (g *GameState) RestoreExile(fief *Fief) PlayerID {
	previousOwner := fief.OwnerID
	exileID := *fief.ExiledNobleID
	for _, noble := range g.Nobles {
		if noble.ID == exileID {
			fief.OwnerID = noble.OwnerID
			break
		}
	}
	fief.HolderNobleID = &exileID
	fief.ExiledNobleID = nil
	fief.Usurped = false
	fief.Crown = false
	return previousOwner
}

// FiefTitleOutcome tells what SettleFiefTitlesOfMissingNobles did to a fief.
type FiefTitleOutcome string

const (
	// FiefTitleVacated: the fief has no titulaire left.
	FiefTitleVacated FiefTitleOutcome = "vacated"
	// FiefTitleRestored: the exile recovered the fief after the usurper died.
	FiefTitleRestored FiefTitleOutcome = "restored"
	// FiefTitleReexiled: the exile died and a claim heir became the new exile.
	FiefTitleReexiled FiefTitleOutcome = "reexiled"
	// FiefTitleLegitimised: the exile died and the usurper became official.
	FiefTitleLegitimised FiefTitleOutcome = "legitimised"
	// FiefTitleClaimed: the holder died and a claim heir was served.
	FiefTitleClaimed FiefTitleOutcome = "claimed"
)

// FiefTitleSettlement reports one fief touched by
// SettleFiefTitlesOfMissingNobles. Heir is set for reexiled and claimed,
// PreviousOwner for restored.
type FiefTitleSettlement struct {
	Fief          *Fief
	Outcome       FiefTitleOutcome
	Heir          NobleID
	PreviousOwner PlayerID
}

// FiefReversion is a vacant fief handed to the crown player by
// ReturnOrphanFiefsToCrown. Fief points into GameState.Fiefs and stays valid
// until that slice changes.
type FiefReversion struct {
	Fief          *Fief
	PreviousOwner PlayerID
}

// CrownPlayer returns the player the crown fiefs return to: the one of the
// royal issuer (the king, else the regent), or of a king held in a dungeon
// when no regent acts in his name. It is false while the throne is vacant and
// no regent stands in.
func (g *GameState) CrownPlayer() (PlayerID, bool) {
	if issuer, ok := g.RoyalIssuer(); ok {
		return issuer.OwnerID, true
	}
	if g.King == nil {
		return "", false
	}
	for _, noble := range g.Nobles {
		if noble.ID == *g.King {
			return noble.OwnerID, true
		}
	}
	return "", false
}

// ReturnOrphanFiefsToCrown hands every vacant fief whose player has no living
// noble to the crown player, as a crown fief (politique.md "Héritage d'un fief
// sans héritier"). Control is derived from the fief owner (control.go), so
// every territory of a returned fief changes hands with it. Nothing happens
// while nobody holds the crown, and a crown fief left with a player who lost
// all their nobles moves on to the new crown player. Call once the dead nobles
// are removed and the queen refreshed.
func (g *GameState) ReturnOrphanFiefsToCrown() []FiefReversion {
	crownPlayer, ok := g.CrownPlayer()
	if !ok {
		return nil
	}
	living := make(map[PlayerID]bool, len(g.Nobles))
	for _, noble := range g.Nobles {
		living[noble.OwnerID] = true
	}
	var returned []FiefReversion
	for i := range g.Fiefs {
		fief := &g.Fiefs[i]
		if fief.HolderNobleID != nil || living[fief.OwnerID] {
			continue
		}
		returned = append(returned, FiefReversion{Fief: fief, PreviousOwner: fief.OwnerID})
		fief.OwnerID = crownPlayer
		fief.Crown = true
	}
	return returned
}

// TaxedFief records one turn a fief's capital was played the seigneurial tax
// card (titres.md "Taxe seigneuriale", #189): it widens revolt eligibility to
// every territory of the fief, independently of famine, for the turn it is
// played and the following one. Turn is the absolute GameState.Turn counter
// rather than a (season, year) pair, so the eligibility window is a plain
// comparison regardless of the winter boundary it may straddle.
type TaxedFief struct {
	FiefID FiefID `json:"fiefId"`
	Turn   int    `json:"turn"`
}

// TithedRegion records one turn a bishopric was levied the tithe (religieux.md
// "Dîme"): like a taxed fief, it widens revolt eligibility to every territory
// of the bishopric for the turn it is levied and the following one. Turn is
// the absolute GameState.Turn counter.
type TithedRegion struct {
	RegionSeed TerritoryID `json:"regionSeed"`
	Turn       int         `json:"turn"`
}

// TrialRevoltWindow records the revolt opportunity an executed trial opens
// on the region (bishopric) where the executed noble stood (specs/dames.md
// § Procès): a revolt card may target any territory of the region
// during the action season of turn FromTurn, the first one after the
// execution. Turn is the absolute GameState.Turn counter.
type TrialRevoltWindow struct {
	RegionSeed TerritoryID `json:"regionSeed"`
	FromTurn   int         `json:"fromTurn"`
}

// courtesyTitles maps a fief title to the form of address of its holder, by
// sex. A noble holding no fief is a "Sieur" or a "Dame".
var courtesyTitles = map[FiefTitle][2]string{
	FiefTitleBarony:     {"Baron", "Baronne"},
	FiefTitleCounty:     {"Comte", "Comtesse"},
	FiefTitleMarquisate: {"Marquis", "Marquise"},
	FiefTitleDuchy:      {"Duc", "Duchesse"},
}

var fiefTitleRanks = map[FiefTitle]int{
	FiefTitleBarony: 1, FiefTitleCounty: 2, FiefTitleMarquisate: 3, FiefTitleDuchy: 4,
}

// HighestHeldFief returns the highest-ranked fief the noble holds, or nil.
func (g *GameState) HighestHeldFief(id NobleID) *Fief {
	var best *Fief
	for i := range g.Fiefs {
		fief := &g.Fiefs[i]
		if fief.HolderNobleID == nil || *fief.HolderNobleID != id {
			continue
		}
		if best == nil || fiefTitleRanks[fief.Title] > fiefTitleRanks[best.Title] {
			best = fief
		}
	}
	return best
}

// FirstName is the first name of the noble, without its family name.
func (n Noble) FirstName() string {
	firstName, _, _ := strings.Cut(n.Name, " ")
	return firstName
}

// NobleDisplayName is the name shown to players. A noble holding no fief is
// its stored Name ("Prénom de Territoire", its birthplace) preceded by
// "Sieur" or "Dame". A fief holder takes the form of address of the highest
// fief it holds ("Baron", "Comtesse", "Marquis", "Duc"...) and the fief's
// capital replaces its birthplace: "Baron Hugon de Rochevent". The spouse of
// a fief holder (active marriage, any sex) bears the courtesy title matching
// the spouse's fief ("Comtesse Mahaut de Rochevent") when it outranks the
// noble's own fief, if any. This is display only: Name itself stays the
// stored identity and no rule reads the courtesy title. It is safe to call
// on a nil state. The dignity of D'Arc replaces the family name of the lady
// who carries it, with or without a fief: "Dame Hugon d'Arc".
func (g *GameState) NobleDisplayName(n Noble) string {
	var best *Fief
	if g != nil {
		best = g.HighestHeldFief(n.ID)
		if marriage, ok := g.MarriageOf(n.ID); ok {
			spouseID := marriage.NobleA
			if spouseID == n.ID {
				spouseID = marriage.NobleB
			}
			if spouseFief := g.HighestHeldFief(spouseID); spouseFief != nil &&
				(best == nil || fiefTitleRanks[spouseFief.Title] > fiefTitleRanks[best.Title]) {
				best = spouseFief
			}
		}
	}
	var forms [2]string
	held := false
	name := n.Name
	if best != nil {
		forms, held = courtesyTitles[best.Title]
	}
	if held {
		for _, territory := range g.Territories {
			if territory.ID == best.CapitalTerritoryID {
				firstName, _, _ := strings.Cut(n.Name, " ")
				name = firstName + " de " + territory.Name
				break
			}
		}
	}
	if n.Has(DignityDArc) {
		firstName, _, _ := strings.Cut(n.Name, " ")
		name = firstName + " d'Arc"
	}
	if !held {
		forms = [2]string{"Sieur", "Dame"}
	}
	if n.Sex == SexFemale {
		return forms[1] + " " + name
	}
	return forms[0] + " " + name
}
