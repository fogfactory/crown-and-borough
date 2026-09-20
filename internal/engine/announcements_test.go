package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestSeedRegionEffectsForecastsCurrentSeason(t *testing.T) {
	state := models.NewGameState()
	state.Turn = 5
	state.Season = models.SeasonSpring
	state.Regions = []models.Region{
		{ID: "ROS", Seed: "ROS", Territories: []models.TerritoryID{"ROS"}},
		{ID: "BOI", Seed: "BOI", Territories: []models.TerritoryID{"BOI"}},
	}
	state.ActiveRegionEffects = []models.ActiveRegionEffect{
		{Kind: models.CardKindRevolt, RegionSeed: "BOI", Season: models.SeasonWinter, Year: 1},
	}
	state.Auguries[2] = models.YearAugury{Year: 2, Calamities: []models.Calamity{
		{CardID: "C1", Kind: models.CardKindPlague, Year: 2, Season: models.SeasonSpring, RegionSeed: "ROS"},
		{CardID: "C2", Kind: models.CardKindBadWeather, Year: 2, Season: models.SeasonSummer, RegionSeed: "BOI"},
		{CardID: "C3", Kind: models.CardKindBadWeather, Year: 2, Season: models.SeasonSpring, RegionSeed: "BOI"},
	}}

	SeedRegionEffects(state)

	if len(state.ActiveRegionEffects) != 2 {
		t.Fatalf("effects = %#v, want the two spring calamities only", state.ActiveRegionEffects)
	}
	first := state.ActiveRegionEffects[0]
	if first.RegionSeed != "BOI" || first.Kind != models.CardKindBadWeather || first.Season != models.SeasonSpring || first.Year != 2 {
		t.Fatalf("first effect = %#v, want spring bad weather in BOI at year 2", first)
	}
	second := state.ActiveRegionEffects[1]
	if second.RegionSeed != "ROS" || second.Kind != models.CardKindPlague {
		t.Fatalf("second effect = %#v, want spring plague in ROS", second)
	}

	state.Season = models.SeasonSummer
	SeedRegionEffects(state)
	if len(state.ActiveRegionEffects) != 1 || state.ActiveRegionEffects[0].RegionSeed != "BOI" || state.ActiveRegionEffects[0].Season != models.SeasonSummer {
		t.Fatalf("summer effects = %#v, want summer bad weather in BOI", state.ActiveRegionEffects)
	}
}

func TestSeedRegionEffectsClearsWithoutCalamities(t *testing.T) {
	state := models.NewGameState()
	state.Turn = 1
	state.Season = models.SeasonSpring
	state.ActiveRegionEffects = []models.ActiveRegionEffect{
		{Kind: models.CardKindPlague, RegionSeed: "ROS", Season: models.SeasonSpring, Year: 1},
	}

	SeedRegionEffects(state)

	if len(state.ActiveRegionEffects) != 0 {
		t.Fatalf("effects = %#v, want empty without scheduled calamities", state.ActiveRegionEffects)
	}
}
