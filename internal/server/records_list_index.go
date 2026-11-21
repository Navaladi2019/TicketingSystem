package server

import (
	"context"
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
// type: not flagged is 400 field_not_filterable (except the fixed legacy
// allow-list), backfilling is 422 field_not_ready, ready compiles to one
// correlated EXISTS on the typed side table inside the list statement.

// maxTextBytes is the longest value record_field_coerce stores for a text
// field (db/migrations/20261120140100_field_index_trigger.sql); a longer one
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
	types := def.PropertyTypes()
	defaults := def.ApplyDefaults(nil)
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
			if !fieldindex.LegacyScan(entity, f.prop) && (h.legacyScan == nil || !h.legacyScan(entity, f.prop)) {
				return notFilterable(f.prop)
			}
			if f.op != "eq" && f.op != "in" {
				return listInvalid(ptr, "operator")
			}
			if perr := parseLegacy(f, types[f.prop], defaults); perr != nil {
				return perr
			}
			continue
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

// parseLegacy parses the values of a legacy-scan data filter f, a property of
// JSON type typ, into {prop: value} documents for r.data @>. A value equal to
// the current version's default (defaults, from ApplyDefaults) also matches a
// record that leaves the property out (I5, TICK-912).
func parseLegacy(f *listFilter, typ string, defaults map[string]any) *problemError {
	f.legacy = true
	for _, raw := range f.raws {
		v, ok := filterValue("", f.prop, typ, raw)
		if !ok {
			return listInvalid("/filter/"+escapePointer(f.prop), "type")
		}
		f.values = append(f.values, v)
		if d, has := defaults[f.prop]; has && sameFilterValue(v, d) {
			f.absentMatches = true
		}
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
	cast := map[fieldindex.ValueType]string{
		fieldindex.TypeText: "text", fieldindex.TypeNum: "double precision",
		fieldindex.TypeTS: "timestamptz", fieldindex.TypeBool: "boolean",
	}[f.idx.ValueType]
	collate := ""
	if f.idx.ValueType == fieldindex.TypeText {
		collate = ` COLLATE "C"`
	}
	fieldArg := next(f.idx.ID.String(), "uuid")
	cmps := make([]string, 0, len(fs))
	for _, f := range fs {
		if f.op == "in" {
			cmps = append(cmps, "v.value = ANY("+next(arrayOf(f.values), cast+"[]")+")")
		} else {
			cmps = append(cmps, "v.value "+sqlOp(f.op)+" "+next(f.values[0], cast)+collate)
		}
	}
	cmp := strings.Join(cmps, " AND ")
	return "EXISTS (SELECT 1 FROM " + table + " v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = " + fieldArg + " AND " + cmp + ")"
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
	h.listIndexed.Record(ctx, d.Seconds(), metric.WithAttributes(
		attribute.String("selectivity_band", selectivityBand(page.Total, page.TotalCapped)),
		attribute.String("acl", acl),
		attribute.String("total_state", string(page.TotalState))))
}
