//go:build integration

// ADR-006-A2 A2-2..A2-5 (TICK-1046): the list's page runs first, the count runs
// under a statement_timeout of its own inside SAVEPOINT list_count, and a first
// page that holds the whole result runs no count. The statements are counted by
// a pgx QueryTracer on the test pool; nothing in production reads it.

package store_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// stmtLog records every statement the pool sends.
type stmtLog struct {
	mu   sync.Mutex
	stmt []string
}

func (l *stmtLog) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	l.mu.Lock()
	l.stmt = append(l.stmt, d.SQL)
	l.mu.Unlock()
	return ctx
}
func (*stmtLog) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (l *stmtLog) reset() { l.mu.Lock(); l.stmt = nil; l.mu.Unlock() }

// counts is the number of count statements sent since reset.
func (l *stmtLog) counts() int { return l.matching("count(*) FROM (SELECT 1") }

func (l *stmtLog) matching(sub string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, s := range l.stmt {
		if strings.Contains(s, sub) {
			n++
		}
	}
	return n
}

// budgetEnv is a migrated cluster and a pool of ONE connection (the pooled-
// connection leak test needs the next transaction on the same backend) with
// the statement tracer.
type budgetEnv struct {
	env
	pool *pgxpool.Pool
	log  *stmtLog
}

func newBudgetEnv(t *testing.T) *budgetEnv {
	t.Helper()
	e := migrated(t)
	log := &stmtLog{}
	cfg := e.app.Config().Copy()
	cfg.MaxConns = 1
	cfg.ConnConfig.Tracer = log
	p, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return &budgetEnv{env: e, pool: p, log: log}
}

// seed makes a tenant with an asset type and n records (no audit entries: the
// test reads counts, not audit).
func (e *budgetEnv) seed(t *testing.T, n int) (tenant, def uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tenant, principal := uuid.New(), uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
		VALUES ($1, $2, $3, true, now())`, tenant, principal, "kc-"+principal.String())
	if err := tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error {
		_, err := entitytypes.Store{}.Create(ctx, tx, tenant, entitytypes.NewInput{Name: "asset", PublishedBy: principal, Definition: assetDefinition()})
		if err != nil {
			return err
		}
		def, err = store.DefinitionID(ctx, tx, tenant, "asset")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	stx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	if _, err := stx.Exec(ctx, `INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state, created_at, updated_at, data)
		SELECT $1, gen_random_uuid(), $2, 1, 1, 'B-' || gen_random_uuid(), 'active', now() - (g || ' seconds')::interval, now(), '{}'
		FROM generate_series(1, $3::int) g`, tenant, def, n); err != nil {
		t.Fatal(err)
	}
	if err := stx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return tenant, def
}

// query is a list of the tenant's records; limit is the client's limit, so the
// fetch size is limit+1 as the handler passes it.
func query(tenant, def uuid.UUID, limit int) store.ListQuery {
	return store.ListQuery{TenantID: tenant, DefinitionID: def, Where: "true", SortColumn: "created_at", Desc: true, Limit: limit + 1, BudgetCount: true}
}

func (e *budgetEnv) list(t *testing.T, q store.ListQuery) *store.ListPage {
	t.Helper()
	var page *store.ListPage
	if err := tenancy.InTenantTx(context.Background(), e.pool, q.TenantID, func(tx pgx.Tx) error {
		var err error
		page, err = store.List(context.Background(), tx, q)
		return err
	}); err != nil {
		t.Fatalf("list: %v", err)
	}
	return page
}

func TestBudgetedCount(t *testing.T) {
	e := newBudgetEnv(t)
	ctx := context.Background()
	tenantA, defA := e.seed(t, 300)
	tenantB, _ := e.seed(t, 20)

	// AC 1 and AC 2 (3, 3b): a list whose count outruns the budget.
	t.Run("failure path keeps the transaction and the tenant", func(t *testing.T) {
		e.log.reset()
		q := query(tenantA, defA, 10)
		q.CountDelay = 300 * time.Millisecond
		err := tenancy.InTenantTx(ctx, e.pool, tenantA, func(tx pgx.Tx) error {
			var before string
			if err := tx.QueryRow(ctx, `SELECT current_setting('statement_timeout')`).Scan(&before); err != nil {
				return err
			}
			page, err := store.List(ctx, tx, q)
			if err != nil {
				t.Errorf("list: %v", err)
				return err
			}
			if page.TotalState != store.TotalUnknown || page.Total != 0 || len(page.Records) != 11 {
				t.Errorf("page = %d records, total %d %q; want 11 records, unknown", len(page.Records), page.Total, page.TotalState)
			}
			var one int
			if err := tx.QueryRow(ctx, `SELECT 1`).Scan(&one); err != nil {
				t.Errorf("SELECT 1 after the rollback: %v", err)
			}
			// navalint keeps the tenant setting's name out of this package, so the
			// tenant context is proved by what RLS lets through: only A's rows.
			var after string
			var foreign, own int
			if err := tx.QueryRow(ctx, `SELECT current_setting('statement_timeout'),
				(SELECT count(*) FROM records WHERE tenant_id <> $1::uuid), (SELECT count(*) FROM records)`, tenantA.String()).Scan(&after, &foreign, &own); err != nil {
				return err
			}
			if after != before {
				t.Errorf("statement_timeout after the list = %q, before = %q", after, before)
			}
			if foreign != 0 || own != 300 {
				t.Errorf("after the rollback: foreign rows %d, own rows %d (want 0 and tenant A's 300)", foreign, own)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if n := e.log.matching("SAVEPOINT list_count; SET LOCAL statement_timeout = '140'"); n != 1 {
			t.Errorf("budget SET LOCAL sent %d times, want once with 140 ms", n)
		}
	})

	// A branched ACL is one statement under one timeout.
	t.Run("branched count is one statement", func(t *testing.T) {
		e.log.reset()
		q := query(tenantA, defA, 10)
		q.Args = []any{uuid.New()}
		q.Branches = []string{"(r.requester_id = $3)", "(r.assignee_id = $3 AND r.requester_id IS DISTINCT FROM $3)"}
		q.CountDelay = 300 * time.Millisecond
		// No row matches the branches, so the first page is short: a keyset page
		// keeps the shortcut out of the way.
		q.Keyset, q.KeysetArgs = "(r.created_at) < ($4::timestamptz)", []any{time.Now().Add(time.Hour)}
		page := e.list(t, q)
		if page.TotalState != store.TotalUnknown {
			t.Errorf("state %q, want unknown", page.TotalState)
		}
		if n := e.log.matching("pg_sleep"); n != 1 {
			t.Errorf("count statements = %d, want 1", n)
		}
		if n := e.log.matching("UNION ALL"); n != 2 { // the page and the count each hold one
			t.Errorf("statements with UNION ALL = %d, want 2 (page and one count)", n)
		}
	})

	// AC 3 (3c): the next transaction on the same pooled connection.
	t.Run("nothing leaks onto the pooled connection", func(t *testing.T) {
		type sess struct {
			timeout string
			// rows is what the connection's session may read outside any
			// transaction: with no tenant set RLS lets nothing through.
			rows int
			pid  uint32
		}
		read := func() sess {
			var s sess
			if err := e.pool.QueryRow(ctx, `SELECT current_setting('statement_timeout'), (SELECT count(*) FROM records), pg_backend_pid()`).
				Scan(&s.timeout, &s.rows, &s.pid); err != nil {
				t.Fatal(err)
			}
			return s
		}
		before := read()
		q := query(tenantA, defA, 10)
		q.CountDelay = 300 * time.Millisecond
		if p := e.list(t, q); p.TotalState != store.TotalUnknown {
			t.Fatalf("state %q, want unknown", p.TotalState)
		}
		if after := read(); after != before {
			t.Errorf("session after an unknown list = %+v, before = %+v", after, before)
		}
		// Tenant B, same backend: the default timeout and its own tenant only.
		err := tenancy.InTenantTx(ctx, e.pool, tenantB, func(tx pgx.Tx) error {
			var timeout string
			var pid uint32
			var foreign, own int
			if err := tx.QueryRow(ctx, `SELECT current_setting('statement_timeout'), pg_backend_pid(),
				(SELECT count(*) FROM records WHERE tenant_id <> $1::uuid), (SELECT count(*) FROM records)`, tenantB.String()).Scan(&timeout, &pid, &foreign, &own); err != nil {
				return err
			}
			if pid != before.pid || timeout != before.timeout || foreign != 0 || own != 20 {
				t.Errorf("tenant B tx: pid %d (want %d), timeout %q (want %q), foreign rows %d, own rows %d (want 0 and 20)", pid, before.pid, timeout, before.timeout, foreign, own)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	// A legacy scan: min(CountBudget, Timeout), proven in the statement text.
	t.Run("legacy scan budget is the smaller timeout", func(t *testing.T) {
		e.log.reset()
		q := query(tenantA, defA, 10)
		q.DataFilter, q.Timeout, q.CountDelay = true, 120*time.Millisecond, 150*time.Millisecond
		if p := e.list(t, q); p.TotalState != store.TotalUnknown {
			t.Errorf("state %q, want unknown", p.TotalState)
		}
		if n := e.log.matching("SAVEPOINT list_count; SET LOCAL statement_timeout = '120'"); n != 1 {
			t.Errorf("SET LOCAL '120' sent %d times, want once", n)
		}
	})

	// No data filter: today's count, no savepoint, no budget.
	t.Run("a list with no data filter keeps its count", func(t *testing.T) {
		e.log.reset()
		q := query(tenantA, defA, 10)
		q.BudgetCount, q.CountDelay = false, 300*time.Millisecond
		p := e.list(t, q)
		if p.TotalState != store.TotalExact || p.Total != 300 {
			t.Errorf("total %d %q, want exact 300", p.Total, p.TotalState)
		}
		if n := e.log.matching("SAVEPOINT"); n != 0 {
			t.Errorf("SAVEPOINT sent %d times, want none", n)
		}
	})

	t.Run("a cancelled request is an error, not unknown", func(t *testing.T) {
		cctx, cancel := context.WithTimeout(ctx, 60*time.Millisecond)
		defer cancel()
		q := query(tenantA, defA, 10)
		q.CountDelay = 3 * time.Second
		err := tenancy.InTenantTx(cctx, e.pool, tenantA, func(tx pgx.Tx) error {
			_, err := store.List(cctx, tx, q)
			return err
		})
		if err == nil {
			t.Fatal("a cancelled request returned a page")
		}
		if errors.Is(err, store.ErrStatementTimeout) {
			t.Errorf("a cancelled request is not a statement timeout: %v", err)
		}
	})

	// AC 5 (4): the first-page shortcut and its boundaries, counted by the tracer.
	for _, c := range []struct {
		name    string
		matches int
		keyset  bool
		counts  int
		total   int64
	}{
		{"fewer than limit", 7, false, 0, 7},
		{"exactly limit", 10, false, 0, 10},
		{"limit plus one", 11, false, 1, 11},
		{"keyset page, few rows", 3, true, 1, 3},
		{"keyset page, many rows", 40, true, 1, 40},
		{"empty first page", 0, false, 0, 0},
	} {
		t.Run("shortcut: "+c.name, func(t *testing.T) {
			tn, df := e.seed(t, c.matches)
			e.log.reset()
			q := query(tn, df, 10)
			if c.keyset {
				q.Keyset, q.KeysetArgs = "(r.created_at) < ($3::timestamptz)", []any{time.Now().Add(time.Hour)}
			}
			q.BudgetCount = false // the shortcut applies to every list
			p := e.list(t, q)
			if n := e.log.counts(); n != c.counts {
				t.Errorf("count statements = %d, want %d", n, c.counts)
			}
			if p.TotalState != store.TotalExact || p.Total != c.total {
				t.Errorf("total %d %q, want exact %d", p.Total, p.TotalState, c.total)
			}
			if c.matches == 11 && len(p.Records) != 11 {
				t.Errorf("a next page is outstanding: %d rows fetched, want 11", len(p.Records))
			}
		})
	}
}
