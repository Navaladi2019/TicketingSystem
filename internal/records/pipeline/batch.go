package pipeline

// Batched round trips of a write (TICK-648, PLAN-003 P3-36b). A create
// sends three batches in the caller's tenancy.InTenantTx transaction, whose
// BEGIN already set the tenant:
//
//  1. prefetch: the write key (system writers with a key), the definition id,
//     the current definition and, for a source the authorizer checks, the
//     ACL policy. The caller may queue these in a batch of its own first
//     (QueuePrefetch); Write then sends nothing here.
//  2. checks: for each unique property, its advisory lock then its lookup,
//     in property order; the assignment group lookup; and PostgreSQL's
//     jsonb text of the payload about to be stored (the audit diff's after).
//  3. write: the INSERT of the record with its number, the write key and
//     the audit entry. The number was taken just before, in a transaction of
//     its own that has already committed (D-3-9 as amended 2026-11-10,
//     write.allocate, alloc.go), so no lock on record_counters is held here.
//
// An update keeps its statements one by one; it reads its definition and
// policy through the same prefetch.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// prefetched is what a write reads before anything else.
type prefetched struct {
	// defID is the entity type's id; uuid.Nil when the tenant has none.
	defID uuid.UUID
	// writeKey is the actor's write under cmd's idempotency key, if any.
	writeKey *store.WriteKey
	// createQuiet: the EventSink has nothing to do for a create of this
	// entity type (NoopSink, or CreateTriggerQueuer said so). False, so the
	// sink runs, whenever that is not known.
	createQuiet bool
}

// prefetchKey is the tx memo key of a prefetch: one per entity type, actor,
// source class and idempotency key, the inputs QueuePrefetch reads by.
type prefetchKey struct {
	entity, idemKey string
	actor           uuid.UUID
	checked         bool // the authorizer runs (not a system source)
}

func keyOf(cmd Command) prefetchKey {
	return prefetchKey{entity: cmd.EntityKey, idemKey: cmd.IdempotencyKey, actor: cmd.Actor.PrincipalID, checked: !cmd.Source.system()}
}

// currentQueuer is a DefinitionReader that can queue Current's read on a
// batch, answering later Current calls in tx from the memo.
// entitytypes.Store is one.
type currentQueuer interface {
	QueueCurrent(tx pgx.Tx, b *pgx.Batch, entityKey string)
}

// policyQueuer is an Authorizer that can queue the policy read every later
// check in tx uses. authz.ACL is one.
type policyQueuer interface {
	QueuePolicy(ctx context.Context, tx pgx.Tx, b *pgx.Batch, p identity.Principal, entity string) error
}

// QueuePrefetch queues on b the reads a Write of cmd in tx starts with, so a
// caller can send them in its own first batch with reads of its own. When
// the batch has run, Write sends no prefetch of its own, and Current of
// cmd's entity type in tx answers from the transaction's memo. tx must be the
// tenancy.InTenantTx transaction the batch is sent in.
func (p *Pipeline) QueuePrefetch(ctx context.Context, tx pgx.Tx, b *pgx.Batch, cmd Command) error {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return fmt.Errorf("pipeline: %w", err)
	}
	_, err = p.queuePrefetch(ctx, tx, b, tenant, cmd)
	return err
}

// queuePrefetch queues the prefetch and returns where its results land. The
// results are also remembered in tx under keyOf(cmd) once the batch ran.
func (p *Pipeline) queuePrefetch(ctx context.Context, tx pgx.Tx, b *pgx.Batch, tenant uuid.UUID, cmd Command) (*prefetched, error) {
	pre := &prefetched{}
	// The key read comes first: when it fails, nothing after it runs.
	if cmd.IdempotencyKey != "" {
		store.QueueWriteKeyByKey(b, tenant, optional(cmd.Actor.PrincipalID), cmd.IdempotencyKey, &pre.writeKey)
	}
	// A create asks the sink, in the same read, whether it fires anything.
	trigger, asks := p.sink.(CreateTriggerQueuer)
	switch {
	case cmd.Op == OpCreate && asks:
		var fires bool
		store.QueueDefinitionIDAndFlag(b, tenant, cmd.EntityKey, trigger.CreateTriggeredSQL(), &pre.defID, &fires)
		afterLast(b, func() { pre.createQuiet = !fires })
	default:
		store.QueueDefinitionID(b, tenant, cmd.EntityKey, &pre.defID)
		_, noop := p.sink.(NoopSink)
		pre.createQuiet = cmd.Op == OpCreate && noop
	}
	if cq, ok := p.defs.(currentQueuer); ok {
		cq.QueueCurrent(tx, b, cmd.EntityKey)
	}
	if !cmd.Source.system() {
		if pq, ok := p.authz.(policyQueuer); ok {
			if err := pq.QueuePolicy(ctx, tx, b, cmd.Actor, cmd.EntityKey); err != nil {
				return nil, fmt.Errorf("pipeline: %w", err)
			}
		}
	}
	// Remembered once the last read has run, so a failed batch leaves
	// nothing behind.
	afterLast(b, func() { tenancy.Remember(tx, keyOf(cmd), pre) })
	return pre, nil
}

// prefetch returns cmd's prefetched reads: from tx's memo when the caller
// queued them (QueuePrefetch), else from a batch it sends now. A memo entry
// serves one write only: a second write in tx with the same key must see
// the first one's write key.
func (p *Pipeline) prefetch(ctx context.Context, tx pgx.Tx, tenant uuid.UUID, cmd Command) (*prefetched, error) {
	if v, ok := tenancy.Recall(tx, keyOf(cmd)); ok {
		tenancy.Forget(tx, keyOf(cmd))
		return v.(*prefetched), nil //nolint:forcetypeassert // only queuePrefetch stores under prefetchKey
	}
	b := &pgx.Batch{}
	pre, err := p.queuePrefetch(ctx, tx, b, tenant, cmd)
	if err != nil {
		return nil, err
	}
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	return pre, nil
}

// createChecks is what the checks batch of a create read.
type createChecks struct {
	taken []bool // per queued unique field, in order
	// fields are the unique properties checked, in order.
	fields    []string
	groupLive bool
}

// checkCreate sends the checks batch of a create of rec (after: its payload
// with defaults) and returns the first refusal in the order the unbatched
// write made them: a taken unique value, then a removed assignment group.
// It returns the payload preview, the audit diff's after.
func (w *write) checkCreate(ctx context.Context, rec *store.Record, after map[string]any, fixed map[string]string) ([]byte, error) {
	b := &pgx.Batch{}
	// taken is never appended past its capacity, so the pointers the
	// queued lookups hold into it stay valid.
	unique := w.def.UniqueFields()
	c := &createChecks{taken: make([]bool, 0, len(unique)), fields: make([]string, 0, len(unique))}
	// Unique properties in property order, so two writers never take the
	// value locks in opposite orders (D-3-22). The lookup follows its lock in
	// the batch, so its snapshot is taken after the lock was granted.
	for _, field := range unique {
		v, ok := after[field]
		if !ok || v == nil {
			continue
		}
		_, isColumn := fixed[field]
		key, value := canonicalValue(v, isColumn)
		store.QueueLockUnique(b, w.tenant, w.defID, field, key)
		c.fields = append(c.fields, field)
		c.taken = append(c.taken, false)
		if err := store.QueueUniqueTaken(b, w.tenant, w.defID, uuid.Nil, field, value, fixed, &c.taken[len(c.taken)-1]); err != nil {
			return nil, fmt.Errorf("pipeline: %w", err)
		}
	}
	// A create changes assignment_group_id from nothing: any value is
	// checked (ADR-004-E5 D6).
	if rec.AssignmentGroupID != nil {
		store.QueueGroupLive(b, w.tenant, *rec.AssignmentGroupID, &c.groupLive)
	}
	var preview json.RawMessage
	if err := store.QueuePayloadPreview(b, rec, fixed, &preview); err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	if err := w.tx.SendBatch(ctx, b).Close(); err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	for i, taken := range c.taken {
		if taken {
			return nil, &ErrUniqueViolation{Field: c.fields[i]}
		}
	}
	if rec.AssignmentGroupID != nil && !c.groupLive {
		return nil, ErrAssignmentGroupRemoved
	}
	return preview, nil
}

// writeCreate sends the write batch of a create: the record, with the
// number rec.Number already taken, the write key and the audit entry, in
// that order. after is the payload preview the audit diff is built from; the
// INSERT returns the stored payload, which must be the same bytes.
func (w *write) writeCreate(ctx context.Context, rec *store.Record, after []byte) (Result, error) {
	auditID, err := uuid.NewV7()
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: new audit id: %w", err)
	}
	fixed := w.def.FixedColumns()
	format := w.format()
	var (
		got      store.Inserted
		numbered store.Numbered
	)
	// A `gapless` type takes its number here, first in this write batch and
	// so last inside this transaction (ADR-005 L1, TICK-648): the counter row
	// stays locked until this transaction ends, and a rollback gives the
	// number back. A `gaps` type arrives with the number its allocation
	// already committed.
	gapless := w.def.NumberingMode() == metadata.NumberingGapless
	queue := func() (*pgx.Batch, error) {
		b := &pgx.Batch{}
		var queueErr error
		if gapless {
			queueErr = store.QueueInsertNumbered(b, rec, w.def.NumberPrefix(), format, fixed, &numbered)
		} else {
			queueErr = store.QueueInsertRecord(b, rec, fixed, &got)
		}
		if queueErr != nil {
			return nil, fmt.Errorf("pipeline: %w", queueErr)
		}
		if w.cmd.IdempotencyKey != "" {
			// Claimed before the audit entry, so a concurrent duplicate by the
			// same principal waits on and fails at record_write_keys_actor_key
			// (D-5-3, TICK-626).
			store.QueueInsertWriteKey(b, &store.WriteKey{
				TenantID: w.tenant, IdempotencyKey: w.cmd.IdempotencyKey, DefinitionID: rec.EntityDefinitionID,
				Action: w.cmd.Op.String(), RecordID: rec.ID, Version: rec.Version,
				SchemaVersion: rec.SchemaVersion, AuditEntryID: auditID,
				ActorPrincipalID: optional(w.cmd.Actor.PrincipalID),
			})
		}
		e := recordEntry{id: auditID, action: audit.ActionCreate, rec: rec, after: after, secretProps: w.def.SecretProperties()}
		entry, err := recordAuditEntry(ctx, w.cmd, e)
		if err != nil {
			return nil, err
		}
		if _, err := audit.QueueAppend(ctx, w.tx, b, entry); err != nil {
			return nil, mapAppendError(err)
		}
		wrapLast(b, func(err error) error { return &mappedErr{mapAppendError(err)} })
		return b, nil
	}
	b, err := queue()
	if err != nil {
		return Result{}, err
	}
	err = w.tx.SendBatch(ctx, b).Close()
	// While both unique indexes of record_counters exist (ADR-005 M2 to
	// M3), a type's first create can lose the race for its counter row with
	// 23505 on the index the upsert does not name (ADR-005-E1). The batch
	// then ran nothing past the upsert; rolled back to the savepoint before
	// it, the batch is sent once more and its upsert, seeing the winner's
	// committed row (READ COMMITTED), takes the UPDATE arm. Once, and for
	// no other error.
	// TODO(TODO-S17-02): remove this retry (and the gapless savepoint) with
	// M3 (S18/R2): with one unique index left, store.CounterRace never fires.
	if gapless && store.CounterRace(err) {
		w.p.allocMetrics().retried(ctx, metadata.NumberingGapless)
		if err = store.RetryNumbered(ctx, w.tx); err != nil {
			return Result{}, fmt.Errorf("pipeline: %w", err)
		}
		if b, err = queue(); err != nil {
			return Result{}, err
		}
		err = w.tx.SendBatch(ctx, b).Close()
	}
	if err != nil {
		var m *mappedErr
		switch {
		case errors.Is(err, store.ErrDuplicateKey):
			return Result{}, &ErrIdempotencyConflict{Concurrent: true}
		case errors.As(err, &m):
			return Result{}, m.err
		}
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	if gapless {
		want, err := format.Render(w.def.NumberPrefix(), numbered.N)
		if err != nil {
			return Result{}, fmt.Errorf("pipeline: %w", err)
		}
		if numbered.Number != want {
			return Result{}, fmt.Errorf("pipeline: stored number %q, want %q", numbered.Number, want)
		}
		rec.Number = numbered.Number
		got = store.Inserted{Number: numbered.Number, Payload: numbered.Payload, CreatedAt: numbered.CreatedAt, UpdatedAt: numbered.UpdatedAt}
	}
	if got.Number != rec.Number {
		return Result{}, fmt.Errorf("pipeline: stored number %q, want %q", got.Number, rec.Number)
	}
	if !bytes.Equal(got.Payload, after) {
		return Result{}, errors.New("pipeline: the stored payload is not the one the audit entry was built from")
	}
	rec.CreatedAt, rec.UpdatedAt = got.CreatedAt, got.UpdatedAt
	// The stored payload, as PostgreSQL returns it, is the record's data
	// from here on: what a read in this transaction would decode.
	data, err := store.DecodeObject(got.Payload)
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: decode stored payload: %w", err)
	}
	for prop := range fixed {
		delete(data, prop)
	}
	rec.Data = data
	ev := Event{Op: w.cmd.Op, EntityDefinitionID: rec.EntityDefinitionID, RecordID: rec.ID, Version: rec.Version,
		Causation: w.cmd.Causation, ActorID: w.cmd.Actor.PrincipalID, Source: w.cmd.Source, SourceRef: w.cmd.SourceRef,
		WorkflowRef: w.cmd.WorkflowRef}
	if !w.sinkQuiet {
		if err := w.p.sink.RecordChanged(ctx, w.tx, ev); err != nil {
			return Result{}, fmt.Errorf("pipeline: event sink: %w", err)
		}
	}
	// Get of this record later in tx answers with the row as stored, with
	// no round trip (TICK-648).
	tenancy.Remember(w.tx, storedKey{rec.ID}, rec)
	return Result{RecordID: rec.ID, Number: rec.Number, Version: rec.Version, SchemaVersion: rec.SchemaVersion}, nil
}

// atCommit reports whether this create queues its write for the round trip
// that commits w.tx (Command.InsertAtCommit, TICK-692).
func (w *write) atCommit() bool {
	if !w.cmd.InsertAtCommit || w.cmd.Op != OpCreate || w.def.NumberingMode() == metadata.NumberingGapless {
		return false
	}
	// A create the sink has work for (a published definition it triggers)
	// sends its statements after the record's, so it writes in a round trip
	// of its own (TODO-S10-02 as narrowed by P3-42).
	if !w.sinkQuiet {
		return false
	}
	_, ok := tenancy.QueueAtCommit(w.tx)
	return ok
}

// writeCreateAtCommit is writeCreate for Command.InsertAtCommit: the same
// statements, in the same order, queued on the batch that commits w.tx
// (TICK-692), so the record, its write key and its audit entry are sent with
// the idempotency completion and COMMIT in one round trip. Nothing is read
// back: the record as stored is the one built here, from the number taken,
// the allocation's instant and the payload preview after, and the INSERT
// fails unless the stored row is exactly that
// (store.QueueInsertRecordAsExpected). A failure is mapped as writeCreate and
// Write map it, inside *ErrAtVersion, and comes out of tenancy.InTenantTx.
func (w *write) writeCreateAtCommit(ctx context.Context, rec *store.Record, after []byte) (Result, error) {
	auditID, err := uuid.NewV7()
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: new audit id: %w", err)
	}
	fixed := w.def.FixedColumns()
	// Built apart and moved onto the commit batch only once every statement
	// is queued, so a failure here leaves the commit batch as it was.
	b := &pgx.Batch{}
	if err := store.QueueInsertRecordAsExpected(b, rec, fixed, after); err != nil {
		return Result{}, fmt.Errorf("pipeline: %w", err)
	}
	if w.cmd.IdempotencyKey != "" {
		store.QueueInsertWriteKey(b, &store.WriteKey{
			TenantID: w.tenant, IdempotencyKey: w.cmd.IdempotencyKey, DefinitionID: rec.EntityDefinitionID,
			Action: w.cmd.Op.String(), RecordID: rec.ID, Version: rec.Version,
			SchemaVersion: rec.SchemaVersion, AuditEntryID: auditID,
			ActorPrincipalID: optional(w.cmd.Actor.PrincipalID),
		})
	}
	e := recordEntry{id: auditID, action: audit.ActionCreate, rec: rec, after: after, secretProps: w.def.SecretProperties()}
	entry, err := recordAuditEntry(ctx, w.cmd, e)
	if err != nil {
		return Result{}, err
	}
	if _, err := audit.QueueAppend(ctx, w.tx, b, entry); err != nil {
		return Result{}, mapAppendError(err)
	}
	wrapLast(b, func(err error) error { return &mappedErr{mapAppendError(err)} })
	// The stored payload is the preview (the INSERT asserts it): the
	// record's data from here on, as writeCreate decodes it.
	data, err := store.DecodeObject(after)
	if err != nil {
		return Result{}, fmt.Errorf("pipeline: decode payload preview: %w", err)
	}
	for prop := range fixed {
		delete(data, prop)
	}
	// No RecordChanged: this path runs only for a create the sink said it
	// has nothing to do for (atCommit, sinkQuiet), and anything the sink
	// sent would have to follow the INSERT queued here.
	commit, ok := tenancy.QueueAtCommit(w.tx)
	if !ok {
		return Result{}, errors.New("pipeline: the transaction has no commit batch")
	}
	for _, qq := range b.QueuedQueries {
		inner := qq.Fn
		qq.Fn = func(br pgx.BatchResults) error {
			var err error
			if inner != nil {
				err = inner(br)
			} else {
				_, err = br.Exec()
			}
			if err != nil {
				return w.mapAtCommit(err)
			}
			return nil
		}
		commit.QueuedQueries = append(commit.QueuedQueries, qq)
	}
	rec.Data = data
	// Get of this record later in tx answers with the row as it will be
	// stored, with no round trip.
	tenancy.Remember(w.tx, storedKey{rec.ID}, rec)
	return Result{RecordID: rec.ID, Number: rec.Number, Version: rec.Version, SchemaVersion: rec.SchemaVersion}, nil
}

// mapAtCommit maps the failure of a statement writeCreateAtCommit queued as
// writeCreate and Write map the same failure of the write batch.
func (w *write) mapAtCommit(err error) error {
	var m *mappedErr
	switch {
	case errors.Is(err, store.ErrDuplicateKey):
		err = &ErrIdempotencyConflict{Concurrent: true}
	case errors.As(err, &m):
		err = m.err
	default:
		err = fmt.Errorf("pipeline: %w", err)
	}
	err = &ErrAtVersion{SchemaVersion: w.defVersion, FixedColumns: w.def.FixedColumns(), StateProperty: w.def.StateProperty(), Err: err}
	if retryable(err) {
		return fmt.Errorf("%w: %w", ErrRetryable, err)
	}
	return err
}

// storedKey is the tx memo key of a record this transaction created: the row
// as stored, which Get returns. An update of the record drops it.
type storedKey struct{ id uuid.UUID }

// mappedErr carries a batch statement's error, already mapped to Write's
// contract, out through pgx.BatchResults.Close.
type mappedErr struct{ err error }

func (m *mappedErr) Error() string { return m.err.Error() }

func (m *mappedErr) Unwrap() error { return m.err }

// afterLast runs fn once the last statement queued on b has succeeded.
func afterLast(b *pgx.Batch, fn func()) {
	wrapLast(b, func(err error) error { return err })
	qq := b.QueuedQueries[len(b.QueuedQueries)-1]
	inner := qq.Fn
	qq.Fn = func(br pgx.BatchResults) error {
		if err := inner(br); err != nil {
			return err
		}
		fn()
		return nil
	}
}

// wrapLast maps the error of the last statement queued on b with wrap.
func wrapLast(b *pgx.Batch, wrap func(error) error) {
	qq := b.QueuedQueries[len(b.QueuedQueries)-1]
	inner := qq.Fn
	qq.Fn = func(br pgx.BatchResults) error {
		var err error
		if inner != nil {
			err = inner(br)
		} else {
			_, err = br.Exec()
		}
		if err != nil {
			return wrap(err)
		}
		return nil
	}
}
