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
