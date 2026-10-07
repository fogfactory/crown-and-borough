package api

import (
	"encoding/json"
	"net/http"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

// simulationActionRequest is one hypothetical change: kill (noble), marry or
// divorce (noble, other), claim (noble is the heir, other the target). Nobles
// are given by code.
type simulationActionRequest struct {
	Type  engine.SimulationActionKind `json:"type"`
	Noble models.NobleCode            `json:"noble"`
	Other models.NobleCode            `json:"other,omitempty"`
}

type victorySimulationRequest struct {
	Actions []simulationActionRequest `json:"actions"`
}

// VictoryReadingView is a player's score and victory status in one state.
type VictoryReadingView struct {
	Score   engine.ScoreBreakdown `json:"score"`
	Victory engine.PlayerVictory  `json:"victory"`
	Status  string                `json:"status"`
	Missing int                   `json:"missing"`
}

// VictorySimulationView compares every player's reading now and after the
// actions. State is the projected state, served like state.json so the client
// draws the hypothetical board without re-implementing any rule.
type VictorySimulationView struct {
	Current   map[models.PlayerID]VictoryReadingView `json:"current"`
	Projected map[models.PlayerID]VictoryReadingView `json:"projected"`
	State     StateView                              `json:"state"`
}

func readingViews(readings map[models.PlayerID]engine.VictoryReading) map[models.PlayerID]VictoryReadingView {
	views := make(map[models.PlayerID]VictoryReadingView, len(readings))
	for id, reading := range readings {
		views[id] = VictoryReadingView{Score: reading.Score, Victory: reading.Victory, Status: reading.Status, Missing: reading.Missing}
	}
	return views
}

func (h *GamesHandler) simulateVictory(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	var request victorySimulationRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<16)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_simulation_request", err.Error())
		return
	}
	snapshot, err := h.store.State(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	viewerID, ok := snapshot.ViewerFor(actor)
	if !ok {
		writeAPIError(w, http.StatusForbidden, "not_member", "only a member of this game can simulate victory")
		return
	}
	actions := make([]engine.SimulationAction, 0, len(request.Actions))
	for _, action := range request.Actions {
		actions = append(actions, engine.SimulationAction{Kind: action.Type, Noble: action.Noble, Other: action.Other})
	}
	simulation, err := engine.SimulateVictory(snapshot.State, h.balance, actions)
	if err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, "invalid_simulation", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, VictorySimulationView{
		Current:   readingViews(simulation.Current),
		Projected: readingViews(simulation.Projected),
		State:     projectStateForPlayer(simulation.State, viewerID, h.balance),
	})
}
