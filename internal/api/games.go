package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	store            store.GameStore
	rules            assetgen.Rules
	actor            ActorResolver
	profiles         store.ProfileStore
	requireProfile   bool
	strictMembership bool
	inviteBaseURL    string
}

type GamesHandlerOptions struct {
	Actor            ActorResolver
	Profiles         store.ProfileStore
	RequireProfile   bool
	StrictMembership bool
	InviteBaseURL    string
}

func NewGamesHandler(gameStore store.GameStore, rules assetgen.Rules, resolve ActorResolver) http.Handler {
	return NewGamesHandlerWithOptions(gameStore, rules, GamesHandlerOptions{Actor: resolve})
}

func NewGamesHandlerWithOptions(gameStore store.GameStore, rules assetgen.Rules, options GamesHandlerOptions) http.Handler {
	profiles := options.Profiles
	if profiles == nil {
		profiles, _ = gameStore.(store.ProfileStore)
	}
	return &GamesHandler{
		store:            gameStore,
		rules:            rules,
		actor:            options.Actor,
		profiles:         profiles,
		requireProfile:   options.RequireProfile,
		strictMembership: options.StrictMembership,
		inviteBaseURL:    options.InviteBaseURL,
	}
}

// NewDevGamesHandler is a convenience constructor for the local multi-game
// test server. Production callers should pass BearerActorResolver instead.
func NewDevGamesHandler(gameStore store.GameStore, rules assetgen.Rules, defaultPlayer string) http.Handler {
	return NewGamesHandlerWithOptions(gameStore, rules, GamesHandlerOptions{Actor: DevActorResolver(defaultPlayer)})
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
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	request, err := decodeCreateRequest(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_game_request", err.Error())
		return
	}
	if h.requireProfile {
		profile, profileErr := h.requireCompletedProfile(r, actor)
		if profileErr != nil {
			h.writeProfileRequirementError(w, profileErr)
			return
		}
		request = applyProfileToCreateRequest(request, profile)
	}
	if h.strictMembership {
		request.StrictMembership = true
	}
	var (
		snapshot   store.GameSnapshot
		invitation store.InvitationSecret
	)
	if creator, ok := h.store.(interface {
		CreateWithInvitation(context.Context, store.Actor, store.CreateRequest) (store.GameCreation, error)
	}); ok {
		creation, createErr := creator.CreateWithInvitation(r.Context(), actor, request)
		if createErr != nil {
			h.writeStoreError(w, createErr)
			return
		}
		snapshot = creation.Snapshot
		invitation = creation.Invitation
	} else {
		var createErr error
		snapshot, createErr = h.store.Create(r.Context(), actor, request)
		if createErr != nil {
			h.writeStoreError(w, createErr)
			return
		}
	}
	response := makeAuthenticatedGameDetailView(snapshot, actor)
	if invitation.Code == "" {
		if inviter, ok := h.store.(interface {
			CreateInvitation(context.Context, store.Actor, store.GameID) (store.InvitationSecret, error)
		}); ok {
			var inviteErr error
			invitation, inviteErr = inviter.CreateInvitation(r.Context(), actor, snapshot.ID)
			if inviteErr != nil {
				h.writeStoreError(w, inviteErr)
				return
			}
		}
	}
	if invitation.Code != "" {
		response.InviteCode = invitation.Code
		response.InviteURL = buildInviteURL(h.inviteBaseURL, invitation.GameID, invitation.Code)
	}
	writeJSON(w, http.StatusCreated, response)
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
	writeJSON(w, http.StatusOK, makeAuthenticatedGameDetailView(snapshot, actor))
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
	case "join":
		if len(parts) != 1 {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		h.join(w, r, actor, id)
	case "invite":
		if len(parts) != 1 {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		h.invite(w, r, actor, id)
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
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
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
		Chains:           toChainSubmissions(request.Chains),
		Winter:           toWinterSubmissions(request.Winter),
		Force:            request.Force,
		ExpectedRevision: expectedRevision,
	})
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	h.writeSubmitResult(w, actor, result)
}

func toChainSubmissions(requests []chainOrderRequest) []engine.ChainSubmission {
	chains := make([]engine.ChainSubmission, len(requests))
	for index, request := range requests {
		chains[index] = engine.ChainSubmission{Noble: request.Noble, Text: request.Text}
	}
	return chains
}

func toWinterSubmissions(requests []winterOrderRequest) []engine.WinterSubmission {
	winter := make([]engine.WinterSubmission, len(requests))
	for index, request := range requests {
		winter[index] = engine.WinterSubmission{Lines: request.Lines}
	}
	return winter
}

func (h *GamesHandler) join(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	joiner, ok := h.store.(interface {
		Join(context.Context, store.Actor, store.GameID, string) (store.JoinResult, error)
	})
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "join_unavailable", "game invitations are not configured")
		return
	}
	if h.requireProfile {
		if _, err := h.requireCompletedProfile(r, actor); err != nil {
			h.writeProfileRequirementError(w, err)
			return
		}
	}
	var request joinGameRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_join_request", err.Error())
		return
	}
	result, err := joiner.Join(r.Context(), actor, id, request.InviteCode)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	response := gameJoinResponse{
		gameDetailView: makeAuthenticatedGameDetailView(result.Snapshot, actor),
		Joined:         result.Joined,
		Player:         makePlayerSlotView(result.Player, result.Snapshot),
	}
	status := http.StatusOK
	if result.Joined {
		status = http.StatusCreated
	}
	writeJSON(w, status, response)
}

func (h *GamesHandler) invite(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	inviter, ok := h.store.(interface {
		CreateInvitation(context.Context, store.Actor, store.GameID) (store.InvitationSecret, error)
	})
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "invite_unavailable", "game invitations are not configured")
		return
	}
	secret, err := inviter.CreateInvitation(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, invitationView{
		GameID:     secret.GameID,
		InviteCode: secret.Code,
		InviteURL:  buildInviteURL(h.inviteBaseURL, secret.GameID, secret.Code),
	})
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
	case errors.Is(err, store.ErrNotCreator):
		writeAPIError(w, http.StatusForbidden, "not_creator", err.Error())
	case errors.Is(err, store.ErrInvalidPlayers):
		writeAPIError(w, http.StatusBadRequest, "invalid_players", err.Error())
	case errors.Is(err, store.ErrGameFull):
		writeAPIError(w, http.StatusConflict, "game_full", err.Error())
	case errors.Is(err, store.ErrInvalidInvitation), errors.Is(err, store.ErrInvitationInactive):
		writeAPIError(w, http.StatusForbidden, "invalid_invitation", "the invitation is invalid or inactive")
	case errors.Is(err, store.ErrProfileRequired):
		writeAPIError(w, http.StatusBadRequest, "profile_required", err.Error())
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

func (h *GamesHandler) requireCompletedProfile(r *http.Request, actor store.Actor) (store.PlayerProfile, error) {
	if h.profiles == nil {
		return store.PlayerProfile{}, store.ErrProfileRequired
	}
	profile, err := h.profiles.EnsureProfile(r.Context(), actor)
	if err != nil {
		return store.PlayerProfile{}, err
	}
	if strings.TrimSpace(profile.DisplayName) == "" {
		return store.PlayerProfile{}, store.ErrProfileRequired
	}
	return profile, nil
}

func (h *GamesHandler) writeProfileRequirementError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrProfileRequired) {
		writeAPIError(w, http.StatusBadRequest, "profile_required", "complete the player profile before creating or joining a game")
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "profile_unavailable", "profile could not be loaded")
}

func applyProfileToCreateRequest(request store.CreateRequest, profile store.PlayerProfile) store.CreateRequest {
	request.StrictMembership = true
	players := make([]engine.PlayerInit, len(request.Players))
	copy(players, request.Players)
	for index := range players {
		players[index].ID = ""
		players[index].Color = ""
		players[index].Name = ""
	}
	if len(players) > 0 {
		players[0].Name = profile.DisplayName
	}
	request.Players = players
	return request
}

func buildInviteURL(baseURL string, gameID store.GameID, code string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}
	return baseURL + "/join?gameId=" + url.QueryEscape(string(gameID)) + "&inviteCode=" + url.QueryEscape(code)
}

type gameOrdersRequest struct {
	Chains   []chainOrderRequest  `json:"chains"`
	Winter   []winterOrderRequest `json:"winter"`
	Force    bool                 `json:"force,omitempty"`
	Revision store.Revision       `json:"revision,omitempty"`
}

type chainOrderRequest struct {
	Noble models.NobleCode `json:"noble"`
	Text  string           `json:"text"`
}

type winterOrderRequest struct {
	Lines string `json:"lines"`
}

type joinGameRequest struct {
	InviteCode string `json:"inviteCode"`
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
	ID            store.GameID     `json:"id"`
	Name          string           `json:"name"`
	Seed          string           `json:"seed"`
	Status        store.Status     `json:"status"`
	Winner        *models.PlayerID `json:"winner,omitempty"`
	Players       []PlayerSlotView `json:"players"`
	Turn          int              `json:"turn"`
	Season        models.Season    `json:"season"`
	Revision      store.Revision   `json:"revision"`
	CurrentPlayer models.PlayerID  `json:"currentPlayer,omitempty"`
	CanInvite     bool             `json:"canInvite,omitempty"`
	InviteCode    string           `json:"inviteCode,omitempty"`
	InviteURL     string           `json:"inviteUrl,omitempty"`
}

type PlayerSlotView struct {
	ID        models.PlayerID `json:"id"`
	Name      string          `json:"name"`
	Color     string          `json:"color"`
	Submitted bool            `json:"submitted"`
}

type invitationView struct {
	GameID     store.GameID `json:"gameId"`
	InviteCode string       `json:"inviteCode"`
	InviteURL  string       `json:"inviteUrl"`
}

type gameJoinResponse struct {
	gameDetailView
	Player PlayerSlotView `json:"player"`
	Joined bool           `json:"joined"`
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

func makeAuthenticatedGameDetailView(snapshot store.GameSnapshot, actor store.Actor) gameDetailView {
	view := makeGameDetailView(snapshot)
	view.CurrentPlayer, _ = snapshotPlayerID(snapshot, actor)
	view.CanInvite = strings.TrimSpace(snapshot.CreatedBy) != "" &&
		strings.TrimSpace(snapshot.CreatedBy) == strings.TrimSpace(actor.ID)
	return view
}

func makePlayerSlotViews(snapshot store.GameSnapshot) []PlayerSlotView {
	views := make([]PlayerSlotView, len(snapshot.Players))
	for index, player := range snapshot.Players {
		_, submitted := snapshot.Submissions[player.ID]
		views[index] = PlayerSlotView{ID: player.ID, Name: player.Name, Color: player.Color, Submitted: submitted}
	}
	return views
}

func makePlayerSlotView(player store.PlayerSlot, snapshot store.GameSnapshot) PlayerSlotView {
	_, submitted := snapshot.Submissions[player.ID]
	return PlayerSlotView{ID: player.ID, Name: player.Name, Color: player.Color, Submitted: submitted}
}

func snapshotPlayerID(snapshot store.GameSnapshot, actor store.Actor) (models.PlayerID, bool) {
	actorID := strings.TrimSpace(actor.ID)
	for _, player := range snapshot.Players {
		if player.ActorID == actorID {
			return player.ID, true
		}
		if actor.Development && player.ActorID == "" && string(player.ID) == actorID {
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
