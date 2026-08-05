package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
)

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	newServer(func(int) ([]byte, error) {
		t.Fatal("health check resolved a map")
		return nil, nil
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /healthz = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMap(t *testing.T) {
	mapJSON := []byte(`{"territories":[]}`)
	resolvedPlayers := 0
	resolve := func(players int) ([]byte, error) {
		resolvedPlayers = players
		return mapJSON, nil
	}
	req := httptest.NewRequest(http.MethodGet, "/api/map?players=3", nil)
	rec := httptest.NewRecorder()

	newServer(resolve).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/map = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != string(mapJSON) {
		t.Errorf("GET /api/map body = %q, want %q", got, mapJSON)
	}
	if resolvedPlayers != 3 {
		t.Errorf("resolver players = %d, want 3", resolvedPlayers)
	}
}

func TestServerMapPlayerValidation(t *testing.T) {
	resolved := make([]int, 0)
	server := newServer(func(players int) ([]byte, error) {
		resolved = append(resolved, players)
		return []byte(`{"territories":[]}`), nil
	})
	tests := []struct {
		path       string
		wantStatus int
		wantCalls  []int
	}{
		{path: "/api/map", wantStatus: http.StatusOK, wantCalls: []int{4}},
		{path: "/api/map?players=2", wantStatus: http.StatusOK, wantCalls: []int{4, 2}},
		{path: "/api/map?players=5", wantStatus: http.StatusOK, wantCalls: []int{4, 2, 5}},
		{path: "/api/map?players=1", wantStatus: http.StatusBadRequest, wantCalls: []int{4, 2, 5}},
		{path: "/api/map?players=6", wantStatus: http.StatusBadRequest, wantCalls: []int{4, 2, 5}},
		{path: "/api/map?players=abc", wantStatus: http.StatusBadRequest, wantCalls: []int{4, 2, 5}},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if recorder.Code != test.wantStatus {
			t.Errorf("GET %s = %d, want %d", test.path, recorder.Code, test.wantStatus)
		}
		if len(resolved) != len(test.wantCalls) {
			t.Errorf("GET %s resolver calls = %v, want %v", test.path, resolved, test.wantCalls)
			continue
		}
		for index, players := range test.wantCalls {
			if resolved[index] != players {
				t.Errorf("GET %s resolver calls = %v, want %v", test.path, resolved, test.wantCalls)
				break
			}
		}
	}
}

func TestMapResolverConfiguration(t *testing.T) {
	resolver := &mapResolver{
		seed:   defaultSeed,
		assets: loadTestAssets(t),
	}

	for _, players := range []int{2, 3, 4, 5} {
		mapJSON, err := resolver.resolve(players)
		if err != nil {
			t.Fatalf("resolve(%d): %v", players, err)
		}

		var data mapgen.MapData
		if err := json.Unmarshal(mapJSON, &data); err != nil {
			t.Fatalf("unmarshal resolve(%d): %v", players, err)
		}
		if got, want := len(data.Territories), mapgen.TerritoriesPerPlayer*players; got != want {
			t.Errorf("resolve(%d) territories = %d, want %d", players, got, want)
		}
		villages := 0
		for _, territory := range data.Territories {
			if territory.Village {
				villages++
			}
		}
		if got, want := villages, players+1; got != want {
			t.Errorf("resolve(%d) villages = %d, want %d", players, got, want)
		}
	}
}

func TestMapResolverCachesStableBytes(t *testing.T) {
	calls := 0
	configs := make([]mapgen.Config, 0)
	resolver := &mapResolver{
		seed: "server-cache",
		generate: func(_ string, _ assetgen.Assets, cfg mapgen.Config) (mapgen.MapData, error) {
			calls++
			configs = append(configs, cfg)
			return mapgen.MapData{Territories: make([]mapgen.Territory, cfg.SiteCount)}, nil
		},
	}

	first, err := resolver.resolve(4)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	second, err := resolver.resolve(4)
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Error("repeated resolve returned different map bytes")
	}
	if _, err := resolver.resolve(2); err != nil {
		t.Fatalf("resolve(2): %v", err)
	}
	if _, err := resolver.resolve(2); err != nil {
		t.Fatalf("second resolve(2): %v", err)
	}
	if calls != 2 {
		t.Errorf("generator calls = %d, want one per player count", calls)
	}
	if got := len(resolver.cache); got != 2 {
		t.Errorf("cache size = %d, want 2", got)
	}
	for _, cfg := range configs {
		players := cfg.SiteCount / mapgen.TerritoriesPerPlayer
		if cfg.Width != 1000 || cfg.Height != 700 {
			t.Errorf("generation viewport = %dx%d, want 1000x700", cfg.Width, cfg.Height)
		}
		if cfg.VillageCount != players+1 {
			t.Errorf("village count = %d for %d players, want %d", cfg.VillageCount, players, players+1)
		}
	}
}

func loadTestAssets(t *testing.T) assetgen.Assets {
	t.Helper()
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load test assets: %v", err)
	}
	return assets
}
