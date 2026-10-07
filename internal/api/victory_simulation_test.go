package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

func simulationFixture(t *testing.T) (http.Handler, store.GameStore, store.GameID, []string) {
	t.Helper()
	handler, gameStore := newPreviewTestHandler(t)
	game := createGameHTTP(t, handler, "P1", `{"name":"Sim","seed":"victory-sim","players":["One","Two"]}`)
	snapshot, err := gameStore.State(context.Background(), store.Actor{ID: "P1", Development: true}, game.ID)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	var own, other string
	for _, noble := range snapshot.State.Nobles {
		if string(noble.OwnerID) == "P1" && own == "" {
			own = noble.Code
		}
		if string(noble.OwnerID) != "P1" && other == "" {
			other = noble.Code
		}
	}
	return handler, gameStore, game.ID, []string{own, other}
}

func TestVictorySimulationProjectsMarriageWithoutCommittingIt(t *testing.T) {
	handler, gameStore, id, codes := simulationFixture(t)
	path := "/api/games/" + string(id) + "/victory/simulate?player=P1"
	response := requestGames(t, handler, http.MethodPost, path, `{"actions":[{"type":"marry","noble":"`+codes[0]+`","other":"`+codes[1]+`"}]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("simulate = %d: %s", response.Code, response.Body.String())
	}
	var view VictorySimulationView
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.Current["P1"].Victory.Mode != "solo" || view.Current["P1"].Victory.Required < 1 {
		t.Fatalf("current reading = %+v", view.Current["P1"])
	}
	if len(view.State.Marriages) != 1 {
		t.Fatalf("projected state marriages = %d, want 1", len(view.State.Marriages))
	}
	after, _ := gameStore.State(context.Background(), store.Actor{ID: "P1", Development: true}, id)
	if len(after.State.Marriages) != 0 {
		t.Fatal("simulation committed the marriage")
	}
}

func TestVictorySimulationKillMovesNobleToDeceased(t *testing.T) {
	handler, _, id, codes := simulationFixture(t)
	path := "/api/games/" + string(id) + "/victory/simulate?player=P1"
	response := requestGames(t, handler, http.MethodPost, path, `{"actions":[{"type":"kill","noble":"`+codes[0]+`"}]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("kill = %d: %s", response.Code, response.Body.String())
	}
	var view VictorySimulationView
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(view.State.Deceased) != 1 || string(view.State.Deceased[0].Code) != codes[0] {
		t.Fatalf("deceased = %+v", view.State.Deceased)
	}
}

func TestVictorySimulationRejectsImpossibleAction(t *testing.T) {
	handler, _, id, _ := simulationFixture(t)
	path := "/api/games/" + string(id) + "/victory/simulate?player=P1"
	for _, body := range []string{
		`{"actions":[{"type":"claim","noble":"ZZZ","other":"YYY"}]}`,
		`{"actions":[{"type":"divorce","noble":"AAA","other":"BBB"}]}`,
		`{"actions":[{"type":"fly","noble":"AAA"}]}`,
	} {
		if response := requestGames(t, handler, http.MethodPost, path, body); response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s = %d: %s", body, response.Code, response.Body.String())
		}
	}
}
