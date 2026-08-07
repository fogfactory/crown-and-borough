package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
)

func TestHotseatServerRoutes(t *testing.T) {
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	session, err := api.NewSession("route-test", []engine.PlayerInit{{Name: "One"}, {Name: "Two"}}, balance, assets)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	server := api.WithCORS(newHotseatServer(session))

	mapRecorder := httptest.NewRecorder()
	server.ServeHTTP(mapRecorder, httptest.NewRequest(http.MethodGet, "/api/map", nil))
	if mapRecorder.Code != http.StatusOK {
		t.Fatalf("GET map = %d", mapRecorder.Code)
	}

	stateRecorder := httptest.NewRecorder()
	server.ServeHTTP(stateRecorder, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	if stateRecorder.Code != http.StatusOK {
		t.Fatalf("GET state = %d", stateRecorder.Code)
	}

	ordersRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"chains":[],"winter":[]}`))
	request.Header.Set("Content-Type", "application/json")
	server.ServeHTTP(ordersRecorder, request)
	if ordersRecorder.Code != http.StatusOK {
		t.Fatalf("POST orders = %d: %s", ordersRecorder.Code, ordersRecorder.Body.String())
	}
	var response struct {
		State struct {
			Turn int `json:"turn"`
		} `json:"state"`
	}
	if err := json.Unmarshal(ordersRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode order response: %v", err)
	}
	if response.State.Turn != 2 {
		t.Errorf("state turn = %d, want 2", response.State.Turn)
	}
}
