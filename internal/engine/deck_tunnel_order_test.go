package engine

import (
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// tunnelTestState is P1's capital castle on AAA, a P2 army on BBB holding the
// P1 noble ONE hostage, and one Tunnel card in P1's hand.
func tunnelTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "BBB", Size: 1}},
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	setCapital(state, "P1", "I1")
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "BBB")
	setNobleStatus(state, "N1", models.NobleStatusHostage)
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindTunnel}},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}},
	}
	return state
}

func playTunnel(order models.DeckOrder) map[models.PlayerID][]models.DeckOrder {
	order.ID, order.Type, order.Kind = "O1", models.DeckOrderTypePlay, models.CardKindTunnel
	return map[models.PlayerID][]models.DeckOrder{"P1": {order}}
}

func TestTunnelEscapeFreesTheNobleOnTheCapital(t *testing.T) {
	for _, status := range []models.NobleStatus{models.NobleStatusHostage, models.NobleStatusDungeon} {
		state := tunnelTestState(t)
		setNobleStatus(state, "N1", status)
		validateTestState(t, state)
		resolution, err := ResolveWithDeckOrders(state, testBalance(), playTunnel(models.DeckOrder{TargetNobleID: "N1"}))
		if err != nil {
			t.Fatalf("%s: ResolveWithDeckOrders: %v", status, err)
		}
		noble := nobleByID(t, resolution.State, "N1")
		if noble.Status != models.NobleStatusFree || noble.LocationID != "AAA" {
			t.Fatalf("%s: noble = %#v, want free on the capital AAA", status, noble)
		}
		if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 0 {
			t.Fatalf("%s: hand = %#v, want the card consumed", status, got)
		}
		events := eventsOfType(resolution.Events, EventTypeTunnel)
		if len(events) != 1 || events[0].Reason != TunnelEscapeReason || events[0].CaptorPlayerID != "P2" ||
			events[0].OwnerID != "P1" || events[0].NobleCode != "ONE" || events[0].TerritoryID != "AAA" || events[0].TargetID != "BBB" {
			t.Fatalf("%s: tunnel events = %#v", status, events)
		}
	}
}

func TestTunnelEscapeFallsBackOnTheSmallestControlledCastle(t *testing.T) {
	state := tunnelTestState(t)
	// An enemy army sits on the capital, so the noble cannot reappear there.
	addAnchorArmy(t, state, "A2", "P2", "AAA")
	// P1 holds two castles with its own armies: the smaller code wins.
	addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CCC"})
	addAnchorArmy(t, state, "A3", "P1", "CCC")
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), playTunnel(models.DeckOrder{TargetNobleID: "N1"}))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.Status != models.NobleStatusFree || noble.LocationID != "CCC" {
		t.Fatalf("noble = %#v, want free on the castle CCC", noble)
	}
}

func TestTunnelIsRejectedWithItsReasonAndKeepsTheCard(t *testing.T) {
	cases := map[string]struct {
		prepare func(*models.GameState)
		order   models.DeckOrder
		reason  string
	}{
		"noble not detained": {
			prepare: func(state *models.GameState) { setNobleStatus(state, "N1", models.NobleStatusFree) },
			order:   models.DeckOrder{TargetNobleID: "N1"},
			reason:  "noble_not_held",
		},
		"noble of another player": {
			prepare: func(state *models.GameState) {
				addNoble(state, "N2", "TWO", "P3", "BBB")
				setNobleStatus(state, "N2", models.NobleStatusHostage)
			},
			order:  models.DeckOrder{TargetNobleID: "N2"},
			reason: "noble_not_held",
		},
		"noble held by its own army": {
			prepare: func(state *models.GameState) { state.Armies[0].OwnerID = "P1" },
			order:   models.DeckOrder{TargetNobleID: "N1"},
			reason:  "noble_not_held",
		},
		"no refuge": {
			prepare: func(state *models.GameState) { addAnchorArmy(t, state, "A2", "P2", "AAA") },
			order:   models.DeckOrder{TargetNobleID: "N1"},
			reason:  "tunnel_no_refuge",
		},
		"own army without a planned attack": {
			prepare: func(state *models.GameState) { addAnchorArmy(t, state, "A2", "P1", "AAA") },
			order:   models.DeckOrder{SourceTerritoryID: "AAA", TargetTerritoryID: "BBB"},
			reason:  "tunnel_no_attack",
		},
		"another player's army, territories not adjacent": {
			order:  models.DeckOrder{SourceTerritoryID: "BBB", TargetTerritoryID: "BBB"},
			reason: "tunnel_no_attack",
		},
		"no army on the origin": {
			order:  models.DeckOrder{SourceTerritoryID: "CCC", TargetTerritoryID: "BBB"},
			reason: "tunnel_no_attack",
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			state := tunnelTestState(t)
			if test.prepare != nil {
				test.prepare(state)
			}
			validateTestState(t, state)
			_, err := ResolveWithDeckOrders(state, testBalance(), playTunnel(test.order))
			if err == nil || !strings.Contains(err.Error(), test.reason) {
				t.Fatalf("error = %v, want the reason %q", err, test.reason)
			}
			if got := state.SpecialDeck.Hands["P1"]; len(got) != 1 {
				t.Fatalf("hand = %#v, want the card kept", got)
			}
		})
	}
}

// tunnelAttackState is a P2 army of the given size on BBB attacking AAA, where
// P1 holds a castle (its capital) with a one-troop army, and a Tunnel card in
// P2's hand. With the castle bonus of 1 the defense is 2, and the attacker's
// force is its size plus the command bonus of its noble (1).
func tunnelAttackState(t *testing.T, attackerSize int, defended bool) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "BBB", Size: attackerSize}},
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	setCapital(state, "P1", "I1")
	if defended {
		addAnchorArmy(t, state, "A2", "P1", "AAA")
	}
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P2", "BBB")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindTunnel}},
		DrawPile: []models.SpecialCardID{}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P2": {"C1"}},
	}
	return state
}

func tunnelAttackOrders(played bool) map[models.PlayerID][]models.DeckOrder {
	if !played {
		return nil
	}
	return map[models.PlayerID][]models.DeckOrder{"P2": {{
		ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindTunnel,
		SourceTerritoryID: "BBB", TargetTerritoryID: "AAA",
	}}}
}

func TestTunnelAttackIgnoresTheFortificationsOfItsTarget(t *testing.T) {
	cases := map[string]struct {
		attackerSize int
		defended     bool
	}{
		// Force 2 only ties the one-troop army behind the castle bonus (1 + 1).
		"defended castle": {attackerSize: 1, defended: true},
		// An empty castle holds against force 1 only: the noble's bonus beats it,
		// so the card changes nothing there but is still consumed and reported.
		"empty castle": {attackerSize: 1, defended: false},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			without := tunnelAttackState(t, test.attackerSize, test.defended)
			validateTestState(t, without)
			resolution, err := ResolveWithDeckOrders(without, testBalance(), tunnelAttackOrders(false))
			if err != nil {
				t.Fatalf("ResolveWithDeckOrders: %v", err)
			}
			wantWithout := models.TerritoryID("AAA")
			if test.defended {
				wantWithout = "BBB"
			}
			if got := armyByID(t, resolution.State, "A1").TerritoryID; got != wantWithout {
				t.Fatalf("without the card A1 is on %s, want %s", got, wantWithout)
			}

			with := tunnelAttackState(t, test.attackerSize, test.defended)
			validateTestState(t, with)
			resolution, err = ResolveWithDeckOrders(with, testBalance(), tunnelAttackOrders(true))
			if err != nil {
				t.Fatalf("ResolveWithDeckOrders: %v", err)
			}
			if got := armyByID(t, resolution.State, "A1").TerritoryID; got != "AAA" {
				t.Fatalf("with the card A1 is on %s, want the attack to succeed", got)
			}
			events := eventsOfType(resolution.Events, EventTypeTunnel)
			if len(events) != 1 || events[0].Reason != TunnelBypassReason || events[0].TerritoryID != "AAA" || events[0].SourceID != "BBB" {
				t.Fatalf("tunnel events = %#v", events)
			}
			if got := resolution.State.SpecialDeck.Hands["P2"]; len(got) != 0 {
				t.Fatalf("hand = %#v, want the card consumed", got)
			}
			combat, found := combatAt(resolution.Events, "AAA")
			if !found {
				t.Fatalf("no combat reported at AAA")
			}
			tunneled := false
			for _, contender := range combat.Contenders {
				tunneled = tunneled || contender.Tunnel
			}
			if !tunneled {
				t.Fatalf("combat contenders = %#v, want the tunneled attack flagged", combat.Contenders)
			}
		})
	}
}

func TestTunnelAttackIsRejectedForAnotherDestination(t *testing.T) {
	state := tunnelAttackState(t, 1, false)
	orders := tunnelAttackOrders(true)
	orders["P2"][0].TargetTerritoryID = "BBB"
	_, err := ResolveWithDeckOrders(state, testBalance(), orders)
	if err == nil || !strings.Contains(err.Error(), "tunnel_no_attack") {
		t.Fatalf("error = %v, want tunnel_no_attack", err)
	}
}

func TestTunnelIsParsedByItsArgument(t *testing.T) {
	state := tunnelTestState(t)
	parsed, parseErrors := orders.ParseDeckOrders("P SO ONE\nP TU BBB - AAA\n", state)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseDeckOrders: %v", parseErrors)
	}
	if len(parsed) != 2 || parsed[0].Kind != models.CardKindTunnel || parsed[0].TargetNobleID != "N1" || parsed[0].SourceTerritoryID != "" ||
		parsed[1].Kind != models.CardKindTunnel || parsed[1].SourceTerritoryID != "BBB" || parsed[1].TargetTerritoryID != "AAA" || parsed[1].TargetNobleID != "" {
		t.Fatalf("orders = %#v", parsed)
	}
	for _, line := range []string{"P SO ZZZ", "P SO BBB - ZZZ", "P SO BBB AAA ZZZ", "P SO BBB / AAA"} {
		if _, parseErrors := orders.ParseDeckOrders(line+"\n", state); len(parseErrors) == 0 {
			t.Errorf("%q must be rejected", line)
		}
	}
}

// Only the designated attack ignores the fortifications: a second attacker on
// the same castle, with no card, still faces them. The rival has to be
// strictly weaker than the tunneled attack (force 2 against 3) and carry no
// noble, otherwise it would also prevent the designated attack; and it must
// beat the defense without fortifications (1) to tell the two attacks apart.
func TestTunnelAttackOnlyBypassesTheFortificationsForTheDesignatedAttack(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB", "CCC"), territory("BBB", "BBB", "AAA"), territory("CCC", "CCC", "AAA")},
		[]models.Army{
			{ID: "A1", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
			{ID: "A3", OwnerID: "P3", TerritoryID: "CCC", Size: 2},
		},
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	setCapital(state, "P1", "I1")
	addAnchorArmy(t, state, "A2", "P1", "AAA")
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P2", "BBB")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	// The rival's noble is elsewhere, so it commands nothing.
	addNoble(state, "N3", "TRI", "P3", "BBB")
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindTunnel}},
		DrawPile: []models.SpecialCardID{}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P2": {"C1"}},
	}
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), tunnelAttackOrders(true))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if got := armyByID(t, resolution.State, "A1").TerritoryID; got != "AAA" {
		t.Fatalf("A1 is on %s, want the tunneled attack to take AAA", got)
	}
	if got := armyByID(t, resolution.State, "A3").TerritoryID; got != "CCC" {
		t.Fatalf("A3 is on %s, want the rival attack to stay home", got)
	}
	combat, found := combatAt(resolution.Events, "AAA")
	if !found {
		t.Fatalf("no combat reported at AAA")
	}
	for _, contender := range combat.Contenders {
		switch {
		case contender.ArmyID == "A1" && !contender.Tunnel:
			t.Errorf("designated attack %#v not flagged as tunneled", contender)
		case contender.ArmyID == "A3" && contender.Tunnel:
			t.Errorf("rival attack %#v flagged as tunneled", contender)
		case contender.Defender && (contender.Force != 2 || contender.TunnelForce == nil || *contender.TunnelForce != 1):
			t.Errorf("defender %#v, want force 2 (castle bonus included) and 1 against a tunnel", contender)
		}
	}
}

func TestTunnelOnAnotherPlayersAttackNeedsNoPlannedAttack(t *testing.T) {
	state := tunnelAttackState(t, 1, true)
	state.SpecialDeck.Hands = map[models.PlayerID][]models.SpecialCardID{"P3": {"C1"}}
	validateTestState(t, state)
	orders := tunnelAttackOrders(true)
	orders["P3"], orders["P2"] = orders["P2"], nil
	resolution, err := ResolveWithDeckOrders(state, testBalance(), orders)
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if got := armyByID(t, resolution.State, "A1").TerritoryID; got != "AAA" {
		t.Fatalf("A1 is on %s, want the advised attack to bypass the fortifications", got)
	}
	events := eventsOfType(resolution.Events, EventTypeTunnel)
	if len(events) != 1 || events[0].Reason != TunnelBypassReason || events[0].OwnerID != "P3" {
		t.Fatalf("tunnel events = %#v", events)
	}
}
