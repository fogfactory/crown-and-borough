package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestBuildTurnReportIncludesNeutralSupplyStock(t *testing.T) {
	state := testState(t,
		[]models.Territory{supplyTerritory("T01", "AAA", models.TerrainPlain)},
		nil,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
	setTerritoryResources(state, "T01", 3)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	if len(report.Supply) != 1 {
		t.Fatalf("supply report = %#v, want one neutral village", report.Supply)
	}
	supply := report.Supply[0]
	if supply.Source != "T01" || supply.Owner != "" || supply.Production != 1 || supply.Demand != 0 || supply.StockAfter != 4 {
		t.Errorf("neutral supply report = %#v, want production 1 and stock 4 without owner", supply)
	}
}

func TestTurnReportContainsResolutionSectionsAndRoundTrips(t *testing.T) {
	assets := loadGameTestAssets(t)
	balance := testBalance()
	game, err := CreateGame("report-test", []PlayerInit{{Name: "One"}, {Name: "Two"}}, balance, assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	noble := game.Nobles[0]
	start := territoryByID(game.Territories, noble.LocationID)
	if len(start.Adjacencies) == 0 {
		t.Fatalf("starting territory %s has no adjacency", start.ID)
	}
	target := territoryByID(game.Territories, start.Adjacencies[0])
	report, err := ResolveTurn(game, balance, OrdersInput{
		Chains: []ChainSubmission{{
			Player: noble.OwnerID,
			Noble:  models.NobleCode(noble.Code),
			Text:   noble.Code + "\n" + start.Code + " A " + target.Code,
		}},
	})
	if err != nil {
		t.Fatalf("ResolveTurn: %v", err)
	}
	if report.Header != (ReportHeader{Year: 1, Season: models.SeasonSpring, Turn: 1}) {
		t.Errorf("report header = %#v", report.Header)
	}
	if len(report.Players) != 2 {
		t.Errorf("player reports = %d, want 2", len(report.Players))
	}
	if len(report.Supply) == 0 {
		t.Error("supply section is empty")
	}
	if len(report.Orders) == 0 {
		t.Error("orders section is empty")
	}
	order := report.Orders[0]
	if order.Owner == "" || order.Noble == "" {
		t.Errorf("order metadata = %#v, want owner and emitting noble", order)
	}
	if len(order.Targets) == 0 || order.Targets[0] != target.ID {
		t.Errorf("order targets = %#v, want %s", order.Targets, target.ID)
	}
	if order.Liaison != models.LiaisonModeSingle {
		t.Errorf("order liaison = %q, want single", order.Liaison)
	}
	if len(report.Moves) == 0 {
		t.Error("moves section is empty")
	}

	serializable := report
	serializable.State = nil
	encoded, err := json.Marshal(serializable)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var decoded TurnReport
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if !reflect.DeepEqual(decoded, serializable) {
		t.Errorf("report round trip differs\nencoded=%s\ndecoded=%#v\nwant=%#v", encoded, decoded, serializable)
	}

	current := report.State
	for current.Season != models.SeasonWinter {
		next, resolveErr := ResolveTurn(current, balance, OrdersInput{})
		if resolveErr != nil {
			t.Fatalf("advance to winter: %v", resolveErr)
		}
		current = next.State
	}
	startCode := territoryByID(current.Territories, current.Nobles[0].LocationID).Code
	winterReport, err := ResolveTurn(current, balance, OrdersInput{
		Winter: []WinterSubmission{{Player: "P1", Lines: "R T " + startCode}},
	})
	if err != nil {
		t.Fatalf("winter ResolveTurn: %v", err)
	}
	if winterReport.Winter == nil {
		t.Fatal("winter report is nil")
	}
	if len(winterReport.Winter.Stocks) == 0 {
		t.Error("winter stock section is empty")
	}
	if len(winterReport.Winter.Investments) == 0 {
		t.Error("winter investment section is empty")
	}
}

func TestBuildTurnReportKeepsCompleteOrderSyntaxFromBeforeSnapshot(t *testing.T) {
	before := models.NewGameState()
	before.Players = []models.Player{{ID: "P1", Name: "One", Color: "red"}}
	before.Territories = []models.Territory{
		{ID: "T01", Code: "ROS", Name: "Rosemont", Terrain: models.TerrainPlain},
		{ID: "T02", Code: "BRU", Name: "Brumecote", Terrain: models.TerrainPlain},
		{ID: "T03", Code: "CHA", Name: "Chanterive", Terrain: models.TerrainPlain},
	}
	owner := models.PlayerID("P1")
	armyID := models.ArmyID("A1")
	before.Armies = []models.Army{{ID: armyID, OwnerID: owner, TerritoryID: "T01", Size: 2}}
	before.TerritoryStates = map[models.TerritoryID]models.TerritoryState{
		"T01": {OwnerID: &owner, Army: &armyID, Infrastructures: []models.InfraID{}},
		"T02": {Infrastructures: []models.InfraID{}},
		"T03": {Infrastructures: []models.InfraID{}},
	}
	before.Nobles = []models.Noble{
		{ID: "N1", Code: "JEA", Name: "Jean", OwnerID: owner, LocationID: "T01", Status: models.NobleStatusFree},
		{ID: "N2", Code: "BOB", Name: "Robert", OwnerID: "P1", LocationID: "T01", Status: models.NobleStatusHostage},
	}
	before.Chains = []models.Chain{{
		ID: "C1", ArmyID: armyID, NobleID: "N1", CurrentIndex: 0,
		Orders: []models.Order{
			{
				ID: "O1", ArmyID: armyID, Type: models.OrderTypeDisperse, PositionID: "T01",
				TargetIDs: []models.TerritoryID{"T02", "T03"},
				NobleAssignments: map[models.TerritoryCode][]models.NobleCode{
					"BRU": {"JEA"},
					"CHA": {"BOB"},
				},
				Liaison: models.LiaisonModeLoop,
			},
			{ID: "O2", ArmyID: armyID, Type: models.OrderTypeHostage, PositionID: "T01", NobleTargetIDs: []models.NobleID{"N2"}, Liaison: models.LiaisonModeSingle},
		},
	}}

	report := BuildTurnReport(before, nil, []Event{
		{Type: EventTypeOrderOutcome, ArmyID: armyID, ChainID: "C1", OrderID: "O1", Outcome: OutcomeInvalid},
		{Type: EventTypeOrderOutcome, ArmyID: armyID, ChainID: "C1", OrderID: "O2", Outcome: OutcomeFailure},
	}, nil)
	if len(report.Orders) != 2 {
		t.Fatalf("orders = %#v, want two orders", report.Orders)
	}
	order := report.Orders[0]
	if order.Owner != owner || order.Noble != "JEA" || order.Source != "T01" {
		t.Errorf("order identity = %#v, want owner/noble/source from before snapshot", order)
	}
	if !reflect.DeepEqual(order.Targets, []models.TerritoryID{"T02", "T03"}) {
		t.Errorf("order targets = %#v", order.Targets)
	}
	if !reflect.DeepEqual(order.NobleAssignments, map[models.TerritoryCode][]models.NobleCode{
		"BRU": {"JEA"},
		"CHA": {"BOB"},
	}) {
		t.Errorf("noble assignments = %#v", order.NobleAssignments)
	}
	if order.Liaison != models.LiaisonModeLoop || order.Outcome != OutcomeInvalid {
		t.Errorf("order resolution metadata = %#v", order)
	}
	nobleOrder := report.Orders[1]
	if nobleOrder.Noble != "JEA" || !reflect.DeepEqual(nobleOrder.NobleTargets, []models.NobleCode{"BOB"}) {
		t.Errorf("noble order targets = %#v, want BOB from before snapshot", nobleOrder)
	}

	serializable := report
	encoded, err := json.Marshal(serializable)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var decoded TurnReport
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if !reflect.DeepEqual(decoded, serializable) {
		t.Errorf("report round trip differs\nencoded=%s\ndecoded=%#v\nwant=%#v", encoded, decoded, serializable)
	}
}

func TestBuildTurnReportMarksWinterInvestmentOutcomes(t *testing.T) {
	before := models.NewGameState()
	before.Turn = 4
	before.Season = models.SeasonWinter
	before.Players = []models.Player{{ID: "P1", Name: "One", Color: "red"}}
	before.Territories = []models.Territory{{ID: "T01", Code: "ROS", Name: "Rosemont", Terrain: models.TerrainPlain}}
	before.TerritoryStates = map[models.TerritoryID]models.TerritoryState{
		"T01": {Infrastructures: []models.InfraID{}},
	}
	rejectedOrder := &models.WinterOrder{
		ID: "O1", Type: models.WinterOrderTypeBuild, TerritoryID: "T01", InfraType: models.InfraTypeMill,
	}
	report := BuildTurnReport(before, nil, []Event{
		{Type: EventTypeBuild, OwnerID: "P1", TerritoryID: "T01", InfrastructureType: models.InfraTypeMill},
		{Type: EventTypeRejected, OwnerID: "P1", Reason: "insufficient_resources", WinterOrder: rejectedOrder},
	}, nil)
	if report.Winter == nil || len(report.Winter.Investments) != 2 {
		t.Fatalf("winter investments = %#v, want two entries", report.Winter)
	}
	if report.Winter.Investments[0].Outcome != OutcomeSuccess {
		t.Errorf("accepted winter outcome = %q, want success", report.Winter.Investments[0].Outcome)
	}
	if report.Winter.Investments[1].Outcome != OutcomeFailure {
		t.Errorf("rejected winter outcome = %q, want failure", report.Winter.Investments[1].Outcome)
	}
}

func TestWinterInvestmentReportCosts(t *testing.T) {
	newSingleStockState := func(t *testing.T) *models.GameState {
		t.Helper()
		state := winterTestState(t, []models.Territory{territory("T01", "AAA")}, nil)
		setTerritoryOwner(state, "T01", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "T01"})
		setTerritoryResources(state, "T01", 2)
		validateTestState(t, state)
		return state
	}

	findInvestment := func(t *testing.T, report TurnReport) WinterInvestmentReport {
		t.Helper()
		if report.Winter == nil || len(report.Winter.Investments) != 1 {
			t.Fatalf("winter investments = %#v, want one investment", report.Winter)
		}
		return report.Winter.Investments[0]
	}

	t.Run("records a payment from one stock", func(t *testing.T) {
		state := newSingleStockState(t)
		resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		investment := findInvestment(t, BuildTurnReport(state, resolution.State, resolution.Events, nil))
		if investment.Outcome != OutcomeSuccess || investment.Cost != 1 {
			t.Errorf("investment = %#v, want success with cost 1", investment)
		}
	})

	t.Run("aggregates payments from multiple stocks", func(t *testing.T) {
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
		setTerritoryResources(state, "T01", 3)
		setTerritoryResources(state, "T02", 3)
		validateTestState(t, state)

		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		investment := findInvestment(t, BuildTurnReport(state, resolution.State, resolution.Events, nil))
		if investment.Outcome != OutcomeSuccess || investment.Cost != 4 {
			t.Errorf("investment = %#v, want success with cost 4", investment)
		}
	})

	t.Run("does not report a cost for an insufficient order", func(t *testing.T) {
		balance := testBalance()
		balance.Costs.Troop = 5
		state := newSingleStockState(t)
		setTerritoryResources(state, "T01", 0)
		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		investment := findInvestment(t, BuildTurnReport(state, resolution.State, resolution.Events, nil))
		if investment.Outcome != OutcomeFailure || investment.Cost != 0 {
			t.Errorf("investment = %#v, want failure with cost 0", investment)
		}
	})

	t.Run("records a free order with zero cost", func(t *testing.T) {
		state := newSingleStockState(t)
		balance := testBalance()
		balance.Costs.Troop = 0
		resolution, err := ResolveWinter(state, balance, map[models.PlayerID][]models.WinterOrder{
			"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "T01"}},
		})
		if err != nil {
			t.Fatalf("ResolveWinter: %v", err)
		}
		investment := findInvestment(t, BuildTurnReport(state, resolution.State, resolution.Events, nil))
		if investment.Outcome != OutcomeSuccess || investment.Cost != 0 {
			t.Errorf("investment = %#v, want success with cost 0", investment)
		}
	})
}

func TestBuildTurnReportDoesNotDuplicateAutomaticCapital(t *testing.T) {
	before := models.NewGameState()
	before.Turn = 4
	before.Season = models.SeasonWinter
	before.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
	before.Territories = []models.Territory{{ID: "T01", Code: "ROS", Name: "Rosemont", Terrain: models.TerrainPlain}}
	before.TerritoryStates = map[models.TerritoryID]models.TerritoryState{
		"T01": {Infrastructures: []models.InfraID{}},
	}
	report := BuildTurnReport(before, nil, []Event{
		{
			Type:               EventTypeBuild,
			OwnerID:            "P1",
			OrderID:            "O1",
			TerritoryID:        "T01",
			InfrastructureType: models.InfraTypeCastle,
			ResourceSpent:      10,
		},
		{
			Type:               EventTypeCapitalElected,
			OwnerID:            "P1",
			OrderID:            "O1",
			TerritoryID:        "T01",
			InfrastructureType: models.InfraTypeCastle,
			Automatic:          true,
		},
		{
			Type:               EventTypeCapitalElected,
			OwnerID:            "P1",
			OrderID:            "O2",
			TerritoryID:        "T01",
			InfrastructureType: models.InfraTypeCastle,
		},
		{
			Type:               EventTypeCapitalElected,
			OwnerID:            "P2",
			OrderID:            "O1",
			TerritoryID:        "T01",
			InfrastructureType: models.InfraTypeCastle,
		},
	}, nil)
	if report.Winter == nil || len(report.Winter.Investments) != 3 {
		t.Fatalf("winter investments = %#v, want build and two explicit capital entries", report.Winter)
	}
	if report.Winter.Investments[0].Kind != EventTypeBuild || report.Winter.Investments[0].Cost != 10 {
		t.Errorf("build investment = %#v, want cost 10", report.Winter.Investments[0])
	}
	if report.Winter.Investments[1].Player != "P1" || report.Winter.Investments[1].Kind != EventTypeCapitalElected || report.Winter.Investments[1].Cost != 0 {
		t.Errorf("explicit capital investment = %#v, want P1 free capital entry", report.Winter.Investments[1])
	}
	if report.Winter.Investments[2].Player != "P2" || report.Winter.Investments[2].Kind != EventTypeCapitalElected || report.Winter.Investments[2].Cost != 0 {
		t.Errorf("explicit capital investment = %#v, want P2 free capital entry", report.Winter.Investments[2])
	}
}
