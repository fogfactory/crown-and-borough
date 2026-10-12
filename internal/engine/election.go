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
	ElectionElected     = "elected"
	ElectionTie         = "tie"
	ElectionNoCandidate = "no_candidate"
	ElectionNoMajority  = "no_majority"
	// ElectionNoAnointment is a royal election whose leading candidate got
	// enough voices but not the blessing of the clergy.
	ElectionNoAnointment = "no_anointment"
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
	// Anointed tells whether the voices of the candidate include the blessing
	// of the clergy (royal election only).
	Anointed bool `json:"anointed,omitempty"`
}

// ElectionResult is the public outcome of one election. Required is the number
// of voices an absolute majority needs (0 for a relative majority); MinVotes
// is the least number of voices a winner needs in a relative-majority
// election; Cast is the total of voices given to a candidate.
type ElectionResult struct {
	Kind       models.ElectionKind `json:"kind"`
	Region     models.RegionID     `json:"region,omitempty"`
	Seat       models.TerritoryID  `json:"seat,omitempty"`
	Result     string              `json:"result"`
	Winner     models.NobleID      `json:"winner,omitempty"`
	Required   int                 `json:"required,omitempty"`
	MinVotes   int                 `json:"minVotes,omitempty"`
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
	// minVotes is the least number of voices the winner needs, and anointed
	// asks that those voices include the blessing of the clergy (royal
	// election, specs/politique.md).
	minVotes int
	anointed bool
	// eligible returns "" when the noble may run, the rejection reason
	// otherwise.
	eligible func(*models.Noble) string
	// voices is each player's weight, computed on the winter snapshot.
	voices map[models.PlayerID]int
	// sources is what gives each player its voices, computed with them.
	sources map[models.PlayerID][]VoiceSource

	candidates []*electionEntry
	filed      map[models.PlayerID]bool
	voted      map[models.PlayerID]bool
}

type electionEntry struct {
	noble *models.Noble
	votes int
	// clergyVotes counts the voices of bishops and above, and highClergy tells
	// that one of them is a cardinal or the pope: the royal anointment.
	clergyVotes int
	highClergy  bool
}

// royalMinVotes and royalMinBishops are the royal election thresholds
// (specs/politique.md § Élection): the voices a winner needs, which are also
// the voting titles needed to open the election, and the voices of bishops (or
// above) that anoint him. A single cardinal or the pope anoints on his own.
const (
	royalMinVotes   = 3
	royalMinBishops = 2
)

// clergyLevel tells whether a voting title is a bishop or above (bishop) and
// whether it is a cardinal or the pope (high): what counts for the royal
// anointment.
func clergyLevel(title models.ReligiousTitle) (bishop, high bool) {
	return title.Rank() >= models.ReligiousTitleBishop.Rank(), title.Rank() >= models.ReligiousTitleCardinal.Rank()
}

func (entry *electionEntry) anointed() bool {
	return entry.highClergy || entry.clergyVotes >= royalMinBishops
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
// the conclave, then the throne.
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
	if ctx.royalElectionOpen() {
		ctx.elections = append(ctx.elections, &election{
			key:      electionKey{kind: models.ElectionKing},
			minVotes: royalMinVotes,
			anointed: true,
			eligible: ctx.kingCandidateRejection,
			filed:    map[models.PlayerID]bool{},
			voted:    map[models.PlayerID]bool{},
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
	// Elected to another bishopric earlier this winter: the title is not
	// conferred yet, but the noble cannot hold a second one.
	if ctx.pendingTitle(noble.ID, models.ElectionBishop) {
		return "candidate_not_eligible"
	}
	return ""
}

// kingCandidateRejection holds the eligibility for the crown: a free man who
// can be king (no bastard) and is not excommunicated, holding at least one
// fief title. His marital status is irrelevant.
func (ctx *resolutionContext) kingCandidateRejection(noble *models.Noble) string {
	switch {
	case !noble.Sex.CanHoldReligiousOrRoyalTitle(), !noble.CanBeKing():
		return "candidate_not_eligible"
	case noble.Status != models.NobleStatusFree:
		return "candidate_not_eligible"
	}
	if _, excommunicated := ctx.state.ExcommunicationOf(noble.ID); excommunicated {
		return "candidate_not_eligible"
	}
	if ctx.state.HighestHeldFief(noble.ID) == nil {
		return "candidate_not_eligible"
	}
	// The crown and the tiara never go to the same noble: neither the pope nor
	// the noble elected pope earlier this winter (the conclave comes first).
	if ctx.state.IsPope(noble.ID) || ctx.pendingTitle(noble.ID, models.ElectionPope) {
		return "candidate_not_eligible"
	}
	return ""
}

// pendingTitle reports whether the noble won an election of the kind earlier
// this winter: the title is conferred at the investiture only, but the noble
// cannot stand for an incompatible one.
func (ctx *resolutionContext) pendingTitle(id models.NobleID, kind models.ElectionKind) bool {
	return slices.ContainsFunc(ctx.pendingTitles, func(title pendingTitle) bool {
		return title.noble == id && title.kind == kind
	})
}

// royalVoiceSources lists, for every player, the titled nobles that give it a
// voice in the royal election: one per noble holding a fief (vacant fiefs have
// no holder), a bishopric, the cardinalate or the papacy, whatever the number
// of titles. A noble in a dungeon gives none; an excommunicated one has lost
// its religious titles but keeps its fiefs.
func (ctx *resolutionContext) royalVoiceSources() map[models.PlayerID][]VoiceSource {
	sources := map[models.PlayerID][]VoiceSource{}
	for _, noble := range ctx.state.Nobles {
		if noble.Status == models.NobleStatusDungeon {
			continue
		}
		fief := ctx.state.HighestHeldFief(noble.ID)
		title := ctx.state.VotingReligiousTitle(noble.ID)
		if fief == nil && title == models.ReligiousTitleNone {
			continue
		}
		source := VoiceSource{
			Kind:      "fief",
			Noble:     models.NobleCode(noble.Code),
			NobleName: ctx.state.NobleDisplayName(noble),
			Title:     title,
			Votes:     1,
		}
		if fief != nil {
			source.Territory = fief.CapitalTerritoryID
		}
		if title != models.ReligiousTitleNone {
			source.Kind = "title"
		}
		sources[noble.OwnerID] = append(sources[noble.OwnerID], source)
	}
	return sources
}

// royalElectionOpen reports whether the royal election is open when the winter
// begins: the throne is vacant, at least royalMinVotes titles vote, and the
// clergy can anoint (two voting bishops or more, or one cardinal or the pope).
func (ctx *resolutionContext) royalElectionOpen() bool {
	if ctx.state.King != nil {
		return false
	}
	voters, bishops, cardinal := 0, 0, false
	for _, sources := range ctx.royalVoiceSources() {
		for _, source := range sources {
			voters += source.Votes
			bishop, high := clergyLevel(source.Title)
			if bishop {
				bishops++
			}
			cardinal = cardinal || high
		}
	}
	return voters >= royalMinVotes && (bishops >= royalMinBishops || cardinal)
}

func (ctx *resolutionContext) popeCandidateRejection(noble *models.Noble) string {
	if reason := ctx.commonCandidateRejection(noble); reason != "" {
		return reason
	}
	if !ctx.state.IsBishop(noble.ID) && !ctx.state.IsCardinal(noble.ID) {
		return "candidate_not_eligible"
	}
	// A king cannot become pope.
	if ctx.state.IsKing(noble.ID) {
		return "candidate_not_eligible"
	}
	return ""
}

// VoiceSource is one reason a player holds voices in an election: a
// territory of the bishopric (the seat or another one) or a religious title of
// one of the player's nobles, or the abbey of one of its abbesses.
type VoiceSource struct {
	Kind      string                `json:"kind"` // "seat", "territory", "title" or "abbey"
	Territory models.TerritoryID    `json:"territory,omitempty"`
	Noble     models.NobleCode      `json:"noble,omitempty"`
	NobleName string                `json:"nobleName,omitempty"`
	Title     models.ReligiousTitle `json:"title,omitempty"`
	Votes     int                   `json:"votes"`
}

// electionVoiceSources lists, for every player, what gives it voices in the
// election, on the state the election stage opens with.
func (ctx *resolutionContext) electionVoiceSources(open *election) map[models.PlayerID][]VoiceSource {
	if open.key.kind == models.ElectionKing {
		return ctx.royalVoiceSources()
	}
	sources := map[models.PlayerID][]VoiceSource{}
	titleSource := func(noble models.Noble, votes int) {
		if votes == 0 {
			return
		}
		sources[noble.OwnerID] = append(sources[noble.OwnerID], VoiceSource{
			Kind:      "title",
			Noble:     models.NobleCode(noble.Code),
			NobleName: ctx.state.NobleDisplayName(noble),
			Title:     ctx.state.VotingReligiousTitle(noble.ID),
			Votes:     votes,
		})
	}
	if open.key.kind == models.ElectionPope {
		// One voice per cardinal whose title is active; a cardinal in a
		// dungeon keeps its seat in the denominator but votes nothing.
		for _, noble := range ctx.state.Nobles {
			if ctx.state.IsCardinal(noble.ID) && ctx.state.VotingReligiousTitle(noble.ID) != models.ReligiousTitleNone {
				titleSource(noble, 1)
			}
		}
		return sources
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
			source := VoiceSource{Kind: "territory", Territory: territoryID, Votes: weights.Territory}
			if territoryID == region.Seed {
				source.Kind, source.Votes = "seat", weights.Seat
			}
			sources[controller] = append(sources[controller], source)
		}
	}
	for _, noble := range ctx.state.Nobles {
		titleSource(noble, weights.TitleVotes(ctx.state.VotingReligiousTitle(noble.ID)))
		if ctx.abbeyVotes(noble, open.seat) {
			sources[noble.OwnerID] = append(sources[noble.OwnerID], VoiceSource{
				Kind:      "abbey",
				Noble:     models.NobleCode(noble.Code),
				NobleName: ctx.state.NobleDisplayName(noble),
				Votes:     weights.Abbess,
			})
		}
	}
	return sources
}

// abbeyVotes reports whether the noble is an abbess whose abbey lies in the
// bishopric seated at seat and whose voice is active: an abbess in a dungeon or
// excommunicated gives none (specs/dames.md § Abbesse).
func (ctx *resolutionContext) abbeyVotes(noble models.Noble, seat models.TerritoryID) bool {
	if !noble.Has(models.DignityAbbess) || noble.AbbeyRegion == "" || noble.AbbeyRegion != seat {
		return false
	}
	if noble.Status == models.NobleStatusDungeon {
		return false
	}
	_, excommunicated := ctx.state.ExcommunicationOf(noble.ID)
	return !excommunicated
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

// resolveWinterElections settles the elections one after the other, bishoprics
// by region identifier then the conclave. Each election validates its own
// candidacies, then its votes, then counts; voices are all computed on the
// winter snapshot, but a noble who won an earlier election of the winter is no
// longer eligible to a later one of the same kind. The titles won are queued
// for the investiture.
func (ctx *resolutionContext) resolveWinterElections() {
	orders := ctx.electionOrders
	ctx.electionOrders = nil
	for _, open := range ctx.elections {
		open.sources = ctx.electionVoiceSources(open)
		open.voices = map[models.PlayerID]int{}
		for playerID, sources := range open.sources {
			for _, source := range sources {
				open.voices[playerID] += source.Votes
			}
		}
		if open.absolute {
			// A cardinal card played, or a cardinal excommunicated, earlier in
			// the winter changes the cardinals in play at the count.
			open.denominator = len(ctx.state.Cardinals)
		}
	}
	for _, recorded := range orders {
		if ctx.matchElection(recorded.order) == nil {
			ctx.rejectWinterOrder(recorded.playerID, recorded.order, "election_not_open")
		}
	}
	for _, open := range ctx.elections {
		for _, recorded := range orders {
			if recorded.order.Type == models.WinterOrderTypeCandidacy && ctx.matchElection(recorded.order) == open {
				ctx.fileCandidacy(open, recorded)
			}
		}
		for _, recorded := range orders {
			if recorded.order.Type == models.WinterOrderTypeVote && ctx.matchElection(recorded.order) == open {
				ctx.castVote(open, recorded)
			}
		}
		ctx.concludeElection(open)
	}
}

func (ctx *resolutionContext) fileCandidacy(open *election, recorded electionOrder) {
	order := recorded.order
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
	open.filed[recorded.playerID] = true
	open.candidates = append(open.candidates, &electionEntry{noble: noble})
}

func (ctx *resolutionContext) castVote(open *election, recorded electionOrder) {
	order := recorded.order
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
	if open.anointed {
		for _, source := range open.sources[recorded.playerID] {
			bishop, high := clergyLevel(source.Title)
			if bishop {
				entry.clergyVotes += source.Votes
			}
			entry.highClergy = entry.highClergy || high
		}
	}
}

// concludeElection decides the election and reports its public totals. A
// relative majority needs a single highest total above zero (and at least
// minVotes); an absolute majority needs strictly more than half of the
// denominator. The royal election also asks that the winner be anointed. A tie
// at the top, a missing anointment, or no candidate, leaves the seat vacant:
// the runner-up never takes its place.
func (ctx *resolutionContext) concludeElection(open *election) {
	result := ElectionResult{
		Kind:       open.key.kind,
		Region:     open.key.region,
		Seat:       open.seat,
		MinVotes:   open.minVotes,
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
			Anointed:  open.anointed && entry.anointed(),
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
	case top.votes == 0 || top.votes < open.minVotes || (open.absolute && top.votes < result.Required):
		result.Result = ElectionNoMajority
	case open.anointed && !top.anointed():
		result.Result = ElectionNoAnointment
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

// investWinterTitles confers the titles won this winter, the crown included:
// the king is crowned here, so his crown gives no voice before next winter. A
// winner that left play or was excommunicated since the count gets nothing.
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
		case models.ElectionKing:
			if ctx.state.King != nil {
				continue
			}
			id := noble.ID
			ctx.state.King = &id
			// The coronation brings back the fiefs left vacant by players
			// without a noble (politique.md "Héritage d'un fief sans
			// héritier"); the queen must be current to find a regent.
			ctx.state.RefreshQueen()
			ctx.returnOrphanFiefsToCrown()
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
