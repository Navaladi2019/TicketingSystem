//go:build workerstack && interpcrash

package workerstack

// The interpreter crash case (P3-24, TICK-351, PLAN-003 D-3-13): a real
// container kill of nava-worker-interptest after the fake RecordWriter
// committed and before ApplyStep completed, then a restart. Temporal retries
// the attempt the dead worker held, and the retry writes the same key, so the
// effect is applied once.
//
// The kill point is P3-22a's pause-after-commit hook: ApplyStep writes a
// marker file after the write returned and blocks; the test kills the
// container while the marker is there. Every write call is one line in
// calls.jsonl with its idempotency key and attempt, and an effect is one
// O_EXCL file per key, both in the interptest-state volume
// (compose.test.yaml), which outlives the killed container. The assertions
// read them through a fresh one-shot container, never from the killed
// process's memory: a count kept there restarts at 0 and reads 1 even after
// a double apply.
//
// Behind its own tag (interpcrash), so the serial `worker` case
// (stack-test-worker) never compiles it; the `interpreter` case runs it
// (make stack-test-interpreter, deploy/compose/test/cases/interpreter.sh).

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

const (
	// applyOnce is a definition whose start step is an update_record, then
	// end: the kill comes on the first activity after NextStep.
	applyOnce = root + "/internal/workflow/interp/testdata/applyonce"
	// stateMount is where the interptest-state volume is mounted.
	stateMount = "/run/interptest/state"
	// crashCaseBudget bounds the case from the start row to the closed run
	// (TICK-351: the whole case <= 90 s).
	crashCaseBudget = 90 * time.Second
)

// crashState is this run's directory in the interptest-state volume: the
// writer's root and the pause directory, as container paths.
type crashState struct {
	base, pause string
}

func newCrashState() crashState {
	base := path.Join(stateMount, "tick-351-"+uuid.NewString())
	return crashState{base: base, pause: path.Join(base, "pause")}
}

// env points the interptest worker's writer and pause hook into the volume.
func (s crashState) env() []string {
	return []string{"NAVA_INTERPTEST_WRITES_DIR=" + s.base, "NAVA_INTERPTEST_PAUSE_DIR=" + s.pause}
}

// containerName is the file name of an interp helper path (PauseMarker,
// ReleaseFile) joined under a container directory, with forward slashes
// whatever the host is.
func containerName(dir string, hostPath func(dir, key string) string, key string) string {
	return path.Join(dir, filepath.Base(hostPath(".", key)))
}

// oneShot runs args in a fresh, removed container of nava-worker-interptest
// (same image and mounts, --no-deps), as user, and returns its stdout. It is
// a process other than the worker, so what it reads is what the volume holds.
func (w *interpWorker) oneShot(ctx context.Context, user string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose", "--profile", interpProfile,
		"run", "--rm", "--no-deps", "-T", "--user", user, "--entrypoint", args[0], interpService}, args[1:]...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), w.env...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return string(out), fmt.Errorf("%w: %s", err, stderr.String())
	}
	return string(out), nil
}

// containerState is the interptest worker container's id, whether it runs,
// its exit code and when it last started.
type containerState struct {
	id, running, exitCode, startedAt, health string
}

func (w *interpWorker) state(ctx context.Context) containerState {
	w.t.Helper()
	id, err := composeIn(ctx, w.env, "ps", "--all", "--quiet", interpService)
	id = strings.TrimSpace(id)
	if err != nil || id == "" || strings.Contains(id, "\n") {
		w.t.Fatalf("compose ps %s: %v %q", interpService, err, id)
	}
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f",
		"{{.State.Running}} {{.State.ExitCode}} {{.State.StartedAt}} {{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}", id).CombinedOutput()
	f := strings.Fields(string(out))
	if err != nil || len(f) != 4 {
		w.t.Fatalf("docker inspect %s: %v %q", id, err, out)
	}
	return containerState{id: id, running: f[0], exitCode: f[1], startedAt: f[2], health: f[3]}
}

// TestInterpreterCrashAppliesOnce: the worker is killed after the update
// committed and before ApplyStep completed, then restarted. The run
// completes, and the storage outside the killed process holds at least two
// calls (attempt 1 and a retry), all with the one key wf:<workflow id>:<step
// id>, and exactly one applied effect.
func TestInterpreterCrashAppliesOnce(t *testing.T) {
	awaitReady(t)
	st := newCrashState()
	w := startInterpWorker(t, st.env()...)
	clients := newInterpClients(t)
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
	defer cancel()

	// The volume's root is root's; this run's directories are the worker
	// user's (uid 10001), and are removed when the test ends.
	if out, err := w.oneShot(ctx, "0", "sh", "-c", fmt.Sprintf("mkdir -p %s && chown -R 10001:10001 %s", st.pause, st.base)); err != nil {
		t.Fatalf("create %s: %v\n%s", st.base, err, out)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if out, err := w.oneShot(ctx, "0", "rm", "-rf", st.base); err != nil {
			t.Errorf("remove %s: %v\n%s", st.base, err, out)
		}
	})

	r := newInterpRow(t)
	version := uuid.New()
	w.installFrom(applyOnce, r.tenant, version)
	w.putRecord(r.tenant, r.record, map[string]any{})
	f, err := os.ReadFile(applyOnce + ".ids.json")
	if err != nil {
		t.Fatal(err)
	}
	var ids map[string]uuid.UUID
	if err := json.Unmarshal(f, &ids); err != nil {
		t.Fatal(err)
	}
	key := interp.IdempotencyKey(r.workflowID(t), ids["apply"])
	marker := containerName(st.pause, interp.PauseMarker, key)

	begin := time.Now()
	r.enqueue(t, version, r.record)
	r.dispatched(t)

	// 1. ApplyStep committed and is blocked after the commit.
	deadline := time.Now().Add(60 * time.Second)
	for {
		if _, err := composeIn(ctx, w.env, "exec", "-T", interpService, "test", "-f", marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			logs, _ := composeIn(ctx, w.env, "logs", "--no-color", "--tail", "100", interpService)
			t.Fatalf("no pause marker %s within 60 s: ApplyStep never reached the pause after commit\n%s", marker, logs)
		}
		time.Sleep(200 * time.Millisecond)
	}
	before := w.state(ctx)

	// 2. A real container kill while it is blocked.
	if out, err := composeIn(ctx, w.env, "kill", "-s", "SIGKILL", interpService); err != nil {
		t.Fatalf("compose kill %s: %v\n%s", interpService, err, out)
	}
	killed := w.state(ctx)
	if killed.running != "false" || killed.exitCode != "137" {
		t.Fatalf("after the kill: running %s exit %s, want false 137 (SIGKILL)", killed.running, killed.exitCode)
	}

	// The killed attempt's call survived the kill, in storage the killed
	// process no longer holds.
	if calls := w.readCalls(ctx, t, st); len(calls) != 1 || calls[0].Key != key || calls[0].Attempt != 1 || !calls[0].Applied {
		t.Fatalf("calls after the kill %+v, want attempt 1 of %s, applied", calls, key)
	}

	// 3. Release the hook for the retry, then restart the same container.
	release := containerName(st.pause, interp.ReleaseFile, key)
	if out, err := w.oneShot(ctx, "10001", "touch", release); err != nil {
		t.Fatalf("create %s: %v\n%s", release, err, out)
	}
	if out, err := composeIn(ctx, w.env, "start", interpService); err != nil {
		t.Fatalf("compose start %s: %v\n%s", interpService, err, out)
	}
	restarted := w.state(ctx)
	for restarted.health != "healthy" {
		if ctx.Err() != nil {
			t.Fatalf("%s not healthy after the restart: %+v", interpService, restarted)
		}
		time.Sleep(time.Second)
		restarted = w.state(ctx)
	}
	restarts := 0
	if restarted.id == before.id && restarted.startedAt != before.startedAt {
		restarts = 1
	}
	if restarts != 1 {
		t.Fatalf("worker container before %+v, after %+v: want the same container started again", before, restarted)
	}

	// 4. The retry completes the run.
	if err := clients.result(t, r.workflowID(t)); err != nil {
		t.Fatalf("nava.interpreter %s after the restart: %v", r.workflowID(t), err)
	}
	elapsed := time.Since(begin)

	// 5. The out-of-process record.
	calls := w.readCalls(ctx, t, st)
	keys := map[string]bool{}
	maxAttempt, appliedCalls := int32(0), 0
	for _, c := range calls {
		keys[c.Key] = true
		maxAttempt = max(maxAttempt, c.Attempt)
		if c.Applied {
			appliedCalls++
		}
	}
	effects := w.readEffects(ctx, t, st, key)
	t.Logf("kill evidence: container %s exit %s, restarts %d (started %s, then %s); calls %d, attempts up to %d; applied effects %d; %v from start row to closed run",
		before.id[:12], killed.exitCode, restarts, before.startedAt, restarted.startedAt, len(calls), maxAttempt, effects, elapsed.Round(time.Second))

	if len(calls) < 2 {
		t.Errorf("calls %d, want >= 2: the killed attempt and its retry (%+v)", len(calls), calls)
	}
	if maxAttempt < 2 {
		t.Errorf("highest attempt %d, want > 1: the retry ran after the kill", maxAttempt)
	}
	if len(keys) != 1 || !keys[key] {
		t.Errorf("distinct keys %v, want exactly {%s}", keys, key)
	}
	if effects != 1 || appliedCalls != 1 {
		t.Errorf("applied effects %d (calls that applied %d), want exactly 1", effects, appliedCalls)
	}
	if got := psql(t, fmt.Sprintf(`SELECT closed_at IS NOT NULL FROM workflow_ref WHERE tenant_id = '%s' AND outbox_id = '%s';`, r.tenant, r.start)); got != "t" {
		t.Errorf("workflow_ref closed: %q, want t", got)
	}
	// TICK-351 contract: the whole case within 90 s. ApplyStep's
	// StartToCloseTimeout (interp.activityStartToClose, 1 min) sets most of
	// it: the server retries the dead worker's attempt only when it expires.
	if elapsed > crashCaseBudget {
		t.Errorf("start row to closed run took %v, want <= %v", elapsed.Round(time.Second), crashCaseBudget)
	}
}

// readCalls reads this run's calls.jsonl from the volume, in a fresh
// container.
func (w *interpWorker) readCalls(ctx context.Context, t *testing.T, st crashState) []interptest.Call {
	t.Helper()
	file := path.Join(st.base, "writes", "calls.jsonl")
	out, err := w.oneShot(ctx, "10001", "sh", "-c", "cat "+file+" 2>/dev/null || true")
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	var calls []interptest.Call
	for line := range strings.Lines(out) {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var c interptest.Call
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			t.Fatalf("%s: %v: %q", file, err, line)
		}
		calls = append(calls, c)
	}
	return calls
}

// readEffects counts the applied-effect files in the volume that hold key,
// in a fresh container.
func (w *interpWorker) readEffects(ctx context.Context, t *testing.T, st crashState, key string) int {
	t.Helper()
	dir := path.Join(st.base, "writes", "applied")
	out, err := w.oneShot(ctx, "10001", "sh", "-c", `for f in `+dir+`/*.json; do [ -f "$f" ] && { cat "$f"; echo; }; done; true`)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	n := 0
	for line := range strings.Lines(out) {
		if strings.TrimSpace(line) == key {
			n++
		}
	}
	return n
}
