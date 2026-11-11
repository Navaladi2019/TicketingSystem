package stackcases

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TICK-652: STACK_TIER=smoke runs only the tier=smoke cases, still filtered
// by AREAS, and names every other case in a SKIP line; STACK_TIER unset or all
// runs every case, so a laptop's make stack-tests is unchanged.
func TestTierSmokeRunsOnlySmokeCases(t *testing.T) {
	cases := map[string][2]string{
		"healthy":  {"job=stack order=0 tier=smoke", ran("healthy")},
		"smoke":    {"job=stack order=10 mode=parallel tier=smoke", ran("smoke")},
		"kcsmoke":  {"job=stack order=14 mode=parallel area=keycloak tier=smoke", ran("kcsmoke")},
		"attacks":  {"job=stack order=15", ran("attacks")},
		"isolated": {"job=stack order=10 mode=parallel", ran("isolated")},
		"restart":  {"job=stack order=50 area=keycloak", ran("restart")},
	}
	for _, c := range []struct {
		env      []string
		run, skp []string
	}{
		{nil, []string{"healthy", "smoke", "kcsmoke", "attacks", "isolated", "restart"}, nil},
		{[]string{"STACK_TIER=all"}, []string{"healthy", "smoke", "kcsmoke", "attacks", "isolated", "restart"}, nil},
		{[]string{"STACK_TIER=smoke"}, []string{"healthy", "smoke", "kcsmoke"}, []string{"attacks (full tier", "isolated (full tier", "restart (full tier"}},
		{[]string{"STACK_TIER=smoke", "PARALLEL=1"}, []string{"healthy", "smoke", "kcsmoke"}, []string{"attacks (full tier", "isolated (full tier", "restart (full tier"}},
		{[]string{"STACK_TIER=smoke", "AREAS=none"}, []string{"healthy", "smoke"}, []string{"kcsmoke (area keycloak", "attacks (full tier", "restart (full tier"}},
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
				name, _, _ := strings.Cut(s, " ")
				if strings.Contains(out, "RAN-"+name+"\n") {
					t.Errorf("%s ran: %s", name, out)
				}
				if !strings.Contains(out, "SKIP "+s) {
					t.Errorf("no %q: %s", "SKIP "+s, out)
				}
			}
			if len(c.skp) == 0 && strings.Contains(out, "SKIP ") {
				t.Errorf("skipped a case: %s", out)
			}
		})
	}
}

// A job with no smoke case passes under STACK_TIER=smoke and says so.
func TestTierSmokeSkipsEveryCase(t *testing.T) {
	dir := fixture(t, map[string][2]string{"dbroles": {"job=stack-db order=62", ran("dbroles")}})
	out, code := run(t, dir, []string{"STACK_TIER=smoke"}, "run", "stack-db")
	if code != 0 || strings.Contains(out, "RAN-dbroles") || !strings.Contains(out, "ok: every stack case in stack-db skipped") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

func TestBadTierFailsLoudly(t *testing.T) {
	dir := fixture(t, map[string][2]string{"a": {"job=stack order=1 tier=smok", ran("a")}})
	if out, code := run(t, dir, nil, "check"); code == 0 || !strings.Contains(out, "unknown tier 'smok'") {
		t.Fatalf("exit %d: %s", code, out)
	}
	dir = fixture(t, map[string][2]string{"a": {"job=stack order=1 tier=smoke", ran("a")}})
	if out, code := run(t, dir, []string{"STACK_TIER=fast"}, "run", "stack"); code == 0 || strings.Contains(out, "RAN-a") || !strings.Contains(out, "STACK_TIER: 'fast'") {
		t.Fatalf("exit %d: %s", code, out)
	}
}

// The smoke tier TICK-652 agreed: a change to it is a deliberate edit of this
// list. The security suites (identity_attacks CI-5, sdk CI-6, the Temporal
// isolation and confinement cases) must never be smoke: a fast run would then
// look like it covered them, and AR-TICK-652-01 says they run on full runs.
func TestRealSmokeTier(t *testing.T) {
	want := []string{"dependencies", "healthy", "identity", "smoke", "temporal", "temporal_health", "worker"}
	out, code := run(t, filepath.Join("..", "..", "..", "deploy", "compose", "test", "cases"), nil, "registry")
	if code != 0 {
		t.Fatalf("registry failed (%d): %s", code, out)
	}
	var got []string
	var name string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if v, ok := strings.CutPrefix(line, "name: "); ok {
			name = v
		}
		if v, ok := strings.CutPrefix(line, "tier: "); ok && v == "smoke" {
			got = append(got, name)
		}
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("smoke tier = %v, want %v", got, want)
	}
	for _, n := range []string{"identity_attacks", "sdk", "temporal_isolation", "temporal_confinement", "temporal_operator_reach", "temporal_custody", "temporal_handshake"} {
		if slices.Contains(got, n) {
			t.Errorf("%s is a full-run security suite, not smoke", n)
		}
	}
}
