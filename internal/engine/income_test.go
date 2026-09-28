package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func incomeEventFor(t *testing.T, events []Event, ownerID models.PlayerID, destinationID models.TerritoryID) Event {
	t.Helper()
	for _, event := range events {
		if event.Type == EventTypeIncome && event.OwnerID == ownerID && event.DestinationID == destinationID {
			return event
		}
	}
	t.Fatalf("events = %#v, want an income event for %q -> %q", events, ownerID, destinationID)
	return Event{}
}

func hasIncomeEventFor(events []Event, ownerID models.PlayerID) bool {
	for _, event := range events {
		if event.Type == EventTypeIncome && event.OwnerID == ownerID {
			return true
		}
	}
	return false
}

// TestResolveTerritoryIncomeCreditsCapital covers the issue's first hotseat
// scenario: a player controlling 5 territories, one with a village, receives
// territory_income (1) per territory plus village_income (1) for the
// village, all credited to the capital.
func TestResolveTerritoryIncomeCreditsCapital(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB", "DDD"),
			territory("DDD", "DDD", "CCC", "EEE"),
			territory("EEE", "EEE", "DDD"),
		},
		nil,
	)
	for _, id := range []models.TerritoryID{"AAA", "BBB", "CCC", "DDD", "EEE"} {
		setTerritoryOwner(state, id, "P1")
	}
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "BBB"})
	setCapital(state, "P1", "I1")
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	event := incomeEventFor(t, resolution.Events, "P1", "AAA")
	if event.TerritoryCount != 5 || event.VillageCount != 1 || event.Production != 6 || event.Lost {
		t.Fatalf("income event = %#v, want 5 territories, 1 village, 6 R credited to the capital", event)
	}
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 6 {
		t.Errorf("capital stock = %d, want 6", got)
	}
}

// TestResolveTerritoryIncomeNeverInWinter checks that ResolveWinter never
// resolves or credits territory income: Resolve itself refuses winter
// states, so this exercises the only path a winter turn can take.
func TestResolveTerritoryIncomeNeverInWinter(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, testBalance(), nil)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if hasIncomeEventFor(resolution.Events, "P1") {
		t.Errorf("events = %#v, want no income event in winter", resolution.Events)
	}
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 0 {
		t.Errorf("AAA stock = %d, want 0, territory income must not apply in winter", got)
	}
}

// TestTerritoryIncomeFallsBackToClosestControlledCastle covers the issue's
// second hotseat scenario: once a player's capital falls, income is routed
// to the closest controlled castle instead.
func TestTerritoryIncomeFallsBackToClosestControlledCastle(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("PLN", "PLN", "NEA", "FAR"),
			territory("NEA", "NEA", "PLN"),
			territory("FAR", "FAR", "PLN", "END"),
			territory("END", "END", "FAR"),
		},
		nil,
	)
	for _, id := range []models.TerritoryID{"PLN", "NEA", "FAR", "END"} {
		setTerritoryOwner(state, id, "P1")
	}
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "NEA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "END"})
	// No capital is designated (it just fell): PLN must reach its closest
	// controlled castle, NEA, rather than the more distant END.
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	event := incomeEventFor(t, resolution.Events, "P1", "NEA")
	if event.TerritoryCount != 2 || event.Production != 2 {
		t.Fatalf("NEA income event = %#v, want PLN and NEA routed there", event)
	}
	other := incomeEventFor(t, resolution.Events, "P1", "END")
	if other.TerritoryCount != 2 || other.Production != 2 {
		t.Fatalf("END income event = %#v, want FAR and END routed there", other)
	}
}

// TestTerritoryIncomeFallbackTrigramTieBreak checks that equidistant
// controlled castles are ordered by trigram (territory ID) when no capital
// exists.
func TestTerritoryIncomeFallbackTrigramTieBreak(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("CTR", "CTR", "ZZZ", "AAA"),
			territory("ZZZ", "ZZZ", "CTR"),
			territory("AAA", "AAA", "CTR"),
		},
		nil,
	)
	for _, id := range []models.TerritoryID{"CTR", "ZZZ", "AAA"} {
		setTerritoryOwner(state, id, "P1")
	}
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "ZZZ"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// ZZZ and AAA each have their own castle and always route to
	// themselves first (distance 0); only CTR, equidistant from both, is
	// actually tie-broken, and must go to the lower trigram AAA.
	event := incomeEventFor(t, resolution.Events, "P1", "AAA")
	if event.TerritoryCount != 2 || event.Production != 2 {
		t.Fatalf("AAA income event = %#v, want CTR routed to the lower trigram AAA over ZZZ", event)
	}
	zzz := incomeEventFor(t, resolution.Events, "P1", "ZZZ")
	if zzz.TerritoryCount != 1 || zzz.Production != 1 {
		t.Fatalf("ZZZ income event = %#v, want only its own territory income", zzz)
	}
}

// TestTerritoryIncomeFallsBackToVillage checks that income falls back to the
// closest controlled village when the player controls no castle at all.
func TestTerritoryIncomeFallsBackToVillage(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("PLN", "PLN", "VIL"),
			territory("VIL", "VIL", "PLN"),
		},
		nil,
	)
	setTerritoryOwner(state, "PLN", "P1")
	setTerritoryOwner(state, "VIL", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "VIL"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	event := incomeEventFor(t, resolution.Events, "P1", "VIL")
	if event.TerritoryCount != 2 || event.VillageCount != 1 || event.Production != 3 {
		t.Fatalf("VIL income event = %#v, want both territories' income (2) plus the village bonus (1)", event)
	}
}

// TestTerritoryIncomeLostWithoutAnySettlement checks that income is lost
// when the player controls neither a capital, a castle, nor a village.
func TestTerritoryIncomeLostWithoutAnySettlement(t *testing.T) {
	state := testState(t, []models.Territory{territory("PLN", "PLN")}, nil)
	setTerritoryOwner(state, "PLN", "P1")
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// Production still reports the amount that would have been earned, so
	// the report can show how much was lost; Lost marks it as uncredited.
	event := incomeEventFor(t, resolution.Events, "P1", "")
	if !event.Lost || event.TerritoryCount != 1 || event.Production != 1 {
		t.Fatalf("income event = %#v, want the lone territory's income lost", event)
	}
	if got := resolution.State.TerritoryStates["PLN"].Resources; got != 0 {
		t.Errorf("PLN stock = %d, want 0, lost income is not banked locally", got)
	}
}

// TestTerritoryIncomeHarvestEffects checks that a bad harvest suppresses
// territory income in its region and an abundant harvest doubles it, exactly
// as they do for rations and mills.
func TestTerritoryIncomeHarvestEffects(t *testing.T) {
	tests := []struct {
		name       string
		calamity   models.CardKind
		bonus      models.CardKind
		wantBase   int
		wantBonus  int
		wantSupp   int
		wantCredit int
	}{
		{name: "no effect", wantBase: 1, wantCredit: 1},
		{name: "bad harvest suppresses it", calamity: models.CardKindFamine, wantSupp: 1, wantCredit: 0},
		{name: "abundant harvest doubles it", bonus: models.CardKindAbundantHarvest, wantBase: 1, wantBonus: 1, wantCredit: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := effectTestState()
			addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
			setTerritoryOwner(state, "AAA", "P1")
			if tt.calamity != "" {
				setCurrentCalamity(state, tt.calamity, "AAA")
			}
			ctx := newResolutionContext(state, testBalance())
			if tt.bonus != "" {
				ctx.deckIntents = []deckOrderIntent{{playerID: "P1", order: models.DeckOrder{ID: "O1", Kind: tt.bonus, RegionSeed: "AAA"}}}
			}
			resolveSeasonEffects(ctx)
			resolveTerritoryIncome(ctx)
			event := incomeEventFor(t, ctx.events, "P1", "AAA")
			if event.BaseProduction != tt.wantBase || event.BonusProduction != tt.wantBonus || event.SuppressedProduction != tt.wantSupp || event.Production != tt.wantCredit {
				t.Fatalf("income event = %#v, want base %d bonus %d suppressed %d credited %d",
					event, tt.wantBase, tt.wantBonus, tt.wantSupp, tt.wantCredit)
			}
		})
	}
}

// TestTerritoryIncomePaysDespiteEnemyOccupation checks that control, not
// occupation, decides who is paid: an enemy army sitting on a controlled
// territory does not interrupt its owner's income.
func TestTerritoryIncomePaysDespiteEnemyOccupation(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		nil,
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	// P2's army occupies P1's controlled territory, but does not own it.
	enemyArmy := models.Army{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1}
	state.Armies = append(state.Armies, enemyArmy)
	aaaState := state.TerritoryStates["AAA"]
	armyID := enemyArmy.ID
	aaaState.Army = &armyID
	state.TerritoryStates["AAA"] = aaaState
	state.NextArmyID = nextArmyID(state.Armies)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	event := incomeEventFor(t, resolution.Events, "P1", "AAA")
	if event.Lost || event.TerritoryCount != 1 || event.Production != 1 {
		t.Fatalf("income event = %#v, want P1 paid despite the occupying enemy army", event)
	}
}

// TestTerritoryIncomeFeedsSupplyTheSameTurn checks that income credited to a
// source is immediately available to cover a deficit in the same
// resolution, since it is credited before rationing.
func TestTerritoryIncomeFeedsSupplyTheSameTurn(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
	)
	setTerritoryOwner(state, "BBB", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if hasFamineEvent(resolution.Events, "A1") {
		t.Fatalf("events = %#v, want A1 fed by BBB's territory income this same turn", resolution.Events)
	}
}

// TestForecastIncomeIgnoresDrawnCalamities checks that ForecastIncome always
// shows the normal, unaffected income, even when a calamity is already
// scheduled for the current season and region.
func TestForecastIncomeIgnoresDrawnCalamities(t *testing.T) {
	state := effectTestState()
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	setTerritoryOwner(state, "AAA", "P1")
	setCurrentCalamity(state, models.CardKindFamine, "AAA")

	events := ForecastIncome(state, testBalance())
	event := incomeEventFor(t, events, "P1", "AAA")
	if event.SuppressedProduction != 0 || event.Production != 1 {
		t.Fatalf("forecast event = %#v, want the normal, unsuppressed income", event)
	}
	if got := state.TerritoryStates["AAA"].Resources; got != 0 {
		t.Errorf("input state was mutated: AAA stock = %d, want 0", got)
	}
}

// TestForecastTerritoryIncomePerTerritory checks the per-territory forecast
// backing the territory detail panel's "rapporte X R à YYY" line.
func TestForecastTerritoryIncomePerTerritory(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		nil,
	)
	setTerritoryOwner(state, "AAA", "P1")
	setTerritoryOwner(state, "BBB", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})

	forecast := ForecastTerritoryIncome(state, testBalance())
	aaa, ok := forecast["AAA"]
	if !ok || aaa.Amount != 1 || aaa.Destination != "AAA" {
		t.Fatalf("AAA forecast = %#v, want 1 R routed to itself", aaa)
	}
	bbb, ok := forecast["BBB"]
	if !ok || bbb.Amount != 1 || bbb.Destination != "AAA" {
		t.Fatalf("BBB forecast = %#v, want 1 R routed to AAA", bbb)
	}
}

// incomeEventForFief is incomeEventFor filtered further by fiefID, for
// scenarios where two income lines share the same destination (a fief
// capital coinciding with the player's own capital, see #196).
func incomeEventForFief(t *testing.T, events []Event, ownerID models.PlayerID, destinationID models.TerritoryID, fiefID models.FiefID) Event {
	t.Helper()
	for _, event := range events {
		if event.Type == EventTypeIncome && event.OwnerID == ownerID && event.DestinationID == destinationID && event.FiefID == fiefID {
			return event
		}
	}
	t.Fatalf("events = %#v, want an income event for %q -> %q (fief %q)", events, ownerID, destinationID, fiefID)
	return Event{}
}

// TestFiefMemberIncomeRoutedToFiefCapital verifies that a fief member's
// territory income is credited to the fief's capital rather than the
// player's own capital (titres.md, #196).
func TestFiefMemberIncomeRoutedToFiefCapital(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("PCP", "PCP", "FCP"),
			territory("FCP", "FCP", "PCP", "MEM"),
			territory("MEM", "MEM", "FCP", "OTH"),
			territory("OTH", "OTH", "MEM"),
		},
		nil,
	)
	for _, id := range []models.TerritoryID{"PCP", "FCP", "MEM", "OTH"} {
		setTerritoryOwner(state, id, "P1")
	}
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "PCP"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "FCP"})
	setCapital(state, "P1", "I1")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "FCP",
		Territories: []models.TerritoryID{"FCP", "MEM", "OTH"}, OwnerID: "P1",
	}}
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	fiefEvent := incomeEventForFief(t, resolution.Events, "P1", "FCP", "F1")
	if fiefEvent.TerritoryCount != 3 || fiefEvent.Production != 3 {
		t.Fatalf("fief income event = %#v, want 3 territories, 3 R credited to the fief capital", fiefEvent)
	}
	playerEvent := incomeEventForFief(t, resolution.Events, "P1", "PCP", "")
	if playerEvent.TerritoryCount != 1 || playerEvent.Production != 1 {
		t.Fatalf("player income event = %#v, want 1 territory routed to the player's own capital", playerEvent)
	}
	if got := resolution.State.TerritoryStates["FCP"].Resources; got != 3 {
		t.Errorf("FCP stock = %d, want 3 (all three fief members)", got)
	}
	if got := resolution.State.TerritoryStates["PCP"].Resources; got != 1 {
		t.Errorf("PCP stock = %d, want 1 (its own income only)", got)
	}

	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	fiefReport := incomeReportFor(t, report.Income, "P1", "FCP")
	if fiefReport.Fief != "FCP" || fiefReport.Title != models.FiefTitleBarony {
		t.Fatalf("fief income report = %#v, want Fief=FCP Title=barony", fiefReport)
	}
}

func incomeReportFor(t *testing.T, reports []IncomeReport, ownerID models.PlayerID, destination models.TerritoryID) IncomeReport {
	t.Helper()
	for _, report := range reports {
		if report.Owner == ownerID && report.Destination == destination && report.Fief != "" {
			return report
		}
	}
	t.Fatalf("income reports = %#v, want a fief line for %q -> %q", reports, ownerID, destination)
	return IncomeReport{}
}

// TestFiefIncomeNotInterceptedWhenCapitalOccupied verifies that an enemy
// army occupying the fief's capital does not intercept the territory income
// still routed to it: control (and thus the income destination) stays with
// the fief's owner until the capital is actually conquered (titres.md).
func TestFiefIncomeNotInterceptedWhenCapitalOccupied(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("FCP", "FCP", "MEM"),
			territory("MEM", "MEM", "FCP", "OTH"),
			territory("OTH", "OTH", "MEM"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "FCP", Size: 3}},
	)
	// testState sets FCP's owner from the P2 army stationed there; restore
	// P1's transitive fief control (an occupied capital is not conquered).
	setTerritoryOwner(state, "FCP", "P1")
	setTerritoryOwner(state, "MEM", "P1")
	setTerritoryOwner(state, "OTH", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "FCP"})
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "FCP",
		Territories: []models.TerritoryID{"FCP", "MEM", "OTH"}, OwnerID: "P1",
	}}
	validateTestState(t, state)

	reports, _ := computeTerritoryIncome(newResolutionContext(cloneGameState(state), testBalance()))
	report, ok := reports["P1"]["FCP"]["F1"]
	if !ok || report.territories != 3 {
		t.Fatalf("reports = %#v, want the fief income still routed to the occupied capital", reports)
	}
}

// TestFiefIncomeSeparateLineWhenCapitalCoincidesWithPlayerCapital verifies
// that a fief whose capital happens to also be the player's own capital
// still produces its own income report line, distinct from the player's
// non-fief territories routed to the same destination (#196).
func TestFiefIncomeSeparateLineWhenCapitalCoincidesWithPlayerCapital(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("CAP", "CAP", "MEM", "OUT"),
			territory("MEM", "MEM", "CAP", "OTH"),
			territory("OTH", "OTH", "MEM"),
			territory("OUT", "OUT", "CAP"),
		},
		nil,
	)
	for _, id := range []models.TerritoryID{"CAP", "MEM", "OTH", "OUT"} {
		setTerritoryOwner(state, id, "P1")
	}
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAP"})
	setCapital(state, "P1", "I1")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "CAP",
		Territories: []models.TerritoryID{"CAP", "MEM", "OTH"}, OwnerID: "P1",
	}}
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	fiefEvent := incomeEventForFief(t, resolution.Events, "P1", "CAP", "F1")
	if fiefEvent.TerritoryCount != 3 {
		t.Fatalf("fief income event = %#v, want the 3 fief members only", fiefEvent)
	}
	nonFiefEvent := incomeEventForFief(t, resolution.Events, "P1", "CAP", "")
	if nonFiefEvent.TerritoryCount != 1 {
		t.Fatalf("non-fief income event = %#v, want OUT only", nonFiefEvent)
	}
	if got := resolution.State.TerritoryStates["CAP"].Resources; got != 4 {
		t.Errorf("CAP stock = %d, want 4 (both lines credited to the same territory)", got)
	}
}
