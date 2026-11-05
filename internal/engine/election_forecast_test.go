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

func TestAbbessAddsHerAbbeyVoiceToHerBishopElectionOnly(t *testing.T) {
	state := electionTestState(t)
	addNoble(state, "N5", "ADE", "P2", "CCC")
	abbess := &state.Nobles[len(state.Nobles)-1]
	abbess.Sex = models.SexFemale
	abbess.Dignities = []models.Dignity{models.DignityAbbess}
	abbess.AbbeyRegion = "AAA"

	voices := func() (int, int) {
		open := ForecastWinterElections(state, testBalance(), "P2")
		return open[0].Voices, open[1].Voices
	}
	// P2 holds CCC (1) in R1; the abbey adds 1 there and nothing in R2.
	if r1, r2 := voices(); r1 != 2 || r2 != 0 {
		t.Fatalf("voices = %d/%d, want 2/0", r1, r2)
	}
	open := ForecastWinterElections(state, testBalance(), "P2")
	if last := open[0].VoiceSources[len(open[0].VoiceSources)-1]; last.Kind != "abbey" || last.Votes != 1 {
		t.Fatalf("last voice source = %+v, want the abbey", last)
	}

	abbess.Status = models.NobleStatusDungeon
	if r1, _ := voices(); r1 != 1 {
		t.Fatalf("imprisoned abbess voices = %d, want 1", r1)
	}
	abbess.Status = models.NobleStatusFree
	state.Excommunications = append(state.Excommunications, models.Excommunication{Noble: "N5", Reason: models.ExcommunicationPapal})
	if r1, _ := voices(); r1 != 1 {
		t.Fatalf("excommunicated abbess voices = %d, want 1", r1)
	}
}
