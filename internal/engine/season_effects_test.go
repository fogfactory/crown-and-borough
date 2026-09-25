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
		"AAA": {},
		"BBB": {},
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
	state.TerritoryStates["AAA"] = models.TerritoryState{Army: armyPointer("A1")}
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
	if ctx.fairWeatherRegions["AAA"] || ctx.goodHarvestRegions["AAA"] {
		t.Fatal("canceling fair weather unexpectedly produced a bonus")
	}
}

// cardEffectState puts a level-2 mill on AAA and a controlled castle on BBB,
// both plains of region AAA.
func cardEffectState() *models.GameState {
	state := effectTestState()
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	setTerritoryOwner(state, "BBB", "P1")
	return state
}

func TestCardEffectsOnRationsAndProduction(t *testing.T) {
	plain := testBalance().RationTerrain[models.TerrainPlain]
	tests := []struct {
		name       string
		calamity   models.CardKind
		bonus      models.CardKind
		wantRation int
		wantSource sourceProductionParts
	}{
		{name: "no effect", wantRation: plain, wantSource: sourceProductionParts{mill: 2}},
		{name: "bad harvest suppresses rations but not mills", calamity: models.CardKindFamine,
			wantRation: 0, wantSource: sourceProductionParts{mill: 2}},
		{name: "good harvest doubles rations but not mills", bonus: models.CardKindAbundantHarvest,
			wantRation: 2 * plain, wantSource: sourceProductionParts{mill: 2}},
		{name: "bad weather suppresses mills", calamity: models.CardKindBadWeather,
			wantRation: plain, wantSource: sourceProductionParts{suppressed: 2}},
		{name: "fair weather doubles mills", bonus: models.CardKindFairWeather,
			wantRation: plain, wantSource: sourceProductionParts{mill: 2, bonus: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := cardEffectState()
			if tt.calamity != "" {
				setCurrentCalamity(state, tt.calamity, "AAA")
			}
			ctx := newResolutionContext(state, testBalance())
			if tt.bonus != "" {
				ctx.deckIntents = []deckOrderIntent{{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: tt.bonus, RegionSeed: "AAA"}}}
			}
			resolveSeasonEffects(ctx)
			if got := rationProduction(ctx, "AAA"); got != tt.wantRation {
				t.Fatalf("ration production = %d, want %d", got, tt.wantRation)
			}
			if got := sourceProductionBreakdown(ctx, "BBB"); got != tt.wantSource {
				t.Fatalf("castle production = %#v, want %#v", got, tt.wantSource)
			}
		})
	}
}

func TestSettlementsNoLongerProduceRations(t *testing.T) {
	state := effectTestState()
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "BBB"})
	ctx := newResolutionContext(state, testBalance())
	resolveSeasonEffects(ctx)
	for _, territoryID := range []models.TerritoryID{"AAA", "BBB"} {
		if got, want := rationProduction(ctx, territoryID), ctx.balance.RationTerrain[models.TerrainPlain]; got != want {
			t.Fatalf("%s ration production = %d, want terrain only %d", territoryID, got, want)
		}
	}
}

func TestResolveSeasonEffectsRevoltBuildsCumulatedNeutralArmy(t *testing.T) {
	state := effectTestState()
	setCurrentCalamity(state, models.CardKindFamine, "AAA")
	balance := testBalance()
	balance.SpecialOrders.Effects.RevoltArmyMinSize = 1
	balance.SpecialOrders.Effects.RevoltArmyMaxSize = 3
	ctx := newResolutionContext(state, balance)
	ctx.deckIntents = []deckOrderIntent{
		{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: models.CardKindRevolt, TargetTerritoryID: "AAA"}},
		{playerID: "P2", order: models.DeckOrder{ID: "O2", Kind: models.CardKindRevolt, TargetTerritoryID: "AAA"}},
	}
	resolveSeasonEffects(ctx)
	if len(ctx.state.Armies) != 1 {
		t.Fatalf("armies = %#v, want one cumulated neutral army", ctx.state.Armies)
	}
	rebels := ctx.state.Armies[0]
	if rebels.OwnerID != models.NeutralPlayerID || rebels.TerritoryID != "AAA" {
		t.Fatalf("rebel army = %#v, want a neutral army at AAA", rebels)
	}
	if rebels.Size < 2 || rebels.Size > 6 {
		t.Fatalf("rebel size = %d, want the sum of two rolls between 1 and 3", rebels.Size)
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
	blocked := 0
	for _, event := range ctx.events {
		if event.Type != EventTypeBadWeatherBlocked {
			continue
		}
		blocked++
		if event.ArmyID != "A1" || event.OwnerID != "P1" || event.RegionSeed != "AAA" || event.TerritoryID != "AAA" || event.TargetID != "BBB" {
			t.Fatalf("blocked event = %#v, want A1 from AAA blocked towards BBB", event)
		}
	}
	if blocked != 1 {
		t.Fatalf("blocked events = %d, want one", blocked)
	}
}

func TestBadWeatherInvalidationPausesChain(t *testing.T) {
	state := effectTestState()
	chainID := models.ChainID("C1")
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1, ChainID: &chainID}}
	state.Chains = []models.Chain{{
		ID:           chainID,
		NobleID:      "N1",
		ArmyID:       "A1",
		CurrentIndex: 0,
		Orders: []models.Order{{
			ID: "O1", Type: models.OrderTypeAttack, PositionID: "AAA",
			TargetIDs: []models.TerritoryID{"BBB"},
		}},
	}}
	setCurrentCalamity(state, models.CardKindBadWeather, "AAA")
	ctx := newResolutionContext(state, testBalance())
	resolveSeasonEffects(ctx)
	record := &orderRecord{armyID: "A1", chainID: chainID, order: state.Chains[0].Orders[0]}
	record.invalidate("bad_weather")

	before, after, progression := ctx.progressRecord(record)
	if progression != ProgressionRetried || before != 0 || after != 0 {
		t.Fatalf("progression = %s (%d→%d), want retried with unchanged index", progression, before, after)
	}
	if len(ctx.state.Chains) != 1 || ctx.state.Chains[0].ID != chainID {
		t.Fatalf("chains = %#v, want the paused chain kept", ctx.state.Chains)
	}
	if ctx.state.Armies[0].ChainID == nil || *ctx.state.Armies[0].ChainID != chainID {
		t.Fatal("paused chain was detached from the army")
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

func TestPlagueEmitsDeathAndSurvivorEvents(t *testing.T) {
	state := effectTestState()
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "AAA")
	setCurrentCalamity(state, models.CardKindPlague, "AAA")
	balance := testBalance()
	balance.SpecialOrders.Effects.PlagueNobleMortalityPercentage = 50
	ctx := newResolutionContext(state, balance)
	resolveSeasonEffects(ctx)

	outcomes := map[models.NobleID]string{}
	for _, event := range ctx.events {
		if event.Type != EventTypePlagueDeath && event.Type != EventTypePlagueSurvived {
			continue
		}
		if event.RegionSeed != "AAA" || event.TerritoryID != "AAA" || event.NobleCode == "" {
			t.Fatalf("noble event = %#v, want region AAA, location AAA and a noble code", event)
		}
		if _, exists := outcomes[event.NobleID]; exists {
			t.Fatalf("noble %s has both a death and a survivor event", event.NobleID)
		}
		outcomes[event.NobleID] = string(event.Type)
	}
	if len(outcomes) != 2 {
		t.Fatalf("noble outcomes = %#v, want one event per noble", outcomes)
	}
	for _, nobleID := range []models.NobleID{"N1", "N2"} {
		if outcomes[nobleID] != string(EventTypePlagueDeath) && outcomes[nobleID] != string(EventTypePlagueSurvived) {
			t.Fatalf("noble %s outcome = %q, want death or survived", nobleID, outcomes[nobleID])
		}
	}
}

func calamityLossEvents(t *testing.T, kind models.CardKind, eventType EventType) (*Event, []Event) {
	t.Helper()
	state := cardEffectState()
	setCurrentCalamity(state, kind, "AAA")
	ctx := newResolutionContext(state, testBalance())
	resolveSeasonEffects(ctx)
	var summary *Event
	details := []Event{}
	for index := range ctx.events {
		event := ctx.events[index]
		if event.Type != eventType {
			continue
		}
		if event.RegionSeed != "AAA" || event.CardKind != kind {
			t.Fatalf("loss event = %#v, want %s in region AAA", event, kind)
		}
		if event.TerritoryID == "" {
			summary = &event
			continue
		}
		details = append(details, event)
	}
	if summary == nil {
		t.Fatalf("events = %#v, want a %s summary", ctx.events, eventType)
	}
	return summary, details
}

func TestFamineEmitsLossSummaryAndDetails(t *testing.T) {
	summary, details := calamityLossEvents(t, models.CardKindFamine, EventTypeFamineLoss)
	if summary.Production != 1 || summary.RationsLost != 6 {
		t.Fatalf("summary = %#v, want 1 R of castle production and 2 × 3 terrain rations lost", summary)
	}
	if len(details) != 1 || details[0].TerritoryID != "BBB" || details[0].InfrastructureType != models.InfraTypeCastle || details[0].Production != 1 {
		t.Fatalf("details = %#v, want the castle at BBB losing 1 R", details)
	}
}

func TestBadWeatherEmitsMillLossSummaryAndDetails(t *testing.T) {
	summary, details := calamityLossEvents(t, models.CardKindBadWeather, EventTypeBadWeatherLoss)
	if summary.Production != 2 || summary.RationsLost != 0 {
		t.Fatalf("summary = %#v, want 2 R of mill production lost", summary)
	}
	if len(details) != 1 || details[0].TerritoryID != "AAA" || details[0].Level != 2 || details[0].Production != 2 {
		t.Fatalf("details = %#v, want the level-2 mill at AAA losing 2 R", details)
	}
}

func TestTwoFairWeathersAgainstBadWeatherApplyOneBonus(t *testing.T) {
	state := effectTestState()
	setCurrentCalamity(state, models.CardKindBadWeather, "AAA")
	ctx := newResolutionContext(state, testBalance())
	ctx.deckIntents = []deckOrderIntent{
		{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
		{playerID: "P2", order: models.DeckOrder{ID: "O2", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
	}
	resolveSeasonEffects(ctx)
	if ctx.badWeatherRegions["AAA"] {
		t.Fatal("bad weather remains active after two fair weathers")
	}
	assertSeasonEffectCounts(t, ctx, 1, 1)
	if !ctx.fairWeatherRegions["AAA"] || ctx.goodHarvestRegions["AAA"] {
		t.Fatalf("bonus regions = %v/%v, want only the residual fair weather bonus", ctx.fairWeatherRegions, ctx.goodHarvestRegions)
	}
}

func TestTwoFairWeathersWithoutBadWeatherApplyOneBonus(t *testing.T) {
	state := effectTestState()
	ctx := newResolutionContext(state, testBalance())
	ctx.deckIntents = []deckOrderIntent{
		{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
		{playerID: "P2", order: models.DeckOrder{ID: "O2", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
	}
	resolveSeasonEffects(ctx)
	assertSeasonEffectCounts(t, ctx, 0, 1)
	if !ctx.fairWeatherRegions["AAA"] || ctx.goodHarvestRegions["AAA"] {
		t.Fatalf("bonus regions = %v/%v, want the fair weather bonus only", ctx.fairWeatherRegions, ctx.goodHarvestRegions)
	}
}

func TestThreeFairWeathersCapOneBonus(t *testing.T) {
	state := effectTestState()
	ctx := newResolutionContext(state, testBalance())
	ctx.deckIntents = []deckOrderIntent{
		{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
		{playerID: "P2", order: models.DeckOrder{ID: "O2", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
		{playerID: "P1", order: models.DeckOrder{ID: "O3", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
	}
	resolveSeasonEffects(ctx)
	assertSeasonEffectCounts(t, ctx, 0, 1)
	if !ctx.fairWeatherRegions["AAA"] {
		t.Fatalf("fair weather regions = %v, want AAA", ctx.fairWeatherRegions)
	}
}

func TestTwoAbundantHarvestsAgainstFamineApplyOneBonus(t *testing.T) {
	state := effectTestState()
	setCurrentCalamity(state, models.CardKindFamine, "AAA")
	ctx := newResolutionContext(state, testBalance())
	ctx.deckIntents = []deckOrderIntent{
		{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: models.CardKindAbundantHarvest, RegionSeed: "AAA"}},
		{playerID: "P2", order: models.DeckOrder{ID: "O2", Kind: models.CardKindAbundantHarvest, RegionSeed: "AAA"}},
	}
	resolveSeasonEffects(ctx)
	if ctx.famineRegions["AAA"] {
		t.Fatal("famine remains active after two abundant harvests")
	}
	assertSeasonEffectCounts(t, ctx, 1, 1)
	if ctx.fairWeatherRegions["AAA"] || !ctx.goodHarvestRegions["AAA"] {
		t.Fatalf("bonus regions = %v/%v, want only the residual good harvest bonus", ctx.fairWeatherRegions, ctx.goodHarvestRegions)
	}
}

func TestFairWeatherAndAbundantHarvestCancelBothIndependently(t *testing.T) {
	state := effectTestState()
	setCurrentCalamity(state, models.CardKindBadWeather, "AAA")
	famineAugury := state.Auguries[state.Year()]
	famineAugury.Calamities = append(famineAugury.Calamities, models.Calamity{
		Kind: models.CardKindFamine, Year: state.Year(), Season: state.Season, RegionSeed: "AAA",
	})
	state.Auguries[state.Year()] = famineAugury
	ctx := newResolutionContext(state, testBalance())
	ctx.deckIntents = []deckOrderIntent{
		{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
		{playerID: "P2", order: models.DeckOrder{ID: "O2", Kind: models.CardKindAbundantHarvest, RegionSeed: "AAA"}},
	}
	resolveSeasonEffects(ctx)
	if ctx.badWeatherRegions["AAA"] || ctx.famineRegions["AAA"] {
		t.Fatal("both calamities must be canceled")
	}
	// Each card is consumed by its own cancellation: no regional bonus remains.
	assertSeasonEffectCounts(t, ctx, 2, 0)
	if ctx.fairWeatherRegions["AAA"] || ctx.goodHarvestRegions["AAA"] {
		t.Fatalf("bonus regions = %v/%v, want none while both cards cancel", ctx.fairWeatherRegions, ctx.goodHarvestRegions)
	}
}

func TestDoubleBonusesAgainstBothCalamitiesApplyOncePerCategory(t *testing.T) {
	state := effectTestState()
	setCurrentCalamity(state, models.CardKindBadWeather, "AAA")
	famineAugury := state.Auguries[state.Year()]
	famineAugury.Calamities = append(famineAugury.Calamities, models.Calamity{
		Kind: models.CardKindFamine, Year: state.Year(), Season: state.Season, RegionSeed: "AAA",
	})
	state.Auguries[state.Year()] = famineAugury
	ctx := newResolutionContext(state, testBalance())
	ctx.deckIntents = []deckOrderIntent{
		{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
		{playerID: "P2", order: models.DeckOrder{ID: "O2", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
		{playerID: "P1", order: models.DeckOrder{ID: "O3", Kind: models.CardKindAbundantHarvest, RegionSeed: "AAA"}},
		{playerID: "P2", order: models.DeckOrder{ID: "O4", Kind: models.CardKindAbundantHarvest, RegionSeed: "AAA"}},
	}
	resolveSeasonEffects(ctx)
	if ctx.badWeatherRegions["AAA"] || ctx.famineRegions["AAA"] {
		t.Fatal("both calamities must be canceled")
	}
	assertSeasonEffectCounts(t, ctx, 2, 2)
	// One residual bonus per card kind: mills and harvest are both doubled.
	if !ctx.fairWeatherRegions["AAA"] || !ctx.goodHarvestRegions["AAA"] {
		t.Fatalf("bonus regions = %v/%v, want both residual bonuses", ctx.fairWeatherRegions, ctx.goodHarvestRegions)
	}
}

func TestFairWeathersInDifferentRegionsApplySeparately(t *testing.T) {
	state := effectTestState()
	state.Regions = []models.Region{
		{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}},
		{ID: "BBB", Seed: "BBB", Territories: []models.TerritoryID{"BBB"}},
	}
	ctx := newResolutionContext(state, testBalance())
	ctx.deckIntents = []deckOrderIntent{
		{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
		{playerID: "P2", order: models.DeckOrder{ID: "O2", Kind: models.CardKindFairWeather, RegionSeed: "BBB"}},
	}
	resolveSeasonEffects(ctx)
	assertSeasonEffectCounts(t, ctx, 0, 2)
	if !ctx.fairWeatherRegions["AAA"] || !ctx.fairWeatherRegions["BBB"] {
		t.Fatalf("fair weather regions = %v, want AAA and BBB", ctx.fairWeatherRegions)
	}
}

func assertSeasonEffectCounts(t *testing.T, ctx *resolutionContext, wantCanceled, wantBonus int) {
	t.Helper()
	canceled := 0
	bonus := 0
	for _, event := range ctx.events {
		switch event.Type {
		case EventTypeCalamityCanceled:
			canceled++
		case EventTypeBonusEffect:
			bonus++
		}
	}
	if canceled != wantCanceled || bonus != wantBonus {
		t.Fatalf("season effect events = %d canceled / %d bonus, want %d/%d", canceled, bonus, wantCanceled, wantBonus)
	}
}
