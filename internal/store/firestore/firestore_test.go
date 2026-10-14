package firestorestore

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestJSONMapRoundTripPreservesCanonicalState(t *testing.T) {
	assets, err := loadFirestoreTestAssets()
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance(testFirestoreAssetDir())
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	state, err := engine.CreateGame("codec-state", []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}}, balance, assets)
	if err != nil {
		t.Fatalf("create state: %v", err)
	}
	mapData, err := mapgen.Generate("codec-state", assets, engine.GameMapConfig(2))
	if err != nil {
		t.Fatalf("generate map: %v", err)
	}
	stateMap, err := jsonMap(state)
	if err != nil {
		t.Fatalf("encode state map: %v", err)
	}
	// The map is stored as a JSON string: its polygons are nested arrays.
	mapJSON, err := jsonString(mapData)
	if err != nil {
		t.Fatalf("encode map json: %v", err)
	}
	var decodedState models.GameState
	if err := decodeJSONMap(stateMap, &decodedState); err != nil {
		t.Fatalf("decode state map: %v", err)
	}
	var decodedMap mapgen.MapData
	if err := decodeJSONString(mapJSON, &decodedMap); err != nil {
		t.Fatalf("decode map json: %v", err)
	}
	if !reflect.DeepEqual(state, &decodedState) {
		t.Fatalf("state round trip changed value")
	}
	// Starts is deliberately excluded from map.json (json:"-"): it is engine
	// wiring, not published map data, so the round trip loses it by design.
	mapData.Starts = nil
	if !reflect.DeepEqual(mapData, decodedMap) {
		t.Fatalf("map round trip changed value")
	}
}

func TestJSONMapRejectsNestedArrays(t *testing.T) {
	_, err := jsonMap(map[string]interface{}{"sites": []interface{}{map[string]interface{}{"edges": [][2]string{{"AAA", "BBB"}}}}})
	if err == nil || !strings.Contains(err.Error(), "sites[0].edges[0]") {
		t.Fatalf("err = %v, want the nested array at sites[0].edges[0]", err)
	}
}

func TestWinterViewsEncodeWithoutNestedArrays(t *testing.T) {
	assets, err := loadFirestoreTestAssets()
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance(testFirestoreAssetDir())
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	state, err := engine.CreateGame("winter-view", []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}, {Name: "Carol"}, {Name: "Dave"}}, balance, assets)
	if err != nil {
		t.Fatalf("create state: %v", err)
	}
	state.Season = models.SeasonWinter
	players := make([]playerDocument, len(state.Players))
	for index, player := range state.Players {
		players[index] = playerDocument{ID: player.ID, Name: player.Name, ActorID: "uid-" + string(player.ID)}
	}
	// The documents the store writes when a turn enters winter, built by the
	// store itself: the player views carry the winter aids.
	s := NewWithClient(balance, assets, Options{})
	views, err := s.viewProjections("winter-view", players, 1, state, time.Now())
	if err != nil {
		t.Fatalf("encode winter views: %v", err)
	}
	if _, err := s.observerDocument("winter-view", "host", 1, state, time.Now(), 0); err != nil {
		t.Fatalf("encode observer winter view: %v", err)
	}
	// A starting house holds a connected castle group: its fief site carries
	// the edges that once reached Firestore as nested arrays.
	var view api.StateView
	if err := decodeJSONMap(views[0].document.State, &view); err != nil {
		t.Fatalf("decode winter view: %v", err)
	}
	if view.WinterAids == nil || len(view.WinterAids.FiefSites) == 0 || len(view.WinterAids.FiefSites[0].Edges) == 0 {
		t.Fatalf("winter aids = %+v, want a fief site with edges", view.WinterAids)
	}
}

// Fields that only fill in rare game situations escape the encoding tests
// above: every type the store encodes through jsonMap is checked for an array
// held directly by another array, whatever its content.
func TestPersistedTypesHoldNoNestedArrays(t *testing.T) {
	for name, value := range map[string]interface{}{
		"GameState":      models.GameState{},
		"PrivacyMeta":    models.PrivacyMeta{},
		"StateView":      api.StateView{},
		"TurnReport":     engine.TurnReport{},
		"TurnReportView": api.TurnReportView{},
	} {
		if path := nestedArrayType(reflect.TypeOf(value), name, false, map[reflect.Type]bool{}); path != "" {
			t.Errorf("%s holds a nested array", path)
		}
	}
}

func nestedArrayType(typ reflect.Type, path string, inArray bool, visiting map[reflect.Type]bool) string {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Slice, reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			return "" // []byte encodes as a string
		}
		if inArray {
			return path
		}
		return nestedArrayType(typ.Elem(), path+"[]", true, visiting)
	case reflect.Map:
		return nestedArrayType(typ.Elem(), path+"{}", false, visiting)
	case reflect.Struct:
		// A struct encodes as an object, even inside an array.
		if visiting[typ] {
			return ""
		}
		visiting[typ] = true
		defer delete(visiting, typ)
		for index := 0; index < typ.NumField(); index++ {
			field := typ.Field(index)
			if !field.IsExported() || field.Tag.Get("json") == "-" {
				continue
			}
			if found := nestedArrayType(field.Type, path+"."+field.Name, false, visiting); found != "" {
				return found
			}
		}
	}
	return ""
}

func TestFirestoreSchemaVersionIsExplicit(t *testing.T) {
	if schemaVersion < 1 {
		t.Fatalf("schema version = %d, want a positive version", schemaVersion)
	}
	document := profileDocument{SchemaVersion: schemaVersion, UID: "alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if document.SchemaVersion != 1 {
		t.Fatalf("profile schema version = %d, want 1", document.SchemaVersion)
	}
}

func TestGameDocumentCarriesDurationAndScores(t *testing.T) {
	state := models.NewGameState()
	state.Players = []models.Player{{ID: "P1", Name: "Alice"}}
	scores := map[models.PlayerID]engine.ScoreBreakdown{
		"P1": {Titles: 1, Total: 1},
	}
	snapshot := store.GameSnapshot{
		ID:        "game-1",
		Name:      "Test",
		Seed:      "seed",
		YearCount: state.YearCount,
		Scores:    scores,
		State:     state,
	}
	document := gameDocumentFromSnapshot(snapshot, time.Now(), time.Now())
	if document.YearCount != models.DefaultGameYears {
		t.Fatalf("document year count = %d, want %d", document.YearCount, models.DefaultGameYears)
	}
	if document.Scores == nil || len(document.Scores) != 1 || document.Scores["P1"].Total != 1 {
		t.Fatalf("document scores = %#v, want P1 score", document.Scores)
	}
	restored := gameSnapshot(assetgen.Balance{}, document, state, mapgen.MapData{}, nil, nil)
	if restored.Scores["P1"].Total != 1 {
		t.Fatalf("restored scores = %#v, want P1 score", restored.Scores)
	}
}

func TestGameDocumentCarriesSpectatorWithoutChangingSchemaVersion(t *testing.T) {
	state := models.NewGameState()
	state.Players = []models.Player{
		{ID: "P1", Name: "Bot A"},
		{ID: "P2", Name: "Bot B"},
	}
	snapshot := store.GameSnapshot{
		ID:           "game-spectator",
		Name:         "Observed",
		CreatedBy:    "host",
		SpectatorUID: "host",
		Players: []store.PlayerSlot{
			{ID: "P1", Name: "Bot A"},
			{ID: "P2", Name: "Bot B"},
		},
		State: state,
	}
	document := gameDocumentFromSnapshot(snapshot, time.Now(), time.Now())
	if document.SchemaVersion != schemaVersion {
		t.Fatalf("spectator schema version = %d, want %d", document.SchemaVersion, schemaVersion)
	}
	if document.SpectatorUID != "host" {
		t.Fatalf("spectator UID = %q, want host", document.SpectatorUID)
	}
	if !reflect.DeepEqual(document.MemberUIDs, []string{"host"}) {
		t.Fatalf("member UIDs = %#v, want host only", document.MemberUIDs)
	}
	restored := gameSnapshot(assetgen.Balance{}, document, state, mapgen.MapData{}, nil, nil)
	if restored.SpectatorUID != "host" {
		t.Fatalf("restored spectator UID = %q, want host", restored.SpectatorUID)
	}
}

func TestFirestoreStoreDefaultsAreSafeWithoutAClient(t *testing.T) {
	adapter := NewWithClient(assetgen.Balance{}, assetgen.Assets{}, Options{})
	if adapter.privacyTracker == nil {
		t.Fatal("privacy tracker is nil")
	}
	if adapter.leaseTimeout != DefaultLeaseTimeout || adapter.maximumReports != DefaultMaximumReports {
		t.Fatalf("defaults = lease %s reports %d", adapter.leaseTimeout, adapter.maximumReports)
	}
	if err := adapter.requireClient(); err != ErrNilClient {
		t.Fatalf("requireClient() = %v, want %v", err, ErrNilClient)
	}
	if err := adapter.Ready(nil); err != ErrNilClient {
		t.Fatalf("Ready without client = %v, want %v", err, ErrNilClient)
	}
	if _, err := adapter.GetProfile(nil, "alice"); err != ErrNilClient {
		t.Fatalf("GetProfile without client = %v, want %v", err, ErrNilClient)
	}
}

func loadFirestoreTestAssets() (assetgen.Assets, error) {
	return assetgen.Load(testFirestoreAssetDir())
}

func testFirestoreAssetDir() string {
	_, source, _, _ := runtime.Caller(0)
	return source[:len(source)-len("firestore_test.go")] + "../../../assets"
}

func TestProgressFromUIDsMapsSubmittedMembersToPlayers(t *testing.T) {
	state := &models.GameState{
		Season:  models.SeasonSpring,
		Players: []models.Player{{ID: "P1"}, {ID: "P2"}},
		Armies: []models.Army{
			{ID: "A1", OwnerID: "P1", Size: 1},
			{ID: "A2", OwnerID: "P2", Size: 1},
		},
		Nobles: []models.Noble{
			{ID: "N1", Sex: models.SexMale, OwnerID: "P1", Status: models.NobleStatusFree},
			{ID: "N2", Sex: models.SexMale, OwnerID: "P2", Status: models.NobleStatusDungeon},
		},
	}
	game := gameDocument{Players: []playerDocument{{ID: "P1", ActorID: "uid-1"}, {ID: "P2", ActorID: "uid-2"}}}
	submitted, remaining := progressFromUIDs(game, state, nil)
	if len(submitted) != 0 || !reflect.DeepEqual(remaining, []models.PlayerID{"P1"}) {
		t.Fatalf("progress = submitted %v remaining %v, want [] and [P1]", submitted, remaining)
	}

	state.Season = models.SeasonWinter
	submitted, remaining = progressFromUIDs(game, state, []string{"uid-1"})
	if !reflect.DeepEqual(submitted, []models.PlayerID{"P1"}) || !reflect.DeepEqual(remaining, []models.PlayerID{"P2"}) {
		t.Fatalf("winter progress = submitted %v remaining %v, want [P1] and [P2]", submitted, remaining)
	}
}

func TestGameDocumentTracksRequiredPlayers(t *testing.T) {
	state := &models.GameState{
		Season:  models.SeasonSpring,
		Players: []models.Player{{ID: "P1"}, {ID: "P2"}},
		Armies: []models.Army{
			{ID: "A1", OwnerID: "P1", Size: 1},
			{ID: "A2", OwnerID: "P2", Size: 1},
		},
		Nobles: []models.Noble{
			{ID: "N1", Sex: models.SexMale, OwnerID: "P1", Status: models.NobleStatusFree},
			{ID: "N2", Sex: models.SexMale, OwnerID: "P2", Status: models.NobleStatusDungeon},
		},
	}
	snapshot := store.GameSnapshot{
		ID:    "game-required",
		State: state,
		Players: []store.PlayerSlot{
			{ID: "P1", ActorID: "uid-1"},
			{ID: "P2", ActorID: "uid-2"},
		},
	}
	document := gameDocumentFromSnapshot(snapshot, time.Now(), time.Now())
	if !reflect.DeepEqual(document.RequiredUIDs, []string{"uid-1"}) {
		t.Fatalf("required UIDs = %v, want [uid-1]: P2's only noble is in the dungeon and has nothing else to submit", document.RequiredUIDs)
	}

	state.Season = models.SeasonWinter
	document = gameDocumentFromSnapshot(snapshot, time.Now(), time.Now())
	if !reflect.DeepEqual(document.RequiredUIDs, []string{"uid-1", "uid-2"}) {
		t.Fatalf("winter required UIDs = %v, want both players since winter always awaits everyone alive", document.RequiredUIDs)
	}
}
