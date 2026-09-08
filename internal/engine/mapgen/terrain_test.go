package mapgen

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestTerrainSeedCount(t *testing.T) {
	tests := []struct {
		name  string
		sites int
		want  int
	}{
		{name: "minimum", sites: 8, want: 5},
		{name: "doubled density", sites: 32, want: 8},
		{name: "site ceiling", sites: 3, want: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := terrainSeedCount(test.sites); got != test.want {
				t.Fatalf("terrain seed count = %d, want %d", got, test.want)
			}
		})
	}
}

func TestTerrainSingletonAllowed(t *testing.T) {
	sites := []point{
		{x: 0, y: 0},
		{x: 10, y: 0},
		{x: 0, y: 10},
		{x: 10, y: 10},
		{x: 5, y: 5},
		{x: 20, y: 20},
	}

	terrain := assignTerrains(newRNG("singleton", "terrain"), sites)
	counts := make(map[models.Terrain]int, len(allTerrains))
	for _, value := range terrain {
		counts[value]++
	}
	singletons := 0
	for _, count := range counts {
		if count == 1 {
			singletons++
		}
	}
	if singletons == 0 {
		t.Fatal("terrain assignment produced no singleton region")
	}
}
