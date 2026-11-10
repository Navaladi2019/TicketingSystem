//go:build integration

// TICK-692 (PLAN-003 P3-36i): a record create in 4 round trips. Round trip 1
// is BEGIN, the tenant, the idempotency claim under lock_timeout and the
// prefetch, in one batch; round trip 4 is the record, the audit entry, the
// key's completion and COMMIT, in one batch. A failure in either batch rolls
// the whole transaction back: no idempotency_keys row, no record, and the
// counter as ADR-005 has it (unchanged, except the one number a `gaps`
// create committed in round trip 3 before its round trip 4 failed: a gap).

package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// installTripFailures makes the claim INSERT (round trip 1) of a key with
// prefix "k-692-rt1-", and the completion UPDATE (round trip 4) of a key
// with prefix "k-692-rt4-", raise.
func installTripFailures(t *testing.T, e *etEnv) {
	t.Helper()
	for _, q := range []string{
		`CREATE FUNCTION fail_692() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF (TG_OP = 'INSERT' AND NEW.key LIKE 'k-692-rt1-%') OR (TG_OP = 'UPDATE' AND NEW.key LIKE 'k-692-rt4-%') THEN
    RAISE EXCEPTION 'injected failure in round trip %', CASE TG_OP WHEN 'INSERT' THEN 1 ELSE 4 END;
  END IF;
  RETURN NEW;
END $$`,
		`CREATE TRIGGER fail_692 BEFORE INSERT OR UPDATE ON idempotency_keys FOR EACH ROW EXECUTE FUNCTION fail_692()`,
	} {
		if _, err := e.super.Exec(t.Context(), q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	t.Cleanup(func() {
		_, _ = e.super.Exec(t.Context(), `DROP TRIGGER IF EXISTS fail_692 ON idempotency_keys`)
		_, _ = e.super.Exec(t.Context(), `DROP FUNCTION IF EXISTS fail_692()`)
	})
}

// recordsNamed counts tenant's records whose short_description is name.
func recordsNamed(t *testing.T, e *etEnv, tenant uuid.UUID, name string) int {
	t.Helper()
	var n int
	if err := e.super.QueryRow(t.Context(),
		`SELECT count(*) FROM records WHERE tenant_id = $1 AND data->>'short_description' = $2`, tenant, name).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// covers: createRecord 500
func TestCreateRecordFailureInFirstOrLastTripLeavesNothing(t *testing.T) {
	e, g := guardedRecEnv(t, 4)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "t-gl", gaplessTypeBody("gl", "GL")); rec.Code != http.StatusCreated {
		t.Fatalf("seed gapless type = %d: %s", rec.Code, rec.Body)
	}
	// One number taken in each type first, so a counter row exists.
	e.mustCreateRec(t, "user-a", "ticket", "seed-gaps", `{"short_description": "seed"}`)
	e.mustCreateRec(t, "user-a", "gl", "seed-gapless", `{"short_description": "seed"}`)
	installTripFailures(t, e)

	for _, c := range []struct {
		name, entity, trip string
		gap                int64 // counter change: a gaps create's committed number
	}{
		{"round trip 1, gaps", "ticket", "rt1", 0},
		{"round trip 1, gapless", "gl", "rt1", 0},
		{"round trip 4, gaps", "ticket", "rt4", 1},
		{"round trip 4, gapless", "gl", "rt4", 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			key := "k-692-" + c.trip + "-" + c.entity
			name := "fail-" + c.trip + "-" + c.entity
			before := counterNext(t, e, e.a.id, c.entity)
			w, rec := e.traced(t, http.MethodPost, "/api/v1/records/"+c.entity, "user-a", key,
				`{"data": {"short_description": "`+name+`"}}`)
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("create = %d, want 500: %s", w.Code, w.Body)
			}
			if n := keyRows(t, e, key); n != 0 {
				t.Errorf("%d idempotency_keys rows after the failed create, want 0", n)
			}
			if n := recordsNamed(t, e, e.a.id, name); n != 0 {
				t.Errorf("%d records after the failed create, want 0", n)
			}
			if after := counterNext(t, e, e.a.id, c.entity); after != before+c.gap {
				t.Errorf("counter next %d -> %d, want +%d", before, after, c.gap)
			}
			// The failure is where the case says: the claim in the first
			// trip, or the completion in the last trip before the rollback,
			// in the batch that also carries the record (gaps) and COMMIT.
			trips := rec.Trips()
			if len(trips) < 2 || strings.ToLower(trips[len(trips)-1].SQL[0]) != "rollback" {
				t.Fatalf("trips %v, want the failed batch then ROLLBACK", trips)
			}
			failed := trips[len(trips)-2]
			switch c.trip {
			case "rt1":
				if len(trips) != 2 || !failed.Batch || failed.SQL[0] != "BEGIN" ||
					!strings.Contains(strings.Join(failed.SQL, " "), "INSERT INTO idempotency_keys") {
					t.Errorf("trips %v, want only the first batch (BEGIN ... claim ...) and ROLLBACK", trips)
				}
			case "rt4":
				all := strings.Join(failed.SQL, " ")
				if len(trips) != 5 || !failed.Batch || failed.SQL[len(failed.SQL)-1] != "COMMIT" ||
					!strings.Contains(all, "UPDATE idempotency_keys") {
					t.Errorf("trips %v, want 4 round trips, the last (completion and COMMIT) failed, then ROLLBACK", trips)
				}
				// A gaps create's record and audit entry ride in that batch;
				// a gapless create's went in the trip before (its number is
				// taken there, ADR-005 L1).
				if c.entity == "ticket" && (!strings.Contains(all, "INSERT INTO records") || !strings.Contains(all, "INSERT INTO audit_log")) {
					t.Errorf("the failed last trip %q has no record or audit INSERT, want both in the COMMIT's batch", failed.SQL)
				}
			}
			// The ROLLBACK after the failed batch runs on a connection the
			// Guard already saw COMMIT on; every trip before it ran under A.
			for i, tr := range trips[:len(trips)-1] {
				if tr.Tenant != e.a.id {
					t.Errorf("trip %d ran under tenant %s, want %s: %q", i, tr.Tenant, e.a.id, tr.SQL)
				}
			}
		})
	}
	if v := g.Violations(); len(v) > 0 {
		t.Errorf("statements ran with no tenant set: %q", v)
	}
	// The connection pool is still sound: a create right after succeeds.
	e.mustCreateRec(t, "user-a", "ticket", "after-failures", `{"short_description": "after"}`)
}
