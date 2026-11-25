//go:build integration

package groupsync_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-947 (ADR-010 D7): nava_groupsync has no DELETE on org_units, so a unit
// that leaves the directory is tombstoned, keeps its first deleted_at while it
// stays gone, and comes back on the same id. A record that names it is never
// touched. A revived unit is a reattachment, so its chain depth is checked
// like a new unit's: its parent may have moved deeper while it was gone.
func TestOrgUnitTombstonedAndRevivedOnTheSameID(t *testing.T) {
	super, pool := cluster(t) // pool logs in as nava_groupsync
	ctx := context.Background()
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir) // MaxOrgUnitDepth 8
	if n := count(t, super, "SELECT count(*) FROM (SELECT 1 WHERE has_table_privilege('nava_groupsync', 'public.org_units', 'DELETE')) x"); n != 0 {
		t.Fatal("nava_groupsync holds DELETE on org_units")
	}
	sync := func(what string, units ...identity.DirectoryGroup) groupsync.Result {
		t.Helper()
		dir.set(tid, identity.DirectoryChanges{OrgUnits: units})
		r := once(t, s)
		if r.Unchanged {
			t.Fatalf("%s: the snapshot was skipped as unchanged", what)
		}
		return r
	}
	mustSync := func(what string, units ...identity.DirectoryGroup) {
		t.Helper()
		if r := sync(what, units...); r.Err != nil {
			t.Fatalf("%s: %v", what, r.Err)
		}
	}
	type unitState struct {
		id        uuid.UUID
		name      string
		deletedAt *time.Time
	}
	state := func(kc string) unitState {
		t.Helper()
		var u unitState
		if err := super.QueryRow(ctx, "SELECT id, name, deleted_at FROM org_units WHERE tenant_id = $1 AND keycloak_group_id = $2",
			tid, kc).Scan(&u.id, &u.name, &u.deletedAt); err != nil {
			t.Fatalf("read unit %s: %v", kc, err)
		}
		return u
	}
	p := unitG("p", "", path(tid, "/ou/p"), "P")
	c := unitG("c", "p", path(tid, "/ou/p/c"), "C")

	// 1. p and c, c under p; a record names c.
	mustSync("p and c", p, c)
	cID := state("c").id
	exec(t, super, `WITH r AS (INSERT INTO records (tenant_id, id, org_unit_id) VALUES ($1, $2, $3) RETURNING tenant_id, id, version)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
		SELECT tenant_id, gen_random_uuid(), 'record', 'create', 'record', id, version, 'system', 'system' FROM r`,
		tid, uuid.New(), cID)
	recState := func() string {
		t.Helper()
		var s string
		if err := super.QueryRow(ctx, "SELECT string_agg(id || ':' || version || ':' || coalesce(org_unit_id::text, 'null') || ':' || xmin::text, ',') FROM records WHERE tenant_id = $1", tid).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	rec := recState()

	// 2. c leaves: tombstoned, on the same row; p stays live; the record is untouched.
	mustSync("without c", p)
	gone := state("c")
	if gone.id != cID || gone.deletedAt == nil {
		t.Fatalf("c after it left = %+v, want id %s tombstoned", gone, cID)
	}
	if state("p").deletedAt != nil {
		t.Error("p was tombstoned with c")
	}
	if got := recState(); got != rec {
		t.Errorf("the record changed when its unit was tombstoned:\n before %s\n after  %s", rec, got)
	}

	// 3. c stays gone through another applied snapshot: its deleted_at is kept.
	mustSync("still without c", p, unitG("q", "", path(tid, "/ou/q"), "Q"))
	if again := state("c"); again.deletedAt == nil || !again.deletedAt.Equal(*gone.deletedAt) {
		t.Errorf("c's deleted_at after a second sync without it = %v, want %v unchanged", again.deletedAt, *gone.deletedAt)
	}

	// 4. While c is gone, p moves to depth 8 under a chain a1 <- ... <- a7.
	chain := []identity.DirectoryGroup{unitG("a1", "", path(tid, "/ou/a1"), "a1")}
	for i := 2; i <= 7; i++ {
		k := "a" + string(rune('0'+i))
		chain = append(chain, unitG(k, "a"+string(rune('0'+i-1)), path(tid, "/ou/"+k), k))
	}
	deepP := unitG("p", "a7", path(tid, "/ou/p"), "P")
	mustSync("p at depth 8", append(chain, deepP)...)

	// 5. c comes back under p: depth 9 > 8. Its parent is unchanged, so only
	// the revive itself puts it on the chain check; the sync is refused and
	// c stays tombstoned.
	if r := sync("c back at depth 9", append(chain, deepP, unitG("c", "p", path(tid, "/ou/p/c"), "C2"))...); !errors.Is(r.Err, groupsync.ErrOrgUnitDepth) {
		t.Fatalf("revive c at depth 9: err = %v, want ErrOrgUnitDepth", r.Err)
	}
	if still := state("c"); still.deletedAt == nil || still.name != "C" {
		t.Errorf("c after the refused revive = %+v, want tombstoned and named C", still)
	}

	// 6. p back at the root, c back under it, renamed: revived on the same id.
	mustSync("c back, renamed", p, unitG("c", "p", path(tid, "/ou/p/c"), "C2"))
	back := state("c")
	if back.id != cID || back.deletedAt != nil || back.name != "C2" {
		t.Errorf("c after it came back = %+v, want id %s, deleted_at NULL, name C2", back, cID)
	}
	if n := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1 AND keycloak_group_id = 'c'", tid); n != 1 {
		t.Errorf("c has %d rows, want 1", n)
	}
	if got := recState(); got != rec {
		t.Errorf("the record changed across the revive:\n before %s\n after  %s", rec, got)
	}
}
