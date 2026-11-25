# Technical Architect critique of ADR-010-E4, round 1 (TICK-1118 comment 01a210f4, 2026-11-25, on PR #870 at `aee7664d`)

**VERDICT: CHANGES.** Keep what was built: that decision is right, and the delta equality holds. Three statements in E4 are wrong against the code (B1 to B3), and all three are text fixes. I read PR #870 at `aee7664d` and ran the shipped code at `57573b39` with the real migrations on PostgreSQL 16.10. On #870, `lint`, `changes`, `runner` and `story-branches` pass; the other jobs were skipped.

## Blocking

**B1. E4-2 (c): the lock and the `23505` inference are wrong.**
- **Known:** the function runs `CREATE POLICY` on `tenants`, `org_units` and `org_unit_closure` before `LOCK TABLE … SHARE`. While uncommitted it holds `AccessExclusiveLock` on all three, so the SHARE lock adds nothing.

| Race | Result |
|---|---|
| Drift repair uncommitted, then the function | The function waits on `org_units`; both succeed; closure right |
| The function uncommitted, then a drift repair | The sync waits at its first `org_units` read; both succeed; closure right |
| Sync has read `org_units`, the function starts, sync inserts a unit | `40P01` deadlock (FK check on `tenants`); the function was the victim |
| The same, but the sync had already inserted a principal | No deadlock; the function waits on `tenants` |
| `nava_app` reads any of the three while it runs (`lock_timeout` 500 ms) | `55P03` on each |
| `nava_app` transaction has done a plain read of the closure | The function waits (`55P03` at 1 s) |

The closure was also right at the end of the two sync-insert races.

- **Failing scenario:** an operator follows "an operator re-runs the function" on a live stack. Requests of **every** tenant that touch these tables then wait until it commits.
- **Replacement for (c), after "as `nava_migrator`":** "It creates temporary policies, so it holds ACCESS EXCLUSIVE on `tenants`, `org_units` and `org_unit_closure` until it commits. It waits for every transaction that has read one of them, and every later reader of any tenant waits for it. It cannot interleave with a delta or a drift repair. Racing a sync that writes units can end in `40P01`; one side rolls back and the closure stays right. Run it in a maintenance window with `lock_timeout` set."
- **Also:** delete the "Inferred, not tested" sentence under Evidence, and fix "Neither lock covers … its table lock does".

**B2. AR-TICK-1118-01: "a missing row hides tickets" is half the risk.**
- **Known:** the policy grammar has `Not` (`internal/authz/document.go:166`), and a unit with no row reads as outside (`under.go:36-39`). `docs/data/org-unit-closure.md` says "`not under` admits it".
- **Failing scenario:** the policy is `not under(HR)` and the row (HR, HR-Payroll) is missing. The caller sees HR-Payroll tickets. An extra row hides them.
- **Replacement:** "An extra row grants through `under` and hides through `not under`; a missing row does the reverse. Either can show a user tickets outside their rules, inside one tenant." Use it in E4, the register and the explainer's Risks.
- **Severity:** Low still looks right to me, because the writers and the tenant wall are unchanged. Security must rate the corrected text: the old one understates it.

**B3. "Every 10 minutes" is wrong.** The interval is 5 minutes, and a longer one is a configuration error: `internal/platform/config/groupsync.go:37` (`MaxGroupsyncInterval`), `internal/groupsync/groupsync.go` (`DefaultInterval`), `compose.yaml:602`. Ten minutes is `LagWarnAfter`. Fix the Codes list, the third Rejected row and the explainer (Today, Risks, options table). That Rejected row's cost doubles; the rejection still stands.

## Non-blocking

- **N1. E4-6 covers single moves only.** An import into a non-empty closure takes the delta: 4.2 to 8.8 s at 10,111 units, against 2.7 s for a one-shot import (shared laptop). So "70 to 468 ms, for the same rows" and "the sync is shorter" need "after a move". E4-2's plan protection does not cover the delta or the unrooted count. After `ANALYZE` on a one-unit platform, the import hits the 2-minute timeout. A forced full rebuild times out there too, so this is **not** a reason to go back to A2-2.
- **N2. "I5 holds" needs a condition:** while the closure equals A2-1's rows. **Inferred from code:** `canonicalU` (`internal/authz/acl.go:697`) drops a unit from the count and probe, but W0, C3 and `Allows` use the uncut U. If a caller's U holds both a and u, an extra row (a, u) can make the count omit tickets the page shows.
- **N3. "Every cycle" in E4-7** means every cycle that gets the lock; a skipped tenant is not counted.
- **N4. Writers in the AR row:** add "or a holder of group sync's database credentials".

## Confirmed sound

| Item | How |
|---|---|
| E4-1, E4-4 | By argument: a unit's rows change only if its old or new path holds a touched unit. An old path is in the stored descendants; a new live path is reached by the walk. `applyOrgUnits` lists every add, parent change, revive and tombstone as touched. Your spike agrees, and its index matches the real `org_units_parent`. After every import sync of mine that finished, the closure equalled the snapshot. |
| E4-3 | `lockSQL` runs before `ChangesSince`; nothing else in group sync writes the closure |
| E4-5 | `db/audited-tables.yaml:161-166` |
| E4-6 figures | Match TICK-1054 comments 01a1ffab and 01a1ffb6 |
| E4-7 | The cursor is in memory; unchanged cycles only run `unrootedUnits`; a rename is written but not touched, so it triggers the drift check |
| E4-8 | Policy `org_unit_closure_select` in `20261121050000` names `nava_groupsync` |

**Answers:** (1) Low is right after B2, subject to Security. (2) The "never a wrong closure" half held in every race I ran; the `23505` half is refuted.

## Follow-ups filed (backlog, for the Technical Lead to schedule)
- [TICK-1122](mention://issue/01a210f2-40b7-7370-bb8a-6b89c284af0c): the function's locks. The migration comment and `docs/data/org-unit-closure.md` also say reads go on.
- [TICK-1123](mention://issue/01a210f2-4148-71e8-95f3-755f9b3e4bda): the import timing.

Neither gates E4. I support the drift-repair counter.

## Spike

- **H1 (pass/fail from E4): the function racing a drift repair ends in `23505`. Refuted:** it did not in either order.
- **M1 (a measurement with no threshold set beforehand): import into a non-empty closure**, whole sync, 10,111 units:

| State before the import | Shipped delta | Forced full rebuild | Delta, nested loops off |
|---|---|---|---|
| Empty (one-shot, shipped path is the full rebuild) | 2.7 s | – | – |
| 1 unit, never analysed | 6.5 / 7.8 s | – | – |
| 111 units, never analysed | 5.0 / 7.6 s | 3.4 s | 5.9 s |
| 11 / 60 / 111 units, analysed then | 5.3 / 4.2 / 5.3 s | – | – |
| 1 / 111 units, 400 other tenants analysed | 4.9 / 8.8 s | – | – |
| 1 unit, analysed then | 2 min timeout, in the delta write | 2 min timeout, in the unrooted count | 2 min timeout, in the unrooted count |

- **Limits:** one run per cell unless two figures are given for a never-analysed row; the host was running another stack, so read ratios, not absolutes.
- **Setup:** `postgres:16.10` throwaway container (removed), Go 1.27.4, Windows 11 host, the real `Syncer` as `nava_groupsync`, `autovacuum_enabled=false` on both tables in M1. The two forced columns used an environment switch I added at the top of the delta branch in `closure.go`: call `fullRebuildClosure`, or `SET LOCAL enable_nestloop = off` until the function returns.
- **Rerun:** save the code as `internal/groupsync/zz_spike_integration_test.go` at `57573b39`, set `NAVA_TEST_PG_URL` to a superuser URL of a throwaway PostgreSQL 16.10, then `go test -tags integration ./internal/groupsync -run TestSpike -v -timeout 20m`.

<details><summary>Spike code (throwaway)</summary>

```go
//go:build integration

package groupsync_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

func spUser(e string) []identity.DirectoryUser {
	return []identity.DirectoryUser{{KeycloakUserID: "u", Email: e, Enabled: true}}
}

func TestSpikeDeltaNoStats(t *testing.T) {
	for _, c := range []struct {
		name    string
		first   int
		analyze bool
		bg      bool // 400 other analysed tenants
	}{
		{"control one-shot import (full rebuild)", 0, false, false},
		{"import after 1 unit never analysed", 1, false, false},
		{"import after 111 units never analysed", 111, false, false},
		{"import after 1 unit analysed while small", 1, true, false},
		{"import after 11 units analysed while small", 11, true, false},
		{"import after 60 units analysed while small", 60, true, false},
		{"import after 111 units analysed while small", 111, true, false},
		{"import after 1 unit 400 other tenants analysed", 1, true, true},
		{"import after 111 units 400 other tenants analysed", 111, true, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			super, pool := cluster(t)
			exec(t, super, "ALTER TABLE org_units SET (autovacuum_enabled=false)")
			exec(t, super, "ALTER TABLE org_unit_closure SET (autovacuum_enabled=false)")
			if c.bg {
				exec(t, super, `INSERT INTO tenants (id,state) SELECT gen_random_uuid(),'active' FROM generate_series(1,400)`)
				exec(t, super, `INSERT INTO org_units (tenant_id,id,keycloak_group_id,name) SELECT t.id,gen_random_uuid(),'b'||i,'b' FROM tenants t, generate_series(1,30) i`)
				exec(t, super, `INSERT INTO org_unit_closure SELECT tenant_id,id,id FROM org_units`)
			}
			tid := uuid.New()
			seedTenant(t, super, tid, 1)
			dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
			s := newSyncer(t, pool, dir)
			units := benchUnits(tid)
			if c.first > 0 {
				dir.set(tid, identity.DirectoryChanges{OrgUnits: units[:c.first], Users: spUser("a@a.test")})
				if r := once(t, s); r.Err != nil {
					t.Fatal(r.Err)
				}
			}
			if c.analyze {
				exec(t, super, "ANALYZE org_units")
				exec(t, super, "ANALYZE org_unit_closure")
			}
			dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: spUser("b@a.test")})
			start := time.Now()
			r := once(t, s)
			t.Logf("RESULT %s: import sync %v err=%v rows=%d", c.name, time.Since(start).Round(time.Millisecond), r.Err,
				count(t, super, "SELECT count(*) FROM org_unit_closure WHERE tenant_id=$1", tid))
			if r.Err != nil {
				return
			}
			sameRows(t, c.name, closureOf(t, super, tid), wantClosure(units))
			for i, mv := range []struct {
				what string
				idx  int
				p    string
			}{{"leaf", 111, "u12"}, {"country", 11, "u2"}, {"region", 1, "u2"}} {
				units[mv.idx] = unit(units[mv.idx].KeycloakGroupID, mv.p, units[mv.idx].Path)
				dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: spUser(fmt.Sprintf("m%d@a.test", i))})
				start = time.Now()
				r := once(t, s)
				t.Logf("RESULT   then move %s: %v err=%v", mv.what, time.Since(start).Round(time.Millisecond), r.Err)
			}
			dir.set(tid, identity.DirectoryChanges{OrgUnits: units[:1], Users: spUser("z@a.test")})
			start = time.Now()
			r = once(t, s)
			t.Logf("RESULT   then tombstone 10,110 units: %v err=%v", time.Since(start).Round(time.Millisecond), r.Err)
		})
	}
}

func spConn(t *testing.T, super *pgx.Conn, role string) *pgx.Conn {
	t.Helper()
	c, err := pgx.ConnectConfig(context.Background(), super.Config())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close(context.Background()) })
	if _, err := c.Exec(context.Background(), "SET ROLE "+role); err != nil {
		t.Fatal(err)
	}
	return c
}

func spLocks(t *testing.T, super *pgx.Conn, pid uint32) string {
	rows, _ := super.Query(context.Background(), `SELECT relation::regclass::text||' '||mode||CASE WHEN granted THEN '' ELSE ' WAITING' END FROM pg_locks WHERE pid=$1 AND relation::regclass::text IN ('tenants','org_units','org_unit_closure') ORDER BY 1`, pid)
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(out, "; ")
}

func TestSpikeRebuildAllRace(t *testing.T) {
	ctx := context.Background()
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	p := func(r string) string { return path(tid, "/ou"+r) }
	base := []identity.DirectoryGroup{unit("r", "", p("/r")), unit("a", "r", p("/r/a")), unit("b", "a", p("/r/a/b")), unit("x", "", p("/x"))}
	dir.set(tid, identity.DirectoryChanges{OrgUnits: base})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	mig, app, gs := spConn(t, super, "nava_migrator"), spConn(t, super, "nava_app"), spConn(t, super, "nava_groupsync")
	tenant := func(c *pgx.Conn) pgx.Tx {
		tx, err := c.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id',$1,true)", tid.String()); err != nil {
			t.Fatal(err)
		}
		return tx
	}
	var iso string
	_ = gs.QueryRow(ctx, "SHOW default_transaction_isolation").Scan(&iso)
	t.Logf("RESULT 0 isolation %s", iso)

	// 1. rebuild_all uncommitted: what does it lock, can a reader read?
	m, _ := mig.Begin(ctx)
	if _, err := m.Exec(ctx, "SELECT org_unit_closure_rebuild_all()"); err != nil {
		t.Fatal(err)
	}
	t.Logf("RESULT 1 rebuild_all (uncommitted) holds: %s", spLocks(t, super, mig.PgConn().PID()))
	for _, q := range []string{"SELECT count(*) FROM org_unit_closure", "SELECT count(*) FROM org_units", "SELECT count(*) FROM tenants"} {
		a := tenant(app)
		_, _ = a.Exec(ctx, "SET LOCAL lock_timeout='500ms'")
		_, err := a.Exec(ctx, q)
		t.Logf("RESULT 1 nava_app %q while it runs: err=%v", q, err)
		_ = a.Rollback(ctx)
	}
	_ = m.Rollback(ctx)

	// 2. a plain reader holds rebuild_all up.
	a := tenant(app)
	_, err := a.Exec(ctx, "SELECT count(*) FROM org_unit_closure")
	if err != nil {
		t.Fatal(err)
	}
	m, _ = mig.Begin(ctx)
	_, _ = m.Exec(ctx, "SET LOCAL lock_timeout='1s'")
	_, err = m.Exec(ctx, "SELECT org_unit_closure_rebuild_all()")
	t.Logf("RESULT 2 rebuild_all while nava_app holds a plain read of the closure: err=%v", err)
	_ = m.Rollback(ctx)
	_ = a.Rollback(ctx)

	// 3. drift repair in flight (uncommitted), then rebuild_all.
	exec(t, super, "DELETE FROM org_unit_closure WHERE ancestor_id<>descendant_id")
	g := tenant(gs)
	if err := groupsync.ApplyForTest(ctx, g, tid, identity.DirectoryChanges{OrgUnits: base}, time.Now(), 8); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := mig.Exec(ctx, "SELECT org_unit_closure_rebuild_all()"); done <- err }()
	time.Sleep(1500 * time.Millisecond)
	t.Logf("RESULT 3 drift repair uncommitted; rebuild_all: %s", spLocks(t, super, mig.PgConn().PID()))
	if err := g.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Logf("RESULT 3 repair committed first; rebuild_all err=%v", <-done)
	sameRows(t, "3", closureOf(t, super, tid), wantClosure(base))

	// 4. other order: rebuild_all in flight, then a drift repair.
	exec(t, super, "DELETE FROM org_unit_closure WHERE ancestor_id<>descendant_id")
	m, _ = mig.Begin(ctx)
	if _, err := m.Exec(ctx, "SELECT org_unit_closure_rebuild_all()"); err != nil {
		t.Fatal(err)
	}
	g = tenant(gs)
	go func() {
		done <- groupsync.ApplyForTest(ctx, g, tid, identity.DirectoryChanges{OrgUnits: base}, time.Now(), 8)
	}()
	time.Sleep(1500 * time.Millisecond)
	t.Logf("RESULT 4 rebuild_all uncommitted; sync: %s", spLocks(t, super, gs.PgConn().PID()))
	_ = m.Commit(ctx)
	err = <-done
	t.Logf("RESULT 4 rebuild_all committed; drift repair err=%v commit=%v", err, g.Commit(ctx))
	sameRows(t, "4", closureOf(t, super, tid), wantClosure(base))

	// 5. sync has read org_units, rebuild_all starts, sync then inserts a unit
	// (FK check on tenants): the statements applyOrgUnits runs, in its order.
	for i, holdTenants := range []bool{false, true} {
		g = tenant(gs)
		if holdTenants { // an earlier principal insert took the tenants FK lock
			_, err = g.Exec(ctx, `INSERT INTO principals (tenant_id,id,keycloak_user_id,enabled,federated_links,synced_at) VALUES ($1,gen_random_uuid(),'pp',true,'[]',now())`, tid)
			if err != nil {
				t.Fatal(err)
			}
		}
		if _, err := g.Exec(ctx, "SELECT id FROM org_units WHERE tenant_id=$1", tid); err != nil {
			t.Fatal(err)
		}
		go func() { _, err := mig.Exec(ctx, "SELECT org_unit_closure_rebuild_all()"); done <- err }()
		time.Sleep(300 * time.Millisecond)
		t.Logf("RESULT 5.%d rebuild_all: %s", i, spLocks(t, super, mig.PgConn().PID()))
		_, err = g.Exec(ctx, "INSERT INTO org_units (tenant_id,id,keycloak_group_id,name) VALUES ($1,gen_random_uuid(),$2,'n')", tid, fmt.Sprintf("n%d", i))
		t.Logf("RESULT 5.%d sync (tenants FK lock held before: %v) insert err=%v", i, holdTenants, err)
		_ = g.Rollback(ctx)
		t.Logf("RESULT 5.%d rebuild_all err=%v", i, <-done)
	}
	sameRows(t, "5", closureOf(t, super, tid), wantClosure(base))
}
```

</details>

VERDICT: CHANGES

RESULT: BLOCKED: B1 to B3 need the one revision, then I re-check only those — Solution Architect
