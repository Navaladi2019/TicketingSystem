package tenantlimits_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/tenantlimits"
)

// fakeTx answers the statements tenantlimits sends, by the first word of
// their table or function, and records them. Any other pgx.Tx method panics
// (the embedded nil interface): tenantlimits must not call it.
type fakeTx struct {
	pgx.Tx
	// rows maps a substring of a query to the value its row scans, or to an
	// error; a query that matches none scans pgx.ErrNoRows.
	rows map[string]any
	// execErr maps a substring of a statement to its error.
	execErr map[string]error
	sent    []string
}

type row struct {
	v   any
	err error
}

func (r row) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if p, ok := dest[0].(*int); ok {
		*p = r.v.(int)
	}
	return nil
}

func (f *fakeTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	f.sent = append(f.sent, sql)
	for k, v := range f.rows {
		if strings.Contains(sql, k) {
			if err, ok := v.(error); ok {
				return row{err: err}
			}
			return row{v: v}
		}
	}
	return row{err: pgx.ErrNoRows}
}

func (f *fakeTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	f.sent = append(f.sent, sql)
	for k, err := range f.execErr {
		if strings.Contains(sql, k) {
			return pgconn.CommandTag{}, err
		}
	}
	return pgconn.NewCommandTag("OK"), nil
}

var errDB = errors.New("connection reset")

// PerType: the stored value, the default with no row, and a read error.
func TestPerType(t *testing.T) {
	tenant := uuid.New()
	got, err := tenantlimits.PerType(t.Context(), &fakeTx{rows: map[string]any{"tenant_limits": 8}}, tenant)
	if err != nil || got != 8 {
		t.Errorf("stored row = %d, %v; want 8", got, err)
	}
	got, err = tenantlimits.PerType(t.Context(), &fakeTx{}, tenant)
	if err != nil || got != fieldindex.DefaultPerTypeLimit {
		t.Errorf("no row = %d, %v; want the default %d", got, err, fieldindex.DefaultPerTypeLimit)
	}
	if _, err := tenantlimits.PerType(t.Context(), &fakeTx{rows: map[string]any{"tenant_limits": errDB}}, tenant); !errors.Is(err, errDB) {
		t.Errorf("read error = %v, want it wrapped", err)
	}
}

// SetPerType refuses, before any statement, a value out of 5..10, a missing,
// blank, oversized or non-UTF-8 reason, and a missing or oversized operator.
// The nil tx would panic on any statement.
func TestSetPerTypeRefusesBadInputBeforeAnyStatement(t *testing.T) {
	tenant := uuid.New()
	for name, c := range map[string]struct {
		n                int
		reason, operator string
		want             error
	}{
		"4":                     {4, "r", "op", tenantlimits.ErrOutOfRange},
		"11":                    {11, "r", "op", tenantlimits.ErrOutOfRange},
		"no reason":             {8, "", "op", tenantlimits.ErrNoReason},
		"a blank reason":        {8, "  \t ", "op", tenantlimits.ErrNoReason},
		"a reason over 500":     {8, strings.Repeat("r", tenantlimits.MaxReasonBytes+1), "op", tenantlimits.ErrNoReason},
		"a reason not UTF-8":    {8, "r\xff", "op", tenantlimits.ErrNoReason},
		"no operator":           {8, "r", "", tenantlimits.ErrNoOperator},
		"an operator over 255":  {8, "r", strings.Repeat("o", 256), tenantlimits.ErrNoOperator},
		"an operator not UTF-8": {8, "r", "o\xff", tenantlimits.ErrNoOperator},
	} {
		if _, err := tenantlimits.SetPerType(t.Context(), nil, tenant, c.n, c.reason, c.operator); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}
}

// SetPerType: an unknown tenant is ErrNotFound; the value already in force
// writes nothing; every database failure is returned wrapped, and nothing
// after it is sent.
func TestSetPerTypePaths(t *testing.T) {
	tenant := uuid.New()
	set := func(tx *fakeTx, n int) (tenantlimits.Limits, error) {
		return tenantlimits.SetPerType(t.Context(), tx, tenant, n, "measured cost check", "op-1")
	}

	if _, err := set(&fakeTx{}, 8); !errors.Is(err, tenantlimits.ErrNotFound) {
		t.Errorf("unknown tenant: err = %v, want ErrNotFound", err)
	}

	same := &fakeTx{rows: map[string]any{"FROM tenants": 1, "FROM tenant_limits": 8}}
	if got, err := set(same, 8); err != nil || got.FilterableFieldsPerType != 8 {
		t.Errorf("same value = %+v, %v; want 8", got, err)
	}
	for _, q := range same.sent {
		if strings.Contains(q, "INSERT") {
			t.Errorf("the value in force wrote %q", q)
		}
	}
	// No row is the default: setting the default writes nothing either.
	def := &fakeTx{rows: map[string]any{"FROM tenants": 1}}
	if got, err := set(def, fieldindex.DefaultPerTypeLimit); err != nil || got.FilterableFieldsPerType != fieldindex.DefaultPerTypeLimit {
		t.Errorf("the default with no row = %+v, %v", got, err)
	}

	for name, tx := range map[string]*fakeTx{
		"the lock":        {execErr: map[string]error{"pg_advisory_xact_lock": errDB}},
		"the tenant read": {rows: map[string]any{"FROM tenants": errDB}},
		"the limit read":  {rows: map[string]any{"FROM tenants": 1, "FROM tenant_limits": errDB}},
		"the write":       {rows: map[string]any{"FROM tenants": 1}, execErr: map[string]error{"INSERT INTO tenant_limits": errDB}},
	} {
		if _, err := set(tx, 9); !errors.Is(err, errDB) {
			t.Errorf("%s fails: err = %v, want it wrapped", name, err)
		}
	}
	// The audit entry fails (the fake cannot answer audit.Append's tenant
	// read): the error is returned as the audit's and the change does not
	// stand.
	if _, err := set(&fakeTx{rows: map[string]any{"FROM tenants": 1}}, 9); err == nil || !strings.Contains(err.Error(), "tenantlimits: audit") {
		t.Errorf("audit failure: err = %v, want a tenantlimits: audit error", err)
	}
}
