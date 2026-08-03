package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
)

func TestMapHandler(t *testing.T) {
	want := mapgen.MapData{Territories: []mapgen.Territory{
		{
			ID:          "T01",
			Code:        "ROS",
			Name:        "Rosemont",
			Terrain:     "plain",
			LieuDit:     true,
			Points:      [][2]int{{0, 0}, {100, 0}, {0, 100}},
			Adjacencies: []string{"T02", "T03"},
		},
	}}
	mapJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal map: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/map", nil)
	request.Header.Set("Origin", developmentOrigin)
	WithCORS(MapHandler(mapJSON)).ServeHTTP(recorder, request)

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

	var got mapgen.MapData
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		errorfMapMismatch(t, got, want)
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
