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

func TestBuildTurnReportProjectsCalamityEffectDetails(t *testing.T) {
	before := winterDeckState()
	after := cloneGameState(before)
	events := []Event{
		{Type: EventTypeCalamityApplied, CardKind: models.CardKindPlague, RegionSeed: "ROS", Season: models.SeasonSpring, ArmyID: "A1", OwnerID: "P1", SizeBefore: 5, SizeAfter: 2},
		{Type: EventTypeBadWeatherBlocked, RegionSeed: "ROS", Season: models.SeasonSpring, ArmyID: "A2", OwnerID: "P2", TerritoryID: "ROS", TargetID: "BOI"},
		{Type: EventTypeFamineLoss, CardKind: models.CardKindFamine, RegionSeed: "ROS", Season: models.SeasonSpring, Production: 3, RationsLost: 2},
		{Type: EventTypePlagueSurvived, RegionSeed: "ROS", Season: models.SeasonSpring, NobleID: "N9", NobleCode: "ROB", TerritoryID: "ROS"},
	}
	report := BuildTurnReport(before, after, events, nil)
	byKind := map[EventType]SeasonEffectReport{}
	for _, effect := range report.SeasonEffects {
		byKind[effect.Kind] = effect
	}
	plague, exists := byKind[EventTypeCalamityApplied]
	if !exists || plague.Army != "A1" || plague.Owner != "P1" || plague.SizeBefore != 5 || plague.SizeAfter != 2 {
		t.Fatalf("plague effect = %#v, want army A1 reduced from 5 to 2", plague)
	}
	blocked, exists := byKind[EventTypeBadWeatherBlocked]
	if !exists || blocked.Owner != "P2" || blocked.Territory != "ROS" || blocked.Target != "BOI" {
		t.Fatalf("blocked effect = %#v, want P2 blocked from ROS to BOI", blocked)
	}
	famine, exists := byKind[EventTypeFamineLoss]
	if !exists || famine.ProductionLost != 3 || famine.RationsLost != 2 {
		t.Fatalf("famine effect = %#v, want 3 R and 2 rations lost", famine)
	}
	survived, exists := byKind[EventTypePlagueSurvived]
	if !exists || survived.Noble != models.NobleCode("ROB") || survived.Territory != "ROS" {
		t.Fatalf("survivor effect = %#v, want noble ROB alive at ROS", survived)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if strings.Contains(string(encoded), "N9") {
		t.Fatalf("public report exposes internal noble ID: %s", encoded)
	}
}
