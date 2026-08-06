package orders

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
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
ros o bob
ros k bob
ros j boi
`, game)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseChain() errors = %#v, want none", parseErrors)
	}
	if chain.NobleID != "N1" || chain.ID != "" || chain.ArmyID != "" || chain.CurrentIndex != 0 {
		t.Fatalf("parsed chain = %#v, want unassigned N1 chain at index 0", chain)
	}
	if len(chain.Orders) != 9 {
		t.Fatalf("parsed order count = %d, want 9", len(chain.Orders))
	}
	wantTypes := []models.OrderType{
		models.OrderTypeAttack,
		models.OrderTypeSupport,
		models.OrderTypeSupport,
		models.OrderTypeHold,
		models.OrderTypePillage,
		models.OrderTypeDisperse,
		models.OrderTypeHostage,
		models.OrderTypeDungeon,
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
	if got, want := chain.Orders[1].TargetIDs, []models.TerritoryID{"T02", "T05"}; !reflect.DeepEqual(got, want) {
		t.Errorf("offensive support targets = %#v, want %#v", got, want)
	}
	if got, want := chain.Orders[5].TargetIDs, []models.TerritoryID{"T01", "T02", "T04", "T02"}; !reflect.DeepEqual(got, want) {
		t.Errorf("D targets = %#v, want repeated destinations %#v", got, want)
	}
	if got, want := chain.Orders[5].NobleAssignments, map[models.TerritoryCode][]models.NobleCode{
		"ROS": {"JEA"},
		"BOI": {"ANN", "JEA"},
		"FOU": {"BOB"},
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("D assignments = %#v, want %#v", got, want)
	}
	if got := chain.Orders[6].NobleTargetIDs; !reflect.DeepEqual(got, []models.NobleID{"N3"}) {
		t.Errorf("O noble targets = %#v, want N3", got)
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
	if got, want := chain.Orders[0].NobleAssignments, map[models.TerritoryCode][]models.NobleCode{
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

	_, unknownNobles := ParseChain("JEA\nROS O ZZZ\nROS D ROS*ZZZ BOI", game)
	if len(unknownNobles) != 2 {
		t.Fatalf("unknown noble errors = %#v, want two errors", unknownNobles)
	}
	for index, wantLine := range []int{2, 3} {
		if unknownNobles[index].Code != ParseCodeInvalidCode || unknownNobles[index].Line != wantLine {
			t.Errorf("unknown noble error %d = %#v, want invalid_code on line %d", index, unknownNobles[index], wantLine)
		}
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
		"JEA\nROS O BOB",
		"JEA\nROS K BOB",
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
		{"pillage extra target", "JEA\nP ROS", func(chain *models.Chain) { chain.Orders[0].TargetIDs = []models.TerritoryID{"T02"} }, "unexpected_target"},
		{"too many noble targets", "JEA\nROS O BOB", func(chain *models.Chain) {
			chain.Orders[0].NobleTargetIDs = append(chain.Orders[0].NobleTargetIDs, "N4")
		}, "too_many_targets"},
		{"D destination not adjacent", "JEA\nROS D BRU", nil, "not_adjacent"},
		{"D assignment destination absent", "JEA\nROS D ROS", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryCode][]models.NobleCode{"BOI": {"JEA"}}
		}, "assignment_destination_not_declared"},
		{"D duplicate noble", "JEA\nROS D ROS BOI", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryCode][]models.NobleCode{"ROS": {"JEA"}, "BOI": {"JEA"}}
		}, "duplicate_assignment_noble"},
		{"D multiple wildcards", "JEA\nROS D ROS BOI", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryCode][]models.NobleCode{"ROS": {"*"}, "BOI": {"*"}}
		}, "multiple_wildcards"},
		{"D unknown noble assignment", "JEA\nROS D ROS", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryCode][]models.NobleCode{"ROS": {"ZZZ"}}
		}, "unknown_assignment_noble"},
		{"duplicate order id", "JEA\nROS A BOI\nH BOI", func(chain *models.Chain) {
			chain.Orders[1].ID = "O1"
		}, "duplicate_order_id"},
		{"assignments on non-D", "JEA\nROS A BOI", func(chain *models.Chain) {
			chain.Orders[0].NobleAssignments = map[models.TerritoryCode][]models.NobleCode{"ROS": {"JEA"}}
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
		{"prisoner emitter", func(game *models.GameState) { game.Nobles[0].Status = models.NobleStatusHostage }, "JEA\nROS A BOI", ErrNoblePrisoner},
		{"dungeon emitter", func(game *models.GameState) { game.Nobles[0].Status = models.NobleStatusDungeon }, "JEA\nROS A BOI", ErrNoblePrisoner},
		{"wrong D size", nil, "JEA\nROS D ROS", ErrDisperseSize},
		{"D does not cover nobles", nil, "JEA\nROS D ROS BOI", ErrNoblesNotCovered},
		{"O target is free", nil, "JEA\nROS O CAL", ErrNobleNotPrisoner},
		{"O target held elsewhere", func(game *models.GameState) { game.Nobles[2].LocationID = "T02" }, "JEA\nROS O BOB", ErrNobleNotPrisoner},
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

func TestAssignChainDisperseAndImmediatePrisonerChecks(t *testing.T) {
	game := orderTestState()
	if err := AssignChain(game, mustParseChain(t, game, "JEA\nROS D ROS* BOI")); err != nil {
		t.Fatalf("wildcard D assignment = %v, want nil", err)
	}
	if got := game.Chains[0].Orders[0].NobleAssignments["ROS"]; !reflect.DeepEqual(got, []models.NobleCode{"*"}) {
		t.Errorf("stored wildcard = %#v, want *", got)
	}

	for _, symbol := range []string{"O", "K"} {
		t.Run(symbol+" accepts co-located prisoner", func(t *testing.T) {
			state := orderTestState()
			if err := AssignChain(state, mustParseChain(t, state, "JEA\nROS "+symbol+" BOB")); err != nil {
				t.Fatalf("AssignChain() = %v, want nil", err)
			}
		})
	}

	state := orderTestState()
	if err := AssignChain(state, mustParseChain(t, state, "JEA\nROS A BOI\nBOI O CAL")); err != nil {
		t.Fatalf("future-position O = %v, want reception to defer it to P1.4", err)
	}
	state = orderTestState()
	if err := AssignChain(state, mustParseChain(t, state, "JEA\nH ROS\nROS O CAL")); err != nil {
		t.Fatalf("later O on receiving position = %v, want reception to defer it to P1.4", err)
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
	assertAssignmentCategory(t, AssignChain(game, mustParseChain(t, game, "ANN\nROS D ROS")), ErrDisperseSize)
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
			chain.Orders[0].NobleAssignments = map[models.TerritoryCode][]models.NobleCode{"ROS": {"JEA"}}
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
		{ID: "T01", Code: "ROS", Name: "Rosemont", Terrain: models.TerrainPlain, Adjacencies: []models.TerritoryID{"T02", "T04"}},
		{ID: "T02", Code: "BOI", Name: "Boisclair", Terrain: models.TerrainForest, Adjacencies: []models.TerritoryID{"T01", "T03", "T05"}},
		{ID: "T03", Code: "BRU", Name: "Bruyeres", Terrain: models.TerrainHill, Adjacencies: []models.TerritoryID{"T02", "T04"}},
		{ID: "T04", Code: "FOU", Name: "Fougeres", Terrain: models.TerrainSwamp, Adjacencies: []models.TerritoryID{"T03", "T01", "T05"}},
		{ID: "T05", Code: "CHA", Name: "Chavaux", Terrain: models.TerrainMountain, Adjacencies: []models.TerritoryID{"T02", "T04"}},
	}
	game.Armies = []models.Army{
		{ID: a1, OwnerID: p1, TerritoryID: "T01", Size: 2},
		{ID: a2, OwnerID: p2, TerritoryID: "T02", Size: 1},
	}
	game.NextArmyID = 3
	game.Nobles = []models.Noble{
		{ID: "N1", Code: "JEA", Name: "Jean", OwnerID: p1, LocationID: "T01", Status: models.NobleStatusFree},
		{ID: "N2", Code: "ANN", Name: "Anne", OwnerID: p1, LocationID: "T01", Status: models.NobleStatusFree},
		{ID: "N3", Code: "BOB", Name: "Bob", OwnerID: p2, LocationID: "T01", Status: models.NobleStatusHostage},
		{ID: "N4", Code: "CAL", Name: "Calixte", OwnerID: p2, LocationID: "T02", Status: models.NobleStatusFree},
	}
	game.TerritoryStates = map[models.TerritoryID]models.TerritoryState{
		"T01": {OwnerID: &p1, Army: &a1, Infrastructures: []models.InfraID{}},
		"T02": {OwnerID: &p2, Army: &a2, Infrastructures: []models.InfraID{}},
		"T03": {Infrastructures: []models.InfraID{}},
		"T04": {Infrastructures: []models.InfraID{}},
		"T05": {Infrastructures: []models.InfraID{}},
	}
	if err := game.Validate(); err != nil {
		panic(err)
	}
	return game
}
