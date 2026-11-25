//go:build integration

package dbtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// P7-02 (TICK-742, ADR-006 D1, D4, D5, R5): record_field_coerce and the
// records triggers that keep the side tables equal to a recompute from
// records.data. Migration 20261125165100_field_index_trigger.sql.

const (
	fieldIndexTrigger = 20261125165100
	// uncoercedSetting is the transaction-local setting the trigger counts
	// non-coercing values in, per entity definition.
	uncoercedSetting = "nava.field_index_uncoerced"

	// insertTypedRecord: tenant $1, id $2, entity type $3 (= the tenant id
	// in these fixtures), number $4, data $5.
	// deleteRecord deletes record $2 of tenant $1 with its audit entry, so
	// the commit check passes.
	deleteRecord = `WITH r AS (DELETE FROM records WHERE tenant_id = $1 AND id = $2 RETURNING tenant_id, id, version)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_before, actor_system, source)
		SELECT tenant_id, gen_random_uuid(), 'record', 'delete', 'record', id, version, 'system', 'system' FROM r`

	insertTypedRecord = `INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, data)
		VALUES ($1, $2, $3, 1, $4, 'open', $5)`
)

// fieldIndexTriggerMigration is the P7-02 migration, and coerceFunction the
// grant it adds to db/grants.yaml.
const (
	fieldIndexTriggerMigration = "20261125165100_field_index_trigger.sql"
	coerceFunction             = "public.record_field_coerce(jsonb,text)"
)

// beforeFieldIndexTrigger is m as a run of fsys sees it when fsys stops
// before fieldIndexTriggerMigration: no EXECUTE on record_field_coerce.
// beforeNumberingMode calls it, so every older-schema run gets it.
func beforeFieldIndexTrigger(fsys fs.FS, m *grants.Manifest) *grants.Manifest {
	if _, err := fs.Stat(fsys, fieldIndexTriggerMigration); err == nil {
		return m
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		r.Functions = maps.Clone(r.Functions)
		delete(r.Functions, coerceFunction)
		gm.Roles[name] = r
	}
	return &gm
}

// coerced is one row of record_field_coerce, decoded in binary, so its
// values do not depend on the session's TimeZone or DateStyle.
type coerced struct {
	ord int16
	val string
}

// coerce runs record_field_coerce(v, vt) as conn in tx and returns its rows.
func coerce(ctx context.Context, tx pgx.Tx, v, vt string) ([]coerced, error) {
	rows, err := tx.Query(ctx, `SELECT ord, value_text, value_num, value_ts, value_bool
		FROM record_field_coerce($1::jsonb, $2) ORDER BY ord`, v, vt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []coerced
	for rows.Next() {
		var (
			ord  int16
			txt  *string
			num  *float64
			ts   *time.Time
			flag *bool
		)
		if err := rows.Scan(&ord, &txt, &num, &ts, &flag); err != nil {
			return nil, err
		}
		set := 0
		c := coerced{ord: ord}
		if txt != nil {
			set++
			c.val = "text:" + *txt
		}
		if num != nil {
			set++
			c.val = fmt.Sprintf("num:%v", *num)
		}
		if ts != nil {
			set++
			c.val = "ts:" + ts.UTC().Format(time.RFC3339Nano)
		}
		if flag != nil {
			set++
			c.val = fmt.Sprintf("bool:%v", *flag)
		}
		if set != 1 {
			return nil, fmt.Errorf("row at ord %d has %d values set, want 1", ord, set)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// one is a single scalar row at ord 0.
func one(val string) []coerced { return []coerced{{0, val}} }

// Acceptance (CI-9): every coercion rule of D5, table-driven, gives the same
// rows in sessions with different TimeZone and DateStyle, as nava_app and as
// nava_worker (the trigger's and the backfill's roles).
func TestP702CoerceRulesAcrossSessions(t *testing.T) {
	migratedPlatform(t)
	long := strings.Repeat("x", 2000)
	cases := []struct {
		name, vt, v string
		want        []coerced
	}{
		{"text string", "text", `"high"`, one("text:high")},
		{"text unicode", "text", `"Größe ✓"`, one("text:Größe ✓")},
		{"text empty string", "text", `""`, one("text:")},
		{"text 2000 bytes", "text", `"` + long + `"`, one("text:" + long)},
		{"text 2001 bytes", "text", `"` + long + `y"`, nil},
		{"text number", "text", `1`, nil},
		{"text boolean", "text", `true`, nil},
		{"text object", "text", `{"a": "b"}`, nil},
		{"text null", "text", `null`, nil},
		{"text array", "text", `["a", "b"]`, []coerced{{0, "text:a"}, {1, "text:b"}}},
		{"text array keeps positions", "text", `["a", 1, null, "d"]`, []coerced{{0, "text:a"}, {3, "text:d"}}},
		{"text nested array", "text", `[["a"]]`, nil},
		{"text empty array", "text", `[]`, nil},

		{"num integer", "num", `42`, one("num:42")},
		{"num decimal", "num", `1.5`, one("num:1.5")},
		{"num negative exponent", "num", `-2.5e-3`, one("num:-0.0025")},
		{"num past 2^53", "num", `12345678901234567890`, one("num:1.2345678901234567e+19")},
		{"num max float8", "num", `1.7976931348623157e308`, one(fmt.Sprintf("num:%v", math.MaxFloat64))},
		{"num 1e400 overflows", "num", `1e400`, nil},
		{"num 1e-400 underflows", "num", `1e-400`, nil},
		{"num numeric string", "num", `"1"`, nil},
		{"num NaN string", "num", `"NaN"`, nil},
		{"num Infinity string", "num", `"Infinity"`, nil},
		{"num -Infinity string", "num", `"-Infinity"`, nil},
		{"num boolean", "num", `true`, nil},
		{"num array", "num", `[1, "2", 3]`, []coerced{{0, "num:1"}, {2, "num:3"}}},

		{"bool true", "bool", `true`, one("bool:true")},
		{"bool false", "bool", `false`, one("bool:false")},
		{"bool string true", "bool", `"true"`, nil},
		{"bool yes", "bool", `"yes"`, nil},
		{"bool t", "bool", `"t"`, nil},
		{"bool number", "bool", `1`, nil},
		{"bool array", "bool", `[true, "no", false]`, []coerced{{0, "bool:true"}, {2, "bool:false"}}},

		{"ts Z", "ts", `"2026-11-15T10:20:30Z"`, one("ts:2026-11-15T10:20:30Z")},
		{"ts lower-case t and z", "ts", `"2026-11-15t10:20:30z"`, one("ts:2026-11-15T10:20:30Z")},
		{"ts Chatham offset", "ts", `"2026-11-15T10:20:30+13:45"`, one("ts:2026-11-14T20:35:30Z")},
		{"ts negative offset", "ts", `"2026-11-15T10:20:30-05:00"`, one("ts:2026-11-15T15:20:30Z")},
		{"ts fraction", "ts", `"2026-11-15T10:20:30.123456Z"`, one("ts:2026-11-15T10:20:30.123456Z")},
		{"ts date", "ts", `"2026-11-15"`, one("ts:2026-11-15T00:00:00Z")},
		{"ts leap day", "ts", `"2028-02-29"`, one("ts:2028-02-29T00:00:00Z")},
		{"ts no offset", "ts", `"2026-11-15T10:20:30"`, nil},
		{"ts space separator", "ts", `"2026-11-15 10:20:30Z"`, nil},
		{"ts offset without colon", "ts", `"2026-11-15T10:20:30+0100"`, nil},
		{"ts named zone", "ts", `"2026-11-15T10:20:30 Pacific/Chatham"`, nil},
		{"ts no seconds", "ts", `"2026-11-15T10:20Z"`, nil},
		{"ts hour 24", "ts", `"2026-11-15T24:00:00Z"`, nil},
		{"ts US date", "ts", `"01/02/2026"`, nil},
		{"ts German date", "ts", `"15.11.2026"`, nil},
		{"ts SQL style", "ts", `"11/15/2026 10:20:30 UTC"`, nil},
		{"ts 30 February", "ts", `"2026-02-30"`, nil},
		{"ts 29 February in a common year", "ts", `"2026-02-29"`, nil},
		{"ts year 0", "ts", `"0000-01-01"`, nil},
		{"ts infinity", "ts", `"infinity"`, nil},
		{"ts epoch", "ts", `"epoch"`, nil},
		{"ts now", "ts", `"now"`, nil},
		{"ts unix number", "ts", `1763200830`, nil},
		{"ts array", "ts", `["2026-11-15", "x", "2026-11-15T00:00:00+01:00"]`,
			[]coerced{{0, "ts:2026-11-15T00:00:00Z"}, {2, "ts:2026-11-14T23:00:00Z"}}},

		{"unknown value type", "date", `"2026-11-15"`, nil},
	}

	sessions := []struct{ name, tz, style string }{
		{"default", "", ""},
		{"Chatham SQL DMY", "Pacific/Chatham", "SQL, DMY"},
		{"Chatham German", "Pacific/Chatham", "German"},
		{"Kiritimati Postgres MDY", "Pacific/Kiritimati", "Postgres, MDY"},
	}
	ctx := context.Background()
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, s := range sessions {
			err := inTenantTx(conn, tenantA, func(tx pgx.Tx) error {
				if s.tz != "" {
					if _, err := tx.Exec(ctx, "SELECT set_config('TimeZone', $1, true), set_config('DateStyle', $2, true)", s.tz, s.style); err != nil {
						return err
					}
				}
				for _, c := range cases {
					got, err := coerce(ctx, tx, c.v, c.vt)
					if err != nil {
						return fmt.Errorf("%s: %w", c.name, err)
					}
					if !slices.Equal(got, c.want) {
						t.Errorf("%s, session %s: %s %s = %v, want %v", role, s.name, c.vt, c.name, got, c.want)
					}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("%s, session %s: %v", role, s.name, err)
			}
		}
	}
}

// Elements past ord 32,767 (the smallint ord) give no row, and never an
// error.
func TestP702CoerceLongArray(t *testing.T) {
	migratedPlatform(t)
	elems := make([]string, 32770)
	for i := range elems {
		elems[i] = `"v"`
	}
	v := "[" + strings.Join(elems, ",") + "]"
	err := inTenantTx(asRole(t, "nava_app"), tenantA, func(tx pgx.Tx) error {
		got, err := coerce(context.Background(), tx, v, "text")
		if err != nil {
			return err
		}
		if len(got) != 32768 || got[len(got)-1].ord != 32767 {
			t.Errorf("rows = %d, last ord %d; want 32768 rows, the last at 32767", len(got), got[len(got)-1].ord)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// triggerField is one registry row of the CI-4 fixture.
type triggerField struct {
	id, key, vt, state string
}

// triggerFields registers, as the superuser, a field of every value type on
// tenant's type, an array field, a backfilling one and a removing one. The
// ids are distinct per tenant.
func triggerFields(t *testing.T, super *pgx.Conn, tenant string) []triggerField {
	t.Helper()
	prefix := "0e000000-0000-4000-8000-00000000" + tenant[len(tenant)-2:]
	fields := []triggerField{
		{prefix + "01", "f_text", "text", "ready"},
		{prefix + "02", "f_num", "num", "backfilling"},
		{prefix + "03", "f_ts", "ts", "ready"},
		{prefix + "04", "f_bool", "bool", "ready"},
		{prefix + "05", "f_tags", "text", "ready"},
		{prefix + "06", "f_gone", "num", "removing"},
	}
	for _, f := range fields {
		exec(t, super, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, requested_version)
			VALUES ($1, $2, $1, $3, $4, $5, 1)`, tenant, f.id, f.key, f.vt, f.state)
	}
	return fields
}

// randomValue is a JSON value for a field of value type vt: usually one
// that coerces, sometimes one that does not, an array, null or absent.
func randomValue(r *rand.Rand, vt string) (string, bool) {
	switch r.IntN(10) {
	case 0:
		return "", false // absent
	case 1:
		return "null", true
	case 2:
		return `"1"`, true // a numeric string: text only
	case 3:
		return `[` + mustValue(r, vt) + `, true, ` + mustValue(r, vt) + `]`, true
	}
	return mustValue(r, vt), true
}

func mustValue(r *rand.Rand, vt string) string {
	switch vt {
	case "text":
		return fmt.Sprintf(`"v%d"`, r.IntN(20))
	case "num":
		return []string{"1", "2.5", "-3", "1e400", "12345678901234567890", `"NaN"`}[r.IntN(6)]
	case "ts":
		return []string{`"2026-11-15T10:00:00Z"`, `"2026-11-15T10:00:00+13:45"`, `"2026-11-15"`,
			`"2026-11-15T10:00:00"`, `"01/02/2026"`}[r.IntN(5)]
	}
	return []string{"true", "false", `"yes"`}[r.IntN(3)]
}

// randomData is a record's data over fields, plus an unflagged key.
func randomData(r *rand.Rand, fields []triggerField) string {
	parts := []string{fmt.Sprintf(`"other": %d`, r.IntN(100))}
	for _, f := range fields {
		if v, ok := randomValue(r, f.vt); ok {
			parts = append(parts, fmt.Sprintf("%q: %s", f.key, v))
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// sideDiff is the symmetric difference between the side tables and a
// recompute of them from records.data, through record_field_coerce, for
// every registry row in backfilling or ready, plus any side row of a
// removing field. Read as the superuser, who sees every tenant.
const sideDiff = `
WITH fields AS (
    SELECT tenant_id, id, entity_definition_id, field_key, value_type FROM record_index_fields
    WHERE state IN ('backfilling', 'ready')
), want AS (
    SELECT f.value_type, r.tenant_id, f.id AS field_id, r.id AS record_id, c.ord,
           coalesce(c.value_text, c.value_num::text, c.value_ts::text, c.value_bool::text) AS value
    FROM records r JOIN fields f ON f.tenant_id = r.tenant_id AND f.entity_definition_id = r.entity_definition_id
    CROSS JOIN LATERAL record_field_coerce(r.data -> f.field_key, f.value_type) c
), got AS (
    SELECT 'text' AS value_type, tenant_id, field_id, record_id, ord, value::text FROM rfv_text
    UNION ALL SELECT 'num', tenant_id, field_id, record_id, ord, value::text FROM rfv_num
    UNION ALL SELECT 'ts', tenant_id, field_id, record_id, ord, value::text FROM rfv_ts
    UNION ALL SELECT 'bool', tenant_id, field_id, record_id, ord, value::text FROM rfv_bool
), d AS (
    (SELECT 'missing' AS kind, * FROM want EXCEPT ALL SELECT 'missing', * FROM got)
    UNION ALL
    (SELECT 'extra', * FROM got EXCEPT ALL SELECT 'extra', * FROM want)
)
SELECT coalesce(string_agg(kind || ' ' || value_type || ' ' || record_id || '/' || field_id || '[' || ord || ']=' || value, '; '), '')
FROM d`

// Acceptance (CI-4): for every role that writes records, random INSERT,
// UPDATE and DELETE sequences over flagged fields of each value type leave
// the side tables equal to a recompute from records.data for every field in
// backfilling or ready, and with no row for a removing field. DELETE runs as
// the superuser: no runtime role may delete a record (db/grants.yaml), and
// the purge must hold for whoever does.
func TestP702TriggerMatchesRecompute(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	fields := map[string][]triggerField{tenantA: triggerFields(t, super, tenantA), tenantB: triggerFields(t, super, tenantB)}

	ctx := context.Background()
	writers := []string{"nava_app", "nava_worker"}
	for wi, role := range writers {
		conn := asRole(t, role)
		r := rand.New(rand.NewPCG(742, uint64(wi)))
		var live []struct{ tenant, id string }
		n := 0
		for op := range 400 {
			tenant := []string{tenantA, tenantB}[r.IntN(2)]
			switch k := r.IntN(10); {
			case k < 4 || len(live) == 0: // INSERT, sometimes two in one transaction
				err := inTenantTx(conn, tenant, func(tx pgx.Tx) error {
					for range 1 + r.IntN(2) {
						n++
						id := fmt.Sprintf("0190c000-0000-7000-8%03d-%012d", wi, n)
						sql := recordWrite(t, super, insertTypedRecord)
						if _, err := tx.Exec(ctx, sql, tenant, id, tenant, fmt.Sprintf("P%d-%d", wi, n), randomData(r, fields[tenant])); err != nil {
							return err
						}
						live = append(live, struct{ tenant, id string }{tenant, id})
					}
					return nil
				})
				if err != nil {
					t.Fatalf("%s op %d insert: %v", role, op, err)
				}
			case k < 8: // UPDATE OF data: a whole new data, one key set, or one key removed
				rec := live[r.IntN(len(live))]
				sql := `UPDATE records SET data = $3::jsonb, version = version + 1 WHERE tenant_id = $1 AND id = $2`
				args := []any{rec.tenant, rec.id, randomData(r, fields[rec.tenant])}
				if r.IntN(2) == 0 {
					f := fields[rec.tenant][r.IntN(len(fields[rec.tenant]))]
					if v, ok := randomValue(r, f.vt); ok {
						sql = `UPDATE records SET data = jsonb_set(data, ARRAY[$4::text], $3::jsonb), version = version + 1 WHERE tenant_id = $1 AND id = $2`
						args = []any{rec.tenant, rec.id, v, f.key}
					} else {
						sql = `UPDATE records SET data = data - $3::text, version = version + 1 WHERE tenant_id = $1 AND id = $2`
						args = []any{rec.tenant, rec.id, f.key}
					}
				}
				if _, err := execIn(conn, rec.tenant, recordWrite(t, super, sql), args...); err != nil {
					t.Fatalf("%s op %d update: %v", role, op, err)
				}
			case k == 8: // a rolled-back write leaves nothing behind
				rec := live[r.IntN(len(live))]
				_ = inTenantTx(conn, rec.tenant, func(tx pgx.Tx) error {
					sql := recordWrite(t, super, `UPDATE records SET data = $3::jsonb, version = version + 1 WHERE tenant_id = $1 AND id = $2`)
					if _, err := tx.Exec(ctx, sql, rec.tenant, rec.id, randomData(r, fields[rec.tenant])); err != nil {
						t.Fatalf("%s op %d update before rollback: %v", role, op, err)
					}
					return fmt.Errorf("roll back")
				})
			default: // DELETE, as the superuser
				i := r.IntN(len(live))
				rec := live[i]
				exec(t, super, deleteRecord, rec.tenant, rec.id)
				live = slices.Delete(live, i, i+1)
			}
		}
		if d := queryString(t, super, sideDiff); d != "" {
			t.Fatalf("%s: side tables differ from the recompute: %s", role, d)
		}
	}

	// The fixture exercised every side table and every writer, and a
	// removing field has no row.
	for _, tbl := range []string{"rfv_text", "rfv_num", "rfv_ts", "rfv_bool"} {
		if queryString(t, super, "SELECT count(*)::text FROM "+tbl) == "0" {
			t.Errorf("%s is empty: the fixture never wrote it", tbl)
		}
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM rfv_num v JOIN record_index_fields f
		ON f.tenant_id = v.tenant_id AND f.id = v.field_id WHERE f.state = 'removing'`); got != "0" {
		t.Errorf("removing field has %s side rows, want 0", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM rfv_text WHERE ord > 0`); got == "0" {
		t.Error("no array element past ord 0 was indexed: the fixture never wrote an array")
	}
	// A row of one tenant never names another tenant's field.
	if got := queryString(t, super, `SELECT count(*)::text FROM (
		SELECT tenant_id, field_id FROM rfv_text UNION ALL SELECT tenant_id, field_id FROM rfv_num
		UNION ALL SELECT tenant_id, field_id FROM rfv_ts UNION ALL SELECT tenant_id, field_id FROM rfv_bool) v
		WHERE NOT EXISTS (SELECT 1 FROM record_index_fields f WHERE f.tenant_id = v.tenant_id AND f.id = v.field_id)`); got != "0" {
		t.Errorf("%s side rows name a field outside their tenant", got)
	}
}

// An UPDATE that leaves data out of its SET list does not run the trigger,
// and one that sets data to the same value rewrites nothing (the
// IS DISTINCT FROM test): the side row keeps its ctid.
func TestP702UnchangedFieldIsNotRewritten(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	triggerFields(t, super, tenantA)
	app := asRole(t, "nava_app")
	const id = "0190c000-0000-7000-8000-000000000001"
	if _, err := execIn(app, tenantA, recordWrite(t, super, insertTypedRecord), tenantA, id, tenantA, "U-1", `{"f_text": "a", "f_num": 1}`); err != nil {
		t.Fatal(err)
	}
	ctid := func(tbl string) string {
		return queryString(t, super, "SELECT ctid::text FROM "+tbl+" WHERE record_id = $1", id)
	}
	text, num := ctid("rfv_text"), ctid("rfv_num")

	for _, sql := range []string{
		`UPDATE records SET state = 'open', version = version + 1 WHERE tenant_id = $1 AND id = $2`,
		`UPDATE records SET data = data, version = version + 1 WHERE tenant_id = $1 AND id = $2`,
	} {
		if _, err := execIn(app, tenantA, recordWrite(t, super, sql), tenantA, id); err != nil {
			t.Fatal(err)
		}
	}
	if got := ctid("rfv_text"); got != text {
		t.Errorf("rfv_text row moved from %s to %s: an unchanged field was rewritten", text, got)
	}
	if _, err := execIn(app, tenantA, recordWrite(t, super, `UPDATE records SET data = jsonb_set(data, '{f_num}', '2'), version = version + 1 WHERE tenant_id = $1 AND id = $2`), tenantA, id); err != nil {
		t.Fatal(err)
	}
	if got := ctid("rfv_text"); got != text {
		t.Errorf("rfv_text row moved from %s to %s when only f_num changed", text, got)
	}
	if got := ctid("rfv_num"); got == num {
		t.Errorf("rfv_num row still at %s after f_num changed", got)
	}
}

// uncounted reads the trigger's counter setting in tx: entity definition ->
// count.
func uncounted(ctx context.Context, tx pgx.Tx) (map[string]int64, error) {
	var s string
	if err := tx.QueryRow(ctx, "SELECT coalesce(current_setting($1, true), '')", uncoercedSetting).Scan(&s); err != nil {
		return nil, err
	}
	if s == "" {
		return nil, nil
	}
	var m map[string]int64
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, fmt.Errorf("%s = %q: %w", uncoercedSetting, s, err)
	}
	return m, nil
}

// Acceptance (R5): a write whose values do not coerce succeeds, writes no
// row for them, and adds their number to the transaction's counter, per
// entity definition. A rolled-back savepoint takes its count back; a value of
// the setting the trigger did not write is discarded, never fails the write.
func TestP702UncoercedValuesAreCounted(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	triggerFields(t, super, tenantA)
	ctx := context.Background()
	const id = "0190c000-0000-7000-8000-000000000002"

	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		exec(t, super, deleteRecord, tenantA, id)
		err := inTenantTx(conn, tenantA, func(tx pgx.Tx) error {
			// f_num "1": 1; f_bool "yes": 1; f_tags: 1 of its 3 non-null
			// elements (null is no value); f_ts without offset: 1;
			// f_gone is removing: not looked at. f_text coerces.
			data := `{"f_text": "ok", "f_num": "1", "f_bool": "yes", "f_tags": ["a", 1, null, "b"],
				"f_ts": "2026-11-15T10:00:00", "f_gone": "x"}`
			if _, err := tx.Exec(ctx, recordWrite(t, super, insertTypedRecord), tenantA, id, tenantA, "C-"+role, data); err != nil {
				return fmt.Errorf("insert: %w", err)
			}
			if m, err := uncounted(ctx, tx); err != nil || m[tenantA] != 4 {
				t.Errorf("%s after insert: counter = %v (%v), want {%s: 4}", role, m, err, tenantA)
			}
			var rows int
			if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM rfv_text WHERE record_id = $1)
				+ (SELECT count(*) FROM rfv_num WHERE record_id = $1) + (SELECT count(*) FROM rfv_ts WHERE record_id = $1)
				+ (SELECT count(*) FROM rfv_bool WHERE record_id = $1)`, id).Scan(&rows); err != nil {
				return err
			}
			if rows != 3 { // f_text, and f_tags' "a" and "b"
				t.Errorf("%s: %d side rows, want 3", role, rows)
			}

			// A savepoint that writes a bad value and rolls back leaves the
			// count as it was.
			sp, err := tx.Begin(ctx)
			if err != nil {
				return err
			}
			if _, err := sp.Exec(ctx, recordWrite(t, super, `UPDATE records SET data = jsonb_set(data, '{f_num}', '"2"'), version = version + 1 WHERE tenant_id = $1 AND id = $2`), tenantA, id); err != nil {
				return fmt.Errorf("update in savepoint: %w", err)
			}
			if err := sp.Rollback(ctx); err != nil {
				return err
			}
			if m, _ := uncounted(ctx, tx); m[tenantA] != 4 {
				t.Errorf("%s after a rolled-back savepoint: counter = %v, want 4", role, m)
			}

			// A setting the trigger did not write is discarded.
			for _, junk := range []string{"not json", `[1]`, fmt.Sprintf(`{"%s": "x"}`, tenantA), fmt.Sprintf(`{"%s": 1.5}`, tenantA)} {
				if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", uncoercedSetting, junk); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, recordWrite(t, super, `UPDATE records SET data = jsonb_set(data, '{f_bool}', to_jsonb(random()::text)), version = version + 1 WHERE tenant_id = $1 AND id = $2`), tenantA, id); err != nil {
					return fmt.Errorf("update over %q: %w", junk, err)
				}
				if m, err := uncounted(ctx, tx); err != nil || m[tenantA] != 1 {
					t.Errorf("%s over %q: counter = %v (%v), want {%s: 1}", role, junk, m, err, tenantA)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s: %v", role, err)
		}
	}
}

// Acceptance (CI-8): the records row and the audit_log entry a record write
// stores are byte-identical with and without flagged fields, the write adds
// no other audit_log or audit_chain_head row, and the hash chain (built from
// those columns, internal/audit/chain) is therefore the same. Each run writes
// the same ids and values and rolls back, so the only difference is the
// registry.
func TestP702AuditBytesUnchanged(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	app := asRole(t, "nava_app")
	ctx := context.Background()
	const (
		id      = "0190c000-0000-7000-8000-000000000003"
		auditID = "0190c000-0000-7000-8000-0000000000a3"
		data    = `{"f_text": "high", "f_num": 1.50, "f_ts": "2026-11-15", "f_bool": true, "f_tags": ["a", "b"], "f_gone": 7, "bad": "x"}`
	)
	write := `WITH r AS (` + insertTypedRecord + ` RETURNING tenant_id, id, version)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
		SELECT tenant_id, $6, 'record', 'create', 'record', id, version, 'system', 'system' FROM r`
	update := `WITH r AS (UPDATE records SET data = data || '{"f_num": 2, "f_text": "low"}', version = version + 1
		WHERE tenant_id = $1 AND id = $2 RETURNING tenant_id, id, version)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_before, version_after, actor_system, source)
		SELECT tenant_id, $3, 'record', 'update', 'record', id, version - 1, version, 'system', 'system' FROM r`

	// run writes the record and its two entries as nava_app, snapshots them
	// inside the transaction and rolls back, so every run writes the same
	// ids and values. The audit columns are every one nava_app may read
	// but occurred_at and txid, which name the transaction.
	errRollback := errors.New("roll back")
	run := func() (snapshot string, sideRows int) {
		t.Helper()
		var rec, entries, heads string
		err := inTenantTx(app, tenantA, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, write, tenantA, id, tenantA, "A-1", data, auditID); err != nil {
				return fmt.Errorf("insert: %w", err)
			}
			if _, err := tx.Exec(ctx, update, tenantA, id, "0190c000-0000-7000-8000-0000000000b3"); err != nil {
				return fmt.Errorf("update: %w", err)
			}
			if err := tx.QueryRow(ctx, `SELECT (to_jsonb(r) - 'created_at' - 'updated_at')::text || ' data=' || r.data::text
				FROM records r WHERE tenant_id = $1 AND id = $2`, tenantA, id).Scan(&rec); err != nil {
				return err
			}
			if err := tx.QueryRow(ctx, `SELECT coalesce(string_agg(row_to_json(a)::text, ' ' ORDER BY a.id), '') FROM (
				SELECT tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_before,
				       version_after, actor_principal_id, actor_system, source, source_ref, change_set_id, causation_id,
				       causation_depth, request_id, backfilled, content_sha256, diff_sha256, diff_truncated, seq,
				       sealed_at, erased_at
				FROM audit_log WHERE tenant_id = $1 AND subject_id = $2) a`, tenantA, id).Scan(&entries); err != nil {
				return err
			}
			if err := tx.QueryRow(ctx, `SELECT count(*)::text FROM audit_chain_head WHERE tenant_id = $1`, tenantA).Scan(&heads); err != nil {
				return err
			}
			if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM rfv_text WHERE record_id = $1)
				+ (SELECT count(*) FROM rfv_num WHERE record_id = $1) + (SELECT count(*) FROM rfv_ts WHERE record_id = $1)
				+ (SELECT count(*) FROM rfv_bool WHERE record_id = $1)`, id).Scan(&sideRows); err != nil {
				return err
			}
			return errRollback
		})
		if !errors.Is(err, errRollback) {
			t.Fatal(err)
		}
		return rec + " | " + entries + " | heads=" + heads, sideRows
	}

	without, rows := run()
	if rows != 0 {
		t.Fatalf("with no flagged field the write left %d side rows", rows)
	}
	triggerFields(t, super, tenantA)
	with, rows := run()
	if rows == 0 {
		t.Fatal("with flagged fields the write left no side rows: the trigger did not run")
	}
	if with != without {
		t.Errorf("record and audit bytes differ with flagged fields:\n without %s\n with    %s", without, with)
	}
	if !strings.Contains(without, `"action":"update"`) {
		t.Errorf("the snapshot holds no update entry: %s", without)
	}
}

// The BEFORE DELETE purge clears every side row of the record, of every
// type, and only that record's, so the NO ACTION record FKs pass.
func TestP702DeletePurgesSideRows(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	triggerFields(t, super, tenantA)
	app := asRole(t, "nava_app")
	data := `{"f_text": "a", "f_num": 1, "f_ts": "2026-11-15", "f_bool": false, "f_tags": ["x", "y"]}`
	for i, id := range []string{"0190c000-0000-7000-8000-000000000004", "0190c000-0000-7000-8000-000000000005"} {
		if _, err := execIn(app, tenantA, recordWrite(t, super, insertTypedRecord), tenantA, id, tenantA, fmt.Sprintf("D-%d", i), data); err != nil {
			t.Fatal(err)
		}
	}
	count := func(id string) string {
		return queryString(t, super, `SELECT ((SELECT count(*) FROM rfv_text WHERE record_id = $1)
			+ (SELECT count(*) FROM rfv_num WHERE record_id = $1) + (SELECT count(*) FROM rfv_ts WHERE record_id = $1)
			+ (SELECT count(*) FROM rfv_bool WHERE record_id = $1))::text`, id)
	}
	if got := count("0190c000-0000-7000-8000-000000000004"); got != "6" {
		t.Fatalf("side rows before delete = %s, want 6", got)
	}
	exec(t, super, deleteRecord, tenantA, "0190c000-0000-7000-8000-000000000004")
	if got := count("0190c000-0000-7000-8000-000000000004"); got != "0" {
		t.Errorf("side rows after delete = %s, want 0", got)
	}
	if got := count("0190c000-0000-7000-8000-000000000005"); got != "6" {
		t.Errorf("the other record's side rows = %s, want 6", got)
	}
}

// The functions are SECURITY INVOKER with the D4/D5 SET list, the triggers
// fire on exactly the D4 events, and only nava_app and nava_worker may
// execute record_field_coerce.
func TestP702Catalog(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	for _, fn := range []string{"record_field_coerce", "record_field_values_sync", "record_field_values_purge"} {
		got := queryString(t, super, `SELECT p.prosecdef::text || ' ' || p.provolatile::text || ' ' || array_to_string(p.proconfig, ',')
			FROM pg_proc p WHERE p.proname = $1 AND p.pronamespace = 'public'::regnamespace`, fn)
		vol := "v"
		if fn == "record_field_coerce" {
			vol = "s"
		}
		if want := "false " + vol + " search_path=pg_catalog, public,TimeZone=UTC,DateStyle=ISO, YMD"; got != want {
			t.Errorf("%s: secdef, volatility, config = %q, want %q", fn, got, want)
		}
	}
	got := queryString(t, super, `SELECT string_agg(tgname || ':' || pg_get_triggerdef(oid), ' | ' ORDER BY tgname)
		FROM pg_trigger WHERE tgrelid = 'records'::regclass AND tgname LIKE 'records_field_values_%'`)
	want := "records_field_values_purge:CREATE TRIGGER records_field_values_purge BEFORE DELETE ON public.records FOR EACH ROW EXECUTE FUNCTION record_field_values_purge()" +
		" | records_field_values_sync:CREATE TRIGGER records_field_values_sync AFTER INSERT OR UPDATE OF data ON public.records FOR EACH ROW EXECUTE FUNCTION record_field_values_sync()"
	if got != want {
		t.Errorf("triggers:\n got  %s\n want %s", got, want)
	}
	got = queryString(t, super, `SELECT coalesce(string_agg(coalesce(r.rolname, 'PUBLIC') || ':' || a.privilege_type, ' ' ORDER BY coalesce(r.rolname, 'PUBLIC'), a.privilege_type), '')
		FROM pg_proc p, aclexplode(p.proacl) a LEFT JOIN pg_roles r ON r.oid = a.grantee
		WHERE p.proname LIKE 'record\_field\_%' AND coalesce(r.rolname, '') <> 'nava_migrator'`)
	if want := "nava_app:EXECUTE nava_worker:EXECUTE"; got != want {
		t.Errorf("EXECUTE on the field-index functions = %q, want %q", got, want)
	}
}

// The migration goes down and up again, and down leaves neither function nor
// trigger behind.
func TestP702MigrationDownUp(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	super := superNava(t)
	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.DownTo(context.Background(), fieldIndexTrigger-1); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := queryString(t, super, `SELECT (SELECT count(*) FROM pg_proc WHERE proname LIKE 'record\_field\_%')
		+ (SELECT count(*) FROM pg_trigger WHERE tgname LIKE 'records\_field\_values\_%')`); got != "0" {
		t.Errorf("after down: %s functions and triggers left", got)
	}
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}
