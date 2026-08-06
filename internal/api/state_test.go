package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestProjectStateMatchesStateContract(t *testing.T) {
	state := projectTestState()
	freshness := map[models.TerritoryID]int{
		"T01": state.Turn,
		"T02": state.Turn - 2,
		"T03": state.Turn,
		"T04": state.Turn,
	}
	view := projectState(state, freshness)

	if view.Turn != state.Turn || view.Season != state.Season {
		t.Errorf("view metadata = %d/%s, want %d/%s", view.Turn, view.Season, state.Turn, state.Season)
	}
	if len(view.Territories) != len(state.Territories) {
		t.Fatalf("view territories = %d, want %d", len(view.Territories), len(state.Territories))
	}
	for index, territory := range state.Territories {
		if view.Territories[index].ID != territory.ID {
			t.Errorf("territory %d = %s, want %s", index, view.Territories[index].ID, territory.ID)
		}
	}
	if got := view.Territories[0]; got.Owner == nil || *got.Owner != "P1" || got.Resources != 3 {
		t.Errorf("T01 view = %+v, want P1 with 3 resources", got)
	}
	if got := view.Territories[0].Army; got == nil || !reflect.DeepEqual(*got, ArmyView{Owner: "P1", Size: 1}) {
		t.Errorf("T01 army = %#v, want owner P1 and size 1", got)
	}
	if view.Territories[2].Army != nil {
		t.Errorf("T03 army = %#v, want nil", view.Territories[2].Army)
	}
	if got := view.Territories[0].Infrastructures; !reflect.DeepEqual(got, []InfraView{{Type: models.InfraTypeCastle, Level: 1}}) {
		t.Errorf("T01 infrastructure = %#v, want nested castle", got)
	}
	if len(view.Nobles) != 1 || view.Nobles[0] != (NobleView{ID: "N1", Name: "Hugues de Rosemont", Owner: "P1", Location: "T01"}) {
		t.Errorf("nobles = %#v, want N1", view.Nobles)
	}

	data, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal StateView: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("unmarshal StateView document: %v", err)
	}
	assertStateJSONTypes(t, document)
	var decoded StateView
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal StateView: %v", err)
	}
	if !reflect.DeepEqual(decoded, view) {
		t.Errorf("StateView JSON round trip = %#v, want %#v", decoded, view)
	}
}

func TestProjectStateFreshnessAndNesting(t *testing.T) {
	state := projectTestState()
	view := projectState(state, map[models.TerritoryID]int{
		"T01": state.Turn,
		"T02": state.Turn - 2,
		"T03": state.Turn,
		"T04": state.Turn - 2,
	})

	if len(view.AsOf) != len(state.Territories) {
		t.Errorf("asOf coverage = %d, want %d", len(view.AsOf), len(state.Territories))
	}
	for _, territory := range state.Territories {
		observed := view.AsOf[territory.ID]
		if observed != state.Turn && observed != state.Turn-2 {
			t.Errorf("asOf[%s] = %d, want %d or %d", territory.ID, observed, state.Turn, state.Turn-2)
		}
	}

	viewByID := make(map[models.TerritoryID]TerritoryView, len(view.Territories))
	for _, territory := range view.Territories {
		viewByID[territory.ID] = territory
	}
	for _, army := range state.Armies {
		projected := viewByID[army.TerritoryID].Army
		if projected == nil || projected.Owner != army.OwnerID || projected.Size != army.Size {
			t.Errorf("army %s is not nested in %s as owner %s size %d", army.ID, army.TerritoryID, army.OwnerID, army.Size)
		}
	}
	for _, infrastructure := range state.Infrastructures {
		found := false
		for _, projected := range viewByID[infrastructure.TerritoryID].Infrastructures {
			if projected.Type == infrastructure.Type && projected.Level == infrastructure.Level {
				found = true
			}
		}
		if !found {
			t.Errorf("infrastructure %s is not nested in %s", infrastructure.ID, infrastructure.TerritoryID)
		}
	}
}

func TestStateHandler(t *testing.T) {
	want := []byte(`{"turn":5,"season":"spring","asOf":{},"territories":[],"nobles":[]}`)
	resolvedPlayers := 0
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/state", nil)

	StateHandler(func(players int) ([]byte, error) {
		resolvedPlayers = players
		return want, nil
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("GET /api/state = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if !bytes.Equal(recorder.Body.Bytes(), want) {
		t.Errorf("response = %q, want %q", recorder.Body.Bytes(), want)
	}
	if resolvedPlayers != DefaultPlayers {
		t.Errorf("resolver players = %d, want %d", resolvedPlayers, DefaultPlayers)
	}
}

func TestStateHandlerPlayers(t *testing.T) {
	for _, test := range []struct {
		name     string
		rawQuery string
		want     int
	}{
		{name: "default", want: 4},
		{name: "minimum", rawQuery: "players=2", want: 2},
		{name: "maximum", rawQuery: "players=5", want: 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolvedPlayers := 0
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/state", nil)
			request.URL.RawQuery = test.rawQuery

			StateHandler(func(players int) ([]byte, error) {
				resolvedPlayers = players
				return []byte(`{}`), nil
			}).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Errorf("GET /api/state?%s = %d, want %d", test.rawQuery, recorder.Code, http.StatusOK)
			}
			if resolvedPlayers != test.want {
				t.Errorf("resolver players = %d, want %d", resolvedPlayers, test.want)
			}
		})
	}
}

func TestStateHandlerErrors(t *testing.T) {
	for _, rawQuery := range []string{"players=1", "players=6", "players=abc", "players=", "players=2&players=3", "players=%zz"} {
		t.Run(rawQuery, func(t *testing.T) {
			called := false
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/state", nil)
			request.URL.RawQuery = rawQuery
			StateHandler(func(int) ([]byte, error) {
				called = true
				return nil, nil
			}).ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Errorf("GET /api/state?%s = %d, want %d", rawQuery, recorder.Code, http.StatusBadRequest)
			}
			if called {
				t.Errorf("resolver called for invalid query %q", rawQuery)
			}
		})
	}

	recorder := httptest.NewRecorder()
	StateHandler(func(int) ([]byte, error) {
		return nil, errors.New("state generation failed")
	}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/state resolver error = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestStateResolverCachesBytes(t *testing.T) {
	assets := loadStateTestAssets(t)
	mapData := generateStateTestMap(t, assets)
	calls := 0
	resolve := StateResolver(func(int) (mapgen.MapData, error) {
		calls++
		return mapData, nil
	}, "state-cache", assets)

	first, err := resolve(4)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	second, err := resolve(4)
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Error("state resolver returned unstable bytes")
	}
	if calls != 1 {
		t.Errorf("map data calls = %d, want 1", calls)
	}
}

func projectTestState() *models.GameState {
	p1 := models.PlayerID("P1")
	p2 := models.PlayerID("P2")
	state := &models.GameState{
		ID:     "state-view",
		Seed:   "state-view",
		Turn:   5,
		Season: models.SeasonSpring,
		Players: []models.Player{
			{ID: p1, Name: "Hugues", Color: "#a84632"},
			{ID: p2, Name: "Aliénor", Color: "#2d5f9e"},
		},
		Territories: []models.Territory{
			{ID: "T01", Code: "ROS", Name: "Rosemont", Terrain: models.TerrainPlain, Adjacencies: []models.TerritoryID{"T02", "T04"}},
			{ID: "T02", Code: "BOI", Name: "Boisclair", Terrain: models.TerrainForest, Adjacencies: []models.TerritoryID{"T01", "T03"}},
			{ID: "T03", Code: "BRU", Name: "Bruyères", Terrain: models.TerrainHill, Adjacencies: []models.TerritoryID{"T02", "T04"}},
			{ID: "T04", Code: "FOU", Name: "Fougères", Terrain: models.TerrainSwamp, Adjacencies: []models.TerritoryID{"T03", "T01"}},
		},
		Nobles: []models.Noble{
			{ID: "N1", Code: "HUG", Name: "Hugues de Rosemont", OwnerID: p1, LocationID: "T01"},
		},
		Armies: []models.Army{
			{ID: "A1", OwnerID: p1, TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: p2, TerritoryID: "T02", Size: 2},
		},
		Infrastructures: []models.Infrastructure{
			{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"},
			{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T04"},
		},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			"T01": {OwnerID: &p1, Resources: 3, Army: ptrArmyID("A1"), Infrastructures: []models.InfraID{"I1"}},
			"T02": {OwnerID: &p2, Resources: 0, Army: ptrArmyID("A2"), Infrastructures: []models.InfraID{}},
			"T03": {OwnerID: nil, Resources: 0, Infrastructures: []models.InfraID{}},
			"T04": {OwnerID: nil, Resources: 0, Infrastructures: []models.InfraID{"I2"}},
		},
	}
	if err := state.Validate(); err != nil {
		panic(err)
	}
	return state
}

func ptrArmyID(id string) *models.ArmyID {
	armyID := models.ArmyID(id)
	return &armyID
}

func assertStateJSONTypes(t *testing.T, document map[string]any) {
	t.Helper()
	if _, ok := document["turn"].(float64); !ok {
		t.Errorf("turn JSON type = %T, want number", document["turn"])
	}
	if _, ok := document["season"].(string); !ok {
		t.Errorf("season JSON type = %T, want string", document["season"])
	}
	if _, ok := document["asOf"].(map[string]any); !ok {
		t.Errorf("asOf JSON type = %T, want object", document["asOf"])
	}
	territories, ok := document["territories"].([]any)
	if !ok || len(territories) == 0 {
		t.Fatalf("territories JSON = %#v, want non-empty array", document["territories"])
	}
	territory, ok := territories[0].(map[string]any)
	if !ok {
		t.Fatalf("first territory JSON = %#v, want object", territories[0])
	}
	army, ok := territory["army"].(map[string]any)
	if !ok {
		t.Errorf("army JSON type = %T, want object", territory["army"])
	} else {
		if _, hasID := army["id"]; hasID {
			t.Error("army JSON must not expose its internal id")
		}
		if _, ok := army["owner"].(string); !ok {
			t.Errorf("army owner JSON type = %T, want string", army["owner"])
		}
		if _, ok := army["size"].(float64); !ok {
			t.Errorf("army size JSON type = %T, want number", army["size"])
		}
	}
	if _, ok := territory["infrastructures"].([]any); !ok {
		t.Errorf("infrastructures JSON type = %T, want array", territory["infrastructures"])
	}
	for _, rawTerritory := range territories {
		candidate, ok := rawTerritory.(map[string]any)
		if !ok || candidate["id"] != "T03" {
			continue
		}
		if candidate["army"] != nil {
			t.Errorf("T03 army JSON = %#v, want null", candidate["army"])
		}
	}
	nobles, ok := document["nobles"].([]any)
	if !ok || len(nobles) != 1 {
		t.Errorf("nobles JSON = %#v, want one-item array", document["nobles"])
	}
}

func loadStateTestAssets(t *testing.T) assetgen.Assets {
	t.Helper()
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	return assets
}

func generateStateTestMap(t *testing.T, assets assetgen.Assets) mapgen.MapData {
	t.Helper()
	mapData, err := mapgen.Generate("state-test-map", assets, mapgen.Config{
		Width:        1000,
		Height:       700,
		SiteCount:    mapgen.TerritoriesPerPlayer * 4,
		VillageCount: 5,
	})
	if err != nil {
		t.Fatalf("generate map: %v", err)
	}
	return mapData
}
