package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestProjectStateDistinguishesKnownHiddenAndAbsentChains(t *testing.T) {
	state := projectTestState()
	privacy := ensurePrivacy(state)
	snapshot := makeChainSnapshot(state.Chains[0], state.Turn)
	putChainSnapshot(privacy, "P1", snapshot)

	known := projectStateForPlayer(state, "P1")
	if got := known.Territories[0].Army.Chain; got == nil || got.Visibility != "known" {
		t.Fatalf("known chain = %#v, want visibility known", got)
	}
	if got := known.Territories[1].Army.Chain; got != nil {
		t.Fatalf("absent chain = %#v, want nil", got)
	}

	hidden := projectStateForPlayer(state, "P2")
	if got := hidden.Territories[0].Army.Chain; got == nil || got.Visibility != "hidden" {
		t.Fatalf("hidden chain = %#v, want visibility hidden", got)
	}
	data, err := json.Marshal(hidden)
	if err != nil {
		t.Fatalf("marshal hidden state: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode hidden state: %v", err)
	}
	chain := firstArmyChainObject(t, document)
	if !reflect.DeepEqual(chain, map[string]any{"visibility": "hidden"}) {
		t.Errorf("hidden chain JSON = %#v, want visibility-only object", chain)
	}
}

func TestThirdPartySnapshotSurvivesReplacementUntilContradiction(t *testing.T) {
	before := projectTestState()
	privacy := ensurePrivacy(before)
	oldSnapshot := makeChainSnapshot(before.Chains[0], before.Turn)
	putChainSnapshot(privacy, "P1", oldSnapshot)
	putChainSnapshot(privacy, "P2", oldSnapshot)

	after := clonePrivacyTestState(t, before)
	after.Chains = []models.Chain{{
		ID: "C2", NobleID: "N1", ArmyID: "A1", CurrentIndex: 0,
		Orders: []models.Order{{ID: "O3", ArmyID: "A1", Type: models.OrderTypeHold, PositionID: "ROS", Liaison: models.LiaisonModeSingle}},
	}}
	newChainID := models.ChainID("C2")
	after.Armies[0].ChainID = &newChainID

	trackTurnPrivacy(before, after, engine.OrdersInput{
		Chains: []engine.ChainSubmission{{Player: "P1", Noble: "HUG", Text: "HUG\nH ROS"}},
	}, engine.TurnReport{
		Receptions: []engine.ReceptionReport{{Player: "P1", Noble: "HUG", Received: true}},
	})

	if _, exists := after.Privacy.ChainKnowledge["P1"]["C1"]; exists {
		t.Error("army owner retained knowledge of the replaced chain")
	}
	if _, exists := after.Privacy.ChainKnowledge["P1"]["C2"]; !exists {
		t.Error("army owner did not learn the replacement chain")
	}
	if _, exists := after.Privacy.ChainKnowledge["P2"]["C1"]; !exists {
		t.Error("third party lost the old chain at reception")
	}
	if _, exists := after.Privacy.ChainKnowledge["P2"]["C2"]; exists {
		t.Error("third party learned the replacement chain")
	}

	// BOI is a destination named by the known chain, so compatible progression
	// does not discard the third party's information.
	after.Armies[0].TerritoryID = "BOI"
	reconcileChainKnowledge(after)
	if _, exists := after.Privacy.ChainKnowledge["P2"]["C1"]; !exists {
		t.Error("compatible third-party snapshot was purged")
	}

	after.Armies[0].TerritoryID = "BRU"
	reconcileChainKnowledge(after)
	if _, exists := after.Privacy.ChainKnowledge["P2"]["C1"]; exists {
		t.Error("contradicted third-party snapshot was retained")
	}
}

func TestHostageHolderLearnsTheEmittedChain(t *testing.T) {
	before := projectTestState()
	before.Nobles[0].Status = models.NobleStatusHostage
	before.Nobles[0].LocationID = "BOI"
	after := clonePrivacyTestState(t, before)

	recordChainKnowledge(before, after, before.Chains[0], nil)
	if _, exists := after.Privacy.ChainKnowledge["P2"][before.Chains[0].ID]; !exists {
		t.Fatal("holder of the hostage noble did not learn the chain")
	}
}

func TestProjectStateDoesNotMutatePrivacyState(t *testing.T) {
	state := projectTestState()
	putChainSnapshot(ensurePrivacy(state), "P1", makeChainSnapshot(state.Chains[0], state.Turn))
	before, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal before projection: %v", err)
	}

	_ = projectStateForPlayer(state, "P1")
	_ = projectStateForPlayer(state, "P2")
	_ = projectState(state)

	after, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal after projection: %v", err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("projection mutated source state\nbefore=%s\nafter=%s", before, after)
	}
}

func TestProjectReportUsesExactParticipationAndGeneralSpectatorView(t *testing.T) {
	report := engine.TurnReport{
		Combats: []engine.CombatReport{{
			Territory: "BOI", BaseDefense: 1, Defense: 2, CastleBonus: 0,
			Contenders: []engine.CombatContender{{ArmyID: "A1", OwnerID: "P1", Force: 3}},
			Supporters: []models.ArmyID{"A2"}, Winner: "A1", CutSupporters: []models.ArmyID{},
			Reason: "attack_wins",
		}},
	}
	privacy := &models.PrivacyMeta{
		ChainKnowledge: map[models.PlayerID]map[models.ChainID]models.ChainSnapshot{},
		CombatParticipation: map[models.PlayerID]map[string]bool{
			"P1": {"combat-BOI": true},
		},
	}

	exact := projectReport(report, "P1", privacy)
	if exact.Combats[0].Visibility != "exact" {
		t.Fatalf("participant visibility = %q, want exact", exact.Combats[0].Visibility)
	}
	general := projectReport(report, "P3", privacy)
	if general.Combats[0].Visibility != "general" {
		t.Fatalf("spectator visibility = %q, want general", general.Combats[0].Visibility)
	}
	data, err := json.Marshal(general)
	if err != nil {
		t.Fatalf("marshal general report: %v", err)
	}
	for _, forbidden := range []string{"contenders", "supporters", "force", "army", "owner", "baseDefense", "defense", "castleBonus", "winner", "dislodged"} {
		if strings.Contains(string(data), `"`+forbidden+`"`) {
			t.Errorf("general report exposes private field %q: %s", forbidden, data)
		}
	}
	if !strings.Contains(string(data), `"visibility":"general"`) || !strings.Contains(string(data), `"summary"`) {
		t.Errorf("general report = %s, want general summary", data)
	}
}

func TestProjectReportRedactsUnknownOrderDetails(t *testing.T) {
	report := engine.TurnReport{
		Orders: []engine.OrderReport{{
			Army: "A1", Chain: "C1", Order: "O1", Owner: "P1", Noble: "HUG",
			Type: models.OrderTypeAttack, Source: "ROS", Target: "BOI",
			Targets: []models.TerritoryID{"BOI"}, Liaison: models.LiaisonModeSingle,
			Outcome: engine.OutcomeSuccess, Progression: engine.ProgressionAdvanced,
			IndexAfter: 1,
		}},
	}
	privacy := &models.PrivacyMeta{
		ChainKnowledge:      map[models.PlayerID]map[models.ChainID]models.ChainSnapshot{},
		CombatParticipation: map[models.PlayerID]map[string]bool{},
	}
	view := projectReport(report, "P2", privacy)
	if view.Orders[0].Visibility != "hidden" || view.Orders[0].Outcome != engine.OutcomeSuccess {
		t.Fatalf("redacted order = %#v, want hidden successful outcome", view.Orders[0])
	}
	data, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal redacted report: %v", err)
	}
	for _, forbidden := range []string{"A1", "C1", "O1", "HUG", "ROS", "BOI", "targets", "source"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("redacted order report exposes %q: %s", forbidden, data)
		}
	}
}

func TestProjectReportNormalizesNilCollections(t *testing.T) {
	view := projectReport(engine.TurnReport{}, "P1", nil)
	data, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal empty projected report: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode empty projected report: %v", err)
	}
	for _, field := range []string{"players", "receptions", "supply", "famines", "combats", "orders", "moves", "nobles"} {
		if value, ok := document[field]; !ok || value == nil {
			t.Errorf("projected report field %q = %#v, want JSON array", field, value)
		}
	}
}

func TestCombatParticipationIncludesSupportingArmies(t *testing.T) {
	before := &models.GameState{Armies: []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
		{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
	}}
	after := &models.GameState{Armies: append([]models.Army(nil), before.Armies...)}
	privacy := ensurePrivacy(after)
	trackCombatParticipation(before, after, []engine.CombatReport{{
		Territory:  "CCC",
		Contenders: []engine.CombatContender{{ArmyID: "A1", OwnerID: "P1"}},
		Supporters: []models.ArmyID{"A2"},
	}}, privacy)

	if !privacy.CombatParticipation["P1"]["combat-CCC"] {
		t.Error("attacker was not marked as a combat participant")
	}
	if !privacy.CombatParticipation["P2"]["combat-CCC"] {
		t.Error("supporting player was not marked as a combat participant")
	}
}

func TestStateHTTPPlayerQueryServesPrivateProjection(t *testing.T) {
	state := projectTestState()
	putChainSnapshot(ensurePrivacy(state), "P1", makeChainSnapshot(state.Chains[0], state.Turn))
	session := &Session{game: state}

	recorder := httptest.NewRecorder()
	session.StateHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/state?player=P2", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET state?player=P2 = %d: %s", recorder.Code, recorder.Body.String())
	}
	var view StateView
	if err := json.Unmarshal(recorder.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode private state: %v", err)
	}
	if view.Territories[0].Army.Chain == nil || view.Territories[0].Army.Chain.Visibility != "hidden" {
		t.Fatalf("P2 state chain = %#v, want hidden", view.Territories[0].Army.Chain)
	}

	unknown := httptest.NewRecorder()
	session.StateHTTP(unknown, httptest.NewRequest(http.MethodGet, "/api/state?player=P9", nil))
	if unknown.Code != http.StatusBadRequest {
		t.Errorf("GET state?player=P9 = %d, want %d", unknown.Code, http.StatusBadRequest)
	}
}

func TestOrdersHTTPTracksPrivateChainProjection(t *testing.T) {
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	session, err := NewSession("privacy-orders", []engine.PlayerInit{{Name: "One"}, {Name: "Two"}}, balance, assets)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	session.mu.RLock()
	noble := session.game.Nobles[0]
	location := noble.LocationID
	session.mu.RUnlock()

	submit := func(player, query, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		session.OrdersHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/orders?player="+query, strings.NewReader(body)))
		return recorder
	}
	chainText := string(noble.Code) + "\n(H " + string(location) + ")"
	encodedChainText := strings.ReplaceAll(chainText, "\n", `\n`)
	first := submit("P1", "P1", `{"player":"P1","chains":[{"player":"P1","noble":"`+string(noble.Code)+`","text":"`+encodedChainText+`"}],"winter":[]}`)
	if first.Code != http.StatusOK {
		t.Fatalf("P1 orders = %d: %s", first.Code, first.Body.String())
	}
	second := submit("P2", "P2", `{"player":"P2","chains":[],"winter":[]}`)
	if second.Code != http.StatusOK {
		t.Fatalf("P2 orders = %d: %s", second.Code, second.Body.String())
	}

	var p2 struct {
		State StateView `json:"state"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &p2); err != nil {
		t.Fatalf("decode P2 response: %v", err)
	}
	var p2Chain *ChainView
	for _, territory := range p2.State.Territories {
		if territory.Army != nil && territory.Army.Owner == "P1" {
			p2Chain = territory.Army.Chain
			break
		}
	}
	if p2Chain == nil || p2Chain.Visibility != "hidden" {
		t.Fatalf("P2 chain = %#v, want hidden", p2Chain)
	}

	p1State := httptest.NewRecorder()
	session.StateHTTP(p1State, httptest.NewRequest(http.MethodGet, "/api/state?player=P1", nil))
	if p1State.Code != http.StatusOK {
		t.Fatalf("GET P1 state = %d: %s", p1State.Code, p1State.Body.String())
	}
	var p1 StateView
	if err := json.Unmarshal(p1State.Body.Bytes(), &p1); err != nil {
		t.Fatalf("decode P1 state: %v", err)
	}
	for _, territory := range p1.Territories {
		if territory.Army != nil && territory.Army.Owner == "P1" {
			if territory.Army.Chain == nil || territory.Army.Chain.Visibility != "known" {
				t.Fatalf("P1 chain = %#v, want known", territory.Army.Chain)
			}
			return
		}
	}
	t.Fatal("P1 army not found in private state")
}

func clonePrivacyTestState(t *testing.T, source *models.GameState) *models.GameState {
	t.Helper()
	data, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("marshal state clone: %v", err)
	}
	var clone models.GameState
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatalf("unmarshal state clone: %v", err)
	}
	return &clone
}

func firstArmyChainObject(t *testing.T, document map[string]any) map[string]any {
	t.Helper()
	territories, ok := document["territories"].([]any)
	if !ok {
		t.Fatalf("territories = %#v, want array", document["territories"])
	}
	for _, rawTerritory := range territories {
		territory, ok := rawTerritory.(map[string]any)
		if !ok {
			continue
		}
		army, ok := territory["army"].(map[string]any)
		if !ok {
			continue
		}
		chain, ok := army["chain"].(map[string]any)
		if ok {
			return chain
		}
	}
	t.Fatal("document contains no army chain object")
	return nil
}
