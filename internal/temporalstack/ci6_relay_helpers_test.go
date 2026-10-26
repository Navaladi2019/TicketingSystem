//go:build temporalstack

package temporalstack

// Helpers for the CI-6 relay and worker cases and CI-7 (PLAN-001 T32b). The
// relay under test is the stack's own nava-relay service: the test writes
// outbox rows the way outbox.Enqueue does, as the superuser inside the
// postgres container (the database port is never published), and the service
// claims them as nava_relay and dispatches them to the compose Temporal with
// its nava-relay certificate. A guarded worker in the test runs the workflow.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/relay"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// relayType is the lowercase workflow type on the D9 allow-list
// (temporal-server/authz/policy.go) that an outbox kind can name. The signal
// names below are outbox kinds too.
const (
	relayType   = "nava.smoke.relay"
	sigNote     = "ci6.note"
	sigRollover = "ci6.rollover"
	sigDone     = "ci6.done"
	// The CI-7 probes (TICK-216): each hands record data to one path into
	// the history, which the guard must refuse.
	sigCI7Activity = "ci7.activity"
	sigCI7Child    = "ci7.child"
	sigCI7Rollover = "ci7.rollover"
	updCI7         = "ci7.update"
	// ci7ProbeActivity is not registered anywhere: the CI-7 case only needs
	// its scheduling to reach (or not reach) the history.
	ci7ProbeActivity = "ci7.probe"
)

// relayRefs is relayProbe's own state in workerguard.RunInput.Refs: the
// applied order, outbox ids only.
type relayRefs struct {
	Order []string `json:"order"`
}

// relayProbe is the workflow the relay starts (type relayType). Its input is
// a workerguard.RunInput: the relay's outbox.Delivery on the first run, plus
// the applied-signal set and the applied order carried across continue-as-new.
//   - ci6.note applies its outbox id once (I7) and records the order;
//   - ci6.rollover drains pending notes and continues as new;
//   - ci7.activity schedules ci7.probe with a record-data argument (CI-7);
//   - ci7.child starts a same-tenant child with a record-data argument;
//   - ci7.rollover continues as new with record data in Refs;
//   - update ci7.update takes a map, which may hold record data;
//   - ci6.done closes workflow_ref (the nava-worker service runs that
//     activity as nava_worker) and returns the applied ids, comma-joined.
//
// The CI-7 outcomes do not matter; whether the record data reached the
// history does.
func relayProbe(ctx workflow.Context, in workerguard.RunInput) (string, error) {
	var refs relayRefs
	if in.Refs != nil {
		if err := json.Unmarshal(in.Refs, &refs); err != nil {
			return "", err
		}
	}
	if err := workflow.SetUpdateHandler(ctx, updCI7, func(workflow.Context, map[string]string) error { return nil }); err != nil {
		return "", err
	}
	notes := workflow.GetSignalChannel(ctx, sigNote)
	apply := func(c workflow.ReceiveChannel) {
		var d outbox.Delivery
		c.Receive(ctx, &d)
		if in.Seen.TryApply(ctx, d.OutboxID) {
			refs.Order = append(refs.Order, d.OutboxID.String())
		}
	}
	title := map[string]string{"title": ci7Marker}
	for {
		var next string
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(notes, func(c workflow.ReceiveChannel, _ bool) { apply(c) })
		for _, name := range []string{sigRollover, sigDone, sigCI7Activity, sigCI7Child, sigCI7Rollover} {
			sel.AddReceive(workflow.GetSignalChannel(ctx, name), func(c workflow.ReceiveChannel, _ bool) {
				c.Receive(ctx, nil)
				next = name
			})
		}
		sel.Select(ctx)
		switch next {
		case sigRollover:
			for notes.Len() > 0 {
				apply(notes)
			}
			b, err := json.Marshal(refs)
			if err != nil {
				return "", err
			}
			in.Refs = b
			return "", workflow.NewContinueAsNewError(ctx, relayType, in)
		case sigCI7Activity:
			actx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: 5 * time.Second,
				RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
			})
			_ = workflow.ExecuteActivity(actx, ci7ProbeActivity, title).Get(ctx, nil)
		case sigCI7Child:
			id, err := workerguard.ChildWorkflowID(ctx, "probe")
			if err != nil {
				return "", err
			}
			cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: id, TaskQueue: taskQueue})
			_ = workflow.ExecuteChildWorkflow(cctx, relayType, title).Get(ctx, nil)
		case sigCI7Rollover:
			b, err := json.Marshal(title)
			if err != nil {
				return "", err
			}
			in.Refs = b
			return "", workflow.NewContinueAsNewError(ctx, relayType, in)
		case sigDone:
			if err := workerguard.CloseWorkflowRef(ctx); err != nil {
				return "", err
			}
			return strings.Join(refs.Order, ","), nil
		}
	}
}

// ci7Marker is record data: a ticket title a tenant typed. It must never be
// in a Temporal history (ADR-001 R1, CI-7).
const ci7Marker = "CI7-RECORD-DATA: printer on fire in room 4"

// relayStack starts a guarded worker running relayProbe and returns clients
// set up as the platform sets them up (ConfigureClient), plus the guard log.
// The worker runs workflows only; the workflow_ref activity runs in the
// nava-worker service.
type relayStack struct {
	relay, core, system client.Client
	logs                *lockedBuffer
}

func newRelayStack(t *testing.T) relayStack {
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
	logs := &lockedBuffer{}
	g, err := workerguard.New(slog.New(slog.NewJSONHandler(logs, nil)), tel.MeterProvider)
	if err != nil {
		t.Fatal(err)
	}
	w := worker.New(dial(t, "nava-worker", setup), taskQueue,
		workerguard.WorkerOptions(worker.Options{LocalActivityWorkerOnly: true}, g))
	w.RegisterWorkflowWithOptions(relayProbe, workflow.RegisterOptions{Name: relayType})
	if err := w.Start(); err != nil {
		t.Fatalf("worker start: %v", err)
	}
	t.Cleanup(w.Stop)
	return relayStack{relay: dial(t, "nava-relay", setup), core: dial(t, "nava-core", setup),
		system: dial(t, "temporal-system"), logs: logs}
}

// guardAlerts counts the guard's alert lines for workflowID and guard.
func guardAlerts(logs *lockedBuffer, workflowID, guard, outcome string) int {
	n := 0
	for line := range strings.Lines(logs.String()) {
		if strings.Contains(line, `"alert":true`) && strings.Contains(line, `"guard":"`+guard+`"`) &&
			strings.Contains(line, `"outcome":"`+outcome+`"`) && strings.Contains(line, workflowID) {
			n++
		}
	}
	return n
}

// --- the stack's PostgreSQL ---------------------------------------------------

// psqlSuper runs sql in one transaction as the superuser over the Unix socket
// inside the postgres container (the only path db/pg_hba.conf opens to it) and
// returns the unaligned, tuples-only output.
func psqlSuper(t *testing.T, sql string) string {
	t.Helper()
	out, err := psqlSuperErr(t, sql)
	if err != nil {
		t.Fatalf("psql: %v\n%s\n-- sql:\n%s", err, out, sql)
	}
	return out
}

// psqlSuperErr is psqlSuper for SQL that may fail: it returns psql's output,
// the error text included, and its error.
func psqlSuperErr(t *testing.T, sql string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "docker", "exec", "-i", containerOf(t, "postgres"), "sh", "-c",
		`PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -t -A -1 -v ON_ERROR_STOP=1 -U pgadmin -d nava -f -`)
	cmd.Stdin = strings.NewReader(sql)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// target is one record of one tenant and the relay-started workflow on it.
// Every id and kind in the SQL below is a UUID or a constant, never free text.
type target struct {
	tenant, record, start uuid.UUID
}

func newID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// newTarget creates an active tenant and a record, and leaves every pending
// row of the tenant dead when the test ends, so no row outlives it.
func newTarget(t *testing.T) *target {
	t.Helper()
	tg := &target{tenant: uuid.New(), record: uuid.New(), start: newID(t)}
	psqlSuper(t, fmt.Sprintf(`INSERT INTO tenants (id, state) VALUES ('%s', 'active');
INSERT INTO records (tenant_id, id) VALUES ('%[1]s', '%s');`, tg.tenant, tg.record))
	t.Cleanup(func() {
		psqlSuper(t, fmt.Sprintf(`UPDATE outbox SET dead_at = now(), last_error = 'CI-6 cleanup'
WHERE tenant_id = '%s' AND dispatched_at IS NULL AND dead_at IS NULL;`, tg.tenant))
	})
	return tg
}

// workflowID is the id the relay gives the start.
func (tg *target) workflowID() string { return relayWorkflowID(tg.tenant, tg.start) }

// relayWorkflowID is the id relay.WorkflowID, the single source the relay
// uses (docs/architecture/outbox-relay.md), gives a relayType start row of
// tenant with outbox id start. It panics if the relay would refuse the row:
// that is a broken fixture, not a test result.
func relayWorkflowID(tenant, start uuid.UUID) string {
	id, ok := relay.WorkflowID(relay.Claimed{ID: start, TenantID: tenant, Kind: relayType, TargetKey: relayType + ":" + start.String()})
	if !ok {
		panic("relay.WorkflowID refused the " + relayType + " fixture")
	}
	return id
}

// refs is a references-only payload (outbox.ValidatePayload).
func (tg *target) refs() string { return `{"record_id":"` + tg.record.String() + `"}` }

// rowSQL is outbox.Enqueue's write: bump the record's version and insert the
// row with it as target_seq.
func (tg *target) rowSQL(id uuid.UUID, kind, payload string) string {
	return fmt.Sprintf(`WITH v AS (UPDATE records SET version = version + 1 WHERE tenant_id = '%[1]s' AND id = '%[2]s' RETURNING version)
INSERT INTO outbox (id, tenant_id, kind, kind_version, dedupe_key, target_key, target_seq, payload)
SELECT '%[3]s', '%[1]s', '%[4]s', 1, '%[3]s', '%[5]s', version, '%[6]s'::jsonb FROM v;
`, tg.tenant, tg.record, id, kind, relayType+":"+tg.start.String(), payload)
}

// startSQL is the start row with the given payload, and its workflow_ref.
func (tg *target) startSQL(payload string) string {
	return tg.rowSQL(tg.start, relayType, payload) + fmt.Sprintf(
		"INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ('%s', '%s', '%s', '%s');\n",
		tg.tenant, tg.start, tg.record, relayType)
}

// signalSQL is a signal row named kind for the start; it returns its id too.
func (tg *target) signalSQL(t *testing.T, kind string) (uuid.UUID, string) {
	t.Helper()
	id := newID(t)
	return id, tg.rowSQL(id, kind, tg.refs())
}

// row is the dispatch state of one outbox row.
type row struct {
	attempts         int
	dispatched, dead bool
	lastError        string
}

func outboxRow(t *testing.T, id uuid.UUID) row {
	t.Helper()
	out := psqlSuper(t, fmt.Sprintf(`SELECT attempts, dispatched_at IS NOT NULL, dead_at IS NOT NULL, coalesce(last_error, '')
FROM outbox WHERE id = '%s';`, id))
	f := strings.SplitN(out, "|", 4)
	if len(f) != 4 {
		t.Fatalf("outbox row %s: %q", id, out)
	}
	n, err := strconv.Atoi(f[0])
	if err != nil {
		t.Fatalf("outbox row %s attempts %q", id, f[0])
	}
	return row{attempts: n, dispatched: f[1] == "t", dead: f[2] == "t", lastError: f[3]}
}

// awaitSettled waits up to 60 s for the relay to mark the row dispatched or dead.
func awaitSettled(t *testing.T, id uuid.UUID) row {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		r := outboxRow(t, id)
		if r.dispatched || r.dead {
			return r
		}
		if time.Now().After(deadline) {
			t.Fatalf("outbox row %s not settled within 60 s: %+v", id, r)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// redeliver makes a settled row pending and available again: what the relay
// sees after it crashed between Temporal accepting the call and
// MarkDispatched, once the lease ended.
func redeliver(t *testing.T, id uuid.UUID) {
	t.Helper()
	psqlSuper(t, fmt.Sprintf(`UPDATE outbox SET dispatched_at = NULL, available_at = now() WHERE id = '%s';`, id))
}

// result waits for the workflow's result, read by nava-core (the relay may
// not read history).
//
// It first waits for the relay to start the workflow: the rows were only
// written, and GetWorkflow on an id that does not exist yet is NotFound at once
// (the first stack run of T32b failed so, while the relay started each one
// shortly after).
func (s relayStack) result(ctx context.Context, t *testing.T, id string) string {
	t.Helper()
	s.awaitStarted(ctx, t, id)
	var out string
	if err := s.core.GetWorkflow(ctx, id, "").Get(ctx, &out); err != nil {
		t.Fatalf("result of %s: %v", id, err)
	}
	return out
}

// terminateAtEnd terminates the workflow, if still open, when the test ends.
func (s relayStack) terminateAtEnd(t *testing.T, id string) {
	t.Cleanup(func() { _ = s.system.TerminateWorkflow(context.Background(), id, "", "CI-6 cleanup") })
}

// awaitStarted waits until the workflow exists, and returns its run id.
func (s relayStack) awaitStarted(ctx context.Context, t *testing.T, id string) string {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		desc, err := s.core.DescribeWorkflowExecution(ctx, id, "")
		if err == nil {
			return desc.GetWorkflowExecutionInfo().GetExecution().GetRunId()
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s not started within 60 s: %v", id, err)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// --- the relay processes -------------------------------------------------------

// relayLogHas reports whether the stack's nava-relay logged a line since
// `since` holding every one of parts.
func relayLogHas(t *testing.T, since string, parts ...string) bool {
	t.Helper()
	out := docker(t, "logs", "--since", since, containerOf(t, "nava-relay"))
	for line := range strings.Lines(out) {
		ok := true
		for _, p := range parts {
			ok = ok && strings.Contains(line, p)
		}
		if ok {
			return true
		}
	}
	return false
}

// relayMetric sums the samples of the Prometheus series name whose labels
// include every one of labels, from the relay container's admin listener.
func relayMetric(t *testing.T, container, name string, labels ...string) float64 {
	t.Helper()
	out := docker(t, "exec", container, "wget", "-q", "-O", "-", "http://127.0.0.1:9090/metrics")
	var sum float64
	for line := range strings.Lines(out) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, name+"{") && !strings.HasPrefix(line, name+" ") {
			continue
		}
		ok := true
		for _, l := range labels {
			ok = ok && strings.Contains(line, l)
		}
		if !ok {
			continue
		}
		v, err := strconv.ParseFloat(line[strings.LastIndexByte(line, ' ')+1:], 64)
		if err != nil {
			t.Fatalf("%s: %q", name, line)
		}
		sum += v
	}
	return sum
}

// extraRelay runs one more nava-relay beside the stack's: the same image,
// environment, certificate mounts and network, so it claims from the same
// outbox and dispatches to the same Temporal. It is removed when the test
// ends. It returns the container id once /readyz answers.
func extraRelay(t *testing.T) string {
	t.Helper()
	svc := containerOf(t, "nava-relay")
	image := strings.TrimSpace(docker(t, "inspect", "-f", "{{.Config.Image}}", svc))
	args := []string{"run", "-d", "--network", networkOf(t, "internal"), "--volumes-from", svc + ":ro"}
	for env := range strings.Lines(docker(t, "inspect", "-f", "{{range .Config.Env}}{{println .}}{{end}}", svc)) {
		if env = strings.TrimSpace(env); env != "" {
			args = append(args, "-e", env)
		}
	}
	id := strings.TrimSpace(docker(t, append(args, image)...))
	t.Cleanup(func() { _, _ = dockerOut("rm", "-f", id) })
	deadline := time.Now().Add(60 * time.Second)
	for {
		if _, err := dockerOut("exec", id, "wget", "-q", "-O", "/dev/null", "http://127.0.0.1:9090/readyz"); err == nil {
			return id
		}
		if time.Now().After(deadline) {
			logs, _ := dockerOut("logs", id)
			t.Fatalf("extra nava-relay not ready within 60 s:\n%s", logs)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func dockerOut(args ...string) (string, error) {
	out, err := exec.CommandContext(context.Background(), "docker", args...).CombinedOutput()
	if err != nil {
		return string(out), errors.Join(err, errors.New(strings.TrimSpace(string(out))))
	}
	return string(out), nil
}
