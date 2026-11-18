package orders

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestParseManeuverSyntax(t *testing.T) {
	game := orderTestState()
	chain, parseErrors := ParseChain("jea\nros m boi/2*jea fou+*\nros m boi\nros m ros+\n", game)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseChain() errors = %#v", parseErrors)
	}
	first := chain.Orders[0]
	if first.Type != models.OrderTypeManeuver || !reflect.DeepEqual(first.TargetIDs, []models.TerritoryID{"BOI", "FOU"}) || !reflect.DeepEqual(first.Counts, []int{2, models.ManeuverRest}) {
		t.Fatalf("order = %#v", first)
	}
	if got := first.NobleAssignments["BOI"]; !reflect.DeepEqual(got, []models.NobleCode{"JEA"}) {
		t.Fatalf("assignments = %#v", first.NobleAssignments)
	}
	if got := chain.Orders[1].Counts; !reflect.DeepEqual(got, []int{1}) {
		t.Fatalf("bare destination counts = %v, want [1]", got)
	}
	if chain.Orders[2].TargetIDs[0] != "ROS" || chain.Orders[2].Counts[0] != models.ManeuverRest {
		t.Fatalf("stay order = %#v", chain.Orders[2])
	}
}

func TestParseManeuverRejectsInvalidForms(t *testing.T) {
	game := orderTestState()
	for _, line := range []string{"ros m", "ros m boi/0", "ros m boi/x", "ros m boi+2", "ros m boi/2*zzz", "ros m +", "ros m /2", "ros m *", "ros m boi/+", "ros m boi/2+", "ros m boi/05", "ros m boi/+2", "ros m boi/2/3", "ros m boi/-1"} {
		if _, parseErrors := ParseChain("jea\n"+line, game); len(parseErrors) == 0 {
			t.Errorf("%q parsed without error", line)
		}
	}
}

func TestValidateManeuver(t *testing.T) {
	game := orderTestState()
	for line, want := range map[string]string{
		"ros m boi+ fou+":         "maneuver_rest_misplaced",
		"ros m boi+ fou/1":        "maneuver_rest_misplaced",
		"ros m boi/1*jea fou*jea": "duplicate_assignment_noble",
	} {
		chain := mustParseChain(t, game, "jea\n"+line)
		if !hasValidationCode(ValidateChain(game, chain), want) {
			t.Errorf("%q: want validation code %q, got %#v", line, want, ValidateChain(game, chain))
		}
	}
	chain := mustParseChain(t, game, "jea\nros m boi/2 ros+")
	if errs := ValidateChain(game, chain); len(errs) != 0 {
		t.Errorf("valid order rejected: %#v", errs)
	}
}

func TestEffectiveEmittedTurn(t *testing.T) {
	if got := (models.Chain{EmittedTurn: 3, CurrentIndex: 2}).EffectiveEmittedTurn(9); got != 3 {
		t.Errorf("field must win, got %d", got)
	}
	if got := (models.Chain{CurrentIndex: 2}).EffectiveEmittedTurn(9); got != 7 {
		t.Errorf("fallback = %d, want 7", got)
	}
}

func TestParseManeuverPreciseNobleSplit(t *testing.T) {
	game := orderTestState()
	chain, parseErrors := ParseChain("jea\nros m boi/2*jea fou* cha+", game)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseChain() errors = %#v", parseErrors)
	}
	order := chain.Orders[0]
	if !reflect.DeepEqual(order.Counts, []int{2, 1, models.ManeuverRest}) {
		t.Fatalf("counts = %v, want [2 1 -1]", order.Counts)
	}
	if !reflect.DeepEqual(order.NobleAssignments["BOI"], []models.NobleCode{"JEA"}) ||
		!reflect.DeepEqual(order.NobleAssignments["FOU"], []models.NobleCode{"*"}) ||
		len(order.NobleAssignments["CHA"]) != 0 {
		t.Fatalf("assignments = %#v", order.NobleAssignments)
	}
}
