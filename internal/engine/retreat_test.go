package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestResolveSelfAttackOnOwnCastle_Size1Succeeds(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	p1 := models.PlayerID("P1")
	bbbState := state.TerritoryStates["BBB"]
	bbbState.OwnerID = &p1
	state.TerritoryStates["BBB"] = bbbState

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	army := armyByID(t, resolution.State, "A1")
	if army.TerritoryID != "BBB" {
		t.Errorf("A1 territory = %q, want BBB (self-capture should succeed)", army.TerritoryID)
	}
	if bbbOwner := resolution.State.TerritoryStates["BBB"].OwnerID; bbbOwner == nil || *bbbOwner != "P1" {
		t.Errorf("BBB owner = %v, want P1", bbbOwner)
	}
}

func TestResolveSelfAttackOnOwnCastle_ContestedByOutsider(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "CCC")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	p1 := models.PlayerID("P1")
	bbbState := state.TerritoryStates["BBB"]
	bbbState.OwnerID = &p1
	state.TerritoryStates["BBB"] = bbbState

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"BBB"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Castle defends with +1 against outsider, causing standoff -> both bounce
	a1 := armyByID(t, resolution.State, "A1")
	if a1.TerritoryID != "AAA" {
		t.Errorf("A1 territory = %q, want AAA after standoff", a1.TerritoryID)
	}
	a2 := armyByID(t, resolution.State, "A2")
	if a2.TerritoryID != "CCC" {
		t.Errorf("A2 territory = %q, want CCC after standoff", a2.TerritoryID)
	}
}

func TestResolveRetreatToControlledEmptyCastle_Bucket1(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CCC"})
	p1 := models.PlayerID("P1")
	cccState := state.TerritoryStates["CCC"]
	cccState.OwnerID = &p1
	state.TerritoryStates["CCC"] = cccState

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	a1 := armyByID(t, resolution.State, "A1")
	if a1.TerritoryID != "CCC" {
		t.Errorf("A1 territory = %q, want CCC after retreat to controlled castle", a1.TerritoryID)
	}
}

func TestResolveRetreatToControlledEmptyCastle_OverridesAttackedTerritory(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA", "DDD"),
			territory("DDD", "DDD", "CCC"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
			{ID: "A3", OwnerID: "P3", TerritoryID: "DDD", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N3", "THR", "P3", "DDD")
	setNobleStatus(state, "N3", models.NobleStatusHostage)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CCC"})
	p1 := models.PlayerID("P1")
	cccState := state.TerritoryStates["CCC"]
	cccState.OwnerID = &p1
	state.TerritoryStates["CCC"] = cccState

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	// A3 attacks CCC (force 1 vs castle bonus 1 -> standoff, bounces back to DDD)
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"CCC"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Even though CCC was attacked this turn, A1 can retreat there because it is controlled by P1!
	a1 := armyByID(t, resolution.State, "A1")
	if a1.TerritoryID != "CCC" {
		t.Errorf("A1 territory = %q, want CCC (bucket 1 overrides attackedTerritories)", a1.TerritoryID)
	}
}

func TestResolveRetreatNeutralEmptyCastleExcluded(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	// CCC is neutral castle
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CCC"})

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Neutral castle cannot be retreated into -> A1 destroyed
	if hasArmy(resolution.State, "A1") {
		t.Error("A1 should be destroyed, neutral castle cannot receive retreat")
	}
}

func TestResolveRetreatEnemyEmptyCastleExcluded(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	// CCC is enemy castle controlled by P3
	p3 := models.PlayerID("P3")
	cccState := state.TerritoryStates["CCC"]
	cccState.OwnerID = &p3
	state.TerritoryStates["CCC"] = cccState
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CCC"})

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Enemy castle cannot be retreated into -> A1 destroyed
	if hasArmy(resolution.State, "A1") {
		t.Error("A1 should be destroyed, enemy castle cannot receive retreat")
	}
}

func TestResolveRetreatEmptyOtherBeatsFriendlyArmy(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC", "DDD"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
			territory("DDD", "DDD", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
			{ID: "A3", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N3", "THR", "P1", "DDD")

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeHold, PositionID: "DDD"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// CCC is neutral empty without castle (bucket 2). DDD is friendly army (bucket 3).
	// Bucket 2 beats bucket 3!
	a1 := armyByID(t, resolution.State, "A1")
	if a1.TerritoryID != "CCC" {
		t.Errorf("A1 territory = %q, want CCC (bucket 2 preferred over friendly army)", a1.TerritoryID)
	}
	a3 := armyByID(t, resolution.State, "A3")
	if a3.Size != 1 {
		t.Errorf("A3 size = %d, want 1 (should not have been merged into)", a3.Size)
	}
}

func TestResolveRetreatFriendlyArmySequentialMerge(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "EEE"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "DDD", "EEE"),
			territory("DDD", "DDD", "CCC"),
			territory("EEE", "EEE", "AAA", "CCC"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
			{ID: "A4", OwnerID: "P2", TerritoryID: "DDD", Size: 3},
			{ID: "A5", OwnerID: "P1", TerritoryID: "EEE", Size: 2},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addNoble(state, "N4", "FOU", "P2", "DDD")
	addNoble(state, "N5", "FIV", "P1", "EEE")

	// Supply A4 at DDD with a controlled castle and stock
	addInfrastructure(state, models.Infrastructure{ID: "I4", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "DDD"})
	p2 := models.PlayerID("P2")
	dddState := state.TerritoryStates["DDD"]
	dddState.OwnerID = &p2
	dddState.Resources = 10
	state.TerritoryStates["DDD"] = dddState

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeHold, PositionID: "CCC"})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"CCC"}})
	addChain(t, state, "A5", "N5", models.Order{Type: models.OrderTypeHold, PositionID: "EEE"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// A1 (size 1) dislodged from AAA, retreats to EEE -> +1 troop (no loss)
	// A3 (size 2) dislodged from CCC, retreats to EEE -> +(2-1) = +1 troop (1 troop lost)
	// Initial A5: 2 troops. After sequential merge: 2 + 1 + 1 = 4 troops!
	a5 := armyByID(t, resolution.State, "A5")
	if a5.Size != 4 {
		t.Errorf("A5 size = %d, want 4 after sequential merges (2 + 1 + 1)", a5.Size)
	}
	if hasArmy(resolution.State, "A1") {
		t.Error("A1 should be absorbed and not in state.Armies")
	}
	if hasArmy(resolution.State, "A3") {
		t.Error("A3 should be absorbed and not in state.Armies")
	}
	// Nobles N1 and N3 should now be at EEE
	n1 := nobleByID(t, resolution.State, "N1")
	if n1.LocationID != "EEE" {
		t.Errorf("N1 location = %q, want EEE", n1.LocationID)
	}
	n3 := nobleByID(t, resolution.State, "N3")
	if n3.LocationID != "EEE" {
		t.Errorf("N3 location = %q, want EEE", n3.LocationID)
	}

	// Check retreat event fields
	var a1Retreat, a3Retreat *Event
	for i := range resolution.Events {
		e := &resolution.Events[i]
		if e.Type == EventTypeRetreat && e.ArmyID == "A1" {
			a1Retreat = e
		}
		if e.Type == EventTypeRetreat && e.ArmyID == "A3" {
			a3Retreat = e
		}
	}
	if a1Retreat == nil || a1Retreat.DestinationKind != RetreatDestinationFriendlyArmy || a1Retreat.TroopsMerged != 1 || a1Retreat.TroopsLost != 0 {
		t.Errorf("A1 retreat event = %#v, want kind friendly_army, merged 1, lost 0", a1Retreat)
	}
	if a3Retreat == nil || a3Retreat.DestinationKind != RetreatDestinationFriendlyArmy || a3Retreat.TroopsMerged != 1 || a3Retreat.TroopsLost != 1 {
		t.Errorf("A3 retreat event = %#v, want kind friendly_army, merged 1, lost 1", a3Retreat)
	}
}

func TestResolveRetreatFriendlyArmyMultipleSize1(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "EEE"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "DDD", "EEE"),
			territory("DDD", "DDD", "CCC"),
			territory("EEE", "EEE", "AAA", "CCC"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P2", TerritoryID: "DDD", Size: 2},
			{ID: "A5", OwnerID: "P1", TerritoryID: "EEE", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addNoble(state, "N4", "FOU", "P2", "DDD")
	addNoble(state, "N5", "FIV", "P1", "EEE")

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeHold, PositionID: "CCC"})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"CCC"}})
	addChain(t, state, "A5", "N5", models.Order{Type: models.OrderTypeHold, PositionID: "EEE"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Both size 1 merge: 1 + 1 + 1 = 3
	a5 := armyByID(t, resolution.State, "A5")
	if a5.Size != 3 {
		t.Errorf("A5 size = %d, want 3 (1 + 1 + 1)", a5.Size)
	}
}

func TestResolveRetreatFriendlyArmyDislodgedHostExcluded(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA", "DDD"),
			territory("DDD", "DDD", "CCC"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P2", TerritoryID: "DDD", Size: 2},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addNoble(state, "N4", "FOU", "P2", "DDD")

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeHold, PositionID: "CCC"})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"CCC"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Both A1 and A3 are dislodged. A3 cannot be a retreat destination for A1 (host is dislodged),
	// and A1 cannot be for A3. Both destroyed!
	if hasArmy(resolution.State, "A1") {
		t.Error("A1 should be destroyed")
	}
	if hasArmy(resolution.State, "A3") {
		t.Error("A3 should be destroyed")
	}
}

func TestResolveRetreatSupplyLineTieBreakWithinBucket(t *testing.T) {
	// P1 has a castle at SRC.
	// AAA is under attack from BBB.
	// Adjacent empty controlled territories are C1 and C2.
	// C1 is distance 1 from SRC (adjacent to SRC).
	// C2 is distance 2 from SRC (SRC -> INT -> C2).
	// C2 is lexicographically smaller than C1 ("C1" vs "C2"), wait:
	// Let's name them "ZZZ" (adjacent to SRC, distance 1) and "AAA_CLOSE" etc.
	// If territory "ZZZ" is distance 1 to SRC, and territory "DDD" is distance 2 to SRC:
	// "DDD" < "ZZZ" alphabetically.
	// But "ZZZ" is closer to supply source!
	// So "ZZZ" must be chosen over "DDD"!
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "DDD", "ZZZ"),
			territory("BBB", "BBB", "AAA"),
			territory("DDD", "DDD", "AAA", "INT"),
			territory("INT", "INT", "DDD", "SRC"),
			territory("ZZZ", "ZZZ", "AAA", "SRC"),
			territory("SRC", "SRC", "ZZZ", "INT"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	p1 := models.PlayerID("P1")
	// Make DDD and ZZZ controlled by P1
	dddState := state.TerritoryStates["DDD"]
	dddState.OwnerID = &p1
	state.TerritoryStates["DDD"] = dddState

	zzzState := state.TerritoryStates["ZZZ"]
	zzzState.OwnerID = &p1
	state.TerritoryStates["ZZZ"] = zzzState

	// SRC is a castle controlled by P1
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "SRC"})
	srcState := state.TerritoryStates["SRC"]
	srcState.OwnerID = &p1
	state.TerritoryStates["SRC"] = srcState

	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	a1 := armyByID(t, resolution.State, "A1")
	// ZZZ is distance 1 from SRC; DDD is distance 2 from SRC.
	// Although DDD is alphabetically before ZZZ, ZZZ wins on supply distance!
	if a1.TerritoryID != "ZZZ" {
		t.Errorf("A1 territory = %q, want ZZZ (closer to supply source than DDD)", a1.TerritoryID)
	}
}

func TestResolveRetreatProcessingOrderByOriginTerritory(t *testing.T) {
	// Two armies dislodged at the same time:
	// A10 at AAA (dislodged by A2 from BBB)
	// A1 at ZZZ (dislodged by A3 from YYY)
	// Both have ONLY territory MID as a valid empty retreat destination.
	// Origin territories: AAA vs ZZZ.
	// "AAA" < "ZZZ", so A10 gets processed first and claims MID!
	// A1 has no other choice, collides on MID, so BOTH are destroyed.
	// But what if MID was the only option and A10 claimed it, while ZZZ also tried?
	// Let's test where MID is contested, but AAA army has an alternative ALT!
	// AAA has candidates: MID, ALT.
	// ZZZ has candidate: MID only.
	// If AAA is processed first:
	// AAA claims MID (its first candidate).
	// Then ZZZ is processed: ZZZ has only MID. MID is claimed.
	// Since ZZZ has no alternative, ZZZ collides on MID -> both destroyed.
	// Now what if AAA was at "ZZZ" and ZZZ was at "AAA"?
	// Let's test the claim:
	// Army A (at "AAA") has options [MID].
	// Army B (at "ZZZ") has options [MID, ALT].
	// Because "AAA" is processed first:
	// Army A claims MID.
	// Then Army B (at "ZZZ") is processed: MID is claimed, but Army B has ALT!
	// So Army B takes ALT!
	// Both survive: Army A on MID, Army B on ALT!
	// If armyID ordering was used (A10 vs A1), A1 would have taken MID, and A10 (having no ALT) would collide and both would die!
	// With territory ordering ("AAA" < "ZZZ"): Army at AAA claims MID, army at ZZZ takes ALT!
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "MID"),
			territory("BBB", "BBB", "AAA"),
			territory("MID", "MID", "AAA", "ZZZ"),
			territory("ZZZ", "ZZZ", "YYY", "MID", "ALT"),
			territory("YYY", "YYY", "ZZZ"),
			territory("ALT", "ALT", "ZZZ"),
		},
		[]models.Army{
			{ID: "A10", OwnerID: "P1", TerritoryID: "AAA", Size: 1}, // at AAA
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
			{ID: "A1", OwnerID: "P1", TerritoryID: "ZZZ", Size: 1}, // at ZZZ, but armyID "A1" < "A10"!
			{ID: "A3", OwnerID: "P2", TerritoryID: "YYY", Size: 2},
		},
	)
	addNoble(state, "N10", "TEN", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N1", "ONE", "P1", "ZZZ")
	addNoble(state, "N3", "THR", "P2", "YYY")

	addChain(t, state, "A10", "N10", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "ZZZ"})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "YYY", TargetIDs: []models.TerritoryID{"ZZZ"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// AAA (A10) was processed first by origin territory lex order ("AAA" < "ZZZ"):
	// A10 took MID.
	// ZZZ (A1) was processed second: MID was taken, so A1 took ALT!
	a10 := armyByID(t, resolution.State, "A10")
	if a10.TerritoryID != "MID" {
		t.Errorf("A10 territory = %q, want MID", a10.TerritoryID)
	}
	a1 := armyByID(t, resolution.State, "A1")
	if a1.TerritoryID != "ALT" {
		t.Errorf("A1 territory = %q, want ALT", a1.TerritoryID)
	}
}
