package authz

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// fakeStore has no path read at all: AdminGroupStore is the only way in.
type fakeStore struct {
	rows map[uuid.UUID]uuid.UUID
	err  error
}

func (f fakeStore) AdminGroupID(_ context.Context, t uuid.UUID) (uuid.UUID, error) {
	if f.err != nil {
		return uuid.Nil, f.err
	}
	g, ok := f.rows[t]
	if !ok {
		return uuid.Nil, ErrNoAdminGroup
	}
	return g, nil
}

func TestRequireTenantAdmin(t *testing.T) {
	ta, tb := uuid.New(), uuid.New()
	ga, gb := uuid.New(), uuid.New()
	st := fakeStore{rows: map[uuid.UUID]uuid.UUID{ta: ga, tb: gb}}
	boom := errors.New("db down")
	tests := []struct {
		name  string
		store AdminGroupStore
		p     identity.Principal
		want  error
	}{
		{"member passes", st, identity.Principal{TenantID: ta, Groups: []uuid.UUID{uuid.New(), ga}}, nil},
		{"non-member", st, identity.Principal{TenantID: ta, Groups: []uuid.UUID{uuid.New()}}, ErrForbidden},
		{"no groups", st, identity.Principal{TenantID: ta}, ErrForbidden},
		{"other tenant's admin group", st, identity.Principal{TenantID: ta, Groups: []uuid.UUID{gb}}, ErrForbidden},
		{"tenant without a row", fakeStore{}, identity.Principal{TenantID: ta, Groups: []uuid.UUID{ga}}, ErrForbidden},
		{"nil group id never matches", fakeStore{rows: map[uuid.UUID]uuid.UUID{ta: uuid.Nil}}, identity.Principal{TenantID: ta, Groups: []uuid.UUID{uuid.Nil}}, ErrForbidden},
		{"store error is not a 403", fakeStore{err: boom}, identity.Principal{TenantID: ta, Groups: []uuid.UUID{ga}}, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := RequireTenantAdmin(t.Context(), tc.store, tc.p)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if tc.want != nil && errors.Is(tc.want, boom) && errors.Is(err, ErrForbidden) {
				t.Fatal("a store failure must not read as forbidden")
			}
		})
	}
}

// pathFreeTx fails any SQL that mentions groups or path, and answers the one
// statement the helper may run.
type pathFreeTx struct {
	pgx.Tx // nil: any method not overridden panics, which fails the test
	t      *testing.T
	group  uuid.UUID
	sql    []string
}

var forbidden = regexp.MustCompile(`(?i)\b(groups|path)\b`)

func (x *pathFreeTx) check(sql string) {
	x.sql = append(x.sql, sql)
	if forbidden.MatchString(sql) {
		x.t.Errorf("the admin check ran SQL that mentions groups or path: %s", sql)
	}
}

func (x *pathFreeTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	x.check(sql)
	return rowOf{x.group}
}

func (x *pathFreeTx) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	x.check(sql)
	return nil, errors.New("Query is not allowed")
}

func (x *pathFreeTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	x.check(sql)
	return pgconn.CommandTag{}, errors.New("Exec is not allowed")
}

type rowOf struct{ id uuid.UUID }

func (r rowOf) Scan(dest ...any) error {
	*(dest[0].(*uuid.UUID)) = r.id
	return nil
}

func TestRequireTenantAdminTxNeverReadsGroupsOrPaths(t *testing.T) {
	g := uuid.New()
	tx := &pathFreeTx{t: t, group: g}
	p := identity.Principal{TenantID: uuid.New(), Groups: []uuid.UUID{g}}
	if err := RequireTenantAdminTx(t.Context(), tx, p); err != nil {
		t.Fatal(err)
	}
	if err := RequireTenantAdminTx(t.Context(), tx, identity.Principal{TenantID: p.TenantID}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-member: %v", err)
	}
	if len(tx.sql) != 2 || !strings.Contains(tx.sql[0], "tenant_admin_group") {
		t.Errorf("statements = %q", tx.sql)
	}
	// The matcher itself catches a path or groups read.
	if !forbidden.MatchString("SELECT path FROM groups") || !forbidden.MatchString("select 1 from groups") || forbidden.MatchString("SELECT group_id FROM tenant_admin_group") {
		t.Error("the SQL matcher is wrong")
	}
}
