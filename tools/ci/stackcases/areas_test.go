package stackcases

import (
	"path/filepath"
	"strings"
	"testing"
)

// TICK-563: AREAS skips the cases of untouched areas with a SKIP line; a
// case without area= always runs; AREAS unset or all runs every case.
func TestAreasSkipUntouchedCases(t *testing.T) {
	cases := map[string][2]string{
		"healthy":  {"job=stack order=0", ran("healthy")},
		"kc":       {"job=stack order=10 mode=parallel area=keycloak", ran("kc")},
		"pgb":      {"job=stack order=10 mode=parallel area=pgbouncer", ran("pgb")},
		"attacks":  {"job=stack order=15", ran("attacks")},
		"restart":  {"job=stack order=50 area=keycloak", ran("restart")},
		"idem":     {"job=stack order=60 area=db", ran("idem")},
		"isolated": {"job=stack order=70 mode=parallel", ran("isolated")},
	}
	for _, c := range []struct {
		env      []string
		run, skp []string
	}{
		{nil, []string{"healthy", "kc", "pgb", "attacks", "restart", "idem", "isolated"}, nil},
		{[]string{"AREAS=all"}, []string{"healthy", "kc", "pgb", "attacks", "restart", "idem", "isolated"}, nil},
		{[]string{"AREAS=none"}, []string{"healthy", "attacks", "isolated"}, []string{"kc keycloak", "pgb pgbouncer", "restart keycloak", "idem db"}},
		{[]string{"AREAS=keycloak,db"}, []string{"healthy", "kc", "attacks", "restart", "idem", "isolated"}, []string{"pgb pgbouncer"}},
		{[]string{"AREAS=none", "PARALLEL=1"}, []string{"healthy", "attacks", "isolated"}, []string{"kc keycloak", "pgb pgbouncer", "restart keycloak", "idem db"}},
	} {
		t.Run(strings.Join(c.env, " "), func(t *testing.T) {
			out, code := run(t, fixture(t, cases), c.env, "run", "stack")
			if code != 0 {
				t.Fatalf("exit %d: %s", code, out)
			}
			for _, n := range c.run {
				if !strings.Contains(out, "RAN-"+n+"\n") {
					t.Errorf("%s did not run: %s", n, out)
				}
			}
			for _, s := range c.skp {
				name, area, _ := strings.Cut(s, " ")
				if strings.Contains(out, "RAN-"+name+"\n") {
					t.Errorf("%s ran: %s", name, out)
				}
				if want := "SKIP " + name + " (area " + area + " not touched)"; !strings.Contains(out, want) {
					t.Errorf("no %q: %s", want, out)
				}
			}
			if len(c.skp) == 0 && strings.Contains(out, "SKIP ") {
				t.Errorf("skipped a case: %s", out)
			}
		})
	}
}

// Every case of a job may be skipped: the run passes and says so.
func TestAreasSkipEveryCase(t *testing.T) {
	dir := fixture(t, map[string][2]string{"idem": {"job=stack-db order=60 area=db", ran("idem")}})
	out, code := run(t, dir, []string{"AREAS=none"}, "run", "stack-db")
	if code != 0 || strings.Contains(out, "RAN-idem") || !strings.Contains(out, "ok: every stack case in stack-db skipped") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

func TestBadAreaFailsLoudly(t *testing.T) {
	dir := fixture(t, map[string][2]string{"a": {"job=stack order=1 area=keycloack", ran("a")}})
	if out, code := run(t, dir, nil, "check"); code == 0 || !strings.Contains(out, "unknown area 'keycloack'") {
		t.Fatalf("exit %d: %s", code, out)
	}
	dir = fixture(t, map[string][2]string{"a": {"job=stack order=1 area=db", ran("a")}})
	if out, code := run(t, dir, []string{"AREAS=keycloak,dbb"}, "run", "stack"); code == 0 || strings.Contains(out, "RAN-a") || !strings.Contains(out, "AREAS: unknown area 'dbb'") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

// The owner's rule: identity_attacks and every tenant-isolation case carry no
// area, so no AREAS value skips them; the five area cases are the ones TICK-563
// names.
func TestRealCaseAreas(t *testing.T) {
	want := map[string]string{
		"keycloak_restart": "keycloak", "dependencies": "keycloak", "identity": "keycloak",
		"pgbouncer": "pgbouncer", "idempotent": "db",
	}
	out, code := run(t, filepath.Join("..", "..", "..", "deploy", "compose", "test", "cases"), nil, "registry")
	if code != 0 {
		t.Fatalf("registry failed (%d): %s", code, out)
	}
	got := map[string]string{}
	var name string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if v, ok := strings.CutPrefix(line, "name: "); ok {
			name = v
		}
		if v, ok := strings.CutPrefix(line, "area: "); ok {
			got[name] = v
		}
	}
	for n, a := range want {
		if got[n] != a {
			t.Errorf("case %s: area %q, want %q", n, got[n], a)
		}
	}
	for n, a := range got {
		if want[n] == "" {
			t.Errorf("case %s has area %q; only the cases TICK-563 names may", n, a)
		}
	}
	for _, n := range []string{"identity_attacks", "temporal_isolation"} {
		if got[n] != "" {
			t.Errorf("%s must never be skipped, has area %q", n, got[n])
		}
	}
}
