package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func inquiry(id models.OrderID, target models.NobleCode) models.WinterOrder {
	return inquiryBy(id, "HUG", target)
}

func inquiryBy(id models.OrderID, inquirer, target models.NobleCode) models.WinterOrder {
	return models.WinterOrder{ID: id, Type: models.WinterOrderTypeInquiry, NobleCode: inquirer, TargetCode: target}
}

// inquiryTestState makes P1's HUG (N1) a cardinal with 20 R on AAA; LEO (N2,
// P2) is the target. OTO (N4) is P1's other noble, not a cardinal.
func inquiryTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := cardinalTestState(t)
	state.Cardinals = []models.NobleID{"N1"}
	return state
}

func giveDignity(state *models.GameState, id models.NobleID, dignity models.Dignity) {
	for i := range state.Nobles {
		if state.Nobles[i].ID == id {
			state.Nobles[i].Dignities = append(state.Nobles[i].Dignities, dignity)
		}
	}
}

func TestInquiryRevealsWitchAndExcommunicates(t *testing.T) {
	state := inquiryTestState(t)
	giveDignity(state, "N2", models.DignityWitch)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "LEO")}})
	if reasons := electionRejections(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %#v", reasons)
	}
	excommunication, excommunicated := resolution.State.ExcommunicationOf("N2")
	if !excommunicated || excommunication.Reason != models.ExcommunicationExOfficio {
		t.Fatalf("excommunication = %#v, want ex officio", excommunication)
	}
	revealed := eventsOfType(resolution.Events, EventTypeDignityRevealed)
	if len(revealed) != 1 || revealed[0].Dignity != models.DignityWitch {
		t.Fatalf("reveal events = %#v", revealed)
	}
	if !nobleByID(t, resolution.State, "N2").DignityRevealed {
		t.Fatalf("witch must be flagged revealed")
	}
}

func TestInquiryUnmasksEon(t *testing.T) {
	state := inquiryTestState(t)
	giveDignity(state, "N2", models.DignityChevalierDEon)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "LEO")}})
	if !nobleByID(t, resolution.State, "N2").EonUnmasked {
		t.Fatalf("eon must be unmasked")
	}
	if _, excommunicated := resolution.State.ExcommunicationOf("N2"); !excommunicated {
		t.Fatalf("eon must be excommunicated")
	}
}

func TestInquiryKeepsSpyBonuses(t *testing.T) {
	state := inquiryTestState(t)
	giveDignity(state, "N2", models.DignitySpy)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "LEO")}})
	if _, excommunicated := resolution.State.ExcommunicationOf("N2"); excommunicated {
		t.Fatalf("spy must not be excommunicated")
	}
	noble := nobleByID(t, resolution.State, "N2")
	if !noble.DignityRevealed || !noble.Has(models.DignitySpy) {
		t.Fatalf("spy = %#v, want revealed and still a spy", noble)
	}
}

func TestInquiryWithoutHiddenDignityStillCosts(t *testing.T) {
	state := inquiryTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "LEO")}})
	if len(eventsOfType(resolution.Events, EventTypeDignityRevealed)) != 0 || nobleByID(t, resolution.State, "N2").DignityRevealed {
		t.Fatalf("nothing must be revealed")
	}
	// An untitled target costs exactly 1 R: a single R is enough.
	state = inquiryTestState(t)
	ts := state.TerritoryStates["AAA"]
	ts.Resources = 1
	state.TerritoryStates["AAA"] = ts
	resolution = resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "LEO")}})
	if reasons := electionRejections(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %#v", reasons)
	}
}

func TestInquiryRejections(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.GameState)
		orders []models.WinterOrder
		want   string
	}{
		{"not a cardinal", func(s *models.GameState) { s.Cardinals = nil }, []models.WinterOrder{inquiry("O1", "LEO")}, "not_cardinal"},
		{"cardinal in a dungeon", func(s *models.GameState) { setNobleStatus(s, "N1", models.NobleStatusDungeon) }, []models.WinterOrder{inquiry("O1", "LEO")}, "not_cardinal"},
		{"one per inquirer", func(s *models.GameState) {}, []models.WinterOrder{inquiry("O0", "LEO"), inquiry("O1", "ABE")}, "inquiry_limit"},
		{"inquirer not a cardinal", func(s *models.GameState) {}, []models.WinterOrder{inquiryBy("O1", "OTO", "LEO")}, "not_cardinal"},
		{"inquirer not owned", func(s *models.GameState) {}, []models.WinterOrder{inquiryBy("O1", "LEO", "ABE")}, "noble_not_owned"},
		{"insufficient resources", func(s *models.GameState) {
			ts := s.TerritoryStates["AAA"]
			ts.Resources = 0
			s.TerritoryStates["AAA"] = ts
		}, []models.WinterOrder{inquiry("O1", "LEO")}, "insufficient_resources"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := inquiryTestState(t)
			test.mutate(state)
			resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": test.orders})
			if got := electionRejections(resolution.Events)["O1"]; got != test.want {
				t.Fatalf("reason = %q, want %q", got, test.want)
			}
		})
	}
}

func TestInquiryCostByTitle(t *testing.T) {
	state := inquiryTestState(t)
	costs := testBalance().Religion.InquiryCost
	state.Bishops = []models.Bishop{{Region: "R1", Noble: "N2"}}
	if got := InquiryCost(state, costs, "N2"); got != 3 {
		t.Fatalf("bishop cost = %d, want 3", got)
	}
	state.Cardinals = []models.NobleID{"N1", "N2"}
	if got := InquiryCost(state, costs, "N2"); got != 4 {
		t.Fatalf("cardinal cost = %d, want 4", got)
	}
	if got := InquiryCost(state, costs, "N3"); got != 1 {
		t.Fatalf("untitled cost = %d, want 1", got)
	}
	state.Marriages = []models.Marriage{{NobleA: "N3", NobleB: "N2"}}
	if got := InquiryCost(state, costs, "N3"); got != 3 {
		t.Fatalf("spouse of a cardinal cost = %d, want 3", got)
	}
}

func TestInquirySecondCardinalInquiresToo(t *testing.T) {
	state := inquiryTestState(t)
	state.Cardinals = []models.NobleID{"N1", "N4"}
	giveDignity(state, "N2", models.DignitySpy)
	giveDignity(state, "N3", models.DignityWitch)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiryBy("O1", "HUG", "LEO"), inquiryBy("O2", "OTO", "ABE")}})
	if reasons := electionRejections(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %#v", reasons)
	}
	if len(eventsOfType(resolution.Events, EventTypeDignityRevealed)) != 2 {
		t.Fatalf("want two reveals, events = %#v", resolution.Events)
	}
}

func TestInquiryEventCarriesCostWhenNothingIsHidden(t *testing.T) {
	state := inquiryTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "LEO")}})
	events := eventsOfType(resolution.Events, EventTypeInquiry)
	if len(events) != 1 || events[0].OrderID != "O1" || events[0].ResourceSpent != 1 {
		t.Fatalf("inquiry events = %#v, want one for O1 costing 1 (untitled)", events)
	}
}

// assassinatedVictim adds the dead noble VIC (N9, P2), killed on the order of
// P3 through its noble ABE (N3).
func assassinatedVictim(state *models.GameState, inquiryCost int) {
	state.RemovedNobles = append(state.RemovedNobles, models.RemovedNoble{
		ID: "N9", Code: "VIC", Name: "Victor", Sex: models.SexMale, OwnerID: "P2",
		Cause: models.DeathCauseAssassination, Turn: 0, InquiryCost: inquiryCost,
		Commissioner: "N3", Author: "P3",
	})
}

func TestInquiryOnADeadNobleCostsTheTitleItBore(t *testing.T) {
	state := inquiryTestState(t)
	assassinatedVictim(state, 4)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "VIC")}})
	if reasons := electionRejections(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %#v", reasons)
	}
	events := eventsOfType(resolution.Events, EventTypeInquiry)
	if len(events) != 1 || events[0].ResourceSpent != 4 || events[0].NobleCode != "VIC" {
		t.Fatalf("inquiry events = %#v, want one on VIC costing 4", events)
	}
	if !resolution.State.Inquired("P1", "N9") {
		t.Fatalf("inquiries = %#v, want P1 on N9", resolution.State.Inquiries)
	}
	if !resolution.State.KnowsAssassination("P1", resolution.State.RemovedNobles[0]) {
		t.Fatalf("the inquirer must know the commissioner")
	}
	if resolution.State.KnowsAssassination("P3", models.RemovedNoble{Cause: models.DeathCauseNatural}) {
		t.Fatalf("a natural death has no commissioner")
	}
}

func TestInquiryOnADeadNobleWithoutRecordedTitleCostsAnUntitledNoble(t *testing.T) {
	state := inquiryTestState(t)
	assassinatedVictim(state, 0)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "VIC")}})
	events := eventsOfType(resolution.Events, EventTypeInquiry)
	if len(events) != 1 || events[0].ResourceSpent != 1 {
		t.Fatalf("inquiry events = %#v, want one costing 1", events)
	}
}

func TestInquiryOnAnUnknownCodeIsRejected(t *testing.T) {
	state := inquiryTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "ZZZ")}})
	if got := electionRejections(resolution.Events)["O1"]; got != "unknown_noble" {
		t.Fatalf("reason = %q, want unknown_noble", got)
	}
}

func TestAssassinationKnowledgeFollowsAuthorAndInquirers(t *testing.T) {
	state := inquiryTestState(t)
	assassinatedVictim(state, 0)
	removed := state.RemovedNobles[0]
	for player, want := range map[models.PlayerID]bool{"P1": false, "P2": false, "P3": true} {
		if got := state.KnowsAssassination(player, removed); got != want {
			t.Errorf("KnowsAssassination(%s) = %v, want %v", player, got, want)
		}
	}
	// An inquiry on the commissioner, whenever it happened, reveals the killing.
	state.RecordInquiry("P1", "N3")
	state.RecordInquiry("P1", "N3")
	if !state.KnowsAssassination("P1", removed) || len(state.Inquiries) != 1 {
		t.Fatalf("inquiries = %#v, want one record revealing the killing", state.Inquiries)
	}
	validateTestState(t, state)
}

func TestInquiryRevealingAnAssassinationEmitsAPrivateEventOnce(t *testing.T) {
	state := inquiryTestState(t)
	assassinatedVictim(state, 4)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "VIC")}})
	events := eventsOfType(resolution.Events, EventTypeAssassinationRevealed)
	if len(events) != 1 || events[0].OwnerID != "P1" || events[0].NobleCode != "VIC" || events[0].CommissionerCode != "ABE" {
		t.Fatalf("revelations = %#v, want one for P1: ABE commissioned VIC", events)
	}
	report := BuildTurnReportWithBalance(state, resolution.State, resolution.Events, nil, 4, testBalance())
	revealed := 0
	for _, effect := range report.SeasonEffects {
		if effect.Kind == EventTypeAssassinationRevealed {
			revealed++
		}
	}
	if revealed != 1 || len(report.Rumors) != 0 {
		t.Fatalf("season effects = %#v, rumors = %#v, want one revelation and no rumor", report.SeasonEffects, report.Rumors)
	}
	// The author already knows the link: an inquiry teaches nothing new.
	again := inquiryTestState(t)
	assassinatedVictim(again, 4)
	again.RecordInquiry("P1", "N9")
	resolution = resolveElectionSheets(t, again, map[models.PlayerID][]models.WinterOrder{"P1": {inquiry("O1", "VIC")}})
	if len(eventsOfType(resolution.Events, EventTypeAssassinationRevealed)) != 0 {
		t.Fatalf("a link already known must not be announced again: %#v", resolution.Events)
	}
}

func TestAssassinationTellsAPriorInquirerPrivately(t *testing.T) {
	state := assassinationTestState(t)
	addNoble(state, "N3", "THR", "P3", "CCC")
	state.RecordInquiry("P3", "N1")
	resolution, err := ResolveWithDeckOrders(state, testBalance(), playAssassination("N1", "N2"))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	events := eventsOfType(resolution.Events, EventTypeAssassinationRevealed)
	if len(events) != 1 || events[0].OwnerID != "P3" || events[0].NobleCode != "TWO" || events[0].CommissionerCode != "ONE" {
		t.Fatalf("revelations = %#v, want one for P3", events)
	}
}
