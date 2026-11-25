package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func attackOrder(army models.ArmyID, source, target models.TerritoryID, outcome Outcome) OrderReport {
	return OrderReport{
		Army: army, Chain: models.ChainID("c-" + string(army)), Owner: "P1", Type: models.OrderTypeAttack,
		Source: source, Target: target, Targets: []models.TerritoryID{target}, Outcome: outcome,
	}
}

func TestBuildMovementsClassifiesVictoriesMovesAndFailures(t *testing.T) {
	combats := []CombatReport{
		{Territory: "BRU", Contested: true, Winner: "A1", Reason: "attack_wins", Contenders: []CombatContender{
			{ArmyID: "A1", OwnerID: "P1", Force: 3}, {ArmyID: "A2", OwnerID: "P2", Force: 2, Defender: true},
		}},
		{Territory: "CHA", Contested: true, Reason: "defense_holds", Contenders: []CombatContender{
			{ArmyID: "A3", OwnerID: "P1", Force: 1}, {Force: 2, Defender: true},
		}},
		{Territory: "VAL", Winner: "A5", Reason: "attack_wins", Contenders: []CombatContender{{ArmyID: "A5", OwnerID: "P1", Force: 1}}},
	}
	orders := []OrderReport{
		attackOrder("A1", "ROS", "BRU", OutcomeSuccess),
		attackOrder("A3", "VAN", "CHA", OutcomeFailure),
		attackOrder("A5", "PYL", "VAL", OutcomeSuccess),
		{Army: "A6", Owner: "P1", Type: models.OrderTypeManeuver, Source: "ROS", Targets: []models.TerritoryID{"BRU", "CHA"}, Outcome: OutcomeFailure, Reason: "dislodged"},
		{Army: "A7", Owner: "P1", Type: models.OrderTypeHold, Source: "ROS", Outcome: OutcomeSuccess},
	}
	got := buildMovements(combats, orders)
	want := []ArmyMovement{
		{Kind: MovementVictory, Army: "A1", Owner: "P1", Legs: []MovementLeg{{From: "ROS", To: "BRU"}}, Combat: "BRU", Chain: "c-A1"},
		{Kind: MovementFailure, Army: "A3", Owner: "P1", Legs: []MovementLeg{{From: "VAN", To: "CHA"}}, Combat: "CHA", Chain: "c-A3", Reason: "defense_holds"},
		{Kind: MovementMove, Army: "A5", Owner: "P1", Legs: []MovementLeg{{From: "PYL", To: "VAL"}}, Chain: "c-A5"},
		{Kind: MovementFailure, Army: "A6", Owner: "P1", Legs: []MovementLeg{{From: "ROS", To: "BRU"}, {From: "ROS", To: "CHA"}}, Reason: "dislodged"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("movements = %+v\nwant %+v", got, want)
	}
}

func TestBuildTurnReportReadsAttackOnEmptyTerritoryAsMove(t *testing.T) {
	state := suppliedNobleCombatState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"},
	})
	validateTestState(t, state)
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	if len(report.Movements) != 1 {
		t.Fatalf("movements = %+v, want the single move", report.Movements)
	}
	move := report.Movements[0]
	if move.Kind != MovementMove || move.Combat != "" || move.Legs[0] != (MovementLeg{From: "AAA", To: "BBB"}) {
		t.Fatalf("movement = %+v, want an unopposed move AAA to BBB", move)
	}
}
