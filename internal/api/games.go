package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/i18n"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

// GamesHandler serves the multi-game API. It deliberately receives the actor
// resolver as a dependency: the local test server can use ?player while a
// hosted server can resolve Firebase identity into the same store.Actor.
type GamesHandler struct {
	store store.GameStore
	rules assetgen.Rules
	actor ActorResolver
}

func NewGamesHandler(gameStore store.GameStore, rules assetgen.Rules, resolve ActorResolver) http.Handler {
	return &GamesHandler{store: gameStore, rules: rules, actor: resolve}
}

// NewDevGamesHandler is a convenience constructor for the local multi-game
// test server. Production callers should pass BearerActorResolver instead.
func NewDevGamesHandler(gameStore store.GameStore, rules assetgen.Rules, defaultPlayer string) http.Handler {
	handler := &GamesHandler{store: gameStore, rules: rules, actor: DevActorResolver(defaultPlayer)}
	return handler
}

func (h *GamesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeAPIError(w, http.StatusInternalServerError, "store_unavailable", "game store is not configured")
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "/api/games" {
		h.handleCollection(w, r)
		return
	}
	const prefix = "/api/games/"
	if !strings.HasPrefix(path, prefix) {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 1 {
		h.handleDetail(w, r, store.GameID(parts[0]))
		return
	}
	h.handleSubresource(w, r, store.GameID(parts[0]), parts[1:])
}

func (h *GamesHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.create(w, r)
		return
	}
	if r.Method == http.MethodGet {
		actor, ok := h.resolveActor(w, r)
		if !ok {
			return
		}
		games, err := h.store.List(r.Context(), actor)
		if err != nil {
			h.writeStoreError(w, err)
			return
		}
		response := make([]gameListView, 0, len(games))
		for _, game := range games {
			response = append(response, makeGameListView(game))
		}
		writeJSON(w, http.StatusOK, response)
		return
	}
	w.Header().Set("Allow", "GET, POST")
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (h *GamesHandler) create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.resolveActor(w, r)
	if !ok {
		return
	}
	request, err := decodeCreateRequest(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_game_request", err.Error())
		return
	}
	snapshot, err := h.store.Create(r.Context(), actor, request)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, makeGameDetailView(snapshot))
}

func (h *GamesHandler) handleDetail(w http.ResponseWriter, r *http.Request, id store.GameID) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	actor, ok := h.resolveActor(w, r)
	if !ok {
		return
	}
	snapshot, err := h.store.Get(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, makeGameDetailView(snapshot))
}

func (h *GamesHandler) handleSubresource(w http.ResponseWriter, r *http.Request, id store.GameID, parts []string) {
	actor, ok := h.resolveActor(w, r)
	if !ok {
		return
	}
	resource := parts[0]
	switch resource {
	case "map":
		if len(parts) != 1 {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		mapData, err := h.store.Map(r.Context(), actor, id)
		if err != nil {
			h.writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, mapData)
	case "state":
		if len(parts) != 1 {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		snapshot, err := h.store.State(r.Context(), actor, id)
		if err != nil {
			h.writeStoreError(w, err)
			return
		}
		playerID, ok := snapshotPlayerID(snapshot, actor)
		if !ok {
			writeAPIError(w, http.StatusForbidden, "not_member", "actor is not a member of this game")
			return
		}
		writeGameState(w, snapshot.Revision, projectStateForPlayer(snapshot.State, playerID))
	case "supply":
		if len(parts) != 1 {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		territory := models.TerritoryID(r.URL.Query().Get("territory"))
		if territory == "" {
			writeAPIError(w, http.StatusBadRequest, "territory_required", "a territory is required")
			return
		}
		line, err := h.store.Supply(r.Context(), actor, id, territory)
		if err != nil {
			h.writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, line)
	case "orders":
		if len(parts) != 1 {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		h.submit(w, r, actor, id)
	case "resolve":
		if len(parts) != 1 {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		h.resolve(w, r, actor, id)
	case "reports":
		h.reports(w, r, actor, id, parts[1:])
	case "rules":
		if len(parts) != 1 {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		if _, err := h.store.Get(r.Context(), actor, id); err != nil {
			h.writeStoreError(w, err)
			return
		}
		h.serveRules(w, r)
	default:
		http.NotFound(w, r)
	}
}

func methodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (h *GamesHandler) submit(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	var request gameOrdersRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_orders_request", err.Error())
		return
	}
	expectedRevision := request.Revision
	if request.Force && expectedRevision == 0 {
		snapshot, err := h.store.Get(r.Context(), actor, id)
		if err != nil {
			h.writeStoreError(w, err)
			return
		}
		expectedRevision = snapshot.Revision
	}
	result, err := h.store.Submit(r.Context(), actor, id, store.SubmitRequest{
		Chains:           request.Chains,
		Winter:           request.Winter,
		Force:            request.Force,
		ExpectedRevision: expectedRevision,
	})
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	h.writeSubmitResult(w, actor, result)
}

func (h *GamesHandler) resolve(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	snapshot, err := h.store.Get(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	var result store.SubmitResult
	if revisioned, ok := h.store.(store.RevisionedGameStore); ok {
		result, err = revisioned.ResolveAt(r.Context(), actor, id, snapshot.Revision)
	} else {
		result, err = h.store.Resolve(r.Context(), actor, id)
	}
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	h.writeSubmitResult(w, actor, result)
}

func (h *GamesHandler) reports(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID, parts []string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(parts) == 0 {
		records, err := h.store.Reports(r.Context(), actor, id)
		if err != nil {
			h.writeStoreError(w, err)
			return
		}
		response := make([]reportSummaryView, len(records))
		for index, record := range records {
			response[index] = reportSummaryView{Index: index, Header: record.Report.Header}
		}
		writeJSON(w, http.StatusOK, response)
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	index, err := strconv.Atoi(parts[0])
	if err != nil || index < 0 {
		writeAPIError(w, http.StatusNotFound, "report_not_found", "report index not found")
		return
	}
	record, err := h.store.Report(r.Context(), actor, id, index)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	playerID, ok := h.playerIDForGame(r.Context(), actor, id)
	if !ok {
		writeAPIError(w, http.StatusForbidden, "not_member", "actor is not a member of this game")
		return
	}
	writeJSON(w, http.StatusOK, projectReport(record.Report, playerID, record.Privacy))
}

func (h *GamesHandler) writeSubmitResult(w http.ResponseWriter, actor store.Actor, result store.SubmitResult) {
	playerID, ok := snapshotPlayerID(result.Snapshot, actor)
	if !ok {
		writeAPIError(w, http.StatusForbidden, "not_member", "actor is not a member of this game")
		return
	}
	response := gameOrdersResponse{
		Status:    result.Status,
		Player:    result.Player,
		Submitted: result.Submitted,
		Remaining: result.Remaining,
		Resolved:  result.Resolved,
		Forced:    result.Forced,
		Revision:  result.Snapshot.Revision,
		State:     projectStateForPlayer(result.Snapshot.State, playerID),
	}
	if result.Report != nil {
		report := projectReport(result.Report.Report, playerID, result.Report.Privacy)
		response.Report = &report
	}
	writeJSON(w, http.StatusOK, response)
}

func writeGameState(w http.ResponseWriter, revision store.Revision, state StateView) {
	writeJSON(w, http.StatusOK, struct {
		StateView
		Revision store.Revision `json:"revision"`
	}{StateView: state, Revision: revision})
}

func (h *GamesHandler) playerIDForGame(ctx context.Context, actor store.Actor, id store.GameID) (models.PlayerID, bool) {
	snapshot, err := h.store.Get(ctx, actor, id)
	if err != nil {
		return "", false
	}
	return snapshotPlayerID(snapshot, actor)
}

func (h *GamesHandler) serveRules(w http.ResponseWriter, r *http.Request) {
	document, ok := h.rules.Document(r.URL.Query().Get("lang"))
	if !ok {
		http.Error(w, "rules translation not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document)
}

func (h *GamesHandler) resolveActor(w http.ResponseWriter, r *http.Request) (store.Actor, bool) {
	actor, err := actorFromRequest(r, h.actor)
	if err != nil {
		writeActorError(w, err)
		return store.Actor{}, false
	}
	return actor, true
}

func (h *GamesHandler) writeStoreError(w http.ResponseWriter, err error) {
	var inputErrors *engine.InputErrors
	switch {
	case errors.As(err, &inputErrors):
		writeResolutionError(w, err, i18n.English)
	case errors.Is(err, store.ErrUnknownGame), errors.Is(err, store.ErrInvalidReport):
		writeAPIError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, store.ErrNotMember):
		writeAPIError(w, http.StatusForbidden, "not_member", err.Error())
	case errors.Is(err, store.ErrInvalidPlayers):
		writeAPIError(w, http.StatusBadRequest, "invalid_players", err.Error())
	case errors.Is(err, store.ErrRevisionConflict):
		writeAPIError(w, http.StatusConflict, "revision_conflict", err.Error())
	case errors.Is(err, store.ErrGameFinished), errors.Is(err, store.ErrEliminated):
		writeAPIError(w, http.StatusConflict, "game_not_playable", err.Error())
	case errors.Is(err, engine.ErrSupplyLineWinter):
		writeAPIError(w, http.StatusConflict, "supply_unavailable", err.Error())
	case errors.Is(err, engine.ErrSupplyLineUnknownTerritory), errors.Is(err, engine.ErrSupplyLineNoSource), errors.Is(err, engine.ErrSupplyLineNoArmy):
		writeAPIError(w, http.StatusNotFound, "supply_target_not_found", err.Error())
	default:
		writeAPIError(w, http.StatusBadRequest, "invalid_orders", err.Error())
	}
}

type gameOrdersRequest struct {
	Chains   []engine.ChainSubmission  `json:"chains"`
	Winter   []engine.WinterSubmission `json:"winter"`
	Force    bool                      `json:"force,omitempty"`
	Revision store.Revision            `json:"revision,omitempty"`
}

type gameOrdersResponse struct {
	Status    string            `json:"status"`
	Player    models.PlayerID   `json:"player"`
	Submitted []models.PlayerID `json:"submitted"`
	Remaining []models.PlayerID `json:"remaining"`
	Resolved  bool              `json:"resolved"`
	Forced    bool              `json:"forced,omitempty"`
	Revision  store.Revision    `json:"revision"`
	State     StateView         `json:"state"`
	Report    *TurnReportView   `json:"report,omitempty"`
}

type gameListView struct {
	ID       store.GameID     `json:"id"`
	Name     string           `json:"name"`
	Seed     string           `json:"seed"`
	Status   store.Status     `json:"status"`
	Players  []PlayerSlotView `json:"players"`
	Turn     int              `json:"turn"`
	Season   models.Season    `json:"season"`
	Revision store.Revision   `json:"revision"`
}

type gameDetailView struct {
	ID       store.GameID     `json:"id"`
	Name     string           `json:"name"`
	Seed     string           `json:"seed"`
	Status   store.Status     `json:"status"`
	Winner   *models.PlayerID `json:"winner,omitempty"`
	Players  []PlayerSlotView `json:"players"`
	Turn     int              `json:"turn"`
	Season   models.Season    `json:"season"`
	Revision store.Revision   `json:"revision"`
}

type PlayerSlotView struct {
	ID        models.PlayerID `json:"id"`
	Name      string          `json:"name"`
	Color     string          `json:"color"`
	Submitted bool            `json:"submitted"`
}

type reportSummaryView struct {
	Index  int                 `json:"index"`
	Header engine.ReportHeader `json:"header"`
}

func makeGameListView(snapshot store.GameSnapshot) gameListView {
	return gameListView{
		ID:       snapshot.ID,
		Name:     snapshot.Name,
		Seed:     snapshot.Seed,
		Status:   snapshot.Status,
		Players:  makePlayerSlotViews(snapshot),
		Turn:     snapshot.State.Turn,
		Season:   snapshot.State.Season,
		Revision: snapshot.Revision,
	}
}

func makeGameDetailView(snapshot store.GameSnapshot) gameDetailView {
	return gameDetailView{
		ID:       snapshot.ID,
		Name:     snapshot.Name,
		Seed:     snapshot.Seed,
		Status:   snapshot.Status,
		Winner:   snapshot.Winner,
		Players:  makePlayerSlotViews(snapshot),
		Turn:     snapshot.State.Turn,
		Season:   snapshot.State.Season,
		Revision: snapshot.Revision,
	}
}

func makePlayerSlotViews(snapshot store.GameSnapshot) []PlayerSlotView {
	views := make([]PlayerSlotView, len(snapshot.Players))
	for index, player := range snapshot.Players {
		_, submitted := snapshot.Submissions[player.ID]
		views[index] = PlayerSlotView{ID: player.ID, Name: player.Name, Color: player.Color, Submitted: submitted}
	}
	return views
}

func snapshotPlayerID(snapshot store.GameSnapshot, actor store.Actor) (models.PlayerID, bool) {
	actorID := strings.TrimSpace(actor.ID)
	for _, player := range snapshot.Players {
		if player.ActorID == actorID {
			return player.ID, true
		}
	}
	return "", false
}

type createGameBody struct {
	Name    string          `json:"name"`
	Seed    string          `json:"seed"`
	Players json.RawMessage `json:"players"`
}

func decodeCreateRequest(r *http.Request) (store.CreateRequest, error) {
	var body createGameBody
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		return store.CreateRequest{}, err
	}
	players, err := decodeCreatePlayers(body.Players)
	if err != nil {
		return store.CreateRequest{}, err
	}
	return store.CreateRequest{Name: body.Name, Seed: body.Seed, Players: players}, nil
}

func decodeCreatePlayers(raw json.RawMessage) ([]engine.PlayerInit, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, fmt.Errorf("players must be provided")
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err == nil {
		players := make([]engine.PlayerInit, len(names))
		for index, name := range names {
			players[index] = engine.PlayerInit{Name: name}
		}
		return players, nil
	}
	var players []engine.PlayerInit
	if err := json.Unmarshal(raw, &players); err == nil {
		return players, nil
	}
	var count int
	if err := json.Unmarshal(raw, &count); err == nil && count >= 0 {
		players = make([]engine.PlayerInit, count)
		return players, nil
	}
	return nil, fmt.Errorf("players must be an array of names or player objects")
}
