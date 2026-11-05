// Package outbox is the producer side of the transactional outbox (ADR-001
// I6, I9, Annex A.2 "Writing outbox rows"; PLAN-001 T29). EnqueueAtVersion and
// EnqueueStart write one outbox row in the caller's tenant transaction; the
// relay (T30a, T30b) is the only reader that dispatches it to Temporal.
//
// The package never writes the records table (PLAN-003 D-3-6, TICK-411): the
// record write pipeline's UPDATE is the one place records.version moves, and
// a producer enqueues at the version that write gave the record.
//
// The tenant is never an argument: every Enqueue function runs in a
// transaction opened by tenancy.InTenantTx and takes tenant_id from the row
// it locks (the records row, or for EnqueueSeal the tenant's
// audit_chain_head row), which it can see only for that transaction's
// tenant.
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/propagation"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

var (
	// ErrCascadeTooDeep means the entry's causation depth is beyond
	// config.CascadeDepthLimit (I9). Nothing is written.
	ErrCascadeTooDeep = errors.New("outbox: cascade too deep")
	// ErrNoOpenWorkflow means a signal entry found no open workflow_ref of its kind
	// on the record.
	ErrNoOpenWorkflow = errors.New("outbox: no open workflow for signal")
	// ErrRecordNotFound means the transaction's tenant has no such record, or the
	// transaction has no tenant.
	ErrRecordNotFound = errors.New("outbox: record not found")
	// ErrInvalidEntry means the entry breaks the contract below. Nothing is written.
	ErrInvalidEntry = errors.New("outbox: invalid entry")
	// ErrVersionMismatch means EnqueueAtVersion was given a version the
	// record does not have in this transaction. Nothing is written.
	ErrVersionMismatch = errors.New("outbox: record version mismatch")
	// ErrHeadNotFound means the transaction's tenant has no audit_chain_head
	// row, or the transaction has no tenant. EnqueueSeal writes nothing.
	ErrHeadNotFound = errors.New("outbox: audit chain head not found")
)

// Target is where the relay delivers a row. It is sealed: only TargetStart
// and TargetSignal implement it.
type Target interface{ isTarget() }

// TargetStart starts a new workflow of Kind (the Temporal workflow type).
type TargetStart struct{ Kind string }

// TargetSignal signals the newest open workflow of Kind on the entry's
// record, found through workflow_ref.
type TargetSignal struct{ Kind, SignalName string }

func (TargetStart) isTarget()  {}
func (TargetSignal) isTarget() {}

// Causation is the cascade record every row carries (I9): the event that
// started the cascade, and how many writes deep this one is. Every row has an
// origin, depth 0 included: a depth-0 write mints a new one (ADR-001 A5-3),
// so a row with no origin cannot slip past the start key.
type Causation struct {
	OriginID uuid.UUID
	Depth    int16
}

// Entry is one outbox row to write.
//
// Kind is the name the relay dispatches to and is stored as outbox.kind: the
// workflow type for a start (it must equal TargetStart.Kind) and the signal
// name for a signal (it must equal TargetSignal.SignalName). The relay tells
// a start from a signal by target_key: only a start row's ends in its own id.
type Entry struct {
	Kind        string
	KindVersion int
	DedupeKey   string
	RecordID    uuid.UUID
	Target      Target
	Causation   Causation
	// Payload holds references only (CI-7): see ValidatePayload. Nil is {}.
	Payload json.RawMessage
}

// kindPattern bounds every kind and signal name, so a target_key
// (<kind>:<uuid>) can carry no free text and so no PII (TICK-94 item 3).
var kindPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,99}$`)

// ValidKind reports whether s may be a kind, a target kind or a signal name.
// The relay checks every row it dispatches with it.
func ValidKind(s string) bool { return kindPattern.MatchString(s) }

// InterpreterKind is the workflow type of the workflow interpreter. Its
// starts go through EnqueueStart only, under a cascade.Claim (A5-6).
const InterpreterKind = "nava.interpreter"

// SealKind is the workflow type of the audit sealer (ADR-004 R16). Its
// starts are not tied to a record and go through EnqueueSeal only
// (ADR-001 A6.3).
const SealKind = "audit.seal"

// Delivery is the single argument the relay passes to Temporal for one row:
// the input of a started workflow, or the body of a signal. Like the row it
// comes from, it holds references only (CI-7). OutboxID is the id the
// workflow's signal handler dedupes on (I7, workerguard.AppliedSignals).
type Delivery struct {
	OutboxID       uuid.UUID       `json:"outbox_id"`
	KindVersion    int16           `json:"kind_version"`
	CausationID    *uuid.UUID      `json:"causation_id,omitempty"`
	CausationDepth int16           `json:"causation_depth"`
	Payload        json.RawMessage `json:"payload"`
}

const maxDedupeKeyLen = 200

const (
	// A duplicate retried after commit returns here, before the record lock;
	// the INSERT below catches the concurrent duplicate.
	dedupeLookupSQL = `SELECT id FROM outbox WHERE dedupe_key = $1`
	// The record lock. The write pipeline has already bumped the version in
	// this transaction (D-3-6), and its UPDATE holds the row lock that orders
	// a target's rows (A.2), so this only locks and reads.
	lockRecordSQL = `SELECT tenant_id, version FROM records WHERE id = $1 FOR UPDATE`
	openRefSQL    = `SELECT outbox_id FROM workflow_ref
WHERE tenant_id = $1 AND record_id = $2 AND kind = $3 AND closed_at IS NULL
ORDER BY created_at DESC, outbox_id DESC LIMIT 1`
	insertSQL = `INSERT INTO outbox (id, tenant_id, kind, kind_version, dedupe_key, target_key, target_seq,
    causation_id, causation_depth, payload, trace_context)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (tenant_id, dedupe_key) DO NOTHING
RETURNING id`
	insertRefSQL = `INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ($1, $2, $3, $4)`
)

// EnqueueAtVersion writes e as an outbox row in tx and returns the row's id.
// The writer has already bumped the record's version in tx: the record write
// pipeline, whose UPDATE is the one place a write bumps it (PLAN-003 D-3-6).
// It locks the records row without bumping it again and uses version as the
// row's target_seq. version must be the record's version in tx, or it
// returns ErrVersionMismatch and writes nothing.
//
// A start row's target_key is <kind>:<its own id>, and its workflow_ref row
// is inserted in tx. A signal row takes the target_key of the newest open
// workflow_ref of its kind on the record, or fails with ErrNoOpenWorkflow.
//
// If the tenant already has a row with e.DedupeKey, it returns that row's id
// and a nil error and writes nothing, before it checks version: a nil error
// does not mean a row was written. Rows one write enqueues for the
// same record share a target_seq. A start's target_key is its own, so that
// matters only for two signals to one workflow from one write: the relay's
// head test (target_seq <= the lowest pending) admits both, so their order is
// not fixed. Rows of later writes still follow them.
//
// The W3C traceparent of ctx's span, if any, is stored in trace_context so the
// relay can link the workflow to the request that wrote the row. tracestate
// and baggage never cross.
//
// tx must come from tenancy.InTenantTx. On any error the caller rolls tx back.
//
// A start of InterpreterKind is ErrInvalidEntry: use EnqueueStart.
func EnqueueAtVersion(ctx context.Context, tx pgx.Tx, e Entry, version int64) (uuid.UUID, error) {
	if err := notOwnEntryPointStart(e); err != nil {
		return uuid.Nil, err
	}
	return enqueue(ctx, tx, e, uuid.Nil, func() (uuid.UUID, int64, error) {
		tenant, got, err := lockRecord(ctx, tx, lockRecordSQL, e.RecordID)
		if err != nil {
			return uuid.Nil, 0, err
		}
		if got != version {
			return uuid.Nil, 0, fmt.Errorf("%w: have %d, given %d", ErrVersionMismatch, got, version)
		}
		return tenant, got, nil
	})
}

// EnqueueStart is the only way to enqueue a start of InterpreterKind (ADR-001
// A5-6). c must be claimed (cascade.ClaimStart returned claimed = true) in
// tx; the row takes c.OutboxID() as its id, so the cascade_start row names
// it. e must be a start of InterpreterKind on c's record.
//
// The row's causation is the one stored on the record by the triggering
// write, read by ClaimStart under the tenant's RLS: e.Causation must be
// empty or equal to it. The depth limit applies as for EnqueueAtVersion.
//
// It locks the records row without bumping its version: the triggering
// write has already bumped it (D-3-6), and a start's target_key is its own,
// so its target_seq orders nothing. A dedupe key already in use is
// ErrInvalidEntry, because the claim's id would then name no row.
//
// tx must come from tenancy.InTenantTx. On any error the caller rolls tx back.
func EnqueueStart(ctx context.Context, tx pgx.Tx, c cascade.Claim, e Entry) (uuid.UUID, error) {
	invalid := func(format string, args ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrInvalidEntry}, args...)...)
	}
	if !c.Claimed() {
		return uuid.Nil, invalid("the start key is not claimed")
	}
	if t, ok := e.Target.(TargetStart); !ok || t.Kind != InterpreterKind || e.Kind != InterpreterKind {
		return uuid.Nil, invalid("EnqueueStart enqueues only a start of %s", InterpreterKind)
	}
	if e.RecordID != c.RecordID() {
		return uuid.Nil, invalid("the entry's record is not the claim's")
	}
	origin, depth := c.Causation()
	claimed := Causation{OriginID: origin, Depth: depth}
	if e.Causation != (Causation{}) && e.Causation != claimed {
		return uuid.Nil, invalid("the entry's causation is not the one stored on the record")
	}
	e.Causation = claimed
	id, err := enqueue(ctx, tx, e, c.OutboxID(), func() (uuid.UUID, int64, error) {
		tenant, seq, err := lockRecord(ctx, tx, lockRecordSQL, e.RecordID)
		if err == nil && tenant != c.TenantID() {
			return uuid.Nil, 0, ErrRecordNotFound
		}
		return tenant, seq, err
	})
	if err == nil && id != c.OutboxID() {
		return uuid.Nil, invalid("dedupe key already used by another row")
	}
	return id, err
}

// notOwnEntryPointStart refuses, in EnqueueAtVersion, the starts that have
// their own entry point: InterpreterKind (EnqueueStart) and SealKind
// (EnqueueSeal).
func notOwnEntryPointStart(e Entry) error {
	t, ok := e.Target.(TargetStart)
	if !ok {
		return nil
	}
	for _, k := range [...]struct{ kind, via string }{{InterpreterKind, "EnqueueStart"}, {SealKind, "EnqueueSeal"}} {
		if t.Kind == k.kind || e.Kind == k.kind {
			return fmt.Errorf("%w: a start of %s goes through %s", ErrInvalidEntry, k.kind, k.via)
		}
	}
	return nil
}

const (
	// The head lock (ADR-001 A6.3, ADR-004 R15). The predicate names the
	// transaction's tenant as well as RLS doing so, so a role that bypasses
	// RLS still locks only its own tenant's head. Taking a lock the caller
	// already holds is a no-op, so this is never a second lock.
	lockHeadSQL = `SELECT tenant_id, version FROM audit_chain_head WHERE tenant_id = $1 FOR UPDATE`
	// causation_id is the row's own id at depth 0: a seal run starts no
	// cascade and carries none forward (I9, A5-3).
	// available_at is computed on the database clock (now() is the
	// transaction start), never the worker's: $7 is the delay in seconds,
	// 0 when no WithDelay was given, which is the column default now().
	insertSealSQL = `INSERT INTO outbox (id, tenant_id, kind, kind_version, dedupe_key, target_key, target_seq,
    causation_id, causation_depth, payload, trace_context, available_at)
VALUES ($1, $2, '` + SealKind + `', 1, $3, $4, $5, $1, 0, '{}', $6, now() + make_interval(secs => $7))
ON CONFLICT (tenant_id, dedupe_key) DO NOTHING
RETURNING id`
	sealDedupeSQL = `SELECT id FROM outbox WHERE tenant_id = $1 AND dedupe_key = $2`
)

// SealOption tunes one EnqueueSeal call (PLAN-005 C5, as amended on
// TICK-440). The zero set of options is the original C5 behaviour.
type SealOption func(*sealOptions)

type sealOptions struct {
	delay time.Duration
}

// WithDelay makes the seal row available d after the transaction's start on
// the database clock: available_at = now() + d (ADR-004 R16, A1's
// seal_interval_s). A negative d makes EnqueueSeal return ErrInvalidEntry.
// Without it the row is available at once, as onboarding (P5-33) needs.
func WithDelay(d time.Duration) SealOption {
	return func(o *sealOptions) { o.delay = d }
}

// SealDedupeKey is the dedupe_key of the seal start row enqueued at
// headVersion (PLAN-005 C5).
func SealDedupeKey(headVersion int64) string {
	return fmt.Sprintf("%s%d", sealDedupePrefix, headVersion)
}

// sealDedupePrefix starts every seal row's dedupe key, and no other row's.
const sealDedupePrefix = SealKind + ":"

// EnqueueSeal writes the start row of one audit sealer run in tx and returns
// its id (PLAN-005 C5, ADR-001 A6.3). The row is not tied to a record:
//
//   - tenant_id is the tenant of tx (tenancy.TenantOf), never an argument;
//   - kind = SealKind, kind_version 1, payload {} (references only);
//   - target_key = audit.seal:<own id>, so the relay starts it as
//     t:<tenant>:audit.seal:<id> with REJECT_DUPLICATE;
//   - target_seq = headVersion, the tenant's audit_chain_head.version;
//   - dedupe_key = SealDedupeKey(headVersion);
//   - causation is the row's own id at depth 0;
//   - available_at = now() + the WithDelay duration, or now() without it;
//   - no workflow_ref row: workflow_ref.record_id references records, and a
//     seal run is never signalled.
//
// The caller holds the tenant's head row FOR UPDATE in tx and passes the
// version it read there (A1 after bumping it, or chain:reseed reusing it).
// EnqueueSeal locks the head itself before it writes, so a row is never
// written without the head lock held in tx; if headVersion is not the
// head's version under that lock it returns ErrVersionMismatch and writes
// nothing. A tenant with no head, or tx with no tenant, is ErrHeadNotFound.
// The head is locked by the tenant of tx, and the insert runs under the
// outbox RLS WITH CHECK, which refuses a row whose tenant_id is not the
// transaction's.
//
// If the tenant already has a row with this dedupe key, it returns that
// row's id and a nil error and writes nothing, whatever delay it was given.
// Only a seal row can hold it: every other producer refuses a dedupe key with
// the SealKind prefix.
//
// tx must come from tenancy.InTenantTx. On any error the caller rolls tx back.
func EnqueueSeal(ctx context.Context, tx pgx.Tx, headVersion int64, opts ...SealOption) (uuid.UUID, error) {
	if headVersion < 1 {
		return uuid.Nil, fmt.Errorf("%w: head version %d is below 1", ErrInvalidEntry, headVersion)
	}
	var o sealOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	if o.delay < 0 {
		return uuid.Nil, fmt.Errorf("%w: negative seal delay", ErrInvalidEntry)
	}
	txTenant, err := tenancy.TenantOf(ctx, tx)
	if errors.Is(err, tenancy.ErrNoTenant) {
		return uuid.Nil, ErrHeadNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("outbox: %w", err)
	}
	var (
		tenant uuid.UUID
		got    int64
	)
	switch err := tx.QueryRow(ctx, lockHeadSQL, txTenant).Scan(&tenant, &got); {
	case errors.Is(err, pgx.ErrNoRows):
		return uuid.Nil, ErrHeadNotFound
	case err != nil:
		return uuid.Nil, fmt.Errorf("outbox: lock audit chain head: %w", err)
	}
	if got != headVersion {
		return uuid.Nil, fmt.Errorf("%w: head has %d, given %d", ErrVersionMismatch, got, headVersion)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("outbox: new id: %w", err)
	}
	// RLS WITH CHECK refuses a row whose tenant_id is not app.tenant_id.
	dedupe := SealDedupeKey(headVersion)
	var inserted uuid.UUID
	err = tx.QueryRow(ctx, insertSealSQL, id, tenant, dedupe, SealKind+":"+id.String(), headVersion,
		traceparent(ctx), o.delay.Seconds()).Scan(&inserted)
	if err == nil {
		return inserted, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("outbox: insert seal row: %w", err)
	}
	var existing uuid.UUID
	if err := tx.QueryRow(ctx, sealDedupeSQL, tenant, dedupe).Scan(&existing); err != nil {
		return uuid.Nil, fmt.Errorf("outbox: dedupe lookup after conflict: %w", err)
	}
	return existing, nil
}

// lockRecord runs the record lock and returns the record's tenant and the
// version that becomes target_seq.
func lockRecord(ctx context.Context, tx pgx.Tx, sql string, record uuid.UUID) (uuid.UUID, int64, error) {
	var (
		tenant uuid.UUID
		seq    int64
	)
	switch err := tx.QueryRow(ctx, sql, record).Scan(&tenant, &seq); {
	case errors.Is(err, pgx.ErrNoRows):
		return uuid.Nil, 0, ErrRecordNotFound
	case err != nil:
		return uuid.Nil, 0, fmt.Errorf("outbox: lock record: %w", err)
	}
	return tenant, seq, nil
}

// enqueue is EnqueueAtVersion and EnqueueStart; lock locks the
// record and returns its tenant and the row's target_seq. id is the row's id,
// or uuid.Nil to mint a UUIDv7.
func enqueue(ctx context.Context, tx pgx.Tx, e Entry, id uuid.UUID, lock func() (uuid.UUID, int64, error)) (uuid.UUID, error) {
	if e.Causation.Depth > config.CascadeDepthLimit {
		return uuid.Nil, fmt.Errorf("%w: depth %d, limit %d", ErrCascadeTooDeep, e.Causation.Depth, config.CascadeDepthLimit)
	}
	payload, err := validate(e)
	if err != nil {
		return uuid.Nil, err
	}

	var existing uuid.UUID
	switch err := tx.QueryRow(ctx, dedupeLookupSQL, e.DedupeKey).Scan(&existing); {
	case err == nil:
		return existing, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return uuid.Nil, fmt.Errorf("outbox: dedupe lookup: %w", err)
	}

	tenant, seq, err := lock()
	if err != nil {
		return uuid.Nil, err
	}

	if id == uuid.Nil {
		if id, err = uuid.NewV7(); err != nil {
			return uuid.Nil, fmt.Errorf("outbox: new id: %w", err)
		}
	}
	var targetKey string
	switch t := e.Target.(type) {
	case TargetStart:
		targetKey = t.Kind + ":" + id.String()
	case TargetSignal:
		var start uuid.UUID
		switch err := tx.QueryRow(ctx, openRefSQL, tenant, e.RecordID, t.Kind).Scan(&start); {
		case errors.Is(err, pgx.ErrNoRows):
			return uuid.Nil, ErrNoOpenWorkflow
		case err != nil:
			return uuid.Nil, fmt.Errorf("outbox: find open workflow: %w", err)
		}
		targetKey = t.Kind + ":" + start.String()
	}

	var inserted uuid.UUID
	err = tx.QueryRow(ctx, insertSQL, id, tenant, e.Kind, e.KindVersion, e.DedupeKey, targetKey, seq,
		e.Causation.OriginID, e.Causation.Depth, payload, traceparent(ctx)).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, dedupeLookupSQL, e.DedupeKey).Scan(&existing); err != nil {
			return uuid.Nil, fmt.Errorf("outbox: dedupe lookup after conflict: %w", err)
		}
		return existing, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("outbox: insert: %w", err)
	}

	if t, ok := e.Target.(TargetStart); ok {
		if _, err := tx.Exec(ctx, insertRefSQL, tenant, id, e.RecordID, t.Kind); err != nil {
			return uuid.Nil, fmt.Errorf("outbox: insert workflow_ref: %w", err)
		}
	}
	return id, nil
}

// validate checks everything that needs no database, and returns the payload
// to store.
func validate(e Entry) (json.RawMessage, error) {
	invalid := func(format string, args ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrInvalidEntry}, args...)...)
	}
	if e.Causation.Depth < 0 {
		return nil, invalid("negative causation depth")
	}
	if e.Causation.OriginID == uuid.Nil {
		return nil, invalid("no causation origin (A5-3)")
	}
	if !kindPattern.MatchString(e.Kind) {
		return nil, invalid("kind must match %s", kindPattern)
	}
	if e.KindVersion < 1 || e.KindVersion > 32767 {
		return nil, invalid("kind version %d out of range", e.KindVersion)
	}
	if e.DedupeKey == "" || len(e.DedupeKey) > maxDedupeKeyLen {
		return nil, invalid("dedupe key must be 1 to %d bytes", maxDedupeKeyLen)
	}
	if strings.HasPrefix(e.DedupeKey, sealDedupePrefix) {
		return nil, invalid("dedupe keys starting %q are EnqueueSeal's", sealDedupePrefix)
	}
	if e.RecordID == uuid.Nil {
		return nil, invalid("no record id")
	}
	switch t := e.Target.(type) {
	case TargetStart:
		if t.Kind != e.Kind {
			return nil, invalid("start target kind differs from entry kind")
		}
	case TargetSignal:
		if !kindPattern.MatchString(t.Kind) {
			return nil, invalid("signal target kind must match %s", kindPattern)
		}
		if t.SignalName != e.Kind {
			return nil, invalid("signal name differs from entry kind")
		}
	default:
		return nil, invalid("no target")
	}
	payload := e.Payload
	if payload == nil {
		payload = json.RawMessage("{}")
	}
	if err := ValidatePayload(payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// payloadKeyPattern bounds every payload key (TICK-240), so a key can carry
// no free text: no spaces, punctuation, digits first or capitals, and at most
// 32 characters. outbox_payload_check (db/migrations/00009) uses the same
// pattern. A per-kind key allow-list was not workable; see
// docs/data/outbox-payload-check.md.
var payloadKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

// ValidatePayload enforces references only (ADR-001 R1, CI-7): a JSON object
// whose every key matches payloadKeyPattern and whose every value is a
// canonical UUID string or an array of them. Record data never enters an
// outbox row, so it never reaches a Temporal history.
//
// The error never quotes the payload, not even a key: a refused key may be
// the record data the rule keeps out.
func ValidatePayload(p json.RawMessage) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(p, &fields); err != nil || fields == nil {
		return fmt.Errorf("%w: payload must be a JSON object", ErrInvalidEntry)
	}
	for name, v := range fields {
		if !payloadKeyPattern.MatchString(name) {
			return fmt.Errorf("%w: payload key must match %s", ErrInvalidEntry, payloadKeyPattern)
		}
		if !isReference(v) {
			return fmt.Errorf("%w: payload value is not a canonical UUID or an array of them", ErrInvalidEntry)
		}
	}
	return nil
}

// isReference reports whether v is a canonical UUID string or an array
// (possibly empty) of them.
func isReference(v json.RawMessage) bool {
	var one string
	if json.Unmarshal(v, &one) == nil {
		return canonicalUUID(one)
	}
	var many []string
	if json.Unmarshal(v, &many) != nil {
		return false
	}
	for _, s := range many {
		if !canonicalUUID(s) {
			return false
		}
	}
	return true
}

// canonicalUUID reports whether s is a UUID in the one form uuid.UUID.String
// writes: 36 characters, lowercase hex, hyphenated. uuid.Parse alone also
// takes the URN, braced and 32-digit forms, and either case.
func canonicalUUID(s string) bool {
	u, err := uuid.Parse(s)
	return err == nil && u.String() == s
}

// traceparent is ctx's span context as a W3C traceparent, or nil (NULL) when
// ctx has no valid span. Only the TraceContext propagator runs, and only its
// traceparent key is kept.
func traceparent(ctx context.Context) *string {
	c := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, c)
	tp := c.Get("traceparent")
	if tp == "" {
		return nil
	}
	return &tp
}
