package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestMillRecipientPrefersCastleOverVillage checks #195's basic case: a
// level-2 mill adjacent to both a castle and a village under its own control
// routes its entire production to the castle.
func TestMillRecipientPrefersCastleOverVillage(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "CAS", "VIL"),
			supplyTerritory("CAS", "CAS", models.TerrainPlain, "MIL"),
			supplyTerritory("VIL", "VIL", models.TerrainPlain, "MIL"),
		},
		nil,
	)
	setTerritoryOwner(state, "MIL", "P1")
	setTerritoryOwner(state, "CAS", "P1")
	setTerritoryOwner(state, "VIL", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAS"})
	addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "VIL"})
	validateTestState(t, state)

	ctx := newResolutionContext(state, testBalance())
	if got := millRecipient(ctx, "MIL"); got != "CAS" {
		t.Fatalf("millRecipient = %q, want CAS", got)
	}
}

// TestMillRecipientIgnoresOtherController checks decision #1: a castle
// adjacent to the mill but controlled by another player is ignored, and the
// mill falls back to a village under its own control.
func TestMillRecipientIgnoresOtherController(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "CAS", "VIL"),
			supplyTerritory("CAS", "CAS", models.TerrainPlain, "MIL"),
			supplyTerritory("VIL", "VIL", models.TerrainPlain, "MIL"),
		},
		nil,
	)
	setTerritoryOwner(state, "MIL", "P1")
	setTerritoryOwner(state, "CAS", "P2")
	setTerritoryOwner(state, "VIL", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAS"})
	addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "VIL"})
	validateTestState(t, state)

	ctx := newResolutionContext(state, testBalance())
	if got := millRecipient(ctx, "MIL"); got != "VIL" {
		t.Fatalf("millRecipient = %q, want VIL (the other player's castle must be ignored)", got)
	}
}

// TestMillRecipientFallsBackToItself checks that a mill with no eligible
// same-control neighbor at all (no castle, no village) stocks itself.
func TestMillRecipientFallsBackToItself(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "CAS"),
			supplyTerritory("CAS", "CAS", models.TerrainPlain, "MIL"),
		},
		nil,
	)
	setTerritoryOwner(state, "MIL", "P1")
	setTerritoryOwner(state, "CAS", "P2")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "MIL"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAS"})
	validateTestState(t, state)

	ctx := newResolutionContext(state, testBalance())
	if got := millRecipient(ctx, "MIL"); got != "MIL" {
		t.Fatalf("millRecipient = %q, want MIL itself", got)
	}
}

// TestMillRecipientBreaksTiesByTrigram checks that when two same-type,
// same-control candidates are both adjacent, the lower trigram wins, exactly
// like #192's territory income destination.
func TestMillRecipientBreaksTiesByTrigram(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "ZZZ", "AAA"),
			supplyTerritory("ZZZ", "ZZZ", models.TerrainPlain, "MIL"),
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "MIL"),
		},
		nil,
	)
	setTerritoryOwner(state, "MIL", "P1")
	setTerritoryOwner(state, "ZZZ", "P1")
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "MIL"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "ZZZ"})
	addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)

	ctx := newResolutionContext(state, testBalance())
	if got := millRecipient(ctx, "MIL"); got != "AAA" {
		t.Fatalf("millRecipient = %q, want AAA (lower trigram wins the tie)", got)
	}
}

// TestMillRecipientUncontrolledMillStocksItself checks #470: an uncontrolled
// mill always credits its own territory, neither a player's adjacent castle
// nor an adjacent neutral village.
func TestMillRecipientUncontrolledMillStocksItself(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "CAS", "VIL"),
			supplyTerritory("CAS", "CAS", models.TerrainPlain, "MIL"),
			supplyTerritory("VIL", "VIL", models.TerrainPlain, "MIL"),
		},
		nil,
	)
	setTerritoryOwner(state, "CAS", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "MIL"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAS"})
	addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "VIL"})
	validateTestState(t, state)

	ctx := newResolutionContext(state, testBalance())
	if got := millRecipient(ctx, "MIL"); got != "MIL" {
		t.Fatalf("millRecipient = %q, want MIL (the uncontrolled mill itself)", got)
	}
}

// TestMillWeatherProductionSuppressedByBadWeather checks that a routed
// mill's production is entirely suppressed by bad weather, like an
// unrouted one (#191's rule is unaffected by #195's routing).
func TestMillWeatherProductionSuppressedByBadWeather(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "CAS"),
			supplyTerritory("CAS", "CAS", models.TerrainPlain, "MIL"),
		},
		// An army holds the mill: outside every fief and capital, only an
		// army gives the mill a controller to feed (#470).
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "MIL", Size: 1}},
	)
	setTerritoryOwner(state, "MIL", "P1")
	setTerritoryOwner(state, "CAS", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAS"})
	validateTestState(t, state)

	ctx := newResolutionContext(state, testBalance())
	ctx.badWeatherRegions[""] = true
	mills := computeMillProduction(ctx)
	if len(mills) != 1 || mills[0].destinationID != "CAS" || mills[0].total() != 0 || mills[0].suppressed != 2 {
		t.Fatalf("mills = %#v, want the routed mill's production entirely suppressed", mills)
	}
}

// TestMillWeatherProductionDoubledByFairWeather checks that a routed mill's
// production is doubled by fair weather.
func TestMillWeatherProductionDoubledByFairWeather(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "CAS"),
			supplyTerritory("CAS", "CAS", models.TerrainPlain, "MIL"),
		},
		// An army holds the mill: outside every fief and capital, only an
		// army gives the mill a controller to feed (#470).
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "MIL", Size: 1}},
	)
	setTerritoryOwner(state, "MIL", "P1")
	setTerritoryOwner(state, "CAS", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAS"})
	validateTestState(t, state)

	ctx := newResolutionContext(state, testBalance())
	ctx.fairWeatherRegions = map[models.TerritoryID]bool{"": true}
	mills := computeMillProduction(ctx)
	if len(mills) != 1 || mills[0].destinationID != "CAS" || mills[0].total() != 4 || mills[0].bonus != 2 {
		t.Fatalf("mills = %#v, want the routed mill's production doubled to 4", mills)
	}
}

// TestIsolatedMillProducesOnItsOwnTileAndFeedsSupply is the end-to-end
// counterpart of TestMillRecipientFallsBackToItself: an isolated mill with
// no eligible neighbor produces on its own tile and becomes a valid supply
// source for its own controller's army standing there.
func TestIsolatedMillProducesOnItsOwnTileAndFeedsSupply(t *testing.T) {
	state := testState(t,
		[]models.Territory{supplyTerritory("MIL", "MIL", models.TerrainMountain)},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "MIL", Size: 1}},
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "MIL"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if hasFamineEvent(resolution.Events, "A1") {
		t.Errorf("events = %#v, want the isolated mill's own production to feed A1", resolution.Events)
	}
	event := supplyEventForSource(t, resolution.Events, "MIL")
	if event.Production != 1 {
		t.Errorf("supply event = %#v, want the isolated mill's own production of 1", event)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	mill := millLineFor(t, report, "MIL")
	if mill.Destination != "MIL" || mill.Production != 1 {
		t.Errorf("mill report line = %#v, want production 1 credited to itself", mill)
	}
}

// TestUncontrolledMillStocksItsOwnTerritory checks #470: a mill outside
// every fief and capital, with no player army on it, produces on its own
// territory turn after turn, like a neutral village, and the report credits
// the production to the mill itself.
func TestUncontrolledMillStocksItsOwnTerritory(t *testing.T) {
	state := testState(t,
		[]models.Territory{supplyTerritory("MIL", "MIL", models.TerrainMountain)},
		nil,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := first.State.TerritoryStates["MIL"].Resources; got != 2 {
		t.Errorf("MIL stock after one turn = %d, want the mill's level 2", got)
	}
	event := supplyEventForSource(t, first.Events, "MIL")
	if event.OwnerID != "" || event.Production != 2 || event.StockAfter != 2 {
		t.Errorf("supply event = %#v, want a neutral production of 2", event)
	}
	report := BuildTurnReport(state, first.State, first.Events, nil)
	mill := millLineFor(t, report, "MIL")
	if mill.Destination != "MIL" || mill.Production != 2 {
		t.Errorf("mill report line = %#v, want production 2 credited to itself", mill)
	}

	second, err := Resolve(first.State, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := second.State.TerritoryStates["MIL"].Resources; got != 4 {
		t.Errorf("MIL stock after two turns = %d, want 4 accumulated", got)
	}
}

// TestUncontrolledMillFollowsTheWeather checks that an uncontrolled mill's
// own production follows the same weather cards as any other mill: bad
// weather suppresses it, fair weather doubles it.
func TestUncontrolledMillFollowsTheWeather(t *testing.T) {
	for _, test := range []struct {
		name                     string
		bad, fair                bool
		total, bonus, suppressed int
	}{
		{name: "bad weather", bad: true, total: 0, suppressed: 2},
		{name: "fair weather", fair: true, total: 4, bonus: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := testState(t,
				[]models.Territory{supplyTerritory("MIL", "MIL", models.TerrainMountain)},
				nil,
			)
			addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
			validateTestState(t, state)

			ctx := newResolutionContext(state, testBalance())
			ctx.badWeatherRegions[""] = test.bad
			ctx.fairWeatherRegions = map[models.TerritoryID]bool{"": test.fair}
			produceNeutralStocks(ctx)
			parts := ctx.supplySources["MIL"]
			if parts.total() != test.total || parts.bonus != test.bonus || parts.suppressed != test.suppressed {
				t.Errorf("neutral mill parts = %#v, want total %d, bonus %d, suppressed %d", parts, test.total, test.bonus, test.suppressed)
			}
			if got := ctx.state.TerritoryStates["MIL"].Resources; got != test.total {
				t.Errorf("MIL stock = %d, want %d", got, test.total)
			}
		})
	}
}

// TestCapturedMillStockFeedsTheCapturingArmy checks #470: the stock an
// uncontrolled mill accumulated becomes a supply source for the army that
// takes control of it.
func TestCapturedMillStockFeedsTheCapturingArmy(t *testing.T) {
	state := testState(t,
		// Mountain: 1 local ration, so a size-3 army (cost 4) needs 3 more
		// from the mill.
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "MIL"),
			supplyTerritory("MIL", "MIL", models.TerrainMountain, "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "MIL"})
	setTerritoryResources(state, "MIL", 5)
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"MIL"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if hasFamineEvent(resolution.Events, "A1") {
		t.Errorf("events = %#v, want A1 fed by the captured mill's stock", resolution.Events)
	}
	// 5 accumulated + 1 produced for P1 this turn - 3 eaten.
	if got := resolution.State.TerritoryStates["MIL"].Resources; got != 3 {
		t.Errorf("MIL stock = %d, want 5 + 1 - 3 = 3", got)
	}
	event := supplyEventForSource(t, resolution.Events, "MIL")
	if event.OwnerID != "P1" || event.Demand != 3 || event.StockConsumed != 2 {
		t.Errorf("supply event = %#v, want P1's source covering 3 with 2 from the accumulated stock", event)
	}
}

// TestCapturedMillStockCanBeTransferred checks #470: an army controlling a
// mill can ship the stock the mill accumulated while uncontrolled with a
// transfer order (T).
func TestCapturedMillStockCanBeTransferred(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "MIL"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "MIL", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "MIL")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "MIL"})
	setTerritoryResources(state, "MIL", 4)
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeTransfer,
		PositionID: "MIL",
		TargetIDs:  []models.TerritoryID{"BBB"},
		Amount:     2,
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	event, ok := findEvent(resolution.Events, EventTypeTransfer)
	if !ok || event.SourceID != "MIL" || event.ResourceAmount != 2 || event.Outcome != OutcomeSuccess {
		t.Errorf("transfer event = %#v, want 2 R shipped from the mill", event)
	}
}

// TestForecastMillProductionIncludesUncontrolledMill checks the territory
// panel forecast shows an uncontrolled mill's production on itself.
func TestForecastMillProductionIncludesUncontrolledMill(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "VIL"),
			supplyTerritory("VIL", "VIL", models.TerrainPlain, "MIL"),
		},
		nil,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 3, TerritoryID: "MIL"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "VIL"})
	validateTestState(t, state)

	forecast := ForecastMillProduction(state, testBalance())
	if got := forecast["MIL"]; got.Production != 3 || got.Destination != "MIL" {
		t.Errorf("forecast = %#v, want 3 R on the mill itself", got)
	}
}

// TestMillReportLineHasRoutedDestination checks the turn report exposes the
// mill's single destination alongside its production.
func TestMillReportLineHasRoutedDestination(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "CAS"),
			supplyTerritory("CAS", "CAS", models.TerrainPlain, "MIL"),
		},
		// An army holds the mill: outside every fief and capital, only an
		// army gives the mill a controller to feed (#470).
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "MIL", Size: 1}},
	)
	setTerritoryOwner(state, "MIL", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAS"})
	// No army garrisons CAS: anchor it as P1's capital so control resolution
	// (now ahead of ravitaillement, #208) does not release it as unanchored
	// before supply can route the mill's production there.
	setCapital(state, "P1", "I2")
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	mill := millLineFor(t, report, "MIL")
	if mill.Destination != "CAS" || mill.Owner != "P1" || mill.Level != 2 || mill.Production != 2 {
		t.Errorf("mill report line = %#v, want production 2 routed to CAS", mill)
	}
}

func millLineFor(t *testing.T, report TurnReport, territoryID models.TerritoryID) MillReport {
	t.Helper()
	for _, line := range report.Mills {
		if line.Territory == territoryID {
			return line
		}
	}
	t.Fatalf("mill lines = %#v, want a line for %q", report.Mills, territoryID)
	return MillReport{}
}
