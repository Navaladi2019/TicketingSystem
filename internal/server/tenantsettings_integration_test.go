//go:build integration

// P6-07 (TICK-672): the tenant time-zone route against a real PostgreSQL,
// through the real handler. Helpers are in entitytypes_integration_test.go.

package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

const tzPath = "/api/v1/tenant/settings/time-zone"

func tzOf(t *testing.T, e *etEnv, tenant uuid.UUID) (tz string) {
	t.Helper()
	if err := e.super.QueryRow(t.Context(), "SELECT time_zone FROM tenants WHERE id = $1", tenant).Scan(&tz); err != nil {
		t.Fatal(err)
	}
	return tz
}

func tzAudit(t *testing.T, e *etEnv, tenant uuid.UUID) (n int) {
	t.Helper()
	if err := e.super.QueryRow(t.Context(),
		"SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND subject_type = 'tenant_setting'", tenant).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func tzBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var v struct {
		TimeZone string `json:"time_zone"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("body %s: %v", rec.Body, err)
	}
	return v.TimeZone
}

// covers: getTenantTimeZone 200
// covers: putTenantTimeZone 200
func TestTenantTimeZoneSetAndRead(t *testing.T) {
	e := newETEnv(t)
	if r := e.do(t, http.MethodGet, tzPath, "admin-a", "", ""); r.Code != 200 || tzBody(t, r) != "UTC" {
		t.Fatalf("default = %d %s", r.Code, r.Body)
	}
	r := e.do(t, http.MethodPut, tzPath, "admin-a", "tz1", `{"time_zone":"Europe/Berlin"}`)
	if r.Code != 200 || tzBody(t, r) != "Europe/Berlin" {
		t.Fatalf("put = %d %s", r.Code, r.Body)
	}
	if got := tzOf(t, e, e.a.id); got != "Europe/Berlin" {
		t.Errorf("stored %q", got)
	}
	if r := e.do(t, http.MethodGet, tzPath, "admin-a", "", ""); tzBody(t, r) != "Europe/Berlin" {
		t.Errorf("read back = %s", r.Body)
	}
	// One audit entry: config update on the tenant, actor from the token.
	var action, domain, source string
	var actor uuid.UUID
	var vb, va *int64
	if err := e.super.QueryRow(t.Context(), `SELECT action, domain, source, actor_principal_id, version_before, version_after
		FROM audit_log WHERE tenant_id = $1 AND subject_type = 'tenant_setting' AND subject_id = $1`, e.a.id).
		Scan(&action, &domain, &source, &actor, &vb, &va); err != nil {
		t.Fatal(err)
	}
	if action != "update" || domain != "config" || source != "user" || actor != e.a.admin.PrincipalID || vb != nil || va != nil {
		t.Errorf("audit = %s %s %s %v %v %v", action, domain, source, actor, vb, va)
	}
	// The same value again writes nothing and audits nothing.
	if r := e.do(t, http.MethodPut, tzPath, "admin-a", "tz2", `{"time_zone":"Europe/Berlin"}`); r.Code != 200 {
		t.Fatalf("same value = %d %s", r.Code, r.Body)
	}
	if n := tzAudit(t, e, e.a.id); n != 1 {
		t.Errorf("%d audit entries after a no-op, want 1", n)
	}
}

// covers: putTenantTimeZone 422
func TestTenantTimeZoneInvalidWritesNothing(t *testing.T) {
	e := newETEnv(t)
	for i, body := range []string{
		`{"time_zone":"Mars/Base"}`, `{"time_zone":""}`, `{"time_zone":"Local"}`, `{"time_zone":" UTC"}`,
		`{"time_zone":1}`, `{}`, `[]`, `{"time_zone":"` + strings.Repeat("A", 80) + `"}`,
	} {
		r := e.do(t, http.MethodPut, tzPath, "admin-a", "bad"+string(rune('a'+i)), body)
		nfProblem(t, r, 422, "invalid_time_zone")
		if strings.Contains(r.Body.String(), "Mars") {
			t.Errorf("the value was echoed: %s", r.Body)
		}
	}
	if got := tzOf(t, e, e.a.id); got != "UTC" {
		t.Errorf("stored %q after refusals", got)
	}
	if n := tzAudit(t, e, e.a.id); n != 0 {
		t.Errorf("%d audit entries after refusals", n)
	}
}

// covers: getTenantTimeZone 401 403
// covers: putTenantTimeZone 400 401 403
func TestTenantTimeZoneAuthAndContract(t *testing.T) {
	e := newETEnv(t)
	body := `{"time_zone":"Asia/Tokyo"}`
	nfProblem(t, e.do(t, http.MethodGet, tzPath, "", "", ""), 401, "invalid_token")
	nfProblem(t, e.do(t, http.MethodGet, tzPath, "user-a", "", ""), 403, "forbidden")
	nfProblem(t, e.do(t, http.MethodPut, tzPath, "", "k1", body), 401, "invalid_token")
	nfProblem(t, e.do(t, http.MethodPut, tzPath, "user-a", "k2", body), 403, "forbidden")
	nfProblem(t, e.do(t, http.MethodPut, tzPath, "lookalike-a", "k3", body), 403, "forbidden")
	nfProblem(t, e.do(t, http.MethodPut, tzPath, "admin-a", "", body), 400, "validation_failed")
	if got := tzOf(t, e, e.a.id); got != "UTC" || tzAudit(t, e, e.a.id) != 0 {
		t.Errorf("a refused request wrote: %q", got)
	}
}

// Idempotent replay writes once and returns the same body.
func TestTenantTimeZoneReplay(t *testing.T) {
	e := newETEnv(t)
	first := e.do(t, http.MethodPut, tzPath, "admin-a", "same", `{"time_zone":"Asia/Tokyo"}`)
	second := e.do(t, http.MethodPut, tzPath, "admin-a", "same", `{"time_zone":"Asia/Tokyo"}`)
	if first.Code != 200 || second.Code != 200 || first.Body.String() != second.Body.String() {
		t.Fatalf("replay = %d %d\n%s\n%s", first.Code, second.Code, first.Body, second.Body)
	}
	if n := tzAudit(t, e, e.a.id); n != 1 {
		t.Errorf("%d audit entries, want 1", n)
	}
}

// A tenant A token reads and sets A's setting only; a forged tenant_id in the
// body, query or path changes nothing of B's.
func TestTenantTimeZoneIsolation(t *testing.T) {
	e := newETEnv(t)
	forged := `{"tenant_id":"` + e.b.id.String() + `","time_zone":"Asia/Tokyo"}`
	if r := e.do(t, http.MethodPut, tzPath, "admin-a", "f1", forged); r.Code != 200 {
		t.Fatalf("put = %d %s", r.Code, r.Body)
	}
	if a, b := tzOf(t, e, e.a.id), tzOf(t, e, e.b.id); a != "Asia/Tokyo" || b != "UTC" {
		t.Errorf("A = %q, B = %q; want Asia/Tokyo and UTC", a, b)
	}
	if tzAudit(t, e, e.b.id) != 0 || tzAudit(t, e, e.a.id) != 1 {
		t.Error("the audit entry is not tenant A's alone")
	}
	// B's admin sees B's own value.
	if r := e.do(t, http.MethodGet, tzPath+"?tenant_id="+e.a.id.String(), "admin-b", "", ""); r.Code != 200 || tzBody(t, r) != "UTC" {
		t.Errorf("B reads = %d %s", r.Code, r.Body)
	}
	// A path that names a tenant is no route: it answers like any unknown path.
	missing := e.do(t, http.MethodGet, "/api/v1/tenant/"+e.b.id.String()+"/settings/time-zone", "admin-a", "", "")
	unknown := e.do(t, http.MethodGet, "/api/v1/tenant/"+uuid.NewString()+"/settings/time-zone", "admin-a", "", "")
	if missing.Code != 404 || unknown.Code != 404 {
		t.Errorf("tenant-scoped path = %d, %d; want 404, 404", missing.Code, unknown.Code)
	}
}
