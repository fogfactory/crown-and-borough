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
	rules, err := assetgen.LoadRules("../../assets")
	if err != nil {
		t.Fatalf("load player rules: %v", err)
	}
	session, err := api.NewSession("route-test", []engine.PlayerInit{{Name: "One"}, {Name: "Two"}}, balance, assets)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	server := api.WithCORS(newHotseatServer(session, rules))

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

	rulesRecorder := httptest.NewRecorder()
	server.ServeHTTP(rulesRecorder, httptest.NewRequest(http.MethodGet, "/api/rules", nil))
	if rulesRecorder.Code != http.StatusOK {
		t.Fatalf("GET rules = %d", rulesRecorder.Code)
	}
	if !strings.Contains(rulesRecorder.Body.String(), "# Règles du jeu") {
		t.Error("GET rules does not contain the player rules heading")
	}

	submit := func(player string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"player":"`+player+`","chains":[],"winter":[]}`))
		request.Header.Set("Content-Type", "application/json")
		server.ServeHTTP(recorder, request)
		return recorder
	}
	if pendingRecorder := submit("P1"); pendingRecorder.Code != http.StatusOK {
		t.Fatalf("P1 POST orders = %d: %s", pendingRecorder.Code, pendingRecorder.Body.String())
	}
	ordersRecorder := submit("P2")
	if ordersRecorder.Code != http.StatusOK {
		t.Fatalf("P2 POST orders = %d: %s", ordersRecorder.Code, ordersRecorder.Body.String())
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
