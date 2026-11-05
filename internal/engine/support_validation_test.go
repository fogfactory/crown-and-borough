package engine

import (
	"errors"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// supportPlanState is P1's armies on AAA (noble ONE) and BBB (noble TWO),
// both next to CCC, and a P2 army on DDD that keeps the game going; the army on AAA belongs to supportedOwner.
func supportPlanState(t *testing.T, supportedOwner models.PlayerID) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB", "CCC"), territory("BBB", "BBB", "AAA", "CCC"), territory("CCC", "CCC", "AAA", "BBB", "DDD"), territory("DDD", "DDD", "CCC")},
		[]models.Army{{ID: "A1", OwnerID: supportedOwner, TerritoryID: "AAA", Size: 2}, {ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 2}, {ID: "A3", OwnerID: "P2", TerritoryID: "DDD", Size: 1}},
	)
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	validateTestState(t, state)
	return state
}

func supportInput(chains ...ChainSubmission) OrdersInput { return OrdersInput{Chains: chains} }

func TestOffensiveSupportOfOwnArmyNeedsItsAttack(t *testing.T) {
	support := ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\nBBB S AAA - CCC"}

	_, err := ResolveTurn(supportPlanState(t, "P1"), testBalance(), supportInput(support))
	var inputErrors *InputErrors
	if !errors.As(err, &inputErrors) || len(inputErrors.Errors) != 1 || inputErrors.Errors[0].Code != "support_no_attack" {
		t.Fatalf("error = %v, want one support_no_attack input error", err)
	}

	attack := ChainSubmission{Player: "P1", Noble: "ONE", Text: "ONE\nAAA A CCC"}
	if _, err := ResolveTurn(supportPlanState(t, "P1"), testBalance(), supportInput(support, attack)); err != nil {
		t.Fatalf("support of a planned attack: %v", err)
	}
}

func TestOffensiveSupportOfAnotherPlayersArmyIsFree(t *testing.T) {
	support := ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\nBBB S AAA - CCC"}
	if _, err := ResolveTurn(supportPlanState(t, "P2"), testBalance(), supportInput(support)); err != nil {
		t.Fatalf("support of another player's army: %v", err)
	}
}
