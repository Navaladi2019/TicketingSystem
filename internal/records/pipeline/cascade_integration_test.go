//go:build integration

// ADR-001 CI-10 as amended by A5 (P3-12, TICK-88), against a real PostgreSQL
// under RLS. The cases (a)–(e) run the real pipeline with startSink, a test
// EventSink that does what A5-4 asks of the real one (P3-42): for every
// write it claims a start of each of its definitions on the record and
// enqueues it, or audits the suppressed start. (f) is also an isolation
// suite path (internal/tenancy/isolation, cascade_paths.go).

package pipeline_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// startSink fires every definition in defs on every write, as A5-4's sink.
type startSink struct {
	defs []uuid.UUID
	// failAfter, if set, fails the sink after it has claimed and enqueued,
	// so the write rolls back.
	failAfter error

	mu     sync.Mutex
	events []pipeline.Event
}

func (s *startSink) RecordChanged(ctx context.Context, tx pgx.Tx, ev pipeline.Event) error {
	s.mu.Lock()
	s.events = append(s.events, ev)
	s.mu.Unlock()
	for _, d := range s.defs {
		c, claimed, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: ev.RecordID, DefinitionID: d})
		if err != nil {
			return err
		}
		if !claimed {
			if err := pipeline.AuditSuppressed(ctx, tx, c, ev, time.Now()); err != nil {
				return err
			}
			continue
		}
		if _, err := outbox.EnqueueStart(ctx, tx, c, outbox.Entry{Kind: outbox.InterpreterKind, KindVersion: 1,
			DedupeKey: "start:" + c.OutboxID().String(), RecordID: ev.RecordID,
			Target: outbox.TargetStart{Kind: outbox.InterpreterKind}}); err != nil {
			return err
		}
	}
	return s.failAfter
}

func (s *startSink) last() pipeline.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.events[len(s.events)-1]
}

// workflowCmd is D's update_record step: a workflow write of R one deeper
// than the delivery that caused it (A5-7), keyed per step (D-3-13).
func workflowCmd(f tenantFixture, id uuid.UUID, version int64, c outbox.Causation, step string) pipeline.Command {
	return pipeline.Command{Op: pipeline.OpUpdate, EntityKey: entityKey, RecordID: id, ExpectedVersion: version,
		Data: map[string]any{"summary": step}, Actor: f.system(), Source: pipeline.SourceWorkflow, Causation: c,
		IdempotencyKey: "wf:run-1:" + step}
}

func starts(t *testing.T, e env, tenant, record, def uuid.UUID) int {
	t.Helper()
	return count(t, e.super, `SELECT count(*) FROM cascade_start WHERE tenant_id = $1 AND record_id = $2 AND definition_id = $3`,
		tenant, record, def)
}

func startRows(t *testing.T, e env, tenant uuid.UUID) int {
	t.Helper()
	return count(t, e.super, `SELECT count(*) FROM outbox WHERE tenant_id = $1 AND kind = $2`, tenant, outbox.InterpreterKind)
}

func auditRows(t *testing.T, e env, tenant uuid.UUID, action string) int {
	t.Helper()
	return count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = $2`, tenant, action)
}

// createWithStart is S11b's step 1: a user creates R, which mints origin O at
// depth 0, and the sink starts D on R. It returns R and O.
func createWithStart(t *testing.T, e env, p *pipeline.Pipeline, sink *startSink, f tenantFixture) (pipeline.Result, uuid.UUID) {
	t.Helper()
	res := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	ev := sink.last()
	if ev.Causation.OriginID == uuid.Nil || ev.Causation.Depth != 0 {
		t.Fatalf("the create's causation %+v, want a minted origin at depth 0", ev.Causation)
	}
	var stored *uuid.UUID
	if err := e.super.QueryRow(context.Background(), `SELECT causation_origin_id FROM records WHERE id = $1`, res.RecordID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == nil || *stored != ev.Causation.OriginID {
		t.Fatalf("stored origin %v, want %s", stored, ev.Causation.OriginID)
	}
	return res, ev.Causation.OriginID
}

// (a) The S11b demo: D writing back to the record that started it commits.
func TestCI10_WriteBackToTriggeringRecordCommits(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	d := uuid.New()
	sink := &startSink{defs: []uuid.UUID{d}}
	p := newPipeline(e, sink)

	r, origin := createWithStart(t, e, p, sink, f)
	if n := starts(t, e, f.id, r.RecordID, d); n != 1 {
		t.Fatalf("%d starts of D after the create, want 1", n)
	}
	var startCausation *uuid.UUID
	var startDepth int16
	if err := e.super.QueryRow(context.Background(), `SELECT o.causation_id, o.causation_depth FROM outbox o
		JOIN cascade_start c ON c.tenant_id = o.tenant_id AND c.outbox_id = o.id WHERE c.record_id = $1`, r.RecordID).
		Scan(&startCausation, &startDepth); err != nil {
		t.Fatalf("the claim names no start row: %v", err)
	}
	if startCausation == nil || *startCausation != origin || startDepth != 0 {
		t.Fatalf("start row causation (%v, %d), want the write's (%s, 0)", startCausation, startDepth, origin)
	}

	res, err := write(context.Background(), e, p, f.id, workflowCmd(f, r.RecordID, 1, outbox.Causation{OriginID: origin, Depth: 1}, "step-1"))
	if err != nil {
		t.Fatalf("D's write-back to R: %v, want it accepted", err)
	}
	if res.Version != 2 {
		t.Fatalf("version %d, want 2", res.Version)
	}
	if n := starts(t, e, f.id, r.RecordID, d); n != 1 {
		t.Errorf("%d starts of D, want 1", n)
	}
	if n := startRows(t, e, f.id); n != 1 {
		t.Errorf("%d start rows, want 1", n)
	}
	if n := auditRows(t, e, f.id, cascade.ActionCycle); n != 1 {
		t.Errorf("%d cascade_cycle rows, want 1 (the write-back fired D again)", n)
	}
	var source string
	var actor, auditOrigin *uuid.UUID
	var depth int16
	var after []byte
	if err := e.super.QueryRow(context.Background(), `SELECT source, actor_id, causation_origin_id, causation_depth, after
		FROM audit_log WHERE tenant_id = $1 AND action = 'cascade_cycle'`, f.id).Scan(&source, &actor, &auditOrigin, &depth, &after); err != nil {
		t.Fatal(err)
	}
	if source != "workflow" || actor != nil || auditOrigin == nil || *auditOrigin != origin || depth != 1 ||
		string(after) != `{"reason": "cascade_cycle", "definition_id": "`+d.String()+`"}` {
		t.Fatalf("cascade_cycle row: source %s actor %v origin %v depth %d after %s", source, actor, auditOrigin, depth, after)
	}
}

// (b) Two update_record steps of D on R, under one origin: both commit, D
// has one start, and each write that fired D again has one cascade_cycle row.
// (c) D re-triggering itself on R, deeper each time: still one start, and one
// row per write.
func TestCI10_SiblingAndSelfRetriggeringWritesAreNotCycles(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	d := uuid.New()
	sink := &startSink{defs: []uuid.UUID{d}}
	p := newPipeline(e, sink)
	ctx := context.Background()

	r, origin := createWithStart(t, e, p, sink, f)
	for i, step := range []string{"step-1", "step-2"} {
		if _, err := write(ctx, e, p, f.id, workflowCmd(f, r.RecordID, int64(i+1), outbox.Causation{OriginID: origin, Depth: 1}, step)); err != nil {
			t.Fatalf("%s: %v, want both steps to commit", step, err)
		}
	}
	if n := starts(t, e, f.id, r.RecordID, d); n != 1 {
		t.Errorf("(b) %d starts of D, want 1", n)
	}
	if n := auditRows(t, e, f.id, cascade.ActionCycle); n != 2 {
		t.Errorf("(b) %d cascade_cycle rows, want 2", n)
	}

	// (c) Each of D's writes fires D again; each run is one deeper.
	for depth := int16(2); depth <= 4; depth++ {
		if _, err := write(ctx, e, p, f.id, workflowCmd(f, r.RecordID, int64(depth+1), outbox.Causation{OriginID: origin, Depth: depth},
			"again-"+string(rune('0'+depth)))); err != nil {
			t.Fatalf("(c) depth %d: %v", depth, err)
		}
	}
	if n := starts(t, e, f.id, r.RecordID, d); n != 1 {
		t.Errorf("(c) %d starts of D, want 1", n)
	}
	if n := startRows(t, e, f.id); n != 1 {
		t.Errorf("(c) %d start rows, want 1", n)
	}
	if n := auditRows(t, e, f.id, cascade.ActionCycle); n != 5 {
		t.Errorf("(c) %d cascade_cycle rows, want 5, one per write D made to R", n)
	}

	// Another definition on the same write is claimed on its own, and a new
	// user write mints a new origin, so D starts again (RS-A5-1).
	d2 := uuid.New()
	sink.defs = []uuid.UUID{d, d2}
	if _, err := write(ctx, e, p, f.id, updateCmd(f, r.RecordID, 6, map[string]any{"summary": "by hand"})); err != nil {
		t.Fatal(err)
	}
	if n := starts(t, e, f.id, r.RecordID, d); n != 2 {
		t.Errorf("%d starts of D after a new user write, want 2", n)
	}
	if n := starts(t, e, f.id, r.RecordID, d2); n != 1 {
		t.Errorf("%d starts of D2, want 1", n)
	}
}

// (d) A retried activity creates no second start: its replay writes nothing
// and calls no sink. A retry of a write that rolled back after claiming
// claims the key afresh, because the rollback released it.
func TestCI10_RetriedActivityCreatesNoSecondStart(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	d := uuid.New()
	sink := &startSink{defs: []uuid.UUID{d}}
	p := newPipeline(e, sink)
	ctx := context.Background()

	r, origin := createWithStart(t, e, p, sink, f)
	cmd := workflowCmd(f, r.RecordID, 1, outbox.Causation{OriginID: origin, Depth: 1}, "step-1")
	first, err := write(ctx, e, p, f.id, cmd)
	if err != nil {
		t.Fatal(err)
	}
	calls := len(sink.events)
	again, err := write(ctx, e, p, f.id, cmd)
	if err != nil || again != first {
		t.Fatalf("retry = %+v, %v; want the first result %+v", again, err, first)
	}
	if len(sink.events) != calls {
		t.Errorf("the retry called the sink")
	}
	if n := starts(t, e, f.id, r.RecordID, d); n != 1 {
		t.Errorf("%d starts of D, want 1", n)
	}
	if n := auditRows(t, e, f.id, cascade.ActionCycle); n != 1 {
		t.Errorf("%d cascade_cycle rows, want 1", n)
	}

	// A create whose sink claimed and then failed: nothing survives, and the
	// retried create claims the key of its own record.
	failing := &startSink{defs: []uuid.UUID{d}, failAfter: errors.New("activity crashed")}
	if _, err := write(ctx, e, newPipeline(e, failing), f.id, createCmd(f, map[string]any{"title": "lost"})); err == nil {
		t.Fatal("the failing sink's write committed")
	}
	lost := failing.last().RecordID
	if n := count(t, e.super, `SELECT count(*) FROM cascade_start WHERE record_id = $1`, lost); n != 0 {
		t.Errorf("the rolled-back write left %d claims", n)
	}
	if n := startRows(t, e, f.id); n != 1 {
		t.Errorf("%d start rows, want 1 (the rolled-back start must not survive)", n)
	}
}

// recordingRunner counts the own-transaction runs the pipeline asks for.
type recordingRunner struct {
	pipeline.TxRunner
	mu   sync.Mutex
	runs int
}

func (r *recordingRunner) InTenantTx(ctx context.Context, tenant uuid.UUID, fn func(pgx.Tx) error) error {
	r.mu.Lock()
	r.runs++
	r.mu.Unlock()
	return r.TxRunner.InTenantTx(ctx, tenant, fn)
}

// countingTx counts the statements the pipeline sends on the caller's tx.
type countingTx struct {
	pgx.Tx
	statements int
}

func (c *countingTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	c.statements++
	return c.Tx.Exec(ctx, sql, args...)
}

func (c *countingTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	c.statements++
	return c.Tx.Query(ctx, sql, args...)
}

func (c *countingTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	c.statements++
	return c.Tx.QueryRow(ctx, sql, args...)
}

// (e) and the TICK-363 contract: a write at depth 9 is refused with
// ErrCascadeTooDeep before any statement in the caller's transaction but the
// read of its tenant, changes no record, and leaves exactly one
// cascade_rejected row, written on the TxRunner, after the caller rolls back.
// Depth 8 is accepted.
func TestCI10_DepthBeyondLimitRejectedAndAuditedOnTheRunner(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	runner := &recordingRunner{TxRunner: pipeline.PoolRunner{Pool: e.app}}
	sink := &countingSink{}
	p := pipeline.New(authz.TenantMemberAuthorizer{}, entitytypes.Store{}, sink, runner, pipeline.SystemClock{})
	ctx := context.Background()
	origin := uuid.New()
	tooDeep := outbox.Causation{OriginID: origin, Depth: config.CascadeDepthLimit + 1}

	refused := func(cmd pipeline.Command) *countingTx {
		t.Helper()
		var counted *countingTx
		err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
			counted = &countingTx{Tx: tx}
			_, err := p.Write(ctx, counted, cmd)
			return err
		})
		if !errors.Is(err, outbox.ErrCascadeTooDeep) {
			t.Fatalf("err = %v, want ErrCascadeTooDeep", err)
		}
		return counted
	}

	// A create at depth 9: no record, one row with no record id.
	create := createCmd(f, map[string]any{"title": "deep"})
	create.Source, create.Causation = pipeline.SourceRule, tooDeep
	if tx := refused(create); tx.statements != 1 {
		t.Errorf("the refused create sent %d statements on the caller's tx, want 1 (the tenant read)", tx.statements)
	}
	if runner.runs != 1 {
		t.Errorf("%d runner transactions, want 1", runner.runs)
	}
	if n := count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1`, f.id); n != 0 {
		t.Errorf("%d records after the refused create, want 0", n)
	}
	if sink.calls() != 0 {
		t.Errorf("the refused write called the sink")
	}
	var record, actor, auditOrigin *uuid.UUID
	var source string
	var depth int16
	var after []byte
	if err := e.super.QueryRow(ctx, `SELECT record_id, actor_id, source, causation_origin_id, causation_depth, after
		FROM audit_log WHERE tenant_id = $1 AND action = 'cascade_rejected'`, f.id).
		Scan(&record, &actor, &source, &auditOrigin, &depth, &after); err != nil {
		t.Fatalf("exactly one cascade_rejected row: %v", err)
	}
	if record != nil || actor == nil || *actor != f.principal.PrincipalID || source != "rule" ||
		auditOrigin == nil || *auditOrigin != origin || depth != 9 || string(after) != `{"reason": "cascade_too_deep"}` {
		t.Fatalf("cascade_rejected row: record %v actor %v source %s origin %v depth %d after %s",
			record, actor, source, auditOrigin, depth, after)
	}

	// An update at depth 9: the record is unchanged, one row names it.
	r := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	update := workflowCmd(f, r.RecordID, 1, tooDeep, "deep")
	refused(update)
	if _, _, _, v, _ := storedData(t, e, r.RecordID); v != 1 {
		t.Errorf("version %d after the refused update, want 1", v)
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = 'cascade_rejected' AND record_id = $2`,
		f.id, r.RecordID); n != 1 {
		t.Errorf("%d cascade_rejected rows naming the record, want 1", n)
	}
	if n := auditRows(t, e, f.id, cascade.ActionRejected); n != 2 {
		t.Errorf("%d cascade_rejected rows in all, want 2", n)
	}
	// The refused update's key was not used: the retry at a legal depth runs.
	update.Causation.Depth = config.CascadeDepthLimit
	if _, err := write(ctx, e, p, f.id, update); err != nil {
		t.Fatalf("depth 8: %v, want accepted", err)
	}
}

// (f) The same origin, record and definition uuids in two tenants never
// collide, and tenant B cannot claim a key in tenant A or reach its record.
func TestCI10_StartKeyIsPerTenant(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	ctx := context.Background()
	record, origin, d := uuid.New(), uuid.New(), uuid.New()
	for _, f := range []tenantFixture{a, b} {
		mustExec(t, e.super, `INSERT INTO records (tenant_id, id, causation_origin_id, causation_depth) VALUES ($1, $2, $3, 0)`,
			f.id, record, origin)
	}
	aOnly := uuid.New()
	mustExec(t, e.super, `INSERT INTO records (tenant_id, id, causation_origin_id, causation_depth) VALUES ($1, $2, $3, 0)`,
		a.id, aOnly, origin)

	claim := func(f tenantFixture, rec uuid.UUID) (bool, error) {
		var claimed bool
		err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
			var err error
			_, claimed, err = cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: rec, DefinitionID: d})
			return err
		})
		return claimed, err
	}
	for _, f := range []tenantFixture{a, b} {
		if claimed, err := claim(f, record); err != nil || !claimed {
			t.Fatalf("tenant %s: claimed %v, %v; want its own claim", f.id, claimed, err)
		}
	}
	if claimed, err := claim(b, aOnly); !errors.Is(err, cascade.ErrRecordNotFound) || claimed {
		t.Fatalf("tenant B claiming on tenant A's record: %v, %v; want ErrRecordNotFound", claimed, err)
	}
	err := tenancy.InTenantTx(ctx, e.app, b.id, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id)
			VALUES ($1, $2, $3, $4, $5)`, a.id, origin, aOnly, d, uuid.New())
		return err
	})
	if sqlState(err) != "42501" {
		t.Fatalf("tenant B inserting tenant A's key: %v, want 42501 (RLS)", err)
	}
	if n := count(t, e.super, `SELECT count(*) FROM cascade_start WHERE record_id = $1`, aOnly); n != 0 {
		t.Errorf("%d claims on tenant A's record, want 0", n)
	}
}

// Two transactions claiming one key: the second waits on R's row lock. If
// the first commits, the second is suppressed; if it rolls back, the second
// claims the key.
func TestCI10_ConcurrentClaimsOfOneKey(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	ctx := context.Background()
	d := uuid.New()

	for _, firstCommits := range []bool{true, false} {
		record := uuid.New()
		mustExec(t, e.super, `INSERT INTO records (tenant_id, id, causation_origin_id, causation_depth) VALUES ($1, $2, $3, 0)`,
			f.id, record, uuid.New())
		key := cascade.Key{RecordID: record, DefinitionID: d}

		claimedFirst := make(chan struct{})
		release := make(chan struct{})
		firstDone := make(chan error, 1)
		go func() {
			firstDone <- tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
				if _, claimed, err := cascade.ClaimStart(ctx, tx, key); err != nil || !claimed {
					return errors.Join(err, errors.New("the first claim was not claimed"))
				}
				close(claimedFirst)
				<-release
				if !firstCommits {
					return errRolledBack
				}
				return nil
			})
		}()
		select {
		case <-claimedFirst:
		case err := <-firstDone:
			t.Fatalf("first claim: %v", err)
		}
		second := make(chan bool, 1)
		secondErr := make(chan error, 1)
		go func() {
			var claimed bool
			secondErr <- tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
				var err error
				_, claimed, err = cascade.ClaimStart(ctx, tx, key)
				return err
			})
			second <- claimed
		}()
		waitForLockWait(t, e)
		close(release)
		if err := <-firstDone; firstCommits && err != nil || !firstCommits && !errors.Is(err, errRolledBack) {
			t.Fatalf("first transaction: %v", err)
		}
		if err := <-secondErr; err != nil {
			t.Fatalf("second claim: %v", err)
		}
		if got := <-second; got == firstCommits {
			t.Errorf("first commits %v: second claimed %v, want %v", firstCommits, got, !firstCommits)
		}
		if n := starts(t, e, f.id, record, d); n != 1 {
			t.Errorf("first commits %v: %d claims, want 1", firstCommits, n)
		}
	}
}

var errRolledBack = errors.New("roll back")

// The start key's guards: a record with no stored causation cannot be
// keyed; EnqueueStart takes only a claimed claim, on its record, with the
// record's stored causation, and uses the claim's id; Enqueue refuses an
// interpreter start.
func TestCascadeStartGuards(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	ctx := context.Background()
	bare, keyed, origin := uuid.New(), uuid.New(), uuid.New()
	mustExec(t, e.super, `INSERT INTO records (tenant_id, id) VALUES ($1, $2)`, f.id, bare)
	mustExec(t, e.super, `INSERT INTO records (tenant_id, id, causation_origin_id, causation_depth) VALUES ($1, $2, $3, 2)`,
		f.id, keyed, origin)
	d := uuid.New()
	start := func(record uuid.UUID, c outbox.Causation) outbox.Entry {
		return outbox.Entry{Kind: outbox.InterpreterKind, KindVersion: 1, DedupeKey: "g-" + uuid.NewString(), RecordID: record,
			Target: outbox.TargetStart{Kind: outbox.InterpreterKind}, Causation: c}
	}
	inRollback := func(fn func(tx pgx.Tx) error) error {
		err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
			if err := fn(tx); err != nil {
				return err
			}
			return errRolledBack
		})
		if errors.Is(err, errRolledBack) {
			return nil
		}
		return err
	}

	if err := inRollback(func(tx pgx.Tx) error {
		_, _, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: bare, DefinitionID: d})
		return err
	}); !errors.Is(err, cascade.ErrNoOrigin) {
		t.Errorf("claim on a record with no causation: %v, want ErrNoOrigin", err)
	}
	if err := inRollback(func(tx pgx.Tx) error {
		_, err := outbox.Enqueue(ctx, tx, start(keyed, outbox.Causation{OriginID: origin, Depth: 2}))
		return err
	}); !errors.Is(err, outbox.ErrInvalidEntry) {
		t.Errorf("Enqueue of an interpreter start: %v, want ErrInvalidEntry", err)
	}
	if err := inRollback(func(tx pgx.Tx) error {
		c, claimed, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: keyed, DefinitionID: d})
		if err != nil || !claimed {
			return errors.Join(err, errors.New("not claimed"))
		}
		if _, err := outbox.EnqueueStart(ctx, tx, c, start(keyed, outbox.Causation{OriginID: uuid.New(), Depth: 2})); !errors.Is(err, outbox.ErrInvalidEntry) {
			t.Errorf("EnqueueStart with a causation the record does not hold: %v, want ErrInvalidEntry", err)
		}
		if _, err := outbox.EnqueueStart(ctx, tx, c, start(bare, outbox.Causation{})); !errors.Is(err, outbox.ErrInvalidEntry) {
			t.Errorf("EnqueueStart on another record: %v, want ErrInvalidEntry", err)
		}
		id, err := outbox.EnqueueStart(ctx, tx, c, start(keyed, outbox.Causation{}))
		if err != nil {
			return err
		}
		if id != c.OutboxID() {
			t.Errorf("start row %s, want the claim's %s", id, c.OutboxID())
		}
		// The row's causation is the record's: TestCI10_WriteBackToTriggeringRecordCommits
		// reads it back (nava_app may not read outbox.causation_*).
		if o, depth := c.Causation(); o != origin || depth != 2 {
			t.Errorf("claim causation (%s, %d), want the record's (%s, 2)", o, depth, origin)
		}
		// A repeat claim in the same transaction is suppressed and cannot
		// enqueue.
		again, claimed, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: keyed, DefinitionID: d})
		if err != nil || claimed {
			return errors.Join(err, errors.New("repeat claimed"))
		}
		if _, err := outbox.EnqueueStart(ctx, tx, again, start(keyed, outbox.Causation{})); !errors.Is(err, outbox.ErrInvalidEntry) {
			t.Errorf("EnqueueStart with a suppressed claim: %v, want ErrInvalidEntry", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// A user write at depth 0 mints a fresh origin per write.
func TestDepthZeroWritesMintTheirOrigin(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	sink := &countingSink{}
	p := newPipeline(e, sink)
	mustCreate(t, e, p, f, map[string]any{"title": "one"})
	mustCreate(t, e, p, f, map[string]any{"title": "two"})
	o1, o2 := sink.events[0].Causation.OriginID, sink.events[1].Causation.OriginID
	if o1 == uuid.Nil || o2 == uuid.Nil || o1 == o2 {
		t.Fatalf("origins %s, %s; want two fresh ones", o1, o2)
	}
	if n := count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1 AND causation_origin_id IS NULL`, f.id); n != 0 {
		t.Errorf("%d records with no origin", n)
	}
}

func sqlState(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}
