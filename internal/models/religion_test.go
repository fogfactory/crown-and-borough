package models_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// religiousState returns a valid state with three nobles, two bishoprics and
// no title yet.
func religiousState() *models.GameState {
	g := validState()
	g.Nobles = []models.Noble{
		{ID: "N1", Sex: models.SexMale, Code: "HUG", Name: "Hugues", OwnerID: "P1", LocationID: "ROS", Status: models.NobleStatusFree},
		{ID: "N2", Sex: models.SexMale, Code: "ADE", Name: "Adhémar", OwnerID: "P2", LocationID: "BCL", Status: models.NobleStatusFree},
		{ID: "N3", Sex: models.SexMale, Code: "GUI", Name: "Guy", OwnerID: "P1", LocationID: "BRU", Status: models.NobleStatusFree},
	}
	g.Regions = []models.Region{
		{ID: "ROS", Name: "Rosemont", Seed: "ROS", Territories: []models.TerritoryID{"BCL", "ROS"}},
		{ID: "BRU", Name: "Bruyères", Seed: "BRU", Territories: []models.TerritoryID{"BRU", "FOU"}},
	}
	return g
}

func TestValidateReligion(t *testing.T) {
	pope := models.NobleID("N1")
	tests := []struct {
		name   string
		mutate func(*models.GameState)
		want   string // empty = valid
	}{
		{"no title", func(*models.GameState) {}, ""},
		{"full hierarchy", func(g *models.GameState) {
			g.Bishops = []models.Bishop{{Region: "ROS", Noble: "N1"}, {Region: "BRU", Noble: "N2"}}
			g.Cardinals = []models.NobleID{"N1"}
			g.Pope = &pope
		}, ""},
		{"two bishops in one region", func(g *models.GameState) {
			g.Bishops = []models.Bishop{{Region: "ROS", Noble: "N1"}, {Region: "ROS", Noble: "N2"}}
		}, "more than one bishop"},
		{"bishop of two regions", func(g *models.GameState) {
			g.Bishops = []models.Bishop{{Region: "ROS", Noble: "N1"}, {Region: "BRU", Noble: "N1"}}
		}, "more than one bishopric"},
		{"unknown region", func(g *models.GameState) {
			g.Bishops = []models.Bishop{{Region: "XXX", Noble: "N1"}}
		}, "unknown bishopric"},
		{"unknown bishop", func(g *models.GameState) {
			g.Bishops = []models.Bishop{{Region: "ROS", Noble: "N9"}}
		}, "unknown bishop"},
		{"cardinal without bishopric", func(g *models.GameState) {
			g.Cardinals = []models.NobleID{"N1"}
		}, "must be a bishop"},
		{"duplicate cardinal", func(g *models.GameState) {
			g.Bishops = []models.Bishop{{Region: "ROS", Noble: "N1"}}
			g.Cardinals = []models.NobleID{"N1", "N1"}
		}, "duplicate"},
		{"pope without title", func(g *models.GameState) {
			g.Pope = &pope
		}, "bishop or a cardinal"},
		{"pope bishop only", func(g *models.GameState) {
			g.Bishops = []models.Bishop{{Region: "ROS", Noble: "N1"}}
			g.Pope = &pope
		}, ""},
		{"excommunicated titleholder", func(g *models.GameState) {
			g.Bishops = []models.Bishop{{Region: "ROS", Noble: "N2"}}
			g.Excommunications = []models.Excommunication{{Noble: "N2", Reason: models.ExcommunicationPapal, By: "N1", Turn: 1}}
		}, "still holds a religious title"},
		{"papal excommunication", func(g *models.GameState) {
			g.Excommunications = []models.Excommunication{{Noble: "N2", Reason: models.ExcommunicationPapal, By: "N1", Turn: 1}}
		}, ""},
		{"papal without pope", func(g *models.GameState) {
			g.Excommunications = []models.Excommunication{{Noble: "N2", Reason: models.ExcommunicationPapal, Turn: 1}}
		}, "unknown pope"},
		{"ex officio with pope", func(g *models.GameState) {
			g.Excommunications = []models.Excommunication{{Noble: "N2", Reason: models.ExcommunicationExOfficio, By: "N1", Turn: 1}}
		}, "no pope"},
		{"invalid reason", func(g *models.GameState) {
			g.Excommunications = []models.Excommunication{{Noble: "N2", Reason: "x", Turn: 1}}
		}, "invalid reason"},
		{"excommunicated twice", func(g *models.GameState) {
			g.Excommunications = []models.Excommunication{
				{Noble: "N2", Reason: models.ExcommunicationExOfficio, Turn: 1},
				{Noble: "N2", Reason: models.ExcommunicationExOfficio, Turn: 1},
			}
		}, "twice"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := religiousState()
			tt.mutate(g)
			err := g.Validate()
			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("Validate = %v, want nil", err)
			case tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)):
				t.Fatalf("Validate = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestVotingReligiousTitleIsTheHighestTitleOfAFreeNoble(t *testing.T) {
	g := religiousState()
	pope := models.NobleID("N1")
	g.Bishops = []models.Bishop{{Region: "ROS", Noble: "N1"}, {Region: "BRU", Noble: "N3"}}
	g.Cardinals = []models.NobleID{"N1"}
	g.Pope = &pope
	g.Excommunications = []models.Excommunication{{Noble: "N2", Reason: models.ExcommunicationPapal, By: "N1", Turn: 1}}
	g.Nobles[2].Status = models.NobleStatusDungeon
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}

	if got := g.ReligiousTitleOf("N1"); got != models.ReligiousTitlePope {
		t.Errorf("N1 title = %q, want pope", got)
	}
	if got := g.VotingReligiousTitle("N1"); got != models.ReligiousTitlePope {
		t.Errorf("N1 votes with %q, want pope", got)
	}
	if got := g.ReligiousTitleOf("N3"); got != models.ReligiousTitleBishop {
		t.Errorf("jailed N3 keeps its title, got %q", got)
	}
	if got := g.VotingReligiousTitle("N3"); got != models.ReligiousTitleNone {
		t.Errorf("jailed N3 votes with %q, want none", got)
	}
	g.Nobles[2].Status = models.NobleStatusHostage
	if got := g.VotingReligiousTitle("N3"); got != models.ReligiousTitleBishop {
		t.Errorf("hostage N3 votes with %q, want bishop", got)
	}
	if got := g.VotingReligiousTitle("N2"); got != models.ReligiousTitleNone {
		t.Errorf("excommunicated N2 votes with %q, want none", got)
	}
}

func TestDropReligiousTitlesOfMissingNobles(t *testing.T) {
	g := religiousState()
	pope := models.NobleID("N1")
	g.Bishops = []models.Bishop{{Region: "ROS", Noble: "N1"}, {Region: "BRU", Noble: "N3"}}
	g.Cardinals = []models.NobleID{"N1", "N3"}
	g.Pope = &pope
	g.Excommunications = []models.Excommunication{
		{Noble: "N2", Reason: models.ExcommunicationPapal, By: "N1", Turn: 1},
		{Noble: "N3", Reason: models.ExcommunicationExOfficio, Turn: 1},
	}
	// Remove N3's titles to keep the state valid, then kill the pope.
	g.Bishops, g.Cardinals = g.Bishops[:1], g.Cardinals[:1]
	g.Excommunications = g.Excommunications[:1]
	g.Nobles = g.Nobles[1:]
	g.RemovedNobles = []models.RemovedNoble{{ID: "N1", Code: "HUG", Sex: models.SexMale, OwnerID: "P1", Cause: models.DeathCauseNatural, Turn: 1}}
	g.DropReligiousTitlesOfMissingNobles()
	if len(g.Bishops) != 0 || len(g.Cardinals) != 0 || g.Pope != nil {
		t.Errorf("titles of the dead pope remain: %v %v %v", g.Bishops, g.Cardinals, g.Pope)
	}
	if len(g.Excommunications) != 0 {
		t.Errorf("papal excommunication outlived the pope: %v", g.Excommunications)
	}
	if err := g.Validate(); err != nil {
		t.Errorf("state invalid after cleanup: %v", err)
	}
}

func TestReligionJSONRoundTrip(t *testing.T) {
	g := religiousState()
	pope := models.NobleID("N1")
	g.Bishops = []models.Bishop{{Region: "ROS", Noble: "N1"}}
	g.Cardinals = []models.NobleID{"N1"}
	g.Pope = &pope
	g.Excommunications = []models.Excommunication{{Noble: "N2", Reason: models.ExcommunicationExOfficio, Turn: 1}}
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	var back models.GameState
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if err := back.Validate(); err != nil {
		t.Fatal(err)
	}
	if !back.IsPope("N1") || !back.IsCardinal("N1") || back.Regions[0].Name != "Rosemont" || len(back.Excommunications) != 1 {
		t.Errorf("round trip lost religion: %+v", back)
	}
	// A game without titles keeps the fields out of the document.
	plain, _ := json.Marshal(religiousState())
	for _, key := range []string{`"bishops"`, `"cardinals"`, `"pope"`, `"excommunications"`} {
		if strings.Contains(string(plain), key) {
			t.Errorf("%s serialized for a state without titles", key)
		}
	}
}
