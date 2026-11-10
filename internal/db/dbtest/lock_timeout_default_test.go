//go:build integration

package dbtest

import "testing"

// TICK-690: tenancy.QueueUnderLockTimeout resets lock_timeout to '0' after
// the statement it bounds, not to the value in effect before. That is right
// only while '0' is what every platform session starts with: no role, no
// database and no role-in-database default sets lock_timeout. A default added
// later (ALTER ROLE ... SET, ALTER DATABASE ... SET) fails here, and the reset
// must then restore the previous value instead.
func TestNoRoleOrDatabaseDefaultSetsLockTimeout(t *testing.T) {
	migratedCluster(t)
	super := connect(t, superConfig(t))
	if got := queryString(t, super, `
		SELECT coalesce(string_agg(coalesce(r.rolname, '*') || '@' || coalesce(d.datname, '*') || ': ' || c, ', '), '')
		FROM pg_db_role_setting s
		CROSS JOIN LATERAL unnest(s.setconfig) c
		LEFT JOIN pg_roles r ON r.oid = s.setrole
		LEFT JOIN pg_database d ON d.oid = s.setdatabase
		WHERE c ILIKE 'lock_timeout=%'`); got != "" {
		t.Errorf("lock_timeout defaults set: %s; want none (the reset to '0' would override them)", got)
	}
	// The session nava_app (the role QueueUnderLockTimeout runs as) starts
	// with: the server default, from no role or database setting.
	app := connect(t, roleConfig(t, "nava_app", "nava"))
	if got := queryString(t, app, "SELECT setting || ' ' || source FROM pg_settings WHERE name = 'lock_timeout'"); got != "0 default" {
		t.Errorf("nava_app's lock_timeout = %q, want \"0 default\"", got)
	}
}
