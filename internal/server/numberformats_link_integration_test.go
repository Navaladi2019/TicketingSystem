//go:build integration

// P6-08 (TICK-673): linking a pinned number-format version to an entity type,
// epochs and disjointness inside the definition publish, and "re-publish
// linked types", against a real PostgreSQL through the real handler. Helpers
// are in entitytypes_integration_test.go and numberformats_integration_test.go.

package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// Templates under a type whose prefix is INC. Their literal heads are:
// linkMonthly "INC-", linkYearly "INC-" (the same head, so they clash),
// linkSlash "INC/" (disjoint from both), linkClash "INC" (digits follow: it
// clashes with epoch 0).
const (
	linkMonthly = `{"template":"{prefix}-{yyyy}{MM}-{seq:4}","overflow":"grow"}`
	linkYearly  = `{"template":"{prefix}-{yyyy}-{seq:3}","overflow":"grow"}`
	linkSlash   = `{"template":"{prefix}/{seq:5}","overflow":"grow"}`
	linkClash   = `{"template":"{prefix}{yy}{seq:3}","overflow":"refuse"}`
)

type linkEnv struct {
	*etEnv
	// deny holds the entity type keys the per-type check refuses.
	mu   sync.Mutex
	deny map[string]bool
}

func newLinkEnv(t *testing.T) *linkEnv {
	t.Helper()
	open, now := true, time.Now()
	le := &linkEnv{deny: map[string]bool{}}
	check := func(ctx context.Context, tx pgx.Tx, p identity.Principal, key string) error {
		le.mu.Lock()
		denied := le.deny[key]
		le.mu.Unlock()
		if denied {
			return authz.ErrForbidden
		}
		return authz.RequireTenantAdminTx(ctx, tx, p)
	}
	le.etEnv = newETEnvWith(t, server.WithNumberFormatGate(nfGate(&open, &now)), server.WithTypePublishCheck(check))
	return le
}

var linkSeq atomic.Int64

// linkKey is a key unique within the process, for idempotency keys.
func linkKey(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, linkSeq.Add(1))
}

// createType creates an entity type of tenant A under the prefix.
func (e *linkEnv) createType(t *testing.T, key, prefix string) {
	t.Helper()
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", linkKey("ct"), etBody(key, prefix)); rec.Code != http.StatusCreated {
		t.Fatalf("create type %s = %d: %s", key, rec.Code, rec.Body)
	}
}

// createFormat creates a format as the token and returns its id.
func (e *linkEnv) createFormat(t *testing.T, token, key, document string) string {
	t.Helper()
	rec := e.do(t, http.MethodPost, nfPath, token, linkKey("cf"), fmt.Sprintf(`{"key":%q,"document":%s}`, key, document))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create format %s = %d: %s", key, rec.Code, rec.Body)
	}
	return nfID(t, rec.Body.Bytes())
}

// publishFormat publishes the next version of a format of tenant A.
func (e *linkEnv) publishFormat(t *testing.T, id string, from int, document string) {
	t.Helper()
	rec := e.doIf(t, http.MethodPut, nfPath+"/"+id, "admin-a", linkKey("pf"), fmt.Sprintf(`W/"%d"`, from), fmt.Sprintf(`{"document":%s}`, document))
	if rec.Code != http.StatusOK {
		t.Fatalf("publish format %s = %d: %s", id, rec.Code, rec.Body)
	}
}

// pubBody is a publish body for the etSchema definition, with number_format
// omitted when nf is empty.
func pubBody(nf string) string {
	if nf == "" {
		return fmt.Sprintf(`{"definition": {"schema": %s, %s}}`, etSchema, etDefTail)
	}
	return fmt.Sprintf(`{"definition": {"schema": %s, %s}, "number_format": %s}`, etSchema, etDefTail, nf)
}

func ref(id string, version int) string { return fmt.Sprintf(`{"id":%q,"version":%d}`, id, version) }

func (e *linkEnv) publish(t *testing.T, token, key, ifMatch, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.doIf(t, http.MethodPost, "/api/v1/entity-types/"+key+"/versions", token, linkKey("pub"), ifMatch, body)
}

// pin is the number-format columns of a type's current definition version.
type pin struct {
	version  int
	formatID *uuid.UUID
	fmtVer   *int
	compiled *string
}

func (p pin) epoch(t *testing.T) int {
	t.Helper()
	if p.compiled == nil {
		t.Fatal("no compiled snapshot")
	}
	var c struct{ Epoch int }
	if err := json.Unmarshal([]byte(*p.compiled), &c); err != nil {
		t.Fatal(err)
	}
	return c.Epoch
}

func (e *linkEnv) pinOf(t *testing.T, key string) pin {
	t.Helper()
	var p pin
	if err := e.super.QueryRow(t.Context(), `
SELECT v.version, v.number_format_id, v.number_format_version, v.number_compiled::text
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id AND v.version = d.current_version
WHERE d.tenant_id = $1 AND d.key = $2`, e.a.id, key).Scan(&p.version, &p.formatID, &p.fmtVer, &p.compiled); err != nil {
		t.Fatal(err)
	}
	return p
}

func (e *linkEnv) epochRows(t *testing.T, key string) (heads []string) {
	t.Helper()
	rows, err := e.super.Query(t.Context(), `
SELECT literal_head FROM numbering_epochs
WHERE tenant_id = $1 AND entity_definition_id = (SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2)
ORDER BY epoch`, e.a.id, key)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			t.Fatal(err)
		}
		heads = append(heads, h)
	}
	return heads
}

// sameProblemBody fails unless the two bodies are equal but for `instance`,
// which carries the request id.
func sameProblemBody(t *testing.T, a, b *httptest.ResponseRecorder) {
	t.Helper()
	norm := func(rec *httptest.ResponseRecorder) string {
		var m map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
			t.Fatal(err)
		}
		delete(m, "instance")
		out, _ := json.Marshal(m)
		return string(out)
	}
	if a.Code != b.Code || norm(a) != norm(b) {
		t.Fatalf("bodies differ:\n%d %s\n%d %s", a.Code, a.Body, b.Code, b.Body)
	}
}

// covers: publishEntityTypeVersion 201
func TestLinkPublishPinsFormatAndKeepsItOnAnUnrelatedPublish(t *testing.T) {
	e := newLinkEnv(t)
	e.createType(t, "incident", "INC")
	f := e.createFormat(t, "admin-a", "monthly", linkMonthly)

	rec := e.publish(t, "admin-a", "incident", "", pubBody(ref(f, 1)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("link = %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Version      int `json:"version"`
		NumberFormat struct {
			ID      string `json:"id"`
			Version int    `json:"version"`
		} `json:"number_format"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.NumberFormat.ID != f || body.NumberFormat.Version != 1 || body.Version != 2 {
		t.Fatalf("body = %s (%v)", rec.Body, err)
	}
	p := e.pinOf(t, "incident")
	if p.version != 2 || p.formatID == nil || p.formatID.String() != f || p.fmtVer == nil || *p.fmtVer != 1 || p.epoch(t) != 1 {
		t.Fatalf("pin = %+v", p)
	}
	if heads := e.epochRows(t, "incident"); len(heads) != 1 || heads[0] != "INC-" {
		t.Fatalf("epochs = %v", heads)
	}

	// An unrelated publish (no number_format) keeps the pinned version and its epoch.
	if rec := e.publish(t, "admin-a", "incident", "", pubBody("")); rec.Code != http.StatusCreated {
		t.Fatalf("unrelated publish = %d: %s", rec.Code, rec.Body)
	}
	q := e.pinOf(t, "incident")
	if q.version != 3 || q.formatID == nil || *q.formatID != *p.formatID || *q.fmtVer != 1 || *q.compiled != *p.compiled {
		t.Fatalf("after unrelated publish: %+v, want the pin of %+v", q, p)
	}
	if heads := e.epochRows(t, "incident"); len(heads) != 1 {
		t.Fatalf("an unrelated publish took an epoch: %v", heads)
	}
	// number_format: null returns to the default numbering.
	if rec := e.publish(t, "admin-a", "incident", "", pubBody("null")); rec.Code != http.StatusCreated {
		t.Fatalf("unpin = %d: %s", rec.Code, rec.Body)
	}
	if r := e.pinOf(t, "incident"); r.version != 4 || r.formatID != nil || r.fmtVer != nil || r.compiled != nil {
		t.Fatalf("after unpin: %+v", r)
	}
}

// covers: publishEntityTypeVersion 422
func TestLinkUnknownAndForeignFormatGiveTheSame422AndWriteNothing(t *testing.T) {
	e := newLinkEnv(t)
	e.createType(t, "incident", "INC")
	foreign := e.createFormat(t, "admin-b", "theirs", linkMonthly)
	versions := etVersionCount(t, e.etEnv)

	foreignRec := e.publish(t, "admin-a", "incident", "", pubBody(ref(foreign, 1)))
	unknownRec := e.publish(t, "admin-a", "incident", "", pubBody(ref(uuid.NewString(), 1)))
	wantProblem(t, foreignRec, http.StatusUnprocessableEntity, problem.NumberFormatNotFound)
	sameProblemBody(t, foreignRec, unknownRec)
	// A real format at a version it does not have is the same refusal.
	own := e.createFormat(t, "admin-a", "mine", linkMonthly)
	missingVersion := e.publish(t, "admin-a", "incident", "", pubBody(ref(own, 9)))
	sameProblemBody(t, foreignRec, missingVersion)

	if got := etVersionCount(t, e.etEnv); got != versions {
		t.Errorf("version rows %d -> %d: a refused publish wrote one", versions, got)
	}
	if heads := e.epochRows(t, "incident"); len(heads) != 0 {
		t.Errorf("a refused publish took an epoch: %v", heads)
	}
}

// covers: publishEntityTypeVersion 422
func TestLinkOverlapIsRefusedNamingTheClashingTemplate(t *testing.T) {
	e := newLinkEnv(t)
	e.createType(t, "incident", "INC")
	clash := e.createFormat(t, "admin-a", "clash", linkClash)
	monthly := e.createFormat(t, "admin-a", "monthly", linkMonthly)
	yearly := e.createFormat(t, "admin-a", "yearly", linkYearly)
	versions := etVersionCount(t, e.etEnv)

	// Against epoch 0, which has no registry row: INC + digits clashes with INC + digits.
	p := wantProblem(t, e.publish(t, "admin-a", "incident", "", pubBody(ref(clash, 1))), http.StatusUnprocessableEntity, problem.NumberFormatOverlap)
	if len(p.Errors) != 1 || p.Errors[0].Pointer != "/number_format" || p.Errors[0].Rule != "overlap:{prefix}{seq:7}" {
		t.Errorf("errors = %+v", p.Errors)
	}

	// Against an earlier epoch: the yearly head INC- equals the monthly head.
	if rec := e.publish(t, "admin-a", "incident", "", pubBody(ref(monthly, 1))); rec.Code != http.StatusCreated {
		t.Fatalf("monthly = %d: %s", rec.Code, rec.Body)
	}
	p = wantProblem(t, e.publish(t, "admin-a", "incident", "", pubBody(ref(yearly, 1))), http.StatusUnprocessableEntity, problem.NumberFormatOverlap)
	if len(p.Errors) != 1 || p.Errors[0].Rule != "overlap:{prefix}-{yyyy}{MM}-{seq:4}" {
		t.Errorf("errors = %+v", p.Errors)
	}

	if got := etVersionCount(t, e.etEnv); got != versions+1 {
		t.Errorf("version rows %d -> %d, want one new (monthly)", versions, got)
	}
	if heads := e.epochRows(t, "incident"); len(heads) != 1 {
		t.Errorf("epochs = %v, want only the monthly epoch", heads)
	}
}

// covers: publishEntityTypeVersion 201
func TestLinkReadoptingATemplateReusesItsEpoch(t *testing.T) {
	e := newLinkEnv(t)
	e.createType(t, "incident", "INC")
	monthly := e.createFormat(t, "admin-a", "monthly", linkMonthly)
	slash := e.createFormat(t, "admin-a", "slash", linkSlash)

	steps := []struct {
		nf    string
		epoch int
		rows  int
	}{
		{ref(monthly, 1), 1, 1}, // new epoch
		{ref(slash, 1), 2, 2},   // disjoint from INC-: next epoch
		{ref(monthly, 1), 1, 2}, // re-adopted: epoch 1 again, no new row
		{"null", 0, 2},
	}
	for i, s := range steps {
		if rec := e.publish(t, "admin-a", "incident", "", pubBody(s.nf)); rec.Code != http.StatusCreated {
			t.Fatalf("step %d = %d: %s", i, rec.Code, rec.Body)
		}
		if heads := e.epochRows(t, "incident"); len(heads) != s.rows {
			t.Fatalf("step %d: epochs %v, want %d rows", i, heads, s.rows)
		}
		if p := e.pinOf(t, "incident"); s.nf != "null" && p.epoch(t) != s.epoch {
			t.Fatalf("step %d: epoch %d, want %d", i, p.epoch(t), s.epoch)
		}
	}
}

// covers: publishEntityTypeVersion 412
func TestLinkStaleIfMatchIs412AndWritesNothing(t *testing.T) {
	e := newLinkEnv(t)
	e.createType(t, "incident", "INC")
	f := e.createFormat(t, "admin-a", "monthly", linkMonthly)
	versions := etVersionCount(t, e.etEnv)

	rec := e.publish(t, "admin-a", "incident", `W/"7"`, pubBody(ref(f, 1)))
	wantProblem(t, rec, http.StatusPreconditionFailed, problem.VersionConflict)
	if rec.Header().Get("ETag") != `W/"1"` {
		t.Errorf("ETag = %q, want the current version", rec.Header().Get("ETag"))
	}
	if etVersionCount(t, e.etEnv) != versions || len(e.epochRows(t, "incident")) != 0 {
		t.Error("a stale If-Match wrote something")
	}
	ok := e.publish(t, "admin-a", "incident", `W/"1"`, pubBody(ref(f, 1)))
	if ok.Code != http.StatusCreated || ok.Header().Get("ETag") != `W/"2"` {
		t.Fatalf("matching If-Match = %d %s %s", ok.Code, ok.Header().Get("ETag"), ok.Body)
	}
}

// covers: publishEntityTypeVersion 422
func TestLinkBadShapesAreInvalidDefinitionAndIdempotentReplayWritesOnce(t *testing.T) {
	e := newLinkEnv(t)
	e.createType(t, "incident", "INC")
	f := e.createFormat(t, "admin-a", "monthly", linkMonthly)
	versions := etVersionCount(t, e.etEnv)

	for name, nf := range map[string]string{
		"id not a uuid": `{"id":"nope","version":1}`,
		"version zero":  fmt.Sprintf(`{"id":%q,"version":0}`, f),
		"no version":    fmt.Sprintf(`{"id":%q}`, f),
		"extra member":  fmt.Sprintf(`{"id":%q,"version":1,"x":1}`, f),
		"a string":      `"monthly"`,
	} {
		p := wantProblem(t, e.publish(t, "admin-a", "incident", "", pubBody(nf)), http.StatusUnprocessableEntity, problem.InvalidDefinition)
		if len(p.Errors) != 1 || p.Errors[0].Pointer != "/number_format" {
			t.Errorf("%s: errors = %+v", name, p.Errors)
		}
	}
	// number_format needs a definition to publish.
	p := wantProblem(t, e.publish(t, "admin-a", "incident", "", `{"number_format":`+ref(f, 1)+`}`), http.StatusUnprocessableEntity, problem.InvalidDefinition)
	if len(p.Errors) != 1 || p.Errors[0].Pointer != "/definition" {
		t.Errorf("errors = %+v", p.Errors)
	}
	if etVersionCount(t, e.etEnv) != versions {
		t.Fatal("invalid input wrote a version")
	}

	key := linkKey("replay")
	body := pubBody(ref(f, 1))
	first := e.doIf(t, http.MethodPost, "/api/v1/entity-types/incident/versions", "admin-a", key, "", body)
	second := e.doIf(t, http.MethodPost, "/api/v1/entity-types/incident/versions", "admin-a", key, "", body)
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated || first.Body.String() != second.Body.String() {
		t.Fatalf("replay: %d %s / %d %s", first.Code, first.Body, second.Code, second.Body)
	}
	if got := etVersionCount(t, e.etEnv); got != versions+1 {
		t.Errorf("version rows %d -> %d: a replay wrote again", versions, got)
	}
	if heads := e.epochRows(t, "incident"); len(heads) != 1 {
		t.Errorf("epochs = %v", heads)
	}
}

// Two concurrent publishes of one type with different new templates: the row
// lock serialises them, so each gets an epoch, or the second is refused as an
// overlap against the first's epoch. Never a 500.
func TestLinkConcurrentPublishesNeverGive500(t *testing.T) {
	e := newLinkEnv(t)
	slashID := e.createFormat(t, "admin-a", "slash", linkSlash)
	monthly := e.createFormat(t, "admin-a", "monthly", linkMonthly)
	yearly := e.createFormat(t, "admin-a", "yearly", linkYearly)

	for i := range 20 {
		key := fmt.Sprintf("race_%d", i)
		e.createType(t, key, etPrefix(100+i))
		// Even rounds race two clashing heads (INC-... twice): exactly one wins.
		// Odd rounds race disjoint heads: both win, with epochs 1 and 2.
		a, b, disjoint := monthly, yearly, false
		if i%2 == 1 {
			a, b, disjoint = monthly, slashID, true
		}
		recs := make([]*httptest.ResponseRecorder, 2)
		var wg sync.WaitGroup
		for j, f := range []string{a, b} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				recs[j] = e.publish(t, "admin-a", key, "", pubBody(ref(f, 1)))
			}()
		}
		wg.Wait()
		created, overlaps := 0, 0
		for _, rec := range recs {
			switch {
			case rec.Code == http.StatusCreated:
				created++
			case rec.Code == http.StatusUnprocessableEntity && strings.Contains(rec.Body.String(), string(problem.NumberFormatOverlap)):
				overlaps++
			default:
				t.Fatalf("round %d: %d %s", i, rec.Code, rec.Body)
			}
		}
		heads := e.epochRows(t, key)
		switch {
		case disjoint && (created != 2 || len(heads) != 2):
			t.Fatalf("round %d: disjoint formats: %d created, epochs %v", i, created, heads)
		case !disjoint && (created != 1 || overlaps != 1 || len(heads) != 1):
			t.Fatalf("round %d: clashing formats: %d created, %d overlaps, epochs %v", i, created, overlaps, heads)
		}
	}
}

// republish posts the re-publish of format id to version.
func (e *linkEnv) republish(t *testing.T, token, id string, version int) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, http.MethodPost, nfPath+"/"+id+"/republish-linked", token, linkKey("rl"), fmt.Sprintf(`{"version":%d}`, version))
}

type republishBody struct {
	FormatID uuid.UUID `json:"format_id"`
	Version  int       `json:"version"`
	Results  []struct {
		EntityType string               `json:"entity_type"`
		Status     string               `json:"status"`
		Version    *int                 `json:"version"`
		Code       string               `json:"code"`
		Errors     []problem.FieldError `json:"errors"`
	} `json:"results"`
}

func decodeRepublish(t *testing.T, rec *httptest.ResponseRecorder) republishBody {
	t.Helper()
	var r republishBody
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &r) != nil {
		t.Fatalf("republish = %d: %s", rec.Code, rec.Body)
	}
	return r
}

// covers: republishLinkedTypes 200
func TestRepublishLinkedRunsOnePublishPerTypeAndReportsEach(t *testing.T) {
	e := newLinkEnv(t)
	// Four types on format F v1. After F v2: x and y publish; z is refused by
	// the per-type check (the caller has format write but not publish on z);
	// w holds an earlier epoch whose head clashes with F v2's, so it overlaps.
	for key, prefix := range map[string]string{"tx": "TXA", "ty": "TYA", "tz": "TZA", "tw": "TWA"} {
		e.createType(t, key, prefix)
	}
	f := e.createFormat(t, "admin-a", "monthly", linkSlash) // head PREFIX/
	other := e.createFormat(t, "admin-a", "other", linkMonthly)
	for _, key := range []string{"tx", "ty", "tz", "tw"} {
		if key == "tw" { // an earlier epoch with head TWA-, then F v1 as epoch 2
			if rec := e.publish(t, "admin-a", key, "", pubBody(ref(other, 1))); rec.Code != http.StatusCreated {
				t.Fatalf("tw other = %d: %s", rec.Code, rec.Body)
			}
		}
		if rec := e.publish(t, "admin-a", key, "", pubBody(ref(f, 1))); rec.Code != http.StatusCreated {
			t.Fatalf("link %s = %d: %s", key, rec.Code, rec.Body)
		}
	}
	// A type linked to the other format, and one with no link, are never touched.
	e.createType(t, "tn", "TNA")
	if rec := e.publish(t, "admin-a", "tn", "", pubBody(ref(other, 1))); rec.Code != http.StatusCreated {
		t.Fatalf("tn = %d: %s", rec.Code, rec.Body)
	}
	e.createType(t, "tu", "TUA")

	// F v2: head PREFIX- (clashes with tw's epoch from `other`), a new epoch elsewhere.
	e.publishFormat(t, f, 1, `{"template":"{prefix}-{yyyy}-{seq:3}","overflow":"grow"}`)
	e.mu.Lock()
	e.deny["tz"] = true
	e.mu.Unlock()
	before := map[string]int{}
	for _, key := range []string{"tn", "tu", "tz", "tw"} {
		before[key] = e.pinOf(t, key).version
	}

	got := decodeRepublish(t, e.republish(t, "admin-a", f, 2))
	if got.FormatID.String() != f || got.Version != 2 || len(got.Results) != 4 {
		t.Fatalf("report = %+v", got)
	}
	want := []struct{ key, status, code string }{
		{"tw", "refused", string(problem.NumberFormatOverlap)},
		{"tx", "published", ""},
		{"ty", "published", ""},
		{"tz", "refused", string(problem.Forbidden)},
	}
	for i, w := range want {
		r := got.Results[i]
		if r.EntityType != w.key || r.Status != w.status || r.Code != w.code || (w.status == "published") != (r.Version != nil) {
			t.Errorf("result %d = %+v, want %+v", i, r, w)
		}
	}
	if len(got.Results[0].Errors) != 1 || got.Results[0].Errors[0].Rule != "overlap:{prefix}-{yyyy}{MM}-{seq:4}" {
		t.Errorf("tw errors = %+v", got.Results[0].Errors)
	}
	for _, key := range []string{"tx", "ty"} {
		p := e.pinOf(t, key)
		if p.fmtVer == nil || *p.fmtVer != 2 || p.epoch(t) != 2 {
			t.Errorf("%s pin = %+v, want F v2 on epoch 2", key, p)
		}
	}
	for _, key := range []string{"tz", "tw"} { // refused: nothing written
		if p := e.pinOf(t, key); p.version != before[key] || *p.fmtVer != 1 {
			t.Errorf("%s was changed by a refusal: %+v", key, p)
		}
	}
	for _, key := range []string{"tn", "tu"} {
		if p := e.pinOf(t, key); p.version != before[key] {
			t.Errorf("%s was republished though it is not linked to F", key)
		}
	}
	// Re-running finds only the refused types still pinned to v1.
	again := decodeRepublish(t, e.republish(t, "admin-a", f, 2))
	if len(again.Results) != 2 {
		t.Errorf("second run = %+v, want the two refused types", again.Results)
	}
}

// covers: republishLinkedTypes 200
// covers: republishLinkedTypes 400
// covers: republishLinkedTypes 401
// covers: republishLinkedTypes 403
// covers: republishLinkedTypes 404
// covers: republishLinkedTypes 422
func TestRepublishLinkedRefusals(t *testing.T) {
	e := newLinkEnv(t)
	f := e.createFormat(t, "admin-a", "monthly", linkMonthly)
	foreign := e.createFormat(t, "admin-b", "theirs", linkMonthly)
	path := nfPath + "/" + f + "/republish-linked"

	if rec := e.do(t, http.MethodPost, path, "", linkKey("rl"), `{"version":1}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token = %d", rec.Code)
	}
	wantProblem(t, e.do(t, http.MethodPost, path, "admin-a", "", `{"version":1}`), http.StatusBadRequest, problem.ValidationFailed)
	wantProblem(t, e.republish(t, "user-a", f, 1), http.StatusForbidden, problem.Forbidden)
	wantProblem(t, e.republish(t, "admin-a", f, 9), http.StatusUnprocessableEntity, problem.NumberFormatNotFound)
	// The contract's minimum: 1 refuses a version below 1 before the handler runs.
	p := wantProblem(t, e.do(t, http.MethodPost, path, "admin-a", linkKey("rl"), `{"version":0}`), http.StatusBadRequest, problem.ValidationFailed)
	if len(p.Errors) != 1 || p.Errors[0].Pointer != "/body/version" {
		t.Errorf("errors = %+v", p.Errors)
	}

	// Another tenant's format, a made-up id and a malformed id are the same 404.
	foreignRec := e.republish(t, "admin-a", foreign, 1)
	wantProblem(t, foreignRec, http.StatusNotFound, problem.NotFound)
	sameProblemBody(t, foreignRec, e.republish(t, "admin-a", uuid.NewString(), 1))
	sameProblemBody(t, foreignRec, e.republish(t, "admin-a", "not-a-uuid", 1))

	// No type is linked: an empty report, not an error.
	if r := decodeRepublish(t, e.republish(t, "admin-a", f, 1)); len(r.Results) != 0 {
		t.Errorf("results = %+v", r.Results)
	}
}
