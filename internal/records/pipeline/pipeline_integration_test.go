//go:build integration

package pipeline_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

func wantFields(t *testing.T, err error, want ...metadata.FieldError) {
	t.Helper()
	var ve *pipeline.ErrValidation
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want *ErrValidation", err)
	}
	if !reflect.DeepEqual(ve.Fields, want) {
		t.Fatalf("fields = %v, want %v", ve.Fields, want)
	}
}

// AC: a schema-invalid create returns field errors, and there is no record,
// no audit row and no EventSink call.
func TestInvalidCreateWritesNothing(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	sink := &countingSink{}
	p := newPipeline(e, sink)

	_, err := write(context.Background(), e, p, f.id, createCmd(f, map[string]any{"priority": "urgent", "summary": 7}))
	wantFields(t, err,
		metadata.FieldError{Path: "/priority", Code: "enum"},
		metadata.FieldError{Path: "/summary", Code: "type"},
		metadata.FieldError{Path: "/title", Code: "required"})
	if n := count(t, e.super, "SELECT count(*) FROM records WHERE tenant_id = $1", f.id); n != 0 {
		t.Errorf("%d records, want 0", n)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", f.id); n != 0 {
		t.Errorf("%d audit rows, want 0", n)
	}
	if sink.calls() != 0 {
		t.Errorf("%d sink calls, want 0", sink.calls())
	}
}

// AC: two concurrent updates with the same expected version: exactly one
// succeeds, the other gets ErrVersionConflict, and there is one audit row.
// The second loads the record while the first is still open, so it passes
// the load check and reaches the UPDATE, which blocks on the first's row lock
// and then matches no row.
func TestConcurrentUpdatesOneWins(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	created := mustCreate(t, e, newPipeline(e, nil), f, map[string]any{"title": "printer"})

	first := newPipeline(e, nil)
	parked, release := make(chan struct{}), make(chan struct{})
	pipeline.SetBeforeReturn(first, func(context.Context) { close(parked); <-release })

	ctx := context.Background()
	var wg sync.WaitGroup
	var firstErr, secondErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, firstErr = write(ctx, e, first, f.id, updateCmd(f, created.RecordID, 1, map[string]any{"summary": "one"}))
	}()
	<-parked
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, secondErr = write(ctx, e, newPipeline(e, nil), f.id, updateCmd(f, created.RecordID, 1, map[string]any{"summary": "two"}))
	}()
	waitForLockWait(t, e)
	close(release)
	wg.Wait()

	if firstErr != nil {
		t.Fatalf("first update: %v", firstErr)
	}
	if !errors.Is(secondErr, pipeline.ErrVersionConflict) {
		t.Fatalf("second update: %v, want ErrVersionConflict", secondErr)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE record_id = $1 AND action = 'update'", created.RecordID); n != 1 {
		t.Fatalf("%d update audit rows, want 1", n)
	}
	data, _, _, version, _ := storedData(t, e, created.RecordID)
	if version != 2 || data["summary"] != "one" {
		t.Fatalf("version %d, summary %v; want 2, one", version, data["summary"])
	}

	// A stale version is refused at load time too.
	if _, err := write(ctx, e, newPipeline(e, nil), f.id, updateCmd(f, created.RecordID, 1, map[string]any{"summary": "x"})); !errors.Is(err, pipeline.ErrVersionConflict) {
		t.Fatalf("stale update: %v, want ErrVersionConflict", err)
	}
}

// AC: every successful write increases records.version by exactly 1, and
// Result.Version is the version after the write.
func TestEveryWriteBumpsVersionOnce(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	res := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	if _, _, _, v, _ := storedData(t, e, res.RecordID); v != 1 || res.Version != 1 {
		t.Fatalf("after create: stored %d, result %d; want 1", v, res.Version)
	}
	for want := int64(2); want <= 5; want++ {
		next, err := write(context.Background(), e, p, f.id, updateCmd(f, res.RecordID, res.Version, map[string]any{"summary": fmt.Sprint(want)}))
		if err != nil {
			t.Fatalf("update to %d: %v", want, err)
		}
		if _, _, _, v, _ := storedData(t, e, res.RecordID); v != want || next.Version != want {
			t.Fatalf("stored %d, result %d; want %d", v, next.Version, want)
		}
		res = next
	}
}

// AC: a record created on v1, then v2 publishes a defaulted field. An update
// of an unrelated field fills the default, never overwrites a present value,
// and moves the record to v2.
func TestUpdateUnderNewVersionFillsDefaults(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	res := mustCreate(t, e, p, f, map[string]any{"title": "printer", "priority": "high"})
	if res.SchemaVersion != 1 {
		t.Fatalf("created under v%d, want v1", res.SchemaVersion)
	}
	publishV2(t, e, f)

	up, err := write(context.Background(), e, p, f.id, updateCmd(f, res.RecordID, 1, map[string]any{"summary": "jammed"}))
	if err != nil {
		t.Fatal(err)
	}
	data, _, _, _, sv := storedData(t, e, res.RecordID)
	if up.SchemaVersion != 2 || sv != 2 {
		t.Fatalf("schema version result %d, stored %d; want 2", up.SchemaVersion, sv)
	}
	if data["channel"] != "email" || data["priority"] != "high" || data["summary"] != "jammed" {
		t.Fatalf("data = %v; want channel email, priority high (kept), summary jammed", data)
	}
}

// AC: merge-patch null on a required field with a default is refilled and
// accepted; without a default it gives ErrValidation at that path; an array
// replaces the stored array whole.
func TestMergePatchNullsAndArrays(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	res := mustCreate(t, e, p, f, map[string]any{"title": "printer", "priority": "high", "tags": []string{"a", "b", "c"}})

	res, err := write(ctx, e, p, f.id, updateCmd(f, res.RecordID, res.Version, map[string]any{"priority": nil}))
	if err != nil {
		t.Fatalf("null on a defaulted required field: %v", err)
	}
	if data, _, _, _, _ := storedData(t, e, res.RecordID); data["priority"] != "low" {
		t.Fatalf("priority = %v, want the default low", data["priority"])
	}

	_, err = write(ctx, e, p, f.id, updateCmd(f, res.RecordID, res.Version, map[string]any{"title": nil}))
	wantFields(t, err, metadata.FieldError{Path: "/title", Code: "required"})

	// Removing the state property leaves the NOT NULL state column nothing.
	_, err = write(ctx, e, p, f.id, updateCmd(f, res.RecordID, res.Version, map[string]any{"status": nil}))
	wantFields(t, err, metadata.FieldError{Path: "/status", Code: "required"})

	if _, err := write(ctx, e, p, f.id, updateCmd(f, res.RecordID, res.Version, map[string]any{"tags": []string{"q"}})); err != nil {
		t.Fatal(err)
	}
	if data, _, _, _, _ := storedData(t, e, res.RecordID); !reflect.DeepEqual(data["tags"], []any{"q"}) {
		t.Fatalf("tags = %v, want [q]", data["tags"])
	}
}

// AC: data never holds a property mapped to a fixed column, and a read
// returns it merged into the payload.
func TestFixedColumnsOnlyInTheirColumn(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	requester := f.principal.PrincipalID
	res := mustCreate(t, e, p, f, map[string]any{"title": "printer", "requester": requester.String()})

	data, state, req, _, _ := storedData(t, e, res.RecordID)
	if _, ok := data["status"]; ok {
		t.Errorf("data holds status: %v", data)
	}
	if _, ok := data["requester"]; ok {
		t.Errorf("data holds requester: %v", data)
	}
	if state != "open" || req == nil || *req != requester {
		t.Fatalf("columns state %q requester %v; want open (the initial state), %s", state, req, requester)
	}

	res, err := write(context.Background(), e, p, f.id, updateCmd(f, res.RecordID, res.Version, map[string]any{"status": "closed", "requester": nil}))
	if err != nil {
		t.Fatal(err)
	}
	data, state, req, _, _ = storedData(t, e, res.RecordID)
	if _, ok := data["status"]; ok || state != "closed" || req != nil {
		t.Fatalf("after update: data %v, state %q, requester %v", data, state, req)
	}

	inTx(t, e, f.id, func(tx pgx.Tx) error {
		rec, err := store.Get(context.Background(), tx, f.id, res.RecordID)
		if err != nil {
			return err
		}
		c, _, err := entitytypes.Store{}.Current(context.Background(), tx, entityKey)
		if err != nil {
			return err
		}
		got := rec.Payload(c.FixedColumns())
		want := map[string]any{"title": "printer", "priority": "low", "status": "closed"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("read payload = %v, want %v", got, want)
		}
		return nil
	})
}

// AC: each committed write has exactly one audit row and exactly one
// EventSink call inside the same transaction; a sink error rolls back the
// record and the audit row.
func TestAuditAndSinkInTheWriteTransaction(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	sink := &countingSink{}
	p := newPipeline(e, sink)
	origin := uuid.New()

	res := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	cmd := updateCmd(f, res.RecordID, 1, map[string]any{"summary": "x"})
	cmd.Source, cmd.Causation = pipeline.SourceRule, outbox.Causation{OriginID: origin, Depth: 1}
	if _, err := write(context.Background(), e, p, f.id, cmd); err != nil {
		t.Fatal(err)
	}
	if sink.calls() != 2 || !reflect.DeepEqual(sink.audits, []int{1, 2}) {
		t.Fatalf("sink calls %d, audit rows seen in the tx %v; want 2, [1 2]", sink.calls(), sink.audits)
	}
	defID := sink.events[0].EntityDefinitionID
	if want := (pipeline.Event{Op: pipeline.OpUpdate, EntityDefinitionID: defID, RecordID: res.RecordID, Version: 2,
		Causation: outbox.Causation{OriginID: origin, Depth: 1}}); sink.events[1] != want {
		t.Fatalf("event = %+v, want %+v", sink.events[1], want)
	}
	var action, source string
	var actor *uuid.UUID
	var depth int16
	var before, after []byte
	if err := e.super.QueryRow(context.Background(), `SELECT action, source, actor_id, causation_depth, before, after
		FROM audit_log WHERE record_id = $1 ORDER BY id DESC LIMIT 1`, res.RecordID).Scan(&action, &source, &actor, &depth, &before, &after); err != nil {
		t.Fatal(err)
	}
	if action != "update" || source != "rule" || actor == nil || *actor != f.principal.PrincipalID || depth != 1 {
		t.Fatalf("audit row %s %s %v depth %d", action, source, actor, depth)
	}
	if !strings.Contains(string(before), `"version": 1`) || !strings.Contains(string(after), `"version": 2`) ||
		!strings.Contains(string(after), `"status": "open"`) {
		t.Fatalf("audit before %s after %s", before, after)
	}

	failing := &countingSink{err: errors.New("sink down")}
	_, err := write(context.Background(), e, newPipeline(e, failing), f.id, createCmd(f, map[string]any{"title": "lost"}))
	if err == nil || !strings.Contains(err.Error(), "sink down") {
		t.Fatalf("err = %v, want the sink's", err)
	}
	if failing.calls() != 1 {
		t.Fatalf("%d sink calls, want 1", failing.calls())
	}
	lost := failing.events[0].RecordID
	if n := count(t, e.super, "SELECT count(*) FROM records WHERE id = $1", lost); n != 0 {
		t.Errorf("the failed write left its record")
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE record_id = $1", lost); n != 0 {
		t.Errorf("the failed write left its audit row")
	}
}

// AC: a second Write with the same IdempotencyKey returns the first Result
// and adds no audit row. A concurrent duplicate gets the retryable
// ErrIdempotencyConflict, and its retry replays.
func TestIdempotencyKeyReplays(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	ctx := context.Background()
	p := newPipeline(e, nil)
	cmd := createCmd(f, map[string]any{"title": "from a workflow"})
	cmd.Actor, cmd.Source, cmd.IdempotencyKey = f.system(), pipeline.SourceWorkflow, "wf:w1:s1"

	first, err := write(ctx, e, p, f.id, cmd)
	if err != nil {
		t.Fatal(err)
	}
	again, err := write(ctx, e, p, f.id, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Fatalf("replay = %+v, want %+v", again, first)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", f.id); n != 1 {
		t.Fatalf("%d audit rows, want 1", n)
	}
	if n := count(t, e.super, "SELECT count(*) FROM records WHERE tenant_id = $1", f.id); n != 1 {
		t.Fatalf("%d records, want 1", n)
	}

	// An update replay returns its first result even after later writes.
	up := updateCmd(f, first.RecordID, 1, map[string]any{"summary": "s2"})
	up.Actor, up.Source, up.IdempotencyKey = f.system(), pipeline.SourceWorkflow, "wf:w1:s2"
	upFirst, err := write(ctx, e, p, f.id, up)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := write(ctx, e, p, f.id, updateCmd(f, first.RecordID, 2, map[string]any{"summary": "user"})); err != nil {
		t.Fatal(err)
	}
	if upAgain, err := write(ctx, e, p, f.id, up); err != nil || upAgain != upFirst {
		t.Fatalf("update replay = %+v, %v; want %+v", upAgain, err, upFirst)
	}
	// The key names one write: reusing it for another record is a bug.
	other := mustCreate(t, e, p, f, map[string]any{"title": "other"})
	reuse := up
	reuse.RecordID = other.RecordID
	if _, err := write(ctx, e, p, f.id, reuse); !errors.Is(err, pipeline.ErrInvalidCommand) {
		t.Fatalf("key reused for another record: %v, want ErrInvalidCommand", err)
	}

	// Concurrent duplicate: the second's audit insert waits on the first's
	// uncommitted key, then hits 23505.
	dup := cmd
	dup.IdempotencyKey = "wf:w2:s1"
	holder := newPipeline(e, nil)
	parked, release := make(chan struct{}), make(chan struct{})
	pipeline.SetBeforeReturn(holder, func(context.Context) { close(parked); <-release })
	var wg sync.WaitGroup
	var holderErr, racerErr error
	var holderRes pipeline.Result
	wg.Add(1)
	go func() { defer wg.Done(); holderRes, holderErr = write(ctx, e, holder, f.id, dup) }()
	<-parked
	wg.Add(1)
	go func() { defer wg.Done(); _, racerErr = write(ctx, e, newPipeline(e, nil), f.id, dup) }()
	waitForLockWait(t, e)
	close(release)
	wg.Wait()
	if holderErr != nil {
		t.Fatal(holderErr)
	}
	if !errors.Is(racerErr, pipeline.ErrIdempotencyConflict) {
		t.Fatalf("racer: %v, want ErrIdempotencyConflict", racerErr)
	}
	if retry, err := write(ctx, e, p, f.id, dup); err != nil || retry != holderRes {
		t.Fatalf("retry = %+v, %v; want %+v", retry, err, holderRes)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE idempotency_key = 'wf:w2:s1'"); n != 1 {
		t.Fatalf("%d rows hold the key, want 1", n)
	}
}

// Tenant isolation: the tenant comes from the transaction, never the
// command. An actor of another tenant is refused before any statement, and
// another tenant's record or entity type reads as not found.
func TestCrossTenantWritesRefused(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	recA := mustCreate(t, e, p, a, map[string]any{"title": "tenant a"})

	// B's actor in A's transaction.
	if _, err := write(ctx, e, p, a.id, createCmd(b, map[string]any{"title": "x"})); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("other tenant's actor: %v, want ErrForbidden", err)
	}
	// A's record from B's transaction.
	if _, err := write(ctx, e, p, b.id, updateCmd(b, recA.RecordID, 1, map[string]any{"summary": "x"})); !errors.Is(err, pipeline.ErrNotFound) {
		t.Fatalf("other tenant's record: %v, want ErrNotFound", err)
	}
	// An entity type only A has.
	inTx(t, e, a.id, func(tx pgx.Tx) error {
		d := definition(schemaV1)
		d.Key, d.NumberPrefix = "change", "CHG"
		_, err := entitytypes.Store{}.Create(ctx, tx, a.id, entitytypes.NewInput{Name: "Change", PublishedBy: a.principal.PrincipalID, Definition: d})
		return err
	})
	cmd := createCmd(b, map[string]any{"title": "x"})
	cmd.EntityKey = "change"
	if _, err := write(ctx, e, p, b.id, cmd); !errors.Is(err, pipeline.ErrNotFound) {
		t.Fatalf("other tenant's entity type: %v, want ErrNotFound", err)
	}
	// An update naming the wrong entity type for the record.
	wrong := updateCmd(a, recA.RecordID, 1, map[string]any{"summary": "x"})
	wrong.EntityKey = "change"
	if _, err := write(ctx, e, p, a.id, wrong); !errors.Is(err, pipeline.ErrNotFound) {
		t.Fatalf("wrong entity type: %v, want ErrNotFound", err)
	}
	if _, _, _, v, _ := storedData(t, e, recA.RecordID); v != 1 {
		t.Fatalf("tenant A's record moved to version %d", v)
	}
	if n := count(t, e.super, "SELECT count(*) FROM records WHERE tenant_id = $1", b.id); n != 0 {
		t.Fatalf("tenant B has %d records", n)
	}
}

// The kill test's child process (TestPipelineKill starts it).
const (
	killHelperEnv = "NAVA_PIPELINE_KILL_HELPER"
	killDSNEnv    = "NAVA_PIPELINE_KILL_DSN"
	killTenantEnv = "NAVA_PIPELINE_KILL_TENANT"
	killActorEnv  = "NAVA_PIPELINE_KILL_ACTOR"
	killMarker    = "pipeline kill helper: parked before commit"
)

// TestPipelineKillProcess is not a test: TestPipelineKill runs it as a child
// process. It creates a record and parks in the pipeline's test-only hook,
// after every statement of the write and before the commit, until killed.
func TestPipelineKillProcess(t *testing.T) {
	if os.Getenv(killHelperEnv) != "1" {
		t.Skip("child process of TestPipelineKill")
	}
	tenant := uuid.MustParse(os.Getenv(killTenantEnv))
	actor := identity.Principal{TenantID: tenant, PrincipalID: uuid.MustParse(os.Getenv(killActorEnv))}
	e := env{app: appPool(t, os.Getenv(killDSNEnv))}
	p := newPipeline(e, nil)
	pipeline.SetBeforeReturn(p, func(context.Context) {
		fmt.Fprintln(os.Stderr, killMarker)
		select {} // parked until SIGKILL
	})
	_, err := write(context.Background(), e, p, tenant, pipeline.Command{Op: pipeline.OpCreate, EntityKey: entityKey,
		Data: map[string]any{"title": "killed"}, Source: pipeline.SourceUser, Actor: actor})
	t.Fatalf("the write returned instead of parking: %v", err)
}

// AC (real process kill, SIGKILL): a kill between the record insert and the
// commit leaves neither the record nor the audit row.
func TestPipelineKill(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestPipelineKillProcess$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), killHelperEnv+"=1", killDSNEnv+"="+appDSN(t, e.platform),
		killTenantEnv+"="+f.id.String(), killActorEnv+"="+f.principal.PrincipalID.String())
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	parked := make(chan struct{})
	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			if s.Text() == killMarker {
				close(parked)
			}
		}
	}()
	select {
	case <-parked:
	case <-time.After(60 * time.Second):
		t.Fatal("the child never parked")
	}

	// The insert and the audit row are written and uncommitted: the child's
	// transaction is open and holds the new row's lock.
	if n := count(t, e.super, `SELECT count(*) FROM pg_stat_activity WHERE usename = 'nava_app' AND state = 'idle in transaction'`); n == 0 {
		t.Fatal("no open nava_app transaction while the child is parked")
	}
	if err := cmd.Process.Kill(); err != nil { // SIGKILL on Unix
		t.Fatal(err)
	}
	_ = cmd.Wait()

	deadline := time.Now().Add(30 * time.Second)
	for count(t, e.super, `SELECT count(*) FROM pg_stat_activity WHERE usename = 'nava_app' AND state = 'idle in transaction'`) != 0 {
		if time.Now().After(deadline) {
			t.Fatal("the killed child's transaction is still open after 30 s")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// The tenant is fresh: the child's write is the only one it ever had.
	if n := count(t, e.super, "SELECT count(*) FROM records WHERE tenant_id = $1", f.id); n != 0 {
		t.Errorf("the killed write left %d records", n)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", f.id); n != 0 {
		t.Errorf("the killed write left %d audit rows", n)
	}
	// The tenant still writes normally.
	mustCreate(t, e, newPipeline(e, nil), f, map[string]any{"title": "after the kill"})
}
