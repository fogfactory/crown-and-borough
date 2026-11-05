package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// pillageScenario resolves a pillage by P1's army on AAA over the given
// infrastructure; BBB is P1's capital castle, so it receives the loot.
func pillageScenario(t *testing.T, infrastructure models.Infrastructure, stock int) Resolution {
	t.Helper()
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	infrastructure.ID = "I1"
	infrastructure.TerritoryID = "AAA"
	addInfrastructure(state, infrastructure)
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	setCapital(state, "P1", "I2")
	setTerritoryResources(state, "AAA", stock)
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypePillage, PositionID: "AAA"})
	validateTestState(t, state)
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return resolution
}

func pillagedInfrastructure(state *models.GameState, id models.InfraID) *models.Infrastructure {
	for index := range state.Infrastructures {
		if state.Infrastructures[index].ID == id {
			return &state.Infrastructures[index]
		}
	}
	return nil
}

func pillageEffectOf(t *testing.T, events []Event) PillageEffect {
	t.Helper()
	pillages := eventsOfType(events, EventTypePillage)
	if len(pillages) != 1 {
		t.Fatalf("pillage events = %#v, want exactly one", pillages)
	}
	return pillages[0].PillageEffect
}

func TestPillageDegradesInfrastructureByType(t *testing.T) {
	tests := []struct {
		name        string
		in          models.Infrastructure
		wantEffect  PillageEffect
		wantPresent bool
		wantLevel   int
		wantWall    bool
		wantRuined  bool
	}{
		{"mill level 2 loses a level", models.Infrastructure{Type: models.InfraTypeMill, Level: 2}, PillageEffectDowngraded, true, 1, false, false},
		{"mill level 1 disappears", models.Infrastructure{Type: models.InfraTypeMill, Level: 1}, PillageEffectDestroyed, false, 0, false, false},
		{"fortified village loses its wall only", models.Infrastructure{Type: models.InfraTypeVillage, Level: 2, Fortified: true}, PillageEffectUnfortified, true, 2, false, false},
		{"village loses a level", models.Infrastructure{Type: models.InfraTypeVillage, Level: 2}, PillageEffectDowngraded, true, 1, false, false},
		{"level 1 village falls to a level-0 ruin", models.Infrastructure{Type: models.InfraTypeVillage, Level: 1}, PillageEffectRuined, true, 0, false, false},
		{"castle becomes a ruin", models.Infrastructure{Type: models.InfraTypeCastle, Level: 1}, PillageEffectRuined, true, 1, false, true},
		{"supply depot disappears", models.Infrastructure{Type: models.InfraTypeSupplyDepot, Level: 1}, PillageEffectDestroyed, false, 0, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolution := pillageScenario(t, test.in, 0)
			if got := pillageEffectOf(t, resolution.Events); got != test.wantEffect {
				t.Errorf("effect = %q, want %q", got, test.wantEffect)
			}
			got := pillagedInfrastructure(resolution.State, "I1")
			if (got != nil) != test.wantPresent {
				t.Fatalf("infrastructure = %#v, want present=%v", got, test.wantPresent)
			}
			if got == nil {
				return
			}
			if got.Level != test.wantLevel || got.Fortified != test.wantWall || got.Ruined != test.wantRuined {
				t.Errorf("infrastructure = %#v, want level %d, fortified %v, ruined %v", got, test.wantLevel, test.wantWall, test.wantRuined)
			}
		})
	}
}

func TestPillageOfRuinIsInvalid(t *testing.T) {
	for name, in := range map[string]models.Infrastructure{
		"ruined castle":   {Type: models.InfraTypeCastle, Level: 1, Ruined: true},
		"level-0 village": {Type: models.InfraTypeVillage, Level: 0},
	} {
		t.Run(name, func(t *testing.T) {
			resolution := pillageScenario(t, in, 0)
			if containsEvent(resolution.Events, EventTypePillage) {
				t.Errorf("events = %#v, want no pillage of a ruin", resolution.Events)
			}
			if got := pillagedInfrastructure(resolution.State, "I1"); got == nil || got.Level != in.Level || got.Ruined != in.Ruined {
				t.Errorf("infrastructure = %#v, want it unchanged", got)
			}
		})
	}
}

// The stock of a source that stops being one is loot on top of the bonus.
func TestPillageLootsTheStockOfAFallenSource(t *testing.T) {
	base := testBalance().PillageBonus + 2*testBalance().TerritoryIncome
	castle := pillageScenario(t, models.Infrastructure{Type: models.InfraTypeCastle, Level: 1}, 3)
	if got := resolutionResources(castle, "BBB"); got != base+3 {
		t.Errorf("capital stock after ruining a castle holding 3 = %d, want %d", got, base+3)
	}
	if got := resolutionResources(castle, "AAA"); got != 0 {
		t.Errorf("ruined castle stock = %d, want 0", got)
	}
	still := pillageScenario(t, models.Infrastructure{Type: models.InfraTypeVillage, Level: 2}, 3)
	if got := resolutionResources(still, "AAA"); got < 3 {
		t.Errorf("stock of a village that stays a source = %d, want it kept", got)
	}
}

func resolutionResources(resolution Resolution, id models.TerritoryID) int {
	return resolution.State.TerritoryStates[id].Resources
}

func TestPillageRuinsPlayerCapitalCastleButNotFiefCapital(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	setCapital(state, "P1", "I1")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypePillage, PositionID: "AAA"})
	validateTestState(t, state)
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if capital := resolution.State.Players[0].CapitalCastleID; capital != nil {
		t.Errorf("capital = %v, want it cleared: a ruined castle outside a fief is no capital", *capital)
	}
}

func TestRuinedCastleOffersNoDefense(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		nil,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA", Ruined: true})
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB"}, OwnerID: "P1",
	}}
	ctx := newResolutionContext(state, testBalance())
	if got := ctx.fortificationBonus("AAA"); got != 0 {
		t.Errorf("fortification bonus of a ruined fief capital = %d, want 0", got)
	}
	if !ctx.hasActiveSettlement("AAA") {
		t.Errorf("a ruined castle that is a fief capital must stay a supply source")
	}
}

func TestRuinedSettlementsAreNotSupplySources(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB", "CCC"), territory("BBB", "BBB", "AAA"), territory("CCC", "CCC", "AAA")},
		nil,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA", Ruined: true})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 0, TerritoryID: "BBB"})
	addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "CCC"})
	ctx := newResolutionContext(state, testBalance())
	for id, want := range map[models.TerritoryID]bool{"AAA": false, "BBB": false, "CCC": true} {
		if got := ctx.hasActiveSettlement(id); got != want {
			t.Errorf("hasActiveSettlement(%s) = %v, want %v", id, got, want)
		}
	}
}

// A ruined castle outside every fief is inert for retreats; a ruined fief
// capital keeps blocking them.
func TestRetreatAnchoringOfRuinedCastles(t *testing.T) {
	for _, test := range []struct {
		name      string
		fief      bool
		wantAlive bool
	}{
		{"ruined fief capital stays anchored and blocks", true, false},
		{"ruined castle outside a fief accepts the retreat", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := testState(t,
				[]models.Territory{
					territory("AAA", "AAA", "BBB", "CCC"),
					territory("BBB", "BBB", "AAA"),
					territory("CCC", "CCC", "AAA", "DDD", "EEE"),
					territory("DDD", "DDD", "CCC", "EEE"),
					territory("EEE", "EEE", "CCC", "DDD"),
				},
				[]models.Army{
					{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
					{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
				},
			)
			addNoble(state, "N1", "ONE", "P1", "AAA")
			addNoble(state, "N2", "TWO", "P2", "BBB")
			addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CCC", Ruined: true})
			if test.fief {
				state.Fiefs = []models.Fief{{
					ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "CCC",
					Territories: []models.TerritoryID{"CCC", "DDD", "EEE"}, OwnerID: "P3",
				}}
			}
			addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
			addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
			validateTestState(t, state)
			resolution, err := Resolve(state, testBalance())
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got := hasArmy(resolution.State, "A1"); got != test.wantAlive {
				t.Errorf("A1 alive = %v, want %v", got, test.wantAlive)
			}
		})
	}
}
