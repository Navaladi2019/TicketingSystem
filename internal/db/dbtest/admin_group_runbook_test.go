//go:build integration

package dbtest

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// TICK-372: the SQL of docs/operator/onboarding.md, "Retry fails with admin
// group missing", runs against the real schema and repairs the tenant. The
// test reads the SQL out of the runbook, so the document cannot drift from it.

var runbookSQL = regexp.MustCompile("(?s)```sql\n(.*?)```")

// runbookBlocks returns the sql blocks of the section, with the psql variables
// replaced by literals (a psql client would do the same).
func runbookBlocks(t *testing.T, tenant, kcGroup string) []string {
	t.Helper()
	doc, err := os.ReadFile("../../../docs/operator/onboarding.md")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ReplaceAll(string(doc), "\r\n", "\n")
	start := strings.Index(s, "## Retry fails with admin group missing")
	end := strings.Index(s, "## TODO: abandoning a provisioning tenant")
	if start < 0 || end < start {
		t.Fatal("runbook section not found")
	}
	var out []string
	for _, m := range runbookSQL.FindAllStringSubmatch(s[start:end], -1) {
		b := strings.ReplaceAll(m[1], ":'tenant_id'", "'"+tenant+"'")
		out = append(out, strings.ReplaceAll(b, ":'kc_group_id'", "'"+kcGroup+"'"))
	}
	if len(out) != 4 {
		t.Fatalf("runbook has %d sql blocks, want 4 (confirm, option A, option B, check)", len(out))
	}
	return out
}

// runBlock runs each statement of the block, logging the rows or the command tag.
func runBlock(t *testing.T, conn *pgx.Conn, label, block string) (rows []int, tags []string) {
	t.Helper()
	for _, stmt := range strings.Split(stripComments(block), ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		r, err := conn.Query(context.Background(), stmt)
		if err != nil {
			t.Fatalf("%s: %v\n%s", label, err, stmt)
		}
		n := 0
		for r.Next() {
			vals, _ := r.Values()
			t.Logf("%s row: %v", label, vals)
			n++
		}
		r.Close()
		if err := r.Err(); err != nil {
			t.Fatalf("%s: %v\n%s", label, err, stmt)
		}
		tag := r.CommandTag().String()
		t.Logf("%s: %s -> %d rows, tag %q", label, firstLine(stmt), n, tag)
		rows, tags = append(rows, n), append(tags, tag)
	}
	return rows, tags
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
	seedTenants(t, super, nil, []string{tid.String()})
	exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
		VALUES ($1::uuid, $2, 'kc-old-admins', '/t/' || $1::text || '/admins')`, tid.String(), onboard.AdminGroupID(tid))
	err := onboardStore(t, 4).RecordAdminGroup(context.Background(), tid, "kc-new-admins")
	if !errors.Is(err, onboard.ErrAdminGroupMissing) {
		t.Fatalf("setup: RecordAdminGroup = %v, want ErrAdminGroupMissing", err)
	}
}

func TestTick372RunbookRepairsStaleAdminGroupRow(t *testing.T) {
	for _, tc := range []struct {
		name  string
		block int
		tag   string
	}{{"option A update", 1, "UPDATE 1"}, {"option B delete", 2, "DELETE 1"}} {
		t.Run(tc.name, func(t *testing.T) {
			migratedPlatform(t)
			super := superNava(t)
			tid := uuid.MustParse(tenantA)
			staleAdminGroup(t, super, tid)
			blocks := runbookBlocks(t, tenantA, "kc-new-admins")

			rows, _ := runBlock(t, super, "confirm", blocks[0])
			if len(rows) != 3 || rows[0] != 1 || rows[1] != 0 || rows[2] != 0 {
				t.Fatalf("confirm rows = %v, want [1 0 0]", rows)
			}
			_, tags := runBlock(t, super, "repair", blocks[tc.block])
			if !slicesContain(tags, tc.tag) {
				t.Fatalf("repair tags = %v, want %s", tags, tc.tag)
			}

			store := onboardStore(t, 4)
			if err := store.RecordAdminGroup(context.Background(), tid, "kc-new-admins"); err != nil {
				t.Fatalf("retry RecordAdminGroup: %v", err)
			}
			if err := store.Activate(context.Background(), tid); err != nil {
				t.Fatalf("retry Activate: %v", err)
			}
			rows, _ = runBlock(t, super, "check", blocks[3])
			if len(rows) != 1 || rows[0] != 1 {
				t.Fatalf("check rows = %v, want [1]", rows)
			}
			if got := queryString(t, super, "SELECT state FROM tenants WHERE id = $1", tenantA); got != "active" {
				t.Errorf("tenant state = %s, want active", got)
			}
			if got := queryString(t, super, "SELECT count(*)::text FROM groups WHERE tenant_id = $1", tenantA); got != "1" {
				t.Errorf("groups = %s, want 1", got)
			}
		})
	}
}

func slicesContain(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
