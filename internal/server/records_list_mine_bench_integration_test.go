//go:build integration

package server_test

import (
	"context"
	"fmt"
	neturl "net/url"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

var mineExecRE = regexp.MustCompile(`Execution Time: ([0-9.]+) ms`)

// TICK-644 (P3-34c) bench, not a test of behaviour: the requester (mine) list
// on P3-01's 800k shape as the single OR statement and as the UNION ALL of
// branches, built by the real authz and pipeline code. Skipped unless
// NAVA_TICK644_URL names the superuser URL of the database seeded by
// scratch/tick-640/setup.go -seed; NAVA_TICK644_PLANS=1 prints the plans.
// See scratch/tick-644/README.md.
func TestMineListBench(t *testing.T) {
	url := os.Getenv("NAVA_TICK644_URL")
	if url == "" {
		t.Skip("NAVA_TICK644_URL is not set")
	}
	plans := os.Getenv("NAVA_TICK644_PLANS") == "1"
	runs := 7
	ctx := context.Background()
	u, err := neturl.Parse(url)
	mineMust(err)
	u.User = neturl.UserPassword("nava_app", "bench-pw-nava_app")
	pool, err := pgxpool.New(ctx, u.String())
	mineMust(err)
	t.Cleanup(pool.Close)
	tenant := uuid.MustParse("809d4580-aaed-4156-5abc-38d58f77f840")
	def := uuid.MustParse("22a88e5c-3a76-b13d-e14d-5bdef46b3dc2")
	actors := map[string]uuid.UUID{
		"req1": uuid.MustParse("fccbb3fd-048d-a9ac-ec41-5c1548c480a0"),
		"req8": uuid.MustParse("f222cde6-739e-cb4e-eaa6-14c1a2bed245"),
	}
	pol := &authz.Policy{Version: 1, Rules: []authz.Rule{{ID: "mine", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true, Assignee: true, Shared: true}}}}
	names := make([]string, 0, len(actors))
	for n := range actors {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		p := identity.Principal{TenantID: tenant, PrincipalID: actors[n]}
		cursor := minePage20(ctx, pool, tenant, def, pol, p)
		for _, page := range []int{1, 20} {
			for _, form := range []string{"before (single OR)", "after (UNION ALL)"} {
				q := mineQuery(tenant, def, pol, p, form[0] == 'a')
				if page == 20 {
					q.Keyset = "(r.updated_at, r.id) < ($" + fmt.Sprint(2+len(q.Args)+1) + "::timestamptz, $" + fmt.Sprint(2+len(q.Args)+2) + "::uuid)"
					q.KeysetArgs = []any{cursor.ts, cursor.id}
				}
				pageSQL, pageArgs, err := pipeline.PageStatement(q)
				mineMust(err)
				countArgs := append([]any{tenant, def}, q.Args...)
				if plans {
					fmt.Printf("\n=== %s page %d %s: page statement (EXPLAIN ANALYZE, BUFFERS)\n", n, page, form)
				}
				pg := mineExplain(ctx, pool, tenant, pageSQL, pageArgs, runs, plans)
				if plans {
					fmt.Printf("\n=== %s page %d %s: count statement (EXPLAIN ANALYZE, BUFFERS)\n", n, page, form)
				}
				ct := mineExplain(ctx, pool, tenant, pipeline.CountStatement(q), countArgs, runs, plans)
				fmt.Printf("%-5s page %-2d %-18s page stmt median %7.2f ms (min %.2f), count stmt median %7.2f ms (min %.2f)\n", n, page, form, pg.med, pg.min, ct.med, ct.min)
			}
		}
	}
}

type mineCur struct {
	ts time.Time
	id uuid.UUID
}

func mineQuery(tenant, def uuid.UUID, pol *authz.Policy, p identity.Principal, branched bool) pipeline.ListQuery {
	q := pipeline.ListQuery{TenantID: tenant, DefinitionID: def, SortColumn: "updated_at", Desc: true, Limit: 51}
	if branched {
		q.Branches, q.Args = authz.ReadBranches(pol, p, "r", 2)
		q.Where = "true"
		return q
	}
	q.Where, q.Args = authz.CompileRead(pol, p, "r", 2)
	return q
}

// page20 is the cursor after 950 rows (page 20 starts at the 951st).
func minePage20(ctx context.Context, pool *pgxpool.Pool, tenant, def uuid.UUID, pol *authz.Policy, p identity.Principal) mineCur {
	where, args := authz.CompileRead(pol, p, "r", 2)
	var c mineCur
	mineMust(tenancy.InTenantTx(ctx, pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT updated_at, id FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND `+where+` ORDER BY updated_at DESC, id DESC OFFSET 950 LIMIT 1`,
			append([]any{tenant, def}, args...)...).Scan(&c.ts, &c.id)
	}))
	return c
}

type mineTiming struct{ med, min float64 }

func mineExplain(ctx context.Context, pool *pgxpool.Pool, tenant uuid.UUID, sql string, args []any, runs int, show bool) mineTiming {
	var ms []float64
	for i := 0; i < runs+1; i++ { // the first run warms the cache
		var sb strings.Builder
		mineMust(tenancy.InTenantTx(ctx, pool, tenant, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+sql, args...)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var line string
				if err := rows.Scan(&line); err != nil {
					return err
				}
				sb.WriteString(line + "\n")
			}
			return rows.Err()
		}))
		var v float64
		fmt.Sscan(mineExecRE.FindStringSubmatch(sb.String())[1], &v)
		if i > 0 {
			ms = append(ms, v)
		}
		if show && i == runs {
			fmt.Println(sb.String())
		}
	}
	slices.Sort(ms)
	return mineTiming{ms[len(ms)/2], ms[0]}
}

func mineMust(err error) {
	if err != nil {
		panic(err)
	}
}
