package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestComputeScoresCountsCategoriesAndCaptiveHolder(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Players: []models.Player{{ID: p1}, {ID: p2}},
		Territories: []models.Territory{
			{ID: "AAA"},
			{ID: "BBB"},
			{ID: "CCC"},
		},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			"AAA": {OwnerID: &p1, Resources: 3, Infrastructures: infraPointer("I1")},
			"BBB": {OwnerID: &p1, Resources: 2, Infrastructures: infraPointer("I2")},
			"CCC": {OwnerID: &p2, Resources: 1, Infrastructures: infraPointer("I3")},
		},
		Infrastructures: []models.Infrastructure{
			{ID: "I1", Type: models.InfraTypeCastle, TerritoryID: "AAA"},
			{ID: "I2", Type: models.InfraTypeMill, TerritoryID: "BBB"},
			{ID: "I3", Type: models.InfraTypeVillage, TerritoryID: "CCC"},
		},
		Armies: []models.Army{
			{ID: "A1", OwnerID: p1, TerritoryID: "AAA", Size: 4},
			{ID: "A2", OwnerID: p2, TerritoryID: "CCC", Size: 2},
		},
		Nobles: []models.Noble{
			{ID: "N1", OwnerID: p1, LocationID: "AAA", Status: models.NobleStatusFree},
			{ID: "N2", OwnerID: p2, LocationID: "AAA", Status: models.NobleStatusHostage},
			{ID: "N3", OwnerID: p1, LocationID: "CCC", Status: models.NobleStatusDungeon},
			{ID: "N4", OwnerID: p2, LocationID: "CCC", Status: models.NobleStatusFree},
		},
	}

	scores := ComputeScores(state)
	if got, want := scores[p1], (ScoreBreakdown{Territories: 2, Castles: 5, Mills: 1, Nobles: 4, Troops: 4, Resources: 5, Total: 21}); got != want {
		t.Fatalf("P1 score = %#v, want %#v", got, want)
	}
	if got, want := scores[p2], (ScoreBreakdown{Territories: 1, Villages: 2, Nobles: 4, Troops: 2, Resources: 1, Total: 10}); got != want {
		t.Fatalf("P2 score = %#v, want %#v", got, want)
	}
}

func TestWinnerForFinishedGameUsesScoreAtYearLimit(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Turn:      5,
		YearCount: 1,
		Players:   []models.Player{{ID: p1}, {ID: p2}},
		Territories: []models.Territory{
			{ID: "AAA"},
			{ID: "BBB"},
		},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			"AAA": {OwnerID: &p1, Resources: 1},
			"BBB": {OwnerID: &p2},
		},
	}

	if !GameFinished(state) {
		t.Fatal("state should be finished at the year limit")
	}
	winner := WinnerForFinishedGame(state)
	if winner == nil || *winner != p1 {
		t.Fatalf("winner = %v, want %s", winner, p1)
	}
}

func TestWinnerForFinishedGamePrefersSoleSurvivor(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Turn:      5,
		YearCount: 1,
		Players:   []models.Player{{ID: p1}, {ID: p2}},
		Territories: []models.Territory{
			{ID: "AAA"},
		},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			"AAA": {OwnerID: &p1},
		},
		Nobles: []models.Noble{{ID: "N2", OwnerID: p2, LocationID: "AAA", Status: models.NobleStatusFree}},
	}

	winner := WinnerForFinishedGame(state)
	if winner == nil || *winner != p1 {
		t.Fatalf("winner = %v, want sole survivor %s", winner, p1)
	}
}

func TestWinnerForFinishedGameReturnsNoWinnerForExactTie(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Turn:      5,
		YearCount: 1,
		Players:   []models.Player{{ID: p1}, {ID: p2}},
		Territories: []models.Territory{
			{ID: "AAA"},
			{ID: "BBB"},
		},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			"AAA": {OwnerID: &p1},
			"BBB": {OwnerID: &p2},
		},
	}

	if winner := WinnerForFinishedGame(state); winner != nil {
		t.Fatalf("winner = %v, want no winner for exact tie", winner)
	}
}

func TestPlayerMustSubmit(t *testing.T) {
	state := &models.GameState{
		Season:  models.SeasonSpring,
		Players: []models.Player{{ID: "P1"}, {ID: "P2"}, {ID: "P3"}},
		Armies: []models.Army{
			{ID: "A1", OwnerID: "P1", Size: 1},
			{ID: "A2", OwnerID: "P2", Size: 1},
		},
		Nobles: []models.Noble{
			{ID: "N1", OwnerID: "P1", Status: models.NobleStatusHostage},
			{ID: "N2", OwnerID: "P2", Status: models.NobleStatusDungeon},
			{ID: "N3", OwnerID: "P3", Status: models.NobleStatusFree},
		},
	}
	for _, test := range []struct {
		season models.Season
		player models.PlayerID
		want   bool
	}{
		{models.SeasonSpring, "P1", true},  // a hostage noble can still emit
		{models.SeasonSpring, "P2", false}, // a dungeon noble cannot emit
		{models.SeasonSpring, "P3", false}, // eliminated despite a free noble
		{models.SeasonWinter, "P2", true},  // winter orders need no noble
		{models.SeasonWinter, "P3", false},
	} {
		state.Season = test.season
		if got := PlayerMustSubmit(state, test.player); got != test.want {
			t.Errorf("PlayerMustSubmit(%s, %s) = %v, want %v", test.season, test.player, got, test.want)
		}
	}
}

func TestPlayerMustSubmitWaitsForPlayerWithCardInHand(t *testing.T) {
	state := &models.GameState{
		Season:  models.SeasonSummer,
		Players: []models.Player{{ID: "P1"}},
		Armies:  []models.Army{{ID: "A1", OwnerID: "P1", Size: 1}},
		Nobles:  []models.Noble{{ID: "N1", OwnerID: "P1", Status: models.NobleStatusDungeon}},
	}
	if PlayerMustSubmit(state, "P1") {
		t.Fatal("PlayerMustSubmit = true without an emitting noble or a card")
	}
	state.SpecialDeck = &models.SpecialDeck{Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}}}
	if !PlayerMustSubmit(state, "P1") {
		t.Fatal("PlayerMustSubmit = false with a playable card in hand")
	}
}
