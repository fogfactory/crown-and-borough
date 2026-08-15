package firestorestore

import "sync/atomic"

// MetricsSnapshot contains logical SDK operations observed by one adapter
// instance. Transaction callbacks may be retried, so counters include those
// attempts and are intended for relative emulator/benchmark comparisons, not
// as a billing replacement.
type MetricsSnapshot struct {
	Reads            uint64
	Writes           uint64
	Transactions     uint64
	ProjectionWrites uint64
}

type firestoreMetrics struct {
	reads            atomic.Uint64
	writes           atomic.Uint64
	transactions     atomic.Uint64
	projectionWrites atomic.Uint64
}

func (s *FirestoreStore) Metrics() MetricsSnapshot {
	if s == nil {
		return MetricsSnapshot{}
	}
	return MetricsSnapshot{
		Reads:            s.metrics.reads.Load(),
		Writes:           s.metrics.writes.Load(),
		Transactions:     s.metrics.transactions.Load(),
		ProjectionWrites: s.metrics.projectionWrites.Load(),
	}
}

func (s *FirestoreStore) recordReads(count int) {
	if count > 0 {
		s.metrics.reads.Add(uint64(count))
	}
}

func (s *FirestoreStore) recordWrites(count int) {
	if count > 0 {
		s.metrics.writes.Add(uint64(count))
	}
}

func (s *FirestoreStore) recordTransaction() {
	s.metrics.transactions.Add(1)
}

func (s *FirestoreStore) recordProjectionWrites(count int) {
	if count > 0 {
		s.metrics.projectionWrites.Add(uint64(count))
	}
}
