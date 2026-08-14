package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/i18n"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Session owns the single in-memory game used by the development hotseat
// server. The mutex covers both the model and its matching static map.
type Session struct {
	mu sync.RWMutex

	game    *models.GameState
	mapData mapgen.MapData
	assets  assetgen.Assets
	balance assetgen.Balance

	defaultSeed    string
	defaultPlayers []engine.PlayerInit
	pending        map[models.PlayerID]engine.OrdersInput
}

// GameSession is an explicit alias for callers that prefer the longer name.
type GameSession = Session

// NewSession creates and stores the default hotseat game.
func NewSession(seed string, players []engine.PlayerInit, balance assetgen.Balance, assets assetgen.Assets) (*Session, error) {
	if seed == "" {
		return nil, fmt.Errorf("api: session seed must not be empty")
	}
	players = clonePlayerInits(players)
	if len(players) == 0 {
		return nil, fmt.Errorf("api: session requires at least one player")
	}
	session := &Session{
		assets:         assets,
		balance:        balance,
		defaultSeed:    seed,
		defaultPlayers: players,
	}
	if err := session.replaceGame(seed, players); err != nil {
		return nil, err
	}
	return session, nil
}

// Create replaces the current game while keeping the startup game as the
// target of Reset.
func (s *Session) Create(seed string, players []engine.PlayerInit) error {
	if seed == "" {
		return fmt.Errorf("api: game seed must not be empty")
	}
	return s.replaceGame(seed, clonePlayerInits(players))
}

// Reset restores the game configured at server startup.
func (s *Session) Reset() error {
	return s.replaceGame(s.defaultSeed, clonePlayerInits(s.defaultPlayers))
}

func (s *Session) replaceGame(seed string, players []engine.PlayerInit) error {
	game, err := engine.CreateGame(seed, players, s.balance, s.assets)
	if err != nil {
		return err
	}
	mapData, err := engine.GenerateMap(seed, len(players), s.assets)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.game = game
	s.mapData = mapData
	s.pending = make(map[models.PlayerID]engine.OrdersInput)
	s.mu.Unlock()
	return nil
}

// MapHTTP serves the immutable map belonging to the current in-memory game.
func (s *Session) MapHTTP(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	mapData := s.mapData
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, mapData)
}

// StateHTTP serves the current state projection. The optional player query is
// a development-only identity selector for hotseat private-view checks.
func (s *Session) StateHTTP(w http.ResponseWriter, r *http.Request) {
	viewer, filtered, err := requestedViewer(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_player", err.Error())
		return
	}
	s.mu.RLock()
	if filtered && !s.hasPlayerLocked(viewer) {
		s.mu.RUnlock()
		writeAPIError(w, http.StatusBadRequest, "unknown_player", fmt.Sprintf("unknown player %q", viewer))
		return
	}
	state := projectState(s.game)
	if filtered {
		state = projectStateForPlayer(s.game, viewer)
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, state)
}

// SupplyHTTP serves the current supply assignment for an army, or the
// reachable zone of a controlled source selected without an army. The
// calculation is read-only and follows the next resolution's supply rules.
func (s *Session) SupplyHTTP(w http.ResponseWriter, r *http.Request) {
	territoryID := models.TerritoryID(r.URL.Query().Get("territory"))
	if territoryID == "" {
		writeAPIError(w, http.StatusBadRequest, "territory_required", "a territory is required")
		return
	}

	s.mu.RLock()
	line, err := engine.FindSupply(s.game, s.balance, territoryID)
	s.mu.RUnlock()
	if err != nil {
		switch {
		case errors.Is(err, engine.ErrSupplyLineWinter):
			writeAPIError(w, http.StatusConflict, "supply_unavailable", err.Error())
		case errors.Is(err, engine.ErrSupplyLineUnknownTerritory),
			errors.Is(err, engine.ErrSupplyLineNoSource):
			writeAPIError(w, http.StatusNotFound, "supply_target_not_found", err.Error())
		default:
			writeAPIError(w, http.StatusInternalServerError, "supply_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, line)
}

// GameHTTP creates a new in-memory game from a seed and player list.
func (s *Session) GameHTTP(w http.ResponseWriter, r *http.Request) {
	request, err := decodeGameRequest(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_game_request", err.Error())
		return
	}
	if request.Seed == "" {
		request.Seed = s.defaultSeed
	}
	if len(request.Players) == 0 {
		request.Players = clonePlayerInits(s.defaultPlayers)
	}
	if err := s.Create(request.Seed, request.Players); err != nil {
		writeAPIError(w, http.StatusBadRequest, "game_creation_failed", err.Error())
		return
	}
	s.writeGameResponse(w)
}

// ResetHTTP restores the default game.
func (s *Session) ResetHTTP(w http.ResponseWriter, _ *http.Request) {
	if err := s.Reset(); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "reset_failed", err.Error())
		return
	}
	s.writeGameResponse(w)
}

// OrdersHTTP records one player's orders. Action turns resolve once every
// player with an emission-capable noble has submitted; winter still waits for
// every player.
// Submitting again replaces that player's pending orders.
func (s *Session) OrdersHTTP(w http.ResponseWriter, r *http.Request) {
	language := i18n.FromRequest(r)
	viewer, filtered, viewerErr := requestedViewer(r)
	if viewerErr != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_player", viewerErr.Error())
		return
	}
	var request ordersRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_orders_request", err.Error())
		return
	}

	if request.Player == "" {
		writeResolutionError(w, &engine.InputErrors{Errors: []engine.InputError{{
			Code: "player_required", Message: i18n.EnglishText(i18n.Message{Key: i18n.ErrorPlayerRequired}), MessageKey: i18n.ErrorPlayerRequired,
		}}}, language)
		return
	}

	s.mu.Lock()
	if !s.hasPlayerLocked(request.Player) {
		s.mu.Unlock()
		writeResolutionError(w, &engine.InputErrors{Errors: []engine.InputError{{
			Player: request.Player, Code: "unknown_player",
			Message:    i18n.EnglishText(i18n.Message{Key: i18n.ErrorUnknownPlayer, Args: []any{request.Player}}),
			MessageKey: i18n.ErrorUnknownPlayer, MessageArgs: []any{request.Player},
		}}}, language)
		return
	}
	if filtered && !s.hasPlayerLocked(viewer) {
		s.mu.Unlock()
		writeAPIError(w, http.StatusBadRequest, "unknown_player", fmt.Sprintf("unknown player %q", viewer))
		return
	}
	if filtered && viewer != request.Player {
		s.mu.Unlock()
		writeAPIError(w, http.StatusBadRequest, "player_mismatch", "query player must match the submitted player")
		return
	}

	input, inputErr := normalizePlayerOrders(request.Player, request.Chains, request.Winter)
	if inputErr != nil {
		s.mu.Unlock()
		writeResolutionError(w, inputErr, language)
		return
	}
	if _, err := engine.ResolveTurn(s.game, s.balance, input); err != nil {
		s.mu.Unlock()
		writeResolutionError(w, err, language)
		return
	}
	if s.pending == nil {
		s.pending = make(map[models.PlayerID]engine.OrdersInput)
	}
	s.pending[request.Player] = input
	submitted, remaining := s.pendingPlayersLocked()
	if len(remaining) != 0 && !request.Force {
		state := projectState(s.game)
		if filtered {
			state = projectStateForPlayer(s.game, viewer)
		}
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, ordersResponse{
			Status: "pending", Player: request.Player, Submitted: submitted, Remaining: remaining, State: state,
		})
		return
	}

	combined := engine.OrdersInput{Chains: []engine.ChainSubmission{}, Winter: []engine.WinterSubmission{}}
	for _, player := range s.game.Players {
		playerOrders := s.pending[player.ID]
		combined.Chains = append(combined.Chains, playerOrders.Chains...)
		combined.Winter = append(combined.Winter, playerOrders.Winter...)
	}
	before := s.game
	report, err := engine.ResolveTurn(before, s.balance, combined)
	if err != nil {
		s.mu.Unlock()
		writeResolutionError(w, err, language)
		return
	}
	trackTurnPrivacy(before, report.State, combined, report)
	s.game = report.State
	s.pending = make(map[models.PlayerID]engine.OrdersInput)
	state := projectState(s.game)
	if filtered {
		state = projectStateForPlayer(s.game, viewer)
	}
	var reportView any = report
	if filtered {
		reportView = projectReport(report, viewer, s.game.Privacy)
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, ordersResponse{
		Status: "resolved", Player: request.Player, Submitted: submitted, State: state, Report: reportView,
	})
}

type ordersRequest struct {
	Player models.PlayerID           `json:"player,omitempty"`
	Chains []engine.ChainSubmission  `json:"chains"`
	Winter []engine.WinterSubmission `json:"winter"`
	Force  bool                      `json:"force,omitempty"`
}

type ordersResponse struct {
	Status    string            `json:"status"`
	Player    models.PlayerID   `json:"player,omitempty"`
	Submitted []models.PlayerID `json:"submitted"`
	Remaining []models.PlayerID `json:"remaining"`
	Report    any               `json:"report,omitempty"`
	State     StateView         `json:"state"`
}

func (s *Session) hasPlayerLocked(playerID models.PlayerID) bool {
	for _, player := range s.game.Players {
		if player.ID == playerID {
			return true
		}
	}
	return false
}

func (s *Session) pendingPlayersLocked() ([]models.PlayerID, []models.PlayerID) {
	submitted := make([]models.PlayerID, 0, len(s.pending))
	remaining := make([]models.PlayerID, 0, len(s.game.Players))
	for _, player := range s.game.Players {
		if _, exists := s.pending[player.ID]; exists {
			submitted = append(submitted, player.ID)
		} else {
			if s.game.Season != models.SeasonWinter && !s.hasEmittingNobleLocked(player.ID) {
				continue
			}
			remaining = append(remaining, player.ID)
		}
	}
	return submitted, remaining
}

func (s *Session) hasEmittingNobleLocked(playerID models.PlayerID) bool {
	for _, noble := range s.game.Nobles {
		if noble.OwnerID == playerID && noble.Status != models.NobleStatusDungeon {
			return true
		}
	}
	return false
}

func normalizePlayerOrders(playerID models.PlayerID, chains []engine.ChainSubmission, winter []engine.WinterSubmission) (engine.OrdersInput, *engine.InputErrors) {
	input := engine.OrdersInput{
		Chains: append([]engine.ChainSubmission(nil), chains...),
		Winter: append([]engine.WinterSubmission(nil), winter...),
	}
	inputErrors := &engine.InputErrors{Errors: []engine.InputError{}}
	for index := range input.Chains {
		if input.Chains[index].Player != "" && input.Chains[index].Player != playerID {
			inputErrors.Errors = append(inputErrors.Errors, engine.InputError{
				Player: playerID, Noble: input.Chains[index].Noble, Code: "foreign_player_order",
				Message:    i18n.EnglishText(i18n.Message{Key: i18n.ErrorForeignChain, Args: []any{index + 1, input.Chains[index].Player}}),
				MessageKey: i18n.ErrorForeignChain, MessageArgs: []any{index + 1, input.Chains[index].Player},
			})
			continue
		}
		input.Chains[index].Player = playerID
	}
	for index := range input.Winter {
		if input.Winter[index].Player != "" && input.Winter[index].Player != playerID {
			inputErrors.Errors = append(inputErrors.Errors, engine.InputError{
				Player: playerID, Code: "foreign_player_order",
				Message:    i18n.EnglishText(i18n.Message{Key: i18n.ErrorForeignWinter, Args: []any{index + 1, input.Winter[index].Player}}),
				MessageKey: i18n.ErrorForeignWinter, MessageArgs: []any{index + 1, input.Winter[index].Player},
			})
			continue
		}
		input.Winter[index].Player = playerID
	}
	if len(inputErrors.Errors) != 0 {
		return engine.OrdersInput{}, inputErrors
	}
	return input, nil
}

func (s *Session) writeGameResponse(w http.ResponseWriter) {
	s.mu.RLock()
	response := struct {
		Map   mapgen.MapData `json:"map"`
		State StateView      `json:"state"`
	}{Map: s.mapData, State: projectState(s.game)}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, response)
}

type gameRequest struct {
	Seed    string
	Players []engine.PlayerInit
}

func decodeGameRequest(r *http.Request) (gameRequest, error) {
	var raw struct {
		Seed    string          `json:"seed"`
		Players json.RawMessage `json:"players"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return gameRequest{}, err
	}
	players, err := decodePlayers(raw.Players)
	if err != nil {
		return gameRequest{}, err
	}
	return gameRequest{Seed: raw.Seed, Players: players}, nil
}

func decodePlayers(raw json.RawMessage) ([]engine.PlayerInit, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var players []engine.PlayerInit
	if err := json.Unmarshal(raw, &players); err == nil {
		return players, nil
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err == nil {
		players = make([]engine.PlayerInit, len(names))
		for index, name := range names {
			players[index] = engine.PlayerInit{Name: name}
		}
		return players, nil
	}
	var count int
	if err := json.Unmarshal(raw, &count); err == nil {
		if count < 1 {
			return nil, fmt.Errorf("players must be a positive integer")
		}
		players = make([]engine.PlayerInit, count)
		return players, nil
	}
	return nil, fmt.Errorf("players must be an array of player objects, names, or a count")
}

func writeResolutionError(w http.ResponseWriter, err error, language i18n.Language) {
	var inputErrors *engine.InputErrors
	if errors.As(err, &inputErrors) {
		localized := engine.InputErrors{Errors: append([]engine.InputError(nil), inputErrors.Errors...)}
		for index := range localized.Errors {
			inputError := &localized.Errors[index]
			if inputError.MessageKey != "" {
				inputError.Message = i18n.Translate(language, i18n.Message{Key: inputError.MessageKey, Args: inputError.MessageArgs})
			}
		}
		message := i18n.Translate(language, i18n.Message{Key: i18n.ErrorPlayerRequired})
		if len(localized.Errors) > 0 {
			message = localized.Errors[0].Message
		}
		writeJSON(w, http.StatusBadRequest, struct {
			Error   string              `json:"error"`
			Message string              `json:"message"`
			Errors  []engine.InputError `json:"errors"`
		}{Error: "invalid_orders", Message: message, Errors: localized.Errors})
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "resolution_failed", err.Error())
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}{Error: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func clonePlayerInits(source []engine.PlayerInit) []engine.PlayerInit {
	if source == nil {
		return nil
	}
	clone := make([]engine.PlayerInit, len(source))
	copy(clone, source)
	return clone
}

// ParsePlayerCount parses the PLAYERS environment value while keeping the
// allowed range in the engine's game creation validation.
func ParsePlayerCount(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	count, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("players must be an integer: %w", err)
	}
	return count, nil
}
