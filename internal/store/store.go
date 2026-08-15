// Package store owns the server-side lifecycle of independent games.
//
// The store deliberately depends on the pure engine types rather than on HTTP
// or a persistence SDK. A future Firestore implementation can satisfy the same
// interface without changing the engine or the API projections.
package store

import (
	"context"
	"errors"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

type GameID string

type Revision uint64

type Actor struct {
	ID string `json:"id"`
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
	Name    string              `json:"name"`
	Seed    string              `json:"seed"`
	Players []engine.PlayerInit `json:"players"`
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

// RevisionedGameStore is the optimistic-concurrency extension used by the
// HTTP forced-resolution route. Firestore can implement the same operation as
// a transaction; the memory store uses the per-game mutex and an equality
// check. GameStore remains smaller so other adapters can adopt it gradually.
type RevisionedGameStore interface {
	GameStore
	ResolveAt(context.Context, Actor, GameID, Revision) (SubmitResult, error)
}

var (
	ErrUnknownGame      = errors.New("store: unknown game")
	ErrNotMember        = errors.New("store: actor is not a member of this game")
	ErrInvalidPlayers   = errors.New("store: player count must be between two and eight")
	ErrGameFinished     = errors.New("store: game is finished")
	ErrEliminated       = errors.New("store: eliminated player cannot submit")
	ErrRevisionConflict = errors.New("store: revision conflict")
	ErrInvalidReport    = errors.New("store: invalid report index")
)
