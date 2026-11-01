//go:build integration

// P3-11 (TICK-355): numbering (D-3-9), the state model and unique properties
// (D-3-22), and the idempotency key mismatch carried from the P3-10 review.
// The concurrency tests run through PgBouncer in transaction mode
// (scripts/ci/pgbouncer/pooled.ini: pool_mode = transaction,
// default_pool_size = 5, max_prepared_statements = 100), so 50 clients share
// 5 server connections, as nava-core does on the compose stack.

package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// throughBouncer returns e with its nava_app pool replaced by one that goes
// through the transaction-mode PgBouncer, maxConns client connections wide.
func throughBouncer(t *testing.T, e env, maxConns int32) env {
	t.Helper()
	raw := os.Getenv("NAVA_TEST_PGBOUNCER_URL")
	if raw == "" {
		t.Fatal("NAVA_TEST_PGBOUNCER_URL is not set; run make test-integration")
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := testPassword("nava_app")
	u.User = url.UserPassword("nava_app", pw)
	u.Path = "/" + e.platform
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = maxConns
	p, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	// The server behind the pooler is the platform database.
	var db string
	if err := p.QueryRow(context.Background(), "SELECT current_database()").Scan(&db); err != nil || db != e.platform {
		t.Fatalf("through PgBouncer: database %q, %v; want %q", db, err, e.platform)
	}
	e.app = p
	return e
}

// numbers returns the tenant's record numbers of the entity type, sorted.
func numbers(t *testing.T, e env, tenant uuid.UUID, key string) []string {
	t.Helper()
	rows, err := e.super.Query(context.Background(), `
SELECT r.number FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
WHERE r.tenant_id = $1 AND d.key = $2 ORDER BY r.number`, tenant, key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func wantNumbers(t *testing.T, prefix string, from, to int) []string {
	t.Helper()
	var out []string
	for n := from; n <= to; n++ {
		out = append(out, fmt.Sprintf("%s%07d", prefix, n))
	}
	return out
}

// createMany runs n creates on workers goroutines and returns how long they
// took and every error.
func createMany(e env, p *pipeline.Pipeline, f tenantFixture, n, workers int) (time.Duration, []error) {
	jobs := make(chan int)
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := time.Now()
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				_, errs[i] = write(context.Background(), e, p, f.id, createCmd(f, map[string]any{"title": fmt.Sprintf("load %d", i)}))
			}
		}()
	}
	for i := range n {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return time.Since(start), errs
}

func noErrors(t *testing.T, errs []error) {
	t.Helper()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
}

// AC: 50 concurrent creates through PgBouncer get INC0000001..INC0000050,
// no duplicates and no gaps. A further 500 creates on 50 workers continue at
// 51 with no gap, and give a steadier creates/s figure for the PR.
func TestConcurrentCreatesNumberWithoutGapsThroughPgBouncer(t *testing.T) {
	e := throughBouncer(t, migrated(t), 50)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)

	took, errs := createMany(e, p, f, 50, 50)
	noErrors(t, errs)
	if got, want := numbers(t, e, f.id, entityKey), wantNumbers(t, "INC", 1, 50); !slices.Equal(got, want) {
		t.Fatalf("numbers = %v, want %v", got, want)
	}
	t.Logf("50 concurrent creates through PgBouncer (5 server connections): %v, %.0f creates/s", took, 50/took.Seconds())

	took, errs = createMany(e, p, f, 500, 50)
	noErrors(t, errs)
	if got, want := numbers(t, e, f.id, entityKey), wantNumbers(t, "INC", 1, 550); !slices.Equal(got, want) {
		t.Fatalf("after 500 more: %d numbers, want INC0000001..INC0000550", len(got))
	}
	t.Logf("500 creates on 50 workers through PgBouncer (5 server connections): %v, %.0f creates/s", took, 500/took.Seconds())
}

// AC: a rolled-back create's number is reused by the next create, and a
// refused create (here a unique violation, after which the write stops before
// taking a number) uses none.
func TestRolledBackCreateNumberIsReused(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	mustCreate(t, e, p, f, map[string]any{"title": "one"})

	rollback := errors.New("roll back")
	var lost pipeline.Result
	err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
		var err error
		if lost, err = p.Write(ctx, tx, createCmd(f, map[string]any{"title": "rolled back"})); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) || lost.Number != "INC0000002" {
		t.Fatalf("rolled-back create = %+v, %v; want INC0000002 then the rollback", lost, err)
	}
	if next := mustCreate(t, e, p, f, map[string]any{"title": "two"}); next.Number != "INC0000002" {
		t.Fatalf("next create got %s, want the rolled-back INC0000002", next.Number)
	}
	if got, want := numbers(t, e, f.id, entityKey), wantNumbers(t, "INC", 1, 2); !slices.Equal(got, want) {
		t.Fatalf("numbers = %v, want %v", got, want)
	}
}

// AC: tenant A and tenant B both start at INC0000001; so does a second type
// in the same tenant, with its own prefix. Past 7 digits the number grows.
func TestNumbersPerTenantAndType(t *testing.T) {
	e := migrated(t)
	a := seedTenantWith(t, e, definition(schemaV1), assetDefinition())
	b := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()

	for _, f := range []tenantFixture{a, b, a} {
		mustCreate(t, e, p, f, map[string]any{"title": "x"})
	}
	if got := numbers(t, e, a.id, entityKey); !slices.Equal(got, wantNumbers(t, "INC", 1, 2)) {
		t.Fatalf("tenant A: %v", got)
	}
	if got := numbers(t, e, b.id, entityKey); !slices.Equal(got, wantNumbers(t, "INC", 1, 1)) {
		t.Fatalf("tenant B: %v, want INC0000001", got)
	}
	asset, err := write(ctx, e, p, a.id, assetCmd(a, map[string]any{"name": "laptop"}))
	if err != nil || asset.Number != "AST0000001" {
		t.Fatalf("first asset = %+v, %v; want AST0000001", asset, err)
	}

	mustExec(t, e.super, `UPDATE record_counters SET next = 10000000 WHERE tenant_id = $1
		AND entity_definition_id = (SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2)`, b.id, entityKey)
	if grown := mustCreate(t, e, p, b, map[string]any{"title": "grown"}); grown.Number != "INC10000000" {
		t.Fatalf("number past 7 digits = %s, want INC10000000", grown.Number)
	}
}

// incidentTemplate is the P3-06 incident seed definition.
func incidentTemplate(t *testing.T) metadata.Definition {
	t.Helper()
	d, err := metadata.Template("incident")
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// AC: every illegal transition of the incident template is refused with
// *ErrIllegalTransition{From, To} and writes nothing; every legal one, and
// staying in a state, succeeds. A create must use the initial state.
func TestIncidentTransitions(t *testing.T) {
	e := migrated(t)
	tmpl := incidentTemplate(t)
	f := seedTenantWith(t, e, tmpl)
	p := newPipeline(e, nil)
	ctx := context.Background()

	// pathTo walks a new record to each state over legal transitions.
	pathTo := map[string][]string{
		"new":         nil,
		"in_progress": {"in_progress"},
		"resolved":    {"in_progress", "resolved"},
		"closed":      {"in_progress", "resolved", "closed"},
	}
	cases := []struct {
		from, to string
		legal    bool
	}{
		{"new", "in_progress", true},
		{"new", "resolved", false},
		{"new", "closed", false},
		{"in_progress", "new", false},
		{"in_progress", "resolved", true},
		{"in_progress", "closed", false},
		{"resolved", "new", false},
		{"resolved", "in_progress", true},
		{"resolved", "closed", true},
		{"closed", "new", false},
		{"closed", "in_progress", false},
		{"closed", "resolved", false},
		{"new", "new", true},
		{"in_progress", "in_progress", true},
		{"resolved", "resolved", true},
		{"closed", "closed", true},
	}
	// The table covers every ordered pair of the template's states.
	if n := len(tmpl.States.States); len(cases) != n*n {
		t.Fatalf("%d cases for %d states", len(cases), n)
	}
	cmd := func(op pipeline.Op, id uuid.UUID, version int64, data map[string]any) pipeline.Command {
		return pipeline.Command{Op: op, EntityKey: "incident", RecordID: id, ExpectedVersion: version,
			Data: data, Actor: f.principal, Source: pipeline.SourceUser}
	}
	for _, c := range cases {
		t.Run(c.from+"->"+c.to, func(t *testing.T) {
			res, err := write(ctx, e, p, f.id, cmd(pipeline.OpCreate, uuid.Nil, 0, map[string]any{"short_description": "printer"}))
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range pathTo[c.from] {
				if res, err = write(ctx, e, p, f.id, cmd(pipeline.OpUpdate, res.RecordID, res.Version, map[string]any{"state": s})); err != nil {
					t.Fatalf("walk to %s: %v", s, err)
				}
			}
			got, err := write(ctx, e, p, f.id, cmd(pipeline.OpUpdate, res.RecordID, res.Version, map[string]any{"state": c.to}))
			_, state, _, version, _ := storedData(t, e, res.RecordID)
			if c.legal {
				if err != nil || state != c.to || got.Version != res.Version+1 {
					t.Fatalf("legal %s -> %s: %+v, %v; state %s", c.from, c.to, got, err, state)
				}
				return
			}
			var ill *pipeline.ErrIllegalTransition
			if !errors.As(err, &ill) || *ill != (pipeline.ErrIllegalTransition{From: c.from, To: c.to}) {
				t.Fatalf("illegal %s -> %s: %v, want *ErrIllegalTransition{%s, %s}", c.from, c.to, err, c.from, c.to)
			}
			if state != c.from || version != res.Version {
				t.Fatalf("refused transition wrote: state %s version %d", state, version)
			}
		})
	}

	for _, s := range []string{"in_progress", "resolved", "closed"} {
		_, err := write(ctx, e, p, f.id, cmd(pipeline.OpCreate, uuid.Nil, 0, map[string]any{"short_description": "x", "state": s}))
		var ill *pipeline.ErrIllegalTransition
		if !errors.As(err, &ill) || *ill != (pipeline.ErrIllegalTransition{To: s}) {
			t.Fatalf("create in %s: %v, want *ErrIllegalTransition{To: %s}", s, err, s)
		}
	}
	if res, err := write(ctx, e, p, f.id, cmd(pipeline.OpCreate, uuid.Nil, 0, map[string]any{"short_description": "x", "state": "new"})); err != nil {
		t.Fatalf("create in the initial state: %+v, %v", res, err)
	}
}

// assetDefinition is a second type with unique properties: serial (a string
// in data), tag (a number in data) and owner (a fixed uuid column).
func assetDefinition() metadata.Definition {
	return metadata.Definition{
		Key: "asset",
		Schema: json.RawMessage(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["name"],
  "properties": {
    "name": {"type": "string", "minLength": 1},
    "status": {"type": "string", "enum": ["active", "retired"]},
    "serial": {"type": "string", "x-nava-unique": true},
    "tag": {"type": "number", "x-nava-unique": true},
    "owner": {"type": "string", "format": "uuid", "x-nava-unique": true}
  }
}`),
		NumberPrefix: "AST",
		FixedColumns: map[string]string{"status": metadata.ColumnState, "owner": metadata.ColumnRequesterID},
		States: metadata.StateModel{
			Initial:     "active",
			States:      []string{"active", "retired"},
			Transitions: []metadata.Transition{{From: "active", To: "retired"}},
		},
	}
}

func assetCmd(f tenantFixture, data map[string]any) pipeline.Command {
	return pipeline.Command{Op: pipeline.OpCreate, EntityKey: "asset", Data: data, Actor: f.principal, Source: pipeline.SourceUser}
}

func assetUpdate(f tenantFixture, id uuid.UUID, version int64, patch map[string]any) pipeline.Command {
	return pipeline.Command{Op: pipeline.OpUpdate, EntityKey: "asset", RecordID: id, ExpectedVersion: version,
		Data: patch, Actor: f.principal, Source: pipeline.SourceUser}
}

func wantUnique(t *testing.T, err error, field string) {
	t.Helper()
	var uv *pipeline.ErrUniqueViolation
	if !errors.As(err, &uv) || uv.Field != field {
		t.Fatalf("err = %v, want *ErrUniqueViolation{%s}", err, field)
	}
}

// Unique properties, one write at a time: data and fixed-column values,
// numbers by value, updates that keep or move a value, empty values, and the
// tenant boundary.
func TestUniqueProperties(t *testing.T) {
	e := migrated(t)
	a := seedTenantWith(t, e, assetDefinition())
	b := seedTenantWith(t, e, assetDefinition())
	p := newPipeline(e, nil)
	ctx := context.Background()
	w := func(f tenantFixture, cmd pipeline.Command) (pipeline.Result, error) { return write(ctx, e, p, f.id, cmd) }
	owner := a.principal.PrincipalID

	first, err := w(a, assetCmd(a, map[string]any{"name": "one", "serial": "SN-1", "tag": 7, "owner": owner.String()}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = w(a, assetCmd(a, map[string]any{"name": "dup", "serial": "SN-1"}))
	wantUnique(t, err, "serial")
	_, err = w(a, assetCmd(a, map[string]any{"name": "dup", "tag": json.Number("7.0")}))
	wantUnique(t, err, "tag")
	_, err = w(a, assetCmd(a, map[string]any{"name": "dup", "owner": strings.ToUpper(owner.String())}))
	wantUnique(t, err, "owner")
	// The string "7" is not the number 7.
	if _, err := w(a, assetCmd(a, map[string]any{"name": "s7", "serial": "7"})); err != nil {
		t.Fatalf("serial \"7\": %v", err)
	}
	// Empty values repeat freely.
	for range 2 {
		if _, err := w(a, assetCmd(a, map[string]any{"name": "no serial"})); err != nil {
			t.Fatalf("no unique values: %v", err)
		}
	}
	// Tenant B may use tenant A's values.
	if _, err := w(b, assetCmd(b, map[string]any{"name": "b", "serial": "SN-1", "tag": 7})); err != nil {
		t.Fatalf("tenant B, tenant A's serial: %v", err)
	}
	// The refused creates took no number.
	if got := numbers(t, e, a.id, "asset"); !slices.Equal(got, wantNumbers(t, "AST", 1, 4)) {
		t.Fatalf("tenant A numbers %v, want AST0000001..4", got)
	}

	// An update that keeps its own values, or changes only other fields, passes.
	res, err := w(a, assetUpdate(a, first.RecordID, 1, map[string]any{"name": "renamed", "serial": "SN-1", "tag": json.Number("7.00")}))
	if err != nil {
		t.Fatalf("update keeping its unique values: %v", err)
	}
	second, err := w(a, assetCmd(a, map[string]any{"name": "two", "serial": "SN-2"}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = w(a, assetUpdate(a, second.RecordID, 1, map[string]any{"serial": "SN-1"}))
	wantUnique(t, err, "serial")
	// Moving the value off the first record frees it.
	if _, err := w(a, assetUpdate(a, first.RecordID, res.Version, map[string]any{"serial": nil})); err != nil {
		t.Fatal(err)
	}
	if _, err := w(a, assetUpdate(a, second.RecordID, 1, map[string]any{"serial": "SN-1"})); err != nil {
		t.Fatalf("serial freed by the first record: %v", err)
	}
}

// AC: concurrent creates with the same unique value through PgBouncer:
// exactly one wins, every other gets *ErrUniqueViolation.
func TestUniqueConcurrentCreatesOneWins(t *testing.T) {
	e := throughBouncer(t, migrated(t), 50)
	f := seedTenantWith(t, e, assetDefinition())
	p := newPipeline(e, nil)
	const n = 50
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = write(context.Background(), e, p, f.id, assetCmd(f, map[string]any{"name": fmt.Sprint(i), "serial": "SN-RACE"}))
		}()
	}
	close(start)
	wg.Wait()
	won := 0
	for _, err := range errs {
		if err == nil {
			won++
			continue
		}
		wantUnique(t, err, "serial")
	}
	if won != 1 {
		t.Fatalf("%d creates won, want exactly 1", won)
	}
	if got := numbers(t, e, f.id, "asset"); !slices.Equal(got, []string{"AST0000001"}) {
		t.Fatalf("numbers %v, want [AST0000001]", got)
	}
}

// AC: a concurrent update and create that set the same unique value: exactly
// one wins, in either order. The first writer is parked after its last
// statement, before commit, holding the value's advisory lock; the second
// blocks on that lock and, once the first commits, finds its row.
func TestUniqueConcurrentUpdateAndCreateOneWins(t *testing.T) {
	e := migrated(t)
	f := seedTenantWith(t, e, assetDefinition())
	ctx := context.Background()

	for _, updateFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("update first %v", updateFirst), func(t *testing.T) {
			serial := fmt.Sprintf("SN-%v", updateFirst)
			target, err := write(ctx, e, newPipeline(e, nil), f.id, assetCmd(f, map[string]any{"name": "target"}))
			if err != nil {
				t.Fatal(err)
			}
			update := assetUpdate(f, target.RecordID, 1, map[string]any{"serial": serial})
			create := assetCmd(f, map[string]any{"name": "new", "serial": serial})
			firstCmd, secondCmd := update, create
			if !updateFirst {
				firstCmd, secondCmd = create, update
			}

			holder := newPipeline(e, nil)
			parked, release := make(chan struct{}), make(chan struct{})
			pipeline.SetBeforeReturn(holder, func(context.Context) { close(parked); <-release })
			var wg sync.WaitGroup
			var firstErr, secondErr error
			wg.Add(1)
			go func() { defer wg.Done(); _, firstErr = write(ctx, e, holder, f.id, firstCmd) }()
			<-parked
			wg.Add(1)
			go func() { defer wg.Done(); _, secondErr = write(ctx, e, newPipeline(e, nil), f.id, secondCmd) }()
			waitForLockWait(t, e)
			close(release)
			wg.Wait()

			if firstErr != nil {
				t.Fatalf("first: %v", firstErr)
			}
			wantUnique(t, secondErr, "serial")
			if n := count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1 AND data ->> 'serial' = $2`, f.id, serial); n != 1 {
				t.Fatalf("%d records hold %s, want 1", n, serial)
			}
		})
	}
}

// The P3-10 review item, decided on TICK-355: a replay whose stored key was
// used for another entity type, op or record is a mismatch
// *ErrIdempotencyConflict (P3-14 maps it to 409) and writes nothing; the
// same write replays.
func TestIdempotencyKeyMismatchConflicts(t *testing.T) {
	e := migrated(t)
	f := seedTenantWith(t, e, definition(schemaV1), assetDefinition())
	p := newPipeline(e, nil)
	ctx := context.Background()
	system := func(cmd pipeline.Command, key string) pipeline.Command {
		cmd.Actor, cmd.Source, cmd.IdempotencyKey = f.system(), pipeline.SourceWorkflow, key
		return cmd
	}
	createC := system(createCmd(f, map[string]any{"title": "keyed"}), "wf:c")
	created, err := write(ctx, e, p, f.id, createC)
	if err != nil {
		t.Fatal(err)
	}
	updateU := system(updateCmd(f, created.RecordID, 1, map[string]any{"summary": "keyed"}), "wf:u")
	updated, err := write(ctx, e, p, f.id, updateU)
	if err != nil {
		t.Fatal(err)
	}
	other := mustCreate(t, e, p, f, map[string]any{"title": "other"})
	asset, err := write(ctx, e, p, f.id, assetCmd(f, map[string]any{"name": "a"}))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		cmd    pipeline.Command
		replay *pipeline.Result
	}{
		{"same create replays", createC, &created},
		{"same update replays", updateU, &updated},
		{"create key, other entity type", system(assetCmd(f, map[string]any{"name": "x"}), "wf:c"), nil},
		{"create key, used by an update", system(updateCmd(f, created.RecordID, 2, map[string]any{"summary": "x"}), "wf:c"), nil},
		{"update key, used by a create", system(createCmd(f, map[string]any{"title": "x"}), "wf:u"), nil},
		{"update key, other record", system(updateCmd(f, other.RecordID, 1, map[string]any{"summary": "x"}), "wf:u"), nil},
		{"update key, other entity type", system(assetUpdate(f, asset.RecordID, 1, map[string]any{"name": "x"}), "wf:u"), nil},
	}
	audits := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", f.id)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := write(ctx, e, p, f.id, c.cmd)
			if c.replay != nil {
				if err != nil || got != *c.replay {
					t.Fatalf("= %+v, %v; want the replay %+v", got, err, *c.replay)
				}
				return
			}
			var conflict *pipeline.ErrIdempotencyConflict
			if !errors.As(err, &conflict) || conflict.Concurrent {
				t.Fatalf("= %+v, %v; want a mismatch *ErrIdempotencyConflict", got, err)
			}
		})
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", f.id); n != audits {
		t.Fatalf("%d audit rows after the table, want %d: a conflict or replay wrote", n, audits)
	}
}
