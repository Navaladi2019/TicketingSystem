//go:build integration

// TICK-873 (PLAN-005-E9 T-E9-4; ADR-004-E9 CI-A18 (d), (h), Sec P2):
// adversarial tests of the number_space_exhausted audit entry. The attacker
// holds a valid token for tenant A. The entry a refused create appends must
// land in A's audit log only (never B's, never through B's entity key), and
// an audit.read principal whose field masks hide record fields of the type
// sees exactly the four diff keys under /number_space and no record data.
// The refusal log line never names the scope key.
//
// Through the real handler on a real PostgreSQL, as nava_app (RLS forced)
// and nava_audit_reader, with the etEnv token resolver every internal/server
// adversarial test uses.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// exhBKey is tenant B's own full refuse type. Tenant A has no type of
// that key, so a create of it under A's token can only reach B's.
const exhBKey = "exhb"

// seedFullTypeB gives tenant B the type exhb, linked to a one-digit refuse
// format and filled to EXHB-9, so a create that reached it would be refused
// and append an entry in B.
func (x *exhEnv) seedFullTypeB(t *testing.T) uuid.UUID {
	t.Helper()
	if rec := x.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", linkKey("exhb-type"), etBody(exhBKey, "EXHB")); rec.Code != http.StatusCreated {
		t.Fatalf("create B's type = %d: %s", rec.Code, rec.Body)
	}
	rec := x.do(t, http.MethodPost, nfPath, "admin-b", linkKey("exhb-fmt"),
		`{"key":"exhb_one_digit","document":{"template":"{prefix}-{seq:1}","overflow":"refuse"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create B's format = %d: %s", rec.Code, rec.Body)
	}
	if rec := x.doIf(t, http.MethodPost, "/api/v1/entity-types/"+exhBKey+"/versions", "admin-b", linkKey("exhb-link"), "", pubBody(ref(nfID(t, rec.Body.Bytes()), 1))); rec.Code != http.StatusCreated {
		t.Fatalf("link B's format = %d: %s", rec.Code, rec.Body)
	}
	for i := 1; i <= 9; i++ {
		if r := x.mustCreateRec(t, "user-b", exhBKey, linkKey("exhb-fill"), `{"short_description": "fill"}`); r.Number != fmt.Sprintf("EXHB-%d", i) {
			t.Fatalf("B's create %d: number %s", i, r.Number)
		}
	}
	// Prove the setup: B's own next create is refused (and leaves B one
	// entry, which the test then counts from).
	wantProblem(t, x.createRec(t, "user-b", exhBKey, linkKey("exhb-own"), `{"short_description": "b's own"}`), http.StatusConflict, problem.NumberSpaceExhausted)
	var def uuid.UUID
	if err := x.super.QueryRow(t.Context(), "SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2", x.b.id, exhBKey).Scan(&def); err != nil {
		t.Fatal(err)
	}
	return def
}

// appCount counts number_space_exhausted entries as nava_app sees them in
// tenant's transaction: no tenant predicate, so only RLS scopes the count.
func (x *exhEnv) appCount(t *testing.T, tenant uuid.UUID) int {
	t.Helper()
	var n int
	if err := tenancy.InTenantTx(t.Context(), x.pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), "SELECT count(*) FROM audit_log WHERE action = 'number_space_exhausted'").Scan(&n)
	}); err != nil {
		t.Fatal(err)
	}
	return n
}

// timelineExhausted reads the audit timeline as token (an audit.read
// principal) and returns its number_space_exhausted items.
func (x *exhEnv) timelineExhausted(t *testing.T, token, query string) []json.RawMessage {
	t.Helper()
	rec, page := x.auditGet(t, token, query)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET audit/entries%s as %s = %d: %s", query, token, rec.Code, rec.Body)
	}
	if page.Next != "" {
		t.Fatalf("GET audit/entries%s as %s has a second page; the counts below would miss it", query, token)
	}
	var out []json.RawMessage
	for _, it := range page.Items {
		if it.Action == "number_space_exhausted" {
			out = append(out, it.Diff)
		}
	}
	return out
}

// CI-A18 (d) [Sec P2]: three refusals under tenant A's token append three
// entries to A's audit log and none to B's, whether B reads through its
// audit.read principal's timeline or as nava_app in its own transaction.
// Fails if the append runs under any tenant other than the token's.
//
// covers: createRecord 409
// covers: listAuditEntries 200
func TestNumberSpaceExhaustedAdversarialEntryStaysInTheTokensTenant(t *testing.T) {
	for _, mode := range exhModes {
		t.Run(mode, func(t *testing.T) {
			x := newExhEnv(t, mode)
			for range 3 {
				x.refuse(t, linkKey("exh-adv-d"))
			}
			if n := x.appCount(t, x.b.id); n != 0 {
				t.Errorf("tenant B (nava_app, B's transaction) sees %d number_space_exhausted entries, want 0", n)
			}
			if got := x.timelineExhausted(t, "admin-b", ""); len(got) != 0 {
				t.Errorf("tenant B's audit timeline has %d number_space_exhausted entries, want 0: %s", len(got), got)
			}
			if n := x.appCount(t, x.a.id); n != 3 {
				t.Errorf("tenant A (nava_app, A's transaction) sees %d number_space_exhausted entries, want 3", n)
			}
			if got := x.timelineExhausted(t, "admin-a", "?subject_type=entity_definition&subject_id="+x.defID.String()); len(got) != 3 {
				t.Errorf("tenant A's audit timeline has %d number_space_exhausted entries, want 3", len(got))
			}
			// As stored, past RLS: every one of the three is A's.
			if rows := x.exhRows(t, x.a.id); len(rows) != 3 {
				t.Errorf("audit_log holds %d of A's number_space_exhausted entries, want 3", len(rows))
			}
			if rows := x.exhRows(t, x.b.id); len(rows) != 0 {
				t.Errorf("audit_log holds %d of B's number_space_exhausted entries, want 0", len(rows))
			}
		})
	}
}

// Forged tenant (as TICK-676): tenant A's token against tenant B's full type
// key is the same 404 not_found as a key no tenant has (no existence
// oracle), refuses nothing and appends no entry in either tenant.
//
// covers: createRecord 404
func TestNumberSpaceExhaustedAdversarialForgedTenantEntityKey(t *testing.T) {
	x := newExhEnv(t, "gaps")
	defB := x.seedFullTypeB(t)
	beforeB := len(x.exhRows(t, x.b.id))
	if beforeB != 1 {
		t.Fatalf("tenant B has %d entries after its own refusal, want 1", beforeB)
	}

	forged := x.createRec(t, "user-a", exhBKey, linkKey("exh-adv-forged"), `{"short_description": "forged"}`)
	unknown := x.createRec(t, "user-a", "nosuchtype", linkKey("exh-adv-unknown"), `{"short_description": "forged"}`)
	wantProblem(t, forged, http.StatusNotFound, problem.NotFound)
	wantProblem(t, unknown, http.StatusNotFound, problem.NotFound)
	if f, u := forged.Body.String(), unknown.Body.String(); f != u {
		t.Errorf("B's key and an unknown key answer differently (existence oracle):\n%s\n%s", f, u)
	}
	if strings.Contains(forged.Body.String(), defB.String()) || strings.Contains(forged.Body.String(), x.b.id.String()) {
		t.Errorf("the 404 names B's definition or tenant: %s", forged.Body)
	}

	if rows := x.exhRows(t, x.a.id); len(rows) != 0 {
		t.Errorf("tenant A has %d number_space_exhausted entries after the forged create, want 0", len(rows))
	}
	if rows := x.exhRows(t, x.b.id); len(rows) != beforeB {
		t.Errorf("tenant B has %d number_space_exhausted entries after A's forged create, want %d", len(rows), beforeB)
	}
	var bRecords int
	if err := x.super.QueryRow(t.Context(), "SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2", x.b.id, defB).Scan(&bRecords); err != nil {
		t.Fatal(err)
	}
	if bRecords != 9 {
		t.Errorf("tenant B's type has %d records, want 9", bRecords)
	}
	var counter int64
	if err := x.super.QueryRow(t.Context(), "SELECT next FROM record_counters WHERE tenant_id = $1 AND entity_definition_id = $2", x.b.id, defB).Scan(&counter); err != nil {
		t.Fatal(err)
	}
	if counter != 11 {
		t.Errorf("tenant B's counter next = %d, want 11 (9 fills and B's own refusal only)", counter)
	}
}

// CI-A18 (h): an audit.read principal whose masks on the type hide record
// fields reads A's entry: its diff is one add of /number_space with exactly
// epoch, scope_key, width and counter, and no record payload path. The
// refusal log line (the test log handler) has no scope_key.
//
// covers: listAuditEntries 200
func TestNumberSpaceExhaustedAdversarialMaskedReaderSeesOnlyTheFourKeys(t *testing.T) {
	for _, mode := range exhModes {
		t.Run(mode, func(t *testing.T) {
			x := newExhEnv(t, mode)
			// Every record field of exh readable and writable by user-a (the
			// creator) and the lookalike group only:
			// admin-a (audit.read) is masked out of all of them.
			only := authz.Subject{Groups: []uuid.UUID{x.a.lookalikeGroup}, Users: []uuid.UUID{x.a.user.PrincipalID}}
			masks := []authz.Mask{
				{Field: "short_description", Read: only, Write: only},
				{Field: "note", Read: only, Write: only},
			}
			rules := []authz.Rule{acltest.AllowUsers(x.a.admin.PrincipalID, x.a.user.PrincipalID, x.a.lookalike.PrincipalID)}
			if err := tenancy.InTenantTx(t.Context(), x.pool, x.a.id, func(tx pgx.Tx) error {
				_, err := acltest.PublishPolicy(t.Context(), tx, exhKey, rules, masks)
				return err
			}); err != nil {
				t.Fatal(err)
			}
			// Precondition: the masks bite admin-a. Its timeline redacts the
			// fill creates' record diffs.
			_, page := x.auditGet(t, "admin-a", "?domain=record&subject_type=record")
			redacted := 0
			for _, it := range page.Items {
				if it.DiffRedacted && strings.Contains(string(it.Diff), `"redacted":"mask"`) {
					redacted++
				}
			}
			if redacted == 0 {
				t.Fatalf("no record entry is masked for admin-a; the reader is not a masked principal: %+v", page.Items)
			}
			logFrom := len(x.logs.String())
			x.refuse(t, linkKey("exh-adv-h"))

			diffs := x.timelineExhausted(t, "admin-a", "?subject_type=entity_definition&subject_id="+x.defID.String())
			if len(diffs) != 1 {
				t.Fatalf("masked reader sees %d number_space_exhausted entries, want 1", len(diffs))
			}
			var ops []struct {
				Op    string                     `json:"op"`
				Path  string                     `json:"path"`
				Value map[string]json.RawMessage `json:"value"`
			}
			if err := json.Unmarshal(diffs[0], &ops); err != nil {
				t.Fatalf("diff %s: %v", diffs[0], err)
			}
			if len(ops) != 1 || ops[0].Op != "add" || ops[0].Path != "/number_space" {
				t.Fatalf("diff %s, want exactly one add of /number_space", diffs[0])
			}
			keys := make([]string, 0, len(ops[0].Value))
			for k := range ops[0].Value {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			if !slices.Equal(keys, []string{"counter", "epoch", "scope_key", "width"}) {
				t.Errorf("diff keys %v, want exactly counter, epoch, scope_key, width", keys)
			}
			body := string(diffs[0])
			for _, leak := range []string{"/data", "short_description", "refused", "note", "state"} {
				if strings.Contains(body, leak) {
					t.Errorf("diff %s carries record payload %q", body, leak)
				}
			}

			lines := 0
			for _, line := range strings.Split(x.logs.String()[logFrom:], "\n") {
				if !strings.Contains(line, "number space exhausted") {
					continue
				}
				lines++
				if strings.Contains(line, "scope") || strings.Contains(line, "=e1 ") || strings.HasSuffix(line, "=e1") || strings.Contains(line, `"e1"`) {
					t.Errorf("refusal log line names the scope key: %s", line)
				}
			}
			if lines != 1 {
				t.Errorf("%d refusal log lines, want 1:\n%s", lines, x.logs.String()[logFrom:])
			}
		})
	}
}
