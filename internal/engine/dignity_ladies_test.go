package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func ladyState(t *testing.T, dignity models.Dignity) *models.GameState {
	t.Helper()
	state := deckOrdersState(t)
	addNoble(state, "N1", "ELE", "P1", "AAA")
	state.Nobles[0].Sex = models.SexFemale
	giveCard(state, "P1", models.NobleCard{Kind: models.NobleCardKindDignity, Code: dignity.Effect().CardCode, Dignity: dignity})
	state.NobleDeck.NamePool = []models.NobleName{
		{Code: "MAH", Name: "Mahaut", Sex: models.SexFemale},
		{Code: "GUI", Name: "Guillaume", Sex: models.SexMale},
	}
	return state
}

func playDignity(code string) models.WinterOrder {
	return models.WinterOrder{ID: "O1", Type: models.WinterOrderTypeDignity, NobleCode: "ELE", CardCode: code}
}

func TestLadyDignityIsRejectedOnMenAndMarriedLadies(t *testing.T) {
	state := ladyState(t, models.DignityDArc)
	addNoble(state, "N2", "GUY", "P1", "AAA")
	giveCard(state, "P1", models.NobleCard{Kind: models.NobleCardKindDignity, Code: "ARC", Dignity: models.DignityDArc})
	addNoble(state, "N3", "HUG", "P2", "AAA")
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3", Turn: 1}}
	state.Nobles[2].Sex = models.SexMale
	validateTestState(t, state)
	orders := []models.WinterOrder{
		{ID: "O1", Type: models.WinterOrderTypeDignity, NobleCode: "GUY", CardCode: "ARC"},
		{ID: "O2", Type: models.WinterOrderTypeDignity, NobleCode: "ELE", CardCode: "ARC"},
	}
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": orders})
	reasons := rejectionReasons(resolution.Events)
	if len(reasons) != 2 || reasons[0] != "dignity_female_only" || reasons[1] != "noble_married" {
		t.Errorf("rejections = %v, want dignity_female_only then noble_married", reasons)
	}
	if len(resolution.State.NobleDeck.Played) != 0 {
		t.Error("a rejected dignity card was consumed")
	}
}

func TestBlockedDignityForbidsMarriageButFreeOneDoesNot(t *testing.T) {
	for dignity, wantRejected := range map[models.Dignity]bool{models.DignityDArc: true, models.DignityAstrologer: false} {
		state := deckOrdersState(t)
		addNoble(state, "N1", "ELE", "P1", "AAA")
		addNoble(state, "N2", "GUY", "P2", "AAA")
		state.Nobles[0].Sex = models.SexFemale
		state.Nobles[0].Dignities = []models.Dignity{dignity}
		resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeMarriage, NobleCode: "ELE", SpouseCode: "GUY"}},
			"P2": {{ID: "O1", Type: models.WinterOrderTypeMarriage, NobleCode: "GUY", SpouseCode: "ELE"}},
		})
		married := len(resolution.State.Marriages) == 1
		if married == wantRejected {
			t.Errorf("%s: married = %v, want %v", dignity, married, !wantRejected)
		}
	}
}

func TestEonReplacesTheLadyByAMaleNobleAndKeepsHerSecret(t *testing.T) {
	state := ladyState(t, models.DignityChevalierDEon)
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {playDignity("EON")}})
	noble := resolution.State.Nobles[0]
	if noble.Sex != models.SexMale || noble.Code != "GUI" || noble.Name != "Guillaume" || !noble.Has(models.DignityChevalierDEon) {
		t.Errorf("noble = %+v, want the male identity taken from the pool", noble)
	}
	if noble.SecretCode != "ELE" || noble.SecretName != "ELE" || noble.SecretSex != models.SexFemale {
		t.Errorf("noble = %+v, want the lady's identity kept as a secret", noble)
	}
	if pool := resolution.State.NobleDeck.NamePool; len(pool) != 1 || pool[0].Code != "MAH" {
		t.Errorf("pool = %v, want the male name taken out", pool)
	}
	if !noble.Sex.CanHoldReligiousOrRoyalTitle() {
		t.Error("the Éon should hold the prerogatives of a man")
	}
	validateTestState(t, resolution.State)
}

func TestDArcAddsForceOnTopOfCommandBonus(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}})
	addNoble(state, "N1", "ELE", "P1", "AAA")
	state.Nobles[0].Sex = models.SexFemale
	state.Nobles[0].Dignities = []models.Dignity{models.DignityDArc}
	ctx := newResolutionContext(state, testBalance())
	if got := nobleCommandBonus(ctx, state.Armies[0]); got != 2 {
		t.Errorf("bonus = %d, want 2 (command bonus and D'Arc)", got)
	}
	state.Nobles[0].Status = models.NobleStatusDungeon
	ctx = newResolutionContext(state, testBalance())
	if got := nobleCommandBonus(ctx, state.Armies[0]); got != 0 {
		t.Errorf("prisoner bonus = %d, want 0", got)
	}
}

func TestHerbalistProtectsFromPlageOnItsCellAndNeighbours(t *testing.T) {
	state := effectTestState()
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 5}, {ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 5}}
	state.TerritoryStates["AAA"] = models.TerritoryState{Army: armyPointer("A1")}
	state.TerritoryStates["BBB"] = models.TerritoryState{Army: armyPointer("A2")}
	addNoble(state, "N1", "ELE", "P1", "BBB")
	state.Nobles[0].Sex = models.SexFemale
	state.Nobles[0].Dignities = []models.Dignity{models.DignityHerbalist}
	setCurrentCalamity(state, models.CardKindPlague, "AAA")
	balance := testBalance()
	balance.SpecialOrders.Effects.PlagueArmyDivisor = 2
	balance.SpecialOrders.Effects.PlagueNobleMortalityPercentage = 100
	ctx := newResolutionContext(state, balance)
	resolveSeasonEffects(ctx)
	if got := ctx.armiesByID["A1"].Size; got != 5 {
		t.Errorf("adjacent protected army = %d, want 5", got)
	}
	if got := ctx.armiesByID["A2"].Size; got == 5 {
		t.Errorf("rival army = %d, want it hit by the plague", got)
	}
	if len(ctx.state.Nobles) != 1 {
		t.Error("the herbalist died of plague")
	}
}

func TestHerbalistSparesRationsAndWitchBurdensRivals(t *testing.T) {
	state := effectTestState()
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}, {ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1}}
	state.TerritoryStates["AAA"] = models.TerritoryState{Army: armyPointer("A1")}
	state.TerritoryStates["BBB"] = models.TerritoryState{Army: armyPointer("A2")}
	base := armyCost(3, testBalance().CostBase)
	addNoble(state, "N1", "ELE", "P1", "AAA")
	state.Nobles[0].Sex = models.SexFemale
	state.Nobles[0].Dignities = []models.Dignity{models.DignityHerbalist}
	ctx := newResolutionContext(state, testBalance())
	if got := armyDemand(ctx, state.Armies[0]); got != base-2 {
		t.Errorf("demand = %d, want %d", got, base-2)
	}
	state.Nobles[0].Status = models.NobleStatusDungeon
	ctx = newResolutionContext(state, testBalance())
	if got := armyDemand(ctx, state.Armies[0]); got != base {
		t.Errorf("prisoner herbalist demand = %d, want %d", got, base)
	}
	// A hostage herbalist keeps her bonus and passes it to her holder.
	state.Nobles[0].Status = models.NobleStatusHostage
	state.Nobles[0].LocationID = "BBB"
	ctx = newResolutionContext(state, testBalance())
	if got := armyDemand(ctx, state.Armies[1]); got != 0 {
		t.Errorf("holder demand = %d, want 0 (cost 1 spared)", got)
	}
	// The witch burdens every rival army of her region, not her owner's.
	state.Nobles[0].Status = models.NobleStatusFree
	state.Nobles[0].LocationID = "AAA"
	state.Nobles[0].Dignities = []models.Dignity{models.DignityWitch}
	ctx = newResolutionContext(state, testBalance())
	if got := armyDemand(ctx, state.Armies[0]); got != base {
		t.Errorf("owner demand = %d, want %d", got, base)
	}
	if got := armyDemand(ctx, state.Armies[1]); got != 2 {
		t.Errorf("rival demand = %d, want 2", got)
	}
}

func TestCalamityVetoStrikesForecastCalamitiesOnce(t *testing.T) {
	state := deckOrdersState(t)
	addNoble(state, "N1", "ELE", "P1", "AAA")
	state.Nobles[0].Sex = models.SexFemale
	state.Nobles[0].Dignities = []models.Dignity{models.DignityAstrologer}
	var pile []models.SpecialCardID
	var cards []models.SpecialCard
	for index, kind := range []models.CardKind{models.CardKindPlague, models.CardKindFairWeather, models.CardKindFamine, models.CardKindBadWeather, models.CardKindPlague, models.CardKindFamine} {
		id := models.SpecialCardID(string(rune('a' + index)))
		cards = append(cards, models.SpecialCard{ID: id, Kind: kind})
		pile = append(pile, id)
	}
	for index := 0; index < 8; index++ {
		id := models.SpecialCardID("z" + string(rune('a'+index)))
		cards = append(cards, models.SpecialCard{ID: id, Kind: models.CardKindFairWeather})
		pile = append(pile, id)
	}
	state.SpecialDeck = &models.SpecialDeck{Cards: cards, DrawPile: pile, Discard: []models.SpecialCardID{}, Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {}, "P2": {}}}
	veto := func(id models.OrderID, indices ...int) models.WinterOrder {
		return models.WinterOrder{ID: id, Type: models.WinterOrderTypeCalamityVeto, NobleCode: "ELE", Indices: indices}
	}
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {veto("O1", 5), veto("O2", 1, 3), veto("O3", 1)},
	})
	reasons := rejectionReasons(resolution.Events)
	if len(reasons) != 2 || reasons[0] != "calamity_not_forecast" || reasons[1] != "calamity_veto_already_used" {
		t.Errorf("rejections = %v", reasons)
	}
	deck := resolution.State.SpecialDeck
	if len(deck.Discard) < 2 || deck.Discard[0] != "a" || deck.Discard[1] != "d" {
		t.Errorf("discard = %v, want the 1st and 3rd forecast calamities (a, d)", deck.Discard)
	}
	// A captor profits from the forecast but cannot strike calamities.
	state.Nobles[0].Status = models.NobleStatusHostage
	other := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P2": {veto("O1", 1)}})
	if reasons := rejectionReasons(other.Events); len(reasons) != 1 || reasons[0] != "noble_not_astrologer" {
		t.Errorf("captor veto rejections = %v", reasons)
	}
}

func TestFreeDignityMayBePlayedOnAMarriedLadyButNotABlockedOne(t *testing.T) {
	state := ladyState(t, models.DignityAstrologer)
	giveCard(state, "P1", models.NobleCard{Kind: models.NobleCardKindDignity, Code: "ARC", Dignity: models.DignityDArc})
	addNoble(state, "N3", "HUG", "P2", "AAA")
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3", Turn: 1}}
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {
		{ID: "O1", Type: models.WinterOrderTypeDignity, NobleCode: "ELE", CardCode: "AST"},
		{ID: "O2", Type: models.WinterOrderTypeDignity, NobleCode: "ELE", CardCode: "ARC"},
	}})
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 1 || reasons[0] != "noble_married" {
		t.Errorf("rejections = %v, want only the blocked dignity refused", reasons)
	}
	if !resolution.State.Nobles[0].Has(models.DignityAstrologer) {
		t.Error("the astrologer dignity was not conferred on the married lady")
	}
}

func TestAbbessNeedsARegionSeed(t *testing.T) {
	state := ladyState(t, models.DignityAbbess)
	state.Regions = []models.Region{{ID: "R1", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}}}
	play := func(id models.OrderID, region models.TerritoryID) models.WinterOrder {
		return models.WinterOrder{ID: id, Type: models.WinterOrderTypeDignity, NobleCode: "ELE", CardCode: "ABB", TerritoryID: region}
	}
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {play("O1", ""), play("O2", "BBB"), play("O3", "AAA")}})
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 2 || reasons[0] != "dignity_region_required" || reasons[1] != "dignity_region_unknown" {
		t.Errorf("rejections = %v", reasons)
	}
	if noble := resolution.State.Nobles[0]; noble.AbbeyRegion != "AAA" || !noble.Has(models.DignityAbbess) {
		t.Errorf("noble = %+v, want an abbess of AAA", noble)
	}
}

func TestDignityCanTargetANobleRecruitedEarlierInTheSameSheet(t *testing.T) {
	state := deckOrdersState(t)
	giveCard(state, "P1", models.NobleCard{Kind: models.NobleCardKindNoble, Code: "MAH", Name: "Mahaut", Sex: models.SexFemale})
	giveCard(state, "P1", models.NobleCard{Kind: models.NobleCardKindDignity, Code: "ARC", Dignity: models.DignityDArc})
	validateTestState(t, state)

	text := "R N MAH AAA\nD N MAH ARC"
	parsed, parseErrors := orders.ParseWinterOrders(text, state)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseWinterOrders(%q) = %v, want the recruited noble to be targetable", text, parseErrors)
	}
	if _, parseErrors := orders.ParseWinterOrders("D N MAH ARC\nR N MAH AAA", state); len(parseErrors) == 0 {
		t.Error("a dignity before the recruit was accepted")
	}
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": parsed})
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %v, want none", reasons)
	}
	found := false
	for _, noble := range resolution.State.Nobles {
		if noble.Code == "MAH" {
			found = noble.Has(models.DignityDArc)
		}
	}
	if !found {
		t.Error("the recruited noble did not receive the dignity")
	}
}
