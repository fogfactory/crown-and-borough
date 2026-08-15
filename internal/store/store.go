// Package store owns the server-side lifecycle of independent games.
//
// The store deliberately depends on the pure engine types rather than on HTTP
// or a persistence SDK. A future Firestore implementation can satisfy the same
// interface without changing the engine or the API projections.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

type GameID string

type Revision uint64

type Actor struct {
	ID          string `json:"id"`
	Email       string `json:"-"`
	Development bool   `json:"-"`
}

// PlayerProfile is the application-owned part of a Firebase identity. UID and
// Email come from the verified token; DisplayName is the only mutable field.
type PlayerProfile struct {
	UID         string    `json:"uid"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

type ProfileStore interface {
	GetProfile(context.Context, string) (PlayerProfile, error)
	EnsureProfile(context.Context, Actor) (PlayerProfile, error)
	UpdateProfile(context.Context, string, string) (PlayerProfile, error)
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
	GameID   GameID          `json:"gameId"`
	UID      string          `json:"uid"`
	PlayerID models.PlayerID `json:"playerId"`
	JoinedAt time.Time       `json:"joinedAt"`
}

type MembershipStore interface {
	ListMemberships(context.Context, GameID) ([]Membership, error)
	ListActorMemberships(context.Context, string) ([]Membership, error)
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
}

type CreateRequest struct {
	Name             string              `json:"name"`
	Seed             string              `json:"seed"`
	Players          []engine.PlayerInit `json:"players"`
	StrictMembership bool                `json:"-"`
}

type SubmitRequest struct {
	Chains           []engine.ChainSubmission  `json:"chains"`
	Winter           []engine.WinterSubmission `json:"winter"`
	Force            bool                      `json:"force,omitempty"`
	ExpectedRevision Revision                  `json:"revision,omitempty"`
}

type ReportRecord struct {
	Report  engine.TurnReport   `json:"report"`
	Privacy *models.PrivacyMeta `json:"privacy,omitempty"`
}

type GameSnapshot struct {
	ID          GameID                                 `json:"id"`
	Name        string                                 `json:"name"`
	Seed        string                                 `json:"seed"`
	Status      Status                                 `json:"status"`
	Winner      *models.PlayerID                       `json:"winner,omitempty"`
	Players     []PlayerSlot                           `json:"players"`
	Map         mapgen.MapData                         `json:"map"`
	State       *models.GameState                      `json:"state"`
	Submissions map[models.PlayerID]engine.OrdersInput `json:"submissions"`
	Reports     []ReportRecord                         `json:"reports"`
	Revision    Revision                               `json:"revision"`
	CreatedBy   string                                 `json:"createdBy,omitempty"`
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
}

type GameCreation struct {
	Snapshot   GameSnapshot
	Invitation InvitationSecret
}

// PrivacyTracker is called while a game is exclusively locked, immediately
// after the engine has produced the next state and report. It is an injection
// point for server-side privacy metadata; the engine itself never needs to
// know about viewers.
type PrivacyTracker func(before, after *models.GameState, input engine.OrdersInput, report engine.TurnReport)

type GameStore interface {
	Create(context.Context, Actor, CreateRequest) (GameSnapshot, error)
	List(context.Context, Actor) ([]GameSnapshot, error)
	Get(context.Context, Actor, GameID) (GameSnapshot, error)
	Map(context.Context, Actor, GameID) (mapgen.MapData, error)
	State(context.Context, Actor, GameID) (GameSnapshot, error)
	Supply(context.Context, Actor, GameID, models.TerritoryID) (engine.SupplyLine, error)
	Submit(context.Context, Actor, GameID, SubmitRequest) (SubmitResult, error)
	Resolve(context.Context, Actor, GameID) (SubmitResult, error)
	Reports(context.Context, Actor, GameID) ([]ReportRecord, error)
	Report(context.Context, Actor, GameID, int) (ReportRecord, error)
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
	ErrUnknownGame        = errors.New("store: unknown game")
	ErrNotMember          = errors.New("store: actor is not a member of this game")
	ErrNotCreator         = errors.New("store: actor is not the game creator")
	ErrInvalidPlayers     = errors.New("store: player count must be between two and eight")
	ErrGameFull           = errors.New("store: game has no free slots")
	ErrGameFinished       = errors.New("store: game is finished")
	ErrEliminated         = errors.New("store: eliminated player cannot submit")
	ErrRevisionConflict   = errors.New("store: revision conflict")
	ErrInvalidReport      = errors.New("store: invalid report index")
	ErrProfileNotFound    = errors.New("store: profile not found")
	ErrProfileRequired    = errors.New("store: a completed player profile is required")
	ErrInvalidDisplayName = errors.New("store: invalid display name")
	ErrInvalidInvitation  = errors.New("store: invalid invitation")
	ErrInvitationInactive = errors.New("store: invitation is inactive")
)
