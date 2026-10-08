package engine

import (
	"slices"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestForecastWinterElectionsAnnouncesRegistryForViewer(t *testing.T) {
	state := electionTestState(t)
	open := ForecastWinterElections(state, testBalance(), "P1")
	if len(open) != 2 {
		t.Fatalf("open elections = %d, want both bishoprics", len(open))
	}
	ros := open[0]
	if ros.Kind != models.ElectionBishop || ros.Seat != "AAA" || ros.Required != 0 {
		t.Fatalf("first election = %+v, want the R1 bishopric", ros)
	}
	// P1 holds the seat (2) and BBB (1).
	if ros.Voices != 3 {
		t.Fatalf("P1 voices = %d, want 3", ros.Voices)
	}
	codes := func(nobles []ElectionNoble) []models.NobleCode {
		var out []models.NobleCode
		for _, noble := range nobles {
			out = append(out, noble.Code)
		}
		return out
	}
	if !slices.Equal(codes(ros.Candidates), []models.NobleCode{"HUG", "OTO"}) || ros.Candidates[0].Name == "" {
		t.Fatalf("P1 candidates = %v, want HUG and OTO with names", ros.Candidates)
	}
	if len(ros.VoiceSources) != 2 || ros.VoiceSources[0].Kind != "seat" || ros.VoiceSources[1].Kind != "territory" {
		t.Fatalf("P1 voice sources = %+v, want the seat then BBB", ros.VoiceSources)
	}
	if other := ForecastWinterElections(state, testBalance(), "P2"); other[0].Voices != 1 || !slices.Equal(codes(other[0].Candidates), []models.NobleCode{"LEO"}) {
		t.Fatalf("P2 view = %+v, want its own voices and nobles only", other[0])
	}
}

func TestForecastWinterElectionsIsWinterOnlyAndLeavesStateUntouched(t *testing.T) {
	state := electionTestState(t)
	state.Season = models.SeasonSpring
	if open := ForecastWinterElections(state, testBalance(), "P1"); open != nil {
		t.Fatalf("outside winter = %v, want nil", open)
	}
	state.Season = models.SeasonWinter
	ForecastWinterElections(state, testBalance(), "P1")
	if len(state.Bishops) != 0 {
		t.Fatalf("forecast changed the state: %v", state.Bishops)
	}
}
