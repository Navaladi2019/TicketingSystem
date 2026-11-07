//go:build integration

package seal_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// CI-A16 (P5-34b) against a real PostgreSQL as nava_worker: the audit.seal
// workflow with the real A1 and A2, faults injected in the test (no docker
// kill).

// workflowRun is one audit.seal run o of tenant. a1 and a2, when set, replace
// the registered activities (fault injection); heartbeats counts A2's.
type workflowRun struct {
	acts       *seal.Activities
	a1         func(context.Context) (seal.Rescheduled, error)
	a2         func(context.Context) error
	heartbeats int
}

func (r *workflowRun) run(t *testing.T, tenant, o uuid.UUID) error {
	t.Helper()
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.SetWorkerOptions(worker.Options{BackgroundActivityContext: asTenant(tenant)})
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: runID(tenant, o)})
	env.SetOnActivityHeartbeatListener(func(info *activity.Info, _ converter.EncodedValues) {
		if info.ActivityType.Name == seal.SealActivity {
			r.heartbeats++
		}
	})
	a1, a2 := r.a1, r.a2
	if a1 == nil {
		a1 = r.acts.Reschedule
	}
	if a2 == nil {
		a2 = r.acts.Seal
	}
	env.RegisterActivityWithOptions(a1, activity.RegisterOptions{Name: seal.RescheduleActivity})
	env.RegisterActivityWithOptions(a2, activity.RegisterOptions{Name: seal.SealActivity})
	seal.RegisterWorkflow(env)
	env.ExecuteWorkflow(seal.WorkflowType, outbox.Delivery{OutboxID: o, KindVersion: 1, Payload: []byte(`{}`)})
	if !env.IsWorkflowCompleted() {
		t.Fatal("the workflow did not complete")
	}
	return env.GetWorkflowError()
}

// lineage seeds a tenant whose head points at run o, and returns both.
func lineage(t *testing.T, e itEnv) (uuid.UUID, uuid.UUID) {
	t.Helper()
	tenant := newTenant(t, e)
	o := uuid.Must(uuid.NewV7())
	mustExec(t, e.super, "UPDATE audit_chain_head SET next_seal_outbox_id = $2 WHERE tenant_id = $1", tenant, o)
	return tenant, o
}

// nextOf is the run the head points at, which must not be o.
func nextOf(t *testing.T, e itEnv, tenant, o uuid.UUID) uuid.UUID {
	t.Helper()
	h := readHead(t, e, tenant)
	if h.next == nil || *h.next == o {
		t.Fatalf("the head points at %v after run %s: A1 scheduled no successor", h.next, o)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND id = $2 AND kind = 'audit.seal'", tenant, *h.next); n != 1 {
		t.Fatalf("no start row for the next run %s", *h.next)
	}
	return *h.next
}

func sealRows(t *testing.T, e itEnv, tenant uuid.UUID) int64 {
	t.Helper()
	return count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND kind = 'audit.seal'", tenant)
}

// CI-A16, kill after A1 and before A2: A1 committed run o′, then the run
// ended with no A2 (the worker is gone; the fault is injected as A2 failing
// before it touches anything). Nothing was sealed, and run o′ seals the batch
// and continues the lineage.
func TestKillBetweenA1AndA2(t *testing.T) {
	e := migrated(t)
	tenant, o := lineage(t, e)
	writeEntries(t, e, tenant, 3)
	acts := &seal.Activities{Pool: e.worker}

	killed := &workflowRun{acts: acts, a2: func(context.Context) error {
		return temporal.NewNonRetryableApplicationError("worker killed", "test.Killed", nil)
	}}
	if err := killed.run(t, tenant, o); err == nil {
		t.Fatal("the killed run completed")
	}
	o2 := nextOf(t, e, tenant, o)
	if got := assertChain(t, e, tenant); got != 0 {
		t.Fatalf("%d sealed by the killed run, want 0", got)
	}

	if err := (&workflowRun{acts: acts}).run(t, tenant, o2); err != nil {
		t.Fatal(err)
	}
	if got := assertChain(t, e, tenant); got != 3 {
		t.Errorf("run o′ sealed %d, want the 3 the killed run left", got)
	}
	nextOf(t, e, tenant, o2)
}

// CI-A16, A1 retried after its commit (TICK-440 review, binding): the first
// A1 attempt commits o′ and then fails as if its worker died before
// reporting. The retry finds the head already at o′ and returns no o′ (stale
// lineage); the workflow still runs A2, which seals the batch. The run
// enqueued exactly one successor, and o′ continues the chain with no gap or
// duplicate seq.
func TestA1RetriedAfterCommitStillSeals(t *testing.T) {
	e := migrated(t)
	tenant, o := lineage(t, e)
	writeEntries(t, e, tenant, 4)
	acts := &seal.Activities{Pool: e.worker}
	before := sealRows(t, e, tenant)

	var results []*uuid.UUID
	r := &workflowRun{acts: acts, a1: func(ctx context.Context) (seal.Rescheduled, error) {
		res, err := acts.Reschedule(ctx)
		results = append(results, res.NextOutboxID)
		if err == nil && activity.GetInfo(ctx).Attempt == 1 {
			return seal.Rescheduled{}, errors.New("worker lost after A1 committed")
		}
		return res, err
	}}
	if err := r.run(t, tenant, o); err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0] == nil || results[1] != nil {
		t.Fatalf("A1 attempts returned %v, want o′ then nothing", results)
	}
	o2 := nextOf(t, e, tenant, o)
	if o2 != *results[0] {
		t.Errorf("the head points at %s, want the first attempt's o′ %s", o2, *results[0])
	}
	if got := sealRows(t, e, tenant) - before; got != 1 {
		t.Errorf("the run enqueued %d successors, want 1", got)
	}
	if got := assertChain(t, e, tenant); got != 4 {
		t.Errorf("%d sealed, want 4: A2 did not run after the stale A1", got)
	}

	writeEntries(t, e, tenant, 2)
	if err := (&workflowRun{acts: acts}).run(t, tenant, o2); err != nil {
		t.Fatal(err)
	}
	if got := assertChain(t, e, tenant); got != 6 {
		t.Errorf("%d sealed after o′, want 6", got)
	}
}

// promMetrics is a seal.Metrics on its own Prometheus registry.
func promMetrics(t *testing.T) (*seal.Metrics, func(name, reason string) float64) {
	t.Helper()
	reg := prometheus.NewRegistry()
	reader, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		t.Fatal(err)
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })
	m, err := seal.NewMetrics(mp)
	if err != nil {
		t.Fatal(err)
	}
	return m, func(name, reason string) float64 {
		mfs, err := reg.Gather()
		if err != nil {
			t.Fatal(err)
		}
		for _, mf := range mfs {
			if mf.GetName() != name {
				continue
			}
			for _, s := range mf.GetMetric() {
				match := reason == ""
				for _, l := range s.GetLabel() {
					if l.GetName() == "reason" && l.GetValue() == reason {
						match = true
					}
				}
				if match && s.GetCounter() != nil {
					return s.GetCounter().GetValue()
				}
				if match && s.GetGauge() != nil {
					return s.GetGauge().GetValue()
				}
			}
		}
		return 0
	}
}

// CI-A16, poison: an entry that cannot be canonicalised fails its A2 batch,
// which uses no seq, and counts in audit_seal_failures_total. The next run
// is still scheduled, because A1 committed first, and meets the same entry:
// it is never skipped, so the tenant's sealing stays blocked and
// audit_unsealed_oldest_seconds passes the 600 s alert.
func TestPoisonBlocksTenantAndAlerts(t *testing.T) {
	e := migrated(t)
	tenant, o := lineage(t, e)
	mustExec(t, e.super, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
		version_after, actor_system, source) VALUES ($1, $2, 'record', 'update', 'record', gen_random_uuid(), $3,
		'system', 'system')`, tenant, uuid.Must(uuid.NewV7()), int64(chain.MaxSafeInteger)+1)
	writeEntries(t, e, tenant, 2)
	var poisonAt time.Time
	if err := e.super.QueryRow(context.Background(), `SELECT min(occurred_at) FROM audit_log WHERE tenant_id = $1`,
		tenant).Scan(&poisonAt); err != nil {
		t.Fatal(err)
	}
	m, metric := promMetrics(t)
	acts := &seal.Activities{Pool: e.worker, Metrics: m}
	headBefore := readHead(t, e, tenant)

	run := o
	var lastAge float64
	for i := 1; i <= 2; i++ {
		err := (&workflowRun{acts: acts}).run(t, tenant, run)
		var app *temporal.ApplicationError
		if !errors.As(err, &app) || app.Type() != seal.ErrTypeUnsealable {
			t.Fatalf("run %d: error %v, want %s", i, err, seal.ErrTypeUnsealable)
		}
		run = nextOf(t, e, tenant, run)
		if got := metric("audit_seal_failures_total", seal.ReasonUnsealable); got != float64(i) {
			t.Errorf("run %d: audit_seal_failures_total{reason=unsealable} = %v, want %d", i, got, i)
		}
		if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND seq IS NOT NULL", tenant); n != 0 {
			t.Errorf("run %d: %d entries sealed past the poison entry, want 0", i, n)
		}
		if h := readHead(t, e, tenant); h.seq != headBefore.seq {
			t.Errorf("run %d: head seq %d, want %d: a seq was used", i, h.seq, headBefore.seq)
		}
		// The gauge is the poison entry's own age (occurred_at cannot be
		// back-dated, so 600 s is TestMetrics' fake clock), and it only
		// grows while the entry blocks the tenant.
		age := metric("audit_unsealed_oldest_seconds", "")
		if upper := time.Since(poisonAt).Seconds(); age <= lastAge || age > upper+1 {
			t.Errorf("run %d: audit_unsealed_oldest_seconds = %v, want in (%v, %v]", i, age, lastAge, upper)
		}
		lastAge = age
		time.Sleep(20 * time.Millisecond)
	}
}

// CI-A16, idle tenant: at the 300 s cap, an idle run schedules its successor
// at most 300 s out (next_seal_at and the start row's available_at agree),
// and that run seals an entry written meanwhile, then drops the interval to
// 10 s.
func TestIdleTenantSealedWithin300s(t *testing.T) {
	e := migrated(t)
	tenant, o := lineage(t, e)
	mustExec(t, e.super, "UPDATE audit_chain_head SET seal_interval_s = $2 WHERE tenant_id = $1", tenant, int32(seal.MaxIntervalSeconds))
	acts := &seal.Activities{Pool: e.worker}

	if err := (&workflowRun{acts: acts}).run(t, tenant, o); err != nil {
		t.Fatal(err)
	}
	o2 := nextOf(t, e, tenant, o)
	var delay float64
	if err := e.super.QueryRow(context.Background(), `SELECT extract(epoch FROM o.available_at - o.created_at)
		FROM outbox o JOIN audit_chain_head h ON h.next_seal_outbox_id = o.id AND h.next_seal_at = o.available_at
		WHERE o.tenant_id = $1 AND o.id = $2`, tenant, o2).Scan(&delay); err != nil {
		t.Fatalf("the start row and next_seal_at disagree: %v", err)
	}
	if delay > seal.MaxIntervalSeconds {
		t.Errorf("idle successor scheduled %v s out, want at most %d", delay, seal.MaxIntervalSeconds)
	}
	if h := readHead(t, e, tenant); h.interval != seal.MaxIntervalSeconds {
		t.Errorf("idle interval %d, want it held at %d", h.interval, seal.MaxIntervalSeconds)
	}

	writeEntries(t, e, tenant, 1)
	if err := (&workflowRun{acts: acts}).run(t, tenant, o2); err != nil {
		t.Fatal(err)
	}
	if got := assertChain(t, e, tenant); got != 1 {
		t.Errorf("%d sealed, want the idle tenant's entry", got)
	}
	if h := readHead(t, e, tenant); h.interval != seal.MinIntervalSeconds {
		t.Errorf("interval %d after sealing, want %d", h.interval, seal.MinIntervalSeconds)
	}
}

// Bounded A2 (TICK-440 review, binding): with a backlog above the cap, one
// A2 call seals MaxBatches batches, heartbeats between them, and returns;
// the capped run sets the interval to 10 s, and the next run continues the
// chain with no gap or duplicate seq. The entries come from three
// transactions with ids running backwards against occurred_at, so A2 must
// seal in (occurred_at, id) order across the run boundary, not in id order
// (Postgres review of PR #486).
func TestBoundedA2ContinuesInNextRun(t *testing.T) {
	e := migrated(t)
	tenant, o := lineage(t, e)
	const maxBatches = 2
	writeEntriesBackwards(t, e, tenant, maxBatches*seal.BatchSize+5, 3)
	acts := &seal.Activities{Pool: e.worker, MaxBatches: maxBatches}

	r := &workflowRun{acts: acts}
	if err := r.run(t, tenant, o); err != nil {
		t.Fatal(err)
	}
	if got := assertChain(t, e, tenant); got != maxBatches*seal.BatchSize {
		t.Fatalf("%d sealed, want the cap of %d", got, maxBatches*seal.BatchSize)
	}
	if r.heartbeats != maxBatches-1 {
		t.Errorf("%d heartbeats, want one between each of the %d batches", r.heartbeats, maxBatches)
	}
	if h := readHead(t, e, tenant); h.interval != seal.MinIntervalSeconds {
		t.Errorf("interval %d after a capped run, want %d", h.interval, seal.MinIntervalSeconds)
	}

	if err := (&workflowRun{acts: acts}).run(t, tenant, nextOf(t, e, tenant, o)); err != nil {
		t.Fatal(err)
	}
	if got := assertChain(t, e, tenant); got != maxBatches*seal.BatchSize+5 {
		t.Errorf("%d sealed after the next run, want %d", got, maxBatches*seal.BatchSize+5)
	}
}

// A failed A1 attempt counts in audit_seal_failures_total{reason=reschedule}
// (TICK-587). A tenant whose head is missing fails A1 with the non-retryable
// nava.audit.seal.HeadNotFound (TICK-588 M1: A1 is otherwise retried without
// a cap): one attempt, counted once, and the run never reaches A2.
func TestRescheduleFailuresCounted(t *testing.T) {
	e := migrated(t)
	tenant := uuid.New() // no head row
	m, metric := promMetrics(t)
	r := &workflowRun{acts: &seal.Activities{Pool: e.worker, Metrics: m}}
	assertAppError(t, r.run(t, tenant, uuid.Must(uuid.NewV7())), seal.ErrTypeHeadNotFound)
	if got := metric("audit_seal_failures_total", seal.ReasonReschedule); got != 1 {
		t.Errorf("audit_seal_failures_total{reason=reschedule} = %v, want 1: HeadNotFound is not retried", got)
	}
	if got := metric("audit_seal_failures_total", seal.ReasonError) + metric("audit_seal_failures_total", seal.ReasonUnsealable); got != 0 {
		t.Errorf("A2 failures counted %v with A1 never committing, want 0", got)
	}
}
