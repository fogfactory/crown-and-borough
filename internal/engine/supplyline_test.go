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
			supplyTerritory("T01", "AAA", models.TerrainPlain, "T02"),
			supplyTerritory("T02", "BBB", models.TerrainPlain, "T01", "T03"),
			supplyTerritory("T03", "CCC", models.TerrainPlain, "T02", "T04"),
			supplyTerritory("T04", "DDD", models.TerrainPlain, "T03", "T05"),
			supplyTerritory("T05", "EEE", models.TerrainPlain, "T04"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T05", Size: 2}},
	)
	for _, territoryID := range []models.TerritoryID{"T01", "T02", "T03", "T04"} {
		setTerritoryOwner(state, territoryID, "P1")
	}
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeSupplyDepot, Level: 1, TerritoryID: "T03"})
	validateTestState(t, state)
	before := cloneGameState(state)

	line, err := FindSupplyLine(state, testBalance(), "T05")
	if err != nil {
		t.Fatalf("FindSupplyLine: %v", err)
	}
	if line.Source == nil || *line.Source != "T01" {
		t.Fatalf("source = %v, want T01", line.Source)
	}
	if line.Distance != 4 || line.Rations != 1 || line.Demand != 1 {
		t.Errorf("line details = %#v, want distance 4, one ration, demand 1", line)
	}
	if want := []models.TerritoryID{"T01", "T02", "T03", "T04", "T05"}; !reflect.DeepEqual(line.Path, want) {
		t.Errorf("path = %v, want %v", line.Path, want)
	}
	if want := []models.TerritoryID{"T01", "T02", "T03", "T04", "T05"}; !reflect.DeepEqual(line.Reachable, want) {
		t.Errorf("reachable = %v, want %v", line.Reachable, want)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("FindSupplyLine mutated its input")
	}
}

func TestFindSupplyLineHandlesLocalRationsAndMissingSources(t *testing.T) {
	t.Run("local rations", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("T01", "AAA", models.TerrainPlain)},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
		)

		line, err := FindSupplyLine(state, testBalance(), "T01")
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

	t.Run("no reachable source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("T01", "AAA", models.TerrainMountain)},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
		)

		line, err := FindSupplyLine(state, testBalance(), "T01")
		if err != nil {
			t.Fatalf("FindSupplyLine: %v", err)
		}
		if line.SelfSupplied || line.Demand != 1 || line.Source != nil {
			t.Errorf("line = %#v, want uncovered demand", line)
		}
	})
}

func TestFindSupplyLineBreaksEqualDistanceByTerritoryCode(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("T01", "ZZZ", models.TerrainPlain, "T02"),
			supplyTerritory("T02", "MID", models.TerrainPlain, "T01", "T03"),
			supplyTerritory("T03", "AAA", models.TerrainPlain, "T02"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T02", Size: 2}},
	)
	setTerritoryOwner(state, "T01", "P1")
	setTerritoryOwner(state, "T03", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T03"})
	validateTestState(t, state)

	line, err := FindSupplyLine(state, testBalance(), "T02")
	if err != nil {
		t.Fatalf("FindSupplyLine: %v", err)
	}
	if line.Source == nil || *line.Source != "T03" {
		t.Fatalf("source = %v, want T03 (AAA)", line.Source)
	}
	if want := []models.TerritoryID{"T03", "T02"}; !reflect.DeepEqual(line.Path, want) {
		t.Errorf("path = %v, want %v", line.Path, want)
	}
}

func TestFindSupplyLineRejectsUnavailableTargets(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("T01", "AAA", models.TerrainPlain),
			supplyTerritory("T02", "BBB", models.TerrainPlain),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
	)

	if _, err := FindSupplyLine(state, testBalance(), "T99"); !errors.Is(err, ErrSupplyLineUnknownTerritory) {
		t.Errorf("unknown territory error = %v, want ErrSupplyLineUnknownTerritory", err)
	}
	if _, err := FindSupplyLine(state, testBalance(), "T02"); !errors.Is(err, ErrSupplyLineNoArmy) {
		t.Errorf("empty territory error = %v, want ErrSupplyLineNoArmy", err)
	}
	state.Season = models.SeasonWinter
	if _, err := FindSupplyLine(state, testBalance(), "T01"); !errors.Is(err, ErrSupplyLineWinter) {
		t.Errorf("winter error = %v, want ErrSupplyLineWinter", err)
	}
}
