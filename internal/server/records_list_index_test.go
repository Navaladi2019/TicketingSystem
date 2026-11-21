package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

func TestCoerceFilterValue(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	ok := []struct {
		vt   fieldindex.ValueType
		raw  string
		want any
	}{
		{fieldindex.TypeText, "abc", "abc"},
		{fieldindex.TypeNum, "-1.5e2", -150.0},
		{fieldindex.TypeNum, "3", 3.0},
		{fieldindex.TypeBool, "true", true},
		{fieldindex.TypeTS, "2026-11-15", day},
		{fieldindex.TypeTS, "2026-11-15T05:30:00+05:30", day},
		{fieldindex.TypeTS, "2026-11-15t00:00:00z", day},
	}
	for _, c := range ok {
		got, good := coerceFilterValue(c.vt, c.raw)
		if !good {
			t.Errorf("%s %q refused", c.vt, c.raw)
			continue
		}
		if tt, isT := c.want.(time.Time); isT {
			if g, _ := got.(time.Time); !g.Equal(tt) {
				t.Errorf("%s %q = %v, want %v", c.vt, c.raw, got, c.want)
			}
		} else if got != c.want {
			t.Errorf("%s %q = %v, want %v", c.vt, c.raw, got, c.want)
		}
	}
	bad := []struct {
		vt  fieldindex.ValueType
		raw string
	}{
		{fieldindex.TypeText, strings.Repeat("a", maxTextBytes+1)},
		{fieldindex.TypeNum, "0x10"}, {fieldindex.TypeNum, "NaN"}, {fieldindex.TypeNum, "1e400"}, {fieldindex.TypeNum, ""},
		{fieldindex.TypeBool, "yes"}, {fieldindex.TypeBool, "TRUE"},
		{fieldindex.TypeTS, "2026-11-15T00:00:00"}, {fieldindex.TypeTS, "2026-13-01"}, {fieldindex.TypeTS, "yesterday"},
		{"nope", "x"},
	}
	for _, c := range bad {
		if _, good := coerceFilterValue(c.vt, c.raw); good {
			t.Errorf("%s %q accepted", c.vt, c.raw)
		}
	}
	if _, good := coerceFilterValue(fieldindex.TypeText, strings.Repeat("a", maxTextBytes)); !good {
		t.Error("a 2,000-byte text is the longest the index stores and must be accepted")
	}
}

func TestExistsTermShape(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	for _, c := range []struct {
		vt     fieldindex.ValueType
		op     string
		vals   []any
		table  string
		wantIn string
	}{
		{fieldindex.TypeText, "eq", []any{"a"}, "rfv_text", `v.value = $2::text COLLATE "C"`},
		{fieldindex.TypeText, "in", []any{"a", "b"}, "rfv_text", `v.value = ANY($2::text[])`},
		{fieldindex.TypeNum, "gte", []any{1.0}, "rfv_num", `v.value >= $2::double precision`},
		{fieldindex.TypeTS, "lt", []any{time.Now()}, "rfv_ts", `v.value < $2::timestamptz`},
		{fieldindex.TypeBool, "eq", []any{true}, "rfv_bool", `v.value = $2::boolean`},
	} {
		n := 0
		var args []any
		next := func(v any, cast string) string {
			n++
			args = append(args, v)
			return "$" + string(rune('0'+n)) + "::" + cast
		}
		// The field id is the first argument, so the value is $2.
		f := listFilter{prop: "f", op: c.op, values: c.vals, idx: &pipeline.IndexField{ID: id, ValueType: c.vt}}
		sql := existsTerm([]listFilter{f}, next, false)
		for _, want := range []string{"EXISTS (SELECT 1 FROM " + c.table + " v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $1::uuid AND ", c.wantIn} {
			if !strings.Contains(sql, want) {
				t.Errorf("%s %s: %s lacks %q", c.vt, c.op, sql, want)
			}
		}
		if len(args) != 2 || args[0] != id.String() {
			t.Errorf("%s %s: args = %v", c.vt, c.op, args)
		}
	}
}

func TestRangeOnOneFieldCompilesToOneExists(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		vt       fieldindex.ValueType
		table    string
		lo, hi   any
		wantCmps string
	}{
		{fieldindex.TypeTS, "rfv_ts", time.Now(), time.Now().Add(time.Hour), `v.value >= $2::timestamptz AND v.value < $3::timestamptz`},
		{fieldindex.TypeNum, "rfv_num", 1.0, 5.0, `v.value >= $2::double precision AND v.value < $3::double precision`},
	} {
		id := uuid.New()
		pl := &listPlan{filters: []listFilter{
			{prop: "f", op: "gte", values: []any{c.lo}, idx: &pipeline.IndexField{ID: id, ValueType: c.vt}},
			{prop: "f", op: "lt", values: []any{c.hi}, idx: &pipeline.IndexField{ID: id, ValueType: c.vt}},
		}}
		sql, args := pl.where(0)
		if n := strings.Count(sql, "EXISTS"); n != 1 {
			t.Errorf("%s: %d EXISTS in %s", c.vt, n, sql)
		}
		if !strings.Contains(sql, c.wantCmps) || len(args) != 3 {
			t.Errorf("%s: sql %s args %v", c.vt, sql, args)
		}
	}
	// Two different fields stay two EXISTS.
	pl := &listPlan{filters: []listFilter{
		{prop: "a", op: "gte", values: []any{1.0}, idx: &pipeline.IndexField{ID: uuid.New(), ValueType: fieldindex.TypeNum}},
		{prop: "b", op: "lt", values: []any{2.0}, idx: &pipeline.IndexField{ID: uuid.New(), ValueType: fieldindex.TypeNum}},
	}}
	if sql, _ := pl.where(0); strings.Count(sql, "EXISTS") != 2 {
		t.Errorf("two fields: %s", sql)
	}
}

func TestNotReadyResponseCarriesOnlyTheProgress(t *testing.T) {
	t.Parallel()
	reg := loadTestRegistry(t)
	h := &handlers{reg: reg, log: slog.New(slog.DiscardHandler)}
	respond := h.responseError(apihttp.New(nil, reg, h.log, ""))
	for _, p := range []int{0, 42, 100} {
		rec := httptest.NewRecorder()
		respond(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil), &notReadyError{progress: p})
		body := rec.Body.String()
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(body, `"code":"field_not_ready"`) ||
			!strings.Contains(body, `"index_progress":`+strconv.Itoa(p)) {
			t.Errorf("progress %d = %d %s", p, rec.Code, body)
		}
		if !strings.Contains(body, `"index_progress"`) || strings.Contains(body, `"errors"`) {
			t.Errorf("progress %d: body %s must carry index_progress and no field detail", p, body)
		}
	}
	rec := httptest.NewRecorder()
	respond(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil), notFilterable("priority"))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"field_not_filterable"`) {
		t.Errorf("not filterable = %d %s", rec.Code, rec.Body)
	}
}

func TestLegacyScanIsKeyedByTypeAndField(t *testing.T) {
	t.Parallel()
	if !fieldindex.LegacyScan("incident", "priority") {
		t.Error("incident priority must stay on the scan until P7-12")
	}
	for _, c := range [][2]string{{"ticket", "priority"}, {"incident", "site"}, {"change", "priority"}, {"", "priority"}} {
		if fieldindex.LegacyScan(c[0], c[1]) {
			t.Errorf("%v must not reach the legacy scan", c)
		}
	}
}

func TestSelectivityBand(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		total  int64
		capped bool
		want   string
	}{
		{0, false, "none"}, {1, false, "lt_10"}, {9, false, "lt_10"}, {10, false, "lt_100"}, {99, false, "lt_100"},
		{100, false, "lt_1000"}, {999, false, "lt_1000"}, {1000, false, "lt_1000"}, {1000, true, "over_1000"},
	} {
		if got := selectivityBand(c.total, c.capped); got != c.want {
			t.Errorf("selectivityBand(%d, %v) = %q, want %q", c.total, c.capped, got, c.want)
		}
	}
}

func TestIndexedListErrorsAndNilHistogram(t *testing.T) {
	t.Parallel()
	if (&notReadyError{progress: 3}).Error() == "" || notFilterable("f").Error() == "" {
		t.Error("the list errors must carry a message for the log")
	}
	// No histogram configured: nothing is recorded and nothing panics.
	(&handlers{}).observeIndexedList(t.Context(), time.Second, &pipeline.ListPage{Total: 5}, true)
}

// placeholderRE finds the numbered placeholders of a statement, in order.
var placeholderRE = regexp.MustCompile(`\$[0-9]+::[a-z]+( precision)?(\[\])?`)

// ADR-006-A2-5: the count's probe form of a filter binds the same arguments
// as the page's form, in the same order, so the count runs with the page's
// arguments; only the value side and the OFFSET differ.
func TestCountWhereBindsTheSameArgumentsAsWhere(t *testing.T) {
	t.Parallel()
	now := time.Now()
	for _, c := range []struct {
		vt     fieldindex.ValueType
		fs     [][2]any // op, value
		opaque string
	}{
		{fieldindex.TypeText, [][2]any{{"eq", "a"}}, `(v.value || '') = $3::text COLLATE "C"`},
		{fieldindex.TypeText, [][2]any{{"in", []any{"a", "a", "b"}}}, `(v.value || '') = ANY($3::text[])`},
		{fieldindex.TypeNum, [][2]any{{"gte", 1.0}, {"lt", 5.0}}, `(v.value + 0) >= $3::double precision AND (v.value + 0) < $4::double precision`},
		{fieldindex.TypeTS, [][2]any{{"gt", now}, {"lte", now.Add(time.Hour)}}, `(v.value + interval '0') > $3::timestamptz AND (v.value + interval '0') <= $4::timestamptz`},
		{fieldindex.TypeBool, [][2]any{{"eq", true}}, `(v.value OR false) = $3::boolean`},
	} {
		id := uuid.New()
		var fs []listFilter
		for _, f := range c.fs {
			vals := []any{f[1]}
			if vs, ok := f[1].([]any); ok {
				vals = vs
			}
			fs = append(fs, listFilter{prop: "f", op: f[0].(string), values: vals, idx: &pipeline.IndexField{ID: id, ValueType: c.vt}})
		}
		// A fixed-column filter after it, so later placeholders must line up too.
		fs = append(fs, listFilter{prop: "state", col: "state", op: "eq", values: []any{"new"}})
		pl := &listPlan{filters: fs}
		if w := pl.countWhere(1); w != "" {
			t.Errorf("%s: no filter is broad, but countWhere = %s", c.vt, w)
		}
		pl.setBroad(id, true)
		where, args := pl.where(1)
		count := pl.countWhere(1)
		if strings.Contains(where, "OFFSET") || strings.Contains(where, "(v.value") {
			t.Errorf("%s: the page's form changed: %s", c.vt, where)
		}
		if !strings.Contains(count, c.opaque) || !strings.Contains(count, " OFFSET 0)") {
			t.Errorf("%s: count %s lacks %q or OFFSET 0", c.vt, count, c.opaque)
		}
		wp, cp := placeholderRE.FindAllString(where, -1), placeholderRE.FindAllString(count, -1)
		if !slices.Equal(wp, cp) || len(wp) != len(args) {
			t.Errorf("%s: placeholders differ: where %v, count %v, %d args", c.vt, wp, cp, len(args))
		}
		pl.clearBroad()
		if w := pl.countWhere(1); w != "" {
			t.Errorf("%s: cleared, but countWhere = %s", c.vt, w)
		}
	}
}

// The pre-count reads one field of the tenant on the value index (plain
// value side), stops at broadCap, and binds the tenant as $1 and the field as
// $2 (ADR-006-A2-7).
func TestPreCountShape(t *testing.T) {
	t.Parallel()
	id, tenant := uuid.New(), uuid.New()
	args := []any{tenant, id.String()}
	next := func(v any, cast string) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args)) + "::" + cast
	}
	fs := []listFilter{
		{prop: "f", op: "gte", values: []any{1.0}, idx: &pipeline.IndexField{ID: id, ValueType: fieldindex.TypeNum}},
		{prop: "f", op: "lt", values: []any{2.0}, idx: &pipeline.IndexField{ID: id, ValueType: fieldindex.TypeNum}},
	}
	sql := preCountSQL(fs, next, defaultBroadCap)
	// A2-7: the rendered SQL, by regex, carries the tenant, the field and the cap.
	for _, re := range []string{`v\.tenant_id = \$1\b`, `v\.field_id = \$2(::uuid)?\b`, `LIMIT ` + strconv.Itoa(defaultBroadCap) + `\b`} {
		if !regexp.MustCompile(re).MatchString(sql) {
			t.Errorf("pre-count %s does not match %s", sql, re)
		}
	}
	want := "SELECT count(*) FROM (SELECT 1 FROM rfv_num v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND " +
		"v.value >= $3::double precision AND v.value < $4::double precision ORDER BY v.value, v.record_id LIMIT " + strconv.Itoa(defaultBroadCap) + ") c"
	if sql != want || len(args) != 4 {
		t.Errorf("pre-count:\n got %s\nwant %s\nargs %v", sql, want, args)
	}
	// An in list is one ordered arm per distinct value, each also bound by the
	// field's other comparisons.
	args = []any{tenant, id.String()}
	fs = []listFilter{
		{prop: "f", op: "in", values: []any{"a", "b", "a"}, idx: &pipeline.IndexField{ID: id, ValueType: fieldindex.TypeText}},
		{prop: "f", op: "gte", values: []any{"a"}, idx: &pipeline.IndexField{ID: id, ValueType: fieldindex.TypeText}},
	}
	sql = preCountSQL(fs, next, defaultBroadCap)
	lim := " LIMIT " + strconv.Itoa(defaultBroadCap)
	arm := func(v, g string) string {
		return "(SELECT 1 FROM rfv_text v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND v.value = " + v + `::text COLLATE "C" AND v.value >= ` + g + `::text COLLATE "C" ORDER BY v.record_id` + lim + ")"
	}
	want = "SELECT count(*) FROM (SELECT 1 FROM (" + arm("$3", "$4") + " UNION ALL " + arm("$5", "$6") + ") u" + lim + ") c"
	if sql != want || len(args) != 6 {
		t.Errorf("in pre-count:\n got %s\nwant %s\nargs %v", sql, want, args)
	}
}

// fieldGroups keeps one group per field, in field-id order, and skips fixed
// columns and legacy-scan filters (A2-12's canonical order).
func TestFieldGroups(t *testing.T) {
	t.Parallel()
	a := uuid.MustParse("00000000-0000-4000-8000-00000000000a")
	b := uuid.MustParse("00000000-0000-4000-8000-00000000000b")
	pl := &listPlan{filters: []listFilter{
		{prop: "x", idx: &pipeline.IndexField{ID: b, ValueType: fieldindex.TypeNum}},
		{prop: "state", col: "state"},
		{prop: "y", idx: &pipeline.IndexField{ID: a, ValueType: fieldindex.TypeText}},
		{prop: "legacy", legacy: true},
		{prop: "x", idx: &pipeline.IndexField{ID: b, ValueType: fieldindex.TypeNum}},
	}}
	g := pl.fieldGroups()
	if len(g) != 2 || len(g[0]) != 1 || g[0][0].idx.ID != a || len(g[1]) != 2 || g[1][0].idx.ID != b {
		t.Errorf("groups = %v", g)
	}
}

// Security condition 3 (ADR-006-A2-6): the indexed-list histogram carries
// exactly its two labels, the selectivity band and ACL presence. The
// pre-count's outcome and count have no path to it.
func TestIndexedListMetricHasOnlyItsLabels(t *testing.T) {
	t.Parallel()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	hist, err := mp.Meter("t").Float64Histogram("h")
	if err != nil {
		t.Fatal(err)
	}
	(&handlers{listIndexed: hist}).observeIndexedList(t.Context(), time.Millisecond, &pipeline.ListPage{Total: 3}, true)
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &rm); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			for _, dp := range m.Data.(metricdata.Histogram[float64]).DataPoints {
				for _, kv := range dp.Attributes.ToSlice() {
					keys = append(keys, string(kv.Key))
				}
			}
		}
	}
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"acl", "selectivity_band"}) {
		t.Errorf("labels = %v", keys)
	}
}
