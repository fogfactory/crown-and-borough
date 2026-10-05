package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestComputeScoresTerritoriesArmiesAndNoblesAloneScoreZero verifies that
// controlling territories, owning armies, and holding nobles no longer
// contribute to the score on their own (#251): only titles (fiefs, for now)
// count, per titres.md § Score de titres.
func TestComputeScoresTerritoriesArmiesAndNoblesAloneScoreZero(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Players: []models.Player{{ID: p1}, {ID: p2}},
		Territories: []models.Territory{
			{ID: "AAA"},
			{ID: "BBB"},
			{ID: "CCC"},
		},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{
			"AAA": {Resources: 3, Infrastructures: infraPointer("I1"), Army: armyPointer("A1")},
			"BBB": {Resources: 2, Infrastructures: infraPointer("I2"), Army: armyPointer("A3")},
			"CCC": {Resources: 1, Infrastructures: infraPointer("I3"), Army: armyPointer("A2")},
		},
		Infrastructures: []models.Infrastructure{
			{ID: "I1", Type: models.InfraTypeCastle, TerritoryID: "AAA"},
			{ID: "I2", Type: models.InfraTypeMill, TerritoryID: "BBB"},
			{ID: "I3", Type: models.InfraTypeVillage, TerritoryID: "CCC"},
		},
		Armies: []models.Army{
			{ID: "A1", OwnerID: p1, TerritoryID: "AAA", Size: 4},
			{ID: "A2", OwnerID: p2, TerritoryID: "CCC", Size: 2},
			{ID: "A3", OwnerID: p1, TerritoryID: "BBB", Size: 1},
		},
		Nobles: []models.Noble{
			{ID: "N1", Sex: models.SexMale, OwnerID: p1, LocationID: "AAA", Status: models.NobleStatusFree},
			{ID: "N2", Sex: models.SexMale, OwnerID: p2, LocationID: "AAA", Status: models.NobleStatusHostage},
			{ID: "N3", Sex: models.SexMale, OwnerID: p1, LocationID: "CCC", Status: models.NobleStatusDungeon},
			{ID: "N4", Sex: models.SexMale, OwnerID: p2, LocationID: "CCC", Status: models.NobleStatusFree},
		},
	}

	scores := ComputeScores(state)
	if got, want := scores[p1], (ScoreBreakdown{Total: 0}); got != want {
		t.Fatalf("P1 score = %#v, want %#v", got, want)
	}
	if got, want := scores[p2], (ScoreBreakdown{Total: 0}); got != want {
		t.Fatalf("P2 score = %#v, want %#v", got, want)
	}
}

// TestComputeScoresTitles verifies that each fief contributes exactly 1
// point to its owner, vacant or not, until it is dissolved, and that the
// title's rank (barony, county, duchy) does not matter (titres.md).
func TestComputeScoresTitles(t *testing.T) {
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state := &models.GameState{
		Players:         []models.Player{{ID: p1}, {ID: p2}},
		Territories:     []models.Territory{{ID: "AAA"}, {ID: "BBB"}, {ID: "CCC"}},
		TerritoryStates: map[models.TerritoryID]models.TerritoryState{},
		Fiefs: []models.Fief{
			{ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA", Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: p1},
			{ID: "F2", Title: models.FiefTitleCounty, CapitalTerritoryID: "AAA", Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: p1},
			{ID: "F3", Title: models.FiefTitleDuchy, CapitalTerritoryID: "AAA", Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: p1},
		},
	}
	scores := ComputeScores(state)
	if got, want := scores[p1], (ScoreBreakdown{Titles: 3, Total: 3}); got != want {
		t.Errorf("P1 score = %#v, want %#v (vacant still counts, rank does not matter)", got, want)
	}
	if got, want := scores[p2], (ScoreBreakdown{Titles: 0, Total: 0}); got != want {
		t.Errorf("P2 score = %#v, want %#v", got, want)
	}

	state.Fiefs = nil
	scores = ComputeScores(state)
	if got, want := scores[p1], (ScoreBreakdown{Titles: 0, Total: 0}); got != want {
		t.Errorf("P1 score after dissolution = %#v, want %#v", got, want)
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
			"AAA": {Resources: 1, Army: armyPointer("A1")},
			"BBB": {Army: armyPointer("A2")},
		},
		Armies: []models.Army{
			{ID: "A1", OwnerID: p1, TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: p2, TerritoryID: "BBB", Size: 1},
		},
		Fiefs: []models.Fief{
			{ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA", Territories: []models.TerritoryID{"AAA"}, OwnerID: p1},
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
			"AAA": {Army: armyPointer("A1")},
		},
		Armies: []models.Army{{ID: "A1", OwnerID: p1, TerritoryID: "AAA", Size: 1}},
		Nobles: []models.Noble{{ID: "N2", Sex: models.SexMale, OwnerID: p2, LocationID: "AAA", Status: models.NobleStatusFree}},
	}

	winner := WinnerForFinishedGame(state)
	if winner == nil || *winner != p1 {
		t.Fatalf("winner = %v, want sole survivor %s", winner, p1)
	}
}

// TestWinnerForFinishedGameReturnsNoWinnerForExactTie covers the common
// transitional case of a 0-0 tie: neither player holds a title, so the
// duration limit ends with no winner (accepted until victory thresholds,
// #253/#254, land).
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
			"AAA": {Army: armyPointer("A1")},
			"BBB": {Army: armyPointer("A2")},
		},
		Armies: []models.Army{
			{ID: "A1", OwnerID: p1, TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: p2, TerritoryID: "BBB", Size: 1},
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
			{ID: "N1", Sex: models.SexMale, OwnerID: "P1", Status: models.NobleStatusHostage},
			{ID: "N2", Sex: models.SexMale, OwnerID: "P2", Status: models.NobleStatusDungeon},
			{ID: "N3", Sex: models.SexMale, OwnerID: "P3", Status: models.NobleStatusFree},
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
		Nobles:  []models.Noble{{ID: "N1", Sex: models.SexMale, OwnerID: "P1", Status: models.NobleStatusDungeon}},
	}
	if PlayerMustSubmit(state, "P1") {
		t.Fatal("PlayerMustSubmit = true without an emitting noble or a card")
	}
	state.SpecialDeck = &models.SpecialDeck{Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}}}
	if !PlayerMustSubmit(state, "P1") {
		t.Fatal("PlayerMustSubmit = false with a playable card in hand")
	}
}
