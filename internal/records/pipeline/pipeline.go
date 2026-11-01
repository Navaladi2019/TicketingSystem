// Package pipeline is the record write pipeline (PLAN-003 D-3-6): Write is the
// single way to create or update a record. In the caller's tenant transaction
// it loads the record and the entity type's current definition, authorises,
// applies the merge patch and defaults, validates against the JSON Schema,
// writes the row with an optimistic version check, writes one audit row and
// calls the EventSink. Nothing commits unless every stage succeeds: the
// caller's tenancy.InTenantTx commits or rolls back.
//
// The tenant is never taken from the command. Write reads it from the
// transaction (tenancy.TenantOf) and refuses an actor of another tenant.
//
// A create takes the next per-tenant, per-type number from record_counters
// (D-3-9). Every write checks the type's state model and its unique
// properties (D-3-22) (P3-11). The cascade depth and cycle checks are P3-12
// (TICK-88), which writes its rejection audit row on the TxRunner (D-3-7a).
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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
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

// String is the audit_log action of the write.
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

// Source is who made the write (audit_log.source).
type Source string

// The sources audit_log_source_check allows.
const (
	SourceUser     Source = "user"
	SourceRule     Source = "rule"
	SourceWorkflow Source = "workflow"
	SourceFunction Source = "function"
	SourceAI       Source = "ai"
	SourceSystem   Source = "system"
)

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
	Actor     identity.Principal
	Source    Source
	Causation outbox.Causation
	// IdempotencyKey is for system writers only (D-3-13): a replay with the
	// same key, entity type, op and (on an update) record returns the earlier
	// Result and writes nothing. The same key on a different write is
	// *ErrIdempotencyConflict.
	IdempotencyKey string
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
	Version   int64
	Causation outbox.Causation
}

// EventSink is told about every write, in the write's transaction (D-3-14).
// An error rolls the whole write back. A sink that enqueues outbox rows uses
// outbox.EnqueueAtVersion with ev.Version, never outbox.Enqueue, so the write
// bumps the version exactly once.
type EventSink interface {
	RecordChanged(ctx context.Context, tx pgx.Tx, ev Event) error
}

// NoopSink is the S9 EventSink: it does nothing. S11b replaces it (P3-42).
type NoopSink struct{}

// RecordChanged does nothing.
func (NoopSink) RecordChanged(context.Context, pgx.Tx, Event) error { return nil }

// DefinitionReader reads compiled definitions in the caller's transaction.
// entitytypes.Store implements it (P3-05a).
type DefinitionReader interface {
	Version(ctx context.Context, tx pgx.Tx, entityKey string, version int) (*metadata.Compiled, error)
	Current(ctx context.Context, tx pgx.Tx, entityKey string) (*metadata.Compiled, int, error)
}

// TxRunner runs fn in a tenant transaction of its own. P3-12 (TICK-88) writes
// a rejected cascade's audit row on it, so the row survives the caller's
// rollback (D-3-7a).
type TxRunner interface {
	InTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(pgx.Tx) error) error
}

// PoolRunner is the TxRunner over a pool, through tenancy.InTenantTx.
type PoolRunner struct{ Pool *pgxpool.Pool }

// InTenantTx implements TxRunner.
func (r PoolRunner) InTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(pgx.Tx) error) error {
	return tenancy.InTenantTx(ctx, r.Pool, tenantID, fn)
}

// Clock gives the time written to updated_at and audit_log.at.
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
// authz.ErrForbidden, *ErrValidation, *ErrIllegalTransition,
// *ErrUniqueViolation, ErrVersionConflict or *ErrIdempotencyConflict for a
// refused write, and a wrapped error for a database failure. On any error the
// caller rolls tx back; after a concurrent *ErrIdempotencyConflict or a
// database error tx is unusable.
func (p *Pipeline) Write(ctx context.Context, tx pgx.Tx, cmd Command) (Result, error) {
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

	if cmd.IdempotencyKey != "" {
		res, found, err := replay(ctx, tx, tenant, cmd)
		if err != nil || found {
			return res, err
		}
	}

	defID, err := store.DefinitionID(ctx, tx, tenant, cmd.EntityKey)
	if err != nil {
		return Result{}, mapNotFound(err)
	}
	def, defVersion, err := p.defs.Current(ctx, tx, cmd.EntityKey)
	if err != nil {
		return Result{}, mapNotFound(err)
	}
	data, err := normalise(cmd.Data)
	if err != nil {
		return Result{}, err
	}

	w := write{p: p, tx: tx, tenant: tenant, cmd: cmd, defID: defID, def: def, defVersion: defVersion}
	var res Result
	switch cmd.Op {
	case OpCreate:
		res, err = w.create(ctx, data)
	case OpUpdate:
		res, err = w.update(ctx, data)
	}
	if err != nil {
		return Result{}, err
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
	if c.IdempotencyKey != "" && (c.Source == SourceUser || len(c.IdempotencyKey) > maxIdempotencyKeyLen) {
		return bad("an idempotency key is for system writers only, at most %d bytes", maxIdempotencyKeyLen)
	}
	if c.Causation.Depth < 0 || (c.Causation.Depth > 0 && c.Causation.OriginID == uuid.Nil) {
		return bad("causation depth without an origin, or negative")
	}
	return nil
}

// replay returns the earlier Result of a write with cmd's idempotency key.
// The stored write must be the same entity type and op, and on an update the
// same record; otherwise the key names another write, *ErrIdempotencyConflict.
func replay(ctx context.Context, tx pgx.Tx, tenant uuid.UUID, cmd Command) (Result, bool, error) {
	kw, err := store.AuditByKey(ctx, tx, tenant, cmd.IdempotencyKey)
	if errors.Is(err, store.ErrNotFound) {
		return Result{}, false, nil
	}
	if err != nil {
		return Result{}, false, fmt.Errorf("pipeline: %w", err)
	}
	if kw.EntityKey != cmd.EntityKey || kw.Action != cmd.Op.String() || (cmd.Op == OpUpdate && kw.RecordID != cmd.RecordID) {
		return Result{}, false, &ErrIdempotencyConflict{}
	}
	a := kw.After
	return Result{RecordID: kw.RecordID, Number: a.Number, Version: a.Version, SchemaVersion: a.SchemaVersion}, true, nil
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
}

func (w *write) create(ctx context.Context, after map[string]any) (Result, error) {
	if err := w.p.authz.CanWrite(ctx, w.cmd.Actor, w.cmd.EntityKey, nil, after); err != nil {
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
	if err := w.checkUnique(ctx, uuid.Nil, nil, after, fixed); err != nil {
		return Result{}, err
	}
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
	// The number is taken after every check, as the last statement before
	// the record, audit and outbox inserts, so the counter row's lock is held
	// only for the rest of the write (D-3-9).
	n, err := store.NextNumber(ctx, w.tx, w.tenant, w.defID)
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	rec.Number = store.FormatNumber(w.def.NumberPrefix(), n)
	if err := store.Insert(ctx, w.tx, rec); err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	return w.finish(ctx, rec, nil, rec.Payload(fixed))
}

func (w *write) update(ctx context.Context, patch map[string]any) (Result, error) {
	rec, err := store.Get(ctx, w.tx, w.tenant, w.cmd.RecordID)
	if err != nil {
		return Result{}, mapNotFound(err)
	}
	if rec.EntityKey != w.cmd.EntityKey {
		return Result{}, ErrNotFound
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
	beforeSnap := &store.Snapshot{Number: rec.Number, Version: rec.Version, SchemaVersion: rec.SchemaVersion, Data: before}

	after := mergePatch(deepCopy(before), patch)
	if err := w.p.authz.CanWrite(ctx, w.cmd.Actor, w.cmd.EntityKey, before, after); err != nil {
		return Result{}, err
	}
	after = w.def.ApplyDefaults(after)
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
	if err := w.checkUnique(ctx, rec.ID, before, after, fixed); err != nil {
		return Result{}, err
	}
	if err := rec.Split(after, fixed); err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	rec.SchemaVersion = w.defVersion
	rec.UpdatedAt = w.p.clock.Now()
	w.causation(rec)
	v, err := store.Update(ctx, w.tx, rec, w.cmd.ExpectedVersion)
	if errors.Is(err, store.ErrStale) {
		return Result{}, ErrVersionConflict
	}
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	rec.Version = v
	return w.finish(ctx, rec, beforeSnap, rec.Payload(fixed))
}

// finish writes the primary audit row and calls the sink.
func (w *write) finish(ctx context.Context, rec *store.Record, before *store.Snapshot, after map[string]any) (Result, error) {
	auditID, err := uuid.NewV7()
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: new audit id: %w", err)
	}
	a := &store.Audit{
		TenantID: w.tenant, ID: auditID, RecordID: rec.ID, Action: w.cmd.Op.String(),
		ActorID: optional(w.cmd.Actor.PrincipalID), Source: string(w.cmd.Source),
		Before:            before,
		After:             &store.Snapshot{Number: rec.Number, Version: rec.Version, SchemaVersion: rec.SchemaVersion, Data: after},
		CausationOriginID: rec.CausationOriginID, CausationDepth: rec.CausationDepth,
		At: rec.UpdatedAt,
	}
	if w.cmd.IdempotencyKey != "" {
		a.IdempotencyKey = &w.cmd.IdempotencyKey
	}
	if err := store.InsertAudit(ctx, w.tx, a); err != nil {
		if errors.Is(err, store.ErrDuplicateKey) {
			return Result{}, &ErrIdempotencyConflict{Concurrent: true}
		}
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	ev := Event{Op: w.cmd.Op, EntityDefinitionID: rec.EntityDefinitionID, RecordID: rec.ID, Version: rec.Version, Causation: w.cmd.Causation}
	if err := w.p.sink.RecordChanged(ctx, w.tx, ev); err != nil {
		return Result{}, fmt.Errorf("pipeline: event sink: %w", err)
	}
	return Result{RecordID: rec.ID, Number: rec.Number, Version: rec.Version, SchemaVersion: rec.SchemaVersion}, nil
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
