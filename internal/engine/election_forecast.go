package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// OpenElection is one election the winter announces, read for a single viewer.
// The registry is frozen when the winter begins, so the list of elections is
// exact while the orders are being entered (specs/hiver.md, principle 4).
// Voices and Candidates are read on the winter snapshot: earlier winter stages
// (marriage, capture, excommunication) can still change them before the count.
//
// Required is the number of voices an absolute majority needs (0 for a
// relative majority). Voices is the viewer's own weight in the election.
// Candidates are the viewer's nobles accepted as candidates on the winter
// snapshot, so a title won this winter never counts.
type OpenElection struct {
	Kind       models.ElectionKind `json:"kind"`
	Region     models.RegionID     `json:"region,omitempty"`
	Seat       models.TerritoryID  `json:"seat,omitempty"`
	Required   int                 `json:"required,omitempty"`
	Voices     int                 `json:"voices"`
	Candidates []models.NobleCode  `json:"candidates"`
}

// ForecastWinterElections returns the elections open when the winter begins,
// bishoprics by region identifier then the conclave, with the viewer's own
// voices and eligible nobles. It returns nil outside winter. It reads no
// order and no ballot: nothing private to another player leaks.
func ForecastWinterElections(state *models.GameState, balance assetgen.Balance, viewer models.PlayerID) []OpenElection {
	if state == nil || state.Season != models.SeasonWinter {
		return nil
	}
	ctx := newResolutionContext(cloneGameState(state), balance)
	ctx.openWinterElections()
	open := make([]OpenElection, 0, len(ctx.elections))
	for _, election := range ctx.elections {
		announced := OpenElection{
			Kind:       election.key.kind,
			Region:     election.key.region,
			Seat:       election.seat,
			Voices:     ctx.electionVoices(election)[viewer],
			Candidates: []models.NobleCode{},
		}
		if election.absolute {
			announced.Required = election.denominator/2 + 1
		}
		for i := range ctx.state.Nobles {
			noble := &ctx.state.Nobles[i]
			if noble.OwnerID == viewer && election.eligible(noble) == "" {
				announced.Candidates = append(announced.Candidates, models.NobleCode(noble.Code))
			}
		}
		open = append(open, announced)
	}
	return open
}
