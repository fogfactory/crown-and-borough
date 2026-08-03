package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	newServer([]byte(`{"territories":[]}`)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /healthz = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMap(t *testing.T) {
	mapJSON := []byte(`{"territories":[]}`)
	req := httptest.NewRequest(http.MethodGet, "/api/map", nil)
	rec := httptest.NewRecorder()

	newServer(mapJSON).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/map = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != string(mapJSON) {
		t.Errorf("GET /api/map body = %q, want %q", got, mapJSON)
	}
}
