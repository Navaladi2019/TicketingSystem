// Command nava-db runs the platform database's one-shot jobs. It is a
// separate binary from nava-core, which never runs migrations.
//
//	nava-db roles    db-roles: as the bootstrap superuser, converge roles,
//	                 memberships, databases and the PgBouncer lookup to db/grants.yaml
//	nava-db migrate  as nava_migrator, directly to PostgreSQL (not PgBouncer):
//	                 apply the migrations, then the grant manifest, then the
//	                 catalogue gate (ADR-001 CI-1, CI-3, CI-4)
//
// The connection comes from the libpq environment (PGHOST, PGPORT, PGUSER,
// ...). PGPASSWORD_FILE, when set, is read for the password, so a secret
// never has to sit in the environment. db-roles reads each LOGIN role's
// password from DBROLES_PASSWORD_<ROLE>.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	log := logging.New(os.Stdout, config.Log{Level: slog.LevelInfo, Format: config.FormatJSON})
	err := run(ctx, os.Args[1:], os.LookupEnv, log)
	stop()
	if err != nil {
		log.ErrorContext(ctx, "nava-db failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, lookup config.LookupFunc, log *slog.Logger) error {
	if len(args) != 1 || (args[0] != "roles" && args[0] != "migrate") {
		return errors.New("usage: nava-db roles|migrate")
	}
	m, err := grants.Parse(db.Grants())
	if err != nil {
		return err
	}
	cfg, err := connConfig(lookup)
	if err != nil {
		return err
	}
	switch args[0] {
	case "roles":
		if cfg.Database == "" {
			cfg.Database = "postgres"
		}
		_, err = roles.Run(ctx, cfg, m, func(role string) (string, bool) {
			return lookup(roles.PasswordEnv(role))
		}, log)
	case "migrate":
		var g *catalog.GlobalTables
		if g, err = catalog.ParseGlobalTables(db.GlobalTables()); err != nil {
			return err
		}
		cfg.Database = m.PlatformDatabase
		_, err = migrate.Run(ctx, cfg, db.Migrations(), m, g, log, migrate.Options{})
	}
	return err
}

func connConfig(lookup config.LookupFunc) (*pgx.ConnConfig, error) {
	// ParseConfig("") reads the PG* environment itself.
	cfg, err := pgx.ParseConfig("")
	if err != nil {
		return nil, err
	}
	if path, ok := lookup("PGPASSWORD_FILE"); ok && path != "" {
		b, err := os.ReadFile(path) //nolint:gosec // the path is the operator's own setting
		if err != nil {
			return nil, fmt.Errorf("PGPASSWORD_FILE: %w", err)
		}
		cfg.Password = strings.TrimRight(string(b), "\r\n")
	}
	return cfg, nil
}
