//go:build integration

// P3-05b (TICK-365): publishing entity-type versions against a real
// PostgreSQL, through the real handler. Helpers are in
// entitytypes_integration_test.go.

package server_test

import (
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
