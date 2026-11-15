package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
)

func validEntry() Entry {
	return Entry{
		Kind: "ticket.approval", KindVersion: 1, DedupeKey: "d1", RecordID: uuid.New(),
		Target: TargetStart{Kind: "ticket.approval"}, Causation: Causation{OriginID: uuid.New()},
	}
}

// Every case is rejected before Enqueue touches tx, so a nil tx proves
// nothing was written.
func TestEnqueueRejectsBeforeAnyWrite(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Entry)
		want   error
	}{
		{"CI-10 depth beyond limit", func(e *Entry) { e.Causation = Causation{OriginID: uuid.New(), Depth: 9} }, ErrCascadeTooDeep},
		{"negative depth", func(e *Entry) { e.Causation.Depth = -1 }, ErrInvalidEntry},
		{"depth with no origin", func(e *Entry) { e.Causation = Causation{Depth: 1} }, ErrInvalidEntry},
		// A5-3: depth 0 too, or a row could slip past the start key.
		{"depth 0 with no origin", func(e *Entry) { e.Causation = Causation{} }, ErrInvalidEntry},
		// A5-6: an interpreter start goes through EnqueueStart only.
		{"interpreter start", func(e *Entry) {
			e.Kind, e.Target = InterpreterKind, TargetStart{Kind: InterpreterKind}
		}, ErrInvalidEntry},
		{"empty kind", func(e *Entry) { e.Kind = ""; e.Target = TargetStart{} }, ErrInvalidEntry},
		{"free-text kind", func(e *Entry) { e.Kind = "Jane Doe"; e.Target = TargetStart{Kind: "Jane Doe"} }, ErrInvalidEntry},
		{"kind with a colon", func(e *Entry) { e.Kind = "a:b"; e.Target = TargetStart{Kind: "a:b"} }, ErrInvalidEntry},
		{"kind version 0", func(e *Entry) { e.KindVersion = 0 }, ErrInvalidEntry},
		{"empty dedupe key", func(e *Entry) { e.DedupeKey = "" }, ErrInvalidEntry},
		{"long dedupe key", func(e *Entry) { e.DedupeKey = strings.Repeat("k", 201) }, ErrInvalidEntry},
		{"no record", func(e *Entry) { e.RecordID = uuid.Nil }, ErrInvalidEntry},
		{"no target", func(e *Entry) { e.Target = nil }, ErrInvalidEntry},
		{"start kind differs", func(e *Entry) { e.Target = TargetStart{Kind: "other"} }, ErrInvalidEntry},
		{"signal name differs", func(e *Entry) { e.Target = TargetSignal{Kind: "ticket.approval", SignalName: "approve"} }, ErrInvalidEntry},
		{"signal kind free text", func(e *Entry) { e.Kind = "approve"; e.Target = TargetSignal{Kind: "A B", SignalName: "approve"} }, ErrInvalidEntry},
		{"record data in payload", func(e *Entry) { e.Payload = json.RawMessage(`{"title":"printer on fire"}`) }, ErrInvalidEntry},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := validEntry()
			c.mutate(&e)
			id, err := EnqueueAtVersion(context.Background(), nil, e, 1)
			if !errors.Is(err, c.want) || id != uuid.Nil {
				t.Fatalf("EnqueueAtVersion = %v, %v; want %v", id, err, c.want)
			}
		})
	}
}

// EnqueueAtVersion's interpreter check and EnqueueStart refuse before any write too.
func TestEnqueueStartPathsRefuseBeforeAnyWrite(t *testing.T) {
	interp := validEntry()
	interp.Kind, interp.Target = InterpreterKind, TargetStart{Kind: InterpreterKind}
	if _, err := EnqueueAtVersion(context.Background(), nil, interp, 1); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("EnqueueAtVersion of an interpreter start = %v, want ErrInvalidEntry", err)
	}
	// Only cascade.ClaimStart builds a claimed Claim; the zero one is not.
	if _, err := EnqueueStart(context.Background(), nil, cascade.Claim{}, interp); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("EnqueueStart with no claim = %v, want ErrInvalidEntry", err)
	}
}

// A seal start has its own entry point (A6.3): EnqueueAtVersion refuses one,
// and EnqueueSeal refuses a head version below 1, both before any query (tx
// is nil, so a query would panic).
func TestSealPathsRefuseBeforeAnyWrite(t *testing.T) {
	seal := validEntry()
	seal.Kind, seal.Target = SealKind, TargetStart{Kind: SealKind}
	if _, err := EnqueueAtVersion(context.Background(), nil, seal, 1); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("EnqueueAtVersion of a seal start = %v, want ErrInvalidEntry", err)
	}
	// The seal dedupe namespace is EnqueueSeal's alone.
	reserved := validEntry()
	reserved.DedupeKey = SealDedupeKey(1)
	if _, err := EnqueueAtVersion(context.Background(), nil, reserved, 1); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("EnqueueAtVersion with a seal dedupe key = %v, want ErrInvalidEntry", err)
	}
	for _, v := range []int64{0, -1} {
		if _, err := EnqueueSeal(context.Background(), nil, v); !errors.Is(err, ErrInvalidEntry) {
			t.Fatalf("EnqueueSeal(%d) = %v, want ErrInvalidEntry", v, err)
		}
	}
	// C5 as amended (TICK-440): a negative delay is refused before any query;
	// the last WithDelay wins, so a later negative one is refused too.
	for _, opts := range [][]SealOption{
		{WithDelay(-time.Nanosecond)},
		{WithDelay(10 * time.Second), WithDelay(-time.Second)},
	} {
		if _, err := EnqueueSeal(context.Background(), nil, 1, opts...); !errors.Is(err, ErrInvalidEntry) {
			t.Fatalf("EnqueueSeal with a negative delay = %v, want ErrInvalidEntry", err)
		}
	}
}

func TestSealDedupeKey(t *testing.T) {
	if got := SealDedupeKey(42); got != "audit.seal:42" {
		t.Fatalf("SealDedupeKey(42) = %q", got)
	}
	if !ValidKind(SealKind) {
		t.Fatal("SealKind is not a valid kind, so the relay would mark its rows dead")
	}
}

func TestDepthAtLimitPassesValidation(t *testing.T) {
	e := validEntry()
	e.Causation = Causation{OriginID: uuid.New(), Depth: 8}
	if _, err := validate(e); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePayload(t *testing.T) {
	id := uuid.NewString()
	for _, ok := range []string{`{}`, `{"ticket_id":"` + id + `"}`, `{"ids":["` + id + `","` + id + `"]}`, `{"ids":[]}`} {
		if err := ValidatePayload(json.RawMessage(ok)); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{
		`null`, `[]`, `"x"`, `{"n":1}`, `{"b":true}`, `{"s":"text"}`, `{"o":{"id":"` + id + `"}}`,
		`{"ids":["` + id + `","x"]}`, `{"z":null}`, `{`,
		// Keys (TICK-240).
		`{"printer on fire":"` + id + `"}`, `{"Ticket_id":"` + id + `"}`, `{"ticket.id":"` + id + `"}`,
		`{"1d":"` + id + `"}`, `{"":"` + id + `"}`, `{"` + strings.Repeat("a", 33) + `":"` + id + `"}`,
		// Non-canonical values (TICK-240).
		`{"ticket_id":"` + strings.ToUpper(id) + `"}`, `{"ticket_id":"urn:uuid:` + id + `"}`,
		`{"ticket_id":"{` + id + `}"}`, `{"ticket_id":"` + strings.ReplaceAll(id, "-", "") + `"}`,
		`{"ids":["` + id + `","` + strings.ToUpper(id) + `"]}`,
	} {
		if err := ValidatePayload(json.RawMessage(bad)); !errors.Is(err, ErrInvalidEntry) {
			t.Errorf("%s: %v, want ErrInvalidEntry", bad, err)
		}
	}
	if err := ValidatePayload(json.RawMessage(`{"` + strings.Repeat("a", 32) + `":"` + id + `"}`)); err != nil {
		t.Errorf("32-character key: %v", err)
	}
}

// A refusal never quotes the payload, not even the key (TICK-240).
func TestValidatePayloadErrorOmitsPayload(t *testing.T) {
	const marker = "printer on fire alice"
	for _, p := range []string{
		`{"` + marker + `":"` + uuid.NewString() + `"}`,
		`{"note":"` + marker + `"}`,
		`{"ids":["` + marker + `"]}`,
	} {
		err := ValidatePayload(json.RawMessage(p))
		if err == nil {
			t.Fatalf("%s accepted", p)
		}
		if strings.Contains(err.Error(), "printer") {
			t.Errorf("error %q quotes the payload", err)
		}
	}
}

func TestTraceparentOnlyTraceContext(t *testing.T) {
	if tp := traceparent(context.Background()); tp != nil {
		t.Fatalf("no span: %q, want nil", *tp)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3},
		SpanID:     trace.SpanID{4, 5, 6},
		TraceFlags: trace.FlagsSampled,
		TraceState: mustTraceState(t, "vendor=secret"),
	})
	member, err := baggage.NewMember("user", "jane@example.com")
	if err != nil {
		t.Fatal(err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatal(err)
	}
	ctx := baggage.ContextWithBaggage(trace.ContextWithSpanContext(context.Background(), sc), bag)
	tp := traceparent(ctx)
	want := "00-01020300000000000000000000000000-0405060000000000-01"
	if tp == nil || *tp != want {
		t.Fatalf("traceparent = %v, want %s (no tracestate, no baggage)", tp, want)
	}
}

func mustTraceState(t *testing.T, s string) trace.TraceState {
	t.Helper()
	ts, err := trace.ParseTraceState(s)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

// insertColumns returns the column list of the INSERT INTO outbox in sql.
func insertColumns(t *testing.T, sql string) []string {
	t.Helper()
	m := regexp.MustCompile(`INSERT INTO outbox \(([^)]*)\)`).FindStringSubmatch(sql)
	if m == nil {
		t.Fatalf("no INSERT INTO outbox in %q", sql)
	}
	var cols []string
	for _, c := range strings.Split(m[1], ",") {
		cols = append(cols, strings.TrimSpace(c))
	}
	return cols
}

// The delayed seal insert (TICK-440) names insertSealSQL's columns plus
// available_at and nothing else, and is the same statement otherwise: same
// values for those columns, same conflict handling. insertSealSQL keeps
// P5-31's column list, which onboarding (P5-33) and the nava_onboard grant
// equal (TestSealInsertMatchesOutbox).
func TestDelayedSealInsertColumns(t *testing.T) {
	base := insertColumns(t, insertSealSQL)
	if got, want := insertColumns(t, insertDelayedSealSQL), append(slices.Clone(base), "available_at"); !slices.Equal(got, want) {
		t.Errorf("delayed seal insert columns %v, want %v", got, want)
	}
	if slices.Contains(base, "available_at") {
		t.Errorf("insertSealSQL names available_at; the no-option path must take its default")
	}
	want := strings.Replace(insertSealSQL, "trace_context)", "trace_context, available_at)", 1)
	want = strings.Replace(want, "$6)", "$6, now() + make_interval(secs => $7))", 1)
	if insertDelayedSealSQL != want {
		t.Errorf("insertDelayedSealSQL differs from insertSealSQL by more than available_at: %s", insertDelayedSealSQL)
	}
}

// EnqueueReseed refuses before any query (tx is nil) on a bad version or a nil
// reseed id; its dedupe key sits in the seal namespace, so EnqueueAtVersion
// still refuses it, and it differs from the plain seal key and between reseeds.
func TestReseedRefusesBeforeAnyWrite(t *testing.T) {
	if _, err := EnqueueReseed(context.Background(), nil, 0, uuid.New()); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("EnqueueReseed(0) = %v, want ErrInvalidEntry", err)
	}
	if _, err := EnqueueReseed(context.Background(), nil, 1, uuid.Nil); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("EnqueueReseed(nil id) = %v, want ErrInvalidEntry", err)
	}
	a, b := uuid.New(), uuid.New()
	if k := ReseedDedupeKey(3, a); k == SealDedupeKey(3) || k == ReseedDedupeKey(3, b) || k != ReseedDedupeKey(3, a) {
		t.Fatalf("reseed dedupe key %q is not distinct and stable", k)
	}
	reserved := validEntry()
	reserved.DedupeKey = ReseedDedupeKey(1, a)
	if _, err := EnqueueAtVersion(context.Background(), nil, reserved, 1); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("EnqueueAtVersion with a reseed dedupe key = %v, want ErrInvalidEntry", err)
	}
}

// AR-041: a row of a listed kind carries exactly its allow-list. Every
// payload here is references only, so only the per-kind rule refuses it.
func TestValidateKindPayload(t *testing.T) {
	ref := `"` + uuid.NewString() + `"`
	cases := []struct {
		kind, payload string
		ok            bool
	}{
		{InterpreterKind, `{"definition_version_id":` + ref + `,"record_id":` + ref + `}`, true},
		{InterpreterKind, `{"definition_version_id":` + ref + `,"record_id":` + ref + `,"extra":` + ref + `}`, false},
		{InterpreterKind, `{"record_id":` + ref + `}`, false},
		{InterpreterKind, `{"definition_version_id":` + ref + `,"record":` + ref + `}`, false},
		{InterpreterSignalKind, `{"signal":` + ref + `}`, true},
		{InterpreterSignalKind, `{"signal":` + ref + `,"record_id":` + ref + `}`, false},
		{InterpreterSignalKind, `{}`, false},
		{SealKind, `{}`, true},
		{SealKind, `{"record_id":` + ref + `}`, false},
		// A kind with no allow-list keeps the pattern rule only.
		{"ticket.approval", `{"anything":` + ref + `}`, true},
		// The references-only rule still applies first.
		{InterpreterSignalKind, `{"signal":"not a uuid"}`, false},
	}
	for _, c := range cases {
		err := ValidateKindPayload(c.kind, json.RawMessage(c.payload))
		if (err == nil) != c.ok || (err != nil && !errors.Is(err, ErrInvalidEntry)) {
			t.Errorf("ValidateKindPayload(%s, %s) = %v, want ok %v", c.kind, c.payload, err, c.ok)
		}
	}
}

// interpSignal is a valid record_changed signal entry.
func interpSignal() Entry {
	return Entry{Kind: InterpreterSignalKind, KindVersion: 1, DedupeKey: "s1", RecordID: uuid.New(),
		Target:    TargetSignal{Kind: InterpreterKind, SignalName: InterpreterSignalKind},
		Causation: Causation{OriginID: uuid.New(), Depth: 1}, Payload: json.RawMessage(`{"signal":"` + uuid.NewString() + `"}`)}
}

// Enqueueing with a key off the allow-list fails for each kind, before any
// query (tx is nil, so a query would panic). EnqueueAtVersion refuses an
// interpreter signal, which must go through EnqueueSignal (D-3-29), and
// EnqueueSignal refuses any other signal and a missing ref.
func TestInterpreterEntriesRefusedBeforeAnyWrite(t *testing.T) {
	ctx := context.Background()
	ref := uuid.New()

	extra := interpSignal()
	extra.Payload = json.RawMessage(`{"signal":"` + uuid.NewString() + `","record_id":"` + uuid.NewString() + `"}`)
	if _, err := EnqueueSignal(ctx, nil, ref, extra, 1); !errors.Is(err, ErrInvalidEntry) {
		t.Errorf("EnqueueSignal with an extra key = %v, want ErrInvalidEntry", err)
	}
	if _, err := EnqueueAtVersion(ctx, nil, interpSignal(), 1); !errors.Is(err, ErrInvalidEntry) {
		t.Errorf("EnqueueAtVersion of an interpreter signal = %v, want ErrInvalidEntry", err)
	}
	if _, err := EnqueueSignal(ctx, nil, uuid.Nil, interpSignal(), 1); !errors.Is(err, ErrInvalidEntry) {
		t.Errorf("EnqueueSignal with no ref = %v, want ErrInvalidEntry", err)
	}
	other := interpSignal()
	other.Kind, other.Target = "approve", TargetSignal{Kind: InterpreterKind, SignalName: "approve"}
	if _, err := EnqueueSignal(ctx, nil, ref, other, 1); !errors.Is(err, ErrInvalidEntry) {
		t.Errorf("EnqueueSignal of another signal name = %v, want ErrInvalidEntry", err)
	}
	toOther := interpSignal()
	toOther.Target = TargetSignal{Kind: "ticket.approval", SignalName: InterpreterSignalKind}
	if _, err := EnqueueSignal(ctx, nil, ref, toOther, 1); !errors.Is(err, ErrInvalidEntry) {
		t.Errorf("EnqueueSignal to another workflow type = %v, want ErrInvalidEntry", err)
	}
}
