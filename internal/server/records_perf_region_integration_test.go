//go:build integration && perf

// TICK-1037 (S31, ADR-010-A2 A2-11): R1 through the handler. Each "region:"
// row is the list a caller with a region policy gets, timed through the real
// handler, so ResolveUnder, the estimate read, the probe (ChooseListShapes)
// and the count and page are all inside the timed request. A row's p95 is
// scored against 150 ms directly.
//
// Data, on the P3-36 seed (large tenant only), ported from the bench
// harnesses so the sets are the crossover cells' (docs/adr/ADR-010/A2/):
//   - the SP-1 tree (bench/acl-perf/region/region.sql): root, 10 regions, 100
//     districts, 1,000 areas, 9,000 branches = 10,111 units, in org_units,
//     with every live path in org_unit_closure; records.org_unit_id is the
//     requester's primary branch (5% NULL);
//   - SP-1c's district hot (pre 2327) and thin (pre 2428);
//   - SP-1e's time-uniform districts m = 1,001 / 3,000 / 10,000 and sp1x's
//     m = 4,500 (region 3);
//   - sp1x's bench_rare (500 records, one per random branch) and bench_conc
//     (20,000 records in district pre 9101), here two real entity types made
//     through the API;
//   - SP-1e's "district mixed" is not built: it is not one of TICK-1037's sets.
//
// Then the TICK-1071 refresher (estimates.Refresher, as nava_worker) runs one
// cycle, so N̂_e and K* are the job's own.
//
// The ACL of ticket, bench_rare and bench_conc is the P3-36 document in
// format 2 plus a last policy "regional": member_of the regional group AND
// under the caller's units. Last is the worst branch position (TICK-951).
//
// Personas, both in the regional group:
//   - agents: the 200 P3-36 agents (desk, mine, regional), whose desk branch
//     fills the page;
//   - count-only: a principal whose only grant is the regional policy (no
//     record names them as requester or assignee, nothing is shared), as
//     S31's in_my_region acceptance user.
// Each row sets the personas' org_unit_members to the set's units.
//
// Method (A2-11): paired wall-clock, 1 client. A row's requests alternate a
// scored request with a base request: the same P3-36 agent on the same path
// without the regional group, so the regional policy folds away (no under
// leaf, no probe). The base latencies are reported beside the row (pairLat),
// for information. The floor gate and the container watch are P3-36's.

package server_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/estimates"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// perfRegionPage is the page statement's LIMIT the list handler passes to
// ChooseListShapes (defaultRecordsLimit + 1).
const perfRegionPage = 51

// perfRegionSet is one region row's data: the caller's units U (by pre-order
// position in bench_ou) and the entity type listed.
type perfRegionSet struct {
	name string
	pres []int
	key  string // entity type key
}

// perfRegionSets are TICK-1037's sets, in the order the rows run.
func perfRegionSets() []perfRegionSet {
	regions08 := make([]int, 0, 9)
	for r := range 9 {
		regions08 = append(regions08, 1+r*1011)
	}
	return []perfRegionSet{
		{"area (|S| 10)", []int{3}, "ticket"},
		{"district (|S| 101)", []int{2}, "ticket"},
		{"region (|S| 1,011)", []int{1}, "ticket"},
		{"root (|S| 10,111)", []int{0}, "ticket"},
		{"district hot", []int{2327}, "ticket"},
		{"district thin", []int{2428}, "ticket"},
		{"uniform m=1001", []int{3035}, "ticket"},
		{"uniform m=3000", []int{3136}, "ticket"},
		{"uniform m=4500", []int{3338}, "ticket"},
		{"uniform m=10000", []int{3237}, "ticket"},
		{"bench_rare at root", []int{0}, "bench_rare"},
		{"bench_conc, regions 0-8 (|S| 9,099)", regions08, "bench_conc"},
	}
}

// perfRegion is the region bench's state once its setup ran.
type perfRegion struct {
	regional uuid.UUID // the regional group
	only     identity.Principal
	defs     map[string]uuid.UUID // type key -> definition id
	units    map[int]uuid.UUID    // pre -> org unit id
	current  string               // the set whose memberships are in place
	report   strings.Builder
}

// mdCell escapes the pipes in a set name ("|S|") for a markdown table cell.
func mdCell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

// isRegionRow reports whether a scenario is a TICK-1037 region row.
func isRegionRow(name string) bool { return strings.HasPrefix(name, "region: ") }

func perfRegionName(s perfRegionSet, persona string) string {
	return "region: " + s.name + ", " + persona
}

// regionActors registers the personas' tokens once: ragent-N is agent-N in
// the regional group; region-only is a fresh principal in that group only.
// agent-N's own requests are the pair's base.
func (e *perfEnv) regionActors(t *testing.T) {
	t.Helper()
	if _, ok := e.tokens["region-only"]; ok {
		return
	}
	regional := uuid.NewMD5(uuid.NameSpaceOID, []byte("regional/0"))
	e.exec(t, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
		e.large.id, regional, "perf-regional-0", "/t/"+e.large.id.String()+"/regional")
	only := identity.Principal{TenantID: e.large.id, PrincipalID: uuid.NewMD5(uuid.NameSpaceOID, []byte("region-only/0")), Groups: []uuid.UUID{regional}}
	e.exec(t, `INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
		VALUES ($1, $2, 'perf-region-only', 'region-only@large.test', true, '[]', now()) ON CONFLICT DO NOTHING`, only.TenantID, only.PrincipalID)
	e.exec(t, "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
		e.large.id, regional, only.PrincipalID)
	e.tokens["region-only"] = only
	for _, tok := range e.agents {
		p := e.tokens[tok]
		g := append(append([]uuid.UUID{}, p.Groups...), regional)
		e.tokens["r"+tok] = identity.Principal{TenantID: p.TenantID, PrincipalID: p.PrincipalID, Groups: g}
		e.agentGroups["r"+tok] = e.agentGroups[tok]
		e.pairBase[tok] = true
	}
	e.region = &perfRegion{regional: regional, only: only, defs: map[string]uuid.UUID{}, units: map[int]uuid.UUID{}}
}

// regionReqs interleaves each scored request with its base: the scored one
// first, so the warm-up and the captured plan are the region request's.
func (e *perfEnv) regionReqs(t *testing.T, key, persona string) []perfReq {
	t.Helper()
	e.regionActors(t)
	path := "/api/v1/records/" + key + "?sort=-updated_at"
	out := make([]perfReq, 0, 2*len(e.agents))
	for _, tok := range e.agents {
		scored := "r" + tok
		if persona == "count-only" {
			scored = "region-only"
		}
		out = append(out, perfReq{scored, path, ""}, perfReq{tok, path, ""})
	}
	return out
}

// regionDefs are the region rows: every set, agents then count-only, 1
// client each (A2-11). NAVA_PERF_ONLY='^region:' selects them alone.
func (e *perfEnv) regionDefs(t *testing.T) []perfDef {
	t.Helper()
	var out []perfDef
	for _, s := range perfRegionSets() {
		for _, persona := range []string{"agents", "count-only"} {
			out = append(out, perfDef{perfRegionName(s, persona), "list", "large", func() []perfReq { return e.regionReqs(t, s.key, persona) }, 1})
		}
	}
	return out
}

func (e *perfEnv) regionExtras(t *testing.T) map[string]perfExtra {
	t.Helper()
	m := map[string]perfExtra{}
	for _, s := range perfRegionSets() {
		for _, persona := range []string{"agents", "count-only"} {
			m[perfRegionName(s, persona)] = perfExtra{before: func() { e.regionBefore(t, s) }}
		}
	}
	return m
}

// regionBefore builds the data once, then puts the personas in s's units.
func (e *perfEnv) regionBefore(t *testing.T, s perfRegionSet) {
	t.Helper()
	e.regionActors(t)
	if len(e.region.defs) == 0 {
		e.regionSetup(t)
	}
	if e.region.current == s.name {
		return
	}
	ids := make([]uuid.UUID, 0, len(s.pres))
	for _, p := range s.pres {
		ids = append(ids, e.region.units[p])
	}
	principals := []uuid.UUID{e.region.only.PrincipalID}
	for _, tok := range e.agents {
		principals = append(principals, e.tokens[tok].PrincipalID)
	}
	e.exec(t, "DELETE FROM org_unit_members WHERE tenant_id = $1", e.large.id)
	e.exec(t, `INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id)
		SELECT $1, u, p FROM unnest($2::uuid[]) u, unnest($3::uuid[]) p ON CONFLICT DO NOTHING`, e.large.id, ids, principals)
	e.exec(t, "ANALYZE org_unit_members")
	e.region.current = s.name
	e.regionSetLine(t, s)
}

// regionSetLine records what the row measures: |S|, m (records of the type
// under S), N̂_e as the refresher wrote it, K* and the shape A2-5 picks.
func (e *perfEnv) regionSetLine(t *testing.T, s perfRegionSet) {
	t.Helper()
	ids := make([]uuid.UUID, 0, len(s.pres))
	for _, p := range s.pres {
		ids = append(ids, e.region.units[p])
	}
	def := e.region.defs[s.key]
	var units, m, ne int64
	if err := e.super.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = ANY($2)),
		(SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $3 AND org_unit_id IN
		    (SELECT descendant_id FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = ANY($2))),
		(SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definition_id = $3)`,
		e.large.id, ids, def).Scan(&units, &m, &ne); err != nil {
		t.Fatalf("region set %s: %v", s.name, err)
	}
	k := authz.KStar(perfRegionPage, ne, true)
	shape := "W0 (m > K*)"
	switch {
	case ne <= int64(k):
		shape = "A2-5a rare: W0, no probe"
	case m <= int64(k):
		shape = "C3 (m <= K*)"
	}
	line := fmt.Sprintf("| %s | %s | %d | %d | %d | %d | %s |", mdCell(s.name), s.key, units, m, ne, k, shape)
	t.Logf("REGION SET %s", line)
	fmt.Fprintln(&e.region.report, line)
}

// perfRegionSetupSQL is region.sql, SP-1c's setupSQL, SP-1e's set (5) and
// sp1x's xSetupSQL on the real schema. $T is the large tenant, $D the ticket
// definition, $R and $C the bench_rare and bench_conc definitions.
const perfRegionSetupSQL = `
SET session_replication_role = replica;
DROP TABLE IF EXISTS bench_ou;
CREATE TABLE bench_ou (pre int PRIMARY KEY, id uuid NOT NULL, parent uuid, depth int NOT NULL, leaf_no int);
INSERT INTO bench_ou (pre, id, parent, depth, leaf_no)
SELECT 0, md5('ou/root')::uuid, NULL::uuid, 0, NULL::int
UNION ALL
SELECT 1 + r * 1011, md5('ou/' || r)::uuid, md5('ou/root')::uuid, 1, NULL::int FROM generate_series(0, 9) r
UNION ALL
SELECT 1 + r * 1011 + 1 + d * 101, md5('ou/' || r || '/' || d)::uuid, md5('ou/' || r)::uuid, 2, NULL
FROM generate_series(0, 9) r, generate_series(0, 9) d
UNION ALL
SELECT 1 + r * 1011 + 1 + d * 101 + 1 + a * 10, md5('ou/' || r || '/' || d || '/' || a)::uuid, md5('ou/' || r || '/' || d)::uuid, 3, NULL
FROM generate_series(0, 9) r, generate_series(0, 9) d, generate_series(0, 9) a
UNION ALL
SELECT 1 + r * 1011 + 1 + d * 101 + 1 + a * 10 + 1 + b, md5('ou/' || r || '/' || d || '/' || a || '/' || b)::uuid,
       md5('ou/' || r || '/' || d || '/' || a)::uuid, 4, ((r * 10 + d) * 10 + a) * 9 + b
FROM generate_series(0, 9) r, generate_series(0, 9) d, generate_series(0, 9) a, generate_series(0, 8) b;
INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name)
SELECT $T, o.id, o.parent, 'bench-ou-' || o.pre, 'ou-' || o.pre FROM bench_ou o ORDER BY o.pre;
-- Every live path: d is in a's subtree when d.pre is in [a.pre, a.pre + size(a)).
INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
SELECT $T, a.id, d.id FROM bench_ou a JOIN bench_ou d
  ON d.pre >= a.pre AND d.pre < a.pre + (ARRAY[10111, 1011, 101, 10, 1])[a.depth + 1];
DROP TABLE IF EXISTS bench_primary;
CREATE TABLE bench_primary AS
SELECT md5('u0/' || u)::uuid AS principal_id,
       CASE WHEN (('x' || substr(md5('oum' || u), 1, 7))::bit(28)::int) % 20 = 0 THEN NULL
            ELSE (SELECT id FROM bench_ou WHERE leaf_no = (('x' || substr(md5('ou' || u), 1, 7))::bit(28)::int) % 9000)
       END AS org_unit_id
FROM generate_series(0, 4999) u;
UPDATE records r SET org_unit_id = p.org_unit_id FROM bench_primary p
WHERE r.tenant_id = $T AND r.requester_id = p.principal_id;
-- SP-1c: district hot (pre 2327) and thin (pre 2428).
UPDATE records r SET org_unit_id = (SELECT id FROM bench_ou WHERE pre = 2327 + 1 + 1 + 3)
WHERE r.id IN (
  SELECT id FROM records WHERE tenant_id = $T AND org_unit_id IN (SELECT id FROM bench_ou WHERE pre >= 1 + 9*1011 AND pre < 1 + 10*1011)
  ORDER BY md5(id::text) LIMIT 20000);
UPDATE records r SET org_unit_id = (SELECT id FROM bench_ou WHERE pre = 1 + 9*1011 + 1 + 1 + 1)
WHERE r.id IN (
  SELECT id FROM records WHERE tenant_id = $T AND org_unit_id IN (SELECT id FROM bench_ou WHERE pre >= 2428 AND pre < 2529)
  ORDER BY md5(id::text) OFFSET 600);
-- SP-1e (5): time-uniform sparse districts in region 3: empty them, then move m random tickets of regions 4-8 in.
UPDATE records SET org_unit_id = (SELECT id FROM bench_ou WHERE pre = 9103)
 WHERE tenant_id = $T AND org_unit_id IN (SELECT id FROM bench_ou WHERE pre >= 3035 AND pre < 3338);
WITH src AS (SELECT id, row_number() OVER () AS rn FROM (SELECT id FROM records WHERE tenant_id = $T AND entity_definition_id = $D
        AND org_unit_id IN (SELECT id FROM bench_ou WHERE pre >= 4045 AND pre < 9100) ORDER BY md5(id::text || 'u1') LIMIT 1001) z),
     br AS (SELECT id, row_number() OVER (ORDER BY pre) - 1 AS bn FROM bench_ou WHERE depth = 4 AND pre >= 3035 AND pre < 3136)
UPDATE records r SET org_unit_id = br.id FROM src, br WHERE r.tenant_id = $T AND r.id = src.id AND br.bn = src.rn % 90;
WITH src AS (SELECT id, row_number() OVER () AS rn FROM (SELECT id FROM records WHERE tenant_id = $T AND entity_definition_id = $D
        AND org_unit_id IN (SELECT id FROM bench_ou WHERE pre >= 4045 AND pre < 9100) ORDER BY md5(id::text || 'u3') LIMIT 3000) z),
     br AS (SELECT id, row_number() OVER (ORDER BY pre) - 1 AS bn FROM bench_ou WHERE depth = 4 AND pre >= 3136 AND pre < 3237)
UPDATE records r SET org_unit_id = br.id FROM src, br WHERE r.tenant_id = $T AND r.id = src.id AND br.bn = src.rn % 90;
WITH src AS (SELECT id, row_number() OVER () AS rn FROM (SELECT id FROM records WHERE tenant_id = $T AND entity_definition_id = $D
        AND org_unit_id IN (SELECT id FROM bench_ou WHERE pre >= 4045 AND pre < 9100) ORDER BY md5(id::text || 'u10') LIMIT 10000) z),
     br AS (SELECT id, row_number() OVER (ORDER BY pre) - 1 AS bn FROM bench_ou WHERE depth = 4 AND pre >= 3237 AND pre < 3338)
UPDATE records r SET org_unit_id = br.id FROM src, br WHERE r.tenant_id = $T AND r.id = src.id AND br.bn = src.rn % 90;
-- sp1x: uniform m = 4,500 (district pre 3338).
UPDATE records SET org_unit_id = (SELECT id FROM bench_ou WHERE pre = 9103)
 WHERE tenant_id = $T AND org_unit_id IN (SELECT id FROM bench_ou WHERE pre >= 3338 AND pre < 3439);
WITH src AS (SELECT id, row_number() OVER () AS rn FROM (SELECT id FROM records WHERE tenant_id = $T AND entity_definition_id = $D
        AND org_unit_id IN (SELECT id FROM bench_ou WHERE pre >= 4045 AND pre < 9100) ORDER BY md5(id::text || 'u45') LIMIT 4500) z),
     br AS (SELECT id, row_number() OVER (ORDER BY pre) - 1 AS bn FROM bench_ou WHERE depth = 4 AND pre >= 3338 AND pre < 3439)
UPDATE records r SET org_unit_id = br.id FROM src, br WHERE r.tenant_id = $T AND r.id = src.id AND br.bn = src.rn % 90;
-- sp1x: bench_rare (500 records, one per random branch) and bench_conc
-- (20,000 records in district pre 9101, region 9), copies of random tickets.
DROP TABLE IF EXISTS bench_x_src;
CREATE UNLOGGED TABLE bench_x_src AS SELECT * FROM records WHERE tenant_id = $T AND entity_definition_id = $D ORDER BY md5(id::text || 'rare') LIMIT 500;
UPDATE bench_x_src m SET id = gen_random_uuid(), entity_definition_id = $R,
  org_unit_id = (SELECT b.id FROM bench_ou b WHERE b.leaf_no = abs(hashtext(m.id::text || 'r')) % 9000);
INSERT INTO bench_x_src SELECT * FROM records WHERE tenant_id = $T AND entity_definition_id = $D ORDER BY md5(id::text || 'conc') LIMIT 20000;
UPDATE bench_x_src m SET id = gen_random_uuid(), entity_definition_id = $C,
  org_unit_id = (SELECT b.id FROM bench_ou b WHERE b.depth = 4 AND b.pre >= 9101 AND b.pre < 9202 ORDER BY b.pre OFFSET (abs(hashtext(m.id::text || 'c')) % 90) LIMIT 1)
  WHERE m.entity_definition_id = $D;
DO $$ DECLARE c text; BEGIN
  SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position) INTO c FROM information_schema.columns
   WHERE table_schema = 'public' AND table_name = 'records' AND is_generated = 'NEVER';
  EXECUTE format('INSERT INTO records (%s) SELECT %s FROM bench_x_src', c, c);
END $$;
DROP TABLE bench_x_src;
SET session_replication_role = origin;
`

// regionSetup creates the two bench types through the API, builds the data,
// compacts and warms the table, runs one estimates cycle and publishes the
// region ACL on the three types.
func (e *perfEnv) regionSetup(t *testing.T) {
	t.Helper()
	ctx := t.Context()
	start := time.Now()
	for _, k := range []string{"bench_rare", "bench_conc"} {
		body := strings.Replace(perfTypeBody(), `"key": "ticket", "name": "Ticket", "number_prefix": "TKT"`,
			fmt.Sprintf(`"key": %q, "name": %q, "number_prefix": %q`, k, k, strings.ToUpper(k[6:10])), 1)
		if rec := e.serve(ctx, http.MethodPost, "/api/v1/entity-types", "admin-large", "perf-type-"+k, body); rec.Code != http.StatusCreated {
			t.Fatalf("create type %s = %d: %s", k, rec.Code, rec.Body)
		}
	}
	for _, k := range []string{"ticket", "bench_rare", "bench_conc"} {
		var id uuid.UUID
		if err := e.super.QueryRow(ctx, "SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2", e.large.id, k).Scan(&id); err != nil {
			t.Fatalf("definition %s: %v", k, err)
		}
		e.region.defs[k] = id
	}
	q := strings.NewReplacer("$T", "'"+e.large.id.String()+"'::uuid", "$D", "'"+e.region.defs["ticket"].String()+"'::uuid",
		"$R", "'"+e.region.defs["bench_rare"].String()+"'::uuid", "$C", "'"+e.region.defs["bench_conc"].String()+"'::uuid").Replace(perfRegionSetupSQL)
	if _, err := e.super.Exec(ctx, q); err != nil {
		t.Fatalf("region setup: %v", err)
	}
	rows, err := e.super.Query(ctx, "SELECT pre, id FROM bench_ou")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var p int
		var id uuid.UUID
		if err := rows.Scan(&p, &id); err != nil {
			t.Fatal(err)
		}
		e.region.units[p] = id
	}
	rows.Close()
	// region.sql's compaction: the UPDATEs wrote a new version of every
	// large-tenant row.
	for _, s := range []string{"VACUUM FULL records", "VACUUM (ANALYZE) records", "ANALYZE org_units", "ANALYZE org_unit_closure",
		`SELECT pg_prewarm(c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public' AND c.relkind IN ('r', 'i')`,
		"CHECKPOINT"} {
		e.exec(t, s)
	}
	e.regionRefresh(t)
	for k := range e.region.defs {
		e.regionPublish(t, k)
	}
	t.Logf("REGION setup (tree, closure, sets, bench_rare, bench_conc, VACUUM FULL, refresh, ACLs): %s", time.Since(start).Round(time.Second))
}

// regionRefresh runs one TICK-1071 refresh cycle as nava_worker. The job
// lists tenants from tenant_identity_binding, which the P3-36 seed does not
// fill, so every perf tenant gets a binding first, as a live tenant has.
func (e *perfEnv) regionRefresh(t *testing.T) {
	t.Helper()
	e.exec(t, `INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id)
		SELECT tenant_id, 'https://kc.perf.test/realms/nava', 'perf-org-' || tn FROM perf_tenant ON CONFLICT DO NOTHING`)
	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("nava_worker", etPassword("nava_worker"))
	u.Path = "/" + e.pool.Config().ConnConfig.Database
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	r, err := estimates.New(estimates.PoolStore{Pool: pool}, noop.NewMeterProvider(), slog.New(slog.DiscardHandler), estimates.Options{Interval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if failed := r.Cycle(t.Context()); failed > 0 {
		t.Fatalf("estimates refresh: %d tenants failed", failed)
	}
	for _, k := range []string{"ticket", "bench_rare", "bench_conc"} {
		var ne int64
		if err := e.super.QueryRow(t.Context(), "SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definition_id = $2",
			e.large.id, e.region.defs[k]).Scan(&ne); err != nil {
			t.Fatalf("estimate %s: %v", k, err)
		}
		t.Logf("REGION estimate %s: N̂_e %d, K* %d", k, ne, authz.KStar(perfRegionPage, ne, true))
	}
}

// perfRegionDoc is the P3-36 document in format 2 plus the regional policy,
// last.
func perfRegionDoc(agents, admins, regional uuid.UUID) authz.Document {
	d := authz.ConvertFormat1(perfACL(agents, admins))
	d.Policies = append(d.Policies, authz.NamedPolicy{Name: "regional", Ops: []string{authz.OpRead}, When: authz.Node{All: []authz.Node{
		{MemberOf: &authz.MemberOf{Groups: []uuid.UUID{regional}}},
		{Under: &authz.Under{Caller: true}},
	}}})
	return d
}

// regionPublish publishes perfRegionDoc on type key of the large tenant
// through Prepare and PublishPrepared (perfPublishDoc, for any key).
func (e *perfEnv) regionPublish(t *testing.T, key string) {
	t.Helper()
	d := perfRegionDoc(perfAgentsGroup(0), e.large.adminGroup, e.region.regional)
	if err := tenancy.InTenantTx(t.Context(), e.pool, e.large.id, func(tx pgx.Tx) error {
		_, err := perfPublishKeyDoc(t.Context(), tx, key, d)
		return err
	}); err != nil {
		t.Fatalf("publish region ACL on %s: %v", key, err)
	}
	t.Logf("REGION ACL on %s: ReadBranchBound=%d", key, authz.ReadBranchBound(d))
}

// regionReport writes the sets table and each region row's paired numbers
// beside the raw report.
func (e *perfEnv) regionReport(t *testing.T, results []*perfResult, out string) {
	t.Helper()
	if e.region == nil {
		return
	}
	var b strings.Builder
	b.WriteString("# TICK-1037 region rows (A2-11)\n\n## Sets (as each row ran)\n\n| Set | Type | \\|S\\| | m | N̂_e | K* | Shape (A2-5) |\n| --- | --- | --- | --- | --- | --- | --- |\n")
	b.WriteString(e.region.report.String())
	b.WriteString("\n## Rows: handler p95 against 150 ms, paired base beside it (1 client)\n\n| Row | Attempt | n | p50 | **p95** | Base n | Base p50 | Base p95 | Δp95 vs base | Gate | Verdict |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range results {
		if !isRegionRow(r.sc.name) {
			continue
		}
		p95, b95 := pct(r.lat, .95), pct(r.pairLat, .95)
		v := "pass"
		switch {
		case r.gated():
			v = "void"
		case p95 > perfListTargetMS:
			v = "**FAIL**"
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %.2f | **%.2f** | %d | %.2f | %.2f | %+.2f | %s | %s |\n", mdCell(r.sc.name), r.attempt, len(r.lat), pct(r.lat, .5), p95,
			len(r.pairLat), pct(r.pairLat, .5), b95, p95-b95, r.gateReason(), v)
	}
	path := strings.TrimSuffix(out, filepath.Ext(out)) + "-region.md"
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", path)
}
