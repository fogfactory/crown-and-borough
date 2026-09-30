package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// prosperityBalance returns testBalance with a small, explicit prosperity
// threshold: unlike testBalance's default (deliberately inert so unrelated
// winter tests never trigger a founding by accident), these tests want
// prosperity active.
func prosperityBalance(threshold int) assetgen.Balance {
	balance := testBalance()
	balance.ProsperityLossThreshold = threshold
	return balance
}

// addVillageWithResources plants a level-1 village at territoryID, owned by
// ownerID, with resources R of stock: at WinterStockDivisor 2 (testBalance),
// conservation keeps ceil(resources/2) and loses floor(resources/2), so an
// even resources value losses exactly resources/2.
func addVillageWithResources(state *models.GameState, infraID models.InfraID, territoryID models.TerritoryID, ownerID models.PlayerID, resources int) {
	addInfrastructure(state, models.Infrastructure{ID: infraID, Type: models.InfraTypeVillage, Level: 1, TerritoryID: territoryID})
	setTerritoryOwner(state, territoryID, ownerID)
	setTerritoryResources(state, territoryID, resources)
}

func TestProsperityFoundingThresholdCounts(t *testing.T) {
	t.Run("below threshold founds nothing", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{
			territory("AAA", "AAA", "AAB"),
			territory("AAB", "AAB", "AAA", "AAC"),
			territory("AAC", "AAC", "AAB"),
		}, nil)
		addVillageWithResources(state, "I1", "AAA", "P1", 6) // loss = 3
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if founded := eventsOfType(resolution.Events, EventTypeProsperityFounded); len(founded) != 0 {
			t.Fatalf("founded = %#v, want none below threshold", founded)
		}
	})

	t.Run("one multiple founds one village", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{
			territory("AAA", "AAA", "AAB"),
			territory("AAB", "AAB", "AAA", "AAC"),
			territory("AAC", "AAC", "AAB"),
		}, nil)
		addVillageWithResources(state, "I1", "AAA", "P1", 8) // loss = 4
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		founded := eventsOfType(resolution.Events, EventTypeProsperityFounded)
		if len(founded) != 1 || founded[0].SourceID != "AAA" || founded[0].DestinationID != "AAC" {
			t.Fatalf("founded = %#v, want one village at AAC from AAA", founded)
		}
		if founded[0].Reason != prosperityReasonFree {
			t.Errorf("reason = %q, want %q", founded[0].Reason, prosperityReasonFree)
		}

		report := BuildReport(state, resolution.State, resolution.Events, nil)
		if report.Winter == nil {
			t.Fatalf("report.Winter is nil")
		}
		found := false
		for _, investment := range report.Winter.Investments {
			if investment.Kind != EventTypeProsperityFounded {
				continue
			}
			found = true
			if investment.Source != "AAA" || investment.Target != "AAC" || investment.Reason != prosperityReasonFree {
				t.Errorf("investment = %#v, want source AAA, target AAC, reason %q", investment, prosperityReasonFree)
			}
		}
		if !found {
			t.Errorf("report investments = %#v, want a prosperity_founded entry", report.Winter.Investments)
		}
	})

	t.Run("two multiples found two villages, ranked by trigram on equal loss", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{
			territory("AAA", "AAA", "AAB"),
			territory("AAB", "AAB", "AAA", "AAC"),
			territory("AAC", "AAC", "AAB"),
			territory("BAA", "BAA", "BAB"),
			territory("BAB", "BAB", "BAA", "BAC"),
			territory("BAC", "BAC", "BAB"),
		}, nil)
		addVillageWithResources(state, "I1", "AAA", "P1", 8) // loss = 4
		addVillageWithResources(state, "I2", "BAA", "P2", 8) // loss = 4, total = 8 = 2*4
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		founded := eventsOfType(resolution.Events, EventTypeProsperityFounded)
		if len(founded) != 2 {
			t.Fatalf("founded = %#v, want two villages", founded)
		}
		// AAA sorts before BAA: with equal loss, it is founded first.
		if founded[0].SourceID != "AAA" || founded[0].DestinationID != "AAC" {
			t.Errorf("first founding = %#v, want from AAA to AAC", founded[0])
		}
		if founded[1].SourceID != "BAA" || founded[1].DestinationID != "BAC" {
			t.Errorf("second founding = %#v, want from BAA to BAC", founded[1])
		}
	})
}

func TestProsperityRankingByLossThenTrigram(t *testing.T) {
	// AAA sorts before ZZZ, but ZZZ loses more: only one founding triggers
	// (total in [N, 2N)), and it must come from the territory that lost more,
	// not the one with the lower trigram.
	state := winterTestState(t, []models.Territory{
		territory("AAA", "AAA", "AAB"),
		territory("AAB", "AAB", "AAA", "AAC"),
		territory("AAC", "AAC", "AAB"),
		territory("ZZZ", "ZZZ", "ZZY"),
		territory("ZZY", "ZZY", "ZZZ", "ZZX"),
		territory("ZZX", "ZZX", "ZZY"),
	}, nil)
	addVillageWithResources(state, "I1", "AAA", "P1", 4) // loss = 2
	addVillageWithResources(state, "I2", "ZZZ", "P2", 8) // loss = 4, total = 6, in [4, 8)
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	founded := eventsOfType(resolution.Events, EventTypeProsperityFounded)
	if len(founded) != 1 || founded[0].SourceID != "ZZZ" || founded[0].DestinationID != "ZZX" {
		t.Fatalf("founded = %#v, want the single founding from ZZZ (higher loss)", founded)
	}
}

func TestProsperityDestinationPriority(t *testing.T) {
	t.Run("fief wins even farther than a merely controlled tile", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{
			territory("ORI", "ORI", "MID"),
			territory("MID", "MID", "ORI", "CTL", "BRI"),
			territory("CTL", "CTL", "MID"),
			territory("BRI", "BRI", "MID", "FIE"),
			territory("FIE", "FIE", "BRI"),
			// A minimal disconnected fief group: FIE plus two extra members,
			// only used to satisfy FiefMinTerritories.
			territory("CFA", "CFA", "CFB"),
			territory("CFB", "CFB", "CFA"),
		}, nil)
		addVillageWithResources(state, "I1", "ORI", "P1", 8) // loss = 4
		setTerritoryOwner(state, "CTL", "P1")                // closer, merely controlled
		state.Fiefs = []models.Fief{{
			ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "CFA",
			Territories: []models.TerritoryID{"CFA", "CFB", "FIE"}, OwnerID: "P1",
		}}
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		founded := eventsOfType(resolution.Events, EventTypeProsperityFounded)
		if len(founded) != 1 || founded[0].DestinationID != "FIE" || founded[0].Reason != prosperityReasonFief {
			t.Fatalf("founded = %#v, want the fief tile FIE despite being farther", founded)
		}
	})

	t.Run("controlled wins over a nearer uncontrolled free tile", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{
			territory("ORI", "ORI", "MID"),
			territory("MID", "MID", "ORI", "OPN"),
			territory("OPN", "OPN", "MID", "CTL"),
			territory("CTL", "CTL", "OPN"),
		}, nil)
		addVillageWithResources(state, "I1", "ORI", "P1", 8) // loss = 4
		setTerritoryOwner(state, "CTL", "P1")
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		founded := eventsOfType(resolution.Events, EventTypeProsperityFounded)
		if len(founded) != 1 || founded[0].DestinationID != "CTL" || founded[0].Reason != prosperityReasonControlled {
			t.Fatalf("founded = %#v, want the controlled tile CTL despite OPN being nearer", founded)
		}
	})
}

func TestProsperityNonAdjacencyConstraint(t *testing.T) {
	// NEAR is the nearest free tile, but it sits next to the castle CAS: it
	// must be skipped in favor of the farther, non-adjacent FAR.
	state := winterTestState(t, []models.Territory{
		territory("ORI", "ORI", "MID"),
		territory("MID", "MID", "ORI", "NEA"),
		territory("NEA", "NEA", "MID", "CAS", "FAR"),
		territory("CAS", "CAS", "NEA"),
		territory("FAR", "FAR", "NEA"),
	}, nil)
	addVillageWithResources(state, "I1", "ORI", "P1", 8) // loss = 4
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAS"})
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	founded := eventsOfType(resolution.Events, EventTypeProsperityFounded)
	if len(founded) != 1 || founded[0].DestinationID != "FAR" {
		t.Fatalf("founded = %#v, want FAR (NEA is adjacent to the castle CAS)", founded)
	}
}

// prosperityDeadEndState builds a map with no free tile eligible anywhere: ORI
// loses stock, MID is adjacent to ORI's village, and NEA is adjacent to the
// castle CAS. extra, when non-empty, is appended to NEA's dead end so the
// cascading fallback has a candidate to upgrade.
func prosperityDeadEndState(t *testing.T, extra *models.Territory) *models.GameState {
	t.Helper()
	territories := []models.Territory{
		territory("ORI", "ORI", "MID"),
		territory("MID", "MID", "ORI", "NEA"),
		territory("CAS", "CAS", "NEA"),
	}
	neaNeighbors := []models.TerritoryID{"MID", "CAS"}
	if extra != nil {
		neaNeighbors = append(neaNeighbors, extra.ID)
	}
	territories = append(territories, territory("NEA", "NEA", neaNeighbors...))
	if extra != nil {
		territories = append(territories, *extra)
	}
	state := winterTestState(t, territories, nil)
	addVillageWithResources(state, "I1", "ORI", "P1", 8) // loss = 4
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAS"})
	return state
}

func TestProsperityCascadingFallback(t *testing.T) {
	t.Run("upgrades the closest supply depot when no free tile qualifies", func(t *testing.T) {
		depot := territory("DEP", "DEP", "NEA")
		state := prosperityDeadEndState(t, &depot)
		addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeSupplyDepot, Level: 1, TerritoryID: "DEP"})
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		founded := eventsOfType(resolution.Events, EventTypeProsperityFounded)
		if len(founded) != 1 || founded[0].DestinationID != "DEP" || founded[0].Reason != prosperityReasonDepotUpgraded {
			t.Fatalf("founded = %#v, want the depot at DEP upgraded", founded)
		}
		infrastructure := infrastructureAtState(t, resolution.State, "DEP")
		if infrastructure.ID != "I3" || infrastructure.Type != models.InfraTypeVillage || infrastructure.Level != 1 {
			t.Fatalf("DEP infrastructure = %#v, want I3 upgraded to a level-1 village", infrastructure)
		}
	})

	t.Run("upgrades the closest mill when no depot exists either", func(t *testing.T) {
		mill := territory("MIL", "MIL", "NEA")
		state := prosperityDeadEndState(t, &mill)
		addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		founded := eventsOfType(resolution.Events, EventTypeProsperityFounded)
		if len(founded) != 1 || founded[0].DestinationID != "MIL" || founded[0].Reason != prosperityReasonMillUpgraded {
			t.Fatalf("founded = %#v, want the mill at MIL upgraded", founded)
		}
		infrastructure := infrastructureAtState(t, resolution.State, "MIL")
		if infrastructure.ID != "I3" || infrastructure.Type != models.InfraTypeVillage || infrastructure.Level != 1 {
			t.Fatalf("MIL infrastructure = %#v, want I3 upgraded to a level-1 village", infrastructure)
		}
	})

	t.Run("does nothing when neither a depot nor a mill exists", func(t *testing.T) {
		state := prosperityDeadEndState(t, nil)
		validateTestState(t, state)
		infrastructuresBefore := len(state.Infrastructures)

		resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if founded := eventsOfType(resolution.Events, EventTypeProsperityFounded); len(founded) != 0 {
			t.Fatalf("founded = %#v, want none: no free tile and no depot or mill to upgrade", founded)
		}
		if len(resolution.State.Infrastructures) != infrastructuresBefore {
			t.Fatalf("infrastructures = %#v, want unchanged", resolution.State.Infrastructures)
		}
	})
}

func TestProsperityFoundedVillageControllerFollowsDestination(t *testing.T) {
	state := winterTestState(t, []models.Territory{
		territory("ORI", "ORI", "MID"),
		territory("MID", "MID", "ORI", "DST"),
		territory("DST", "DST", "MID"),
	}, nil)
	addVillageWithResources(state, "I1", "ORI", "P1", 8) // loss = 4
	setTerritoryOwner(state, "DST", "P2")                // controlled by a different player than the origin
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	founded := eventsOfType(resolution.Events, EventTypeProsperityFounded)
	if len(founded) != 1 || founded[0].DestinationID != "DST" {
		t.Fatalf("founded = %#v, want a founding at DST", founded)
	}
	if founded[0].OwnerID != "P2" {
		t.Errorf("owner = %q, want P2 (the destination's controller), not P1 (the origin's)", founded[0].OwnerID)
	}
}

func TestProsperityFoundedVillageNeutralWhenDestinationUncontrolled(t *testing.T) {
	state := winterTestState(t, []models.Territory{
		territory("ORI", "ORI", "MID"),
		territory("MID", "MID", "ORI", "DST"),
		territory("DST", "DST", "MID"),
	}, nil)
	addVillageWithResources(state, "I1", "ORI", "P1", 8) // loss = 4, DST stays neutral
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	founded := eventsOfType(resolution.Events, EventTypeProsperityFounded)
	if len(founded) != 1 || founded[0].OwnerID != "" {
		t.Fatalf("founded = %#v, want a neutral founding at DST", founded)
	}
}

func TestProsperityOriginUntouched(t *testing.T) {
	state := winterTestState(t, []models.Territory{
		territory("ORI", "ORI", "MID"),
		territory("MID", "MID", "ORI", "DST"),
		territory("DST", "DST", "MID"),
	}, nil)
	addVillageWithResources(state, "I1", "ORI", "P1", 8) // loss = 4
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, prosperityBalance(4), nil)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if founded := eventsOfType(resolution.Events, EventTypeProsperityFounded); len(founded) != 1 {
		t.Fatalf("founded = %#v, want exactly one founding", founded)
	}
	origin := infrastructureAtState(t, resolution.State, "ORI")
	if origin.ID != "I1" || origin.Type != models.InfraTypeVillage || origin.Level != 1 || origin.Fortified {
		t.Fatalf("origin infrastructure = %#v, want unchanged", origin)
	}
	// Only the normal winter conservation applies: ceil(8/2) = 4, not further
	// reduced by the founding it triggered elsewhere.
	if resources := resolution.State.TerritoryStates["ORI"].Resources; resources != 4 {
		t.Errorf("origin resources = %d, want 4 (normal conservation only)", resources)
	}
}

func TestProsperityIgnoresStockSpentByWinterOrders(t *testing.T) {
	// A village starts winter with 20 R and spends 3 R on a same-turn build
	// order, leaving 17 before conservation; conservation then keeps ceil(17/2)
	// = 9 and loses floor(17/2) = 8. Prosperity must count only that 8 R
	// conservation loss, not the 20 - 9 = 11 R difference against the
	// pre-order stock: a threshold of 9 founds nothing under the correct
	// measurement, but would found one village if the order's spend were
	// wrongly folded into the loss.
	state := winterTestState(t, []models.Territory{
		territory("AAA", "AAA", "BBB"),
		territory("BBB", "BBB", "AAA"),
	}, nil)
	addVillageWithResources(state, "I1", "AAA", "P1", 20)
	setTerritoryOwner(state, "BBB", "P1")
	validateTestState(t, state)

	balance := prosperityBalance(9)
	resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "BBB", InfraType: models.InfraTypeSupplyDepot}},
	})
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if resources := resolution.State.TerritoryStates["AAA"].Resources; resources != 9 {
		t.Fatalf("AAA resources = %d, want 9 (ceil(17/2))", resources)
	}
	if founded := eventsOfType(resolution.Events, EventTypeProsperityFounded); len(founded) != 0 {
		t.Fatalf("founded = %#v, want none: conservation loss (8) is below threshold (9) once the order's spend is excluded", founded)
	}
}

func TestResolveWinterRejectsInvalidProsperityThreshold(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	validateTestState(t, state)

	balance := prosperityBalance(0)
	if _, err := ResolveWinter(state, balance, nil); err == nil {
		t.Fatalf("ResolveWinter: want an error for a non-positive prosperity threshold")
	}
}
