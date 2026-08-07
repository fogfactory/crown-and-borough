package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
)

func TestHotseatSessionResolvesAndRecreatesGame(t *testing.T) {
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	session, err := NewSession("session-test", []engine.PlayerInit{{Name: "One"}, {Name: "Two"}}, balance, assets)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	stateRecorder := httptest.NewRecorder()
	session.StateHTTP(stateRecorder, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	if stateRecorder.Code != http.StatusOK {
		t.Fatalf("GET state = %d", stateRecorder.Code)
	}
	var initial StateView
	if err := json.Unmarshal(stateRecorder.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode initial state: %v", err)
	}
	if initial.Turn != 1 || len(initial.Players) != 2 {
		t.Fatalf("initial state = %#v", initial)
	}
	mapBefore := httptest.NewRecorder()
	session.MapHTTP(mapBefore, httptest.NewRequest(http.MethodGet, "/api/map", nil))
	var mapDocument mapgen.MapData
	if err := json.Unmarshal(mapBefore.Body.Bytes(), &mapDocument); err != nil {
		t.Fatalf("decode map: %v", err)
	}
	winterCode := mapDocument.Territories[0].Code

	ordersRecorder := httptest.NewRecorder()
	session.OrdersHTTP(ordersRecorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"chains":[],"winter":[]}`)))
	if ordersRecorder.Code != http.StatusOK {
		t.Fatalf("POST orders = %d: %s", ordersRecorder.Code, ordersRecorder.Body.String())
	}
	var response struct {
		Report struct {
			Header struct {
				Turn int `json:"turn"`
			} `json:"header"`
		} `json:"report"`
		State StateView `json:"state"`
	}
	if err := json.Unmarshal(ordersRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode orders response: %v", err)
	}
	if response.Report.Header.Turn != 1 || response.State.Turn != 2 {
		t.Fatalf("orders response = %#v, want report turn 1 and state turn 2", response)
	}
	mapAfter := httptest.NewRecorder()
	session.MapHTTP(mapAfter, httptest.NewRequest(http.MethodGet, "/api/map", nil))
	if !bytes.Equal(mapBefore.Body.Bytes(), mapAfter.Body.Bytes()) {
		t.Fatal("map changed after resolving a turn")
	}

	invalidRecorder := httptest.NewRecorder()
	session.OrdersHTTP(invalidRecorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"chains":[{"player":"P1","noble":"BAD","text":"BAD"}],"winter":[]}`)))
	if invalidRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid POST orders = %d, want 400", invalidRecorder.Code)
	}
	if !bytes.Contains(invalidRecorder.Body.Bytes(), []byte(`"line":1`)) {
		t.Fatalf("invalid response = %s, want line number", invalidRecorder.Body.String())
	}

	for index := 0; index < 2; index++ {
		advanceRecorder := httptest.NewRecorder()
		session.OrdersHTTP(advanceRecorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"chains":[],"winter":[]}`)))
		if advanceRecorder.Code != http.StatusOK {
			t.Fatalf("advance turn %d = %d: %s", index, advanceRecorder.Code, advanceRecorder.Body.String())
		}
	}
	winterChainRecorder := httptest.NewRecorder()
	session.OrdersHTTP(winterChainRecorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"chains":[{}],"winter":[]}`)))
	if winterChainRecorder.Code != http.StatusBadRequest {
		t.Fatalf("chains in winter = %d, want 400", winterChainRecorder.Code)
	}
	winterOrdersRecorder := httptest.NewRecorder()
	session.OrdersHTTP(winterOrdersRecorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"chains":[],"winter":[{"player":"P1","lines":"R T `+winterCode+`"}]}`)))
	if winterOrdersRecorder.Code != http.StatusOK {
		t.Fatalf("winter orders = %d, want 200: %s", winterOrdersRecorder.Code, winterOrdersRecorder.Body.String())
	}

	gameRecorder := httptest.NewRecorder()
	session.GameHTTP(gameRecorder, httptest.NewRequest(http.MethodPost, "/api/game", strings.NewReader(`{"seed":"replacement","players":3}`)))
	if gameRecorder.Code != http.StatusOK {
		t.Fatalf("POST game = %d: %s", gameRecorder.Code, gameRecorder.Body.String())
	}
	var gameResponse struct {
		Map struct {
			Territories []struct{} `json:"territories"`
		} `json:"map"`
		State StateView `json:"state"`
	}
	if err := json.Unmarshal(gameRecorder.Body.Bytes(), &gameResponse); err != nil {
		t.Fatalf("decode game response: %v", err)
	}
	if gameResponse.State.Turn != 1 || len(gameResponse.State.Players) != 3 {
		t.Fatalf("replacement state = %#v", gameResponse.State)
	}
	resetRecorder := httptest.NewRecorder()
	session.ResetHTTP(resetRecorder, httptest.NewRequest(http.MethodPost, "/api/reset", nil))
	if resetRecorder.Code != http.StatusOK {
		t.Fatalf("POST reset = %d: %s", resetRecorder.Code, resetRecorder.Body.String())
	}
	var resetResponse struct {
		State StateView `json:"state"`
	}
	if err := json.Unmarshal(resetRecorder.Body.Bytes(), &resetResponse); err != nil {
		t.Fatalf("decode reset response: %v", err)
	}
	if resetResponse.State.Turn != 1 || len(resetResponse.State.Players) != 2 {
		t.Fatalf("reset state = %#v, want default game", resetResponse.State)
	}
	outOfSeasonRecorder := httptest.NewRecorder()
	session.OrdersHTTP(outOfSeasonRecorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"chains":[],"winter":[{"player":"P1","lines":"R T `+winterCode+`"}]}`)))
	if outOfSeasonRecorder.Code != http.StatusBadRequest {
		t.Fatalf("winter orders out of season = %d, want 400", outOfSeasonRecorder.Code)
	}
}
