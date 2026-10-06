package api

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestProjectStateForPlayerProjectsOnlyCurrentHand(t *testing.T) {
	state := models.NewGameState()
	state.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindFairWeather}, {ID: "C2", Kind: models.CardKindPlague}},
		DrawPile: []models.SpecialCardID{"C2"}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}, "P2": {}},
	}
	balance := assetgen.Balance{SpecialOrders: assetgen.SpecialOrdersBalance{HandLimit: 4}}
	p1 := ProjectStateForPlayer(state, "P1", balance)
	p2 := ProjectStateForPlayer(state, "P2", assetgen.Balance{})
	if p1.HandLimit != 4 {
		t.Fatalf("handLimit = %d, want the balance hand limit", p1.HandLimit)
	}
	if len(p1.SpecialHand) != 1 || p1.SpecialHand[0] != models.CardKindFairWeather {
		t.Fatalf("P1 special hand = %#v, want fair_weather", p1.SpecialHand)
	}
	if len(p2.SpecialHand) != 0 {
		t.Fatalf("P2 special hand = %#v, want empty", p2.SpecialHand)
	}
}

func TestProjectStateForPlayerProjectsPendingAnnouncements(t *testing.T) {
	state := models.NewGameState()
	state.Turn = 5
	state.Season = models.SeasonSpring
	state.Players = []models.Player{{ID: "P1", Name: "One"}}
	state.Auguries[2] = models.YearAugury{Year: 2, Calamities: []models.Calamity{
		{CardID: "C1", Kind: models.CardKindPlague, Year: 2, Season: models.SeasonSpring, RegionSeed: "ROS"},
		{CardID: "C2", Kind: models.CardKindBadWeather, Year: 2, Season: models.SeasonSummer, RegionSeed: "ROS"},
	}}
	state.Auguries[3] = models.YearAugury{Year: 3, Calamities: []models.Calamity{
		{CardID: "C3", Kind: models.CardKindFamine, Year: 3, Season: models.SeasonAutumn, RegionSeed: "BOI"},
	}}
	view := ProjectStateForPlayer(state, "P1", assetgen.Balance{})
	if len(view.Announcements) != 3 {
		t.Fatalf("announcements = %#v, want three pending calamities", view.Announcements)
	}
	if view.Announcements[0].Kind != models.CardKindPlague || view.Announcements[0].Season != models.SeasonSpring {
		t.Fatalf("first announcement = %#v, want spring plague of the current year", view.Announcements[0])
	}
	if view.Announcements[2].Kind != models.CardKindFamine || view.Announcements[2].Year != 3 {
		t.Fatalf("last announcement = %#v, want next-year famine", view.Announcements[2])
	}
}

func TestProjectStateForPlayerProjectsOnlyOwnNobleHand(t *testing.T) {
	state := models.NewGameState()
	state.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
	state.NobleDeck = &models.NobleDeck{
		Cards: []models.NobleCard{
			{ID: "K1", Kind: models.NobleCardKindNoble, Code: "ELE", Name: "Eleonore", Sex: models.SexFemale},
			{ID: "K2", Kind: models.NobleCardKindDignity, Code: "BAS", Dignity: models.DignityBastard},
			{ID: "K3", Kind: models.NobleCardKindNoble, Code: "GUI", Name: "Guy", Sex: models.SexMale},
		},
		DrawPile: []models.NobleCardID{"K3"},
		Hands:    map[models.PlayerID][]models.NobleCardID{"P1": {"K1", "K2"}, "P2": {}},
		Discard:  []models.NobleCardID{},
		Played:   []models.NobleCardPlay{},
	}
	p1 := ProjectStateForPlayer(state, "P1", assetgen.Balance{})
	p2 := ProjectStateForPlayer(state, "P2", assetgen.Balance{})
	if len(p1.NobleHand) != 2 || p1.NobleHand[0].Code != "ELE" || p1.NobleHand[1].Dignity != models.DignityBastard {
		t.Fatalf("P1 noble hand = %#v, want the Eleonore and bastard cards", p1.NobleHand)
	}
	if p1.NobleDiscardSize != 0 {
		t.Fatalf("noble discard size = %d, want 0", p1.NobleDiscardSize)
	}
	if len(p2.NobleHand) != 0 || p1.NobleDeckSize != 1 || p2.NobleDeckSize != 1 {
		t.Fatalf("P2 noble hand = %#v, deck sizes = %d/%d, want an empty hand and 1 card in the pile", p2.NobleHand, p1.NobleDeckSize, p2.NobleDeckSize)
	}
}
