package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestSeatArmsFollowsTheCharterTable(t *testing.T) {
	cases := []struct {
		index int
		want  models.Arms
		color string
	}{
		{0, models.Arms{Field: models.TinctureGules, Metal: models.MetalOr, Charge: models.ChargeBend}, "#b3322b"},
		{1, models.Arms{Field: models.TinctureAzure, Metal: models.MetalArgent, Charge: models.ChargeFess}, "#2b5c9e"},
		{7, models.Arms{Field: models.TinctureMurrey, Metal: models.MetalArgent, Charge: models.ChargeBordure}, "#8c2f5a"},
		// Past eight seats: same tincture, inverted metal, charge shifted by four.
		{8, models.Arms{Field: models.TinctureGules, Metal: models.MetalArgent, Charge: models.ChargeSaltire}, "#b3322b"},
		{9, models.Arms{Field: models.TinctureAzure, Metal: models.MetalOr, Charge: models.ChargePale}, "#2b5c9e"},
	}
	for _, tc := range cases {
		arms, color := SeatArms(tc.index)
		if arms != tc.want || color != tc.color {
			t.Errorf("SeatArms(%d) = %+v %s, want %+v %s", tc.index, arms, color, tc.want, tc.color)
		}
	}
}

func TestSeatArmsNeverRepeatInAHotseatGame(t *testing.T) {
	seen := map[models.Arms]int{}
	for index := 0; index < MaximumGamePlayers; index++ {
		arms, _ := SeatArms(index)
		if previous, ok := seen[arms]; ok {
			t.Fatalf("seats %d and %d share the arms %+v", previous, index, arms)
		}
		seen[arms] = index
	}
}

func TestCreateGameGivesEachSeatItsTinctureAndArms(t *testing.T) {
	game, err := CreateGame("arms-test", []PlayerInit{{Name: "One"}, {Name: "Two", Color: "#123456"}}, testBalance(), loadGameTestAssets(t))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	first, second := game.Players[0], game.Players[1]
	if first.Color != "#b3322b" || first.Arms == nil || first.Arms.Charge != models.ChargeBend {
		t.Errorf("P1 = %s %+v, want gules with a bend", first.Color, first.Arms)
	}
	// An explicit color wins over the tincture; the arms still follow the seat.
	if second.Color != "#123456" || second.Arms == nil || second.Arms.Field != models.TinctureAzure {
		t.Errorf("P2 = %s %+v, want the explicit color and azure arms", second.Color, second.Arms)
	}
}
