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
	after.Auguries[2] = models.YearAugury{Year: 2, Calamities: []models.Calamity{
		{CardID: "C2", Kind: models.CardKindPlague, Year: 2, Season: models.SeasonSpring, RegionSeed: "ROS"},
	}}
	events := []Event{
		{Type: EventTypeDeckDraw, Phase: winterPhase, CardID: "C1", CardKind: models.CardKindFairWeather, OwnerID: "P1"},
		{Type: EventTypeCalamityScheduled, Phase: winterPhase, CardID: "C2", CardKind: models.CardKindPlague, RegionSeed: "ROS", Season: models.SeasonSpring, Year: 2},
		{Type: EventTypeBonusEffect, Phase: 0, CardKind: models.CardKindFairWeather, RegionSeed: "ROS", Season: models.SeasonSpring},
		{Type: EventTypeNeutralArmy, Phase: 0, ArmyID: "A3", OwnerID: models.NeutralPlayerID, TerritoryID: "ROS", Troops: 2},
	}
	report := BuildTurnReport(before, after, events, nil)
	if report.Winter == nil || len(report.Winter.Cards) != 1 {
		t.Fatalf("winter cards = %#v, want one card report for the scheduled calamity without player draws", report.Winter)
	}
	scheduled := report.Winter.Cards[0]
	if scheduled.EventType != EventTypeCalamityScheduled || scheduled.Kind != models.CardKindPlague || scheduled.Season != models.SeasonSpring || scheduled.Region != "ROS" {
		t.Fatalf("scheduled card = %#v, want plague calamity scheduled for spring in ROS", scheduled)
	}
	if len(report.Announcements) != 1 {
		t.Fatalf("announcements = %#v, want one pending announcement", report.Announcements)
	}
	if got := report.Announcements[0]; got.Kind != models.CardKindPlague || got.Season != models.SeasonSpring || got.Region != "ROS" || got.Year != 2 {
		t.Fatalf("announcement = %#v, want plague scheduled for spring in ROS at year 2", got)
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
