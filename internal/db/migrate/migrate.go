// Package migrate runs the platform database's migrations with goose,
// converges the audit_log policies generated from db/audited-tables.yaml,
// applies db/grants.yaml with the generated writer grants merged in (ADR-004
// R12), then runs the catalogue gate (ADR-001 CI-1, CI-3, CI-4; R2; ADR-004
// CI-A3).
// On a fresh database a grant the manifest step revokes also fails the run
// (TICK-233). It runs as nava_migrator in a one-shot job
// connected directly to PostgreSQL: goose's session advisory lock needs one
// server session for the whole run, which PgBouncer's transaction pooling
// does not give (and get_auth refuses the migrator through the pooler).
// nava-core never runs it.
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// Options tune a run. The zero value is the production setting.
type Options struct {
	// LockRetrySeconds is how often a run that finds the migration lock held
	// tries again (default 2). LockAttempts bounds the tries (default 300,
	// so a second run waits up to 10 minutes).
	LockRetrySeconds uint64
	LockAttempts     uint64

	// LockTimeout is the lock_timeout the migrations run under (default 5s):
	// a DDL statement that waits longer than this for a table lock fails with
	// SQLSTATE 55P03 instead of queueing every reader and writer of the table
	// behind it (TICK-579). It is a session setting, so a transactional
	// migration can still override it with SET LOCAL. A NO TRANSACTION
	// migration that sets its own must RESET it before it ends.
	LockTimeout time.Duration
	// DDLAttempts bounds how many times a run that hit the lock_timeout runs
	// the pending migrations again (default 10). DDLBackoff is the wait
	// before the first retry (default 1s); each retry doubles it, up to
	// DDLMaxBackoff (default 30s). The failed migration's transaction rolled
	// back, and every NO TRANSACTION migration is safe to re-run
	// (TestNoTransactionMigrationsAreIdempotent), so a retry resumes at it.
	DDLAttempts   int
	DDLBackoff    time.Duration
	DDLMaxBackoff time.Duration
}

func (o Options) withDefaults() Options {
	if o.LockRetrySeconds == 0 {
		o.LockRetrySeconds = 2
	}
	if o.LockAttempts == 0 {
		o.LockAttempts = 300
	}
	if o.LockTimeout == 0 {
		o.LockTimeout = 5 * time.Second
	}
	if o.DDLAttempts == 0 {
		o.DDLAttempts = 10
	}
	if o.DDLBackoff == 0 {
		o.DDLBackoff = time.Second
	}
	if o.DDLMaxBackoff == 0 {
		o.DDLMaxBackoff = 30 * time.Second
	}
	return o
}

// Result is what one run applied.
type Result struct {
	Applied  []string // migration file names, in order
	Policies audited.PolicyResult
	Grants   grants.Result
}

// Run applies every pending migration in fsys, then the generated audit_log
// policies, then the grant manifest m with a's generated writer grants merged
// in, then the catalogue gate (with a's CI-A3 check) (R2's post-run check): a run that leaves the catalogue
// off the isolation invariants fails, with every problem logged. Two runs at
// once are serialised by a session advisory lock: the second waits, then
// finds nothing to apply.
//
// The manifest step revokes, and logs, every grant db/grants.yaml does not
// list, so the gate then sees a healed catalogue. On a deployed database that
// is drift made outside the migrations and the run still succeeds. On a fresh
// one, a run that applied the first migration, nothing but the migrations can
// have made the grant, so each revoke is a gate problem too (ADR-001 A.3:
// grants come only from the manifest). CI migrates fresh databases, so a
// migration that grants by hand fails there.
func Run(ctx context.Context, cfg *pgx.ConnConfig, fsys fs.FS, m *grants.Manifest, a *audited.Manifest, g *catalog.GlobalTables, log *slog.Logger, opt Options) (Result, error) {
	var res Result
	m, err := a.MergeGrants(m)
	if err != nil {
		return res, err
	}
	applied, fresh, err := upFresh(ctx, cfg, fsys, log, opt)
	if err != nil {
		return res, err
	}
	res.Applied = applied

	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return res, err
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()
	res.Policies, err = a.ApplyPolicies(ctx, conn, log)
	if err != nil {
		return res, fmt.Errorf("apply generated audit_log policies: %w", err)
	}
	res.Grants, err = grants.Apply(ctx, conn, m, log)
	if err != nil {
		return res, fmt.Errorf("apply grants manifest: %w", err)
	}
	problems, err := catalog.Check(ctx, conn, m, g)
	if err != nil {
		return res, err
	}
	a3, err := a.Check(ctx, conn, m, g)
	if err != nil {
		return res, fmt.Errorf("CI-A3 check: %w", err)
	}
	problems = append(problems, a3...)
	if fresh {
		for _, r := range res.Grants.Revoked {
			problems = append(problems, "CI-4: a migration granted "+r+", which db/grants.yaml does not list "+
				"(revoked on a fresh database; grant it in db/grants.yaml, not in a migration)")
		}
	}
	for _, p := range problems {
		log.ErrorContext(ctx, "catalogue gate", slog.String("problem", p))
	}
	if len(problems) > 0 {
		return res, fmt.Errorf("catalogue gate (ADR-001 CI-1, CI-3, CI-4): %d problems:\n%s",
			len(problems), strings.Join(problems, "\n"))
	}
	log.InfoContext(ctx, "migrate done",
		slog.Int("migrations_applied", len(res.Applied)),
		slog.Int("policies_created", len(res.Policies.Created)),
		slog.Int("policies_dropped", len(res.Policies.Dropped)),
		slog.Int("grants_granted", len(res.Grants.Granted)),
		slog.Int("grants_revoked", len(res.Grants.Revoked)))
	return res, nil
}

// Up applies the pending migrations under the session lock and returns the
// names of the files it applied.
func Up(ctx context.Context, cfg *pgx.ConnConfig, fsys fs.FS, log *slog.Logger, opt Options) ([]string, error) {
	applied, _, err := upFresh(ctx, cfg, fsys, log, opt)
	return applied, err
}

// upFresh is Up, and reports whether the run applied the first migration in
// fsys: the database had none applied before it.
func upFresh(ctx context.Context, cfg *pgx.ConnConfig, fsys fs.FS, log *slog.Logger, opt Options) ([]string, bool, error) {
	opt = opt.withDefaults()
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockTimeout(opt.LockRetrySeconds, opt.LockAttempts))
	if err != nil {
		return nil, false, err
	}
	// The advisory lock is pg_try_advisory_lock, which never waits, so the
	// lock_timeout bounds only the migrations' own lock waits.
	mcfg := cfg.Copy()
	if mcfg.RuntimeParams == nil {
		mcfg.RuntimeParams = map[string]string{}
	}
	mcfg.RuntimeParams["lock_timeout"] = fmt.Sprintf("%dms", opt.LockTimeout.Milliseconds())
	db := stdlib.OpenDB(*mcfg)
	defer func() { _ = db.Close() }()
	return up(ctx, db, fsys, locker, log, opt)
}

// isLockTimeout reports whether err is a lock_timeout expiry (55P03
// lock_not_available).
func isLockTimeout(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "55P03"
}

func up(ctx context.Context, db *sql.DB, fsys fs.FS, locker lock.SessionLocker, log *slog.Logger, opt Options) ([]string, bool, error) {
	// Out of order: new migrations carry timestamp versions (TICK-290), and
	// one from a long-lived branch can be older than one already applied.
	// goose applies such a missing migration instead of refusing the run.
	p, err := goose.NewProvider(goose.DialectPostgres, db, fsys,
		goose.WithAllowOutofOrder(true),
		goose.WithSessionLocker(locker),
		goose.WithDisableGlobalRegistry(true),
		goose.WithSlog(log))
	if err != nil {
		return nil, false, err
	}
	// A migration that hits the lock_timeout is retried with a doubling
	// backoff. Each attempt takes and releases the advisory lock, so another
	// run may apply the rest meanwhile; this one then finds nothing pending.
	var results []*goose.MigrationResult
	backoff := opt.DDLBackoff
	for attempt := 1; ; attempt++ {
		r, err := p.Up(ctx)
		if err == nil {
			results = append(results, r...)
			break
		}
		var partial *goose.PartialError
		if errors.As(err, &partial) {
			results = append(results, partial.Applied...)
		}
		if !isLockTimeout(err) || attempt >= opt.DDLAttempts {
			if isLockTimeout(err) {
				err = fmt.Errorf("lock_timeout %s hit on %d attempts: %w", opt.LockTimeout, attempt, err)
			}
			return nil, false, fmt.Errorf("migrate: %w", err)
		}
		log.WarnContext(ctx, "migration hit lock_timeout, retrying",
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", opt.DDLAttempts),
			slog.Duration("backoff", backoff),
			slog.String("error", err.Error()))
		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, false, fmt.Errorf("migrate: %w (after %w)", ctx.Err(), err)
		case <-t.C:
		}
		backoff = min(2*backoff, opt.DDLMaxBackoff)
	}
	names := make([]string, 0, len(results))
	for _, r := range results {
		names = append(names, r.Source.Path)
	}
	// Sources are in version order; results in the order applied.
	sources := p.ListSources()
	fresh := len(results) > 0 && len(sources) > 0 && results[0].Source.Version == sources[0].Version
	return names, fresh, nil
}
