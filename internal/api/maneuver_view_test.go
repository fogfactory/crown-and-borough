package api

import (
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
	view := projectOrder(order)
	if !reflect.DeepEqual(view.Counts, []int{2, models.ManeuverRest}) {
		t.Fatalf("counts = %v, want [2 -1]", view.Counts)
	}
	view.Counts[0] = 9
	if order.Counts[0] != 2 {
		t.Fatal("the view must not alias the stored counts")
	}
}
