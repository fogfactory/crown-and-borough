package models

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

// NobleDisplayName is the name shown to players: the noble's Name preceded by
// the form of address of the highest fief it holds ("Baron", "Comtesse",
// "Marquis", "Duc"...), or by the default "Sieur"/"Dame" when it holds none.
// Name itself stays the stored identity. It is safe to call on a nil state.
func (g *GameState) NobleDisplayName(n Noble) string {
	var best FiefTitle
	if g != nil {
		for _, fief := range g.Fiefs {
			if fief.HolderNobleID != nil && *fief.HolderNobleID == n.ID && fiefTitleRanks[fief.Title] > fiefTitleRanks[best] {
				best = fief.Title
			}
		}
	}
	forms, held := courtesyTitles[best]
	if !held {
		forms = [2]string{"Sieur", "Dame"}
	}
	if n.Sex == SexFemale {
		return forms[1] + " " + n.Name
	}
	return forms[0] + " " + n.Name
}
