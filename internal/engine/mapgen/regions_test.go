package mapgen

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestValidateRegions(t *testing.T) {
	territories := []Territory{
		{ID: "AAA", Adjacencies: []string{"BBB"}},
		{ID: "BBB", Adjacencies: []string{"AAA", "CCC"}},
		{ID: "CCC", Adjacencies: []string{"BBB", "DDD"}},
		{ID: "DDD", Adjacencies: []string{"CCC"}},
	}
	valid := []models.Region{
		{ID: "R-AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB"}},
		{ID: "R-CCC", Seed: "CCC", Territories: []models.TerritoryID{"CCC", "DDD"}},
	}
	if err := ValidateRegions(territories, valid); err != nil {
		t.Fatalf("valid regions: %v", err)
	}
	cases := []struct {
		name    string
		regions []models.Region
	}{
		{name: "missing territory", regions: []models.Region{{ID: "R-AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}}}},
		{name: "duplicate territory", regions: []models.Region{{ID: "R-AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB"}}, {ID: "R-CCC", Seed: "CCC", Territories: []models.TerritoryID{"BBB", "CCC", "DDD"}}}},
		{name: "unsorted territories", regions: []models.Region{{ID: "R-AAA", Seed: "AAA", Territories: []models.TerritoryID{"BBB", "AAA"}}, {ID: "R-CCC", Seed: "CCC", Territories: []models.TerritoryID{"CCC", "DDD"}}}},
		{name: "disconnected territories", regions: []models.Region{{ID: "R-AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "CCC"}}, {ID: "R-BBD", Seed: "BBB", Territories: []models.TerritoryID{"BBB", "DDD"}}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateRegions(territories, test.regions); err == nil {
				t.Fatal("ValidateRegions() = nil, want error")
			}
		})
	}
}
