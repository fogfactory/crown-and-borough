package store

import (
	"context"
	"sync"
)

var _ BugReportStore = (*MemoryBugReportStore)(nil)

// MemoryBugReportStore keeps bug reports in process memory (development).
type MemoryBugReportStore struct {
	mu      sync.Mutex
	reports map[string]BugReport
}

func NewMemoryBugReportStore() *MemoryBugReportStore {
	return &MemoryBugReportStore{reports: map[string]BugReport{}}
}

func (s *MemoryBugReportStore) CreateBugReport(_ context.Context, report BugReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, taken := s.reports[report.ID]; taken {
		return ErrBugReportExists
	}
	s.reports[report.ID] = report
	return nil
}

func (s *MemoryBugReportStore) SetBugReportIssue(_ context.Context, id, issueURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if report, ok := s.reports[id]; ok {
		report.IssueURL = issueURL
		s.reports[id] = report
	}
	return nil
}

func (s *MemoryBugReportStore) DeleteBugReport(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.reports, id)
	return nil
}

// BugReport returns a saved report, for tests and local inspection.
func (s *MemoryBugReportStore) BugReport(id string) (BugReport, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report, ok := s.reports[id]
	return report, ok
}

// Len returns the number of saved reports, for tests.
func (s *MemoryBugReportStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.reports)
}
