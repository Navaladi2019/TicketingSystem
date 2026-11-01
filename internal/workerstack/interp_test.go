//go:build workerstack

package workerstack

// The interpreter stack cases (TICK-367, P3-22b): the stack's nava-relay
// starts nava.interpreter from an outbox row under the real Authorizer, and
// the interptest stack worker (nava-worker-interptest, compose.test.yaml)
// runs it with the S11a fakes over a read-only fixture mount.
//
// nava-worker-interptest polls workflow and activity tasks on nava-tq. It is
// up only inside startInterpWorker's test, in the serial `worker` case: the
// parallel group (the temporalstack suite, whose workers register their own
// test bodies as nava.interpreter, TICK-374) has ended by then, and demo_outbox (the relay's
// nava.smoke.relay starts) sorts after it (run-cases.sh, PARALLEL=1). Its
// cleanup removes the container and waits until it is gone, failed test or
// not, so no interpreter poller outlives the case.
//
// The always-on nava-worker also polls activity tasks on nava-tq and does not
// register NextStep or ApplyStep: an attempt it took would fail as
// ActivityNotRegisteredError (retryable), spend one of the bounded attempts
// and add its backoff. So startInterpWorker stops it for the test and its
// cleanup starts it again and waits until it is healthy, failed test or not
// (Technical Lead, 2026-11-01): nava-worker-interptest is the only nava-tq
// poller while the case runs.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/relay"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

const (
	interpService = "nava-worker-interptest"
	// alwaysOnWorker is the production-image worker; the interpreter cases
	// stop it while they run.
	alwaysOnWorker = service
	interpProfile  = "interptest"
	// fixtureV1 is P3-20's fixture-v1. With priority "low" it runs a 5 s
	// timer, takes the condition's else edge and ends.
	fixtureV1 = root + "/internal/workflow/def/testdata/fixture-v1"
)

// composeIn runs `docker compose --profile interptest` from the repository
// root with env added to the environment, and returns stdout and stderr.
func composeIn(ctx context.Context, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose", "--profile", interpProfile}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// interpWorker is a running nava-worker-interptest over a fixture directory.
// env is the compose environment it was started with; a later compose call
// on it passes the same, so compose never recreates it.
type interpWorker struct {
	t   *testing.T
	dir string
	env []string
}

// startInterpWorker stops the always-on nava-worker, builds and starts
// nava-worker-interptest with a fresh fixture directory mounted read-only,
// and waits for its /readyz healthcheck. Its cleanups remove
// nava-worker-interptest, wait until it has exited, and start nava-worker
// again until it is healthy. extraEnv is added to the compose environment
// (the crash case's writes and pause directories).
func startInterpWorker(t *testing.T, extraEnv ...string) *interpWorker {
	t.Helper()
	dir := t.TempDir()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	env := append([]string{"NAVA_INTERPTEST_FIXTURES=" + abs}, extraEnv...)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	// Cleanups run last-registered first: nava-worker-interptest is removed,
	// then the always-on nava-worker is started again. Both are registered
	// before what they undo, so a failed or half-done step is undone too.
	t.Cleanup(func() { restartAlwaysOnWorker(t, env) })
	if out, err := composeIn(ctx, env, "stop", alwaysOnWorker); err != nil {
		t.Fatalf("compose stop %s: %v\n%s", alwaysOnWorker, err, out)
	}
	t.Cleanup(func() { stopInterpWorker(t, env) })
	// --no-deps: the stack is up; --wait: the service's /readyz healthcheck.
	if out, err := composeIn(ctx, env, "up", "-d", "--build", "--no-deps", "--wait", "--wait-timeout", "180", interpService); err != nil {
		logs, _ := composeIn(ctx, env, "logs", "--no-color", "--tail", "100", interpService)
		t.Fatalf("compose up %s: %v\n%s\nlogs:\n%s", interpService, err, out, logs)
	}
	return &interpWorker{t: t, dir: abs, env: env}
}

// restartAlwaysOnWorker starts the always-on nava-worker again and waits for
// its healthcheck (/readyz), so the next test and the demo_outbox case find
// it ready. It runs on a fresh context: the test's may be done.
func restartAlwaysOnWorker(t *testing.T, env []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if out, err := composeIn(ctx, env, "up", "-d", "--no-deps", "--wait", "--wait-timeout", "180", alwaysOnWorker); err != nil {
		t.Errorf("compose up %s after the interpreter case: %v\n%s", alwaysOnWorker, err, out)
	}
}

// stopInterpWorker removes the service's container and waits until compose
// lists none, so no nava-tq poller is left behind. It runs on a fresh context:
// the test's may be done.
func stopInterpWorker(t *testing.T, env []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if out, err := composeIn(ctx, env, "rm", "--stop", "--force", interpService); err != nil {
		t.Errorf("compose rm %s: %v\n%s", interpService, err, out)
	}
	for {
		out, err := composeIn(ctx, env, "ps", "--all", "--quiet", interpService)
		if err == nil && strings.TrimSpace(out) == "" {
			return
		}
		if ctx.Err() != nil {
			t.Errorf("%s still listed after rm: %v\n%s", interpService, err, out)
			return
		}
		time.Sleep(time.Second)
	}
}

// install writes fixture-v1 as version of tenant.
func (w *interpWorker) install(tenant, version uuid.UUID) {
	w.t.Helper()
	w.installFrom(fixtureV1, tenant, version)
}

// installFrom writes the fixture at path (path.json and path.ids.json) as
// version of tenant.
func (w *interpWorker) installFrom(path string, tenant, version uuid.UUID) {
	w.t.Helper()
	d, err := os.ReadFile(path + ".json")
	if err != nil {
		w.t.Fatal(err)
	}
	ids, err := os.ReadFile(path + ".ids.json")
	if err != nil {
		w.t.Fatal(err)
	}
	if err := interptest.PutDefinition(w.dir, tenant, version, d, ids); err != nil {
		w.t.Fatal(err)
	}
	w.readable()
}

func (w *interpWorker) putRecord(tenant, record uuid.UUID, data map[string]any) {
	w.t.Helper()
	if err := interptest.PutRecord(w.dir, tenant, record, data); err != nil {
		w.t.Fatal(err)
	}
	w.readable()
}

// readable opens the tree to the container's user (uid 10001), which is not
// the test's: interptest writes 0600 files and 0750 directories, and
// t.TempDir is 0700. The mount itself is read-only.
func (w *interpWorker) readable() {
	w.t.Helper()
	err := filepath.WalkDir(w.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if d.IsDir() {
			mode = 0o755
		}
		return os.Chmod(path, mode)
	})
	if err != nil {
		w.t.Fatal(err)
	}
}

// psql runs sql in one transaction as the superuser over the Unix socket in
// the postgres container (the database port is never published), as the
// temporalstack relay cases do, and returns the unaligned tuples.
func psql(t *testing.T, sql string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "compose", "exec", "-T", "postgres", "sh", "-c",
		`PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -t -A -1 -v ON_ERROR_STOP=1 -U pgadmin -d nava -f -`)
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psql: %v\n%s\n-- sql:\n%s", err, out, sql)
	}
	return strings.TrimSpace(string(out))
}

// interpRow is one tenant's record and the interpreter start row the test
// writes for it. Every value in the SQL is a UUID or a constant.
type interpRow struct {
	tenant, record, start uuid.UUID
}

// newInterpRow creates an active tenant and a record (records table), and
// leaves the tenant's pending rows dead when the test ends.
func newInterpRow(t *testing.T) *interpRow {
	t.Helper()
	start, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	r := &interpRow{tenant: uuid.New(), record: uuid.New(), start: start}
	psql(t, fmt.Sprintf(`INSERT INTO tenants (id, state) VALUES ('%s', 'active');
INSERT INTO records (tenant_id, id) VALUES ('%[1]s', '%s');`, r.tenant, r.record))
	t.Cleanup(func() {
		psql(t, fmt.Sprintf(`UPDATE outbox SET dead_at = now(), last_error = 'TICK-367 cleanup'
WHERE tenant_id = '%s' AND dispatched_at IS NULL AND dead_at IS NULL;`, r.tenant))
	})
	return r
}

// targetKey is the outbox start form for the interpreter (D-3-13 as amended
// 2026-11-01): the kind and the row's own id.
func (r *interpRow) targetKey() string { return interp.WorkflowType + ":" + r.start.String() }

// workflowID is the id the relay gives the start, from relay.WorkflowID.
func (r *interpRow) workflowID(t *testing.T) string {
	t.Helper()
	c := relay.Claimed{ID: r.start, TenantID: r.tenant, Kind: interp.WorkflowType, TargetKey: r.targetKey()}
	if !c.IsStart() {
		t.Fatalf("the relay does not read %s as a start", r.targetKey())
	}
	id, ok := relay.WorkflowID(c)
	if !ok {
		t.Fatalf("relay.WorkflowID refused %s", r.targetKey())
	}
	return id
}

// enqueue writes what outbox.Enqueue writes for a TargetStart of
// nava.interpreter: the record's version bump, the row with payload
// {definition_version_id, record_id} (D-3-12), and its workflow_ref.
func (r *interpRow) enqueue(t *testing.T, version, record uuid.UUID) {
	t.Helper()
	payload, err := json.Marshal(map[string]uuid.UUID{"definition_version_id": version, "record_id": record})
	if err != nil {
		t.Fatal(err)
	}
	psql(t, fmt.Sprintf(`WITH v AS (UPDATE records SET version = version + 1 WHERE tenant_id = '%[1]s' AND id = '%[2]s' RETURNING version)
INSERT INTO outbox (id, tenant_id, kind, kind_version, dedupe_key, target_key, target_seq, payload)
SELECT '%[3]s', '%[1]s', '%[4]s', 1, '%[3]s', '%[5]s', version, '%[6]s'::jsonb FROM v;
INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ('%[1]s', '%[3]s', '%[2]s', '%[4]s');`,
		r.tenant, r.record, r.start, interp.WorkflowType, r.targetKey(), payload))
}

// dispatched waits up to 60 s for the relay to mark the start row
// dispatched; a dead row fails the test with its last_error.
func (r *interpRow) dispatched(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		out := psql(t, fmt.Sprintf(`SELECT dispatched_at IS NOT NULL, dead_at IS NOT NULL, coalesce(last_error, '') FROM outbox WHERE id = '%s';`, r.start))
		f := strings.SplitN(out, "|", 3)
		if len(f) == 3 && f[0] == "t" {
			return
		}
		if len(f) == 3 && f[1] == "t" {
			t.Fatalf("the relay marked the start row dead: %s", f[2])
		}
		if time.Now().After(deadline) {
			t.Fatalf("start row %s not dispatched within 60 s: %q", r.start, out)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// interpClients are nava-core's client (it reads results; the relay may not
// read history) and temporal-system's, for cleanup.
type interpClients struct{ core, system client.Client }

func newInterpClients(t *testing.T) interpClients {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	tt, err := tel.NewTemporal(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	setup := func(o *client.Options) { workerguard.ConfigureClient(o, tt) }
	return interpClients{core: dial(t, "nava-core", setup), system: dial(t, "temporal-system")}
}

// result waits for the run to close and returns its error. It terminates the
// workflow at the end of the test if it is still open.
func (c interpClients) result(t *testing.T, id string) error {
	t.Helper()
	t.Cleanup(func() { _ = c.system.TerminateWorkflow(context.Background(), id, "", "TICK-367 cleanup") })
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	return c.core.GetWorkflow(ctx, id, "").Get(ctx, nil)
}

// TestInterpreterStartedByTheRelayCompletes: an outbox start row of kind
// nava.interpreter is dispatched by the stack's nava-relay under the real
// Authorizer, and nava-worker-interptest runs it to the end step, whose
// CloseWorkflowRef closes the row's workflow_ref.
func TestInterpreterStartedByTheRelayCompletes(t *testing.T) {
	awaitReady(t)
	w := startInterpWorker(t)
	clients := newInterpClients(t)

	r := newInterpRow(t)
	version := uuid.New()
	w.install(r.tenant, version)
	w.putRecord(r.tenant, r.record, map[string]any{"priority": "low"})
	r.enqueue(t, version, r.record)
	r.dispatched(t)

	if err := clients.result(t, r.workflowID(t)); err != nil {
		t.Fatalf("nava.interpreter %s: %v", r.workflowID(t), err)
	}
	if got := psql(t, fmt.Sprintf(`SELECT closed_at IS NOT NULL FROM workflow_ref WHERE tenant_id = '%s' AND outbox_id = '%s';`, r.tenant, r.start)); got != "t" {
		t.Errorf("workflow_ref closed: %q, want t", got)
	}
}

// TestInterpreterCrossTenantIsNotFound: a tenant A start row naming tenant
// B's definition version, or B's record, fails as not found. The tenant is
// the row's, which the relay puts in the nava-tenant header; nothing in the
// payload names one.
func TestInterpreterCrossTenantIsNotFound(t *testing.T) {
	awaitReady(t)
	w := startInterpWorker(t)
	clients := newInterpClients(t)

	for _, tc := range []struct {
		name string
		// setup installs the fixtures for tenants a and b and returns the
		// version and record ids the start payload names.
		setup func(a, b uuid.UUID) (version, record uuid.UUID)
	}{
		{"B's definition version", func(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
			version, record := uuid.New(), uuid.New()
			w.install(b, version)
			w.putRecord(a, record, map[string]any{"priority": "low"})
			return version, record
		}},
		{"B's record", func(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
			version, record := uuid.New(), uuid.New()
			w.install(a, version)
			w.putRecord(b, record, map[string]any{"priority": "low"})
			return version, record
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newInterpRow(t)
			b := uuid.New()
			version, record := tc.setup(a.tenant, b)
			a.enqueue(t, version, record)
			a.dispatched(t)

			err := clients.result(t, a.workflowID(t))
			var app *temporal.ApplicationError
			if !errors.As(err, &app) {
				t.Fatalf("nava.interpreter %s: %v, want a not-found rejection", a.workflowID(t), err)
			}
			want := "workflow rejected: " + string(workerguard.RejectNotFound)
			if app.Type() != workerguard.ErrTypeWorkflowRejected || app.Message() != want {
				t.Errorf("failure %s %q, want %s %q", app.Type(), app.Message(), workerguard.ErrTypeWorkflowRejected, want)
			}
			// Not found must come from the header tenant's lookup, never
			// from a run that read B's data and then failed.
			if got := psql(t, fmt.Sprintf(`SELECT closed_at IS NULL FROM workflow_ref WHERE tenant_id = '%s' AND outbox_id = '%s';`, a.tenant, a.start)); got != "t" {
				t.Errorf("workflow_ref open: %q, want t (the run never reached its end step)", got)
			}
		})
	}
}
