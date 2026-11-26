package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

func signedDelivery(t *testing.T, handler http.Handler, secret, event, body string) int {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	request := httptest.NewRequest(http.MethodPost, "/api/github/webhook", strings.NewReader(body))
	request.Header.Set("X-GitHub-Event", event)
	request.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func TestBugWebhookDeletesTheReportOfAClosedIssue(t *testing.T) {
	reports := newFakeReports()
	reports.reports["g-t1-p-issue-7"] = store.BugReport{ID: "g-t1-p-issue-7", IssueURL: "https://github.com/o/r/issues/7"}
	reports.reports["g-t1-p-issue-8"] = store.BugReport{ID: "g-t1-p-issue-8", IssueURL: "https://github.com/o/r/issues/8"}
	handler := NewBugWebhookHandler("s3cret", "o/r", reports)

	closed := func(number, repo string) string {
		return `{"action":"closed","issue":{"html_url":"https://github.com/o/r/issues/` + number + `"},"repository":{"full_name":"` + repo + `"}}`
	}
	// Ignored: other events, other actions, other repositories.
	for _, c := range []struct{ event, body string }{
		{"ping", `{"zen":"x"}`},
		{"issues", strings.Replace(closed("7", "o/r"), "closed", "reopened", 1)},
		{"issues", closed("7", "other/repo")},
	} {
		if code := signedDelivery(t, handler, "s3cret", c.event, c.body); code != http.StatusNoContent {
			t.Fatalf("%s = %d", c.event, code)
		}
	}
	if len(reports.reports) != 2 {
		t.Fatalf("ignored deliveries deleted reports: %v", reports.reports)
	}
	if code := signedDelivery(t, handler, "s3cret", "issues", closed("7", "o/r")); code != http.StatusNoContent {
		t.Fatalf("closed = %d", code)
	}
	if _, left := reports.reports["g-t1-p-issue-7"]; left || len(reports.reports) != 1 {
		t.Fatalf("reports = %v", reports.reports)
	}
	// Closing an issue without report is not an error.
	if code := signedDelivery(t, handler, "s3cret", "issues", closed("99", "o/r")); code != http.StatusNoContent {
		t.Fatalf("unknown issue = %d", code)
	}
}

func TestBugWebhookRejectsBadSignatures(t *testing.T) {
	reports := newFakeReports()
	reports.reports["g-t1-p-issue-7"] = store.BugReport{ID: "g-t1-p-issue-7", IssueURL: "https://github.com/o/r/issues/7"}
	handler := NewBugWebhookHandler("s3cret", "o/r", reports)
	body := `{"action":"closed","issue":{"html_url":"https://github.com/o/r/issues/7"},"repository":{"full_name":"o/r"}}`
	if code := signedDelivery(t, handler, "wrong", "issues", body); code != http.StatusUnauthorized {
		t.Fatalf("wrong secret = %d", code)
	}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	request.Header.Set("X-GitHub-Event", "issues")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || len(reports.reports) != 1 {
		t.Fatalf("unsigned = %d, reports %v", response.Code, reports.reports)
	}
}

func TestBugWebhookIsOffWithoutSecret(t *testing.T) {
	if NewBugWebhookHandler("", "o/r", newFakeReports()) != nil {
		t.Fatal("an empty secret must disable the webhook")
	}
}
