package engine

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestBuildTurnReportProjectsCardAndSeasonEffects(t *testing.T) {
	before := winterDeckState()
	after := cloneGameState(before)
	events := []Event{
		{Type: EventTypeDeckDraw, Phase: winterPhase, CardID: "C1", CardKind: models.CardKindFairWeather, OwnerID: "P1"},
		{Type: EventTypeCalamityScheduled, Phase: winterPhase, CardID: "C2", CardKind: models.CardKindPlague, RegionSeed: "ROS", Season: models.SeasonSpring, Year: 2},
		{Type: EventTypeBonusEffect, Phase: 0, CardKind: models.CardKindFairWeather, RegionSeed: "ROS", Season: models.SeasonSpring},
		{Type: EventTypeNeutralArmy, Phase: 0, ArmyID: "A3", OwnerID: models.NeutralPlayerID, TerritoryID: "ROS", Troops: 2},
	}
	report := BuildTurnReport(before, after, events, nil)
	if report.Winter == nil || len(report.Winter.Cards) != 2 {
		t.Fatalf("winter cards = %#v, want two card reports", report.Winter)
	}
	if len(report.SeasonEffects) != 2 {
		t.Fatalf("season effects = %#v, want two effects", report.SeasonEffects)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if strings.Contains(string(encoded), "C1") || strings.Contains(string(encoded), "C2") {
		t.Fatalf("public report exposes internal card ID: %s", encoded)
	}
}
