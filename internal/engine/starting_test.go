package engine

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestSelectStartingTerritoriesFindsSeparatedCandidates(t *testing.T) {
	adjacencies := cycleAdjacencies(8)
	candidates := make([]models.TerritoryID, 0, 7)
	for index := 2; index <= 8; index++ {
		candidates = append(candidates, models.TerritoryID(fmt.Sprintf("T%02d", index)))
	}

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
	if _, err := SelectStartingTerritories([]models.TerritoryID{"T01", "T02"}, cycleAdjacencies(6), 2); err == nil {
		t.Fatal("SelectStartingTerritories returned no error for an impossible distance")
	}
}

func cycleAdjacencies(count int) map[models.TerritoryID][]models.TerritoryID {
	adjacencies := make(map[models.TerritoryID][]models.TerritoryID, count)
	for index := 0; index < count; index++ {
		adjacencies[models.TerritoryID(fmt.Sprintf("T%02d", index+1))] = []models.TerritoryID{
			models.TerritoryID(fmt.Sprintf("T%02d", (index+count-1)%count+1)),
			models.TerritoryID(fmt.Sprintf("T%02d", (index+1)%count+1)),
		}
	}
	return adjacencies
}
