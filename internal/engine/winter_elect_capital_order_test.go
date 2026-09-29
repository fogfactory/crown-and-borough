package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestElectCapitalOrderApply(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	ctx := newResolutionContext(state, testBalance())
	electCapitalOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	if player := ctx.playerByID("P1"); player == nil || player.CapitalCastleID == nil || *player.CapitalCastleID != "I1" {
		t.Fatalf("capital = %#v, want I1", player)
	}
	if len(eventsOfType(ctx.events, EventTypeCapitalElected)) != 1 {
		t.Fatalf("capital events = %#v, want one event", ctx.events)
	}
}

// TestElectCapitalOrderRejectsFortifiedVillage verifies that a fortified
// village can never be elected a player's capital: E C still requires an
// actual controlled castle (#193).
func TestElectCapitalOrderRejectsFortifiedVillage(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA", Fortified: true})
	ctx := newResolutionContext(state, testBalance())
	electCapitalOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	if player := ctx.playerByID("P1"); player == nil || player.CapitalCastleID != nil {
		t.Fatalf("capital = %#v, want none: a fortified village is not a castle", player)
	}
	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "capital_requires_controlled_castle" {
		t.Fatalf("rejected events = %#v, want capital_requires_controlled_castle", rejected)
	}
}

// TestElectCapitalOrderRejectsOccupiedTerritory verifies that designating a
// capital on a territory occupied against its controller is rejected
// (titres.md, #196).
func TestElectCapitalOrderRejectsOccupiedTerritory(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")},
		[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 2}},
	)
	holdAsFiefMember(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	ctx := newResolutionContext(state, testBalance())
	electCapitalOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	if player := ctx.playerByID("P1"); player != nil && player.CapitalCastleID != nil {
		t.Fatalf("capital = %#v, want none elected on an occupied territory", player)
	}
	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "territory_occupied_by_other_player" {
		t.Fatalf("rejected events = %#v, want territory_occupied_by_other_player", rejected)
	}
}
