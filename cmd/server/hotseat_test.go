package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

// newTestServer builds the application server over a memory store configured
// like newGameStore does for the given mode.
func newTestServer(t *testing.T, development bool) *http.ServeMux {
	t.Helper()
	server, _ := newTestServerWithStore(t, development)
	return server
}

func newTestServerWithStore(t *testing.T, development bool) (*http.ServeMux, *store.MemoryStore) {
	t.Helper()
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	rules, err := assetgen.LoadRules("../../assets", balance)
	if err != nil {
		t.Fatalf("load player rules: %v", err)
	}
	options := store.MemoryStoreOptions{PrivacyTracker: api.TrackTurnPrivacy, StrictMembership: !development}
	if development {
		options.MaximumPlayers = engine.MaximumGamePlayers
	}
	gameStore := store.NewMemoryStoreWithOptions(balance, assets, options)
	server := newApplicationServer(serverOptions{
		Rules:       rules,
		Balance:     balance,
		Store:       gameStore,
		Development: development,
		CreatorGate: creatorGateForEnvironment(development),
	})
	return server, gameStore
}

func serve(t *testing.T, server http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	return recorder
}

func TestHotseatPlaysATurnThroughTheGamesAPI(t *testing.T) {
	t.Setenv("SEED", "route-test")
	t.Setenv("PLAYERS", "2")
	server, gameStore := newTestServerWithStore(t, true)
	if err := createHotseatGame(context.Background(), gameStore); err != nil {
		t.Fatalf("createHotseatGame: %v", err)
	}

	list := serve(t, server, http.MethodGet, "/api/games", "")
	if list.Code != http.StatusOK {
		t.Fatalf("GET games = %d: %s", list.Code, list.Body.String())
	}
	var games []struct {
		ID   string `json:"id"`
		Seed string `json:"seed"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &games); err != nil || len(games) != 1 || games[0].Seed != "route-test" {
		t.Fatalf("hotseat games = %s (%v), want the SEED game", list.Body.String(), err)
	}
	base := "/api/games/" + games[0].ID

	for _, path := range []string{base + "/map", base + "/state?player=P2", base + "/balance", "/api/rules"} {
		if response := serve(t, server, http.MethodGet, path, ""); response.Code != http.StatusOK {
			t.Fatalf("GET %s = %d: %s", path, response.Code, response.Body.String())
		}
	}

	if pending := serve(t, server, http.MethodPost, base+"/orders?player=P1", `{"chains":[],"winter":[]}`); pending.Code != http.StatusOK {
		t.Fatalf("P1 orders = %d: %s", pending.Code, pending.Body.String())
	}
	resolved := serve(t, server, http.MethodPost, base+"/orders?player=P2", `{"chains":[],"winter":[]}`)
	if resolved.Code != http.StatusOK {
		t.Fatalf("P2 orders = %d: %s", resolved.Code, resolved.Body.String())
	}
	var response struct {
		Status string `json:"status"`
		State  struct {
			Turn int `json:"turn"`
		} `json:"state"`
	}
	if err := json.Unmarshal(resolved.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode orders response: %v", err)
	}
	if response.Status != "resolved" || response.State.Turn != 2 {
		t.Errorf("orders response = %s, want a resolved turn 2", resolved.Body.String())
	}
}

func TestHotseatAllowsEveryEnginePlayerCount(t *testing.T) {
	server := newTestServer(t, true)
	created := serve(t, server, http.MethodPost, "/api/games", `{"name":"Large","seed":"large-hotseat","players":10}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create 10-player hotseat game = %d: %s", created.Code, created.Body.String())
	}
}

func TestLegacyHotseatRoutesAreGone(t *testing.T) {
	for _, development := range []bool{true, false} {
		server := newTestServer(t, development)
		for _, route := range []struct{ method, path string }{
			{http.MethodGet, "/api/map"},
			{http.MethodGet, "/api/state?player=P1"},
			{http.MethodGet, "/api/balance"},
			{http.MethodGet, "/api/supply?territory=ROS"},
			{http.MethodPost, "/api/orders"},
			{http.MethodPost, "/api/game"},
			{http.MethodPost, "/api/reset"},
		} {
			response := serve(t, server, route.method, route.path, "")
			if response.Code != http.StatusNotFound && response.Code != http.StatusMethodNotAllowed {
				t.Errorf("development=%v %s %s = %d, want 404", development, route.method, route.path, response.Code)
			}
		}
	}
}

func TestApplicationServerDoesNotTrustPlayerQueryOutsideDevMode(t *testing.T) {
	server := newTestServer(t, false)
	response := serve(t, server, http.MethodGet, "/api/games?player=P1", "")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("GET games outside dev mode = %d: %s", response.Code, response.Body.String())
	}
}
