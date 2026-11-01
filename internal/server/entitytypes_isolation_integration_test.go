//go:build integration

// Adversarial tenant-isolation tests for the entity-type endpoints (TICK-343,
// Security & Multi-Tenancy review). The attacker holds a valid token for
// tenant B and tries to learn about, or receive, tenant A's entity types.
// They reuse newETEnv from entitytypes_integration_test.go.

package server_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// etSchemaB is valid and differs from etSchema, so the two tenants' rows
// can be told apart by their bytes and sha256.
const etSchemaB = `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,"required":["state"],"properties":{"state":{"type":"string","enum":["new","closed"],"default":"new"},"tenant_b_only":{"type":"string"}}}`

func etBodyWithSchema(key, prefix, schema string) string {
	return fmt.Sprintf(`{"key": %q, "name": "Type %s", "number_prefix": %q, "definition": {"schema": %s, %s}}`,
		key, key, prefix, schema, etDefTail)
}

func etSHA(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Tenant B's admin creates the key and prefix tenant A already uses. A 409
// would tell B that A has an entity type named "incident" (a uniqueness
// oracle across tenants); it must be 201, and each tenant reads its own row.
func TestEntityTypeKeyAndPrefixAreNotACrossTenantOracle(t *testing.T) {
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "ka", etBodyWithSchema("incident", "INC", etSchema)); rec.Code != http.StatusCreated {
		t.Fatalf("tenant A create = %d: %s", rec.Code, rec.Body)
	}
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "kb", etBodyWithSchema("incident", "INC", etSchemaB)); rec.Code != http.StatusCreated {
		t.Fatalf("tenant B create with tenant A's key and prefix = %d, want 201 (no cross-tenant 409): %s", rec.Code, rec.Body)
	}

	for _, tc := range []struct{ token, schema, other string }{
		{"user-a", etSchema, etSchemaB},
		{"user-b", etSchemaB, etSchema},
	} {
		for _, path := range []string{"/api/v1/entity-types/incident", "/api/v1/entity-types/incident/versions/1"} {
			rec := e.do(t, http.MethodGet, path, tc.token, "", "")
			if rec.Code != http.StatusOK {
				t.Fatalf("%s GET %s = %d: %s", tc.token, path, rec.Code, rec.Body)
			}
			var got struct {
				Schema json.RawMessage `json:"schema"`
				SHA    string          `json:"schema_sha256"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if string(got.Schema) != tc.schema || got.SHA != etSHA(tc.schema) {
				t.Errorf("%s GET %s returned another tenant's schema (sha %s, want %s)", tc.token, path, got.SHA, etSHA(tc.schema))
			}
			if got.SHA == etSHA(tc.other) {
				t.Errorf("%s GET %s returned the other tenant's version", tc.token, path)
			}
		}
		rec := e.do(t, http.MethodGet, "/api/v1/entity-types", tc.token, "", "")
		var list struct {
			Items []struct{ Key string } `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("%s list = %d %v: %s", tc.token, rec.Code, err, rec.Body)
		}
		if len(list.Items) != 1 {
			t.Errorf("%s list has %d items, want only its own 1", tc.token, len(list.Items))
		}
	}
}

// Tenant B's admin sends the same Idempotency-Key string tenant A's admin
// used, first with A's exact bytes and then with other bytes. A replay of A's
// stored response would hand B A's schema; a 409 idempotency_key_reuse would
// tell B that A used the key. The key is scoped to (tenant, principal), so
// each request must create B's own row.
func TestIdempotencyKeyIsNotSharedAcrossTenants(t *testing.T) {
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "shared-1", etBodyWithSchema("secret_type", "SEC", etSchema)); rec.Code != http.StatusCreated {
		t.Fatalf("tenant A create = %d: %s", rec.Code, rec.Body)
	}
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "shared-2", etBodyWithSchema("other_a", "OTA", etSchema)); rec.Code != http.StatusCreated {
		t.Fatalf("tenant A create = %d: %s", rec.Code, rec.Body)
	}
	// Same key, same bytes as A: a cross-tenant replay would answer 201
	// without writing B's row.
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "shared-1", etBodyWithSchema("secret_type", "SEC", etSchema)); rec.Code != http.StatusCreated {
		t.Fatalf("tenant B, A's key and bytes = %d, want 201: %s", rec.Code, rec.Body)
	}
	// Same key, different bytes: a cross-tenant reuse check would answer 409.
	rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "shared-2", etBodyWithSchema("b_type", "BTY", etSchemaB))
	if rec.Code != http.StatusCreated {
		t.Fatalf("tenant B, A's key and other bytes = %d, want 201: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "tenant_b_only") {
		t.Errorf("tenant B's create did not return its own schema: %s", rec.Body)
	}
	var keys []string
	rows, err := e.super.Query(t.Context(), "SELECT key FROM entity_definitions WHERE tenant_id = $1 ORDER BY key", e.b.id)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			t.Fatal(err)
		}
		keys = append(keys, k)
	}
	rows.Close()
	if strings.Join(keys, ",") != "b_type,secret_type" {
		t.Errorf("tenant B has entity types %v, want [b_type secret_type]: a request was replayed from tenant A", keys)
	}
}

// One principal id that is an admin of tenants A and B (two tokens, one per
// tenant) sends the same Idempotency-Key to each. The scope includes the
// tenant, so each tenant gets its own row: no replay and no 409.
func TestSamePrincipalAndKeyInTwoTenantsAreIndependent(t *testing.T) {
	e := newETEnv(t)
	tokens := e.tokens
	both := e.a.admin
	inB := identity.Principal{TenantID: e.b.id, PrincipalID: both.PrincipalID, Groups: []uuid.UUID{e.b.adminGroup}}
	if _, err := e.super.Exec(t.Context(), "INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at) VALUES ($1, $2, 'dual', 'dual@b.test', true, '[]', now())", e.b.id, inB.PrincipalID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(t.Context(), "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $3)", e.b.id, e.b.adminGroup, inB.PrincipalID); err != nil {
		t.Fatal(err)
	}
	tokens["dual-a"], tokens["dual-b"] = both, inB

	const key = "same-key"
	for _, tc := range []struct{ token, entity, prefix string }{{"dual-a", "alpha", "ALP"}, {"dual-b", "beta", "BET"}} {
		if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", tc.token, key, etBody(tc.entity, tc.prefix)); rec.Code != http.StatusCreated {
			t.Fatalf("%s create = %d, want 201 (no replay, no 409): %s", tc.token, rec.Code, rec.Body)
		}
	}
	for _, tc := range []struct {
		tenant uuid.UUID
		want   string
	}{{e.a.id, "alpha"}, {e.b.id, "beta"}} {
		var got string
		if err := e.super.QueryRow(t.Context(), "SELECT string_agg(key, ',') FROM entity_definitions WHERE tenant_id = $1", tc.tenant).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("tenant %s has %q, want %q", tc.tenant, got, tc.want)
		}
	}
}
