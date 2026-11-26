//go:build integration

package schemabackfill_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/schemabackfill"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// CI-2 (D3-D7): 500 records missing the default plus 50 holding it. The 500
// get it with version + 1, schema_version 2, the same updated_at and exactly
// one rewrite entry carrying the publish's causation; the 50 are never
// queued: no entry, no bump. The job ends done, rewritten 500, skipped 0,
// with one "backfill started" and one "backfill finished" entry, an empty
// queue, and no outbox row (no workflow start or signal, D6). The run's
// history carries references only.
func TestBackfillRewritesMissingDefaultsOnce(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e)
	missing := f.insertRecords(t, e, 500, plain)
	holding := f.insertRecords(t, e, 50, func(i int) map[string]any {
		d := plain(i)
		d["channel"] = "phone"
		return d
	})
	cause := f.publish(t, e, schemaV2)
	j := f.queueJob(t, e, 2, &cause, "channel", "priority")
	before := map[uuid.UUID]stored{}
	for _, id := range append(append([]uuid.UUID{}, missing...), holding...) {
		before[id] = record(t, e, id)
	}
	outboxBefore := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", f.tenant)

	r := newRun(t, e, j, nil)
	if err := r.start(j); err != nil {
		t.Fatalf("workflow: %v", err)
	}
	r.requireGuardClean()
	r.assertNoValues()

	for _, id := range missing {
		got := record(t, e, id)
		if got.data["channel"] != "email" || got.version != before[id].version+1 || got.schemaVersion != 2 {
			t.Fatalf("record %s: %v version %d schema %d; want channel email, version %d, schema 2",
				id, got.data, got.version, got.schemaVersion, before[id].version+1)
		}
		if !got.updatedAt.Equal(before[id].updatedAt) {
			t.Fatalf("record %s: updated_at moved (D5)", id)
		}
		if n := materializeEntries(t, e, id); n != 1 {
			t.Fatalf("record %s: %d rewrite entries, want 1", id, n)
		}
	}
	var origin uuid.UUID
	var depth int
	if err := e.super.QueryRow(context.Background(), `SELECT causation_id, causation_depth FROM audit_log
		WHERE subject_id = $1 AND actor_system = 'schema-backfill'`, missing[0]).Scan(&origin, &depth); err != nil {
		t.Fatal(err)
	}
	if origin != cause || depth != 1 {
		t.Errorf("rewrite causation = %s/%d, want the publish entry %s at depth 1", origin, depth, cause)
	}
	for _, id := range holding {
		got := record(t, e, id)
		if got.version != before[id].version || got.data["channel"] != "phone" {
			t.Fatalf("held record %s changed: %v version %d", id, got.data, got.version)
		}
		if n := materializeEntries(t, e, id); n != 0 {
			t.Fatalf("held record %s: %d rewrite entries, want 0", id, n)
		}
	}
	s := job(t, e, j)
	if s.state != "done" || s.rewritten != 500 || s.skipped != 0 || s.failed != 0 || s.started == nil || s.finished == nil {
		t.Errorf("job = %+v, want done 500/0/0 with started_at and finished_at", s)
	}
	if n := pending(t, e, j); n != 0 {
		t.Errorf("pending = %d, want 0", n)
	}
	entries := jobEntries(t, e, f)
	if len(entries) != 2 || !strings.Contains(entries[0], `"filling"`) || !strings.Contains(entries[1], `"done"`) ||
		!strings.Contains(entries[1], `"value":500`) {
		t.Errorf("job entries = %v, want started (filling) then finished (done, rewritten 500)", entries)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", f.tenant); n != outboxBefore {
		t.Errorf("outbox rows = %d, want %d: a rewrite starts and signals nothing (D6)", n, outboxBefore)
	}
}

// CI-6 (D7, D9): (a) a record the current version refuses and (b) a forced
// CHECK violation each give one failure row with the id and a code, never a
// value; the job goes on past them and ends failed_records. With the R6
// window shortened, the R6 check then trips the counter once and logs one
// WARN line with no record id (P10-06 item 4).
func TestBackfillPoisonRecordsFailAndTripR6(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e)
	good := f.insertRecords(t, e, 20, plain)
	refused := f.insertRecords(t, e, 1, func(int) map[string]any {
		return map[string]any{"title": valueSecret + "refused", "priority": "medium"} // not in v2's enum
	})[0]
	poison := f.insertRecords(t, e, 1, func(int) map[string]any {
		return map[string]any{"title": "poison-" + f.tenant.String(), "priority": "low"}
	})[0]
	constraint := "it_poison_" + strings.ReplaceAll(f.tenant.String()[:8], "-", "")
	mustExec(t, e.super, fmt.Sprintf(`ALTER TABLE records ADD CONSTRAINT %s
		CHECK (NOT (data->>'title' = 'poison-%s' AND data ? 'channel')) NOT VALID`, constraint, f.tenant))
	t.Cleanup(func() { mustExec(t, e.super, "ALTER TABLE records DROP CONSTRAINT "+constraint) })
	cause := f.publish(t, e, schemaV2)
	j := f.queueJob(t, e, 2, &cause, "channel", "priority")

	r := newRun(t, e, j, &schemabackfill.Workflow{R6After: time.Hour})
	if err := r.start(j); err != nil {
		t.Fatalf("workflow: %v", err)
	}
	r.requireGuardClean()
	r.assertNoValues()

	s := job(t, e, j)
	if s.state != "failed_records" || s.rewritten != 20 || s.failed != 2 {
		t.Fatalf("job = %+v, want failed_records with 20 rewritten and 2 failed", s)
	}
	codes := map[uuid.UUID]string{}
	rows, err := e.super.Query(context.Background(), `SELECT record_id, code FROM schema_backfill_failures
		WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = 2`, f.tenant, f.def)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id uuid.UUID
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			t.Fatal(err)
		}
		codes[id] = code
	}
	if len(codes) != 2 || codes[refused] != pipeline.CodeValidationFailed || codes[poison] != schemabackfill.CodeDatabaseError {
		t.Errorf("failures = %v, want %s: %s and %s: %s", codes, refused, pipeline.CodeValidationFailed, poison, schemabackfill.CodeDatabaseError)
	}
	for _, id := range good {
		if record(t, e, id).data["channel"] != "email" {
			t.Fatalf("record %s was not rewritten past the poison records", id)
		}
	}
	if pending(t, e, j) != 0 {
		t.Error("failed ids are still pending")
	}
	if n := r.counter("nava_schema_backfill_r6_tripped", "", ""); n != 1 {
		t.Errorf("r6 counter = %d, want 1", n)
	}
	logs := r.logs.String()
	if strings.Count(logs, "schema backfill R6 tripped") != 1 || strings.Contains(logs, refused.String()+`","alert_name":"schema_backfill_r6`) {
		t.Errorf("want one R6 WARN line; logs: %s", logs)
	}
}

// R6, negative: a failed_records job that a newer job follows before its
// window ends does not trip (P10-06 item 4).
func TestBackfillR6NotTrippedWhenANewerJobExists(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e)
	f.insertRecords(t, e, 1, func(int) map[string]any { return map[string]any{"title": "x", "priority": "medium"} })
	cause := f.publish(t, e, schemaV2)
	j := f.queueJob(t, e, 2, &cause, "channel", "priority")
	r := newRun(t, e, j, &schemabackfill.Workflow{R6After: time.Hour})
	r.env.RegisterDelayedCallback(func() { f.queueJob(t, e, 3, nil, "channel", "priority", "source") }, 30*time.Minute)
	if err := r.start(j); err != nil {
		t.Fatalf("workflow: %v", err)
	}
	if s := job(t, e, j); s.state != "failed_records" {
		t.Fatalf("job = %+v, want failed_records", s)
	}
	if n := r.counter("nava_schema_backfill_r6_tripped", "", ""); n != 0 {
		t.Errorf("r6 counter = %d, want 0: a newer job exists", n)
	}
}

// CI-3 (3): a job of tenant A changes no row of tenant B, even when B has
// the same type key, its own job at the same version and its own pending
// rows.
func TestBackfillTouchesNoOtherTenant(t *testing.T) {
	e := migrated(t)
	a, b := newFixture(t, e), newFixture(t, e)
	a.insertRecords(t, e, 10, plain)
	bIDs := b.insertRecords(t, e, 10, plain)
	ca, cb := a.publish(t, e, schemaV2), b.publish(t, e, schemaV2)
	ja := a.queueJob(t, e, 2, &ca, "channel", "priority")
	jb := b.queueJob(t, e, 2, &cb, "channel", "priority")
	mustExec(t, e.super, `UPDATE schema_backfills SET state = 'running' WHERE tenant_id = $1`, b.tenant)
	for _, id := range bIDs {
		mustExec(t, e.super, `INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id)
			VALUES ($1, $2, 2, $3)`, b.tenant, b.def, id)
	}
	bBefore := map[uuid.UUID]stored{}
	for _, id := range bIDs {
		bBefore[id] = record(t, e, id)
	}

	r := newRun(t, e, ja, nil)
	if err := r.start(ja); err != nil {
		t.Fatalf("workflow: %v", err)
	}
	if s := job(t, e, ja); s.state != "done" || s.rewritten != 10 {
		t.Fatalf("tenant A job = %+v, want done with 10 rewritten", s)
	}
	for _, id := range bIDs {
		if got := record(t, e, id); got.version != bBefore[id].version || got.data["channel"] != nil {
			t.Fatalf("tenant B record %s changed: %+v", id, got)
		}
	}
	if s := job(t, e, jb); s.state != "running" || s.rewritten != 0 || s.started != nil {
		t.Errorf("tenant B job = %+v, want it untouched", s)
	}
	if n := pending(t, e, jb); n != 10 {
		t.Errorf("tenant B pending = %d, want 10", n)
	}
}

// CI-3 (5): with no app.tenant_id, the job's reads see no row as nava_worker,
// even when they name the tenant explicitly.
func TestBackfillReadsNothingWithoutTenant(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e)
	f.insertRecords(t, e, 3, plain)
	cause := f.publish(t, e, schemaV2)
	j := f.queueJob(t, e, 2, &cause, "channel", "priority")
	mustExec(t, e.super, `INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id)
		SELECT tenant_id, entity_definition_id, 2, id FROM records WHERE tenant_id = $1`, f.tenant)
	ctx := context.Background()
	tx, err := e.worker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var n int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM ("+strings.TrimSuffix(schemabackfill.LockJobSQL, " FOR UPDATE")+") x",
		j.Tenant, j.Definition, j.Version).Scan(&n); err != nil || n != 0 {
		t.Errorf("job read with no tenant = %d, %v; want 0 rows", n, err)
	}
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM ("+strings.Replace(schemabackfill.PageSQL, "$5", "100", 1)+") x",
		j.Tenant, j.Definition, j.Version, nil).Scan(&n); err != nil || n != 0 {
		t.Errorf("page read with no tenant = %d, %v; want 0 rows", n, err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2
		AND NOT (r.data ?& $3::text[])`, j.Tenant, j.Definition, []string{"channel"}).Scan(&n); err != nil || n != 0 {
		t.Errorf("fill read with no tenant = %d, %v; want 0 rows", n, err)
	}
}

// CI-11 (worker half): a page that fails after some records committed is
// retried; the retry finds those ids gone from the queue, so each rewritten
// record has exactly one rewrite entry, and the job finishes. A cancelled
// in-flight transaction (a worker stopping mid-page) commits nothing and
// leaves its id pending for the retry.
func TestBackfillRetriedPageAuditsOnce(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e)
	ids := f.insertRecords(t, e, 250, plain)
	cause := f.publish(t, e, schemaV2)
	j := f.queueJob(t, e, 2, &cause, "channel", "priority")
	r := newRun(t, e, j, nil)
	var n, fails atomic.Int32
	schemabackfill.SetAfterRecord(r.acts, func(context.Context, uuid.UUID) error {
		if n.Add(1)%70 == 0 && fails.Add(1) <= 2 {
			return errors.New("injected page failure after a commit")
		}
		return nil
	})
	if err := r.start(j); err != nil {
		t.Fatalf("workflow: %v", err)
	}
	if fails.Load() < 2 {
		t.Fatalf("injected %d failures, want 2", fails.Load())
	}
	if r.started[schemabackfill.PageActivity] < 5 {
		t.Errorf("page starts = %d, want the retries too", r.started[schemabackfill.PageActivity])
	}
	for _, id := range ids {
		if got := materializeEntries(t, e, id); got != 1 {
			t.Fatalf("record %s: %d rewrite entries, want exactly 1", id, got)
		}
	}
	if s := job(t, e, j); s.state != "done" || s.rewritten != 250 {
		t.Errorf("job = %+v, want done with 250 rewritten", s)
	}

	// A cancelled transaction commits neither the rewrite nor the pending
	// delete.
	f2 := newFixture(t, e)
	id := f2.insertRecords(t, e, 1, plain)[0]
	c2 := f2.publish(t, e, schemaV2)
	j2 := f2.queueJob(t, e, 2, &c2, "channel", "priority")
	mustExec(t, e.super, `UPDATE schema_backfills SET state = 'running' WHERE tenant_id = $1`, f2.tenant)
	mustExec(t, e.super, `INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id)
		VALUES ($1, $2, 2, $3)`, f2.tenant, f2.def, id)
	ctx, cancel := context.WithCancel(context.Background())
	err := tenancy.InTenantTx(ctx, e.worker, f2.tenant, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL synchronous_commit = off"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM schema_backfill_pending WHERE tenant_id = $1 AND record_id = $2`, f2.tenant, id); err != nil {
			return err
		}
		cancel()
		return ctx.Err()
	})
	if err == nil {
		t.Fatal("the cancelled transaction committed")
	}
	if pending(t, e, j2) != 1 || materializeEntries(t, e, id) != 0 {
		t.Error("a cancelled record transaction left a trace")
	}
}

// P10-06 item 2, admission race: two pending jobs of one tenant, on two
// types, admitted at the same moment (both UPDATEs released together):
// exactly one reaches filling, the other stays pending, waits, and is
// admitted after the first ends.
func TestBackfillOneJobPerTenant(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e)
	f.insertRecords(t, e, 30, plain)
	// A second type in the same tenant, with its own default.
	var def2 uuid.UUID
	f.inApp(t, e, func(tx pgx.Tx) error {
		d := definition(schemaV1)
		d.Key, d.NumberPrefix = "problem", "PRB"
		_, err := entitytypes.Store{}.Create(context.Background(), tx, f.tenant, entitytypes.NewInput{
			Name: "problem", PublishedBy: f.principal, Definition: d})
		return err
	})
	if err := e.super.QueryRow(context.Background(), "SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = 'problem'",
		f.tenant).Scan(&def2); err != nil {
		t.Fatal(err)
	}
	cause := f.publish(t, e, schemaV2)
	j1 := f.queueJob(t, e, 2, &cause, "channel", "priority")
	j2 := schemabackfill.Job{Tenant: f.tenant, Definition: def2, Version: 1}
	mustExec(t, e.super, `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, default_keys)
		VALUES ($1, $2, 1, '{priority}')`, f.tenant, def2)

	r1 := newRun(t, e, j1, nil)
	r2 := newRun(t, e, j2, &schemabackfill.Workflow{AdmitWait: time.Second, AdmitWaitMax: time.Second})
	var ready sync.WaitGroup
	ready.Add(2)
	release := make(chan struct{})
	var once1, once2 sync.Once
	gate := func(once *sync.Once) func(context.Context) {
		return func(context.Context) {
			once.Do(func() {
				ready.Done()
				<-release
			})
		}
	}
	schemabackfill.SetBeforeAdmit(r1.acts, gate(&once1))
	schemabackfill.SetBeforeAdmit(r2.acts, gate(&once2))
	go func() { ready.Wait(); close(release) }()
	// The first winner holds the slot through a slow page, so the loser
	// waits at least once.
	slow := func(context.Context, uuid.UUID) error { time.Sleep(5 * time.Millisecond); return nil }
	schemabackfill.SetAfterRecord(r1.acts, slow)
	schemabackfill.SetAfterRecord(r2.acts, slow)

	errs := make(chan error, 2)
	go func() { errs <- r1.start(j1) }()
	go func() { errs <- r2.start(j2) }()
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("workflow: %v", err)
		}
	}
	s1, s2 := job(t, e, j1), job(t, e, j2)
	if s1.state != "done" || s2.state != "done" {
		t.Fatalf("jobs = %+v, %+v; want both done", s1, s2)
	}
	first, second := s1, s2
	if s2.started.Before(*s1.started) {
		first, second = s2, s1
	}
	if second.started.Before(*first.finished) {
		t.Errorf("the jobs overlapped: second started %v before the first finished %v", second.started, first.finished)
	}
	if r1.started[schemabackfill.AdmitActivity]+r2.started[schemabackfill.AdmitActivity] < 3 {
		t.Errorf("admissions = %d + %d, want the loser to ask again", r1.started[schemabackfill.AdmitActivity],
			r2.started[schemabackfill.AdmitActivity])
	}
}

// CI-5 (D8): a publish of N+1 with another default during N's run: N ends
// superseded with its pending and failure rows gone; N+1, run after it, ends
// done, and every record holds both keys.
func TestBackfillSupersededByANewerJob(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e)
	ids := f.insertRecords(t, e, 300, plain)
	c2 := f.publish(t, e, schemaV2)
	j2 := f.queueJob(t, e, 2, &c2, "channel", "priority")
	r := newRun(t, e, j2, nil)
	var n atomic.Int32
	var j3 schemabackfill.Job
	schemabackfill.SetAfterRecord(r.acts, func(context.Context, uuid.UUID) error {
		if n.Add(1) == 50 {
			c3 := f.publish(t, e, schemaV3)
			j3 = f.queueJob(t, e, 3, &c3, "channel", "priority", "source")
			mustExec(t, e.super, `INSERT INTO schema_backfill_failures (tenant_id, entity_definition_id, target_version, record_id, code)
				VALUES ($1, $2, 2, $3, 'validation_failed')`, f.tenant, f.def, uuid.New())
		}
		return nil
	})
	if err := r.start(j2); err != nil {
		t.Fatalf("job 2: %v", err)
	}
	s2 := job(t, e, j2)
	if s2.state != "superseded" || s2.finished == nil {
		t.Fatalf("job 2 = %+v, want superseded", s2)
	}
	if pending(t, e, j2) != 0 || count(t, e.super, `SELECT count(*) FROM schema_backfill_failures
		WHERE tenant_id = $1 AND target_version = 2`, f.tenant) != 0 {
		t.Error("job 2 kept pending or failure rows")
	}
	r3 := newRun(t, e, j3, nil)
	if err := r3.start(j3); err != nil {
		t.Fatalf("job 3: %v", err)
	}
	if s := job(t, e, j3); s.state != "done" {
		t.Fatalf("job 3 = %+v, want done", s)
	}
	for _, id := range ids {
		got := record(t, e, id)
		if got.data["channel"] != "email" || got.data["source"] != "web" || got.schemaVersion != 3 {
			t.Fatalf("record %s = %+v, want both defaults at schema 3", id, got)
		}
		if m := materializeEntries(t, e, id); m < 1 || m > 2 {
			t.Fatalf("record %s: %d rewrite entries, want 1 or 2", id, m)
		}
	}
}

// CI-4 (D7): four live writers on the type during the job. No update is
// lost (each record ends with its writer's last value and the default), and
// live conflicts stay rare: a rewrite holds a record's row lock for one
// record's transaction only.
func TestBackfillWithLiveWriters(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e)
	ids := f.insertRecords(t, e, 400, plain)
	cause := f.publish(t, e, schemaV2)
	j := f.queueJob(t, e, 2, &cause, "channel", "priority")
	p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{},
		pipeline.PoolRunner{Pool: e.pipe}, pipeline.SystemClock{})

	// The writers start once the job has committed its first record, so they
	// run against the job, not before it.
	r := newRun(t, e, j, nil)
	begin := make(chan struct{})
	var beginOnce sync.Once
	schemabackfill.SetAfterRecord(r.acts, func(context.Context, uuid.UUID) error {
		beginOnce.Do(func() { close(begin) })
		time.Sleep(2 * time.Millisecond)
		return nil
	})
	ctx, stop := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var conflicts atomic.Int64
	last := make([]map[uuid.UUID]string, 4)
	for w := range 4 {
		last[w] = map[uuid.UUID]string{}
		wg.Go(func() {
			select {
			case <-begin:
			case <-ctx.Done():
				return
			}
			for n := 0; ctx.Err() == nil && n < 2000; n++ {
				id := ids[(n*4+w)%len(ids)]
				val := fmt.Sprintf("w%d-%d", w, n)
				for {
					var version int64
					err := tenancy.InTenantTx(context.Background(), e.app, f.tenant, func(tx pgx.Tx) error {
						if err := tx.QueryRow(context.Background(), "SELECT version FROM records WHERE tenant_id = $1 AND id = $2",
							f.tenant, id).Scan(&version); err != nil {
							return err
						}
						_, err := p.Write(context.Background(), tx, pipeline.Command{Op: pipeline.OpUpdate, EntityKey: entityKey,
							RecordID: id, ExpectedVersion: version, Data: map[string]any{"summary": val},
							Actor: identity.Principal{TenantID: f.tenant}, Source: pipeline.SourceSystem})
						return err
					})
					if errors.Is(err, pipeline.ErrVersionConflict) || errors.Is(err, pipeline.ErrRetryable) {
						conflicts.Add(1)
						continue
					}
					if err != nil {
						t.Errorf("live write: %v", err)
						return
					}
					last[w][id] = val
					break
				}
			}
		})
	}
	err := r.start(j)
	stop()
	wg.Wait()
	if err != nil {
		t.Fatalf("workflow: %v", err)
	}
	for w := range 4 {
		for id, val := range last[w] {
			if got := record(t, e, id).data; got["summary"] != val || got["channel"] != "email" {
				t.Fatalf("record %s = %v, want summary %s (writer %d's last) and channel email: a lost update", id, got, val, w)
			}
		}
	}
	for _, id := range ids {
		if record(t, e, id).data["channel"] != "email" {
			t.Fatalf("record %s has no channel", id)
		}
	}
	t.Logf("live version conflicts: %d", conflicts.Load())
	// Every record was queued (the fill ran before any live write); a live
	// write applies the defaults itself, so the job skips the records it
	// reached after one (D3 compare).
	if s := job(t, e, j); s.state != "done" || s.rewritten == 0 || s.rewritten+s.skipped != 400 || s.failed != 0 {
		t.Errorf("job = %+v, want done with all 400 handled, some rewritten, none failed", s)
	}
	writes := 0
	for w := range 4 {
		writes += len(last[w])
	}
	if writes == 0 {
		t.Fatal("no live write ran during the job")
	}
}
