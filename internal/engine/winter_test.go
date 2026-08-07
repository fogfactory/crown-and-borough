package engine

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func winterTestState(t *testing.T, territories []models.Territory, armies []models.Army) *models.GameState {
	t.Helper()
	state := testState(t, territories, armies)
	state.Turn = 4
	state.Season = models.SeasonWinter
	return state
}

func setCapital(state *models.GameState, playerID models.PlayerID, infrastructureID models.InfraID) {
	for index := range state.Players {
		if state.Players[index].ID != playerID {
			continue
		}
		capitalID := infrastructureID
		state.Players[index].CapitalCastleID = &capitalID
		return
	}
}

func capitalID(t *testing.T, state *models.GameState, playerID models.PlayerID) models.InfraID {
	t.Helper()
	for _, player := range state.Players {
		if player.ID == playerID {
			if player.CapitalCastleID == nil {
				t.Fatalf("player %q has no capital", playerID)
			}
			return *player.CapitalCastleID
		}
	}
	t.Fatalf("player %q not found", playerID)
	return ""
}

func eventsOfType(events []Event, eventType EventType) []Event {
	filtered := make([]Event, 0)
	for _, event := range events {
		if event.Type == eventType {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func firstRejectedEvent(t *testing.T, events []Event) Event {
	t.Helper()
	for _, event := range events {
		if event.Type == EventTypeRejected {
			return event
		}
	}
	t.Fatalf("events = %#v, want rejected event", events)
	return Event{}
}

func infrastructureAtState(t *testing.T, state *models.GameState, territoryID models.TerritoryID) models.Infrastructure {
	t.Helper()
	territoryState := state.TerritoryStates[territoryID]
	if len(territoryState.Infrastructures) != 1 {
		t.Fatalf("territory %q infrastructures = %#v, want exactly one", territoryID, territoryState.Infrastructures)
	}
	for _, infrastructure := range state.Infrastructures {
		if infrastructure.ID == territoryState.Infrastructures[0] {
			return infrastructure
		}
	}
	t.Fatalf("infrastructure %q not found", territoryState.Infrastructures[0])
	return models.Infrastructure{}
}

func TestParseWinterOrders(t *testing.T) {
	state := winterTestState(t,
		[]models.Territory{
			territory("T01", "AAA"), territory("T02", "BBB"), territory("T03", "CCC"),
			territory("T04", "DDD"), territory("T05", "EEE"), territory("T06", "FFF"),
			territory("T07", "GGG"), territory("T08", "HHH"),
		},
		nil,
	)
	addNoble(state, "N1", "NOB", "P1", "T01")
	validateTestState(t, state)

	t.Run("parses every form case insensitively", func(t *testing.T) {
		parsed, parseErrors := orders.ParseWinterOrders(`
            r n aaa
            R T BBB # troop
            c m ccc
            C C DDD
            c r eee
            C T FFF
            c d ggg
            e c hhh
            l n nob
        `, state)
		if len(parseErrors) != 0 {
			t.Fatalf("ParseWinterOrders errors = %#v", parseErrors)
		}
		if len(parsed) != 9 {
			t.Fatalf("len(parsed) = %d, want 9", len(parsed))
		}
		wantTypes := []models.WinterOrderType{
			models.WinterOrderTypeRecruitNoble,
			models.WinterOrderTypeRecruitTroop,
			models.WinterOrderTypeBuild,
			models.WinterOrderTypeBuild,
			models.WinterOrderTypeBuild,
			models.WinterOrderTypeBuild,
			models.WinterOrderTypeBuild,
			models.WinterOrderTypeElectCapital,
			models.WinterOrderTypeLiberateNoble,
		}
		for index, wantType := range wantTypes {
			if parsed[index].Type != wantType {
				t.Errorf("parsed[%d].Type = %q, want %q", index, parsed[index].Type, wantType)
			}
			if wantID := models.OrderID("O" + string(rune('1'+index))); parsed[index].ID != wantID {
				t.Errorf("parsed[%d].ID = %q, want %q", index, parsed[index].ID, wantID)
			}
		}
		if parsed[2].InfraType != models.InfraTypeMill || parsed[6].InfraType != models.InfraTypeSupplyDepot {
			t.Errorf("build infrastructure types = %#v", parsed)
		}
		if parsed[8].NobleCode != "NOB" {
			t.Errorf("liberation code = %q, want NOB", parsed[8].NobleCode)
		}
	})

	t.Run("rejects the complete batch on malformed or unknown lines", func(t *testing.T) {
		parsed, parseErrors := orders.ParseWinterOrders("R T AAA\nC X BBB\nR N XXX\nL N BAD", state)
		if parsed != nil {
			t.Errorf("parsed = %#v, want nil for invalid batch", parsed)
		}
		if len(parseErrors) != 3 {
			t.Fatalf("len(parseErrors) = %d, want 3: %#v", len(parseErrors), parseErrors)
		}
		for index, wantLine := range []int{2, 3, 4} {
			if parseErrors[index].Line != wantLine {
				t.Errorf("parseErrors[%d].Line = %d, want %d", index, parseErrors[index].Line, wantLine)
			}
		}
	})
}

func TestResolveWinterPaymentOrder(t *testing.T) {
	t.Run("target settlement pays before another source", func(t *testing.T) {
		balance := testBalance()
		balance.Costs.Troop = 4
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "BBB", "T02"),
				territory("T02", "AAA", "T01"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
		)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		setTerritoryOwner(state, "T02", "P1")
		setCapital(state, "P1", "I2")
		setTerritoryResources(state, "T01", 3)
		setTerritoryResources(state, "T02", 3)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 0 {
			t.Errorf("target stock = %d, want 0 after paying first", got)
		}
		if got := resolution.State.TerritoryStates["T02"].Resources; got != 1 {
			t.Errorf("castle stock = %d, want 1 after paying the remainder then conserving", got)
		}
		recruits := eventsOfType(resolution.Events, EventTypeRecruit)
		if len(recruits) != 1 || recruits[0].OrderID != "O1" || recruits[0].ResourceSpent != 4 {
			t.Errorf("recruit events = %#v, want order O1 and resource spend 4", recruits)
		}
	})

	t.Run("equidistant sources use territory code", func(t *testing.T) {
		balance := testBalance()
		balance.Costs.Troop = 2
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "ZZZ", "T02", "T03"),
				territory("T02", "BBB", "T01"),
				territory("T03", "AAA", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P1")
		setTerritoryOwner(state, "T03", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T02"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T03"})
		setTerritoryResources(state, "T02", 2)
		setTerritoryResources(state, "T03", 2)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T03"].Resources; got != 0 {
			t.Errorf("AAA source stock = %d, want 0", got)
		}
		if got := resolution.State.TerritoryStates["T02"].Resources; got != 1 {
			t.Errorf("BBB source stock = %d, want 1 after conservation", got)
		}
	})

	t.Run("insufficient total stock rejects without partial payment", func(t *testing.T) {
		balance := testBalance()
		balance.Costs.Troop = 5
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
		)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		setTerritoryOwner(state, "T02", "P1")
		setTerritoryResources(state, "T01", 2)
		setTerritoryResources(state, "T02", 2)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 1 {
			t.Errorf("target stock = %d, want 1 from conservation only", got)
		}
		if got := resolution.State.TerritoryStates["T02"].Resources; got != 1 {
			t.Errorf("castle stock = %d, want 1 from conservation only", got)
		}
		if got := armyByID(t, resolution.State, "A1").Size; got != 1 {
			t.Errorf("army size = %d, want unchanged", got)
		}
		if event := firstRejectedEvent(t, resolution.Events); event.Reason != "insufficient_resources" {
			t.Errorf("rejection reason = %q, want insufficient_resources", event.Reason)
		} else if event.ResourceSpent != 0 {
			t.Errorf("rejected resource spend = %d, want 0", event.ResourceSpent)
		}
	})

	t.Run("submitted order priority determines which order receives the budget", func(t *testing.T) {
		balance := testBalance()
		balance.Costs.Troop = 3
		state := winterTestState(t,
			[]models.Territory{territory("T01", "AAA")},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
		)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(state, "T01", 4)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P1": {
				{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"},
				{ID: "O2", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"},
			},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := armyByID(t, resolution.State, "A1").Size; got != 2 {
			t.Errorf("army size = %d, want 2 after only the first recruitment", got)
		}
		if event := firstRejectedEvent(t, resolution.Events); event.OrderID != "O2" {
			t.Errorf("rejected order = %q, want O2", event.OrderID)
		}
	})
}

func TestResolveWinterRecruitNoble(t *testing.T) {
	t.Run("creates unique deterministic nobles at a controlled settlement with an army", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{territory("T01", "AAA")},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
		)
		state.Seed = "winter-nobles"
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(state, "T01", 8)
		validateTestState(t, state)
		ordersByPlayer := map[models.PlayerID][]models.WinterOrder{
			"P1": {
				{ID: "O1", Type: models.WinterOrderTypeRecruitNoble, TerritoryID: "T01"},
				{ID: "O2", Type: models.WinterOrderTypeRecruitNoble, TerritoryID: "T01"},
			},
		}

		first, err := ResolveWinter(state, testBalance(), ordersByPlayer)
		if err != nil {
			t.Fatalf("ResolveWinter(first) = %v", err)
		}
		second, err := ResolveWinter(state, testBalance(), ordersByPlayer)
		if err != nil {
			t.Fatalf("ResolveWinter(second) = %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Error("same state, balance, and orders produced different winter resolutions")
		}
		if len(first.State.Nobles) != 2 {
			t.Fatalf("len(Nobles) = %d, want 2", len(first.State.Nobles))
		}
		if first.State.Nobles[0].Code == first.State.Nobles[1].Code {
			t.Errorf("noble codes = %q and %q, want unique", first.State.Nobles[0].Code, first.State.Nobles[1].Code)
		}
		for _, noble := range first.State.Nobles {
			if noble.LocationID != "T01" || noble.OwnerID != "P1" || noble.Status != models.NobleStatusFree {
				t.Errorf("noble = %#v, want free P1 noble at T01", noble)
			}
			if !strings.HasSuffix(noble.Name, " de AAA") {
				t.Errorf("noble name = %q, want territory suffix", noble.Name)
			}
		}
		recruits := eventsOfType(first.Events, EventTypeRecruit)
		if len(recruits) != 2 || recruits[0].NobleCode == "" || recruits[0].NobleName == "" {
			t.Errorf("recruit events = %#v, want noble code and name", recruits)
		}
		for index, recruit := range recruits {
			wantOrderID := models.OrderID("O" + strconv.Itoa(index+1))
			if recruit.OrderID != wantOrderID || recruit.ResourceSpent != testBalance().Costs.Noble {
				t.Errorf("recruit[%d] = %#v, want order %s and resource spend %d", index, recruit, wantOrderID, testBalance().Costs.Noble)
			}
		}
	})

	tests := []struct {
		name   string
		state  func(t *testing.T) *models.GameState
		reason string
	}{
		{
			name: "target not controlled",
			state: func(t *testing.T) *models.GameState {
				state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
				addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
				setTerritoryOwner(state, "T01", "P2")
				return state
			},
			reason: "territory_not_controlled",
		},
		{
			name: "settlement has no army",
			state: func(t *testing.T) *models.GameState {
				state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
				addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
				setTerritoryOwner(state, "T01", "P1")
				return state
			},
			reason: "noble_requires_owned_army",
		},
		{
			name: "army occupies a controlled wild territory",
			state: func(t *testing.T) *models.GameState {
				return winterTestState(t,
					[]models.Territory{territory("T01", "AAA")},
					[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
				)
			},
			reason: "noble_requires_settlement",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.state(t)
			validateTestState(t, state)
			resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
				"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitNoble, TerritoryID: "T01"}},
			})
			if err != nil {
				t.Fatalf("ResolveWinter: %v", err)
			}
			if event := firstRejectedEvent(t, resolution.Events); event.Reason != tt.reason {
				t.Errorf("rejection reason = %q, want %q", event.Reason, tt.reason)
			}
			if len(resolution.State.Nobles) != len(state.Nobles) {
				t.Errorf("nobles = %#v, want no recruitment", resolution.State.Nobles)
			}
		})
	}
}

func TestResolveWinterLiberateNoble(t *testing.T) {
	newState := func(t *testing.T, withCapital bool) *models.GameState {
		t.Helper()
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P2")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		if withCapital {
			setCapital(state, "P1", "I1")
		}
		addNoble(state, "N1", "NOB", "P1", "T02")
		state.Nobles[0].Status = models.NobleStatusHostage
		return state
	}

	t.Run("returns a prisoner to its owner's capital", func(t *testing.T) {
		state := newState(t, true)
		validateTestState(t, state)
		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeLiberateNoble, NobleCode: "NOB"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		noble := nobleByID(t, resolution.State, "N1")
		if noble.Status != models.NobleStatusFree || noble.LocationID != "T01" {
			t.Errorf("liberated noble = %#v, want free at T01", noble)
		}
		liberations := eventsOfType(resolution.Events, EventTypeLiberation)
		if len(liberations) != 1 || liberations[0].PreviousStatus != models.NobleStatusHostage || liberations[0].OrderID != "O1" || liberations[0].ResourceSpent != 0 {
			t.Errorf("liberation events = %#v, want order O1 and resource spend 0", liberations)
		}
	})

	t.Run("uses the configured liberation cost", func(t *testing.T) {
		state := newState(t, true)
		setTerritoryResources(state, "T01", 2)
		validateTestState(t, state)
		balance := testBalance()
		balance.Costs.Liberation = 2
		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeLiberateNoble, NobleCode: "NOB"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 0 {
			t.Errorf("capital stock = %d, want 0 after liberation payment", got)
		}
	})

	tests := []struct {
		name   string
		mutate func(*models.GameState)
		reason string
	}{
		{
			name: "noble is already free",
			mutate: func(state *models.GameState) {
				state.Nobles[0].Status = models.NobleStatusFree
			},
			reason: "noble_not_prisoner",
		},
		{
			name: "noble belongs to another player",
			mutate: func(state *models.GameState) {
				state.Nobles[0].OwnerID = "P2"
			},
			reason: "noble_not_owned",
		},
		{
			name: "owner has no capital",
			mutate: func(state *models.GameState) {
				state.Players[0].CapitalCastleID = nil
			},
			reason: "no_capital",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newState(t, true)
			tt.mutate(state)
			validateTestState(t, state)
			resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
				"P1": {{ID: "O1", Type: models.WinterOrderTypeLiberateNoble, NobleCode: "NOB"}},
			})
			if err != nil {
				t.Fatalf("ResolveWinter: %v", err)
			}
			if event := firstRejectedEvent(t, resolution.Events); event.Reason != tt.reason {
				t.Errorf("rejection reason = %q, want %q", event.Reason, tt.reason)
			}
		})
	}
}

func TestResolveWinterRecruitTroop(t *testing.T) {
	t.Run("creates a new army with the next matricule", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		setTerritoryOwner(state, "T01", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(state, "T01", 2)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		army := armyByID(t, resolution.State, "A1")
		if army.OwnerID != "P1" || army.TerritoryID != "T01" || army.Size != 1 {
			t.Errorf("new army = %#v", army)
		}
		recruits := eventsOfType(resolution.Events, EventTypeRecruit)
		if len(recruits) != 1 || recruits[0].ArmyID != "A1" || recruits[0].OrderID != "O1" || recruits[0].ResourceSpent != 1 {
			t.Errorf("recruit events = %#v, want matricule A1, order O1, and resource spend 1", recruits)
		}
	})

	t.Run("adds one troop to an existing army", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{territory("T01", "AAA")},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2}},
		)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(state, "T01", 2)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := armyByID(t, resolution.State, "A1").Size; got != 3 {
			t.Errorf("army size = %d, want 3", got)
		}
		if len(resolution.State.Armies) != 1 {
			t.Errorf("len(Armies) = %d, want no second army", len(resolution.State.Armies))
		}
	})

	t.Run("rejects an uncontrolled target", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if event := firstRejectedEvent(t, resolution.Events); event.Reason != "territory_not_controlled" {
			t.Errorf("rejection = %#v", event)
		}
	})
}

func TestResolveWinterConstruction(t *testing.T) {
	t.Run("builds relay tower and depot on controlled empty tiles", func(t *testing.T) {
		for _, infrastructureType := range []models.InfraType{
			models.InfraTypePostRelay,
			models.InfraTypeWatchtower,
			models.InfraTypeSupplyDepot,
		} {
			t.Run(string(infrastructureType), func(t *testing.T) {
				balance := testBalance()
				state := winterTestState(t,
					[]models.Territory{
						territory("T01", "AAA", "T02"),
						territory("T02", "BBB", "T01"),
					},
					nil,
				)
				setTerritoryOwner(state, "T01", "P1")
				setTerritoryOwner(state, "T02", "P1")
				addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
				setTerritoryResources(state, "T01", 20)
				validateTestState(t, state)

				resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
					"P1": {{ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "T02", InfraType: infrastructureType}},
				})
				if err != nil {
					t.Fatalf("ResolveWinter: %v", err)
				}
				infrastructure := infrastructureAtState(t, resolution.State, "T02")
				if infrastructure.Type != infrastructureType || infrastructure.Level != 1 {
					t.Errorf("infrastructure = %#v, want %q level 1", infrastructure, infrastructureType)
				}
				builds := eventsOfType(resolution.Events, EventTypeBuild)
				if len(builds) != 1 || builds[0].InfrastructureType != infrastructureType || builds[0].Level != 1 || builds[0].OrderID != "O1" {
					t.Errorf("build events = %#v", builds)
				}
				wantCost, exists := infrastructureCost(balance.Costs, infrastructureType)
				if !exists || builds[0].ResourceSpent != wantCost {
					t.Errorf("build resource spend = %d, want %d", builds[0].ResourceSpent, wantCost)
				}
			})
		}
	})

	t.Run("upgrades a mill and charges the mill cost", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		setTerritoryResources(state, "T02", 3)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "T01", InfraType: models.InfraTypeMill}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := infrastructureAtState(t, resolution.State, "T01").Level; got != 2 {
			t.Errorf("mill level = %d, want 2", got)
		}
		upgrades := eventsOfType(resolution.Events, EventTypeUpgrade)
		if len(upgrades) != 1 || upgrades[0].Level != 2 || upgrades[0].OrderID != "O1" || upgrades[0].ResourceSpent != testBalance().Costs.Mill {
			t.Errorf("upgrade events = %#v", upgrades)
		}
	})

	t.Run("mill requires a productive adjacent tile when the target is empty", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		setTerritoryOwner(state, "T01", "P1")
		validateTestState(t, state)
		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "T01", InfraType: models.InfraTypeMill}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if event := firstRejectedEvent(t, resolution.Events); event.Reason != "mill_requires_productive_neighbor" {
			t.Errorf("rejection = %#v", event)
		}

		state = winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(state, "T01", 3)
		validateTestState(t, state)
		resolution, err = ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "T02", InfraType: models.InfraTypeMill}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter(adjacent productive): %v", err)
		}
		if infrastructure := infrastructureAtState(t, resolution.State, "T02"); infrastructure.Type != models.InfraTypeMill {
			t.Errorf("infrastructure = %#v, want mill", infrastructure)
		}
	})

	t.Run("orphaned mills cannot be upgraded", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA"),
				territory("T02", "BBB"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		setTerritoryResources(state, "T02", 3)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "T01", InfraType: models.InfraTypeMill}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := infrastructureAtState(t, resolution.State, "T01").Level; got != 1 {
			t.Errorf("orphaned mill level = %d, want unchanged", got)
		}
		if got := resolution.State.TerritoryStates["T02"].Resources; got != 2 {
			t.Errorf("funding stock = %d, want conservation only with no payment", got)
		}
		if event := firstRejectedEvent(t, resolution.Events); event.Reason != "mill_requires_productive_neighbor" {
			t.Errorf("rejection = %#v", event)
		}
	})

	t.Run("castle replaces a village and becomes the first capital", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		setTerritoryOwner(state, "T01", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(state, "T01", 10)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "T01", InfraType: models.InfraTypeCastle}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		infrastructure := infrastructureAtState(t, resolution.State, "T01")
		if infrastructure.Type != models.InfraTypeCastle {
			t.Errorf("replacement infrastructure = %#v, want castle", infrastructure)
		}
		if got := capitalID(t, resolution.State, "P1"); got != infrastructure.ID {
			t.Errorf("capital = %q, want new castle %q", got, infrastructure.ID)
		}
		capitalEvents := eventsOfType(resolution.Events, EventTypeCapitalElected)
		buildEvents := eventsOfType(resolution.Events, EventTypeBuild)
		if len(capitalEvents) != 1 || len(buildEvents) != 1 || capitalEvents[0].OrderID != "O1" || buildEvents[0].OrderID != "O1" || !capitalEvents[0].Automatic || buildEvents[0].ResourceSpent != testBalance().Costs.Castle {
			t.Errorf("events = %#v, want automatic capital and build events for order O1", resolution.Events)
		}
	})

	t.Run("castle replacement preserves the village stock", func(t *testing.T) {
		balance := testBalance()
		balance.Costs.Castle = 0
		state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		setTerritoryOwner(state, "T01", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(state, "T01", 5)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "T01", InfraType: models.InfraTypeCastle}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 3 {
			t.Errorf("replacement stock = %d, want ceil(5/2)=3", got)
		}
	})

	t.Run("rejects construction on an occupied infrastructure or uncontrolled tile", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "T01"})
		validateTestState(t, state)
		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {
				{ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "T01", InfraType: models.InfraTypeWatchtower},
				{ID: "O2", Type: models.WinterOrderTypeBuild, TerritoryID: "T02", InfraType: models.InfraTypeWatchtower},
			},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		rejections := eventsOfType(resolution.Events, EventTypeRejected)
		if len(rejections) != 2 || rejections[0].Reason != "structure_present" || rejections[1].Reason != "territory_not_controlled" {
			t.Errorf("rejections = %#v", rejections)
		}
	})
}

func TestResolveWinterCastleFeedsSupplyOnFollowingActionTurn(t *testing.T) {
	state := winterTestState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01", "T03"),
			territory("T03", "CCC", "T02", "T04"),
			territory("T04", "DDD", "T03", "T05"),
			territory("T05", "EEE", "T04"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T05", Size: 2}},
	)
	setTerritoryOwner(state, "T01", "P1")
	setTerritoryOwner(state, "T02", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
	setTerritoryResources(state, "T01", 10)
	validateTestState(t, state)

	winter, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "T02", InfraType: models.InfraTypeCastle}},
	})
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if infrastructure := infrastructureAtState(t, winter.State, "T02"); infrastructure.Type != models.InfraTypeCastle {
		t.Fatalf("infrastructure = %#v, want castle", infrastructure)
	}
	winter.State.Turn = 5
	winter.State.Season = models.SeasonSpring

	action, err := Resolve(winter.State, testBalance())
	if err != nil {
		t.Fatalf("Resolve(after castle): %v", err)
	}
	if hasFamineEvent(action.Events, "A1") {
		t.Errorf("events = %#v, want new castle source to feed A1", action.Events)
	}
}

func TestResolveWinterStocksAndRepatriation(t *testing.T) {
	t.Run("invests before conserving and repatriating", func(t *testing.T) {
		balance := testBalance()
		balance.Costs.Troop = 6
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T02"})
		setCapital(state, "P1", "I1")
		setTerritoryResources(state, "T02", 10)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T02"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T02"].Resources; got != 1 {
			t.Errorf("village stock = %d, want 1 after 10-6 -> 4 -> 2 -> keep 1", got)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 1 {
			t.Errorf("capital stock = %d, want village surplus", got)
		}
		stocks := eventsOfType(resolution.Events, EventTypeWinterStock)
		if len(stocks) != 2 || stocks[1].TerritoryID != "T02" || stocks[1].StockBefore != 10 || stocks[1].StockAfter != 1 {
			t.Errorf("winter stock events = %#v", stocks)
		}
	})

	t.Run("noncapital castle keeps two after conservation", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		setCapital(state, "P1", "I1")
		setTerritoryResources(state, "T02", 5)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T02"].Resources; got != 2 {
			t.Errorf("noncapital castle stock = %d, want 2", got)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 1 {
			t.Errorf("capital stock = %d, want 1", got)
		}
	})

	t.Run("uncontrolled village is conserved but not repatriated", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T02"})
		setCapital(state, "P1", "I1")
		setTerritoryResources(state, "T02", 5)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T02"].Resources; got != 3 {
			t.Errorf("uncontrolled village stock = %d, want ceil(5/2)=3", got)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 0 {
			t.Errorf("capital stock = %d, want no transfer from uncontrolled village", got)
		}
		stocks := eventsOfType(resolution.Events, EventTypeWinterStock)
		if len(stocks) != 2 || stocks[1].TerritoryID != "T02" || stocks[1].OwnerID != "" || stocks[1].StockBefore != 5 || stocks[1].StockAfter != 3 {
			t.Errorf("neutral winter stock events = %#v, want neutral owner and conservation", stocks)
		}
	})

	t.Run("a player without a capital keeps its controlled stock in place", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		setTerritoryOwner(state, "T01", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(state, "T01", 5)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 3 {
			t.Errorf("stock = %d, want 3 with no repatriation", got)
		}
	})

	t.Run("capital accumulates without a retention cap", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		setCapital(state, "P1", "I1")
		setTerritoryResources(state, "T01", 5)
		setTerritoryResources(state, "T02", 5)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 4 {
			t.Errorf("capital stock = %d, want 4 (3 retained plus 1 transfer)", got)
		}
	})

	t.Run("uses configured conservation and retention caps", func(t *testing.T) {
		balance := testBalance()
		balance.WinterStockDivisor = 3
		balance.VillageStockCap = 0
		balance.CastleStockCap = 0
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T02"})
		setCapital(state, "P1", "I1")
		setTerritoryResources(state, "T02", 5)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, balance, nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := resolution.State.TerritoryStates["T02"].Resources; got != 0 {
			t.Errorf("village stock = %d, want configured cap 0", got)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 2 {
			t.Errorf("capital stock = %d, want ceil(5/3)=2 transferred from village", got)
		}
	})
}

func TestResolveWinterCapital(t *testing.T) {
	t.Run("elect capital replaces the current designation", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		setCapital(state, "P1", "I1")
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeElectCapital, TerritoryID: "T02"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if got := capitalID(t, resolution.State, "P1"); got != "I2" {
			t.Errorf("capital = %q, want I2", got)
		}
		events := eventsOfType(resolution.Events, EventTypeCapitalElected)
		if len(events) != 1 || events[0].InfrastructureID != "I2" || events[0].OrderID != "O1" || events[0].ResourceSpent != 0 || events[0].Automatic {
			t.Errorf("capital events = %#v", events)
		}
	})

	t.Run("election requires a controlled castle", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		setTerritoryOwner(state, "T01", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		validateTestState(t, state)
		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeElectCapital, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if event := firstRejectedEvent(t, resolution.Events); event.Reason != "capital_requires_controlled_castle" {
			t.Errorf("rejection = %#v", event)
		}
	})

	t.Run("destroyed capital is cleared and blocks liberation", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{territory("T01", "AAA")},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
		)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		setCapital(state, "P1", "I1")
		setTerritoryResources(state, "T01", 5)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P1", "T01")
		state.Nobles[1].Status = models.NobleStatusHostage
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypePillage, PositionID: "T01"})
		state.Turn = 1
		state.Season = models.SeasonSpring
		validateTestState(t, state)

		action, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve(pillage) = %v", err)
		}
		for _, player := range action.State.Players {
			if player.ID == "P1" && player.CapitalCastleID != nil {
				t.Errorf("capital = %q, want cleared after destruction", *player.CapitalCastleID)
			}
		}
		if got := action.State.TerritoryStates["T01"].Resources; got != 0 {
			t.Errorf("destroyed castle stock = %d, want cleared with the settlement", got)
		}
		action.State.Turn = 4
		action.State.Season = models.SeasonWinter
		winter, err := ResolveWinter(action.State, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeLiberateNoble, NobleCode: "TWO"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter(after destruction) = %v", err)
		}
		if event := firstRejectedEvent(t, winter.Events); event.Reason != "no_capital" {
			t.Errorf("rejection = %#v", event)
		}
	})

	t.Run("losing and retaking a castle does not restore its former capital designation", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{territory("T01", "AAA")},
			[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "T01", Size: 1}},
		)
		setTerritoryOwner(state, "T01", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		setCapital(state, "P1", "I1")
		validateTestState(t, state)

		lost, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve(control loss): %v", err)
		}
		for _, player := range lost.State.Players {
			if player.ID == "P1" && player.CapitalCastleID != nil {
				t.Errorf("capital = %q, want cleared after control loss", *player.CapitalCastleID)
			}
		}
		lost.State.Armies[0].OwnerID = "P1"
		lost.State.Turn = 2
		lost.State.Season = models.SeasonSummer
		retaken, err := Resolve(lost.State, testBalance())
		if err != nil {
			t.Fatalf("Resolve(recapture): %v", err)
		}
		if owner := retaken.State.TerritoryStates["T01"].OwnerID; owner == nil || *owner != "P1" {
			t.Errorf("territory owner = %v, want P1", owner)
		}
		for _, player := range retaken.State.Players {
			if player.ID == "P1" && player.CapitalCastleID != nil {
				t.Errorf("capital = %q, must remain nil until redesignation", *player.CapitalCastleID)
			}
		}
	})
}

func TestResolveWinterTruceMultiPlayerAndDeterminism(t *testing.T) {
	t.Run("rejects the resolver that does not match the current season", func(t *testing.T) {
		winter := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		setTerritoryOwner(winter, "T01", "P1")
		addInfrastructure(winter, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(winter, "T01", 5)
		validateTestState(t, winter)
		if _, err := Resolve(winter, testBalance()); err == nil || !strings.Contains(err.Error(), "must use ResolveWinter") {
			t.Errorf("Resolve(winter) error = %v", err)
		}
		if winter.Season != models.SeasonWinter || winter.Turn != 4 || winter.TerritoryStates["T01"].Resources != 5 {
			t.Errorf("Resolve changed a rejected winter input: %#v", winter)
		}

		action := testState(t, []models.Territory{territory("T01", "AAA")}, nil)
		setTerritoryOwner(action, "T01", "P1")
		addInfrastructure(action, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(action, "T01", 5)
		validateTestState(t, action)
		if _, err := ResolveWinter(action, testBalance(), nil); err == nil || !strings.Contains(err.Error(), "must use Resolve") {
			t.Errorf("ResolveWinter(action) error = %v", err)
		}
		if action.Season != models.SeasonSpring || action.Turn != 1 || action.TerritoryStates["T01"].Resources != 5 {
			t.Errorf("ResolveWinter changed a rejected action input: %#v", action)
		}
	})

	t.Run("rejects an unusable winter stock divisor", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		validateTestState(t, state)
		balance := testBalance()
		balance.WinterStockDivisor = 0
		if _, err := ResolveWinter(state, balance, nil); err == nil || !strings.Contains(err.Error(), "winter stock divisor") {
			t.Errorf("ResolveWinter error = %v, want divisor validation", err)
		}
	})

	t.Run("does not resolve chains, combat, or supply", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{territory("T01", "AAA")},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 10}},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "T01"})
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, testBalance(), nil)
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		if resolution.State.Turn != state.Turn || resolution.State.Season != state.Season {
			t.Errorf("calendar = %d/%q, want unchanged %d/%q", resolution.State.Turn, resolution.State.Season, state.Turn, state.Season)
		}
		if !reflect.DeepEqual(resolution.State.Chains, state.Chains) || !reflect.DeepEqual(resolution.State.Armies, state.Armies) {
			t.Errorf("winter changed chains or armies: chains=%#v armies=%#v", resolution.State.Chains, resolution.State.Armies)
		}
		for _, forbidden := range []EventType{EventTypeSupply, EventTypeFamine, EventTypeCombat, EventTypeMovement, EventTypeOrderOutcome} {
			if containsEvent(resolution.Events, forbidden) {
				t.Errorf("events = %#v, must not contain %q during winter", resolution.Events, forbidden)
			}
		}
	})

	t.Run("processes players by ID and conserves globally once", func(t *testing.T) {
		balance := testBalance()
		balance.Costs.Troop = 0
		state := winterTestState(t,
			[]models.Territory{
				territory("T01", "AAA"),
				territory("T02", "BBB"),
			},
			nil,
		)
		setTerritoryOwner(state, "T01", "P1")
		setTerritoryOwner(state, "T02", "P2")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T02"})
		setTerritoryResources(state, "T01", 5)
		setTerritoryResources(state, "T02", 5)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P2": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T02"}},
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		recruits := eventsOfType(resolution.Events, EventTypeRecruit)
		if len(recruits) != 2 || recruits[0].OwnerID != "P1" || recruits[1].OwnerID != "P2" {
			t.Errorf("recruit event order = %#v, want P1 then P2", recruits)
		}
		if got := resolution.State.TerritoryStates["T01"].Resources; got != 3 {
			t.Errorf("P1 stock = %d, want 3 after one conservation pass", got)
		}
		if got := resolution.State.TerritoryStates["T02"].Resources; got != 3 {
			t.Errorf("P2 stock = %d, want 3 after one conservation pass", got)
		}
	})

	t.Run("is pure and deterministic", func(t *testing.T) {
		state := winterTestState(t,
			[]models.Territory{territory("T01", "AAA")},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
		)
		state.Seed = "deterministic-winter"
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
		setCapital(state, "P1", "I1")
		setTerritoryResources(state, "T01", 10)
		validateTestState(t, state)
		before := cloneGameState(state)
		ordersByPlayer := map[models.PlayerID][]models.WinterOrder{
			"P1": {
				{ID: "O1", Type: models.WinterOrderTypeRecruitNoble, TerritoryID: "T01"},
				{ID: "O2", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"},
			},
		}

		first, err := ResolveWinter(state, testBalance(), ordersByPlayer)
		if err != nil {
			t.Fatalf("ResolveWinter(first): %v", err)
		}
		second, err := ResolveWinter(state, testBalance(), ordersByPlayer)
		if err != nil {
			t.Fatalf("ResolveWinter(second): %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Error("same input produced different output")
		}
		if !reflect.DeepEqual(state, before) {
			t.Errorf("ResolveWinter mutated its input: got %#v, want %#v", state, before)
		}
	})

	t.Run("reports an unknown submitted player deterministically", func(t *testing.T) {
		state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		validateTestState(t, state)
		_, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"Z9": nil,
			"A0": nil,
		})
		if err == nil || !strings.Contains(err.Error(), `unknown player "A0"`) {
			t.Errorf("ResolveWinter error = %v, want deterministic A0 error", err)
		}
	})
}

func TestResolveUsesConfiguredBalance(t *testing.T) {
	state := testState(t, []models.Territory{territory("T01", "AAA")}, nil)
	setTerritoryOwner(state, "T01", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
	validateTestState(t, state)
	balance := testBalance()
	balance.BaseProduction = 7

	resolution, err := Resolve(state, balance)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := resolution.State.TerritoryStates["T01"].Resources; got != 7 {
		t.Errorf("stock = %d, want configured base production 7", got)
	}
}
