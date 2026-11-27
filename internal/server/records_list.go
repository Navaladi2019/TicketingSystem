package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
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
	filterKeyRE = regexp.MustCompile(`^filter\[([^\[\]]+)\](?:\[(in|eq|lt|lte|gt|gte)\])?$`)
	numberRE    = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)
)

// listFilter is one parsed filter. col is the fixed column ("" for a top-level
// data property); values are SQL-ready: text, uuid strings or JSON documents.
type listFilter struct {
	prop   string
	col    string
	values []any
	// op is eq, in, lt, lte, gt or gte. A fixed column takes eq and in only.
	op string
	// raws are the unparsed values of a data filter: how they parse depends on
	// the registry row of the field, so resolve parses them (ADR-006 D6).
	raws []string
	// idx is the ready registry row a data filter reads through its side
	// table; nil for a fixed column.
	idx *pipeline.IndexField
}

// listPlan is a request resolved against one entity type version.
type listPlan struct {
	filters  []listFilter
	sortProp string // the property or created_at / updated_at named
	sortCol  string
	desc     bool
	limit    int
	// indexTerm rewrites the side-table EXISTS of one data field; nil is the
	// real term. Only the I5 mutants set it (WithListIndexTerm, integration
	// builds only).
	indexTerm IndexTerm
	// win is the open backfill window of the type, when one touches a filter
	// of this request (ADR-009 D12); nil otherwise.
	win *filterWindow
}

// filterWindow is what D12 needs of the type's newest schema backfill job
// while its window is open and a data filter names a field that job defaults.
type filterWindow struct {
	// fields are the filtered fields in the job's default_keys, with the
	// default the current version gives each.
	fields map[string]any
	// union is true for the UNION ALL shape: the job is running or has failed
	// records, no more than 1,000 rows remain, and exactly one filtered field
	// is in the window. Otherwise every field in the window uses the OR shape.
	union bool
	// target is the newest job's target_version, the key of its queue.
	target int
}

// windowFor returns the default of prop's filter when the window covers it.
func (pl *listPlan) windowFor(prop string) (any, bool) {
	if pl.win == nil {
		return nil, false
	}
	d, ok := pl.win.fields[prop]
	return d, ok
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
		prop, op := m[1], m[2]
		if op == "" {
			op = "eq"
		}
		isIn := op == "in"
		ptr := "/filter/" + escapePointer(prop)
		if len(q[k]) != 1 {
			return nil, listInvalid(ptr, "repeated")
		}
		_, known := types[prop]
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
		f := listFilter{prop: prop, col: fixed[prop], op: op}
		if f.col == "" {
			// A data filter is parsed once the registry says how (resolve).
			f.raws = raws
			pl.filters = append(pl.filters, f)
			continue
		}
		if op != "eq" && op != "in" {
			return nil, listInvalid(ptr, "operator")
		}
		for _, raw := range raws {
			v, ok := filterValue(f.col, raw)
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

// filterValue converts one raw filter value for a fixed column.
func filterValue(col, raw string) (any, bool) {
	if col == metadata.ColumnState {
		return raw, raw != ""
	}
	id, err := uuid.Parse(raw)
	return id.String(), err == nil
}

// checkPlan is the masked-field gate for everything the request names.
func checkPlan(pol *authz.Policy, p identity.Principal, pl *listPlan) error {
	return maskedFieldError(authz.CheckQueryFields(pol, p, pl.fields()))
}

func maskedFieldError(err error) error {
	if mf, ok := errors.AsType[*authz.ErrMaskedField](err); ok {
		return &problemError{problem.MaskedField, []problem.FieldError{invalid("/"+escapePointer(mf.Field), "masked")}, err}
	}
	return err
}

// where renders the filters as one predicate with placeholders from base+1.
// While a backfill window is open, a filter on a field the job defaults is
// answered for records not yet rewritten too (ADR-009 D12): in the OR shape,
// (<stored>) OR (<true for the default> AND <field absent>); in the UNION ALL
// shape this is the first arm and the stored predicate alone.
func (pl *listPlan) where(base int) (string, []any) { return pl.render(base, false) }

// whereSecond renders the second arm of the UNION ALL shape: the records that
// lack the field in storage, whose default satisfies the filter, and that the
// stored predicate does not already return. It exists only when win.union.
func (pl *listPlan) whereSecond(base int) (string, []any) { return pl.render(base, true) }

func (pl *listPlan) render(base int, second bool) (string, []any) {
	var (
		terms []string
		args  []any
	)
	next := func(v any, cast string) string {
		args = append(args, v)
		return "$" + strconv.Itoa(base+len(args)) + "::" + cast
	}
	done := map[uuid.UUID]bool{}
	for _, f := range pl.filters {
		switch f.col {
		case "":
			if f.idx != nil {
				// One EXISTS per field: every comparison of the field is ANDed
				// inside it, so one element must satisfy them all (ADR-006 D7).
				if done[f.idx.ID] {
					continue
				}
				done[f.idx.ID] = true
				var group []listFilter
				for _, g := range pl.filters {
					if g.col == "" && g.idx != nil && g.idx.ID == f.idx.ID {
						group = append(group, g)
					}
				}
				stored := existsTerm(group, next)
				if pl.indexTerm != nil {
					table, _ := f.idx.ValueType.SideTable()
					stored = pl.indexTerm(stored, table, f.idx.ID.String())
				}
				def, inWindow := pl.windowFor(f.prop)
				switch {
				case !inWindow || (pl.win.union && !second):
					terms = append(terms, stored)
				case second:
					dt, _ := defaultTrue(group, def, next)
					terms = append(terms, "("+queued(pl.win.target, next)+" AND "+dt+" AND "+fieldAbsent(f, next)+" AND NOT "+stored+")")
				default:
					dt, _ := defaultTrue(group, def, next)
					terms = append(terms, "("+stored+" OR ("+dt+" AND "+fieldAbsent(f, next)+"))")
				}
				continue
			}
			// A data filter with no side table is never answered from the data
			// document (the JSONB scan is gone): fail closed, match nothing.
			terms = append(terms, "false")
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

// hasIndexFilter reports whether any filter reads a typed side table.
func (pl *listPlan) hasIndexFilter() bool {
	return slices.ContainsFunc(pl.filters, func(f listFilter) bool { return f.idx != nil })
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
	maps.Copy(cq, q)
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
		pl.indexTerm = h.listIndexTerm
		// Before any SQL: filters, sort and the default sort.
		if err := checkPlan(pol, p, pl); err != nil {
			return err
		}
		// ADR-006 D6 after masked: flagged, ready, then the values.
		if err := h.resolveFilters(ctx, tx, p.TenantID, defID, req.Entity, def, pl); err != nil {
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
		// A rule that ORs subjects (requester, assignee, share) is read as one
		// index-ordered branch per subject (TICK-644); CompileRead is the single
		// statement when there is nothing to split. One plan for both, so a
		// voided policy is counted once per request (ADR-010 D5b).
		// The caller's units for an under leaf, before rendering; the leaf
		// reads the closure in SQL (ADR-010 D7 as amended, TICK-1034).
		if pol, err = authz.ResolveUnder(ctx, tx, pol, p); err != nil {
			return err
		}
		// The page shape of each region term, from a capped probe in this
		// transaction (ADR-010-A2 A2-5, A2-5a). It selects SQL text only:
		// the rows are the same in every shape. A probe error is a 500,
		// never another shape. Nothing about the choice reaches the
		// response or a log line (A2-6 requirement 2).
		shapes, err := authz.ChooseListShapes(ctx, tx, pol, p, defID, pl.limit+1)
		if err != nil {
			return err
		}
		read := authz.ReadListShaped(pol, p, "r", 2, shapes)
		// ADR-009 D12: the newest backfill job, read once, decides the shape of
		// a filter on a field it defaults.
		if err := h.resolveWindow(ctx, tx, p.TenantID, defID, def, pl); err != nil {
			return err
		}
		filt, filtArgs := pl.where(2 + len(read.Args))
		var filt2 string
		if pl.win != nil && pl.win.union {
			var args2 []any
			filt2, args2 = pl.whereSecond(2 + len(read.Args) + len(filtArgs))
			filtArgs = append(slices.Clone(filtArgs), args2...)
		}
		lq := pipeline.ListQuery{
			TenantID: p.TenantID, DefinitionID: defID,
			Where: filt, Args: append(slices.Clone(read.Args), filtArgs...), Branches: read.Branches,
			CountBranches: read.CountBranches, Materialize: read.Materialize,
			SortColumn: pl.sortCol, Desc: pl.desc, Limit: pl.limit + 1,
		}
		if read.Branches == nil {
			lq.Where = read.Where + " AND " + filt
			lq.CountWhere = read.CountWhere + " AND " + filt
		}
		if filt2 != "" {
			lq.UnionWhere = filt2
			if read.Branches == nil {
				lq.UnionWhere = read.Where + " AND " + filt2
				lq.UnionCountWhere = read.CountWhere + " AND " + filt2
			}
		}
		for _, d := range read.Drivers {
			lq.CountDrivers = append(lq.CountDrivers, pipeline.CountDriver{From: d.From, Key: d.Key})
		}
		if pos != nil {
			lq.Keyset, lq.KeysetArgs = pl.keyset(*pos, 2+len(lq.Args))
		}
		lq.BudgetCount = pl.hasIndexFilter()
		lq.CountDelay, lq.SkipCount = h.countDelay, h.skipCount
		if h.listStatement != nil {
			if stmt, stmtArgs, err := pipeline.PageStatement(lq); err == nil {
				h.listStatement(stmt, stmtArgs)
			}
		}
		started := time.Now()
		page, err := pipeline.List(ctx, tx, lq)
		if err == nil && pl.hasIndexFilter() {
			narrowing := read.Branches != nil || len(read.Args) > 0 || strings.Contains(read.Where, "r.")
			h.observeIndexedList(ctx, time.Since(started), page, narrowing)
		}
		if err != nil {
			return err
		}
		out = api.RecordList{Items: make([]api.Record, 0, min(len(page.Records), pl.limit)), TotalState: api.RecordListTotalState(page.TotalState)}
		if page.TotalState != pipeline.TotalUnknown {
			out.Total = &page.Total
		}
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
			// ADR-009 D11: the effective value, then the field masks.
			out.Items = append(out.Items, shapeRecord(pol, p, rec, pipeline.Effective(def, rec, fixed), fixed))
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
