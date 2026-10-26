// Package purge deletes expired rows from idempotency_keys, outbox and
// workflow_ref, connected as nava_purge (ADR-001 A.3, D28). The role can
// select only the key and expiry columns and delete only the rows its paired
// *_purge policies match, so the windows here can narrow what is deleted but
// never widen it.
package purge

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// MaxBatch is the most rows one DELETE statement removes.
const MaxBatch = 1000

// Tables, in the order a run purges them. The names are the only attribute
// value of the deleted-rows metric.
const (
	TableIdempotencyKeys = "idempotency_keys"
	TableOutbox          = "outbox"
	TableWorkflowRef     = "workflow_ref"
)

// Beginner starts the transaction of one batch. *pgxpool.Pool and *pgx.Conn
// satisfy it.
type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Windows are the retention windows (see config.Purge).
type Windows struct {
	OutboxDispatched time.Duration
	OutboxDead       time.Duration
	WorkflowRef      time.Duration
}

// DefaultWindows are the ADR-001 A.3 windows.
func DefaultWindows() Windows {
	return Windows{
		OutboxDispatched: config.MinOutboxDispatchedRetention,
		OutboxDead:       config.MinOutboxDeadRetention,
		WorkflowRef:      config.MinWorkflowRefRetention,
	}
}

// Options tune a Purger. Zero fields take the defaults.
type Options struct {
	Windows   Windows
	BatchSize int // at most MaxBatch; default MaxBatch
	Interval  time.Duration
}

// Result is what one run did to one table.
type Result struct {
	Table   string
	Deleted int64
	// Skipped is true when another purger held the table's advisory lock, so
	// this run stopped without deleting more.
	Skipped bool
	Err     error
}

// Purger runs the purge. Build it with New.
type Purger struct {
	db      Beginner
	log     *slog.Logger
	opts    Options
	deleted metric.Int64Counter
	targets []target
}

// target is one table: the statement deletes up to $1 rows and takes the
// window seconds as $2 (and $3).
type target struct {
	table string
	sql   string
	args  func(batch int, w Windows) []any
}

func secs(d time.Duration) float64 { return d.Seconds() }

// New registers the deleted-rows counter on mp.
func New(db Beginner, mp metric.MeterProvider, log *slog.Logger, opts Options) (*Purger, error) {
	if opts.BatchSize <= 0 || opts.BatchSize > MaxBatch {
		opts.BatchSize = MaxBatch
	}
	if opts.Interval <= 0 {
		opts.Interval = time.Hour
	}
	if opts.Windows == (Windows{}) {
		opts.Windows = DefaultWindows()
	}
	c, err := mp.Meter("nava-purge").Int64Counter("nava.purge.deleted_rows",
		metric.WithDescription("Rows nava-purge deleted, by table."), metric.WithUnit("{row}"))
	if err != nil {
		return nil, fmt.Errorf("purge: %w", err)
	}
	return &Purger{db: db, log: log, opts: opts, deleted: c, targets: targetList()}, nil
}

func targetList() []target {
	return []target{
		{
			table: TableIdempotencyKeys,
			sql: `DELETE FROM idempotency_keys WHERE (tenant_id, principal_id, key) IN (
			        SELECT tenant_id, principal_id, key FROM idempotency_keys
			        WHERE expires_at < now() LIMIT $1)`,
			args: func(batch int, _ Windows) []any { return []any{batch} },
		},
		{
			table: TableOutbox,
			sql: `DELETE FROM outbox WHERE (tenant_id, id) IN (
			        SELECT tenant_id, id FROM outbox
			        WHERE dispatched_at < now() - make_interval(secs => $2)
			           OR dead_at < now() - make_interval(secs => $3) LIMIT $1)`,
			args: func(batch int, w Windows) []any {
				return []any{batch, secs(w.OutboxDispatched), secs(w.OutboxDead)}
			},
		},
		{
			table: TableWorkflowRef,
			sql: `DELETE FROM workflow_ref WHERE (tenant_id, outbox_id) IN (
			        SELECT tenant_id, outbox_id FROM workflow_ref
			        WHERE closed_at < now() - make_interval(secs => $2) LIMIT $1)`,
			args: func(batch int, w Windows) []any { return []any{batch, secs(w.WorkflowRef)} },
		},
	}
}

// Statement is one table's batch DELETE with its arguments.
type Statement struct {
	Table string
	SQL   string
	Args  []any
}

// Statements are the batch DELETEs a run executes, in order, for EXPLAIN in
// tests (TICK-146: each must be able to use its purge index).
func Statements(batch int, w Windows) []Statement {
	ts := targetList()
	out := make([]Statement, 0, len(ts))
	for _, t := range ts {
		out = append(out, Statement{Table: t.table, SQL: t.sql, Args: t.args(batch, w)})
	}
	return out
}

// LockName is the text whose hash is the advisory lock key of table's batches.
func LockName(table string) string { return "nava-purge:" + table }

// Once purges every table, batch by batch, and returns one Result per table.
// A failing table does not stop the others; its error is in its Result.
func (p *Purger) Once(ctx context.Context) []Result {
	out := make([]Result, 0, len(p.targets))
	for _, t := range p.targets {
		r := p.purgeTable(ctx, t)
		p.deleted.Add(ctx, r.Deleted, metric.WithAttributes(attribute.String("table", t.table)))
		switch {
		case r.Err != nil:
			p.log.ErrorContext(ctx, "purge failed", slog.String("table", t.table),
				slog.Int64("deleted", r.Deleted), slog.Any("error", r.Err))
		case r.Skipped:
			p.log.InfoContext(ctx, "purge skipped: another purger holds the table",
				slog.String("table", t.table), slog.Int64("deleted", r.Deleted))
		default:
			p.log.InfoContext(ctx, "purged", slog.String("table", t.table), slog.Int64("deleted", r.Deleted))
		}
		out = append(out, r)
	}
	return out
}

func (p *Purger) purgeTable(ctx context.Context, t target) Result {
	r := Result{Table: t.table}
	for ctx.Err() == nil {
		n, locked, err := p.batch(ctx, t)
		if err != nil {
			r.Err = err
			return r
		}
		if !locked {
			r.Skipped = true
			return r
		}
		r.Deleted += n
		if n < int64(p.opts.BatchSize) {
			return r
		}
	}
	r.Err = ctx.Err()
	return r
}

// batch deletes up to BatchSize rows in one transaction that holds the table's
// advisory lock. locked is false when another transaction holds it.
func (p *Purger) batch(ctx context.Context, t target) (n int64, locked bool, err error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))`,
		LockName(t.table)).Scan(&locked); err != nil {
		return 0, false, fmt.Errorf("advisory lock: %w", err)
	}
	if !locked {
		return 0, false, nil
	}
	tag, err := tx.Exec(ctx, t.sql, t.args(p.opts.BatchSize, p.opts.Windows)...)
	if err != nil {
		return 0, true, fmt.Errorf("delete: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, true, fmt.Errorf("commit: %w", err)
	}
	return tag.RowsAffected(), true, nil
}

// Run purges once at once, then every Interval, until ctx is cancelled.
func (p *Purger) Run(ctx context.Context) error {
	t := time.NewTicker(p.opts.Interval)
	defer t.Stop()
	for {
		p.Once(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}
