package engine

import (
	"errors"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestFindSupplyLineUsesDepotRangeAndReconstructsPath(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
			supplyTerritory("CCC", "CCC", models.TerrainPlain, "BBB", "DDD"),
			supplyTerritory("DDD", "DDD", models.TerrainPlain, "CCC", "EEE"),
			supplyTerritory("EEE", "EEE", models.TerrainPlain, "DDD"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "EEE", Size: 2}},
	)
	for _, territoryID := range []models.TerritoryID{"AAA", "BBB", "CCC", "DDD"} {
		setTerritoryOwner(state, territoryID, "P1")
	}
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeSupplyDepot, Level: 1, TerritoryID: "CCC"})
	validateTestState(t, state)
	before := cloneGameState(state)

	line, err := FindSupplyLine(state, testBalance(), "EEE")
	if err != nil {
		t.Fatalf("FindSupplyLine: %v", err)
	}
	if line.Source == nil || *line.Source != "AAA" {
		t.Fatalf("source = %v, want AAA", line.Source)
	}
	if line.Distance != 4 || line.Rations != 1 || line.Demand != 1 {
		t.Errorf("line details = %#v, want distance 4, one ration, demand 1", line)
	}
	if want := []models.TerritoryID{"AAA", "BBB", "CCC", "DDD", "EEE"}; !reflect.DeepEqual(line.Path, want) {
		t.Errorf("path = %v, want %v", line.Path, want)
	}
	if want := []models.TerritoryID{"AAA", "BBB", "CCC", "DDD", "EEE"}; !reflect.DeepEqual(line.Reachable, want) {
		t.Errorf("reachable = %v, want %v", line.Reachable, want)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("FindSupplyLine mutated its input")
	}
}

func TestFindSupplyLineHandlesLocalRationsAndMissingSources(t *testing.T) {
	t.Run("local rations", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
		)

		line, err := FindSupplyLine(state, testBalance(), "AAA")
		if err != nil {
			t.Fatalf("FindSupplyLine: %v", err)
		}
		if !line.SelfSupplied || line.Rations != 1 || line.Demand != 0 {
			t.Errorf("line = %#v, want local self-supply", line)
		}
		if line.Source != nil || len(line.Path) != 0 || len(line.Reachable) != 0 {
			t.Errorf("local supply line = %#v, want no remote source or path", line)
		}
	})

	t.Run("source zone on a self-supplied army territory", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
		)
		setTerritoryOwner(state, "BBB", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		validateTestState(t, state)

		line, err := FindSupplyLine(state, testBalance(), "AAA")
		if err != nil {
			t.Fatalf("FindSupplyLine: %v", err)
		}
		if !line.SelfSupplied || !reflect.DeepEqual(line.Reachable, []models.TerritoryID{"AAA", "BBB"}) {
			t.Errorf("line = %#v, want self-supply and source zone", line)
		}
	})

	t.Run("no reachable source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainMountain)},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
		)

		line, err := FindSupplyLine(state, testBalance(), "AAA")
		if err != nil {
			t.Fatalf("FindSupplyLine: %v", err)
		}
		if line.SelfSupplied || line.Demand != 1 || line.Source != nil {
			t.Errorf("line = %#v, want uncovered demand", line)
		}
	})
}

func TestFindSupplyLineBreaksEqualDistanceByTerritoryID(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("ZZZ", "ZZZ", models.TerrainPlain, "MID"),
			supplyTerritory("MID", "MID", models.TerrainPlain, "ZZZ", "AAA"),
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "MID"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "MID", Size: 2}},
	)
	setTerritoryOwner(state, "ZZZ", "P1")
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "ZZZ"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)

	line, err := FindSupplyLine(state, testBalance(), "MID")
	if err != nil {
		t.Fatalf("FindSupplyLine: %v", err)
	}
	if line.Source == nil || *line.Source != "AAA" {
		t.Fatalf("source = %v, want AAA", line.Source)
	}
	if want := []models.TerritoryID{"AAA", "MID"}; !reflect.DeepEqual(line.Path, want) {
		t.Errorf("path = %v, want %v", line.Path, want)
	}
}

func TestFindSupplyZoneForControlledCastleAndVillage(t *testing.T) {
	for _, test := range []struct {
		name          string
		kind          models.InfraType
		wantReachable []models.TerritoryID
	}{
		{
			name:          "castle",
			kind:          models.InfraTypeCastle,
			wantReachable: []models.TerritoryID{"AAA", "BBB", "CCC", "DDD"},
		},
		{
			name:          "village",
			kind:          models.InfraTypeVillage,
			wantReachable: []models.TerritoryID{"AAA", "BBB", "CCC", "DDD"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := testState(t,
				[]models.Territory{
					supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
					supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
					supplyTerritory("CCC", "CCC", models.TerrainPlain, "BBB", "DDD"),
					supplyTerritory("DDD", "DDD", models.TerrainPlain, "CCC", "EEE"),
					supplyTerritory("EEE", "EEE", models.TerrainPlain, "DDD", "FFF"),
					supplyTerritory("FFF", "FFF", models.TerrainPlain, "EEE"),
				},
				nil,
			)
			setTerritoryOwner(state, "AAA", "P1")
			addInfrastructure(state, models.Infrastructure{ID: "I1", Type: test.kind, Level: 1, TerritoryID: "AAA"})
			validateTestState(t, state)

			zone, err := FindSupplyZone(state, testBalance(), "AAA")
			if err != nil {
				t.Fatalf("FindSupplyZone: %v", err)
			}
			if zone.Kind != SupplyLineKindSource || zone.ArmyOwner != "P1" || zone.Source == nil || *zone.Source != "AAA" {
				t.Errorf("zone = %#v, want source AAA owned by P1", zone)
			}
			if !reflect.DeepEqual(zone.Reachable, test.wantReachable) {
				t.Errorf("reachable = %v, want %v", zone.Reachable, test.wantReachable)
			}
			if len(zone.Path) != 0 {
				t.Errorf("source zone path = %v, want no army path", zone.Path)
			}
		})
	}

	t.Run("depot is not a source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
			nil,
		)
		setTerritoryOwner(state, "AAA", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeSupplyDepot, Level: 1, TerritoryID: "AAA"})
		validateTestState(t, state)

		if _, err := FindSupplyZone(state, testBalance(), "AAA"); !errors.Is(err, ErrSupplyLineNoSource) {
			t.Errorf("depot source error = %v, want ErrSupplyLineNoSource", err)
		}
	})

	t.Run("depot is not an army source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainMountain)},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
		)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeSupplyDepot, Level: 1, TerritoryID: "AAA"})
		validateTestState(t, state)

		line, err := FindSupplyLine(state, testBalance(), "AAA")
		if err != nil {
			t.Fatalf("FindSupplyLine: %v", err)
		}
		if line.Source != nil || len(line.Reachable) != 0 {
			t.Errorf("line = %#v, want no army source or reachable source zone", line)
		}
	})
}

func TestFindSupplyPrefersArmyOnAControlledSource(t *testing.T) {
	state := testState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)

	selection, err := FindSupply(state, testBalance(), "AAA")
	if err != nil {
		t.Fatalf("FindSupply: %v", err)
	}
	if selection.Kind != SupplyLineKindArmy || selection.ArmySize != 2 {
		t.Errorf("selection = %#v, want army selection to take precedence", selection)
	}
}

func TestFindSupplyLineRejectsUnavailableTargets(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain),
			supplyTerritory("BBB", "BBB", models.TerrainPlain),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)

	if _, err := FindSupplyLine(state, testBalance(), "ZZZ"); !errors.Is(err, ErrSupplyLineUnknownTerritory) {
		t.Errorf("unknown territory error = %v, want ErrSupplyLineUnknownTerritory", err)
	}
	if _, err := FindSupplyLine(state, testBalance(), "BBB"); !errors.Is(err, ErrSupplyLineNoArmy) {
		t.Errorf("empty territory error = %v, want ErrSupplyLineNoArmy", err)
	}
	if _, err := FindSupplyZone(state, testBalance(), "BBB"); !errors.Is(err, ErrSupplyLineNoSource) {
		t.Errorf("empty source error = %v, want ErrSupplyLineNoSource", err)
	}
	state.Season = models.SeasonWinter
	if _, err := FindSupplyLine(state, testBalance(), "AAA"); !errors.Is(err, ErrSupplyLineWinter) {
		t.Errorf("winter error = %v, want ErrSupplyLineWinter", err)
	}
	if _, err := FindSupplyZone(state, testBalance(), "AAA"); !errors.Is(err, ErrSupplyLineWinter) {
		t.Errorf("winter source error = %v, want ErrSupplyLineWinter", err)
	}
	if _, err := FindSupply(state, testBalance(), "AAA"); !errors.Is(err, ErrSupplyLineWinter) {
		t.Errorf("winter selection error = %v, want ErrSupplyLineWinter", err)
	}
}
