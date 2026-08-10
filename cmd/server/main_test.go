package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
)

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	newServer(
		func(int) ([]byte, error) {
			t.Fatal("health check resolved a map")
			return nil, nil
		},
		func(int) ([]byte, error) {
			t.Fatal("health check resolved state")
			return nil, nil
		},
	).ServeHTTP(rec, req)

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

	newServer(resolve, func(int) ([]byte, error) {
		t.Fatal("map request resolved state")
		return nil, nil
	}).ServeHTTP(rec, req)

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
	server := newServer(
		func(players int) ([]byte, error) {
			resolved = append(resolved, players)
			return []byte(`{"territories":[]}`), nil
		},
		func(int) ([]byte, error) {
			t.Fatal("map validation resolved state")
			return nil, nil
		},
	)
	tests := []struct {
		path       string
		wantStatus int
		wantCalls  []int
	}{
		{path: "/api/map", wantStatus: http.StatusOK, wantCalls: []int{4}},
		{path: "/api/map?players=2", wantStatus: http.StatusOK, wantCalls: []int{4, 2}},
		{path: "/api/map?players=16", wantStatus: http.StatusOK, wantCalls: []int{4, 2, 16}},
		{path: "/api/map?players=1", wantStatus: http.StatusBadRequest, wantCalls: []int{4, 2, 16}},
		{path: "/api/map?players=17", wantStatus: http.StatusBadRequest, wantCalls: []int{4, 2, 16}},
		{path: "/api/map?players=abc", wantStatus: http.StatusBadRequest, wantCalls: []int{4, 2, 16}},
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

func TestState(t *testing.T) {
	stateJSON := []byte(`{"turn":5,"season":"spring","territories":[],"nobles":[]}`)
	resolvedPlayers := 0
	resolve := func(players int) ([]byte, error) {
		resolvedPlayers = players
		return stateJSON, nil
	}
	req := httptest.NewRequest(http.MethodGet, "/api/state?players=3", nil)
	rec := httptest.NewRecorder()

	newServer(
		func(int) ([]byte, error) {
			t.Fatal("state request resolved map bytes")
			return nil, nil
		},
		resolve,
	).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/state = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != string(stateJSON) {
		t.Errorf("GET /api/state body = %q, want %q", got, stateJSON)
	}
	if resolvedPlayers != 3 {
		t.Errorf("state resolver players = %d, want 3", resolvedPlayers)
	}
}

func TestServerStatePlayerValidationAndErrors(t *testing.T) {
	resolved := make([]int, 0)
	server := newServer(
		func(int) ([]byte, error) {
			t.Fatal("state validation resolved map bytes")
			return nil, nil
		},
		func(players int) ([]byte, error) {
			resolved = append(resolved, players)
			return []byte(`{}`), nil
		},
	)
	for _, test := range []struct {
		path       string
		wantStatus int
		wantCalls  []int
	}{
		{path: "/api/state", wantStatus: http.StatusOK, wantCalls: []int{4}},
		{path: "/api/state?players=2", wantStatus: http.StatusOK, wantCalls: []int{4, 2}},
		{path: "/api/state?players=16", wantStatus: http.StatusOK, wantCalls: []int{4, 2, 16}},
		{path: "/api/state?players=1", wantStatus: http.StatusBadRequest, wantCalls: []int{4, 2, 16}},
		{path: "/api/state?players=17", wantStatus: http.StatusBadRequest, wantCalls: []int{4, 2, 16}},
	} {
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

	errorServer := newServer(
		func(int) ([]byte, error) { return nil, nil },
		func(int) ([]byte, error) { return nil, errors.New("state failed") },
	)
	recorder := httptest.NewRecorder()
	errorServer.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/state resolver error = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestMapResolverConfiguration(t *testing.T) {
	resolver := &mapResolver{
		seed:   defaultSeed,
		assets: loadTestAssets(t),
	}

	for _, players := range []int{2, 3, 4, 5, 16} {
		mapJSON, err := resolver.resolve(players)
		if err != nil {
			t.Fatalf("resolve(%d): %v", players, err)
		}

		var data mapgen.MapData
		if err := json.Unmarshal(mapJSON, &data); err != nil {
			t.Fatalf("unmarshal resolve(%d): %v", players, err)
		}
		if got, want := len(data.Territories), engine.GameMapConfig(players).SiteCount; got != want {
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
	if len(configs) != 2 {
		t.Fatalf("captured configs = %d, want 2", len(configs))
	}
	for index, cfg := range configs {
		players := []int{4, 2}[index]
		want := engine.GameMapConfig(players)
		if cfg != want {
			t.Errorf("generation config for %d players = %+v, want %+v", players, cfg, want)
		}
		if cfg.Width != 1000 || cfg.Height != 700 {
			t.Errorf("generation viewport = %dx%d, want 1000x700", cfg.Width, cfg.Height)
		}
		if cfg.VillageCount != players+1 {
			t.Errorf("village count = %d for %d players, want %d", cfg.VillageCount, players, players+1)
		}
	}
}

func TestMapAndStateShareGeneratedMapData(t *testing.T) {
	assets := loadTestAssets(t)
	for _, paths := range [][]string{
		{"/api/map?players=4", "/api/state?players=4"},
		{"/api/state?players=4", "/api/map?players=4"},
	} {
		t.Run(paths[0], func(t *testing.T) {
			calls := 0
			resolver := &mapResolver{
				seed:   "shared-map-cache",
				assets: assets,
				generate: func(seed string, assets assetgen.Assets, cfg mapgen.Config) (mapgen.MapData, error) {
					calls++
					return mapgen.Generate(seed, assets, cfg)
				},
			}
			server := newServer(
				resolver.resolve,
				api.StateResolver(resolver.resolveData, resolver.seed, assets),
			)

			var mapData mapgen.MapData
			var state api.StateView
			for _, path := range paths {
				recorder := httptest.NewRecorder()
				server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
				if recorder.Code != http.StatusOK {
					t.Fatalf("GET %s = %d, want %d", path, recorder.Code, http.StatusOK)
				}
				if path == "/api/map?players=4" {
					if err := json.Unmarshal(recorder.Body.Bytes(), &mapData); err != nil {
						t.Fatalf("unmarshal map response: %v", err)
					}
					continue
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &state); err != nil {
					t.Fatalf("unmarshal state response: %v", err)
				}
			}
			if calls != 1 {
				t.Errorf("generator calls = %d, want 1 shared map generation", calls)
			}
			if len(resolver.cache) != 1 {
				t.Errorf("map cache size = %d, want 1", len(resolver.cache))
			}
			if len(mapData.Territories) != len(state.Territories) {
				t.Fatalf("map territories = %d, state territories = %d", len(mapData.Territories), len(state.Territories))
			}
			stateIDs := make(map[string]bool, len(state.Territories))
			for _, territory := range state.Territories {
				stateIDs[string(territory.ID)] = true
			}
			for _, territory := range mapData.Territories {
				if !stateIDs[territory.ID] {
					t.Errorf("state response is missing map territory %s", territory.ID)
				}
			}
		})
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
