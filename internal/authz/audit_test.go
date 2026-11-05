package authz

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// auditTx answers the two group-id reads by table; a missing table is no row.
type auditTx struct {
	pgx.Tx
	rows map[string]uuid.UUID
	err  error
	sql  []string
}

type errRow struct{ err error }

func (r errRow) Scan(...any) error { return r.err }

func (x *auditTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	x.sql = append(x.sql, sql)
	if x.err != nil {
		return errRow{x.err}
	}
	for table, id := range x.rows {
		if strings.Contains(sql, table) {
			return rowOf{id}
		}
	}
	return errRow{pgx.ErrNoRows}
}

func TestRequireAuditPermission(t *testing.T) {
	admin, aud, other := uuid.New(), uuid.New(), uuid.New()
	tn := uuid.New()
	both := map[string]uuid.UUID{"tenant_admin_group": admin, "tenant_auditor_group": aud}
	boom := errors.New("db down")
	tests := []struct {
		name   string
		rows   map[string]uuid.UUID
		err    error
		groups []uuid.UUID
		perm   string
		want   error
	}{
		{"admin read", both, nil, []uuid.UUID{admin}, PermAuditRead, nil},
		{"admin export", both, nil, []uuid.UUID{admin}, PermAuditExport, nil},
		{"auditor read", both, nil, []uuid.UUID{aud}, PermAuditRead, nil},
		{"auditor export", both, nil, []uuid.UUID{aud}, PermAuditExport, nil},
		{"agent", both, nil, []uuid.UUID{other}, PermAuditRead, ErrForbidden},
		{"no groups", both, nil, nil, PermAuditRead, ErrForbidden},
		{"auditor row missing", map[string]uuid.UUID{"tenant_admin_group": admin}, nil, []uuid.UUID{aud}, PermAuditRead, ErrForbidden},
		{"admin still passes without auditor row", map[string]uuid.UUID{"tenant_admin_group": admin}, nil, []uuid.UUID{admin}, PermAuditRead, nil},
		{"both rows missing", nil, nil, []uuid.UUID{aud}, PermAuditRead, ErrForbidden},
		{"nil auditor id never matches", map[string]uuid.UUID{"tenant_auditor_group": uuid.Nil}, nil, []uuid.UUID{uuid.Nil}, PermAuditRead, ErrForbidden},
		{"other permission", both, nil, []uuid.UUID{admin, aud}, "records.write", ErrForbidden},
		{"empty permission", both, nil, []uuid.UUID{admin}, "", ErrForbidden},
		{"store error is not a 403", nil, boom, []uuid.UUID{aud}, PermAuditRead, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx := &auditTx{rows: tc.rows, err: tc.err}
			err := RequireAuditPermission(t.Context(), tx, identity.Principal{TenantID: tn, Groups: tc.groups}, tc.perm)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if errors.Is(tc.want, boom) && errors.Is(err, ErrForbidden) {
				t.Fatal("a store failure must not read as forbidden")
			}
			for _, q := range tx.sql {
				if forbidden.MatchString(q) {
					t.Errorf("SQL mentions groups or path: %s", q)
				}
			}
		})
	}
}
