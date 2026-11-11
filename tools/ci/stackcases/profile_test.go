package stackcases

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TICK-696: `run-cases.sh profiles <jobs>` prints operator when a case that
// runs under AREAS and STACK_TIER has profile=operator, nothing otherwise.
// make/stack.mk sets COMPOSE_PROFILES from it when STACK_TIER is set, so a
// fast run neither pulls nor starts temporal-ui, OpenObserve and Prometheus.
func TestProfilesFollowTheSelectedCases(t *testing.T) {
	cases := map[string][2]string{
		"healthy": {"job=stack,stack-db order=0 tier=smoke", ran("healthy")},
		"smoke":   {"job=stack order=10 mode=parallel tier=smoke", ran("smoke")},
		"trace":   {"job=stack order=52 profile=operator", ran("trace")},
		"kcui":    {"job=stack order=53 area=keycloak tier=smoke profile=operator", ran("kcui")},
		"dbroles": {"job=stack-db order=62", ran("dbroles")},
	}
	for _, c := range []struct {
		jobs string
		env  []string
		want string
	}{
		{"stack", nil, "operator"},                                      // laptop: every case
		{"stack", []string{"STACK_TIER=all"}, "operator"},               // full run
		{"stack", []string{"STACK_TIER=all", "AREAS=none"}, "operator"}, // trace has no area
		{"stack", []string{"STACK_TIER=smoke"}, "operator"},             // kcui is smoke
		{"stack", []string{"STACK_TIER=smoke", "AREAS=none"}, ""},       // kcui's area untouched
		{"stack", []string{"STACK_TIER=smoke", "AREAS=pgbouncer"}, ""},
		{"stack,stack-db", []string{"STACK_TIER=smoke", "AREAS=keycloak"}, "operator"},
		{"stack-db", []string{"STACK_TIER=all"}, ""},
		{"stack-db", nil, ""},
	} {
		t.Run(c.jobs+" "+strings.Join(c.env, " "), func(t *testing.T) {
			out, code := run(t, fixture(t, cases), c.env, "profiles", c.jobs)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, out)
			}
			if got := strings.TrimSpace(out); got != c.want {
				t.Errorf("profiles = %q, want %q", got, c.want)
			}
			if strings.Contains(out, "RAN-") {
				t.Errorf("profiles ran a case: %s", out)
			}
		})
	}
}

// An unknown profile, STACK_TIER or AREAS fails loudly: an empty answer would
// leave the operator services off a run that needs them.
func TestBadProfileFailsLoudly(t *testing.T) {
	dir := fixture(t, map[string][2]string{"a": {"job=stack order=1 profile=operater", ran("a")}})
	for _, args := range [][]string{{"check"}, {"profiles", "stack"}, {"run", "stack"}} {
		if out, code := run(t, dir, nil, args...); code == 0 || !strings.Contains(out, "unknown profile 'operater'") {
			t.Errorf("%v: exit %d: %s", args, code, out)
		}
	}
	dir = fixture(t, map[string][2]string{"a": {"job=stack order=1 profile=operator", ran("a")}})
	for _, env := range []string{"STACK_TIER=fast", "AREAS=keyclok"} {
		if out, code := run(t, dir, []string{env}, "profiles", "stack"); code == 0 || strings.Contains(out, "operator\n") {
			t.Errorf("%s: exit %d: %s", env, code, out)
		}
	}
}

// operatorServices: the services of compose.yaml in the operator profile.
func operatorServices(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot, "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Services map[string]struct {
			Profiles []string `yaml:"profiles"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var svcs []string
	for name, s := range doc.Services {
		if slices.Contains(s.Profiles, "operator") {
			svcs = append(svcs, name)
		}
	}
	slices.Sort(svcs)
	return svcs
}

// A case that names a service of the operator profile must carry
// profile=operator, or a fast run would select it with the service absent
// (TICK-696). The check greps each case's code (whole-line comments left out),
// like the other registry guards; TEMPORAL_UI_PORT counts as temporal-ui.
func TestOperatorServicesNeedTheProfile(t *testing.T) {
	svcs := operatorServices(t)
	if want := []string{"openobserve", "prometheus", "temporal-ui"}; !slices.Equal(svcs, want) {
		// Not an error by itself; the guard below covers whatever the profile holds.
		t.Logf("operator profile services = %v (TICK-696 had %v)", svcs, want)
	}
	if len(svcs) == 0 {
		t.Fatal("no service in compose.yaml's operator profile: the parse is broken")
	}
	var alts []string
	for _, s := range svcs {
		alts = append(alts, strings.ReplaceAll(regexp.QuoteMeta(s), "-", "[-_]"))
	}
	uses := regexp.MustCompile(`(?i)\b(` + strings.Join(alts, "|") + `)`)
	files, err := filepath.Glob(filepath.Join(casesDir, "*.sh"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no case files in %s: %v", casesDir, err)
	}
	header := regexp.MustCompile(`(?m)^# stack-case: (.*)$`)
	comment := regexp.MustCompile(`(?m)^[ \t]*#.*$`) // whole-line comments, the header among them
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		h := header.FindSubmatch(raw)
		if h == nil {
			t.Errorf("%s: no stack-case header", f)
			continue
		}
		body := comment.ReplaceAll(raw, nil)
		if m := uses.Find(body); m != nil && !slices.Contains(strings.Fields(string(h[1])), "profile=operator") {
			t.Errorf("%s names %q (operator profile) but its header lacks profile=operator", filepath.Base(f), m)
		}
	}
}

// The profile=operator cases TICK-696 agreed. trace, logs and redact read the
// Collector's debug exporter, but run with OpenObserve as its second exporter,
// as on a laptop; the temporal_* two check temporal-ui itself. All five are
// full-tier, so a fast run starts none of the three services.
func TestRealOperatorCases(t *testing.T) {
	want := []string{"logs", "redact", "temporal_custody", "temporal_operator_reach", "trace"}
	out, code := run(t, casesDir, nil, "registry")
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
		if v, ok := strings.CutPrefix(line, "profile: "); ok && v == "operator" {
			got = append(got, name)
		}
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("profile=operator cases = %v, want %v", got, want)
	}
	for _, env := range [][]string{{"STACK_TIER=smoke"}, {"STACK_TIER=smoke", "AREAS=none"}} {
		if out, code := run(t, casesDir, env, "profiles", "stack,stack-db"); code != 0 || strings.TrimSpace(out) != "" {
			t.Errorf("%v: a fast run selects the operator profile (exit %d): %q", env, code, out)
		}
	}
	if out, code := run(t, casesDir, []string{"STACK_TIER=all"}, "profiles", "stack,stack-db"); code != 0 || strings.TrimSpace(out) != "operator" {
		t.Errorf("a full run does not select the operator profile (exit %d): %q", code, out)
	}
}

// The stack job's STACK_TIER repeats the workflow's CI_FULL expression (job
// env cannot read env); a change to one without the other would start or skip
// the full-tier cases and the operator profile on the wrong runs (TICK-696).
func TestStackTierRepeatsCIFull(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	full := regexp.MustCompile(`(?m)^  CI_FULL: \$\{\{ (.*) \}\}$`).FindSubmatch(raw)
	tier := regexp.MustCompile(`(?m)^      STACK_TIER: \$\{\{ \((.*)\) && 'all' \|\| 'smoke' \}\}$`).FindAllSubmatch(raw, -1)
	if full == nil || len(tier) != 1 {
		t.Fatalf("ci.yml: workflow CI_FULL found %v, stack job STACK_TIER lines %d (want 1)", full != nil, len(tier))
	}
	if string(tier[0][1]) != string(full[1]) {
		t.Errorf("STACK_TIER condition\n  %s\ndiffers from CI_FULL\n  %s", tier[0][1], full[1])
	}
}
