//go:build integration

package schemabackfill_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/schemabackfill"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
)

// Each activity on its own, against a job row put in the state a retried
// attempt (a commit lost under synchronous_commit=off, a worker lost after
// it committed) would find. A probe workflow under the job's id runs the
// one activity, once, so the activity reads the job from the id as it does
// in production.

const probeWorkflow = "schema_backfill.test.probe"

// probe runs the activity named name once and returns its raw result. The
// page activity gets an empty cursor, the only argument any activity takes.
func probe(ctx workflow.Context, name string) (json.RawMessage, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1}})
	var args []any
	if name == schemabackfill.PageActivity {
		args = append(args, schemabackfill.Cursor{})
	}
	var out json.RawMessage
	err := workflow.ExecuteActivity(ctx, name, args...).Get(ctx, &out)
	return out, err
}

type probeOpts struct {
	id        string        // workflow id; the job's when empty
	tenantCtx *uuid.UUID    // a verified tenant put on every activity context, with no guard
	log       *bytes.Buffer // the activities' log
	slowAfter time.Duration // the barrier's slow threshold
	acts      func(*schemabackfill.Activities)
}

// runActivity runs activity name for job j through probe and returns its
// raw result and error.
func runActivity(t *testing.T, e itEnv, j schemabackfill.Job, name string, o probeOpts) (json.RawMessage, error) {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	id := o.id
	if id == "" {
		id = schemabackfill.WorkflowID(j)
	}
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: id})
	wo := worker.Options{}
	if o.tenantCtx != nil {
		wo.BackgroundActivityContext = workerguard.ContextWithTenant(context.Background(), *o.tenantCtx)
	}
	env.SetWorkerOptions(wo)
	env.SetTestTimeout(time.Minute)
	acts := &schemabackfill.Activities{Pool: e.worker, Materializer: recordports.NewMaterializer(e.pipe), SlowAfter: o.slowAfter}
	if o.log != nil {
		acts.Log = slog.New(slog.NewJSONHandler(&syncBuffer{b: o.log}, nil))
	}
	if o.acts != nil {
		o.acts(acts)
	}
	acts.Register(env)
	env.RegisterWorkflowWithOptions(probe, workflow.RegisterOptions{Name: probeWorkflow})
	env.ExecuteWorkflow(probeWorkflow, name)
	if !env.IsWorkflowCompleted() {
		t.Fatal("probe did not complete")
	}
	var out json.RawMessage
	err := env.GetWorkflowError()
	if err == nil {
		_ = env.GetWorkflowResult(&out)
	}
	return out, err
}

type syncBuffer struct{ b *bytes.Buffer }

func (s *syncBuffer) Write(p []byte) (int, error) { return s.b.Write(p) }

func noJob(err error) bool {
	var app *temporal.ApplicationError
	return errors.As(err, &app) && app.Type() == schemabackfill.ErrTypeNoJob && app.NonRetryable()
}

// setState puts the job row in state, with started_at as a run would have.
func setState(t *testing.T, e itEnv, j schemabackfill.Job, state string) {
	t.Helper()
	mustExec(t, e.super, `UPDATE schema_backfills SET state = $4, started_at = CASE WHEN $4 = 'pending' THEN NULL ELSE now() END
		WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3`, j.Tenant, j.Definition, j.Version, state)
}

// queue puts ids in the job's queue, as a fill would.
func queue(t *testing.T, e itEnv, j schemabackfill.Job, ids ...uuid.UUID) {
	t.Helper()
	for _, id := range ids {
		mustExec(t, e.super, `INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id)
			VALUES ($1, $2, $3, $4)`, j.Tenant, j.Definition, j.Version, id)
	}
}

func jobAt(t *testing.T, e itEnv, state string) (fixture, schemabackfill.Job) {
	t.Helper()
	f := newFixture(t, e)
	cause := f.publish(t, e, schemaV2)
	j := f.queueJob(t, e, 2, &cause, "channel", "priority")
	setState(t, e, j, state)
	return f, j
}

// CI-11, admission retried: a job already filling or running (the first
// attempt committed, its result was lost) is admitted again with no second
// "backfill started" entry; a job already over ends the workflow and writes
// nothing.
func TestAdmitIsIdempotent(t *testing.T) {
	e := migrated(t)
	for _, state := range []string{"filling", "running"} {
		t.Run(state, func(t *testing.T) {
			f, j := jobAt(t, e, state)
			out, err := runActivity(t, e, j, schemabackfill.AdmitActivity, probeOpts{})
			if err != nil || !strings.Contains(string(out), `"admitted"`) {
				t.Fatalf("admit = %s, %v; want admitted", out, err)
			}
			if n := len(jobEntries(t, e, f)); n != 0 {
				t.Errorf("job entries = %d, want none: the first attempt wrote the started entry", n)
			}
			if s := job(t, e, j); s.state != state {
				t.Errorf("state = %s, want %s", s.state, state)
			}
		})
	}
	for _, state := range []string{"done", "failed_records", "superseded"} {
		t.Run(state, func(t *testing.T) {
			f, j := jobAt(t, e, state)
			out, err := runActivity(t, e, j, schemabackfill.AdmitActivity, probeOpts{})
			if err != nil || !strings.Contains(string(out), `"ended"`) {
				t.Fatalf("admit = %s, %v; want ended", out, err)
			}
			if n := len(jobEntries(t, e, f)); n != 0 || job(t, e, j).state != state {
				t.Errorf("an ended job changed: %d entries, state %s", n, job(t, e, j).state)
			}
		})
	}
}

// D8 at admission: a pending job whose type has a newer job is superseded,
// with its pending and failure rows dropped and one "finished" entry, and
// ends the workflow.
func TestAdmitSupersedesAnOlderJob(t *testing.T) {
	e := migrated(t)
	f, j := jobAt(t, e, "pending")
	id := f.insertRecords(t, e, 1, plain)[0]
	queue(t, e, j, id)
	mustExec(t, e.super, `INSERT INTO schema_backfill_failures (tenant_id, entity_definition_id, target_version, record_id, code)
		VALUES ($1, $2, 2, $3, 'validation_failed')`, f.tenant, f.def, uuid.New())
	f.queueJob(t, e, 3, nil, "channel", "priority", "source")

	out, err := runActivity(t, e, j, schemabackfill.AdmitActivity, probeOpts{})
	if err != nil || !strings.Contains(string(out), `"ended"`) {
		t.Fatalf("admit = %s, %v; want ended", out, err)
	}
	s := job(t, e, j)
	if s.state != "superseded" || s.finished == nil || pending(t, e, j) != 0 ||
		count(t, e.super, `SELECT count(*) FROM schema_backfill_failures WHERE tenant_id = $1 AND target_version = 2`, f.tenant) != 0 {
		t.Errorf("job = %+v, pending %d; want superseded with its rows dropped", s, pending(t, e, j))
	}
	if entries := jobEntries(t, e, f); len(entries) != 1 || !strings.Contains(entries[0], `"superseded"`) {
		t.Errorf("job entries = %v, want one finished entry with state superseded", entries)
	}
}

// CI-11, fill retried: a job already running is not filled again, so a
// record written since is not queued and nothing is counted twice; a job
// in any other state but filling ends the workflow.
func TestFillIsIdempotent(t *testing.T) {
	e := migrated(t)
	f, j := jobAt(t, e, "running")
	ids := f.insertRecords(t, e, 3, plain)
	queue(t, e, j, ids[0])
	out, err := runActivity(t, e, j, schemabackfill.FillActivity, probeOpts{})
	if err != nil || strings.Contains(string(out), `"ended"`) {
		t.Fatalf("fill of a running job = %s, %v; want no end", out, err)
	}
	if n := pending(t, e, j); n != 1 || job(t, e, j).state != "running" {
		t.Errorf("pending = %d, state %s; want the one queued id and running", n, job(t, e, j).state)
	}

	for _, state := range []string{"pending", "done", "superseded"} {
		t.Run(state, func(t *testing.T) {
			f, j := jobAt(t, e, state)
			f.insertRecords(t, e, 2, plain)
			out, err := runActivity(t, e, j, schemabackfill.FillActivity, probeOpts{})
			if err != nil || !strings.Contains(string(out), `"ended"`) {
				t.Fatalf("fill of a %s job = %s, %v; want ended", state, out, err)
			}
			if n := pending(t, e, j); n != 0 {
				t.Errorf("pending = %d, want nothing queued", n)
			}
		})
	}
}

// CI-11, end retried: a job no longer running gets no second "finished"
// entry; one already failed_records still reports it, so the R6 check runs.
func TestEndIsIdempotent(t *testing.T) {
	e := migrated(t)
	for state, wantFailed := range map[string]bool{"done": false, "failed_records": true} {
		t.Run(state, func(t *testing.T) {
			f, j := jobAt(t, e, state)
			out, err := runActivity(t, e, j, schemabackfill.EndActivity, probeOpts{})
			if err != nil || strings.Contains(string(out), `"failed_records"`) != wantFailed {
				t.Fatalf("end of a %s job = %s, %v", state, out, err)
			}
			if n := len(jobEntries(t, e, f)); n != 0 || job(t, e, j).state != state {
				t.Errorf("a %s job changed: %d entries, state %s", state, n, job(t, e, j).state)
			}
		})
	}
}

// D2, the activity guard: every activity reads the job from its workflow id
// only, and refuses with a non-retryable ErrTypeNoJob an id that is not a
// job's, a job with no row under RLS, and a verified tenant that is not the
// id's (another tenant's context reaching this job's activity).
func TestActivitiesRefuseAWorkflowThatIsNotTheJob(t *testing.T) {
	e := migrated(t)
	f, j := jobAt(t, e, "pending")
	for _, name := range []string{schemabackfill.AdmitActivity, schemabackfill.BarrierActivity, schemabackfill.FillActivity,
		schemabackfill.PageActivity, schemabackfill.EndActivity, schemabackfill.R6Activity} {
		t.Run(name, func(t *testing.T) {
			if _, err := runActivity(t, e, j, name, probeOpts{id: "t:" + f.tenant.String() + ":wf:1"}); !noJob(err) {
				t.Errorf("not a job's id: %v, want non-retryable ErrTypeNoJob", err)
			}
			other := uuid.New()
			if _, err := runActivity(t, e, j, name, probeOpts{tenantCtx: &other}); !noJob(err) {
				t.Errorf("another verified tenant: %v, want non-retryable ErrTypeNoJob", err)
			}
		})
	}
	if s := job(t, e, j); s.state != "pending" || s.started != nil {
		t.Errorf("a refused activity changed the job: %+v", s)
	}
	// No row under RLS for the id: tenant B's view of tenant A's job.
	missing := schemabackfill.Job{Tenant: uuid.New(), Definition: f.def, Version: 2}
	for _, name := range []string{schemabackfill.AdmitActivity, schemabackfill.FillActivity, schemabackfill.EndActivity} {
		if _, err := runActivity(t, e, missing, name, probeOpts{}); !noJob(err) {
			t.Errorf("%s with no row: %v, want non-retryable ErrTypeNoJob", name, err)
		}
	}
	// The R6 check of a job with no row does nothing.
	if _, err := runActivity(t, e, missing, schemabackfill.R6Activity, probeOpts{}); err != nil {
		t.Errorf("r6 with no row: %v, want nothing", err)
	}
}

// A database error in a job-level activity is logged by its shape only: the
// step, the SQLSTATE and the constraint, never the error's text (which can
// quote a value). The attempt fails retryable.
func TestActivityDatabaseErrorLogsItsShapeOnly(t *testing.T) {
	e := migrated(t)
	f, j := jobAt(t, e, "pending")
	constraint := "it_admit_" + strings.ReplaceAll(f.tenant.String()[:8], "-", "")
	mustExec(t, e.super, `ALTER TABLE schema_backfills ADD CONSTRAINT `+constraint+
		` CHECK (NOT (tenant_id = '`+f.tenant.String()+`' AND state = 'filling')) NOT VALID`)
	t.Cleanup(func() { mustExec(t, e.super, "ALTER TABLE schema_backfills DROP CONSTRAINT "+constraint) })

	var log bytes.Buffer
	_, err := runActivity(t, e, j, schemabackfill.AdmitActivity, probeOpts{log: &log})
	var app *temporal.ApplicationError
	if err == nil || (errors.As(err, &app) && app.NonRetryable()) {
		t.Fatalf("admit = %v, want a retryable failure", err)
	}
	line := log.String()
	for _, want := range []string{`"step":"admit"`, `"sqlstate":"23514"`, `"constraint":"` + constraint + `"`} {
		if !strings.Contains(line, want) {
			t.Errorf("log %s lacks %s", line, want)
		}
	}
	if strings.Contains(line, "violates") || strings.Contains(line, f.tenant.String()+"'") {
		t.Errorf("log carries the error's text: %s", line)
	}
	if s := job(t, e, j); s.state != "pending" {
		t.Errorf("state = %s, want pending: the failed admission rolled back", s.state)
	}
}

// The barrier raises its slow alert once a transaction open at the
// admission outlives SlowAfter, and still waits for it.
func TestBarrierSlowAlert(t *testing.T) {
	e := migrated(t)
	_, j := jobAt(t, e, "filling")
	ctx := context.Background()
	conn, err := e.super.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var xid string
	if err := tx.QueryRow(ctx, "SELECT pg_current_xact_id()::text").Scan(&xid); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(2500 * time.Millisecond)
		_ = tx.Rollback(ctx)
	}()
	var log bytes.Buffer
	start := time.Now()
	if _, err := runActivity(t, e, j, schemabackfill.BarrierActivity, probeOpts{log: &log, slowAfter: 500 * time.Millisecond}); err != nil {
		t.Fatalf("barrier: %v", err)
	}
	if waited := time.Since(start); waited < 2*time.Second {
		t.Errorf("barrier returned after %v, before the open transaction ended", waited)
	}
	if n := strings.Count(log.String(), "schema_backfill_barrier_slow"); n != 1 {
		t.Errorf("slow alerts = %d, want 1; log: %s", n, log.String())
	}
}

// A record deleted after the fill (its id still queued: no foreign key to
// records) is nothing to write: the page skips it, drops it from the queue
// and the job ends done, with no failure row.
func TestPageSkipsADeletedRecord(t *testing.T) {
	e := migrated(t)
	f, j := jobAt(t, e, "pending")
	ids := f.insertRecords(t, e, 3, plain)
	r := newRun(t, e, j, nil)
	var once bool
	schemabackfill.SetAfterRecord(r.acts, func(context.Context, uuid.UUID) error {
		if !once {
			once = true
			mustExec(t, e.super, `WITH d AS (DELETE FROM records WHERE id = $1 RETURNING tenant_id, id, version)
				INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_before, actor_system, source)
				SELECT tenant_id, gen_random_uuid(), 'record', 'delete', 'record', id, version, 'system', 'system' FROM d`, ids[2])
		}
		return nil
	})
	if err := r.start(j); err != nil {
		t.Fatalf("workflow: %v", err)
	}
	s := job(t, e, j)
	if s.state != "done" || s.rewritten != 2 || s.skipped != 1 || s.failed != 0 || pending(t, e, j) != 0 {
		t.Errorf("job = %+v, want done with 2 rewritten, the deleted record skipped, no failure", s)
	}
}

// A page of a job that is no longer running (a retried page after the job
// ended, or one that a newer job superseded through another path) ends the
// workflow and touches nothing.
func TestPageOfAJobNoLongerRunningEnds(t *testing.T) {
	e := migrated(t)
	for _, state := range []string{"done", "superseded", "filling"} {
		t.Run(state, func(t *testing.T) {
			f, j := jobAt(t, e, state)
			id := f.insertRecords(t, e, 1, plain)[0]
			queue(t, e, j, id)
			out, err := runActivity(t, e, j, schemabackfill.PageActivity, probeOpts{})
			if err != nil || !strings.Contains(string(out), `"ended"`) {
				t.Fatalf("page of a %s job = %s, %v; want ended", state, out, err)
			}
			if pending(t, e, j) != 1 || materializeEntries(t, e, id) != 0 {
				t.Error("a page of a job no longer running rewrote a record")
			}
		})
	}
}

// An id another attempt has already taken from the queue (a page retried
// while the first attempt still ran) is left alone: no rewrite, no count.
func TestRewriteLeavesAnIDAnotherAttemptTook(t *testing.T) {
	e := migrated(t)
	f, j := jobAt(t, e, "pending")
	ids := f.insertRecords(t, e, 4, plain)
	r := newRun(t, e, j, nil)
	var stolen uuid.UUID
	schemabackfill.SetAfterRecord(r.acts, func(_ context.Context, done uuid.UUID) error {
		if stolen != uuid.Nil {
			return nil
		}
		// Take the next queued id, as the other attempt would.
		if err := e.super.QueryRow(context.Background(), `DELETE FROM schema_backfill_pending
			WHERE record_id = (SELECT record_id FROM schema_backfill_pending
				WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = 2 AND record_id > $3
				ORDER BY record_id LIMIT 1)
			RETURNING record_id`, f.tenant, f.def, done).Scan(&stolen); err != nil {
			t.Errorf("take an id: %v", err)
		}
		return nil
	})
	if err := r.start(j); err != nil {
		t.Fatalf("workflow: %v", err)
	}
	if stolen == uuid.Nil {
		t.Fatal("no id was taken")
	}
	if n := materializeEntries(t, e, stolen); n != 0 {
		t.Errorf("the taken id got %d rewrite entries, want 0: the other attempt owns it", n)
	}
	if s := job(t, e, j); s.state != "done" || s.rewritten != int64(len(ids)-1) || s.skipped != 0 {
		t.Errorf("job = %+v, want done with %d rewritten and the taken id not counted", s, len(ids)-1)
	}
}
