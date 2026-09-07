package assetgen

import (
	"strings"
	"testing"
)

func TestLoadRulesRendersBalanceValuesInBothLanguages(t *testing.T) {
	balance, err := LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("LoadBalance = %v", err)
	}
	rules, err := LoadRules("../../../assets", balance)
	if err != nil {
		t.Fatalf("LoadRules = %v", err)
	}
	for _, language := range []string{"fr", "en"} {
		document, ok := rules.Document(language)
		if !ok {
			t.Fatalf("rules[%s] missing", language)
		}
		text := string(document)
		for _, value := range []string{"{{special_orders.deck_size}}", "{{special_orders.hand_limit}}", "{{special_orders.card.plague}}"} {
			if strings.Contains(text, value) {
				t.Errorf("rules[%s] still contains placeholder %q", language, value)
			}
		}
		if !strings.Contains(text, "30") || !strings.Contains(text, "4") {
			t.Errorf("rules[%s] does not contain rendered balance values", language)
		}
	}
}
