package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestWinterAidsFiefSiteIsAConnectedGroupWithACastle(t *testing.T) {
	state := foundFiefTestState(t)
	// DDD is held by another player's army: it splits the line in two.
	placeArmyAt(state, "AX", "P2", "DDD", 3)
	aids := ForecastWinterAids(state, testBalance(), "P1")
	if len(aids.FiefSites) != 1 {
		t.Fatalf("sites = %+v, want the single AAA-CCC site", aids.FiefSites)
	}
	site := aids.FiefSites[0]
	if want := []models.TerritoryID{"AAA", "BBB", "CCC"}; !reflect.DeepEqual(site.Territories, want) {
		t.Errorf("territories = %v, want %v", site.Territories, want)
	}
	if want := []models.TerritoryID{"AAA"}; !reflect.DeepEqual(site.Castles, want) {
		t.Errorf("castles = %v, want %v", site.Castles, want)
	}
	if len(site.Edges) != 2 {
		t.Errorf("edges = %v, want AAA-BBB and BBB-CCC", site.Edges)
	}
	if aids.FiefCostPerTerritory != testBalance().Costs.FiefPerTerritory {
		t.Errorf("cost per territory = %d", aids.FiefCostPerTerritory)
	}
}

func TestForecastRevoltTargetsFollowFamineTaxAndSeason(t *testing.T) {
	state := foundFiefTestState(t)
	state.Season = models.SeasonSpring
	state.Turn = 1
	state.Regions = []models.Region{
		{ID: "R1", Name: "One", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB"}},
		{ID: "R2", Name: "Two", Seed: "CCC", Territories: []models.TerritoryID{"CCC", "DDD", "EEE", "FFF", "GGG"}},
	}
	if got := ForecastRevoltTargets(state, testBalance()); len(got) != 0 {
		t.Fatalf("targets = %v, want none without famine, tax or trial", got)
	}
	// A famine announced for the summer is not active this spring; an active one is.
	year := state.Year()
	state.Auguries = map[int]models.YearAugury{year: {Year: year, Calamities: []models.Calamity{
		{Kind: models.CardKindFamine, Year: year, Season: models.SeasonSummer, RegionSeed: "CCC"},
		{Kind: models.CardKindFamine, Year: year, Season: models.SeasonSpring, RegionSeed: "AAA"},
	}}}
	got := ForecastRevoltTargets(state, testBalance())
	if len(got) != 2 || got[0] != "AAA" || got[1] != "BBB" {
		t.Errorf("targets = %v, want the territories of the region in famine now (AAA, BBB)", got)
	}
	state.Season = models.SeasonWinter
	if got := ForecastRevoltTargets(state, testBalance()); got != nil {
		t.Errorf("targets in winter = %v, want nil", got)
	}
}
