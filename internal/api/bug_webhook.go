package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

const bugWebhookLimit = 1 << 20

// BugReportDeleter removes the private reports of a bug issue.
type BugReportDeleter interface {
	DeleteBugReportsByIssue(ctx context.Context, issueURL string) error
}

// NewBugWebhookHandler serves the GitHub webhook of the bug report issues:
// when an issue is closed, it deletes the private report linked to it
// (matched on the issue's URL), so triaged reports purge themselves. A
// delivery is accepted only if the HMAC-SHA256 of its body under the shared
// secret matches X-Hub-Signature-256. Events of a repository other than
// repository ("owner/name") and every event but a closed issue are
// acknowledged and ignored. It returns nil when the secret is empty: the route stays off.
func NewBugWebhookHandler(secret, repository string, reports BugReportDeleter) http.Handler {
	if secret == "" || reports == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST only")
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, bugWebhookLimit))
		if err != nil {
			writeAPIError(w, http.StatusRequestEntityTooLarge, "webhook_too_large", "the delivery is too large")
			return
		}
		if !validWebhookSignature(secret, body, r.Header.Get("X-Hub-Signature-256")) {
			writeAPIError(w, http.StatusUnauthorized, "invalid_signature", "the signature does not match")
			return
		}
		if r.Header.Get("X-GitHub-Event") != "issues" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var event struct {
			Action string `json:"action"`
			Issue  struct {
				HTMLURL string `json:"html_url"`
			} `json:"issue"`
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
		}
		if err := json.Unmarshal(body, &event); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_webhook", "the payload is not JSON")
			return
		}
		if event.Action != "closed" || event.Issue.HTMLURL == "" || !strings.EqualFold(event.Repository.FullName, repository) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := reports.DeleteBugReportsByIssue(context.WithoutCancel(r.Context()), event.Issue.HTMLURL); err != nil {
			// A non-2xx answer makes GitHub show the delivery as failed, so
			// it can be redelivered.
			log.Printf("bug report %s: could not delete it after the issue was closed: %v", event.Issue.HTMLURL, err)
			writeAPIError(w, http.StatusInternalServerError, "delete_failed", "the report could not be deleted")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func validWebhookSignature(secret string, body []byte, header string) bool {
	hexSum, ok := strings.CutPrefix(header, "sha256=")
	if !ok {
		return false
	}
	given, err := hex.DecodeString(hexSum)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), given)
}
