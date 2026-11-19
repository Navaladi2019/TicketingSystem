//go:build integration

// P6-11 (TICK-676): adversarial tests for number formats, links and epochs
// (ADR-005 §3.7 C8-C13, §5), written as attacks, not happy paths. Threat
// model: a caller holding a valid token for one tenant, or a tenant admin
// without the right on one entity type, aiming at rows they must not reach.
// Each test names the guard it breaks against. Helpers are in
// numberformats_integration_test.go, numberformats_link_integration_test.go
// and entitytypes_integration_test.go.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

// attackHeaders is a response's headers without the ones that differ per
// request by design (the request id, tracing). Two responses an attacker
// compares must agree on the rest: status, body and these headers.
func attackHeaders(rec *httptest.ResponseRecorder) map[string]string {
	out := map[string]string{}
	for k, v := range rec.Header() {
		switch http.CanonicalHeaderKey(k) {
		case "X-Request-Id", "Traceparent", "Tracestate", "Date":
			continue
		}
		out[http.CanonicalHeaderKey(k)] = strings.Join(v, ",")
	}
	return out
}

// sameAnswer fails unless foreign and missing are the same answer to an
// attacker: status, body (but for `instance`, the request id) and headers.
func sameAnswer(t *testing.T, what string, foreign, missing *httptest.ResponseRecorder) {
	t.Helper()
	norm := func(rec *httptest.ResponseRecorder) string {
		var m map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
			return rec.Body.String()
		}
		delete(m, "instance")
		b, _ := json.Marshal(m)
		return string(b)
	}
	if foreign.Code != missing.Code || norm(foreign) != norm(missing) {
		t.Errorf("%s: foreign %d %s, missing %d %s", what, foreign.Code, foreign.Body, missing.Code, missing.Body)
	}
	if fh, mh := attackHeaders(foreign), attackHeaders(missing); fmt.Sprint(fh) != fmt.Sprint(mh) {
		t.Errorf("%s: headers differ: foreign %v, missing %v", what, fh, mh)
	}
	if foreign.Header().Get("ETag") != "" {
		t.Errorf("%s: a refused request carries an ETag %q", what, foreign.Header().Get("ETag"))
	}
}

// Attack 1 (C8, I4): tenant B's admin aims every number-format route at
// tenant A's format: read, read a version, publish (with a current, a stale
// and no If-Match), re-publish linked, and link it to B's own type. Every
// answer is the made-up id's answer: status, body and headers. A stale-version
// check that ran before the tenant's format lookup would answer a foreign id
// 412 with its current version, and a made-up one 404: an existence oracle.
// covers: getNumberFormat 404
// covers: publishNumberFormatVersion 404
// covers: republishLinkedTypes 404
func TestNumberFormatAttackForeignIDEveryRouteIsTheMissingAnswer(t *testing.T) {
	e := newLinkEnv(t)
	e.createType(t, "incident", "INC")
	a := e.createFormat(t, "admin-a", "monthly", linkSlash)
	e.publishFormat(t, a, 1, linkMonthly) // A's format is at version 2
	if rec := e.publish(t, "admin-a", "incident", "", pubBody(ref(a, 1))); rec.Code != http.StatusCreated {
		t.Fatalf("link = %d %s", rec.Code, rec.Body)
	}
	before := nfSnapshot(t, e.etEnv)
	pinBefore := e.pinOf(t, "incident")
	// B has a type of its own to link into.
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", linkKey("ct"), etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("B's type = %d %s", rec.Code, rec.Body)
	}
	missing := uuid.NewString()
	doc := `{"document":` + linkYearly + `}`

	type route struct {
		name string
		call func(id string) *httptest.ResponseRecorder
	}
	routes := []route{
		{"GET", func(id string) *httptest.ResponseRecorder {
			return e.do(t, http.MethodGet, nfPath+"/"+id, "admin-b", "", "")
		}},
		{"GET ?version=1", func(id string) *httptest.ResponseRecorder {
			return e.do(t, http.MethodGet, nfPath+"/"+id+"?version=1", "admin-b", "", "")
		}},
		{"GET ?version=2", func(id string) *httptest.ResponseRecorder {
			return e.do(t, http.MethodGet, nfPath+"/"+id+"?version=2", "user-b", "", "")
		}},
		{"PUT current If-Match", func(id string) *httptest.ResponseRecorder {
			return e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-b", linkKey("p"), `W/"2"`, doc)
		}},
		{"PUT stale If-Match", func(id string) *httptest.ResponseRecorder {
			return e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-b", linkKey("p"), `W/"1"`, doc)
		}},
		{"PUT no If-Match", func(id string) *httptest.ResponseRecorder {
			return e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-b", linkKey("p"), "", doc)
		}},
		{"PUT bad body", func(id string) *httptest.ResponseRecorder {
			return e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-b", linkKey("p"), `W/"2"`, `{"document":{"template":"X"}}`)
		}},
		{"republish v2", func(id string) *httptest.ResponseRecorder { return e.republish(t, "admin-b", id, 2) }},
		{"republish v9", func(id string) *httptest.ResponseRecorder { return e.republish(t, "admin-b", id, 9) }},
		{"link v1 into B's type", func(id string) *httptest.ResponseRecorder {
			return e.publish(t, "admin-b", "incident", "", pubBody(ref(id, 1)))
		}},
		{"link v2 into B's type", func(id string) *httptest.ResponseRecorder {
			return e.publish(t, "admin-b", "incident", "", pubBody(ref(id, 2)))
		}},
	}
	for _, r := range routes {
		foreign, made := r.call(a), r.call(missing)
		if foreign.Code < 400 {
			t.Errorf("%s on A's format = %d %s: tenant B reached it", r.name, foreign.Code, foreign.Body)
			continue
		}
		sameAnswer(t, r.name, foreign, made)
	}
	if after := nfSnapshot(t, e.etEnv); after != before {
		t.Errorf("tenant B's requests wrote in tenant A: %v -> %v", before, after)
	}
	if p := e.pinOf(t, "incident"); p.version != pinBefore.version || *p.fmtVer != *pinBefore.fmtVer {
		t.Errorf("A's pin moved: %+v -> %+v", pinBefore, p)
	}
	var bEpochs int
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM numbering_epochs WHERE tenant_id = $1`, e.b.id).Scan(&bEpochs); err != nil || bEpochs != 0 {
		t.Errorf("tenant B holds %d epochs (%v): A's compiled template reached B", bEpochs, err)
	}
}

// Attack 1, timing class: a foreign id and a made-up id take the same path
// (one tenant transaction, one SELECT that RLS leaves empty), so neither is
// an order of magnitude slower. The bound is loose on purpose: it catches a
// second code path (a lookup without RLS, then a refusal), not noise.
func TestNumberFormatAttackForeignIDTimingClassIsTheMissingOne(t *testing.T) {
	e := newLinkEnv(t)
	a := e.createFormat(t, "admin-a", "monthly", linkMonthly)
	const n = 25
	var foreign, missing []time.Duration
	for range n { // interleaved, so drift hits both
		start := time.Now()
		if rec := e.do(t, http.MethodGet, nfPath+"/"+a, "admin-b", "", ""); rec.Code != http.StatusNotFound {
			t.Fatalf("foreign = %d", rec.Code)
		}
		foreign = append(foreign, time.Since(start))
		start = time.Now()
		if rec := e.do(t, http.MethodGet, nfPath+"/"+uuid.NewString(), "admin-b", "", ""); rec.Code != http.StatusNotFound {
			t.Fatalf("missing = %d", rec.Code)
		}
		missing = append(missing, time.Since(start))
	}
	median := func(d []time.Duration) time.Duration {
		s := slices.Clone(d)
		slices.Sort(s)
		return s[len(s)/2]
	}
	f, m := median(foreign), median(missing)
	if f > 3*m+5*time.Millisecond || m > 3*f+5*time.Millisecond {
		t.Errorf("timing class differs: foreign median %v, missing median %v", f, m)
	}
}

// Attack 1, by key: a format key is unique per tenant only, so tenant B
// creating A's key is a 201, not 409 (no uniqueness oracle); B's list never
// shows A's format; a key in the id slot is the one 404.
// covers: listNumberFormats 200
func TestNumberFormatAttackForeignKeyIsNoOracle(t *testing.T) {
	e := newLinkEnv(t)
	a := e.createFormat(t, "admin-a", "visa_monthly", linkMonthly)
	byKey := e.do(t, http.MethodGet, nfPath+"/visa_monthly", "admin-b", "", "")
	sameAnswer(t, "GET by key", byKey, e.do(t, http.MethodGet, nfPath+"/no_such_key", "admin-b", "", ""))

	rec := e.do(t, http.MethodPost, nfPath, "admin-b", linkKey("cf"), `{"key":"visa_monthly","document":`+linkMonthly+`}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("B creating A's key = %d %s: a cross-tenant uniqueness oracle", rec.Code, rec.Body)
	}
	if b := nfID(t, rec.Body.Bytes()); b == a {
		t.Fatalf("B got A's id")
	}
	list := e.do(t, http.MethodGet, nfPath, "admin-b", "", "")
	if list.Code != http.StatusOK || strings.Contains(list.Body.String(), a) {
		t.Errorf("B's list = %d %s: shows A's format", list.Code, list.Body)
	}
	if strings.Count(list.Body.String(), `"key":"visa_monthly"`) != 1 {
		t.Errorf("B's list = %s, want only B's own visa_monthly", list.Body)
	}
}

// Attack 1, list cursor: a next_cursor minted for tenant A's list is not a
// position in tenant B's list. B replaying it is refused, and never pages
// through A's keys.
// covers: listNumberFormats 400
func TestNumberFormatAttackForeignCursorIsRefused(t *testing.T) {
	e := newLinkEnv(t)
	for _, k := range []string{"aa", "bb", "cc"} {
		e.createFormat(t, "admin-a", k, linkMonthly)
	}
	page := e.do(t, http.MethodGet, nfPath+"?limit=1", "admin-a", "", "")
	var body struct {
		NextCursor string `json:"next_cursor"`
	}
	if page.Code != http.StatusOK || json.Unmarshal(page.Body.Bytes(), &body) != nil || body.NextCursor == "" {
		t.Fatalf("A's first page = %d %s", page.Code, page.Body)
	}
	rec := e.do(t, http.MethodGet, nfPath+"?limit=1&cursor="+body.NextCursor, "admin-b", "", "")
	wantProblem(t, rec, http.StatusBadRequest, problem.InvalidCursor)
	if strings.Contains(rec.Body.String(), `"bb"`) || strings.Contains(rec.Body.String(), `"cc"`) {
		t.Errorf("B's replay leaked A's keys: %s", rec.Body)
	}
}

// Attack 2 (I2, C12): every body of the format routes carries forged
// tenant_id, id, version, source, source_ref, actor and created_by fields,
// and a forged compiled form claiming epoch 0 (the default's, which skips
// the M3 gate and the disjointness check). The rows land in A, under
// server-set ids and versions; the compiled form is the server's compile of
// the document; the audit actor is the token's principal with source user
// and no source_ref.
func TestNumberFormatAttackForgedFieldsInEveryFormatBody(t *testing.T) {
	e := newLinkEnv(t)
	e.createType(t, "incident", "INC")
	forgedID := uuid.New()
	def := numbering.Default()
	forgedCompiled, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	forge := fmt.Sprintf(`"tenant_id":%q,"id":%q,"format_id":%q,"version":7,"current_version":9,"epoch":0,`+
		`"compiled":%s,"compiled_sha256":%q,"source":"plugin","source_ref":"evil-client","actor_system":"importer",`+
		`"actor":{"principal_id":%q,"system":"importer"},"actor_principal_id":%q,"created_by":%q,"published_by":%q`,
		e.b.id, forgedID, forgedID, forgedCompiled, def.CompiledSHA256,
		e.a.user.PrincipalID, e.a.user.PrincipalID, e.b.admin.PrincipalID, e.a.user.PrincipalID)

	// create
	rec := e.do(t, http.MethodPost, nfPath, "admin-a", linkKey("cf"), `{`+forge+`,"key":"forged","document":`+linkSlash+`}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", rec.Code, rec.Body)
	}
	id := nfID(t, rec.Body.Bytes())
	if id == forgedID.String() {
		t.Fatalf("the body chose the format id")
	}
	// publish
	if rec := e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-a", linkKey("pf"), `W/"1"`, `{`+forge+`,"document":`+linkMonthly+`}`); rec.Code != http.StatusOK {
		t.Fatalf("publish = %d %s", rec.Code, rec.Body)
	}
	for v, doc := range map[int]string{1: linkSlash, 2: linkMonthly} {
		var (
			tenant, by uuid.UUID
			compiled   string
		)
		if err := e.super.QueryRow(t.Context(), `SELECT tenant_id, created_by, compiled::text FROM number_format_versions WHERE format_id = $1 AND version = $2`, id, v).
			Scan(&tenant, &by, &compiled); err != nil {
			t.Fatalf("version %d: %v", v, err)
		}
		var f numbering.Format
		if err := json.Unmarshal([]byte(doc), &f); err != nil {
			t.Fatal(err)
		}
		want, err := numbering.CompileFormat(f)
		if err != nil {
			t.Fatal(err)
		}
		var got numbering.Compiled
		if err := json.Unmarshal([]byte(compiled), &got); err != nil {
			t.Fatal(err)
		}
		if tenant != e.a.id || by != e.a.admin.PrincipalID {
			t.Errorf("version %d stored under tenant %v by %v, want A by A's admin", v, tenant, by)
		}
		if got.CompiledSHA256 != want.CompiledSHA256 || got.Epoch != want.Epoch || got.Epoch == 0 {
			t.Errorf("version %d compiled = %s: the body's compiled form (epoch 0) reached storage", v, compiled)
		}
	}
	var versions int
	if err := e.super.QueryRow(t.Context(), `SELECT current_version FROM number_formats WHERE id = $1`, id).Scan(&versions); err != nil || versions != 2 {
		t.Errorf("current_version = %d (%v), want 2: the body set it", versions, err)
	}
	e.wantUserAudit(t, "number_format", id, 2)

	// link: forged top-level fields are ignored; inside number_format a forged
	// member is a refusal, so a compiled form or tenant cannot ride in there.
	for _, nf := range []string{
		fmt.Sprintf(`{"id":%q,"version":2,"tenant_id":%q}`, id, e.b.id),
		fmt.Sprintf(`{"id":%q,"version":2,"compiled":%s}`, id, forgedCompiled),
		fmt.Sprintf(`{"id":%q,"version":2,"epoch":0}`, id),
	} {
		rec := e.publish(t, "admin-a", "incident", "", pubBody(nf))
		p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidDefinition)
		if len(p.Errors) != 1 || p.Errors[0].Pointer != "/number_format" {
			t.Errorf("forged member in number_format: errors = %+v", p.Errors)
		}
	}
	body := strings.Replace(pubBody(ref(id, 2)), "{", "{"+forge+",", 1)
	if rec := e.publish(t, "admin-a", "incident", "", body); rec.Code != http.StatusCreated {
		t.Fatalf("link with forged fields = %d %s", rec.Code, rec.Body)
	}
	p := e.pinOf(t, "incident")
	if p.formatID == nil || p.formatID.String() != id || *p.fmtVer != 2 || p.epoch(t) != 1 {
		t.Errorf("pin = %+v epoch %d, want format %s v2 at epoch 1", p, p.epoch(t), id)
	}
	e.wantPublishedBy(t, "incident", p.version)

	// republish: the forged fields ride along with the version.
	e.publishFormat(t, id, 2, linkMonthly)
	rp := e.do(t, http.MethodPost, nfPath+"/"+id+"/republish-linked", "admin-a", linkKey("rl"), `{`+forge+`,"version":3}`)
	if r := decodeRepublish(t, rp); len(r.Results) != 1 || r.Results[0].Status != "published" {
		t.Fatalf("republish = %s", rp.Body)
	}
	e.wantPublishedBy(t, "incident", e.pinOf(t, "incident").version)

	var bRows int
	if err := e.super.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM number_formats WHERE tenant_id = $1)
		+ (SELECT count(*) FROM numbering_epochs WHERE tenant_id = $1)
		+ (SELECT count(*) FROM audit_log WHERE tenant_id = $1)`, e.b.id).Scan(&bRows); err != nil || bRows != 0 {
		t.Errorf("tenant B gained %d rows (%v) from tenant A's forged bodies", bRows, err)
	}
}

// wantUserAudit fails unless the audit row of subject at versionAfter has
// A's admin as actor, no system actor, source user and no source_ref (C12).
func (e *linkEnv) wantUserAudit(t *testing.T, subjectType, subject string, versionAfter int) {
	t.Helper()
	var (
		actor     *uuid.UUID
		system    *string
		source    string
		sourceRef *string
	)
	if err := e.super.QueryRow(t.Context(), `SELECT actor_principal_id, actor_system, source, source_ref FROM audit_log
		WHERE tenant_id = $1 AND subject_type = $2 AND subject_id::text = $3 AND version_after = $4`, e.a.id, subjectType, subject, versionAfter).
		Scan(&actor, &system, &source, &sourceRef); err != nil {
		t.Fatalf("%s %s v%d audit: %v", subjectType, subject, versionAfter, err)
	}
	if actor == nil || *actor != e.a.admin.PrincipalID || system != nil || source != "user" || sourceRef != nil {
		t.Errorf("%s audit actor=%v system=%v source=%s source_ref=%v, want A's admin, user, none", subjectType, actor, system, source, sourceRef)
	}
}

// wantPublishedBy fails unless the type's definition version was published by
// A's admin, with the definition audit row to match.
func (e *linkEnv) wantPublishedBy(t *testing.T, key string, version int) {
	t.Helper()
	var by *uuid.UUID
	var defID string
	if err := e.super.QueryRow(t.Context(), `SELECT v.published_by, d.id::text FROM entity_definitions d
		JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id
		WHERE d.tenant_id = $1 AND d.key = $2 AND v.version = $3`, e.a.id, key, version).Scan(&by, &defID); err != nil {
		t.Fatal(err)
	}
	if by == nil || *by != e.a.admin.PrincipalID {
		t.Errorf("%s v%d published_by = %v, want A's admin %v", key, version, by, e.a.admin.PrincipalID)
	}
	e.wantUserAudit(t, "entity_definition", defID, version)
}

// Attack 3 (C11 (5), C13): a tenant admin who may publish formats but not
// the definition of type X. "Re-publish linked types" refuses X per type and
// still publishes the type they may publish. Today the per-type right is the
// TypePublishCheck seam (WithTypePublishCheck); in production both rights are
// "tenant admin".
// covers: republishLinkedTypes 200
func TestNumberFormatAttackRepublishRefusesPerType(t *testing.T) {
	e := newLinkEnv(t)
	e.createType(t, "visa", "VISA")
	e.createType(t, "incident", "INC")
	f := e.createFormat(t, "admin-a", "slash", linkSlash)
	for _, k := range []string{"visa", "incident"} {
		if rec := e.publish(t, "admin-a", k, "", pubBody(ref(f, 1))); rec.Code != http.StatusCreated {
			t.Fatalf("link %s = %d %s", k, rec.Code, rec.Body)
		}
	}
	e.publishFormat(t, f, 1, linkMonthly)
	visaBefore := e.pinOf(t, "visa")

	e.mu.Lock()
	e.deny["visa"] = true
	e.mu.Unlock()
	r := decodeRepublish(t, e.republish(t, "admin-a", f, 2))
	got := map[string]string{}
	for _, l := range r.Results {
		got[l.EntityType] = l.Status + "/" + l.Code
	}
	if got["visa"] != "refused/"+string(problem.Forbidden) || got["incident"] != "published/" {
		t.Errorf("report = %v, want visa refused forbidden and incident published", got)
	}
	if p := e.pinOf(t, "visa"); p.version != visaBefore.version || *p.fmtVer != 1 {
		t.Errorf("visa pin moved without the right: %+v -> %+v", visaBefore, p)
	}
}

// Attack 3, the direct path (C11 (5)): the same caller links X, bumps X's
// pinned version and unpins X through POST /entity-types/X/versions. Each is
// a definition publish of X, so each must be refused by the same per-type
// right as the re-publish loop.
//
// FINDING TICK-676-F1 (will bite at scale, Medium): PublishEntityTypeVersion
// checks authz.RequireTenantAdminTx, not canPublishType, so a per-type right
// plugged into the TypePublishCheck seam gates only "re-publish linked
// types", and the direct route links, bumps and unpins X anyway. Not
// exploitable today: the seam's production value is RequireTenantAdminTx, the
// same check. The fix routes the definition publish (and a mode-only change)
// through canPublishType; this test then runs. TODO(TODO-S18-01).
func TestNumberFormatAttackDirectLinkRefusesPerType(t *testing.T) {
	t.Skip("TODO-S18-01: the definition publish route does not use the TypePublishCheck seam (finding TICK-676-F1)")
	e := newLinkEnv(t)
	e.createType(t, "visa", "VISA")
	f := e.createFormat(t, "admin-a", "slash", linkSlash)
	g := e.createFormat(t, "admin-a", "monthly", linkMonthly)
	e.publishFormat(t, f, 1, `{"template":"{prefix}/{seq:6}","overflow":"grow"}`)
	if rec := e.publish(t, "admin-a", "visa", "", pubBody(ref(f, 1))); rec.Code != http.StatusCreated {
		t.Fatalf("link = %d %s", rec.Code, rec.Body)
	}
	before := e.pinOf(t, "visa")
	epochs := e.epochRows(t, "visa")

	e.mu.Lock()
	e.deny["visa"] = true
	e.mu.Unlock()
	for name, nf := range map[string]string{"bump": ref(f, 2), "relink": ref(g, 1), "unpin": "null"} {
		wantProblem(t, e.publish(t, "admin-a", "visa", "", pubBody(nf)), http.StatusForbidden, problem.Forbidden)
		if p := e.pinOf(t, "visa"); p.version != before.version {
			t.Errorf("%s: visa moved to v%d without the right", name, p.version)
		}
	}
	if got := e.epochRows(t, "visa"); !slices.Equal(got, epochs) {
		t.Errorf("epochs %v -> %v without the right", epochs, got)
	}
}

// Attack 4 (C10): many link publishes race on one type, two of them per
// template (the same compiled hash) and some of them clashing. Never a 500;
// never two epoch rows for one epoch or one template; every accepted pin's
// epoch is the registry's epoch for its template.
func TestNumberFormatAttackLinkRaceNeverForksAnEpoch(t *testing.T) {
	e := newLinkEnv(t)
	templates := []string{
		`{"template":"{prefix}/{seq:5}","overflow":"grow"}`,
		`{"template":"{prefix}.{seq:5}","overflow":"grow"}`,
		`{"template":"{prefix}_{seq:5}","overflow":"grow"}`,
		`{"template":"{prefix}-{yyyy}-{seq:4}","overflow":"grow"}`,
		`{"template":"{prefix}-{yy}{MM}-{seq:4}","overflow":"grow"}`, // clashes with the one above: head INC-
	}
	var formats []string
	for i, doc := range templates {
		// Two formats per template: a second format id with the same template
		// must reuse the epoch, never take a second one.
		formats = append(formats, e.createFormat(t, "admin-a", fmt.Sprintf("t%d_a", i), doc), e.createFormat(t, "admin-a", fmt.Sprintf("t%d_b", i), doc))
	}
	for round := range 5 {
		key := fmt.Sprintf("race_%d", round)
		e.createType(t, key, etPrefix(200+round))
		recs := make([]*httptest.ResponseRecorder, len(formats))
		var wg sync.WaitGroup
		for i, f := range formats {
			wg.Add(1)
			go func() {
				defer wg.Done()
				recs[i] = e.publish(t, "admin-a", key, "", pubBody(ref(f, 1)))
			}()
		}
		wg.Wait()
		for i, rec := range recs {
			ok := rec.Code == http.StatusCreated ||
				(rec.Code == http.StatusUnprocessableEntity && strings.Contains(rec.Body.String(), string(problem.NumberFormatOverlap)))
			if !ok {
				t.Fatalf("round %d format %d: %d %s", round, i, rec.Code, rec.Body)
			}
		}
		var rows, distinctEpochs, distinctHashes, distinctHeads int
		if err := e.super.QueryRow(t.Context(), `
SELECT count(*), count(DISTINCT epoch), count(DISTINCT compiled_sha256), count(DISTINCT literal_head)
FROM numbering_epochs WHERE tenant_id = $1
  AND entity_definition_id = (SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2)`, e.a.id, key).
			Scan(&rows, &distinctEpochs, &distinctHashes, &distinctHeads); err != nil {
			t.Fatal(err)
		}
		if rows != distinctEpochs || rows != distinctHashes || rows != distinctHeads || rows != 4 {
			t.Fatalf("round %d: %d epoch rows, %d epochs, %d hashes, %d heads", round, rows, distinctEpochs, distinctHashes, distinctHeads)
		}
		// Every version a 201 wrote pins the registry's epoch of its template.
		var bad int
		if err := e.super.QueryRow(t.Context(), `
SELECT count(*) FROM entity_definition_versions v
JOIN entity_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id
LEFT JOIN numbering_epochs n ON n.tenant_id = v.tenant_id AND n.entity_definition_id = v.definition_id
  AND encode(n.compiled_sha256, 'hex') = v.number_compiled->>'CompiledSHA256'
  AND n.epoch = (v.number_compiled->>'Epoch')::int
WHERE d.tenant_id = $1 AND d.key = $2 AND v.number_compiled IS NOT NULL AND n.epoch IS NULL`, e.a.id, key).Scan(&bad); err != nil {
			t.Fatal(err)
		}
		if bad != 0 {
			t.Fatalf("round %d: %d versions pin an epoch the registry does not hold for their template", round, bad)
		}
	}
}

// Attack 5 (§3.3, TA-1): try to get two accepted epochs of one type that
// render the same string, through the API. Every variant changes only what
// the literal head does not see (overflow, width, date parts, a time zone,
// a digit straight after the prefix), so it is refused as an overlap and
// writes nothing. The tenant's time zone changing between the two publishes
// does not open a gap either: the zone is not part of the template's
// identity, so the same template reuses its epoch and its counters.
// covers: publishEntityTypeVersion 422
func TestNumberFormatAttackNoTwoEpochsRenderOneString(t *testing.T) {
	e := newLinkEnv(t)
	e.createType(t, "visa", "VISA")
	base := e.createFormat(t, "admin-a", "base", `{"template":"{prefix}-{yyyy}{MM}-{seq:4}","overflow":"grow"}`)
	if rec := e.publish(t, "admin-a", "visa", "", pubBody(ref(base, 1))); rec.Code != http.StatusCreated {
		t.Fatalf("link base = %d %s", rec.Code, rec.Body)
	}
	heads := e.epochRows(t, "visa")

	attacks := map[string]string{
		"same template, overflow refuse":   `{"template":"{prefix}-{yyyy}{MM}-{seq:4}","overflow":"refuse"}`,
		"same head, wider seq":             `{"template":"{prefix}-{yyyy}{MM}-{seq:5}","overflow":"grow"}`,
		"same head, no date":               `{"template":"{prefix}-{seq:10}","overflow":"refuse"}`,
		"same head, other date parts":      `{"template":"{prefix}-{yy}{MM}{dd}-{seq:4}","overflow":"grow"}`,
		"same head, other time zone":       `{"template":"{prefix}-{yyyy}{MM}-{seq:5}","overflow":"grow","time_zone":"Pacific/Kiritimati"}`,
		"head is a prefix, then a digit":   `{"template":"{prefix}-2{seq:4}","overflow":"refuse"}`,
		"default head, overflow refuse":    `{"template":"{prefix}{seq:7}","overflow":"refuse"}`,
		"default head, digit after prefix": `{"template":"{prefix}0{seq:6}","overflow":"refuse"}`,
		"default head, date after prefix":  `{"template":"{prefix}{yy}{seq:5}","overflow":"refuse"}`,
	}
	names := make([]string, 0, len(attacks))
	for n := range attacks {
		names = append(names, n)
	}
	sort.Strings(names)
	versions := etVersionCount(t, e.etEnv)
	for i, name := range names {
		f := e.createFormat(t, "admin-a", fmt.Sprintf("atk_%d", i), attacks[name])
		rec := e.publish(t, "admin-a", "visa", "", pubBody(ref(f, 1)))
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), string(problem.NumberFormatOverlap)) {
			t.Errorf("%s: %d %s, want 422 %s", name, rec.Code, rec.Body, problem.NumberFormatOverlap)
		}
	}
	if got := etVersionCount(t, e.etEnv); got != versions {
		t.Errorf("refused links wrote %d version rows", got-versions)
	}
	if got := e.epochRows(t, "visa"); !slices.Equal(got, heads) {
		t.Errorf("epochs %v -> %v: a clashing template got an epoch", heads, got)
	}

	// The tenant moves its time zone, then links the same template under
	// another format id and with a zone of its own: the same epoch, no new row.
	if rec := e.do(t, http.MethodPut, tzPath, "admin-a", linkKey("tz"), `{"time_zone":"Pacific/Kiritimati"}`); rec.Code != http.StatusOK {
		t.Fatalf("time zone = %d %s", rec.Code, rec.Body)
	}
	twin := e.createFormat(t, "admin-a", "twin", `{"template":"{prefix}-{yyyy}{MM}-{seq:4}","overflow":"grow","time_zone":"America/Adak"}`)
	if rec := e.publish(t, "admin-a", "visa", "", pubBody(ref(twin, 1))); rec.Code != http.StatusCreated {
		t.Fatalf("link twin = %d %s", rec.Code, rec.Body)
	}
	if p := e.pinOf(t, "visa"); p.epoch(t) != 1 {
		t.Errorf("the same template under another zone took epoch %d, want 1 (its counters)", p.epoch(t))
	}
	if got := e.epochRows(t, "visa"); !slices.Equal(got, heads) {
		t.Errorf("epochs %v -> %v after a zone change", heads, got)
	}
}
