package purge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// The SQL itself runs against real roles in internal/db/dbtest
// (TestT13PurgeAssertRoleRealRoles); these cover the Go around it.

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
	if err := AssertRole(t.Context(), fakeDB{fakeRow{role: "nava_purge", ok: true}}); err != nil {
		t.Fatalf("nava_purge: %v, want nil", err)
	}
	// What the SQL reports for a BYPASSRLS role, a superuser or another role.
	for _, role := range []string{"nava_bypass", "postgres", "nava_app"} {
		err := AssertRole(t.Context(), fakeDB{fakeRow{role: role}})
		if err == nil || !strings.Contains(err.Error(), role) {
			t.Errorf("%s: err = %v, want an error naming the role", role, err)
		}
	}
}

func TestAssertRoleQueryFailureHidesCause(t *testing.T) {
	cause := errors.New("postgres://nava_purge:s3cret@db/nava refused")
	err := AssertRole(t.Context(), fakeDB{fakeRow{err: cause}})
	if err == nil || strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("err = %v, want an error without the DSN", err)
	}
}
