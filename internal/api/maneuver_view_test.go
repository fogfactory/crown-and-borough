package api

import (
	"encoding/json"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestProjectOrderKeepsManeuverCounts(t *testing.T) {
	order := models.Order{
		Type:       models.OrderTypeManeuver,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB", "CCC"},
		Counts:     []int{2, models.ManeuverRest},
		Liaison:    models.LiaisonModeSingle,
	}
	view := projectOrder(order, nil)
	if !reflect.DeepEqual(view.Counts, []int{2, models.ManeuverRest}) {
		t.Fatalf("counts = %v, want [2 -1]", view.Counts)
	}
	view.Counts[0] = 9
	if order.Counts[0] != 2 {
		t.Fatal("the view must not alias the stored counts")
	}
}

func TestProjectStateCarriesManeuverLegs(t *testing.T) {
	state := projectTestState()
	state.Chains[0].CurrentIndex = 0
	state.Chains[0].Orders = []models.Order{{
		ID: "O1", ArmyID: "A1", Type: models.OrderTypeManeuver, PositionID: "ROS",
		TargetIDs: []models.TerritoryID{"BOI"}, Counts: []int{models.ManeuverRest},
		Liaison: models.LiaisonModeSingle,
	}}
	view := projectStateForPlayer(state, models.SpectatorViewer, assetgen.Balance{})
	legs := view.Territories[0].Army.Chain.Orders[0].Legs
	if len(legs) != 1 || legs[0].Target != "BOI" || !legs[0].CarriesChain || legs[0].Troops != state.Armies[0].Size {
		t.Fatalf("legs = %+v, want the whole army going to BOI with the order sheet", legs)
	}
}

func TestManeuverLegsJSONShapeIsPinned(t *testing.T) {
	view := projectOrder(models.Order{
		Type: models.OrderTypeManeuver, PositionID: "ROS", TargetIDs: []models.TerritoryID{"BOI"},
		Counts: []int{models.ManeuverRest}, Liaison: models.LiaisonModeSingle,
	}, []engine.ManeuverLeg{{Target: "BOI", Troops: 3, Nobles: []models.NobleCode{"HUG"}, AnyNobles: true, CarriesChain: true}})
	encoded, err := json.Marshal(view.Legs)
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"target":"BOI","troops":3,"nobles":["HUG"],"anyNobles":true,"carriesChain":true}]`
	if string(encoded) != want {
		t.Fatalf("legs JSON = %s, want %s", encoded, want)
	}
}

func TestHiddenChainExposesNoManeuverLegs(t *testing.T) {
	state := projectTestState()
	state.Chains[0].CurrentIndex = 0
	state.Chains[0].Orders = []models.Order{{
		ID: "O1", ArmyID: "A1", Type: models.OrderTypeManeuver, PositionID: "ROS",
		TargetIDs: []models.TerritoryID{"BOI"}, Counts: []int{models.ManeuverRest},
		Liaison: models.LiaisonModeSingle,
	}}
	viewer := models.PlayerID("P2")
	view := projectStateForPlayer(state, viewer, assetgen.Balance{})
	chain := view.Territories[0].Army.Chain
	if chain == nil || chain.Visibility != "hidden" || len(chain.Orders) != 0 {
		t.Fatalf("chain = %+v, want a hidden chain without orders or legs", chain)
	}
}
