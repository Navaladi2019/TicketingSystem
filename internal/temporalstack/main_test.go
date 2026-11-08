//go:build temporalstack

package temporalstack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// The nava-tq rule (TICK-453, Technical Lead 2026-11-07): every worker that
// polls workflow tasks on a queue registers the same workflow types. A worker that
// takes a workflow task of a type it lacks fails it, and the run waits for a retry to
// reach the right worker, so a case that breaks the rule races, it does not
// fail cleanly.
//
// The always-on nava-worker registers audit.seal on nava-tq, and this
// suite's bodies register nava.interpreter there (TICK-374). So every stack
// case that runs this package (sdk, demo_outbox) swaps nava-worker for
// nava-worker-activities (deploy/compose/compose.test.yaml): the same image
// and identity with NAVA_WORKER_WORKFLOWS=off. It still runs the service's
// activities, which the bodies schedule and do not run themselves
// (nava.workflow_ref.Close at the end of every nava.interpreter run), and it
// polls no workflow task, so the bodies are the only workflow pollers. The raw
// workflow-task polls of the CI-6 helpers (pollTaskFor) need that too.
// TestMain swaps them back when the package ends. A stopped nava-worker with
// no activity worker in its place leaves every run open until its wait
// expires, so TestMain fails at once if nava-worker-activities is not ready,
// and tools/ci/stackcases TestTemporalstackKeepsTheServiceActivities checks
// the swap in this file.
//
// A new case that starts its own workflow worker on nava-tq does the same;
// a case that needs the sealer (nava-worker up) starts no workflow worker of
// its own. Both cases that run this package are serial for that reason: a
// parallel case only reads the stack (deploy/compose/test/run-cases.sh).

// The always-on worker and its activity-only stand-in, and the stand-in's
// compose profile.
const (
	alwaysOnWorker = "nava-worker"
	activityWorker = "nava-worker-activities"
	activityOnly   = "activityonly"
)

func TestMain(m *testing.M) {
	if out, err := compose(4*time.Minute, "stop", alwaysOnWorker); err != nil {
		fmt.Fprintf(os.Stderr, "temporalstack: compose stop %s: %v: %s", alwaysOnWorker, err, out)
		os.Exit(1)
	}
	code := 1
	// --no-deps: the stack is up; --wait: its /readyz healthcheck, so the
	// service's activities have a poller before the first body starts.
	if out, err := compose(4*time.Minute, "--profile", activityOnly, "up", "-d", "--no-deps", "--wait", "--wait-timeout", "180", activityWorker); err != nil {
		fmt.Fprintf(os.Stderr, "temporalstack: compose up %s: %v: %s", activityWorker, err, out)
	} else {
		code = m.Run()
	}
	if out, err := compose(2*time.Minute, "--profile", activityOnly, "rm", "--stop", "--force", activityWorker); err != nil {
		fmt.Fprintf(os.Stderr, "temporalstack: compose rm %s: %v: %s", activityWorker, err, out)
		code = 1
	}
	if out, err := compose(4*time.Minute, "up", "-d", "--no-deps", "--wait", "--wait-timeout", "180", alwaysOnWorker); err != nil {
		fmt.Fprintf(os.Stderr, "temporalstack: compose up %s after the suite: %v: %s", alwaysOnWorker, err, out)
		code = 1
	}
	os.Exit(code)
}

// compose runs `docker compose` from the repository root, so COMPOSE_FILE and
// COMPOSE_PROJECT_NAME from make apply, as for the workerstack helpers.
func compose(timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	return string(out), err
}
