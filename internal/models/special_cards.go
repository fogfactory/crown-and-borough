package models

type SpecialCardID string

type RegionID string

type CardKind string

const (
	CardKindFairWeather     CardKind = "fair_weather"
	CardKindAbundantHarvest CardKind = "abundant_harvest"
	CardKindPlague          CardKind = "plague"
	CardKindBadWeather      CardKind = "bad_weather"
	CardKindRevolt          CardKind = "revolt"
	CardKindFamine          CardKind = "famine"
)

func (k CardKind) IsValid() bool {
	switch k {
	case CardKindFairWeather, CardKindAbundantHarvest, CardKindPlague,
		CardKindBadWeather, CardKindRevolt, CardKindFamine:
		return true
	}
	return false
}

func (k CardKind) IsBonus() bool {
	return k == CardKindFairWeather || k == CardKindAbundantHarvest
}

func (k CardKind) IsCalamity() bool {
	return k == CardKindPlague || k == CardKindBadWeather || k == CardKindRevolt || k == CardKindFamine
}

func (k CardKind) CanceledCalamity() (CardKind, bool) {
	switch k {
	case CardKindFairWeather:
		return CardKindBadWeather, true
	case CardKindAbundantHarvest:
		return CardKindFamine, true
	default:
		return "", false
	}
}

type SpecialCard struct {
	ID   SpecialCardID `json:"id"`
	Kind CardKind      `json:"kind"`
}

type Region struct {
	ID          RegionID      `json:"id"`
	Seed        TerritoryID   `json:"seed"`
	Territories []TerritoryID `json:"territories"`
}

type Calamity struct {
	CardID     SpecialCardID `json:"cardId"`
	Kind       CardKind      `json:"kind"`
	Year       int           `json:"year"`
	Season     Season        `json:"season"`
	RegionSeed TerritoryID   `json:"regionSeed"`
}

type YearAugury struct {
	Year       int            `json:"year"`
	Capacities map[Season]int `json:"capacities"`
	Revealed   bool           `json:"revealed"`
	Calamities []Calamity     `json:"calamities"`
}

type SpecialDeck struct {
	Cards    []SpecialCard                `json:"cards"`
	DrawPile []SpecialCardID              `json:"drawPile"`
	Discard  []SpecialCardID              `json:"discard"`
	Hands    map[PlayerID][]SpecialCardID `json:"hands"`
}
