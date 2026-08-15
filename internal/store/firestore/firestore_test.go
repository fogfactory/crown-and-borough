package firestorestore

import (
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
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
	mapMap, err := jsonMap(mapData)
	if err != nil {
		t.Fatalf("encode map map: %v", err)
	}
	var decodedState models.GameState
	if err := decodeJSONMap(stateMap, &decodedState); err != nil {
		t.Fatalf("decode state map: %v", err)
	}
	var decodedMap mapgen.MapData
	if err := decodeJSONMap(mapMap, &decodedMap); err != nil {
		t.Fatalf("decode map map: %v", err)
	}
	if !reflect.DeepEqual(state, &decodedState) {
		t.Fatalf("state round trip changed value")
	}
	if !reflect.DeepEqual(mapData, decodedMap) {
		t.Fatalf("map round trip changed value")
	}
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
