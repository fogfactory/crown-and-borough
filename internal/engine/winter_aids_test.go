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
	if want := []FiefEdge{{From: "AAA", To: "BBB"}, {From: "BBB", To: "CCC"}}; !reflect.DeepEqual(site.Edges, want) {
		t.Errorf("edges = %v, want %v", site.Edges, want)
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

func TestWinterAidsOwnFiefs(t *testing.T) {
	state := annexTestState(t)
	// DDD (held by P1) is annexable; EEE is not adjacent; FFF is held by P2.
	placeArmyAt(state, "AX", "P2", "FFF", 3)
	aids := ForecastWinterAids(state, testBalance(), "P1")
	if len(aids.OwnFiefs) != 1 {
		t.Fatalf("own fiefs = %+v, want F1 only", aids.OwnFiefs)
	}
	fief := aids.OwnFiefs[0]
	if fief.Capital != "AAA" || fief.Holder != "HUG" || fief.Title != models.FiefTitleBarony {
		t.Errorf("fief = %+v", fief)
	}
	if want := []models.TerritoryID{"DDD"}; !reflect.DeepEqual(fief.Annexable, want) {
		t.Errorf("annexable = %v, want %v", fief.Annexable, want)
	}
	if fief.NextTitle != models.FiefTitleCounty || fief.RankCost != 3*testBalance().Costs.RankPerTerritory {
		t.Errorf("next title/cost = %q/%d", fief.NextTitle, fief.RankCost)
	}
	if aids.AnnexCostPerTerritory != testBalance().Costs.AnnexPerTerritory || aids.RankCostPerTerritory != testBalance().Costs.RankPerTerritory {
		t.Errorf("per-territory costs = %d/%d", aids.AnnexCostPerTerritory, aids.RankCostPerTerritory)
	}
	state.Fiefs[0].Title = models.FiefTitleDuchy
	if got := ForecastWinterAids(state, testBalance(), "P1").OwnFiefs[0].NextTitle; got != "" {
		t.Errorf("duchy next title = %q, want none", got)
	}
}

func TestWinterAidsRoyalPowersBelongToThePlayerOfTheKing(t *testing.T) {
	state := royalOrdersState(t)
	aids := ForecastWinterAids(state, testBalance(), "P1")
	if aids.Royal == nil || aids.Royal.Issuer != "HUG" || aids.Royal.OrdersPerWinter != testBalance().Royal.OrdersPerWinter {
		t.Fatalf("royal aid of the king's player = %+v, want issuer HUG", aids.Royal)
	}
	if other := ForecastWinterAids(state, testBalance(), "P2"); other.Royal != nil {
		t.Errorf("royal aid of another player = %+v, want none", other.Royal)
	}
	state.King = nil
	if vacant := ForecastWinterAids(state, testBalance(), "P1"); vacant.Royal != nil {
		t.Errorf("royal aid of a vacant throne = %+v, want none", vacant.Royal)
	}
}
