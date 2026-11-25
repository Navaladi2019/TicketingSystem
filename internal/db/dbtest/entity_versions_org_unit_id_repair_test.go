//go:build integration

package dbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/repairtest"
)

// TICK-1125: the operator's pre-upgrade check and repair, run from
// docs/data/entity-versions-org-unit-id-reserved.md on a database of the
// release before S31. The numbers in the test comments are the issue's.

const (
	orgUnitIDRepairDoc = "../../../docs/data/entity-versions-org-unit-id-reserved.md"
	// repairOldName and repairNewName are the property's name before and after.
	repairOldName   = "org_unit_id"
	repairNewName   = "old_unit_id"
	repairSourceRef = "CHG-1125/run-1"
	// repairMarker is in every refused schema and value; no message may hold it.
	repairMarker = "marker-1125"
	// repairType is a type whose property is a plain string.
	repairType = `{"type":"object","additionalProperties":false,"required":["state"],"properties":` +
		`{"state":{"type":"string","enum":["new","done"]},"org_unit_id":{"title":"` + repairMarker + `","type":"string"}}}`
	repairClean = `{"type":"object","additionalProperties":false,"required":["state"],"properties":` +
		`{"state":{"type":"string","enum":["new","done"]}}}`
)

// preS31Cluster is a database of the release before S31, with its grants.
func preS31Cluster(t *testing.T) (*grants.Manifest, *pgx.Conn) {
	t.Helper()
	m := freshCluster(t)
	if _, err := migrateFS(t, m, migrationsThrough(t, lastBeforeS31)); err != nil {
		t.Fatalf("migrate through %s: %v", lastBeforeS31, err)
	}
	return m, superNava(t)
}

func repairBlock(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(orgUnitIDRepairDoc)
	if err != nil {
		t.Fatal(err)
	}
	sql, err := repairtest.Block(strings.ReplaceAll(string(b), "\r\n", "\n"), name)
	if err != nil {
		t.Fatal(err)
	}
	return sql
}

// repairVersion is one stored version; fixed "" maps state only.
type repairVersion struct{ schema, fixed string }

// repairRecord is one stored record with its create entry. Its id is set by
// repairSeed.
type repairRecord struct {
	data      string
	requester bool
	// entries is the number of history entries, 0 meaning the create only.
	entries int
}

// repairFixture is one tenant with one type, as an earlier release stored it.
type repairFixture struct {
	tenant, typ string
	records     []string
}

func inReplica(t *testing.T, super *pgx.Conn, fn func()) {
	t.Helper()
	exec(t, super, "SET session_replication_role = replica")
	fn()
	exec(t, super, "SET session_replication_role = DEFAULT")
}

func repairSeed(t *testing.T, super *pgx.Conn, versions []repairVersion, records ...repairRecord) *repairFixture {
	t.Helper()
	f := &repairFixture{tenant: uuid.NewString(), typ: uuid.NewString()}
	inReplica(t, super, func() {
		exec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", f.tenant)
		exec(t, super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
			VALUES ($1::uuid, $1, $1::text, 'u@example.test', true, '[]', now())`, f.tenant)
		exec(t, super, `INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
			VALUES ($1, $2, 'ticket', 'Ticket', 'TCK', $3)`, f.tenant, f.typ, len(versions))
		for i, v := range versions {
			f.addVersion(t, super, i+1, v)
		}
		for i, r := range records {
			f.records = append(f.records, uuid.NewString())
			var requester any
			if r.requester {
				requester = f.tenant
			}
			exec(t, super, `INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, requester_id, data, version,
				created_at, updated_at) VALUES ($1, $2, $3, 1, $4, 'new', $5, $6::text::jsonb, $7, now() - interval '2 days', now() - interval '1 day')`,
				f.tenant, f.records[i], f.typ, fmt.Sprintf("TCK-%d", i+1), requester, r.data, max(r.entries, 1))
			// A value too big for one create entry came in a second entry.
			if len(r.data) > 60000 {
				f.addEntry(t, super, f.records[i], "create", 1, `[{"op":"add","path":"","value":{"state":"new"}}]`)
				f.addEntry(t, super, f.records[i], "update", 2, queryString(t, super, `SELECT '[{"op":"add","path":"/org_unit_id","value":' ||
					(data -> 'org_unit_id')::text || '}]' FROM records WHERE tenant_id = $1 AND id = $2`, f.tenant, f.records[i]))
				exec(t, super, "UPDATE records SET version = 2 WHERE id = $1", f.records[i])
				continue
			}
			f.addEntry(t, super, f.records[i], "create", 1, `[{"op":"add","path":"","value":`+queryString(t, super,
				"SELECT (data || jsonb_build_object('state', state))::text FROM records WHERE tenant_id = $1 AND id = $2", f.tenant, f.records[i])+`}]`)
			if r.entries > 1 {
				exec(t, super, `INSERT INTO audit_log (tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id,
					version_before, version_after, actor_system, source)
					SELECT $1, gen_random_uuid(), now() - interval '1 hour' + g * interval '1 ms', 'record', 'update', 'record', $2, $3, g, g + 1, 'system', 'system'
					FROM generate_series(1, $4::int - 1) g`, f.tenant, f.records[i], f.typ, r.entries)
			}
		}
	})
	return f
}

func (f *repairFixture) addVersion(t *testing.T, super *pgx.Conn, version int, v repairVersion) {
	t.Helper()
	compat, fixed := "compatible", v.fixed
	if version == 1 {
		compat = "initial"
	}
	if fixed == "" {
		fixed = `{"state": "state"}`
	}
	exec(t, super, `INSERT INTO entity_definition_versions
		(tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by, published_at)
		VALUES ($1, gen_random_uuid(), $2, $3, $4::text::json, $5, $6::text::jsonb,
			'{"initial": "new", "states": ["new", "done"], "transitions": [{"from": "new", "to": "done"}]}', $7, $1, now() - interval '3 days')`,
		f.tenant, f.typ, version, v.schema, schemaSum(v.schema), fixed, compat)
}

// addEntry stores a record entry a day before this transaction.
func (f *repairFixture) addEntry(t *testing.T, super *pgx.Conn, record, action string, version int, diff string) {
	t.Helper()
	truncated := len(diff) > audit.MaxDiffBytes
	if truncated {
		diff = "[]"
	}
	salt := bytes.Repeat([]byte{7}, 16)
	sum := chain.DiffSHA256([16]byte(salt), []byte(diff))
	exec(t, super, `INSERT INTO audit_log (tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id,
		version_before, version_after, actor_system, source, diff, diff_salt, diff_sha256, diff_truncated)
		VALUES ($1, gen_random_uuid(), now() - interval '1 day' + $5::int * interval '1 ms', 'record', $4, 'record', $2, $3, NULLIF($5::int - 1, 0), $5,
			'system', 'system', $6::text::json, $7, $8, $9)`, f.tenant, record, f.typ, action, version, diff, salt, sum[:], truncated)
}

func (f *repairFixture) inputs() repairtest.Inputs {
	return repairtest.Inputs{TenantID: f.tenant, TypeID: f.typ, NewName: repairNewName, SourceRef: repairSourceRef, RecordLimit: "100000"}
}

func paste(t *testing.T, conn *pgx.Conn, sql string, onErrorRollback bool) repairtest.Output {
	t.Helper()
	out, err := repairtest.Paste(context.Background(), conn, sql, onErrorRollback)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// pasteOne runs paste 1 with in and leaves the transaction as it is.
func pasteOne(t *testing.T, conn *pgx.Conn, in repairtest.Inputs) repairtest.Output {
	t.Helper()
	sql, err := repairtest.WithInputs(repairBlock(t, repairtest.PasteOne), in)
	if err != nil {
		t.Fatal(err)
	}
	return paste(t, conn, sql, false)
}

func pasteTwo(t *testing.T, conn *pgx.Conn) error {
	t.Helper()
	return paste(t, conn, repairBlock(t, repairtest.PasteTwo), false).Err
}

// repair runs both pastes, as an operator who pastes on does, and returns the
// first error.
func repair(t *testing.T, conn *pgx.Conn, in repairtest.Inputs) error {
	t.Helper()
	one := pasteOne(t, conn, in)
	two := pasteTwo(t, conn)
	if s := conn.PgConn().TxStatus(); s != 'I' {
		t.Fatalf("transaction status %c after paste 2", s)
	}
	if one.Err != nil {
		return one.Err
	}
	return two
}

func mustRepair(t *testing.T, conn *pgx.Conn, in repairtest.Inputs) {
	t.Helper()
	if err := repair(t, conn, in); err != nil {
		t.Fatalf("repair: %v %s", err, detail(err))
	}
}

func detail(err error) string {
	if pe, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pe.Detail
	}
	return ""
}

// repairTables hold everything a repair could write.
func repairTables() []string {
	return []string{"entity_definitions", "entity_definition_versions", "records", "audit_log", "acl_versions",
		"workflow_definitions", "workflow_definition_versions", "workflow_ref"}
}

// tenantState is a digest of every row of one tenant, or of all for "".
func tenantState(t *testing.T, super *pgx.Conn, tenant string) string {
	t.Helper()
	var parts []string
	for _, table := range repairTables() {
		parts = append(parts, "SELECT '"+table+"' || x::text AS r FROM "+table+" x WHERE $1 = '' OR tenant_id::text = $1")
	}
	return queryString(t, super, "SELECT md5(string_agg(r, '|' ORDER BY r)) FROM ("+strings.Join(parts, " UNION ALL ")+") s", tenant)
}

// triggerModes is every trigger with its mode.
func triggerModes(t *testing.T, super *pgx.Conn) string {
	t.Helper()
	return queryString(t, super, "SELECT string_agg(tgrelid::regclass || '.' || tgname || tgenabled::text, ',' ORDER BY tgrelid, tgname) FROM pg_trigger")
}

// wantRefused runs both pastes and checks the refusal: its prefix and
// SQLSTATE, that its DETAIL names want, that it shows no tenant data, and that
// nothing was written.
func wantRefused(t *testing.T, super *pgx.Conn, in repairtest.Inputs, prefix, want string) {
	t.Helper()
	before := tenantState(t, super, "")
	err := repair(t, superNava(t), in)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "P0001" || !strings.HasPrefix(pe.Message, prefix) || !strings.Contains(pe.Detail, want) {
		t.Errorf("repair = %v, DETAIL %q; want P0001 %s naming %q", err, detail(err), prefix, want)
	} else if strings.Contains(pe.Message+pe.Detail+pe.Hint+pe.Where, repairMarker) {
		t.Errorf("the refusal shows tenant data: %+v", pe)
	}
	if tenantState(t, super, "") != before {
		t.Errorf("%s: the refused repair wrote something", prefix)
	}
}

func versionRows(schemas ...string) []repairVersion {
	out := make([]repairVersion, len(schemas))
	for i, s := range schemas {
		out[i] = repairVersion{schema: s}
	}
	return out
}

// withProp is repairType with the property's schema replaced; withRoot adds
// root keywords.
func withProp(prop string) string {
	return strings.Replace(repairType, `{"title":"`+repairMarker+`","type":"string"}`, prop, 1)
}

func withRoot(keywords string) string {
	return strings.Replace(repairType, `{"type":"object",`, `{"type":"object",`+keywords+`,`, 1)
}

func withOther(props string) string {
	return strings.Replace(repairType, `"properties":{`, `"properties":{`+props+`,`, 1)
}

func (f *repairFixture) addWorkflow(t *testing.T, super *pgx.Conn, current int, bodies ...string) {
	t.Helper()
	id := uuid.NewString()
	inReplica(t, super, func() {
		exec(t, super, `INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version)
			VALUES ($1, $2, 'flow', $3, $4)`, f.tenant, id, f.typ, current)
		for i, b := range bodies {
			exec(t, super, `INSERT INTO workflow_definition_versions (tenant_id, id, definition_id, version, body, body_sha256, step_ids,
				trigger_on, trigger_entity_definition_id) VALUES ($1, gen_random_uuid(), $2, $3, $4::text::json, $5, '{}', 'record.created', $6)`,
				f.tenant, id, i+1, b, schemaSum(b), f.typ)
		}
	})
}

func (f *repairFixture) addRun(t *testing.T, super *pgx.Conn, record string, closed bool) {
	t.Helper()
	inReplica(t, super, func() {
		exec(t, super, `INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind, closed_at)
			VALUES ($1, gen_random_uuid(), $2, 'nava.interpreter', CASE WHEN $3 THEN now() END)`, f.tenant, record, closed)
	})
}

func (f *repairFixture) addMasks(t *testing.T, super *pgx.Conn, version int, fields ...string) {
	t.Helper()
	masks := []map[string]any{}
	for _, name := range fields {
		masks = append(masks, map[string]any{"field": name, "read": map[string]any{}, "write": map[string]any{}})
	}
	b, _ := json.Marshal(masks)
	inReplica(t, super, func() {
		exec(t, super, `INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
			VALUES ($1, $2, $3, '[]', $4::text::jsonb)`, f.tenant, f.typ, version, string(b))
	})
}

func workflowSet(field string) string {
	return `{"steps":[{"key":"a","type":"update_record","set":{"` + field + `":"x"}}]}`
}

func workflowField(field string) string {
	return `{"steps":[{"key":"a","type":"condition","field":"/` + field + `","op":"ne","value":"x"}]}`
}

// repairRefusal is one type the repair refuses; seed may add to it.
type repairRefusal struct {
	name, prefix, want string
	versions           []repairVersion
	records            []repairRecord
	seed               func(t *testing.T, super *pgx.Conn, f *repairFixture)
	limit              string
}

// repairRefusals are the cases of refusals R1 to R8, R10 and R11 (3). R9 is in
// TestTICK1125Pastes.
func repairRefusals() []repairRefusal {
	one := versionRows(repairType)
	valued := []repairRecord{{data: `{"org_unit_id": "` + repairMarker + `"}`}}
	masked := func(versions ...[]string) func(*testing.T, *pgx.Conn, *repairFixture) {
		return func(t *testing.T, super *pgx.Conn, f *repairFixture) {
			for i, fields := range versions {
				f.addMasks(t, super, i+1, fields...)
			}
		}
	}
	flow := func(current int, bodies ...string) func(*testing.T, *pgx.Conn, *repairFixture) {
		return func(t *testing.T, super *pgx.Conn, f *repairFixture) { f.addWorkflow(t, super, current, bodies...) }
	}
	entry := func(diff string) func(*testing.T, *pgx.Conn, *repairFixture) {
		return func(t *testing.T, super *pgx.Conn, f *repairFixture) {
			inReplica(t, super, func() { f.addEntry(t, super, f.records[0], "update", 2, diff) })
		}
	}
	mapped := `{"state": "state", "org_unit_id": "requester_id"}`
	out := []repairRefusal{
		{name: "a mask on the old name", prefix: "R1:", want: "access version 1", versions: one, seed: masked([]string{"org_unit_id"})},
		{name: "a mask on the new name", prefix: "R1:", want: "access version 2", versions: one, seed: masked(nil, []string{"state", repairNewName})},

		{name: "the current workflow sets the old name", prefix: "R2:", want: "version 1", versions: one, seed: flow(1, workflowSet(repairOldName))},
		{name: "the current workflow tests the old name", prefix: "R2:", want: "version 1", versions: one, seed: flow(1, workflowField(repairOldName))},
		{name: "the current workflow tests below the old name", prefix: "R2:", want: "version 1", versions: one, seed: flow(1, workflowField(repairOldName+"/x"))},
		{name: "a branch case tests the old name", prefix: "R2:", want: "version 1", versions: one,
			seed: flow(1, `{"steps":[{"key":"a","type":"branch","cases":[{"field":"/org_unit_id","op":"eq","value":1,"next":"b"}]}]}`)},
		{name: "a condition names only the new name", prefix: "R2:", want: "version 1", versions: one, seed: flow(1, workflowField(repairNewName))},
		{name: "the current workflow sets the new name", prefix: "R2:", want: "version 1", versions: one, seed: flow(1, workflowSet(repairNewName))},
		{name: "a workflow body jsonb cannot read", prefix: "R2:", want: "version 1", versions: one, seed: flow(1, esc(`{"steps":["~u0000"]}`))},
		{name: "an older workflow version with an open run", prefix: "R2:", want: "version 1", versions: one, records: valued,
			seed: func(t *testing.T, super *pgx.Conn, f *repairFixture) {
				f.addWorkflow(t, super, 2, workflowSet(repairOldName), `{"steps":[]}`)
				f.addRun(t, super, f.records[0], false)
			}},

		{name: "x-nava-secret", prefix: "R3:", want: "version 1", versions: versionRows(withProp(`{"type":"string","x-nava-secret":true}`))},
		{name: "writeOnly in an older version", prefix: "R3:", want: "version 1",
			versions: versionRows(withProp(`{"type":"string","writeOnly":true}`), repairType)},

		{name: "the new name is a property", prefix: "R4:", want: "version 2", versions: versionRows(repairType, withOther(`"old_unit_id":{}`))},
		{name: "the new name is in a record", prefix: "R4:", want: "record ", versions: one, records: []repairRecord{{data: `{"old_unit_id": 1}`}}},
		{name: "the new name in an entry's path", prefix: "R4:", want: "entry ", versions: one, records: valued,
			seed: entry(`[{"op":"remove","path":"/old_unit_id/x"}]`)},
		{name: "the new name in an entry's from", prefix: "R4:", want: "entry ", versions: one, records: valued,
			seed: entry(`[{"op":"move","from":"/old_unit_id","path":"/x"}]`)},
		{name: "the new name in a root op", prefix: "R4:", want: "entry ", versions: one, records: valued,
			seed: entry(`[{"op":"replace","path":"","value":{"old_unit_id":1}}]`)},

		{name: "a percent-encoded reference", prefix: "R5:", want: "version 1", versions: versionRows(withOther(`"b":{"$ref":"#/properties/org%5Funit_id"}`))},
		{name: "a percent-encoded dynamic reference", prefix: "R5:", want: "version 1", versions: versionRows(withOther(`"b":{"$dynamicRef":"#%61"}`))},
		{name: "a reference", prefix: "R5:", want: "version 1", versions: versionRows(withOther(`"b":{"$ref":"#/properties/org_unit_id"}`))},
		{name: "a reference with an escaped letter", prefix: "R5:", want: "version 1",
			versions: versionRows(withOther(esc(`"b":{"$ref":"#/properties/~u006frg_unit_id"}`)))},
		{name: "allOf requires the old name", prefix: "R5:", want: "version 1", versions: versionRows(withRoot(`"allOf":[{"required":["org_unit_id"]}]`))},
		{name: "dependentRequired names only the new name", prefix: "R5:", want: "version 1", versions: versionRows(withRoot(`"dependentRequired":{"state":["old_unit_id"]}`))},
		{name: "if names only the new name", prefix: "R5:", want: "version 1", versions: versionRows(withRoot(`"if":{"properties":{"old_unit_id":{"const":1}}},"then":false`))},
		{name: "the new name required", prefix: "R5:", want: "version 1", versions: versionRows(strings.Replace(repairType, `["state"]`, `["state","old_unit_id"]`, 1))},
		{name: "a clean older version names the old name", prefix: "R5:", want: "version 1",
			versions: versionRows(strings.Replace(repairClean, `{"type":"object",`, `{"type":"object","allOf":[{"required":["org_unit_id"]}],`, 1), repairType)},
		{name: "propertyNames", prefix: "R5:", want: "version 1", versions: versionRows(withRoot(`"propertyNames":{"maxLength":40}`))},
		{name: "patternProperties", prefix: "R5:", want: "version 1", versions: versionRows(withOther(`"b":{"patternProperties":{"^x":{}}}`))},
		{name: "a nested org_unit_id", prefix: "R5:", want: "version 1", versions: versionRows(withOther(`"b":{"properties":{"org_unit_id":{}}}`))},
		{name: "a nested org_unit_id in the property", prefix: "R5:", want: "version 1", versions: versionRows(withProp(`{"properties":{"org_unit_id":{}}}`))},
		{name: "another property with the name in its name", prefix: "R5:", want: "version 1", versions: versionRows(withOther(`"my_org_unit_id":{}`))},
		{name: "a root key the audit masker reads as properties", prefix: "R5:", want: "version 1",
			versions: versionRows(withRoot(`"Properties":{"org_unit_id":{"x-nava-secret":true}}`))},
		{name: "a property named title that refers to it", prefix: "R5:", want: "version 1", versions: versionRows(withOther(`"title":{"$ref":"#/properties/org_unit_id"}`))},

		{name: "a NUL escape", prefix: "R6:", want: "version 1", versions: versionRows(withRoot(esc(`"description":"x~u0000"`)))},
		{name: "a lone surrogate", prefix: "R6:", want: "version 1", versions: versionRows(withRoot(esc(`"description":"x~ud800"`)))},
		{name: "a NUL escape in a clean version", prefix: "R6:", want: "version 1", versions: versionRows(strings.Replace(repairClean, `{"type":"object",`, esc(`{"type":"object","title":"~u0000",`), 1), repairType)},
		{name: "a duplicate key", prefix: "R6:", want: "version 1", versions: versionRows(withRoot(`"title":"a","title":"b"`))},
		{name: "a nested duplicate key", prefix: "R6:", want: "version 1", versions: versionRows(withOther(`"b":{"title":"a","title":"b"}`))},
		{name: "a number jsonb cannot hold", prefix: "R6:", want: "version 1", versions: versionRows(withOther(`"b":{"maximum":1e200000}`))},
		{name: "R6 before a mask", prefix: "R6:", want: "version 1", versions: versionRows(withRoot(esc(`"description":"x~u0000"`))), seed: masked([]string{"org_unit_id"})},

		{name: "different fixed_columns", prefix: "R8:", want: "version 1, version 2",
			versions: []repairVersion{{repairType, ""}, {repairType, `{"state": "state", "b": "assignee_id"}`}}},
		{name: "a clean version maps the property", prefix: "R8:", want: "version 1",
			versions: []repairVersion{{repairClean, mapped}, {repairType, mapped}}},
		{name: "a column the repair does not know", prefix: "R8:", want: "version 1",
			versions: []repairVersion{{repairType, `{"state": "state", "org_unit_id": "created_by"}`}}},

		{name: "more records than the limit", prefix: "R10:", want: "2 records, limit 1", versions: one, records: append(valued, valued...), limit: "1"},
		{name: "a mapped property also in data", prefix: "R11:", want: "record ",
			versions: []repairVersion{{repairType, mapped}}, records: []repairRecord{{data: `{"org_unit_id": "` + repairMarker + `"}`, requester: true}}},
	}
	value := func(name, v string) {
		out = append(out, repairRefusal{name: "a value: " + name, prefix: "R7:", want: "record ", versions: one, records: []repairRecord{{data: `{"org_unit_id": ` + v + `}`}}})
		out = append(out, repairRefusal{name: "a schema value: " + name, prefix: "R7:", want: "version 1", versions: versionRows(withProp(`{"default":`+v+`}`), repairType)})
	}
	value("a URL", `"https://u:p@`+repairMarker+`"`)
	value("a URL across nested keys", `{"https:":{"":{"u:p@h":1}}}`)
	value("a tab in the URL", esc(`"ht~ttps:/~t/u:p@h"`))
	value("a line feed in the URL", esc(`"ht~ntps:/~n/u:p@h"`))
	value("a carriage return in the URL", esc(`"ht~rtps:/~r/u:p@h"`))
	value("the lax form", `"https:u:p x@h"`)
	value("an array", `["u@h"]`)
	for _, token := range []string{"AKIA", "ASIA", "ghp_", "gho_", "ghu_", "ghs_", "ghr_", "github_pat_", "xox", "-----BEGIN", "eyJ"} {
		out = append(out,
			repairRefusal{name: "a schema string with " + token, prefix: "R7:", want: "version 1", versions: versionRows(withProp(`{"description":"a ` + token + `b"}`))},
			repairRefusal{name: "a schema key with " + token, prefix: "R7:", want: "version 1", versions: versionRows(withProp(`{"properties":{"`+token+`b":{}}}`), repairType)})
	}
	return out
}

// Tests 3 and 4, and the inputs of 20: every refusal names its rows and writes
// nothing.
func TestTICK1125Refuses(t *testing.T) {
	_, super := preS31Cluster(t)
	for _, c := range repairRefusals() {
		t.Run(strings.TrimSuffix(c.prefix, ":")+"/"+c.name, func(t *testing.T) {
			f := repairSeed(t, super, c.versions, c.records...)
			if c.seed != nil {
				c.seed(t, super, f)
			}
			in := f.inputs()
			if c.limit != "" {
				in.RecordLimit = c.limit
			}
			wantRefused(t, super, in, c.prefix, c.want)
		})
	}
	f := repairSeed(t, super, versionRows(repairType), repairRecord{data: `{"org_unit_id": "a"}`})
	other := repairSeed(t, super, versionRows(repairType))
	for _, c := range []struct {
		name, message string
		edit          func(*repairtest.Inputs)
	}{
		{"another tenant's type", "type ", func(in *repairtest.Inputs) { in.TypeID = other.typ }},
		{"another tenant's id", "type ", func(in *repairtest.Inputs) { in.TenantID = other.tenant }},
		{"the placeholder tenant", "tenant_id ", func(in *repairtest.Inputs) { in.TenantID = "<tenant id>" }},
		{"an upper-case type id", "type_id ", func(in *repairtest.Inputs) { in.TypeID = strings.ToUpper("abcdef" + in.TypeID[6:]) }},
		{"an empty type id", "type_id ", func(in *repairtest.Inputs) { in.TypeID = "" }},
		{"a platform field as the name", "new_name ", func(in *repairtest.Inputs) { in.NewName = "state" }},
		{"the old name", "new_name ", func(in *repairtest.Inputs) { in.NewName = repairOldName }},
		{"a name of 12 bytes", "new_name ", func(in *repairtest.Inputs) { in.NewName = "old_unit_ids" }},
		{"a name with an upper-case letter", "new_name ", func(in *repairtest.Inputs) { in.NewName = "old_Unit" }},
		{"a name with a line feed", "new_name ", func(in *repairtest.Inputs) { in.NewName = "old_u\n" }},
		{"the placeholder source_ref", "source_ref ", func(in *repairtest.Inputs) { in.SourceRef = "<change ticket>/<run id>" }},
		{"a source_ref with no run", "source_ref ", func(in *repairtest.Inputs) { in.SourceRef = "CHG-1" }},
		{"a source_ref with a name in it", "source_ref ", func(in *repairtest.Inputs) { in.SourceRef = "CHG-1/a b" }},
		{"a limit that is no number", "record_limit ", func(in *repairtest.Inputs) { in.RecordLimit = "all" }},
		{"a negative limit", "record_limit ", func(in *repairtest.Inputs) { in.RecordLimit = "-1" }},
	} {
		t.Run("input/"+c.name, func(t *testing.T) {
			in := f.inputs()
			c.edit(&in)
			wantRefused(t, super, in, c.message, "")
		})
	}
	// The controls: what the cases above are one step away from is repaired.
	for _, c := range []repairRefusal{
		{name: "an older access version masks it", versions: versionRows(repairType), seed: func(t *testing.T, super *pgx.Conn, f *repairFixture) {
			f.addMasks(t, super, 1, repairOldName)
			f.addMasks(t, super, 2)
		}},
		{name: "an older workflow version and a closed run", versions: versionRows(repairType), records: []repairRecord{{data: `{"org_unit_id": "a"}`}},
			seed: func(t *testing.T, super *pgx.Conn, f *repairFixture) {
				f.addWorkflow(t, super, 2, workflowSet(repairOldName), `{"steps":[]}`)
				f.addRun(t, super, f.records[0], true)
			}},
		{name: "writeOnly false", versions: versionRows(withProp(`{"type":"string","writeOnly":false}`))},
		{name: "the name in annotations", versions: versionRows(withRoot(`"title":"org_unit_id","description":"old_unit_id","$comment":"org_unit_id"`))},
		{name: "values near R7", versions: versionRows(repairType), records: []repairRecord{
			{data: `{"org_unit_id": "a:b"}`}, {data: `{"org_unit_id": "a@b"}`}, {data: `{"org_unit_id": {"a:": "b"}}`},
			{data: `{"org_unit_id": null}`}, {data: `{"other": "https://u:p@h"}`}}},
		{name: "as many records as the limit", versions: versionRows(repairType), records: []repairRecord{{data: `{"org_unit_id": 1}`}}, limit: "1"},
	} {
		t.Run("repaired/"+c.name, func(t *testing.T) {
			f := repairSeed(t, super, c.versions, c.records...)
			if c.seed != nil {
				c.seed(t, super, f)
			}
			in := f.inputs()
			if c.limit != "" {
				in.RecordLimit = c.limit
			}
			mustRepair(t, superNava(t), in)
			wantNoOldName(t, super, f)
		})
	}
}

// wantNoOldName fails if a version or a record of the type still has it.
func wantNoOldName(t *testing.T, super *pgx.Conn, f *repairFixture) {
	t.Helper()
	if got := queryString(t, super, `SELECT (SELECT count(*) FROM entity_definition_versions
			WHERE definition_id = $1 AND schema -> 'properties' -> 'org_unit_id' IS NOT NULL) || ' versions, ' ||
		(SELECT count(*) FROM records WHERE entity_definition_id = $1 AND data ? 'org_unit_id') || ' records'`, f.typ); got != "0 versions, 0 records" {
		t.Errorf("after the repair: %s with the old name", got)
	}
}

// repairEntry is one entry the repair wrote.
type repairEntry struct {
	ID, Domain, Action, SubjectType, SubjectID, EntityDefinitionID, Before, After string
	Actor, Source, SourceRef, Content, Salt, Diff                                 string
	Truncated                                                                     bool
}

// repairEntries are the entries of the repair, with diff_sha256 checked.
func repairEntries(t *testing.T, super *pgx.Conn, f *repairFixture, where string) []repairEntry {
	t.Helper()
	rows, err := super.Query(context.Background(), `SELECT id::text, domain, action, subject_type, subject_id::text,
		coalesce(entity_definition_id::text, ''), coalesce(version_before::text, ''), coalesce(version_after::text, ''),
		coalesce(actor_system, ''), source, coalesce(source_ref, ''), coalesce(encode(content_sha256, 'hex'), ''), encode(diff_salt, 'hex'), diff::text,
		diff_truncated, diff_sha256 = sha256(diff_salt || convert_to(diff::text, 'UTF8')) AND NOT backfilled AND seq IS NULL
		FROM audit_log WHERE tenant_id = $1 AND actor_system = 'operator' AND `+where+` ORDER BY version_after NULLS FIRST, id`, f.tenant)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []repairEntry
	for rows.Next() {
		var (
			e  repairEntry
			ok bool
		)
		if err := rows.Scan(&e.ID, &e.Domain, &e.Action, &e.SubjectType, &e.SubjectID, &e.EntityDefinitionID, &e.Before, &e.After, &e.Actor,
			&e.Source, &e.SourceRef, &e.Content, &e.Salt, &e.Diff, &e.Truncated, &ok); err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("entry %s: diff_sha256 does not match, or it is backfilled or sealed", e.ID)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// sameOps fails unless got has the ops, the order and the values of
// jsondiff.Diff(before, after).
func sameOps(t *testing.T, what, got, before, after string) {
	t.Helper()
	want, err := jsondiff.Diff(json.RawMessage(before), json.RawMessage(after))
	if err != nil {
		t.Fatal(err)
	}
	decode := func(b []byte) (ops []map[string]any) {
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		if err := d.Decode(&ops); err != nil {
			t.Fatalf("%s: %v in %s", what, err, b)
		}
		return ops
	}
	if g, w := decode([]byte(got)), decode(want); !reflect.DeepEqual(g, w) {
		t.Errorf("%s: diff %s, want the ops of %s", what, got, want)
	}
}

func recordJSON(t *testing.T, super *pgx.Conn, id string) string {
	t.Helper()
	return queryString(t, super, "SELECT (to_jsonb(r) - 'version' - 'data')::text FROM records r WHERE id = $1", id)
}

// payload is the record's payload as an audit diff sees it.
func payload(t *testing.T, super *pgx.Conn, id string) string {
	t.Helper()
	return queryString(t, super, "SELECT data::text FROM records WHERE id = $1", id)
}

// replayed rebuilds the record's history from its stored entries.
func replayed(t *testing.T, super *pgx.Conn, record string) []read.Change {
	t.Helper()
	rows, err := super.Query(context.Background(), `SELECT action, diff_truncated, diff::text FROM audit_log
		WHERE subject_type = 'record' AND subject_id = $1 ORDER BY occurred_at, id`, record)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var entries []read.Entry
	for rows.Next() {
		var (
			e    read.Entry
			diff *string
		)
		if err := rows.Scan(&e.Action, &e.DiffTruncated, &diff); err != nil {
			t.Fatal(err)
		}
		if diff != nil {
			e.Diff = json.RawMessage(*diff)
		}
		entries = append(entries, e)
	}
	return read.Replay(entries)
}

// laterUpdate writes one more update of the record as nava_app, as the
// pipeline does.
func laterUpdate(t *testing.T, conn *pgx.Conn, f *repairFixture, record, diff, data string) error {
	t.Helper()
	salt := bytes.Repeat([]byte{9}, 16)
	sum := chain.DiffSHA256([16]byte(salt), []byte(diff))
	_, err := execIn(conn, f.tenant, `WITH w AS (UPDATE records SET version = version + 1, data = $4::text::jsonb
			WHERE tenant_id = $1 AND id = $2 RETURNING version)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after,
			actor_system, source, diff, diff_salt, diff_sha256)
		SELECT $1, gen_random_uuid(), 'record', 'update', 'record', $2, $3, version - 1, version, 'system', 'system', $5::text::json, $6, $7 FROM w`,
		f.tenant, record, f.typ, data, diff, salt, sum[:])
	return err
}

// Tests 2 (salts), 7, 8, 9, 11, 12, 17, 21 and 22 (ops).
func TestTICK1125Repairs(t *testing.T) {
	_, super := preS31Cluster(t)
	app := asRole(t, "nava_app")
	bystander := repairSeed(t, super, versionRows(repairType, repairType), repairRecord{data: `{"org_unit_id": "a"}`})
	bystander.addMasks(t, super, 1, repairOldName)
	bystanderState := tenantState(t, super, bystander.tenant)

	t.Run("content", func(t *testing.T) {
		// An escaped key, whitespace, numbers and the order of both lists.
		older := esc(`{ "type": "object", "required": ["z", "~u006frg_unit_id", "state"], "additionalProperties": false, "title": "org_unit_id",
  "properties": { "z": {"maximum": 1e2, "default": 1.50}, "~u006frg_unit_id": { "type" : "string", "description": "org_unit_id" }, "caf~u00e9": {}, "state": {"type":"string"} } }`)
		olderWant := `{"type":"object","required":["z", "old_unit_id","state"],"additionalProperties":false,"title":"org_unit_id","properties":` +
			`{"z":{"maximum": 1e2, "default": 1.50},"old_unit_id":{ "type" : "string", "description": "org_unit_id" },"café":{},"state":{"type":"string"}}}`
		olderWant = strings.Replace(olderWant, `["z", "old`, `["z","old`, 1)
		f := repairSeed(t, super, versionRows(older, repairType),
			repairRecord{data: `{"org_unit_id": "unit 1", "n": 1.50}`}, repairRecord{data: `{"org_unit_id": {"b": [1, null], "a": 1e2}}`}, repairRecord{data: `{"n": 1}`})
		wantStored(t, super, f.tenant, 1, esc("~u006f"))
		meta := func() string {
			return queryString(t, super, `SELECT string_agg((to_jsonb(v) - 'schema' - 'schema_sha256')::text, ',' ORDER BY version) FROM entity_definition_versions v
				WHERE definition_id = $1`, f.typ) + queryString(t, super, "SELECT to_jsonb(d)::text FROM entity_definitions d WHERE id = $1", f.typ)
		}
		metaBefore := meta()
		var rowBefore, payBefore []string
		for _, id := range f.records {
			rowBefore, payBefore = append(rowBefore, recordJSON(t, super, id)), append(payBefore, payload(t, super, id))
		}
		stored := queryString(t, super, "SELECT md5(string_agg(a::text, '|' ORDER BY id)) FROM audit_log a WHERE tenant_id = $1", f.tenant)

		mustRepair(t, superNava(t), f.inputs())

		wantNoOldName(t, super, f)
		for v, want := range map[int]string{1: olderWant, 2: strings.Replace(repairType, repairOldName, repairNewName, 1)} {
			if got := queryString(t, super, "SELECT schema::text FROM entity_definition_versions WHERE definition_id = $1 AND version = $2", f.typ, v); got != want {
				t.Errorf("version %d\n got %s\nwant %s", v, got, want)
			}
			if !queryBool(t, super, `SELECT schema_sha256 = $3 FROM entity_definition_versions WHERE definition_id = $1 AND version = $2`, f.typ, v, schemaSum(want)) {
				t.Errorf("version %d: schema_sha256 is not the hash of its text", v)
			}
		}
		if got := meta(); got != metaBefore {
			t.Errorf("the rest of the type changed:\n got %s\nwant %s", got, metaBefore)
		}
		if got := queryString(t, super, `SELECT md5(string_agg(a::text, '|' ORDER BY id)) FROM audit_log a
			WHERE tenant_id = $1 AND actor_system IS DISTINCT FROM 'operator'`, f.tenant); got != stored {
			t.Error("a stored audit entry changed")
		}
		for i, id := range f.records {
			if got := recordJSON(t, super, id); got != rowBefore[i] {
				t.Errorf("record %d: %s, want state, updated_at and the rest as %s", i, got, rowBefore[i])
			}
			wantVersion, want := "2", strings.Replace(payBefore[i], `"`+repairOldName+`"`, `"`+repairNewName+`"`, 1)
			if i == 2 {
				wantVersion = "1"
			}
			if !queryBool(t, super, "SELECT version::text = $2 AND data = $3::text::jsonb FROM records WHERE id = $1", id, wantVersion, want) {
				t.Errorf("record %d: want version %s and %s, got %s", i, wantVersion, want, payload(t, super, id))
			}
		}

		versions := repairEntries(t, super, f, "domain = 'config'")
		if len(versions) != 2 {
			t.Fatalf("%d config entries, want 2", len(versions))
		}
		for i, before := range []string{older, repairType} {
			v := fmt.Sprint(i + 1)
			after := queryString(t, super, "SELECT schema::text FROM entity_definition_versions WHERE definition_id = $1 AND version = $2", f.typ, i+1)
			e := versions[i]
			sameOps(t, "version "+v, e.Diff, queryString(t, super, "SELECT $1::text::jsonb::text", before), queryString(t, super, "SELECT $1::text::jsonb::text", after))
			want := repairEntry{ID: e.ID, Domain: "config", Action: "update", SubjectType: "entity_definition", SubjectID: f.typ, Before: v, After: v, Actor: "operator",
				Source: "system", SourceRef: repairSourceRef, Content: fmt.Sprintf("%x", schemaSum(after)), Salt: e.Salt, Diff: e.Diff}
			if e != want {
				t.Errorf("version entry %+v, want %+v", e, want)
			}
		}
		records := repairEntries(t, super, f, "domain = 'record'")
		if len(records) != 2 {
			t.Fatalf("%d record entries, want 2", len(records))
		}
		for _, e := range records {
			i := slices.Index(f.records, e.SubjectID)
			sameOps(t, "record", e.Diff, payBefore[i], payload(t, super, e.SubjectID))
			want := repairEntry{ID: e.ID, Domain: "record", Action: "update", SubjectType: "record", SubjectID: e.SubjectID, EntityDefinitionID: f.typ, Before: "1", After: "2",
				Actor: "operator", Source: "system", SourceRef: repairSourceRef, Salt: e.Salt, Diff: e.Diff}
			if e != want {
				t.Errorf("record entry %+v, want %+v", e, want)
			}
		}
		// 9: the history replays across the repair and a later update.
		if err := laterUpdate(t, app, f, f.records[0], `[{"op":"replace","path":"/old_unit_id","value":"unit 2"}]`, `{"old_unit_id": "unit 2", "n": 1.50}`); err != nil {
			t.Fatal(err)
		}
		changes := replayed(t, super, f.records[0])
		if len(changes) != 3 || !changes[0].Known || !changes[1].Known || !changes[2].Known ||
			changes[1].Before[repairOldName] != "unit 1" || changes[1].After[repairNewName] != "unit 1" || changes[2].After[repairNewName] != "unit 2" {
			t.Errorf("replay = %+v", changes)
		}
		// 11: a second run says so and changes nothing.
		before := tenantState(t, super, "")
		conn, notices := noticeConn(t)
		mustRepair(t, conn, f.inputs())
		if tenantState(t, super, "") != before {
			t.Error("the second run wrote something")
		}
		if want := []string{"nothing to repair", "the plan is empty: nothing is written"}; !slices.Equal(*notices, want) {
			t.Errorf("the second run said %q, want %q", *notices, want)
		}
	})

	t.Run("salts", func(t *testing.T) {
		recs := make([]repairRecord, 50)
		for i := range recs {
			recs[i] = repairRecord{data: `{"org_unit_id": "same"}`}
		}
		f := repairSeed(t, super, versionRows(repairType, repairType), recs...)
		mustRepair(t, superNava(t), f.inputs())
		seen := map[string]bool{}
		for _, e := range repairEntries(t, super, f, "true") {
			seen[e.Salt] = true
		}
		if len(seen) != 52 {
			t.Errorf("%d distinct salts, want 52", len(seen))
		}
	})

	for _, c := range []struct {
		column, value string
		changed       int
	}{{"requester_id", "", 1}, {"state", "new", 2}} {
		t.Run("mapped to "+c.column, func(t *testing.T) {
			fixed := `{"st": "state", "org_unit_id": "requester_id"}`
			if c.column == "state" {
				fixed = `{"org_unit_id": "state"}`
			}
			want := strings.Replace(fixed, repairOldName, repairNewName, 1)
			f := repairSeed(t, super, []repairVersion{{repairType, fixed}, {repairType, fixed}},
				repairRecord{data: `{"n": 1}`, requester: true}, repairRecord{data: `{"n": 2}`})
			before := []string{recordJSON(t, super, f.records[0]), recordJSON(t, super, f.records[1])}
			mustRepair(t, superNava(t), f.inputs())
			if got := queryString(t, super, `SELECT count(*)::text FROM entity_definition_versions
				WHERE definition_id = $1 AND fixed_columns = $2::text::jsonb`, f.typ, want); got != "2" {
				t.Errorf("%s versions with fixed_columns %s, want 2", got, want)
			}
			config := repairEntries(t, super, f, "domain = 'config' AND version_after IS NULL")
			if len(config) != 1 {
				t.Fatalf("%d fixed_columns entries, want 1", len(config))
			}
			e := config[0]
			doc := func(fixed string) string {
				return queryString(t, super, "SELECT jsonb_build_object('fixed_columns', $1::text::jsonb)::text", fixed)
			}
			sameOps(t, "fixed_columns", e.Diff, doc(fixed), doc(want))
			wantEntry := repairEntry{ID: e.ID, Domain: "config", Action: "update", SubjectType: "entity_definition", SubjectID: f.typ, Actor: "operator", Source: "system",
				SourceRef: repairSourceRef, Content: fmt.Sprintf("%x", schemaSum(doc(want))), Salt: e.Salt, Diff: e.Diff}
			if e != wantEntry {
				t.Errorf("fixed_columns entry %+v, want %+v", e, wantEntry)
			}
			recs := repairEntries(t, super, f, "domain = 'record'")
			if len(recs) != c.changed {
				t.Fatalf("%d record entries, want %d", len(recs), c.changed)
			}
			value := c.value
			if value == "" {
				value = f.tenant // the seeded principal has the tenant's id
			}
			for _, r := range recs {
				sameOps(t, "mapped record", r.Diff, `{"org_unit_id": "`+value+`"}`, `{"old_unit_id": "`+value+`"}`)
			}
			for i, id := range f.records {
				if got := recordJSON(t, super, id); got != before[i] {
					t.Errorf("record %d: %s, want %s", i, got, before[i])
				}
				wantVersion := "2"
				if i >= c.changed {
					wantVersion = "1"
				}
				if got := queryString(t, super, "SELECT version || data::text FROM records WHERE id = $1", id); got != wantVersion+fmt.Sprintf(`{"n": %d}`, i+1) {
					t.Errorf("record %d: %s, want version %s and the data unchanged", i, got, wantVersion)
				}
			}
		})
	}

	// 12: what the operator sees first. The add op of a string of n bytes is
	// n + 45 bytes, and the whole diff n + 86.
	t.Run("lists", func(t *testing.T) {
		const overhead = len(`[{"op":"add","path":"/old_unit_id","value":""},{"op":"remove","path":"/org_unit_id"}]`)
		big := func(n int) string { return `{"org_unit_id": "` + strings.Repeat("x", n) + `"}` }
		f := repairSeed(t, super, versionRows(repairType, withProp(`{"title":"`+strings.Repeat("x", 66000)+`"}`)),
			repairRecord{data: `{"org_unit_id": "a"}`, entries: 500}, repairRecord{data: `{"org_unit_id": "a"}`, entries: 499}, repairRecord{data: `{"org_unit_id": "a"}`, entries: 501},
			repairRecord{data: big(audit.MaxDiffBytes - overhead)}, repairRecord{data: big(audit.MaxDiffBytes - overhead + 1)}, repairRecord{data: big(70000)}, repairRecord{data: `{}`})
		before := tenantState(t, super, "")
		conn := superNava(t)
		out := pasteOne(t, conn, f.inputs())
		if out.Err != nil {
			t.Fatal(out.Err)
		}
		if len(out.Rows) < 4 {
			t.Fatalf("%d lists", len(out.Rows))
		}
		lists := out.Rows[len(out.Rows)-4:]
		truncated := [][]string{{f.records[4]}, {f.records[5]}}
		slices.SortFunc(truncated, slices.Compare[[]string])
		for i, want := range [][][]string{{{"1", "f"}, {"2", "t"}}, {{"6"}}, {{f.records[0]}}, truncated} {
			if !reflect.DeepEqual(lists[i], want) {
				t.Errorf("list %d = %v, want %v", i+1, lists[i], want)
			}
		}
		if got := queryString(t, superNava(t), "SELECT count(*)::text FROM audit_log WHERE tenant_id = $1 AND actor_system = 'operator'", f.tenant); got != "0" {
			t.Errorf("%s entries while the operator reads the lists", got)
		}
		if tenantState(t, conn, "") != before {
			t.Error("paste 1 wrote to a platform table")
		}
		if err := pasteTwo(t, conn); err != nil {
			t.Fatal(err)
		}
		// The stored diffs: the cut keeps whole ops in order, as truncateDiff
		// does. 9: history is unknown from there.
		for _, e := range repairEntries(t, super, f, "true") {
			ops := 2
			switch {
			case e.SubjectID == f.records[4]: // one byte over: the add still fits
				ops = 1
			case e.SubjectID == f.records[5], e.Domain == "config" && e.After == "2":
				ops = 0
			}
			var stored []json.RawMessage
			if err := json.Unmarshal([]byte(e.Diff), &stored); err != nil {
				t.Fatal(err)
			}
			if e.Truncated != (ops < 2) || len(stored) != ops || len(e.Diff) > audit.MaxDiffBytes {
				t.Errorf("entry of %s %s: truncated %v with %d ops; want %d of 2 ops", e.Domain, e.SubjectID, e.Truncated, len(stored), ops)
			}
		}
		for i, wantKnown := range map[int]bool{3: true, 4: false} {
			if changes := replayed(t, super, f.records[i]); len(changes) != 3 || !changes[1].Known || changes[2].Known != wantKnown {
				t.Errorf("record %d: the repair's entry is not known %v: %d changes", i, wantKnown, len(changes))
			}
		}
		if err := laterUpdate(t, app, f, f.records[4], `[{"op":"replace","path":"/old_unit_id","value":"z"}]`, `{"old_unit_id": "z"}`); err != nil {
			t.Fatal(err)
		}
		if changes := replayed(t, super, f.records[4]); len(changes) != 4 || changes[3].Known {
			t.Error("the history is known again after a truncated entry")
		}
	})

	// 17: none of the repairs above touched the other tenant.
	if tenantState(t, super, bystander.tenant) != bystanderState {
		t.Error("another tenant's rows changed")
	}
}

// waitForLock returns once a backend other than conn's waits for a lock.
func waitForLock(t *testing.T, super *pgx.Conn) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if queryBool(t, super, `SELECT EXISTS (SELECT 1 FROM pg_locks WHERE NOT granted AND relation = 'entity_definition_versions'::regclass)`) {
			return
		}
	}
	t.Fatal("nothing waits for the versions lock")
}

// Tests 2 (isolation), 13, 14, 15, 16, 20 (session) and R9 of 3.
func TestTICK1125Pastes(t *testing.T) {
	_, super := preS31Cluster(t)
	seed := func(t *testing.T) *repairFixture {
		return repairSeed(t, super, versionRows(repairType), repairRecord{data: `{"org_unit_id": "a"}`}, repairRecord{data: `{"org_unit_id": "b"}`})
	}
	unchanged := func(t *testing.T, before string) {
		t.Helper()
		if tenantState(t, super, "") != before {
			t.Error("something was written")
		}
	}

	t.Run("paste 2 alone, after ROLLBACK and after a refusal", func(t *testing.T) {
		f := seed(t)
		before := tenantState(t, super, "")
		if err := pasteTwo(t, superNava(t)); err == nil {
			t.Error("paste 2 alone: no error")
		}
		conn := superNava(t)
		if out := pasteOne(t, conn, f.inputs()); out.Err != nil {
			t.Fatal(out.Err)
		}
		exec(t, conn, "ROLLBACK")
		if err := pasteTwo(t, conn); err == nil {
			t.Error("paste 2 after ROLLBACK: no error")
		}
		unchanged(t, before)
		f.addMasks(t, super, 1, repairOldName)
		before = tenantState(t, super, "")
		for _, psql := range []bool{false, true} {
			conn := superNava(t)
			sql, err := repairtest.WithInputs(repairBlock(t, repairtest.PasteOne), f.inputs())
			if err != nil {
				t.Fatal(err)
			}
			if out := paste(t, conn, sql, psql); out.Err == nil || !strings.Contains(out.Err.Error(), "R1:") {
				t.Errorf("paste 1 = %v, want R1", out.Err)
			}
			paste(t, conn, repairBlock(t, repairtest.PasteTwo), psql)
		}
		unchanged(t, before)
	})

	t.Run("a failed lock under ON_ERROR_ROLLBACK", func(t *testing.T) {
		f := seed(t)
		before := tenantState(t, super, "")
		reader := superNava(t)
		exec(t, reader, "BEGIN")
		exec(t, reader, "SELECT count(*) FROM entity_definition_versions")
		for _, psql := range []bool{true, false} {
			conn := superNava(t)
			sql, err := repairtest.WithInputs(repairBlock(t, repairtest.PasteOne), f.inputs())
			if err != nil {
				t.Fatal(err)
			}
			out := paste(t, conn, sql, psql)
			if sqlState(out.Err) != "55P03" {
				t.Errorf("paste 1 = %v, want 55P03", out.Err)
			}
			paste(t, conn, repairBlock(t, repairtest.PasteTwo), psql)
			unchanged(t, before)
		}
		exec(t, reader, "ROLLBACK")
		mustRepair(t, superNava(t), f.inputs())
	})

	t.Run("a mask committed while the repair waits", func(t *testing.T) {
		f := seed(t)
		writer := superNava(t)
		exec(t, writer, "SET session_replication_role = replica")
		exec(t, writer, "BEGIN")
		exec(t, writer, "SELECT count(*) FROM entity_definition_versions")
		exec(t, writer, `INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
			VALUES ($1, $2, 1, '[]', '[{"field": "org_unit_id", "read": {}, "write": {}}]')`, f.tenant, f.typ)
		conn := superNava(t)
		exec(t, conn, "SET SESSION CHARACTERISTICS AS TRANSACTION ISOLATION LEVEL REPEATABLE READ")
		done := make(chan error, 1)
		go func() { done <- repair(t, conn, f.inputs()) }()
		waitForLock(t, super)
		exec(t, writer, "COMMIT")
		if err := <-done; err == nil || !strings.Contains(err.Error(), "R1:") {
			t.Errorf("repair = %v, want R1", err)
		}
		wantEntries(t, super, f, 0)
	})

	t.Run("session settings", func(t *testing.T) {
		f := seed(t)
		before := tenantState(t, super, "")
		conn := superNava(t)
		exec(t, conn, "SET session_replication_role = replica")
		if out := pasteOne(t, conn, f.inputs()); out.Err == nil || !strings.Contains(out.Err.Error(), "session_replication_role") {
			t.Errorf("replica before paste 1 = %v", out.Err)
		}
		_ = pasteTwo(t, conn)
		unchanged(t, before)
		conn = superNava(t)
		if out := pasteOne(t, conn, f.inputs()); out.Err != nil {
			t.Fatal(out.Err)
		}
		exec(t, conn, "SET session_replication_role = replica")
		if err := pasteTwo(t, conn); err == nil || !strings.Contains(err.Error(), "session_replication_role") {
			t.Errorf("replica before paste 2 = %v", err)
		}
		unchanged(t, before)
		for _, c := range []struct{ name, from, to, want string }{
			{"isolation", "BEGIN ISOLATION LEVEL READ COMMITTED;", "BEGIN ISOLATION LEVEL REPEATABLE READ;", "READ COMMITTED"},
			{"lock_timeout", "SET LOCAL lock_timeout = '5s';", "SET LOCAL lock_timeout = '1h';", "lock_timeout"},
		} {
			sql, err := repairtest.WithInputs(repairBlock(t, repairtest.PasteOne), f.inputs())
			if err != nil || strings.Count(sql, c.from) != 1 {
				t.Fatalf("%s: %v", c.name, err)
			}
			conn := superNava(t)
			if out := paste(t, conn, strings.Replace(sql, c.from, c.to, 1), false); out.Err == nil || !strings.Contains(out.Err.Error(), c.want) {
				t.Errorf("%s: paste 1 = %v", c.name, out.Err)
			}
			_ = pasteTwo(t, conn)
			unchanged(t, before)
		}
	})

	t.Run("a failed assertion rolls everything back", func(t *testing.T) {
		f := seed(t)
		exec(t, super, `CREATE FUNCTION t1125_skip() RETURNS trigger LANGUAGE plpgsql AS
			'BEGIN IF NEW.id = TG_ARGV[0]::uuid THEN RETURN NULL; END IF; RETURN NEW; END'`)
		exec(t, super, "CREATE TRIGGER t1125_skip BEFORE UPDATE ON records FOR EACH ROW EXECUTE FUNCTION t1125_skip('"+f.records[0]+"')")
		before := tenantState(t, super, "")
		if err := repair(t, superNava(t), f.inputs()); err == nil || !strings.Contains(err.Error(), "updated 1 records, planned 2") {
			t.Errorf("repair = %v, want the record count", err)
		}
		unchanged(t, before)
		exec(t, super, "DROP TRIGGER t1125_skip ON records")
		exec(t, super, "DROP FUNCTION t1125_skip()")
		mustRepair(t, superNava(t), f.inputs())
	})

	// 24. An empty plan is in TestTICK1125Repairs/content.
	t.Run("the success notice", func(t *testing.T) {
		f := repairSeed(t, super, versionRows(repairType, repairType), repairRecord{data: `{"org_unit_id": "a"}`}, repairRecord{data: `{"org_unit_id": "b"}`})
		exec(t, super, `CREATE FUNCTION t1125_skip() RETURNS trigger LANGUAGE plpgsql AS 'BEGIN RETURN NULL; END'`)
		exec(t, super, "CREATE TRIGGER t1125_skip BEFORE UPDATE ON records FOR EACH ROW EXECUTE FUNCTION t1125_skip()")
		conn, notices := noticeConn(t)
		one, err := repairtest.WithInputs(repairBlock(t, repairtest.PasteOne), f.inputs())
		if err != nil {
			t.Fatal(err)
		}
		paste(t, conn, one, true)
		out := paste(t, conn, repairBlock(t, repairtest.PasteTwo), true)
		if out.Err == nil || len(*notices) != 0 {
			t.Errorf("a failed paste 2 = %v, notices %q", out.Err, *notices)
		}
		exec(t, super, "DROP TRIGGER t1125_skip ON records")
		exec(t, super, "DROP FUNCTION t1125_skip()")
		mustRepair(t, conn, f.inputs())
		if want := []string{"repaired: 2 versions, 2 records"}; !slices.Equal(*notices, want) {
			t.Errorf("notices %q, want %q", *notices, want)
		}
	})

	// 23: an input changed after paste 1 changes nothing that is written.
	for _, c := range []struct{ setting, value string }{
		{"nava_repair.tenant_id", uuid.NewString()}, {"nava_repair.type_id", uuid.NewString()}, {"nava_repair.new_name", "old_region"},
		{"nava_repair.source_ref", "Jane Doe"}, {"app.tenant_id", uuid.NewString()},
	} {
		t.Run("a changed "+c.setting, func(t *testing.T) {
			f := seed(t)
			conn := superNava(t)
			if out := pasteOne(t, conn, f.inputs()); out.Err != nil {
				t.Fatal(out.Err)
			}
			exec(t, conn, "SELECT set_config($1, $2, true)", c.setting, c.value)
			if err := pasteTwo(t, conn); err != nil {
				t.Fatal(err)
			}
			wantNoOldName(t, super, f)
			if got := queryString(t, super, "SELECT count(*)::text FROM records WHERE entity_definition_id = $1 AND data ? 'old_unit_id'", f.typ); got != "2" {
				t.Errorf("%s records with the checked name, want 2", got)
			}
			if got := queryString(t, super, "SELECT count(*)::text FROM audit_log WHERE tenant_id = $1 AND source_ref = $2", f.tenant, repairSourceRef); got != "3" {
				t.Errorf("%s entries with the checked source_ref, want 3", got)
			}
		})
	}

	t.Run("paste 1 again on an open transaction", func(t *testing.T) {
		f := seed(t)
		for name, second := range map[string]*repairFixture{
			"a refused type": seed(t), "a clean type": repairSeed(t, super, versionRows(repairClean)),
		} {
			before := tenantState(t, super, "")
			conn := superNava(t)
			if out := pasteOne(t, conn, f.inputs()); out.Err != nil {
				t.Fatal(out.Err)
			}
			one, err := repairtest.WithInputs(repairBlock(t, repairtest.PasteOne), second.inputs())
			if err != nil {
				t.Fatal(err)
			}
			block := one[strings.Index(one, "SELECT set_config"):strings.Index(one, "CREATE TEMP TABLE")] + one[strings.Index(one, "DO $nava_repair$"):]
			if out := paste(t, conn, block, true); out.Err == nil || !strings.Contains(out.Err.Error(), "the plan already has rows") {
				t.Errorf("%s: paste 1 again = %v", name, out.Err)
			}
			exec(t, conn, "ROLLBACK")
			unchanged(t, before)
		}
	})

	t.Run("records are written as nava_app until COMMIT", func(t *testing.T) {
		f := seed(t)
		exec(t, super, `CREATE FUNCTION t1125_role() RETURNS trigger LANGUAGE plpgsql AS
			'BEGIN IF current_user <> ''nava_app'' THEN RAISE EXCEPTION ''written as %'', current_user; END IF; RETURN NULL; END'`)
		exec(t, super, `CREATE CONSTRAINT TRIGGER t1125_role AFTER UPDATE ON records DEFERRABLE INITIALLY DEFERRED
			FOR EACH ROW EXECUTE FUNCTION t1125_role()`)
		if err := repair(t, superNava(t), f.inputs()); err != nil {
			t.Error(err)
		}
		exec(t, super, "DROP TRIGGER t1125_role ON records")
		exec(t, super, "DROP FUNCTION t1125_role()")
		wantNoOldName(t, super, f)
	})

	t.Run("a record that is not at its planned version", func(t *testing.T) {
		f := seed(t)
		before := tenantState(t, super, "")
		conn := superNava(t)
		if out := pasteOne(t, conn, f.inputs()); out.Err != nil {
			t.Fatal(out.Err)
		}
		exec(t, conn, "SET LOCAL session_replication_role = replica")
		exec(t, conn, "UPDATE records SET version = version + 1 WHERE id = $1", f.records[0])
		exec(t, conn, "SET LOCAL session_replication_role = origin")
		if err := pasteTwo(t, conn); err == nil || !strings.Contains(err.Error(), "updated 1 records, planned 2") {
			t.Errorf("paste 2 = %v, want the record count", err)
		}
		unchanged(t, before)
	})

	t.Run("the trigger modes", func(t *testing.T) {
		f := seed(t)
		exec(t, super, "ALTER TABLE acl_versions ENABLE REPLICA TRIGGER acl_versions_tenant_ids")
		exec(t, super, "ALTER TABLE records ENABLE ALWAYS TRIGGER records_audit_commit_check")
		modes, before := triggerModes(t, super), tenantState(t, super, "")
		conn := superNava(t)
		if out := pasteOne(t, conn, f.inputs()); out.Err != nil {
			t.Fatal(out.Err)
		}
		exec(t, conn, "ALTER TABLE workflow_definitions DISABLE TRIGGER USER")
		if err := pasteTwo(t, conn); err == nil || !strings.Contains(err.Error(), "trigger") {
			t.Errorf("a mode changed between the pastes: %v", err)
		}
		unchanged(t, before)
		mustRepair(t, superNava(t), f.inputs())
		if got := triggerModes(t, super); got != modes {
			t.Error("a trigger's mode changed")
		}
		exec(t, super, "ALTER TABLE acl_versions ENABLE TRIGGER acl_versions_tenant_ids")
		exec(t, super, "ALTER TABLE records ENABLE TRIGGER records_audit_commit_check")
	})

	t.Run("R9 an entry that is not older", func(t *testing.T) {
		f := seed(t)
		inReplica(t, super, func() {
			exec(t, super, `INSERT INTO audit_log (tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id, actor_system, source)
				VALUES ($1, gen_random_uuid(), now() + interval '1 hour', 'record', 'update', 'record', $2, $3, 'system', 'system')`, f.tenant, f.records[0], f.typ)
		})
		wantRefused(t, super, f.inputs(), "R9:", "entry ")
	})

	t.Run("R9 a writer that began before", func(t *testing.T) {
		f := seed(t)
		for _, role := range []string{"nava_app", "nava_worker"} {
			writer := asRole(t, role)
			exec(t, writer, "BEGIN")
			exec(t, writer, "SELECT now()")
			wantRefused(t, super, f.inputs(), "R9:", role)
			exec(t, writer, "ROLLBACK")
		}
		// A transaction of another role, or in another database, is no reason.
		purge := asRole(t, "nava_purge")
		exec(t, purge, "BEGIN")
		exec(t, purge, "SELECT now()")
		exec(t, super, "CREATE DATABASE t1125_other")
		defer exec(t, super, "DROP DATABASE t1125_other WITH (FORCE)")
		other := connect(t, roleConfig(t, "nava_app", "t1125_other"))
		exec(t, other, "BEGIN")
		exec(t, other, "SELECT now()")
		mustRepair(t, superNava(t), f.inputs())
		exec(t, purge, "ROLLBACK")
		exec(t, other, "ROLLBACK")
	})

	t.Run("a writer that began after waits", func(t *testing.T) {
		f := seed(t)
		conn := superNava(t)
		if out := pasteOne(t, conn, f.inputs()); out.Err != nil {
			t.Fatal(out.Err)
		}
		app := asRole(t, "nava_app")
		done := make(chan error, 1)
		go func() {
			_, err := execIn(app, f.tenant, `WITH s AS (SELECT count(*) FROM entity_definition_versions),
					w AS (UPDATE records SET version = version + 1, data = jsonb_build_object('old_unit_id', 'later') WHERE tenant_id = $1 AND id = $2
						AND (SELECT count FROM s) > 0 RETURNING version)
				INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before, version_after,
					actor_system, source, diff, diff_salt, diff_sha256)
				SELECT $1, gen_random_uuid(), 'record', 'update', 'record', $2, $3, version - 1, version, 'system', 'system',
					'[{"op":"replace","path":"/old_unit_id","value":"later"}]', $4, sha256($4 || '[{"op":"replace","path":"/old_unit_id","value":"later"}]'::bytea) FROM w`,
				f.tenant, f.records[0], f.typ, bytes.Repeat([]byte{3}, 16))
			done <- err
		}()
		waitForLock(t, super)
		if err := pasteTwo(t, conn); err != nil {
			t.Fatal(err)
		}
		if err := <-done; err != nil {
			t.Fatalf("the writer: %v", err)
		}
		changes := replayed(t, super, f.records[0])
		if len(changes) != 3 || !changes[1].Known || !changes[2].Known || changes[2].After[repairNewName] != "later" {
			t.Errorf("replay = %+v", changes)
		}
	})
}

// noticeConn is a superuser connection and the NOTICEs it got.
func noticeConn(t *testing.T) (*pgx.Conn, *[]string) {
	t.Helper()
	notices := &[]string{}
	cfg := superConfig(t).Copy()
	cfg.Database = "nava"
	cfg.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) { *notices = append(*notices, n.Message) }
	return connect(t, cfg), notices
}

func wantEntries(t *testing.T, super *pgx.Conn, f *repairFixture, n int) {
	t.Helper()
	if got := len(repairEntries(t, super, f, "true")); got != n {
		t.Errorf("%d entries of the repair, want %d", got, n)
	}
}

// Test 6: the upgrade is refused, the documented SQL runs, the upgrade applies.
func TestTICK1125UpgradeAfterRepair(t *testing.T) {
	m, super := preS31Cluster(t)
	cases := []struct {
		name     string
		versions []repairVersion
	}{
		{"current", versionRows(repairClean, repairType)},
		{"older only", versionRows(repairType, repairClean)},
		{"both", versionRows(repairType, repairType)},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := repairSeed(t, super, c.versions, repairRecord{data: `{"org_unit_id": "a"}`})
			modes := triggerModes(t, super)
			_, err := migrateFS(t, m, migrationsThrough(t, orgUnitIDReservedMigration))
			if pe := refusal(t, "upgrade", err); pe == nil || pe.ConstraintName != orgUnitIDReservedConstraint {
				t.Fatalf("upgrade before the repair: %v", err)
			}
			mustRepair(t, superNava(t), f.inputs())
			if triggerModes(t, super) != modes {
				t.Error("a trigger's mode changed")
			}
			if i < len(cases)-1 {
				if err := migrateOnlyFS(t, m, migrationsThrough(t, orgUnitIDReservedMigration)); err != nil {
					t.Fatalf("upgrade after the repair: %v", err)
				}
				undoOrgUnitIDReserved(t, m, super)
				return
			}
			migrateAll(t, m)
			if !hasOrgUnitIDConstraint(t, super) {
				t.Error("the upgrade left no constraint")
			}
		})
	}
}

// Tests 18 and 19.
func TestTICK1125Checks(t *testing.T) {
	_, super := preS31Cluster(t)
	check := func(t *testing.T) [][][]string {
		t.Helper()
		out := paste(t, superNava(t), repairBlock(t, repairtest.Check), false)
		if out.Err != nil || len(out.Rows) != 2 {
			t.Fatalf("check = %v, %d results", out.Err, len(out.Rows))
		}
		return out.Rows
	}
	if got := check(t); len(got[0]) != 0 || !reflect.DeepEqual(got[1], [][]string{{"0", "0"}}) {
		t.Errorf("on a database with no version: %v", got)
	}
	clean := repairSeed(t, super, versionRows(repairClean))
	if got := check(t); len(got[0]) != 0 || !reflect.DeepEqual(got[1], [][]string{{"1", fmt.Sprint(len(repairClean))}}) {
		t.Errorf("on a clean database: %v", got)
	}
	nul := esc(`{"title":"~u0000","properties":{"~u006frg_unit_id":{}}}`)
	long := strings.Replace(repairType, repairMarker, strings.Repeat("é", 5000), 1)
	f := repairSeed(t, super, versionRows(nul, repairClean, long))
	size := fmt.Sprint(2*len(repairClean) + len(nul) + len(long))
	got := check(t)
	if want := [][]string{{f.tenant, f.typ, "ticket", "1", "f"}, {f.tenant, f.typ, "ticket", "3", "t"}}; !reflect.DeepEqual(got[0], want) {
		t.Errorf("refused versions %v, want %v", got[0], want)
	}
	if want := [][]string{{"4", size}}; !reflect.DeepEqual(got[1], want) {
		t.Errorf("counts %v, want %v", got[1], want)
	}

	unit, foreign, unknown := uuid.NewString(), uuid.NewString(), uuid.NewString()
	under := func(ids ...string) string {
		return doc2([]any{map[string]any{"name": "c", "expr": map[string]any{"under": map[string]any{"of": ids[:1]}}}}, []any{
			policy("p", map[string]any{"any": []any{map[string]any{"under": map[string]any{"of": "caller"}}, map[string]any{"under": map[string]any{"of": ids}}}})})
	}
	inReplica(t, super, func() {
		exec(t, super, "INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, 'ou', 'Own')", clean.tenant, unit)
		exec(t, super, "INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, 'ou', 'Foreign')", f.tenant, foreign)
		insert := "INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks) VALUES ($1, $2, $3, $4::text::jsonb, '[]')"
		exec(t, super, insert, clean.tenant, clean.typ, 1, under(unit, strings.ToUpper(unit)))
		exec(t, super, insert, clean.tenant, clean.typ, 2, `[]`)
	})
	ar := func() [][]string {
		out := paste(t, superNava(t), repairBlock(t, repairtest.ARS3001), false)
		if out.Err != nil {
			t.Fatal(out.Err)
		}
		return out.Rows[0]
	}
	if got := ar(); len(got) != 0 {
		t.Errorf("AR-S30-01 on the tenant's own unit: %v", got)
	}
	// The CHECK keeps an id that is not a UUID out; a row stored before it
	// must be listed and must not make the query raise.
	exec(t, super, "ALTER TABLE acl_versions DROP CONSTRAINT acl_versions_rules_check")
	inReplica(t, super, func() {
		exec(t, super, "INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks) VALUES ($1, $2, 3, $3::text::jsonb, '[]')",
			clean.tenant, clean.typ, under(unit, foreign, unknown, "not-a-uuid"))
	})
	var want [][]string
	for _, id := range []string{foreign, unknown, "not-a-uuid"} {
		want = append(want, []string{clean.tenant, clean.typ, "3", id})
	}
	got2 := ar()
	slices.SortFunc(want, slices.Compare[[]string])
	slices.SortFunc(got2, slices.Compare[[]string])
	if !reflect.DeepEqual(got2, want) {
		t.Errorf("AR-S30-01 = %v, want %v", got2, want)
	}
}
