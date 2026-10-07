package api

import (
	"encoding/json"
	"net/http"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

type victorySimulationRequest struct {
	MarriageEnds []struct {
		Noble  models.NobleCode `json:"noble"`
		Spouse models.NobleCode `json:"spouse"`
	} `json:"marriageEnds"`
	Deaths    []models.NobleCode `json:"deaths"`
	Marriages []struct {
		Noble  models.NobleCode `json:"noble"`
		Spouse models.NobleCode `json:"spouse"`
	} `json:"marriages"`
	Claims []struct {
		Heir   models.NobleCode `json:"heir"`
		Target models.NobleCode `json:"target"`
	} `json:"claims"`
}

// VictoryReadingView is the score and victory status of the player in one state.
type VictoryReadingView struct {
	Score   engine.ScoreBreakdown `json:"score"`
	Victory engine.PlayerVictory  `json:"victory"`
	Status  string                `json:"status"`
	Missing int                   `json:"missing"`
}

// VictorySimulationView compares the current state with the hypothetical one.
type VictorySimulationView struct {
	Current   VictoryReadingView `json:"current"`
	Projected VictoryReadingView `json:"projected"`
}

func readingView(reading engine.VictoryReading) VictoryReadingView {
	return VictoryReadingView{Score: reading.Score, Victory: reading.Victory, Status: reading.Status, Missing: reading.Missing}
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
	playerID, ok := snapshot.PlayerFor(actor)
	if !ok {
		writeAPIError(w, http.StatusForbidden, "not_member", "only a player of this game can simulate victory")
		return
	}
	var scenario engine.VictoryScenario
	scenario.Deaths = request.Deaths
	for _, end := range request.MarriageEnds {
		scenario.MarriageEnds = append(scenario.MarriageEnds, engine.MarriageHypothesis{Noble: end.Noble, Spouse: end.Spouse})
	}
	for _, marriage := range request.Marriages {
		scenario.Marriages = append(scenario.Marriages, engine.MarriageHypothesis{Noble: marriage.Noble, Spouse: marriage.Spouse})
	}
	for _, claim := range request.Claims {
		scenario.Claims = append(scenario.Claims, engine.ClaimHypothesis{Heir: claim.Heir, Target: claim.Target})
	}
	projection, err := engine.SimulateVictory(snapshot.State, h.balance, playerID, scenario)
	if err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, "invalid_simulation", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, VictorySimulationView{
		Current:   readingView(projection.Current),
		Projected: readingView(projection.Projected),
	})
}
