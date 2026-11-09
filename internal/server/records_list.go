package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// List endpoint (TICK-619, P3-34). The page and `total` share one predicate:
// tenant (RLS), entity type, authz.CompileRead, then the filters. Every field
// the request names (a filter, the sort, the default sort, the sort key a
// cursor carries) goes through authz.CheckQueryFields before any SQL runs.

const (
	listRecordsRoute    = "listRecords"
	defaultRecordsLimit = 50
	maxInValues         = 100
	defaultSortField    = "updated_at"
)

type rawQueryKey struct{}

// withRawQuery makes the request's query string available to handlers that
// take dynamic parameters (filter[<property>]) the generated binder cannot.
func withRawQuery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), rawQueryKey{}, r.URL.Query())))
	})
}

var (
	filterKeyRE = regexp.MustCompile(`^filter\[([^\[\]]+)\](\[in\])?$`)
	numberRE    = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)
)

// listFilter is one parsed filter. col is the fixed column ("" for a top-level
// data property); values are SQL-ready: text, uuid strings or JSON documents.
type listFilter struct {
	prop   string
	col    string
	values []any
}

// listPlan is a request resolved against one entity type version.
type listPlan struct {
	filters  []listFilter
	sortProp string // the property or created_at / updated_at named
	sortCol  string
	desc     bool
	limit    int
}

// fields are the top-level names the plan reads, for CheckQueryFields.
func (pl *listPlan) fields() []string {
	out := make([]string, 0, len(pl.filters)+1)
	for _, f := range pl.filters {
		out = append(out, f.prop)
	}
	return append(out, pl.sortProp)
}

func listInvalid(pointer, rule string) *problemError {
	return &problemError{problem.ValidationFailed, []problem.FieldError{invalid(pointer, rule)}, errors.New("invalid list query")}
}

// planList parses the query against def. It touches no database.
func planList(def *metadata.Compiled, q url.Values, sort *string, limit *int) (*listPlan, *problemError) {
	types, fixed := def.PropertyTypes(), def.FixedColumns()
	secret := def.SecretProperties()
	pl := &listPlan{sortProp: defaultSortField, sortCol: defaultSortField, desc: true, limit: defaultRecordsLimit}
	if limit != nil {
		pl.limit = *limit
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if !strings.HasPrefix(k, "filter") {
			continue
		}
		m := filterKeyRE.FindStringSubmatch(k)
		if m == nil {
			return nil, listInvalid("/filter", "form")
		}
		prop, isIn := m[1], m[2] != ""
		ptr := "/filter/" + escapePointer(prop)
		if len(q[k]) != 1 {
			return nil, listInvalid(ptr, "repeated")
		}
		typ, known := types[prop]
		if !known || slices.Contains(secret, prop) {
			return nil, listInvalid(ptr, "unknown")
		}
		raws := []string{q[k][0]}
		if isIn {
			raws = strings.Split(q[k][0], ",")
			if len(raws) > maxInValues {
				return nil, listInvalid(ptr, "too_many")
			}
		}
		f := listFilter{prop: prop, col: fixed[prop]}
		for _, raw := range raws {
			v, ok := filterValue(f.col, prop, typ, raw)
			if !ok {
				return nil, listInvalid(ptr, "type")
			}
			f.values = append(f.values, v)
		}
		pl.filters = append(pl.filters, f)
	}
	if sort != nil && *sort != "" {
		name, desc := strings.CutPrefix(*sort, "-")
		col, isFixed := fixed[name]
		switch {
		case name == "created_at" || name == "updated_at":
			pl.sortProp, pl.sortCol = name, name
		case isFixed:
			pl.sortProp, pl.sortCol = name, col
		default:
			return nil, listInvalid("/sort", "unsortable")
		}
		pl.desc = desc
	}
	return pl, nil
}

// filterValue converts one raw filter value for a fixed column or a data
// property of type typ. A data value is the JSON document {prop: value}.
func filterValue(col, prop, typ, raw string) (any, bool) {
	switch col {
	case "":
	case metadata.ColumnState:
		return raw, raw != ""
	default:
		id, err := uuid.Parse(raw)
		return id.String(), err == nil
	}
	var v any
	switch typ {
	case "string":
		v = raw
	case "integer":
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, false
		}
		v = n
	case "number":
		if !numberRE.MatchString(raw) {
			return nil, false
		}
		v = json.Number(raw)
	case "boolean":
		if raw != "true" && raw != "false" {
			return nil, false
		}
		v = raw == "true"
	default:
		return nil, false // objects, arrays and unions have no equality filter
	}
	doc, err := json.Marshal(map[string]any{prop: v})
	return string(doc), err == nil
}

// checkPlan is the masked-field gate for everything the request names.
func checkPlan(pol *authz.Policy, p identity.Principal, pl *listPlan) error {
	return maskedFieldError(authz.CheckQueryFields(pol, p, pl.fields()))
}

func maskedFieldError(err error) error {
	var mf *authz.ErrMaskedField
	if errors.As(err, &mf) {
		return &problemError{problem.MaskedField, []problem.FieldError{invalid("/"+escapePointer(mf.Field), "masked")}, err}
	}
	return err
}

// where renders the filters as one predicate with placeholders from base+1.
func (pl *listPlan) where(base int) (string, []any) {
	var (
		terms []string
		args  []any
	)
	next := func(v any, cast string) string {
		args = append(args, v)
		return "$" + strconv.Itoa(base+len(args)) + "::" + cast
	}
	for _, f := range pl.filters {
		switch f.col {
		case "":
			ors := make([]string, 0, len(f.values))
			for _, v := range f.values {
				ors = append(ors, "r.data @> "+next(v, "jsonb"))
			}
			terms = append(terms, "("+strings.Join(ors, " OR ")+")")
		case metadata.ColumnState:
			terms = append(terms, "r.state = ANY("+next(textsOf(f.values), "text[]")+")")
		default:
			terms = append(terms, "r."+f.col+" = ANY("+next(textsOf(f.values), "uuid[]")+")")
		}
	}
	if len(terms) == 0 {
		return "true", nil
	}
	return strings.Join(terms, " AND "), args
}

// hasDataFilter reports whether any filter reads the data document.
func (pl *listPlan) hasDataFilter() bool {
	return slices.ContainsFunc(pl.filters, func(f listFilter) bool { return f.col == "" })
}

func textsOf(vs []any) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i], _ = v.(string)
	}
	return out
}

// listPos is the position a cursor carries: the sort it was issued for, the
// last row's sort value (nil for NULL) and id.
type listPos struct {
	Sort string  `json:"s"`
	Desc bool    `json:"d"`
	Val  *string `json:"v"`
	ID   string  `json:"i"`
	// Upd is the last row's updated_at, carried by the sorts that tie-break on
	// it (state and the foreign keys); empty for created_at and updated_at.
	Upd string `json:"u,omitempty"`
}

// tsLayouts returns the layouts of a timestamptz rendered as text.
func tsLayouts() []string {
	return []string{"2006-01-02 15:04:05.999999Z07:00", "2006-01-02 15:04:05.999999Z07"}
}

func validTS(v string) bool {
	return slices.ContainsFunc(tsLayouts(), func(l string) bool { _, err := time.Parse(l, v); return err == nil })
}

func (pl *listPlan) validPos(pos listPos) bool {
	if pos.Sort != pl.sortProp || pos.Desc != pl.desc {
		return false
	}
	if _, err := uuid.Parse(pos.ID); err != nil {
		return false
	}
	if pipeline.TieBreaksOnUpdatedAt(pl.sortCol) != (pos.Upd != "") || (pos.Upd != "" && !validTS(pos.Upd)) {
		return false
	}
	if pos.Val == nil {
		return true
	}
	switch pl.sortCol {
	case "created_at", "updated_at":
		return validTS(*pos.Val)
	case metadata.ColumnState:
		return true
	}
	_, err := uuid.Parse(*pos.Val)
	return err == nil
}

// keyset renders the page predicate after pos, placeholders from base+1. It
// follows the store's ORDER BY: descending is NULLS FIRST, ascending NULLS
// LAST, and the tie-break is id (created_at, updated_at) or updated_at, id
// (state and the foreign keys). A row comparison keeps the predicate a range
// condition on the sort index, so a deep page costs what page 1 does; the
// NULL segment of a nullable column is handled apart.
func (pl *listPlan) keyset(pos listPos, base int) (string, []any) {
	col := "r." + pl.sortCol
	n := base
	arg := func(cast string) string { n++; return "$" + strconv.Itoa(n) + "::" + cast }
	op := ">"
	if pl.desc {
		op = "<"
	}
	// tail is the row comparison of the columns after col.
	tailCols, tailArgs, args := "r.id", "", []any{}
	if pipeline.TieBreaksOnUpdatedAt(pl.sortCol) {
		tailCols = "r.updated_at, r.id"
	}
	valArg := ""
	if pos.Val != nil {
		cast := "uuid"
		switch pl.sortCol {
		case "created_at", "updated_at":
			cast = "timestamptz"
		case metadata.ColumnState:
			cast = "text"
		}
		valArg = arg(cast)
		args = append(args, *pos.Val)
	}
	if pos.Upd != "" {
		tailArgs = arg("timestamptz") + ", "
		args = append(args, pos.Upd)
	}
	tailArgs += arg("uuid")
	args = append(args, pos.ID)
	tail := "(" + tailCols + ") " + op + " (" + tailArgs + ")"
	if pos.Val == nil {
		if pl.desc {
			// NULLs come first: the rest of the segment, then every non-NULL.
			return "(" + col + " IS NOT NULL OR " + tail + ")", args
		}
		return "(" + col + " IS NULL AND " + tail + ")", args
	}
	pred := "((" + col + ", " + tailCols + ") " + op + " (" + valArg + ", " + tailArgs + "))"
	if !pl.desc && nullableSort(pl.sortCol) {
		pred = "(" + pred + " OR " + col + " IS NULL)" // NULLS LAST follow every value
	}
	return pred, args
}

// nullableSort reports whether col may hold NULL.
func nullableSort(col string) bool {
	switch col {
	case "assignee_id", "assignment_group_id", "requester_id":
		return true
	}
	return false
}

// ListRecords serves GET /api/v1/records/{entity}.
func (h *handlers) ListRecords(ctx context.Context, req api.ListRecordsRequestObject) (api.ListRecordsResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil || h.acl == nil || h.cursors == nil {
		return nil, notImplemented()
	}
	if impossibleKey(req.Entity) {
		return h.notFound(ctx), nil
	}
	q, _ := ctx.Value(rawQueryKey{}).(url.Values)
	cq := url.Values{}
	for k, v := range q {
		cq[k] = v
	}
	cq.Set("_entity", req.Entity)
	claims, err := cursor.ClaimsFor(ctx, listRecordsRoute, cq)
	if err != nil {
		return nil, err
	}
	var out api.RecordList
	err = tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		def, curVersion, err := h.store.Current(ctx, tx, req.Entity)
		if err != nil {
			return err
		}
		defID, err := pipeline.DefinitionID(ctx, tx, p.TenantID, req.Entity)
		if err != nil {
			return err
		}
		pol, err := h.policy(ctx, tx, p, req.Entity)
		if err != nil {
			return err
		}
		pl, perr := planList(def, q, req.Params.Sort, req.Params.Limit)
		if perr != nil {
			return perr
		}
		// Before any SQL: filters, sort and the default sort.
		if err := checkPlan(pol, p, pl); err != nil {
			return err
		}
		var pos *listPos
		if req.Params.Cursor != nil {
			raw, err := h.cursors.Decode(claims, *req.Params.Cursor)
			if err != nil {
				return &problemError{problem.InvalidCursor, nil, err}
			}
			pos = new(listPos)
			if err := json.Unmarshal(raw, pos); err != nil {
				return &problemError{problem.InvalidCursor, nil, err}
			}
			// The sort key the cursor carries is a named field too.
			if err := maskedFieldError(authz.CheckQueryFields(pol, p, []string{pos.Sort})); err != nil {
				return err
			}
			if !pl.validPos(*pos) {
				return &problemError{problem.InvalidCursor, nil, errors.New("cursor does not fit the query")}
			}
		}
		readSQL, readArgs := authz.CompileRead(pol, p, "r", 2)
		filt, filtArgs := pl.where(2 + len(readArgs))
		lq := pipeline.ListQuery{
			TenantID: p.TenantID, DefinitionID: defID,
			Where: readSQL + " AND " + filt, Args: append(slices.Clone(readArgs), filtArgs...),
			SortColumn: pl.sortCol, Desc: pl.desc, Limit: pl.limit + 1,
		}
		if pos != nil {
			lq.Keyset, lq.KeysetArgs = pl.keyset(*pos, 2+len(lq.Args))
		}
		lq.DataFilter, lq.Timeout = pl.hasDataFilter(), h.dataFilterTimeout
		page, err := pipeline.List(ctx, tx, lq)
		if err != nil {
			if errors.Is(err, pipeline.ErrStatementTimeout) {
				return &problemError{problem.FilterTooExpensive, nil, err}
			}
			return err
		}
		out = api.RecordList{Items: make([]api.Record, 0, min(len(page.Records), pl.limit)), Total: page.Total}
		if page.TotalCapped {
			out.TotalCapped = &page.TotalCapped
		}
		fixedBy := map[int]map[string]string{}
		for _, rec := range page.Records[:min(len(page.Records), pl.limit)] {
			fixed, seen := fixedBy[rec.SchemaVersion]
			if !seen {
				if fixed, err = h.fixedColumnsOf(ctx, tx, req.Entity, rec.SchemaVersion, curVersion, def); err != nil {
					return err
				}
				fixedBy[rec.SchemaVersion] = fixed
			}
			out.Items = append(out.Items, shapeRecord(pol, p, rec, def.ApplyDefaults(rec.Payload(fixed)), fixed))
		}
		if len(page.Records) > pl.limit {
			last := pl.limit - 1
			pos := listPos{Sort: pl.sortProp, Desc: pl.desc, Val: page.SortKeys[last], ID: page.Records[last].ID.String()}
			if pipeline.TieBreaksOnUpdatedAt(pl.sortCol) {
				pos.Upd = page.Records[last].UpdatedAt.UTC().Format(tsLayouts()[0])
			}
			raw, err := json.Marshal(pos)
			if err != nil {
				return err
			}
			next, err := h.cursors.Encode(claims, raw)
			if err != nil {
				return fmt.Errorf("list records: %w", err)
			}
			out.NextCursor = &next
		}
		return nil
	})
	if errors.Is(err, entitytypes.ErrNotFound) || errors.Is(err, pipeline.ErrNotFound) {
		return h.notFound(ctx), nil
	}
	if err != nil {
		return nil, err
	}
	return api.ListRecords200JSONResponse(out), nil
}

func (r *notFoundResponse) VisitListRecordsResponse(w http.ResponseWriter) error { return r.write(w) }

var _ api.ListRecordsResponseObject = (*notFoundResponse)(nil)
