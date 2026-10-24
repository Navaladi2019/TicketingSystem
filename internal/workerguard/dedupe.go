package workerguard

import (
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// SignalDedupeWindow is how long an applied signal's outbox id is kept
// (PLAN-001 D-P8): 48 h, longer than the relay's 24 h maximum redelivery age
// from first attempt to dead_at, as ADR-001 A.2 requires.
const SignalDedupeWindow = 48 * time.Hour

// AppliedSignals is the set of outbox ids of the signals a workflow has
// applied (I7). The relay may deliver a signal more than once; the handler
// applies one only if TryApply says so. Entries older than SignalDedupeWindow
// are dropped, which bounds the set.
//
// Pass it to the next run as part of the continue-as-new input, so a signal
// redelivered after continue-as-new is still applied once. It is plain data
// (JSON-encoded by the default converter); the zero value is empty and ready.
type AppliedSignals struct {
	// Applied maps an outbox id to the workflow time it was applied.
	Applied map[uuid.UUID]time.Time `json:"applied,omitempty"`
}

// TryApply reports whether the signal with this outbox id is new, and if so
// records it. It uses workflow time, so it is deterministic on replay. A nil
// id is never applied: every relay signal carries its outbox id.
func (s *AppliedSignals) TryApply(ctx workflow.Context, outboxID uuid.UUID) bool {
	if outboxID == uuid.Nil {
		return false
	}
	now := workflow.Now(ctx)
	s.prune(now)
	if _, seen := s.Applied[outboxID]; seen {
		return false
	}
	if s.Applied == nil {
		s.Applied = make(map[uuid.UUID]time.Time)
	}
	s.Applied[outboxID] = now
	return true
}

// Len is the number of ids held.
func (s *AppliedSignals) Len() int { return len(s.Applied) }

// prune drops the ids applied before now - SignalDedupeWindow. The result
// does not depend on map order, so replay is deterministic.
func (s *AppliedSignals) prune(now time.Time) {
	cutoff := now.Add(-SignalDedupeWindow)
	for id, at := range s.Applied {
		if at.Before(cutoff) {
			delete(s.Applied, id)
		}
	}
}
