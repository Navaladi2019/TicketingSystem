//go:build integration

// P5-17 (TICK-435): GET /api/v1/audit/entries against a real PostgreSQL,
// through the real handler (ADR-004 R7, R8).

package server_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

type auditPage struct {
	Items []struct {
		ID           string          `json:"id"`
		Action       string          `json:"action"`
		Domain       string          `json:"domain"`
		SubjectID    string          `json:"subject_id"`
		Diff         json.RawMessage `json:"diff"`
		DiffRedacted bool            `json:"diff_redacted"`
		DiffSalt     string          `json:"diff_salt"`
		DiffSHA256   string          `json:"diff_sha256"`
	} `json:"items"`
	Next string `json:"next_page_token"`
}

func (e *etEnv) auditGet(t *testing.T, token, query string) (*httptest.ResponseRecorder, auditPage) {
	t.Helper()
	rec := e.do(t, http.MethodGet, "/api/v1/audit/entries"+query, token, "", "")
	var p auditPage
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
	}
	return rec, p
}

// Filters, stable non-overlapping pages, CI-A14 on the read (an unredacted
// entry carries the salt and its diff verifies), and the gate.
//
// covers: listAuditEntries 200 400 403
func TestListAuditEntriesPagesFiltersAndGate(t *testing.T) {
	e := newPatchEnv(t)
	r := e.mustCreateRec(t, "user-a", "job", "al-1", `{"short_description": "one"}`)
	for i, note := range []string{"a", "b"} {
		if rec := e.patch(t, "user-a", "job", r.ID, fmt.Sprintf(`W/"%d"`, i+1), fmt.Sprintf(`{"data": {"note": %q}}`, note)); rec.Code != http.StatusOK {
			t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
		}
	}
	q := "?subject_id=" + r.ID

	// Pages of one walk the three entries newest first, with no overlap.
	var seen, actions []string
	query := q + "&limit=1"
	for range 5 {
		rec, p := e.auditGet(t, "admin-a", query)
		if rec.Code != http.StatusOK || len(p.Items) != 1 {
			t.Fatalf("page = %d: %s", rec.Code, rec.Body)
		}
		it := p.Items[0]
		if slices.Contains(seen, it.ID) {
			t.Fatalf("entry %s on two pages", it.ID)
		}
		seen = append(seen, it.ID)
		actions = append(actions, it.Action)
		// CI-A14: nothing is masked here, so the salt is there and verifies.
		if it.DiffRedacted || it.DiffSalt == "" {
			t.Errorf("unredacted entry %s: redacted=%v salt=%q", it.ID, it.DiffRedacted, it.DiffSalt)
		}
		salt, _ := hex.DecodeString(it.DiffSalt)
		var s [16]byte
		copy(s[:], salt)
		if got := chain.DiffSHA256(s, it.Diff); hex.EncodeToString(got[:]) != it.DiffSHA256 {
			t.Errorf("entry %s: SHA-256(salt || diff) does not match diff_sha256", it.ID)
		}
		if p.Next == "" {
			break
		}
		query = q + "&limit=1&page_token=" + url.QueryEscape(p.Next)
	}
	if want := []string{"update", "update", "create"}; !slices.Equal(actions, want) {
		t.Errorf("walk = %v, want %v", actions, want)
	}

	// Filters: the domain and the subject type narrow; a time window past
	// everything is empty.
	if _, p := e.auditGet(t, "admin-a", "?domain=config&limit=200"); len(p.Items) == 0 {
		t.Error("control: the seeded entity types have config entries")
	} else {
		for _, it := range p.Items {
			if it.Domain != "config" {
				t.Errorf("domain filter returned %s", it.Domain)
			}
		}
	}
	if _, p := e.auditGet(t, "admin-a", "?domain=record&subject_type=record&subject_id="+r.ID); len(p.Items) != 3 {
		t.Errorf("record filter = %d entries, want 3", len(p.Items))
	}
	if rec, p := e.auditGet(t, "admin-a", "?from="+url.QueryEscape("2999-01-01T00:00:00Z")); rec.Code != http.StatusOK || len(p.Items) != 0 || p.Next != "" {
		t.Errorf("a window in the future = %d, %d entries: want an empty page", rec.Code, len(p.Items))
	}
	// after_seq keeps sealed entries only; none is sealed here.
	if _, p := e.auditGet(t, "admin-a", q+"&after_seq=0"); len(p.Items) != 0 {
		t.Errorf("after_seq=0 returned %d unsealed entries", len(p.Items))
	}

	// The gate: a member without audit.read, and no token.
	if rec, _ := e.auditGet(t, "user-a", ""); rec.Code != http.StatusForbidden || patchProblemCode(t, rec) != "forbidden" {
		t.Errorf("member without audit.read = %d: %s", rec.Code, rec.Body)
	}
	// Even a bad token is a 403 for them: the gate comes first.
	if rec, _ := e.auditGet(t, "user-a", "?page_token=bad"); rec.Code != http.StatusForbidden {
		t.Errorf("member with a bad page_token = %d, want 403", rec.Code)
	}
	if rec, _ := e.auditGet(t, "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", rec.Code)
	}
	// Validation: limit above 500, a bad token, a bad filter.
	if rec, _ := e.auditGet(t, "admin-a", "?limit=501"); rec.Code != http.StatusBadRequest {
		t.Errorf("limit=501 = %d, want 400", rec.Code)
	}
	if rec, _ := e.auditGet(t, "admin-a", "?page_token=not-a-token"); rec.Code != http.StatusBadRequest || patchProblemCode(t, rec) != "invalid_cursor" {
		t.Errorf("bad page_token = %d: %s", rec.Code, rec.Body)
	}
	if rec, _ := e.auditGet(t, "admin-a", "?domain=nope"); rec.Code != http.StatusBadRequest {
		t.Errorf("bad domain = %d, want 400", rec.Code)
	}
}

// A page token of one filter is refused on another, and tenant B's admin
// reads none of tenant A's entries.
func TestListAuditEntriesTokenBindingAndTenant(t *testing.T) {
	e := newPatchEnv(t)
	r := e.mustCreateRec(t, "user-a", "job", "al-2", `{"short_description": "one"}`)
	if rec := e.patch(t, "user-a", "job", r.ID, `W/"1"`, `{"data": {"note": "a"}}`); rec.Code != http.StatusOK {
		t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
	}
	_, p := e.auditGet(t, "admin-a", "?subject_id="+r.ID+"&limit=1")
	if p.Next == "" {
		t.Fatal("control: two entries make a second page")
	}
	// The same token on another filter.
	if rec, _ := e.auditGet(t, "admin-a", "?domain=config&limit=1&page_token="+url.QueryEscape(p.Next)); rec.Code != http.StatusBadRequest {
		t.Errorf("token on another filter = %d, want 400", rec.Code)
	}
	// Tenant B's admin, the same token and the same filter.
	if rec, _ := e.auditGet(t, "admin-b", "?subject_id="+r.ID+"&limit=1&page_token="+url.QueryEscape(p.Next)); rec.Code != http.StatusBadRequest {
		t.Errorf("tenant A's token for tenant B = %d, want 400", rec.Code)
	}
	if rec, pb := e.auditGet(t, "admin-b", "?subject_id="+r.ID); rec.Code != http.StatusOK || len(pb.Items) != 0 {
		t.Errorf("tenant B reads tenant A's entries: %d, %d items", rec.Code, len(pb.Items))
	}
}

// seedSecretEntry creates a vault record in tenant A, whose definition marks
// /token secret, and seeds an update entry on it whose diff holds the secret
// hunter2 at /token. It returns the record id.
func (e *etEnv) seedSecretEntry(t *testing.T) string {
	t.Helper()
	const body = `{"key": "vault", "name": "Vault", "number_prefix": "VLT", "definition": {"schema": {
  "$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "additionalProperties": false,
  "required": ["state"], "properties": {"state": {"type": "string", "enum": ["new"], "default": "new"},
  "token": {"type": "string", "x-nava-secret": true}}},
  "fixed_columns": {"state": "state"}, "states": {"initial": "new", "states": ["new"], "transitions": []}}}`
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "mk-a", body); rec.Code != http.StatusCreated {
		t.Fatalf("seed type = %d: %s", rec.Code, rec.Body)
	}
	r := e.mustCreateRec(t, "user-a", "vault", "mk-1", `{}`)
	var def string
	if err := e.super.QueryRow(t.Context(), `SELECT entity_definition_id::text FROM records WHERE id = $1`, r.ID).Scan(&def); err != nil {
		t.Fatal(err)
	}
	const diff = `[{"op":"add","path":"/token","value":"hunter2"},{"op":"add","path":"/n","value":1}]`
	if _, err := e.super.Exec(t.Context(), `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
		entity_definition_id, version_before, version_after, actor_system, source, diff, diff_salt)
		VALUES ($1, gen_random_uuid(), 'record', 'update', 'record', $2, $3, 1, 2, 'workflow', 'workflow', $4::json, '\x00112233445566778899aabbccddeeff')`,
		e.a.id, r.ID, def, diff); err != nil {
		t.Fatal(err)
	}
	return r.ID
}

// R7 end to end: an entry whose diff holds a value at a pointer the
// definition marks secret is read with {"redacted":"mask"} in its place, no
// salt, and diff_redacted true. The write path refuses such a diff (R6), so
// the row is seeded as the superuser, as an entry written under an older
// schema would be.
func TestListAuditEntriesMasksSecretProperties(t *testing.T) {
	e := newETEnv(t)
	id := e.seedSecretEntry(t)
	rec, p := e.auditGet(t, "admin-a", "?subject_id="+id+"&limit=1")
	if rec.Code != http.StatusOK || len(p.Items) != 1 {
		t.Fatalf("read = %d: %s", rec.Code, rec.Body)
	}
	it := p.Items[0]
	const want = `[{"op":"add","path":"/token","value":{"redacted":"mask"}},{"op":"add","path":"/n","value":1}]`
	if string(it.Diff) != want || !it.DiffRedacted || it.DiffSalt != "" {
		t.Errorf("masked entry = diff %s redacted=%v salt=%q; want %s, true, no salt", it.Diff, it.DiffRedacted, it.DiffSalt, want)
	}
	if rec.Body.Len() > 0 && json.Valid(rec.Body.Bytes()) && strings.Contains(rec.Body.String(), "hunter2") {
		t.Errorf("the secret is in the response: %s", rec.Body)
	}
}

// R7 on the history endpoint (TICK-541; PSE review of PR #438): the history
// replays the same diffs, so the secret is {"redacted":"mask"} in after and
// absent from the whole body. Before TICK-541 only the list endpoint's
// masking was tested.
func TestRecordHistoryMasksSecretProperties(t *testing.T) {
	e := newETEnv(t)
	id := e.seedSecretEntry(t)
	rec := e.do(t, http.MethodGet, "/api/v1/records/vault/"+id+"/history", "user-a", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("history = %d: %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "hunter2") {
		t.Errorf("the secret is in the history response: %s", rec.Body)
	}
	var p struct {
		Items []struct {
			Action string                     `json:"action"`
			After  map[string]json.RawMessage `json:"after"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	// Newest first: the seeded update, then the create.
	if len(p.Items) != 2 || p.Items[0].Action != "update" {
		t.Fatalf("history = %s, want the seeded update and the create", rec.Body)
	}
	if got := string(p.Items[0].After["token"]); got != `{"redacted":"mask"}` {
		t.Errorf("after.token = %s, want {\"redacted\":\"mask\"}; body %s", got, rec.Body)
	}
	if got := string(p.Items[0].After["n"]); got != "1" {
		t.Errorf("after.n = %s, want 1 (the unmasked op still replays); body %s", got, rec.Body)
	}
}

// failingMasker is a masker whose definition has no visible version.
type failingMasker struct{ def uuid.UUID }

func (f failingMasker) MaskedPaths(context.Context, pgx.Tx, read.Subject) ([]patchwalk.Pointer, error) {
	return nil, fmt.Errorf("masker: %w", &read.NoDefinitionVersionsError{DefinitionID: f.def})
}

// ADR-004-A1 A1-4: when the masker cannot tell which properties are secret,
// both audit reads answer 500 internal with no diff and no salt, and the log
// names the definition and nothing from the entry.
func TestAuditReadsFailClosedWhenMaskerHasNoVersions(t *testing.T) {
	def := uuid.New()
	e := newETEnvWith(t, server.WithAuditMasker(failingMasker{def}))
	id := e.seedSecretEntry(t)
	for _, tc := range []struct{ name, token, path string }{
		{"list", "admin-a", "/api/v1/audit/entries?subject_id=" + id},
		{"history", "user-a", "/api/v1/records/vault/" + id + "/history"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := e.do(t, http.MethodGet, tc.path, tc.token, "", "")
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500: %s", rec.Code, rec.Body)
			}
			body := rec.Body.String()
			for _, bad := range []string{`"diff"`, "diff_salt", "hunter2"} {
				if strings.Contains(body, bad) {
					t.Errorf("body contains %s: %s", bad, body)
				}
			}
			if !strings.Contains(body, "internal_error") {
				t.Errorf("body is not the internal problem: %s", body)
			}
		})
	}
	logs := e.logs.String()
	if !strings.Contains(logs, def.String()) {
		t.Errorf("log does not name definition %s: %s", def, logs)
	}
	for _, bad := range []string{"hunter2", "00112233445566778899aabbccddeeff", "/token"} {
		if strings.Contains(logs, bad) {
			t.Errorf("log contains %q: %s", bad, logs)
		}
	}
}
