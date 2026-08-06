package models_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func ptrID(id string) *models.PlayerID {
	p := models.PlayerID(id)
	return &p
}

func ptrArmyID(id string) *models.ArmyID {
	a := models.ArmyID(id)
	return &a
}

// validState returns a complete, valid state: 2 players, 4 territories in a
// ring with commune trigrams, 2 armies, 1 noble, 1 mill level 2 and 1 castle,
// 1 neutral territory, resources. Tests mutate a fresh instance to build
// negative cases.
func validState() *models.GameState {
	g := models.NewGameState()
	g.ID = "game-1"
	g.Seed = "seed-1"
	g.Players = []models.Player{
		{ID: "P1", Name: "Alpha", Color: "red"},
		{ID: "P2", Name: "Beta", Color: "blue"},
	}
	g.Territories = []models.Territory{
		{ID: "T01", Code: "ROS", Name: "Rosemont", Terrain: models.TerrainPlain, Adjacencies: []models.TerritoryID{"T02", "T04"}},
		{ID: "T02", Code: "BCL", Name: "Boisclair", Terrain: models.TerrainForest, Adjacencies: []models.TerritoryID{"T01", "T03"}},
		{ID: "T03", Code: "BRU", Name: "Bruyères", Terrain: models.TerrainHill, Adjacencies: []models.TerritoryID{"T02", "T04"}},
		{ID: "T04", Code: "FOU", Name: "Fougères", Terrain: models.TerrainSwamp, Adjacencies: []models.TerritoryID{"T03", "T01"}},
	}
	g.Armies = []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
		{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 2},
	}
	g.Nobles = []models.Noble{
		{ID: "N1", Code: "HUG", Name: "Hugues", OwnerID: "P1", LocationID: "T01"},
	}
	g.Infrastructures = []models.Infrastructure{
		{ID: "I1", Type: models.InfraTypeMill, Level: 2, TerritoryID: "T01"},
		{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T03"},
	}
	g.TerritoryStates = map[models.TerritoryID]models.TerritoryState{
		"T01": {OwnerID: ptrID("P1"), Resources: 5, Army: ptrArmyID("A1"), Infrastructures: []models.InfraID{"I1"}},
		"T02": {OwnerID: ptrID("P2"), Resources: 0, Army: ptrArmyID("A2")},
		"T03": {OwnerID: ptrID("P1"), Resources: 1, Infrastructures: []models.InfraID{"I2"}},
		"T04": {OwnerID: nil, Resources: 0},
	}
	return g
}

func TestTerrainIsValid(t *testing.T) {
	for _, valid := range []models.Terrain{
		models.TerrainPlain, models.TerrainForest, models.TerrainHill,
		models.TerrainMountain, models.TerrainSwamp,
	} {
		if !valid.IsValid() {
			t.Errorf("Terrain %q: want valid", valid)
		}
	}
	for _, invalid := range []models.Terrain{"", "MARSH", "any", "forest ", "plain2"} {
		if invalid.IsValid() {
			t.Errorf("Terrain %q: want invalid", invalid)
		}
	}
}

func TestSeasonIsValid(t *testing.T) {
	for _, valid := range []models.Season{
		models.SeasonSpring, models.SeasonSummer, models.SeasonAutumn, models.SeasonWinter,
	} {
		if !valid.IsValid() {
			t.Errorf("Season %q: want valid", valid)
		}
	}
	for _, invalid := range []models.Season{"", "snow", "SPRING", "spring "} {
		if invalid.IsValid() {
			t.Errorf("Season %q: want invalid", invalid)
		}
	}
}

func TestInfraTypeIsValid(t *testing.T) {
	for _, valid := range []models.InfraType{
		models.InfraTypeMill, models.InfraTypePostRelay, models.InfraTypeWatchtower,
		models.InfraTypeSupplyDepot, models.InfraTypeCastle, models.InfraTypeVillage,
	} {
		if !valid.IsValid() {
			t.Errorf("InfraType %q: want valid", valid)
		}
	}
	for _, invalid := range []models.InfraType{"", "bank", "mill ", "Mill", "castle\t"} {
		if invalid.IsValid() {
			t.Errorf("InfraType %q: want invalid", invalid)
		}
	}
}

func TestSeasonForTurn(t *testing.T) {
	cases := []struct {
		turn int
		want models.Season
	}{
		{1, models.SeasonSpring},
		{2, models.SeasonSummer},
		{3, models.SeasonAutumn},
		{4, models.SeasonWinter},
		{5, models.SeasonSpring},
		{8, models.SeasonWinter},
		{9, models.SeasonSpring},
	}
	for _, tc := range cases {
		if got := models.SeasonForTurn(tc.turn); got != tc.want {
			t.Errorf("SeasonForTurn(%d) = %q, want %q", tc.turn, got, tc.want)
		}
	}
}

func TestYear(t *testing.T) {
	cases := []struct {
		turn int
		want int
	}{
		{1, 1}, {4, 1}, {5, 2}, {8, 2}, {9, 3},
	}
	for _, tc := range cases {
		g := models.NewGameState()
		g.Turn = tc.turn
		if got := g.Year(); got != tc.want {
			t.Errorf("Year() with turn %d = %d, want %d", tc.turn, got, tc.want)
		}
	}
}

func TestNewGameState(t *testing.T) {
	g := models.NewGameState()
	if g.Turn != 1 {
		t.Errorf("Turn = %d, want 1", g.Turn)
	}
	if g.Season != models.SeasonSpring {
		t.Errorf("Season = %q, want %q", g.Season, models.SeasonSpring)
	}
	if err := g.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidateValidState(t *testing.T) {
	if err := validState().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidateErrors(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(g *models.GameState)
		want   string
	}{
		{"turn zero", func(g *models.GameState) { g.Turn = 0 }, "turn"},
		{"invalid season", func(g *models.GameState) { g.Season = "snow" }, "invalid season"},
		{"season mismatch with turn", func(g *models.GameState) { g.Turn = 2 }, "season"},
		{"duplicate player id", func(g *models.GameState) {
			g.Players = append(g.Players, models.Player{ID: "P1", Name: "Alpha2", Color: "green"})
		}, "duplicate id"},
		{"duplicate territory id", func(g *models.GameState) {
			g.Territories = append(g.Territories, models.Territory{ID: "T01", Code: "XRO", Name: "X", Terrain: models.TerrainPlain})
		}, "duplicate id"},
		{"duplicate territory code", func(g *models.GameState) {
			g.Territories = append(g.Territories, models.Territory{ID: "T05", Code: "BCL", Name: "X", Terrain: models.TerrainPlain})
		}, "duplicate code"},
		{"invalid terrain", func(g *models.GameState) { g.Territories[1].Terrain = "MARSH" }, "invalid terrain"},
		{"asymmetric adjacency", func(g *models.GameState) { g.Territories[0].Adjacencies = []models.TerritoryID{"T02"} }, "asymmetric"},
		{"adjacency to unknown territory", func(g *models.GameState) {
			g.Territories[0].Adjacencies = []models.TerritoryID{"T02", "T99"}
		}, "does not exist"},
		{"self adjacency", func(g *models.GameState) {
			g.Territories[0].Adjacencies = []models.TerritoryID{"T02", "T01"}
		}, "self-adjacency"},
		{"duplicate adjacency", func(g *models.GameState) {
			g.Territories[0].Adjacencies = []models.TerritoryID{"T02", "T02"}
		}, "duplicate adjacency"},
		{"duplicate army id", func(g *models.GameState) {
			g.Armies = append(g.Armies, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "T02", Size: 1})
		}, "duplicate id"},
		{"army size zero", func(g *models.GameState) { g.Armies[0].Size = 0 }, "size"},
		{"army unknown owner", func(g *models.GameState) { g.Armies[0].OwnerID = "P9" }, "unknown owner"},
		{"army unknown territory", func(g *models.GameState) { g.Armies[0].TerritoryID = "T99" }, "unknown territory"},
		{"army not referenced by territory state", func(g *models.GameState) {
			g.TerritoryStates["T01"] = models.TerritoryState{OwnerID: ptrID("P1"), Resources: 5, Infrastructures: []models.InfraID{"I1"}}
		}, "does not reference it"},
		{"state references army stationed elsewhere", func(g *models.GameState) {
			g.TerritoryStates["T02"] = models.TerritoryState{OwnerID: ptrID("P2"), Resources: 0, Army: ptrArmyID("A1")}
		}, "stationed in"},
		{"state references unknown army", func(g *models.GameState) {
			g.TerritoryStates["T02"] = models.TerritoryState{OwnerID: ptrID("P2"), Resources: 0, Army: ptrArmyID("A9")}
		}, "unknown army"},
		{"duplicate noble id", func(g *models.GameState) {
			g.Nobles = append(g.Nobles, models.Noble{ID: "N1", Code: "ANN", Name: "Anne", OwnerID: "P2", LocationID: "T02"})
		}, "duplicate id"},
		{"duplicate noble code", func(g *models.GameState) {
			g.Nobles = append(g.Nobles, models.Noble{ID: "N2", Code: "HUG", Name: "Hugues II", OwnerID: "P2", LocationID: "T02"})
		}, "duplicate code"},
		{"noble unknown owner", func(g *models.GameState) { g.Nobles[0].OwnerID = "P9" }, "unknown owner"},
		{"noble unknown territory", func(g *models.GameState) { g.Nobles[0].LocationID = "T99" }, "unknown territory"},
		{"duplicate infrastructure id", func(g *models.GameState) {
			g.Infrastructures = append(g.Infrastructures, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "T02"})
		}, "duplicate id"},
		{"invalid infra type", func(g *models.GameState) { g.Infrastructures[0].Type = "bank" }, "invalid type"},
		{"infra level zero", func(g *models.GameState) { g.Infrastructures[0].Level = 0 }, "level"},
		{"infra unknown territory", func(g *models.GameState) { g.Infrastructures[0].TerritoryID = "T99" }, "unknown territory"},
		{"infra not listed in territory state", func(g *models.GameState) {
			g.TerritoryStates["T01"] = models.TerritoryState{OwnerID: ptrID("P1"), Resources: 5, Army: ptrArmyID("A1")}
		}, "does not list it"},
		{"state lists infra built elsewhere", func(g *models.GameState) {
			g.TerritoryStates["T02"] = models.TerritoryState{OwnerID: ptrID("P2"), Resources: 0, Army: ptrArmyID("A2"), Infrastructures: []models.InfraID{"I2"}}
		}, "built in"},
		{"state lists unknown infra", func(g *models.GameState) {
			g.TerritoryStates["T02"] = models.TerritoryState{OwnerID: ptrID("P2"), Resources: 0, Army: ptrArmyID("A2"), Infrastructures: []models.InfraID{"I9"}}
		}, "unknown infrastructure"},
		{"multiple infrastructures in territory state", func(g *models.GameState) {
			g.TerritoryStates["T01"] = models.TerritoryState{OwnerID: ptrID("P1"), Resources: 5, Army: ptrArmyID("A1"), Infrastructures: []models.InfraID{"I1", "I2"}}
		}, "multiple infrastructures"},
		{"missing territory state entry", func(g *models.GameState) {
			delete(g.TerritoryStates, "T04")
		}, "missing TerritoryState"},
		{"orphan territory state entry", func(g *models.GameState) {
			g.TerritoryStates["T99"] = models.TerritoryState{}
		}, "references unknown territory"},
		{"state owner unknown", func(g *models.GameState) {
			g.TerritoryStates["T04"] = models.TerritoryState{OwnerID: ptrID("P9")}
		}, "unknown owner"},
		{"negative resources", func(g *models.GameState) {
			g.TerritoryStates["T01"] = models.TerritoryState{OwnerID: ptrID("P1"), Resources: -1, Army: ptrArmyID("A1"), Infrastructures: []models.InfraID{"I1"}}
		}, "negative resources"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := validState()
			tc.mutate(g)
			err := g.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error %q does not contain %q", err, tc.want)
			}
		})
	}
}

// TestCodeInvariants pins the trigram requirement for territories and nobles.
func TestCodeInvariants(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(g *models.GameState)
		want   string
	}{
		{"territory with 2-letter code", func(g *models.GameState) { g.Territories[0].Code = "RO" }, "3 uppercase"},
		{"territory with 4-letter code", func(g *models.GameState) { g.Territories[1].Code = "FROS" }, "3 uppercase"},
		{"territory with lowercase code", func(g *models.GameState) { g.Territories[0].Code = "ros" }, "3 uppercase"},
		{"territory code with digit", func(g *models.GameState) { g.Territories[1].Code = "FR0" }, "3 uppercase"},
		{"noble with 4-letter code", func(g *models.GameState) { g.Nobles[0].Code = "HUGU" }, "3 uppercase"},
		{"noble with lowercase code", func(g *models.GameState) { g.Nobles[0].Code = "hug" }, "3 uppercase"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := validState()
			tc.mutate(g)
			err := g.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error %q does not contain %q", err, tc.want)
			}
		})
	}
}

func TestGameStateJSONRoundTrip(t *testing.T) {
	g := validState()
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if bytes.Contains(data, []byte(`"lieuDit"`)) {
		t.Error("JSON still contains the removed lieuDit flag")
	}
	infraJSON, err := json.Marshal(g.Infrastructures[0])
	if err != nil {
		t.Fatalf("Marshal infrastructure: %v", err)
	}
	if bytes.Contains(infraJSON, []byte(`"owner"`)) {
		t.Errorf("infrastructure JSON %s still contains an owner field", infraJSON)
	}
	if !bytes.Contains(data, []byte(`"armies"`)) || !bytes.Contains(data, []byte(`"army":"A1"`)) || !bytes.Contains(data, []byte(`"army":null`)) {
		t.Errorf("GameState JSON %s does not contain the army list and nullable territory army pointers", data)
	}
	var got models.GameState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(*g, got) {
		t.Fatalf("round-trip mismatch:\nwant %+v\ngot  %+v", *g, got)
	}
	if err := got.Validate(); err != nil {
		t.Errorf("Validate() of round-tripped state = %v, want nil", err)
	}
}

func TestNewGameStateJSONEmptyCollections(t *testing.T) {
	data, err := json.Marshal(models.NewGameState())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{
		`"players":[]`, `"territories":[]`, `"nobles":[]`, `"armies":[]`,
		`"infrastructures":[]`, `"territoryStates":{}`,
	} {
		if !bytes.Contains(data, []byte(key)) {
			t.Errorf("JSON %s does not contain %s", data, key)
		}
	}
}
