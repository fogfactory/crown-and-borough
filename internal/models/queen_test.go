package models_test

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func queenState() *models.GameState {
	g := models.NewGameState()
	g.Players = []models.Player{{ID: "P1"}, {ID: "P2"}}
	g.Nobles = []models.Noble{
		{ID: "K1", OwnerID: "P1", Sex: models.SexMale},
		{ID: "Q1", OwnerID: "P2", Sex: models.SexFemale},
		{ID: "K2", OwnerID: "P1", Sex: models.SexMale},
		{ID: "Q2", OwnerID: "P2", Sex: models.SexFemale},
		{ID: "X", OwnerID: "P1", Sex: models.SexMale},
	}
	g.Marriages = []models.Marriage{{NobleA: "K1", NobleB: "Q1", Turn: 1}}
	king := models.NobleID("K1")
	g.King = &king
	return g
}

func killNoble(g *models.GameState, id models.NobleID) {
	for index, noble := range g.Nobles {
		if noble.ID == id {
			g.Nobles = append(g.Nobles[:index], g.Nobles[index+1:]...)
			return
		}
	}
}

func requireQueen(t *testing.T, g *models.GameState, want models.NobleID) {
	t.Helper()
	g.RefreshQueen()
	switch {
	case want == "" && g.Queen != nil:
		t.Fatalf("queen = %q, want none", *g.Queen)
	case want != "" && (g.Queen == nil || *g.Queen != want):
		t.Fatalf("queen = %v, want %q", g.Queen, want)
	}
}

func TestQueenIsTheSpouseOfTheKing(t *testing.T) {
	g := queenState()
	requireQueen(t, g, "Q1")
	if g.IsDowagerQueen("Q1") {
		t.Error("the spouse of the king in place is not a dowager")
	}
}

func TestWidowKeepsTheTitleUntilANewKingIsMarried(t *testing.T) {
	g := queenState()
	requireQueen(t, g, "Q1")

	// The king dies, an unmarried heir is crowned: Q1 stays queen, dowager.
	killNoble(g, "K1")
	g.King = nil
	requireQueen(t, g, "Q1")
	if !g.IsDowagerQueen("Q1") {
		t.Error("the widow of a vacant throne is a dowager queen")
	}
	king := models.NobleID("K2")
	g.King = &king
	requireQueen(t, g, "Q1")
	if !g.IsDowagerQueen("Q1") {
		t.Error("the widow of the previous king is a dowager queen")
	}

	// The new king marries: his spouse is the queen, the widow loses the title.
	g.Marriages = append(g.Marriages, models.Marriage{NobleA: "K2", NobleB: "Q2", Turn: 3})
	requireQueen(t, g, "Q2")
	if g.IsQueen("Q1") {
		t.Error("the widow must lose the title once a married king is in place")
	}

	// The new queen, widowed in turn, does not bring the old widow back.
	killNoble(g, "K2")
	g.King = nil
	requireQueen(t, g, "Q2")
}

func TestWidowWhoMarriesTheNewKingIsHisQueen(t *testing.T) {
	g := queenState()
	requireQueen(t, g, "Q1")
	killNoble(g, "K1")
	king := models.NobleID("K2")
	g.King = &king
	requireQueen(t, g, "Q1")
	g.Marriages = append(g.Marriages, models.Marriage{NobleA: "K2", NobleB: "Q1", Turn: 3})
	requireQueen(t, g, "Q1")
	if g.IsDowagerQueen("Q1") {
		t.Error("married to the king in place, she is no dowager")
	}
}

func TestQueenTitleEnds(t *testing.T) {
	t.Run("the widow remarries", func(t *testing.T) {
		g := queenState()
		requireQueen(t, g, "Q1")
		killNoble(g, "K1")
		g.King = nil
		g.Marriages = append(g.Marriages, models.Marriage{NobleA: "X", NobleB: "Q1", Turn: 3})
		requireQueen(t, g, "")
	})
	t.Run("the widow dies", func(t *testing.T) {
		g := queenState()
		requireQueen(t, g, "Q1")
		killNoble(g, "K1")
		g.King = nil
		killNoble(g, "Q1")
		requireQueen(t, g, "")
	})
	t.Run("the pope dissolves the marriage of the king", func(t *testing.T) {
		g := queenState()
		requireQueen(t, g, "Q1")
		g.Marriages[0].Dissolved = true
		requireQueen(t, g, "")
	})
	t.Run("the queen dies before the king", func(t *testing.T) {
		g := queenState()
		requireQueen(t, g, "Q1")
		killNoble(g, "Q1")
		requireQueen(t, g, "")
	})
}

func TestRefreshQueenFindsTheWidowOfAKingWhoJustDied(t *testing.T) {
	g := queenState()
	killNoble(g, "K1") // Queen was never refreshed
	requireQueen(t, g, "Q1")
}

func TestRoyalIssuerIsTheKingThenTheRegent(t *testing.T) {
	g := queenState()
	requireQueen(t, g, "Q1")
	if issuer, ok := g.RoyalIssuer(); !ok || issuer.ID != "K1" {
		t.Fatalf("issuer = %v/%v, want the king K1", issuer.ID, ok)
	}
	if _, ok := g.Regent(); ok {
		t.Error("a reigning king leaves no regent")
	}

	// A hostage king still reigns.
	g.Nobles[0].Status = models.NobleStatusHostage
	if issuer, ok := g.RoyalIssuer(); !ok || issuer.ID != "K1" {
		t.Fatalf("issuer = %v/%v, want the hostage king K1", issuer.ID, ok)
	}

	// A king in a dungeon is replaced by his queen, hostage or free.
	g.Nobles[0].Status = models.NobleStatusDungeon
	g.Nobles[1].Status = models.NobleStatusHostage
	if issuer, ok := g.RoyalIssuer(); !ok || issuer.ID != "Q1" {
		t.Fatalf("issuer = %v/%v, want the regent Q1", issuer.ID, ok)
	}
	g.Nobles[1].Status = models.NobleStatusDungeon
	if _, ok := g.RoyalIssuer(); ok {
		t.Error("a queen in a dungeon cannot be regent")
	}

	// A vacant throne is held by the dowager queen.
	g.Nobles[1].Status = models.NobleStatusFree
	killNoble(g, "K1")
	g.King = nil
	requireQueen(t, g, "Q1")
	if issuer, ok := g.RoyalIssuer(); !ok || issuer.ID != "Q1" {
		t.Fatalf("issuer = %v/%v, want the dowager regent Q1", issuer.ID, ok)
	}

	// She loses it when she remarries.
	g.Marriages = append(g.Marriages, models.Marriage{NobleA: "Q1", NobleB: "X", Turn: 3})
	requireQueen(t, g, "")
	if _, ok := g.RoyalIssuer(); ok {
		t.Error("a remarried widow is no regent")
	}
}
