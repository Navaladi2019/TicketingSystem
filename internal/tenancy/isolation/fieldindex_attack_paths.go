package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// fieldIndexAttackPaths are the S21 attack rows (P7-08, TICK-748; ADR-006 D3,
// D6, D7, D14, R8, R9): flagged fields, data filters, publish and the
// backfill's statements, attacked as tenant A (and as tenant A's agents)
// through the real nava-core handler and as every runtime role.
//
// Rows 1, 2, 3 and 5 build the same fixture (s21Setup): iso_et_s21 in both
// tenants under the same key, every value type flagged and ready, tenant A's
// cost masked to its admins and its records split by a row ACL, tenant B's
// records holding values tenant A has not and values tenant A has. Every row
// has positive controls, so a dead route or a broken fixture fails instead of
// passing as a denial. A row deletes everything it writes (s21Cleanup,
// etIso.cleanup).
//
// Each row was shown red against a deliberately broken variant of the
// defence it proves; the variants are listed in the PR body (TICK-748).
func fieldIndexAttackPaths() []Path {
	db := func(c, name string, fn func(context.Context, *etIso, *s21Fixture) error) Path {
		return Path{Name: "S21 attack " + name, Case: c, Role: "nava_app", DB: func(ctx context.Context, cl Cluster) error {
			e, err := newETIso(ctx, cl)
			if err != nil {
				return err
			}
			defer e.stop()
			clean := func() error {
				if err := s21Cleanup(ctx, e); err != nil {
					return err
				}
				return e.cleanup()
			}
			if err := clean(); err != nil { // rows a killed earlier run left
				return err
			}
			fx, err := s21Setup(ctx, e)
			if err == nil {
				err = fn(ctx, e, fx)
			}
			if cerr := clean(); err == nil {
				err = cerr
			}
			return err
		}}
	}
	return []Path{
		db("CI-2", "1: tenant A learns nothing of tenant B's values or field states through a data filter: hits, total, status, error code and index_progress are the ones a value no tenant has gets, whatever state tenant B's fields are in", s21FilterOracle),
		db("CI-8", "2: a value only in a record the agent cannot read gives the same bytes as a value nobody has, for every operator; a masked field is the same 400 masked_field in every registry state, flagged or not", s21ACLOracle),
		db("CI-2", "3: publish writes registry and outbox rows for the caller's own tenant only; the backfill's statements, run as tenant A with tenant B's field id, read, write and flip nothing", s21Publish),
		{Name: "S21 attack 4: no runtime role can UPDATE a side table or write another tenant's side or registry row; registry column grants are exactly ADR-006 D14", Case: "CI-2", Role: "nava_app", DB: s21RoleGrants},
		db("CI-2", "5: field_not_ready carries the registry's percentage and nothing else: the same bytes for every value, caller and record count, whatever tenant B's progress", s21NotReadyNoCounts),
		db("CI-2", "3b: FieldIndexBackfill started from tenant A's outbox row naming tenant B (payload tenant_id or field_id) binds the tenant from the row (relay header), never from the payload, and writes nothing", s21Backfill3bRow),
	}
}

// backfillAttacker is the suite's Cluster when it can run row 3b's check:
// the check drives the real coordinator and FieldIndexBackfill, whose
// workflow types only internal/fieldindex may name outside tests
// (TestOnlyFieldindexNamesItsWorkflowTypes), so
// fieldindex_backfill_isolation_test.go gives the suite's cluster the method.
type backfillAttacker interface {
	S21Backfill3b(context.Context, *etIso, *s21Fixture) error
}

// s21Backfill3bRow is row 3b (P7-08b, TICK-975): see s21Backfill3b in
// fieldindex_backfill_isolation_test.go.
func s21Backfill3bRow(ctx context.Context, e *etIso, fx *s21Fixture) error {
	ba, ok := e.cluster.(backfillAttacker)
	if !ok {
		return errors.New("row 3b is not linked: fieldindex_backfill_isolation_test.go builds only under the integration tag")
	}
	return ba.S21Backfill3b(ctx, e, fx)
}

const (
	s21Key     = "iso_et_s21"
	s21Records = "/api/v1/records/" + s21Key
	// The callers: the admin reads every record and cost; the desk reads only
	// records in its group and never cost; the plain user reads nothing.
	s21Admin = etCallerA
	s21Desk  = auditCallerAAuditor
	s21Plain = auditCallerAAgent
	// ticketDefA is tenant A's seeded ticket type (testdata/seed.sql).
	ticketDefA = "0a000000-0000-4000-8000-0000000006a1"
	fieldTSA   = "0a000000-0000-4000-8000-0000000014a3"
	fieldBoolA = "0a000000-0000-4000-8000-0000000014a4"
	fieldTSB   = "0b000000-0000-4000-8000-0000000014b3"
)

// s21Definition is iso_et_s21 with the properties in flags marked
// x-nava-filterable: note (text), qty and cost (num), due (ts), vip (bool),
// label (text). The per-type quota is 5 (D12): tenant A flags s21FlagsA,
// tenant B s21FlagsB, which has label and not cost.
func s21Definition(flags ...string) string {
	prop := func(key, typ string) string {
		f := ""
		if slices.Contains(flags, key) {
			f = `,"x-nava-filterable":true`
		}
		return `"` + key + `":{` + typ + f + `}`
	}
	schema := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,` +
		`"required":["state"],"properties":{"state":{"type":"string","enum":["new","closed"],"default":"new"},` +
		strings.Join([]string{
			prop("note", `"type":"string"`), prop("qty", `"type":"number"`), prop("cost", `"type":"number"`),
			prop("due", `"type":"string","format":"date-time"`), prop("vip", `"type":"boolean"`), prop("label", `"type":"string"`),
		}, ",") + `,"group":{"type":"string","format":"uuid"}}}`
	return fmt.Sprintf(`{"schema": %s, "fixed_columns": {"state": "state", "group": "assignment_group_id"},
		"states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}`, schema)
}

//nolint:gochecknoglobals // fixed lists, read only
var (
	s21FlagsA = []string{"note", "qty", "cost", "due", "vip"}
	s21FlagsB = []string{"note", "qty", "due", "vip", "label"}
)

func s21TypeBody(prefix string, flags []string) string {
	return fmt.Sprintf(`{"key": %q, "name": "Iso S21", "number_prefix": %q, "definition": %s}`, s21Key, prefix, s21Definition(flags...))
}

// s21Rules: the admins read and write everything; the desk reads and writes
// the records of its own groups.
func s21Rules() []authz.Rule {
	return []authz.Rule{
		{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: s10UUIDs(adminGroupA)}},
		{ID: "desk", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: s10UUIDs(auditorGroupA)},
			Where: []authz.Cond{{Column: "assignment_group_id", Op: authz.CondIn, Caller: authz.CallerMyGroups}}},
	}
}

// s21Masks: cost is the admins'.
func s21Masks() []authz.Mask {
	admins := authz.Subject{Groups: s10UUIDs(adminGroupA)}
	return []authz.Mask{{Field: "cost", Read: admins, Write: admins}}
}

// s21Fixture: desk is tenant A's record the desk reads; admin is tenant A's
// record only the admin reads; b and bShared are tenant B's, bShared holding
// desk's values. defA and defB are the two iso_et_s21 definitions.
type s21Fixture struct {
	desk, admin, b, bShared string
	defA, defB              string
}

func s21Setup(ctx context.Context, e *etIso) (*s21Fixture, error) {
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", s21Admin, s21TypeBody("ISV", s21FlagsA), nil); code != http.StatusCreated {
		return nil, fmt.Errorf("control: tenant A creating %s = %d %s, want 201", s21Key, code, body)
	}
	if err := tenancy.InTenantTx(ctx, e.pool, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(ctx, tx, s21Key, s21Rules(), s21Masks())
		return err
	}); err != nil {
		return nil, fmt.Errorf("control: publishing tenant A's policy: %w", err)
	}
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerBAdmin, s21TypeBody("ISW", s21FlagsB), nil); code != http.StatusCreated {
		return nil, fmt.Errorf("control: tenant B creating its own %s = %d %s, want 201", s21Key, code, body)
	}
	fx := &s21Fixture{}
	for _, d := range []struct {
		tenant string
		out    *string
	}{{TenantA, &fx.defA}, {TenantB, &fx.defB}} {
		if err := e.super.QueryRow(ctx, `SELECT id::text FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, d.tenant, s21Key).Scan(d.out); err != nil {
			return nil, err
		}
	}
	// Control: the creates registered the flags, backfilling, under each
	// tenant's own definition: five each.
	for _, c := range []struct {
		tenant, def string
		want        int
	}{{TenantA, fx.defA, 5}, {TenantB, fx.defB, 5}} {
		var n int
		if err := e.super.QueryRow(ctx, `SELECT count(*) FROM record_index_fields WHERE tenant_id = $1 AND entity_definition_id = $2 AND state = 'backfilling'`,
			c.tenant, c.def).Scan(&n); err != nil {
			return nil, err
		}
		if n != c.want {
			return nil, fmt.Errorf("control: tenant %s's create registered %d backfilling fields, want %d", c.tenant, n, c.want)
		}
	}
	// No backfill runs here (P7-05): the fields are flipped ready before any
	// record, so the trigger indexes every write below.
	if err := s21SetState(ctx, e, "", "", "ready", 100); err != nil {
		return nil, err
	}
	var err error
	if fx.desk, err = s21Create(ctx, e, s21Admin, fmt.Sprintf(
		`{"data": {"note": "a-desk", "qty": 10, "cost": 600, "due": "2026-01-10T00:00:00Z", "vip": true, "label": "a-l", "group": %q}}`, auditorGroupA)); err != nil {
		return nil, err
	}
	if fx.admin, err = s21Create(ctx, e, s21Admin, fmt.Sprintf(
		`{"data": {"note": "a-secret", "qty": 4242, "cost": 7, "due": "2030-01-01T00:00:00Z", "vip": false, "label": "a-s", "group": %q}}`, adminGroupA)); err != nil {
		return nil, err
	}
	if fx.b, err = s21Create(ctx, e, auditCallerBAgent,
		`{"data": {"note": "b-only", "qty": 9999, "cost": 31337, "due": "2035-01-01T00:00:00Z", "vip": false, "label": "b-only"}}`); err != nil {
		return nil, err
	}
	if fx.bShared, err = s21Create(ctx, e, auditCallerBAgent,
		`{"data": {"note": "a-desk", "qty": 10, "cost": 600, "due": "2026-01-10T00:00:00Z", "vip": true, "label": "a-l"}}`); err != nil {
		return nil, err
	}
	// Controls: the trigger indexed both tenants' writes (tenant B's values
	// are there to be found), and the policy splits tenant A's records.
	var n int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM rfv_text v JOIN record_index_fields f ON f.tenant_id = v.tenant_id AND f.id = v.field_id
		WHERE v.tenant_id = $1 AND f.entity_definition_id = $2 AND f.field_key = 'note' AND v.value IN ('b-only', 'a-desk')`, TenantB, fx.defB).Scan(&n); err != nil {
		return nil, err
	}
	if n != 2 {
		return nil, fmt.Errorf("control: tenant B's note side rows = %d, want 2", n)
	}
	for _, c := range []struct {
		token, query string
		want         []string
	}{
		{s21Admin, "filter[note]=a-desk", []string{fx.desk}},
		{s21Admin, "filter[qty][gte]=0", []string{fx.desk, fx.admin}},
		{s21Desk, "filter[qty][gte]=0", []string{fx.desk}},
		{s21Plain, "filter[qty][gte]=0", nil},
		{etCallerBAdmin, "filter[note]=a-desk", []string{fx.bShared}},
	} {
		if err := s21ListHas(ctx, e, c.token, c.query, c.want...); err != nil {
			return nil, fmt.Errorf("control: %w", err)
		}
	}
	return fx, nil
}

func s21Create(ctx context.Context, e *etIso, token, body string) (string, error) {
	r := e.ci8Do(ctx, e.h, http.MethodPost, s21Records, token, body, nil)
	if r.code != http.StatusCreated {
		return "", fmt.Errorf("control: %s creating a record = %d %s, want 201", token, r.code, r.body)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(r.body), &out); err != nil || out.ID == "" {
		return "", fmt.Errorf("control: the created record has no id: %w: %s", err, r.body)
	}
	return out.ID, nil
}

// s21SetState sets state and progress on iso_et_s21's registry rows, as the
// superuser (the backfill is P7-05's): of tenant (every tenant when empty)
// and field key (every field when empty).
func s21SetState(ctx context.Context, e *etIso, tenant, key, state string, progress int) error {
	_, err := e.super.Exec(ctx, `UPDATE record_index_fields f SET state = $3, progress = $4, updated_at = now()
		FROM entity_definitions d WHERE d.tenant_id = f.tenant_id AND d.id = f.entity_definition_id AND d.key = $5
		AND ($1 = '' OR f.tenant_id::text = $1) AND ($2 = '' OR f.field_key = $2)`, tenant, key, state, progress, s21Key)
	if err != nil {
		return fmt.Errorf("set %s %s to %s: %w", tenant, key, state, err)
	}
	return nil
}

// s21FieldID is the registry id of tenant's iso_et_s21 field key.
func s21FieldID(ctx context.Context, e *etIso, tenant, key string) (string, error) {
	var id string
	err := e.super.QueryRow(ctx, `SELECT f.id::text FROM record_index_fields f JOIN entity_definitions d
		ON d.tenant_id = f.tenant_id AND d.id = f.entity_definition_id
		WHERE f.tenant_id = $1 AND d.key = $2 AND f.field_key = $3`, tenant, s21Key, key).Scan(&id)
	return id, err
}

// s21Cleanup deletes what the rows write past etIso.cleanup: the side rows,
// field-index outbox rows (backfill and remove) and registry rows of every
// iso_et_* type. Triggers are off, as in s10Cleanup.
func s21Cleanup(ctx context.Context, e *etIso) error {
	tx, err := e.super.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	const iso = `SELECT f.tenant_id, f.id FROM record_index_fields f JOIN entity_definitions d
		ON d.tenant_id = f.tenant_id AND d.id = f.entity_definition_id WHERE d.key LIKE 'iso\_et\_%'`
	stmts := []string{`SET LOCAL session_replication_role = replica`}
	for _, t := range []string{"rfv_text", "rfv_num", "rfv_ts", "rfv_bool"} {
		stmts = append(stmts, `DELETE FROM `+t+` v WHERE (v.tenant_id, v.field_id) IN (`+iso+`)`)
	}
	stmts = append(stmts,
		`DELETE FROM outbox WHERE kind IN ('`+outbox.FieldIndexBackfillKind+`', '`+outbox.FieldIndexRemoveKind+`') AND (payload->>'field_id')::uuid IN (SELECT id FROM (`+iso+`) i)`,
		`DELETE FROM record_index_fields WHERE (tenant_id, id) IN (`+iso+`)`)
	for _, q := range stmts {
		if _, err := tx.Exec(ctx, q); err != nil {
			return fmt.Errorf("s21 cleanup: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// s21Answer is a list response reduced to what may differ between two
// requests: the status and the body without its instance, re-encoded so
// member order cannot differ.
type s21Answer struct {
	code int
	body string
}

func s21Ask(ctx context.Context, e *etIso, token, query string) (s21Answer, error) {
	r := e.ci8Do(ctx, e.h, http.MethodGet, s21Records+"?"+query, token, "", nil)
	body, err := withoutInstance(r.body)
	if err != nil {
		return s21Answer{}, fmt.Errorf("%s ?%s = %d: %w", token, query, r.code, err)
	}
	return s21Answer{r.code, body}, nil
}

// s21ListHas: token's list holds exactly want, in any order, with total
// len(want).
func s21ListHas(ctx context.Context, e *etIso, token, query string, want ...string) error {
	r := e.ci8Do(ctx, e.h, http.MethodGet, s21Records+"?"+query, token, "", nil)
	var p s10Page
	if r.code != http.StatusOK {
		return fmt.Errorf("%s ?%s = %d %s, want 200", token, query, r.code, r.body)
	}
	if err := json.Unmarshal([]byte(r.body), &p); err != nil || p.Total == nil {
		return fmt.Errorf("%s ?%s: not a page: %w: %s", token, query, err, r.body)
	}
	got := p.ids()
	slices.Sort(got)
	want = slices.Clone(want)
	slices.Sort(want)
	if !slices.Equal(got, want) || *p.Total != len(want) {
		return fmt.Errorf("%s's list ?%s holds %v with total %d, want %v", token, query, got, *p.Total, want)
	}
	return nil
}

// s21SamePair: as token, the query naming a value that exists only where the
// caller must not see it, and the query naming a value nobody has, are both
// 200 with no item, total 0, and the same bytes.
func s21SamePair(ctx context.Context, e *etIso, token string, pair [2]string) error {
	var ans [2]s21Answer
	for i, q := range pair {
		a, err := s21Ask(ctx, e, token, q)
		if err != nil {
			return err
		}
		if a.code != http.StatusOK {
			return fmt.Errorf("%s ?%s = %d %s, want 200", token, q, a.code, a.body)
		}
		var p s10Page
		if err := json.Unmarshal([]byte(a.body), &p); err != nil || p.Total == nil {
			return fmt.Errorf("%s ?%s: not a page: %w: %s", token, q, err, a.body)
		}
		if len(p.Items) != 0 || *p.Total != 0 {
			return fmt.Errorf("%s ?%s found %v with total %d, want nothing", token, q, p.ids(), *p.Total)
		}
		ans[i] = a
	}
	if ans[0] != ans[1] {
		return fmt.Errorf("%s: ?%s (a hidden value) and ?%s (no value) differ:\n%s\n%s", token, pair[0], pair[1], ans[0].body, ans[1].body)
	}
	return nil
}

// Row 1. Tenant A holds a valid token and names, in a filter, values only
// tenant B has, under the same type key and the same field keys. Every
// answer is the one a value no tenant has gets. Then tenant B's fields go
// through every registry state, and tenant A's answers do not move by one
// byte: tenant A's error codes and index_progress are tenant A's registry's
// alone, so they cannot tell tenant A whether tenant B flagged a field, how
// far its backfill is, or that it is being removed. Timing class: the side
// table is reached only through the correlated EXISTS on tenant A's own
// record ids (D7), so tenant B's rows are never visited; the pushdown of the
// tenant predicate into the index condition is pinned by
// TestListFilterPushdownIsAnIndexCond and the single statement by D7's
// static test.
func s21FilterOracle(ctx context.Context, e *etIso, fx *s21Fixture) error {
	for _, pair := range [][2]string{
		{"filter[note]=b-only", "filter[note]=zz-none"},
		{"filter[note][in]=b-only,zz-x", "filter[note][in]=zz-y,zz-x"},
		{"filter[qty]=9999", "filter[qty]=12345"},
		{"filter[qty][gte]=5000", "filter[qty][gte]=50000"},
		{"filter[qty][in]=9999,1", "filter[qty][in]=12345,1"},
		{"filter[cost]=31337", "filter[cost]=31338"},
		{"filter[due]=2035-01-01T00:00:00Z", "filter[due]=2036-01-01T00:00:00Z"},
		{"filter[due][gt]=2031-01-01T00:00:00Z", "filter[due][gt]=2040-01-01T00:00:00Z"},
		{"filter[vip]=false&filter[note]=b-only", "filter[vip]=false&filter[note]=zz-none"},
		{"filter[state]=new&filter[qty][lt]=10000&filter[qty][gt]=5000", "filter[state]=new&filter[qty][lt]=100000&filter[qty][gt]=50000"},
	} {
		if err := s21SamePair(ctx, e, s21Admin, pair); err != nil {
			return err
		}
	}
	// A value both tenants hold finds tenant A's record only.
	for _, q := range []string{"filter[note]=a-desk", "filter[qty]=10", "filter[cost]=600", "filter[vip]=true", "filter[due]=2026-01-10T00:00:00Z"} {
		if err := s21ListHas(ctx, e, s21Admin, q, fx.desk); err != nil {
			return err
		}
	}
	// label is flagged by tenant B only: tenant A's filter on it is tenant
	// A's 400 field_not_filterable.
	queries := []string{"filter[note]=b-only", "filter[note]=a-desk", "filter[qty][gt]=1", "filter[vip]=false",
		"filter[label]=b-only", "filter[label]=a-l", "filter[qty]=abc", "filter[due][lt]=2026-13-01"}
	base := make([]s21Answer, len(queries))
	for i, q := range queries {
		a, err := s21Ask(ctx, e, s21Admin, q)
		if err != nil {
			return err
		}
		base[i] = a
	}
	for i, want := range []int{200, 200, 200, 200, 400, 400, 400, 400} {
		if base[i].code != want {
			return fmt.Errorf("tenant A's ?%s = %d %s, want %d: only tenant A's own registry decides", queries[i], base[i].code, base[i].body, want)
		}
	}
	for _, st := range []struct {
		state    string
		progress int
	}{{"backfilling", 13}, {"removing", 100}, {"backfilling", 0}, {"ready", 100}} {
		if err := s21SetState(ctx, e, TenantB, "", st.state, st.progress); err != nil {
			return err
		}
		for i, q := range queries {
			a, err := s21Ask(ctx, e, s21Admin, q)
			if err != nil {
				return err
			}
			if a != base[i] {
				return fmt.Errorf("tenant B's fields %s %d%%: tenant A's ?%s moved from %d %s to %d %s",
					st.state, st.progress, q, base[i].code, base[i].body, a.code, a.body)
			}
		}
	}
	// Controls: tenant A's own state does move tenant A's answer, and only
	// tenant A's progress is told.
	if err := s21SetState(ctx, e, TenantB, "note", "backfilling", 77); err != nil {
		return err
	}
	if err := s21SetState(ctx, e, TenantA, "note", "backfilling", 13); err != nil {
		return err
	}
	if err := s21NotReady(ctx, e, s21Admin, "filter[note]=b-only", 13); err != nil {
		return fmt.Errorf("control: %w", err)
	}
	if err := s21SetState(ctx, e, TenantB, "note", "ready", 100); err != nil {
		return err
	}
	if err := s21SetState(ctx, e, TenantA, "note", "removing", 100); err != nil {
		return err
	}
	a, err := s21Ask(ctx, e, s21Admin, "filter[note]=b-only")
	if err != nil {
		return err
	}
	if a.code != http.StatusBadRequest || !strings.Contains(a.body, `"field_not_filterable"`) {
		return fmt.Errorf("control: tenant A's note removing, tenant B's ready: ?filter[note]=b-only = %d %s, want 400 field_not_filterable", a.code, a.body)
	}
	return s21SetState(ctx, e, TenantA, "note", "ready", 100)
}

// s21NotReady: token's query is 422 field_not_ready with index_progress
// progress.
func s21NotReady(ctx context.Context, e *etIso, token, query string, progress int) error {
	a, err := s21Ask(ctx, e, token, query)
	if err != nil {
		return err
	}
	var p struct {
		Code          string `json:"code"`
		IndexProgress *int   `json:"index_progress"`
	}
	if err := json.Unmarshal([]byte(a.body), &p); err != nil {
		return err
	}
	if a.code != http.StatusUnprocessableEntity || p.Code != "field_not_ready" || p.IndexProgress == nil || *p.IndexProgress != progress {
		return fmt.Errorf("%s ?%s = %d %s, want 422 field_not_ready with index_progress %d", token, query, a.code, a.body, progress)
	}
	return nil
}

// Row 2. Inside tenant A. The desk cannot read the admin record, and the
// plain user reads nothing: a value only an unreadable record holds gives
// the bytes a value nobody holds gives, for every operator, alone or with
// other filters. The desk cannot read cost: its filter on cost is the same
// 400 masked_field, with no index_progress, whether the field is ready,
// backfilling, being removed or not flagged at all, and whether a record
// holds the value or not. The mask is checked before the registry (D6 step
// 1), so the desk cannot learn the field's index state either (R9: the mask
// is row-independent, so the check needs no row).
func s21ACLOracle(ctx context.Context, e *etIso, fx *s21Fixture) error {
	for _, pair := range [][2]string{
		{"filter[note]=a-secret", "filter[note]=zz-none"},
		{"filter[note][in]=a-secret,zz-x", "filter[note][in]=zz-y,zz-x"},
		{"filter[qty]=4242", "filter[qty]=4243"},
		{"filter[qty][gt]=100", "filter[qty][gt]=100000"},
		{"filter[qty][lte]=4242&filter[qty][gte]=4242", "filter[qty][lte]=4243&filter[qty][gte]=4243"},
		{"filter[due][gte]=2029-01-01T00:00:00Z", "filter[due][gte]=2045-01-01T00:00:00Z"},
		{"filter[vip]=false", "filter[vip]=false&filter[note]=zz-none"},
		{"filter[state]=new&filter[note]=a-secret", "filter[state]=new&filter[note]=zz-none"},
		{"filter[group]=" + adminGroupA + "&filter[qty]=4242", "filter[group]=" + adminGroupA + "&filter[qty]=4243"},
	} {
		if err := s21SamePair(ctx, e, s21Desk, pair); err != nil {
			return err
		}
	}
	for _, pair := range [][2]string{
		{"filter[note]=a-desk", "filter[note]=zz-none"},
		{"filter[qty][gte]=0", "filter[qty][lt]=0"},
	} {
		if err := s21SamePair(ctx, e, s21Plain, pair); err != nil {
			return err
		}
	}
	// Masked, in every state. Each query names cost with a value the desk's
	// own record holds, one only the admin record holds, and one nobody
	// holds; all must answer one body.
	masked := []string{"filter[cost]=600", "filter[cost]=7", "filter[cost]=1", "filter[cost][gt]=0",
		"filter[cost][in]=600,1", "filter[note]=a-desk&filter[cost]=600", "filter[cost]=abc"}
	var want s21Answer
	check := func(state string) error {
		for _, q := range masked {
			a, err := s21Ask(ctx, e, s21Desk, q)
			if err != nil {
				return err
			}
			switch {
			case a.code != http.StatusBadRequest || !strings.Contains(a.body, `"code":"masked_field"`):
				return fmt.Errorf("cost %s: the desk's ?%s = %d %s, want 400 masked_field", state, q, a.code, a.body)
			case strings.Contains(a.body, "index_progress") || strings.Contains(a.body, `"total"`):
				return fmt.Errorf("cost %s: the desk's ?%s tells the index state or a count: %s", state, q, a.body)
			case want.body == "":
				want = a
			case a != want:
				return fmt.Errorf("cost %s: the desk's ?%s differs from ?%s ready:\n%s\n%s", state, q, masked[0], a.body, want.body)
			}
		}
		return nil
	}
	for _, st := range []struct {
		state    string
		progress int
		admin    string // the admin's answer to filter[cost]=600: the control that the state took
	}{{"ready", 100, "200"}, {"backfilling", 50, "field_not_ready"}, {"removing", 100, "field_not_filterable"}} {
		if err := s21SetState(ctx, e, TenantA, "cost", st.state, st.progress); err != nil {
			return err
		}
		a, err := s21Ask(ctx, e, s21Admin, "filter[cost]=600")
		if err != nil {
			return err
		}
		if (st.admin == "200") != (a.code == http.StatusOK) || (st.admin != "200" && !strings.Contains(a.body, `"`+st.admin+`"`)) {
			return fmt.Errorf("control: cost %s: the admin's ?filter[cost]=600 = %d %s, want %s", st.state, a.code, a.body, st.admin)
		}
		if err := check(st.state); err != nil {
			return err
		}
	}
	// Not flagged at all: the registry row and its side rows are gone.
	costID, err := s21FieldID(ctx, e, TenantA, "cost")
	if err != nil {
		return err
	}
	for _, q := range []string{`DELETE FROM rfv_num WHERE tenant_id = $1 AND field_id = $2`, `DELETE FROM record_index_fields WHERE tenant_id = $1 AND id = $2`} {
		if _, err := e.super.Exec(ctx, q, TenantA, costID); err != nil {
			return err
		}
	}
	a, err := s21Ask(ctx, e, s21Admin, "filter[cost]=600")
	if err != nil {
		return err
	}
	if a.code != http.StatusBadRequest || !strings.Contains(a.body, `"field_not_filterable"`) {
		return fmt.Errorf("control: cost unflagged: the admin's ?filter[cost]=600 = %d %s, want 400 field_not_filterable", a.code, a.body)
	}
	return check("not flagged")
}

// s21Registry is every registry row of tenant, read past RLS, as one string.
func s21Registry(ctx context.Context, e *etIso, tenant string) (string, error) {
	var s string
	err := e.super.QueryRow(ctx, `SELECT coalesce(string_agg(id::text || ' ' || entity_definition_id::text || ' ' || field_key || ' ' || value_type || ' ' ||
		state || ' ' || progress::text || ' ' || requested_version::text, ', ' ORDER BY id), '') FROM record_index_fields WHERE tenant_id = $1`, tenant).Scan(&s)
	return s, err
}

// s21Backfills is every backfill outbox row, read past RLS: tenant_id,
// payload.
func s21Backfills(ctx context.Context, e *etIso) ([][3]string, error) {
	rows, err := e.super.Query(ctx, `SELECT tenant_id::text, coalesce(payload->>'tenant_id', ''), coalesce(payload->>'field_id', '')
		FROM outbox WHERE kind = $1 ORDER BY id`, outbox.FieldIndexBackfillKind)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) ([3]string, error) {
		var v [3]string
		err := r.Scan(&v[0], &v[1], &v[2])
		return v, err
	})
}

// Row 3. Publish: tenant A publishing a flagged definition to tenant B's
// only key gets the unknown key's 404 and writes no registry or outbox row
// anywhere; publishing a new flag on its own iso_et_s21, with tenant B
// forged in a header, writes one registry row and one outbox row, both
// tenant A's, whose payload names tenant A and tenant A's new field, and
// leaves tenant B's same-key type alone. Then the backfill's statements
// (ADR-006 D10; the workflow is P7-05's) run as nava_worker in tenant A with
// tenant B's field id, as a crafted outbox payload would hand them: the
// registry lock finds nothing, the side-table insert is refused by the
// registry foreign key, the flip and the progress update reach no row, and
// tenant B's registry and side rows are unchanged.
func s21Publish(ctx context.Context, e *etIso, fx *s21Fixture) error {
	regB, err := s21Registry(ctx, e, TenantB)
	if err != nil {
		return err
	}
	regA, err := s21Registry(ctx, e, TenantA)
	if err != nil {
		return err
	}
	backfills, err := s21Backfills(ctx, e)
	if err != nil {
		return err
	}
	body := fmt.Sprintf(`{"definition": %s}`, s21Definition(append(slices.Clone(s21FlagsA), "label")...))
	if err := e.sameNotFound(ctx, http.MethodPost, body, "/api/v1/entity-types/incident/versions", "/api/v1/entity-types/iso_et_none/versions"); err != nil {
		return err
	}
	for _, c := range []struct{ tenant, before string }{{TenantA, regA}, {TenantB, regB}} {
		after, err := s21Registry(ctx, e, c.tenant)
		if err != nil {
			return err
		}
		if after != c.before {
			return fmt.Errorf("publish to tenant B's key: tenant %s's registry moved:\n%s\n%s", c.tenant, c.before, after)
		}
	}
	if after, err := s21Backfills(ctx, e); err != nil || !slices.Equal(after, backfills) {
		return fmt.Errorf("publish to tenant B's key: backfill outbox rows moved (%w): %v -> %v", err, backfills, after)
	}
	// Tenant A flags label on its own iso_et_s21, naming tenant B in a header.
	// Its five flags are the quota, so vip's registry row and side rows go
	// first, past the API (removal is S22's); vip stays flagged in the
	// definition, so the publish adds label only.
	vipA, err := s21FieldID(ctx, e, TenantA, "vip")
	if err != nil {
		return err
	}
	for _, q := range []string{`DELETE FROM rfv_bool WHERE tenant_id = $1 AND field_id = $2`, `DELETE FROM record_index_fields WHERE tenant_id = $1 AND id = $2`} {
		if _, err := e.super.Exec(ctx, q, TenantA, vipA); err != nil {
			return err
		}
	}
	code, out := e.do(ctx, http.MethodPost, "/api/v1/entity-types/"+s21Key+"/versions", s21Admin, body, http.Header{"X-Tenant-Id": {TenantB}})
	if code != http.StatusCreated {
		return fmt.Errorf("control: tenant A flagging label = %d %s, want 201", code, out)
	}
	labelA, err := s21FieldID(ctx, e, TenantA, "label")
	if err != nil {
		return fmt.Errorf("tenant A's label registry row: %w", err)
	}
	after, err := s21Backfills(ctx, e)
	if err != nil {
		return err
	}
	added := slices.DeleteFunc(slices.Clone(after), func(r [3]string) bool { return slices.Contains(backfills, r) })
	if want := [][3]string{{TenantA, TenantA, labelA}}; !slices.Equal(added, want) {
		return fmt.Errorf("tenant A's flag wrote backfill rows %v, want %v", added, want)
	}
	if got, err := s21Registry(ctx, e, TenantB); err != nil || got != regB {
		return fmt.Errorf("tenant A's flag moved tenant B's registry (%w):\n%s\n%s", err, regB, got)
	}
	return s21BackfillAsA(ctx, e, fx)
}

// s21BackfillAsA runs the D10 statements as nava_worker in tenant A against
// tenant B's note field, and against tenant A's own as the control.
func s21BackfillAsA(ctx context.Context, e *etIso, fx *s21Fixture) error {
	noteB, err := s21FieldID(ctx, e, TenantB, "note")
	if err != nil {
		return err
	}
	noteA, err := s21FieldID(ctx, e, TenantA, "note")
	if err != nil {
		return err
	}
	regB, err := s21Registry(ctx, e, TenantB)
	if err != nil {
		return err
	}
	sideB := func() (int, error) {
		var n int
		err := e.super.QueryRow(ctx, `SELECT count(*) FROM rfv_text WHERE tenant_id = $1 AND field_id = $2`, TenantB, noteB).Scan(&n)
		return n, err
	}
	nB, err := sideB()
	if err != nil || nB != 2 {
		return fmt.Errorf("control: tenant B's note side rows = %d (%w), want 2", nB, err)
	}
	worker, err := e.cluster.Pool("nava_worker")
	if err != nil {
		return err
	}
	type step struct {
		name  string
		q     string
		args  func(field string) []any
		ownN  int64  // rows it reaches for tenant A's own field
		other string // "" for 0 rows, else the SQLSTATE class that must refuse it
	}
	steps := []step{
		{"lock the registry row by tenant and id", `SELECT state FROM record_index_fields WHERE tenant_id = $1 AND id = $2 FOR SHARE`,
			func(f string) []any { return []any{TenantA, f} }, 1, ""},
		{"lock the registry row by id alone", `SELECT state FROM record_index_fields WHERE id = $1 FOR SHARE`,
			func(f string) []any { return []any{f} }, 1, ""},
		{"read the field's records", `SELECT r.id FROM records r JOIN record_index_fields f ON f.tenant_id = r.tenant_id AND f.entity_definition_id = r.entity_definition_id
			WHERE f.id = $1 ORDER BY r.id LIMIT 1000 FOR SHARE OF r`,
			func(f string) []any { return []any{f} }, 2, ""},
		{"index a value under the field", `INSERT INTO rfv_text (tenant_id, field_id, record_id, ord, value) VALUES ($1, $2, $3, 9, 'forged')`,
			func(f string) []any { return []any{TenantA, f, fx.desk} }, 1, "23"},
		{"delete the field's values", `DELETE FROM rfv_text WHERE field_id = $1`,
			func(f string) []any { return []any{f} }, 2, ""},
		{"report progress", `UPDATE record_index_fields SET progress = 42, updated_at = now() WHERE id = $1`,
			func(f string) []any { return []any{f} }, 1, ""},
		{"flip the field to ready", `UPDATE record_index_fields SET state = 'ready', updated_at = now() WHERE id = $1`,
			func(f string) []any { return []any{f} }, 1, ""},
	}
	for _, s := range steps {
		err := inTenant(ctx, worker, TenantA, func(tx pgx.Tx) error {
			// Control: tenant A's own field.
			n, err := execOne(ctx, tx, s.q, s.args(noteA)...)
			if err != nil || n != s.ownN {
				return fmt.Errorf("control: nava_worker %s on tenant A's own field: %d rows, %w; want %d", s.name, n, err, s.ownN)
			}
			n, err = execOne(ctx, tx, s.q, s.args(noteB)...)
			switch {
			case s.other == "" && (err != nil || n != 0):
				return fmt.Errorf("nava_worker in tenant A: %s with tenant B's field id reached %d rows (%w), want 0", s.name, n, err)
			case s.other != "" && (err == nil || !strings.HasPrefix(sqlState(err), s.other)):
				return fmt.Errorf("nava_worker in tenant A: %s with tenant B's field id = %d rows, %w; want SQLSTATE class %s", s.name, n, err, s.other)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	// A crafted start: the only enqueue path stamps the transaction's
	// tenant into the payload, so a row of tenant A can name tenant B's
	// field but never tenant B. The workflow then finds the field missing
	// in tenant A (above) and stops: row 3b runs it.
	app, err := e.cluster.Pool("nava_app")
	if err != nil {
		return err
	}
	// The row is committed, so it is read past RLS (nava_app cannot read
	// outbox payloads); s21Cleanup deletes it with tenant B's field.
	var id uuid.UUID
	if err := tenancy.InTenantTx(ctx, app, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
		var err error
		id, err = outbox.EnqueueFieldIndexBackfill(ctx, tx, uuid.MustParse(noteB), 1)
		return err
	}); err != nil {
		return fmt.Errorf("control: crafted enqueue: %w", err)
	}
	var tenant, payloadTenant string
	if err := e.super.QueryRow(ctx, `SELECT tenant_id::text, payload->>'tenant_id' FROM outbox WHERE id = $1`, id).Scan(&tenant, &payloadTenant); err != nil {
		return err
	}
	if tenant != TenantA || payloadTenant != TenantA {
		return fmt.Errorf("crafted enqueue in tenant A wrote a row of tenant %s naming tenant %s", tenant, payloadTenant)
	}
	if got, err := s21Registry(ctx, e, TenantB); err != nil || got != regB {
		return fmt.Errorf("the backfill statements moved tenant B's registry (%w):\n%s\n%s", err, regB, got)
	}
	if n, err := sideB(); err != nil || n != nB {
		return fmt.Errorf("the backfill statements moved tenant B's note side rows: %d -> %d (%w)", nB, n, err)
	}
	return nil
}

// d14 is ADR-006 D14 as role → privilege → registry columns ("*" for the
// whole row) on record_index_fields, and the side-table privileges.
func d14Registry() map[string]map[string][]string {
	return map[string]map[string][]string{
		"nava_app":    {"SELECT": {"*"}, "INSERT": {"*"}, "UPDATE": {"state", "progress", "requested_version", "updated_at"}},
		"nava_worker": {"SELECT": {"*"}, "UPDATE": {"state", "progress", "updated_at"}, "DELETE": {"*"}},
	}
}

func d14Side() map[string][]string {
	return map[string][]string{"nava_app": {"SELECT", "INSERT", "DELETE"}, "nava_worker": {"SELECT", "INSERT", "DELETE"}}
}

// s21RuntimeRoles are the manifest roles that open SQL sessions on the
// platform database (CONNECT and USAGE on public), but the owner.
func s21RuntimeRoles() ([]string, error) {
	m, err := grants.Parse(db.Grants())
	if err != nil {
		return nil, err
	}
	var out []string
	for _, name := range m.RoleNames() {
		r := m.Roles[name]
		if name == m.Owner() || !slices.Contains(r.Database, "CONNECT") || !slices.Contains(r.Schemas["public"], "USAGE") {
			continue
		}
		out = append(out, name)
	}
	for _, must := range []string{"nava_app", "nava_worker", "nava_relay", "nava_erasure"} {
		if !slices.Contains(out, must) {
			return nil, fmt.Errorf("control: runtime roles %v miss %s", out, must)
		}
	}
	return out, nil
}

// s21RecordWritersReadRegistry is D14's last bullet: every runtime role the
// manifest lets write records holds the registry SELECT of D14, so the
// records trigger, which runs as the writer (D4, SECURITY INVOKER), can read
// the flagged fields. s21RoleProbe then proves each such SELECT works and
// reaches tenant A's rows only.
func s21RecordWritersReadRegistry(roles []string) error {
	m, err := grants.Parse(db.Grants())
	if err != nil {
		return err
	}
	var writers []string
	for _, role := range roles {
		recs := m.Roles[role].Tables["records"]
		_, ins := recs["INSERT"]
		_, upd := recs["UPDATE"]
		if !ins && !upd {
			continue
		}
		writers = append(writers, role)
		if _, ok := d14Registry()[role]["SELECT"]; !ok {
			return fmt.Errorf("%s writes records but D14 gives it no SELECT on record_index_fields", role)
		}
	}
	if !slices.Contains(writers, "nava_app") || !slices.Contains(writers, "nava_worker") {
		return fmt.Errorf("control: the records writers %v miss nava_app or nava_worker", writers)
	}
	return nil
}

// s21SideRows are, per side table, tenant A's and tenant B's own field on
// their own record, and a value literal.
func s21SideRows() []struct{ table, fieldA, fieldB, value string } {
	return []struct{ table, fieldA, fieldB, value string }{
		{"rfv_text", SharedField, SharedField, "'iso-s21'"},
		{"rfv_num", fieldNumA, FieldNumB, "3"},
		{"rfv_ts", fieldTSA, fieldTSB, "now()"},
		{"rfv_bool", fieldBoolA, FieldBoolB, "true"},
	}
}

// Row 4. Every runtime role, in tenant A's transaction, as raw SQL: UPDATE
// of a side table is refused (42501) for every role and column; a side or
// registry row for tenant B, or for tenant A pointing at tenant B's field or
// record, is refused; only the D14 roles read or write the side tables and
// the registry, and the registry's UPDATE columns are exactly D14's. Each
// granted statement is shown to work on tenant A's own rows, so a refusal
// cannot come from a broken statement. The catalogue form of the same grants
// is TestP701Grants.
func s21RoleGrants(ctx context.Context, c Cluster) error {
	roles, err := s21RuntimeRoles()
	if err != nil {
		return err
	}
	if err := s21RecordWritersReadRegistry(roles); err != nil {
		return err
	}
	super, err := c.Super()
	if err != nil {
		return err
	}
	rows, err := super.Query(ctx, `SELECT attname::text FROM pg_attribute WHERE attrelid = 'record_index_fields'::regclass AND attnum > 0 AND NOT attisdropped ORDER BY attnum`)
	if err != nil {
		return err
	}
	cols, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil || len(cols) != 9 {
		return fmt.Errorf("control: record_index_fields columns %v (%w), want the 9 of D2", cols, err)
	}
	for _, role := range roles {
		pool, err := c.Pool(role)
		if err != nil {
			return err
		}
		if err := inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
			return s21RoleProbe(ctx, tx, role, cols)
		}); err != nil {
			return fmt.Errorf("%s: %w", role, err)
		}
	}
	return nil
}

// s21Want checks one probe: granted means it must run and reach want rows
// (-1 for any); otherwise it must be refused with 42501.
func s21Want(what string, granted bool, n int64, err error, want int64) error {
	if !granted {
		return refused(what, err)
	}
	if err != nil {
		return fmt.Errorf("%s: %w, want it granted", what, err)
	}
	if want >= 0 && n != want {
		return fmt.Errorf("%s: %d rows, want %d", what, n, want)
	}
	return nil
}

// s21Denied: the statement reaches no row or is refused by a grant, a policy
// or an integrity constraint.
func s21Denied(what string, n int64, err error) error {
	if err == nil && n == 0 {
		return nil
	}
	if err != nil && (sqlState(err) == "42501" || strings.HasPrefix(sqlState(err), "23")) {
		return nil
	}
	return fmt.Errorf("%s: %d rows, %w; want it denied", what, n, err)
}

func s21RoleProbe(ctx context.Context, tx pgx.Tx, role string, cols []string) error {
	reg := d14Registry()[role]
	side := d14Side()[role]
	has := func(priv string) bool { _, ok := reg[priv]; return ok }
	// Side tables.
	for _, s := range s21SideRows() {
		n, err := execOne(ctx, tx, `SELECT 1 FROM `+s.table+` WHERE tenant_id <> $1`, TenantA)
		if slices.Contains(side, "SELECT") {
			if err != nil || n != 0 {
				return fmt.Errorf("%s: read another tenant's rows: %d rows, %w", s.table, n, err)
			}
			if n, err := execOne(ctx, tx, `SELECT 1 FROM `+s.table+` WHERE field_id = $1`, s.fieldA); err != nil || n == 0 {
				return fmt.Errorf("control: %s: read tenant A's own rows: %d rows, %w", s.table, n, err)
			}
		} else if err := refused(s.table+": SELECT", err); err != nil {
			return err
		}
		for _, col := range []string{"value", "tenant_id", "field_id", "record_id", "ord"} {
			_, err := execOne(ctx, tx, `UPDATE `+s.table+` SET `+col+` = `+col+` WHERE tenant_id = $1`, TenantA)
			if err := refused(s.table+": UPDATE "+col+" of tenant A's own rows", err); err != nil {
				return err
			}
		}
		insert := `INSERT INTO ` + s.table + ` (tenant_id, field_id, record_id, ord, value) VALUES ($1, $2, $3, 7, ` + s.value + `)`
		n, err = execOne(ctx, tx, insert, TenantA, s.fieldA, RecordA1)
		if err := s21Want(s.table+": INSERT tenant A's own row", slices.Contains(side, "INSERT"), n, err, 1); err != nil {
			return err
		}
		for _, c := range []struct {
			what                  string
			tenant, field, record string
		}{
			{"a row of tenant B", TenantB, s.fieldB, RecordB1},
			{"tenant A's field on tenant B's record", TenantA, s.fieldA, RecordB1},
			{"tenant B's field on tenant A's record", TenantA, s.fieldB, RecordA1},
		} {
			if s.fieldB == s.fieldA && c.field == s.fieldB && c.tenant == TenantA {
				continue // the shared id is tenant A's own field too
			}
			n, err := execOne(ctx, tx, insert, c.tenant, c.field, c.record)
			if err := s21Denied(s.table+": INSERT "+c.what, n, err); err != nil {
				return err
			}
		}
		n, err = execOne(ctx, tx, `DELETE FROM `+s.table+` WHERE tenant_id = $1`, TenantB)
		if slices.Contains(side, "DELETE") {
			if err != nil || n != 0 {
				return fmt.Errorf("%s: DELETE tenant B's rows: %d rows, %w; want 0", s.table, n, err)
			}
			if n, err := execOne(ctx, tx, `DELETE FROM `+s.table+` WHERE tenant_id = $1 AND ord = 7`, TenantA); err != nil || n != 1 {
				return fmt.Errorf("control: %s: DELETE tenant A's own row: %d rows, %w", s.table, n, err)
			}
		} else if err := refused(s.table+": DELETE", err); err != nil {
			return err
		}
	}
	// The registry.
	n, err := execOne(ctx, tx, `SELECT tenant_id FROM record_index_fields`)
	if err := s21Want("record_index_fields: SELECT", has("SELECT"), n, err, -1); err != nil {
		return err
	}
	if has("SELECT") {
		if n == 0 {
			return errors.New("control: record_index_fields: SELECT found none of tenant A's rows")
		}
		if n, err := execOne(ctx, tx, `SELECT 1 FROM record_index_fields WHERE tenant_id <> $1`, TenantA); err != nil || n != 0 {
			return fmt.Errorf("record_index_fields: read another tenant's rows: %d rows, %w", n, err)
		}
	}
	for _, col := range cols {
		n, err := execOne(ctx, tx, `UPDATE record_index_fields SET `+col+` = `+col+` WHERE tenant_id = $1 AND id = $2`, TenantA, fieldBoolA)
		granted := slices.Contains(reg["UPDATE"], col)
		if err := s21Want("record_index_fields: UPDATE "+col+" of tenant A's own row", granted, n, err, 1); err != nil {
			return err
		}
		if granted {
			if n, err := execOne(ctx, tx, `UPDATE record_index_fields SET `+col+` = `+col+` WHERE tenant_id = $1`, TenantB); err != nil || n != 0 {
				return fmt.Errorf("record_index_fields: UPDATE %s of tenant B's rows: %d rows, %w; want 0", col, n, err)
			}
		}
	}
	const insertField = `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, requested_version)
		VALUES ($1, gen_random_uuid(), $2, 'iso_s21_probe', 'text', 'backfilling', 1)`
	n, err = execOne(ctx, tx, insertField, TenantA, ticketDefA)
	if err := s21Want("record_index_fields: INSERT tenant A's own row", has("INSERT"), n, err, 1); err != nil {
		return err
	}
	for _, c := range [][2]string{{TenantB, EntityDefinitionB1}, {TenantA, EntityDefinitionB1}} {
		n, err := execOne(ctx, tx, insertField, c[0], c[1])
		if err := s21Denied("record_index_fields: INSERT tenant "+c[0]+" on tenant B's type", n, err); err != nil {
			return err
		}
	}
	n, err = execOne(ctx, tx, `DELETE FROM record_index_fields WHERE id = $1`, FieldRemovingB)
	if has("DELETE") {
		if err != nil || n != 0 {
			return fmt.Errorf("record_index_fields: DELETE tenant B's row: %d rows, %w; want 0", n, err)
		}
		// Tenant A's own row with side rows: the grant and the policy pass, so
		// the registry foreign key of D3 is what refuses it.
		if _, err := execOne(ctx, tx, `DELETE FROM record_index_fields WHERE tenant_id = $1 AND id = $2`, TenantA, fieldBoolA); sqlState(err) != "23503" {
			return fmt.Errorf("control: record_index_fields: DELETE tenant A's row with side rows = %w, want 23503", err)
		}
	} else if err := refused("record_index_fields: DELETE", err); err != nil {
		return err
	}
	return nil
}

// Row 5. While tenant A's qty is backfilling at 37%, every filter on it is
// the same 422 field_not_ready: for a value the admin's records hold, one
// only an unreadable record holds, one only tenant B holds and one nobody
// holds; for the admin and the desk; before and after more records of both
// tenants match; and whatever tenant B's own progress is. The body has the
// problem members and index_progress only, and no number but 422 and 37.
func s21NotReadyNoCounts(ctx context.Context, e *etIso, _ *s21Fixture) error {
	if err := s21SetState(ctx, e, TenantA, "qty", "backfilling", 37); err != nil {
		return err
	}
	if err := s21SetState(ctx, e, TenantB, "qty", "backfilling", 88); err != nil {
		return err
	}
	queries := []string{"filter[qty]=10", "filter[qty]=4242", "filter[qty]=9999", "filter[qty]=1", "filter[qty][gt]=0",
		"filter[qty][in]=10,4242", "filter[qty][lt]=0", "filter[note]=a-desk&filter[qty]=10"}
	var want s21Answer
	round := func(when string) error {
		for _, token := range []string{s21Admin, s21Desk} {
			for _, q := range queries {
				if err := s21NotReady(ctx, e, token, q, 37); err != nil {
					return fmt.Errorf("%s: %w", when, err)
				}
				a, err := s21Ask(ctx, e, token, q)
				if err != nil {
					return err
				}
				if want.body == "" {
					want = a
					if err := s21OnlyProgress(a.body); err != nil {
						return err
					}
				}
				if a != want {
					return fmt.Errorf("%s: %s ?%s differs from %s ?%s:\n%s\n%s", when, token, q, s21Admin, queries[0], a.body, want.body)
				}
			}
		}
		return nil
	}
	if err := round("before"); err != nil {
		return err
	}
	for i := range 3 {
		if _, err := s21Create(ctx, e, s21Admin, fmt.Sprintf(`{"data": {"note": "more-%d", "qty": 10, "group": %q}}`, i, auditorGroupA)); err != nil {
			return err
		}
		if _, err := s21Create(ctx, e, auditCallerBAgent, fmt.Sprintf(`{"data": {"note": "more-%d", "qty": 10}}`, i)); err != nil {
			return err
		}
	}
	if err := s21SetState(ctx, e, TenantB, "qty", "ready", 100); err != nil {
		return err
	}
	return round("after six more records and tenant B ready")
}

var s21Numbers = regexp.MustCompile(`[0-9]+`)

// s21OnlyProgress: the 422 body has the problem members and index_progress,
// and no number but its status and the percentage.
func s21OnlyProgress(body string) error {
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return err
	}
	for k := range m {
		switch k {
		case "type", "title", "status", "code", "detail", "index_progress":
		default:
			return fmt.Errorf("field_not_ready carries member %q: %s", k, body)
		}
	}
	for _, n := range s21Numbers.FindAllString(body, -1) {
		if n != "422" && n != "37" {
			return fmt.Errorf("field_not_ready carries the number %s: %s", n, body)
		}
	}
	return nil
}
