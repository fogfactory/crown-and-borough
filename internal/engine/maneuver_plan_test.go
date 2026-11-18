package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func planOf(t *testing.T, order models.Order, size int, blocked []bool, nobles ...models.NobleCode) ManeuverPlan {
	t.Helper()
	return PlanManeuver(order, size, blocked, nobles)
}

func groupAt(plan ManeuverPlan, target models.TerritoryID) ManeuverLeg {
	for _, group := range plan.Groups {
		if group.Target == target {
			return group
		}
	}
	return ManeuverLeg{}
}

func TestPlanManeuverUnclaimedNoblesFollowTheLastGroup(t *testing.T) {
	order := maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1, models.TerritoryID("CCC"), rest)
	plan := planOf(t, order, 4, nil, "HUG", "TWO")
	if got := groupAt(plan, "CCC"); got.Troops != 3 || !got.CarriesChain || !reflect.DeepEqual(got.Nobles, []models.NobleCode{"HUG", "TWO"}) {
		t.Fatalf("CCC = %+v, want 3 troops, the sheet and both nobles", got)
	}
	if got := groupAt(plan, "BBB"); got.Troops != 1 || got.CarriesChain || len(got.Nobles) != 0 {
		t.Fatalf("BBB = %+v, want one troop and nothing else", got)
	}
	if plan.Stay.Troops != 0 || plan.StayListed || len(plan.Legs()) != 2 {
		t.Fatalf("stay = %+v, legs = %+v, want the origin left out", plan.Stay, plan.Legs())
	}
}

func TestPlanManeuverNamedNoblesKeepTheirGroup(t *testing.T) {
	order := maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1, models.TerritoryID("CCC"), rest)
	order.NobleAssignments["BBB"] = []models.NobleCode{"HUG"}
	plan := planOf(t, order, 4, nil, "HUG", "TWO")
	if got := groupAt(plan, "BBB").Nobles; !reflect.DeepEqual(got, []models.NobleCode{"HUG"}) {
		t.Fatalf("BBB nobles = %v, want [HUG]", got)
	}
	if got := groupAt(plan, "CCC").Nobles; !reflect.DeepEqual(got, []models.NobleCode{"TWO"}) {
		t.Fatalf("CCC nobles = %v, want the unnamed one [TWO]", got)
	}
}

func TestPlanManeuverOriginLastKeepsNoblesAndSheetHome(t *testing.T) {
	order := maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1, models.TerritoryID("AAA"), rest)
	plan := planOf(t, order, 4, nil, "HUG")
	if !plan.Stay.CarriesChain || !reflect.DeepEqual(plan.Stay.Nobles, []models.NobleCode{"HUG"}) || plan.Stay.Troops != 3 {
		t.Fatalf("stay = %+v, want 3 troops, HUG and the sheet", plan.Stay)
	}
	if legs := plan.Legs(); legs[0].Target != "AAA" || len(legs) != 2 {
		t.Fatalf("legs = %+v, want the origin first", legs)
	}
}

func TestPlanManeuverWildcardTakesEveryUnclaimedNoble(t *testing.T) {
	order := maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1, models.TerritoryID("CCC"), rest)
	order.NobleAssignments["CCC"] = []models.NobleCode{"*"}
	order.NobleAssignments["BBB"] = []models.NobleCode{"HUG"}
	plan := planOf(t, order, 4, nil, "HUG", "TWO", "THR")
	if got := groupAt(plan, "CCC").Nobles; !reflect.DeepEqual(got, []models.NobleCode{"TWO", "THR"}) {
		t.Fatalf("CCC nobles = %v, want the two unclaimed ones", got)
	}
	// Without knowing who stands there, the wildcard stays as written.
	if got := groupAt(PlanManeuver(order, 4, nil, nil), "CCC"); !got.AnyNobles || len(got.Nobles) != 0 {
		t.Fatalf("CCC = %+v, want the unknown-nobles marker and no literal code", got)
	}
}

func TestPlanManeuverEmptiedOriginSendsNoblesWithAGroupThatGetsTroops(t *testing.T) {
	// CCC+ gets nothing: HUG cannot stay alone on the emptied origin, it follows
	// BBB, and the sheet follows nobody since its group gets no troops.
	order := maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2, models.TerritoryID("CCC"), rest)
	plan := planOf(t, order, 2, nil, "HUG")
	if len(plan.Stay.Nobles) != 0 || !reflect.DeepEqual(groupAt(plan, "BBB").Nobles, []models.NobleCode{"HUG"}) {
		t.Fatalf("plan = %+v, want HUG with the BBB group", plan)
	}
	for _, leg := range plan.Legs() {
		if leg.CarriesChain {
			t.Fatalf("leg %+v carries the sheet, but its group gets no troops", leg)
		}
	}
	// With troops staying, the noble of an unfunded group stays with them.
	kept := planOf(t, maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1, models.TerritoryID("AAA"), 1, models.TerritoryID("CCC"), 1), 2, nil, "HUG")
	if !reflect.DeepEqual(kept.Stay.Nobles, []models.NobleCode{"HUG"}) || kept.Stay.Troops != 1 {
		t.Fatalf("plan = %+v, want HUG kept with the 2 troops that stay", kept)
	}
}

func TestPlanManeuverWeatherBlockedLastEntryHandsTheSheetBack(t *testing.T) {
	blocked := PlanManeuver(maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1, models.TerritoryID("CCC"), rest), 4, []bool{false, true}, nil)
	if blocked.LastTarget != "BBB" || !groupAt(blocked, "BBB").CarriesChain {
		t.Fatalf("plan = %+v, want BBB to carry the sheet", blocked)
	}
}

func TestPlanManeuverFlagsAssignmentsThatCannotBeHonored(t *testing.T) {
	order := maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1, models.TerritoryID("CCC"), rest)
	order.NobleAssignments["BBB"] = []models.NobleCode{"HUG"}
	if plan := planOf(t, order, 4, nil, "TWO"); !plan.Invalid {
		t.Error("a noble that does not stand with the army must invalidate the plan")
	}
	order.NobleAssignments["CCC"] = []models.NobleCode{"HUG"}
	if plan := planOf(t, order, 4, nil, "HUG"); !plan.Invalid {
		t.Error("a noble named twice must invalidate the plan")
	}
	// Without knowing who stands there nothing can be judged.
	if plan := PlanManeuver(maneuverOrder("AAA", false, models.TerritoryID("BBB"), 1), 4, nil, nil); plan.Invalid {
		t.Error("an order without assignments is never invalid")
	}
}

func TestPlanManeuverToleratesMissingCounts(t *testing.T) {
	order := maneuverOrder("AAA", false, models.TerritoryID("BBB"), 2)
	order.TargetIDs = append(order.TargetIDs, "CCC")
	plan := PlanManeuver(order, 4, nil, nil)
	if got := groupAt(plan, "CCC").Troops; got != 1 || plan.Stay.Troops != 1 {
		t.Fatalf("plan = %+v, want the destination without a count to take one troop", plan)
	}
}

func TestPreviewProjectsManeuverLegsForTheCurrentOrderOnly(t *testing.T) {
	state := testState(t, maneuverMap(), []models.Army{
		{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 4},
		{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
	})
	addNoble(state, "N1", "HUG", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "CCC")
	keepTestArmiesSupplied(state)
	validateTestState(t, state)
	preview, err := PreviewOrders(state, testBalance(), "P1", OrdersInput{Chains: []ChainSubmission{
		{Player: "P1", Noble: "HUG", Text: "HUG\nAAA M BBB/1 CCC+\nCCC M BBB+"},
	}})
	if err != nil {
		t.Fatalf("PreviewOrders: %v", err)
	}
	legs := preview.Chains[0].Legs
	if len(legs) != 2 || len(legs[0]) != 2 || len(legs[1]) != 1 {
		t.Fatalf("legs = %+v, want two legs for the current order and one for the next", legs)
	}
	if got := legs[0][1]; got.Target != "CCC" || !got.CarriesChain || !reflect.DeepEqual(got.Nobles, []models.NobleCode{"HUG"}) {
		t.Errorf("current order carrier = %+v, want CCC with the sheet and HUG", got)
	}
	if got := legs[1][0]; got.Target != "BBB" || len(got.Nobles) != 0 {
		t.Errorf("next order = %+v, want no unclaimed nobles placed", got)
	}
}
