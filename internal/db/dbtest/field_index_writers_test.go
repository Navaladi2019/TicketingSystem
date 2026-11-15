//go:build integration

package dbtest

import (
	"strings"
	"testing"
)

// Security review of P7-01 (TICK-741, PR #692): P7-02's trigger on records is
// SECURITY INVOKER and runs as whoever writes the record. Under FORCE RLS a
// writer with no registry SELECT policy sees no record_index_fields row, so
// the trigger silently indexes nothing and never purges: a later filter
// misses the record, or an erasure that rewrites records.data leaves the
// erased value behind in rfv_text. This guard derives the writer set from
// the catalog instead of a hand list, so a new role granted INSERT, UPDATE or
// DELETE on records (Epic 30's erasure job, say) fails here until the
// field-index policies and grants cover it.
//
// Excluded: the owner (nava_migrator), superusers and the predefined pg_*
// roles (pg_write_all_data has implied privileges and no platform role
// is a member of it), who never run the
// trigger through a tenant path; a data migration that writes records is a
// separate, reviewed path.
func TestP701EveryRecordWriterReadsTheRegistry(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)

	writers := strings.Fields(queryString(t, super, `
		SELECT coalesce(string_agg(r.rolname, ' ' ORDER BY r.rolname), '')
		FROM pg_roles r
		WHERE NOT r.rolsuper AND left(r.rolname, 3) <> 'pg_'
		  AND r.oid <> (SELECT relowner FROM pg_class WHERE oid = 'records'::regclass)
		  AND (has_table_privilege(r.oid, 'records', 'INSERT')
		    OR has_table_privilege(r.oid, 'records', 'DELETE')
		    OR has_any_column_privilege(r.oid, 'records', 'UPDATE'))`))
	if len(writers) == 0 {
		t.Fatal("no role writes records: the writer query is broken")
	}

	// hasPolicy reports whether role has a permissive policy for cmd on tbl
	// (cmd itself or ALL, the role named or PUBLIC).
	hasPolicy := func(tbl, cmd, role string) bool {
		return queryString(t, super, `
			SELECT count(*)::text FROM pg_policies
			WHERE schemaname = 'public' AND tablename = $1 AND permissive = 'PERMISSIVE'
			  AND cmd IN ($2, 'ALL') AND ($3 = ANY (roles) OR 'public' = ANY (roles))`,
			tbl, cmd, role) != "0"
	}

	for _, role := range writers {
		// The trigger's registry lookup.
		if !hasPolicy("record_index_fields", "SELECT", role) {
			t.Errorf("%s writes records but has no SELECT policy on record_index_fields", role)
		}
		if queryString(t, super, `SELECT has_table_privilege($1, 'record_index_fields', 'SELECT')::text`, role) != "true" {
			t.Errorf("%s writes records but has no SELECT grant on record_index_fields", role)
		}
		// The trigger's DELETE-then-INSERT of the record's side rows (D4).
		for _, tbl := range []string{"rfv_text", "rfv_num", "rfv_ts", "rfv_bool"} {
			for _, cmd := range []string{"SELECT", "INSERT", "DELETE"} {
				if !hasPolicy(tbl, cmd, role) {
					t.Errorf("%s writes records but has no %s policy on %s", role, cmd, tbl)
				}
				if queryString(t, super, `SELECT has_table_privilege($1, $2, $3)::text`, role, tbl, cmd) != "true" {
					t.Errorf("%s writes records but has no %s grant on %s", role, cmd, tbl)
				}
			}
		}
	}
	t.Logf("record writers checked: %v", writers)
}
