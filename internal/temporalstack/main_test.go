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
// polls a task queue registers the same workflow types. A worker that takes a
// workflow task of a type it lacks fails it, and the run waits for a retry to
// reach the right worker, so a case that breaks the rule races, it does not
// fail cleanly.
//
// The always-on nava-worker registers audit.seal on nava-tq (ADR-004 R16),
// and this suite's bodies register nava.interpreter there (TICK-374). So
// every stack case that runs this package (sdk, demo_outbox) stops
// nava-worker first, as the workerstack interpreter cases do, and starts it
// again when the package ends. The raw workflow-task polls of the CI-6
// helpers (pollTaskFor) need the queue to themselves for the same reason.
// A new case that starts its own worker on nava-tq follows the same rule;
// a case that needs nava-worker up (the sealer) starts no worker of its own.
// Both cases that run this package are serial for that reason: a parallel
// case only reads the stack (deploy/compose/test/run-cases.sh).

// alwaysOnWorker is the compose service of the production-image worker.
const alwaysOnWorker = "nava-worker"

func TestMain(m *testing.M) {
	if out, err := compose(4*time.Minute, "stop", alwaysOnWorker); err != nil {
		fmt.Fprintf(os.Stderr, "temporalstack: compose stop %s: %v\n%s", alwaysOnWorker, err, out)
		os.Exit(1)
	}
	code := m.Run()
	// --no-deps: the stack is up; --wait: its /readyz healthcheck, so the
	// next case finds the sealer running.
	if out, err := compose(4*time.Minute, "up", "-d", "--no-deps", "--wait", "--wait-timeout", "180", alwaysOnWorker); err != nil {
		fmt.Fprintf(os.Stderr, "temporalstack: compose up %s after the suite: %v\n%s", alwaysOnWorker, err, out)
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
