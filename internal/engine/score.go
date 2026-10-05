package engine

import (
	"errors"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

var ErrGameFinished = errors.New("engine: game is finished")

// ScoreBreakdown contains the points currently held by one player. The total
// is deliberately stored alongside the categories so API consumers do not
// need to duplicate the scoring formula.
type ScoreBreakdown struct {
	Titles int `json:"titles"`
	Total  int `json:"total"`
}

// ComputeScores calculates the public score for every player in the state,
// per the title score (titres.md § Score de titres): each title held is
// worth 1 point regardless of rank, replacing the former GDD §9 formula
// (territories, infrastructure, armies, nobles). For now the only title
// source is state.Fiefs — a fief still counts while vacant, until it is
// dissolved. Cardinal, pape, roi, and dignité titles will add further
// sources to this count once Succession & Couronnement lands (#243-266);
// the loop below is structured so each source stays a self-contained pass
// over state, without pulling in that machinery ahead of time.
func ComputeScores(state *models.GameState) map[models.PlayerID]ScoreBreakdown {
	scores := make(map[models.PlayerID]ScoreBreakdown)
	if state == nil {
		return scores
	}
	for _, player := range state.Players {
		scores[player.ID] = ScoreBreakdown{}
	}

	// Each fief earns 1 point regardless of size, vacant or not, until it
	// is dissolved (titres.md).
	for _, fief := range state.Fiefs {
		score, exists := scores[fief.OwnerID]
		if !exists {
			continue
		}
		score.Titles++
		scores[fief.OwnerID] = score
	}

	for playerID, score := range scores {
		score.Total = score.Titles
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
	for _, fief := range state.Fiefs {
		if fief.OwnerID == playerID {
			return true
		}
	}
	for _, player := range state.Players {
		if player.ID == playerID && player.CapitalCastleID != nil {
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

// PlayerMustSubmit reports whether the current turn waits for playerID. An
// eliminated player never submits. During an action season, a player is only
// awaited when they can still order something: a free or hostage noble can
// emit a chain, and a card in hand can be played as a special order.
func PlayerMustSubmit(state *models.GameState, playerID models.PlayerID) bool {
	if !PlayerAlive(state, playerID) {
		return false
	}
	if state.Season == models.SeasonWinter {
		return true
	}
	if state.SpecialDeck != nil && len(state.SpecialDeck.Hands[playerID]) > 0 {
		return true
	}
	for _, noble := range state.Nobles {
		if noble.OwnerID == playerID && noble.Status != models.NobleStatusDungeon {
			return true
		}
	}
	return false
}

// VictoryThresholds returns the solo and alliance title-score thresholds for
// playerCount players (titres.md § Seuil de victoire et fin de partie). Each is
// the configured share of the game territories (mapgen.TerritoriesPerPlayer
// per player) divided by the reference fief size, rounded up. The alliance
// threshold is always strictly above the solo one. Both are zero when the
// balance defines no victory block, which disables threshold victories.
func VictoryThresholds(balance assetgen.Balance, playerCount int) (solo, alliance int) {
	victory := balance.Victory
	if victory.ReferenceFiefSize < 1 || victory.SoloTerritoryPercent < 1 {
		return 0, 0
	}
	denominator := 100 * victory.ReferenceFiefSize
	territories := mapgen.TerritoriesPerPlayer * playerCount
	solo = (territories*victory.SoloTerritoryPercent + denominator - 1) / denominator
	alliance = (territories*victory.AllianceTerritoryPercent + denominator - 1) / denominator
	if alliance <= solo {
		alliance = solo + 1
	}
	return solo, alliance
}

// thresholdWinners returns the players whose title score reaches the solo
// threshold. No player can hold an active head yet (marriage categories and
// alliance weights are still to come), so every player is evaluated against
// the solo threshold; the alliance threshold only applies once an active head
// exists.
func thresholdWinners(state *models.GameState, balance assetgen.Balance) []models.PlayerID {
	solo, _ := VictoryThresholds(balance, len(state.Players))
	if solo < 1 {
		return nil
	}
	scores := ComputeScores(state)
	var reached []models.PlayerID
	for _, player := range state.Players {
		if scores[player.ID].Total >= solo {
			reached = append(reached, player.ID)
		}
	}
	return reached
}

// GameFinished reports whether a state has reached an elimination, supremacy
// threshold or duration end condition. Turn values after the final winter are
// one greater than the configured number of years times four.
func GameFinished(state *models.GameState, balance assetgen.Balance) bool {
	if state == nil {
		return false
	}
	alive := 0
	for _, player := range state.Players {
		if PlayerAlive(state, player.ID) {
			alive++
		}
	}
	return alive <= 1 || len(thresholdWinners(state, balance)) > 0 ||
		(state.YearCount > 0 && state.Turn > state.YearCount*4)
}

// WinnerForFinishedGame returns the winner once GameFinished is true. A sole
// survivor wins immediately; otherwise, among the players that crossed the
// supremacy threshold (or, at the duration limit, among all players) the
// highest title score wins. An exact tie for the top score has no winner.
// Major/minor victory (#254) will refine this.
func WinnerForFinishedGame(state *models.GameState, balance assetgen.Balance) *models.PlayerID {
	if state == nil || !GameFinished(state, balance) {
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

	candidates := thresholdWinners(state, balance)
	if len(candidates) == 0 {
		candidates = make([]models.PlayerID, 0, len(state.Players))
		for _, player := range state.Players {
			candidates = append(candidates, player.ID)
		}
	}
	scores := ComputeScores(state)
	var winner models.PlayerID
	highest := -1
	tied := false
	for _, id := range candidates {
		score := scores[id].Total
		if score > highest {
			winner = id
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
