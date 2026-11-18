// Package store owns the server-side lifecycle of independent games.
//
// The store deliberately depends on the pure engine types rather than on HTTP
// or a persistence SDK. A future Firestore implementation can satisfy the same
// interface without changing the engine or the API projections.
package store

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/turn"
)

type GameID string

type Revision uint64

type Actor struct {
	ID          string `json:"id"`
	Email       string `json:"-"`
	GameCreator bool   `json:"-"`
	Development bool   `json:"-"`
}

// PlayerProfile is the application-owned part of a Firebase identity. UID and
// Email come from the verified token; DisplayName is the only mutable field.
type PlayerProfile struct {
	UID         string `json:"uid"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	// LastSeenVersion is the last deployed version whose release notes the
	// player has acknowledged; empty until the first acknowledgement.
	LastSeenVersion string `json:"lastSeenVersion"`
	// HideReleaseNotes opts the player out of the release notes dialog.
	HideReleaseNotes bool `json:"hideReleaseNotes"`
	// Notifications holds the opt-in email notification settings; the zero
	// value (every notification off) is what profiles created before the
	// feature carry.
	Notifications NotificationPreferences `json:"notifications"`
	CreatedAt     time.Time               `json:"-"`
	UpdatedAt     time.Time               `json:"-"`
}

// ReleaseNotesUpdate carries the release notes preferences to change; nil
// fields are left untouched.
type ReleaseNotesUpdate struct {
	LastSeenVersion  *string
	HideReleaseNotes *bool
}

type ProfileStore interface {
	GetProfile(context.Context, string) (PlayerProfile, error)
	EnsureProfile(context.Context, Actor) (PlayerProfile, error)
	UpdateProfile(context.Context, string, string) (PlayerProfile, error)
	UpdateReleaseNotes(context.Context, string, ReleaseNotesUpdate) (PlayerProfile, error)
	UpdateNotifications(context.Context, string, NotificationsUpdate) (PlayerProfile, error)
}

type Invitation struct {
	GameID    GameID    `json:"gameId"`
	CreatedBy string    `json:"createdBy"`
	CodeHash  string    `json:"codeHash"`
	CreatedAt time.Time `json:"createdAt"`
	Active    bool      `json:"active"`
}

// InvitationSecret is intentionally short-lived. Implementations must retain
// only Invitation.CodeHash and return the clear code to the caller once.
type InvitationSecret struct {
	GameID    GameID
	CreatedBy string
	Code      string
}

type InvitationStore interface {
	CreateInvitation(context.Context, GameID, string) (InvitationSecret, error)
	LookupInvitation(context.Context, string) (Invitation, error)
}

type Membership struct {
	GameID    GameID          `json:"gameId"`
	UID       string          `json:"uid"`
	PlayerID  models.PlayerID `json:"playerId"`
	Spectator bool            `json:"spectator,omitempty"`
	JoinedAt  time.Time       `json:"joinedAt"`
}

type MembershipStore interface {
	ListMemberships(context.Context, GameID) ([]Membership, error)
	ListActorMemberships(context.Context, string) ([]Membership, error)
}

// BugReportCollection is the Firestore collection holding bug reports, apart
// from the games.
const BugReportCollection = "bugReports"

// BugReport is the private part of a player's bug report: their own game view,
// kept out of the public GitHub issue, which only cites Ref.
type BugReport struct {
	// ID is human readable: <game>-t<turn>-<player>-<ref>.
	ID string
	// Ref is the opaque random suffix of ID, the only part the public issue
	// shows (the rest names the game and the player).
	Ref      string
	GameID   GameID
	Turn     int
	PlayerID models.PlayerID
	UID      string
	Title    string
	IssueURL string
	// GameView is the masked JSON projection of the game the player sees.
	GameView  string
	CreatedAt time.Time
}

// BugReportStore keeps bug reports in a collection of their own, apart from
// the games, so maintainers can read and delete them freely.
type BugReportStore interface {
	// CreateBugReport fails if the ID is taken; it never overwrites a report.
	CreateBugReport(context.Context, BugReport) error
	SetBugReportIssue(ctx context.Context, id, issueURL string) error
	DeleteBugReport(ctx context.Context, id string) error
}

type Status string

const (
	StatusPlaying  Status = "playing"
	StatusFinished Status = "finished"
)

const (
	MinimumPlayers = 2
	MaximumPlayers = 8
)

type PlayerSlot struct {
	ID      models.PlayerID `json:"id"`
	Name    string          `json:"name"`
	Color   string          `json:"color"`
	ActorID string          `json:"actorId,omitempty"`
	// AccountName is the display name of the member's account, recorded
	// outside incognito games only (and absent from games that predate it).
	// Name is the house name, a separate concept.
	AccountName string `json:"accountName,omitempty"`
	// NameLocked is set by the house's first order submission: from then on an
	// incognito game no longer lets the house change its name.
	NameLocked bool `json:"nameLocked,omitempty"`
}

type CreateRequest struct {
	Name             string              `json:"name"`
	Seed             string              `json:"seed"`
	Players          []engine.PlayerInit `json:"players"`
	YearCount        int                 `json:"years"`
	Spectate         bool                `json:"spectate,omitempty"`
	StrictMembership bool                `json:"-"`
	// Incognito hides the accounts behind the houses: slots carry placeholder
	// names the players replace themselves, and the game waits for every slot
	// to be taken before orders are accepted.
	Incognito bool `json:"incognito,omitempty"`
}

type SubmitRequest struct {
	Chains           []engine.ChainSubmission  `json:"chains"`
	Winter           []engine.WinterSubmission `json:"winter"`
	Special          []engine.DeckSubmission   `json:"special"`
	Force            bool                      `json:"force,omitempty"`
	ExpectedRevision Revision                  `json:"revision,omitempty"`
}

type ReportRecord struct {
	Report  engine.TurnReport   `json:"report"`
	Privacy *models.PrivacyMeta `json:"privacy,omitempty"`
}

type GameSnapshot struct {
	ID           GameID                                    `json:"id"`
	Name         string                                    `json:"name"`
	Seed         string                                    `json:"seed"`
	YearCount    int                                       `json:"yearCount"`
	Status       Status                                    `json:"status"`
	Winner       *models.PlayerID                          `json:"winner,omitempty"`
	Scores       map[models.PlayerID]engine.ScoreBreakdown `json:"scores"`
	Players      []PlayerSlot                              `json:"players"`
	Map          mapgen.MapData                            `json:"map"`
	State        *models.GameState                         `json:"state"`
	Submissions  map[models.PlayerID]engine.OrdersInput    `json:"submissions"`
	Reports      []ReportRecord                            `json:"reports"`
	Revision     Revision                                  `json:"revision"`
	CreatedBy    string                                    `json:"createdBy,omitempty"`
	SpectatorUID string                                    `json:"spectatorUid,omitempty"`
	Incognito    bool                                      `json:"incognito,omitempty"`
	// Lobby is the caller's own part of a game whose lobby is still open; nil
	// once the game has started.
	Lobby *LobbyView `json:"-"`
	// FixedSeed is the seed the creator imposed on a lobby, if any.
	FixedSeed string `json:"-"`
}

type SubmitResult struct {
	Status    string            `json:"status"`
	Player    models.PlayerID   `json:"player"`
	Submitted []models.PlayerID `json:"submitted"`
	Remaining []models.PlayerID `json:"remaining"`
	Resolved  bool              `json:"resolved"`
	Forced    bool              `json:"forced,omitempty"`
	Report    *ReportRecord     `json:"report,omitempty"`
	Snapshot  GameSnapshot      `json:"snapshot"`
}

type JoinResult struct {
	Snapshot GameSnapshot
	Player   PlayerSlot
	Joined   bool
	// Started is true when this very join took the last slot and so started
	// the game.
	Started bool
}

// JoinOptions is what a joiner prepares for the lobby along with the join,
// applied before the game can start: the last joiner's name and vote count
// like everyone else's. Empty fields are left out; the validation is the
// lobby routes'.
type JoinOptions struct {
	HouseName   string
	SeedRanking []int
}

// JoinPreview is what someone holding a valid invitation (or a member) may see
// of a game before joining it: enough to prepare a house name and a vote.
type JoinPreview struct {
	ID        GameID
	Name      string
	Players   int
	Seated    int
	Incognito bool
	Started   bool
	// Member is true when the caller already holds a slot.
	Member bool
	// SeedCandidates is empty once the game started or when the creator
	// imposed the seed.
	SeedCandidates []string
}

// RenameResult is the outcome of InvitationGameStore.RenameHouse.
type RenameResult struct {
	Snapshot GameSnapshot
	Player   PlayerSlot
	// PreviousName is the name the house carried before the call; it equals
	// Player.Name when the call changed nothing.
	PreviousName string
}

type GameCreation struct {
	Snapshot   GameSnapshot
	Invitation InvitationSecret
}

type PlayerSubmission struct {
	Turn      int                `json:"turn"`
	Season    models.Season      `json:"season"`
	Submitted bool               `json:"submitted"`
	Orders    engine.OrdersInput `json:"orders"`
}

// PrivacyTracker is called while a game is exclusively locked. See
// turn.PrivacyTracker.
type PrivacyTracker = turn.PrivacyTracker

// IsAssignedActor reports whether a slot's ActorID names a real member.
// Development stores use "slot:" placeholders for slots nobody claimed yet.
func IsAssignedActor(actorID string) bool {
	return actorID != "" && !strings.HasPrefix(actorID, "slot:")
}

// SlotsFilled reports whether every player slot is held by a member.
func SlotsFilled(players []PlayerSlot) bool {
	for _, player := range players {
		if !IsAssignedActor(strings.TrimSpace(player.ActorID)) {
			return false
		}
	}
	return true
}

// Started reports whether the game has a board: an online game builds its
// map and state when its last slot is taken, and until then (its lobby) it
// has neither.
func (s GameSnapshot) Started() bool {
	return s.State != nil
}

// SeatIncognitoPlayers draws which member holds which house once every slot is
// taken. The members are sorted before a seed-derived shuffle, so the outcome
// depends on the seed and the set of members only, never on the order they
// joined in. Houses keep their slot, hence their starting territory.
func SeatIncognitoPlayers(seed string, players []PlayerSlot) {
	actors := make([]string, len(players))
	for index, player := range players {
		actors[index] = player.ActorID
	}
	slices.Sort(actors)
	digest := sha256.Sum256([]byte(seed + "|incognito-seats"))
	rng := rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
	for index, pick := range rng.Perm(len(players)) {
		players[index].ActorID = actors[pick]
	}
}

// ValidateHouseRename applies the rules of a house rename in a started game to
// the slot of playerID: the house must not have submitted orders yet, and the
// normalized name must differ from every other house's. It returns the
// normalized name.
func ValidateHouseRename(players []PlayerSlot, playerID models.PlayerID, name string) (string, error) {
	normalized, err := NormalizeDisplayName(name)
	if err != nil {
		return "", err
	}
	for _, player := range players {
		if player.ID == playerID {
			if player.NameLocked && player.Name != normalized {
				return "", ErrNameLocked
			}
			continue
		}
		if strings.EqualFold(player.Name, normalized) {
			return "", ErrHouseNameTaken
		}
	}
	return normalized, nil
}

// PlayerForActor returns the player slot claimed by actor. A development actor
// may also address an unclaimed slot by its player ID.
func PlayerForActor(players []PlayerSlot, actor Actor) (models.PlayerID, bool) {
	actorID := strings.TrimSpace(actor.ID)
	if actorID == "" {
		return "", false
	}
	for _, player := range players {
		if player.ActorID == actorID && IsAssignedActor(player.ActorID) {
			return player.ID, true
		}
		if actor.Development && player.ActorID == "" && string(player.ID) == actorID {
			return player.ID, true
		}
	}
	return "", false
}

// IsSpectator reports whether actor is the observer host of a game.
func IsSpectator(spectatorUID string, actor Actor) bool {
	spectatorUID = strings.TrimSpace(spectatorUID)
	return spectatorUID != "" && spectatorUID == strings.TrimSpace(actor.ID)
}

// PlayerFor returns the player slot claimed by actor in this game.
func (s GameSnapshot) PlayerFor(actor Actor) (models.PlayerID, bool) {
	return PlayerForActor(s.Players, actor)
}

// ViewerFor returns the identity used to project this game for actor: its
// player ID, or models.SpectatorViewer for the observer host.
func (s GameSnapshot) ViewerFor(actor Actor) (models.PlayerID, bool) {
	if playerID, ok := s.PlayerFor(actor); ok {
		return playerID, true
	}
	if IsSpectator(s.SpectatorUID, actor) {
		return models.SpectatorViewer, true
	}
	return "", false
}

type GameStore interface {
	Create(context.Context, Actor, CreateRequest) (GameSnapshot, error)
	List(context.Context, Actor) ([]GameSnapshot, error)
	Get(context.Context, Actor, GameID) (GameSnapshot, error)
	Map(context.Context, Actor, GameID) (mapgen.MapData, error)
	State(context.Context, Actor, GameID) (GameSnapshot, error)
	Supply(context.Context, Actor, GameID, models.TerritoryID, string) (engine.SupplyLine, error)
	Submit(context.Context, Actor, GameID, SubmitRequest) (SubmitResult, error)
	Resolve(context.Context, Actor, GameID) (SubmitResult, error)
	Reports(context.Context, Actor, GameID) ([]ReportRecord, error)
	Report(context.Context, Actor, GameID, int) (ReportRecord, error)
	MySubmission(context.Context, Actor, GameID) (PlayerSubmission, error)
}

// TransferSupplyStore is implemented by stores that expose the action-turn
// transfer overlay in addition to the ordinary supply projection.
type TransferSupplyStore interface {
	TransferSupply(context.Context, Actor, GameID, models.TerritoryID, models.TerritoryID) (engine.TransferLine, error)
}

// InvitationGameStore is the optional online extension of GameStore. Keeping
// it separate lets existing engine and hotseat adapters remain source
// compatible while the authenticated API uses invitations and memberships.
type InvitationGameStore interface {
	GameStore
	ProfileStore
	MembershipStore
	Join(context.Context, Actor, GameID, string) (JoinResult, error)
	CreateInvitation(context.Context, Actor, GameID) (InvitationSecret, error)
	// JoinWith is Join with the lobby entry (house name, seed ranking) the
	// joiner prepared, validated and applied before the start plan is drawn.
	JoinWith(context.Context, Actor, GameID, string, JoinOptions) (JoinResult, error)
	// JoinPreview describes a game to a member or to the holder of its
	// invitation code (the last argument); anyone else gets
	// ErrInvalidInvitation.
	JoinPreview(context.Context, Actor, GameID, string) (JoinPreview, error)
	// RenameHouse names the actor's house; in the lobby it only records the
	// name the house takes when the game starts.
	RenameHouse(context.Context, Actor, GameID, string) (RenameResult, error)
	// SetSeedVote records the actor's ranking of the candidate seeds while
	// the lobby is open; an empty ranking withdraws the vote.
	SetSeedVote(context.Context, Actor, GameID, []int) (GameSnapshot, error)
	// SeedCandidateMap generates the map of one candidate seed for the game's
	// player count; it reveals nothing but the map. It is open to a member
	// and to the holder of the invitation code (the fourth argument).
	SeedCandidateMap(context.Context, Actor, GameID, string, int) (mapgen.MapData, error)
}

// RevisionedGameStore is the optimistic-concurrency extension used by the
// HTTP forced-resolution route. Firestore can implement the same operation as
// a transaction; the memory store uses the per-game mutex and an equality
// check. GameStore remains smaller so other adapters can adopt it gradually.
type RevisionedGameStore interface {
	GameStore
	ResolveAt(context.Context, Actor, GameID, Revision) (SubmitResult, error)
}

var (
	ErrUnknownGame          = errors.New("store: unknown game")
	ErrNotMember            = errors.New("store: actor is not a member of this game")
	ErrNotCreator           = errors.New("store: actor is not the game creator")
	ErrInvalidPlayers       = errors.New("store: player count must be between two and eight")
	ErrGameFull             = errors.New("store: game has no free slots")
	ErrGameFinished         = errors.New("store: game is finished")
	ErrEliminated           = errors.New("store: eliminated player cannot submit")
	ErrRevisionConflict     = errors.New("store: revision conflict")
	ErrInvalidReport        = errors.New("store: invalid report index")
	ErrProfileNotFound      = errors.New("store: profile not found")
	ErrProfileRequired      = errors.New("store: a completed player profile is required")
	ErrInvalidDisplayName   = errors.New("store: invalid display name")
	ErrInvalidReleaseNotes  = errors.New("store: invalid release notes update")
	ErrInvalidNotifications = errors.New("store: invalid notification settings update")
	ErrInvalidInvitation    = errors.New("store: invalid invitation")
	ErrInvitationInactive   = errors.New("store: invitation is inactive")
	ErrSpectator            = errors.New("store: spectator host cannot join a player slot")
	ErrInvalidYears         = errors.New("store: year count must be between one and fifty")
	ErrGameNotStarted       = errors.New("store: the game waits for every player to join")
	ErrInvalidSeedRanking   = errors.New("store: a seed ranking is a permutation of the candidate indexes")
	ErrGameStarted          = errors.New("store: the game has started")
	ErrInvalidCandidate     = errors.New("store: unknown seed candidate")
	ErrNameLocked           = errors.New("store: house name is locked once orders were submitted")
	ErrHouseNameTaken       = errors.New("store: house name is already taken in this game")
)

func normalizeYearCount(value int) (int, error) {
	years, err := models.NormalizeGameYears(value)
	if err != nil {
		return 0, ErrInvalidYears
	}
	return years, nil
}
