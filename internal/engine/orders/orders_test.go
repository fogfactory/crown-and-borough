package orders

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestParseChainAllOrderForms(t *testing.T) {
	game := orderTestState()
	chain, parseErrors := ParseChain(`
# The leading blank line and comment preserve source line numbers.
  jea # Hugues
ros a boi # attack
(fou s boi - cha)
ros s boi
h ros
p ros
ros d ros*jea boi*ann*jea fou*bob boi
ros j boi
`, game)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseChain() errors = %#v, want none", parseErrors)
	}
	if chain.NobleID != "N1" || chain.ID != "" || chain.ArmyID != "" || chain.CurrentIndex != 0 {
		t.Fatalf("parsed chain = %#v, want unassigned N1 chain at index 0", chain)
	}
	if len(chain.Orders) != 7 {
		t.Fatalf("parsed order count = %d, want 7", len(chain.Orders))
	}
	wantTypes := []models.OrderType{
		models.OrderTypeAttack,
		models.OrderTypeSupport,
		models.OrderTypeSupport,
		models.OrderTypeHold,
		models.OrderTypePillage,
		models.OrderTypeDisperse,
		models.OrderTypeJoin,
	}
	for index, wantType := range wantTypes {
		order := chain.Orders[index]
		if order.ID != models.OrderID("O"+string(rune('1'+index))) {
			t.Errorf("order %d id = %q, want O%d", index, order.ID, index+1)
		}
		if order.Type != wantType {
			t.Errorf("order %d type = %q, want %q", index, order.Type, wantType)
		}
		if order.ArmyID != "" {
			t.Errorf("order %d army = %q, want unassigned", index, order.ArmyID)
		}
	}
	if chain.Orders[1].Liaison != models.LiaisonModeLoop || chain.Orders[0].Liaison != models.LiaisonModeSingle {
		t.Errorf("liaisons = %q/%q, want loop/single", chain.Orders[1].Liaison, chain.Orders[0].Liaison)
	}
	if got, want := chain.Orders[1].TargetIDs, []models.TerritoryID{"BOI", "CHA"}; !reflect.DeepEqual(got, want) {
		t.Errorf("offensive support targets = %#v, want %#v", got, want)
	}
	if got, want := chain.Orders[5].TargetIDs, []models.TerritoryID{"ROS", "BOI", "FOU", "BOI"}; !reflect.DeepEqual(got, want) {
		t.Errorf("D targets = %#v, want repeated destinations %#v", got, want)
	}
	if got, want := chain.Orders[5].NobleAssignments, map[models.TerritoryID][]models.NobleCode{
		"ROS": {"JEA"},
		"BOI": {"ANN", "JEA"},
		"FOU": {"BOB"},
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("D assignments = %#v, want %#v", got, want)
	}
}

func TestParseChainWildcardAssignmentsAndPartialResults(t *testing.T) {
	game := orderTestState()
	chain, parseErrors := ParseChain(`
JEA
ROS D ROS* BOI*ANN BOI*JEA
ROS X BOI
H ROS
`, game)
	if len(parseErrors) != 1 || parseErrors[0].Code != ParseCodeUnknownSymbol || parseErrors[0].Line != 4 {
		t.Fatalf("ParseChain() errors = %#v, want unknown_symbol on line 4", parseErrors)
	}
	if len(chain.Orders) != 2 || chain.Orders[0].ID != "O1" || chain.Orders[1].ID != "O2" {
		t.Fatalf("valid orders = %#v, want O1 D and O2 H", chain.Orders)
	}
	if got, want := chain.Orders[0].NobleAssignments, map[models.TerritoryID][]models.NobleCode{
		"ROS": {"*"},
		"BOI": {"ANN", "JEA"},
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("wildcard assignments = %#v, want %#v", got, want)
	}
}

func TestParseChainErrorsCollectOriginalLines(t *testing.T) {
	game := orderTestState()
	_, parseErrors := ParseChain(`
ZZZ
ROS X BOI
BAD A BOI
ROS A
ROS A BOI FOU
ROS D
(ROS A BOI
`, game)
	want := []struct {
		line int
		code string
	}{
		{2, ParseCodeNobleNotFound},
		{3, ParseCodeUnknownSymbol},
		{4, ParseCodeInvalidCode},
		{5, ParseCodeMissingTarget},
		{6, ParseCodeTooManyTargets},
		{7, ParseCodeMissingTarget},
		{8, ParseCodeUnclosedParenthesis},
	}
	if len(parseErrors) != len(want) {
		t.Fatalf("ParseChain() errors = %#v, want %d errors", parseErrors, len(want))
	}
	for index, expected := range want {
		got := parseErrors[index]
		if got.Line != expected.line || got.Code != expected.code || got.Message == "" {
			t.Errorf("error %d = %#v, want line %d code %q with message", index, got, expected.line, expected.code)
		}
	}

	_, noHeader := ParseChain("\n # comment only\n", game)
	if len(noHeader) != 1 || noHeader[0].Code != ParseCodeNoHeader || noHeader[0].Line != 0 {
		t.Errorf("no-header errors = %#v, want no_header at line 0", noHeader)
	}
	_, badHeader := ParseChain("JEA EXTRA\nROS A BOI", game)
	if len(badHeader) != 1 || badHeader[0].Code != ParseCodeBadHeader || badHeader[0].Line != 1 {
		t.Errorf("bad-header errors = %#v, want bad_header at line 1", badHeader)
	}
	_, badAssignment := ParseChain("JEA\nROS D * BOI", game)
	if len(badAssignment) != 1 || badAssignment[0].Code != ParseCodeInvalidCode || badAssignment[0].Line != 2 {
		t.Errorf("bad D assignment errors = %#v, want invalid_code at line 2", badAssignment)
	}

	_, legacyStatusOrders := ParseChain("JEA\nROS K BOB\nROS D ROS*ZZZ BOI", game)
	if len(legacyStatusOrders) != 2 {
		t.Fatalf("legacy status errors = %#v, want two errors", legacyStatusOrders)
	}
	if legacyStatusOrders[0].Code != ParseCodeUnknownSymbol || legacyStatusOrders[0].Line != 2 {
		t.Errorf("legacy status error = %#v, want unknown_symbol on line 2", legacyStatusOrders[0])
	}
	if legacyStatusOrders[1].Code != ParseCodeInvalidCode || legacyStatusOrders[1].Line != 3 {
		t.Errorf("unknown assignment error = %#v, want invalid_code on line 3", legacyStatusOrders[1])
	}
}

func TestValidateChainStaticRulesAndPurity(t *testing.T) {
	game := orderTestState()
	for _, text := range []string{
		"JEA\nROS A BOI",
		"JEA\nROS S BOI",
		"JEA\nFOU S BOI - CHA",
		"JEA\nH ROS",
		"JEA\nP ROS",
		"JEA\nROS J BOI",
		"JEA\nROS D ROS BOI",
	} {
		chain := mustParseChain(t, game, text)
		before := jsonCloneChain(chain)
		if validationErrors := ValidateChain(game, chain); len(validationErrors) != 0 {
			t.Errorf("ValidateChain(%q) = %#v, want none", text, validationErrors)
		}
		if !reflect.DeepEqual(chain, before) {
			t.Errorf("ValidateChain(%q) mutated chain:\n before=%#v\n after=%#v", text, before, chain)
		}
	}

	cases := []struct {
		name   string
		text   string
		mutate func(*models.Chain)
		code   string
	}{
		{"attack not adjacent", "JEA\nROS A BRU", nil, "not_adjacent"},
		{"defensive support self", "JEA\nROS S ROS", nil, "support_same_position"},
		{"offensive support source not adjacent", "JEA\nROS S BOI - CHA", nil, "not_adjacent"},
		{"offensive support target not adjacent", "JEA\nFOU S ROS - BRU", nil, "not_adjacent"},
		{"join not last", "JEA\nROS J BOI\nH BOI", nil, "join_not_last"},
		{"pillage extra target", "JEA\nP ROS", func(chain *models.Chain) { chain.Orders[0].TargetIDs = []models.TerritoryID{"BOI"} }, "unexpected_target"},
		{"D destination not adjacent", "JEA\nROS D BRU", nil, "not_adjacent"},
		{"D assignment destination absent", "JEA\nROS D ROS", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryID][]models.NobleCode{"BOI": {"JEA"}}
		}, "assignment_destination_not_declared"},
		{"D duplicate noble", "JEA\nROS D ROS BOI", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryID][]models.NobleCode{"ROS": {"JEA"}, "BOI": {"JEA"}}
		}, "duplicate_assignment_noble"},
		{"D multiple wildcards", "JEA\nROS D ROS BOI", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryID][]models.NobleCode{"ROS": {"*"}, "BOI": {"*"}}
		}, "multiple_wildcards"},
		{"D unknown noble assignment", "JEA\nROS D ROS", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryID][]models.NobleCode{"ROS": {"ZZZ"}}
		}, "unknown_assignment_noble"},
		{"duplicate order id", "JEA\nROS A BOI\nH BOI", func(chain *models.Chain) {
			chain.Orders[1].ID = "O1"
		}, "duplicate_order_id"},
		{"assignments on non-D", "JEA\nROS A BOI", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryID][]models.NobleCode{"ROS": {"JEA"}}
		}, "unexpected_noble_assignments"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			chain := mustParseChain(t, game, test.text)
			if test.mutate != nil {
				test.mutate(&chain)
			}
			if validationErrors := ValidateChain(game, chain); !hasValidationCode(validationErrors, test.code) {
				t.Fatalf("ValidateChain() = %#v, want code %q", validationErrors, test.code)
			}
		})
	}
}

func TestVisibleOrderMessagesUseTerritoryIDs(t *testing.T) {
	game := orderTestState()
	chain := mustParseChain(t, game, "JEA\nROS A BRU")
	validationErrors := ValidateChain(game, chain)
	if len(validationErrors) == 0 {
		t.Fatal("ValidateChain() returned no errors, want non-adjacent diagnostic")
	}
	message := validationErrors[0].Error()
	if !strings.Contains(message, `"BRU"`) || !strings.Contains(message, `"ROS"`) {
		t.Errorf("validation message = %q, want territory codes", message)
	}

	noArmyChain := mustParseChain(t, game, "JEA\nBRU A BOI")
	assignmentError := AssignChain(game, noArmyChain)
	if assignmentError == nil {
		t.Fatal("AssignChain() returned nil, want no-army reception error")
	}
	assignmentMessage := assignmentError.Error()
	if !strings.Contains(assignmentMessage, `"BRU"`) {
		t.Errorf("assignment message = %q, want receiving territory code", assignmentMessage)
	}
}

func TestAssignChainReceptionAndReplacement(t *testing.T) {
	game := orderTestState()
	first := mustParseChain(t, game, "JEA\nROS A BOI")
	if err := AssignChain(game, first); err != nil {
		t.Fatalf("first AssignChain() = %v, want nil", err)
	}
	if game.NextChainID != 2 || len(game.Chains) != 1 || game.Chains[0].ID != "C1" || game.Chains[0].ArmyID != "A1" {
		t.Fatalf("state after first assignment = %+v, want C1 on A1 and next C2", game)
	}
	if game.Armies[0].ChainID == nil || *game.Armies[0].ChainID != "C1" || game.Chains[0].Orders[0].ArmyID != "A1" {
		t.Errorf("inverse chain links = army %#v, order %#v", game.Armies[0], game.Chains[0].Orders[0])
	}
	if game.Nobles[0].LastEmissionTurn != game.Turn {
		t.Errorf("JEA last emission = %d, want %d", game.Nobles[0].LastEmissionTurn, game.Turn)
	}
	if first.ID != "" || first.ArmyID != "" || first.Orders[0].ArmyID != "" {
		t.Error("AssignChain mutated the submitted chain")
	}
	if err := game.Validate(); err != nil {
		t.Fatalf("Validate() after first assignment = %v", err)
	}

	assertAssignmentCategory(t, AssignChain(game, mustParseChain(t, game, "JEA\nH ROS")), ErrEmissionCapacity)

	game.Turn = 2
	game.Season = models.SeasonForTurn(game.Turn)
	replacement := mustParseChain(t, game, "ANN\nROS A BOI")
	if err := AssignChain(game, replacement); err != nil {
		t.Fatalf("replacement AssignChain() = %v, want nil", err)
	}
	if game.NextChainID != 3 || len(game.Chains) != 1 || game.Chains[0].ID != "C2" || game.Chains[0].NobleID != "N2" {
		t.Errorf("replacement result = chains %#v next=%d, want only C2 emitted by N2", game.Chains, game.NextChainID)
	}
	if game.Nobles[0].LastEmissionTurn != 1 || game.Nobles[1].LastEmissionTurn != 2 {
		t.Errorf("emission history = JEA:%d ANN:%d, want 1/2", game.Nobles[0].LastEmissionTurn, game.Nobles[1].LastEmissionTurn)
	}
	assertAssignmentCategory(t, AssignChain(game, mustParseChain(t, game, "ANN\nH ROS")), ErrEmissionCapacity)

	game.Turn = 3
	game.Season = models.SeasonForTurn(game.Turn)
	if err := AssignChain(game, mustParseChain(t, game, "ANN\nH ROS")); err != nil {
		t.Fatalf("next-turn emission = %v, want nil", err)
	}
	if game.Chains[0].ID != "C3" || game.NextChainID != 4 {
		t.Errorf("next-turn replacement = %#v next=%d, want C3 and 4", game.Chains, game.NextChainID)
	}
}

func TestAssignChainReceptionFailuresAreAtomic(t *testing.T) {
	for _, test := range []struct {
		name    string
		prepare func(*models.GameState)
		text    string
		want    error
	}{
		{"no army at first position", nil, "JEA\nBRU A BOI", ErrNoArmyOnPosition},
		{"foreign army at first position", nil, "JEA\nBOI A ROS", ErrArmyNotOwned},
		{"dungeon emitter", func(game *models.GameState) { game.Nobles[0].Status = models.NobleStatusDungeon }, "JEA\nROS A BOI", ErrNoblePrisoner},
		{"static invalid chain", nil, "JEA\nROS J BOI\nH BOI", ErrInvalidChain},
	} {
		t.Run(test.name, func(t *testing.T) {
			game := orderTestState()
			if test.prepare != nil {
				test.prepare(game)
			}
			before := marshalGame(t, game)
			chain := mustParseChain(t, game, test.text)
			assertAssignmentCategory(t, AssignChain(game, chain), test.want)
			if after := marshalGame(t, game); !bytes.Equal(before, after) {
				t.Fatalf("failed assignment mutated game:\n before=%s\n after=%s", before, after)
			}
		})
	}
}

func TestAssignChainAllowsHostageEmitter(t *testing.T) {
	t.Run("co-located hostage", func(t *testing.T) {
		game := orderTestState()
		game.Nobles[0].Status = models.NobleStatusHostage
		if err := AssignChain(game, mustParseChain(t, game, "JEA\nROS A BOI")); err != nil {
			t.Fatalf("AssignChain() = %v, want hostage noble to emit", err)
		}
	})
	t.Run("hostage held by an enemy army", func(t *testing.T) {
		game := orderTestState()
		if err := AssignChain(game, mustParseChain(t, game, "BOB\nBOI A ROS")); err != nil {
			t.Fatalf("AssignChain() = %v, want enemy-held hostage to emit for its owner's army", err)
		}
	})
}

func TestAssignChainDefersNonAdjacentDiagnostic(t *testing.T) {
	game := orderTestState()
	text := "JEA\nROS A BOI\nBOI S ROS\nBOI A FOU"
	chain := mustParseChain(t, game, text)
	validationErrors := ValidateChain(game, chain)
	if !hasValidationCode(validationErrors, "not_adjacent") {
		t.Fatalf("ValidateChain() = %#v, want a not_adjacent diagnostic", validationErrors)
	}
	for _, validationError := range validationErrors {
		if !validationError.Deferrable() {
			t.Fatalf("ValidateChain() = %#v, want only deferrable errors", validationErrors)
		}
	}
	if err := AssignChain(game, chain); err != nil {
		t.Fatalf("AssignChain() = %v, want reception to defer not_adjacent", err)
	}
	if len(game.Chains) != 1 || game.Chains[0].ID != "C1" || game.Chains[0].ArmyID != "A1" {
		t.Fatalf("stored chains = %#v, want C1 carried by A1", game.Chains)
	}
	if game.Chains[0].Orders[2].ID != "O3" || len(game.Chains[0].Orders[2].TargetIDs) != 1 || game.Chains[0].Orders[2].TargetIDs[0] != "FOU" {
		t.Errorf("stored O3 = %#v, want preserved FOU target", game.Chains[0].Orders[2])
	}
	if err := game.Validate(); err != nil {
		t.Fatalf("state after deferred reception is invalid: %v", err)
	}
}

func TestAssignChainRejectsDeferrableMixedWithBlockingErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		text   string
		mutate func(*models.Chain)
		want   error
	}{
		{"duplicate order id after non-adjacent order", "JEA\nROS A BOI\nBOI A FOU", func(chain *models.Chain) {
			chain.Orders[1].ID = chain.Orders[0].ID
		}, ErrInvalidChain},
		{"join not last and not adjacent", "JEA\nROS J BRU\nH BRU", nil, ErrInvalidChain},
	} {
		t.Run(test.name, func(t *testing.T) {
			game := orderTestState()
			chain := mustParseChain(t, game, test.text)
			if test.mutate != nil {
				test.mutate(&chain)
			}
			if validationErrors := ValidateChain(game, chain); !hasValidationCode(validationErrors, "not_adjacent") {
				t.Fatalf("ValidateChain() = %#v, want a not_adjacent diagnostic", validationErrors)
			}
			before := marshalGame(t, game)
			assertAssignmentCategory(t, AssignChain(game, chain), test.want)
			if after := marshalGame(t, game); !bytes.Equal(before, after) {
				t.Fatalf("rejected mixed chain mutated game:\n before=%s\n after=%s", before, after)
			}
		})
	}
}

func TestAssignChainDisperse(t *testing.T) {
	game := orderTestState()
	if err := AssignChain(game, mustParseChain(t, game, "JEA\nROS D ROS* BOI")); err != nil {
		t.Fatalf("wildcard D assignment = %v, want nil", err)
	}
	if got := game.Chains[0].Orders[0].NobleAssignments["ROS"]; !reflect.DeepEqual(got, []models.NobleCode{"*"}) {
		t.Errorf("stored wildcard = %#v, want *", got)
	}

}

func TestAssignChainDefersDisperseSizeAndNobleCoverage(t *testing.T) {
	for _, text := range []string{
		"JEA\nROS D ROS",
		"JEA\nROS D ROS BOI",
		"JEA\nROS A BOI\nBOI D FOU",
	} {
		t.Run(text, func(t *testing.T) {
			game := orderTestState()
			if err := AssignChain(game, mustParseChain(t, game, text)); err != nil {
				t.Fatalf("AssignChain() = %v, want execution-time dispersion validation", err)
			}
		})
	}
}

func TestAssignChainAcceptsRepeatedDisperseDestination(t *testing.T) {
	game := orderTestState()
	if err := AssignChain(game, mustParseChain(t, game, "JEA\nROS D BOI BOI")); err != nil {
		t.Fatalf("AssignChain() = %v, want repeated D destination to be accepted", err)
	}
	if got, want := game.Chains[0].Orders[0].TargetIDs, []models.TerritoryID{"BOI", "BOI"}; !reflect.DeepEqual(got, want) {
		t.Errorf("stored D targets = %#v, want %#v", got, want)
	}
}

func TestAssignChainRejectsPendingDisperseExecutor(t *testing.T) {
	game := orderTestState()
	chainID := models.ChainID("C1")
	game.Armies = append(game.Armies, models.Army{ID: "A3", OwnerID: "P1", TerritoryID: "FOU", Size: 1})
	state := game.TerritoryStates["FOU"]
	armyID := models.ArmyID("A3")
	ownerID := models.PlayerID("P1")
	state.Army = &armyID
	state.OwnerID = &ownerID
	game.TerritoryStates["FOU"] = state
	game.Armies[0].ChainID = &chainID
	game.Chains = []models.Chain{{
		ID:           chainID,
		NobleID:      "N1",
		ArmyID:       "A1",
		CurrentIndex: 0,
		Orders: []models.Order{{
			ID:               "O1",
			Type:             models.OrderTypeDisperse,
			ArmyID:           "A1",
			PositionID:       "ROS",
			TargetIDs:        []models.TerritoryID{"BOI"},
			NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BOI": {"JEA"}},
			Liaison:          models.LiaisonModeLoop,
		}},
		PendingDisperse: &models.PendingDisperse{
			ArmyID:           "A3",
			SourceID:         "FOU",
			TargetIDs:        []models.TerritoryID{"CHA"},
			NobleAssignments: map[models.TerritoryID][]models.NobleCode{},
		},
	}}
	game.NextArmyID = 4
	game.NextChainID = 2
	if err := game.Validate(); err != nil {
		t.Fatalf("prepared game state is invalid: %v", err)
	}
	before := marshalGame(t, game)
	assertAssignmentCategory(t, AssignChain(game, mustParseChain(t, game, "ANN\nH FOU")), ErrInvalidChain)
	if after := marshalGame(t, game); !bytes.Equal(before, after) {
		t.Fatalf("rejected pending executor assignment mutated game:\n before=%s\n after=%s", before, after)
	}
}

func TestAssignChainFailurePreservesExistingChain(t *testing.T) {
	game := orderTestState()
	if err := AssignChain(game, mustParseChain(t, game, "JEA\nROS A BOI")); err != nil {
		t.Fatalf("initial AssignChain() = %v", err)
	}
	game.Turn = 2
	game.Season = models.SeasonForTurn(game.Turn)
	before := marshalGame(t, game)
	assertAssignmentCategory(t, AssignChain(game, mustParseChain(t, game, "ANN\nROS J BOI\nH BOI")), ErrInvalidChain)
	if after := marshalGame(t, game); !bytes.Equal(before, after) {
		t.Fatalf("failed replacement mutated existing chain:\n before=%s\n after=%s", before, after)
	}
}

func TestAssignChainRejectsMalformedDirectChains(t *testing.T) {
	for _, test := range []struct {
		name   string
		text   string
		mutate func(*models.Chain)
	}{
		{"duplicate order id", "JEA\nROS A BOI\nH BOI", func(chain *models.Chain) {
			chain.Orders[1].ID = chain.Orders[0].ID
		}},
		{"D assignments on attack", "JEA\nROS A BOI", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryID][]models.NobleCode{"ROS": {"JEA"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			game := orderTestState()
			chain := mustParseChain(t, game, test.text)
			test.mutate(&chain)
			before := marshalGame(t, game)
			assertAssignmentCategory(t, AssignChain(game, chain), ErrInvalidChain)
			if after := marshalGame(t, game); !bytes.Equal(before, after) {
				t.Fatalf("malformed chain mutated game:\n before=%s\n after=%s", before, after)
			}
		})
	}
}

func TestAssignedChainJSONRoundTrip(t *testing.T) {
	game := orderTestState()
	if err := AssignChain(game, mustParseChain(t, game, "JEA\nROS D ROS* BOI")); err != nil {
		t.Fatalf("AssignChain() = %v", err)
	}
	data := marshalGame(t, game)
	var decoded models.GameState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal GameState: %v", err)
	}
	if !reflect.DeepEqual(*game, decoded) {
		t.Fatalf("GameState round trip mismatch:\nwant %#v\ngot %#v", *game, decoded)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("Validate decoded GameState: %v", err)
	}
}

func mustParseChain(t *testing.T, game *models.GameState, text string) models.Chain {
	t.Helper()
	chain, parseErrors := ParseChain(text, game)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseChain(%q) errors = %#v", text, parseErrors)
	}
	return chain
}

func hasValidationCode(validationErrors []ValidationError, want string) bool {
	for _, validationError := range validationErrors {
		if validationError.Code == want {
			return true
		}
	}
	return false
}

func assertAssignmentCategory(t *testing.T, err error, want error) {
	t.Helper()
	if err == nil {
		t.Fatalf("AssignChain() = nil, want errors.Is(_, %v)", want)
	}
	if !errors.Is(err, want) {
		t.Fatalf("AssignChain() = %v, want errors.Is(_, %v)", err, want)
	}
	var assignmentError *AssignmentError
	if !errors.As(err, &assignmentError) {
		t.Fatalf("AssignChain() = %T, want *AssignmentError", err)
	}
	if assignmentError.Code != want.Error() || assignmentError.Message == "" {
		t.Errorf("AssignmentError = %#v, want code %q and message", assignmentError, want.Error())
	}
}

func marshalGame(t *testing.T, game *models.GameState) []byte {
	t.Helper()
	data, err := json.Marshal(game)
	if err != nil {
		t.Fatalf("Marshal GameState: %v", err)
	}
	return data
}

func jsonCloneChain(chain models.Chain) models.Chain {
	data, err := json.Marshal(chain)
	if err != nil {
		panic(err)
	}
	var copy models.Chain
	if err := json.Unmarshal(data, &copy); err != nil {
		panic(err)
	}
	return copy
}

func orderTestState() *models.GameState {
	p1 := models.PlayerID("P1")
	p2 := models.PlayerID("P2")
	a1 := models.ArmyID("A1")
	a2 := models.ArmyID("A2")
	game := models.NewGameState()
	game.ID = "orders-test"
	game.Seed = "orders-test"
	game.Players = []models.Player{
		{ID: p1, Name: "Hugues", Color: "red"},
		{ID: p2, Name: "Adele", Color: "blue"},
	}
	game.Territories = []models.Territory{
		{ID: "ROS", Name: "Rosemont", Terrain: models.TerrainPlain, Adjacencies: []models.TerritoryID{"BOI", "FOU"}},
		{ID: "BOI", Name: "Boisclair", Terrain: models.TerrainForest, Adjacencies: []models.TerritoryID{"ROS", "BRU", "CHA"}},
		{ID: "BRU", Name: "Bruyeres", Terrain: models.TerrainHill, Adjacencies: []models.TerritoryID{"BOI", "FOU"}},
		{ID: "FOU", Name: "Fougeres", Terrain: models.TerrainSwamp, Adjacencies: []models.TerritoryID{"BRU", "ROS", "CHA"}},
		{ID: "CHA", Name: "Chavaux", Terrain: models.TerrainMountain, Adjacencies: []models.TerritoryID{"BOI", "FOU"}},
	}
	game.Armies = []models.Army{
		{ID: a1, OwnerID: p1, TerritoryID: "ROS", Size: 2},
		{ID: a2, OwnerID: p2, TerritoryID: "BOI", Size: 1},
	}
	game.NextArmyID = 3
	game.Nobles = []models.Noble{
		{ID: "N1", Code: "JEA", Name: "Jean", OwnerID: p1, LocationID: "ROS", Status: models.NobleStatusFree},
		{ID: "N2", Code: "ANN", Name: "Anne", OwnerID: p1, LocationID: "ROS", Status: models.NobleStatusFree},
		{ID: "N3", Code: "BOB", Name: "Bob", OwnerID: p2, LocationID: "ROS", Status: models.NobleStatusHostage},
		{ID: "N4", Code: "CAL", Name: "Calixte", OwnerID: p2, LocationID: "BOI", Status: models.NobleStatusFree},
	}
	game.TerritoryStates = map[models.TerritoryID]models.TerritoryState{
		"ROS": {OwnerID: &p1, Army: &a1, Infrastructures: []models.InfraID{}},
		"BOI": {OwnerID: &p2, Army: &a2, Infrastructures: []models.InfraID{}},
		"BRU": {Infrastructures: []models.InfraID{}},
		"FOU": {Infrastructures: []models.InfraID{}},
		"CHA": {Infrastructures: []models.InfraID{}},
	}
	if err := game.Validate(); err != nil {
		panic(err)
	}
	return game
}
