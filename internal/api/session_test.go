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
	submit := func(player string, body string) (int, []byte) {
		t.Helper()
		recorder := httptest.NewRecorder()
		session.OrdersHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(body)))
		return recorder.Code, recorder.Body.Bytes()
	}

	status, body := submit("P1", `{"player":"P1","chains":[],"winter":[]}`)
	if status != http.StatusOK {
		t.Fatalf("P1 POST orders = %d: %s", status, body)
	}
	var response struct {
		Status    string   `json:"status"`
		Submitted []string `json:"submitted"`
		Remaining []string `json:"remaining"`
		Report    struct {
			Header struct {
				Turn int `json:"turn"`
			} `json:"header"`
		} `json:"report"`
		State StateView `json:"state"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode orders response: %v", err)
	}
	if response.Status != "pending" || len(response.Submitted) != 1 || len(response.Remaining) != 1 || response.State.Turn != 1 {
		t.Fatalf("P1 response = %#v, want pending turn 1", response)
	}
	// Updating an already submitted player replaces its pending input without
	// resolving the turn.
	status, body = submit("P1", `{"player":"P1","chains":[],"winter":[]}`)
	if status != http.StatusOK {
		t.Fatalf("P1 update = %d: %s", status, body)
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode P1 update: %v", err)
	}
	if response.Status != "pending" || response.State.Turn != 1 {
		t.Fatalf("P1 update response = %#v, want pending turn 1", response)
	}

	status, body = submit("P2", `{"player":"P2","chains":[],"winter":[]}`)
	if status != http.StatusOK {
		t.Fatalf("P2 POST orders = %d: %s", status, body)
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode resolved orders: %v", err)
	}
	if response.Status != "resolved" || response.Report.Header.Turn != 1 || response.State.Turn != 2 {
		t.Fatalf("resolved response = %#v, want report turn 1 and state turn 2", response)
	}
	mapAfter := httptest.NewRecorder()
	session.MapHTTP(mapAfter, httptest.NewRequest(http.MethodGet, "/api/map", nil))
	if !bytes.Equal(mapBefore.Body.Bytes(), mapAfter.Body.Bytes()) {
		t.Fatal("map changed after resolving a turn")
	}

	invalidRecorder := httptest.NewRecorder()
	session.OrdersHTTP(invalidRecorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"player":"P1","chains":[{"player":"P1","noble":"BAD","text":"BAD"}],"winter":[]}`)))
	if invalidRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid POST orders = %d, want 400", invalidRecorder.Code)
	}
	if !bytes.Contains(invalidRecorder.Body.Bytes(), []byte(`"line":1`)) {
		t.Fatalf("invalid response = %s, want line number", invalidRecorder.Body.String())
	}

	for index := 0; index < 2; index++ {
		status, body = submit("P1", `{"player":"P1","chains":[],"winter":[]}`)
		if status != http.StatusOK {
			t.Fatalf("advance P1 turn %d = %d: %s", index, status, body)
		}
		status, body = submit("P2", `{"player":"P2","chains":[],"winter":[]}`)
		if status != http.StatusOK {
			t.Fatalf("advance P2 turn %d = %d: %s", index, status, body)
		}
	}
	status, body = submit("P1", `{"player":"P1","chains":[{}],"winter":[]}`)
	if status != http.StatusBadRequest {
		t.Fatalf("chains in winter = %d, want 400: %s", status, body)
	}
	status, body = submit("P1", `{"player":"P1","chains":[],"winter":[{"player":"P1","lines":"R T `+winterCode+`"}]}`)
	if status != http.StatusOK {
		t.Fatalf("winter P1 orders = %d, want 200: %s", status, body)
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode pending winter response: %v", err)
	}
	if response.Status != "pending" || response.State.Turn != 4 {
		t.Fatalf("pending winter response = %#v, want turn 4", response)
	}
	status, body = submit("P2", `{"player":"P2","chains":[],"winter":[]}`)
	if status != http.StatusOK {
		t.Fatalf("winter P2 orders = %d, want 200: %s", status, body)
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
