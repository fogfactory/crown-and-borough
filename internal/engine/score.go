package engine

import (
	"errors"
	"slices"

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
// A player with an active head (specs/titres.md § Seuil de victoire) can never
// win alone: they are evaluated in alliance mode against the alliance
// threshold, with the spouse's player as Partner. Every other player is
// evaluated against the solo threshold.
func ComputeVictoryStatus(state *models.GameState, balance assetgen.Balance) VictoryStatus {
	status := VictoryStatus{Players: map[models.PlayerID]PlayerVictory{}}
	if state == nil {
		return status
	}
	status.SoloThreshold, status.AllianceThreshold = VictoryThresholds(balance, len(state.Players))
	for _, player := range state.Players {
		if partner, locked := activeHeadPartner(state, balance, player.ID); locked {
			status.Players[player.ID] = PlayerVictory{Mode: VictoryModeAlliance, Required: status.AllianceThreshold, Partner: &partner}
			continue
		}
		status.Players[player.ID] = PlayerVictory{Mode: VictoryModeSolo, Required: status.SoloThreshold}
	}
	return status
}

// activeHeadPartner returns the player married through player's active head,
// and whether player has one. A head marriage between two nobles of the same
// house locks nobody.
func activeHeadPartner(state *models.GameState, balance assetgen.Balance, player models.PlayerID) (models.PlayerID, bool) {
	marriage, found := ActiveHeadMarriage(state, balance, player)
	if !found {
		return "", false
	}
	houseA, houseB := marriageHouses(state, marriage)
	if houseA == houseB {
		return "", false
	}
	if houseA == player {
		return houseB, true
	}
	return houseA, true
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

// victoryUnit is what is scored against a threshold: one player without an
// active head against the solo threshold, or the two spouses of an active head
// with their combined score against the alliance threshold.
type victoryUnit struct {
	players  []models.PlayerID
	score    int
	required int
	alliance bool
	// crown and territories break a tie on score: whether the unit holds
	// the crown, then the territories its players control.
	crown       bool
	territories int
}

func (u victoryUnit) reached() bool { return u.required >= 1 && u.score >= u.required }

// victoryUnits lists the units of the state, one per solo player and one per
// alliance (deduplicated when both spouses have the marriage as active head).
func victoryUnits(state *models.GameState, balance assetgen.Balance) []victoryUnit {
	solo, alliance := VictoryThresholds(balance, len(state.Players))
	scores := ComputeScores(state, balance)
	held := map[models.PlayerID]int{}
	for _, controller := range state.TerritoryControllers() {
		held[controller]++
	}
	var units []victoryUnit
	// The crown has no engine state yet (titles of king are still to come);
	// crownHolder stays empty until it does.
	var crownHolder models.PlayerID
	seen := map[[2]models.PlayerID]bool{}
	for _, player := range state.Players {
		partner, locked := activeHeadPartner(state, balance, player.ID)
		if _, known := scores[partner]; !locked || !known {
			units = append(units, victoryUnit{players: []models.PlayerID{player.ID}, score: scores[player.ID].Total, required: solo, crown: crownHolder == player.ID, territories: held[player.ID]})
			continue
		}
		key := [2]models.PlayerID{player.ID, partner}
		if partner < player.ID {
			key = [2]models.PlayerID{partner, player.ID}
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		units = append(units, victoryUnit{
			players:     []models.PlayerID{key[0], key[1]},
			score:       scores[player.ID].Total + scores[partner].Total,
			required:    alliance,
			alliance:    true,
			crown:       crownHolder != "" && (crownHolder == key[0] || crownHolder == key[1]),
			territories: held[player.ID] + held[partner],
		})
	}
	return units
}

// thresholdReached reports whether any unit crossed its threshold.
func thresholdReached(state *models.GameState, balance assetgen.Balance) bool {
	for _, unit := range victoryUnits(state, balance) {
		if unit.reached() {
			return true
		}
	}
	return false
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
	return alive <= 1 || thresholdReached(state, balance) ||
		(state.YearCount > 0 && state.Turn > state.YearCount*4)
}

// VictoryOutcome is the result of a finished game (specs/titres.md § Victoire
// majeure, victoire mineure, échec). Major lists the major winners: one
// player, or the two spouses of a winning alliance with no hierarchy between
// them. Minor is the single minor winner, if any; every other player fails.
type VictoryOutcome struct {
	Major []models.PlayerID `json:"major"`
	Minor *models.PlayerID  `json:"minor,omitempty"`
}

// ResolveVictory returns the outcome once GameFinished is true. A sole
// survivor wins immediately; otherwise, among the units that crossed their
// threshold (or, at the duration limit, among all units) the highest score
// wins, ties resolved by compareUnits; units still tied are all major winners. The minor victory goes to
// the player best linked, through a chain of marriages, to a major winner.
func ResolveVictory(state *models.GameState, balance assetgen.Balance) VictoryOutcome {
	outcome := VictoryOutcome{Major: []models.PlayerID{}}
	if state == nil || !GameFinished(state, balance) {
		return outcome
	}
	var alive []models.PlayerID
	for _, player := range state.Players {
		if PlayerAlive(state, player.ID) {
			alive = append(alive, player.ID)
		}
	}
	if len(alive) == 1 {
		outcome.Major = alive
		return outcome
	}

	units := victoryUnits(state, balance)
	var candidates []victoryUnit
	for _, unit := range units {
		if unit.reached() {
			candidates = append(candidates, unit)
		}
	}
	reachedThreshold := len(candidates) > 0
	if !reachedThreshold {
		candidates = units
	}
	// Several units crossing their threshold at once: a solo win beats an
	// alliance win, then the higher score, then the crown, then the most
	// territories. Units still tied share the victory. A game where nobody
	// holds a title has no winner.
	var best []victoryUnit
	for _, unit := range candidates {
		if len(best) == 0 {
			best = []victoryUnit{unit}
			continue
		}
		switch compareUnits(unit, best[0], reachedThreshold) {
		case 1:
			best = []victoryUnit{unit}
		case 0:
			best = append(best, unit)
		}
	}
	if len(best) == 0 || best[0].score == 0 {
		return outcome
	}
	for _, unit := range best {
		for _, id := range unit.players {
			if !slices.Contains(outcome.Major, id) {
				outcome.Major = append(outcome.Major, id)
			}
		}
	}
	outcome.Minor = minorWinner(state, balance, outcome.Major)
	return outcome
}

// compareUnits returns 1 when a beats b, -1 when b beats a and 0 on a tie.
// soloFirst ranks a solo unit above an alliance one.
func compareUnits(a, b victoryUnit, soloFirst bool) int {
	if soloFirst && a.alliance != b.alliance {
		if a.alliance {
			return -1
		}
		return 1
	}
	for _, pair := range [3][2]int{{a.score, b.score}, {boolInt(a.crown), boolInt(b.crown)}, {a.territories, b.territories}} {
		if pair[0] != pair[1] {
			if pair[0] > pair[1] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// minorWinner picks, among the players linked to a major winner by a chain of
// active marriages of any category, the one whose strongest link into that
// chain has the highest alliance weight. A tie has no minor winner.
func minorWinner(state *models.GameState, balance assetgen.Balance, major []models.PlayerID) *models.PlayerID {
	type link struct {
		a, b   models.PlayerID
		weight int
	}
	var links []link
	for _, marriage := range state.Marriages {
		weight, ok := AllianceWeight(state, balance, marriage)
		houseA, houseB := marriageHouses(state, marriage)
		if ok && houseA != houseB {
			links = append(links, link{houseA, houseB, weight})
		}
	}
	reached := map[models.PlayerID]bool{}
	for _, id := range major {
		reached[id] = true
	}
	for grew := true; grew; {
		grew = false
		for _, l := range links {
			if reached[l.a] != reached[l.b] {
				reached[l.a], reached[l.b] = true, true
				grew = true
			}
		}
	}
	isMajor := map[models.PlayerID]bool{}
	for _, id := range major {
		isMajor[id] = true
	}
	best := map[models.PlayerID]int{}
	for _, l := range links {
		for _, side := range [2][2]models.PlayerID{{l.a, l.b}, {l.b, l.a}} {
			if reached[side[0]] && !isMajor[side[0]] && reached[side[1]] {
				best[side[0]] = max(best[side[0]], l.weight)
			}
		}
	}
	var winner models.PlayerID
	top, tied := -1, false
	for _, player := range state.Players {
		weight, linked := best[player.ID]
		switch {
		case !linked:
		case weight > top:
			winner, top, tied = player.ID, weight, false
		case weight == top:
			tied = true
		}
	}
	if tied || top < 0 {
		return nil
	}
	return &winner
}

// WinnerForFinishedGame returns the sole major winner once GameFinished is
// true, or nil when nobody won or two spouses won jointly (see ResolveVictory
// for the full outcome).
func WinnerForFinishedGame(state *models.GameState, balance assetgen.Balance) *models.PlayerID {
	outcome := ResolveVictory(state, balance)
	if len(outcome.Major) != 1 {
		return nil
	}
	return &outcome.Major[0]
}
