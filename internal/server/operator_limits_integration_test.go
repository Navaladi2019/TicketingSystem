//go:build integration

package server_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/tenantlimits"
)

// S22 P7-11 (TICK-751, folded into TICK-916; ADR-006 D12, I8): an operator
// raises a tenant's per-type filterable limit through an audited
// tenant_limits row; publish enforces it.

func limitsPath(tenant uuid.UUID) string {
	return "/api/v1/operator/tenants/" + tenant.String() + "/limits"
}

func limitsBody(n int, reason string) string {
	return fmt.Sprintf(`{"filterable_fields_per_type": %d, "reason": %q}`, n, reason)
}

// limitsState is tenant's stored limit ("none" without a row) and its
// tenant_limits audit entries.
func (v *idEnv) limitsState(t *testing.T, tenant uuid.UUID) (string, int) {
	t.Helper()
	var stored string
	if err := v.e.super.QueryRow(t.Context(), `SELECT coalesce((SELECT filterable_fields_per_type::text FROM tenant_limits WHERE tenant_id = $1), 'none')`,
		tenant).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	return stored, v.count(t, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND subject_type = 'tenant_limits'`, tenant)
}

// covers: setTenantLimits 200
// The operator sets 10: 200, one row, one config entry under tenant_limits
// with the operator as actor and the reason in the diff. The same request
// again is 200 and no second entry. Tenant B is untouched.
func TestSetTenantLimitsRaisesOnceAndAudits(t *testing.T) {
	v := newIDEnv(t, nil)
	op := v.operatorToken(t)
	a, b := v.e.a.id, v.e.b.id
	body := limitsBody(10, "measured cost check (TICK-754)")

	for i, key := range []string{"raise", "raise"} {
		rec := v.call(t, http.MethodPut, limitsPath(a), op, key, body)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"filterable_fields_per_type":10`) {
			t.Fatalf("call %d = %d %s, want 200 with 10", i, rec.Code, rec.Body)
		}
	}
	if stored, entries := v.limitsState(t, a); stored != "10" || entries != 1 {
		t.Errorf("tenant A = %s with %d entries, want 10 with 1 (the replay writes none)", stored, entries)
	}
	var domain, action, actor, source, diff, updatedBy string
	if err := v.e.super.QueryRow(t.Context(), `SELECT a.domain, a.action, a.actor_system, a.source, a.diff::text, l.updated_by
		FROM audit_log a JOIN tenant_limits l ON l.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.subject_type = 'tenant_limits' AND a.subject_id = $1`, a).
		Scan(&domain, &action, &actor, &source, &diff, &updatedBy); err != nil {
		t.Fatal(err)
	}
	if domain != "config" || action != "update" || source != "system" || actor != "operator:"+updatedBy || updatedBy == "" ||
		!strings.Contains(diff, "measured cost check") || !strings.Contains(diff, `"value":10`) {
		t.Errorf("entry = %s %s actor %q source %s diff %s (updated_by %q)", domain, action, actor, source, diff, updatedBy)
	}
	if stored, entries := v.limitsState(t, b); stored != "none" || entries != 0 {
		t.Errorf("tenant B = %s with %d entries, want none and 0", stored, entries)
	}

	// Lowering it back to 5 is a change: one more entry.
	if rec := v.call(t, http.MethodPut, limitsPath(a), op, "lower", limitsBody(5, "rollback")); rec.Code != http.StatusOK {
		t.Fatalf("lower = %d %s", rec.Code, rec.Body)
	}
	if stored, entries := v.limitsState(t, a); stored != "5" || entries != 2 {
		t.Errorf("tenant A after lowering = %s with %d entries, want 5 with 2", stored, entries)
	}
}

// covers: setTenantLimits 400
// 4, 11, a missing or blank reason, a missing value or no body: 400
// validation_failed, nothing written.
func TestSetTenantLimitsRefusesBadInputAndWritesNothing(t *testing.T) {
	v := newIDEnv(t, nil)
	op := v.operatorToken(t)
	a := v.e.a.id
	for name, body := range map[string]string{
		"4":              limitsBody(4, "x"),
		"11":             limitsBody(11, "x"),
		"no reason":      `{"filterable_fields_per_type": 8}`,
		"a blank reason": limitsBody(8, "   "),
		"no value":       `{"reason": "x"}`,
		"a string value": `{"filterable_fields_per_type": "8", "reason": "x"}`,
	} {
		rec := v.call(t, http.MethodPut, limitsPath(a), op, "bad-"+name, body)
		if rec.Code != http.StatusBadRequest || problemCode(rec) != problem.ValidationFailed {
			t.Errorf("%s = %d %s, want 400 validation_failed", name, rec.Code, rec.Body)
		}
	}
	if stored, entries := v.limitsState(t, a); stored != "none" || entries != 0 {
		t.Errorf("tenant A = %s with %d entries, want none and 0", stored, entries)
	}
}

// covers: setTenantLimits 422
// A body with a duplicate member name is 422 invalid_tenant_limits (TICK-964's
// strict JSON), and nothing is written.
func TestSetTenantLimitsDuplicateMemberIs422(t *testing.T) {
	v := newIDEnv(t, nil)
	a := v.e.a.id
	body := `{"filterable_fields_per_type": 6, "filterable_fields_per_type": 10, "reason": "dup"}`
	wantStatus(t, v.call(t, http.MethodPut, limitsPath(a), v.operatorToken(t), "dup", body), http.StatusUnprocessableEntity, problem.InvalidTenantLimits)
	if stored, entries := v.limitsState(t, a); stored != "none" || entries != 0 {
		t.Errorf("tenant A = %s with %d entries, want none and 0", stored, entries)
	}
}

// covers: setTenantLimits 401
// covers: setTenantLimits 403
// No token or a forged one is 401; a tenant admin's own token is 403, for its
// own tenant too. Nothing is written.
func TestSetTenantLimitsRefusesNonOperators(t *testing.T) {
	v := newIDEnv(t, nil)
	a := v.e.a.id
	body := limitsBody(10, "self-service")
	for _, c := range []struct {
		name, token string
		status      int
		code        problem.Code
	}{
		{"no token", "", 401, problem.InvalidToken},
		{"forged token", "a.b.c", 401, problem.InvalidToken},
		{"tenant admin token", v.tenantToken(t, "a-admin", "tenant-a", "org-a"), 403, problem.Forbidden},
		{"tenant user token", v.tenantToken(t, "a-user", "tenant-a", "org-a"), 403, problem.Forbidden},
	} {
		wantStatus(t, v.call(t, http.MethodPut, limitsPath(a), c.token, "k-"+c.name, body), c.status, c.code)
	}
	if stored, entries := v.limitsState(t, a); stored != "none" || entries != 0 {
		t.Errorf("tenant A = %s with %d entries, want none and 0", stored, entries)
	}
}

// covers: setTenantLimits 404
// An unknown tenant is 404 and writes nothing anywhere.
func TestSetTenantLimitsUnknownTenantIs404(t *testing.T) {
	v := newIDEnv(t, nil)
	wantStatus(t, v.call(t, http.MethodPut, limitsPath(uuid.New()), v.operatorToken(t), "k", limitsBody(10, "x")), http.StatusNotFound, problem.NotFound)
	if n := v.count(t, "SELECT count(*) FROM tenant_limits"); n != 0 {
		t.Errorf("tenant_limits rows = %d, want 0", n)
	}
	if n := v.count(t, "SELECT count(*) FROM audit_log WHERE subject_type = 'tenant_limits'"); n != 0 {
		t.Errorf("tenant_limits entries = %d, want 0", n)
	}
}

// limitsHandler is an operator handler with the limits route on pool, or off
// when pool is nil.
func limitsHandler(t *testing.T, v *idEnv, opts ...server.Option) http.Handler {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	adapter := v.realm.Resolver(t, pgstore.New(v.e.pool), operatorSub)
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, adapter, pgstore.New(v.e.pool),
		append([]server.Option{server.WithOperator(adapter, nil), server.OperatorOnly()}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func serveLimits(t *testing.T, h http.Handler, token, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", "k")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// covers: setTenantLimits 500
// A database failure is 500 with no cause in the body.
func TestSetTenantLimitsDatabaseDownIs500(t *testing.T) {
	v := newIDEnv(t, nil)
	dead := roleLoginPool(t, "nava_app", 1)
	dead.Close()
	rec := serveLimits(t, limitsHandler(t, v, server.WithOperatorLimits(dead)), v.operatorToken(t), limitsPath(v.e.a.id), limitsBody(10, "x"))
	wantStatus(t, rec, http.StatusInternalServerError, problem.InternalError)
	if strings.Contains(rec.Body.String(), "closed") {
		t.Errorf("500 body leaks the cause: %s", rec.Body)
	}
}

// covers: setTenantLimits 501
// Without WithOperatorLimits the route answers 501 and writes nothing.
func TestSetTenantLimitsNotConfiguredIs501(t *testing.T) {
	v := newIDEnv(t, nil)
	rec := serveLimits(t, limitsHandler(t, v), v.operatorToken(t), limitsPath(v.e.a.id), limitsBody(10, "x"))
	wantStatus(t, rec, http.StatusNotImplemented, problem.NotImplemented)
	if n := v.count(t, "SELECT count(*) FROM tenant_limits"); n != 0 {
		t.Errorf("tenant_limits rows = %d, want 0", n)
	}
}

// setLimit raises tenant's limit to n through tenantlimits on the nava_app
// pool, with its audit entry, as the operator route does.
func setLimit(t *testing.T, e *etEnv, tenant uuid.UUID, n int) {
	t.Helper()
	err := tenancy.InTenantTx(t.Context(), e.pool, tenant, func(tx pgx.Tx) error {
		_, err := tenantlimits.SetPerType(t.Context(), tx, tenant, n, "test", "op-test")
		return err
	})
	if err != nil {
		t.Fatalf("set limit: %v", err)
	}
}

// AC (ADR-006 D12): with tenant A raised to 10, publishing 10 flags on one
// type is 201 and an 11th is 422 index_quota_exceeded; tenant B, still at the
// default, is refused a 6th.
//
// covers: publishEntityTypeVersion 201 422
func TestPublishHonoursTheTenantsRaisedLimit(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "k-create-b", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("create B = %d: %s", rec.Code, rec.Body)
	}
	setLimit(t, e, e.a.id, 10)
	names := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprintf("f%02d", i)
		}
		return out
	}
	if rec := etPublishKey(t, e, "incident", "admin-a", "k-10", etFlaggedProps(t, names(10)...)); rec.Code != http.StatusCreated {
		t.Fatalf("tenant A publishes 10 flags = %d: %s", rec.Code, rec.Body)
	}
	wantProblem(t, etPublishKey(t, e, "incident", "admin-a", "k-11", etFlaggedProps(t, names(11)...)), http.StatusUnprocessableEntity, problem.IndexQuotaExceeded)
	if n := len(etRegistry(t, e, e.a.id)); n != 10 {
		t.Errorf("tenant A registry rows = %d, want 10", n)
	}
	wantProblem(t, etPublishKey(t, e, "incident", "admin-b", "k-b6", etFlaggedProps(t, names(6)...)), http.StatusUnprocessableEntity, problem.IndexQuotaExceeded)
	if n := len(etRegistry(t, e, e.b.id)); n != 0 {
		t.Errorf("tenant B registry rows = %d, want 0", n)
	}
}
