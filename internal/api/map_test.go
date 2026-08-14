package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
)

func TestMapHandler(t *testing.T) {
	want := mapgen.MapData{Territories: []mapgen.Territory{
		{
			ID:          "ROS",
			Name:        "Rosemont",
			Terrain:     "plain",
			Village:     true,
			Points:      [][2]int{{0, 0}, {100, 0}, {0, 100}},
			Adjacencies: []string{"BOI", "BRU"},
			Impassable:  []string{},
		},
	}}
	mapJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal map: %v", err)
	}
	resolvedPlayers := 0
	resolve := func(players int) ([]byte, error) {
		resolvedPlayers = players
		return mapJSON, nil
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/map", nil)
	request.Header.Set("Origin", developmentOrigin)
	WithCORS(MapHandler(resolve)).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("GET /api/map = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != developmentOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, developmentOrigin)
	}
	if got := recorder.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want Origin", got)
	}
	if resolvedPlayers != DefaultPlayers {
		t.Errorf("resolver players = %d, want %d", resolvedPlayers, DefaultPlayers)
	}

	var got mapgen.MapData
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		errorfMapMismatch(t, got, want)
	}
}

func TestMapHandlerPlayers(t *testing.T) {
	tests := []struct {
		name     string
		rawQuery string
		want     int
	}{
		{name: "default", want: 4},
		{name: "minimum", rawQuery: "players=2", want: 2},
		{name: "three players", rawQuery: "players=3", want: 3},
		{name: "four players", rawQuery: "players=4", want: 4},
		{name: "maximum", rawQuery: "players=5", want: 5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolvedPlayers := 0
			resolver := func(players int) ([]byte, error) {
				resolvedPlayers = players
				return []byte(`{"territories":[]}`), nil
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/map", nil)
			request.URL.RawQuery = test.rawQuery

			MapHandler(resolver).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Errorf("GET /api/map?%s = %d, want %d", test.rawQuery, recorder.Code, http.StatusOK)
			}
			if resolvedPlayers != test.want {
				t.Errorf("resolver players = %d, want %d", resolvedPlayers, test.want)
			}
			if got := recorder.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
		})
	}
}

func TestMapHandlerRejectsInvalidPlayers(t *testing.T) {
	tests := []struct {
		name     string
		rawQuery string
	}{
		{name: "below range", rawQuery: "players=1"},
		{name: "above range", rawQuery: "players=17"},
		{name: "not a number", rawQuery: "players=abc"},
		{name: "empty", rawQuery: "players="},
		{name: "multiple values", rawQuery: "players=2&players=3"},
		{name: "malformed query", rawQuery: "players=%zz"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			resolver := func(int) ([]byte, error) {
				called = true
				return nil, nil
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/map", nil)
			request.URL.RawQuery = test.rawQuery

			MapHandler(resolver).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf("GET /api/map?%s = %d, want %d", test.rawQuery, recorder.Code, http.StatusBadRequest)
			}
			if called {
				t.Error("resolver was called for invalid players")
			}
		})
	}
}

func TestMapHandlerResolverError(t *testing.T) {
	resolver := func(int) ([]byte, error) {
		return nil, errors.New("map generation failed")
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/map", nil)

	MapHandler(resolver).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/map = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestWithCORSOptions(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/map", nil)

	WithCORS(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Errorf("OPTIONS /api/map = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if nextCalled {
		t.Error("OPTIONS request reached the wrapped handler")
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("OPTIONS response body = %q, want empty", recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != developmentOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, developmentOrigin)
	}
}

func errorfMapMismatch(t *testing.T, got, want mapgen.MapData) {
	t.Helper()
	t.Errorf("map response = %#v, want %#v", got, want)
}
