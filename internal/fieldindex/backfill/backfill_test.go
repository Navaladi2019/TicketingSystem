package backfill

import (
	"encoding/json"
	"testing"

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

// A batch reuses the counts only when they are at its own cursor: the first
// batch, a retried batch (its cursor is behind the memo) and a batch on
// another worker count the type.
func TestProgressCountsAfter(t *testing.T) {
	l1, l2, l3 := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	known := &progressCounts{last: l2, done: 20, total: 50}
	if got, ok := known.after(&l2, l3, 10); !ok || got != (progressCounts{last: l3, done: 30, total: 50}) {
		t.Errorf("after the memo's cursor: %+v, %v", got, ok)
	}
	for name, c := range map[string]struct {
		known  *progressCounts
		cursor *uuid.UUID
	}{"no memo": {nil, &l2}, "first batch": {known, nil}, "retried batch": {known, &l1}} {
		if _, ok := c.known.after(c.cursor, l3, 10); ok {
			t.Errorf("%s: reused the counts", name)
		}
	}
}

func TestProgressMemo(t *testing.T) {
	var m progressMemo
	a := child{tenant: uuid.MustParse(tenantAID), field: uuid.MustParse(fieldAID), version: 1}
	b := a
	b.version = 2
	last := uuid.Must(uuid.NewV7())
	pc := progressCounts{last: last, done: 10, total: 50}
	if m.get(a) != nil {
		t.Fatal("empty memo returned counts")
	}
	m.put(a, batchResult{next: &last, progress: &pc})
	if got := m.get(a); got == nil || *got != pc {
		t.Fatalf("get after put: %+v", got)
	}
	if m.get(b) != nil {
		t.Fatal("another version of the field read the counts")
	}
	m.put(a, batchResult{progress: &pc}) // the last batch: no cursor
	if m.get(a) != nil {
		t.Fatal("a finished child kept its entry")
	}
	for i := range progressMemoMax + 1 {
		c := a
		c.version = i + 1
		m.put(c, batchResult{next: &last, progress: &pc})
	}
	if n := len(m.m); n > progressMemoMax {
		t.Fatalf("memo holds %d entries, want at most %d", n, progressMemoMax)
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
