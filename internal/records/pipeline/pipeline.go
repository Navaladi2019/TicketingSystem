// Package pipeline is the record write pipeline (PLAN-003 D-3-6): Write is the
// single way to create or update a record. In the caller's tenant transaction
// it loads the record and the entity type's current definition, checks that
// the caller can read the record (an unreadable one is ErrNotFound), applies
// the merge patch and defaults, validates against the JSON Schema, authorises
// the write against the record before and after it and against the field
// masks (authz.Authorizer, A1, P3-33;
// skipped only for the system sources rule, workflow, function and system),
// writes the row with an optimistic version check, writes one ADR-004 audit
// entry through audit.Append and calls the EventSink. Nothing commits unless
// every stage succeeds: the caller's tenancy.InTenantTx commits or rolls back.
// The records commit-check trigger (ADR-004 decision 2) is held by D-5-14
// until P5-13c (TICK-530); until then the P3-13 architecture test keeps every
// records write in this package.
//
// The entry (domain record, subject the record) carries the entity type, the
// versions before and after, the writer, the cascade origin and depth, and
// the RFC 6902 diff jsondiff.Diff builds from the payloads as PostgreSQL
// returns them (store.PayloadJSON), never from the request. Its action is
// create, update, or state_change when an update moves the record to another
// state. The secret pointers are the definition's x-nava-secret and writeOnly
// properties; a value at one is refused (R6).
//
// The tenant is never taken from the command. Write reads it from the
// transaction (tenancy.TenantOf) and refuses an actor of another tenant.
//
// A create takes the next per-tenant, per-type number from record_counters
// (D-3-9 as amended 2026-11-10: gaps allowed, no reclaim) after every check,
// in a short transaction of its own on the allocator pool that commits
// before the record is inserted (ADR-005 L2s, tenancy.AllocTx, alloc.go). A
// create that then fails leaves a gap.
// Every write checks the type's state model and its unique properties
// (D-3-22) (P3-11). A write that changes assignment_group_id must
// name a live group of the tenant (ADR-004-E5 D6); an unchanged value passes
// even after its group is tombstoned (D7).
//
// Cascades (ADR-001 I9 as amended by A5; P3-12, TICK-88): a depth-0 write
// mints a new origin, a workflow write must be deeper than 0, and a write
// beyond config.CascadeDepthLimit is refused with outbox.ErrCascadeTooDeep
// before any record change, its cascade_too_deep entry written on the
// TxRunner so it survives the caller's rollback (D-3-7a, ADR-004-E1). There is no cycle
// check on writes: the EventSink keys starts with cascade.ClaimStart and
// suppresses a repeat (A5-1, A5-4), so a writer never sees a cycle.
package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Op is the kind of write.
type Op int

// The writes Write performs.
const (
	OpCreate Op = iota + 1
	OpUpdate
)

// String is the write's op as record_write_keys.action stores it. The audit
// entry's action is create, update or state_change (audit.go).
func (o Op) String() string {
	switch o {
	case OpCreate:
		return "create"
	case OpUpdate:
		return "update"
	default:
		return "unknown"
	}
}

// Source is who made the write (audit_log.source, ADR-004 R2).
type Source string

// The R2 sources a record write may name.
const (
	SourceUser     Source = "user"
	SourceRule     Source = "rule"
	SourceWorkflow Source = "workflow"
	SourceFunction Source = "function"
	SourceAI       Source = "ai"
	SourceSystem   Source = "system"
)

// system reports whether a write from s skips the authorizer: the explicit
// allowlist of contract amendment A1. A user, an AI (no privilege through the
// AI path) and any source not listed here, empty or unknown, are authorized
// as their principal (fail closed).
func (s Source) system() bool {
	switch s {
	case SourceRule, SourceWorkflow, SourceFunction, SourceSystem:
		return true
	}
	return false
}

func (s Source) valid() bool {
	switch s {
	case SourceUser, SourceRule, SourceWorkflow, SourceFunction, SourceAI, SourceSystem:
		return true
	}
	return false
}

// maxIdempotencyKeyLen bounds a key; D-3-13's wf:<workflow id>:<step id> is
// well under it.
const maxIdempotencyKeyLen = 200

// Command is one write.
type Command struct {
	Op        Op
	EntityKey string
	// RecordID names the record to update; it must be uuid.Nil on a create,
	// which takes a new UUIDv7.
	RecordID uuid.UUID
	// ExpectedVersion is the version the update was based on (the ETag); 0 on
	// a create.
	ExpectedVersion int64
	// Data is the full payload on a create, and a JSON Merge Patch (RFC 7396)
	// on an update: null removes a key, objects merge, arrays and every other
	// value replace. Values must marshal to JSON.
	Data map[string]any
	// Actor is the resolved principal. Its TenantID must be the transaction's
	// tenant. A user write names a principal; a system writer may not.
	Actor  identity.Principal
	Source Source
	// Causation is the cascade the write belongs to. Leave it empty for a
	// depth-0 write (a person or an API client): Write mints a new origin.
	// A deeper write names its origin and depth, taken from the delivery
	// that caused it (A5-7); a workflow write is never depth 0.
	Causation outbox.Causation
	// IdempotencyKey is for system writers only (D-3-13): a replay by the
	// same Actor.PrincipalID (none for a system writer) with the same key,
	// entity type, op and (on an update) record returns the earlier Result
	// and writes nothing. Each principal has its own key space: another
	// principal's write with the key is never replayed (TICK-480) and never
	// conflicts (TICK-626: record_write_keys is unique per principal). The
	// same key on a different write by the same principal is
	// *ErrIdempotencyConflict.
	IdempotencyKey string
	// SourceRef is the AI decision record's id, a lowercase UUID: required
	// when Source is SourceAI and refused otherwise (ADR-004 R4).
	SourceRef *string
	// InsertAtCommit lets a create of a `gaps` type queue its write (the
	// record, the write key, the audit entry) for the round trip that
	// commits tx (tenancy.QueueAtCommit) instead of sending it as a round
	// trip of its own (TICK-692). Set it only when the caller sends nothing
	// on tx after Write returns but statements it queues at commit itself.
	// Write then answers with the record as it will be stored, from the
	// number, the allocation's instant and the payload preview; the INSERT
	// fails, and nothing commits, unless the stored row is exactly that
	// (store.QueueInsertRecordAsExpected). A failure of the queued write
	// comes out of tenancy.InTenantTx mapped as Write would map it,
	// *ErrAtVersion included. It is ignored for an update, a `gapless`
	// type, a tx with no commit batch (a savepoint, or not from
	// tenancy.InTenantTx) and a create the EventSink has work for (it may
	// send statements that must follow the INSERT); those write as before.
	InsertAtCommit bool
	// WorkflowRef is the outbox id of the writing run's workflow_ref, for a
	// write a workflow makes; uuid.Nil otherwise. Only a system source may
	// name one. The EventSink never signals that run about its own write
	// (D-3-14).
	WorkflowRef uuid.UUID
}

// Result is what a write produced.
type Result struct {
	RecordID      uuid.UUID
	Number        string
	Version       int64
	SchemaVersion int
}

// Event is what the EventSink is told about a committed-to-be write:
// references only, never record data (CI-7).
type Event struct {
	Op                 Op
	EntityDefinitionID uuid.UUID
	RecordID           uuid.UUID
	// Version is the record's version after this write.
	Version int64
	// Causation is the write's stored causation: the origin, minted for a
	// depth-0 write, and the depth.
	Causation outbox.Causation
	// ActorID, Source and SourceRef are the writer, for the cascade_cycle
	// entry of a suppressed start (A5-4). ActorID is uuid.Nil for a system
	// writer.
	ActorID   uuid.UUID
	Source    Source
	SourceRef *string
	// WorkflowRef is Command.WorkflowRef: the run whose own write this is,
	// which the sink never signals about it (D-3-14). uuid.Nil for a write
	// no workflow made.
	WorkflowRef uuid.UUID
}

// EventSink is told about every write, in the write's transaction (D-3-14).
// An error rolls the whole write back. A sink that starts nava.interpreter
// claims each start with cascade.ClaimStart and enqueues it with
// outbox.EnqueueStart; on a repeat it enqueues nothing, writes
// AuditSuppressed and returns nil (A5-4). Any other outbox row is
// enqueued with outbox.EnqueueAtVersion and ev.Version: outbox never bumps
// the version, so the write bumps it exactly once.
type EventSink interface {
	RecordChanged(ctx context.Context, tx pgx.Tx, ev Event) error
}

// CreateTriggerQueuer is an EventSink that can tell, on the write's prefetch
// read of its entity type, that a create of that type gives it nothing to
// do. CreateTriggeredSQL is a constant boolean SQL expression over that
// entity_definitions row, aliased e (e.tenant_id, e.id), true when a create
// fires anything; it is a column of the read Write sends anyway, so it costs
// no statement and no round trip. When it is false, Write does not call
// RecordChanged for the create, so a create with Command.InsertAtCommit keeps
// its write on the commit round trip (TICK-692). A create has no workflow on
// its record yet, so only a start could be due.
type CreateTriggerQueuer interface {
	CreateTriggeredSQL() string
}

// NoopSink is the S9 EventSink: it does nothing. The API's pipeline uses
// the real one (internal/records/eventsink, P3-42); tests that start no
// workflow keep this one.
type NoopSink struct{}

// RecordChanged does nothing.
func (NoopSink) RecordChanged(context.Context, pgx.Tx, Event) error { return nil }

// DefinitionReader reads compiled definitions in the caller's transaction.
// entitytypes.Store implements it (P3-05a).
type DefinitionReader interface {
	Version(ctx context.Context, tx pgx.Tx, entityKey string, version int) (*metadata.Compiled, error)
	Current(ctx context.Context, tx pgx.Tx, entityKey string) (*metadata.Compiled, int, error)
}

// TxRunner runs fn in a tenant transaction of its own, while the caller's
// transaction is open. Write records a cascade_rejected audit row on it, so
// the row survives the caller's rollback (D-3-7a). A TxRunner that is also an
// Allocator takes a `gaps` create's number (ADR-005 L2s, TICK-657). Each
// call needs a connection the callers' transactions do not hold: a
// PoolRunner over the callers' own pool can starve, every connection held by
// a create waiting for a second one. nava-core gives it a pool of its own
// (the allocator pool, cmd/nava-core/allocator.go).
type TxRunner interface {
	InTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(pgx.Tx) error) error
}

// PoolRunner is the TxRunner over a pool, through tenancy.InTenantTx, so
// the transaction's BEGIN sets the tenant and RLS applies to it. It is also
// the Allocator, through tenancy.AllocTx (alloc.go).
type PoolRunner struct {
	Pool *pgxpool.Pool
	// AllocTimeout bounds an allocation (ADR-005 A5); zero is
	// DefaultAllocTimeout.
	AllocTimeout time.Duration
	// Metrics records the allocator's metrics (A8); nil records none.
	Metrics *AllocMetrics
}

// InTenantTx implements TxRunner.
func (r PoolRunner) InTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(pgx.Tx) error) error {
	return tenancy.InTenantTx(ctx, r.Pool, tenantID, fn)
}

// Clock gives the time written to updated_at.
type Clock interface{ Now() time.Time }

// SystemClock is the wall clock.
type SystemClock struct{}

// Now returns time.Now().
func (SystemClock) Now() time.Time { return time.Now() }

var (
	// ErrInvalidCommand means the command breaks its contract (a caller bug,
	// not user input). Nothing is written.
	ErrInvalidCommand = errors.New("pipeline: invalid command")
	// ErrNotFound means the transaction's tenant has no such entity type, or
	// no such record of that type. The cases are not told apart.
	ErrNotFound = errors.New("pipeline: not found")
	// ErrVersionConflict means the record is not at the expected version.
	ErrVersionConflict = errors.New("pipeline: version conflict")
	// ErrRetryable means PostgreSQL aborted the transaction to break a
	// deadlock (40P01) or a serialization failure (40001). Nothing in it
	// commits: roll tx back and run the whole transaction again. It wraps the
	// database error. A transaction that holds two or more pipeline writes
	// can deadlock with another one (P3-11's locks are ordered within one
	// write, not across writes: TICK-88, from the PR #360 review); cascades
	// themselves never share one, because each runs in its own activity.
	// It also means a create's number allocation passed its timeout
	// (ADR-005 A5): nothing was inserted, and a retry takes a new number.
	ErrRetryable = errors.New("pipeline: transaction aborted by a deadlock or serialization failure, or a number allocation timed out, retry")
	// ErrAssignmentGroupRemoved means the write changes assignment_group_id to
	// a group the tenant does not have live: tombstoned, another tenant's
	// (live or tombstoned) or unknown. The cases are not told apart
	// (ADR-004-E5 D6, N4). Nothing is written. A value the record already
	// holds is never refused (D7).
	ErrAssignmentGroupRemoved = errors.New("pipeline: assignment group removed or unknown")
)

// ErrIdempotencyConflict means the idempotency key is held by another write.
// The HTTP layer maps it to 409 (P3-14), as PLAN-001 T22 does for a key
// reused with a different body.
type ErrIdempotencyConflict struct {
	// Concurrent: a concurrent write with the same key committed first, and
	// the caller's transaction is aborted. Roll it back and retry: the retry
	// replays the first write's Result, or gets the mismatch below.
	//
	// Otherwise the key is stored for a write of another entity type, op or
	// record. A retry gets the same error.
	Concurrent bool
}

func (e *ErrIdempotencyConflict) Error() string {
	if e.Concurrent {
		return "pipeline: concurrent write with the same idempotency key, retry"
	}
	return "pipeline: idempotency key already used for another entity type, op or record"
}

// ErrIllegalTransition is a state change the type's state model does not
// list. On a create From is empty: a create must start in the initial state.
type ErrIllegalTransition struct {
	From, To string
}

func (e *ErrIllegalTransition) Error() string {
	if e.From == "" {
		return fmt.Sprintf("pipeline: a record is created in its initial state, not %q", e.To)
	}
	return fmt.Sprintf("pipeline: illegal state transition %q -> %q", e.From, e.To)
}

// ErrUniqueViolation is a value of a unique property ("x-nava-unique": true)
// that another record of the entity type in the tenant already holds.
type ErrUniqueViolation struct {
	Field string
}

func (e *ErrUniqueViolation) Error() string {
	return fmt.Sprintf("pipeline: value of unique property %q is taken", e.Field)
}

// ErrValidation is a payload that fails the entity type's schema. Fields are
// sorted by path, then code.
type ErrValidation struct {
	Fields []metadata.FieldError
}

func (e *ErrValidation) Error() string {
	paths := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		paths[i] = f.Path + " (" + f.Code + ")"
	}
	return "pipeline: validation failed at " + strings.Join(paths, ", ")
}

// ErrAtVersion is a refused or failed write that got as far as reading the
// entity type's current definition. It names the version the write used and
// that version's fixed-column mapping and state property, so a caller maps the
// error to payload properties under the same version the pipeline did, not
// under a definition it read itself, which a concurrent publish can have
// replaced (TICK-413). Err is the cause, unchanged: errors.Is and errors.As
// see through ErrAtVersion, and its text is the cause's.
type ErrAtVersion struct {
	SchemaVersion int
	FixedColumns  map[string]string
	StateProperty string
	Err           error
}

func (e *ErrAtVersion) Error() string { return e.Err.Error() }

func (e *ErrAtVersion) Unwrap() error { return e.Err }

// Pipeline writes records. It holds no per-write state and is safe for
// concurrent use.
type Pipeline struct {
	authz  authz.Authorizer
	defs   DefinitionReader
	sink   EventSink
	runner TxRunner
	clock  Clock
	// beforeReturn, if set, runs after every statement of a successful write
	// and before Write returns, so before the caller commits. Tests only
	// (export_test.go): the kill test parks a process here.
	beforeReturn func(context.Context)
}

// New returns a Pipeline. Every argument is required.
func New(a authz.Authorizer, defs DefinitionReader, sink EventSink, tx TxRunner, clock Clock) *Pipeline {
	if a == nil || defs == nil || sink == nil || tx == nil || clock == nil {
		panic("pipeline: New needs every dependency")
	}
	return &Pipeline{authz: a, defs: defs, sink: sink, runner: tx, clock: clock}
}

// Runner returns the pipeline's own-transaction runner (D-3-7a).
func (p *Pipeline) Runner() TxRunner { return p.runner }

// Write performs cmd in tx, which must come from tenancy.InTenantTx at READ
// COMMITTED (the default). It returns ErrInvalidCommand, ErrNotFound,
// authz.ErrForbidden, *authz.ErrFieldNotWritable, *ErrValidation, *ErrIllegalTransition,
// *ErrUniqueViolation, ErrAssignmentGroupRemoved, ErrVersionConflict,
// *ErrIdempotencyConflict or
// outbox.ErrCascadeTooDeep for a refused write; ErrRetryable, wrapping the
// database error, when PostgreSQL aborted tx for a deadlock (40P01) or a
// serialization failure (40001): roll back and retry the whole transaction;
// and a wrapped error for any other database failure. An error raised after
// the definition was read is wrapped in *ErrAtVersion, naming the version
// the write used. A value at a secret pointer, or a URL with userinfo, in
// the audit diff is *ErrValidation with the code audit_secret_in_diff
// (ADR-004 R6). On any error the caller rolls tx back; after a concurrent
// *ErrIdempotencyConflict, ErrRetryable, an audit.Append failure (including
// audit.ErrDuplicateEntry, which it wraps) or a database error tx is
// unusable, and Write never retries in it.
//
// The depth check runs before any statement in tx but the read of its
// tenant: a write beyond the limit changes nothing in tx and writes one
// cascade_too_deep entry on the TxRunner, committed before Write returns.
// If that entry cannot be written, the error also wraps the cause.
func (p *Pipeline) Write(ctx context.Context, tx pgx.Tx, cmd Command) (Result, error) {
	res, err := p.perform(ctx, tx, cmd)
	if retryable(err) {
		return Result{}, fmt.Errorf("%w: %w", ErrRetryable, err)
	}
	return res, err
}

// retryable reports whether PostgreSQL aborted the transaction for a
// deadlock (40P01) or a serialization failure (40001).
func retryable(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && (pe.Code == "40P01" || pe.Code == "40001")
}

// perform is Write before its error mapping.
func (p *Pipeline) perform(ctx context.Context, tx pgx.Tx, cmd Command) (Result, error) {
	if err := cmd.check(); err != nil {
		return Result{}, err
	}
	tenant, err := tenancy.TenantOf(ctx, tx)
	if errors.Is(err, tenancy.ErrNoTenant) {
		return Result{}, fmt.Errorf("%w: %w", ErrInvalidCommand, err)
	}
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	if cmd.Actor.TenantID != tenant {
		return Result{}, authz.ErrForbidden
	}
	if cmd.Causation.Depth > config.CascadeDepthLimit {
		return Result{}, p.rejectTooDeep(ctx, tenant, cmd)
	}
	if cmd.Causation.Depth == 0 {
		origin, err := uuid.NewV7()
		if err != nil {
			return Result{}, fmt.Errorf("pipeline: new origin: %w", err)
		}
		cmd.Causation.OriginID = origin
	}

	pre, err := p.prefetch(ctx, tx, tenant, cmd)
	if err != nil {
		return Result{}, err
	}
	if cmd.IdempotencyKey != "" {
		res, found, err := replay(pre, cmd)
		if err != nil || found {
			return res, err
		}
	}

	if pre.defID == uuid.Nil {
		return Result{}, ErrNotFound
	}
	defID := pre.defID
	def, defVersion, err := p.defs.Current(ctx, tx, cmd.EntityKey)
	if err != nil {
		return Result{}, mapNotFound(err)
	}
	w := write{p: p, tx: tx, tenant: tenant, cmd: cmd, defID: defID, def: def, defVersion: defVersion,
		sinkQuiet: cmd.Op == OpCreate && pre.createQuiet}
	res, err := w.do(ctx)
	if err != nil {
		return Result{}, &ErrAtVersion{SchemaVersion: defVersion, FixedColumns: def.FixedColumns(), StateProperty: def.StateProperty(), Err: err}
	}
	if p.beforeReturn != nil {
		p.beforeReturn(ctx)
	}
	return res, nil
}

// check validates the command's shape.
func (c *Command) check() error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrInvalidCommand}, args...)...)
	}
	switch c.Op {
	case OpCreate:
		if c.RecordID != uuid.Nil || c.ExpectedVersion != 0 {
			return bad("a create names no record and no expected version")
		}
	case OpUpdate:
		if c.RecordID == uuid.Nil || c.ExpectedVersion < 1 {
			return bad("an update names a record and an expected version of 1 or more")
		}
	default:
		return bad("unknown op %d", c.Op)
	}
	if c.EntityKey == "" {
		return bad("no entity key")
	}
	if !c.Source.valid() {
		return bad("unknown source")
	}
	if c.Source == SourceUser && c.Actor.PrincipalID == uuid.Nil {
		return bad("a user write names a principal")
	}
	if c.WorkflowRef != uuid.Nil && !c.Source.system() {
		return bad("only a system writer names a workflow ref")
	}
	if c.IdempotencyKey != "" && (c.Source == SourceUser || len(c.IdempotencyKey) > maxIdempotencyKeyLen) {
		return bad("an idempotency key is for system writers only, at most %d bytes", maxIdempotencyKeyLen)
	}
	if c.Causation.Depth < 0 || (c.Causation.Depth > 0 && c.Causation.OriginID == uuid.Nil) {
		return bad("causation depth without an origin, or negative")
	}
	// A depth-0 write starts a cascade, so its origin is minted, never
	// given: a given one could reuse another cascade's start keys (A5-3).
	if c.Causation.Depth == 0 && c.Causation.OriginID != uuid.Nil {
		return bad("a depth-0 write names no origin; Write mints it")
	}
	// A workflow write is one deeper than the delivery that caused it, so it
	// cannot restart the chain (A5-7).
	if c.Source == SourceWorkflow && c.Causation.Depth == 0 {
		return fmt.Errorf("%w: %w: a workflow write at depth 0", ErrInvalidCommand, outbox.ErrInvalidEntry)
	}
	// ADR-004 R4: an AI write names its decision record, as a lowercase UUID.
	if c.Source == SourceAI {
		if c.SourceRef == nil {
			return bad("an AI write names its decision record (source_ref)")
		}
		if u, err := uuid.Parse(*c.SourceRef); err != nil || u.String() != *c.SourceRef {
			return bad("source_ref is not a lowercase UUID")
		}
	} else if c.SourceRef != nil {
		return bad("only an AI write names a source_ref")
	}
	return nil
}

// rejectTooDeep writes the cascade_too_deep entry of a write beyond the depth
// limit in a transaction of its own (appendTooDeep), and returns the
// rejection.
func (p *Pipeline) rejectTooDeep(ctx context.Context, tenant uuid.UUID, cmd Command) error {
	rejected := fmt.Errorf("pipeline: %w: depth %d, limit %d", outbox.ErrCascadeTooDeep, cmd.Causation.Depth, config.CascadeDepthLimit)
	err := p.runner.InTenantTx(ctx, tenant, func(tx pgx.Tx) error { return appendTooDeep(ctx, tx, tenant, cmd) })
	if err != nil {
		return errors.Join(rejected, fmt.Errorf("pipeline: audit the rejected cascade: %w", err))
	}
	return rejected
}

// replay returns the earlier Result of a write by cmd's actor with cmd's
// idempotency key, read from record_write_keys (D-5-3), never from the audit
// log. The key is scoped by principal: another actor's write with the same
// key is not found here, and the unique index is per principal (TICK-626),
// so this write goes ahead as the caller's own. The stored
// write must be the same entity type (by definition id) and op, and on an
// update the same record; otherwise the key names another write,
// *ErrIdempotencyConflict.
func replay(pre *prefetched, cmd Command) (Result, bool, error) {
	k := pre.writeKey
	if k == nil {
		return Result{}, false, nil
	}
	// An entity key the tenant does not have cannot name the stored write.
	if pre.defID == uuid.Nil || k.DefinitionID != pre.defID || k.Action != cmd.Op.String() || (cmd.Op == OpUpdate && k.RecordID != cmd.RecordID) {
		return Result{}, false, &ErrIdempotencyConflict{}
	}
	return Result{RecordID: k.RecordID, Number: k.Number, Version: k.Version, SchemaVersion: k.SchemaVersion}, true, nil
}

// write is the state of one Write.
type write struct {
	p          *Pipeline
	tx         pgx.Tx
	tenant     uuid.UUID
	cmd        Command
	defID      uuid.UUID
	def        *metadata.Compiled
	defVersion int
	// sinkQuiet: a create the EventSink said, in the prefetch, it has
	// nothing to do for (CreateTriggerQueuer). RecordChanged is not called.
	sinkQuiet bool
}

// do performs the write under w.def.
func (w *write) do(ctx context.Context) (Result, error) {
	data, err := normalise(w.cmd.Data)
	if err != nil {
		return Result{}, err
	}
	if w.cmd.Op == OpUpdate {
		return w.update(ctx, data)
	}
	return w.create(ctx, data)
}

func (w *write) create(ctx context.Context, after map[string]any) (Result, error) {
	// The mask write check runs first, on the caller's own data: the defaults
	// applied below are not its writes, and no other error (validation,
	// state, row scope, unique) can come back for a masked field (P3-33).
	if err := w.canWriteFields(ctx, nil, after, nil); err != nil {
		return Result{}, err
	}
	after = w.def.ApplyDefaults(after)
	// The state column is never NULL, and a record starts in the initial
	// state.
	if _, ok := after[w.def.StateProperty()]; !ok {
		after[w.def.StateProperty()] = w.def.InitialState()
	}
	if fe := w.def.Validate(after); fe != nil {
		return Result{}, &ErrValidation{Fields: fe}
	}
	if state, _ := after[w.def.StateProperty()].(string); state != w.def.InitialState() {
		return Result{}, &ErrIllegalTransition{To: state}
	}
	fixed := w.def.FixedColumns()
	id, err := uuid.NewV7()
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: new record id: %w", err)
	}
	now := w.p.clock.Now()
	rec := &store.Record{
		TenantID: w.tenant, ID: id, EntityDefinitionID: w.defID, EntityKey: w.cmd.EntityKey,
		Version: 1, SchemaVersion: w.defVersion,
		CreatedBy: optional(w.cmd.Actor.PrincipalID), CreatedAt: now, UpdatedAt: now,
	}
	w.causation(rec)
	if err := rec.Split(after, fixed); err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	w.defaultRequester(rec, after, fixed)
	// A new record has no shares. The check runs before the unique and
	// group lookups, so a refused writer learns nothing from them.
	if err := w.canWrite(ctx, nil, rowOf(rec, nil)); err != nil {
		return Result{}, err
	}
	// The unique and group checks, and the payload the audit entry is built
	// from, go in one batch (TICK-648).
	preview, err := w.checkCreate(ctx, rec, after, fixed)
	if err != nil {
		return Result{}, err
	}
	// The number is taken after every check (ADR-001 I4, ADR-005 A6): a
	// refused or invalid create never moves the counter. It is taken in a
	// transaction of its own that commits before this one inserts the record
	// (ADR-005 L2s), so a create that fails from here on leaves a gap.
	// A `gapless` type takes it last inside this transaction instead
	// (ADR-005 L1, writeCreate). The mode is the one the definition had
	// when this create read it (§2.1.1).
	if w.def.NumberingMode() != metadata.NumberingGapless {
		alloc, err := w.allocate(ctx)
		if err != nil {
			return Result{}, err
		}
		rec.Number = store.FormatNumber(w.def.NumberPrefix(), alloc.Number)
		// The record is created at the allocation's instant (A7), so a date
		// in a number will always match created_at.
		rec.CreatedAt, rec.UpdatedAt = alloc.Instant, alloc.Instant
	}
	if w.atCommit() {
		return w.writeCreateAtCommit(ctx, rec, preview)
	}
	return w.writeCreate(ctx, rec, preview)
}

// defaultRequester sets requester_id to the caller when a user or AI create
// leaves the requester property out (contract amendment A4), before the write
// check, so the check sees the row that would be stored. An explicit value,
// null included, is the caller's own and goes through the check as sent. A
// system source gets no default: it has no principal to name. A type that maps
// no property to requester_id has nothing to default.
func (w *write) defaultRequester(rec *store.Record, after map[string]any, fixed map[string]string) {
	if w.cmd.Source.system() || w.cmd.Actor.PrincipalID == uuid.Nil {
		return
	}
	for prop, col := range fixed {
		if col != metadata.ColumnRequesterID {
			continue
		}
		if _, sent := after[prop]; !sent {
			id := w.cmd.Actor.PrincipalID
			rec.RequesterID = &id
		}
		return
	}
}

func (w *write) update(ctx context.Context, patch map[string]any) (Result, error) {
	// Without the groups read: an update never uses the record's group
	// tombstone flag, and the worker's role cannot read groups.
	rec, err := store.GetForWrite(ctx, w.tx, w.tenant, w.cmd.RecordID)
	if err != nil {
		return Result{}, mapNotFound(err)
	}
	if rec.EntityKey != w.cmd.EntityKey {
		return Result{}, ErrNotFound
	}
	// An unreadable record is a missing one, checked before the version so
	// it never yields a version conflict (TICK-358 carry). Its shares are
	// the caller's (pair-matched) and do not change in this write.
	var beforeRow *authz.Row
	if !w.cmd.Source.system() {
		shares, err := authz.SharesFor(ctx, w.tx, w.cmd.Actor, rec.ID)
		if err != nil {
			return Result{}, fmt.Errorf("pipeline: %w", err)
		}
		beforeRow = rowOf(rec, shares)
		if err := w.p.authz.CanRead(ctx, w.tx, w.cmd.Actor, w.cmd.EntityKey, *beforeRow); err != nil {
			if errors.Is(err, authz.ErrNotVisible) {
				return Result{}, ErrNotFound
			}
			return Result{}, fmt.Errorf("pipeline: %w", err)
		}
	}
	if rec.Version != w.cmd.ExpectedVersion {
		return Result{}, ErrVersionConflict
	}
	// The row was split under its own version's mapping.
	stored := w.def
	if rec.SchemaVersion != w.defVersion {
		if stored, err = w.p.defs.Version(ctx, w.tx, w.cmd.EntityKey, rec.SchemaVersion); err != nil {
			return Result{}, fmt.Errorf("pipeline: read stored definition: %w", err)
		}
	}
	before := rec.Payload(stored.FixedColumns())

	// merged is the caller's write: the stored payload with its patch, before
	// defaults. The mask write check runs on it right away, before
	// validation, the state model, the row check and the unique lookup, so
	// no other answer can depend on a masked field's stored value (P3-33,
	// TICK-633 B1). ApplyDefaults returns a new map.
	merged := mergePatch(deepCopy(before), patch)
	if err := w.canWriteFields(ctx, before, merged, patch); err != nil {
		return Result{}, err
	}
	after := w.def.ApplyDefaults(merged)
	fe := w.def.Validate(after)
	if _, ok := after[w.def.StateProperty()]; !ok {
		fe = appendField(fe, metadata.FieldError{Path: "/" + escape(w.def.StateProperty()), Code: "required"})
	}
	if fe != nil {
		return Result{}, &ErrValidation{Fields: fe}
	}
	// Staying in the same state is no transition.
	if to, _ := after[w.def.StateProperty()].(string); to != rec.State && !w.def.AllowsTransition(rec.State, to) {
		return Result{}, &ErrIllegalTransition{From: rec.State, To: to}
	}

	fixed := w.def.FixedColumns()
	// The after row is the record as this write leaves it: before and after
	// must both be in the caller's write scope, so a write cannot move a
	// record out of it. Checked before the unique and group lookups.
	if !w.cmd.Source.system() {
		next := *rec
		if err := next.Split(after, fixed); err != nil {
			return Result{}, fmt.Errorf("pipeline: %w", err)
		}
		if err := w.canWrite(ctx, beforeRow, rowOf(&next, beforeRow.SharedWith)); err != nil {
			return Result{}, err
		}
	}
	if err := w.checkUnique(ctx, rec.ID, before, after, fixed); err != nil {
		return Result{}, err
	}
	// The audit diff's before is the stored row as PostgreSQL returns it,
	// under the mapping it was split with; it must still be at the expected
	// version.
	beforeJSON, err := store.PayloadJSON(ctx, w.tx, w.tenant, rec.ID, rec.Version, stored.FixedColumns())
	if errors.Is(err, store.ErrStale) {
		return Result{}, ErrVersionConflict
	}
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	prev := &prior{payload: beforeJSON, version: rec.Version, state: rec.State, secretProps: stored.SecretProperties()}
	storedGroup := rec.AssignmentGroupID
	if err := rec.Split(after, fixed); err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	if err := w.checkGroup(ctx, storedGroup, rec.AssignmentGroupID); err != nil {
		return Result{}, err
	}
	rec.SchemaVersion = w.defVersion
	rec.UpdatedAt = w.p.clock.Now()
	w.causation(rec)
	tenancy.Forget(w.tx, storedKey{rec.ID})
	v, err := store.Update(ctx, w.tx, rec, w.cmd.ExpectedVersion)
	if errors.Is(err, store.ErrStale) {
		return Result{}, ErrVersionConflict
	}
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	rec.Version = v
	return w.finish(ctx, rec, prev)
}

// prior is what an update read of the record before it changed it: its
// payload as PostgreSQL returned it, version and state, and the secret
// properties of the definition version it was written under.
type prior struct {
	payload     []byte
	version     int64
	state       string
	secretProps []string
}

// finish claims the idempotency key, writes the write's audit entry through
// audit.Append and calls the sink. prev is nil on a create.
func (w *write) finish(ctx context.Context, rec *store.Record, prev *prior) (Result, error) {
	auditID, err := uuid.NewV7()
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: new audit id: %w", err)
	}
	if w.cmd.IdempotencyKey != "" {
		// The key is claimed before the audit entry, so a concurrent duplicate
		// by the same principal waits on and fails at
		// record_write_keys_actor_key (D-5-3, TICK-626).
		if err := store.InsertWriteKey(ctx, w.tx, &store.WriteKey{
			TenantID: w.tenant, IdempotencyKey: w.cmd.IdempotencyKey, DefinitionID: rec.EntityDefinitionID,
			Action: w.cmd.Op.String(), RecordID: rec.ID, Number: rec.Number, Version: rec.Version,
			SchemaVersion: rec.SchemaVersion, AuditEntryID: auditID,
			ActorPrincipalID: optional(w.cmd.Actor.PrincipalID),
		}); err != nil {
			if errors.Is(err, store.ErrDuplicateKey) {
				return Result{}, &ErrIdempotencyConflict{Concurrent: true}
			}
			return Result{}, fmt.Errorf("pipeline: %w", err)
		}
	}
	after, err := store.PayloadJSON(ctx, w.tx, w.tenant, rec.ID, rec.Version, w.def.FixedColumns())
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	e := recordEntry{id: auditID, action: audit.ActionCreate, rec: rec, after: after, secretProps: w.def.SecretProperties()}
	if prev != nil {
		e.action = audit.ActionUpdate
		if rec.State != prev.state {
			e.action = audit.ActionStateChange
		}
		e.before, e.versionBefore = prev.payload, &prev.version
		e.secretProps = unionSorted(e.secretProps, prev.secretProps)
	}
	if err := appendRecordEntry(ctx, w.tx, w.cmd, e); err != nil {
		return Result{}, err
	}
	ev := Event{Op: w.cmd.Op, EntityDefinitionID: rec.EntityDefinitionID, RecordID: rec.ID, Version: rec.Version,
		Causation: w.cmd.Causation, ActorID: w.cmd.Actor.PrincipalID, Source: w.cmd.Source, SourceRef: w.cmd.SourceRef,
		WorkflowRef: w.cmd.WorkflowRef}
	if err := w.p.sink.RecordChanged(ctx, w.tx, ev); err != nil {
		return Result{}, fmt.Errorf("pipeline: event sink: %w", err)
	}
	return Result{RecordID: rec.ID, Number: rec.Number, Version: rec.Version, SchemaVersion: rec.SchemaVersion}, nil
}

// checkGroup refuses a write that changes assignment_group_id to a group the
// tenant does not have live (ADR-004-E5 D6): stored is the record's value
// before the write (nil on a create), next its value after. An unchanged
// value passes even when its group is tombstoned (D7), and clearing the
// field passes. The lookup runs in w.tx, under its RLS, never against a
// cached list, and before any write statement, so a refused write leaves no
// records, audit_log, outbox or record_write_keys row. It also runs before
// the INSERT or UPDATE would reach records_assignment_group_fkey, so another
// tenant's group and an unknown id end here on create and on update alike,
// never at a 23503 (N4).
func (w *write) checkGroup(ctx context.Context, stored, next *uuid.UUID) error {
	if next == nil || (stored != nil && *stored == *next) {
		return nil
	}
	live, err := store.GroupLive(ctx, w.tx, w.tenant, *next)
	if err != nil {
		return fmt.Errorf("pipeline: %w", err)
	}
	if !live {
		return ErrAssignmentGroupRemoved
	}
	return nil
}

// canWrite asks the authorizer, unless the write's source is a system one
// (Source.system). A refusal is authz.ErrForbidden; a failure to decide is
// wrapped.
func (w *write) canWrite(ctx context.Context, before, after *authz.Row) error {
	if w.cmd.Source.system() {
		return nil
	}
	err := w.p.authz.CanWrite(ctx, w.tx, w.cmd.Actor, w.cmd.EntityKey, before, after)
	if err != nil && !errors.Is(err, authz.ErrForbidden) {
		return fmt.Errorf("pipeline: %w", err)
	}
	return err
}

// canWriteFields is the mask write check of the authorise stage (P3-33),
// skipped for the same system sources as canWrite. before is the stored
// payload (nil on a create), after the caller's data without defaults, and
// patch the caller's merge patch (nil on a create). A refusal is
// *authz.ErrFieldNotWritable or authz.ErrForbidden; a failure to decide is
// wrapped. It is the first check on the caller's data and runs before every
// write statement, so a refused write leaves no record, audit or outbox row.
func (w *write) canWriteFields(ctx context.Context, before, after, patch map[string]any) error {
	if w.cmd.Source.system() {
		return nil
	}
	err := w.p.authz.CanWriteFields(ctx, w.tx, w.cmd.Actor, w.cmd.EntityKey, before, after, patch)
	var nw *authz.ErrFieldNotWritable
	if err != nil && !errors.As(err, &nw) && !errors.Is(err, authz.ErrForbidden) {
		return fmt.Errorf("pipeline: %w", err)
	}
	return err
}

// rowOf is what the ACL sees of rec: its fixed columns, and shares, the
// caller's pair-matched shares from authz.SharesFor.
func rowOf(rec *store.Record, shares []uuid.UUID) *authz.Row {
	return &authz.Row{
		ID: rec.ID, State: rec.State, RequesterID: rec.RequesterID, AssigneeID: rec.AssigneeID,
		AssignmentGroupID: rec.AssignmentGroupID, SharedWith: shares,
	}
}

func (w *write) causation(rec *store.Record) {
	rec.CausationOriginID = optional(w.cmd.Causation.OriginID)
	rec.CausationDepth = w.cmd.Causation.Depth
}

// checkUnique enforces the type's unique properties on after (D-3-22). On a
// create (before nil) it checks every unique property after holds; on an
// update only those whose value changed. For each, in property order so two
// writers never take the same locks in opposite orders, it takes the value's
// advisory lock and then looks for another record (any but self) that holds
// the value. A missing or null value is not checked: many records may leave
// a unique property empty.
func (w *write) checkUnique(ctx context.Context, self uuid.UUID, before, after map[string]any, fixed map[string]string) error {
	for _, field := range w.def.UniqueFields() {
		v, ok := after[field]
		if !ok || v == nil {
			continue
		}
		_, isColumn := fixed[field]
		key, value := canonicalValue(v, isColumn)
		if old, ok := before[field]; ok && old != nil {
			if oldKey, _ := canonicalValue(old, isColumn); oldKey == key {
				continue
			}
		}
		if err := store.LockUnique(ctx, w.tx, w.tenant, w.defID, field, key); err != nil {
			return fmt.Errorf("pipeline: %w", err)
		}
		taken, err := store.UniqueTaken(ctx, w.tx, w.tenant, w.defID, self, field, value, fixed)
		if err != nil {
			return fmt.Errorf("pipeline: %w", err)
		}
		if taken {
			return &ErrUniqueViolation{Field: field}
		}
	}
	return nil
}

// canonicalValue returns a unique value's canonical form, the advisory lock's
// key, and the value to compare. Values jsonb equality calls equal get the
// same key: numbers are exact rationals (1, 1.0 and 1e0 are one key) and
// object keys are sorted. A property stored in a fixed uuid column is
// compared in uuid.String's lowercase form, as the column returns it.
func canonicalValue(v any, column bool) (key string, value any) {
	if s, ok := v.(string); ok && column {
		if id, err := uuid.Parse(s); err == nil {
			s = id.String()
		}
		return canonical(s), s
	}
	return canonical(v), v
}

// canonical writes a decoded JSON value in canonical form.
func canonical(v any) string {
	var b strings.Builder
	writeCanonical(&b, v)
	return b.String()
}

func writeCanonical(b *strings.Builder, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(x))
	case json.Number:
		if r, ok := new(big.Rat).SetString(x.String()); ok {
			b.WriteString(r.RatString())
		} else {
			b.WriteString(x.String())
		}
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			writeCanonical(b, e)
		}
		b.WriteByte(']')
	case map[string]any:
		b.WriteByte('{')
		for i, k := range slices.Sorted(maps.Keys(x)) {
			if i > 0 {
				b.WriteByte(',')
			}
			writeCanonical(b, k)
			b.WriteByte(':')
			writeCanonical(b, x[k])
		}
		b.WriteByte('}')
	default: // a string; normalise decodes every number as json.Number
		j, _ := json.Marshal(x)
		b.Write(j)
	}
}

func optional(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func mapNotFound(err error) error {
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, entitytypes.ErrNotFound) {
		return ErrNotFound
	}
	return fmt.Errorf("pipeline: %w", err)
}

// normalise deep-copies data into the JSON form Validate takes (objects as
// map[string]any, numbers as json.Number), so a caller's Go types and later
// changes never reach the stored row.
func normalise(data map[string]any) (map[string]any, error) {
	if data == nil {
		return map[string]any{}, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("%w: data does not marshal to JSON", ErrInvalidCommand)
	}
	out, err := store.DecodeObject(b)
	if err != nil {
		return nil, fmt.Errorf("%w: data is not a JSON object", ErrInvalidCommand)
	}
	return out, nil
}

// escape escapes one JSON pointer token (RFC 6901).
func escape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}

// appendField adds f unless fe has it, keeping fe sorted by path then code.
func appendField(fe []metadata.FieldError, f metadata.FieldError) []metadata.FieldError {
	for i, g := range fe {
		if g == f {
			return fe
		}
		if g.Path > f.Path || (g.Path == f.Path && g.Code > f.Code) {
			return append(fe[:i:i], append([]metadata.FieldError{f}, fe[i:]...)...)
		}
	}
	return append(fe, f)
}

// deepCopy copies a decoded JSON object.
func deepCopy(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)         // decoded JSON always marshals
	out, _ := store.DecodeObject(b) // and decodes back
	return out
}

// Replayed reports whether actor already has a committed update of recordID
// under idempotency key: a read-only lookup of record_write_keys in tx, which
// must come from tenancy.InTenantTx of actor's tenant. It writes nothing and
// takes no part in a write; a caller whose own check refused a replayed
// command (the workflow writer, whose causation went stale after the write
// committed) uses it to answer with the earlier result instead. A key that
// names another actor's write, a create or another record is not a replay.
func Replayed(ctx context.Context, tx pgx.Tx, actor identity.Principal, recordID uuid.UUID, key string) (bool, error) {
	if key == "" || recordID == uuid.Nil {
		return false, nil
	}
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return false, fmt.Errorf("pipeline: %w", err)
	}
	if actor.TenantID != tenant {
		return false, authz.ErrForbidden
	}
	var k *store.WriteKey
	b := &pgx.Batch{}
	store.QueueWriteKeyByKey(b, tenant, optional(actor.PrincipalID), key, &k)
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		return false, fmt.Errorf("pipeline: read write key: %w", err)
	}
	return k != nil && k.Action == OpUpdate.String() && k.RecordID == recordID, nil
}
