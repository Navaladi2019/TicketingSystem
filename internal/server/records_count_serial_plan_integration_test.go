//go:build integration

// ADR-006-E2 E2-2 (3c) (TICK-1097): with the budgeted count's SET LOCALs in
// force (the real Exec text, captured by a pgx tracer) the count plan holds no
// Gather, where the control (statement_timeout only) holds one. Two shapes:
//
//   - (a) the count the store builds for the TICK-1049 / TestOneGroupACL...
//     query: the kp priority filter and the onboard default incident ACL of a
//     one-group agent (band (ii)).
//   - (b) the a6 "three broad fields" count of
//     docs/adr/ADR-006/spikes/a2-bounded-probe/count-plans.md: the same ACL
//     arms with a num, a text and a bool EXISTS.
//
// The planner is made to want workers at session level first (zeroed parallel
// costs), so any fixture produces the control's Gather; the control fails, it
// never skips.

package server_test

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// execTrace keeps the statements a pool sends.
type execTrace struct {
	mu   sync.Mutex
	stmt []string
}

func (l *execTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	l.mu.Lock()
	l.stmt = append(l.stmt, d.SQL)
	l.mu.Unlock()
	return ctx
}
func (*execTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (l *execTrace) reset() { l.mu.Lock(); l.stmt = nil; l.mu.Unlock() }

func (l *execTrace) savepointExec() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, s := range l.stmt {
		if strings.HasPrefix(s, "SAVEPOINT list_count;") {
			return s
		}
	}
	return ""
}

func planHasGather(t *testing.T, raw string) bool {
	t.Helper()
	var doc []map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	var walk func(n map[string]any) bool
	walk = func(n map[string]any) bool {
		if nt, _ := n["Node Type"].(string); nt == "Gather" || nt == "Gather Merge" {
			return true
		}
		return slices.ContainsFunc(sgKids(n), walk)
	}
	return walk(doc[0]["Plan"].(map[string]any))
}

// sideRows gives every record of the tenant one side row of value for a new
// ready field of type vt, so a filter on it matches everything.
func (e *sgEnv) sideRows(t *testing.T, vt, key, value string) uuid.UUID {
	t.Helper()
	id := e.flagField(t, e.a.id, "asset", key, vt, "ready", 100)
	tx, err := e.super.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(t.Context()) }()
	if _, err := tx.Exec(t.Context(), "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(t.Context(), `INSERT INTO rfv_`+vt+` (tenant_id, field_id, record_id, ord, value)
		SELECT tenant_id, $2, id, 0, `+value+` FROM records WHERE tenant_id = $1`, e.a.id, id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	return id
}

// andExists is q with one more EXISTS term, numbered after q's arguments.
func andExists(q pipeline.ListQuery, vt fieldindex.ValueType, field uuid.UUID, value any) pipeline.ListQuery {
	term, args := server.ExistsTermForTest(vt, field, "eq", value)
	base := 2 + len(q.Args)
	term = sgPlaceholder.ReplaceAllStringFunc(term, func(m string) string {
		n, _ := strconv.Atoi(m[1:])
		return "$" + strconv.Itoa(base+n)
	})
	q.Where += " AND " + term
	q.Args = append(slices.Clone(q.Args), args...)
	return q
}

func TestBudgetedCountPlanHasNoGatherOnTheKitShapes(t *testing.T) {
	e := newSGEnvSized(t, 40000)
	ctx := t.Context()
	alice := e.agent(t, "alice", e.big)
	pt := e.sideRows(t, "text", "pt", "'x'")
	pb := e.sideRows(t, "bool", "pb", "true")
	for _, tbl := range []string{"rfv_text", "rfv_bool"} {
		if _, err := e.super.Exec(ctx, "VACUUM (ANALYZE) "+tbl); err != nil {
			t.Fatal(err)
		}
	}

	trace := &execTrace{}
	cfg := e.etEnv.pool.Config().Copy()
	cfg.MaxConns = 1 // one backend, so the guard reads the connection the explains used
	cfg.ConnConfig.Tracer = trace
	traced, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(traced.Close)

	shapes := []struct {
		name string
		q    func(p identity.Principal) pipeline.ListQuery
	}{
		{"a, kp priority filter and default ACL", e.query},
		{"b, a6 three broad fields", func(p identity.Principal) pipeline.ListQuery {
			q := andExists(e.query(p), fieldindex.TypeText, pt, "x")
			return andExists(q, fieldindex.TypeBool, pb, true)
		}},
	}
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			q := s.q(alice)
			q.SkipCount = false
			if len(q.Branches) < 2 {
				t.Fatalf("branches %v: the default ACL reads as one branch per subject", q.Branches)
			}
			// The real list, so the real Exec text and its budgeted path.
			trace.reset()
			var page *pipeline.ListPage
			if err := tenancy.InTenantTx(ctx, traced, e.a.id, func(tx pgx.Tx) error {
				var err error
				page, err = pipeline.List(ctx, tx, q)
				return err
			}); err != nil {
				t.Fatal(err)
			}
			if len(page.Records) <= 50 {
				t.Fatalf("page of %d records, state %q: the list must hold a next page so the count runs", len(page.Records), page.TotalState)
			}
			open := trace.savepointExec()
			if open == "" {
				t.Fatal("no SAVEPOINT list_count Exec captured: the list took no budgeted count")
			}
			args := append([]any{e.a.id, e.def}, q.Args...)
			explain := func(prelude string) string {
				var raw string
				err := tenancy.InTenantTx(ctx, traced, e.a.id, func(tx pgx.Tx) error {
					for _, st := range []string{
						`SET LOCAL parallel_setup_cost = 0`, `SET LOCAL parallel_tuple_cost = 0`,
						`SET LOCAL min_parallel_table_scan_size = 0`, `SET LOCAL min_parallel_index_scan_size = 0`,
						prelude,
					} {
						if _, err := tx.Exec(ctx, st); err != nil {
							return err
						}
					}
					return tx.QueryRow(ctx, `EXPLAIN (FORMAT JSON) `+pipeline.CountStatement(q), args...).Scan(&raw)
				})
				if err != nil {
					t.Fatal(err)
				}
				return raw
			}
			if !planHasGather(t, explain(`SAVEPOINT list_count; SET LOCAL statement_timeout = '170'`)) {
				t.Fatal("control: no Gather with only statement_timeout set; the fixture proves nothing")
			}
			if planHasGather(t, explain(open)) {
				t.Error("the count plan holds a Gather with the count's SET LOCALs in force")
			}
		})
	} // Nothing survives on the connection: the next transaction has the
	// server default of every setting the explains changed.
	for _, name := range []string{"parallel_setup_cost", "parallel_tuple_cost", "min_parallel_table_scan_size", "min_parallel_index_scan_size"} {
		var got, want string
		err := tenancy.InTenantTx(ctx, traced, e.a.id, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT setting, reset_val FROM pg_settings WHERE name = $1`, name).Scan(&got, &want)
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("next transaction: %s = %q, want the server default %q", name, got, want)
		}
	}
}
