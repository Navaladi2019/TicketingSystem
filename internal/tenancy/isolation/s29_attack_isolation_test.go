//go:build integration

package isolation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/relay"
	"github.com/Navaladi2019/TicketingSystem/internal/schemabackfill"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
)

// The bodies of rows S29 4 to 6 (TICK-753, absorbed TICK-926): they drive
// the records pipeline, the real schema.backfill workflow and activities on
// the Temporal SDK test environment behind the worker guard, and the relay's
// dispatcher, so they live in a test file under the integration tag, as row
// 3b's do. s22_s29_attack_paths.go registers the rows and calls these
// through s29Attacker.

func (c cluster) S29Materialize4(ctx context.Context, e *etIso, fx *s29Fixture) error {
	return s29Materialize4(ctx, c, e, fx)
}

func (c cluster) S29Job5(ctx context.Context, e *etIso, fx *s29Fixture) error {
	return s29Job5(ctx, c, e, fx)
}

func (c cluster) S29Relay6(ctx context.Context, e *etIso, fx *s29Fixture) error {
	return s29Relay6(ctx, c, e, fx)
}

// s29Snap is everything of tenant's iso_et_s29 a backfill could change, read
// past RLS: each record's version, schema version and stored body, the
// record audit entries, and the type's jobs, queue and failures.
func s29Snap(ctx context.Context, e *etIso, tenant, def string) (string, error) {
	var s string
	err := e.super.QueryRow(ctx, `SELECT
		coalesce((SELECT string_agg(id::text || ' v' || version::text || ' s' || schema_version::text || ' ' || data::text, '; ' ORDER BY id)
			FROM records WHERE tenant_id = $1 AND entity_definition_id = $2), '') || ' | ' ||
		(SELECT count(*)::text FROM audit_log WHERE tenant_id = $1 AND subject_type = 'record'
			AND subject_id IN (SELECT id FROM records WHERE tenant_id = $1 AND entity_definition_id = $2)) || ' | ' ||
		coalesce((SELECT string_agg(target_version::text || ':' || state || ':' || rewritten::text || '/' || skipped::text || '/' || failed::text, ',' ORDER BY target_version)
			FROM schema_backfills WHERE tenant_id = $1 AND entity_definition_id = $2), '') || ' | ' ||
		coalesce((SELECT string_agg(record_id::text, ',' ORDER BY record_id) FROM schema_backfill_pending WHERE tenant_id = $1 AND entity_definition_id = $2), '') || ' | ' ||
		coalesce((SELECT string_agg(record_id::text || ':' || code, ',' ORDER BY record_id) FROM schema_backfill_failures WHERE tenant_id = $1 AND entity_definition_id = $2), '')`,
		tenant, def).Scan(&s)
	return s, err
}

// s29HasPrio is whether record's stored body holds prio, past RLS.
func s29HasPrio(ctx context.Context, e *etIso, record string) (bool, error) {
	var has bool
	err := e.super.QueryRow(ctx, `SELECT data ? 'prio' FROM records WHERE id = $1`, record).Scan(&has)
	return has, err
}

// Row S29 4 (TICK-917, in TICK-916; ADR-009 D3, CI-3 (4)). The worker's own
// Materializer (recordports, the pipeline as cmd/nava-worker builds it), as
// nava_worker in tenant A's transaction:
//   - control: tenant A's a1, which lacks prio, is written;
//   - tenant B's b1 by id, as a forged pending row would name it, is not
//     written (the pipeline reads it under RLS and finds nothing);
//   - tenant B's b1 with tenant B as the target's (the actor's) tenant, in
//     tenant A's transaction, is not written either;
//   - pipeline.Write(OpMaterialize) on tenant A's own a1 with a patch, a
//     non-system source, a principal, or another actor_system is refused
//     with a typed code.
//
// Tenant B's records, entries and jobs do not move, and a1 holds nothing
// (every transaction rolls back).
func s29Materialize4(ctx context.Context, cl Cluster, e *etIso, fx *s29Fixture) error {
	pipe, err := cl.PoolSize("nava_worker", 2)
	if err != nil {
		return err
	}
	defer pipe.Close()
	pool, err := cl.PoolSize("nava_worker", 2)
	if err != nil {
		return err
	}
	defer pool.Close()
	m := recordports.NewMaterializer(pipe)
	a, b := uuid.MustParse(TenantA), uuid.MustParse(TenantB)
	beforeB, err := s29Snap(ctx, e, TenantB, fx.defB)
	if err != nil {
		return err
	}
	if err := inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
		written, err := m.Materialize(ctx, tx, schemabackfill.Target{Tenant: a, EntityKey: s29Key, RecordID: uuid.MustParse(fx.a1)})
		if err != nil || !written {
			return fmt.Errorf("control: materializing tenant A's a1 in tenant A = %v, %w; want written", written, s22Err(err))
		}
		return nil
	}); err != nil {
		return err
	}
	for _, t := range []struct {
		what   string
		target schemabackfill.Target
	}{
		{"tenant B's b1 by id", schemabackfill.Target{Tenant: a, EntityKey: s29Key, RecordID: uuid.MustParse(fx.b1)}},
		{"tenant B's b1 as tenant B", schemabackfill.Target{Tenant: b, EntityKey: s29Key, RecordID: uuid.MustParse(fx.b1)}},
	} {
		if err := inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
			written, err := m.Materialize(ctx, tx, t.target)
			if written {
				return fmt.Errorf("materializing %s in tenant A's transaction wrote it (err %w)", t.what, s22Err(err))
			}
			return nil
		}); err != nil {
			return err
		}
	}
	base := pipeline.Command{Op: pipeline.OpMaterialize, EntityKey: s29Key, RecordID: uuid.MustParse(fx.a1),
		Actor: identity.Principal{TenantID: a}, Source: pipeline.SourceSystem, ActorSystem: pipeline.ActorSchemaBackfill}
	for _, f := range []struct {
		what  string
		forge func(*pipeline.Command)
	}{
		{"a non-empty patch", func(c *pipeline.Command) { c.Data = map[string]any{"prio": "high"} }},
		{"a non-system source", func(c *pipeline.Command) { c.Source = pipeline.SourceWorkflow }},
		{"a principal", func(c *pipeline.Command) { c.Actor.PrincipalID = uuid.MustParse(principalA1) }},
		{"another actor_system", func(c *pipeline.Command) { c.ActorSystem = "interpreter" }},
		{"no actor_system", func(c *pipeline.Command) { c.ActorSystem = "" }},
	} {
		cmd := base
		f.forge(&cmd)
		if err := inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
			res, err := m.Pipeline.Write(ctx, tx, cmd)
			code := pipeline.ErrorCode(err)
			if err == nil || res.Written || code == "" || code == pipeline.CodeDatabaseError {
				return fmt.Errorf("OpMaterialize with %s = written %v, code %q (%w); want a typed refusal", f.what, res.Written, code, s22Err(err))
			}
			return nil
		}); err != nil {
			return err
		}
	}
	if has, err := s29HasPrio(ctx, e, fx.a1); err != nil || has {
		return fmt.Errorf("control: a1 holds prio after rolled-back writes (%v, %w)", has, s22Err(err))
	}
	if after, err := s29Snap(ctx, e, TenantB, fx.defB); err != nil || after != beforeB {
		return fmt.Errorf("tenant A's materializes moved tenant B (%w):\n%s\n%s", s22Err(err), beforeB, after)
	}
	return nil
}

// s29Run is one schema.backfill run on the Temporal SDK test environment,
// behind the worker guard and the nava-tenant Propagator as cmd/nava-worker
// runs it, with every activity argument, result and heartbeat in ledger.
type s29Run struct {
	env    *testsuite.TestWorkflowEnvironment
	mu     sync.Mutex
	logs   bytes.Buffer
	acts   map[string]int
	ledger []string
}

func (r *s29Run) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.logs.Write(p)
}

func (r *s29Run) note(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	r.mu.Lock()
	r.ledger = append(r.ledger, string(b))
	r.mu.Unlock()
}

func (r *s29Run) values(v converter.EncodedValues) {
	if v == nil || !v.HasValues() {
		return
	}
	var raw json.RawMessage
	if v.Get(&raw) == nil {
		r.note(raw)
	}
}

// newS29Run runs workflow id under header (the nava-tenant the relay's
// client writes from the outbox row's tenant).
func newS29Run(id string, header uuid.UUID, acts *schemabackfill.Activities) (*s29Run, error) {
	var s testsuite.WorkflowTestSuite
	r := &s29Run{env: s.NewTestWorkflowEnvironment(), acts: map[string]int{}}
	g, err := workerguard.New(slog.New(slog.NewJSONHandler(r, nil)), noop.NewMeterProvider())
	if err != nil {
		return nil, err
	}
	p, err := converter.GetDefaultDataConverter().ToPayload(header.String())
	if err != nil {
		return nil, err
	}
	r.env.SetWorkerOptions(workerguard.WorkerOptions(worker.Options{DeadlockDetectionTimeout: 30 * time.Second}, g))
	r.env.SetContextPropagators([]workflow.ContextPropagator{workerguard.Propagator{}})
	r.env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: id, TaskQueue: schemabackfill.TaskQueue})
	r.env.SetHeader(&commonpb.Header{Fields: map[string]*commonpb.Payload{workerguard.TenantHeader: p}})
	r.env.SetTestTimeout(3 * time.Minute)
	acts.Register(r.env)
	(&schemabackfill.Workflow{}).Register(r.env)
	r.env.SetOnActivityStartedListener(func(info *activity.Info, _ context.Context, args converter.EncodedValues) {
		r.mu.Lock()
		r.acts[info.ActivityType.Name]++
		r.mu.Unlock()
		r.values(args)
	})
	r.env.SetOnActivityCompletedListener(func(_ *activity.Info, result converter.EncodedValue, _ error) {
		if result == nil || !result.HasValue() {
			return
		}
		var raw json.RawMessage
		if result.Get(&raw) == nil {
			r.note(raw)
		}
	})
	r.env.SetOnActivityHeartbeatListener(func(_ *activity.Info, details converter.EncodedValues) { r.values(details) })
	return r, nil
}

func (r *s29Run) start(def uuid.UUID) error {
	d := outbox.Delivery{OutboxID: uuid.Must(uuid.NewV7()), KindVersion: 1,
		Payload: json.RawMessage(`{"entity_definition_id":"` + def.String() + `"}`)}
	r.note(d)
	r.env.ExecuteWorkflow(schemabackfill.WorkflowType, d)
	if !r.env.IsWorkflowCompleted() {
		return errors.New("control: the workflow did not complete")
	}
	return r.env.GetWorkflowError()
}

func (r *s29Run) count(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.acts[name]
}

// s29SeedJobs gives each tenant a pending job on its own iso_et_s29 at the
// current version, default_keys {prio}, and tenant B its queued b1 (past
// RLS); it returns the version. A job the v2 publish wrote (TICK-919) is
// reset to pending.
func s29SeedJobs(ctx context.Context, e *etIso, fx *s29Fixture) (int, error) {
	var version int
	if err := e.super.QueryRow(ctx, `SELECT current_version FROM entity_definitions WHERE id = $1`, fx.defA).Scan(&version); err != nil {
		return 0, err
	}
	var vb int
	if err := e.super.QueryRow(ctx, `SELECT current_version FROM entity_definitions WHERE id = $1`, fx.defB).Scan(&vb); err != nil || vb != version {
		return 0, fmt.Errorf("control: tenant B's iso_et_s29 is at v%d (%w), want v%d as tenant A's", vb, s22Err(err), version)
	}
	tx, err := e.super.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, s := range []struct {
		q    string
		args []any
	}{
		{`SET LOCAL session_replication_role = replica`, nil},
		{`INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, state, default_keys)
			VALUES ($1, $2, $5, 'pending', '{prio}'), ($3, $4, $5, 'pending', '{prio}')
			ON CONFLICT (tenant_id, entity_definition_id, target_version) DO UPDATE
			SET state = 'pending', default_keys = '{prio}', rewritten = 0, skipped = 0, failed = 0, started_at = NULL, finished_at = NULL`,
			[]any{TenantA, fx.defA, TenantB, fx.defB, version}},
		{`DELETE FROM schema_backfill_pending WHERE entity_definition_id IN ($1, $2)`, []any{fx.defA, fx.defB}},
		{`INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id) VALUES ($1, $2, $3, $4)`,
			[]any{TenantB, fx.defB, version, fx.b1}},
	} {
		if _, err := tx.Exec(ctx, s.q, s.args...); err != nil {
			return 0, fmt.Errorf("seed the S29 jobs: %w", err)
		}
	}
	return version, tx.Commit(ctx)
}

// Row S29 5 (TICK-920, CI-3 (3)). Both tenants hold iso_et_s29 under one key
// with a pending job on prio at the same version; tenant B's job has b1
// queued, and b1 lacks prio. The real schema.backfill workflow and
// activities, as nava_worker:
//  1. Control: t:A:schema.backfill:<tenant A's definition>:v under tenant
//     A's header ends; a1 and a3 now hold prio (a2 had it).
//  2. t:A:schema.backfill:<tenant B's definition>:v under tenant A's header,
//     the start a tenant A outbox row naming tenant B's definition gets from
//     the relay (row 6): admission finds no job under RLS and nothing fills.
//  3. t:B:…:<tenant B's definition>:v under tenant A's header: the worker
//     guard refuses it and nothing fills.
//
// Tenant B's records, entries, job, queue and failures do not move in any
// run, and no activity argument, result or heartbeat holds a record value.
func s29Job5(ctx context.Context, cl Cluster, e *etIso, fx *s29Fixture) error {
	version, err := s29SeedJobs(ctx, e, fx)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = e.super.Exec(context.WithoutCancel(ctx), `DELETE FROM audit_log WHERE subject_id IN ($1, $2)`, fx.defA, fx.defB)
	}()
	beforeB, err := s29Snap(ctx, e, TenantB, fx.defB)
	if err != nil {
		return err
	}
	pool, err := cl.PoolSize("nava_worker", 4)
	if err != nil {
		return err
	}
	defer pool.Close()
	pipe, err := cl.PoolSize("nava_worker", 2)
	if err != nil {
		return err
	}
	defer pipe.Close()
	acts := &schemabackfill.Activities{Pool: pool, Materializer: recordports.NewMaterializer(pipe)}
	a, b := uuid.MustParse(TenantA), uuid.MustParse(TenantB)
	defA, defB := uuid.MustParse(fx.defA), uuid.MustParse(fx.defB)
	var ledger []string

	// 1. Control.
	r, err := newS29Run(schemabackfill.WorkflowID(schemabackfill.Job{Tenant: a, Definition: defA, Version: version}), a, acts)
	if err != nil {
		return err
	}
	if err := r.start(defA); err != nil {
		return fmt.Errorf("control: tenant A's own job = %w, worker log %s", err, r.logs.String())
	}
	for _, rec := range []string{fx.a1, fx.a3} {
		if has, err := s29HasPrio(ctx, e, rec); err != nil || !has {
			return fmt.Errorf("control: tenant A's job left %s without prio (%w); counts %v", rec, s22Err(err), r.acts)
		}
	}
	ledger = append(ledger, r.ledger...)
	if after, err := s29Snap(ctx, e, TenantB, fx.defB); err != nil || after != beforeB {
		return fmt.Errorf("tenant A's job on its own type moved tenant B's same-key type (%w):\n%s\n%s", s22Err(err), beforeB, after)
	}

	// 2 and 3.
	for _, c := range []struct {
		what   string
		job    schemabackfill.Job
		header uuid.UUID
	}{
		{"t:A with tenant B's definition", schemabackfill.Job{Tenant: a, Definition: defB, Version: version}, a},
		{"t:B under tenant A's header", schemabackfill.Job{Tenant: b, Definition: defB, Version: version}, a},
	} {
		r, err := newS29Run(schemabackfill.WorkflowID(c.job), c.header, acts)
		if err != nil {
			return err
		}
		err = r.start(c.job.Definition)
		if n := r.count(schemabackfill.FillActivity) + r.count(schemabackfill.PageActivity) + r.count(schemabackfill.EndActivity); n != 0 {
			return fmt.Errorf("%s: fill, page or end ran %d times (workflow %w), want none", c.what, n, s22Err(err))
		}
		ledger = append(ledger, r.ledger...)
		if after, err := s29Snap(ctx, e, TenantB, fx.defB); err != nil || after != beforeB {
			return fmt.Errorf("%s moved tenant B (%w):\n%s\n%s", c.what, s22Err(err), beforeB, after)
		}
	}
	if len(ledger) == 0 {
		return errors.New("control: the ledger recorded nothing")
	}
	for _, entry := range ledger {
		for _, v := range []string{`"low"`, `"high"`, `"a1"`, `"a2"`, `"a3"`, `"b1"`, `"b2"`} {
			if strings.Contains(entry, v) {
				return fmt.Errorf("an activity payload carries the record value %s: %s", v, entry)
			}
		}
	}
	return nil
}

// s29Temporal is the relay's Temporal port, recording each start.
type s29Temporal struct {
	mu     sync.Mutex
	starts []s29Start
}

type s29Start struct {
	id, queue string
	input     string
}

func (t *s29Temporal) ExecuteWorkflow(_ context.Context, o client.StartWorkflowOptions, _ any, args ...any) (client.WorkflowRun, error) {
	in, _ := json.Marshal(args)
	t.mu.Lock()
	t.starts = append(t.starts, s29Start{id: o.ID, queue: o.TaskQueue, input: string(in)})
	t.mu.Unlock()
	return nil, nil
}

func (t *s29Temporal) SignalWorkflow(context.Context, string, string, string, any) error { return nil }

func (t *s29Temporal) SignalWithStartWorkflow(context.Context, string, string, any, client.StartWorkflowOptions, any, ...any) (client.WorkflowRun, error) {
	return nil, nil
}

func (t *s29Temporal) startsOf(prefix string) []s29Start {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.DeleteFunc(slices.Clone(t.starts), func(s s29Start) bool { return !strings.Contains(s.id, prefix) })
}

// Row S29 6 (TICK-918, in TICK-920; contract C3). Three schema.backfill
// start rows of tenant A, through the only enqueue path
// (outbox.EnqueueSchemaBackfill, as nava_app in tenant A): one naming tenant
// B's definition, one naming tenant A's (the control), and one naming tenant
// A's whose payload is then forged, past RLS, to carry tenant B's tenant_id.
// The real Relay and Dispatcher run as nava_relay in one transaction that is
// rolled back, with a Temporal port that records each start:
//   - the forged row is marked dead and never reaches Temporal;
//   - the row naming tenant B's definition starts t:<tenant A>:… (the row's
//     tenant, never the payload's) on the schema-backfill queue; row 5 shows
//     that job finds nothing;
//   - the control starts t:<tenant A>:…:<tenant A's definition>;
//   - no start input names a tenant, and no start is under tenant B.
func s29Relay6(ctx context.Context, cl Cluster, e *etIso, fx *s29Fixture) error {
	app, err := cl.Pool("nava_app")
	if err != nil {
		return err
	}
	defA, defB := uuid.MustParse(fx.defA), uuid.MustParse(fx.defB)
	var rowB, rowA, rowF uuid.UUID
	if err := tenancy.InTenantTx(ctx, app, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
		var err error
		if rowB, err = outbox.EnqueueSchemaBackfill(ctx, tx, defB, 7531); err != nil {
			return err
		}
		if rowA, err = outbox.EnqueueSchemaBackfill(ctx, tx, defA, 7532); err != nil {
			return err
		}
		rowF, err = outbox.EnqueueSchemaBackfill(ctx, tx, defA, 7533)
		return err
	}); err != nil {
		return fmt.Errorf("control: enqueue the start rows in tenant A: %w", err)
	}
	if _, err := e.super.Exec(ctx, `UPDATE outbox SET payload = payload || jsonb_build_object('tenant_id', $2::text) WHERE id = $1`, rowF, TenantB); err != nil {
		return fmt.Errorf("forge the payload: %w", err)
	}
	relayPool, err := cl.PoolSize("nava_relay", 1)
	if err != nil {
		return err
	}
	defer relayPool.Close()
	tx, err := relayPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	log := slog.New(slog.DiscardHandler)
	rl, err := relay.New(tx, noop.NewMeterProvider(), log, relay.Options{})
	if err != nil {
		return err
	}
	tc := &s29Temporal{}
	d, err := relay.NewDispatcher(rl, tc, tracenoop.NewTracerProvider(), noop.NewMeterProvider(), log, relay.DispatcherOptions{Concurrency: 1})
	if err != nil {
		return err
	}
	settled := func(id uuid.UUID) (dead, dispatched bool, err error) {
		err = tx.QueryRow(ctx, `SELECT dead_at IS NOT NULL, dispatched_at IS NOT NULL FROM outbox WHERE id = $1`, id).Scan(&dead, &dispatched)
		return dead, dispatched, err
	}
	for range 50 {
		if _, err := d.Poll(ctx); err != nil {
			return fmt.Errorf("relay poll: %w", err)
		}
		done := true
		for _, id := range []uuid.UUID{rowB, rowA, rowF} {
			dead, dispatched, err := settled(id)
			if err != nil {
				return err
			}
			done = done && (dead || dispatched)
		}
		if done {
			break
		}
	}
	if dead, dispatched, err := settled(rowF); err != nil || !dead || dispatched {
		return fmt.Errorf("the forged row (payload tenant_id is tenant B): dead %v, dispatched %v (%w); want dead", dead, dispatched, s22Err(err))
	}
	wantA := schemabackfill.WorkflowID(schemabackfill.Job{Tenant: uuid.MustParse(TenantA), Definition: defA, Version: 7532})
	wantB := schemabackfill.WorkflowID(schemabackfill.Job{Tenant: uuid.MustParse(TenantA), Definition: defB, Version: 7531})
	for _, w := range []string{wantA, wantB} {
		got := tc.startsOf(w)
		if len(got) != 1 || got[0].queue != schemabackfill.TaskQueue {
			return fmt.Errorf("starts of %s = %+v, want one on %s", w, got, schemabackfill.TaskQueue)
		}
		if strings.Contains(got[0].input, "tenant") {
			return fmt.Errorf("the start input of %s names a tenant: %s", w, got[0].input)
		}
	}
	if forged := tc.startsOf(":7533"); len(forged) != 0 {
		return fmt.Errorf("the forged row reached Temporal: %+v", forged)
	}
	if underB := tc.startsOf("t:" + TenantB + ":" + schemabackfill.WorkflowType); len(underB) != 0 {
		return fmt.Errorf("a tenant A row started a tenant B job: %+v", underB)
	}
	return nil
}
