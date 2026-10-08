package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func religionState() *models.GameState {
	state := models.NewGameState()
	state.Players = []models.Player{{ID: "P1"}, {ID: "P2"}, {ID: "P3"}}
	state.Regions = []models.Region{{ID: "AAA", Name: "Aaa", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}}}
	state.Nobles = []models.Noble{
		{ID: "N1", OwnerID: "P1", Status: models.NobleStatusFree},
		{ID: "N2", OwnerID: "P1", Status: models.NobleStatusFree},
		{ID: "N3", OwnerID: "P2", Status: models.NobleStatusDungeon},
	}
	return state
}

func TestTitleVotesCountsOnlyTheHighestActiveTitle(t *testing.T) {
	state := religionState()
	pope := models.NobleID("N1")
	state.Bishops = []models.Bishop{{Region: "AAA", Noble: "N1"}}
	state.Cardinals = []models.NobleID{"N1"}
	state.Pope = &pope
	balance := testBalance()

	if got := TitleVotes(state, "P1", balance); got != 3 {
		t.Errorf("pope-cardinal-bishop votes = %d, want 3 (highest title only)", got)
	}
	state.Excommunications = []models.Excommunication{{Noble: "N1", Reason: models.ExcommunicationExOfficio}}
	state.Bishops, state.Cardinals, state.Pope = nil, nil, nil
	if got := TitleVotes(state, "P1", balance); got != 0 {
		t.Errorf("excommunicated votes = %d, want 0", got)
	}
	// A jailed bishop keeps the title but its votes are suspended.
	state.Bishops = []models.Bishop{{Region: "AAA", Noble: "N3"}}
	if got := TitleVotes(state, "P2", balance); got != 0 {
		t.Errorf("dungeon bishop votes = %d, want 0", got)
	}
	state.Nobles[2].Status = models.NobleStatusHostage
	if got := TitleVotes(state, "P2", balance); got != 1 {
		t.Errorf("hostage bishop votes = %d, want 1", got)
	}
	for players, want := range map[int]int{2: 1, 5: 1, 6: 2, 11: 2, 12: 3} {
		if got := models.CardinalCap(players, 1, 6); got != want {
			t.Errorf("cardinal cap for %d players = %d, want %d", players, got, want)
		}
	}
	if got := CardinalCap(state, balance); got != 1 {
		t.Errorf("cardinal cap for 3 players = %d, want 1", got)
	}
}

func TestCloneGameStateDeepCopiesReligion(t *testing.T) {
	state := religionState()
	pope := models.NobleID("N1")
	state.Bishops = []models.Bishop{{Region: "AAA", Noble: "N1"}}
	state.Cardinals = []models.NobleID{"N1"}
	state.Pope = &pope
	state.Excommunications = []models.Excommunication{{Noble: "N2", Reason: models.ExcommunicationExOfficio}}
	clone := cloneGameState(state)
	clone.Bishops[0].Noble = "N2"
	clone.Cardinals[0] = "N2"
	*clone.Pope = "N2"
	clone.Excommunications[0].Noble = "N1"
	if state.Bishops[0].Noble != "N1" || state.Cardinals[0] != "N1" || *state.Pope != "N1" || state.Excommunications[0].Noble != "N2" {
		t.Errorf("clone shares religion storage with the source: %+v", state)
	}
}
