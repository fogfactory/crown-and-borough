package firestorestore

import (
	"context"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

var _ store.BugReportStore = (*FirestoreStore)(nil)

type bugReportDocument struct {
	ID       string          `firestore:"id"`
	Ref      string          `firestore:"ref"`
	GameID   store.GameID    `firestore:"gameId"`
	Turn     int             `firestore:"turn"`
	PlayerID models.PlayerID `firestore:"playerId"`
	UID      string          `firestore:"uid"`
	Title    string          `firestore:"title"`
	IssueURL string          `firestore:"issueUrl,omitempty"`
	// GameViewJSON is stored as a string, like mapJson, so it keeps the exact
	// shape of GET /api/games/{id}/state.
	GameViewJSON string    `firestore:"gameViewJson"`
	CreatedAt    time.Time `firestore:"createdAt"`
}

// bugReportRef points into the top-level collection, apart from games, so
// reports can be read and deleted without touching game documents.
func (s *FirestoreStore) bugReportRef(id string) *cloudfirestore.DocumentRef {
	return s.client.Collection(store.BugReportCollection).Doc(id)
}

func (s *FirestoreStore) CreateBugReport(ctx context.Context, report store.BugReport) error {
	if err := s.requireClient(); err != nil {
		return err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	_, err := s.bugReportRef(report.ID).Create(operationContext, bugReportDocument{
		ID:           report.ID,
		Ref:          report.Ref,
		GameID:       report.GameID,
		Turn:         report.Turn,
		PlayerID:     report.PlayerID,
		UID:          report.UID,
		Title:        report.Title,
		IssueURL:     report.IssueURL,
		GameViewJSON: report.GameView,
		CreatedAt:    report.CreatedAt.UTC(),
	})
	s.recordWrites(1)
	if status.Code(err) == codes.AlreadyExists {
		return store.ErrBugReportExists
	}
	return err
}

func (s *FirestoreStore) SetBugReportIssue(ctx context.Context, id, issueURL string) error {
	if err := s.requireClient(); err != nil {
		return err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	_, err := s.bugReportRef(id).Update(operationContext, []cloudfirestore.Update{{Path: "issueUrl", Value: issueURL}})
	s.recordWrites(1)
	return err
}

func (s *FirestoreStore) DeleteBugReport(ctx context.Context, id string) error {
	if err := s.requireClient(); err != nil {
		return err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	_, err := s.bugReportRef(id).Delete(operationContext)
	s.recordWrites(1)
	return err
}
