package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// taxTestState builds a 3-territory barony (FCP capital, MEM carrying a
// village, OTH bare) owned by P1, with a fresh seigneurial tax card in its
// hand, mirroring the #189 hotseat scenario: "3-territory barony, 1 village,
// +8 R instead of +4 when taxed".
func taxTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{
			territory("PCP", "PCP", "FCP"),
			territory("FCP", "FCP", "PCP", "MEM"),
			territory("MEM", "MEM", "FCP", "OTH"),
			territory("OTH", "OTH", "MEM"),
		},
		nil,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "PCP"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "FCP"})
	addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "MEM"})
	setCapital(state, "P1", "I1")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "FCP",
		Territories: []models.TerritoryID{"FCP", "MEM", "OTH"}, OwnerID: "P1",
	}}
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindSeigneurialTax}, {ID: "C2", Kind: models.CardKindSeigneurialTax}},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C1", "C2"}},
	}
	return state
}

// TestSeigneurialTaxDoublesFiefIncome mirrors the #189 hotseat test 1: the
// seigneur taxes his barony's capital and its territorial income (village
// included) doubles for the turn, while mills stay untouched.
func TestSeigneurialTaxDoublesFiefIncome(t *testing.T) {
	state := taxTestState(t)
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetTerritoryID: "FCP"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	fiefEvent := incomeEventForFief(t, resolution.Events, "P1", "FCP", "F1")
	if fiefEvent.Production != 8 {
		t.Fatalf("fief income event = %#v, want 8 R (doubled from 4: 3 territories + 1 village)", fiefEvent)
	}
	if got := resolution.State.TerritoryStates["FCP"].Resources; got != 8 {
		t.Errorf("FCP stock = %d, want 8", got)
	}
	applied := false
	for _, event := range resolution.Events {
		if event.Type == EventTypeBonusEffect && event.CardKind == models.CardKindSeigneurialTax && event.FiefID == "F1" {
			applied = true
		}
	}
	if !applied {
		t.Fatalf("events = %#v, want a bonus_effect event for the applied tax", resolution.Events)
	}
	if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 1 || got[0] != "C2" {
		t.Fatalf("P1 hand = %#v, want the played card consumed", got)
	}
}

// TestSeigneurialTaxRejectedForNonOwner mirrors hotseat test 2: playing the
// tax as a player who does not hold the targeted fief is rejected
// explicitly.
func TestSeigneurialTaxRejectedForNonOwner(t *testing.T) {
	state := taxTestState(t)
	state.SpecialDeck.Hands = map[models.PlayerID][]models.SpecialCardID{"P2": {"C1", "C2"}}
	validateTestState(t, state)
	_, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P2": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetTerritoryID: "FCP"}},
	})
	if err == nil {
		t.Fatal("ResolveWithDeckOrders: want an error rejecting a tax played by a non-owner")
	}
}

// TestSeigneurialTaxRejectedOnNonFiefCapital mirrors hotseat test 2's other
// case: XXX must be a fief capital, not any region seed or plain territory.
func TestSeigneurialTaxRejectedOnNonFiefCapital(t *testing.T) {
	state := taxTestState(t)
	validateTestState(t, state)
	_, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetTerritoryID: "MEM"}},
	})
	if err == nil {
		t.Fatal("ResolveWithDeckOrders: want an error rejecting a tax targeting a non-capital territory")
	}
}

// TestSeigneurialTaxNoStackingOnSameFief mirrors #189's "no stacking"
// decision: a second tax on the same fief the same turn is consumed with no
// effect, reported explicitly, and does not further double the income.
func TestSeigneurialTaxNoStackingOnSameFief(t *testing.T) {
	state := taxTestState(t)
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {
			{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetTerritoryID: "FCP"},
			{ID: "O2", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetTerritoryID: "FCP"},
		},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	fiefEvent := incomeEventForFief(t, resolution.Events, "P1", "FCP", "F1")
	if fiefEvent.Production != 8 {
		t.Fatalf("fief income event = %#v, want a single doubling (8 R), not stacked", fiefEvent)
	}
	deniedFound := false
	for _, event := range resolution.Events {
		if event.Type == EventTypeCardCanceled && event.CardKind == models.CardKindSeigneurialTax &&
			event.Reason == "seigneurial_tax_already_applied" && event.FiefID == "F1" {
			deniedFound = true
		}
	}
	if !deniedFound {
		t.Fatalf("events = %#v, want the second tax reported as consumed without effect", resolution.Events)
	}
	if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 0 {
		t.Fatalf("P1 hand = %#v, want both cards consumed", got)
	}
	if got := resolution.State.SpecialDeck.Discard; len(got) != 2 {
		t.Fatalf("discard = %#v, want both consumed cards in the discard, none restored", got)
	}
}

// TestSeigneurialTaxOpensRevoltWindowOnEveryFiefTerritory mirrors hotseat
// test 3: taxing the fief's capital allows Révolte on any territory of the
// fief, capital included, independently of famine, for the turn the tax is
// played and the following one, and no longer after that.
func TestSeigneurialTaxOpensRevoltWindowOnEveryFiefTerritory(t *testing.T) {
	state := taxTestState(t)
	state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, models.SpecialCard{ID: "C3", Kind: models.CardKindRevolt})
	state.SpecialDeck.Hands["P1"] = append(state.SpecialDeck.Hands["P1"], "C3")
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())
	definition := cardDefinitions[models.CardKindRevolt]

	// Before any tax is played, no famine and no tax window: OTH is not
	// eligible.
	if ok, _ := definition.CanPlay(&ExecutionContext{resolution: ctx, season: models.SeasonSpring}, models.DeckOrder{TargetTerritoryID: "OTH"}); ok {
		t.Fatalf("revolt on OTH before any tax = %t, want false", ok)
	}

	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetTerritoryID: "FCP"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}

	// Same turn: every fief territory, not just the capital, is eligible.
	sameTurnCtx := newResolutionContext(resolution.State, testBalance())
	for _, territoryID := range []models.TerritoryID{"FCP", "MEM", "OTH"} {
		if ok, reason := definition.CanPlay(&ExecutionContext{resolution: sameTurnCtx, season: models.SeasonSpring}, models.DeckOrder{TargetTerritoryID: territoryID}); !ok {
			t.Fatalf("revolt on %q the turn the tax is played = %t/%q, want true", territoryID, ok, reason)
		}
	}

	// The following turn: still eligible.
	nextTurnState := cloneGameState(resolution.State)
	nextTurnState.Turn = resolution.State.Turn + 1
	nextTurnCtx := newResolutionContext(nextTurnState, testBalance())
	if ok, reason := definition.CanPlay(&ExecutionContext{resolution: nextTurnCtx, season: models.SeasonSummer}, models.DeckOrder{TargetTerritoryID: "OTH"}); !ok {
		t.Fatalf("revolt on OTH the turn after the tax = %t/%q, want true", ok, reason)
	}

	// Two turns later: no longer eligible without famine.
	laterState := cloneGameState(resolution.State)
	laterState.Turn = resolution.State.Turn + 2
	laterCtx := newResolutionContext(laterState, testBalance())
	if ok, _ := definition.CanPlay(&ExecutionContext{resolution: laterCtx, season: models.SeasonAutumn}, models.DeckOrder{TargetTerritoryID: "OTH"}); ok {
		t.Fatalf("revolt on OTH two turns after the tax = %t, want false", ok)
	}
}

// TestSeigneurialTaxDoublingCanceledWhenCapitalCapturedSameTurn covers #208:
// the fief's capital changing hands the same turn the tax is played must not
// let the doubling either apply to the old owner (who no longer controls the
// fief by the time income credits, after captures) or leak to the conqueror
// (who never played the tax). It must simply not apply, with no error.
func TestSeigneurialTaxDoublingCanceledWhenCapitalCapturedSameTurn(t *testing.T) {
	state := taxTestState(t)
	state.Territories = append(state.Territories, territory("ENY", "ENY", "FCP"))
	state.TerritoryStates["ENY"] = models.TerritoryState{}
	for i := range state.Territories {
		if state.Territories[i].ID == "FCP" {
			state.Territories[i].Adjacencies = append(state.Territories[i].Adjacencies, "ENY")
		}
	}
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "ENY", Size: 3}}
	enyState := state.TerritoryStates["ENY"]
	armyID := models.ArmyID("A1")
	enyState.Army = &armyID
	state.TerritoryStates["ENY"] = enyState
	state.NextArmyID = nextArmyID(state.Armies)
	addNoble(state, "N2", "TWO", "P2", "ENY")
	addChain(t, state, "A1", "N2", models.Order{
		Type: models.OrderTypeAttack, PositionID: "ENY", TargetIDs: []models.TerritoryID{"FCP"},
	})
	validateTestState(t, state)

	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetTerritoryID: "FCP"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if resolution.State.Fiefs[0].OwnerID != "P2" {
		t.Fatalf("fief owner = %q, want P2 after the uncontested capture", resolution.State.Fiefs[0].OwnerID)
	}
	// P2, the conqueror, must receive the fief's normal (undoubled) income:
	// 3 territories plus MEM's village, 4 R, not 8.
	conquerorEvent := incomeEventForFief(t, resolution.Events, "P2", "FCP", "F1")
	if conquerorEvent.Production != 4 {
		t.Fatalf("conqueror income event = %#v, want 4 R, not doubled", conquerorEvent)
	}
	for _, event := range resolution.Events {
		if event.Type == EventTypeIncome && event.OwnerID == "P1" && event.FiefID == "F1" {
			t.Fatalf("events = %#v, want no income event for P1 on the fief it no longer controls", event)
		}
	}
}

// TestSeigneurialTaxAndRevoltInTheSameSubmission mirrors hotseat test 3's
// sharpest case: the tax and a revolt on another territory of the same
// fief, both played in the very same order submission (not a follow-up
// turn). resolveDeckOrders checks each order's CanPlay before
// resolveSeasonEffects has applied any card's effect, so this only works if
// the fiefs targeted by a same-batch tax are flagged on the real resolution
// context up front, not only on validateActionDeckOrders' throwaway one.
func TestSeigneurialTaxAndRevoltInTheSameSubmission(t *testing.T) {
	state := taxTestState(t)
	state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, models.SpecialCard{ID: "C3", Kind: models.CardKindRevolt})
	state.SpecialDeck.Hands["P1"] = append(state.SpecialDeck.Hands["P1"], "C3")
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {
			{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetTerritoryID: "FCP"},
			{ID: "O2", Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: "OTH"},
		},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	for _, event := range resolution.Events {
		if event.Type == EventTypeRejected && event.OrderID == "O2" {
			t.Fatalf("revolt on OTH in the same submission as the tax = rejected (%q), want accepted", event.Reason)
		}
	}
	playedRevolt := false
	for _, event := range resolution.Events {
		if event.Type == EventTypeDeckOrderPlayed && event.CardKind == models.CardKindRevolt {
			playedRevolt = true
		}
	}
	if !playedRevolt {
		t.Fatalf("events = %#v, want a deck_order_played event for the revolt", resolution.Events)
	}
}
