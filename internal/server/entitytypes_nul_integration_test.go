//go:build integration

// TICK-1121: a \u0000 escape in a schema is refused with a 422 that points at
// the string, on create and on publish, through the real handler against a
// real PostgreSQL. jsonb refuses the escape (22P05), and the audit entry
// reads the stored schema as jsonb, so it used to be a 500.

package server_test

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

const etNote = `"note": {"type": "string", "description": "a <b>&amp; tag"}`

type etNULCase struct{ name, note, want string }

// etNULCases put a NUL escape in a description, a property name and an enum
// value of etSchema; want is the pointer the 422 must carry.
func etNULCases() []etNULCase {
	return []etNULCase{
		{"description", `"note": {"type": "string", "description": "x\u0000y"}`, "/definition/schema/properties/note/description"},
		{"property name", `"no\u0000te": {"type": "string"}`, "/definition/schema/properties/no\x00te"},
		{"enum value", `"note": {"type": "string", "enum": ["a", "b\u0000"]}`, "/definition/schema/properties/note/enum/1"},
	}
}

func etDefinitionWith(note string) string {
	return fmt.Sprintf(`{"schema": %s, %s}`, strings.Replace(etSchema, etNote, note, 1), etDefTail)
}

func etCreateWith(key, prefix, note string) string {
	return fmt.Sprintf(`{"key": %q, "name": "Type %s", "number_prefix": %q, "definition": %s}`, key, key, prefix, etDefinitionWith(note))
}

// etWrites counts tenant A's version, audit and outbox rows.
func (e *etEnv) etWrites(t *testing.T) [3]int {
	t.Helper()
	var n [3]int
	for i, table := range []string{"entity_definition_versions", "audit_log", "outbox"} {
		if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE tenant_id = $1", e.a.id).Scan(&n[i]); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
	}
	return n
}

func wantPointer(t *testing.T, p etProblem, want string) {
	t.Helper()
	if got := pointers(p); !slices.Equal(got, []string{want}) {
		t.Errorf("pointers = %q, want [%q]", got, want)
	}
}

// covers: createEntityType 422
// covers: publishEntityTypeVersion 422
func TestNULInSchemaIsRefusedWithItsPointer(t *testing.T) {
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-base", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("create the base type = %d: %s", rec.Code, rec.Body)
	}
	for i, c := range etNULCases() {
		t.Run(c.name, func(t *testing.T) {
			before := e.etWrites(t)
			rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", fmt.Sprintf("k-create-%d", i), etCreateWith(fmt.Sprintf("nul_%d", i), "NUL", c.note))
			wantPointer(t, wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidDefinition), c.want)
			rec = e.do(t, http.MethodPost, "/api/v1/entity-types/incident/versions", "admin-a", fmt.Sprintf("k-publish-%d", i), `{"definition": `+etDefinitionWith(c.note)+`}`)
			wantPointer(t, wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidDefinition), c.want)
			if after := e.etWrites(t); after != before {
				t.Errorf("versions, audit, outbox rows = %v, were %v: a refused write must leave nothing", after, before)
			}
		})
	}
}

// covers: createEntityType 201
// covers: publishEntityTypeVersion 201
func TestEscapedBackslashAndSurrogatePairAreAccepted(t *testing.T) {
	e := newETEnv(t)
	for i, note := range []string{
		`"note": {"type": "string", "description": "x\\u0000y"}`,
		`"note": {"type": "string", "description": "😀"}`,
	} {
		key := fmt.Sprintf("ok_%d", i)
		if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-c-"+key, etCreateWith(key, "OK"+string(rune('A'+i)), note)); rec.Code != http.StatusCreated {
			t.Fatalf("create %s = %d: %s", note, rec.Code, rec.Body)
		}
	}
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-base", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("create the base type = %d: %s", rec.Code, rec.Body)
	}
	for i, note := range []string{
		`"note": {"type": "string", "description": "x\\u0000y"}`,
		`"note": {"type": "string", "description": "😀"}`,
	} {
		if rec := e.do(t, http.MethodPost, "/api/v1/entity-types/incident/versions", "admin-a", fmt.Sprintf("k-p-%d", i), `{"definition": `+etDefinitionWith(note)+`}`); rec.Code != http.StatusCreated {
			t.Fatalf("publish %s = %d: %s", note, rec.Code, rec.Body)
		}
	}
}

// A version stored before publish refused a NUL still reads 200.
//
// covers: getEntityType 200
// covers: getEntityTypeVersion 200
func TestStoredVersionWithNULStillReads(t *testing.T) {
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-base", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	stored := strings.Replace(etSchema, etNote, etNULCases()[0].note, 1)
	sum := sha256.Sum256([]byte(stored))
	// A row an earlier release stored: written as the superuser, triggers off.
	for _, q := range []string{
		"SET session_replication_role = replica",
		`UPDATE entity_definition_versions SET schema = $2::text::json, schema_sha256 = $3 WHERE tenant_id = $1 AND version = 1`,
		"RESET session_replication_role",
	} {
		var args []any
		if strings.HasPrefix(q, "UPDATE") {
			args = []any{e.a.id, stored, sum[:]}
		}
		if _, err := e.super.Exec(t.Context(), q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	for _, path := range []string{"/api/v1/entity-types/incident", "/api/v1/entity-types/incident/versions/1"} {
		rec := e.do(t, http.MethodGet, path, "user-a", "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d: %s", path, rec.Code, rec.Body)
		}
		if !strings.Contains(rec.Body.String(), `x\u0000y`) {
			t.Errorf("GET %s does not return the stored NUL escape: %s", path, rec.Body)
		}
	}
}
