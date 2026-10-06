package models_test

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestSuccessionLineFollowsPurchaseOrder(t *testing.T) {
	g := models.NewGameState()
	g.Players = []models.Player{{ID: "P1"}, {ID: "P2"}}
	g.Nobles = []models.Noble{
		{ID: "N10", OwnerID: "P1"},
		{ID: "N2", OwnerID: "P1"},
		{ID: "N3", OwnerID: "P2"},
		{ID: "N7", OwnerID: "P1", Status: models.NobleStatusDungeon},
	}
	g.RemovedNobles = []models.RemovedNoble{{ID: "N1", OwnerID: "P1"}}

	var got []models.NobleID
	for _, n := range g.SuccessionLine("P1") {
		got = append(got, n.ID)
	}
	want := []models.NobleID{"N2", "N7", "N10"}
	if len(got) != len(want) {
		t.Fatalf("line = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line = %v, want %v", got, want)
		}
	}
	lines := g.SuccessionLines()
	if len(lines["P2"]) != 1 || len(lines["P1"]) != 3 {
		t.Errorf("SuccessionLines = %v", lines)
	}
}

func TestSuccessionLinePlacesBastardsLast(t *testing.T) {
	g := models.NewGameState()
	g.Players = []models.Player{{ID: "P1"}}
	g.Nobles = []models.Noble{
		{ID: "N1", OwnerID: "P1", Dignities: []models.Dignity{models.DignityBastard}},
		{ID: "N2", OwnerID: "P1"},
		{ID: "N3", OwnerID: "P1", Dignities: []models.Dignity{models.DignityBastard}},
		{ID: "N4", OwnerID: "P1"},
	}
	var got []models.NobleID
	for _, n := range g.SuccessionLine("P1") {
		got = append(got, n.ID)
	}
	want := []models.NobleID{"N2", "N4", "N1", "N3"}
	for i := range want {
		if len(got) != len(want) || got[i] != want[i] {
			t.Fatalf("line = %v, want %v", got, want)
		}
	}
}

func TestCanReceiveTitleBastardOnlyWhenLastOfLineage(t *testing.T) {
	g := models.NewGameState()
	g.Players = []models.Player{{ID: "P1"}}
	g.Nobles = []models.Noble{
		{ID: "N1", OwnerID: "P1", Dignities: []models.Dignity{models.DignityBastard}},
		{ID: "N2", OwnerID: "P1"},
	}
	if g.CanReceiveTitle("N1", models.FiefTitleBarony) {
		t.Error("bastard N1 may receive a title while the non-bastard N2 lives")
	}
	if !g.CanReceiveTitle("N2", models.FiefTitleBarony) {
		t.Error("non-bastard N2 cannot receive a title as head of the line")
	}
	g.Nobles = g.Nobles[:1]
	if !g.CanReceiveTitle("N1", models.FiefTitleBarony) {
		t.Error("the last of the lineage, a bastard, cannot receive a title")
	}
}

func TestDignityEffects(t *testing.T) {
	plain := models.Noble{}
	bastard := models.Noble{Dignities: []models.Dignity{models.DignityBastard}}
	if plain.IsBastard() || plain.NobleLimitBonus() != 0 || plain.LastInSuccession() || !plain.CanBeKing() || !plain.MarriageIsAlliance() || plain.CapturedToDungeon() {
		t.Errorf("a noble without dignity must be unaffected: %+v", plain)
	}
	if !bastard.IsBastard() || bastard.NobleLimitBonus() != 1 || !bastard.LastInSuccession() || bastard.CanBeKing() || bastard.MarriageIsAlliance() || !bastard.CapturedToDungeon() {
		t.Errorf("a bastard must carry every bastard effect: %+v", bastard)
	}
	if dignity, ok := models.DignityForCardCode("BAS"); !ok || dignity != models.DignityBastard {
		t.Errorf("DignityForCardCode(BAS) = %q, %v", dignity, ok)
	}
	if _, ok := models.DignityForCardCode("XXX"); ok {
		t.Error("DignityForCardCode(XXX) found a dignity")
	}
}

func TestValidateNobleDeck(t *testing.T) {
	newState := func() *models.GameState {
		g := models.NewGameState()
		g.Players = []models.Player{{ID: "P1"}}
		g.NobleDeck = &models.NobleDeck{
			Cards: []models.NobleCard{
				{ID: "K1", Kind: models.NobleCardKindNoble, Code: "ELE", Name: "Eleonore", Sex: models.SexFemale},
				{ID: "K2", Kind: models.NobleCardKindDignity, Code: "BAS", Dignity: models.DignityBastard},
				{ID: "K3", Kind: models.NobleCardKindDignity, Code: "BAS", Dignity: models.DignityBastard},
			},
			DrawPile: []models.NobleCardID{"K1"},
			Discard:  []models.NobleCardID{"K3"},
			Hands:    map[models.PlayerID][]models.NobleCardID{"P1": {"K2"}},
			Played:   []models.NobleCardPlay{},
			NamePool: []models.NobleName{{Code: "GUI", Name: "Guy", Sex: models.SexMale}},
		}
		return g
	}
	if err := newState().Validate(); err != nil {
		t.Fatalf("valid deck: %v", err)
	}
	tests := map[string]func(*models.GameState){
		"card in two places": func(g *models.GameState) { g.NobleDeck.Discard = append(g.NobleDeck.Discard, "K1") },
		"card nowhere":       func(g *models.GameState) { g.NobleDeck.Discard = nil },
		"unknown discard":    func(g *models.GameState) { g.NobleDeck.Discard = append(g.NobleDeck.Discard, "K9") },
		"played on unknown noble": func(g *models.GameState) {
			g.NobleDeck.Discard = nil
			g.NobleDeck.Played = []models.NobleCardPlay{{Card: "K3", Noble: "N9"}}
		},
		"dignity card on noble without it": func(g *models.GameState) {
			g.NobleDeck.Discard = nil
			g.NobleDeck.Played = []models.NobleCardPlay{{Card: "K3", Noble: "N1"}}
			g.Nobles = []models.Noble{{ID: "N1", Code: "ABC", Sex: models.SexMale, OwnerID: "P1", Status: models.NobleStatusFree}}
		},
		"pool name used by a card": func(g *models.GameState) {
			g.NobleDeck.NamePool = []models.NobleName{{Code: "ELE", Name: "Eleonore", Sex: models.SexFemale}}
		},
		"invalid pool name": func(g *models.GameState) {
			g.NobleDeck.NamePool = []models.NobleName{{Code: "gu", Name: "Guy", Sex: models.SexMale}}
		},
		"unknown card":   func(g *models.GameState) { g.NobleDeck.DrawPile = append(g.NobleDeck.DrawPile, "K9") },
		"unknown player": func(g *models.GameState) { g.NobleDeck.Hands["P9"] = nil },
		"duplicate noble code": func(g *models.GameState) {
			g.NobleDeck.Cards[1] = models.NobleCard{ID: "K2", Kind: models.NobleCardKindNoble, Code: "ELE", Name: "E", Sex: models.SexFemale}
		},
		"bad dignity code": func(g *models.GameState) { g.NobleDeck.Cards[1].Code = "XXX" },
		"noble code in play": func(g *models.GameState) {
			g.Nobles = []models.Noble{{ID: "N1", Code: "ELE", Sex: models.SexFemale, OwnerID: "P1", Status: models.NobleStatusFree}}
		},
		"unknown noble dignity": func(g *models.GameState) {
			g.Nobles = []models.Noble{{ID: "N1", Code: "ABC", Sex: models.SexMale, OwnerID: "P1", Status: models.NobleStatusFree, Dignities: []models.Dignity{"king"}}}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			g := newState()
			mutate(g)
			if g.Nobles != nil && len(g.Nobles) > 0 {
				g.Territories = []models.Territory{{ID: "AAA", Terrain: models.TerrainPlain}}
				g.TerritoryStates = map[models.TerritoryID]models.TerritoryState{"AAA": {}}
				g.Nobles[0].LocationID = "AAA"
			}
			if err := g.Validate(); err == nil {
				t.Error("Validate accepted an invalid deck state")
			}
		})
	}
}

func TestValidateNobleDeckPlayedCards(t *testing.T) {
	g := models.NewGameState()
	g.Players = []models.Player{{ID: "P1"}}
	g.Territories = []models.Territory{{ID: "AAA", Terrain: models.TerrainPlain}}
	g.TerritoryStates = map[models.TerritoryID]models.TerritoryState{"AAA": {}}
	g.Nobles = []models.Noble{
		{ID: "N1", Code: "ELE", Sex: models.SexFemale, OwnerID: "P1", LocationID: "AAA", Status: models.NobleStatusFree, Dignities: []models.Dignity{models.DignityBastard}},
	}
	g.NobleDeck = &models.NobleDeck{
		Cards: []models.NobleCard{
			{ID: "K1", Kind: models.NobleCardKindNoble, Code: "ELE", Name: "Eleonore", Sex: models.SexFemale},
			{ID: "K2", Kind: models.NobleCardKindDignity, Code: "BAS", Dignity: models.DignityBastard},
		},
		DrawPile: []models.NobleCardID{},
		Discard:  []models.NobleCardID{},
		Hands:    map[models.PlayerID][]models.NobleCardID{"P1": {}},
		Played:   []models.NobleCardPlay{{Card: "K1", Noble: "N1"}, {Card: "K2", Noble: "N1"}},
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("valid played cards: %v", err)
	}
	g.NobleDeck.Played[0].Noble = "N9"
	if err := g.Validate(); err == nil {
		t.Error("Validate accepted a card played on an unknown noble")
	}
}

func TestValidateClaims(t *testing.T) {
	build := func(claims ...models.Claim) *models.GameState {
		g := models.NewGameState()
		g.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
		g.Territories = []models.Territory{{ID: "AAA", Name: "AAA", Terrain: models.TerrainPlain}}
		g.TerritoryStates = map[models.TerritoryID]models.TerritoryState{"AAA": {}}
		g.Nobles = []models.Noble{
			{ID: "N1", Code: "ONE", Name: "One", Sex: models.SexMale, OwnerID: "P1", LocationID: "AAA", Status: models.NobleStatusFree},
			{ID: "N2", Code: "TWO", Name: "Two", Sex: models.SexFemale, OwnerID: "P2", LocationID: "AAA", Status: models.NobleStatusFree},
			{ID: "N3", Code: "TRE", Name: "Tre", Sex: models.SexMale, OwnerID: "P1", LocationID: "AAA", Status: models.NobleStatusFree},
		}
		g.Claims = claims
		return g
	}
	if err := build(models.Claim{Heir: "N1", Target: "N2", Spouse: "N3"}).Validate(); err != nil {
		t.Fatalf("valid claim rejected: %v", err)
	}
	for name, claims := range map[string][]models.Claim{
		"unknown heir":   {{Heir: "N9", Target: "N2", Spouse: "N3"}},
		"unknown target": {{Heir: "N1", Target: "N9", Spouse: "N3"}},
		"unknown spouse": {{Heir: "N1", Target: "N2", Spouse: "N9"}},
		"same owner":     {{Heir: "N1", Target: "N3", Spouse: "N2"}},
		"two claims":     {{Heir: "N1", Target: "N2", Spouse: "N3"}, {Heir: "N1", Target: "N2", Spouse: "N3"}},
		"future turn":    {{Heir: "N1", Target: "N2", Spouse: "N3", Turn: 5}},
	} {
		if err := build(claims...).Validate(); err == nil {
			t.Errorf("%s: invalid claims accepted", name)
		}
	}
}
