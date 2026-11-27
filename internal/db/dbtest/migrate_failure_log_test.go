//go:build integration

package dbtest

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
)

// TICK-1135: a failed migration's log has the error's HINT, SQLSTATE and
// constraint, and not the DETAIL, internal query or context, where PostgreSQL
// puts row values and SQL text. leak-marker-1135 stands for tenant data.
func TestTICK1135FailedMigrationLogsHintNotDetail(t *testing.T) {
	const marker = "leak-marker-1135"
	m := migratedCluster(t)
	for _, c := range []struct {
		name, up                   string
		sqlstate, constraint, hint string
		hidden                     func(*pgconn.PgError) string // the field the marker is in
	}{
		{
			name: "a raised DETAIL",
			up: `DO $$ BEGIN
				RAISE EXCEPTION 't1135 refused' USING ERRCODE = 'check_violation', CONSTRAINT = 't1135_check',
					DETAIL = 'leak-marker-1135', HINT = 't1135: run the job again.';
			END $$;`,
			sqlstate: "23514", constraint: "t1135_check", hint: "t1135: run the job again.",
			hidden: func(pe *pgconn.PgError) string { return pe.Detail },
		},
		{
			name: "the failing row of a constraint",
			up: `DO $$ BEGIN
				CREATE TABLE t1135 (note text CONSTRAINT t1135_check CHECK (note IS NULL));
				INSERT INTO t1135 VALUES ('leak-marker-1135');
			END $$;`,
			sqlstate: "23514", constraint: "t1135_check",
			hidden: func(pe *pgconn.PgError) string { return pe.Detail },
		},
		{
			name:     "the query of a function",
			up:       `DO $$ BEGIN EXECUTE 'SELECT 1 AS "leak-marker-1135" FROM'; END $$;`,
			sqlstate: "42601",
			hidden:   func(pe *pgconn.PgError) string { return pe.InternalQuery },
		},
		{
			name:     "the context of a function",
			up:       `DO $$ BEGIN EXECUTE 'SELECT 1 / (length(''leak-marker-1135'') - 16)'; END $$;`,
			sqlstate: "22012",
			hidden:   func(pe *pgconn.PgError) string { return pe.Where },
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			log, out := newLog()
			_, err := migrate.Up(context.Background(), roleConfig(t, m.Owner(), m.PlatformDatabase),
				withExtraMigration(t, "29991231000002_t1135.sql",
					"-- +goose Up\n-- +goose StatementBegin\n"+c.up+"\n-- +goose StatementEnd\n"),
				log, migrate.Options{LockRetrySeconds: 1})
			var pe *pgconn.PgError
			if !errors.As(err, &pe) || !strings.Contains(c.hidden(pe), marker) {
				t.Fatalf("Up = %v (%+v): the fixture hides nothing", err, pe)
			}
			if strings.Contains(out.String(), marker) {
				t.Errorf("the log holds %s:\n%s", marker, out.String())
			}

			n := 0
			for dec := json.NewDecoder(strings.NewReader(out.String())); dec.More(); {
				var line map[string]any
				if err := dec.Decode(&line); err != nil {
					t.Fatal(err)
				}
				if line["msg"] != "migration failed" {
					continue
				}
				n++
				for key, want := range map[string]string{
					"level": "ERROR", "error": err.Error(),
					"sqlstate": c.sqlstate, "constraint": c.constraint, "hint": c.hint,
				} {
					if got, _ := line[key].(string); got != want {
						t.Errorf("logged %s = %q, want %q", key, got, want)
					}
				}
			}
			if n != 1 {
				t.Errorf("%d \"migration failed\" lines, want 1; log:\n%s", n, out.String())
			}
		})
	}
}
