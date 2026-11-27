package store

import (
	"context"
	"testing"
)

func TestMemoryBugReportStoreNeverOverwrites(t *testing.T) {
	reports := NewMemoryBugReportStore()
	ctx := context.Background()
	if err := reports.CreateBugReport(ctx, BugReport{ID: "a", GameView: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := reports.CreateBugReport(ctx, BugReport{ID: "a", GameView: "second"}); err == nil {
		t.Fatal("a duplicate ID must be refused")
	}
	if report := reports.reports["a"]; report.GameView != "first" {
		t.Fatalf("report = %+v", report)
	}
	reports.CreateBugReport(ctx, BugReport{ID: "b", IssueURL: "https://x/2"})
	reports.CreateBugReport(ctx, BugReport{ID: "c", IssueURL: "https://x/3"})
	if err := reports.DeleteBugReportsByIssue(ctx, "https://x/2"); err != nil || len(reports.reports) != 2 {
		t.Fatalf("delete: %v, %d left", err, len(reports.reports))
	}
	if _, left := reports.reports["b"]; left {
		t.Fatal("the report of the issue must be deleted")
	}
	if err := reports.DeleteBugReportsByIssue(ctx, "https://x/9"); err != nil {
		t.Fatalf("deleting for an unknown issue must succeed: %v", err)
	}
}
