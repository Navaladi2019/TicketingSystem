//go:build integration

// Adversarial tenant-isolation guards for the number-format admin API
// (TICK-672, P6-07; Security & Multi-Tenancy review of PR #643). Threat
// model: an attacker holding a valid token for tenant B, aiming at tenant A's
// formats. Each test asserts both the response and that tenant A's rows are
// untouched. Helpers are in numberformats_integration_test.go and
// entitytypes_integration_test.go.

package server_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// nfSnapshot counts tenant A's format rows, version rows and number_format
// audit rows, so a test can prove a refused request wrote nothing in A.
func nfSnapshot(t *testing.T, e *etEnv) [3]int {
	t.Helper()
	return [3]int{nfCount(t, e, "number_formats"), nfCount(t, e, "number_format_versions"), nfAudit(t, e)}
}

// Tenant B's admin, naming tenant A's tenant_id in the body of a PUT on A's
// format id, gets the missing-id 404 byte for byte and writes nothing in A
// (I2: the tenant is the token's; nothing in a body moves it).
func TestNumberFormatIsolationForeignPublishWithForgedTenant(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	id := nfID(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k1", nfMonthly).Body.Bytes())
	before := nfSnapshot(t, e)

	body := fmt.Sprintf(`{"tenant_id":%q,"document":{"template":"{prefix}{seq:7}","overflow":"grow"}}`, e.a.id)
	missing := e.doIf(t, http.MethodPut, nfPath+"/"+uuid.NewString(), "admin-b", "b1", `W/"1"`, body)
	foreign := e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-b", "b2", `W/"1"`, body)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing = %d %s", missing.Code, missing.Body)
	}
	if foreign.Code != http.StatusNotFound || !bytes.Equal(foreign.Body.Bytes(), missing.Body.Bytes()) {
		t.Errorf("foreign PUT = %d %s, want the missing id's 404 %s", foreign.Code, foreign.Body, missing.Body)
	}
	if after := nfSnapshot(t, e); after != before {
		t.Errorf("tenant B's PUT wrote in tenant A: %v -> %v", before, after)
	}
	var cur int
	if err := e.super.QueryRow(t.Context(), "SELECT current_version FROM number_formats WHERE id = $1", id).Scan(&cur); err != nil || cur != 1 {
		t.Errorf("A's format current_version = %d (%v), want 1", cur, err)
	}
}

// A version read of another tenant's id is the same 404 as a missing id: the
// ?version= branch must not become an existence oracle.
func TestNumberFormatIsolationForeignVersionRead(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	id := nfID(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k1", nfMonthly).Body.Bytes())

	missing := e.do(t, http.MethodGet, nfPath+"/"+uuid.NewString()+"?version=1", "user-b", "", "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing = %d %s", missing.Code, missing.Body)
	}
	for _, tok := range []string{"user-b", "admin-b"} {
		r := e.do(t, http.MethodGet, nfPath+"/"+id+"?version=1", tok, "", "")
		if r.Code != http.StatusNotFound || !bytes.Equal(r.Body.Bytes(), missing.Body.Bytes()) {
			t.Errorf("%s GET foreign ?version=1 = %d %s, want the missing id's 404", tok, r.Code, r.Body)
		}
	}
}

// Tenant B reusing tenant A's Idempotency-Key string with the same body gets
// its own create, never a replay of A's response (keys are per tenant and
// principal), and the row lands in B.
func TestNumberFormatIsolationIdempotencyKeyIsPerTenant(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	a := e.do(t, http.MethodPost, nfPath, "admin-a", "shared-key", nfMonthly)
	b := e.do(t, http.MethodPost, nfPath, "admin-b", "shared-key", nfMonthly)
	if a.Code != http.StatusCreated || b.Code != http.StatusCreated {
		t.Fatalf("creates = %d %d\n%s\n%s", a.Code, b.Code, a.Body, b.Body)
	}
	idA, idB := nfID(t, a.Body.Bytes()), nfID(t, b.Body.Bytes())
	if idA == idB || bytes.Equal(a.Body.Bytes(), b.Body.Bytes()) {
		t.Fatalf("tenant B got tenant A's replayed response: %s", b.Body)
	}
	var tenant uuid.UUID
	if err := e.super.QueryRow(t.Context(), "SELECT tenant_id FROM number_formats WHERE id = $1", idB).Scan(&tenant); err != nil || tenant != e.b.id {
		t.Errorf("B's format stored under %v (%v), want tenant B", tenant, err)
	}
}

// C12: actor and source of the publish audit row come from the resolved
// principal. A body naming another principal, a system actor, a source and a
// created_by is ignored; created_by in the response is the token's principal.
func TestNumberFormatIsolationPublishAuditActorFromPrincipalOnly(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	id := nfID(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k1", nfMonthly).Body.Bytes())

	forged := fmt.Sprintf(`{"tenant_id":%q,"actor":{"principal_id":%q,"system":"importer"},"actor_principal_id":%q,"source":"system","created_by":%q,`+
		`"document":{"template":"{prefix}-{yy}{MM}-{seq:5}","overflow":"grow"}}`,
		e.b.id, e.a.user.PrincipalID, e.a.user.PrincipalID, e.b.admin.PrincipalID)
	r := e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-a", "k2", `W/"1"`, forged)
	if r.Code != http.StatusOK {
		t.Fatalf("publish = %d %s", r.Code, r.Body)
	}
	if !bytes.Contains(r.Body.Bytes(), []byte(fmt.Sprintf(`"created_by":%q`, e.a.admin.PrincipalID))) {
		t.Errorf("created_by is not the token's principal: %s", r.Body)
	}
	var (
		actor  *uuid.UUID
		system *string
		source string
	)
	if err := e.super.QueryRow(t.Context(), `SELECT actor_principal_id, actor_system, source FROM audit_log
		WHERE tenant_id = $1 AND subject_type = 'number_format' AND subject_id = $2 AND version_after = 2`, e.a.id, id).
		Scan(&actor, &system, &source); err != nil {
		t.Fatal(err)
	}
	if actor == nil || *actor != e.a.admin.PrincipalID || system != nil || source != "user" {
		t.Errorf("audit actor=%v system=%v source=%s, want principal %v, no system, user", actor, system, source, e.a.admin.PrincipalID)
	}
	var stored uuid.UUID
	if err := e.super.QueryRow(t.Context(), `SELECT tenant_id FROM number_format_versions WHERE format_id = $1 AND version = 2`, id).Scan(&stored); err != nil || stored != e.a.id {
		t.Errorf("version 2 stored under %v (%v), want tenant A", stored, err)
	}
	var n int
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND subject_type = 'number_format'`, e.b.id).Scan(&n); err != nil || n != 0 {
		t.Errorf("tenant B has %d number_format audit rows (%v), want 0", n, err)
	}
}
