package verify

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
)

// TestRecordEraseCap: an audit/erase entry lists at most 1,500 ids
// (R23 as amended by ADR-004-E1, the writer's audit.MaxErasedIDs). One with
// 1,501 ids lists nothing, so the entries it names verify as unlisted.
func TestRecordEraseCap(t *testing.T) {
	if maxErasedIDs != 1500 {
		t.Fatalf("maxErasedIDs = %d, want 1500 (ADR-004-E1)", maxErasedIDs)
	}
	for _, c := range []struct {
		n    int
		want int
	}{{1500, 1500}, {1501, 0}} {
		ids := make([]string, c.n)
		for i := range ids {
			ids[i] = uuid.NewString()
		}
		b, err := json.Marshal([]map[string]any{{"op": "add", "path": "/erased_ids", "value": ids}})
		if err != nil {
			t.Fatal(err)
		}
		d := string(b)
		ref := uuid.New()
		into := map[string]map[string]bool{}
		recordErase(into, entryLine{
			Envelope: chain.Envelope{Domain: "audit", Action: "erase", SubjectType: "erasure_request", SubjectID: ref},
			Diff:     &d,
		})
		if got := len(into[ref.String()]); got != c.want {
			t.Errorf("%d ids: %d listed, want %d", c.n, got, c.want)
		}
	}
}
