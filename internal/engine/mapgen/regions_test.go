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

func TestGenerateRegionsUsesStableTieBreakAndSortsOutput(t *testing.T) {
	territories := []Territory{
		{ID: "ZZZ", Adjacencies: []string{"AAA", "BBB"}},
		{ID: "AAA", Village: true, Adjacencies: []string{"ZZZ"}},
		{ID: "BBB", Village: true, Adjacencies: []string{"ZZZ"}},
	}
	regions, err := generateRegions(territories)
	if err != nil {
		t.Fatalf("generateRegions = %v", err)
	}
	if len(regions) != 2 || regions[0].Seed != "AAA" || regions[1].Seed != "BBB" {
		t.Fatalf("regions = %#v, want AAA then BBB", regions)
	}
	if got := regions[0].Territories; len(got) != 2 || got[0] != "AAA" || got[1] != "ZZZ" {
		t.Fatalf("AAA territories = %#v, want [AAA ZZZ]", got)
	}
	if got := regions[1].Territories; len(got) != 1 || got[0] != "BBB" {
		t.Fatalf("BBB territories = %#v, want [BBB]", got)
	}
}

func TestGenerateRegionsRequiresConnectedCoverage(t *testing.T) {
	territories := []Territory{
		{ID: "AAA", Village: true},
		{ID: "BBB"},
	}
	if _, err := generateRegions(territories); err == nil {
		t.Fatal("generateRegions = nil error, want disconnected coverage error")
	}
}
