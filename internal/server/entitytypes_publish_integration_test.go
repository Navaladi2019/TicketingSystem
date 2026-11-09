//go:build integration

// P3-05b (TICK-365): publishing entity-type versions against a real
// PostgreSQL, through the real handler. Helpers are in
// entitytypes_integration_test.go.

package server_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const etVersionsPath = "/api/v1/entity-types/incident/versions"

// etPublishBody is a publish body whose schema is etSchema with the given
// old/new replacement pairs applied, so each case changes one thing.
func etPublishBody(t *testing.T, replacements ...string) string {
	t.Helper()
	schema := etSchema
	for i := 0; i < len(replacements); i += 2 {
		if !strings.Contains(schema, replacements[i]) {
			t.Fatalf("etSchema has no %q", replacements[i])
		}
		schema = strings.Replace(schema, replacements[i], replacements[i+1], 1)
	}
	return fmt.Sprintf(`{"definition": {"schema": %s, %s}}`, schema, etDefTail)
}

func etCreateIncident(t *testing.T, e *etEnv) {
	t.Helper()
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-create", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
}

func etVersionCount(t *testing.T, e *etEnv) int {
	t.Helper()
	var n int
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM entity_definition_versions WHERE tenant_id = $1", e.a.id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// covers: publishEntityTypeVersion 201
// covers: getEntityType 200
// covers: getEntityTypeVersion 200
func TestPublishGivesNextVersionAndLeavesV1Untouched(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	v1 := e.do(t, http.MethodGet, etVersionsPath+"/1", "user-a", "", "").Body.String()

	body := etPublishBody(t, `"description": "a <b>&amp; tag"`, `"description": "changed"`)
	rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-pub-1", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish = %d: %s", rec.Code, rec.Body)
	}
	var v struct {
		Version int    `json:"version"`
		Current bool   `json:"current"`
		Compat  string `json:"compat"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v.Version != 2 || !v.Current || v.Compat != "compatible" {
		t.Errorf("published = %+v, want v2 current compatible", v)
	}
	if got := e.do(t, http.MethodGet, "/api/v1/entity-types/incident", "user-a", "", "").Body.String(); got != rec.Body.String() {
		t.Errorf("current differs from the publish response:\n%s\n%s", got, rec.Body)
	}
	// v1 reads byte for byte as before, except that it is no longer current.
	after := e.do(t, http.MethodGet, etVersionsPath+"/1", "user-a", "", "").Body.String()
	if want := strings.Replace(v1, `"current":true`, `"current":false`, 1); after != want {
		t.Errorf("v1 changed:\n%s\n%s", after, want)
	}
	sum := sha256.Sum256([]byte(etSchema))
	if !strings.Contains(after, `"schema_sha256":"`+hex.EncodeToString(sum[:])+`"`) || !strings.Contains(after, `"schema":`+etSchema+`,`) {
		t.Errorf("v1 schema bytes or hash changed: %s", after)
	}
	// published_by is the authenticated principal.
	var by uuid.UUID
	if err := e.super.QueryRow(t.Context(), "SELECT published_by FROM entity_definition_versions WHERE tenant_id = $1 AND version = 2", e.a.id).Scan(&by); err != nil || by != e.a.admin.PrincipalID {
		t.Errorf("published_by = %v (err %v), want %v", by, err, e.a.admin.PrincipalID)
	}
}

// covers: publishEntityTypeVersion 201
func TestPublishFiftyConcurrentGivesVersions2To51(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	const n = 50
	// A root title is an annotation, so every publish is Compatible with
	// whichever version is current when it gets the lock.
	bodies := make([]string, n)
	for i := range n {
		bodies[i] = etPublishBody(t, `"type": "object",`, fmt.Sprintf(`"title": "rev %d", "type": "object",`, i))
	}
	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = e.do(t, http.MethodPost, etVersionsPath, "admin-a", fmt.Sprintf("k-%d", i), bodies[i]).Code
		}()
	}
	wg.Wait()
	for i, c := range codes {
		if c != http.StatusCreated {
			t.Errorf("publish %d = %d, want 201", i, c)
		}
	}
	rows, err := e.super.Query(t.Context(), "SELECT version FROM entity_definition_versions WHERE tenant_id = $1 ORDER BY version", e.a.id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		got = append(got, v)
	}
	if len(got) != n+1 {
		t.Fatalf("%d versions, want %d", len(got), n+1)
	}
	for i, v := range got {
		if v != i+1 {
			t.Fatalf("versions = %v, want 1..%d with no gap or duplicate", got, n+1)
		}
	}
	var cur int
	if err := e.super.QueryRow(t.Context(), "SELECT current_version FROM entity_definitions WHERE tenant_id = $1", e.a.id).Scan(&cur); err != nil || cur != n+1 {
		t.Errorf("current_version = %d (err %v), want %d", cur, err, n+1)
	}
}

// covers: publishEntityTypeVersion 201 422
func TestPublishClassifiesAgainstTheCurrentVersion(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	breaking := map[string]struct{ old, new, path string }{
		"changed-default": {`"default": "new"`, `"default": "closed"`, "/definition/schema/properties/state/default"},
		"added-unique":    {`"minLength": 1,`, `"minLength": 1, "x-nava-unique": true,`, "/definition/schema/properties/short_description/x-nava-unique"},
	}
	for name, c := range breaking {
		rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-"+name, etPublishBody(t, c.old, c.new))
		p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.SchemaBreakingChange)
		if got := pointers(p); !slices.Contains(got, c.path) {
			t.Errorf("%s: pointers = %v, want %s among them", name, got, c.path)
		}
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Fatalf("%d versions after refused publishes, want 1", n)
	}
	// Adding a default to an existing property needs a backfill and publishes.
	rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-backfill", etPublishBody(t, `"maxLength": 160`, `"maxLength": 160, "default": "none"`))
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"compat":"needs_backfill"`) {
		t.Fatalf("backfill publish = %d: %s", rec.Code, rec.Body)
	}
	// A definition that does not compile is 422 invalid_definition.
	wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-bad", `{"definition": {"schema": {"type": "array"}, "fixed_columns": {}, "states": {}}}`),
		http.StatusUnprocessableEntity, problem.InvalidDefinition)
	if n := etVersionCount(t, e); n != 2 {
		t.Errorf("%d versions, want 2", n)
	}
}

// covers: publishEntityTypeVersion 403 404
func TestPublishRequiresTheTenantAdminGroup(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	for _, token := range []string{"user-a", "lookalike-a"} {
		wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, token, "k-"+token, etPublishBody(t)), http.StatusForbidden, problem.Forbidden)
		// The check precedes the body.
		wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, token, "kb-"+token, `{"definition": 5}`), http.StatusForbidden, problem.Forbidden)
	}
	// Tenant B's admin cannot see tenant A's type: 404.
	wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "admin-b", "k-b", etPublishBody(t)), http.StatusNotFound, problem.NotFound)
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("%d versions, want 1", n)
	}
}

// covers: publishEntityTypeVersion 400 401 422
func TestPublishRejectsNumberPrefixAndBadBodies(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	body := fmt.Sprintf(`{"number_prefix": "ZZZ", "definition": {"schema": %s, %s}}`, etSchema, etDefTail)
	p := wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k1", body), http.StatusUnprocessableEntity, problem.InvalidDefinition)
	if got := pointers(p); len(got) != 1 || got[0] != "/number_prefix" {
		t.Errorf("pointers = %v, want [/number_prefix]", got)
	}
	wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k2", `{}`), http.StatusUnprocessableEntity, problem.InvalidDefinition)
	wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k3", `{"definition": 5}`), http.StatusUnprocessableEntity, problem.InvalidDefinition)
	wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "admin-a", "", etPublishBody(t)), http.StatusBadRequest, problem.ValidationFailed)
	wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "", "k4", etPublishBody(t)), http.StatusUnauthorized, problem.InvalidToken)
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("%d versions, want 1", n)
	}
}

// covers: publishEntityTypeVersion 404
func TestPublishUnknownAndForeignKeysGiveTheSameNotFound(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	unknown := e.do(t, http.MethodPost, "/api/v1/entity-types/no_such_type/versions", "admin-a", "k1", etPublishBody(t))
	foreign := e.do(t, http.MethodPost, etVersionsPath, "admin-b", "k2", etPublishBody(t))
	long := e.do(t, http.MethodPost, "/api/v1/entity-types/"+strings.Repeat("k", 200)+"/versions", "admin-a", "k3", etPublishBody(t))
	for _, rec := range []*struct {
		code int
		body string
	}{{unknown.Code, unknown.Body.String()}, {foreign.Code, foreign.Body.String()}, {long.Code, long.Body.String()}} {
		if rec.code != http.StatusNotFound || rec.body != unknown.Body.String() || strings.Contains(rec.body, "instance") {
			t.Errorf("got %d %s, want the one 404 without instance", rec.code, rec.body)
		}
	}
}

// covers: publishEntityTypeVersion 201 409
func TestPublishIdempotentReplay(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	body := etPublishBody(t, `"description": "a <b>&amp; tag"`, `"description": "changed"`)
	first := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "same", body)
	if first.Code != http.StatusCreated {
		t.Fatalf("publish = %d: %s", first.Code, first.Body)
	}
	again := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "same", body)
	if again.Code != http.StatusCreated || again.Body.String() != first.Body.String() {
		t.Errorf("replay = %d, want the same 201 body", again.Code)
	}
	wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "admin-a", "same", etPublishBody(t)), http.StatusConflict, problem.IdempotencyKeyReuse)
	if n := etVersionCount(t, e); n != 2 {
		t.Errorf("%d versions, want 2", n)
	}
}

func TestPublishRefusesNoPrincipalAndAddsNoRow(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	ctx := t.Context()
	err := tenancy.InTenantTx(ctx, e.pool, e.a.id, func(tx pgx.Tx) error {
		_, err := entitytypes.Store{}.Publish(ctx, tx, e.a.id, entitytypes.PublishInput{Key: "incident", Definition: etDefinition(t, "incident", "INC")})
		return err
	})
	if !errors.Is(err, entitytypes.ErrNoPublisher) {
		t.Fatalf("Publish with no principal = %v, want ErrNoPublisher", err)
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("%d versions, want 1", n)
	}
}

// An impossible key gets the byte-identical 404 whatever the body is: the key
// shape is checked before the body is parsed (the admin check stays first).
// covers: publishEntityTypeVersion 404
func TestPublishOverlongKeyWithMalformedBodyIsNotFound(t *testing.T) {
	e := newETEnv(t)
	long := "/api/v1/entity-types/" + strings.Repeat("k", 65) + "/versions"
	good := e.do(t, http.MethodPost, long, "admin-a", "k-good", etPublishBody(t))
	bad := e.do(t, http.MethodPost, long, "admin-a", "k-bad", `{"number_prefix": 1}`)
	if good.Code != http.StatusNotFound || bad.Code != http.StatusNotFound {
		t.Fatalf("over-long key: good body %d, malformed body %d, want 404 both: %s / %s", good.Code, bad.Code, good.Body, bad.Body)
	}
	if good.Body.String() != bad.Body.String() {
		t.Errorf("404 bodies differ: %s vs %s", good.Body, bad.Body)
	}
}

// A stored current version that no longer compiles is a store defect: publish
// answers 500, not a 422 with pointers into the stored schema.
// covers: publishEntityTypeVersion 500
func TestPublishOverABrokenStoredVersionIsInternalError(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	broken := `{"type":"array"}`
	sum := sha256.Sum256([]byte(broken))
	if _, err := e.super.Exec(t.Context(), "UPDATE entity_definition_versions SET schema = $1::json, schema_sha256 = $2 WHERE tenant_id = $3", broken, sum[:], e.a.id); err != nil {
		t.Fatal(err)
	}
	rec := e.do(t, http.MethodPost, "/api/v1/entity-types/incident/versions", "admin-a", "k-broken", etPublishBody(t))
	wantProblem(t, rec, http.StatusInternalServerError, problem.InternalError)
}

// P5-14 (TICK-432): publishing writes one config-domain audit entry.

type etAuditRow struct {
	Action        string
	VersionBefore *int64
	VersionAfter  int64
	ContentSHA    []byte
	Source        string
	Actor         uuid.UUID
	RequestID     *string
}

func etAuditRows(t *testing.T, e *etEnv) []etAuditRow {
	t.Helper()
	rows, err := e.super.Query(t.Context(), `
SELECT action, version_before, version_after, content_sha256, source, actor_principal_id, request_id
FROM audit_log WHERE tenant_id = $1 AND domain = 'config' AND subject_type = 'entity_definition'
  AND source <> 'system' ORDER BY version_after`, e.a.id) // not the system entries of the test ACLs (acltest)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []etAuditRow
	for rows.Next() {
		var r etAuditRow
		if err := rows.Scan(&r.Action, &r.VersionBefore, &r.VersionAfter, &r.ContentSHA, &r.Source, &r.Actor, &r.RequestID); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPublishWritesCreateThenPublishEntries(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	body := etPublishBody(t, `"description": "a <b>&amp; tag"`, `"description": "changed"`)
	if rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-aud-1", body); rec.Code != http.StatusCreated {
		t.Fatalf("publish = %d: %s", rec.Code, rec.Body)
	}
	rows := etAuditRows(t, e)
	if len(rows) != 2 {
		t.Fatalf("%d entries, want 2: %+v", len(rows), rows)
	}
	for i, want := range []string{"create", "publish"} {
		var sum []byte
		if err := e.super.QueryRow(t.Context(),
			"SELECT schema_sha256 FROM entity_definition_versions WHERE tenant_id = $1 AND version = $2", e.a.id, i+1).Scan(&sum); err != nil {
			t.Fatal(err)
		}
		r := rows[i]
		if r.Action != want || r.VersionAfter != int64(i+1) || r.Source != "user" || r.Actor != e.a.admin.PrincipalID || hex.EncodeToString(r.ContentSHA) != hex.EncodeToString(sum) {
			t.Errorf("entry %d = %+v, want %s v%d with the version's schema_sha256", i, r, want, i+1)
		}
	}
	for i, r := range rows {
		if r.RequestID == nil || *r.RequestID == "" {
			t.Errorf("entry %d has no request_id", i)
		}
	}
	// S3: the stored diff is built from the jsonb read back, not the request.
	var v1, v2, stored string
	for n, dst := range map[int]*string{1: &v1, 2: &v2} {
		if err := e.super.QueryRow(t.Context(),
			"SELECT schema::jsonb::text FROM entity_definition_versions WHERE tenant_id = $1 AND version = $2", e.a.id, n).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	want, err := jsondiff.Diff([]byte(v1), []byte(v2))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.super.QueryRow(t.Context(),
		"SELECT diff::text FROM audit_log WHERE tenant_id = $1 AND domain = 'config' AND version_after = 2 AND source <> 'system'", e.a.id).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != string(want) {
		t.Errorf("stored diff = %s, want %s", stored, want)
	}
	if rows[0].VersionBefore != nil || rows[1].VersionBefore == nil || *rows[1].VersionBefore != 1 {
		t.Errorf("version_before = %v, %v; want nil, 1", rows[0].VersionBefore, rows[1].VersionBefore)
	}
}

func TestRolledBackPublishWritesNoVersionAndNoEntry(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	errAbort := errors.New("abort after publish")
	ctx := context.Background()
	def := etDefinition(t, "incident", "INC")
	def.Schema = json.RawMessage(strings.Replace(etSchema, `"description": "a <b>&amp; tag"`, `"description": "changed"`, 1))
	err := tenancy.InTenantTx(ctx, e.pool, e.a.id, func(tx pgx.Tx) error {
		v, err := entitytypes.Store{}.Publish(ctx, tx, e.a.id, entitytypes.PublishInput{Key: "incident", PublishedBy: e.a.admin.PrincipalID, Definition: def})
		if err != nil {
			t.Fatalf("Publish: %v", err)
		}
		if v.Version != 2 {
			t.Fatalf("version = %d, want 2", v.Version)
		}
		return errAbort // the version row and the entry were written; now roll back
	})
	if !errors.Is(err, errAbort) {
		t.Fatalf("tx = %v, want the abort", err)
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("%d versions after rollback, want 1", n)
	}
	if rows := etAuditRows(t, e); len(rows) != 1 || rows[0].Action != "create" {
		t.Errorf("entries = %+v, want only the create", rows)
	}
}

// M1: adding the marker to a property that already has a default must be
// refused, though the diff never carries the default.
func TestMarkerAddedToExistingDefaultIsRefused(t *testing.T) {
	const canary = "hunter2-canary-value"
	plain := strings.Replace(etSchema, `"note": {"type": "string", "description": "a <b>&amp; tag"}`,
		`"note": {"type": "string", "default": "`+canary+`"}`, 1)
	marked := strings.Replace(plain, `"default": "`+canary+`"`, `"default": "`+canary+`", "x-nava-secret": true`, 1)
	e := newETEnv(t)
	create := fmt.Sprintf(`{"key": "incident", "name": "T", "number_prefix": "INC", "definition": {"schema": %s, %s}}`, plain, etDefTail)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-m1-0", create); rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	body := fmt.Sprintf(`{"definition": {"schema": %s, %s}}`, marked, etDefTail)
	rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-m1-1", body)
	p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.ConfigSecretInContent)
	if got := pointers(p); !slices.Equal(got, []string{"/definition/schema/properties/note/default"}) {
		t.Errorf("pointers = %v", got)
	}
	if strings.Contains(rec.Body.String(), canary) {
		t.Error("response echoes the secret value")
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("%d versions, want 1", n)
	}
	if rows := etAuditRows(t, e); len(rows) != 1 {
		t.Errorf("%d entries, want 1", len(rows))
	}
}

func TestSecretDefaultIsRefusedAndNothingIsWritten(t *testing.T) {
	const canary = "hunter2-canary-value"
	secretSchema := strings.Replace(etSchema, `"note": {"type": "string", "description": "a <b>&amp; tag"}`,
		`"note": {"type": "string", "x-nava-secret": true, "default": "`+canary+`"}`, 1)
	e := newETEnv(t)

	// Create: no definition, version or entry.
	body := fmt.Sprintf(`{"key": "incident", "name": "T", "number_prefix": "INC", "definition": {"schema": %s, %s}}`, secretSchema, etDefTail)
	rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-sec-0", body)
	p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.ConfigSecretInContent)
	if got := pointers(p); !slices.Equal(got, []string{"/definition/schema/properties/note/default"}) {
		t.Errorf("pointers = %v", got)
	}
	if strings.Contains(rec.Body.String(), canary) {
		t.Error("response echoes the secret value")
	}
	// The create path goes through secretProblem: the warning names the path.
	if got := e.logs.String(); !strings.Contains(got, "entity type refused: secret in content") ||
		!strings.Contains(got, "/definition/schema/properties/note/default") || strings.Contains(got, canary) {
		t.Errorf("log lacks the secret warning with the path, or carries the value: %s", got)
	}
	if n := etVersionCount(t, e); n != 0 {
		t.Errorf("%d versions after a refused create, want 0", n)
	}

	// Publish: v1 stays current, no v2 row, no second entry.
	etCreateIncident(t, e)
	body = fmt.Sprintf(`{"definition": {"schema": %s, %s}}`, secretSchema, etDefTail)
	rec = e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-sec-1", body)
	wantProblem(t, rec, http.StatusUnprocessableEntity, problem.ConfigSecretInContent)
	if strings.Contains(rec.Body.String(), canary) {
		t.Error("response echoes the secret value")
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("%d versions, want 1", n)
	}
	if rows := etAuditRows(t, e); len(rows) != 1 {
		t.Errorf("%d entries, want 1", len(rows))
	}
}
