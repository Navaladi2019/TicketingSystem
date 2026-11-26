package schemabackfill

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/barrier"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The registered activity names.
const (
	AdmitActivity   = "schema_backfill.Admit"
	BarrierActivity = "schema_backfill.Barrier"
	FillActivity    = "schema_backfill.Fill"
	PageActivity    = "schema_backfill.Page"
	EndActivity     = "schema_backfill.End"
	R6Activity      = "schema_backfill.CheckR6"
)

// ErrTypeNoJob is the ApplicationError type of a workflow id, input or job
// row that is not a schema backfill's: an id that does not parse, a start
// payload naming another definition, or no schema_backfills row for the id
// under RLS (D2). It is non-retryable: a retry reads the same id and row. It
// is not on the worker guard's allow-list, so the guard records it as
// nava.Opaque; the type matters only to this worker's own log.
const ErrTypeNoJob = "nava.schema_backfill.NoJob"

// ActorSystem is the audit actor_system of every entry the job writes (D2),
// pipeline.ActorSchemaBackfill.
const ActorSystem = "schema-backfill"

// DefaultPageSize is how many pending ids one page reads (D7).
const DefaultPageSize = 100

// DefaultSlowAfter is how long the barrier waits before it logs
// schema_backfill_barrier_slow.
const DefaultSlowAfter = 15 * time.Minute

// The job row states (contract C1).
const (
	statePending       = "pending"
	stateFilling       = "filling"
	stateRunning       = "running"
	stateDone          = "done"
	stateFailedRecords = "failed_records"
	stateSuperseded    = "superseded"
)

// oneActiveIndex is the partial unique index that admits one filling or
// running job per tenant (D7, P10-01).
const oneActiveIndex = "schema_backfills_one_active_per_tenant"

// AdmitResult is Admit's result. Admitted is set when the job holds its
// tenant's slot (filling or running); Ended when it is over (superseded, or
// already ended) and the workflow completes. Neither: another job of the
// tenant is active, so this one stays pending and the workflow waits on a
// timer before it asks again. The values are the job's definition id:
// references only, as the worker guard requires of every result.
type AdmitResult struct {
	Admitted *uuid.UUID `json:"admitted,omitempty"`
	Ended    *uuid.UUID `json:"ended,omitempty"`
}

// FillResult is Fill's result: Ended is set when the job is no longer this
// workflow's to fill (superseded, or gone past running), and the workflow
// completes.
type FillResult struct {
	Ended *uuid.UUID `json:"ended,omitempty"`
}

// Cursor is Page's input: the last pending id the previous page handled, or
// none to start a pass from the first id.
type Cursor struct {
	After *uuid.UUID `json:"after,omitempty"`
}

// PageResult is Page's result. Next is the cursor of the next page; Ended is
// set when the job was superseded (or is no longer running) and the workflow
// completes. Neither: the queue is empty, and the workflow ends the job. The
// counters are never in it: each record's transaction adds to the job row.
type PageResult struct {
	Next  *uuid.UUID `json:"next,omitempty"`
	Ended *uuid.UUID `json:"ended,omitempty"`
}

// EndResult is End's result: FailedRecords is set when the job ended
// failed_records, so the workflow runs the R6 check after its timer.
type EndResult struct {
	FailedRecords *uuid.UUID `json:"failed_records,omitempty"`
}

// Activities are the workflow's activities on a nava_worker pool. Register
// them with Register. Metrics and Log may be nil.
type Activities struct {
	Pool *pgxpool.Pool
	// Materializer rewrites one record (recordports.Materializer).
	Materializer Materializer
	Metrics      *Metrics
	Log          *slog.Logger
	// PageSize is how many pending ids a page reads; 0 is DefaultPageSize.
	PageSize int
	// SlowAfter is the barrier's slow threshold; 0 is DefaultSlowAfter.
	SlowAfter time.Duration

	// afterRecord, when set, runs after each record's transaction has
	// committed; an error fails the page, which Temporal retries (test-only:
	// CI-11's retry after a commit).
	afterRecord func(ctx context.Context, record uuid.UUID) error
	// beforeAdmit, when set, runs in Admit before the pending → filling
	// UPDATE (test-only: it lines two admissions up).
	beforeAdmit func(ctx context.Context)
}

// Register adds the activities to r under their fixed names.
func (a *Activities) Register(r worker.ActivityRegistry) {
	r.RegisterActivityWithOptions(a.Admit, activity.RegisterOptions{Name: AdmitActivity})
	r.RegisterActivityWithOptions(a.Barrier, activity.RegisterOptions{Name: BarrierActivity})
	r.RegisterActivityWithOptions(a.Fill, activity.RegisterOptions{Name: FillActivity})
	r.RegisterActivityWithOptions(a.Page, activity.RegisterOptions{Name: PageActivity})
	r.RegisterActivityWithOptions(a.End, activity.RegisterOptions{Name: EndActivity})
	r.RegisterActivityWithOptions(a.CheckR6, activity.RegisterOptions{Name: R6Activity})
}

// noJob is the non-retryable error of a job that is not one (ErrTypeNoJob).
// Its message is a constant: it names no id.
func noJob(msg string) error {
	return temporal.NewNonRetryableApplicationError("schema backfill: "+msg, ErrTypeNoJob, nil)
}

// jobOf is the calling workflow's job, from its id only (D2). The tenant must
// also be the one the worker guard verified for the activity: an activity
// with no verified tenant is refused, not trusted (defence in depth; the
// guard refuses a missing nava-tenant header first).
func jobOf(ctx context.Context) (Job, error) {
	j, err := ParseWorkflowID(activity.GetInfo(ctx).WorkflowExecution.ID)
	if err != nil {
		return Job{}, noJob("workflow id is not a job's")
	}
	if tenant, ok := workerguard.TenantFromContext(ctx); !ok || tenant != j.Tenant {
		return Job{}, noJob("activity tenant is missing or not the workflow id's")
	}
	return j, nil
}

// The job's SQL. Every statement names the tenant as $1 as well as RLS doing
// so (D2), and the job's key as $2 (entity_definition_id) and $3
// (target_version).
const (
	// lockJobSQL reads the job row and locks it for the transaction.
	lockJobSQL = `SELECT state, default_keys, causation_audit_id, rewritten, skipped, failed
FROM schema_backfills WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3 FOR UPDATE`
	// newerSQL: a newer job exists for the type (D8).
	newerSQL = `SELECT EXISTS (SELECT 1 FROM schema_backfills
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version > $3)`
	// admitSQL is the admission (D7, P10-06): the partial unique index
	// schema_backfills_one_active_per_tenant refuses it with 23505 while
	// another job of the tenant is filling or running.
	admitSQL = `UPDATE schema_backfills SET state = 'filling', started_at = now()
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3 AND state = 'pending'`
	supersedeSQL = `UPDATE schema_backfills SET state = 'superseded', finished_at = now()
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3`
	dropPendingSQL = `DELETE FROM schema_backfill_pending
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3`
	dropFailuresSQL = `DELETE FROM schema_backfill_failures
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3`
	// fillSQL queues every record of the type that lacks one of the job's
	// default keys (D7). The keys are the job row's, never recomputed.
	fillSQL = `INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id)
SELECT r.tenant_id, r.entity_definition_id, $3, r.id FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND NOT (r.data ?& $4::text[])
ON CONFLICT DO NOTHING`
	runSQL = `UPDATE schema_backfills SET state = 'running'
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3 AND state = 'filling'`
	// pageContextSQL reads, once per page, the job's state and causation and
	// the type's key.
	pageContextSQL = `SELECT b.state, b.causation_audit_id, d.key
FROM schema_backfills b JOIN entity_definitions d ON d.tenant_id = b.tenant_id AND d.id = b.entity_definition_id
WHERE b.tenant_id = $1 AND b.entity_definition_id = $2 AND b.target_version = $3`
	// causationDepthSQL is the depth of the job's causation audit entry.
	causationDepthSQL = `SELECT causation_depth FROM audit_log WHERE tenant_id = $1 AND id = $2`
	// pageSQL reads the next pending ids with no row lock (D7).
	pageSQL = `SELECT record_id FROM schema_backfill_pending
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3 AND ($4::uuid IS NULL OR record_id > $4)
ORDER BY record_id LIMIT $5`
	// takePendingSQL removes one id from the queue, in the record's own
	// transaction; it locks the queue row, so a second attempt at the same
	// id waits for the first and then finds nothing to do.
	takePendingSQL = `DELETE FROM schema_backfill_pending
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3 AND record_id = $4`
	countSQL = `UPDATE schema_backfills SET rewritten = rewritten + $4, skipped = skipped + $5
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3`
	failSQL = `INSERT INTO schema_backfill_failures (tenant_id, entity_definition_id, target_version, record_id, code)
VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`
	failCountSQL = `UPDATE schema_backfills SET failed = failed + 1
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3`
	hasFailuresSQL = `SELECT EXISTS (SELECT 1 FROM schema_backfill_failures
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3)`
	endSQL = `UPDATE schema_backfills SET state = $4, finished_at = now()
WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3`
	// r6SQL reads, for the R6 check, the job's state, whether a newer job
	// exists, and the type's key.
	r6SQL = `SELECT b.state, d.key, EXISTS (SELECT 1 FROM schema_backfills n
	WHERE n.tenant_id = b.tenant_id AND n.entity_definition_id = b.entity_definition_id AND n.target_version > b.target_version)
FROM schema_backfills b JOIN entity_definitions d ON d.tenant_id = b.tenant_id AND d.id = b.entity_definition_id
WHERE b.tenant_id = $1 AND b.entity_definition_id = $2 AND b.target_version = $3`
)

// jobRow is the job row as lockJobSQL reads it.
type jobRow struct {
	state                      string
	defaultKeys                []string
	causation                  *uuid.UUID
	rewritten, skipped, failed int64
}

// inJob runs fn in an InTenantTx of the job's tenant with the job row read
// and locked. A job with no row under RLS is ErrTypeNoJob (D2).
func (a *Activities) inJob(ctx context.Context, j Job, fn func(tx pgx.Tx, row jobRow) error) error {
	return tenancy.InTenantTx(ctx, a.Pool, j.Tenant, func(tx pgx.Tx) error {
		var row jobRow
		err := tx.QueryRow(ctx, lockJobSQL, j.Tenant, j.Definition, j.Version).
			Scan(&row.state, &row.defaultKeys, &row.causation, &row.rewritten, &row.skipped, &row.failed)
		if errors.Is(err, pgx.ErrNoRows) {
			return noJob("no schema_backfills row for the workflow id")
		}
		if err != nil {
			return fmt.Errorf("read job: %w", err)
		}
		return fn(tx, row)
	})
}

// terminal reports whether a job in state s is over.
func terminal(s string) bool {
	return s == stateDone || s == stateFailedRecords || s == stateSuperseded
}

// Admit is the job's admission (D7, D8; P10-05 step 1, P10-06 items 1 and
// 2), in one transaction with the job row locked:
//
//   - no row under RLS: ErrTypeNoJob, non-retryable;
//   - a newer job for the type: this one is superseded (supersede), Ended;
//   - already over: Ended; already filling or running (a retried attempt
//     whose commit was lost): Admitted;
//   - pending: the pending → filling UPDATE, with started_at and the
//     "backfill started" audit entry. A 23505 on
//     schema_backfills_one_active_per_tenant means another job of the
//     tenant is active: nothing changes and neither field is set, so the
//     workflow waits and asks again. It is not an error.
func (a *Activities) Admit(ctx context.Context) (AdmitResult, error) {
	j, err := jobOf(ctx)
	if err != nil {
		return AdmitResult{}, err
	}
	def := j.Definition
	var res AdmitResult
	err = a.inJob(ctx, j, func(tx pgx.Tx, row jobRow) error {
		if terminal(row.state) {
			res.Ended = &def
			return nil
		}
		gone, err := a.supersedeIfNewer(ctx, tx, j, row)
		if err != nil || gone {
			if gone {
				res.Ended = &def
			}
			return err
		}
		if row.state != statePending {
			res.Admitted = &def
			return nil
		}
		if a.beforeAdmit != nil {
			a.beforeAdmit(ctx)
		}
		if _, err := tx.Exec(ctx, admitSQL, j.Tenant, j.Definition, j.Version); err != nil {
			return err
		}
		if err := a.auditJob(ctx, tx, j, row.causation, "backfill started",
			jobDoc(j, statePending, nil), jobDoc(j, stateFilling, nil)); err != nil {
			return err
		}
		res.Admitted = &def
		return nil
	})
	if pe := (*pgconn.PgError)(nil); errors.As(err, &pe) && pe.Code == "23505" && pe.ConstraintName == oneActiveIndex {
		return AdmitResult{}, nil
	}
	if err != nil {
		return AdmitResult{}, a.fail(ctx, "admit", err)
	}
	if res.Admitted != nil {
		a.logger().InfoContext(ctx, "schema backfill admitted", a.attrs(ctx, j)...)
	}
	return res, nil
}

// supersedeIfNewer supersedes the job when a newer one exists for its type
// (D8; P10-06 item 1), in tx with the row locked: state superseded,
// finished_at, its pending and failure rows deleted, and the "backfill
// finished" audit entry with state superseded. It reports whether it did.
func (a *Activities) supersedeIfNewer(ctx context.Context, tx pgx.Tx, j Job, row jobRow) (bool, error) {
	var newer bool
	if err := tx.QueryRow(ctx, newerSQL, j.Tenant, j.Definition, j.Version).Scan(&newer); err != nil {
		return false, fmt.Errorf("read newer job: %w", err)
	}
	if !newer {
		return false, nil
	}
	for _, q := range []string{supersedeSQL, dropPendingSQL, dropFailuresSQL} {
		if _, err := tx.Exec(ctx, q, j.Tenant, j.Definition, j.Version); err != nil {
			return false, fmt.Errorf("supersede: %w", err)
		}
	}
	if err := a.auditJob(ctx, tx, j, row.causation, "backfill finished",
		jobDoc(j, row.state, nil), jobDoc(j, stateSuperseded, &row)); err != nil {
		return false, err
	}
	return true, nil
}

// Barrier is the ADR-006 D9 barrier (fieldindex/barrier): it waits until
// every transaction in progress at the admission has ended, so the fill sees
// every record a write before the publish committed (D7).
func (a *Activities) Barrier(ctx context.Context) error {
	j, err := jobOf(ctx)
	if err != nil {
		return err
	}
	err = barrier.Wait(ctx, a.Pool, barrier.Options{SlowAfter: a.slowAfter(),
		OnSlow: func(ctx context.Context, open int64, waited time.Duration) {
			a.logger().WarnContext(ctx, "schema backfill barrier slow: a transaction open at the admission is still running",
				a.attrs(ctx, j, slog.Bool("alert", true), slog.String("alert_name", "schema_backfill_barrier_slow"),
					slog.Int64("open_transactions", open), slog.Duration("waited", waited))...)
		}})
	if err != nil {
		return fmt.Errorf("schema backfill %w", err)
	}
	return nil
}

// Fill queues the job's records and flips it to running, in one transaction
// (D7, P10-05 step 3): every record of the type with NOT (data ?& keys),
// keys the job row's own default_keys. A job no longer filling changes
// nothing: running (a retried attempt whose commit was lost) goes on to the
// pages, anything else ends the workflow.
func (a *Activities) Fill(ctx context.Context) (FillResult, error) {
	j, err := jobOf(ctx)
	if err != nil {
		return FillResult{}, err
	}
	def := j.Definition
	var res FillResult
	var queued int64
	err = a.inJob(ctx, j, func(tx pgx.Tx, row jobRow) error {
		switch row.state {
		case stateRunning:
			return nil
		case stateFilling:
		default:
			res.Ended = &def
			return nil
		}
		tag, err := tx.Exec(ctx, fillSQL, j.Tenant, j.Definition, j.Version, row.defaultKeys)
		if err != nil {
			return fmt.Errorf("fill: %w", err)
		}
		queued = tag.RowsAffected()
		_, err = tx.Exec(ctx, runSQL, j.Tenant, j.Definition, j.Version)
		return err
	})
	if err != nil {
		return FillResult{}, a.fail(ctx, "fill", err)
	}
	a.logger().InfoContext(ctx, "schema backfill queue filled", a.attrs(ctx, j, slog.Int64("queued", queued))...)
	return res, nil
}

// pageContext is what a page reads once: the type's key and the rewrite's
// causation.
type pageContext struct {
	state     string
	entityKey string
	causation outbox.Causation
}

// Page handles the next page of the queue (D7, P10-05 step 4):
//
//  1. the supersede check (D8): a newer job for the type supersedes this
//     one, and the page ends the workflow; so does a job no longer running;
//  2. up to PageSize pending ids after in.After, with no row lock; none
//     after a cursor starts the next pass from the first id, and none at
//     all means the queue is empty;
//  3. each id in its own transaction (rewrite).
//
// It returns the last id it handled as the next cursor.
func (a *Activities) Page(ctx context.Context, in Cursor) (PageResult, error) {
	j, err := jobOf(ctx)
	if err != nil {
		return PageResult{}, err
	}
	def := j.Definition
	pc, ended, err := a.startPage(ctx, j)
	if err != nil {
		return PageResult{}, a.fail(ctx, "page", err)
	}
	if ended {
		return PageResult{Ended: &def}, nil
	}
	ids, err := a.pendingIDs(ctx, j, in.After)
	if err == nil && len(ids) == 0 && in.After != nil {
		ids, err = a.pendingIDs(ctx, j, nil)
	}
	if err != nil {
		return PageResult{}, a.fail(ctx, "page", err)
	}
	if len(ids) == 0 {
		return PageResult{}, nil
	}
	for _, id := range ids {
		if err := a.rewrite(ctx, j, pc, id); err != nil {
			return PageResult{}, a.fail(ctx, "page", err)
		}
		if a.afterRecord != nil {
			if err := a.afterRecord(ctx, id); err != nil {
				return PageResult{}, err
			}
		}
	}
	last := ids[len(ids)-1]
	return PageResult{Next: &last}, nil
}

// startPage runs the supersede check and reads the page context, in one
// transaction. It reports ended when the job was superseded or is no longer
// running.
func (a *Activities) startPage(ctx context.Context, j Job) (pageContext, bool, error) {
	var pc pageContext
	ended := false
	err := a.inJob(ctx, j, func(tx pgx.Tx, row jobRow) error {
		if row.state != stateRunning {
			ended = true
			return nil
		}
		gone, err := a.supersedeIfNewer(ctx, tx, j, row)
		if err != nil || gone {
			ended = gone
			return err
		}
		var causation *uuid.UUID
		if err := tx.QueryRow(ctx, pageContextSQL, j.Tenant, j.Definition, j.Version).Scan(&pc.state, &causation, &pc.entityKey); err != nil {
			return fmt.Errorf("read page context: %w", err)
		}
		pc.causation, err = recordCausation(ctx, tx, j.Tenant, causation)
		return err
	})
	return pc, ended, err
}

// recordCausation is a rewrite's causation (D2): the job's causation audit
// entry as the origin at that entry's depth + 1; empty, so the pipeline mints
// an origin at depth 0, for a job with none.
func recordCausation(ctx context.Context, tx pgx.Tx, tenant uuid.UUID, origin *uuid.UUID) (outbox.Causation, error) {
	if origin == nil {
		return outbox.Causation{}, nil
	}
	var depth int16
	err := tx.QueryRow(ctx, causationDepthSQL, tenant, *origin).Scan(&depth)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return outbox.Causation{}, fmt.Errorf("read causation depth: %w", err)
	}
	return outbox.Causation{OriginID: *origin, Depth: depth + 1}, nil
}

// pendingIDs reads up to PageSize pending ids after after (or from the
// first), with no row lock.
func (a *Activities) pendingIDs(ctx context.Context, j Job, after *uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tenancy.InTenantTx(ctx, a.Pool, j.Tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, pageSQL, j.Tenant, j.Definition, j.Version, after, a.pageSize())
		if err != nil {
			return err
		}
		ids, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read pending ids: %w", err)
	}
	return ids, nil
}

// rewrite handles one pending id (D7) in its own InTenantTx with
// synchronous_commit off: the pending row is taken, the record is
// materialized, and the job's rewritten or skipped count goes up, all
// committing together. An id no longer pending (another attempt took it)
// is left alone. A version conflict rolls back and leaves the id pending. A
// refusal or a database error rolls back, then a separate transaction moves
// the id to schema_backfill_failures with its code (no savepoints). Only a
// cancelled context, or a failure of that separate transaction, is an error:
// the activity is retried and the id is still pending.
func (a *Activities) rewrite(ctx context.Context, j Job, pc pageContext, id uuid.UUID) error {
	var written bool
	// synchronous_commit off (tenancy.WithSynchronousCommitOff): a commit
	// lost in a crash leaves the id pending, and the next pass repeats it.
	err := tenancy.InTenantTx(ctx, a.Pool, j.Tenant, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, takePendingSQL, j.Tenant, j.Definition, j.Version, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		written, err = a.Materializer.Materialize(ctx, tx, Target{Tenant: j.Tenant, EntityKey: pc.entityKey, RecordID: id, Causation: pc.causation})
		if err != nil {
			return err
		}
		var rw, sk int64
		if written {
			rw = 1
		} else {
			sk = 1
		}
		_, err = tx.Exec(ctx, countSQL, j.Tenant, j.Definition, j.Version, rw, sk)
		return err
	}, tenancy.WithSynchronousCommitOff())
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ErrConflict) {
		return nil
	}
	code := CodeDatabaseError
	if r := (*Refusal)(nil); errors.As(err, &r) {
		code = r.Code
	}
	// The error's text is not logged: a database or validation error can
	// quote a value. The record id and the code are references.
	a.logger().WarnContext(ctx, "schema backfill record failed; moved to failures",
		a.attrs(ctx, j, slog.String("record_id", id.String()), slog.String("code", code))...)
	return a.recordFailure(ctx, j, id, code)
}

// recordFailure moves id from the queue to schema_backfill_failures with
// code and adds one to the job's failed count, in one transaction of its
// own. An id no longer pending is left alone.
func (a *Activities) recordFailure(ctx context.Context, j Job, id uuid.UUID, code string) error {
	return tenancy.InTenantTx(ctx, a.Pool, j.Tenant, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, takePendingSQL, j.Tenant, j.Definition, j.Version, id)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		if _, err := tx.Exec(ctx, failSQL, j.Tenant, j.Definition, j.Version, id, code); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, failCountSQL, j.Tenant, j.Definition, j.Version)
		return err
	})
}

// End ends the job (P10-05 step 6): done when it has no failure rows,
// otherwise failed_records; finished_at; and the "backfill finished" audit
// entry with the counts. A job no longer running (a retried attempt whose
// commit was lost, or superseded) changes nothing; one that ended
// failed_records still reports it, for the R6 check.
func (a *Activities) End(ctx context.Context) (EndResult, error) {
	j, err := jobOf(ctx)
	if err != nil {
		return EndResult{}, err
	}
	def := j.Definition
	var end string
	var counts jobRow
	err = a.inJob(ctx, j, func(tx pgx.Tx, row jobRow) error {
		counts = row
		if row.state != stateRunning {
			end = row.state
			return nil
		}
		var failed bool
		if err := tx.QueryRow(ctx, hasFailuresSQL, j.Tenant, j.Definition, j.Version).Scan(&failed); err != nil {
			return err
		}
		end = stateDone
		if failed {
			end = stateFailedRecords
		}
		if _, err := tx.Exec(ctx, endSQL, j.Tenant, j.Definition, j.Version, end); err != nil {
			return err
		}
		return a.auditJob(ctx, tx, j, row.causation, "backfill finished",
			jobDoc(j, stateRunning, nil), jobDoc(j, end, &row))
	})
	if err != nil {
		return EndResult{}, a.fail(ctx, "end", err)
	}
	a.logger().InfoContext(ctx, "schema backfill finished", a.attrs(ctx, j, slog.String("state", end),
		slog.Int64("rewritten", counts.rewritten), slog.Int64("skipped", counts.skipped), slog.Int64("failed", counts.failed))...)
	if end == stateFailedRecords {
		return EndResult{FailedRecords: &def}, nil
	}
	return EndResult{}, nil
}

// CheckR6 is the R6 trigger check (D-10-3; P10-06 item 4), run when the
// workflow's timer after a failed_records end fires: it re-reads the job's
// own row under RLS, and if it is still failed_records and still the newest
// for its type, it adds one to nava_schema_backfill_r6_tripped and logs one
// WARN line with the tenant, the type key and the target version. No record
// id and no value; no cross-tenant read.
func (a *Activities) CheckR6(ctx context.Context) error {
	j, err := jobOf(ctx)
	if err != nil {
		return err
	}
	var state, key string
	var newer bool
	err = tenancy.InTenantTx(ctx, a.Pool, j.Tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, r6SQL, j.Tenant, j.Definition, j.Version).Scan(&state, &key, &newer)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return a.fail(ctx, "r6 check", err)
	}
	if state != stateFailedRecords || newer {
		return nil
	}
	a.Metrics.r6Tripped(ctx)
	a.logger().WarnContext(ctx, "schema backfill R6 tripped: the type is still failed_records after the R6 window",
		slog.String("tenant_id", j.Tenant.String()), slog.String("entity_type", key),
		slog.Int("target_version", j.Version), slog.Bool("alert", true), slog.String("alert_name", "schema_backfill_r6_tripped"))
	return nil
}

// fail logs an activity's database error by its shape and returns it. A
// non-retryable ErrTypeNoJob passes through. The error's text is logged
// only for errors this package built; a database error can quote a value,
// so only its SQLSTATE and constraint are.
func (a *Activities) fail(ctx context.Context, step string, err error) error {
	if app := (*temporal.ApplicationError)(nil); errors.As(err, &app) {
		return err
	}
	attrs := []any{slog.String("step", step)}
	if pe := (*pgconn.PgError)(nil); errors.As(err, &pe) {
		attrs = append(attrs, slog.String("sqlstate", pe.Code), slog.String("constraint", pe.ConstraintName))
	}
	a.logger().ErrorContext(ctx, "schema backfill activity failed; it is retried", attrs...)
	return fmt.Errorf("schema backfill %s: %w", step, err)
}

// jobDoc is the document a "backfill started" or "backfill finished" audit
// entry diffs (D4): the job's target version and state, and with counts set
// its counts. It holds no record data.
func jobDoc(j Job, state string, counts *jobRow) json.RawMessage {
	b := map[string]any{"state": state, "target_version": j.Version}
	if counts != nil {
		b["rewritten"], b["skipped"], b["failed"] = counts.rewritten, counts.skipped, counts.failed
	}
	raw, err := json.Marshal(map[string]any{"backfill": b})
	if err != nil {
		panic(err) // a map of strings and integers always marshals
	}
	return raw
}

// auditJob writes one entity-type configuration entry of the job (D4,
// P10-05 steps 1 and 6) in tx: domain config, action update, subject the
// entity definition, actor_system schema-backfill, source system, causation
// the job's causation audit entry at its depth + 1 (none for a job without
// one). Like a numbering-mode change (internal/entitytypes/audit.go), the
// diff is before → after of a small document and content_sha256 hashes the
// after document (R3); no version moves. what names the entry in errors.
func (a *Activities) auditJob(ctx context.Context, tx pgx.Tx, j Job, causation *uuid.UUID, what string, before, after json.RawMessage) error {
	diff, err := jsondiff.Diff(before, after)
	if err != nil {
		return fmt.Errorf("%s: diff: %w", what, err)
	}
	c, err := recordCausation(ctx, tx, j.Tenant, causation)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(after)
	sys := ActorSystem
	def := j.Definition
	e := audit.Entry{
		Domain:             audit.DomainConfig,
		Action:             audit.ActionUpdate,
		SubjectType:        audit.SubjectEntityDefinition,
		SubjectID:          def,
		EntityDefinitionID: &def,
		Actor:              audit.Actor{System: &sys},
		Source:             audit.SourceSystem,
		CausationDepth:     c.Depth,
		ContentSHA256:      &sum,
		Diff:               diff,
	}
	if causation != nil {
		e.CausationID = causation
	}
	if _, err := audit.Append(ctx, tx, e); err != nil {
		return fmt.Errorf("%s: audit: %w", what, err)
	}
	return nil
}

func (a *Activities) pageSize() int {
	if a.PageSize > 0 {
		return a.PageSize
	}
	return DefaultPageSize
}

func (a *Activities) slowAfter() time.Duration {
	if a.SlowAfter > 0 {
		return a.SlowAfter
	}
	return DefaultSlowAfter
}

func (a *Activities) logger() *slog.Logger {
	if a.Log != nil {
		return a.Log
	}
	return slog.New(slog.DiscardHandler)
}

// attrs are a log line's job attributes: ids and the version, never data.
func (a *Activities) attrs(ctx context.Context, j Job, extra ...any) []any {
	out := []any{slog.String("tenant_id", j.Tenant.String()), slog.String("entity_definition_id", j.Definition.String()),
		slog.Int("target_version", j.Version)}
	if activity.IsActivity(ctx) {
		out = append(out, slog.String("workflow_id", activity.GetInfo(ctx).WorkflowExecution.ID))
	}
	return append(out, extra...)
}

// Metrics are the job's metrics. A nil *Metrics records nothing.
type Metrics struct {
	r6 metric.Int64Counter
}

// NewMetrics registers the job's metrics on mp.
func NewMetrics(mp metric.MeterProvider) (*Metrics, error) {
	m := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/schemabackfill")
	r6, err := m.Int64Counter("nava_schema_backfill_r6_tripped",
		metric.WithDescription("Schema backfill jobs still failed_records and newest for their type when the R6 window (7 days) ended (ADR-009 R6, PLAN-010 D-10-3). Alert on any increase; the tenant, type and version are in the WARN line, not here."))
	if err != nil {
		return nil, err
	}
	return &Metrics{r6: r6}, nil
}

func (m *Metrics) r6Tripped(ctx context.Context) {
	if m == nil {
		return
	}
	m.r6.Add(ctx, 1)
}
