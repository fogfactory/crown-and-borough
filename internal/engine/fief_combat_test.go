package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestFortificationBonusReplacesCastleWithCityBonus verifies that a fief
// capital's castle defends with the city bonus (2, replacing rather than
// stacking with the plain castle bonus of 1), reported as CastleBonus on the
// combat event exactly like an ordinary castle (titres.md).
func TestFortificationBonusReplacesCastleWithCityBonus(t *testing.T) {
	t.Run("plain castle defends with the castle bonus", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "BBB", Size: 1}},
		)
		setTerritoryOwner(state, "AAA", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		keepTestArmiesSupplied(state)
		addNoble(state, "N1", "ONE", "P2", "BBB")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		event, found := combatAt(resolution.Events, "AAA")
		if !found || event.CastleBonus != 1 {
			t.Fatalf("combat at AAA = %#v, found=%t, want castleBonus 1", event, found)
		}
	})

	t.Run("fief capital defends with the city bonus", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB", "ZZZ"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "BBB"),
				territory("ZZZ", "ZZZ", "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "ZZZ", Size: 1}},
		)
		setTerritoryOwner(state, "AAA", "P1")
		setTerritoryOwner(state, "BBB", "P1")
		setTerritoryOwner(state, "CCC", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		state.Fiefs = []models.Fief{{
			ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
			Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1",
		}}
		keepTestArmiesSupplied(state)
		addNoble(state, "N1", "ONE", "P2", "ZZZ")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "ZZZ", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		event, found := combatAt(resolution.Events, "AAA")
		if !found || event.CastleBonus != 2 {
			t.Fatalf("combat at AAA = %#v, found=%t, want castleBonus 2 (city)", event, found)
		}
		// The empty castle threshold rises with it: a lone attacker of force 1
		// cannot overcome the city's bonus of 2, so the attack fails outright.
		if outcome, found := outcomeForArmy(resolution.Events, "A1"); !found || outcome.Outcome != OutcomeFailure {
			t.Errorf("A1 outcome = %#v, found=%t, want failure against the city threshold", outcome, found)
		}
	})
}

// TestFortificationBonusAutoCaptureException verifies that a fief owner
// recapturing (or garrisoning) its own empty city gets no defensive bonus at
// all: the exception already carved out for castleOwnedByAllAttackers
// applies identically to the city bonus.
func TestFortificationBonusAutoCaptureException(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "AAA", "BBB"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 1}},
	)
	setTerritoryOwner(state, "AAA", "P1")
	setTerritoryOwner(state, "CCC", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1",
	}}
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "BBB")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	event, found := combatAt(resolution.Events, "AAA")
	if !found || event.CastleBonus != 0 {
		t.Fatalf("combat at AAA = %#v, found=%t, want castleBonus 0 (auto-capture exception)", event, found)
	}
}

// TestRevoltAgainstCityUsesCityBonus verifies that a revolt against the
// occupant of a fief capital defends with the city bonus.
func TestRevoltAgainstCityUsesCityBonus(t *testing.T) {
	state := effectTestState()
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}}
	armyID := models.ArmyID("A1")
	state.TerritoryStates["AAA"] = models.TerritoryState{OwnerID: ptrIDEngine("P1"), Army: &armyID}
	setTerritoryOwner(state, "BBB", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB"}, OwnerID: "P1",
	}}
	ctx := newResolutionContext(state, testBalance())
	occupant := ctx.currentArmyAt("AAA")
	if occupant == nil {
		t.Fatalf("no occupant at AAA")
	}
	resolveRevoltCombat(ctx, "AAA", 2, occupant)

	event, found := combatAt(ctx.events, "AAA")
	if !found || event.CastleBonus != 2 {
		t.Fatalf("combat at AAA = %#v, found=%t, want castleBonus 2 (city)", event, found)
	}
	// Defense = occupant size (1) + city bonus (2) = 3, which beats a rebel
	// force of 2: the defense holds.
	if event.Defense != 3 || event.Reason != "defense_holds" {
		t.Errorf("event = %#v, want defense 3 holding", event)
	}
}

func ptrIDEngine(id models.PlayerID) *models.PlayerID {
	return &id
}
