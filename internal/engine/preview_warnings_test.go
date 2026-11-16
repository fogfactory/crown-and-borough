package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func previewWarningsFor(t *testing.T, state *models.GameState, chains ...ChainSubmission) []PreviewWarning {
	t.Helper()
	preview, err := PreviewOrders(state, testBalance(), "P1", OrdersInput{Chains: chains})
	if err != nil {
		t.Fatalf("PreviewOrders: %v", err)
	}
	return preview.Warnings
}

func requireWarning(t *testing.T, warnings []PreviewWarning, code, level string) PreviewWarning {
	t.Helper()
	for _, warning := range warnings {
		if warning.Code == code {
			if warning.Level != level {
				t.Fatalf("warning %s level = %q, want %q", code, warning.Level, level)
			}
			if warning.MessageKey == "" || warning.Message == "" {
				t.Fatalf("warning %s = %#v, want a localizable message", code, warning)
			}
			return warning
		}
	}
	t.Fatalf("warnings = %#v, want one %s", warnings, code)
	return PreviewWarning{}
}

func storeChain(t *testing.T, state *models.GameState, text string) {
	t.Helper()
	chain, errs := orders.ParseChain(text, state)
	if len(errs) != 0 {
		t.Fatalf("parse %q: %v", text, errs)
	}
	if err := orders.AssignChain(state, chain); err != nil {
		t.Fatalf("assign %q: %v", text, err)
	}
	// A chain under way was received on a previous turn.
	for index := range state.Nobles {
		state.Nobles[index].LastEmissionTurn = 0
	}
}

func TestPreviewWarnsAStarvingArmyCannotAttack(t *testing.T) {
	state := supportPlanState(t, "P1")
	state.Armies[1].Starving = true

	warnings := previewWarningsFor(t, state, ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\nBBB A CCC"})
	warning := requireWarning(t, warnings, "starving_attack", WarningLevelWarning)
	if warning.Line != 2 || warning.Noble != "TWO" || warning.Source != InputSourceChain {
		t.Errorf("warning = %#v, want line 2 of noble TWO on the chain source", warning)
	}
}

func TestPreviewWarnsAChainReplacesOneUnderWay(t *testing.T) {
	state := supportPlanState(t, "P1")
	storeChain(t, state, "TWO\nBBB A CCC\nCCC A DDD")

	warnings := previewWarningsFor(t, state, ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\nH BBB"})
	requireWarning(t, warnings, "chain_replaced", WarningLevelWarning)
	if warnings := previewWarningsFor(t, state); len(warnings) != 0 {
		t.Errorf("warnings without a submission = %#v, want none: the chain under way can run", warnings)
	}
}

func TestPreviewWarnsAChainUnderWayThatCannotRun(t *testing.T) {
	state := supportPlanState(t, "P1")
	storeChain(t, state, "TWO\nP BBB")
	// The village the army stands on is gone.
	territoryState := state.TerritoryStates["BBB"]
	territoryState.Infrastructures = nil
	state.TerritoryStates["BBB"] = territoryState
	kept := state.Infrastructures[:0]
	for _, infrastructure := range state.Infrastructures {
		if infrastructure.TerritoryID != "BBB" {
			kept = append(kept, infrastructure)
		}
	}
	state.Infrastructures = kept

	warnings := previewWarningsFor(t, state)
	warning := requireWarning(t, warnings, "running_chain_pillage_no_infrastructure", WarningLevelWarning)
	if warning.Line != 0 || warning.Noble != "TWO" {
		t.Errorf("warning = %#v, want no line and the emitting noble", warning)
	}
	// A new order on the army settles it.
	if warnings := previewWarningsFor(t, state, ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\nH BBB"}); len(warnings) != 1 || warnings[0].Code != "chain_replaced" {
		t.Errorf("warnings after a new order = %#v, want only the replacement notice", warnings)
	}
}

func TestPreviewWarnsLoopedDispersionWithTooManyDestinations(t *testing.T) {
	state := supportPlanState(t, "P1")

	warnings := previewWarningsFor(t, state, ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\n(BBB D AAA CCC AAA)"})
	requireWarning(t, warnings, "disperse_too_many", WarningLevelWarning)
}

func TestPreviewSupportOfAnotherPlayersArmyIsANote(t *testing.T) {
	state := supportPlanState(t, "P2")

	warnings := previewWarningsFor(t, state, ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\nBBB S AAA"})
	requireWarning(t, warnings, "support_opposing", WarningLevelNote)
}

func TestPreviewWarnsASupportOfOwnArmyThatAttacks(t *testing.T) {
	state := supportPlanState(t, "P1")

	warnings := previewWarningsFor(t, state,
		ChainSubmission{Player: "P1", Noble: "ONE", Text: "ONE\nAAA A CCC"},
		ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\nBBB S AAA"},
	)
	requireWarning(t, warnings, "support_attacker", WarningLevelWarning)
}

func TestPreviewWarningsNeverRevealAnotherPlayersChain(t *testing.T) {
	// P2's chain under way attacks CCC, which cancels P1's join from it.
	setup := func() *models.GameState {
		state := testState(t,
			[]models.Territory{territory("AAA", "AAA", "CCC"), territory("CCC", "CCC", "AAA", "DDD"), territory("DDD", "DDD", "CCC")},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "CCC", Size: 2}, {ID: "A2", OwnerID: "P2", TerritoryID: "DDD", Size: 1}},
		)
		keepTestArmiesSupplied(state)
		addNoble(state, "N2", "TWO", "P1", "CCC")
		addNoble(state, "N3", "TRE", "P2", "DDD")
		validateTestState(t, state)
		return state
	}
	state := setup()
	storeChain(t, state, "TRE\nDDD A CCC")
	join := ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\nCCC J AAA"}

	withEnemy := previewWarningsFor(t, state, join)
	without := previewWarningsFor(t, setup(), join)
	if len(withEnemy) != 0 || len(without) != 0 {
		t.Fatalf("warnings with an enemy chain = %#v, without = %#v: want none, whatever the enemy plans", withEnemy, without)
	}
}

func TestPreviewKnowsAnOpposingChainThePlayerHasLearned(t *testing.T) {
	setup := func(learned bool) *models.GameState {
		state := supportPlanState(t, "P2")
		addNoble(state, "N3", "TRE", "P2", "BBB")
		storeChain(t, state, "TRE\nAAA A CCC")
		if learned {
			// P1 read the chain while it held the noble hostage; the noble
			// has been freed since.
			chainID := *state.Armies[0].ChainID
			state.Privacy = &models.PrivacyMeta{ChainKnowledge: map[models.PlayerID]map[models.ChainID]models.ChainSnapshot{
				"P1": {chainID: {ID: chainID, NobleID: "N3", ArmyID: "A1", Orders: state.Chains[0].Orders}},
			}}
		}
		return state
	}
	support := ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\nBBB S AAA"}

	warnings := previewWarningsFor(t, setup(true), support)
	requireWarning(t, warnings, "support_attacker", WarningLevelWarning)

	// A chain P1 never learned stays secret: the support only earns a note.
	warnings = previewWarningsFor(t, setup(false), support)
	requireWarning(t, warnings, "support_opposing", WarningLevelNote)
}

func TestPreviewWarnsAnAttackOnOwnArmyThatStaysPut(t *testing.T) {
	attack := ChainSubmission{Player: "P1", Noble: "TWO", Text: "TWO\nBBB A AAA"}

	// The army on AAA has no order: it stays, and the attack would dislodge it.
	warnings := previewWarningsFor(t, supportPlanState(t, "P1"), attack)
	warning := requireWarning(t, warnings, "attack_own_army", WarningLevelWarning)
	if warning.Line != 2 || warning.Noble != "TWO" {
		t.Errorf("warning = %#v, want line 2 of noble TWO", warning)
	}

	// The army on AAA leaves: the attack has nothing to dislodge.
	leaving := ChainSubmission{Player: "P1", Noble: "ONE", Text: "ONE\nAAA A CCC"}
	for _, warning := range previewWarningsFor(t, supportPlanState(t, "P1"), attack, leaving) {
		if warning.Code == "attack_own_army" {
			t.Errorf("warnings = %#v, want no attack_own_army when the target army leaves", warning)
		}
	}

	// An army that holds, or whose own chain under way attacks nothing, stays too.
	holding := ChainSubmission{Player: "P1", Noble: "ONE", Text: "ONE\nH AAA"}
	requireWarning(t, previewWarningsFor(t, supportPlanState(t, "P1"), attack, holding), "attack_own_army", WarningLevelWarning)
	state := supportPlanState(t, "P1")
	storeChain(t, state, "TWO\nBBB A AAA")
	requireWarning(t, previewWarningsFor(t, state), "attack_own_army", WarningLevelWarning)

	// An army of another player is not concerned.
	for _, warning := range previewWarningsFor(t, supportPlanState(t, "P2"), attack) {
		if warning.Code == "attack_own_army" {
			t.Errorf("warnings = %#v, want none for an opposing army", warning)
		}
	}
}
