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
	"fmt"
	"io/fs"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
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
	if opt.LockRetrySeconds == 0 {
		opt.LockRetrySeconds = 2
	}
	if opt.LockAttempts == 0 {
		opt.LockAttempts = 300
	}
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockTimeout(opt.LockRetrySeconds, opt.LockAttempts))
	if err != nil {
		return nil, false, err
	}
	db := stdlib.OpenDB(*cfg)
	defer func() { _ = db.Close() }()
	return up(ctx, db, fsys, locker, log)
}

func up(ctx context.Context, db *sql.DB, fsys fs.FS, locker lock.SessionLocker, log *slog.Logger) ([]string, bool, error) {
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
	results, err := p.Up(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("migrate: %w", err)
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
