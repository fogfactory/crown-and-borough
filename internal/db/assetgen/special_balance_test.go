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
	if balance.SpecialOrders.HandLimit != 4 || balance.SpecialOrders.DrawOrdersLimit != 2 || balance.SpecialOrders.DeckSize != 30 {
		t.Fatalf("special order limits = %#v", balance.SpecialOrders)
	}
	if balance.SpecialOrders.CalamitySlots[models.SeasonWinter] != 1 || balance.SpecialOrders.CalamityWeights[models.CardKindPlague] != 1 {
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
			return strings.Replace(value, "    winter: 1\n", "    winter: 1\n    autumn: 1\n", 1)
		}, want: "invalid season"},
		{name: "non integral calamity count", edit: func(value string) string {
			return strings.Replace(value, "calamity_percentage: 30", "calamity_percentage: 33", 1)
		}, want: "integer card count"},
		{name: "negative weight", edit: func(value string) string { return strings.Replace(value, "    plague: 1", "    plague: -1", 1) }, want: "must be >= 0"},
		{name: "zero weights", edit: func(value string) string {
			return strings.Replace(value, "    plague: 1\n    bad_weather: 6\n    revolt: 4\n    famine: 4", "    plague: 0\n    bad_weather: 0\n    revolt: 0\n    famine: 0", 1)
		}, want: "must not all be zero"},
		{name: "invalid revolt bounds", edit: func(value string) string {
			return strings.Replace(value, "revolt_army_min_size: 2\n    revolt_army_max_size: 3", "revolt_army_min_size: 4\n    revolt_army_max_size: 3", 1)
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
	canonical := []models.CardKind{models.CardKindPlague, models.CardKindBadWeather, models.CardKindRevolt, models.CardKindFamine}
	counts := weightedCounts(9, map[models.CardKind]int{
		models.CardKindPlague:     1,
		models.CardKindBadWeather: 6,
		models.CardKindRevolt:     4,
		models.CardKindFamine:     4,
	}, canonical)
	want := map[models.CardKind]int{
		models.CardKindPlague:     1,
		models.CardKindBadWeather: 4,
		models.CardKindRevolt:     2,
		models.CardKindFamine:     2,
	}
	for kind, expected := range want {
		if counts[kind] != expected {
			t.Errorf("count[%q] = %d, want %d", kind, counts[kind], expected)
		}
	}
}
