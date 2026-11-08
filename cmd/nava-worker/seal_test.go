package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// TestRegisterSealer: the worker registers audit.seal and exactly A1 and A2
// under their fixed names, on every build, and with a workflow registered it
// polls workflow tasks.
func TestRegisterSealer(t *testing.T) {
	r := &recordingRegistry{}
	registerSealer(r, &seal.Activities{}, true)
	if !slices.Equal(r.workflows, []string{seal.WorkflowType}) || seal.WorkflowType != "audit.seal" {
		t.Errorf("workflows %v, want [audit.seal]", r.workflows)
	}
	if want := []string{seal.RescheduleActivity, seal.SealActivity}; !slices.Equal(r.activities, want) {
		t.Errorf("activities %v, want %v", r.activities, want)
	}

	tel, _ := newTemporal(t)
	g, err := workerguard.New(slog.New(slog.DiscardHandler), tel.MeterProvider)
	if err != nil {
		t.Fatal(err)
	}
	if o := workerOptions(g, true, make(chan error, 1)); o.DisableWorkflowWorker {
		t.Error("the workflow poller is off with audit.seal registered")
	}
}

// Activity-only (WorkflowsEnv=off): A1 and A2 are registered, audit.seal is
// not, and the worker polls no workflow task.
func TestRegisterSealerActivitiesOnly(t *testing.T) {
	r := &recordingRegistry{}
	registerSealer(r, &seal.Activities{}, false)
	if len(r.workflows) != 0 {
		t.Errorf("workflows %v registered with %s=off", r.workflows, WorkflowsEnv)
	}
	if want := []string{seal.RescheduleActivity, seal.SealActivity}; !slices.Equal(r.activities, want) {
		t.Errorf("activities %v, want %v", r.activities, want)
	}
}

func TestWorkflowsMode(t *testing.T) {
	for v, want := range map[string]bool{"": true, "on": true, "off": false} {
		got, err := workflowsMode(func(string) (string, bool) { return v, v != "" })
		if err != nil || got != want {
			t.Errorf("%s=%q: (%v, %v), want (%v, nil)", WorkflowsEnv, v, got, err, want)
		}
	}
	for _, v := range []string{"false", "OFF", "0", "no"} {
		if _, err := workflowsMode(func(string) (string, bool) { return v, true }); err == nil {
			t.Errorf("%s=%q accepted", WorkflowsEnv, v)
		}
	}
}

// workerOverlays are the compose files a stack is brought up with: compose.yaml
// and every overlay beside it or in deploy/compose. Only compose.test.yaml is
// left out: its nava-worker-activities is the one service that turns
// WorkflowsEnv off, under its own name (TICK-589).
func workerOverlays(t *testing.T) []string {
	t.Helper()
	var files []string
	for _, pattern := range []string{"../../compose*.yaml", "../../deploy/compose/*.yaml"} {
		m, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, m...)
	}
	files = slices.DeleteFunc(files, func(f string) bool { return filepath.Base(f) == "compose.test.yaml" })
	// A moved directory or a broken pattern must not empty the list and pass.
	for _, want := range []string{"compose.yaml", "compose.ci.yaml", "compose.internal.yaml"} {
		if !slices.ContainsFunc(files, func(f string) bool { return filepath.Base(f) == want }) {
			t.Fatalf("%s is not among the guarded compose files %v", want, files)
		}
	}
	return files
}

// workerSealerOff says why a compose file's nava-worker could run without
// audit.seal, or "" if it cannot: its environment sets WorkflowsEnv to any
// value (map or list form, merge keys resolved), or it reads an env_file the
// guard cannot see into. A file with no nava-worker service passes.
func workerSealerOff(doc []byte) (string, error) {
	var f struct {
		Services map[string]struct {
			Environment any `yaml:"environment"`
			EnvFile     any `yaml:"env_file"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(doc, &f); err != nil {
		return "", err
	}
	svc, ok := f.Services["nava-worker"]
	if !ok {
		return "", nil
	}
	if svc.EnvFile != nil {
		return "nava-worker reads an env_file, which may set " + WorkflowsEnv, nil
	}
	switch env := svc.Environment.(type) {
	case nil:
	case map[string]any:
		if v, set := env[WorkflowsEnv]; set {
			return fmt.Sprintf("nava-worker sets %s: %v", WorkflowsEnv, v), nil
		}
	case []any:
		for _, e := range env {
			if s := fmt.Sprint(e); s == WorkflowsEnv || strings.HasPrefix(s, WorkflowsEnv+"=") {
				return "nava-worker sets " + s, nil
			}
		}
	default:
		return "", fmt.Errorf("nava-worker environment is a %T", env)
	}
	return "", nil
}

func readCompose(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// No compose file a stack is brought up with turns the sealer off on the
// always-on worker (TICK-589): neither compose.yaml nor any overlay sets
// WorkflowsEnv on nava-worker. Only the test overlay's nava-worker-activities
// turns it off, reusing the nava-worker image with the nava-worker identity.
func TestComposeWorkerWorkflowsOn(t *testing.T) {
	for _, p := range workerOverlays(t) {
		why, err := workerSealerOff([]byte(readCompose(t, p)))
		if err != nil {
			t.Errorf("%s: %v", p, err)
		} else if why != "" {
			t.Errorf("%s: %s; the always-on worker would not run audit.seal", p, why)
		}
	}
	read := func(p string) string { return readCompose(t, p) }
	if svc := serviceBlock(t, read("../../compose.yaml"), "nava-worker"); strings.Contains(svc, WorkflowsEnv) {
		t.Errorf("compose.yaml's nava-worker sets %s", WorkflowsEnv)
	}
	act := serviceBlock(t, read("../../deploy/compose/compose.test.yaml"), "nava-worker-activities")
	for _, want := range []string{
		WorkflowsEnv + `: "off"`, "image: nava/nava-worker:${NAVA_IMAGE_TAG:-dev}", "pull_policy: never",
		"profiles: [activityonly]", "nava-worker.pem", "postgres://nava_worker:", "/readyz",
	} {
		if !strings.Contains(act, want) {
			t.Errorf("nava-worker-activities has no %q:\n%s", want, act)
		}
	}
	if strings.Contains(act, "build:") {
		t.Error("nava-worker-activities builds an image: it must reuse nava-worker's")
	}
}

// plantWorkflowsOff adds `NAVA_WORKER_WORKFLOWS: "off"` to nava-worker in a
// compose file: into its environment if the file defines the service, else as
// a new nava-worker override under services (added if the file has none).
func plantWorkflowsOff(doc string) string {
	const (
		svc   = "\n  nava-worker:\n"
		env   = "\n    environment:\n"
		plant = "      " + WorkflowsEnv + ": \"off\"\n"
	)
	if before, block, ok := strings.Cut(doc, svc); ok {
		// environment belongs to nava-worker only if no other service starts first.
		if head, rest, ok := strings.Cut(block, env); ok && !slices.ContainsFunc(strings.Split(head, "\n"), func(l string) bool {
			return l != "" && !strings.HasPrefix(l, "    ")
		}) {
			return before + svc + head + env + plant + rest
		}
		return before + svc + "    environment:\n" + plant + block
	}
	override := "  nava-worker:\n    environment:\n" + plant
	if before, rest, ok := strings.Cut(doc, "\nservices:\n"); ok {
		return before + "\nservices:\n" + override + rest
	}
	if rest, ok := strings.CutPrefix(doc, "services:\n"); ok {
		return "services:\n" + override + rest
	}
	return strings.TrimRight(doc, "\n") + "\nservices:\n" + override
}

// The guard is live in every file it covers: a planted
// NAVA_WORKER_WORKFLOWS: "off" on nava-worker fails it, whether the file
// defines nava-worker (compose.yaml), only other services (compose.ci.yaml) or
// no services at all (compose.internal.yaml).
func TestComposeWorkerWorkflowsOnCatchesPlant(t *testing.T) {
	for _, p := range workerOverlays(t) {
		t.Run(filepath.Base(p), func(t *testing.T) {
			planted := plantWorkflowsOff(readCompose(t, p))
			why, err := workerSealerOff([]byte(planted))
			if err != nil {
				t.Fatalf("the planted file does not parse: %v\n%s", err, planted)
			}
			if !strings.Contains(why, WorkflowsEnv) {
				t.Errorf("a planted %s: \"off\" passed the guard (%q):\n%s", WorkflowsEnv, why, planted)
			}
		})
	}
}

// Every form compose accepts for an environment, and env_file, trip the guard;
// a nava-worker without WorkflowsEnv, and other services, do not.
func TestWorkerSealerOff(t *testing.T) {
	const w = "services:\n  nava-worker:\n"
	for name, tc := range map[string]struct {
		doc    string
		caught bool
	}{
		"map":            {w + "    environment:\n      NAVA_WORKER_WORKFLOWS: \"off\"\n", true},
		"map unquoted":   {w + "    environment:\n      NAVA_WORKER_WORKFLOWS: off\n", true},
		"map on":         {w + "    environment:\n      NAVA_WORKER_WORKFLOWS: \"on\"\n", true},
		"list":           {w + "    environment:\n      - NAVA_WORKER_WORKFLOWS=off\n", true},
		"list from host": {w + "    environment: [NAVA_WORKER_WORKFLOWS]\n", true},
		"flow":           {"services:\n  nava-worker: {environment: {NAVA_WORKER_WORKFLOWS: \"off\"}}\n", true},
		"merge key": {"x-off: &off\n  NAVA_WORKER_WORKFLOWS: \"off\"\n" +
			w + "    environment:\n      <<: *off\n      NAVA_LOG_FORMAT: json\n", true},
		"env_file":      {w + "    env_file: worker.env\n", true},
		"other env":     {w + "    environment:\n      NAVA_LOG_FORMAT: json\n", false},
		"longer name":   {w + "    environment:\n      - NAVA_WORKER_WORKFLOWS_X=off\n", false},
		"other service": {"services:\n  nava-worker-activities:\n    environment:\n      NAVA_WORKER_WORKFLOWS: \"off\"\n", false},
		"no services":   {"networks:\n  default:\n    internal: true\n", false},
	} {
		why, err := workerSealerOff([]byte(tc.doc))
		if err != nil {
			t.Errorf("%s: %v", name, err)
		} else if (why != "") != tc.caught {
			t.Errorf("%s: guard said %q, want caught=%v", name, why, tc.caught)
		}
	}
}
