//go:build integration

// P6-07 (TICK-672): the number-format admin API against a real PostgreSQL,
// through the real handler. Helpers are in entitytypes_integration_test.go.

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/numberformats"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

const (
	nfPath    = "/api/v1/number-formats"
	nfMonthly = `{"key":"monthly","document":{"template":"{prefix}-{yyyy}{MM}-{seq:4}","overflow":"grow"}}`
	nfDefault = `{"key":"plain","document":{"template":"{prefix}{seq:7}","overflow":"grow"}}`
)

// nfGate is a gate whose probe answers *open and whose clock is *now.
func nfGate(open *bool, now *time.Time) *numberformats.Gate {
	return numberformats.NewGate(func(context.Context, numberformats.Querier) (bool, error) { return *open, nil },
		func() time.Time { return *now })
}

// doIf is do with an If-Match header.
func (e *etEnv) doIf(t *testing.T, method, path, token, idemKey, ifMatch, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func nfCount(t *testing.T, e *etEnv, table string) (n int) {
	t.Helper()
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE tenant_id = $1", e.a.id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func nfAudit(t *testing.T, e *etEnv) (n int) {
	t.Helper()
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND subject_type = 'number_format'", e.a.id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func nfID(t *testing.T, body []byte) string {
	t.Helper()
	var v struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &v); err != nil || v.ID == "" {
		t.Fatalf("no id in %s: %v", body, err)
	}
	return v.ID
}

func nfProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var p struct {
		Code string `json:"code"`
	}
	if rec.Code != status || json.Unmarshal(rec.Body.Bytes(), &p) != nil || p.Code != code {
		t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body, status, code)
	}
}

// covers: createNumberFormat 201
// covers: getNumberFormat 200
// covers: listNumberFormats 200
func TestNumberFormatCreateGetList(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	rec := e.do(t, http.MethodPost, nfPath, "admin-a", "k1", nfMonthly)
	if rec.Code != http.StatusCreated || rec.Header().Get("ETag") != `W/"1"` {
		t.Fatalf("create = %d %s %s", rec.Code, rec.Header().Get("ETag"), rec.Body)
	}
	id := nfID(t, rec.Body.Bytes())
	var v struct {
		Key            string         `json:"key"`
		Version        int            `json:"version"`
		CurrentVersion int            `json:"current_version"`
		Compiled       map[string]any `json:"compiled"`
		CompiledSHA256 string         `json:"compiled_sha256"`
		CreatedBy      uuid.UUID      `json:"created_by"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v.Key != "monthly" || v.Version != 1 || v.CurrentVersion != 1 || len(v.CompiledSHA256) != 64 || v.Compiled["Head"] != "{prefix}-" || v.CreatedBy != e.a.admin.PrincipalID {
		t.Errorf("body = %+v", v)
	}
	// Audit: one config entry, actor and source from the token.
	var actor uuid.UUID
	var source, action string
	if err := e.super.QueryRow(t.Context(), `SELECT actor_principal_id, source, action FROM audit_log
		WHERE tenant_id = $1 AND subject_type = 'number_format' AND subject_id = $2`, e.a.id, id).Scan(&actor, &source, &action); err != nil {
		t.Fatal(err)
	}
	if actor != e.a.admin.PrincipalID || source != "user" || action != "create" {
		t.Errorf("audit = %v %s %s", actor, source, action)
	}
	// Any user reads.
	got := e.do(t, http.MethodGet, nfPath+"/"+id, "user-a", "", "")
	if got.Code != http.StatusOK || got.Body.String() != rec.Body.String() {
		t.Errorf("get = %d\n%s\n%s", got.Code, got.Body, rec.Body)
	}
	if g := e.do(t, http.MethodGet, nfPath+"/"+id+"?version=1", "user-a", "", ""); g.Code != 200 {
		t.Errorf("get v1 = %d", g.Code)
	}
	list := e.do(t, http.MethodGet, nfPath, "user-a", "", "")
	if list.Code != 200 || !strings.Contains(list.Body.String(), id) {
		t.Errorf("list = %d %s", list.Code, list.Body)
	}
	// Another tenant's list does not show it.
	if l := e.do(t, http.MethodGet, nfPath, "user-b", "", ""); l.Code != 200 || strings.Contains(l.Body.String(), id) {
		t.Errorf("tenant B list = %d %s", l.Code, l.Body)
	}
}

// covers: getNumberFormat 404
func TestNumberFormatGetMissingIsOneIdenticalBody(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	id := nfID(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k1", nfMonthly).Body.Bytes())
	missing := e.do(t, http.MethodGet, nfPath+"/"+uuid.NewString(), "user-b", "", "")
	cases := map[string]*httptest.ResponseRecorder{
		"foreign": e.do(t, http.MethodGet, nfPath+"/"+id, "user-b", "", ""),
		"junk":    e.do(t, http.MethodGet, nfPath+"/not-a-uuid", "user-b", "", ""),
		"version": e.do(t, http.MethodGet, nfPath+"/"+id+"?version=9", "user-a", "", ""),
		// PUT on another tenant's id is the same 404.
		"put foreign": e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-b", "k2", `W/"1"`, `{"document":{"template":"{prefix}{seq:7}","overflow":"grow"}}`),
	}
	if missing.Code != 404 {
		t.Fatalf("missing = %d", missing.Code)
	}
	for name, r := range cases {
		if r.Code != 404 || !bytes.Equal(r.Body.Bytes(), missing.Body.Bytes()) {
			t.Errorf("%s = %d %s, want the missing id's 404", name, r.Code, r.Body)
		}
	}
}

// covers: createNumberFormat 401
// covers: createNumberFormat 403
// covers: publishNumberFormatVersion 401
// covers: publishNumberFormatVersion 403
func TestNumberFormatAuth(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	id := nfID(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k1", nfMonthly).Body.Bytes())
	total := func() int {
		return nfCount(t, e, "number_formats") + nfCount(t, e, "number_format_versions")*10 + nfAudit(t, e)*100
	}
	before := total()
	put := `{"document":{"template":"{prefix}{seq:7}","overflow":"grow"}}`
	if r := e.do(t, http.MethodPost, nfPath, "", "k2", nfDefault); r.Code != 401 {
		t.Errorf("create no token = %d", r.Code)
	}
	if r := e.do(t, http.MethodPost, nfPath, "user-a", "k3", nfDefault); r.Code != 403 {
		t.Errorf("create non-admin = %d %s", r.Code, r.Body)
	}
	if r := e.do(t, http.MethodPost, nfPath, "lookalike-a", "k4", nfDefault); r.Code != 403 {
		t.Errorf("create lookalike = %d", r.Code)
	}
	if r := e.doIf(t, http.MethodPut, nfPath+"/"+id, "", "k5", `W/"1"`, put); r.Code != 401 {
		t.Errorf("put no token = %d", r.Code)
	}
	if r := e.doIf(t, http.MethodPut, nfPath+"/"+id, "user-a", "k6", `W/"1"`, put); r.Code != 403 {
		t.Errorf("put non-admin = %d", r.Code)
	}
	if after := total(); after != before {
		t.Errorf("a refused request wrote: %d -> %d", before, after)
	}
}

// covers: createNumberFormat 409
// covers: createNumberFormat 422
func TestNumberFormatCreateRefusals(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	if r := e.do(t, http.MethodPost, nfPath, "admin-a", "k1", nfMonthly); r.Code != 201 {
		t.Fatal(r.Body)
	}
	heads, versions, audits := nfCount(t, e, "number_formats"), nfCount(t, e, "number_format_versions"), nfAudit(t, e)
	nfProblem(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k2", nfMonthly), 409, "number_format_exists")
	nfProblem(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k3",
		`{"key":"nopfx","document":{"template":"X{prefix}{seq:4}","overflow":"grow"}}`), 422, "number_format_prefix_required")
	nfProblem(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k4",
		`{"key":"noseq","document":{"template":"{prefix}-{yyyy}","overflow":"grow"}}`), 422, "number_format_invalid")
	nfProblem(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k5",
		`{"key":"BAD KEY","document":{"template":"{prefix}{seq:4}","overflow":"grow"}}`), 422, "number_format_invalid")
	nfProblem(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k6", `{"key":"nodoc"}`), 422, "number_format_invalid")
	nfProblem(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k7",
		`{"key":"zone","document":{"template":"{prefix}{seq:4}","overflow":"grow","time_zone":"Mars/Base"}}`), 422, "number_format_invalid")
	if h, v, a := nfCount(t, e, "number_formats"), nfCount(t, e, "number_format_versions"), nfAudit(t, e); h != heads || v != versions || a != audits {
		t.Errorf("invalid input wrote: heads %d->%d versions %d->%d audit %d->%d", heads, h, versions, v, audits, a)
	}
}

// covers: publishNumberFormatVersion 200
// covers: publishNumberFormatVersion 404
// covers: publishNumberFormatVersion 412
// covers: publishNumberFormatVersion 422
// covers: publishNumberFormatVersion 428
func TestNumberFormatPublish(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	id := nfID(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k1", nfMonthly).Body.Bytes())
	p := nfPath + "/" + id
	v2 := `{"document":{"template":"{prefix}-{yy}{MM}-{seq:5}","overflow":"refuse"}}`

	nfProblem(t, e.doIf(t, http.MethodPut, p, "admin-a", "k2", "", v2), 428, "if_match_required")
	nfProblem(t, e.doIf(t, http.MethodPut, p, "admin-a", "k3", "*", v2), 428, "if_match_required")
	stale := e.doIf(t, http.MethodPut, p, "admin-a", "k4", `W/"7"`, v2)
	nfProblem(t, stale, 412, "version_conflict")
	if stale.Header().Get("ETag") != `W/"1"` {
		t.Errorf("412 ETag = %q", stale.Header().Get("ETag"))
	}
	nfProblem(t, e.doIf(t, http.MethodPut, p, "admin-a", "k5", `W/"1"`, `{"document":{"template":"{prefix}","overflow":"grow"}}`), 422, "number_format_invalid")
	nfProblem(t, e.doIf(t, http.MethodPut, p, "admin-a", "k6", `W/"1"`, `{"document":{"template":"{yyyy}{prefix}-{seq:4}","overflow":"grow"}}`), 422, "number_format_prefix_required")
	if nfCount(t, e, "number_format_versions") != 1 || nfAudit(t, e) != 1 {
		t.Fatal("a refused publish wrote")
	}
	nfProblem(t, e.doIf(t, http.MethodPut, nfPath+"/"+uuid.NewString(), "admin-a", "k7", `W/"1"`, v2), 404, "not_found")

	ok := e.doIf(t, http.MethodPut, p, "admin-a", "k8", `W/"1"`, v2)
	if ok.Code != 200 || ok.Header().Get("ETag") != `W/"2"` || !strings.Contains(ok.Body.String(), `"version":2`) || !strings.Contains(ok.Body.String(), `"current_version":2`) {
		t.Fatalf("publish = %d %s %s", ok.Code, ok.Header().Get("ETag"), ok.Body)
	}
	// The same If-Match is now stale.
	nfProblem(t, e.doIf(t, http.MethodPut, p, "admin-a", "k9", `W/"1"`, v2), 412, "version_conflict")
	// Version 1 still reads.
	if g := e.do(t, http.MethodGet, p+"?version=1", "user-a", "", ""); g.Code != 200 || !strings.Contains(g.Body.String(), `"version":1`) {
		t.Errorf("v1 = %d %s", g.Code, g.Body)
	}
	var action string
	var vb, va int
	if err := e.super.QueryRow(t.Context(), `SELECT action, version_before, version_after FROM audit_log
		WHERE tenant_id = $1 AND subject_type = 'number_format' AND subject_id = $2 AND version_after = 2`, e.a.id, id).Scan(&action, &vb, &va); err != nil || action != "publish" || vb != 1 {
		t.Errorf("publish audit = %s %d %d (%v)", action, vb, va, err)
	}
}

func TestNumberFormatConcurrentPublishesAgainstOneIfMatch(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	id := nfID(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k1", nfMonthly).Body.Bytes())
	const n = 8
	codes := make(chan int, n)
	for i := range n {
		go func() {
			body := fmt.Sprintf(`{"document":{"template":"{prefix}-{yyyy}-{seq:%d}","overflow":"grow"}}`, i+1)
			codes <- e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-a", fmt.Sprintf("c%d", i), `W/"1"`, body).Code
		}()
	}
	won := 0
	for range n {
		switch c := <-codes; c {
		case 200:
			won++
		case 412:
		default:
			t.Errorf("code %d", c)
		}
	}
	if won != 1 {
		t.Errorf("%d publishes won against the same If-Match, want 1", won)
	}
}

// The gate: with M3 not applied a non-default publish is refused and the
// default is not; after M3 the answer changes only when the 60 s cache ends
// (injected clock).
//
// covers: createNumberFormat 422
// covers: publishNumberFormatVersion 422
func TestNumberFormatGate(t *testing.T) {
	open, now := false, time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	const unavailable = "numbering_templates_unavailable"

	nfProblem(t, e.do(t, http.MethodPost, nfPath, "admin-a", "g1", nfMonthly), 422, unavailable)
	if nfCount(t, e, "number_formats") != 0 || nfCount(t, e, "number_format_versions") != 0 || nfAudit(t, e) != 0 {
		t.Fatal("a gated create wrote")
	}
	def := e.do(t, http.MethodPost, nfPath, "admin-a", "g2", nfDefault)
	if def.Code != 201 {
		t.Fatalf("default create = %d %s", def.Code, def.Body)
	}
	id := nfID(t, def.Body.Bytes())
	put := `{"document":{"template":"{prefix}-{yyyy}-{seq:4}","overflow":"grow"}}`
	nfProblem(t, e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-a", "g3", `W/"1"`, put), 422, unavailable)

	open = true // M3 runs
	now = now.Add(numberformats.GateTTL - time.Second)
	nfProblem(t, e.do(t, http.MethodPost, nfPath, "admin-a", "g4", nfMonthly), 422, unavailable)
	now = now.Add(time.Second)
	if r := e.do(t, http.MethodPost, nfPath, "admin-a", "g5", nfMonthly); r.Code != 201 {
		t.Fatalf("after the TTL = %d %s", r.Code, r.Body)
	}
	if r := e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-a", "g6", `W/"1"`, put); r.Code != 200 {
		t.Fatalf("publish after the TTL = %d %s", r.Code, r.Body)
	}
}

// The production probe reads the catalog: on the migrated database
// record_counters_pkey covers scope_key (M3).
func TestNumberFormatCatalogProbeSeesM3(t *testing.T) {
	e := newETEnv(t)
	open, err := numberformats.CatalogProbe(t.Context(), e.pool)
	if err != nil || !open {
		t.Fatalf("CatalogProbe = %v, %v; want open on a migrated database", open, err)
	}
	if r := e.do(t, http.MethodPost, nfPath, "admin-a", "p1", nfMonthly); r.Code != 201 {
		t.Errorf("create with the default gate = %d %s", r.Code, r.Body)
	}
}

// Idempotent replay of the create writes once and returns the same body.
func TestNumberFormatCreateReplay(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	first := e.do(t, http.MethodPost, nfPath, "admin-a", "same", nfMonthly)
	second := e.do(t, http.MethodPost, nfPath, "admin-a", "same", nfMonthly)
	if first.Code != 201 || second.Code != 201 || first.Body.String() != second.Body.String() {
		t.Fatalf("replay = %d %d\n%s\n%s", first.Code, second.Code, first.Body, second.Body)
	}
	if nfCount(t, e, "number_formats") != 1 || nfCount(t, e, "number_format_versions") != 1 || nfAudit(t, e) != 1 {
		t.Error("the replay wrote again")
	}
}

// A forged tenant_id in the body is ignored: the format is stored under the
// token's tenant.
func TestNumberFormatForgedTenantIDIsIgnored(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	body := fmt.Sprintf(`{"tenant_id":%q,"key":"forged","document":{"template":"{prefix}-{yyyy}-{seq:4}","overflow":"grow"}}`, e.b.id)
	rec := e.do(t, http.MethodPost, nfPath, "admin-a", "f1", body)
	if rec.Code != 201 {
		t.Fatal(rec.Body)
	}
	id := nfID(t, rec.Body.Bytes())
	var tenant uuid.UUID
	if err := e.super.QueryRow(t.Context(), "SELECT tenant_id FROM number_formats WHERE id = $1", id).Scan(&tenant); err != nil || tenant != e.a.id {
		t.Errorf("stored under %v (%v), want tenant A", tenant, err)
	}
	// The same key in tenant B is free: keys are unique per tenant.
	if r := e.do(t, http.MethodPost, nfPath, "admin-b", "f2", strings.Replace(body, e.b.id.String(), e.a.id.String(), 1)); r.Code != 201 {
		t.Errorf("tenant B create of the same key = %d %s", r.Code, r.Body)
	}
}

// The contract's own refusals: the validator's 400 and the authentication's
// 401, before any handler code.
//
// covers: createNumberFormat 400
// covers: publishNumberFormatVersion 400
// covers: listNumberFormats 400 401
// covers: getNumberFormat 400 401
func TestNumberFormatContractRefusals(t *testing.T) {
	open, now := true, time.Now()
	e := newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)))
	id := nfID(t, e.do(t, http.MethodPost, nfPath, "admin-a", "k1", nfMonthly).Body.Bytes())
	p := nfPath + "/" + id
	// No Idempotency-Key.
	nfProblem(t, e.do(t, http.MethodPost, nfPath, "admin-a", "", nfMonthly), 400, "validation_failed")
	nfProblem(t, e.doIf(t, http.MethodPut, p, "admin-a", "", `W/"1"`, nfDefault), 400, "validation_failed")
	// Out-of-range parameters.
	nfProblem(t, e.do(t, http.MethodGet, nfPath+"?limit=0", "user-a", "", ""), 400, "validation_failed")
	nfProblem(t, e.do(t, http.MethodGet, p+"?version=0", "user-a", "", ""), 400, "validation_failed")
	// No token.
	nfProblem(t, e.do(t, http.MethodGet, nfPath, "", "", ""), 401, "invalid_token")
	nfProblem(t, e.do(t, http.MethodGet, p, "", "", ""), 401, "invalid_token")
}
