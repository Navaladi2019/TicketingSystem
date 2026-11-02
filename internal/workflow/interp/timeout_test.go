package interp_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// maxApplyStepStartToClose is the P3-24 contract bound on one ApplyStep
// attempt (TICK-387). A worker killed mid-attempt is retried only when the
// attempt's StartToCloseTimeout expires, so the crash stack case's 90 s
// budget depends on it. It is a literal on purpose: the test must not follow
// the code's constant.
const maxApplyStepStartToClose = 10 * time.Second

// TestApplyStepStartToCloseWithinContract: every ApplyStep (and NextStep,
// which shares the options) the run scheduled carried a StartToCloseTimeout
// set and at most 10 s.
func TestApplyStepStartToCloseWithinContract(t *testing.T) {
	h, f, version, record := v1(t, "high")
	h.signalAt(time.Hour, uuid.New(), f.id(t, def.SignalKey("approve")))
	h.start(version, record)
	h.wantCompleted()
	seen := map[string]bool{}
	for _, s := range h.scheduledOptions() {
		if s.activity != interp.NextStepActivity && s.activity != interp.ApplyStepActivity {
			continue
		}
		seen[s.activity] = true
		if got := s.options.StartToCloseTimeout; got <= 0 || got > maxApplyStepStartToClose {
			t.Errorf("%s scheduled with StartToCloseTimeout %v, want > 0 and <= %v", s.activity, got, maxApplyStepStartToClose)
		}
	}
	if !seen[interp.ApplyStepActivity] {
		t.Fatal("ApplyStep was never scheduled")
	}
}
