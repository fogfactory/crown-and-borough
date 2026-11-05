package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestBuildOrderApply verifies that C C on a controlled village fortifies it
// in place (same InfraID, InfraType stays village) rather than replacing it
// with a castle (#193).
func TestBuildOrderApply(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeCastle}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	infrastructure := ctx.infrastructureAt("AAA")
	if infrastructure == nil || infrastructure.ID != "I0" || infrastructure.Type != models.InfraTypeVillage || !infrastructure.Fortified {
		t.Fatalf("infrastructure = %#v, events = %#v, want fortified village I0", infrastructure, ctx.events)
	}
	fortifications := eventsOfType(ctx.events, EventTypeFortify)
	if len(fortifications) != 1 || fortifications[0].ResourceSpent != testBalance().Costs.Castle {
		t.Fatalf("fortify events = %#v, want one event spending %d", ctx.events, testBalance().Costs.Castle)
	}
	if got := ctx.state.TerritoryStates["AAA"].Resources; got != 0 {
		t.Errorf("stock = %d, want 0 after spending the fortification cost", got)
	}
}

// TestBuildOrderRejectsFortifyingAnAlreadyFortifiedVillage verifies that a
// second C C on an already-fortified village is rejected with no charge
// (#193).
func TestBuildOrderRejectsFortifyingAnAlreadyFortifiedVillage(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA", Fortified: true})
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeCastle}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "village_already_fortified" {
		t.Fatalf("rejected events = %#v, want village_already_fortified", rejected)
	}
	if got := ctx.state.TerritoryStates["AAA"].Resources; got != 10 {
		t.Errorf("stock = %d, want unchanged 10 (no prelevement)", got)
	}
}

// TestBuildOrderRejectsOccupiedTerritory verifies that a winter build order
// on a territory occupied against its controller is rejected without any
// prelevement (titres.md, #196).
func TestBuildOrderRejectsOccupiedTerritory(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")},
		[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 2}},
	)
	holdAsFiefMember(state, "AAA", "P1")
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeCastle}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	if infrastructure := ctx.infrastructureAt("AAA"); infrastructure != nil {
		t.Fatalf("infrastructure = %#v, want none built on an occupied territory", infrastructure)
	}
	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "territory_occupied_by_other_player" {
		t.Fatalf("rejected events = %#v, want territory_occupied_by_other_player", rejected)
	}
	if got := ctx.state.TerritoryStates["AAA"].Resources; got != 10 {
		t.Errorf("stock = %d, want unchanged 10 (no prelevement)", got)
	}
}

// TestBuildOrderRebuildsARuinedCastle verifies that C C on a ruined castle
// lifts the ruin in place and charges the castle cost.
func TestBuildOrderRebuildsARuinedCastle(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA", Ruined: true})
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeCastle}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	infrastructure := ctx.infrastructureAt("AAA")
	if infrastructure == nil || infrastructure.ID != "I0" || infrastructure.Ruined {
		t.Fatalf("infrastructure = %#v, events = %#v, want castle I0 rebuilt", infrastructure, ctx.events)
	}
	if got := ctx.state.TerritoryStates["AAA"].Resources; got != 10-testBalance().Costs.Castle {
		t.Errorf("stock = %d, want %d after paying the castle cost", got, 10-testBalance().Costs.Castle)
	}
	if got := capitalID(t, ctx.state, "P1"); got != "I0" {
		t.Errorf("capital = %q, want the rebuilt castle I0 for a player without one", got)
	}
}

// TestBuildOrderReplacesARuinedCastle verifies that any build other than C C
// treats a ruined castle's tile as empty: the ruin is replaced and its stock
// stays on the tile.
func TestBuildOrderReplacesARuinedCastle(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA", Ruined: true})
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeSupplyDepot}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	infrastructure := ctx.infrastructureAt("AAA")
	if infrastructure == nil || infrastructure.Type != models.InfraTypeSupplyDepot {
		t.Fatalf("infrastructure = %#v, events = %#v, want a supply depot replacing the ruin", infrastructure, ctx.events)
	}
	if len(ctx.state.Infrastructures) != 1 {
		t.Errorf("infrastructures = %#v, want the ruin gone", ctx.state.Infrastructures)
	}
	if got := ctx.state.TerritoryStates["AAA"].Resources; got != 10-testBalance().Costs.SupplyDepot {
		t.Errorf("stock = %d, want %d after paying the depot cost", got, 10-testBalance().Costs.SupplyDepot)
	}
}

// TestBuildOrderBuildsACastleOnARuinedVillage verifies that C C on a ruined
// village builds a castle in its place instead of fortifying the ruin.
func TestBuildOrderBuildsACastleOnARuinedVillage(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeVillage, Level: 0, TerritoryID: "AAA"})
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeCastle}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	infrastructure := ctx.infrastructureAt("AAA")
	if infrastructure == nil || !infrastructure.IsLiveCastle() {
		t.Fatalf("infrastructure = %#v, events = %#v, want a castle replacing the ruined village", infrastructure, ctx.events)
	}
	if got := capitalID(t, ctx.state, "P1"); got != infrastructure.ID {
		t.Errorf("capital = %q, want the new castle %q", got, infrastructure.ID)
	}
}

// TestElectCapitalRejectsARuinedCastle verifies that a ruin cannot be elected
// capital.
func TestElectCapitalRejectsARuinedCastle(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA", Ruined: true})
	ctx := newResolutionContext(state, testBalance())
	electCapitalOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "capital_castle_ruined" {
		t.Fatalf("rejected events = %#v, want capital_castle_ruined", rejected)
	}
}

// TestBuildOrderRebuildKeepsTheFief verifies that C C on a ruined castle that
// is a fief capital rebuilds it without dissolving the fief.
func TestBuildOrderRebuildKeepsTheFief(t *testing.T) {
	ctx := ruinedFiefCapitalContext(t)
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeCastle}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	if infrastructure := ctx.infrastructureAt("AAA"); infrastructure == nil || infrastructure.ID != "I0" || !infrastructure.IsLiveCastle() {
		t.Fatalf("infrastructure = %#v, events = %#v, want castle I0 rebuilt", infrastructure, ctx.events)
	}
	if ctx.fiefByCapital("AAA") == nil {
		t.Errorf("fiefs = %#v, want F1 kept after rebuilding its capital", ctx.state.Fiefs)
	}
	if dissolved := eventsOfType(ctx.events, EventTypeFiefDissolved); len(dissolved) != 0 {
		t.Errorf("dissolved events = %#v, want none", dissolved)
	}
}

// TestBuildOrderRejectsReplacingARuinedFiefCapital verifies that only C C can
// be built on a ruined fief capital: any other build is rejected and the fief
// is kept.
func TestBuildOrderRejectsReplacingARuinedFiefCapital(t *testing.T) {
	ctx := ruinedFiefCapitalContext(t)
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeSupplyDepot}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "ruined_fief_capital_rebuild_only" {
		t.Fatalf("rejected events = %#v, want ruined_fief_capital_rebuild_only", rejected)
	}
	if infrastructure := ctx.infrastructureAt("AAA"); infrastructure == nil || infrastructure.ID != "I0" {
		t.Errorf("infrastructure = %#v, want the ruined castle I0 kept", infrastructure)
	}
	if ctx.fiefByCapital("AAA") == nil {
		t.Errorf("fiefs = %#v, want F1 kept", ctx.state.Fiefs)
	}
	if got := ctx.state.TerritoryStates["AAA"].Resources; got != 10 {
		t.Errorf("stock = %d, want unchanged 10", got)
	}
}

// ruinedFiefCapitalContext is a winter where P1 holds the fief F1 whose
// capital AAA is a ruined castle I0 with 10 R in stock.
func ruinedFiefCapitalContext(t *testing.T) *resolutionContext {
	t.Helper()
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	setTerritoryOwner(state, "BBB", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA", Ruined: true})
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB"}, OwnerID: "P1",
	}}
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	return ctx
}
