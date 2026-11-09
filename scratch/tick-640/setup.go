//go:build ignore

// TICK-640 (P3-34b) bench setup: builds the manifest roles, runs the real
// migrations and, with -seed, loads P3-01's 800k shape as P3-36's
// TestRecordsACLPerf800k does (bench/records-list/02_seed.sql on the real
// schema: the same RNG seed, distributions and tenant sizes), from seed.sql.
// Entity types are inserted directly: the list SQL reads only
// entity_definitions.key.
//
//	go run scratch/tick-640/setup.go -url postgres://pgadmin:bench@127.0.0.1:PORT/postgres [-seed] [-skip NAME]
//
// -skip leaves out every migration whose file name contains NAME (repeatable
// by comma), so the bench can seed the "before" schema and then apply this
// task's migrations over the seeded rows with a second run without -skip.
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
)

func pw(role string) string { return "bench-pw-" + role }

func main() {
	url := flag.String("url", "", "superuser URL to the postgres database")
	seed := flag.Bool("seed", false, "load P3-01's 800k shape")
	skip := flag.String("skip", "", "comma-separated substrings of migration file names to leave out")
	seedFile := flag.String("seed-file", "scratch/tick-640/seed.sql", "the seed statements (run from the repository root)")
	flag.Parse()
	if err := run(*url, *seed, *skip, *seedFile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// filtered copies the migrations, leaving out the skipped files.
func filtered(src fs.FS, skip string) (fs.FS, error) {
	if skip == "" {
		return src, nil
	}
	out := fstest.MapFS{}
	err := fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		for _, s := range strings.Split(skip, ",") {
			if strings.Contains(p, s) {
				return nil
			}
		}
		b, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		out[p] = &fstest.MapFile{Data: b}
		return nil
	})
	return out, err
}

func run(url string, seed bool, skip, seedFile string) error {
	ctx := context.Background()
	m, am, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		return err
	}
	superCfg, err := pgx.ParseConfig(url)
	if err != nil {
		return err
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if _, err := roles.Run(ctx, superCfg, m, func(r string) (string, bool) { return pw(r), true }, log); err != nil {
		return fmt.Errorf("roles: %w", err)
	}
	owner := superCfg.Copy()
	owner.User, owner.Password, owner.Database = m.Owner(), pw(m.Owner()), m.PlatformDatabase
	gt, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		return err
	}
	migs, err := filtered(db.Migrations(), skip)
	if err != nil {
		return err
	}
	start := time.Now()
	res, err := migrate.Run(ctx, owner, migs, m, am, gt, log, migrate.Options{LockRetrySeconds: 1})
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	fmt.Printf("migrate: applied %v in %s\n", res.Applied, time.Since(start).Round(time.Millisecond))
	if !seed {
		return nil
	}
	sup := superCfg.Copy()
	sup.Database = m.PlatformDatabase
	conn, err := pgx.ConnectConfig(ctx, sup)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	start = time.Now()
	b, err := os.ReadFile(seedFile)
	if err != nil {
		return err
	}
	for _, q := range strings.Split(string(b), ";\n") {
		if strings.TrimSpace(stripComments(q)) == "" {
			continue
		}
		if _, err := conn.Exec(ctx, q); err != nil {
			return fmt.Errorf("seed %.60s: %w", q, err)
		}
	}
	fmt.Printf("seeded in %s\n", time.Since(start).Round(time.Second))
	return nil
}

// stripComments drops "--" comment lines.
func stripComments(q string) string {
	var out []string
	for _, l := range strings.Split(q, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "--") {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
