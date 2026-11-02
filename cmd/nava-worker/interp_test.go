package main

import (
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

const (
	modulePath    = "github.com/Navaladi2019/TicketingSystem"
	interptestPkg = modulePath + "/internal/workflow/interp/interptest"
	interpPkg     = modulePath + "/internal/workflow/interp"
)

// goList runs go list from the repository root and returns its lines.
func goList(t *testing.T, args ...string) []string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "go", append([]string{"list"}, args...)...)
	cmd.Dir = "../.."
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok { //nolint:errorlint // only for the message
			stderr = string(ee.Stderr)
		}
		t.Fatalf("go list %s: %v\n%s", strings.Join(args, " "), err, stderr)
	}
	return strings.Fields(string(out))
}

// TestProductionBuildLinksNoFakes is the gate that keeps the S11a fakes out of
// the production image (TICK-367): the untagged build of cmd/nava-worker,
// which the image's Dockerfile builds, links neither interptest (the fixture
// store, the file records and the counting writer) nor the interp package's
// pause-after-commit hook. The interptest build does link them, so the check
// is not vacuous.
func TestProductionBuildLinksNoFakes(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go list")
	}
	prod := goList(t, "-deps", "./cmd/nava-worker")
	if !slices.Contains(prod, interpPkg) {
		t.Fatalf("the production build does not link %s: the dependency list is not what this test expects", interpPkg)
	}
	if slices.Contains(prod, interptestPkg) {
		t.Errorf("the production build of cmd/nava-worker links %s", interptestPkg)
	}
	if files := goList(t, "-f", "{{range .GoFiles}}{{.}} {{end}}", "./internal/workflow/interp"); slices.Contains(files, "pause_interptest.go") ||
		!slices.Contains(files, "pause_default.go") {
		t.Errorf("the production build of interp compiles %v: want pause_default.go, not pause_interptest.go", files)
	}
	if tagged := goList(t, "-tags", "interptest", "-deps", "./cmd/nava-worker"); !slices.Contains(tagged, interptestPkg) {
		t.Errorf("the interptest build does not link %s: the gate above proves nothing", interptestPkg)
	}

	// The image builds the untagged binary: the tag ARG defaults to empty,
	// and the always-on nava-worker service passes no build args.
	b, err := os.ReadFile("../../deploy/compose/nava-worker/Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	argDefault, builds := "", 0
	for line := range strings.Lines(string(b)) {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "ARG NAVA_BUILD_TAGS"); ok {
			argDefault = v
		}
		if strings.Contains(line, "go build") {
			builds++
			if !strings.Contains(line, `-tags "${NAVA_BUILD_TAGS}"`) || !strings.HasSuffix(line, "./cmd/nava-worker") {
				t.Errorf("the nava-worker image build is not `-tags \"${NAVA_BUILD_TAGS}\" ... ./cmd/nava-worker`: %s", line)
			}
		}
	}
	if argDefault != `=""` {
		t.Errorf("the Dockerfile's NAVA_BUILD_TAGS default is %q, want empty (ARG NAVA_BUILD_TAGS=\"\")", argDefault)
	}
	if builds != 1 {
		t.Errorf("the Dockerfile has %d go build lines, want 1", builds)
	}
	if svc := composeService(t, "../../compose.yaml", "nava-worker"); svc == "" || strings.Contains(svc, "args:") ||
		strings.Contains(svc, "NAVA_BUILD_TAGS") || strings.Contains(svc, "NAVA_INTERPTEST") {
		t.Errorf("compose.yaml's nava-worker service sets build args or interptest variables:\n%s", svc)
	}
}

// composeService returns the lines of service name in the compose file at
// path: from "  <name>:" to the next two-space key.
func composeService(t *testing.T, path, name string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	in := false
	for line := range strings.Lines(string(b)) {
		trimmed := strings.TrimRight(line, "\r\n")
		if len(trimmed) > 2 && trimmed[:2] == "  " && trimmed[2] != ' ' && trimmed[2] != '#' {
			in = trimmed == "  "+name+":"
			continue
		}
		if in {
			out.WriteString(line)
		}
	}
	return out.String()
}

// TestRegisterInterpreter: with ports, the worker registers nava.interpreter
// and exactly NextStep and ApplyStep under their fixed names, and NextStep's
// name is the one activity type the guard lets record a Directive.
func TestRegisterInterpreter(t *testing.T) {
	dir := t.TempDir()
	r := &recordingRegistry{}
	registerInterpreter(r, &interp.Activities{
		Definitions: interptest.FixtureStore{Root: dir},
		Records:     interptest.FileRecords{Root: dir},
		Writer:      &interptest.FileWriter{Root: dir},
	})
	if !slices.Equal(r.workflows, []string{interp.WorkflowType}) {
		t.Errorf("workflows %v, want [%s]", r.workflows, interp.WorkflowType)
	}
	want := []string{interp.NextStepActivity, interp.ApplyStepActivity}
	if !slices.Equal(r.activities, want) {
		t.Errorf("activities %v, want %v", r.activities, want)
	}
	if interp.NextStepActivity != workerguard.DirectiveActivity {
		t.Errorf("NextStep is %q, the guard's Directive activity %q", interp.NextStepActivity, workerguard.DirectiveActivity)
	}
}

// TestWorkerOptionsPollWorkflowsWithTheInterpreter: the workflow poller is on
// only when the interpreter is registered.
func TestWorkerOptionsPollWorkflowsWithTheInterpreter(t *testing.T) {
	tel, _ := newTemporal(t)
	g, err := workerguard.New(slog.New(slog.DiscardHandler), tel.MeterProvider)
	if err != nil {
		t.Fatal(err)
	}
	if o := workerOptions(g, true, make(chan error, 1)); o.DisableWorkflowWorker {
		t.Error("the workflow poller is off with the interpreter registered")
	}
}

// recordingRegistry records the names registered on it.
type recordingRegistry struct {
	worker.Registry
	workflows, activities []string
}

func (r *recordingRegistry) RegisterWorkflowWithOptions(_ any, o workflow.RegisterOptions) {
	r.workflows = append(r.workflows, o.Name)
}

func (r *recordingRegistry) RegisterActivityWithOptions(_ any, o activity.RegisterOptions) {
	r.activities = append(r.activities, o.Name)
}
