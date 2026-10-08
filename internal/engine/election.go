package engine

import (
	"slices"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Generic election engine (specs/religieux.md, specs/hiver.md § Règles propres
// aux élections). An election is a seat, a rule to count it and an eligibility
// test; the same machinery serves the bishoprics and the conclave, and will
// serve the royal election.
//
// The registry of open elections is frozen when the winter begins. Candidacies
// and votes only record what the players asked for during the order stage; the
// election stage validates them against the world as the earlier stages left
// it, counts every election on one snapshot, and the titles won are conferred
// at the investiture, never before.

// Election result codes carried by an EventTypeElectionResult event.
const (
	ElectionElected      = "elected"
	ElectionTie          = "tie"
	ElectionNoCandidate  = "no_candidate"
	ElectionNoMajority   = "no_majority"
	electionNoVoiceError = "no_voice"
)

// ElectionCandidate is one candidate of an election with the voices the
// ballots gave it. Individual ballots are never reported, only these totals.
type ElectionCandidate struct {
	Noble     models.NobleID   `json:"noble"`
	NobleCode models.NobleCode `json:"nobleCode"`
	NobleName string           `json:"nobleName"`
	Owner     models.PlayerID  `json:"owner"`
	Votes     int              `json:"votes"`
}

// ElectionResult is the public outcome of one election. Required is the number
// of voices an absolute majority needs (0 for a relative majority); Cast is
// the total of voices given to a candidate.
type ElectionResult struct {
	Kind       models.ElectionKind `json:"kind"`
	Region     models.RegionID     `json:"region,omitempty"`
	Seat       models.TerritoryID  `json:"seat,omitempty"`
	Result     string              `json:"result"`
	Winner     models.NobleID      `json:"winner,omitempty"`
	Required   int                 `json:"required,omitempty"`
	Cast       int                 `json:"cast"`
	Candidates []ElectionCandidate `json:"candidates"`
}

type electionKey struct {
	kind   models.ElectionKind
	region models.RegionID
}

// election is one open election of the winter.
type election struct {
	key electionKey
	// seat is the seed village the orders name (bishop elections only).
	seat models.TerritoryID
	// absolute asks for strictly more than half of denominator voices; a
	// relative majority only asks for the highest total.
	absolute    bool
	denominator int
	// eligible returns "" when the noble may run, the rejection reason
	// otherwise.
	eligible func(*models.Noble) string
	// voices is each player's weight, computed on the winter snapshot.
	voices map[models.PlayerID]int

	candidates []*electionEntry
	filed      map[models.PlayerID]bool
	voted      map[models.PlayerID]bool
}

type electionEntry struct {
	noble *models.Noble
	votes int
}

// electionOrder is a candidacy or a vote recorded during the order stage.
type electionOrder struct {
	playerID models.PlayerID
	order    models.WinterOrder
}

// pendingTitle is a title won during the winter, conferred at the investiture.
type pendingTitle struct {
	noble  models.NobleID
	kind   models.ElectionKind
	region models.RegionID
}

// openWinterElections computes the registry of the elections open when the
// winter begins: a bishopric without bishop whose every territory is
// controlled or occupied, and the conclave when the throne is vacant and at
// least two cardinals hold their title. Bishoprics come by region identifier,
// the conclave last.
func (ctx *resolutionContext) openWinterElections() {
	ctx.elections = nil
	regions := slices.Clone(ctx.state.Regions)
	sort.Slice(regions, func(i, j int) bool { return regions[i].ID < regions[j].ID })
	for _, region := range regions {
		if _, hasBishop := ctx.state.BishopOf(region.ID); hasBishop {
			continue
		}
		if !ctx.bishopricFullyHeld(region) {
			continue
		}
		ctx.elections = append(ctx.elections, &election{
			key:      electionKey{kind: models.ElectionBishop, region: region.ID},
			seat:     region.Seed,
			eligible: ctx.bishopCandidateRejection,
			filed:    map[models.PlayerID]bool{},
			voted:    map[models.PlayerID]bool{},
		})
	}
	if ctx.state.Pope == nil && len(ctx.state.Cardinals) >= 2 {
		ctx.elections = append(ctx.elections, &election{
			key:         electionKey{kind: models.ElectionPope},
			absolute:    true,
			denominator: len(ctx.state.Cardinals),
			eligible:    ctx.popeCandidateRejection,
			filed:       map[models.PlayerID]bool{},
			voted:       map[models.PlayerID]bool{},
		})
	}
	for _, open := range ctx.elections {
		ctx.events = append(ctx.events, Event{
			Type:        EventTypeElectionOpened,
			Phase:       winterPhase,
			TerritoryID: open.seat,
			Election:    &ElectionResult{Kind: open.key.kind, Region: open.key.region, Seat: open.seat, Candidates: []ElectionCandidate{}},
		})
	}
}

// bishopricFullyHeld reports whether every territory of the bishopric is
// controlled or occupied by a player when the winter begins.
func (ctx *resolutionContext) bishopricFullyHeld(region models.Region) bool {
	if len(region.Territories) == 0 {
		return false
	}
	for _, territoryID := range region.Territories {
		if _, held := ctx.controllerAtStart(territoryID); !held {
			return false
		}
	}
	return true
}

// commonCandidateRejection holds the eligibility conditions every election
// shares: a free, unmarried, unexcommunicated man.
func (ctx *resolutionContext) commonCandidateRejection(noble *models.Noble) string {
	switch {
	case !noble.Sex.CanHoldReligiousOrRoyalTitle():
		return "candidate_not_eligible"
	case noble.Status != models.NobleStatusFree:
		return "candidate_not_eligible"
	}
	if _, married := ctx.state.MarriageOf(noble.ID); married {
		return "candidate_not_eligible"
	}
	if _, excommunicated := ctx.state.ExcommunicationOf(noble.ID); excommunicated {
		return "candidate_not_eligible"
	}
	return ""
}

func (ctx *resolutionContext) bishopCandidateRejection(noble *models.Noble) string {
	if reason := ctx.commonCandidateRejection(noble); reason != "" {
		return reason
	}
	if ctx.state.IsBishop(noble.ID) {
		return "candidate_not_eligible"
	}
	return ""
}

func (ctx *resolutionContext) popeCandidateRejection(noble *models.Noble) string {
	if reason := ctx.commonCandidateRejection(noble); reason != "" {
		return reason
	}
	if !ctx.state.IsBishop(noble.ID) && !ctx.state.IsCardinal(noble.ID) {
		return "candidate_not_eligible"
	}
	return ""
}

// electionVoices computes the weight of every player in the election, on the
// state the election stage opens with.
func (ctx *resolutionContext) electionVoices(open *election) map[models.PlayerID]int {
	voices := map[models.PlayerID]int{}
	if open.key.kind == models.ElectionPope {
		// One voice per cardinal whose title is active; a cardinal in a
		// dungeon keeps its seat in the denominator but votes nothing.
		for _, noble := range ctx.state.Nobles {
			if ctx.state.IsCardinal(noble.ID) && ctx.state.VotingReligiousTitle(noble.ID) != models.ReligiousTitleNone {
				voices[noble.OwnerID]++
			}
		}
		return voices
	}
	weights := ctx.balance.Religion.Votes
	for _, region := range ctx.state.Regions {
		if region.ID != open.key.region {
			continue
		}
		for _, territoryID := range region.Territories {
			controller, held := ctx.controllerAtStart(territoryID)
			if !held {
				continue
			}
			if territoryID == region.Seed {
				voices[controller] += weights.Seat
			} else {
				voices[controller] += weights.Territory
			}
		}
	}
	for _, player := range ctx.state.Players {
		if votes := TitleVotes(ctx.state, player.ID, ctx.balance); votes > 0 {
			voices[player.ID] += votes
		}
	}
	return voices
}

// matchElection finds the open election an order names, or nil.
func (ctx *resolutionContext) matchElection(order models.WinterOrder) *election {
	for _, open := range ctx.elections {
		if open.key.kind != order.Election {
			continue
		}
		if open.key.kind == models.ElectionBishop && open.seat != order.TerritoryID {
			continue
		}
		return open
	}
	return nil
}

func (ctx *resolutionContext) candidateEntry(open *election, noble *models.Noble) *electionEntry {
	for _, entry := range open.candidates {
		if entry.noble.ID == noble.ID {
			return entry
		}
	}
	return nil
}

// resolveWinterElections validates the recorded candidacies, then the votes,
// counts every election on one snapshot and queues the titles won for the
// investiture. Candidacies are validated in resolution order (players by
// identifier, then sheet order): a noble can run in one election only, the
// first valid candidacy keeps it.
func (ctx *resolutionContext) resolveWinterElections() {
	orders := ctx.electionOrders
	ctx.electionOrders = nil
	for _, open := range ctx.elections {
		open.voices = ctx.electionVoices(open)
	}
	running := map[models.NobleID]bool{}
	for _, recorded := range orders {
		if recorded.order.Type != models.WinterOrderTypeCandidacy {
			continue
		}
		ctx.fileCandidacy(recorded, running)
	}
	for _, recorded := range orders {
		if recorded.order.Type != models.WinterOrderTypeVote {
			continue
		}
		ctx.castVote(recorded)
	}
	for _, open := range ctx.elections {
		ctx.concludeElection(open)
	}
}

func (ctx *resolutionContext) fileCandidacy(recorded electionOrder, running map[models.NobleID]bool) {
	order := recorded.order
	open := ctx.matchElection(order)
	if open == nil {
		ctx.rejectWinterOrder(recorded.playerID, order, "election_not_open")
		return
	}
	nobleID, exists := ctx.noblesByCode[order.NobleCode]
	noble := ctx.noblesByID[nobleID]
	if !exists || noble == nil {
		ctx.rejectWinterOrder(recorded.playerID, order, "unknown_noble")
		return
	}
	if noble.OwnerID != recorded.playerID {
		ctx.rejectWinterOrder(recorded.playerID, order, "noble_not_owned")
		return
	}
	if reason := open.eligible(noble); reason != "" {
		ctx.rejectWinterOrder(recorded.playerID, order, reason)
		return
	}
	if open.filed[recorded.playerID] {
		ctx.rejectWinterOrder(recorded.playerID, order, "candidacy_already_filed")
		return
	}
	if running[noble.ID] {
		ctx.rejectWinterOrder(recorded.playerID, order, "candidate_already_running")
		return
	}
	open.filed[recorded.playerID] = true
	running[noble.ID] = true
	open.candidates = append(open.candidates, &electionEntry{noble: noble})
}

func (ctx *resolutionContext) castVote(recorded electionOrder) {
	order := recorded.order
	open := ctx.matchElection(order)
	if open == nil {
		ctx.rejectWinterOrder(recorded.playerID, order, "election_not_open")
		return
	}
	nobleID, exists := ctx.noblesByCode[order.NobleCode]
	noble := ctx.noblesByID[nobleID]
	var entry *electionEntry
	if exists && noble != nil {
		entry = ctx.candidateEntry(open, noble)
	}
	if entry == nil {
		ctx.rejectWinterOrder(recorded.playerID, order, "unknown_candidate")
		return
	}
	voice := open.voices[recorded.playerID]
	if voice == 0 {
		ctx.rejectWinterOrder(recorded.playerID, order, electionNoVoiceError)
		return
	}
	if open.voted[recorded.playerID] {
		ctx.rejectWinterOrder(recorded.playerID, order, "vote_already_cast")
		return
	}
	open.voted[recorded.playerID] = true
	entry.votes += voice
}

// concludeElection decides the election and reports its public totals. A
// relative majority needs a single highest total above zero; an absolute
// majority needs strictly more than half of the denominator. A tie at the top,
// or no candidate, leaves the seat vacant.
func (ctx *resolutionContext) concludeElection(open *election) {
	result := ElectionResult{
		Kind:       open.key.kind,
		Region:     open.key.region,
		Seat:       open.seat,
		Result:     ElectionNoCandidate,
		Candidates: make([]ElectionCandidate, 0, len(open.candidates)),
	}
	if open.absolute {
		result.Required = open.denominator/2 + 1
	}
	var top *electionEntry
	tied := false
	for _, entry := range open.candidates {
		result.Candidates = append(result.Candidates, ElectionCandidate{
			Noble:     entry.noble.ID,
			NobleCode: models.NobleCode(entry.noble.Code),
			NobleName: ctx.state.NobleDisplayName(*entry.noble),
			Owner:     entry.noble.OwnerID,
			Votes:     entry.votes,
		})
		result.Cast += entry.votes
		switch {
		case top == nil || entry.votes > top.votes:
			top, tied = entry, false
		case entry.votes == top.votes:
			tied = true
		}
	}
	switch {
	case top == nil:
	case tied:
		result.Result = ElectionTie
	case top.votes == 0 || (open.absolute && top.votes < result.Required):
		result.Result = ElectionNoMajority
	default:
		result.Result = ElectionElected
		result.Winner = top.noble.ID
		ctx.pendingTitles = append(ctx.pendingTitles, pendingTitle{
			noble: top.noble.ID, kind: open.key.kind, region: open.key.region,
		})
	}
	ctx.events = append(ctx.events, Event{
		Type:        EventTypeElectionResult,
		Phase:       winterPhase,
		TerritoryID: open.seat,
		Election:    &result,
	})
}

// investWinterTitles confers the titles won this winter. A winner that left
// play or was excommunicated since the count gets nothing.
func (ctx *resolutionContext) investWinterTitles() {
	pending := ctx.pendingTitles
	ctx.pendingTitles = nil
	for _, title := range pending {
		noble := ctx.noblesByID[title.noble]
		if noble == nil {
			continue
		}
		if _, excommunicated := ctx.state.ExcommunicationOf(noble.ID); excommunicated {
			continue
		}
		switch title.kind {
		case models.ElectionBishop:
			ctx.state.Bishops = append(ctx.state.Bishops, models.Bishop{Region: title.region, Noble: noble.ID})
		case models.ElectionPope:
			id := noble.ID
			ctx.state.Pope = &id
		}
		ctx.events = append(ctx.events, Event{
			Type:      EventTypeInvestiture,
			Phase:     winterPhase,
			OwnerID:   noble.OwnerID,
			NobleID:   noble.ID,
			NobleCode: models.NobleCode(noble.Code),
			NobleName: ctx.state.NobleDisplayName(*noble),
			Election:  &ElectionResult{Kind: title.kind, Region: title.region, Candidates: []ElectionCandidate{}},
		})
	}
}
