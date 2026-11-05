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
	if err := reports.SetBugReportIssue(ctx, "a", "https://x/1"); err != nil {
		t.Fatal(err)
	}
	if report := reports.reports["a"]; report.GameView != "first" || report.IssueURL != "https://x/1" {
		t.Fatalf("report = %+v", report)
	}
	if err := reports.DeleteBugReport(ctx, "a"); err != nil || len(reports.reports) != 0 {
		t.Fatalf("delete: %v, %d left", err, len(reports.reports))
	}
}
