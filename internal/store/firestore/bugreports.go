package firestorestore

import (
	"context"
	"errors"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

var _ store.BugReportStore = (*FirestoreStore)(nil)

// bugReportDocument is stored under its readable ID; IssueURL links it back to
// the issue and is what the webhook finds it by when the issue is closed.
type bugReportDocument struct {
	GameID   store.GameID    `firestore:"gameId"`
	Turn     int             `firestore:"turn"`
	PlayerID models.PlayerID `firestore:"playerId"`
	UID      string          `firestore:"uid"`
	Title    string          `firestore:"title"`
	IssueURL string          `firestore:"issueUrl"`
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
	// Create, not Set: an ID collision must never overwrite another report.
	_, err := s.bugReportRef(report.ID).Create(operationContext, bugReportDocument{
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
	return wrapOperation("create bug report", err)
}

func (s *FirestoreStore) DeleteBugReportsByIssue(ctx context.Context, issueURL string) error {
	if err := s.requireClient(); err != nil {
		return err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	iter := s.client.Collection(store.BugReportCollection).Where("issueUrl", "==", issueURL).Documents(operationContext)
	defer iter.Stop()
	for {
		snapshot, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return nil
		}
		if err != nil {
			return wrapOperation("find bug reports", err)
		}
		_, err = snapshot.Ref.Delete(operationContext)
		s.recordWrites(1)
		if err != nil {
			return wrapOperation("delete bug report", err)
		}
	}
}
