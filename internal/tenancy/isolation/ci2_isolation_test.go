//go:build integration

package isolation

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The tenant states CI-2 probes in. They set app.tenant_id by hand, which
// only this package's tests and internal/tenancy may (navalint setconfig).
const (
	setEmptySQL = "SELECT set_config('app.tenant_id', '', true)"
	// setPriorSQL is the SET LOCAL of an earlier transaction on the same
	// server connection; setPriorSessionSQL is the session-wide leak the
	// pool-bleed fixture uses to prove the case can fail.
	setPriorSQL        = "SELECT set_config('app.tenant_id', $1, true), pg_backend_pid()"
	setPriorSessionSQL = "SELECT set_config('app.tenant_id', $1, false), pg_backend_pid()"
)

const (
	statusPass    = "PASS"
	statusFail    = "FAIL"
	statusAllowed = "ALLOWLISTED"
	statusPending = "PENDING"
)

// errRollback ends a tenancy.InTenantTx callback so the probe is rolled back.
var errRollback = errors.New("rollback")

// TestCI2EveryRoleEveryTenantTable is ADR-001 CI-2 over every tenant table
// and every runtime role: with no tenant, an empty one and after a prior SET LOCAL,
// directly and through PgBouncer, every granted SELECT, UPDATE and DELETE
// reaches 0 rows or fails; with tenant A set it reaches none of tenant B's.
func TestCI2EveryRoleEveryTenantTable(t *testing.T) {
	lines := runCI2(t, ci2Opts{})
	suite.report.ci2 = lines
	ownRecords := int64(-1)
	for _, l := range lines {
		if l.Status == statusFail {
			t.Errorf("%s × %s: %s", l.Role, l.Table, l.Detail)
		}
		if l.Role == "nava_app" && l.Table == "public.records" {
			ownRecords = l.own
		}
	}
	// The harness itself: tenant A's caller sees tenant A's rows, so the
	// zeros above are RLS at work, not a broken seed or tenant setting.
	if ownRecords != 2 {
		t.Errorf("nava_app saw %d of tenant A's 2 records with tenant A set; the harness proves nothing", ownRecords)
	}
}

type ci2Opts struct {
	roles            []string // nil: every runtime role and the owner
	tables           []string // nil: every tenant table
	sessionWidePrior bool     // the prior transaction sets the tenant session-wide
}

// ci2Line is one role × table of the report.
type ci2Line struct {
	Role, Table, Status, Detail string
	own                         int64 // tenant A's rows SELECT saw with tenant A set; -1 when not run
}

// grant is what role may do on a table, and the statements that probe it.
type grant struct {
	cmd        string // SELECT, UPDATE or DELETE
	all, other string // every row; rows of tenants other than $1
}

type probeResult struct {
	n    int64
	code string // SQLSTATE when the statement failed
}

// leaked: the statement reached a row. An integrity error means it did.
func (r probeResult) leaked() bool {
	return (r.code == "" && r.n > 0) || strings.HasPrefix(r.code, "23")
}

func (r probeResult) String() string {
	if r.code != "" {
		return "err " + r.code
	}
	return strconv.FormatInt(r.n, 10)
}

// cell is one probe of one grant: its label and result.
type cell struct {
	label string
	r     probeResult
}

// tableRun collects one role × table.
type tableRun struct {
	t      catalog.TenantTable
	grants []grant
	cells  map[string][]cell // by command
	fail   []string          // harness and seed failures
	own    int64
}

func runCI2(t *testing.T, o ci2Opts) []ci2Line {
	t.Helper()
	e := suite
	ctx := t.Context()
	super := e.superConn(t)
	roles := o.roles
	if roles == nil {
		roles = e.runtimeRoles()
	}
	tables := e.tables
	if o.tables != nil {
		tables = nil
		for _, tt := range e.tables {
			if slices.Contains(o.tables, tt.Schema+"."+tt.Name) {
				tables = append(tables, tt)
			}
		}
	}
	exempt := map[string]string{}
	for _, x := range Exemptions() {
		exempt[x.Role+"|"+x.Table+"|"+x.Command] = x.Policy
	}

	var lines []ci2Line
	for _, role := range roles {
		label := role
		if role == e.m.Owner() {
			label += " (owner)"
		}
		direct, code := e.connect(t, role, false)
		if code != "" {
			for _, tt := range tables {
				lines = append(lines, ci2Line{Role: label, Table: tt.Schema + "." + tt.Name, Status: statusPass, own: -1,
					Detail: "cannot connect to the platform database (SQLSTATE " + code + ")"})
			}
			continue
		}
		pooled := e.m.Roles[role].PgBouncerLogin
		var setter, prober *pgx.Conn
		if pooled {
			// Two clients of one PgBouncer pool of size 1: the prober's
			// transactions run on the server connection the setter used.
			if setter, code = e.connect(t, role, true); code == "" {
				prober, code = e.connect(t, role, true)
			}
			if code != "" {
				t.Fatalf("%s is on the PgBouncer allow-list but PgBouncer refused it: SQLSTATE %s", role, code)
			}
		}

		runs := make([]*tableRun, len(tables))
		for i, tt := range tables {
			runs[i] = newTableRun(t, super, role, tt)
		}
		// No tenant first, while neither connection has ever set one.
		for _, step := range []struct {
			label  string
			conn   *pgx.Conn
			setup  string
			prior  bool
			pooled bool
		}{
			{"direct none", direct, "", false, false},
			{"pgbouncer none", prober, "", false, true},
			{"direct ''", direct, setEmptySQL, false, false},
			{"pgbouncer ''", prober, setEmptySQL, false, true},
			{"direct prior", direct, "", true, false},
			{"pgbouncer prior", prober, "", true, true},
		} {
			if step.pooled && !pooled {
				continue
			}
			setterConn := direct
			if step.pooled {
				setterConn = setter
			}
			for _, run := range runs {
				for _, g := range run.grants {
					var wantPID uint32
					if step.prior {
						wantPID = setPrior(t, setterConn, o.sessionWidePrior)
					}
					r, pid := probeConn(t, step.conn, step.setup, g, g.all)
					if step.prior && pid != wantPID {
						run.fail = append(run.fail, fmt.Sprintf("%s: backend %d, prior SET LOCAL ran on %d; PgBouncer did not reuse it", step.label, pid, wantPID))
					}
					run.cells[g.cmd] = append(run.cells[g.cmd], cell{step.label, r})
				}
			}
		}
		if o.sessionWidePrior {
			for _, c := range []*pgx.Conn{direct, setter} {
				if c != nil {
					if _, err := c.Exec(ctx, "RESET app.tenant_id"); err != nil {
						t.Fatal(err)
					}
				}
			}
		}

		// Tenant A set the platform's way, through the same path.
		pool := e.pool(t, role, pooled)
		for _, run := range runs {
			for _, g := range run.grants {
				r := probeTenantA(t, pool, g.other)
				run.cells[g.cmd] = append(run.cells[g.cmd], cell{"tenant A, other tenants' rows", r})
				if g.cmd == "SELECT" {
					if own := probeTenantA(t, pool, g.all); own.code == "" {
						run.own = own.n
					}
				}
			}
		}
		for _, run := range runs {
			lines = append(lines, run.line(label, exempt, role))
		}
	}
	return lines
}

// newTableRun reads what role may do on tt and checks that the seed holds
// rows of both tenants.
func newTableRun(t *testing.T, super *pgx.Conn, role string, tt catalog.TenantTable) *tableRun {
	t.Helper()
	ctx := t.Context()
	qn, key := tt.QualifiedName(), pgx.Identifier{tt.Key}.Sanitize()
	run := &tableRun{t: tt, cells: map[string][]cell{}, own: -1}

	var a, b int64
	if err := super.QueryRow(ctx, "SELECT count(*) FILTER (WHERE "+key+" = $1), count(*) FILTER (WHERE "+key+" = $2) FROM "+qn,
		TenantA, TenantB).Scan(&a, &b); err != nil {
		t.Fatal(err)
	}
	if a == 0 || b == 0 {
		run.fail = append(run.fail, fmt.Sprintf("seed holds %d rows of tenant A and %d of tenant B; add both to testdata/seed.sql", a, b))
	}

	var canSelect, keyReadable, canDelete bool
	if err := super.QueryRow(ctx, `
		SELECT has_any_column_privilege($1::name, $2::regclass, 'SELECT'),
		       has_column_privilege($1::name, $2::regclass, $3::text, 'SELECT'),
		       has_table_privilege($1::name, $2::regclass, 'DELETE')`, role, qn, tt.Key).
		Scan(&canSelect, &keyReadable, &canDelete); err != nil {
		t.Fatal(err)
	}
	// The column UPDATE probes: one the role may update, preferably one it
	// may also read, so SET c = c needs nothing more. Otherwise SET c = NULL;
	// on a NOT NULL column a reached row then fails with class 23.
	var updCol string
	var updReadable bool
	err := super.QueryRow(ctx, `
		SELECT attname::text, has_column_privilege($1::name, attrelid, attnum, 'SELECT')
		FROM pg_catalog.pg_attribute
		WHERE attrelid = $2::regclass AND attnum > 0 AND NOT attisdropped
		  AND has_column_privilege($1::name, attrelid, attnum, 'UPDATE')
		ORDER BY 2 DESC, attnum LIMIT 1`, role, qn).Scan(&updCol, &updReadable)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal(err)
	}

	other := " WHERE " + key + " <> $1"
	if canSelect {
		sel := "SELECT count(*) FROM " + qn
		run.grants = append(run.grants, grant{"SELECT", sel, sel + other})
	}
	if updCol != "" {
		c := pgx.Identifier{updCol}.Sanitize()
		set := "NULL"
		if updReadable {
			set = c
		}
		upd := "UPDATE " + qn + " SET " + c + " = " + set
		run.grants = append(run.grants, grant{"UPDATE", upd, upd + other})
	}
	if canDelete {
		del := "DELETE FROM " + qn
		run.grants = append(run.grants, grant{"DELETE", del, del + other})
	}
	// With tenant A set, "other tenants' rows" is a predicate on the key: a
	// role that may touch rows but not read the key would fail it with
	// 42501 and pass without proving anything.
	if len(run.grants) > 0 && !keyReadable {
		run.fail = append(run.fail, "the role may touch rows but not read "+tt.Key+", so tenant A's rows cannot be told from tenant B's")
	}
	return run
}

// setPrior runs and commits a transaction that sets tenant A, and returns
// its server backend.
func setPrior(t *testing.T, conn *pgx.Conn, sessionWide bool) uint32 {
	t.Helper()
	q := setPriorSQL
	if sessionWide {
		q = setPriorSessionSQL
	}
	var pid uint32
	err := pgx.BeginFunc(t.Context(), conn, func(tx pgx.Tx) error {
		var s string
		return tx.QueryRow(t.Context(), q, TenantA).Scan(&s, &pid)
	})
	if err != nil {
		t.Fatal(err)
	}
	return pid
}

// probeConn runs setup then q in one transaction it rolls back, and returns
// the result and the server backend that ran it.
func probeConn(t *testing.T, conn *pgx.Conn, setup string, g grant, q string) (probeResult, uint32) {
	t.Helper()
	ctx := t.Context()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if setup != "" {
		if _, err := tx.Exec(ctx, setup); err != nil {
			t.Fatal(err)
		}
	}
	var pid uint32
	if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return run(t, tx, g.cmd, q), pid
}

// probeTenantA runs q with $1 = tenant A inside tenancy.InTenantTx(tenant A)
// and rolls it back.
func probeTenantA(t *testing.T, pool *pgxpool.Pool, q string) probeResult {
	t.Helper()
	var r probeResult
	err := tenancy.InTenantTx(t.Context(), pool, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
		cmd := strings.Fields(q)[0]
		if strings.Contains(q, "$1") {
			r = run(t, tx, cmd, q, TenantA)
		} else {
			r = run(t, tx, cmd, q)
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatal(err)
	}
	return r
}

// run executes one probe. A server error is a result; anything else ends
// the test.
func run(t *testing.T, tx pgx.Tx, cmd, q string, args ...any) probeResult {
	t.Helper()
	ctx := t.Context()
	var r probeResult
	var err error
	if cmd == "SELECT" {
		err = tx.QueryRow(ctx, q, args...).Scan(&r.n)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, q, args...)
		r.n = tag.RowsAffected()
	}
	if err != nil {
		if r.code = sqlState(err); r.code == "" {
			t.Fatalf("%s: %v", q, err)
		}
		r.n = 0
	}
	return r
}

// line is the report line: FAIL on any reached row that no exemption
// covers or any harness failure, ALLOWLISTED when only exempt probes reached
// rows.
func (run *tableRun) line(label string, exempt map[string]string, role string) ci2Line {
	name := run.t.Schema + "." + run.t.Name
	l := ci2Line{Role: label, Table: name, Status: statusPass, own: run.own}
	var parts []string
	var allowed []string
	for _, g := range run.grants {
		var vals []string
		for _, c := range run.cells[g.cmd] {
			vals = append(vals, c.label+"="+c.r.String())
			if !c.r.leaked() {
				continue
			}
			if p, ok := exempt[role+"|"+name+"|"+g.cmd]; ok {
				if !slices.Contains(allowed, p) {
					allowed = append(allowed, p)
				}
				continue
			}
			l.Status = statusFail
			run.fail = append(run.fail, fmt.Sprintf("%s reached %s rows (%s)", g.cmd, c.r, c.label))
		}
		parts = append(parts, g.cmd+": "+strings.Join(vals, ", "))
	}
	if len(run.grants) == 0 {
		parts = append(parts, "no SELECT, UPDATE or DELETE grant")
	}
	if run.own >= 0 {
		parts = append(parts, fmt.Sprintf("own rows with tenant A set: %d", run.own))
	}
	if len(run.fail) > 0 {
		l.Status = statusFail
		parts = append(run.fail, parts...)
	} else if len(allowed) > 0 {
		l.Status = statusAllowed
		parts = append([]string{"by design: " + strings.Join(allowed, ", ")}, parts...)
	}
	l.Detail = strings.Join(parts, "; ")
	return l
}
