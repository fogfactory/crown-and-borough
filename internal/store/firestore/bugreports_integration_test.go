//go:build integration

package firestorestore

import (
	"context"
	"os"
	"testing"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestFirestoreBugReportsDeletedByIssue(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	backend := NewWithClient(assetgen.Balance{}, assetgen.Assets{}, Options{Client: client})
	suffix := time.Now().Format("150405.000000")
	issue := func(n string) string { return "https://github.com/o/r/issues/" + suffix + n }
	for _, id := range []string{"1", "2"} {
		report := store.BugReport{ID: "g-t1-p-issue-" + suffix + id, IssueURL: issue(id), GameView: "{}", CreatedAt: time.Now()}
		if err := backend.CreateBugReport(ctx, report); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
		if err := backend.CreateBugReport(ctx, report); err == nil {
			t.Fatalf("report %s must not be overwritten", id)
		}
	}
	if err := backend.DeleteBugReportsByIssue(ctx, issue("1")); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := backend.DeleteBugReportsByIssue(ctx, issue("9")); err != nil {
		t.Fatalf("delete unknown: %v", err)
	}
	for id, want := range map[string]bool{"1": false, "2": true} {
		snapshot, err := client.Collection(store.BugReportCollection).Doc("g-t1-p-issue-" + suffix + id).Get(ctx)
		if exists := err == nil && snapshot.Exists(); exists != want {
			t.Fatalf("report %s exists = %v, want %v", id, exists, want)
		}
	}
}
