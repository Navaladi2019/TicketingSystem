//go:build temporalstack

package temporalstack

import (
	"strings"
	"testing"

	"go.temporal.io/sdk/worker"
)

// interpService is the compose service that runs the real interpreter on
// nava-tq (compose.test.yaml, profile interptest). The workerstack interpreter
// cases start it and remove it before they end.
const interpService = "nava-worker-interptest"

// startPlatformWorker starts w, a worker on nava-tq with a test body
// registered as the platform type, and stops it when the test ends.
//
// The suite's bodies and nava-worker-interptest register the same workflow
// type on the same queue (TICK-374), so a task of one could reach the other.
// They must never poll nava-tq at the same time (Technical Lead, 2026-11-01).
// The stack-case order keeps them apart: the sdk case, which runs this suite,
// sorts before the serial worker and interpreter cases, which start
// nava-worker-interptest (tools/ci/stackcases TestInterpreterQueueOrdering
// asserts it). requireQueueFree checks it at runtime too, so a hand-run
// `make stack-test-sdk` beside an interpreter case fails instead of racing.
func startPlatformWorker(t *testing.T, w worker.Worker) {
	t.Helper()
	requireQueueFree(t)
	if err := w.Start(); err != nil {
		t.Fatalf("worker start: %v", err)
	}
	t.Cleanup(w.Stop)
}

// requireQueueFree fails the test if nava-worker-interptest has a running
// container in the compose project.
func requireQueueFree(t *testing.T) {
	t.Helper()
	out := docker(t, "ps", "-q", "--filter", "label=com.docker.compose.project="+composeProject(),
		"--filter", "label=com.docker.compose.service="+interpService)
	if id := strings.TrimSpace(out); id != "" {
		t.Fatalf("%s is running (%s): it polls nava-tq as %s, so this suite's test bodies "+
			"must not; run the suite when no interpreter case is running", interpService, id, platformType)
	}
}
