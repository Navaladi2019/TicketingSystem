//go:build integration

// TICK-688 (PLAN-003 P3-36h): record create claims its Idempotency-Key inside
// its own write transaction, the first write, so the claim, the record and
// the stored response commit together in one synchronous commit. A concurrent
// duplicate waits on the uncommitted key for at most idempotency.ClaimWait:
// it replays when the first request commits, runs as the creator when the
// first rolls back, and gets 409 idempotency_in_flight past the wait. A
// failed create leaves no idempotency_keys row.
//
// The first request is held inside its INSERT INTO records by a trigger that
// takes an advisory lock the test holds; the test watches pg_locks to know
// when each request is where it should be.

package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// gateLock is the advisory lock the gate trigger waits on.
const gateLock = 688

// installGate makes every INSERT INTO records whose short_description is
// "gated" wait for gateLock first. With failFirst, the first such insert
// (a sequence, which does not roll back) then raises, a 500 after the claim;
// later ones go ahead.
func installGate(t *testing.T, e *etEnv) {
	t.Helper()
	for _, q := range []string{
		`CREATE SEQUENCE gate_688_seq`,
		`CREATE TABLE gate_688_fail (on_first boolean NOT NULL)`,
		`CREATE FUNCTION gate_688() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
  IF NEW.data->>'short_description' = 'gated' THEN
    PERFORM pg_advisory_xact_lock(688);
    IF nextval('gate_688_seq') = 1 AND EXISTS (SELECT 1 FROM gate_688_fail WHERE on_first) THEN
      RAISE EXCEPTION 'injected failure after the claim';
    END IF;
  END IF;
  RETURN NEW;
END $$`,
		`CREATE TRIGGER gate_688 BEFORE INSERT ON records FOR EACH ROW EXECUTE FUNCTION gate_688()`,
	} {
		if _, err := e.super.Exec(t.Context(), q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
}

func failFirstGated(t *testing.T, e *etEnv) {
	t.Helper()
	if _, err := e.super.Exec(t.Context(), `INSERT INTO gate_688_fail VALUES (true)`); err != nil {
		t.Fatal(err)
	}
}

// closeGate takes gateLock on the test's connection; openGate releases it.
func closeGate(t *testing.T, e *etEnv) {
	t.Helper()
	if _, err := e.super.Exec(t.Context(), `SELECT pg_advisory_lock($1)`, gateLock); err != nil {
		t.Fatal(err)
	}
}

func openGate(t *testing.T, e *etEnv) {
	t.Helper()
	if _, err := e.super.Exec(t.Context(), `SELECT pg_advisory_unlock($1)`, gateLock); err != nil {
		t.Fatal(err)
	}
}

// waitLocks waits until n lock requests of locktype are waiting (not granted).
func waitLocks(t *testing.T, e *etEnv, locktype string, n int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var got int
		if err := e.super.QueryRow(t.Context(),
			`SELECT count(*) FROM pg_locks WHERE locktype = $1 AND NOT granted`, locktype).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d waiting %s locks after 10s, want %d", got, locktype, n)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// goCreate sends a create in the background; the result arrives on the
// channel. It never calls t.Fatal (not allowed off the test goroutine).
func goCreate(t *testing.T, e *etEnv, token, entity, key, data string) <-chan *httptest.ResponseRecorder {
	ch := make(chan *httptest.ResponseRecorder, 1)
	go func() { ch <- e.createRec(t, token, entity, key, data) }()
	return ch
}

func recv(t *testing.T, ch <-chan *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(30 * time.Second):
		t.Fatal("no response after 30s")
		return nil
	}
}

func keyRows(t *testing.T, e *etEnv, key string) int {
	t.Helper()
	var n int
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM idempotency_keys WHERE key = $1`, key).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// counterNext is the tenant's record_counters.next for entity (0 when no
// number was ever taken).
func counterNext(t *testing.T, e *etEnv, tenant uuid.UUID, entity string) int64 {
	t.Helper()
	var n int64
	if err := e.super.QueryRow(t.Context(), `
SELECT coalesce((SELECT c.next FROM record_counters c
                   JOIN entity_definitions d ON d.tenant_id = c.tenant_id AND d.id = c.entity_definition_id
                  WHERE d.tenant_id = $1 AND d.key = $2), 0)`, tenant, entity).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func decodeRec(t *testing.T, r *httptest.ResponseRecorder) recBody {
	t.Helper()
	var b recBody
	if err := json.Unmarshal(r.Body.Bytes(), &b); err != nil {
		t.Fatalf("body is not a record: %v: %s", err, r.Body)
	}
	return b
}

// covers: createRecord 201
// Two creates with one key, the second sent while the first holds its
// uncommitted claim: exactly one record; the second waits, then replays the
// first's response, record id and number.
func TestCreateRecordConcurrentSameKeyWaitsThenReplays(t *testing.T) {
	e := newRecEnv(t)
	installGate(t, e)
	closeGate(t, e)
	first := goCreate(t, e, "user-a", "ticket", "k-688-dup", `{"short_description": "gated"}`)
	waitLocks(t, e, "advisory", 1) // the first holds its claim, inside its insert
	second := goCreate(t, e, "user-a", "ticket", "k-688-dup", `{"short_description": "gated"}`)
	waitLocks(t, e, "transactionid", 1) // the second waits on the first's key
	openGate(t, e)
	r1, r2 := recv(t, first), recv(t, second)
	if r1.Code != http.StatusCreated || r1.Header().Get(idempotency.ReplayedHeader) == "true" {
		t.Fatalf("first = %d replayed=%q: %s", r1.Code, r1.Header().Get(idempotency.ReplayedHeader), r1.Body)
	}
	if r2.Code != http.StatusCreated || r2.Header().Get(idempotency.ReplayedHeader) != "true" || r2.Body.String() != r1.Body.String() {
		t.Fatalf("second = %d replayed=%q, want the first's 201 replayed:\n%s\nwant:\n%s",
			r2.Code, r2.Header().Get(idempotency.ReplayedHeader), r2.Body, r1.Body)
	}
	b1, b2 := decodeRec(t, r1), decodeRec(t, r2)
	if b1.ID != b2.ID || b1.Number != b2.Number || r1.Header().Get("ETag") != r2.Header().Get("ETag") {
		t.Errorf("replay id/number/ETag %s/%s/%s, want %s/%s/%s", b2.ID, b2.Number, r2.Header().Get("ETag"),
			b1.ID, b1.Number, r1.Header().Get("ETag"))
	}
	if n := e.recordCount(t, "a"); n != 1 {
		t.Errorf("tenant A has %d records, want exactly 1", n)
	}
	if n := keyRows(t, e, "k-688-dup"); n != 1 {
		t.Errorf("%d key rows, want 1", n)
	}
}

// The first create rolls back (a 500 after its claim) while a duplicate waits
// on its key: the duplicate then creates the record. One record, one key row,
// the duplicate's.
func TestCreateRecordRollbackUnderAWaiterLetsTheWaiterCreate(t *testing.T) {
	e := newRecEnv(t)
	installGate(t, e)
	failFirstGated(t, e)
	closeGate(t, e)
	first := goCreate(t, e, "user-a", "ticket", "k-688-rb", `{"short_description": "gated"}`)
	waitLocks(t, e, "advisory", 1)
	second := goCreate(t, e, "user-a", "ticket", "k-688-rb", `{"short_description": "gated"}`)
	waitLocks(t, e, "transactionid", 1)
	openGate(t, e)
	r1, r2 := recv(t, first), recv(t, second)
	if r1.Code != http.StatusInternalServerError {
		t.Fatalf("first = %d, want the injected 500: %s", r1.Code, r1.Body)
	}
	if r2.Code != http.StatusCreated || r2.Header().Get(idempotency.ReplayedHeader) == "true" {
		t.Fatalf("second = %d replayed=%q, want a fresh 201: %s", r2.Code, r2.Header().Get(idempotency.ReplayedHeader), r2.Body)
	}
	if n := e.recordCount(t, "a"); n != 1 {
		t.Errorf("tenant A has %d records, want exactly 1", n)
	}
	var state string
	var stored []byte
	if err := e.super.QueryRow(t.Context(),
		`SELECT state, response_body FROM idempotency_keys WHERE key = 'k-688-rb'`).Scan(&state, &stored); err != nil {
		t.Fatal(err)
	}
	if state != "completed" || string(stored) != r2.Body.String() {
		t.Errorf("key = %s %s, want completed with the second's 201", state, stored)
	}
	again := e.createRec(t, "user-a", "ticket", "k-688-rb", `{"short_description": "gated"}`)
	if again.Code != http.StatusCreated || again.Body.String() != r2.Body.String() || again.Header().Get(idempotency.ReplayedHeader) != "true" {
		t.Errorf("third = %d, want the second's 201 replayed: %s", again.Code, again.Body)
	}
}

// covers: createRecord 409
// The first create holds its claim past ClaimWait: the duplicate gets 409
// idempotency_in_flight with Retry-After after about ClaimWait, gives its
// connection back, and leaves no row; the first then commits normally.
func TestCreateRecordDuplicatePastClaimWaitIs409(t *testing.T) {
	e := newRecEnv(t)
	installGate(t, e)
	closeGate(t, e)
	first := goCreate(t, e, "user-a", "ticket", "k-688-wait", `{"short_description": "gated"}`)
	waitLocks(t, e, "advisory", 1)
	start := time.Now()
	dup := e.createRec(t, "user-a", "ticket", "k-688-wait", `{"short_description": "gated"}`)
	took := time.Since(start)
	wantProblem(t, dup, http.StatusConflict, problem.IdempotencyInFlight)
	if got, want := dup.Header().Get("Retry-After"), strconv.Itoa(int(idempotency.Lease/time.Second)); got != want {
		t.Errorf("Retry-After = %q, want %s", got, want)
	}
	if took < idempotency.ClaimWait-100*time.Millisecond || took > idempotency.ClaimWait+3*time.Second {
		t.Errorf("the duplicate answered after %v, want about ClaimWait (%v)", took, idempotency.ClaimWait)
	}
	// Only the first request holds a connection of the handler's pool now.
	if n := e.pool.Stat().AcquiredConns(); n != 1 {
		t.Errorf("%d connections acquired after the 409, want 1 (the first create's)", n)
	}
	var waiting int
	if err := e.super.QueryRow(t.Context(),
		`SELECT count(*) FROM pg_locks WHERE locktype = 'transactionid' AND NOT granted`).Scan(&waiting); err != nil {
		t.Fatal(err)
	}
	if waiting != 0 {
		t.Errorf("%d lock waits left after the 409, want 0", waiting)
	}
	openGate(t, e)
	r1 := recv(t, first)
	if r1.Code != http.StatusCreated {
		t.Fatalf("first = %d after the gate opened: %s", r1.Code, r1.Body)
	}
	if n := e.recordCount(t, "a"); n != 1 {
		t.Errorf("tenant A has %d records, want exactly 1", n)
	}
	if n := keyRows(t, e, "k-688-wait"); n != 1 {
		t.Errorf("%d key rows, want 1 (the first's, no orphan from the 409)", n)
	}
	again := e.createRec(t, "user-a", "ticket", "k-688-wait", `{"short_description": "gated"}`)
	if again.Code != http.StatusCreated || again.Body.String() != r1.Body.String() {
		t.Errorf("retry after the first committed = %d, want its 201 replayed: %s", again.Code, again.Body)
	}
}

// gaplessTypeBody is recTypeBody with numbering_mode gapless.
func gaplessTypeBody(key, prefix string) string {
	return strings.Replace(recTypeBody(key, prefix), `"number_prefix": "`+prefix+`",`,
		`"number_prefix": "`+prefix+`", "numbering_mode": "gapless",`, 1)
}

// A failed create (invalid, unauthorised, a 500 after the claim) leaves no
// idempotency_keys row: its claim rolled back with it. The invalid and the
// unauthorised create take no number in either mode. A 500 at the insert
// takes none in gapless mode; in gaps mode its number, committed on the
// allocator pool before the insert, is a gap (D-3-9).
func TestCreateRecordFailureLeavesNoKeyRow(t *testing.T) {
	e := newRecEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "t-gl", gaplessTypeBody("gl", "GL")); rec.Code != http.StatusCreated {
		t.Fatalf("seed gapless type = %d: %s", rec.Code, rec.Body)
	}
	e.noTestACL = true // no published ACL: every create of "bare" is 403
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "t-bare", recTypeBody("bare", "BAR")); rec.Code != http.StatusCreated {
		t.Fatalf("seed bare type = %d: %s", rec.Code, rec.Body)
	}
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "t-bare-gl", gaplessTypeBody("baregl", "BGL")); rec.Code != http.StatusCreated {
		t.Fatalf("seed bare gapless type = %d: %s", rec.Code, rec.Body)
	}
	e.noTestACL = false
	// One number taken in each type first, so a counter row exists.
	e.mustCreateRec(t, "user-a", "ticket", "seed-gaps", `{"short_description": "seed"}`)
	e.mustCreateRec(t, "user-a", "gl", "seed-gapless", `{"short_description": "seed"}`)
	installGate(t, e)
	failFirstGated(t, e)

	for _, c := range []struct {
		name, entity, data string
		status             int
		takesNumber        bool
	}{
		{"invalid, gaps", "ticket", `{}`, http.StatusUnprocessableEntity, false},
		{"invalid, gapless", "gl", `{}`, http.StatusUnprocessableEntity, false},
		{"unauthorised, gaps", "bare", `{"short_description": "x"}`, http.StatusForbidden, false},
		{"unauthorised, gapless", "baregl", `{"short_description": "x"}`, http.StatusForbidden, false},
		{"500 after the claim, gapless", "gl", `{"short_description": "gated"}`, http.StatusInternalServerError, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			key := "k-688-fail-" + c.entity + "-" + strconv.Itoa(c.status)
			before := counterNext(t, e, e.a.id, c.entity)
			rec := e.createRec(t, "user-a", c.entity, key, c.data)
			if rec.Code != c.status {
				t.Fatalf("create = %d, want %d: %s", rec.Code, c.status, rec.Body)
			}
			if n := keyRows(t, e, key); n != 0 {
				t.Errorf("%d key rows after the failed create, want 0", n)
			}
			if after := counterNext(t, e, e.a.id, c.entity); after != before {
				t.Errorf("counter next %d -> %d, want unchanged", before, after)
			}
		})
	}
	// Reset the one-shot failure for the gaps-mode 500.
	if _, err := e.super.Exec(t.Context(), `ALTER SEQUENCE gate_688_seq RESTART`); err != nil {
		t.Fatal(err)
	}
	t.Run("500 after the claim, gaps", func(t *testing.T) {
		before := counterNext(t, e, e.a.id, "ticket")
		rec := e.createRec(t, "user-a", "ticket", "k-688-fail-gaps-500", `{"short_description": "gated"}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("create = %d, want 500: %s", rec.Code, rec.Body)
		}
		if n := keyRows(t, e, "k-688-fail-gaps-500"); n != 0 {
			t.Errorf("%d key rows after the failed create, want 0", n)
		}
		if after := counterNext(t, e, e.a.id, "ticket"); after != before+1 {
			t.Errorf("counter next %d -> %d, want one gap (D-3-9)", before, after)
		}
		// The retry runs at once, with no lease to wait out.
		retry := e.createRec(t, "user-a", "ticket", "k-688-fail-gaps-500", `{"short_description": "gated"}`)
		if retry.Code != http.StatusCreated || retry.Header().Get(idempotency.ReplayedHeader) == "true" {
			t.Errorf("retry = %d, want a fresh 201: %s", retry.Code, retry.Body)
		}
	})
}

// A replay takes no number, in gaps and in gapless mode (I4, ADR-005).
func TestCreateRecordReplayTakesNoNumber(t *testing.T) {
	e := newRecEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "t-gl", gaplessTypeBody("gl", "GL")); rec.Code != http.StatusCreated {
		t.Fatalf("seed gapless type = %d: %s", rec.Code, rec.Body)
	}
	for _, entity := range []string{"ticket", "gl"} {
		t.Run(entity, func(t *testing.T) {
			key := "k-688-replay-" + entity
			first := e.mustCreateRec(t, "user-a", entity, key, `{"short_description": "once"}`)
			after := counterNext(t, e, e.a.id, entity)
			again := e.createRec(t, "user-a", entity, key, `{"short_description": "once"}`)
			if again.Code != http.StatusCreated || again.Header().Get(idempotency.ReplayedHeader) != "true" {
				t.Fatalf("replay = %d replayed=%q: %s", again.Code, again.Header().Get(idempotency.ReplayedHeader), again.Body)
			}
			if b := decodeRec(t, again); b.Number != first.Number {
				t.Errorf("replay number %s, want %s", b.Number, first.Number)
			}
			if n := counterNext(t, e, e.a.id, entity); n != after {
				t.Errorf("counter next %d -> %d after the replay, want unchanged", after, n)
			}
			next := e.mustCreateRec(t, "user-a", entity, key+"-next", `{"short_description": "two"}`)
			if next.Number == first.Number {
				t.Errorf("the next create reused number %s", first.Number)
			}
		})
	}
}

// The same key in two tenants: two independent records, neither a replay.
func TestCreateRecordSameKeyInTwoTenantsCreatesTwo(t *testing.T) {
	e := newRecEnv(t)
	ra := e.createRec(t, "user-a", "ticket", "k-688-shared", `{"short_description": "same"}`)
	rb := e.createRec(t, "user-b", "incident", "k-688-shared", `{"short_description": "same"}`)
	for name, r := range map[string]*httptest.ResponseRecorder{"A": ra, "B": rb} {
		if r.Code != http.StatusCreated || r.Header().Get(idempotency.ReplayedHeader) == "true" {
			t.Fatalf("tenant %s create = %d replayed=%q: %s", name, r.Code, r.Header().Get(idempotency.ReplayedHeader), r.Body)
		}
	}
	if decodeRec(t, ra).ID == decodeRec(t, rb).ID {
		t.Error("the two tenants' creates returned one record")
	}
	if a, b := e.recordCount(t, "a"), e.recordCount(t, "b"); a != 1 || b != 1 {
		t.Errorf("records A=%d B=%d, want 1 each", a, b)
	}
	var tenants int
	if err := e.super.QueryRow(context.Background(),
		`SELECT count(DISTINCT tenant_id) FROM idempotency_keys WHERE key = 'k-688-shared'`).Scan(&tenants); err != nil {
		t.Fatal(err)
	}
	if tenants != 2 {
		t.Errorf("key rows in %d tenants, want 2", tenants)
	}
}

// Adversarial (Security review, TICK-688): tenant B holds the key open,
// uncommitted, inside its create; tenant A's create with the same key must
// neither wait on B's claim, nor replay B's response, nor get B's 409. The
// claim's conflict target includes tenant_id, so B's uncommitted row is no
// lock A can meet: A answers 201 well inside ClaimWait.
func TestCreateRecordSameKeyHeldOpenInOtherTenantDoesNotBlock(t *testing.T) {
	e := newRecEnv(t)
	installGate(t, e)
	closeGate(t, e)
	held := goCreate(t, e, "user-b", "incident", "k-688-xt", `{"short_description": "gated"}`)
	waitLocks(t, e, "advisory", 1) // B's claim is in, uncommitted; B waits at its insert
	start := time.Now()
	ra := e.createRec(t, "user-a", "ticket", "k-688-xt", `{"short_description": "free"}`)
	took := time.Since(start)
	if ra.Code != http.StatusCreated || ra.Header().Get(idempotency.ReplayedHeader) == "true" {
		t.Fatalf("tenant A create = %d replayed=%q while B held the key: %s",
			ra.Code, ra.Header().Get(idempotency.ReplayedHeader), ra.Body)
	}
	if took >= idempotency.ClaimWait/2 {
		t.Errorf("tenant A waited %v on tenant B's uncommitted key, want no wait", took)
	}
	var waiting int
	if err := e.super.QueryRow(t.Context(),
		`SELECT count(*) FROM pg_locks WHERE locktype = 'transactionid' AND NOT granted`).Scan(&waiting); err != nil {
		t.Fatal(err)
	}
	if waiting != 0 {
		t.Errorf("%d transactionid lock waits while B holds its key, want 0", waiting)
	}
	openGate(t, e)
	rb := recv(t, held)
	if rb.Code != http.StatusCreated || rb.Header().Get(idempotency.ReplayedHeader) == "true" {
		t.Fatalf("tenant B create = %d replayed=%q: %s", rb.Code, rb.Header().Get(idempotency.ReplayedHeader), rb.Body)
	}
	if decodeRec(t, ra).ID == decodeRec(t, rb).ID {
		t.Error("the two tenants' creates returned one record")
	}
	if a, b := e.recordCount(t, "a"), e.recordCount(t, "b"); a != 1 || b != 1 {
		t.Errorf("records A=%d B=%d, want 1 each", a, b)
	}
}
