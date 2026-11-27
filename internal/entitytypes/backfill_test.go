package entitytypes

import "testing"

// The window is open until the newest job is done (ADR-009 Terms).
func TestBackfillStatusOpen(t *testing.T) {
	for state, want := range map[string]bool{
		"pending": true, "filling": true, "running": true, "failed_records": true,
		"done": false, "superseded": false, "": false,
	} {
		if got := (&BackfillStatus{State: state}).Open(); got != want {
			t.Errorf("Open(%q) = %v, want %v", state, got, want)
		}
	}
}
