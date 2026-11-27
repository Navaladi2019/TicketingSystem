package server

import (
	"context"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// List filters on data fields (ADR-006 D6, D7, P7-06). After the masked check,
// each data filter is resolved against the field-index registry of the entity
// type: not flagged is 400 field_not_filterable, backfilling is 422 field_not_ready, ready compiles to one
// correlated EXISTS on the typed side table inside the list statement.

// maxTextBytes is the longest value record_field_coerce stores for a text
// field (db/migrations/20261127100200_field_index_trigger.sql); a longer one
// can match nothing and is refused as a value of the wrong type.
const maxTextBytes = 2000

// The shapes record_field_coerce accepts for a timestamp: an RFC 3339
// date-time with an explicit offset, or a day.
var (
	tsDateTimeRE = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])[Tt]([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9](\.[0-9]+)?([Zz]|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`)
	tsDayRE      = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$`)
)

type errString string

func (e errString) Error() string { return string(e) }

// notReadyError is 422 field_not_ready: the registry's whole percentage is the
// only thing the body says about the field (ADR-006 D6).
type notReadyError struct{ progress int }

func (e *notReadyError) Error() string { return "list filter on a field that is backfilling" }

func notFilterable(prop string) *problemError {
	return &problemError{code: problem.FieldNotFilterable,
		fields: []problem.FieldError{invalid("/filter/"+escapePointer(prop), "not_filterable")},
		err:    errString("list filter on a field that is not flagged filterable")}
}

// resolveFilters applies D6 steps 2 to 4 to every data filter of pl, in the
// order the request named them, and parses the values by the registry's value
// type. The masked check ran before it. It reads the registry only.
func (h *handlers) resolveFilters(ctx context.Context, tx pgx.Tx, tenantID, defID uuid.UUID, entity string, def *metadata.Compiled, pl *listPlan) error {
	hasData := false
	for _, f := range pl.filters {
		if f.col == "" {
			hasData = true
			break
		}
	}
	if !hasData {
		return nil
	}
	reg, err := pipeline.IndexFields(ctx, tx, tenantID, defID)
	if err != nil {
		return err
	}
	for i := range pl.filters {
		f := &pl.filters[i]
		if f.col != "" {
			continue
		}
		ptr := "/filter/" + escapePointer(f.prop)
		row, flagged := reg[f.prop]
		if flagged && row.State == fieldindex.StateRemoving {
			flagged = false
		}
		if !flagged {
			return notFilterable(f.prop)
		}
		if row.State == fieldindex.StateBackfilling {
			progress := min(max(row.Progress, 0), 100)
			return &notReadyError{progress: progress}
		}
		if (f.op != "eq" && f.op != "in") && row.ValueType != fieldindex.TypeNum && row.ValueType != fieldindex.TypeTS {
			return listInvalid(ptr, "operator")
		}
		for _, raw := range f.raws {
			v, ok := coerceFilterValue(row.ValueType, raw)
			if !ok {
				return listInvalid(ptr, "type")
			}
			f.values = append(f.values, v)
		}
		f.idx = &row
	}
	return nil
}

// coerceFilterValue parses one raw filter value by the rules of
// record_field_coerce for value type vt: text at most 2,000 bytes, a finite
// double, true or false, an RFC 3339 date-time with an offset or a day (00:00
// UTC).
func coerceFilterValue(vt fieldindex.ValueType, raw string) (any, bool) {
	switch vt {
	case fieldindex.TypeText:
		return raw, len(raw) <= maxTextBytes
	case fieldindex.TypeNum:
		if !numberRE.MatchString(raw) {
			return nil, false
		}
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsInf(n, 0) || math.IsNaN(n) {
			return nil, false
		}
		return n, true
	case fieldindex.TypeBool:
		if raw != "true" && raw != "false" {
			return nil, false
		}
		return raw == "true", true
	case fieldindex.TypeTS:
		switch {
		case tsDateTimeRE.MatchString(raw):
			t, err := time.Parse(time.RFC3339Nano, strings.ToUpper(raw))
			return t, err == nil
		case tsDayRE.MatchString(raw):
			t, err := time.Parse(time.DateOnly, raw)
			return t, err == nil
		}
	}
	return nil, false
}

// sqlOp maps a filter operator to its SQL comparison; the operator is one
// planList accepted, so the fallback is eq.
func sqlOp(op string) string {
	switch op {
	case "lt":
		return "<"
	case "lte":
		return "<="
	case "gt":
		return ">"
	case "gte":
		return ">="
	}
	return "="
}

// existsTerm renders a ready data filter as a correlated EXISTS on the typed
// side table (ADR-006 D7). The table name comes from fieldindex.SideTable's
// fixed set, never from the request. fs are all the filters of one field: their
// comparisons are ANDed inside the one EXISTS, so on an array field a single
// element must satisfy every bound (ADR-006 D7), and the planner sees one
// range, not two unselective halves. A text value is compared under COLLATE
// "C", the column's own collation, so the value index serves it.
func existsTerm(fs []listFilter, next func(v any, cast string) string) string {
	f := fs[0]
	table, _ := f.idx.ValueType.SideTable()
	fieldArg := next(f.idx.ID.String(), "uuid")
	return "EXISTS (SELECT 1 FROM " + table + " v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = " + fieldArg + " AND " + fieldCmps(fs, "v.value", next) + ")"
}

// fieldCmps renders the comparisons of fs, all the filters of one field, over
// col, ANDed. It is the one place a data filter's operator and value become
// SQL, for the stored value (existsTerm) and for a default (defaultTrue).
func fieldCmps(fs []listFilter, col string, next func(v any, cast string) string) string {
	f := fs[0]
	cast := map[fieldindex.ValueType]string{
		fieldindex.TypeText: "text", fieldindex.TypeNum: "double precision",
		fieldindex.TypeTS: "timestamptz", fieldindex.TypeBool: "boolean",
	}[f.idx.ValueType]
	collate := ""
	if f.idx.ValueType == fieldindex.TypeText {
		collate = ` COLLATE "C"`
	}
	cmps := make([]string, 0, len(fs))
	for _, f := range fs {
		if f.op == "in" {
			cmps = append(cmps, col+" = ANY("+next(arrayOf(f.values), cast+"[]")+")")
		} else {
			cmps = append(cmps, col+" "+sqlOp(f.op)+" "+next(f.values[0], cast)+collate)
		}
	}
	return strings.Join(cmps, " AND ")
}

// defaultColumn is the column of record_field_coerce that holds a value of
// type vt.
func defaultColumn(vt fieldindex.ValueType) string {
	switch vt {
	case fieldindex.TypeNum:
		return "value_num"
	case fieldindex.TypeTS:
		return "value_ts"
	case fieldindex.TypeBool:
		return "value_bool"
	}
	return "value_text"
}

// defaultTrue renders "the filters of this field are true for def, the type's
// default for it" (ADR-009 D12), evaluated in SQL through record_field_coerce,
// the coercion the side table's own rows went through, never by Go-side
// equality. It reads no row of records, so the planner evaluates it once per
// statement.
func defaultTrue(fs []listFilter, def any, next func(v any, cast string) string) (string, error) {
	raw, err := json.Marshal(def)
	if err != nil {
		return "", err
	}
	f := fs[0]
	vt := string(f.idx.ValueType)
	return "EXISTS (SELECT 1 FROM record_field_coerce(" + next(string(raw), "jsonb") + ", " + next(vt, "text") + ") d WHERE " +
		fieldCmps(fs, "d."+defaultColumn(f.idx.ValueType), next) + ")", nil
}

// fieldAbsent renders "the stored JSON lacks the key", the ADR-009 D12 test for
// an absent field: key presence, the test the D7 queue fill and ApplyDefaults
// use. A stored null or a value that does not coerce has the key, so it is not
// absent and the default does not stand in for it. The key is bound from the
// registry row of the compiled schema, never from request text. S22 attack 4
// allows this one site, by function name and fragment.
func fieldAbsent(f listFilter, next func(v any, cast string) string) string {
	return "NOT (r.data ? " + next(f.idx.Key, "text") + ")"
}

// arrayOf returns vs as a typed slice pgx encodes as a PostgreSQL array.
func arrayOf(vs []any) any {
	switch vs[0].(type) {
	case string:
		out := make([]string, len(vs))
		for i, v := range vs {
			out[i], _ = v.(string)
		}
		return out
	case float64:
		out := make([]float64, len(vs))
		for i, v := range vs {
			out[i], _ = v.(float64)
		}
		return out
	case time.Time:
		out := make([]time.Time, len(vs))
		for i, v := range vs {
			out[i], _ = v.(time.Time)
		}
		return out
	}
	out := make([]bool, len(vs))
	for i, v := range vs {
		out[i], _ = v.(bool)
	}
	return out
}

// selectivityBand names how many rows a list matched, from the list's own
// capped total (ADR-006 D7: no separate count): none, under 10, under 100,
// under 1,000, or more than the cap.
func selectivityBand(total int64, capped bool) string {
	switch {
	case capped:
		return "over_1000"
	case total == 0:
		return "none"
	case total < 10:
		return "lt_10"
	case total < 100:
		return "lt_100"
	}
	return "lt_1000"
}

// observeIndexedList records a field-index list's duration (ADR-006 D13, R-a).
// The labels are the selectivity band and ACL presence; never a tenant, a
// field or a value.
func (h *handlers) observeIndexedList(ctx context.Context, d time.Duration, page *pipeline.ListPage, narrowing bool) {
	if h.listIndexed == nil {
		return
	}
	acl := "open"
	if narrowing {
		acl = "narrowing"
	}
	// A count that ran out of budget has no total: these are the large lists, so
	// the band is "unknown", never "none" (ADR-006-A2 A2-5).
	band := "unknown"
	if page.TotalState != pipeline.TotalUnknown {
		band = selectivityBand(page.Total, page.TotalCapped)
	}
	h.listIndexed.Record(ctx, d.Seconds(), metric.WithAttributes(
		attribute.String("selectivity_band", band),
		attribute.String("acl", acl),
		attribute.String("total_state", string(page.TotalState))))
}

// IndexTerm rewrites the side-table EXISTS of one data field: real is the term
// existsTerm rendered, table its side table and field the registry id. Only
// the I5 mutants of the isolation suite supply one (WithListIndexTerm,
// integration builds only); production lists render real unchanged.
type IndexTerm func(real, table, field string) string
