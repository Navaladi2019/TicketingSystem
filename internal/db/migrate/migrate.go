// Package migrate runs the platform database's migrations with goose, applies
// db/grants.yaml, then runs the catalogue gate (ADR-001 CI-1, CI-3, CI-4; R2). It runs as nava_migrator in a one-shot job
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
	Applied []string // migration file names, in order
	Grants  grants.Result
}

// Run applies every pending migration in fsys, then the grant manifest, then
// the catalogue gate (R2's post-run check): a run that leaves the catalogue
// off the isolation invariants fails, with every problem logged. Two runs at
// once are serialised by a session advisory lock: the second waits, then
// finds nothing to apply.
func Run(ctx context.Context, cfg *pgx.ConnConfig, fsys fs.FS, m *grants.Manifest, g *catalog.GlobalTables, log *slog.Logger, opt Options) (Result, error) {
	var res Result
	applied, err := Up(ctx, cfg, fsys, log, opt)
	if err != nil {
		return res, err
	}
	res.Applied = applied

	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return res, err
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()
	res.Grants, err = grants.Apply(ctx, conn, m, log)
	if err != nil {
		return res, fmt.Errorf("apply grants manifest: %w", err)
	}
	problems, err := catalog.Check(ctx, conn, m, g)
	if err != nil {
		return res, err
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
		slog.Int("grants_granted", len(res.Grants.Granted)),
		slog.Int("grants_revoked", len(res.Grants.Revoked)))
	return res, nil
}

// Up applies the pending migrations under the session lock and returns the
// names of the files it applied.
func Up(ctx context.Context, cfg *pgx.ConnConfig, fsys fs.FS, log *slog.Logger, opt Options) ([]string, error) {
	if opt.LockRetrySeconds == 0 {
		opt.LockRetrySeconds = 2
	}
	if opt.LockAttempts == 0 {
		opt.LockAttempts = 300
	}
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockTimeout(opt.LockRetrySeconds, opt.LockAttempts))
	if err != nil {
		return nil, err
	}
	db := stdlib.OpenDB(*cfg)
	defer func() { _ = db.Close() }()
	return up(ctx, db, fsys, locker, log)
}

func up(ctx context.Context, db *sql.DB, fsys fs.FS, locker lock.SessionLocker, log *slog.Logger) ([]string, error) {
	p, err := goose.NewProvider(goose.DialectPostgres, db, fsys,
		goose.WithSessionLocker(locker),
		goose.WithDisableGlobalRegistry(true),
		goose.WithSlog(log))
	if err != nil {
		return nil, err
	}
	results, err := p.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	names := make([]string, 0, len(results))
	for _, r := range results {
		names = append(names, r.Source.Path)
	}
	return names, nil
}
