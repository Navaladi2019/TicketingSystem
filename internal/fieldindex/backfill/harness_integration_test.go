//go:build integration

package backfill_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The TICK-745 integration tests run the real coordinator, FieldIndexBackfill
// and activities against a real PostgreSQL (NAVA_TEST_PG_URL, make
// test-integration), on the Temporal SDK test environment behind the whole
// worker guard and the nava-tenant Propagator, as cmd/nava-worker runs them.
// Every test is therefore also a guard round-trip (QA item 4): a result,
// child input or continue-as-new input the guard refused would fail the run.

type itEnv struct {
	super  *pgxpool.Pool // the platform database as superuser: seeding, reading back
	app    *pgxpool.Pool // nava_app: the live writers
	worker *pgxpool.Pool // nava_worker: the activities
}

//nolint:gochecknoglobals // one migrated cluster per package run
var (
	envOnce sync.Once
	envErr  error
	shared  itEnv
)

func testPassword(role string) (string, bool) { return "test-" + role, true }

func migrated(t *testing.T) itEnv {
	t.Helper()
	url := os.Getenv("NAVA_TEST_PG_URL")
	if url == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	envOnce.Do(func() { shared, envErr = setup(url) })
	if envErr != nil {
		t.Fatal(envErr)
	}
	return shared
}

func setup(url string) (itEnv, error) {
	ctx := context.Background()
	super, err := pgx.ParseConfig(url)
	if err != nil {
		return itEnv{}, err
	}
	m, am, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		return itEnv{}, err
	}
	conn, err := pgx.ConnectConfig(ctx, super)
	if err != nil {
		return itEnv{}, err
	}
	for _, d := range m.DatabaseNames() {
		if _, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{d}.Sanitize()+" WITH (FORCE)"); err != nil {
			return itEnv{}, err
		}
	}
	for _, r := range m.RoleNames() {
		if _, err := conn.Exec(ctx, "DROP ROLE IF EXISTS "+pgx.Identifier{r}.Sanitize()); err != nil {
			return itEnv{}, err
		}
	}
	_ = conn.Close(ctx)
	log := slog.New(slog.DiscardHandler)
	if _, err := roles.Run(ctx, super, m, testPassword, log); err != nil {
		return itEnv{}, err
	}
	g, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		return itEnv{}, err
	}
	owner := super.Copy()
	owner.User = m.Owner()
	owner.Password, _ = testPassword(m.Owner())
	owner.Database = m.PlatformDatabase
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, am, g, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
		return itEnv{}, err
	}
	pool := func(user string, maxConns int32) (*pgxpool.Pool, error) {
		cfg, err := pgxpool.ParseConfig("")
		if err != nil {
			return nil, err
		}
		cfg.ConnConfig = super.Copy()
		cfg.ConnConfig.Database = m.PlatformDatabase
		if user != "" {
			cfg.ConnConfig.User = user
			cfg.ConnConfig.Password, _ = testPassword(user)
		}
		cfg.MaxConns = maxConns
		return pgxpool.NewWithConfig(ctx, cfg)
	}
	var e itEnv
	if e.super, err = pool("", 8); err != nil {
		return itEnv{}, err
	}
	if e.app, err = pool("nava_app", 8); err != nil {
		return itEnv{}, err
	}
	if e.worker, err = pool("nava_worker", 8); err != nil {
		return itEnv{}, err
	}
	return e, nil
}

func mustExec(t testing.TB, p *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := p.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func queryString(t testing.TB, p *pgxpool.Pool, sql string, args ...any) string {
	t.Helper()
	var s string
	if err := p.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return s
}

// fixture is one tenant with one entity type and one flagged field.
type fixture struct {
	tenant, def, field uuid.UUID
	key                string
}

// newFixture seeds, as the superuser, an active tenant and an entity type
// (with its config entry, so the commit check passes), and a field of value
// type vt registered in state at requested_version 1, as a publish (P7-04)
// would.
func newFixture(t testing.TB, e itEnv, vt string, state fieldindex.State) fixture {
	t.Helper()
	f := fixture{tenant: uuid.New(), def: uuid.New(), field: uuid.New(), key: "f_" + vt}
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", f.tenant)
	mustExec(t, e.super, `WITH d AS (INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
		VALUES ($1, $2, 'incident', 'Incident', 'INC', 1) RETURNING tenant_id, id)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source, content_sha256)
		SELECT tenant_id, gen_random_uuid(), 'config', 'create', 'entity_definition', id, 'onboarding', 'onboarding', sha256(''::bytea) FROM d`,
		f.tenant, f.def)
	if state != "" {
		f.register(t, e, vt, state)
	}
	return f
}

// register inserts the registry row: the flip to backfilling of a publish.
func (f fixture) register(t testing.TB, e itEnv, vt string, state fieldindex.State) {
	t.Helper()
	mustExec(t, e.super, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, requested_version)
		VALUES ($1, $2, $3, $4, $5, $6, 1)`, f.tenant, f.field, f.def, f.key, vt, string(state))
}

// insertRecordSQL writes one record and its audit entry in one statement
// (the records commit check): tenant $1, id $2, type $3, number $4, data $5.
const insertRecordSQL = `WITH r AS (INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, data)
	VALUES ($1, $2, $3, 1, $4, 'open', $5::jsonb) RETURNING tenant_id, id, version)
	INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
	SELECT tenant_id, gen_random_uuid(), 'record', 'create', 'record', id, version, 'system', 'system' FROM r`

// updateRecordSQL sets the field to $3 (a JSON value) on record $2, with
// its audit entry.
const updateRecordSQL = `WITH r AS (UPDATE records SET data = jsonb_set(data, ARRAY[$4::text], $3::jsonb), version = version + 1
	WHERE tenant_id = $1 AND id = $2 RETURNING tenant_id, id, version)
	INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
	SELECT tenant_id, gen_random_uuid(), 'record', 'update', 'record', id, version, 'system', 'system' FROM r`

// valueSecret marks the text values the fixtures write: CI-5c asserts it
// never reaches a payload.
const valueSecret = "fieldvalue-secret-"

func recordData(f fixture, i int) string {
	return fmt.Sprintf(`{%q: %d, "f_text": %q, "other": "x"}`, f.key, i, fmt.Sprintf("%s%d", valueSecret, i))
}

// insertRecords writes n records of f's type as nava_app, one transaction
// each, and returns their ids in order.
func insertRecords(t testing.TB, e itEnv, f fixture, n int) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.Must(uuid.NewV7())
		if err := writeRecord(context.Background(), e, f, ids[i], i); err != nil {
			t.Fatal(err)
		}
	}
	return ids
}

func writeRecord(ctx context.Context, e itEnv, f fixture, id uuid.UUID, i int) error {
	return tenancy.InTenantTx(ctx, e.app, f.tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, insertRecordSQL, f.tenant, id, f.def, fmt.Sprintf("INC-%s", id.String()[24:]), recordData(f, i))
		return err
	})
}

// sideDiff is CI-4's recompute check for one field: the side rows that a
// recompute from records.data has and the side tables lack (missing), and
// the other way round (extra). Read as the superuser.
const sideDiff = `
WITH f AS (SELECT tenant_id, id, entity_definition_id, field_key, value_type FROM record_index_fields WHERE tenant_id = $1 AND id = $2),
want AS (
    SELECT r.id AS record_id, c.ord, coalesce(c.value_text, c.value_num::text, c.value_ts::text, c.value_bool::text) AS value
    FROM records r JOIN f ON f.tenant_id = r.tenant_id AND f.entity_definition_id = r.entity_definition_id
    CROSS JOIN LATERAL record_field_coerce(r.data -> f.field_key, f.value_type) c
), got AS (
    SELECT record_id, ord, value::text FROM rfv_text WHERE tenant_id = $1 AND field_id = $2
    UNION ALL SELECT record_id, ord, value::text FROM rfv_num WHERE tenant_id = $1 AND field_id = $2
    UNION ALL SELECT record_id, ord, value::text FROM rfv_ts WHERE tenant_id = $1 AND field_id = $2
    UNION ALL SELECT record_id, ord, value::text FROM rfv_bool WHERE tenant_id = $1 AND field_id = $2
), d AS (
    (SELECT 'missing' AS kind, * FROM want EXCEPT ALL SELECT 'missing', * FROM got)
    UNION ALL
    (SELECT 'extra', * FROM got EXCEPT ALL SELECT 'extra', * FROM want)
)
SELECT count(*) FILTER (WHERE kind = 'missing') || ' missing, ' || count(*) FILTER (WHERE kind = 'extra') || ' extra' FROM d`

const noDiff = "0 missing, 0 extra"

func diffOf(t testing.TB, e itEnv, f fixture) string {
	t.Helper()
	return queryString(t, e.super, sideDiff, f.tenant, f.field)
}

func sideRows(t testing.TB, e itEnv, f fixture) string {
	t.Helper()
	return queryString(t, e.super, `SELECT count(*)::text FROM rfv_num WHERE tenant_id = $1 AND field_id = $2`, f.tenant, f.field)
}

type registryState struct {
	state     string
	progress  int16
	version   int
	updatedAt time.Time
}

func registry(t testing.TB, e itEnv, f fixture) registryState {
	t.Helper()
	var r registryState
	if err := e.super.QueryRow(context.Background(), `SELECT state, progress, requested_version, updated_at
		FROM record_index_fields WHERE tenant_id = $1 AND id = $2`, f.tenant, f.field).
		Scan(&r.state, &r.progress, &r.version, &r.updatedAt); err != nil {
		t.Fatal(err)
	}
	return r
}

// run is one workflow execution on the SDK test environment, behind the
// guard, with every payload it recorded in its ledger.
type run struct {
	t      *testing.T
	env    *testsuite.TestWorkflowEnvironment
	reader *sdkmetric.ManualReader
	logs   *lockedWriter

	mu       sync.Mutex
	ledger   []json.RawMessage // every argument, result, input and heartbeat
	children []string          // "start <id>" and "done <id>", in order
	batches  map[string]int    // Batch activity completions by workflow id
	results  []backfill.Cursor // every Batch result, in order
}

type runOptions struct {
	acts        *backfill.Activities
	coordinator *backfill.Coordinator
	backfill    *backfill.Backfill
	// barrier, when set, replaces fieldindex.Barrier (the test seam).
	barrier func(context.Context) error
	// remove, when set, is registered as FieldIndexRemove instead of the
	// real one, remover (zero: the defaults).
	remove  func(workflow.Context, backfill.FieldRef) error
	remover *backfill.Remove
	// onBatch runs after each Batch activity completes, with its workflow id.
	onBatch func(workflowID string)
	// onHeartbeat runs on each Barrier heartbeat.
	onHeartbeat func()
	workflowID  string
}

func tenantHeader(t testing.TB, tenant uuid.UUID) *commonpb.Header {
	t.Helper()
	p, err := converter.GetDefaultDataConverter().ToPayload(tenant.String())
	if err != nil {
		t.Fatal(err)
	}
	return &commonpb.Header{Fields: map[string]*commonpb.Payload{workerguard.TenantHeader: p}}
}

func newRun(t *testing.T, tenant uuid.UUID, o runOptions) *run {
	t.Helper()
	var s testsuite.WorkflowTestSuite
	r := &run{t: t, env: s.NewTestWorkflowEnvironment(), reader: sdkmetric.NewManualReader(),
		logs: &lockedWriter{w: &bytes.Buffer{}}, batches: map[string]int{}}
	g, err := workerguard.New(slog.New(slog.NewJSONHandler(r.logs, nil)),
		sdkmetric.NewMeterProvider(sdkmetric.WithReader(r.reader)))
	if err != nil {
		t.Fatal(err)
	}
	header := tenantHeader(t, tenant)
	r.env.SetWorkerOptions(workerguard.WorkerOptions(worker.Options{
		Interceptors:             []interceptor.WorkerInterceptor{&relayHeader{tenant: header.Fields[workerguard.TenantHeader]}},
		DeadlockDetectionTimeout: 30 * time.Second,
	}, g))
	r.env.SetContextPropagators([]workflow.ContextPropagator{workerguard.Propagator{}})
	id := o.workflowID
	if id == "" {
		id = fieldindex.CoordinatorWorkflowID(tenant.String())
	}
	r.env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: id})
	r.env.SetHeader(header)
	r.env.SetTestTimeout(5 * time.Minute)

	acts := o.acts
	acts.Register(r.env)
	if o.barrier != nil {
		r.env.RegisterActivityWithOptions(o.barrier, activity.RegisterOptions{Name: backfill.BarrierActivity})
	}
	c := o.coordinator
	if c == nil {
		c = &backfill.Coordinator{}
	}
	c.Register(r.env)
	b := o.backfill
	if b == nil {
		b = &backfill.Backfill{}
	}
	b.Register(r.env)
	if o.remove != nil {
		r.env.RegisterWorkflowWithOptions(o.remove, workflow.RegisterOptions{Name: fieldindex.RemoveWorkflowType})
	} else {
		rm := o.remover
		if rm == nil {
			rm = &backfill.Remove{}
		}
		rm.Register(r.env)
	}

	r.env.SetOnActivityStartedListener(func(_ *activity.Info, _ context.Context, args converter.EncodedValues) {
		r.record(args)
	})
	r.env.SetOnActivityCompletedListener(func(info *activity.Info, result converter.EncodedValue, _ error) {
		r.recordOne(result)
		if info.ActivityType.Name == backfill.BatchActivity {
			var c backfill.Cursor
			if result != nil && result.HasValue() {
				_ = result.Get(&c)
			}
			r.mu.Lock()
			r.batches[info.WorkflowExecution.ID]++
			r.results = append(r.results, c)
			r.mu.Unlock()
			if o.onBatch != nil {
				o.onBatch(info.WorkflowExecution.ID)
			}
		}
	})
	r.env.SetOnActivityHeartbeatListener(func(info *activity.Info, details converter.EncodedValues) {
		r.record(details)
		if info.ActivityType.Name == backfill.BarrierActivity && o.onHeartbeat != nil {
			o.onHeartbeat()
		}
	})
	r.env.SetOnChildWorkflowStartedListener(func(info *workflow.Info, _ workflow.Context, args converter.EncodedValues) {
		r.record(args)
		r.mu.Lock()
		r.children = append(r.children, "start "+info.WorkflowExecution.ID)
		r.mu.Unlock()
	})
	r.env.SetOnChildWorkflowCompletedListener(func(info *workflow.Info, result converter.EncodedValue, _ error) {
		r.recordOne(result)
		r.mu.Lock()
		r.children = append(r.children, "done "+info.WorkflowExecution.ID)
		r.mu.Unlock()
	})
	return r
}

func (r *run) record(v converter.EncodedValues) {
	if v == nil || !v.HasValues() {
		return
	}
	// Decode up to four values as raw JSON; an activity here takes at most one.
	raws := make([]json.RawMessage, 4)
	ptrs := make([]any, 0, 4)
	for i := range raws {
		ptrs = append(ptrs, &raws[i])
	}
	for n := len(ptrs); n > 0; n-- {
		if v.Get(ptrs[:n]...) == nil {
			r.mu.Lock()
			for _, raw := range raws[:n] {
				r.ledger = append(r.ledger, append(json.RawMessage(nil), raw...))
			}
			r.mu.Unlock()
			return
		}
	}
	r.t.Errorf("a recorded value does not decode as JSON")
}

func (r *run) recordOne(v converter.EncodedValue) {
	if v == nil || !v.HasValue() {
		return
	}
	var raw json.RawMessage
	if err := v.Get(&raw); err != nil {
		r.t.Errorf("a recorded value does not decode as JSON: %v", err)
		return
	}
	r.mu.Lock()
	r.ledger = append(r.ledger, raw)
	r.mu.Unlock()
}

// signal sends item to the coordinator after delay (test time).
func (r *run) signal(item fieldindex.EnqueueItem, delay time.Duration) {
	r.mu.Lock()
	b, _ := json.Marshal(item)
	r.ledger = append(r.ledger, b)
	r.mu.Unlock()
	r.env.RegisterDelayedCallback(func() { r.env.SignalWorkflow(fieldindex.SignalEnqueue, item) }, delay)
}

// coordinate runs the coordinator with input in (zero: the relay's first
// start) and returns its error.
func (r *run) coordinate(in workerguard.RunInput) error {
	r.t.Helper()
	if in.Payload == nil {
		in.Delivery = outbox.Delivery{OutboxID: uuid.Must(uuid.NewV7()), KindVersion: 1, Payload: json.RawMessage(`{}`)}
	}
	r.env.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, in)
	if !r.env.IsWorkflowCompleted() {
		r.t.Fatal("the coordinator did not complete")
	}
	return r.env.GetWorkflowError()
}

// guardDenies returns the guard's non-allow decisions by guard/outcome.
func (r *run) guardDenies() map[string]int64 {
	r.t.Helper()
	var rm metricdata.ResourceMetrics
	if err := r.reader.Collect(context.Background(), &rm); err != nil {
		r.t.Fatal(err)
	}
	out := map[string]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != workerguard.DecisionsMetric {
				continue
			}
			for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
				g, _ := dp.Attributes.Value("guard")
				o, _ := dp.Attributes.Value("outcome")
				if o.AsString() != "allow" && dp.Value > 0 {
					out[g.AsString()+"/"+o.AsString()] += dp.Value
				}
			}
		}
	}
	return out
}

// requireGuardClean fails on any guard deny.
func (r *run) requireGuardClean() {
	r.t.Helper()
	if d := r.guardDenies(); len(d) > 0 {
		r.t.Fatalf("the worker guard denied %v; logs: %s", d, r.logText())
	}
}

func (r *run) logText() string {
	r.logs.mu.Lock()
	defer r.logs.mu.Unlock()
	return r.logs.w.String()
}

func (r *run) childLog() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.children...)
}

func item(f fixture, version int) fieldindex.EnqueueItem {
	return fieldindex.EnqueueItem{Op: fieldindex.OpBackfill, FieldID: f.field.String(), RequestedVersion: version,
		OutboxID: uuid.Must(uuid.NewV7()).String()}
}

func backfillChild(f fixture, version int) string {
	return backfill.ChildID(f.tenant, fieldindex.BackfillWorkflowType, f.field, version)
}

// removeItem is the coordinator item of f's field_index.remove row.
func removeItem(f fixture, version int) fieldindex.EnqueueItem {
	it := item(f, version)
	it.Op = fieldindex.OpRemove
	return it
}

func removeChild(f fixture, version int) string {
	return backfill.ChildID(f.tenant, fieldindex.RemoveWorkflowType, f.field, version)
}

// lockedWriter serialises writes to a buffer shared with the test.
type lockedWriter struct {
	mu sync.Mutex
	w  *bytes.Buffer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// relayHeader stands in for the transport: the relay's client Propagator
// writes the nava-tenant header on its SignalWithStart, and the server
// delivers it with the signal. The SDK test environment's SignalWorkflow
// sends none, so this interceptor, ahead of the guard, sets the tenant's.
type relayHeader struct {
	interceptor.WorkerInterceptorBase
	tenant *commonpb.Payload
}

func (h *relayHeader) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &relayHeaderWorkflow{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, tenant: h.tenant}
}

type relayHeaderWorkflow struct {
	interceptor.WorkflowInboundInterceptorBase
	tenant *commonpb.Payload
}

func (w *relayHeaderWorkflow) HandleSignal(ctx workflow.Context, in *interceptor.HandleSignalInput) error {
	if h := interceptor.WorkflowHeader(ctx); h != nil {
		h[workerguard.TenantHeader] = w.tenant
	}
	return w.Next.HandleSignal(ctx, in)
}
