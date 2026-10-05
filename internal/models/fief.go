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

// NobleDisplayName is the name shown to players. A noble holding no fief is
// its stored Name ("Prénom de Territoire", its birthplace) preceded by
// "Sieur" or "Dame". A fief holder takes the form of address of the highest
// fief it holds ("Baron", "Comtesse", "Marquis", "Duc"...) and the fief's
// capital replaces its birthplace: "Baron Hugon de Rochevent". Name itself
// stays the stored identity. It is safe to call on a nil state.
func (g *GameState) NobleDisplayName(n Noble) string {
	var best *Fief
	if g != nil {
		for i := range g.Fiefs {
			fief := &g.Fiefs[i]
			if fief.HolderNobleID == nil || *fief.HolderNobleID != n.ID {
				continue
			}
			if best == nil || fiefTitleRanks[fief.Title] > fiefTitleRanks[best.Title] {
				best = fief
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
	if !held {
		forms = [2]string{"Sieur", "Dame"}
	}
	if n.Sex == SexFemale {
		return forms[1] + " " + name
	}
	return forms[0] + " " + name
}
