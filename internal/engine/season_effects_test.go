package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func effectTestState() *models.GameState {
	state := models.NewGameState()
	state.ID = "effects"
	state.Seed = "effects"
	state.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
	state.Territories = []models.Territory{
		{ID: "AAA", Name: "AAA", Terrain: models.TerrainPlain, Adjacencies: []models.TerritoryID{"BBB"}},
		{ID: "BBB", Name: "BBB", Terrain: models.TerrainPlain, Adjacencies: []models.TerritoryID{"AAA"}},
	}
	state.TerritoryStates = map[models.TerritoryID]models.TerritoryState{
		"AAA": {Infrastructures: []models.InfraID{}},
		"BBB": {Infrastructures: []models.InfraID{}},
	}
	state.Regions = []models.Region{{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB"}}}
	state.Auguries = map[int]models.YearAugury{}
	return state
}

func setCurrentCalamity(state *models.GameState, kind models.CardKind, region models.TerritoryID) {
	state.Auguries[state.Year()] = models.YearAugury{
		Year:       state.Year(),
		Calamities: []models.Calamity{{Kind: kind, Year: state.Year(), Season: state.Season, RegionSeed: region}},
	}
}

func TestResolveSeasonEffectsPlagueReducesArmies(t *testing.T) {
	state := effectTestState()
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 5}}
	state.TerritoryStates["AAA"] = models.TerritoryState{Army: armyPointer("A1"), Infrastructures: []models.InfraID{}}
	setCurrentCalamity(state, models.CardKindPlague, "AAA")
	balance := testBalance()
	balance.SpecialOrders.Effects.PlagueArmyDivisor = 2
	ctx := newResolutionContext(state, balance)
	resolveSeasonEffects(ctx)
	if got := ctx.armiesByID["A1"].Size; got != 3 {
		t.Fatalf("plague army size = %d, want 3", got)
	}
	if got := ctx.startArmiesByID["A1"].Size; got != 3 {
		t.Fatalf("plague start army size = %d, want 3", got)
	}
}

func TestResolveSeasonEffectsFairWeatherCancelsBadWeatherWithoutBonus(t *testing.T) {
	state := effectTestState()
	setCurrentCalamity(state, models.CardKindBadWeather, "AAA")
	ctx := newResolutionContext(state, testBalance())
	ctx.deckIntents = []deckOrderIntent{{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}}}
	resolveSeasonEffects(ctx)
	if ctx.badWeatherRegions["AAA"] {
		t.Fatal("bad weather remains active after fair weather cancellation")
	}
	if ctx.bonusMillRegions["AAA"] != 0 || ctx.bonusRationRegions["AAA"] != 0 {
		t.Fatal("canceling fair weather unexpectedly produced a bonus")
	}
}

func TestResolveSeasonEffectsFamineDisablesMillContribution(t *testing.T) {
	state := effectTestState()
	state.Infrastructures = []models.Infrastructure{{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "AAA"}}
	state.TerritoryStates["AAA"] = models.TerritoryState{Infrastructures: []models.InfraID{"I1"}}
	setCurrentCalamity(state, models.CardKindFamine, "AAA")
	ctx := newResolutionContext(state, testBalance())
	resolveSeasonEffects(ctx)
	if got := sourceProduction(ctx, "AAA"); got != ctx.balance.BaseProduction {
		t.Fatalf("famine source production = %d, want %d", got, ctx.balance.BaseProduction)
	}
	if got := rationProduction(ctx, "AAA"); got != ctx.balance.RationTerrain[models.TerrainPlain] {
		t.Fatalf("famine ration production = %d, want terrain production %d", got, ctx.balance.RationTerrain[models.TerrainPlain])
	}
}

func TestResolveSeasonEffectsRevoltCreatesNeutralArmies(t *testing.T) {
	state := effectTestState()
	setCurrentCalamity(state, models.CardKindFamine, "AAA")
	balance := testBalance()
	balance.SpecialOrders.Effects.RevoltArmyCount = 3
	balance.SpecialOrders.Effects.RevoltArmyMinSize = 2
	balance.SpecialOrders.Effects.RevoltArmyMaxSize = 3
	ctx := newResolutionContext(state, balance)
	ctx.deckIntents = []deckOrderIntent{{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: models.CardKindRevolt, RegionSeed: "AAA"}}}
	resolveSeasonEffects(ctx)
	if len(ctx.state.Armies) != 2 {
		t.Fatalf("armies = %d, want 2 neutral armies", len(ctx.state.Armies))
	}
	for _, army := range ctx.state.Armies {
		if army.OwnerID != models.NeutralPlayerID || army.Size < 2 || army.Size > 3 {
			t.Fatalf("revolt army = %#v, want neutral size 2..3", army)
		}
	}
}

func TestResolveSeasonEffectsPlagueNobleMortalityIsDeterministic(t *testing.T) {
	for _, test := range []struct {
		name             string
		lastEmissionTurn int
		wantChain        bool
	}{
		{name: "current chain is removed", lastEmissionTurn: 1, wantChain: false},
		{name: "historical chain continues", lastEmissionTurn: 0, wantChain: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := effectTestState()
			addNoble(state, "N1", "ONE", "P1", "AAA")
			state.Nobles[0].LastEmissionTurn = test.lastEmissionTurn
			setCurrentCalamity(state, models.CardKindPlague, "AAA")
			if test.wantChain {
				state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}}
				territoryState := state.TerritoryStates["AAA"]
				territoryState.Army = armyPointer("A1")
				state.TerritoryStates["AAA"] = territoryState
				chainID := models.ChainID("C1")
				state.Armies[0].ChainID = &chainID
				state.Chains = []models.Chain{{ID: chainID, NobleID: "N1", ArmyID: "A1", CurrentIndex: 0, Orders: []models.Order{{ID: "O1", Type: models.OrderTypeHold, ArmyID: "A1", PositionID: "AAA", Liaison: models.LiaisonModeSingle}}}}
			}
			balance := testBalance()
			balance.SpecialOrders.Effects.PlagueNobleMortalityPercentage = 100
			ctx := newResolutionContext(state, balance)
			resolveSeasonEffects(ctx)
			if len(ctx.state.Nobles) != 0 {
				t.Fatalf("nobles = %#v, want noble removed", ctx.state.Nobles)
			}
			if (len(ctx.state.Chains) != 0) != test.wantChain {
				t.Fatalf("chains = %#v, want historical chain kept = %t", ctx.state.Chains, test.wantChain)
			}
			if test.wantChain && ctx.state.Armies[0].ChainID == nil {
				t.Fatal("historical chain was detached from army")
			}
		})
	}
}

func TestEnumerateOrderRejectsBadWeatherMovementFromRegion(t *testing.T) {
	state := effectTestState()
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}}
	state.TerritoryStates["AAA"] = models.TerritoryState{Army: armyPointer("A1")}
	setCurrentCalamity(state, models.CardKindBadWeather, "AAA")
	ctx := newResolutionContext(state, testBalance())
	resolveSeasonEffects(ctx)
	record := &orderRecord{armyID: "A1", order: models.Order{ID: "O1", Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}}}
	ctx.enumerateOrder(record, state.Armies[0], false)
	if record.outcome != OutcomeInvalid || record.reason != "bad_weather" {
		t.Fatalf("order record = %#v, want bad_weather invalidation", record)
	}
}

func TestEnumerateOrderRejectsBadWeatherDestination(t *testing.T) {
	state := effectTestState()
	state.Regions = []models.Region{{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}}, {ID: "BBB", Seed: "BBB", Territories: []models.TerritoryID{"BBB"}}}
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 1}}
	state.TerritoryStates["BBB"] = models.TerritoryState{Army: armyPointer("A1")}
	setCurrentCalamity(state, models.CardKindBadWeather, "AAA")
	ctx := newResolutionContext(state, testBalance())
	resolveSeasonEffects(ctx)
	record := &orderRecord{armyID: "A1", order: models.Order{ID: "O1", Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}}}
	ctx.enumerateOrder(record, state.Armies[0], false)
	if record.outcome != OutcomeInvalid || record.reason != "bad_weather" {
		t.Fatalf("order record = %#v, want bad_weather destination invalidation", record)
	}
}

func TestBadWeatherFiltersDisperseDestinations(t *testing.T) {
	state := effectTestState()
	state.Regions = []models.Region{{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}}, {ID: "BBB", Seed: "BBB", Territories: []models.TerritoryID{"BBB"}}}
	setCurrentCalamity(state, models.CardKindBadWeather, "AAA")
	ctx := newResolutionContext(state, testBalance())
	resolveSeasonEffects(ctx)
	if got := ctx.filterBadWeatherDisperseTargets([]models.TerritoryID{"AAA", "BBB"}); len(got) != 1 || got[0] != "BBB" {
		t.Fatalf("filtered destinations = %#v, want [BBB]", got)
	}
}
