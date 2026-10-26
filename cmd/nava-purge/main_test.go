package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeRow struct {
	role string
	ok   bool
	err  error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*string) = r.role
	*dest[1].(*bool) = r.ok
	return nil
}

type fakeDB struct{ row fakeRow }

func (f fakeDB) QueryRow(context.Context, string, ...any) pgx.Row { return f.row }

func TestAssertRole(t *testing.T) {
	if err := assertRole(t.Context(), fakeDB{fakeRow{role: "nava_purge", ok: true}}); err != nil {
		t.Fatalf("nava_purge: %v, want nil", err)
	}
	// What the SQL reports for a BYPASSRLS role, a superuser or another role.
	for _, role := range []string{"nava_bypass", "postgres", "nava_app"} {
		err := assertRole(t.Context(), fakeDB{fakeRow{role: role}})
		if err == nil || !strings.Contains(err.Error(), role) {
			t.Errorf("%s: err = %v, want an error naming the role", role, err)
		}
	}
}

func TestAssertRoleQueryFailureHidesCause(t *testing.T) {
	cause := errors.New("postgres://nava_purge:s3cret@db/nava refused")
	err := assertRole(t.Context(), fakeDB{fakeRow{err: cause}})
	if err == nil || strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("err = %v, want an error without the DSN", err)
	}
}

// Unreachable database: run fails on the role check and never reaches the
// purger, so nothing is deleted.
func TestRunFailsWithoutDatabase(t *testing.T) {
	env := map[string]string{"NAVA_DATABASE_URL": "postgres://nava_purge:s3cret@127.0.0.1:1/nava"}
	err := run(t.Context(), func(k string) (string, bool) { v, ok := env[k]; return v, ok }, &strings.Builder{})
	if err == nil || strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("run = %v, want an error without the DSN", err)
	}
}
