package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
)

// recordCI8Paths are the ADR-001 CI-8 cases on the record routes (P3-16,
// TICK-87; PLAN-003 D-3-10, D-3-19). Tenant B, holding a valid token, names
// tenant A's record by id on GET, PATCH and GET …/history. Every answer must
// be the one a random UUID gets: same status, same headers (Content-Type,
// and no ETag), same body byte for byte. 404 bodies carry no instance
// (D-3-10), so nothing is stripped before the comparison.
//
// Tenant B has a type with tenant A's key (iso_et_rec), so B's request gets
// past the entity type lookup and reaches the record lookup: without it, the
// 404 would come from the type and the record path would be untested.
//
// (e) and (f) also run against ci8Leaky, a handler with the existence oracle
// these cases exist to catch, and must fail there: a check that passes a
// leaky handler proves nothing (the mutation control).
//
// The masked-field filter case (400) needs S10's masks and lands with P3-37
// (D-3-19); it is registered pending.
func recordCI8Paths() []Path {
	db := func(name string, fn func(context.Context, *etIso, *ci8Fixture) error) Path {
		return Path{Name: "records CI-8: " + name, Case: "CI-8", Role: "nava_app", DB: func(ctx context.Context, cl Cluster) error {
			e, err := newETIso(ctx, cl)
			if err != nil {
				return err
			}
			defer e.stop()
			if err := e.cleanup(); err != nil { // rows a killed earlier run left
				return err
			}
			fx, err := ci8Setup(ctx, e)
			if err == nil {
				err = fn(ctx, e, fx)
			}
			if cerr := e.cleanup(); err == nil {
				err = cerr
			}
			return err
		}}
	}
	return []Path{
		db("(a) tenant B's GET of tenant A's record is a random UUID's 404: status, headers and body byte-identical", ci8Get),
		db("(b) tenant B's PATCH and GET history of tenant A's record are a random UUID's 404, byte-identical, and change nothing", ci8PatchAndHistory),
		db("(c) tenant A's PATCH with its record's previous ETag is 412 version_conflict, and changes nothing and audits nothing", ci8StaleIfMatch),
		db("(e) tenant B's PATCH of tenant A's record with A's current or a stale ETag is a random UUID's 404, never 412", ci8CurrentAndStaleETag),
		db("(f) tenant B's PATCH of tenant A's record with no If-Match is a random UUID's 404, not 428", ci8NoIfMatch),
		{Name: "records CI-8: (d) a filter on a masked field is 400", Case: "CI-8", Pending: "P3-37 (S10, PLAN-003 D-3-19)",
			DB: func(context.Context, Cluster) error { return errors.New("pending P3-37: field masks land in S10") }},
	}
}

const ci8Missing = "0194f2c0-0000-7000-8000-0000000c1800"

// ci8Fixture: tenant A's record id of iso_et_rec, at version 2 (so W/"1" is
// stale and W/"2" current), with its audit row count; and tenant B's own
// record of its own iso_et_rec, for the positive controls.
type ci8Fixture struct {
	a, b   string
	audits int
}

func ci8Setup(ctx context.Context, e *etIso) (*ci8Fixture, error) {
	a, err := recCreate(ctx, e)
	if err != nil {
		return nil, err
	}
	if code, body := e.do(ctx, http.MethodPatch, "/api/v1/records/iso_et_rec/"+a, etCallerA, `{"data": {"note": "v2"}}`,
		ci8IfMatch(`W/"1"`)); code != http.StatusOK {
		return nil, fmt.Errorf("control: tenant A patching its record to version 2 = %d %s, want 200", code, body)
	}
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerBAdmin, etCreateBody("iso_et_rec", "ISB", ""), nil); code != http.StatusCreated {
		return nil, fmt.Errorf("control: tenant B creating its own iso_et_rec = %d %s, want 201", code, body)
	}
	b, err := ci8CreateB(ctx, e)
	if err != nil {
		return nil, err
	}
	fx := &ci8Fixture{a: a, b: b}
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE record_id = $1`, a).Scan(&fx.audits); err != nil {
		return nil, err
	}
	// The create and the patch to version 2 each audit once. Without this, a
	// pipeline that stopped auditing would leave 0 and "no new audit row"
	// below would pass without testing anything.
	if fx.audits != 2 {
		return nil, fmt.Errorf("control: tenant A's record has %d audit rows after its create and one patch, want 2", fx.audits)
	}
	return fx, nil
}

func ci8CreateB(ctx context.Context, e *etIso) (string, error) {
	code, body := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_rec", etCallerB, `{"data": {"note": "b"}}`, nil)
	if code != http.StatusCreated {
		return "", fmt.Errorf("control: tenant B creating its own record = %d %s, want 201", code, body)
	}
	var r struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil || r.ID == "" {
		return "", fmt.Errorf("control: tenant B's record has no id: %w: %s", err, body)
	}
	return r.ID, nil
}

func ci8IfMatch(v string) http.Header {
	h := http.Header{"Content-Type": {"application/merge-patch+json"}}
	if v != "" {
		h.Set("If-Match", v)
	}
	return h
}

// ci8Resp is a whole response: status, headers and body.
type ci8Resp struct {
	code int
	hdr  http.Header
	body string
}

// ci8Do is etIso.do through h, keeping the response headers.
func (e *etIso) ci8Do(ctx context.Context, h http.Handler, method, path, token, body string, header http.Header) ci8Resp {
	req := httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" && header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if method == http.MethodPost || method == http.MethodPatch {
		e.keys++
		req.Header.Set("Idempotency-Key", fmt.Sprintf("iso-et-%d", e.keys))
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return ci8Resp{code: rec.Code, hdr: rec.Result().Header, body: rec.Body.String()}
}

// ci8Same: other (tenant A's record) must be missing's 404 (a random UUID),
// byte for byte. Only the per-request X-Request-Id may differ.
func ci8Same(what string, other, missing ci8Resp) error {
	switch {
	case missing.code != http.StatusNotFound:
		return fmt.Errorf("control: %s for a random UUID = %d %s, want 404", what, missing.code, missing.body)
	case other.code != http.StatusNotFound:
		return fmt.Errorf("%s for tenant A's record = %d %s, want the random UUID's 404", what, other.code, other.body)
	case other.body != missing.body:
		return fmt.Errorf("%s: tenant A's record's 404 body differs from the random UUID's:\n%s\n%s", what, other.body, missing.body)
	case strings.Contains(missing.body, `"instance"`):
		return fmt.Errorf("%s: the 404 carries an instance (D-3-10): %s", what, missing.body)
	case missing.hdr.Get("Content-Type") == "":
		return fmt.Errorf("control: %s: the 404 has no Content-Type", what)
	}
	keys := func(h http.Header) []string {
		var ks []string
		for k := range h {
			if k != requestid.Header {
				ks = append(ks, k)
			}
		}
		slices.Sort(ks)
		return ks
	}
	ok, mk := keys(other.hdr), keys(missing.hdr)
	if !slices.Equal(ok, mk) {
		return fmt.Errorf("%s: tenant A's record's 404 has headers %v, the random UUID's %v", what, ok, mk)
	}
	for _, k := range ok {
		if !slices.Equal(other.hdr.Values(k), missing.hdr.Values(k)) {
			return fmt.Errorf("%s: header %s is %q for tenant A's record, %q for the random UUID", what, k, other.hdr.Values(k), missing.hdr.Values(k))
		}
	}
	if missing.hdr.Get("ETag") != "" {
		return fmt.Errorf("%s: the 404 carries an ETag %q", what, missing.hdr.Get("ETag"))
	}
	return nil
}

// ci8Unchanged: tenant A's record is still at version 2 with note v2, and
// has no audit row it did not have.
func ci8Unchanged(ctx context.Context, e *etIso, fx *ci8Fixture) error {
	var (
		v    int64
		note string
		n    int
	)
	// One row with this id in any tenant, and it is tenant A's: a forged
	// write cannot have filed a second copy elsewhere.
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM records WHERE id = $1`, fx.a).Scan(&n); err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%d records have tenant A's record's id across all tenants, want 1", n)
	}
	if err := e.super.QueryRow(ctx, `SELECT version, coalesce(data->>'note', '') FROM records WHERE tenant_id = $1 AND id = $2`,
		TenantA, fx.a).Scan(&v, &note); err != nil {
		return err
	}
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE record_id = $1`, fx.a).Scan(&n); err != nil {
		return err
	}
	if v != 2 || note != "v2" || n != fx.audits {
		return fmt.Errorf("tenant A's record is at version %d, note %q, %d audit rows; want 2, v2, %d", v, note, n, fx.audits)
	}
	return nil
}

func ci8Get(ctx context.Context, e *etIso, fx *ci8Fixture) error {
	// Positive control: tenant B reads its own record with 200.
	if r := e.ci8Do(ctx, e.h, http.MethodGet, "/api/v1/records/iso_et_rec/"+fx.b, etCallerB, "", nil); r.code != http.StatusOK {
		return fmt.Errorf("control: tenant B reading its own record = %d %s, want 200", r.code, r.body)
	}
	other := e.ci8Do(ctx, e.h, http.MethodGet, "/api/v1/records/iso_et_rec/"+fx.a, etCallerB, "", nil)
	missing := e.ci8Do(ctx, e.h, http.MethodGet, "/api/v1/records/iso_et_rec/"+ci8Missing, etCallerB, "", nil)
	return ci8Same("GET", other, missing)
}

func ci8PatchAndHistory(ctx context.Context, e *etIso, fx *ci8Fixture) error {
	if r := e.ci8Do(ctx, e.h, http.MethodGet, "/api/v1/records/iso_et_rec/"+fx.b+"/history", etCallerB, "", nil); r.code != http.StatusOK {
		return fmt.Errorf("control: tenant B reading its own record's history = %d %s, want 200", r.code, r.body)
	}
	for _, c := range []struct {
		method, suffix, body string
		hdr                  http.Header
	}{
		{http.MethodPatch, "", `{"data": {"note": "forged"}}`, ci8IfMatch(`W/"2"`)},
		{http.MethodGet, "/history", "", nil},
	} {
		other := e.ci8Do(ctx, e.h, c.method, "/api/v1/records/iso_et_rec/"+fx.a+c.suffix, etCallerB, c.body, c.hdr)
		missing := e.ci8Do(ctx, e.h, c.method, "/api/v1/records/iso_et_rec/"+ci8Missing+c.suffix, etCallerB, c.body, c.hdr)
		if err := ci8Same(c.method+" "+c.suffix, other, missing); err != nil {
			return err
		}
	}
	return ci8Unchanged(ctx, e, fx)
}

func ci8StaleIfMatch(ctx context.Context, e *etIso, fx *ci8Fixture) error {
	r := e.ci8Do(ctx, e.h, http.MethodPatch, "/api/v1/records/iso_et_rec/"+fx.a, etCallerA, `{"data": {"note": "stale"}}`, ci8IfMatch(`W/"1"`))
	switch {
	case r.code != http.StatusPreconditionFailed:
		return fmt.Errorf("tenant A's PATCH with the previous ETag = %d %s, want 412", r.code, r.body)
	case !strings.Contains(r.body, `"code":"version_conflict"`):
		return fmt.Errorf("tenant A's 412 is not version_conflict: %s", r.body)
	case r.hdr.Get("ETag") != `W/"2"`:
		return fmt.Errorf("tenant A's 412 carries ETag %q, want the current W/\"2\"", r.hdr.Get("ETag"))
	}
	return ci8Unchanged(ctx, e, fx)
}

// ci8ETagCases is (e) through h.
func ci8ETagCases(ctx context.Context, e *etIso, h http.Handler, fx *ci8Fixture) error {
	for _, tag := range []string{`W/"2"`, `W/"1"`} {
		body := `{"data": {"note": "forged"}}`
		other := e.ci8Do(ctx, h, http.MethodPatch, "/api/v1/records/iso_et_rec/"+fx.a, etCallerB, body, ci8IfMatch(tag))
		missing := e.ci8Do(ctx, h, http.MethodPatch, "/api/v1/records/iso_et_rec/"+ci8Missing, etCallerB, body, ci8IfMatch(tag))
		if other.code == http.StatusPreconditionFailed {
			return fmt.Errorf("PATCH If-Match %s: tenant A's record answers 412 to tenant B: an existence oracle: %s", tag, other.body)
		}
		if err := ci8Same("PATCH If-Match "+tag, other, missing); err != nil {
			return err
		}
	}
	return ci8Unchanged(ctx, e, fx)
}

func ci8CurrentAndStaleETag(ctx context.Context, e *etIso, fx *ci8Fixture) error {
	// Positive control: tenant B's own record does answer 412 to a stale ETag,
	// so the 404 below is the tenant scope, not a dead route.
	if r := e.ci8Do(ctx, e.h, http.MethodPatch, "/api/v1/records/iso_et_rec/"+fx.b, etCallerB, `{"data": {"note": "x"}}`, ci8IfMatch(`W/"9"`)); r.code != http.StatusPreconditionFailed {
		return fmt.Errorf("control: tenant B's PATCH of its own record with a stale ETag = %d %s, want 412", r.code, r.body)
	}
	if err := ci8ETagCases(ctx, e, e.h, fx); err != nil {
		return err
	}
	return ci8Mutation(ctx, e, fx, ci8ETagCases)
}

// ci8NoIfMatchCases is (f) through h.
func ci8NoIfMatchCases(ctx context.Context, e *etIso, h http.Handler, fx *ci8Fixture) error {
	body := `{"data": {"note": "forged"}}`
	other := e.ci8Do(ctx, h, http.MethodPatch, "/api/v1/records/iso_et_rec/"+fx.a, etCallerB, body, ci8IfMatch(""))
	missing := e.ci8Do(ctx, h, http.MethodPatch, "/api/v1/records/iso_et_rec/"+ci8Missing, etCallerB, body, ci8IfMatch(""))
	if err := ci8Same("PATCH without If-Match", other, missing); err != nil {
		return err
	}
	return ci8Unchanged(ctx, e, fx)
}

func ci8NoIfMatch(ctx context.Context, e *etIso, fx *ci8Fixture) error {
	// Positive control: tenant B's own record does answer 428.
	if r := e.ci8Do(ctx, e.h, http.MethodPatch, "/api/v1/records/iso_et_rec/"+fx.b, etCallerB, `{"data": {"note": "x"}}`, ci8IfMatch("")); r.code != http.StatusPreconditionRequired {
		return fmt.Errorf("control: tenant B's PATCH of its own record without If-Match = %d %s, want 428", r.code, r.body)
	}
	if err := ci8NoIfMatchCases(ctx, e, e.h, fx); err != nil {
		return err
	}
	return ci8Mutation(ctx, e, fx, ci8NoIfMatchCases)
}

// ci8Mutation runs a case against ci8Leaky, which must fail it.
func ci8Mutation(ctx context.Context, e *etIso, fx *ci8Fixture, run func(context.Context, *etIso, http.Handler, *ci8Fixture) error) error {
	if err := run(ctx, e, ci8Leaky(e), fx); err == nil {
		return errors.New("mutation control: the case passes against a handler with a cross-tenant existence oracle")
	}
	return nil
}

// ci8Leaky is the real handler with the bug (e) and (f) catch: its PATCH
// finds the record by id alone, past row-level security (as a BYPASSRLS
// role or a SECURITY DEFINER lookup would), checks If-Match before the tenant
// (428), and maps an UPDATE … WHERE tenant_id AND id AND version that touched
// 0 rows to 412. It answers before the real handler, so it writes nothing.
func ci8Leaky(e *etIso) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		var version int64
		if r.Method != http.MethodPatch || e.super.QueryRow(r.Context(), `SELECT version FROM records WHERE id::text = $1`, id).Scan(&version) != nil {
			e.h.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/problem+json")
		if r.Header.Get("If-Match") == "" {
			w.WriteHeader(http.StatusPreconditionRequired)
			_, _ = w.Write([]byte(`{"type":"about:blank","title":"Precondition Required","status":428,"code":"if_match_required","detail":"If-Match is required."}`))
			return
		}
		w.Header().Set("ETag", fmt.Sprintf(`W/"%d"`, version))
		w.WriteHeader(http.StatusPreconditionFailed)
		_, _ = fmt.Fprintf(w, `{"type":"about:blank","title":"Precondition Failed","status":412,"code":"version_conflict","detail":"Current ETag: W/\"%d\"."}`, version)
	})
}
