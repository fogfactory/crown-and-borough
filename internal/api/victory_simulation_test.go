package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestVictorySimulationProjectsMarriageWithoutCommittingIt(t *testing.T) {
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
	path := "/api/games/" + string(game.ID) + "/victory/simulate?player=P1"
	response := requestGames(t, handler, http.MethodPost, path, `{"marriages":[{"noble":"`+own+`","spouse":"`+other+`"}]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("simulate = %d: %s", response.Code, response.Body.String())
	}
	var view VictorySimulationView
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.Current.Victory.Mode != "solo" || view.Current.Victory.Required < 1 {
		t.Fatalf("current reading = %+v", view.Current)
	}
	if view.Projected.Score.Total < view.Current.Score.Total {
		t.Fatalf("a marriage lowered the score: %+v", view)
	}
	after, _ := gameStore.State(context.Background(), store.Actor{ID: "P1", Development: true}, game.ID)
	if len(after.State.Marriages) != len(snapshot.State.Marriages) {
		t.Fatal("simulation committed the marriage")
	}
}

func TestVictorySimulationRejectsImpossibleScenario(t *testing.T) {
	handler, _ := newPreviewTestHandler(t)
	game := createGameHTTP(t, handler, "P1", `{"name":"Sim","seed":"victory-sim","players":["One","Two"]}`)
	path := "/api/games/" + string(game.ID) + "/victory/simulate?player=P1"
	response := requestGames(t, handler, http.MethodPost, path, `{"claims":[{"heir":"ZZZ","target":"YYY"}]}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("simulate = %d: %s", response.Code, response.Body.String())
	}
}

func TestVictorySimulationAcceptsDeathsAndMarriageEnds(t *testing.T) {
	handler, gameStore := newPreviewTestHandler(t)
	game := createGameHTTP(t, handler, "P1", `{"name":"Sim","seed":"victory-sim","players":["One","Two"]}`)
	snapshot, err := gameStore.State(context.Background(), store.Actor{ID: "P1", Development: true}, game.ID)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	dead := snapshot.State.Nobles[0].Code
	path := "/api/games/" + string(game.ID) + "/victory/simulate?player=P1"
	response := requestGames(t, handler, http.MethodPost, path, `{"deaths":["`+dead+`"]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("deaths = %d: %s", response.Code, response.Body.String())
	}
	response = requestGames(t, handler, http.MethodPost, path, `{"marriageEnds":[{"noble":"AAA","spouse":"BBB"}]}`)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("marriage end = %d: %s", response.Code, response.Body.String())
	}
}
