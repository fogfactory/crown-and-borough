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

// TestMillRecipientNeutralMillPrefersNeutralVillage checks decision #2: a
// neutral mill treats "neutral" like any other controller, so it routes to a
// neutral village (there is no "neutral castle"), ignoring a player's castle
// even if adjacent.
func TestMillRecipientNeutralMillPrefersNeutralVillage(t *testing.T) {
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
	if got := millRecipient(ctx, "MIL"); got != "VIL" {
		t.Fatalf("millRecipient = %q, want VIL (the neutral village), ignoring the player's castle", got)
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
		nil,
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
		nil,
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

// TestMillReportLineHasRoutedDestination checks the turn report exposes the
// mill's single destination alongside its production.
func TestMillReportLineHasRoutedDestination(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("MIL", "MIL", models.TerrainPlain, "CAS"),
			supplyTerritory("CAS", "CAS", models.TerrainPlain, "MIL"),
		},
		nil,
	)
	setTerritoryOwner(state, "MIL", "P1")
	setTerritoryOwner(state, "CAS", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAS"})
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
