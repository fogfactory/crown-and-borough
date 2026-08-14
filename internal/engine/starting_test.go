package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestSelectStartingTerritoriesFindsSeparatedCandidates(t *testing.T) {
	adjacencies := cycleAdjacencies(8)
	candidates := append([]models.TerritoryID(nil), cycleTerritoryIDs(8)[1:]...)

	first, err := SelectStartingTerritories(candidates, adjacencies, 2)
	if err != nil {
		t.Fatalf("SelectStartingTerritories: %v", err)
	}
	second, err := SelectStartingTerritories(candidates, adjacencies, 2)
	if err != nil {
		t.Fatalf("second SelectStartingTerritories: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("selection is not deterministic: first=%v second=%v", first, second)
	}
	if distance := testGraphDistance(adjacencies, first[0], first[1]); distance < MinimumStartingDistance {
		t.Fatalf("starting distance = %d, want at least %d", distance, MinimumStartingDistance)
	}
}

func TestSelectStartingTerritoriesRejectsImpossibleDistance(t *testing.T) {
	if _, err := SelectStartingTerritories([]models.TerritoryID{"AAA", "BBB"}, cycleAdjacencies(6), 2); err == nil {
		t.Fatal("SelectStartingTerritories returned no error for an impossible distance")
	}
}

func cycleAdjacencies(count int) map[models.TerritoryID][]models.TerritoryID {
	adjacencies := make(map[models.TerritoryID][]models.TerritoryID, count)
	ids := cycleTerritoryIDs(count)
	for index := 0; index < count; index++ {
		adjacencies[ids[index]] = []models.TerritoryID{
			ids[(index+count-1)%count],
			ids[(index+1)%count],
		}
	}
	return adjacencies
}

func cycleTerritoryIDs(count int) []models.TerritoryID {
	ids := []models.TerritoryID{"AAA", "BBB", "CCC", "DDD", "EEE", "FFF", "GGG", "HHH"}
	return ids[:count]
}
