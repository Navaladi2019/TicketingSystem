//go:build integration

package dbtest

import (
	"context"
	"errors"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// TICK-372: the SQL of docs/operator/onboarding.md, "Retry fails with admin
// group missing", runs against the real schema, as the role the runbook names
// (the bootstrap superuser, pgadmin), and repairs the tenant. The test reads
// the SQL out of the runbook, so the document cannot drift from it. TICK-530
// (ADR-004-E5 N3): the same section repairs the auditors group, and the
// section "Pointer missing after a group came back (R-E5-4)" re-records either
// pointer (admin_group_rerecord_test.go).

var runbookSQL = regexp.MustCompile("(?s)```sql\n(.*?)```")

const (
	blkConfirm, blkA, blkB, blkCommit, blkRollback, blkCheck = 0, 1, 2, 3, 4, 5
	runbookBlockCount                                        = 6
)

// pointerGroup is one of onboarding's two recorded groups: the runbook's
// group_name and pointer_table psql variables, and the group's platform id.
type pointerGroup struct {
	name, table string
	id          func(uuid.UUID) uuid.UUID
}

var (
	adminsGroup   = pointerGroup{"admins", "tenant_admin_group", onboard.AdminGroupID}
	auditorsGroup = pointerGroup{"auditors", "tenant_auditor_group", onboard.AuditorGroupID}
	pointerGroups = []pointerGroup{adminsGroup, auditorsGroup}
)

// sectionBlocks returns the sql blocks of the runbook section that starts at
// heading and ends at the heading end ("" = the end of the file), with the
// psql variables replaced by literals, as a psql client would replace them
// (:"pointer_table" is an identifier).
func sectionBlocks(t *testing.T, heading, end, tenant, kcGroup string, g pointerGroup) []string {
	t.Helper()
	doc, err := os.ReadFile("../../../docs/operator/onboarding.md")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ReplaceAll(string(doc), "\r\n", "\n")
	start := strings.Index(s, heading)
	stop := len(s)
	if end != "" {
		stop = strings.Index(s, end)
	}
	if start < 0 || stop < start {
		t.Fatalf("runbook section %q not found", heading)
	}
	r := strings.NewReplacer(":'tenant_id'", "'"+tenant+"'", ":'kc_group_id'", "'"+kcGroup+"'",
		":'group_name'", "'"+g.name+"'", `:"pointer_table"`, `"`+g.table+`"`)
	var out []string
	for _, m := range runbookSQL.FindAllStringSubmatch(s[start:stop], -1) {
		out = append(out, r.Replace(m[1]))
	}
	return out
}

// runbookBlocks returns the sql blocks of "Retry fails with admin group
// missing" for group g.
func runbookBlocks(t *testing.T, tenant, kcGroup string, g pointerGroup) []string {
	t.Helper()
	out := sectionBlocks(t, "## Retry fails with admin group missing", "## TODO: abandoning a provisioning tenant", tenant, kcGroup, g)
	if len(out) != runbookBlockCount {
		t.Fatalf("runbook has %d sql blocks, want %d (confirm, A, B, commit, rollback, check)", len(out), runbookBlockCount)
	}
	return out
}

// result is what a block's statements returned: the row values of each
// statement and its command tag.
type result struct {
	rows [][][]any
	tags []string
}

func (r result) counts() []int {
	var n []int
	for _, rs := range r.rows {
		n = append(n, len(rs))
	}
	return n
}

// runBlock runs each statement of the block, logging the rows or the command tag.
func runBlock(t *testing.T, conn *pgx.Conn, label, block string) result {
	t.Helper()
	var res result
	for _, stmt := range strings.Split(stripComments(block), ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		r, err := conn.Query(context.Background(), stmt)
		if err != nil {
			t.Fatalf("%s: %v\n%s", label, err, stmt)
		}
		var rows [][]any
		for r.Next() {
			vals, _ := r.Values()
			t.Logf("%s row: %v", label, vals)
			rows = append(rows, vals)
		}
		r.Close()
		if err := r.Err(); err != nil {
			t.Fatalf("%s: %v\n%s", label, err, stmt)
		}
		tag := r.CommandTag().String()
		t.Logf("%s: %s -> %d rows, tag %q", label, firstLine(stmt), len(rows), tag)
		res.rows, res.tags = append(res.rows, rows), append(res.tags, tag)
	}
	return res
}

func stripComments(s string) string {
	var b []string
	for _, l := range strings.Split(s, "\n") {
		if i := strings.Index(l, "--"); i >= 0 {
			l = l[:i]
		}
		b = append(b, l)
	}
	return strings.Join(b, "\n")
}

func firstLine(s string) string { return strings.Fields(strings.TrimSpace(s))[0] }

func staleAdminGroup(t *testing.T, super *pgx.Conn, tid uuid.UUID) {
	t.Helper()
	staleGroup(t, super, tid, adminsGroup)
}

// staleGroup sets up the section's failure for g in a provisioning tenant: a
// groups row at g's platform id and path under an old Keycloak id
// (kc-old-<name>), so recording the current id (kc-new-<name>) fails with
// ErrAdminGroupMissing. For the auditors group the admins group is recorded
// first, as onboarding does.
func staleGroup(t *testing.T, super *pgx.Conn, tid uuid.UUID, g pointerGroup) {
	t.Helper()
	seedTenants(t, super, nil, []string{tid.String()})
	if g.name == auditorsGroup.name {
		if err := recordGroup(t, adminsGroup, tid, "kc-admins"); err != nil {
			t.Fatalf("setup: RecordAdminGroup = %v", err)
		}
	}
	exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
		VALUES ($1::uuid, $2, $3, '/t/' || $1::text || '/' || $4::text)`, tid.String(), g.id(tid), "kc-old-"+g.name, g.name)
	if err := recordGroup(t, g, tid, "kc-new-"+g.name); !errors.Is(err, onboard.ErrAdminGroupMissing) {
		t.Fatalf("setup: record %s = %v, want ErrAdminGroupMissing", g.name, err)
	}
}

// recordGroup is onboarding's record step for g.
func recordGroup(t *testing.T, g pointerGroup, tid uuid.UUID, kc string) error {
	t.Helper()
	store := onboardStore(t, 4)
	if g.name == auditorsGroup.name {
		return store.RecordAuditorGroup(context.Background(), tid, kc)
	}
	return store.RecordAdminGroup(context.Background(), tid, kc)
}

func TestTick372RunbookRepairsStaleAdminGroupRow(t *testing.T) {
	for _, g := range pointerGroups {
		for _, tc := range []struct {
			name  string
			block int
			tag   string
		}{{"option A update", blkA, "UPDATE 1"}, {"option B delete", blkB, "DELETE 1"}} {
			t.Run(g.name+" "+tc.name, func(t *testing.T) {
				migratedPlatform(t)
				super := superNava(t)
				tid := uuid.MustParse(tenantA)
				staleGroup(t, super, tid, g)
				kc := "kc-new-" + g.name
				blocks := runbookBlocks(t, tenantA, kc, g)

				if got := runBlock(t, super, "confirm", blocks[blkConfirm]).counts(); !reflect.DeepEqual(got, []int{1, 0, 0}) {
					t.Fatalf("confirm rows = %v, want [1 0 0]", got)
				}
				rep := runBlock(t, super, "repair", blocks[tc.block])
				if !reflect.DeepEqual(rep.tags, []string{"BEGIN", tc.tag}) {
					t.Fatalf("repair tags = %v, want [BEGIN %s]", rep.tags, tc.tag)
				}
				// The tag is exactly 1, so the operator commits.
				if got := runBlock(t, super, "commit", blocks[blkCommit]).tags; !reflect.DeepEqual(got, []string{"COMMIT"}) {
					t.Fatalf("commit tags = %v", got)
				}

				if err := recordGroup(t, g, tid, kc); err != nil {
					t.Fatalf("retry record %s: %v", g.name, err)
				}
				if err := onboardStore(t, 4).Activate(context.Background(), tid); err != nil {
					t.Fatalf("retry Activate: %v", err)
				}
				chk := runBlock(t, super, "check", blocks[blkCheck])
				if len(chk.rows) != 1 || len(chk.rows[0]) != 1 {
					t.Fatalf("check rows = %v, want one row", chk.rows)
				}
				row := chk.rows[0][0]
				if row[0] != "active" || row[2] != kc {
					t.Errorf("check row = %v, want state active and keycloak_group_id %s", row, kc)
				}
				if got := queryString(t, super, "SELECT count(*)::text FROM groups WHERE tenant_id = $1 AND path = '/t/' || $1::text || '/' || $2::text",
					tenantA, g.name); got != "1" {
					t.Errorf("%s groups = %s, want 1", g.name, got)
				}
			})
		}
	}
}

// ADR-004-E5 D1: group sync may already have tombstoned the stale row when
// Keycloak dropped the old group. Option A clears the tombstone with the new
// id, so the retry finds a live row and the pointer names it.
func TestTick372RunbookOptionARevivesTombstonedRow(t *testing.T) {
	for _, g := range pointerGroups {
		t.Run(g.name, func(t *testing.T) {
			migratedPlatform(t)
			super := superNava(t)
			tid := uuid.MustParse(tenantA)
			staleGroup(t, super, tid, g)
			exec(t, super, "UPDATE groups SET deleted_at = now() WHERE tenant_id = $1 AND id = $2", tenantA, g.id(tid))
			kc := "kc-new-" + g.name
			blocks := runbookBlocks(t, tenantA, kc, g)

			if got := runBlock(t, super, "confirm", blocks[blkConfirm]).counts(); !reflect.DeepEqual(got, []int{1, 0, 0}) {
				t.Fatalf("confirm rows = %v, want [1 0 0]", got)
			}
			if got := runBlock(t, super, "repair", blocks[blkA]).tags; !reflect.DeepEqual(got, []string{"BEGIN", "UPDATE 1"}) {
				t.Fatalf("repair tags = %v, want [BEGIN UPDATE 1]", got)
			}
			runBlock(t, super, "commit", blocks[blkCommit])
			if err := recordGroup(t, g, tid, kc); err != nil {
				t.Fatalf("retry record %s: %v", g.name, err)
			}
			if got := queryString(t, super, `SELECT (group_id = $2)::text FROM `+g.table+` WHERE tenant_id = $1`, tenantA, g.id(tid)); got != "true" {
				t.Errorf("%s names the revived row: %s, want true", g.table, got)
			}
		})
	}
}

// path is not unique: when option B would delete two rows, the tag is not 1,
// and the runbook's ROLLBACK block leaves both rows in place.
func TestTick372RunbookRollbackWhenTagIsNotOne(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	tid := uuid.MustParse(tenantA)
	staleAdminGroup(t, super, tid)
	exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
		VALUES ($1::uuid, gen_random_uuid(), 'kc-other', '/t/' || $1::text || '/admins')`, tenantA)
	blocks := runbookBlocks(t, tenantA, "kc-new-admins", adminsGroup)

	if got := runBlock(t, super, "confirm", blocks[blkConfirm]).counts(); got[0] != 2 {
		t.Fatalf("confirm rows = %v, want 2 stale rows first", got)
	}
	if got := runBlock(t, super, "repair", blocks[blkB]).tags; !reflect.DeepEqual(got, []string{"BEGIN", "DELETE 2"}) {
		t.Fatalf("repair tags = %v, want [BEGIN DELETE 2]", got)
	}
	if got := runBlock(t, super, "rollback", blocks[blkRollback]).tags; !reflect.DeepEqual(got, []string{"ROLLBACK"}) {
		t.Fatalf("rollback tags = %v", got)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM groups WHERE tenant_id = $1", tenantA); got != "2" {
		t.Errorf("groups = %s, want 2 (rolled back)", got)
	}
}

// The runbook says nava_migrator, the owner, cannot run it: it is NOBYPASSRLS
// and the tables are FORCE RLS, so it sees nothing. Pinned so the claim holds.
func TestTick372RunbookSeesNothingAsMigrator(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	staleAdminGroup(t, super, uuid.MustParse(tenantA))
	blocks := runbookBlocks(t, tenantA, "kc-new-admins", adminsGroup)

	owner := asRole(t, "nava_migrator")
	if got := runBlock(t, owner, "confirm as nava_migrator", blocks[blkConfirm]).counts(); !reflect.DeepEqual(got, []int{0, 0, 0}) {
		t.Errorf("confirm rows as nava_migrator = %v, want [0 0 0]", got)
	}
	if got := runBlock(t, owner, "repair as nava_migrator", blocks[blkA]).tags; !reflect.DeepEqual(got, []string{"BEGIN", "UPDATE 0"}) {
		t.Errorf("repair tags as nava_migrator = %v, want [BEGIN UPDATE 0]", got)
	}
	runBlock(t, owner, "rollback", blocks[blkRollback])
}
