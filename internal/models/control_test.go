package models_test

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestTerritoryControllerPriority(t *testing.T) {
	g := validFiefState()
	// BRU: P1's fief capital. ROS: fief member holding P1's army. BCL: fief
	// member occupied by P2's army. FOU: bare ground.
	g.Players[0].CapitalCastleID = ptrInfraID("I2")

	for _, tc := range []struct {
		territory  models.TerritoryID
		want       models.PlayerID
		controlled bool
	}{
		{"BRU", "P1", true},
		{"ROS", "P1", true},
		{"BCL", "P1", true}, // occupied by P2's army, still P1's fief member
		{"FOU", "", false},
	} {
		got, controlled := g.TerritoryController(tc.territory)
		if controlled != tc.controlled || got != tc.want {
			t.Errorf("TerritoryController(%s) = %q, %t; want %q, %t", tc.territory, got, controlled, tc.want, tc.controlled)
		}
	}
}

func TestTerritoryControllerFromCapitalAndArmy(t *testing.T) {
	g := validState()
	g.Players[0].CapitalCastleID = ptrInfraID("I2")

	if got, controlled := g.TerritoryController("BRU"); !controlled || got != "P1" {
		t.Errorf("capital BRU controller = %q, %t; want P1", got, controlled)
	}
	if got, controlled := g.TerritoryController("BCL"); !controlled || got != "P2" {
		t.Errorf("army-held BCL controller = %q, %t; want P2", got, controlled)
	}
	if got, controlled := g.TerritoryController("FOU"); controlled {
		t.Errorf("bare FOU controller = %q; want none", got)
	}
}

func TestTerritoryControllerIgnoresNeutralArmy(t *testing.T) {
	g := validState()
	g.Armies[1].OwnerID = models.NeutralPlayerID

	if got, controlled := g.TerritoryController("BCL"); controlled {
		t.Errorf("BCL controller = %q; want none: a revolt army administers nothing", got)
	}
	if _, controlled := g.TerritoryControllers()["BCL"]; controlled {
		t.Errorf("TerritoryControllers() controls BCL; want it uncontrolled")
	}
}

func TestTerritoryControllersMatchesTerritoryController(t *testing.T) {
	for name, g := range map[string]*models.GameState{"plain": validState(), "fief": validFiefState()} {
		g.Players[0].CapitalCastleID = ptrInfraID("I2")
		bulk := g.TerritoryControllers()
		for _, territory := range g.Territories {
			want, wantControlled := g.TerritoryController(territory.ID)
			got, gotControlled := bulk[territory.ID]
			if gotControlled != wantControlled || got != want {
				t.Errorf("%s: TerritoryControllers()[%s] = %q, %t; TerritoryController = %q, %t", name, territory.ID, got, gotControlled, want, wantControlled)
			}
		}
	}
}
