package engine

import (
	"slices"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// armyRevoltState puts a player army of P1 on AAA, with no territorial revolt
// condition in the region, and gives P1 and P2 one Révolte card each.
func armyRevoltState(t *testing.T, terrain models.Terrain, army models.Army) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", terrain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
		},
		[]models.Army{army},
	)
	state.Regions = []models.Region{{ID: "ROS", Seed: "ROS", Territories: []models.TerritoryID{"AAA", "BBB"}}}
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindRevolt}, {ID: "C2", Kind: models.CardKindRevolt}},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}, "P2": {"C2"}},
	}
	addNoble(state, "N1", "ONE", "P1", "AAA")
	validateTestState(t, state)
	return state
}

func fixedRevoltBalance(roll int) assetgen.Balance {
	balance := testBalance()
	balance.SpecialOrders.Effects.RevoltArmyMinSize = roll
	balance.SpecialOrders.Effects.RevoltArmyMaxSize = roll
	return balance
}

func revoltOn(id models.OrderID, territoryID models.TerritoryID) models.DeckOrder {
	return models.DeckOrder{ID: id, Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: territoryID}
}

func armyRevoltEvent(t *testing.T, events []Event, armyID models.ArmyID) Event {
	t.Helper()
	for _, event := range events {
		if event.Type == EventTypeArmyRevolt && event.ArmyID == armyID {
			return event
		}
	}
	t.Fatalf("events = %#v, want an army revolt event for %q", events, armyID)
	return Event{}
}

func TestRevoltCanTargetOnlyAStarvingPlayerArmy(t *testing.T) {
	for _, test := range []struct {
		name string
		army models.Army
		want bool
	}{
		{name: "starving player army", army: models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3, Starving: true, StarvingStrength: 1}, want: true},
		{name: "fed player army", army: models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := armyRevoltState(t, models.TerrainPlain, test.army)
			ctx := newResolutionContext(state, testBalance())
			for _, playerID := range []models.PlayerID{"P1", "P2"} {
				ok, _ := (revoltCardDefinition{}).CanPlay(&ExecutionContext{resolution: ctx, playerID: playerID, season: models.SeasonSpring}, revoltOn("O1", "AAA"))
				if ok != test.want {
					t.Errorf("CanPlay for %s = %t, want %t", playerID, ok, test.want)
				}
			}
			if got := slices.Contains(ForecastRevoltTargets(state, testBalance()), "AAA"); got != test.want {
				t.Errorf("AAA among the revolt targets = %t, want %t", got, test.want)
			}
		})
	}
}

func TestRevoltTargetsIgnoreStarvingNeutralArmiesAndWinter(t *testing.T) {
	state := armyRevoltState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3, Starving: true, StarvingStrength: 1})
	state.Armies[0].OwnerID = models.NeutralPlayerID
	if targets := ForecastRevoltTargets(state, testBalance()); len(targets) != 0 {
		t.Errorf("targets = %v, want none on a neutral army", targets)
	}
	state.Armies[0].OwnerID = "P1"
	state.Season = models.SeasonWinter
	if targets := ForecastRevoltTargets(state, testBalance()); targets != nil {
		t.Errorf("winter targets = %v, want none", targets)
	}
}

func TestRevoltReachingArmySizeTurnsStarvingArmyNeutralAndKillsItsNobles(t *testing.T) {
	state := armyRevoltState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3, Starving: true, StarvingStrength: 1})
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	validateTestState(t, state)

	resolution, err := ResolveWithDeckOrders(state, fixedRevoltBalance(3), map[models.PlayerID][]models.DeckOrder{
		"P2": {revoltOn("O1", "AAA")},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	army := armyByID(t, resolution.State, "A1")
	if army.OwnerID != models.NeutralPlayerID || army.TerritoryID != "AAA" || army.Starving || army.ChainID != nil {
		t.Fatalf("A1 = %+v, want a neutral army left on AAA without chain nor starving status", army)
	}
	if len(resolution.State.Chains) != 0 {
		t.Errorf("chains = %#v, want the army's chain dropped", resolution.State.Chains)
	}
	died := false
	for _, removed := range resolution.State.RemovedNobles {
		if removed.ID == "N1" && removed.Cause == models.DeathCauseRevolt {
			died = true
		}
	}
	if !died {
		t.Errorf("removed nobles = %#v, want N1 dead of the revolt", resolution.State.RemovedNobles)
	}
	event := armyRevoltEvent(t, resolution.Events, "A1")
	if event.Reason != ArmyRevoltNeutralizedReason || event.Troops != 3 || event.SizeBefore != 3 || event.PreviousOwnerID != "P1" {
		t.Errorf("army revolt event = %#v, want a roll of 3 neutralizing P1's three troops", event)
	}
	for _, other := range resolution.Events {
		if other.Type == EventTypeNeutralArmy || (other.Type == EventTypeCombat && other.TerritoryID == "AAA") {
			t.Errorf("event %#v: a revolt on a starving army raises no rebels and fights no combat", other)
		}
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	found := false
	for _, effect := range report.SeasonEffects {
		if effect.Kind == EventTypeArmyRevolt && effect.Territory == "AAA" && effect.VictimOwner == "P1" {
			found = true
		}
	}
	if !found {
		t.Errorf("season effects = %#v, want the army revolt reported", report.SeasonEffects)
	}
}

func TestRevoltsOnTheSameArmySumTheirRolls(t *testing.T) {
	// Two one-troop rolls reach a two-troop army: the threshold is inclusive,
	// and the owner may play on their own army.
	state := armyRevoltState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2, Starving: true, StarvingStrength: 0})
	resolution, err := ResolveWithDeckOrders(state, fixedRevoltBalance(1), map[models.PlayerID][]models.DeckOrder{
		"P1": {revoltOn("O1", "AAA")},
		"P2": {revoltOn("O2", "AAA")},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.OwnerID != models.NeutralPlayerID {
		t.Fatalf("A1 = %+v, want it neutral after two summed rolls of 1", army)
	}
	event := armyRevoltEvent(t, resolution.Events, "A1")
	if event.Troops != 2 || event.Reason != ArmyRevoltNeutralizedReason {
		t.Errorf("army revolt event = %#v, want one event with the summed roll of 2", event)
	}
	count := 0
	for _, other := range resolution.Events {
		if other.Type == EventTypeArmyRevolt {
			count++
		}
	}
	if count != 1 {
		t.Errorf("army revolt events = %d, want the cards resolved together in one", count)
	}
}

func TestRevoltBelowArmySizeLeavesTheArmyToItsOwner(t *testing.T) {
	state := armyRevoltState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3, Starving: true, StarvingStrength: 1})
	resolution, err := ResolveWithDeckOrders(state, fixedRevoltBalance(2), map[models.PlayerID][]models.DeckOrder{
		"P2": {revoltOn("O1", "AAA")},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.OwnerID != "P1" {
		t.Fatalf("A1 = %+v, want it kept by P1", army)
	}
	if !slices.ContainsFunc(resolution.State.Nobles, func(noble models.Noble) bool { return noble.ID == "N1" }) {
		t.Error("N1 should survive a failed revolt")
	}
	if event := armyRevoltEvent(t, resolution.Events, "A1"); event.Reason != ArmyRevoltSuppressedReason || event.Troops != 2 {
		t.Errorf("army revolt event = %#v, want a suppressed roll of 2", event)
	}
	if hand := resolution.State.SpecialDeck.Hands["P2"]; len(hand) != 0 {
		t.Errorf("P2 hand = %v, want the card spent", hand)
	}
}

func TestRevoltOnStarvingArmyTakesPrecedenceOverTerritorialFamine(t *testing.T) {
	state := armyRevoltState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3, Starving: true, StarvingStrength: 1})
	state.Auguries[1] = models.YearAugury{
		Year:       1,
		Capacities: map[models.Season]int{models.SeasonSpring: 1, models.SeasonSummer: 1, models.SeasonAutumn: 1},
		Calamities: []models.Calamity{{CardID: "C9", Kind: models.CardKindFamine, Season: models.SeasonSpring, Year: 1, RegionSeed: "ROS"}},
	}
	state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, models.SpecialCard{ID: "C9", Kind: models.CardKindFamine})
	validateTestState(t, state)

	resolution, err := ResolveWithDeckOrders(state, fixedRevoltBalance(2), map[models.PlayerID][]models.DeckOrder{
		"P2": {revoltOn("O1", "AAA")},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	// The calamity strips AAA of its rations: what the famine does to A1 at
	// the end of the turn is the famine rule, not the revolt's.
	if event := armyRevoltEvent(t, resolution.Events, "A1"); event.Reason != ArmyRevoltSuppressedReason || event.SizeBefore != 3 {
		t.Errorf("army revolt event = %#v, want a roll of 2 suppressed by 3 troops", event)
	}
	for _, event := range resolution.Events {
		if event.Type == EventTypeNeutralArmy || (event.Type == EventTypeCombat && event.TerritoryID == "AAA") {
			t.Errorf("event %#v: the card should play on the starving army, not raise rebels", event)
		}
	}
}

func TestNeutralizedArmyFollowsTheNeutralFamineRule(t *testing.T) {
	// A mountain feeds one ration: the three neutral troops (4 rations) lose
	// one at this turn's ravitaillement.
	state := armyRevoltState(t, models.TerrainMountain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3, Starving: true, StarvingStrength: 0})
	resolution, err := ResolveWithDeckOrders(state, fixedRevoltBalance(3), map[models.PlayerID][]models.DeckOrder{
		"P2": {revoltOn("O1", "AAA")},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	army := armyByID(t, resolution.State, "A1")
	if army.OwnerID != models.NeutralPlayerID || army.Size != 2 || army.Starving {
		t.Fatalf("A1 = %+v, want a neutral army down to 2 troops, without starving status", army)
	}
	found := false
	for _, event := range resolution.Events {
		if event.Type == EventTypeFamine && event.ArmyID == "A1" && event.OwnerID == models.NeutralPlayerID && event.TroopsLost == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("events = %#v, want the neutral famine loss of one troop", resolution.Events)
	}
}
