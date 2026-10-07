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
	// Alliance is the bonus earned from influence marriages, those that are
	// not the active head of both houses (specs/succession.md § Poids
	// d'alliance).
	Alliance int `json:"alliance"`
	Total    int `json:"total"`
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
func ComputeScores(state *models.GameState, balance assetgen.Balance) map[models.PlayerID]ScoreBreakdown {
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

	// Each dignity a living noble carries (the bastard, specs/succession.md
	// § Bâtard) counts as a title.
	for _, noble := range state.Nobles {
		score, exists := scores[noble.OwnerID]
		if !exists {
			continue
		}
		score.Titles += len(noble.Dignities)
		scores[noble.OwnerID] = score
	}

	applyMarriageBonuses(state, balance, scores)

	for playerID, score := range scores {
		score.Total = score.Titles + score.Alliance
		scores[playerID] = score
	}
	return scores
}

// applyMarriageBonuses adds the marriage bonuses to each house's score
// (specs/succession.md § Poids d'alliance). A marriage that is the active head
// of both houses is a full alliance: no individual bonus, the two scores are
// combined for the joint victory instead. Every other active alliance is an
// influence marriage: each house gains the title count of the spouse's house,
// which counts toward a solo win or a win through another head. Bonuses use
// the spouse house's own title count, never a bonus-inflated score, so
// marriages cannot feed each other.
func applyMarriageBonuses(state *models.GameState, balance assetgen.Balance, scores map[models.PlayerID]ScoreBreakdown) {
	titles := make(map[models.PlayerID]int, len(scores))
	for playerID, score := range scores {
		titles[playerID] = score.Titles
	}
	for _, marriage := range state.Marriages {
		houseA, houseB := marriageHouses(state, marriage)
		if houseA == houseB {
			continue
		}
		categoryA, okA := EffectiveMarriageCategory(state, balance, houseA, marriage)
		categoryB, okB := EffectiveMarriageCategory(state, balance, houseB, marriage)
		if !okA || !okB || (categoryA == AllianceHead && categoryB == AllianceHead) {
			continue
		}
		for _, side := range [2][2]models.PlayerID{{houseA, houseB}, {houseB, houseA}} {
			player, spouse := side[0], side[1]
			if score, exists := scores[player]; exists {
				score.Alliance += titles[spouse]
				scores[player] = score
			}
		}
	}
}

// Victory modes a player is currently evaluated under.
const (
	VictoryModeSolo     = "solo"
	VictoryModeAlliance = "alliance"
)

// PlayerVictory describes what one player needs to win: the mode they are
// evaluated under, the title score required in that mode and, for an alliance,
// the partner whose score is added to theirs.
type PlayerVictory struct {
	Mode     string           `json:"mode"`
	Required int              `json:"required"`
	Partner  *models.PlayerID `json:"partner,omitempty"`
}

// VictoryStatus is the public view of the victory thresholds for a state.
type VictoryStatus struct {
	SoloThreshold     int                               `json:"soloThreshold"`
	AllianceThreshold int                               `json:"allianceThreshold"`
	Players           map[models.PlayerID]PlayerVictory `json:"players"`
}

// ComputeVictoryStatus returns the thresholds and each player's victory mode.
// No player can hold an active head yet (marriage categories and alliance
// weights are still to come, #254), so every player is in solo mode; once a
// head exists its owner switches to alliance mode with the spouse's player as
// Partner and the alliance threshold as Required.
func ComputeVictoryStatus(state *models.GameState, balance assetgen.Balance) VictoryStatus {
	status := VictoryStatus{Players: map[models.PlayerID]PlayerVictory{}}
	if state == nil {
		return status
	}
	status.SoloThreshold, status.AllianceThreshold = VictoryThresholds(balance, len(state.Players))
	for _, player := range state.Players {
		status.Players[player.ID] = PlayerVictory{Mode: VictoryModeSolo, Required: status.SoloThreshold}
	}
	return status
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
	scores := ComputeScores(state, balance)
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
	scores := ComputeScores(state, balance)
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
