package backfill

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

const (
	tenantAID = "0192aaaa-0000-7000-8000-00000000000a"
	fieldAID  = "0192bbbb-0000-7000-8000-0000000000f1"
)

func TestChildIDRoundTrip(t *testing.T) {
	tenantA := uuid.MustParse(tenantAID)
	fieldA := uuid.MustParse(fieldAID)
	for _, v := range []int{0, 1, 32767} {
		id := ChildID(tenantA, fieldindex.BackfillWorkflowType, fieldA, v)
		c, err := parseChildID(id, fieldindex.BackfillWorkflowType)
		if err != nil || c.tenant != tenantA || c.field != fieldA || c.version != v {
			t.Errorf("%s: %+v, %v", id, c, err)
		}
		if _, err := parseChildID(id, fieldindex.RemoveWorkflowType); err == nil {
			t.Errorf("%s parsed as a removal", id)
		}
	}
}

func TestParseChildIDRejects(t *testing.T) {
	tenantA := uuid.MustParse(tenantAID)
	fieldA := uuid.MustParse(fieldAID)
	base := "t:" + tenantA.String() + ":" + fieldindex.BackfillWorkflowType + ":"
	for _, id := range []string{
		"", base, base + fieldA.String(), base + fieldA.String() + ":", base + fieldA.String() + ":-1",
		base + fieldA.String() + ":32768", base + fieldA.String() + ":01", base + fieldA.String() + ":1:2",
		base + upper(fieldA) + ":1", base + uuid.Nil.String() + ":1",
		"t:" + tenantA.String() + ":field_index.coordinator", fieldindex.BackfillWorkflowType + ":" + fieldA.String() + ":1",
	} {
		if c, err := parseChildID(id, fieldindex.BackfillWorkflowType); err == nil {
			t.Errorf("%q parsed as %+v", id, c)
		}
	}
}

// upper is the upper-case form of a UUID: not canonical.
func upper(u uuid.UUID) string {
	b := []byte(u.String())
	for i, c := range b {
		if c >= 'a' && c <= 'f' {
			b[i] = c - 'a' + 'A'
		}
	}
	return string(b)
}

func TestPacer(t *testing.T) {
	var p pacer
	if got := p.next(100 * time.Millisecond); got != InitialPause {
		t.Fatalf("first pause %v, want %v", got, InitialPause)
	}
	// A batch 50% slower than the baseline: contention, back off.
	if got := p.next(150 * time.Millisecond); got != 2*InitialPause {
		t.Fatalf("slow batch: pause %v, want %v", got, 2*InitialPause)
	}
	for range 10 {
		p.next(time.Second)
	}
	if p.pause != MaxPause {
		t.Fatalf("pause %v after sustained contention, want the cap %v", p.pause, MaxPause)
	}
	// Back at the baseline: the pause comes down, never below the floor.
	for range 10 {
		p.next(100 * time.Millisecond)
	}
	if p.pause != InitialPause {
		t.Fatalf("pause %v back at baseline, want the floor %v", p.pause, InitialPause)
	}
	// Within 25% but over 10%: hold.
	p.pause = 2 * time.Second
	if got := p.next(120 * time.Millisecond); got != 2*time.Second {
		t.Fatalf("pause %v at +20%%, want it held", got)
	}
	// A zero duration (no clock movement) changes nothing.
	if got := p.next(0); got != 2*time.Second {
		t.Fatalf("pause %v for a zero duration, want it held", got)
	}
}

func TestProgressPercent(t *testing.T) {
	for _, c := range []struct {
		done, total int64
		want        int16
	}{{0, 0, 0}, {0, 10, 0}, {1, 3, 33}, {10, 50, 20}, {50, 50, 99}, {60, 50, 99}} {
		if got := progressPercent(c.done, c.total); got != c.want {
			t.Errorf("progressPercent(%d, %d) = %d, want %d", c.done, c.total, got, c.want)
		}
	}
}

func TestParseItem(t *testing.T) {
	fieldA := uuid.MustParse(fieldAID)
	ok := fieldindex.EnqueueItem{Op: fieldindex.OpBackfill, FieldID: fieldA.String(), RequestedVersion: 2,
		OutboxID: uuid.Must(uuid.NewV7()).String()}
	if q, _, good := parseItem(ok); !good || q.field != fieldA || q.version != 2 {
		t.Fatalf("parseItem(ok) = %+v, %v", q, good)
	}
	for name, mutate := range map[string]func(*fieldindex.EnqueueItem){
		"op":           func(i *fieldindex.EnqueueItem) { i.Op = "drop" },
		"field":        func(i *fieldindex.EnqueueItem) { i.FieldID = "x" },
		"nil field":    func(i *fieldindex.EnqueueItem) { i.FieldID = uuid.Nil.String() },
		"upper field":  func(i *fieldindex.EnqueueItem) { i.FieldID = upper(fieldA) },
		"outbox":       func(i *fieldindex.EnqueueItem) { i.OutboxID = "" },
		"version 0":    func(i *fieldindex.EnqueueItem) { i.RequestedVersion = 0 },
		"version high": func(i *fieldindex.EnqueueItem) { i.RequestedVersion = 32768 },
	} {
		it := ok
		mutate(&it)
		if _, _, good := parseItem(it); good {
			t.Errorf("%s: accepted %+v", name, it)
		}
	}
}

// QA item 4, statically: every value the workflows record passes the worker
// guard's references-only rule (CI-7), the same check workerguard applies
// (outbox.ValidatePayload on the JSON encoding). The integration tests run
// the real guard over the real runs.
func TestRecordedShapesAreReferencesOnly(t *testing.T) {
	fieldA := uuid.MustParse(fieldAID)
	c := uuid.Must(uuid.NewV7())
	refs, _ := json.Marshal(pendingRefs{Pending: []uuid.UUID{fieldA, c}})
	for name, v := range map[string]any{
		"batch result":          Cursor{Cursor: &c},
		"batch result, done":    Cursor{},
		"batch input, first":    Cursor{},
		"child input":           Input{FieldID: fieldA},
		"child continue-as-new": Input{FieldID: fieldA, Cursor: &c},
		"classify input":        FieldRef{FieldID: fieldA},
		"classify backfill":     Classified{Backfill: &fieldA},
		"classify remove":       Classified{Remove: &fieldA},
		"classify nothing":      Classified{},
		"coordinator refs":      json.RawMessage(refs),
	} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := outbox.ValidatePayload(b); err != nil {
			t.Errorf("%s %s: %v", name, b, err)
		}
	}
	// The continue-as-new input is a RunInput, whose Payload must pass too.
	in := workerguard.RunInput{Delivery: outbox.Delivery{Payload: json.RawMessage(`{}`)}, Refs: refs}
	b, _ := json.Marshal(in)
	var back workerguard.RunInput
	if err := json.Unmarshal(b, &back); err != nil || outbox.ValidatePayload(back.Payload) != nil || outbox.ValidatePayload(back.Refs) != nil {
		t.Errorf("coordinator continue-as-new input %s does not round-trip", b)
	}
}
