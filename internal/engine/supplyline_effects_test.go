package engine

import (
	"errors"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// effectSupplyState places a player army of the given size alone on a plain
// territory of region AAA, with one card of each given kind in P1's hand.
func effectSupplyState(t *testing.T, size int, calamity models.CardKind, hand ...models.CardKind) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: size}},
	)
	state.Regions = []models.Region{{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}}}
	deck := &models.SpecialDeck{
		Cards:    []models.SpecialCard{},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {}},
	}
	for index, kind := range hand {
		cardID := models.SpecialCardID("C" + string(rune('1'+index)))
		deck.Cards = append(deck.Cards, models.SpecialCard{ID: cardID, Kind: kind})
		deck.Hands["P1"] = append(deck.Hands["P1"], cardID)
	}
	if calamity != "" {
		deck.Cards = append(deck.Cards, models.SpecialCard{ID: "C9", Kind: calamity})
		state.Auguries[1] = models.YearAugury{
			Year:       1,
			Capacities: map[models.Season]int{models.SeasonSpring: 1, models.SeasonSummer: 1, models.SeasonAutumn: 1},
			Calamities: []models.Calamity{{CardID: "C9", Kind: calamity, Season: models.SeasonSpring, Year: 1, RegionSeed: "AAA"}},
		}
	}
	state.SpecialDeck = deck
	validateTestState(t, state)
	return state
}

func playOrder(kind models.CardKind, target models.TerritoryID) map[models.PlayerID][]models.DeckOrder {
	order := models.DeckOrder{ID: "O1", Type: models.DeckOrderTypePlay, Kind: kind, RegionSeed: target}
	if kind == models.CardKindRevolt {
		order.RegionSeed = ""
		order.TargetTerritoryID = target
	}
	return map[models.PlayerID][]models.DeckOrder{"P1": {order}}
}

func TestFindSupplyProjectsCurrentSeasonEffects(t *testing.T) {
	balance := testBalance()
	balance.SpecialOrders.Effects.PlagueArmyDivisor = 2

	tests := []struct {
		name       string
		size       int
		calamity   models.CardKind
		hand       []models.CardKind
		deckOrders map[models.PlayerID][]models.DeckOrder
		wantSize   int
		wantFamine int
		wantBonus  int
		wantLocal  int
		wantDemand int
	}{
		{name: "no effect", size: 3, wantSize: 3, wantLocal: 3, wantDemand: 1},
		{name: "famine suppresses terrain rations", size: 3, calamity: models.CardKindFamine, wantSize: 3, wantFamine: 3, wantLocal: 0, wantDemand: 4},
		{
			name: "played abundant harvest cancels the famine", size: 3, calamity: models.CardKindFamine,
			hand:       []models.CardKind{models.CardKindAbundantHarvest},
			deckOrders: playOrder(models.CardKindAbundantHarvest, "AAA"),
			wantSize:   3, wantLocal: 3, wantDemand: 1,
		},
		{
			name: "held but unplayed abundant harvest changes nothing", size: 3, calamity: models.CardKindFamine,
			hand:     []models.CardKind{models.CardKindAbundantHarvest},
			wantSize: 3, wantFamine: 3, wantLocal: 0, wantDemand: 4,
		},
		{
			name: "abundant harvest without calamity doubles terrain rations", size: 3,
			hand:       []models.CardKind{models.CardKindAbundantHarvest},
			deckOrders: playOrder(models.CardKindAbundantHarvest, "AAA"),
			wantSize:   3, wantBonus: 3, wantLocal: 6, wantDemand: 0,
		},
		{
			name: "fair weather leaves rations unchanged", size: 3,
			hand:       []models.CardKind{models.CardKindFairWeather},
			deckOrders: playOrder(models.CardKindFairWeather, "AAA"),
			wantSize:   3, wantLocal: 3, wantDemand: 1,
		},
		{
			name: "card not in hand is ignored", size: 3,
			deckOrders: playOrder(models.CardKindAbundantHarvest, "AAA"),
			wantSize:   3, wantLocal: 3, wantDemand: 1,
		},
		{name: "plague halves the army before supply", size: 3, calamity: models.CardKindPlague, wantSize: 2, wantLocal: 3, wantDemand: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := effectSupplyState(t, tt.size, tt.calamity, tt.hand...)
			line, err := FindSupplyWithIntents(state, balance, "AAA", tt.deckOrders)
			if err != nil {
				t.Fatalf("FindSupplyWithIntents: %v", err)
			}
			if line.ArmySize != tt.wantSize || line.FamineRations != tt.wantFamine || line.BonusRations != tt.wantBonus ||
				line.LocalProduction != tt.wantLocal || line.Demand != tt.wantDemand {
				t.Fatalf("line = %#v, want size %d, famine %d, bonus %d, local %d, demand %d",
					line, tt.wantSize, tt.wantFamine, tt.wantBonus, tt.wantLocal, tt.wantDemand)
			}
			if state.Armies[0].Size != tt.size || len(state.SpecialDeck.Hands["P1"]) != len(tt.hand) {
				t.Fatal("FindSupplyWithIntents mutated its input")
			}
		})
	}
}

func TestFindSupplyIgnoresPlayedRevolt(t *testing.T) {
	state := effectSupplyState(t, 1, models.CardKindFamine, models.CardKindRevolt)
	state.Territories = append(state.Territories, supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"))
	state.Territories[0].Adjacencies = []models.TerritoryID{"BBB"}
	state.TerritoryStates["BBB"] = models.TerritoryState{}
	state.Regions[0].Territories = []models.TerritoryID{"AAA", "BBB"}
	validateTestState(t, state)

	revolt := playOrder(models.CardKindRevolt, "BBB")
	line, err := FindSupplyWithIntents(state, testBalance(), "AAA", revolt)
	if err != nil {
		t.Fatalf("FindSupplyWithIntents: %v", err)
	}
	if line.FamineRations != 3 || line.LocalProduction != 0 {
		t.Fatalf("line = %#v, want the famine still active", line)
	}
	// A projected revolt would raise a neutral army on BBB and reveal its roll.
	if _, err := FindSupplyWithIntents(state, testBalance(), "BBB", revolt); !errors.Is(err, ErrSupplyLineNoSource) {
		t.Fatalf("BBB supply error = %v, want no army and no source on BBB", err)
	}
}
