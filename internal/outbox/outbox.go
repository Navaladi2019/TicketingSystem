// Package outbox is the producer side of the transactional outbox (ADR-001
// I6, I9, Annex A.2 "Writing outbox rows"; PLAN-001 T29). Enqueue writes one
// outbox row in the caller's tenant transaction; the relay (T30a, T30b) is the
// only reader that dispatches it to Temporal.
//
// The tenant is never an argument: Enqueue runs in a transaction opened by
// tenancy.InTenantTx and takes tenant_id from the records row it locks, which
// row-level security lets it see only for that transaction's tenant.
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/propagation"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
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
// started the cascade, and how many writes deep this one is. A write a person
// or an API client made directly is depth 0 and may have no origin.
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

const maxDedupeKeyLen = 200

const (
	// Without it a duplicate retried after commit would still bump the
	// record's version; the INSERT below catches the concurrent duplicate.
	dedupeLookupSQL = `SELECT id FROM outbox WHERE dedupe_key = $1`
	// The row lock that orders a target's rows (A.2): concurrent producers on
	// one record queue here and take consecutive versions in commit order.
	lockRecordSQL = `UPDATE records SET version = version + 1 WHERE id = $1 RETURNING tenant_id, version`
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

// Enqueue writes e as an outbox row in tx and returns the row's id.
//
// It locks e.RecordID's records row and bumps its version; the new version is
// the row's target_seq. A start row's target_key is <kind>:<its own id>, and
// its workflow_ref row is inserted in tx. A signal row takes the target_key
// of the newest open workflow_ref of its kind on the record, or fails with
// ErrNoOpenWorkflow.
//
// If the tenant already has a row with e.DedupeKey, Enqueue returns that
// row's id and a nil error and writes nothing more. Only when the duplicate
// commits concurrently has the record's version already been bumped; a gap in
// target_seq is harmless, because the relay orders on it and never counts it.
//
// The W3C traceparent of ctx's span, if any, is stored in trace_context so the
// relay can link the workflow to the request that wrote the row. tracestate
// and baggage never cross.
//
// tx must come from tenancy.InTenantTx. On any error the caller rolls tx back.
func Enqueue(ctx context.Context, tx pgx.Tx, e Entry) (uuid.UUID, error) {
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

	var (
		tenant uuid.UUID
		seq    int64
	)
	switch err := tx.QueryRow(ctx, lockRecordSQL, e.RecordID).Scan(&tenant, &seq); {
	case errors.Is(err, pgx.ErrNoRows):
		return uuid.Nil, ErrRecordNotFound
	case err != nil:
		return uuid.Nil, fmt.Errorf("outbox: lock record: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("outbox: new id: %w", err)
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

	var causationID *uuid.UUID
	if e.Causation.OriginID != uuid.Nil {
		causationID = &e.Causation.OriginID
	}
	var inserted uuid.UUID
	err = tx.QueryRow(ctx, insertSQL, id, tenant, e.Kind, e.KindVersion, e.DedupeKey, targetKey, seq,
		causationID, e.Causation.Depth, payload, traceparent(ctx)).Scan(&inserted)
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
	if e.Causation.Depth > 0 && e.Causation.OriginID == uuid.Nil {
		return nil, invalid("causation depth %d with no origin", e.Causation.Depth)
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

// ValidatePayload enforces references only (ADR-001 R1, CI-7): a JSON object
// whose every value is a UUID string or an array of UUID strings. Record data
// never enters an outbox row, so it never reaches a Temporal history.
func ValidatePayload(p json.RawMessage) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(p, &fields); err != nil || fields == nil {
		return fmt.Errorf("%w: payload must be a JSON object", ErrInvalidEntry)
	}
	for name, v := range fields {
		var one string
		if json.Unmarshal(v, &one) == nil {
			if _, err := uuid.Parse(one); err != nil {
				return fmt.Errorf("%w: payload field %q is not a reference", ErrInvalidEntry, name)
			}
			continue
		}
		var many []string
		if err := json.Unmarshal(v, &many); err != nil {
			return fmt.Errorf("%w: payload field %q is not a reference", ErrInvalidEntry, name)
		}
		for _, s := range many {
			if _, err := uuid.Parse(s); err != nil {
				return fmt.Errorf("%w: payload field %q is not a reference", ErrInvalidEntry, name)
			}
		}
	}
	return nil
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
