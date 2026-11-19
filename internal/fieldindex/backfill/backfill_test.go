package backfill

import (
	"encoding/json"
	"math/rand/v2"
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

// pauses feeds ds to a fresh pacer and returns the pause after each.
func pauses(p *pacer, ds ...time.Duration) []time.Duration {
	out := make([]time.Duration, len(ds))
	for i, d := range ds {
		out[i] = p.next(d)
	}
	return out
}

func flat(d time.Duration, n int) []time.Duration {
	out := make([]time.Duration, n)
	for i := range out {
		out[i] = d
	}
	return out
}

// A noisy but uncontended series, ±30% around a flat mean, stays at the
// floor: the case that ratcheted the old pacer to its cap (TICK-991).
func TestPacerNoiseStaysAtTheFloor(t *testing.T) {
	for seed := range uint64(10) {
		r := rand.New(rand.NewPCG(seed, 991))
		ds := make([]time.Duration, 2000)
		for i := range ds {
			ds[i] = time.Duration(float64(400*time.Millisecond) * (0.7 + 0.6*r.Float64()))
		}
		atFloor := 0
		for _, got := range pauses(&pacer{}, ds...) {
			if got == InitialPause {
				atFloor++
			}
		}
		if atFloor < len(ds)*99/100 {
			t.Errorf("seed %d: %d of %d pauses at the floor, want at least 99%%", seed, atFloor, len(ds))
		}
	}
}

// Slow drift, +50% over 300 batches with no contention, is followed, not
// treated as contention.
func TestPacerFollowsDrift(t *testing.T) {
	ds := make([]time.Duration, 300)
	for i := range ds {
		ds[i] = 400*time.Millisecond + time.Duration(i)*200*time.Millisecond/300
	}
	for i, got := range pauses(&pacer{}, ds...) {
		if got != InitialPause {
			t.Fatalf("batch %d: pause %v under slow drift, want the floor", i, got)
		}
	}
}

// Contention doubles the batch: the pause backs off to the cap; when it
// ends, the pause comes back to the floor.
func TestPacerBacksOffAndRecovers(t *testing.T) {
	var p pacer
	pauses(&p, flat(400*time.Millisecond, 50)...)
	if p.pause != InitialPause {
		t.Fatalf("pause %v before contention, want the floor", p.pause)
	}
	got := pauses(&p, flat(800*time.Millisecond, 15)...)
	if got[1] != InitialPause {
		t.Errorf("pause %v at the second contended batch, want the floor: two slow batches are not contention", got[1])
	}
	if got[len(got)-1] != MaxPause {
		t.Errorf("pauses %v under contention, want them to reach the cap %v", got, MaxPause)
	}
	got = pauses(&p, flat(400*time.Millisecond, 10)...)
	if got[len(got)-1] != InitialPause {
		t.Errorf("pauses %v after contention ended, want the floor", got)
	}
	// A zero duration (no clock movement) changes nothing.
	p.pause = 2 * time.Second
	if got := p.next(0); got != 2*time.Second {
		t.Fatalf("pause %v for a zero duration, want it held", got)
	}
}

// The pause never leaves [InitialPause, MaxPause], and the cap is well below
// the 10 s that made a 400k-record field take an hour.
func TestPacerBounds(t *testing.T) {
	if MaxPause > 5*time.Second {
		t.Fatalf("MaxPause %v, want at most 5 s (TICK-991)", MaxPause)
	}
	var p pacer
	for i, got := range pauses(&p, append(flat(time.Millisecond, 10), flat(time.Hour, 100)...)...) {
		if got < InitialPause || got > MaxPause {
			t.Fatalf("batch %d: pause %v out of bounds", i, got)
		}
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
