//go:build integration

package schemabackfill_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
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
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/schemabackfill"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
)

// The TICK-920 integration tests run the real schema.backfill workflow and
// activities against a real PostgreSQL (NAVA_TEST_PG_URL, make
// test-integration), as nava_worker, on the Temporal SDK test environment
// behind the whole worker guard and the nava-tenant Propagator, as
// cmd/nava-worker runs them. Every run is therefore also a guard round trip:
// an argument, result or heartbeat the guard refused would fail it.

type itEnv struct {
	super  *pgxpool.Pool // seeding and reading back
	app    *pgxpool.Pool // nava_app: publishes and live writers
	worker *pgxpool.Pool // nava_worker: the activities
	pipe   *pgxpool.Pool // nava_worker: the pipeline's own runner
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
	if e.pipe, err = pool("nava_worker", 4); err != nil {
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

func count(t testing.TB, p *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := p.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

const entityKey = "incident"

// The incident type: v1 with no channel; v2 adds channel with a default; v3
// adds source with a default too.
const (
	schemaV1 = `{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "additionalProperties": false,
  "required": ["title", "priority"],
  "properties": {"title": {"type": "string", "minLength": 1}, "status": {"type": "string", "enum": ["open", "closed"]},
    "priority": {"type": "string", "enum": ["low", "high"], "default": "low"}, "summary": {"type": "string"}}}`
	schemaV2 = `{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "additionalProperties": false,
  "required": ["title", "priority"],
  "properties": {"title": {"type": "string", "minLength": 1}, "status": {"type": "string", "enum": ["open", "closed"]},
    "priority": {"type": "string", "enum": ["low", "high"], "default": "low"}, "summary": {"type": "string"},
    "channel": {"type": "string", "default": "email"}}}`
	schemaV3 = `{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "additionalProperties": false,
  "required": ["title", "priority"],
  "properties": {"title": {"type": "string", "minLength": 1}, "status": {"type": "string", "enum": ["open", "closed"]},
    "priority": {"type": "string", "enum": ["low", "high"], "default": "low"}, "summary": {"type": "string"},
    "channel": {"type": "string", "default": "email"}, "source": {"type": "string", "default": "web"}}}`
)

func definition(schema string) metadata.Definition {
	return metadata.Definition{
		Key: entityKey, Schema: json.RawMessage(schema), NumberPrefix: "INC",
		FixedColumns: map[string]string{"status": metadata.ColumnState},
		States: metadata.StateModel{Initial: "open", States: []string{"open", "closed"},
			Transitions: []metadata.Transition{{From: "open", To: "closed"}}},
	}
}

// fixture is one tenant with the incident type at v1.
type fixture struct {
	tenant, principal, def uuid.UUID
}

func newFixture(t *testing.T, e itEnv) fixture {
	t.Helper()
	f := fixture{tenant: uuid.New(), principal: uuid.New()}
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", f.tenant)
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
		VALUES ($1, $2, $3, true, now())`, f.tenant, f.principal, "kc-"+f.principal.String())
	f.inApp(t, e, func(tx pgx.Tx) error {
		_, err := entitytypes.Store{}.Create(context.Background(), tx, f.tenant, entitytypes.NewInput{
			Name: entityKey, PublishedBy: f.principal, Definition: definition(schemaV1)})
		return err
	})
	if err := e.super.QueryRow(context.Background(), "SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2",
		f.tenant, entityKey).Scan(&f.def); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f fixture) inApp(t testing.TB, e itEnv, fn func(pgx.Tx) error) {
	t.Helper()
	if err := tenancy.InTenantTx(context.Background(), e.app, f.tenant, fn); err != nil {
		t.Fatal(err)
	}
}

// publish publishes schema as the next version and returns the publish's
// audit entry id (the job's causation, D2).
func (f fixture) publish(t testing.TB, e itEnv, schema string) uuid.UUID {
	t.Helper()
	f.inApp(t, e, func(tx pgx.Tx) error {
		_, err := entitytypes.Store{}.Publish(context.Background(), tx, f.tenant, entitytypes.PublishInput{
			Key: entityKey, PublishedBy: f.principal, Definition: definition(schema)})
		return err
	})
	var id uuid.UUID
	if err := e.super.QueryRow(context.Background(), `SELECT id FROM audit_log WHERE tenant_id = $1 AND subject_id = $2
		AND action = 'publish' ORDER BY occurred_at DESC, id DESC LIMIT 1`, f.tenant, f.def).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// queueJob writes the job row the publish (P10-04) writes, pending, with
// keys and causation, and returns the job.
func (f fixture) queueJob(t testing.TB, e itEnv, version int, causation *uuid.UUID, keys ...string) schemabackfill.Job {
	t.Helper()
	mustExec(t, e.super, `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, default_keys, causation_audit_id)
		VALUES ($1, $2, $3, $4, $5)`, f.tenant, f.def, version, keys, causation)
	return schemabackfill.Job{Tenant: f.tenant, Definition: f.def, Version: version}
}

// insertRecordSQL writes one v1 record and its create entry (the records
// commit check), as a record written before the default existed:
// tenant $1, id $2, type $3, number $4, data $5.
const insertRecordSQL = `WITH r AS (INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state, data)
	VALUES ($1, $2, $3, 1, 1, $4, 'open', $5::jsonb) RETURNING tenant_id, id, version)
	INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
	SELECT tenant_id, gen_random_uuid(), 'record', 'create', 'record', id, version, 'system', 'system' FROM r`

// valueSecret marks every value the fixtures write: no payload of a run may
// carry it.
const valueSecret = "recordvalue-secret-"

// insertRecords writes n records with data(i), one statement each, and
// returns their ids in order.
func (f fixture) insertRecords(t testing.TB, e itEnv, n int, data func(i int) map[string]any) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.Must(uuid.NewV7())
		d := data(i)
		raw, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		mustExec(t, e.super, insertRecordSQL, f.tenant, ids[i], f.def, "INC-"+ids[i].String()[24:], string(raw))
	}
	return ids
}

func plain(i int) map[string]any {
	return map[string]any{"title": fmt.Sprintf("%s%d", valueSecret, i), "priority": "low"}
}

// stored is a record as stored.
type stored struct {
	data          map[string]any
	version       int64
	schemaVersion int
	updatedAt     time.Time
}

func record(t testing.TB, e itEnv, id uuid.UUID) stored {
	t.Helper()
	var s stored
	var raw []byte
	if err := e.super.QueryRow(context.Background(), `SELECT data, version, schema_version, updated_at FROM records WHERE id = $1`, id).
		Scan(&raw, &s.version, &s.schemaVersion, &s.updatedAt); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		t.Fatal(err)
	}
	return s
}

// jobState is the job row as stored.
type jobState struct {
	state                      string
	rewritten, skipped, failed int64
	started, finished          *time.Time
}

func job(t testing.TB, e itEnv, j schemabackfill.Job) jobState {
	t.Helper()
	var s jobState
	if err := e.super.QueryRow(context.Background(), `SELECT state, rewritten, skipped, failed, started_at, finished_at
		FROM schema_backfills WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3`,
		j.Tenant, j.Definition, j.Version).Scan(&s.state, &s.rewritten, &s.skipped, &s.failed, &s.started, &s.finished); err != nil {
		t.Fatal(err)
	}
	return s
}

func pending(t testing.TB, e itEnv, j schemabackfill.Job) int {
	t.Helper()
	return count(t, e.super, `SELECT count(*) FROM schema_backfill_pending
		WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3`, j.Tenant, j.Definition, j.Version)
}

// materializeEntries counts the rewrite entries of record id (D4).
func materializeEntries(t testing.TB, e itEnv, id uuid.UUID) int {
	t.Helper()
	return count(t, e.super, `SELECT count(*) FROM audit_log WHERE subject_type = 'record' AND subject_id = $1
		AND actor_system = 'schema-backfill'`, id)
}

// jobEntries is the job's entity-type configuration entries' diffs, oldest
// first (D4).
func jobEntries(t testing.TB, e itEnv, f fixture) []string {
	t.Helper()
	rows, err := e.super.Query(context.Background(), `SELECT diff::text FROM audit_log WHERE tenant_id = $1
		AND subject_type = 'entity_definition' AND subject_id = $2 AND actor_system = 'schema-backfill'
		ORDER BY occurred_at, id`, f.tenant, f.def)
	if err != nil {
		t.Fatal(err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// lockedWriter is a log sink the guard and the activities share.
type lockedWriter struct {
	mu sync.Mutex
	w  *bytes.Buffer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

func (l *lockedWriter) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.String()
}

// run is one job's workflow on the SDK test environment, behind the guard,
// with every payload it recorded in its ledger.
type run struct {
	t       *testing.T
	env     *testsuite.TestWorkflowEnvironment
	acts    *schemabackfill.Activities
	reader  *sdkmetric.ManualReader
	logs    *lockedWriter
	mu      sync.Mutex
	ledger  []json.RawMessage
	started map[string]int // activity starts by name
}

func newRun(t *testing.T, e itEnv, j schemabackfill.Job, wf *schemabackfill.Workflow) *run {
	t.Helper()
	var s testsuite.WorkflowTestSuite
	r := &run{t: t, env: s.NewTestWorkflowEnvironment(), reader: sdkmetric.NewManualReader(),
		logs: &lockedWriter{w: &bytes.Buffer{}}, started: map[string]int{}}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r.reader))
	log := slog.New(slog.NewJSONHandler(r.logs, nil))
	g, err := workerguard.New(log, mp)
	if err != nil {
		t.Fatal(err)
	}
	m, err := schemabackfill.NewMetrics(mp)
	if err != nil {
		t.Fatal(err)
	}
	p, err := converter.GetDefaultDataConverter().ToPayload(j.Tenant.String())
	if err != nil {
		t.Fatal(err)
	}
	header := &commonpb.Header{Fields: map[string]*commonpb.Payload{workerguard.TenantHeader: p}}
	r.env.SetWorkerOptions(workerguard.WorkerOptions(worker.Options{
		DeadlockDetectionTimeout: 30 * time.Second,
	}, g))
	r.env.SetContextPropagators([]workflow.ContextPropagator{workerguard.Propagator{}})
	r.env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: schemabackfill.WorkflowID(j), TaskQueue: schemabackfill.TaskQueue})
	r.env.SetHeader(header)
	r.env.SetTestTimeout(5 * time.Minute)
	r.acts = &schemabackfill.Activities{Pool: e.worker, Materializer: recordports.NewMaterializer(e.pipe), Metrics: m, Log: log}
	r.acts.Register(r.env)
	if wf == nil {
		wf = &schemabackfill.Workflow{}
	}
	wf.Register(r.env)
	r.env.SetOnActivityStartedListener(func(info *activity.Info, _ context.Context, args converter.EncodedValues) {
		r.mu.Lock()
		r.started[info.ActivityType.Name]++
		r.mu.Unlock()
		r.record(args)
	})
	r.env.SetOnActivityCompletedListener(func(_ *activity.Info, result converter.EncodedValue, _ error) {
		r.recordOne(result)
	})
	r.env.SetOnActivityHeartbeatListener(func(_ *activity.Info, details converter.EncodedValues) { r.record(details) })
	return r
}

func (r *run) record(v converter.EncodedValues) {
	if v == nil || !v.HasValues() {
		return
	}
	var raw json.RawMessage
	if v.Get(&raw) == nil {
		r.mu.Lock()
		r.ledger = append(r.ledger, append(json.RawMessage(nil), raw...))
		r.mu.Unlock()
	}
}

func (r *run) recordOne(v converter.EncodedValue) {
	if v == nil || !v.HasValue() {
		return
	}
	var raw json.RawMessage
	if v.Get(&raw) == nil {
		r.mu.Lock()
		r.ledger = append(r.ledger, raw)
		r.mu.Unlock()
	}
}

// start runs the workflow to completion with the relay's Delivery for j.
func (r *run) start(j schemabackfill.Job) error {
	r.t.Helper()
	payload := json.RawMessage(`{"entity_definition_id":"` + j.Definition.String() + `"}`)
	d := outbox.Delivery{OutboxID: uuid.Must(uuid.NewV7()), KindVersion: 1, Payload: payload}
	r.mu.Lock()
	b, _ := json.Marshal(d)
	r.ledger = append(r.ledger, b)
	r.mu.Unlock()
	r.env.ExecuteWorkflow(schemabackfill.WorkflowType, d)
	if !r.env.IsWorkflowCompleted() {
		r.t.Fatal("workflow did not complete")
	}
	return r.env.GetWorkflowError()
}

// assertNoValues is the history check (like S21's CI-5c): no argument,
// result, input or heartbeat carries a record value, and each is references
// only (or a barrier heartbeat, transaction ids).
func (r *run) assertNoValues() {
	r.t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.ledger) == 0 {
		r.t.Fatal("the ledger recorded nothing")
	}
	for _, raw := range r.ledger {
		if strings.Contains(string(raw), valueSecret) || strings.Contains(string(raw), "email") {
			r.t.Errorf("a recorded payload carries a record value: %s", raw)
		}
	}
	if strings.Contains(r.logs.String(), valueSecret) {
		r.t.Errorf("a log line carries a record value")
	}
	if n := r.counter(workerguard.DecisionsMetric, "outcome", "allow"); n == 0 {
		r.t.Errorf("the guard recorded no allowed decision: the run did not go through it")
	}
}

// requireGuardClean fails on any guard deny: every argument, result and
// heartbeat of the run was references only.
func (r *run) requireGuardClean() {
	r.t.Helper()
	if n := r.counterNot(workerguard.DecisionsMetric, "outcome", "allow"); n != 0 {
		r.t.Fatalf("the guard denied %d decisions; logs: %s", n, r.logs.String())
	}
}

// counter sums the points of an Int64 counter named name whose attribute key
// equals value ("" key: every point).
func (r *run) counter(name, key, value string) int64 {
	return r.sum(name, func(attrs map[string]string) bool { return key == "" || attrs[key] == value })
}

func (r *run) counterNot(name, key, value string) int64 {
	return r.sum(name, func(attrs map[string]string) bool { return attrs[key] != value })
}

func (r *run) sum(name string, match func(map[string]string) bool) int64 {
	var rm metricdata.ResourceMetrics
	if err := r.reader.Collect(context.Background(), &rm); err != nil {
		r.t.Fatal(err)
	}
	var n int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}
			for _, dp := range sum.DataPoints {
				attrs := map[string]string{}
				for _, kv := range dp.Attributes.ToSlice() {
					attrs[string(kv.Key)] = kv.Value.String()
				}
				if match(attrs) {
					n += dp.Value
				}
			}
		}
	}
	return n
}
