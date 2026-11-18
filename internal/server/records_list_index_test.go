package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

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
		sql := existsTerm(f, next)
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
