package assetgen

import (
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestLoadSpecialOrdersBalance(t *testing.T) {
	dir := t.TempDir()
	writeAssets(t, dir, "", "")
	writeBalance(t, dir, validBalance)
	balance, err := LoadBalance(dir)
	if err != nil {
		t.Fatalf("LoadBalance = %v", err)
	}
	if balance.SpecialOrders.HandLimit != 4 || balance.SpecialOrders.DrawOrdersLimit != 2 || balance.SpecialOrders.DeckSize(2) != 22 {
		t.Fatalf("special order limits = %#v", balance.SpecialOrders)
	}
	if balance.SpecialOrders.CalamitySlots[models.SeasonAutumn] != 1 || balance.SpecialOrders.CalamityWeights[models.CardKindPlague] != 1 {
		t.Fatalf("special order weights/slots = %#v", balance.SpecialOrders)
	}
}

func TestLoadSpecialOrdersBalanceRejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name string
		edit func(string) string
		want string
	}{
		{name: "unknown season", edit: func(value string) string {
			return strings.Replace(value, "    autumn: 1\n", "    autumn: 1\n    flood: 1\n", 1)
		}, want: "invalid season"},
		{name: "no bonus cards", edit: func(value string) string {
			return strings.Replace(strings.Replace(value, "bonus_cards_base: 6", "bonus_cards_base: 0", 1), "bonus_cards_per_player: 6", "bonus_cards_per_player: 0", 1)
		}, want: "must not both be zero"},
		{name: "negative calamity base", edit: func(value string) string {
			return strings.Replace(value, "calamity_cards_base: 2", "calamity_cards_base: -1", 1)
		}, want: "calamity_cards_base"},
		{name: "zero slot players per extra", edit: func(value string) string {
			return strings.Replace(value, "calamity_slot_players_per_extra: 4", "calamity_slot_players_per_extra: 0", 1)
		}, want: "calamity_slot_players_per_extra"},
		{name: "negative weight", edit: func(value string) string { return strings.Replace(value, "    plague: 1", "    plague: -1", 1) }, want: "must be >= 0"},
		{name: "zero weights", edit: func(value string) string {
			return strings.Replace(value, "    plague: 1\n    bad_weather: 6\n    famine: 6", "    plague: 0\n    bad_weather: 0\n    famine: 0", 1)
		}, want: "must not all be zero"},
		{name: "invalid revolt bounds", edit: func(value string) string {
			return strings.Replace(value, "revolt_army_min_size: 1\n    revolt_army_max_size: 3", "revolt_army_min_size: 4\n    revolt_army_max_size: 3", 1)
		}, want: "minimum exceeds maximum"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			writeAssets(t, dir, "", "")
			writeBalance(t, dir, test.edit(validBalance))
			if _, err := LoadBalance(dir); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadBalance error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestWeightedCountsUsesLargestRemaindersInCanonicalOrder(t *testing.T) {
	canonical := []models.CardKind{models.CardKindPlague, models.CardKindBadWeather, models.CardKindFamine}
	counts := weightedCounts(9, map[models.CardKind]int{
		models.CardKindPlague:     1,
		models.CardKindBadWeather: 6,
		models.CardKindFamine:     6,
	}, canonical)
	want := map[models.CardKind]int{
		models.CardKindPlague:     1,
		models.CardKindBadWeather: 4,
		models.CardKindFamine:     4,
	}
	for kind, expected := range want {
		if counts[kind] != expected {
			t.Errorf("count[%q] = %d, want %d", kind, counts[kind], expected)
		}
	}
}

// The deck grows with the table: 6 + 6 bonus cards and 2 + 1 calamities per
// player, and one more calamity slot per season for every 4 players.
func TestSpecialOrdersScaleWithPlayers(t *testing.T) {
	balance, err := LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("LoadBalance = %v", err)
	}
	special := balance.SpecialOrders
	for _, test := range []struct{ players, bonus, calamities, slots int }{
		{2, 18, 4, 1}, {3, 24, 5, 1}, {4, 30, 6, 2}, {6, 42, 8, 2}, {8, 54, 10, 3},
	} {
		if got := special.BonusCardCount(test.players); got != test.bonus {
			t.Errorf("BonusCardCount(%d) = %d, want %d", test.players, got, test.bonus)
		}
		if got := special.CalamityCardCount(test.players); got != test.calamities {
			t.Errorf("CalamityCardCount(%d) = %d, want %d", test.players, got, test.calamities)
		}
		if got := special.DeckSize(test.players); got != test.bonus+test.calamities {
			t.Errorf("DeckSize(%d) = %d, want %d", test.players, got, test.bonus+test.calamities)
		}
		if got := special.CalamitySlotsFor(models.SeasonSummer, test.players); got != test.slots {
			t.Errorf("CalamitySlotsFor(summer, %d) = %d, want %d", test.players, got, test.slots)
		}
	}
}

// With 4 bonus cards in every hand, a player who draws 2 cards meets at least
// one calamity once in two, from 2 to 8 players: the criterion the deck sizes
// were chosen for.
func TestSpecialDeckCalamityOddsWithFourCardHands(t *testing.T) {
	balance, err := LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("LoadBalance = %v", err)
	}
	special := balance.SpecialOrders
	for players := 2; players <= 8; players++ {
		bonus := special.BonusCardCount(players) - 4*players
		calamities := special.CalamityCardCount(players)
		if bonus <= 0 {
			t.Fatalf("%d players: no bonus card left in the pile with 4-card hands", players)
		}
		total := bonus + calamities
		noCalamity := float64(bonus*(bonus-1)) / float64(total*(total-1))
		if noCalamity > 0.5 {
			t.Errorf("%d players: P(no calamity in 2 draws) = %.2f, want <= 0.50", players, noCalamity)
		}
		if reserve := special.BonusCardCount(players) - special.HandLimit*players; reserve < 6 {
			t.Errorf("%d players: %d bonus cards left with full hands, want at least 6", players, reserve)
		}
	}
}
