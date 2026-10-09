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
