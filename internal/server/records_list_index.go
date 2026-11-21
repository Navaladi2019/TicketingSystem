package server

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
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

// The bounded count probe (ADR-006-A2, TICK-1022). A narrowing list's capped
// count joins a broad filter's side rows by a hash that spills, and the
// planner picks it from the side table's visibility map, so the count reads
// 100-490 ms on a table autovacuum has not caught up with
// (docs/data/tick-1022-post-backfill-plan.md). For a broad filter on a small
// visible set, the count probes the side table once per visible record
// instead. Both bounds are calibrated at 400k records per type (A2-10).
const (
	// defaultBroadCap is the side-row count at which a filter is broad: the
	// pre-count stops there.
	defaultBroadCap = 20000
	// defaultVisibleCap is the most visible records a probing count may walk:
	// a broad value the caller's records rarely carry probes all of them
	// (A2-3a, the visible-set guard).
	defaultVisibleCap = 30000
	// maxPreCounted is the most fields one request pre-counts (A2-12): the
	// first ones in field-id order. Further fields keep today's SQL.
	maxPreCounted = 3
)

// countProbe is the probe's configuration: the two caps, and two hooks that
// see each pre-count's outcome and each probe decision. Only tests set any of
// it (WithCountProbe); the outcome and the count never reach a log, a span or
// a metric (A2-6).
type countProbe struct {
	caps       func() (broad, visible int)
	pre, probe func(bool)
}

// limits are the caps in force: the defaults unless a test set them.
func (c countProbe) limits() (broad, visible int) {
	if c.caps == nil {
		return defaultBroadCap, defaultVisibleCap
	}
	return c.caps()
}

// sideCast is the SQL type of a value type's side value.
func sideCast(vt fieldindex.ValueType) string {
	switch vt {
	case fieldindex.TypeText:
		return "text"
	case fieldindex.TypeNum:
		return "double precision"
	case fieldindex.TypeTS:
		return "timestamptz"
	}
	return "boolean"
}

// opaqueValue wraps v.value so that no index of the side table can serve the
// comparison, and a probe seeks the primary key (tenant_id, record_id,
// field_id) instead of walking a value range per record. Each keeps every
// stored value as it is: + 0 keeps +-Infinity, + interval '0' keeps
// +-infinity, and appending the empty string keeps the column's "C"
// collation. OR false folds away: a bool probe relies on OFFSET 0 alone and
// seeks (value, record_id) on its value index, one seek too.
func opaqueValue(vt fieldindex.ValueType) string {
	switch vt {
	case fieldindex.TypeText:
		return "(v.value || '')"
	case fieldindex.TypeNum:
		return "(v.value + 0)"
	case fieldindex.TypeTS:
		return "(v.value + interval '0')"
	}
	return "(v.value OR false)"
}

// sideCmp renders the comparisons of one field's filters on its side value,
// binding the values through next in filter order. probe renders the value
// opaque; the placeholders and their order are the same either way.
func sideCmp(fs []listFilter, next func(v any, cast string) string, probe bool) string {
	vt := fs[0].idx.ValueType
	cast := sideCast(vt)
	collate := ""
	if vt == fieldindex.TypeText {
		collate = ` COLLATE "C"`
	}
	col := "v.value"
	if probe {
		col = opaqueValue(vt)
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

// existsTerm renders a ready data filter as a correlated EXISTS on the typed
// side table (ADR-006 D7). The table name comes from fieldindex.SideTable's
// fixed set, never from the request. fs are all the filters of one field: their
// comparisons are ANDed inside the one EXISTS, so on an array field a single
// element must satisfy every bound (ADR-006 D7), and the planner sees one
// range, not two unselective halves. A text value is compared under COLLATE
// "C", the column's own collation, so the value index serves it.
//
// probe is the count's form of a broad filter (ADR-006-A2): OFFSET 0 keeps the
// EXISTS a per-record SubPlan, which PostgreSQL neither pulls up into a semi
// join nor hashes, and the value is opaque, so each probe is one seek.
func existsTerm(fs []listFilter, next func(v any, cast string) string, probe bool) string {
	table, _ := fs[0].idx.ValueType.SideTable()
	fieldArg := next(fs[0].idx.ID.String(), "uuid")
	tail := ")"
	if probe {
		tail = " OFFSET 0)"
	}
	return "EXISTS (SELECT 1 FROM " + table + " v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = " + fieldArg + " AND " + sideCmp(fs, next, probe) + tail
}

// preCountSQL is the pre-count of one field's filters (ADR-006-A2): its side
// rows in the tenant, on the value index, up to broadCap. It is the only
// statement besides the list's EXISTS that reads a side table (D7 as amended
// by A2), and it carries no row ACL: its result only picks the count's shape
// and is never returned (A2-6, R13).
//
// LIMIT bounds the read only on a plan that stops early, and the planner
// prices the pre-count from statistics pooled over every tenant's rows: a
// value broad elsewhere and rare here would take a Seq Scan or a Bitmap Heap
// Scan that reads every match before the LIMIT applies (Technical Architect,
// A2 rev 0, finding 5). So the rows are ordered by the value index's own
// columns, which only an ordered scan of that index returns without a sort.
// An in list is one capped arm per distinct value: PostgreSQL 16 returns no
// order from = ANY on a column after the first, so each arm is an equality.
func preCountSQL(fs []listFilter, next func(v any, cast string) string, broadCap int) string {
	table, _ := fs[0].idx.ValueType.SideTable()
	lim := " LIMIT " + strconv.Itoa(broadCap)
	head := "SELECT 1 FROM " + table + " v WHERE v.tenant_id = $1 AND v.field_id = $2::uuid AND "
	in := slices.IndexFunc(fs, func(f listFilter) bool { return f.op == "in" })
	if in < 0 {
		return "SELECT count(*) FROM (" + head + sideCmp(fs, next, false) + " ORDER BY v.value, v.record_id" + lim + ") c"
	}
	rest := slices.Delete(slices.Clone(fs), in, in+1)
	var vals []any
	for _, v := range fs[in].values {
		if !slices.Contains(vals, v) {
			vals = append(vals, v)
		}
	}
	arms := make([]string, len(vals))
	for k, v := range vals {
		eq := []listFilter{{prop: fs[in].prop, op: "eq", values: []any{v}, idx: fs[in].idx}}
		arms[k] = "(" + head + sideCmp(append(eq, rest...), next, false) + " ORDER BY v.record_id" + lim + ")"
	}
	return "SELECT count(*) FROM (SELECT 1 FROM (" + strings.Join(arms, " UNION ALL ") + ") u" + lim + ") c"
}

// fieldGroups are the plan's ready side-table filters grouped by field, in
// field-id order (A2-12's canonical order, independent of how the request
// spells its filters).
func (pl *listPlan) fieldGroups() [][]listFilter {
	var out [][]listFilter
	at := map[uuid.UUID]int{}
	for _, f := range pl.filters {
		if f.col != "" || f.idx == nil {
			continue
		}
		i, ok := at[f.idx.ID]
		if !ok {
			i = len(out)
			at[f.idx.ID] = i
			out = append(out, nil)
		}
		out[i] = append(out[i], f)
	}
	slices.SortFunc(out, func(a, b []listFilter) int { return strings.Compare(a[0].idx.ID.String(), b[0].idx.ID.String()) })
	return out
}

// markBroad runs the pre-count of the first maxPreCounted ready filtered
// fields (A2-12) and marks the
// fields that reach broadCap. It reports whether any did. The caller runs it
// only on a narrowing list, after every refusal (ADR-006-A2-2).
func (pl *listPlan) markBroad(ctx context.Context, tx pgx.Tx, tenant uuid.UUID, cp countProbe) (bool, error) {
	broadCap, _ := cp.limits()
	found := false
	groups := pl.fieldGroups()
	if len(groups) > maxPreCounted {
		groups = groups[:maxPreCounted]
	}
	for _, fs := range groups {
		args := []any{tenant, fs[0].idx.ID.String()}
		next := func(v any, cast string) string {
			args = append(args, v)
			return "$" + strconv.Itoa(len(args)) + "::" + cast
		}
		var n int
		if err := tx.QueryRow(ctx, preCountSQL(fs, next, broadCap), args...).Scan(&n); err != nil {
			return false, fmt.Errorf("list filter pre-count: %w", err)
		}
		broad := n >= broadCap
		if cp.pre != nil {
			cp.pre(broad)
		}
		if broad {
			found = true
			pl.setBroad(fs[0].idx.ID, true)
		}
	}
	return found, nil
}

// setBroad marks or clears every filter of field id.
func (pl *listPlan) setBroad(id uuid.UUID, b bool) {
	for i := range pl.filters {
		if pl.filters[i].idx != nil && pl.filters[i].idx.ID == id {
			pl.filters[i].broad = b
		}
	}
}

// clearBroad unmarks every filter: the count keeps the plain form.
func (pl *listPlan) clearBroad() {
	for i := range pl.filters {
		pl.filters[i].broad = false
	}
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
		attribute.String("acl", acl)))
}
