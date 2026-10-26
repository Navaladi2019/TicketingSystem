//go:build integration

package dbtest

import (
	"strings"
	"testing"
)

// TICK-233 adversarial (Security & Multi-Tenancy review of PR #139): every
// grant shape a migration could hand-write, beyond the plain table GRANT and
// the global default privilege the PR covers, fails a fresh migrator run and
// is named. Each grant here would widen a runtime role past db/grants.yaml if
// it reached a deployed database unnoticed.
func TestT10AdversarialHandGrantShapesFailFreshRun(t *testing.T) {
	for _, c := range []struct{ name, up, want string }{
		// A column the manifest does not list for nava_app.
		{"column",
			"GRANT UPDATE (state) ON tenants TO nava_app;",
			"UPDATE column (state) ON public.tenants TO nava_app"},
		// PUBLIC reaches every login role, the pooler's included.
		{"public",
			"GRANT SELECT ON records TO PUBLIC;",
			"SELECT table public.records TO PUBLIC"},
		// A listed privilege, re-granted WITH GRANT OPTION: nava_app could pass
		// it on to any role.
		{"grant_option_on_listed",
			"GRANT SELECT ON records TO nava_app WITH GRANT OPTION;",
			"SELECT table public.records TO nava_app WITH GRANT OPTION"},
		{"schema",
			"GRANT CREATE ON SCHEMA public TO nava_app;",
			"CREATE schema public TO nava_app"},
		{"database",
			"GRANT TEMPORARY ON DATABASE nava TO nava_relay;",
			"TEMPORARY database nava TO nava_relay"},
		{"function",
			"CREATE FUNCTION t233_probe() RETURNS int LANGUAGE sql AS 'SELECT 1';\n" +
				"GRANT EXECUTE ON FUNCTION t233_probe() TO nava_app;",
			"EXECUTE function public.t233_probe() TO nava_app"},
		// A schema-scoped default: every table the migrator creates in public
		// from then on would inherit it.
		{"schema_default_privilege",
			"ALTER DEFAULT PRIVILEGES FOR ROLE nava_migrator IN SCHEMA public GRANT SELECT ON TABLES TO nava_relay;",
			"SELECT default DEFAULT TABLES IN SCHEMA public TO nava_relay"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := freshCluster(t)
			_, err := migrateFS(t, m, withExtraMigration(t, "99999_t233_adversarial.sql", "-- +goose Up\n"+c.up+"\n"))
			if err == nil {
				t.Fatalf("migrator passed a fresh run with %s", c.up)
			}
			// The whole grant, in its Grant.String form, must be named: a
			// looser match would pass on a revoke of some other grant.
			for _, line := range strings.Split(err.Error(), "\n") {
				if strings.HasPrefix(line, "CI-4: a migration granted "+c.want+", ") {
					return
				}
			}
			t.Fatalf("no CI-4 hand-grant problem naming %q:\n%v", c.want, err)
		})
	}
}
