//go:build bench

// TICK-637 benchmark: the records list's exact count(*) and page under the
// compiled ACL predicate, at 1M (tenant L) and 100k (tenant M) incident
// records. The predicate comes from the real authz.CompileRead; the count and
// page SQL are internal/records/store/list.go's text, built the same way as
// internal/server/records_list.go builds it (default sort created_at DESC,
// limit 50 + 1, no filters). Each execution runs in its own transaction with
// the transaction-local app.tenant_id, as nava_app, in QueryExecModeExec (the
// server's mode, so every execution gets a custom plan).
//
//	go run -tags bench ./scratch/tick-637-list-count -dsn 'postgres://nava_app:bench@localhost:55437/nava' -out scratch/tick-637-list-count/out
package main

import (
	"context"
	"crypto/md5" //nolint:gosec // matches seed.sql's md5-derived ids, not security
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// id is seed.sql's pg_temp.uid: md5(tenant || kind || n)::uuid.
func id(t uuid.UUID, kind string, n int) uuid.UUID {
	sum := md5.Sum([]byte(fmt.Sprintf("%s%s%d", t, kind, n))) //nolint:gosec // see import
	u, _ := uuid.FromBytes(sum[:])
	return u
}

type tenant struct {
	name string
	id   uuid.UUID
}

type scenario struct {
	name string
	pol  *authz.Policy
	p    identity.Principal
}

// defaultIncident is the assumed onboarding default (P3-35 is not merged):
// requester, assignee and shares read their own; agents read their teams'
// queue; the service desk reads everything.
func defaultIncident(t uuid.UUID) *authz.Policy {
	rw := []string{authz.OpRead, authz.OpWrite}
	return &authz.Policy{Version: 1, Rules: []authz.Rule{
		{ID: "own", Ops: rw, Who: authz.Subject{Requester: true, Assignee: true, Shared: true}},
		{ID: "team", Ops: rw, Who: authz.Subject{Groups: []uuid.UUID{id(t, "g", 1)}},
			Where: []authz.Cond{{Column: "assignment_group_id", Op: authz.CondIn, Caller: authz.CallerMyGroups}}},
		{ID: "desk", Ops: rw, Who: authz.Subject{Groups: []uuid.UUID{id(t, "g", 2)}}},
	}}
}

// shareHeavy grants reads through shares (and own requests) only.
func shareHeavy() *authz.Policy {
	return &authz.Policy{Version: 1, Rules: []authz.Rule{
		{ID: "shared", Ops: []string{authz.OpRead}, Who: authz.Subject{Shared: true}},
		{ID: "own", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true}},
	}}
}

func scenarios(t uuid.UUID) []scenario {
	p := func(user int, groups ...int) identity.Principal {
		pr := identity.Principal{TenantID: t, PrincipalID: id(t, "u", user)}
		for _, g := range groups {
			pr.Groups = append(pr.Groups, id(t, "g", g))
		}
		return pr
	}
	return []scenario{
		{"default/end-user", defaultIncident(t), p(300, 30)},
		{"default/agent-3-teams", defaultIncident(t), p(301, 1, 10, 11, 12)},
		{"default/service-desk", defaultIncident(t), p(302, 2)},
		{"share-heavy/partner", shareHeavy(), p(303, 0, 20, 21)},
	}
}

const base = ` FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND `

const cols = `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.created_at::text`

type query struct {
	name string
	sql  string
	args []any
}

func queries(ctx context.Context, pool *pgxpool.Pool, t tenant, s scenario) ([]query, error) {
	def := id(t.id, "d", 0)
	acl, aclArgs := authz.CompileRead(s.pol, s.p, "r", 2)
	if *rewrite {
		acl = rewriteShare(acl)
	}
	where := acl + " AND true"
	args := append([]any{t.id, def}, aclArgs...)
	n := len(args)
	order := ` ORDER BY r.created_at DESC NULLS LAST, r.id ASC LIMIT `
	qs := []query{
		{"count exact", `SELECT count(*)` + base + where, args},
		{"count capped 1001", `SELECT count(*) FROM (SELECT 1` + base + where + ` LIMIT 1001) x`, args},
		{"page 1", cols + base + where + order + fmt.Sprintf("$%d", n+1), append(slices.Clone(args), 51)},
	}
	// Page 20: the keyset after row 950, as records_list.go renders it.
	var cts string
	var cid uuid.UUID
	err := inTx(ctx, pool, t.id, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT r.created_at::text, r.id`+base+where+
			` ORDER BY r.created_at DESC NULLS LAST, r.id ASC OFFSET 949 LIMIT 1`, args...).Scan(&cts, &cid)
	})
	if err == nil {
		ks := fmt.Sprintf(" AND (r.created_at < $%[1]d::timestamptz OR r.created_at IS NULL OR (r.created_at = $%[1]d::timestamptz AND r.id > $%[2]d::uuid))", n+1, n+2)
		qs = append(qs, query{"page 20", cols + base + where + ks + order + fmt.Sprintf("$%d", n+3),
			append(slices.Clone(args), cts, cid.String(), 51)})
	} else if err != pgx.ErrNoRows {
		return nil, err
	}
	return qs, nil
}

// rewriteShare renders CompileRead's correlated share EXISTS as an
// uncorrelated InitPlan array, r.id = ANY(ARRAY(SELECT record_id ...)), so
// every arm of the OR can be a bitmap index scan (candidate fix F2).
func rewriteShare(acl string) string {
	const from = "EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND "
	const to = "r.id = ANY(ARRAY(SELECT acl_share.record_id FROM record_shares acl_share WHERE acl_share.tenant_id = $1 AND "
	if !strings.Contains(acl, from) {
		return acl
	}
	acl = strings.Replace(acl, from, to, 1)
	return strings.Replace(acl, "::uuid[]))))", "::uuid[])))))", 1)
}

var (
	rewrite = flag.Bool("rewrite", false, "render the share arm as an InitPlan array (F2)")
	jitOff  = flag.Bool("jitoff", false, "SET LOCAL jit = off in every transaction (F1)")
)

func inTx(ctx context.Context, pool *pgxpool.Pool, t uuid.UUID, fn func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, t.String()); err != nil {
			return err
		}
		if *jitOff {
			if _, err := tx.Exec(ctx, `SET LOCAL jit = off`); err != nil {
				return err
			}
		}
		return fn(tx)
	})
}

func run(ctx context.Context, pool *pgxpool.Pool, t uuid.UUID, q query) (time.Duration, error) {
	var d time.Duration
	err := inTx(ctx, pool, t, func(tx pgx.Tx) error {
		start := time.Now()
		rows, err := tx.Query(ctx, q.sql, q.args...)
		if err != nil {
			return err
		}
		for rows.Next() {
		}
		rows.Close()
		d = time.Since(start)
		return rows.Err()
	})
	return d, err
}

func pct(ds []time.Duration, p float64) float64 {
	s := slices.Clone(ds)
	slices.Sort(s)
	i := int(float64(len(s))*p+0.999999) - 1
	i = max(0, min(i, len(s)-1))
	return float64(s[i].Microseconds()) / 1000
}

func explain(ctx context.Context, pool *pgxpool.Pool, t uuid.UUID, q query) (string, error) {
	var b strings.Builder
	err := inTx(ctx, pool, t, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+q.sql, q.args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				return err
			}
			b.WriteString(line + "\n")
		}
		return rows.Err()
	})
	return b.String(), err
}

func main() {
	dsn := flag.String("dsn", "", "nava_app DSN")
	out := flag.String("out", "out", "directory for plans")
	n := flag.Int("n", 30, "timed executions per query")
	clients := flag.Int("clients", 1, "concurrent clients per query")
	label := flag.String("label", "baseline", "run label")
	only := flag.String("only", "", "comma-separated query names to run (empty: all)")
	flag.Parse()
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(*dsn)
	if err != nil {
		panic(err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	cfg.MaxConns = int32(max(*clients, 2))
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	dir := filepath.Join(*out, *label)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	plans, err := os.Create(filepath.Join(dir, "plans.txt"))
	if err != nil {
		panic(err)
	}
	defer plans.Close()
	fmt.Printf("| Tenant | Scenario | Query | Visible rows | p50 ms | p95 ms | max ms | Buffers (EXPLAIN) |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, t := range []tenant{
		{"L (1M)", uuid.MustParse("00000000-0000-4000-8000-00000000000a")},
		{"M (100k)", uuid.MustParse("00000000-0000-4000-8000-00000000000b")},
	} {
		for _, s := range scenarios(t.id) {
			qs, err := queries(ctx, pool, t, s)
			if err != nil {
				panic(fmt.Errorf("%s %s: %w", t.name, s.name, err))
			}
			var visible int64
			if err := inTx(ctx, pool, t.id, func(tx pgx.Tx) error {
				return tx.QueryRow(ctx, qs[0].sql, qs[0].args...).Scan(&visible)
			}); err != nil {
				panic(err)
			}
			for _, q := range qs {
				if *only != "" && !slices.Contains(strings.Split(*only, ","), q.name) {
					continue
				}
				plan, err := explain(ctx, pool, t.id, q)
				if err != nil {
					panic(fmt.Errorf("%s %s %s: %w", t.name, s.name, q.name, err))
				}
				fmt.Fprintf(plans, "==== %s | %s | %s\n%s\n%s\n", t.name, s.name, q.name, q.sql, plan)
				buf := ""
				for _, l := range strings.Split(plan, "\n") {
					if strings.Contains(l, "Buffers:") {
						buf = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "Buffers:"))
						break
					}
				}
				var mu sync.Mutex
				var ds []time.Duration
				var wg sync.WaitGroup
				var ferr error
				for c := 0; c < *clients; c++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for i := 0; i < *n; i++ {
							d, err := run(ctx, pool, t.id, q)
							mu.Lock()
							if err != nil {
								ferr = err
							}
							ds = append(ds, d)
							mu.Unlock()
						}
					}()
				}
				wg.Wait()
				if ferr != nil {
					panic(ferr)
				}
				fmt.Printf("| %s | %s | %s | %d | %.1f | **%.1f** | %.1f | %s |\n", t.name, s.name, q.name, visible,
					pct(ds, 0.5), pct(ds, 0.95), pct(ds, 1), buf)
			}
		}
	}
}
