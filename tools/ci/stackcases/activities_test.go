package stackcases

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The service-activities rule (TICK-453): a stack case that stops the
// always-on nava-worker, so that its own workflow worker is the only one on
// nava-tq, must start nava-worker-activities in its place before any test
// runs. The suite's nava.interpreter bodies schedule nava.workflow_ref.Close
// and never run it; with no activity worker every run stays open until its
// wait expires, and the package fails only on its 20-minute timeout. These
// checks fail `task test` at once instead.

const (
	temporalstackMain = repoRoot + "/internal/temporalstack/main_test.go"
	composeTest       = repoRoot + "/deploy/compose/compose.test.yaml"
)

// activitiesViolations returns one message per broken rule in a
// temporalstack TestMain (main) and the compose test overlay (overlay).
func activitiesViolations(main, overlay string) []string {
	var out []string
	stop := strings.Index(main, `"stop", alwaysOnWorker`)
	up := strings.Index(main, `"up", "-d", "--no-deps", "--wait", "--wait-timeout", "180", activityWorker)`)
	run := strings.Index(main, "m.Run()")
	switch {
	case !strings.Contains(main, `activityWorker = "nava-worker-activities"`) || !strings.Contains(main, `activityOnly   = "activityonly"`):
		out = append(out, "temporalstack TestMain does not name nava-worker-activities and its activityonly profile")
	case stop >= 0 && up < 0:
		out = append(out, "temporalstack TestMain stops nava-worker but never starts nava-worker-activities (with --wait): nothing would run nava.workflow_ref.Close")
	case stop >= 0 && (run < 0 || up > run):
		out = append(out, "temporalstack TestMain starts nava-worker-activities after m.Run: the first tests would have no activity worker")
	case !strings.Contains(main, `"--profile", activityOnly, "up"`):
		out = append(out, "temporalstack TestMain starts nava-worker-activities without its profile")
	}
	svc := serviceText(overlay, "nava-worker-activities")
	switch {
	case svc == "":
		out = append(out, "compose.test.yaml defines no nava-worker-activities service")
	case !strings.Contains(svc, `NAVA_WORKER_WORKFLOWS: "off"`):
		out = append(out, `nava-worker-activities does not set NAVA_WORKER_WORKFLOWS: "off": it would poll workflow tasks beside the suite's bodies`)
	case !strings.Contains(svc, "profiles: [activityonly]"):
		out = append(out, "nava-worker-activities is not in the activityonly profile: a plain `up` would start it")
	}
	return out
}

// serviceText returns the lines of one top-level service of a compose file,
// "" if it has none.
func serviceText(compose, name string) string {
	compose = strings.ReplaceAll(compose, "\r\n", "\n")
	_, rest, ok := strings.Cut(compose, "\n  "+name+":\n")
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(rest, "\n") {
		if line != "" && !strings.HasPrefix(line, "    ") {
			break
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func TestTemporalstackKeepsTheServiceActivities(t *testing.T) {
	read := func(p string) string {
		b, err := os.ReadFile(filepath.Clean(p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	main, overlay := read(temporalstackMain), read(composeTest)
	if !strings.Contains(main, `"stop", alwaysOnWorker`) {
		t.Fatal("temporalstack TestMain no longer stops nava-worker: this check is not about the code it reads any more")
	}
	for _, v := range activitiesViolations(main, overlay) {
		t.Error(v)
	}

	// Mutation check: each way of breaking the rule is reported.
	upLine := `"--profile", activityOnly, "up", "-d", "--no-deps", "--wait", "--wait-timeout", "180", activityWorker)`
	for what, m := range map[string]struct{ main, overlay string }{
		"no activity worker":  {strings.ReplaceAll(main, upLine, `"--profile", activityOnly, "ps")`), overlay},
		"started after m.Run": {strings.Replace(main, "code := 1", "code := m.Run()", 1), overlay},
		"workflows on":        {main, strings.Replace(overlay, `NAVA_WORKER_WORKFLOWS: "off"`, `NAVA_WORKER_WORKFLOWS: "on"`, 1)},
		"no service":          {main, strings.Replace(overlay, "\n  nava-worker-activities:\n", "\n  nava-worker-gone:\n", 1)},
		"no profile":          {main, strings.Replace(overlay, "profiles: [activityonly]", "profiles: []", 1)},
	} {
		if got := activitiesViolations(m.main, m.overlay); len(got) == 0 {
			t.Errorf("%s: not reported", what)
		}
	}
}
