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

// StateHTTP serves the current global T0 state projection.
func (s *Session) StateHTTP(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	state := projectState(s.game, nil)
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, state)
}

// SupplyHTTP serves the current supply assignment for the army at one
// territory. The calculation is read-only and follows the next resolution's
// supply rules.
func (s *Session) SupplyHTTP(w http.ResponseWriter, r *http.Request) {
	territoryID := models.TerritoryID(r.URL.Query().Get("territory"))
	if territoryID == "" {
		writeAPIError(w, http.StatusBadRequest, "territory_required", "a territory is required")
		return
	}

	s.mu.RLock()
	line, err := engine.FindSupplyLine(s.game, s.balance, territoryID)
	s.mu.RUnlock()
	if err != nil {
		switch {
		case errors.Is(err, engine.ErrSupplyLineWinter):
			writeAPIError(w, http.StatusConflict, "supply_unavailable", err.Error())
		case errors.Is(err, engine.ErrSupplyLineUnknownTerritory), errors.Is(err, engine.ErrSupplyLineNoArmy):
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

// OrdersHTTP records one player's orders. The turn is resolved only when every
// player has submitted once for the current turn; submitting again replaces
// that player's pending orders.
func (s *Session) OrdersHTTP(w http.ResponseWriter, r *http.Request) {
	var request ordersRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_orders_request", err.Error())
		return
	}

	if request.Player == "" {
		writeResolutionError(w, &engine.InputErrors{Errors: []engine.InputError{{
			Code: "player_required", Message: "one player's orders must be submitted at a time",
		}}})
		return
	}

	s.mu.Lock()
	if !s.hasPlayerLocked(request.Player) {
		s.mu.Unlock()
		writeResolutionError(w, &engine.InputErrors{Errors: []engine.InputError{{
			Player: request.Player, Code: "unknown_player",
			Message: fmt.Sprintf("player %q does not exist", request.Player),
		}}})
		return
	}

	input, inputErr := normalizePlayerOrders(request.Player, request.Chains, request.Winter)
	if inputErr != nil {
		s.mu.Unlock()
		writeResolutionError(w, inputErr)
		return
	}
	if _, err := engine.ResolveTurn(s.game, s.balance, input); err != nil {
		s.mu.Unlock()
		writeResolutionError(w, err)
		return
	}
	if s.pending == nil {
		s.pending = make(map[models.PlayerID]engine.OrdersInput)
	}
	s.pending[request.Player] = input
	submitted, remaining := s.pendingPlayersLocked()
	if len(remaining) != 0 && !request.Force {
		state := projectState(s.game, nil)
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
	report, err := engine.ResolveTurn(s.game, s.balance, combined)
	if err != nil {
		s.mu.Unlock()
		writeResolutionError(w, err)
		return
	}
	s.game = report.State
	s.pending = make(map[models.PlayerID]engine.OrdersInput)
	state := projectState(s.game, nil)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, ordersResponse{
		Status: "resolved", Player: request.Player, Submitted: submitted, State: state, Report: &report,
	})
}

type ordersRequest struct {
	Player models.PlayerID           `json:"player,omitempty"`
	Chains []engine.ChainSubmission  `json:"chains"`
	Winter []engine.WinterSubmission `json:"winter"`
	Force  bool                      `json:"force,omitempty"`
}

type ordersResponse struct {
	Status    string             `json:"status"`
	Player    models.PlayerID    `json:"player,omitempty"`
	Submitted []models.PlayerID  `json:"submitted"`
	Remaining []models.PlayerID  `json:"remaining"`
	Report    *engine.TurnReport `json:"report,omitempty"`
	State     StateView          `json:"state"`
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
			remaining = append(remaining, player.ID)
		}
	}
	return submitted, remaining
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
				Message: fmt.Sprintf("chain %d belongs to player %q", index+1, input.Chains[index].Player),
			})
			continue
		}
		input.Chains[index].Player = playerID
	}
	for index := range input.Winter {
		if input.Winter[index].Player != "" && input.Winter[index].Player != playerID {
			inputErrors.Errors = append(inputErrors.Errors, engine.InputError{
				Player: playerID, Code: "foreign_player_order",
				Message: fmt.Sprintf("winter submission %d belongs to player %q", index+1, input.Winter[index].Player),
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
	}{Map: s.mapData, State: projectState(s.game, nil)}
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

func writeResolutionError(w http.ResponseWriter, err error) {
	var inputErrors *engine.InputErrors
	if errors.As(err, &inputErrors) {
		message := "invalid order submission"
		if len(inputErrors.Errors) > 0 {
			message = inputErrors.Errors[0].Message
		}
		writeJSON(w, http.StatusBadRequest, struct {
			Error   string              `json:"error"`
			Message string              `json:"message"`
			Errors  []engine.InputError `json:"errors"`
		}{Error: "invalid_orders", Message: message, Errors: inputErrors.Errors})
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
