//go:build integration

// The I5 route rows (TICK-900, Epic 06 TICK-9): the data-filter agreement,
// the list route and the masked-field oracle, driven through the real
// nava-core handler (etIso) over the suite's database as nava_app. The world
// is i5_route_gen_test.go's; the oracle is the Go single-record path:
// authz.Allows on the record's row, and the Go filter (i5rQuery.matches) on
// the record as the read path serves it, defaults applied (an admin GET under
// the allow-all ACL, checked against the generator's own defaulted record
// first).

package isolation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const (
	i5RowMasked       = "I5 masked-field oracle: two records that differ only in a field masked for the caller are the same to filter, sort, total and paging (Epic 06 AC)"
	i5RowFilterMutant = "I5 mutant %s: the suite catches a broken data-filter compiler"
	i5rPath           = "/api/v1/records/" + i5rKey
	// i5rBudget bounds the route rows: setup, every case and the mutants
	// (about 40 s on an idle host, 90 s on a loaded one).
	i5rBudget = 240 * time.Second
	// i5rMutantCases is how many cases each filter mutant runs on.
	i5rMutantCases = 12
	// i5rMinMatchedTenths is the floor on the share of filter requests that
	// match at least one record, in tenths. The generator reaches 11-23 %
	// (seeds 20261109 / 7 / 4242 / 31337: 82 / 145 / 163 / 158 of 720), so
	// 10 % passes every seed with a correct compiler while a compiler or
	// generator that matches (almost) nothing still turns the row red.
	i5rMinMatchedTenths = 1
)

// i5rNoRateLimit lifts the per-tenant limiter (50 rps by default) off the
// route rows' handlers: they send thousands of requests as one tenant, and a
// fast runner hit 429 rate_limited and broke the run (TICK-912 CI). The rows
// test filter agreement, not the limiter.
var i5rNoRateLimit = server.WithRateLimit(config.RateLimit{Rate: 1e6, Burst: 1 << 20})

// i5FilterMutants are broken data-filter compilers for the list route
// (server.WithListDataTerm). arg is the bound JSON document {prop: value}.
func i5FilterMutants() map[string]func(arg string) string {
	return map[string]func(arg string) string{
		// type_blind compares the text of the values, so the string "1" of a
		// v1 row matches the integer filter 1.
		"type_blind": func(arg string) string {
			return "EXISTS (SELECT 1 FROM jsonb_each(" + arg + ") f WHERE r.data->>f.key = f.value#>>'{}')"
		},
		// null_matches lets a JSON null match any value.
		"null_matches": func(arg string) string {
			return "(r.data @> " + arg + " OR EXISTS (SELECT 1 FROM jsonb_each(" + arg + ") f WHERE r.data->f.key = 'null'::jsonb))"
		},
	}
}

func i5FilterMutantNames() []string {
	var names []string
	for n := range i5FilterMutants() {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// i5rRun is one route run: the harness, the generated world and the oracle
// views of every record.
type i5rRun struct {
	e     *etIso
	seed  uint64
	w     *i5rWorld
	cases []i5rCase
	all   []i5rRecord // w.Records and the twins
	twins [2]i5rRecord
	views map[uuid.UUID]map[string]any
	// mutants are one harness per filter mutant; token registers a caller
	// in every harness's map, which its handler resolves tokens with.
	mutants map[string]*etIso
}

func (run *i5rRun) token(name string, p identity.Principal) string {
	run.e.tokens[name] = p
	for _, m := range run.mutants {
		m.tokens[name] = p
	}
	return name
}

// i5rCleanup deletes everything a route run writes, as the superuser.
func i5rCleanup(ctx context.Context, e *etIso) error {
	if err := s10Cleanup(ctx, e); err != nil {
		return err
	}
	if err := e.cleanup(); err != nil {
		return err
	}
	tx, err := e.super.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, q := range []string{
		`SET LOCAL session_replication_role = replica`,
		`DELETE FROM principals WHERE keycloak_user_id LIKE 'kc-iso-i5r-%'`,
		`DELETE FROM groups WHERE keycloak_group_id LIKE 'kc-iso-i5r-%'`,
	} {
		if _, err := tx.Exec(ctx, q); err != nil {
			return fmt.Errorf("i5r cleanup: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// i5rSetup builds the route run's world: the type at v1, v2 published, the
// pool's extra principals and groups, and the records and shares written past
// the API (as rows of an older version would be).
func i5rSetup(ctx context.Context, cl Cluster, seed uint64) (*i5rRun, func() error, error) {
	e, err := newETIso(ctx, cl, i5rNoRateLimit)
	if err != nil {
		return nil, nil, err
	}
	run := &i5rRun{e: e, seed: seed, mutants: map[string]*etIso{}}
	stop := func() error {
		err := i5rCleanup(ctx, e)
		e.stop()
		for _, m := range run.mutants {
			m.stop()
		}
		return err
	}
	for name, term := range i5FilterMutants() {
		m, err := newETIso(ctx, cl, i5rNoRateLimit, server.WithListDataTerm(term))
		if err != nil {
			return nil, stop, err
		}
		run.mutants[name] = m
	}
	if err := i5rCleanup(ctx, e); err != nil { // rows a killed earlier run left
		return nil, stop, err
	}
	run.w, run.cases = i5rGenerate(seed)
	a, b := i5rTwins(run.w)
	run.twins = [2]i5rRecord{a, b}
	run.all = append(slices.Clone(run.w.Records), a, b)

	body := fmt.Sprintf(`{"key": %q, "name": "Iso I5 route", "number_prefix": %q, "definition": %s}`, i5rKey, i5rPrefix, i5rDefinition(1))
	if code, out := e.do(ctx, http.MethodPost, "/api/v1/entity-types", s10Admin, body, nil); code != http.StatusCreated {
		return nil, stop, fmt.Errorf("control: creating %s = %d %s, want 201", i5rKey, code, out)
	}
	if code, out := e.do(ctx, http.MethodPost, "/api/v1/entity-types/"+i5rKey+"/versions", s10Admin, `{"definition": `+i5rDefinition(2)+`}`, nil); code != http.StatusCreated {
		return nil, stop, fmt.Errorf("control: publishing v2 of %s (prio's default) = %d %s, want 201", i5rKey, code, out)
	}
	if err := i5rSeed(ctx, e, run.all); err != nil {
		return nil, stop, fmt.Errorf("seed: %w", err)
	}
	// The oracle's view of each record is the read path's: GET as the admin
	// under the allow-all ACL the harness published, which must be the
	// generator's defaulted record (v2's prio default on every row that has no
	// prio, a JSON null kept, a v1 string kept).
	run.views = map[uuid.UUID]map[string]any{}
	for _, rec := range run.all {
		r := e.ci8Do(ctx, e.h, http.MethodGet, i5rPath+"/"+rec.ID.String(), s10Admin, "", nil)
		var got struct {
			Data map[string]any `json:"data"`
		}
		dec := json.NewDecoder(strings.NewReader(r.body))
		dec.UseNumber()
		if r.code != http.StatusOK || dec.Decode(&got) != nil {
			return nil, stop, fmt.Errorf("control: the admin's GET of %s = %d %s, want 200", rec.ID, r.code, r.body)
		}
		gj, _ := json.Marshal(got.Data)
		wj, _ := json.Marshal(rec.defaulted())
		if !bytes.Equal(gj, wj) {
			return nil, stop, fmt.Errorf("control: the read path serves %s (schema_version %d, stored %s) as %s, want %s",
				rec.ID, rec.SchemaVersion, i5rJSON(rec.Data), gj, wj)
		}
		run.views[rec.ID] = got.Data
	}
	return run, stop, nil
}

func i5rJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

// i5rSeed writes the pool's extra principals and groups, the records and
// their shares as the superuser, past every trigger: the rows the read path
// must serve as a row of an older version, whatever wrote it.
func i5rSeed(ctx context.Context, e *etIso, recs []i5rRecord) error {
	tx, err := e.super.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var def uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, TenantA, i5rKey).Scan(&def); err != nil {
		return err
	}
	b := &pgx.Batch{}
	b.Queue(`SET LOCAL session_replication_role = replica`)
	for _, id := range i5rExtraUsers() {
		b.Queue(`INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, federated_links, synced_at) VALUES ($1, $2, $3, true, '[]', now())`,
			TenantA, id, "kc-iso-i5r-"+id.String())
	}
	for _, id := range i5rExtraGroups() {
		b.Queue(`INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)`,
			TenantA, id, "kc-iso-i5r-"+id.String(), "/t/"+TenantA+"/iso-i5r-"+id.String())
	}
	for i, rec := range recs {
		data, err := json.Marshal(rec.Data)
		if err != nil {
			return err
		}
		b.Queue(`INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
				requester_id, assignee_id, assignment_group_id, data, created_at, updated_at)
			VALUES ($1, $2, 1, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, now() - make_interval(secs => $11), now() - make_interval(secs => $11))`,
			TenantA, rec.ID, def, rec.SchemaVersion, fmt.Sprintf("%s%07d", i5rPrefix, i+1), rec.State,
			rec.Requester, rec.Assignee, rec.Group, string(data), rec.Age)
		for _, s := range rec.Shares {
			col := "user_id"
			if s.Kind == "group" {
				col = "group_id"
			}
			b.Queue(`INSERT INTO record_shares (tenant_id, record_id, principal_kind, `+col+`) VALUES ($1, $2, $3, $4)`, TenantA, rec.ID, s.Kind, s.ID)
		}
	}
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// publish makes c's rules and masks tenant A's current ACL of the type and
// returns the policy as the route reads it.
func (run *i5rRun) publish(ctx context.Context, c i5rCase) (*authz.Policy, error) {
	var pol *authz.Policy
	err := tenancy.InTenantTx(ctx, run.e.pool, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
		if _, err := acltest.PublishPolicy(ctx, tx, i5rKey, c.Rules, c.Masks); err != nil {
			return err
		}
		var err error
		pol, err = authz.PolicyStore{}.Current(ctx, tx, i5rKey)
		return err
	})
	return pol, err
}

// i5rListPage is a list response.
type i5rListPage struct {
	Items      []map[string]any `json:"items"`
	Total      *int             `json:"total"`
	NextCursor *string          `json:"next_cursor"`
}

func (p i5rListPage) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, it := range p.Items {
		id, _ := it["id"].(string)
		out = append(out, id)
	}
	return out
}

func i5rList(ctx context.Context, h *etIso, token, query string) (ci8Resp, i5rListPage, error) {
	r := h.ci8Do(ctx, h.h, http.MethodGet, i5rPath+"?"+query, token, "", nil)
	var p i5rListPage
	if r.code != http.StatusOK {
		return r, p, nil
	}
	dec := json.NewDecoder(strings.NewReader(r.body))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil || p.Total == nil {
		return r, p, fmt.Errorf("?%s: not a page: %v: %s", query, err, r.body)
	}
	return r, p, nil
}

// i5rCounts is how much the list route and masked-field rows discriminated.
type i5rCounts struct {
	pages, proper, masked, open int
}

// i5rFilterResult is one filter request's verdict: the records on which the
// route and the Go side differ.
type i5rFilterResult struct {
	checks, matched, masked, defaultHits int
	// diffs keys each disagreement as case/principal/query/record.
	diffs map[string]string
	first string
	// causes counts the disagreements by the shape of the record's field
	// that the request filters on (the first field whose shape is not a
	// plain value, or "a present value").
	causes map[string]int
}

// filters runs every filter request of every case against h and compares it
// with the Go side.
func (run *i5rRun) filters(ctx context.Context, h *etIso, ci int, c i5rCase, pol *authz.Policy, res *i5rFilterResult) error {
	for k, p := range c.Principals {
		tok := run.token(fmt.Sprintf("i5r-c%d-p%d", ci, k), p)
		for qi, q := range c.Queries[k] {
			query := q.encode() + "&limit=200"
			resp, page, err := i5rList(ctx, h, tok, query)
			if err != nil {
				return err
			}
			res.checks++
			where := fmt.Sprintf("seed %d, policy #%d %s, principal #%d %s, ?%s", run.seed, ci, i5rJSON(pol), k, i5rJSON(p), query)
			if slices.Contains(q.names(), "cost") && !authz.Readable(pol, p, "cost") {
				res.masked++
				if _, code, perr := s10Problem(resp); perr != nil || resp.code != http.StatusBadRequest || code != "masked_field" {
					return fmt.Errorf("%s: a filter on cost, masked from the caller = %d %s, want 400 masked_field", where, resp.code, resp.body)
				}
				continue
			}
			if resp.code != http.StatusOK {
				return fmt.Errorf("%s = %d %s, want 200", where, resp.code, resp.body)
			}
			got := map[string]bool{}
			for _, id := range page.ids() {
				got[id] = true
			}
			want := 0
			for _, rec := range run.all {
				goOK := false
				if _, _, ok := authz.Allows(pol, p, authz.OpRead, rec.row(p)); ok && q.matches(run.views[rec.ID]) {
					goOK = true
					want++
					if rec.Shapes["prio"] == i5rDefaulted && slices.ContainsFunc(q, func(f i5rFilter) bool { return f.Prop == "prio" }) {
						res.defaultHits++
					}
				}
				if sqlOK := got[rec.ID.String()]; sqlOK != goOK {
					side := "the Go check admits it and the list does not"
					if sqlOK {
						side = "the list returns it and the Go check does not"
					}
					d := fmt.Sprintf("%s: record %s (schema_version %d, shapes %s, stored %s, read as %s): %s",
						where, rec.ID, rec.SchemaVersion, i5rJSON(rec.Shapes), i5rJSON(rec.Data), i5rJSON(run.views[rec.ID]), side)
					res.diffs[fmt.Sprintf("%d/%d/%d/%s", ci, k, qi, rec.ID)] = d
					cause := "a present value"
					for _, f := range q {
						if sh, ok := rec.Shapes[f.Prop]; ok && sh != i5rPresent {
							cause = f.Prop + " " + sh
							break
						}
					}
					if res.causes == nil {
						res.causes = map[string]int{}
					}
					res.causes[cause]++
					if res.first == "" {
						res.first = d
					}
				}
			}
			if want > 0 {
				res.matched++
			}
			if *page.Total != len(page.Items) || *page.Total != len(got) {
				return fmt.Errorf("%s: total %d for %d items (%d distinct) on one page", where, *page.Total, len(page.Items), len(got))
			}
			if !authz.Readable(pol, p, "cost") {
				for _, it := range page.Items {
					if d, _ := it["data"].(map[string]any); d != nil {
						if _, ok := d["cost"]; ok {
							return fmt.Errorf("%s: an item carries cost, masked from the caller: %s", where, i5rJSON(it))
						}
					}
				}
			}
		}
	}
	return nil
}

// lists pages through every case's unfiltered lists: a record the caller
// cannot read is on no page, every readable one is on exactly one, and every
// page's total is the number of readable records.
func (run *i5rRun) lists(ctx context.Context, ci int, c i5rCase, pol *authz.Policy, st *i5rCounts) error {
	for k, p := range c.Principals {
		tok := run.token(fmt.Sprintf("i5r-c%d-p%d", ci, k), p)
		readable := map[string]bool{}
		for _, rec := range run.all {
			if _, _, ok := authz.Allows(pol, p, authz.OpRead, rec.row(p)); ok {
				readable[rec.ID.String()] = true
			}
		}
		if len(readable) > 0 && len(readable) < len(run.all) {
			st.proper++
		}
		for _, spec := range c.Pages[k] {
			base := url.Values{"limit": {fmt.Sprint(spec.Limit)}}
			if spec.Sort != "" {
				base.Set("sort", spec.Sort)
			}
			seen := map[string]bool{}
			cursor := ""
			for n := 0; ; n++ {
				q := url.Values{}
				for key, v := range base {
					q[key] = v
				}
				if cursor != "" {
					q.Set("cursor", cursor)
				}
				where := fmt.Sprintf("seed %d, policy #%d %s, principal #%d %s, ?%s (page %d)", run.seed, ci, i5rJSON(pol), k, i5rJSON(p), base.Encode(), n+1)
				resp, page, err := i5rList(ctx, run.e, tok, q.Encode())
				if err != nil {
					return err
				}
				if resp.code != http.StatusOK {
					return fmt.Errorf("%s = %d %s, want 200", where, resp.code, resp.body)
				}
				st.pages++
				if *page.Total != len(readable) {
					return fmt.Errorf("%s: total %d, want %d readable records", where, *page.Total, len(readable))
				}
				for _, id := range page.ids() {
					switch {
					case !readable[id]:
						return fmt.Errorf("%s: record %s is on the page and Allows refuses it", where, id)
					case seen[id]:
						return fmt.Errorf("%s: record %s is on two pages", where, id)
					}
					seen[id] = true
				}
				if page.NextCursor == nil {
					break
				}
				if n > len(run.all) {
					return fmt.Errorf("%s: the cursor never ends", where)
				}
				cursor = *page.NextCursor
			}
			if len(seen) != len(readable) {
				return fmt.Errorf("seed %d, policy #%d, principal #%d, %s: the pages hold %d of the %d readable records",
					run.seed, ci, k, base.Encode(), len(seen), len(readable))
			}
		}
	}
	return nil
}

// masked is the masked-field oracle: for a caller who reads both twins and
// may not read cost, every request naming cost is the same 400 whichever
// twin's value it names, or one neither has; and every request that does not
// name cost treats the twins alike: both or neither, same total, the same
// item but for id and number, and one page each at limit 1.
func (run *i5rRun) masked(ctx context.Context, ci int, c i5rCase, pol *authz.Policy, st *i5rCounts) error {
	a := run.twins[0]
	for k, p := range c.Principals {
		_, _, readsA := authz.Allows(pol, p, authz.OpRead, a.row(p))
		if !readsA {
			continue
		}
		tok := run.token(fmt.Sprintf("i5r-c%d-p%d", ci, k), p)
		where := fmt.Sprintf("seed %d, policy #%d %s, principal #%d %s", run.seed, ci, i5rJSON(pol), k, i5rJSON(p))
		if authz.Readable(pol, p, "cost") {
			// The positive control: a caller the mask admits tells the twins
			// apart, so the masked caller's sameness is the mask's doing.
			_, page, err := i5rList(ctx, run.e, tok, "filter[tag]=twin&filter[cost]=1")
			if err != nil {
				return err
			}
			if ids := page.ids(); len(ids) != 1 || ids[0] != a.ID.String() {
				return fmt.Errorf("control: %s: ?filter[tag]=twin&filter[cost]=1 = %v, want only twin %s", where, ids, a.ID)
			}
			st.open++
			continue
		}
		st.masked++
		var first string
		for _, q := range []string{"filter[cost]=1", "filter[cost]=900", "filter[cost]=5", "filter[cost][in]=1,900", "filter[cost][in]=5,6",
			"filter[tag]=twin&filter[cost]=1", "filter[tag]=twin&filter[cost]=5", "sort=cost", "sort=-cost"} {
			r := run.e.ci8Do(ctx, run.e.h, http.MethodGet, i5rPath+"?"+q, tok, "", nil)
			body, code, err := s10Problem(r)
			if err != nil || r.code != http.StatusBadRequest || (code != "masked_field" && !strings.HasPrefix(q, "sort=")) {
				return fmt.Errorf("%s: ?%s = %d %s, want 400 masked_field", where, q, r.code, r.body)
			}
			if strings.HasPrefix(q, "sort=") {
				continue // cost is not a fixed column: unsortable for every caller, so the same answer either way
			}
			if strings.Contains(body, "900") || strings.Contains(body, `"total"`) {
				return fmt.Errorf("%s: ?%s: the 400 carries the value or a total: %s", where, q, r.body)
			}
			if first == "" {
				first = body
			} else if body != first {
				return fmt.Errorf("%s: the 400 of ?%s differs from the first:\n%s\n%s", where, q, body, first)
			}
		}
		_, page, err := i5rList(ctx, run.e, tok, "filter[tag]=twin")
		if err != nil {
			return err
		}
		if len(page.Items) != 2 || *page.Total != 2 {
			return fmt.Errorf("%s: ?filter[tag]=twin = %v total %d, want both twins", where, page.ids(), *page.Total)
		}
		strip := func(it map[string]any) string {
			it = mapsClone(it)
			for _, f := range []string{"id", "number"} {
				delete(it, f)
			}
			return i5rJSON(it)
		}
		if sa, sb := strip(page.Items[0]), strip(page.Items[1]); sa != sb {
			return fmt.Errorf("%s: the twins differ on the list beyond id and number:\n%s\n%s", where, sa, sb)
		}
		cursor, n := "", 0
		for {
			q := "filter[tag]=twin&limit=1"
			if cursor != "" {
				q += "&cursor=" + url.QueryEscape(cursor)
			}
			_, pg, err := i5rList(ctx, run.e, tok, q)
			if err != nil {
				return err
			}
			if len(pg.Items) != 1 || *pg.Total != 2 {
				return fmt.Errorf("%s: page %d of ?%s = %v total %d, want one twin and total 2", where, n+1, q, pg.ids(), *pg.Total)
			}
			n++
			if pg.NextCursor == nil {
				break
			}
			cursor = *pg.NextCursor
		}
		if n != 2 {
			return fmt.Errorf("%s: the twins took %d pages at limit 1, want 2", where, n)
		}
	}
	return nil
}

func mapsClone(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// i5RouteRows runs the route rows and adds them with add. filterMutant, when
// set, runs the data filters row against that mutant instead of the real
// compiler (NAVA_I5_MUTANT), which must turn it red. The filter mutants run
// on the first i5rMutantCases cases: they are caught early, and the rows
// stay inside the budget.
func i5RouteRows(ctx context.Context, cl Cluster, seed uint64, filterMutant string, add func(name, status, detail string)) {
	start := time.Now()
	run, stop, err := i5rSetup(ctx, cl, seed)
	defer func() {
		if stop != nil {
			if err := stop(); err != nil {
				add(i5RowFilters, statusFail, "cleanup: "+err.Error())
			}
		}
	}()
	rows := []string{i5RowFilters, i5RowListPage, i5RowMasked}
	broke := func(format string, a ...any) {
		for _, row := range rows {
			add(row, statusFail, fmt.Sprintf("seed %d: ", seed)+fmt.Sprintf(format, a...))
		}
	}
	if err != nil {
		broke("setup: %v", err)
		return
	}
	handlers := map[string]*etIso{"": run.e}
	for name, m := range run.mutants {
		handlers[name] = m
	}
	results := map[string]*i5rFilterResult{}
	for name := range handlers {
		results[name] = &i5rFilterResult{diffs: map[string]string{}}
	}
	var st i5rCounts
	var listErr, maskErr error
	for ci, c := range run.cases {
		pol, err := run.publish(ctx, c)
		if err != nil {
			broke("publish policy #%d: %v", ci, err)
			return
		}
		for name, h := range handlers {
			if name != "" && name != filterMutant && ci >= i5rMutantCases {
				continue
			}
			if err := run.filters(ctx, h, ci, c, pol, results[name]); err != nil {
				add(i5RowFilters, statusFail, fmt.Sprintf("seed %d, %s: the run broke: %v", seed, i5HandlerName(name), err))
				return
			}
		}
		if listErr == nil {
			listErr = run.lists(ctx, ci, c, pol, &st)
		}
		if maskErr == nil {
			maskErr = run.masked(ctx, ci, c, pol, &st)
		}
	}
	// The masked-field oracle's fixed case: every user reads every record,
	// and only Users[0] may read cost, so every other caller reads both twins
	// with cost masked whatever the random cases drew.
	if maskErr == nil {
		fixed := i5rMaskCase(run.w)
		pol, err := run.publish(ctx, fixed)
		if err != nil {
			broke("publish the masked-field case: %v", err)
			return
		}
		maskErr = run.masked(ctx, len(run.cases), fixed, pol, &st)
	}
	took := time.Since(start).Round(time.Millisecond)
	if took > i5rBudget {
		broke("%s, over the %s budget of the route rows", took, i5rBudget)
		return
	}

	// The data filters row: the real compiler, or the mutant NAVA_I5_MUTANT names.
	row := results[filterMutant]
	switch {
	case len(row.diffs) > 0:
		add(i5RowFilters, statusFail, fmt.Sprintf("seed %d, %s: %d record disagreements in %d filter requests, by the filtered field's shape %s; first: %s",
			seed, i5HandlerName(filterMutant), len(row.diffs), row.checks, i5rJSON(row.causes), row.first))
	case row.matched*10 < row.checks*i5rMinMatchedTenths || row.masked == 0 || row.defaultHits == 0:
		add(i5RowFilters, statusFail, fmt.Sprintf("seed %d: %d of %d requests match a record, %d are masked, %d hits on a defaulted prio; the row proves little",
			seed, row.matched, row.checks, row.masked, row.defaultHits))
	default:
		add(i5RowFilters, statusPass, fmt.Sprintf("seed %d, %s: %d policies × %d principals × %d filter requests over %d records agree (%d match a record, %d hits on a defaulted prio, %d are 400 masked_field); %s",
			seed, i5HandlerName(filterMutant), len(run.cases), i5rPrincipals, i5rFilters, len(run.all), row.matched, row.defaultHits, row.masked, took))
	}

	// A mutant is caught when it disagrees where the real compiler agreed.
	for _, name := range i5FilterMutantNames() {
		caught := ""
		for key := range results[name].diffs {
			if _, real := results[""].diffs[key]; !real && (caught == "" || key < caught) {
				caught = key
			}
		}
		if caught != "" {
			add(fmt.Sprintf(i5RowFilterMutant, name), statusPass, "caught: "+results[name].diffs[caught])
		} else {
			add(fmt.Sprintf(i5RowFilterMutant, name), statusFail, fmt.Sprintf("seed %d: the mutant agreed wherever the real list route did; the suite cannot see it", seed))
		}
	}

	callers := len(run.cases) * i5rPrincipals
	switch {
	case listErr != nil:
		add(i5RowListPage, statusFail, listErr.Error())
	case st.proper < callers/5:
		add(i5RowListPage, statusFail, fmt.Sprintf("seed %d: only %d of %d callers read a proper subset of the records; the row proves little", seed, st.proper, callers))
	default:
		add(i5RowListPage, statusPass, fmt.Sprintf("seed %d: %d policies × %d principals × %d paged sorts, %d pages: no unreadable record on any page, every readable one on exactly one, total = readable count on every page; %d callers read a proper subset",
			seed, len(run.cases), i5rPrincipals, i5rLists, st.pages, st.proper))
	}
	switch {
	case maskErr != nil:
		add(i5RowMasked, statusFail, maskErr.Error())
	case st.masked == 0 || st.open == 0:
		add(i5RowMasked, statusFail, fmt.Sprintf("seed %d: %d callers read the twins with cost masked and %d with it open; the row needs both", seed, st.masked, st.open))
	default:
		add(i5RowMasked, statusPass, fmt.Sprintf("seed %d: %d callers with cost masked could not tell the twins apart (filter eq and in: one 400 masked_field body; total, paging and items alike); %d callers with cost open could (control)",
			seed, st.masked, st.open))
	}
}

func i5HandlerName(mutant string) string {
	if mutant == "" {
		return "the real list route"
	}
	return "filter mutant " + mutant
}
