package engine

import (
	"errors"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

var ErrGameFinished = errors.New("engine: game is finished")

// ScoreBreakdown contains the points currently held by one player. The total
// is deliberately stored alongside the categories so API consumers do not
// need to duplicate the scoring formula.
type ScoreBreakdown struct {
	Territories int `json:"territories"`
	Villages    int `json:"villages"`
	Mills       int `json:"mills"`
	Castles     int `json:"castles"`
	Nobles      int `json:"nobles"`
	Troops      int `json:"troops"`
	Resources   int `json:"resources"`
	Total       int `json:"total"`
}

// ComputeScores calculates the public score for every player in the state.
// Infrastructure, resources, and captive nobles are awarded to the player who
// controls their current territory; a free noble is awarded to its owner.
func ComputeScores(state *models.GameState) map[models.PlayerID]ScoreBreakdown {
	scores := make(map[models.PlayerID]ScoreBreakdown)
	if state == nil {
		return scores
	}
	for _, player := range state.Players {
		scores[player.ID] = ScoreBreakdown{}
	}

	infrastructures := make(map[models.InfraID]models.Infrastructure, len(state.Infrastructures))
	for _, infrastructure := range state.Infrastructures {
		infrastructures[infrastructure.ID] = infrastructure
	}
	for _, territory := range state.Territories {
		territoryState := state.TerritoryStates[territory.ID]
		if territoryState.OwnerID == nil {
			continue
		}
		playerID := *territoryState.OwnerID
		score, exists := scores[playerID]
		if !exists {
			continue
		}
		score.Territories++
		score.Resources += territoryState.Resources
		for _, infrastructureID := range territoryState.Infrastructures {
			infrastructure, exists := infrastructures[infrastructureID]
			if !exists {
				continue
			}
			switch infrastructure.Type {
			case models.InfraTypeVillage:
				score.Villages += 2
			case models.InfraTypeMill:
				score.Mills++
			case models.InfraTypeCastle:
				score.Castles += 5
			}
		}
		scores[playerID] = score
	}

	for _, army := range state.Armies {
		score, exists := scores[army.OwnerID]
		if !exists {
			continue
		}
		score.Troops += army.Size
		scores[army.OwnerID] = score
	}

	for _, noble := range state.Nobles {
		playerID := noble.OwnerID
		if noble.Status == models.NobleStatusHostage || noble.Status == models.NobleStatusDungeon {
			territoryState, exists := state.TerritoryStates[noble.LocationID]
			if !exists || territoryState.OwnerID == nil {
				continue
			}
			playerID = *territoryState.OwnerID
		}
		score, exists := scores[playerID]
		if !exists {
			continue
		}
		score.Nobles += 2
		scores[playerID] = score
	}

	for playerID, score := range scores {
		score.Total = score.Territories + score.Villages + score.Mills + score.Castles +
			score.Nobles + score.Troops + score.Resources
		scores[playerID] = score
	}
	return scores
}

// PlayerAlive reports whether a player still controls a territory or owns a
// live army. Nobles alone do not keep a player in the game.
func PlayerAlive(state *models.GameState, playerID models.PlayerID) bool {
	if state == nil {
		return false
	}
	for _, territoryState := range state.TerritoryStates {
		if territoryState.OwnerID != nil && *territoryState.OwnerID == playerID {
			return true
		}
	}
	for _, army := range state.Armies {
		if army.OwnerID == playerID {
			return true
		}
	}
	return false
}

// GameFinished reports whether a state has reached an elimination or duration
// end condition. Turn values after the final winter are one greater than the
// configured number of years times four.
func GameFinished(state *models.GameState) bool {
	if state == nil {
		return false
	}
	alive := 0
	for _, player := range state.Players {
		if PlayerAlive(state, player.ID) {
			alive++
		}
	}
	return alive <= 1 || (state.YearCount > 0 && state.Turn > state.YearCount*4)
}

// WinnerForFinishedGame returns the winner once GameFinished is true. A sole
// survivor wins immediately, otherwise the highest score wins at the duration
// limit. An exact tie has no winner.
func WinnerForFinishedGame(state *models.GameState) *models.PlayerID {
	if state == nil || !GameFinished(state) {
		return nil
	}
	alive := make([]models.PlayerID, 0, len(state.Players))
	for _, player := range state.Players {
		if PlayerAlive(state, player.ID) {
			alive = append(alive, player.ID)
		}
	}
	if len(alive) == 1 {
		winner := alive[0]
		return &winner
	}
	if state.YearCount == 0 || state.Turn <= state.YearCount*4 {
		return nil
	}

	scores := ComputeScores(state)
	var winner models.PlayerID
	highest := -1
	tied := false
	for _, player := range state.Players {
		score := scores[player.ID].Total
		if score > highest {
			winner = player.ID
			highest = score
			tied = false
		} else if score == highest {
			tied = true
		}
	}
	if tied || winner == "" {
		return nil
	}
	return &winner
}
