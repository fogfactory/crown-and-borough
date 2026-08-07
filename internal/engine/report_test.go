package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestTurnReportContainsResolutionSectionsAndRoundTrips(t *testing.T) {
	assets := loadGameTestAssets(t)
	balance := testBalance()
	game, err := CreateGame("report-test", []PlayerInit{{Name: "One"}, {Name: "Two"}}, balance, assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	noble := game.Nobles[0]
	start := territoryByID(game.Territories, noble.LocationID)
	if len(start.Adjacencies) == 0 {
		t.Fatalf("starting territory %s has no adjacency", start.ID)
	}
	target := territoryByID(game.Territories, start.Adjacencies[0])
	report, err := ResolveTurn(game, balance, OrdersInput{
		Chains: []ChainSubmission{{
			Player: noble.OwnerID,
			Noble:  models.NobleCode(noble.Code),
			Text:   noble.Code + "\n" + start.Code + " A " + target.Code,
		}},
	})
	if err != nil {
		t.Fatalf("ResolveTurn: %v", err)
	}
	if report.Header != (ReportHeader{Year: 1, Season: models.SeasonSpring, Turn: 1}) {
		t.Errorf("report header = %#v", report.Header)
	}
	if len(report.Players) != 2 {
		t.Errorf("player reports = %d, want 2", len(report.Players))
	}
	if len(report.Supply) == 0 {
		t.Error("supply section is empty")
	}
	if len(report.Orders) == 0 {
		t.Error("orders section is empty")
	}
	if len(report.Moves) == 0 {
		t.Error("moves section is empty")
	}

	serializable := report
	serializable.State = nil
	encoded, err := json.Marshal(serializable)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var decoded TurnReport
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if !reflect.DeepEqual(decoded, serializable) {
		t.Errorf("report round trip differs\nencoded=%s\ndecoded=%#v\nwant=%#v", encoded, decoded, serializable)
	}

	current := report.State
	for current.Season != models.SeasonWinter {
		next, resolveErr := ResolveTurn(current, balance, OrdersInput{})
		if resolveErr != nil {
			t.Fatalf("advance to winter: %v", resolveErr)
		}
		current = next.State
	}
	startCode := territoryByID(current.Territories, current.Nobles[0].LocationID).Code
	winterReport, err := ResolveTurn(current, balance, OrdersInput{
		Winter: []WinterSubmission{{Player: "P1", Lines: "R T " + startCode}},
	})
	if err != nil {
		t.Fatalf("winter ResolveTurn: %v", err)
	}
	if winterReport.Winter == nil {
		t.Fatal("winter report is nil")
	}
	if len(winterReport.Winter.Stocks) == 0 {
		t.Error("winter stock section is empty")
	}
	if len(winterReport.Winter.Investments) == 0 {
		t.Error("winter investment section is empty")
	}
}
