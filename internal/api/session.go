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

// OrdersHTTP resolves one complete hotseat turn and returns both report and
// projected state.
func (s *Session) OrdersHTTP(w http.ResponseWriter, r *http.Request) {
	var input engine.OrdersInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_orders_request", err.Error())
		return
	}

	s.mu.Lock()
	report, err := engine.ResolveTurn(s.game, s.balance, input)
	if err == nil {
		s.game = report.State
	}
	state := projectState(s.game, nil)
	s.mu.Unlock()
	if err != nil {
		writeResolutionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Report engine.TurnReport `json:"report"`
		State  StateView         `json:"state"`
	}{Report: report, State: state})
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
