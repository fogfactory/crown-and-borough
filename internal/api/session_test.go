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
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestPendingPlayersSkipsPlayersWithoutEmittingNobleOnActionTurns(t *testing.T) {
	session := &Session{
		game: &models.GameState{
			Season: models.SeasonSummer,
			Players: []models.Player{
				{ID: "P1"},
				{ID: "P2"},
			},
			Nobles: []models.Noble{
				{OwnerID: "P1", Status: models.NobleStatusFree},
				{OwnerID: "P2", Status: models.NobleStatusDungeon},
			},
		},
		pending: map[models.PlayerID]engine.OrdersInput{
			"P1": {},
		},
	}

	submitted, remaining := session.pendingPlayersLocked()
	if len(submitted) != 1 || submitted[0] != "P1" {
		t.Fatalf("submitted = %#v, want [P1]", submitted)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining = %#v, want no required players", remaining)
	}

	session.game.Nobles[1].Status = models.NobleStatusHostage
	submitted, remaining = session.pendingPlayersLocked()
	if len(submitted) != 1 || len(remaining) != 1 || remaining[0] != "P2" {
		t.Fatalf("hostage remaining = submitted %#v, remaining %#v; want P2 required", submitted, remaining)
	}

	session.game.Season = models.SeasonWinter
	session.pending = map[models.PlayerID]engine.OrdersInput{}
	_, remaining = session.pendingPlayersLocked()
	if len(remaining) != 2 || remaining[0] != "P1" || remaining[1] != "P2" {
		t.Fatalf("winter remaining = %#v, want [P1 P2]", remaining)
	}
}

func TestOrdersHTTPLocalizesValidationErrors(t *testing.T) {
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	session, err := NewSession("localized-errors", []engine.PlayerInit{{Name: "One"}, {Name: "Two"}}, balance, assets)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	for _, test := range []struct {
		name     string
		language string
		contains string
	}{
		{name: "English", language: "en", contains: "the first content line must contain exactly one noble code"},
		{name: "French", language: "fr", contains: "la première ligne de contenu doit contenir exactement un code de noble"},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/orders?lang="+test.language, strings.NewReader(`{"player":"P1","chains":[{"noble":"BAD","text":"BAD EXTRA"}],"winter":[]}`))
			session.OrdersHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("POST orders = %d: %s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), test.contains) {
				t.Fatalf("localized error = %s, want %q", recorder.Body.String(), test.contains)
			}
		})
	}
}

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
	resetMapRecorder := httptest.NewRecorder()
	session.MapHTTP(resetMapRecorder, httptest.NewRequest(http.MethodGet, "/api/map", nil))
	var resetMap mapgen.MapData
	if err := json.Unmarshal(resetMapRecorder.Body.Bytes(), &resetMap); err != nil {
		t.Fatalf("decode reset map: %v", err)
	}
	if len(resetMap.Territories) == 0 {
		t.Fatal("reset map has no territories")
	}
	winterCode = resetMap.Territories[0].Code
	emitter := NobleView{}
	for _, noble := range resetResponse.State.Nobles {
		if noble.Owner == "P1" && noble.Status == "free" {
			emitter = noble
			break
		}
	}
	if emitter.Code == "" {
		t.Fatal("could not find a free P1 noble for the visible reception error test")
	}
	emptyTerritory := mapgen.Territory{}
	for _, candidate := range resetMap.Territories {
		for _, stateTerritory := range resetResponse.State.Territories {
			if string(stateTerritory.ID) == candidate.ID && stateTerritory.Army == nil && len(candidate.Adjacencies) > 0 {
				emptyTerritory = candidate
				break
			}
		}
		if emptyTerritory.ID != "" {
			break
		}
	}
	if emptyTerritory.ID == "" {
		t.Fatal("could not find an empty territory for the visible reception error test")
	}
	emptyTarget := mapgen.Territory{}
	for _, candidate := range resetMap.Territories {
		if candidate.ID == emptyTerritory.Adjacencies[0] {
			emptyTarget = candidate
			break
		}
	}
	if emptyTarget.ID == "" {
		t.Fatalf("could not resolve adjacency %s on reset map", emptyTerritory.Adjacencies[0])
	}
	visibleErrorRecorder := httptest.NewRecorder()
	visibleErrorBody := `{"player":"P1","chains":[{"player":"P1","noble":"` + string(emitter.Code) + `","text":"` + string(emitter.Code) + `\n` + emptyTerritory.Code + ` A ` + emptyTarget.Code + `"}],"winter":[]}`
	session.OrdersHTTP(visibleErrorRecorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(visibleErrorBody)))
	if visibleErrorRecorder.Code != http.StatusOK {
		t.Fatalf("empty receiving position submission = %d: %s", visibleErrorRecorder.Code, visibleErrorRecorder.Body.String())
	}
	visibleErrorRecorder = httptest.NewRecorder()
	session.OrdersHTTP(visibleErrorRecorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"player":"P2","chains":[],"winter":[]}`)))
	if visibleErrorRecorder.Code != http.StatusOK {
		t.Fatalf("empty receiving position resolution = %d: %s", visibleErrorRecorder.Code, visibleErrorRecorder.Body.String())
	}
	var visibleResponse struct {
		Report struct {
			Receptions []engine.ReceptionReport `json:"receptions"`
		} `json:"report"`
	}
	if err := json.Unmarshal(visibleErrorRecorder.Body.Bytes(), &visibleResponse); err != nil {
		t.Fatalf("decode visible reception response: %v", err)
	}
	if len(visibleResponse.Report.Receptions) != 1 {
		t.Fatalf("visible reception response = %s, want one reception", visibleErrorRecorder.Body.String())
	}
	reason := visibleResponse.Report.Receptions[0].Reason
	if strings.Contains(reason, emptyTerritory.ID) {
		t.Fatalf("visible reception reason = %q, must not expose %s", reason, emptyTerritory.ID)
	}
	if !strings.Contains(reason, emptyTerritory.Code) {
		t.Fatalf("visible reception reason = %q, want code %s", reason, emptyTerritory.Code)
	}
	outOfSeasonRecorder := httptest.NewRecorder()
	session.OrdersHTTP(outOfSeasonRecorder, httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"chains":[],"winter":[{"player":"P1","lines":"R T `+winterCode+`"}]}`)))
	if outOfSeasonRecorder.Code != http.StatusBadRequest {
		t.Fatalf("winter orders out of season = %d, want 400", outOfSeasonRecorder.Code)
	}
}

func TestHotseatSessionServesSupplyLine(t *testing.T) {
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	session, err := NewSession("supply-line-test", []engine.PlayerInit{{Name: "One"}, {Name: "Two"}}, balance, assets)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	stateRecorder := httptest.NewRecorder()
	session.StateHTTP(stateRecorder, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	var state StateView
	if err := json.Unmarshal(stateRecorder.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	var armyTerritory string
	var emptyTerritory string
	for _, territory := range state.Territories {
		if territory.Army != nil && armyTerritory == "" {
			armyTerritory = string(territory.ID)
		}
		if territory.Army == nil && len(territory.Infrastructures) == 0 && emptyTerritory == "" {
			emptyTerritory = string(territory.ID)
		}
	}
	if armyTerritory == "" || emptyTerritory == "" {
		t.Fatalf("state does not contain both an army and an empty territory: %#v", state.Territories)
	}

	recorder := httptest.NewRecorder()
	session.SupplyHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/supply?territory="+armyTerritory, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET supply = %d: %s", recorder.Code, recorder.Body.String())
	}
	var line engine.SupplyLine
	if err := json.Unmarshal(recorder.Body.Bytes(), &line); err != nil {
		t.Fatalf("decode supply line: %v", err)
	}
	if line.Kind != engine.SupplyLineKindArmy || string(line.Territory) != armyTerritory || line.ArmySize < 1 {
		t.Errorf("supply line = %#v, want army at %s", line, armyTerritory)
	}

	missing := httptest.NewRecorder()
	session.SupplyHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/supply", nil))
	if missing.Code != http.StatusBadRequest {
		t.Errorf("missing territory = %d, want %d", missing.Code, http.StatusBadRequest)
	}
	empty := httptest.NewRecorder()
	session.SupplyHTTP(empty, httptest.NewRequest(http.MethodGet, "/api/supply?territory="+emptyTerritory, nil))
	if empty.Code != http.StatusNotFound {
		t.Errorf("empty territory = %d, want %d", empty.Code, http.StatusNotFound)
	}

	session.mu.Lock()
	sourceTerritoryID := models.TerritoryID(emptyTerritory)
	sourceOwner := models.PlayerID("P1")
	sourceState := session.game.TerritoryStates[sourceTerritoryID]
	sourceState.OwnerID = &sourceOwner
	sourceState.Infrastructures = append(sourceState.Infrastructures, models.InfraID("I-supply-test"))
	session.game.TerritoryStates[sourceTerritoryID] = sourceState
	session.game.Infrastructures = append(session.game.Infrastructures, models.Infrastructure{
		ID:          "I-supply-test",
		Type:        models.InfraTypeCastle,
		Level:       1,
		TerritoryID: sourceTerritoryID,
	})
	session.mu.Unlock()

	sourceRecorder := httptest.NewRecorder()
	session.SupplyHTTP(sourceRecorder, httptest.NewRequest(http.MethodGet, "/api/supply?territory="+emptyTerritory, nil))
	if sourceRecorder.Code != http.StatusOK {
		t.Fatalf("source supply = %d: %s", sourceRecorder.Code, sourceRecorder.Body.String())
	}
	var sourceZone engine.SupplyLine
	if err := json.Unmarshal(sourceRecorder.Body.Bytes(), &sourceZone); err != nil {
		t.Fatalf("decode source supply zone: %v", err)
	}
	if sourceZone.Kind != engine.SupplyLineKindSource || sourceZone.Source == nil || *sourceZone.Source != sourceTerritoryID {
		t.Errorf("source zone = %#v, want source %s", sourceZone, sourceTerritoryID)
	}
}
