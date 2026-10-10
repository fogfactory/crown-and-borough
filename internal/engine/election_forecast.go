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
// snapshot, so a title won this winter never counts. VoiceSources explains
// the viewer's voices. Eligible lists the nobles of every player who may stand
// (public data), to help a player choose whom to vote for: whether they do
// stand is on the other players' private sheets.
type OpenElection struct {
	Kind         models.ElectionKind `json:"kind"`
	Region       models.RegionID     `json:"region,omitempty"`
	Seat         models.TerritoryID  `json:"seat,omitempty"`
	Required     int                 `json:"required,omitempty"`
	Voices       int                 `json:"voices"`
	VoiceSources []VoiceSource       `json:"voiceSources"`
	Candidates   []ElectionNoble     `json:"candidates"`
	Eligible     []ElectionNoble     `json:"eligible"`
}

// ElectionNoble is a potential candidate: code for the order, display name for
// the player.
type ElectionNoble struct {
	Code  models.NobleCode `json:"code"`
	Name  string           `json:"name"`
	Owner models.PlayerID  `json:"owner"`
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
			Kind:         election.key.kind,
			Region:       election.key.region,
			Seat:         election.seat,
			VoiceSources: []VoiceSource{},
			Candidates:   []ElectionNoble{},
			Eligible:     []ElectionNoble{},
		}
		for _, source := range ctx.electionVoiceSources(election)[viewer] {
			announced.VoiceSources = append(announced.VoiceSources, source)
			announced.Voices += source.Votes
		}
		if election.absolute {
			announced.Required = election.denominator/2 + 1
		}
		for i := range ctx.state.Nobles {
			noble := &ctx.state.Nobles[i]
			if election.eligible(noble) != "" {
				continue
			}
			entry := ElectionNoble{Code: models.NobleCode(noble.Code), Name: ctx.state.NobleDisplayName(*noble), Owner: noble.OwnerID}
			announced.Eligible = append(announced.Eligible, entry)
			if noble.OwnerID == viewer {
				announced.Candidates = append(announced.Candidates, entry)
			}
		}
		open = append(open, announced)
	}
	return open
}
