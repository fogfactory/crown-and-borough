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
	balance          assetgen.Balance
	actor            ActorResolver
	profiles         store.ProfileStore
	requireProfile   bool
	strictMembership bool
	inviteBaseURL    string
	creatorGate      CreatorGate
	mux              *http.ServeMux
}

type GamesHandlerOptions struct {
	Actor            ActorResolver
	Balance          assetgen.Balance
	Profiles         store.ProfileStore
	RequireProfile   bool
	StrictMembership bool
	InviteBaseURL    string
	CreatorGate      CreatorGate
}

func NewGamesHandler(gameStore store.GameStore, rules assetgen.Rules, resolve ActorResolver) http.Handler {
	return NewGamesHandlerWithOptions(gameStore, rules, GamesHandlerOptions{Actor: resolve})
}

func NewGamesHandlerWithOptions(gameStore store.GameStore, rules assetgen.Rules, options GamesHandlerOptions) http.Handler {
	profiles := options.Profiles
	if profiles == nil {
		profiles, _ = gameStore.(store.ProfileStore)
	}
	creatorGate := options.CreatorGate
	if creatorGate == nil {
		creatorGate = AllowAllCreatorGate{}
	}
	h := &GamesHandler{
		store:            gameStore,
		rules:            rules,
		balance:          options.Balance,
		actor:            options.Actor,
		profiles:         profiles,
		requireProfile:   options.RequireProfile,
		strictMembership: options.StrictMembership,
		inviteBaseURL:    options.InviteBaseURL,
		creatorGate:      creatorGate,
	}
	h.mux = h.routes()
	return h
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
	h.mux.ServeHTTP(w, r)
}

// gameResourceHandler is a route handler that already has its actor and game
// ID resolved; see withActor.
type gameResourceHandler func(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID)

// withActor resolves the acting identity and the {id} path value once, so
// individual routes only deal with their own resource logic. On failure it
// has already written the error response.
func (h *GamesHandler) withActor(next gameResourceHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := h.resolveActor(w, r)
		if !ok {
			return
		}
		next(w, r, actor, store.GameID(r.PathValue("id")))
	}
}

// routes wires the games API surface on a dedicated mux, using method- and
// wildcard-qualified patterns so unmatched methods and paths fall back to the
// standard library's own 404/405 handling.
func (h *GamesHandler) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/games", h.list)
	mux.HandleFunc("POST /api/games", h.create)
	mux.HandleFunc("GET /api/games/{id}", h.withActor(h.detail))
	mux.HandleFunc("GET /api/games/{id}/map", h.withActor(h.getMap))
	mux.HandleFunc("GET /api/games/{id}/state", h.withActor(h.getState))
	mux.HandleFunc("GET /api/games/{id}/balance", h.withActor(h.getBalance))
	mux.HandleFunc("GET /api/games/{id}/supply", h.withActor(h.getSupply))
	mux.HandleFunc("POST /api/games/{id}/orders", h.withActor(h.submit))
	mux.HandleFunc("POST /api/games/{id}/orders/preview", h.withActor(h.preview))
	mux.HandleFunc("GET /api/games/{id}/my-submission", h.withActor(h.mySubmission))
	mux.HandleFunc("GET /api/games/{id}/submitted-orders", h.withActor(h.submittedOrders))
	mux.HandleFunc("POST /api/games/{id}/join", h.withActor(h.join))
	mux.HandleFunc("GET /api/games/{id}/invite", h.withActor(h.invite))
	mux.HandleFunc("POST /api/games/{id}/resolve", h.withActor(h.resolve))
	mux.HandleFunc("GET /api/games/{id}/reports", h.withActor(h.reportsList))
	mux.HandleFunc("GET /api/games/{id}/reports/{index}", h.withActor(h.report))
	mux.HandleFunc("GET /api/games/{id}/rules", h.withActor(h.getRules))
	return mux
}

func (h *GamesHandler) list(w http.ResponseWriter, r *http.Request) {
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
		response = append(response, makeGameListView(game, actor))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *GamesHandler) create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.resolveActor(w, r)
	if !ok {
		return
	}
	if !h.creatorGate.Allowed(actor) {
		writeAPIError(w, http.StatusForbidden, "creator_not_allowed", "only authorized accounts can create games")
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

func (h *GamesHandler) detail(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	snapshot, err := h.store.Get(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, makeAuthenticatedGameDetailView(snapshot, actor))
}

func (h *GamesHandler) getMap(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	mapData, err := h.store.Map(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapData)
}

func (h *GamesHandler) getState(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	snapshot, err := h.store.State(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	viewerID, ok := snapshot.ViewerFor(actor)
	if !ok {
		writeAPIError(w, http.StatusForbidden, "not_member", "actor is not a member of this game")
		return
	}
	writeGameState(w, snapshot.Revision, projectStateForPlayer(snapshot.State, viewerID))
}

func (h *GamesHandler) getBalance(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	if _, err := h.store.Get(r.Context(), actor, id); err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, winterCostsView(h.balance))
}

func (h *GamesHandler) getSupply(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	territory := models.TerritoryID(r.URL.Query().Get("territory"))
	if territory == "" {
		writeAPIError(w, http.StatusBadRequest, "territory_required", "a territory is required")
		return
	}
	if target := models.TerritoryID(r.URL.Query().Get("target")); target != "" {
		transferStore, ok := h.store.(store.TransferSupplyStore)
		if !ok {
			writeAPIError(w, http.StatusInternalServerError, "supply_failed", "transfer overlay is unavailable")
			return
		}
		line, err := transferStore.TransferSupply(r.Context(), actor, id, territory, target)
		if err != nil {
			h.writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, line)
		return
	}
	// special carries the viewer's drafted deck orders, so the projection
	// reflects the cards they intend to play this turn.
	line, err := h.store.Supply(r.Context(), actor, id, territory, r.URL.Query().Get("special"))
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, line)
}

func (h *GamesHandler) getRules(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	if _, err := h.store.Get(r.Context(), actor, id); err != nil {
		h.writeStoreError(w, err)
		return
	}
	h.serveRules(w, r)
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
		Special:          toDeckSubmissions(request.Special),
		Force:            request.Force,
		ExpectedRevision: expectedRevision,
	})
	var inputErrors *engine.InputErrors
	if errors.As(err, &inputErrors) {
		writeResolutionError(w, err, i18n.FromRequest(r))
		return
	}
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

func (h *GamesHandler) mySubmission(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	submission, err := h.store.MySubmission(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	chains := make([]chainSubmissionView, len(submission.Orders.Chains))
	for index, chain := range submission.Orders.Chains {
		chains[index] = chainSubmissionView{
			Noble: chain.Noble,
			Text:  chain.Text,
		}
	}
	var winter *winterSubmissionView
	if len(submission.Orders.Winter) > 0 {
		winter = &winterSubmissionView{
			Lines: submission.Orders.Winter[0].Lines,
		}
	}
	writeJSON(w, http.StatusOK, mySubmissionView{
		Turn:      submission.Turn,
		Season:    submission.Season,
		Submitted: submission.Submitted,
		Chains:    chains,
		Winter:    winter,
	})
}

func (h *GamesHandler) submittedOrders(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	snapshot, err := h.store.Get(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	if !store.IsSpectator(snapshot.SpectatorUID, actor) {
		writeAPIError(w, http.StatusForbidden, "spectator_only", "submitted orders are visible only to the observer host")
		return
	}
	response := submittedOrdersView{
		Turn:        snapshot.State.Turn,
		Season:      snapshot.State.Season,
		Submissions: make([]submittedPlayerOrdersView, 0, len(snapshot.Submissions)),
	}
	for _, player := range snapshot.Players {
		input, submitted := snapshot.Submissions[player.ID]
		if !submitted {
			continue
		}
		chains := make([]chainSubmissionView, len(input.Chains))
		for index, chain := range input.Chains {
			chains[index] = chainSubmissionView{Noble: chain.Noble, Text: chain.Text}
		}
		var winter *winterSubmissionView
		if len(input.Winter) > 0 {
			winter = &winterSubmissionView{Lines: input.Winter[0].Lines}
		}
		preview, err := previewView(snapshot.State, h.balance, player.ID, input, i18n.FromRequest(r))
		if err != nil {
			h.writeStoreError(w, err)
			return
		}
		response.Submissions = append(response.Submissions, submittedPlayerOrdersView{
			Player:  player.ID,
			Chains:  chains,
			Winter:  winter,
			Preview: preview,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func toDeckSubmissions(requests []deckOrderRequest) []engine.DeckSubmission {
	deck := make([]engine.DeckSubmission, len(requests))
	for index, request := range requests {
		deck[index] = engine.DeckSubmission{Text: request.Text}
	}
	return deck
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
	if strings.TrimSpace(snapshot.CreatedBy) != strings.TrimSpace(actor.ID) {
		h.writeStoreError(w, store.ErrNotCreator)
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

func (h *GamesHandler) reportsList(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
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
}

func (h *GamesHandler) report(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || index < 0 {
		writeAPIError(w, http.StatusNotFound, "report_not_found", "report index not found")
		return
	}
	record, err := h.store.Report(r.Context(), actor, id, index)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	playerID, ok := h.viewerIDForGame(r.Context(), actor, id)
	if !ok {
		writeAPIError(w, http.StatusForbidden, "not_member", "actor is not a member of this game")
		return
	}
	writeJSON(w, http.StatusOK, projectReport(record.Report, playerID, record.Privacy))
}

func (h *GamesHandler) writeSubmitResult(w http.ResponseWriter, actor store.Actor, result store.SubmitResult) {
	playerID, ok := result.Snapshot.PlayerFor(actor)
	viewerID := playerID
	if !ok && store.IsSpectator(result.Snapshot.SpectatorUID, actor) {
		ok = true
		viewerID = models.SpectatorViewer
	}
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
		State:     projectStateForPlayer(result.Snapshot.State, viewerID),
	}
	if result.Report != nil {
		report := projectReport(result.Report.Report, viewerID, result.Report.Privacy)
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

func (h *GamesHandler) viewerIDForGame(ctx context.Context, actor store.Actor, id store.GameID) (models.PlayerID, bool) {
	snapshot, err := h.store.Get(ctx, actor, id)
	if err != nil {
		return "", false
	}
	return snapshot.ViewerFor(actor)
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
	case errors.Is(err, store.ErrInvalidYears):
		writeAPIError(w, http.StatusBadRequest, "invalid_years", err.Error())
	case errors.Is(err, store.ErrGameFull):
		writeAPIError(w, http.StatusConflict, "game_full", err.Error())
	case errors.Is(err, store.ErrInvalidInvitation), errors.Is(err, store.ErrInvitationInactive):
		writeAPIError(w, http.StatusForbidden, "invalid_invitation", "the invitation is invalid or inactive")
	case errors.Is(err, store.ErrSpectator):
		writeAPIError(w, http.StatusConflict, "spectator_only", "the game creator is an observer and cannot join a player slot")
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
	if len(players) > 0 && !request.Spectate {
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
	Special  []deckOrderRequest   `json:"special"`
	Force    bool                 `json:"force,omitempty"`
	Revision store.Revision       `json:"revision,omitempty"`
}

type deckOrderRequest struct {
	Text string `json:"text"`
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

type mySubmissionView struct {
	Turn      int                   `json:"turn"`
	Season    models.Season         `json:"season"`
	Submitted bool                  `json:"submitted"`
	Chains    []chainSubmissionView `json:"chains"`
	Winter    *winterSubmissionView `json:"winter,omitempty"`
}

type submittedOrdersView struct {
	Turn        int                         `json:"turn"`
	Season      models.Season               `json:"season"`
	Submissions []submittedPlayerOrdersView `json:"submissions"`
}

type submittedPlayerOrdersView struct {
	Player  models.PlayerID       `json:"player"`
	Chains  []chainSubmissionView `json:"chains"`
	Winter  *winterSubmissionView `json:"winter,omitempty"`
	Preview OrdersPreviewView     `json:"preview"`
}

type chainSubmissionView struct {
	Noble models.NobleCode `json:"noble"`
	Text  string           `json:"text"`
}

type winterSubmissionView struct {
	Lines string `json:"lines"`
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
	ID        store.GameID                              `json:"id"`
	Name      string                                    `json:"name"`
	Seed      string                                    `json:"seed"`
	YearCount int                                       `json:"yearCount"`
	Scores    map[models.PlayerID]engine.ScoreBreakdown `json:"scores"`
	Status    store.Status                              `json:"status"`
	Players   []PlayerSlotView                          `json:"players"`
	Turn      int                                       `json:"turn"`
	Season    models.Season                             `json:"season"`
	Revision  store.Revision                            `json:"revision"`
	Spectator bool                                      `json:"spectator"`
}

type gameDetailView struct {
	ID              store.GameID                              `json:"id"`
	Name            string                                    `json:"name"`
	Seed            string                                    `json:"seed"`
	YearCount       int                                       `json:"yearCount"`
	Scores          map[models.PlayerID]engine.ScoreBreakdown `json:"scores"`
	Status          store.Status                              `json:"status"`
	Winner          *models.PlayerID                          `json:"winner,omitempty"`
	Players         []PlayerSlotView                          `json:"players"`
	Turn            int                                       `json:"turn"`
	Season          models.Season                             `json:"season"`
	Revision        store.Revision                            `json:"revision"`
	CurrentPlayer   models.PlayerID                           `json:"currentPlayer,omitempty"`
	CanInvite       bool                                      `json:"canInvite,omitempty"`
	InviteAvailable bool                                      `json:"inviteAvailable"`
	InviteCode      string                                    `json:"inviteCode,omitempty"`
	InviteURL       string                                    `json:"inviteUrl,omitempty"`
	Spectator       bool                                      `json:"spectator"`
}

type PlayerSlotView struct {
	ID        models.PlayerID `json:"id"`
	Name      string          `json:"name"`
	Color     string          `json:"color"`
	Submitted bool            `json:"submitted"`
	Required  bool            `json:"required"`
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

func makeGameListView(snapshot store.GameSnapshot, actor store.Actor) gameListView {
	return gameListView{
		ID:        snapshot.ID,
		Name:      snapshot.Name,
		Seed:      snapshot.Seed,
		YearCount: snapshot.YearCount,
		Scores:    snapshot.Scores,
		Status:    snapshot.Status,
		Players:   makePlayerSlotViews(snapshot),
		Turn:      snapshot.State.Turn,
		Season:    snapshot.State.Season,
		Revision:  snapshot.Revision,
		Spectator: strings.TrimSpace(snapshot.SpectatorUID) != "" &&
			strings.TrimSpace(snapshot.SpectatorUID) == strings.TrimSpace(actor.ID),
	}
}

func makeGameDetailView(snapshot store.GameSnapshot) gameDetailView {
	return gameDetailView{
		ID:        snapshot.ID,
		Name:      snapshot.Name,
		Seed:      snapshot.Seed,
		YearCount: snapshot.YearCount,
		Scores:    snapshot.Scores,
		Status:    snapshot.Status,
		Winner:    snapshot.Winner,
		Players:   makePlayerSlotViews(snapshot),
		Turn:      snapshot.State.Turn,
		Season:    snapshot.State.Season,
		Revision:  snapshot.Revision,
	}
}

func makeAuthenticatedGameDetailView(snapshot store.GameSnapshot, actor store.Actor) gameDetailView {
	view := makeGameDetailView(snapshot)
	view.CurrentPlayer, _ = snapshot.PlayerFor(actor)
	view.CanInvite = strings.TrimSpace(snapshot.CreatedBy) != "" &&
		strings.TrimSpace(snapshot.CreatedBy) == strings.TrimSpace(actor.ID)
	view.InviteAvailable = view.CanInvite && hasFreePlayerSlot(snapshot.Players)
	view.Spectator = store.IsSpectator(snapshot.SpectatorUID, actor)
	return view
}

func hasFreePlayerSlot(players []store.PlayerSlot) bool {
	for _, player := range players {
		if strings.TrimSpace(player.ActorID) == "" || strings.HasPrefix(player.ActorID, "slot:") {
			return true
		}
	}
	return false
}

func makePlayerSlotViews(snapshot store.GameSnapshot) []PlayerSlotView {
	views := make([]PlayerSlotView, len(snapshot.Players))
	for index, player := range snapshot.Players {
		views[index] = makePlayerSlotView(player, snapshot)
	}
	return views
}

func makePlayerSlotView(player store.PlayerSlot, snapshot store.GameSnapshot) PlayerSlotView {
	_, submitted := snapshot.Submissions[player.ID]
	return PlayerSlotView{
		ID:        player.ID,
		Name:      player.Name,
		Color:     player.Color,
		Submitted: submitted,
		Required:  engine.PlayerMustSubmit(snapshot.State, player.ID),
	}
}

type createGameBody struct {
	Name     string          `json:"name"`
	Seed     string          `json:"seed"`
	Years    int             `json:"years"`
	Spectate bool            `json:"spectate,omitempty"`
	Players  json.RawMessage `json:"players"`
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
	return store.CreateRequest{Name: body.Name, Seed: body.Seed, Players: players, YearCount: body.Years, Spectate: body.Spectate}, nil
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
