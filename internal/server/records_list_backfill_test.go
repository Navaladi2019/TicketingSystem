package server

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

func idxFilter(prop string, vt fieldindex.ValueType, op string, vals ...any) listFilter {
	return listFilter{prop: prop, op: op, values: vals, idx: &pipeline.IndexField{ID: uuid.New(), Key: prop, ValueType: vt}}
}

// Outside a window a data filter is the stored predicate alone.
func TestRenderWithoutAWindowIsTheStoredPredicate(t *testing.T) {
	pl := &listPlan{filters: []listFilter{idxFilter("severity", fieldindex.TypeText, "eq", "low")}}
	where, args := pl.where(2)
	if strings.Contains(where, "record_field_coerce") || strings.Contains(where, "NOT EXISTS") || len(args) != 2 {
		t.Errorf("where = %s (%d args), want the EXISTS only", where, len(args))
	}
}

// The OR shape: stored OR (default is true AND field absent).
func TestRenderORShapeInAnOpenWindow(t *testing.T) {
	pl := &listPlan{
		filters: []listFilter{idxFilter("severity", fieldindex.TypeText, "eq", "low")},
		win:     &filterWindow{fields: map[string]any{"severity": "low"}, target: 2},
	}
	where, _ := pl.where(2)
	for _, want := range []string{" OR (", "record_field_coerce(", "NOT (r.data ? $"} {
		if !strings.Contains(where, want) {
			t.Errorf("OR shape %s lacks %q", where, want)
		}
	}
	if strings.Contains(where, "schema_backfill") {
		t.Errorf("the OR shape reads the job tables: %s", where)
	}
}

// The UNION ALL shape: the first arm is the stored predicate, the second is
// queued AND default true AND absent AND NOT stored, with its own placeholders.
func TestRenderUnionShapeArms(t *testing.T) {
	pl := &listPlan{
		filters: []listFilter{idxFilter("severity", fieldindex.TypeText, "in", "low", "mid")},
		win:     &filterWindow{fields: map[string]any{"severity": "low"}, union: true, target: 3},
	}
	first, a1 := pl.where(2)
	second, a2 := pl.whereSecond(2 + len(a1))
	if strings.Contains(first, "record_field_coerce") || strings.Contains(first, "schema_backfill") {
		t.Errorf("first arm = %s, want the stored predicate", first)
	}
	for _, want := range []string{"schema_backfill_pending", "schema_backfill_failures", "record_field_coerce(", "AND NOT EXISTS", "$1", "$2"} {
		if !strings.Contains(second, want) {
			t.Errorf("second arm %s lacks %q", second, want)
		}
	}
	if len(a2) <= len(a1) {
		t.Errorf("second arm binds %d args, want more than the first has: %d", len(a2), len(a1))
	}
}

// A field the window does not cover keeps the stored predicate in both arms.
func TestRenderLeavesOtherFieldsAlone(t *testing.T) {
	pl := &listPlan{
		filters: []listFilter{idxFilter("severity", fieldindex.TypeText, "eq", "low"), idxFilter("cost", fieldindex.TypeNum, "gte", 2.5)},
		win:     &filterWindow{fields: map[string]any{"severity": "low"}, union: true, target: 2},
	}
	second, _ := pl.whereSecond(2)
	if strings.Count(second, "record_field_coerce(") != 1 {
		t.Errorf("second arm = %s, want one default test, for severity only", second)
	}
	if _, ok := pl.windowFor("cost"); ok {
		t.Error("cost is not in the window")
	}
	if _, ok := (&listPlan{}).windowFor("severity"); ok {
		t.Error("a plan with no window has no window field")
	}
}

func TestDefaultColumnAndDefaultTrueForEveryType(t *testing.T) {
	for vt, col := range map[fieldindex.ValueType]string{
		fieldindex.TypeText: "value_text", fieldindex.TypeNum: "value_num", fieldindex.TypeTS: "value_ts", fieldindex.TypeBool: "value_bool",
	} {
		if got := defaultColumn(vt); got != col {
			t.Errorf("defaultColumn(%s) = %s, want %s", vt, got, col)
		}
		var vals []any
		switch vt {
		case fieldindex.TypeNum:
			vals = []any{1.5}
		case fieldindex.TypeTS:
			vals = []any{time.Unix(0, 0).UTC()}
		case fieldindex.TypeBool:
			vals = []any{true}
		default:
			vals = []any{"x"}
		}
		next := func(_ any, cast string) string { return "$n::" + cast }
		sql, err := defaultTrue([]listFilter{idxFilter("f", vt, "eq", vals...)}, vals[0], next)
		if err != nil || !strings.Contains(sql, "d."+col) {
			t.Errorf("%s: sql %q err %v, want a comparison on d.%s", vt, sql, err, col)
		}
	}
	// A default that cannot be JSON is reported, not rendered.
	if _, err := defaultTrue([]listFilter{idxFilter("f", fieldindex.TypeText, "eq", "x")}, make(chan int), func(any, string) string { return "" }); err == nil {
		t.Error("an unmarshalable default was rendered")
	}
}

func TestQueuedNamesBothTablesAndTheTarget(t *testing.T) {
	var args []any
	sql := queued(7, func(v any, cast string) string { args = append(args, v); return "$9::" + cast })
	if !strings.Contains(sql, "schema_backfill_pending") || !strings.Contains(sql, "schema_backfill_failures") || strings.Count(sql, "$9::integer") != 2 || len(args) != 1 {
		t.Errorf("queued = %s (%v)", sql, args)
	}
}

func TestUnionShapeByStateAndRemaining(t *testing.T) {
	for _, c := range []struct {
		state     string
		remaining int64
		want      bool
	}{
		{"pending", 3, false}, {"filling", 3, false},
		{"running", 1000, true}, {"running", 1001, false},
		{"failed_records", 0, true}, {"failed_records", 1001, false},
		{"done", 0, false},
	} {
		if got := unionShape(&entitytypes.BackfillStatus{State: c.state, Remaining: c.remaining}); got != c.want {
			t.Errorf("unionShape(%s, %d) = %v, want %v", c.state, c.remaining, got, c.want)
		}
	}
}

func TestBackfillJSONUsesUTC(t *testing.T) {
	loc := time.FixedZone("x", 3600)
	started := time.Date(2026, 1, 2, 3, 4, 5, 0, loc)
	got := backfillJSON(&entitytypes.BackfillStatus{State: "running", TargetVersion: 4, Rewritten: 1, Skipped: 2, Failed: 3, Remaining: 4, StartedAt: &started})
	if got.State != "running" || got.TargetVersion != 4 || got.Failed != 3 || got.Remaining != 4 || got.FinishedAt != nil {
		t.Errorf("backfill = %+v", got)
	}
	if got.StartedAt == nil || got.StartedAt.Location() != time.UTC || got.StartedAt.Hour() != 2 {
		t.Errorf("started_at = %v, want 02:04:05 UTC", got.StartedAt)
	}
	if utcPtr(nil) != nil {
		t.Error("utcPtr(nil) is not nil")
	}
}

// ADR-009 D12: absent is key presence. The window's term is NOT (r.data ? key)
// with the key bound; a stored null or a non-coercible value has the key, so no
// side-row test stands in for it, in the OR shape and in the UNION ALL arm.
// noSideRow matches the old absent term, "no side row for the field", and not
// the stored predicate's own NOT EXISTS, which also compares the value.
var noSideRow = regexp.MustCompile(`NOT EXISTS \(SELECT 1 FROM rfv_\w+ v WHERE [^()]*field_id = \$\d+::uuid\)`)

func TestRenderAbsentIsKeyPresenceNotNoSideRow(t *testing.T) {
	pl := &listPlan{
		filters: []listFilter{idxFilter("severity", fieldindex.TypeText, "eq", "low")},
		win:     &filterWindow{fields: map[string]any{"severity": "low"}, target: 2},
	}
	or, args := pl.where(2)
	pl.win.union = true
	second, args2 := pl.whereSecond(2)
	for name, c := range map[string]struct {
		sql  string
		args []any
	}{"OR": {or, args}, "UNION second arm": {second, args2}} {
		if !strings.Contains(c.sql, "NOT (r.data ? $") {
			t.Errorf("%s: %s lacks the key-presence term", name, c.sql)
		}
		if noSideRow.MatchString(c.sql) || strings.Contains(c.sql, "'severity'") {
			t.Errorf("%s: side-row absence kept or key inlined: %s", name, c.sql)
		}
		if !slices.Contains(c.args, any("severity")) {
			t.Errorf("%s: key not bound as a parameter: %v", name, c.args)
		}
	}
	// Outside the window the term is not there at all.
	out := &listPlan{filters: []listFilter{idxFilter("severity", fieldindex.TypeText, "eq", "low")}}
	if w, _ := out.where(2); strings.Contains(w, "r.data") {
		t.Errorf("outside the window: %s reads data", w)
	}
}
