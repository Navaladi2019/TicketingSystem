package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// PATCH and history of a record (TICK-358, P3-15). Check order of PATCH:
// authn, entity type and record (the one 404), If-Match missing (428), stale
// (412), body (422), then the authorizer inside the pipeline.

const getRecordHistoryRoute = "getRecordHistory"

// staleError is a failed If-Match: the record is at version current.
type staleError struct{ current int64 }

func (e *staleError) Error() string {
	return "record is at version " + strconv.FormatInt(e.current, 10)
}

var errIfMatchRequired = errors.New("if-match missing")

// ifMatchVersions reads an If-Match value: ok is false when it is missing or
// `*`, which counts as missing. Weak and strong forms are the same, and a
// list matches any of its members. A member that is not one of our ETags is
// version 0, which no record has.
func ifMatchVersions(v *string) (versions []int64, ok bool) {
	if v == nil || strings.TrimSpace(*v) == "" || strings.TrimSpace(*v) == "*" {
		return nil, false
	}
	for _, m := range strings.Split(*v, ",") {
		m = strings.TrimSpace(m)
		m = strings.TrimPrefix(m, "W/")
		m = strings.Trim(m, `"`)
		n, err := strconv.ParseInt(m, 10, 64)
		if err != nil || n < 1 {
			n = 0
		}
		versions = append(versions, n)
	}
	return versions, true
}

// patchBody is the 200 body's data. The row was written under the version the
// pipeline read, which a publish since the handler's own read may have moved
// (TICK-413, TICK-481): map it under that one, as create and GET do. Defaults
// stay the current ones.
func patchBody(ctx context.Context, tx pgx.Tx, defs entitytypes.DefinitionReader, entity string, rec *pipeline.Record, curVersion int, cur *metadata.Compiled) (map[string]any, error) {
	fixed, err := fixedColumnsUnder(ctx, tx, defs, entity, rec.SchemaVersion, curVersion, cur)
	if err != nil {
		return nil, err
	}
	return cur.ApplyDefaults(rec.Payload(fixed)), nil
}

// PatchRecord serves PATCH /api/v1/records/{entity}/{id}.
func (h *handlers) PatchRecord(ctx context.Context, req api.PatchRecordRequestObject) (api.PatchRecordResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil || h.pipe == nil {
		return nil, notImplemented()
	}
	id, perr := uuid.Parse(req.Id)
	if impossibleKey(req.Entity) || perr != nil {
		return h.notFound(ctx), nil
	}
	var raw json.RawMessage
	if req.Body != nil {
		raw = *req.Body
	}
	var (
		rec       *pipeline.Record
		data      map[string]any
		tried     map[string]string // the handler's read; recordError prefers the pipeline's version
		stateProp string
	)
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		cur, curVersion, err := h.store.Current(ctx, tx, req.Entity)
		if err != nil {
			return err
		}
		tried = cur.FixedColumns()
		stateProp = cur.StateProperty()
		before, err := pipeline.Get(ctx, tx, p.TenantID, id)
		if err != nil {
			return err
		}
		if before.EntityKey != req.Entity {
			return pipeline.ErrNotFound
		}
		// Only now may the caller learn that the record needs If-Match.
		versions, ok := ifMatchVersions(req.Params.IfMatch)
		if !ok {
			return errIfMatchRequired
		}
		if !slices.Contains(versions, before.Version) {
			return &staleError{before.Version}
		}
		patch, perr := parsePatchBody(raw, stateProp)
		if perr != nil {
			return perr
		}
		if _, err = h.pipe.Write(ctx, tx, pipeline.Command{
			// Causation stays empty on purpose: origin and depth never come from
			// the request (TICK-88 security review).
			Op: pipeline.OpUpdate, EntityKey: req.Entity, RecordID: id, ExpectedVersion: before.Version,
			Data: patch, Actor: p, Source: pipeline.SourceUser,
		}); err != nil {
			return err
		}
		if rec, err = pipeline.Get(ctx, tx, p.TenantID, id); err != nil {
			return err
		}
		data, err = patchBody(ctx, tx, h.store, req.Entity, rec, curVersion, cur)
		return err
	})
	if err != nil {
		var st *staleError
		switch {
		case errors.Is(err, entitytypes.ErrNotFound) || errors.Is(err, pipeline.ErrNotFound):
			return h.notFound(ctx), nil
		case errors.Is(err, errIfMatchRequired):
			return nil, &problemError{problem.IfMatchRequired, nil, err}
		case errors.As(err, &st):
			return h.versionConflict(ctx, st.current), nil
		case errors.Is(err, pipeline.ErrVersionConflict):
			// A writer committed between our read and our write.
			current, rerr := h.currentVersion(ctx, p.TenantID, id)
			if errors.Is(rerr, pipeline.ErrNotFound) {
				return h.notFound(ctx), nil
			}
			if rerr != nil {
				return nil, rerr
			}
			return h.versionConflict(ctx, current), nil
		}
		return nil, h.recordError(ctx, err, tried, stateProp)
	}
	return api.PatchRecord200JSONResponse{
		Body:    recordBody(rec, data),
		Headers: api.PatchRecord200ResponseHeaders{ETag: etag(rec.Version)},
	}, nil
}

// currentVersion reads a record's version in a transaction of its own.
func (h *handlers) currentVersion(ctx context.Context, tenant, id uuid.UUID) (int64, error) {
	var v int64
	err := tenancy.InTenantTx(ctx, h.pool, tenant, func(tx pgx.Tx) error {
		r, err := pipeline.Get(ctx, tx, tenant, id)
		if err != nil {
			return err
		}
		v = r.Version
		return nil
	})
	return v, err
}

// parsePatchBody reads `{data, state}` into the merge patch the pipeline
// takes. A top-level state is the state property's value and wins over the
// same property inside data.
func parsePatchBody(raw json.RawMessage, stateProp string) (map[string]any, *problemError) {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(raw, &env); err != nil || env == nil {
		return nil, recordInvalid(invalid("", "type"))
	}
	d, hasData := env["data"]
	s, hasState := env["state"]
	if !hasData && !hasState {
		return nil, recordInvalid(invalid("/data", "required"))
	}
	patch := map[string]any{}
	if hasData {
		var err error
		if patch, err = pipeline.DecodeObject(d); err != nil || patch == nil || !isObject(d) {
			return nil, recordInvalid(invalid("/data", "type"))
		}
	}
	if hasState {
		var state string
		if err := json.Unmarshal(s, &state); err != nil || strings.TrimSpace(string(s)) == "null" {
			return nil, recordInvalid(invalid("/state", "type"))
		}
		patch[stateProp] = state
	}
	return patch, nil
}

func isObject(b json.RawMessage) bool {
	t := strings.TrimSpace(string(b))
	return strings.HasPrefix(t, "{")
}

// versionConflictResponse is the 412: the problem with the current ETag in
// the header and in detail, and no record data.
type versionConflictResponse struct {
	body problem.Body
	etag string
}

func (h *handlers) versionConflict(ctx context.Context, current int64) *versionConflictResponse {
	b := h.reg.Body(ctx, problem.VersionConflict)
	b.Detail = fmt.Sprintf("%s Current ETag: %s.", b.Detail, etag(current))
	return &versionConflictResponse{body: b, etag: etag(current)}
}

func (r *versionConflictResponse) VisitPatchRecordResponse(w http.ResponseWriter) error {
	out, err := json.Marshal(r.body)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", problem.ContentType)
	w.Header().Set("ETag", r.etag)
	w.WriteHeader(r.body.Status)
	_, err = w.Write(out)
	return err
}

var _ api.PatchRecordResponseObject = (*versionConflictResponse)(nil)

// GetRecordHistory serves GET /api/v1/records/{entity}/{id}/history.
func (h *handlers) GetRecordHistory(ctx context.Context, req api.GetRecordHistoryRequestObject) (api.GetRecordHistoryResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil || h.cursors == nil {
		return nil, notImplemented()
	}
	id, perr := uuid.Parse(req.Id)
	if impossibleKey(req.Entity) || perr != nil {
		return h.notFound(ctx), nil
	}
	limit := defaultListLimit
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	// The record is part of the query, so a cursor of one record's history
	// does not decode on another's.
	claims, err := cursor.ClaimsFor(ctx, getRecordHistoryRoute, url.Values{"entity": {req.Entity}, "id": {id.String()}})
	if err != nil {
		return nil, err
	}
	var (
		afterAt time.Time
		afterID uuid.UUID
	)
	if req.Params.Cursor != nil {
		pos, err := h.cursors.Decode(claims, *req.Params.Cursor)
		if err != nil {
			return nil, &problemError{problem.InvalidCursor, nil, err}
		}
		if afterAt, afterID, err = decodeHistoryPos(pos); err != nil {
			return nil, &problemError{problem.InvalidCursor, nil, err}
		}
	}
	var rows []pipeline.Change
	err = tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if _, _, err := h.store.Current(ctx, tx, req.Entity); err != nil {
			return err
		}
		rec, err := pipeline.Get(ctx, tx, p.TenantID, id)
		if err != nil {
			return err
		}
		if rec.EntityKey != req.Entity {
			return pipeline.ErrNotFound
		}
		rows, err = pipeline.History(ctx, tx, p.TenantID, id, afterAt, afterID, limit+1)
		return err
	})
	if errors.Is(err, pipeline.ErrNotFound) || errors.Is(err, entitytypes.ErrNotFound) {
		return h.notFound(ctx), nil
	}
	if err != nil {
		return nil, err
	}
	out := api.RecordHistory{Items: make([]api.RecordChange, 0, min(len(rows), limit))}
	for _, c := range rows[:min(len(rows), limit)] {
		after := c.After
		if after == nil {
			after = map[string]any{}
		}
		item := api.RecordChange{
			At: c.At.UTC(), Source: c.Source, Action: api.RecordChangeAction(c.Action),
			ActorId: c.ActorID, Changed: ChangedPaths(c.Before, c.After), After: after,
		}
		if c.Before != nil {
			item.Before = &c.Before
		}
		out.Items = append(out.Items, item)
	}
	if len(rows) > limit {
		last := rows[limit-1]
		next, err := h.cursors.Encode(claims, encodeHistoryPos(last.At, last.ID))
		if err != nil {
			return nil, err
		}
		out.NextCursor = &next
	}
	return api.GetRecordHistory200JSONResponse(out), nil
}

// ChangedPaths lists, sorted, the JSON pointers under /data of the properties
// that differ between before and after. Objects are compared key by key;
// arrays and scalars count as one value (merge-patch semantics, RFC 7396).
func ChangedPaths(before, after map[string]any) []string {
	out := []string{}
	diff("/data", before, after, &out)
	slices.Sort(out)
	return out
}

func diff(prefix string, before, after map[string]any, out *[]string) {
	keys := map[string]struct{}{}
	for k := range before {
		keys[k] = struct{}{}
	}
	for k := range after {
		keys[k] = struct{}{}
	}
	for k := range maps.Keys(keys) {
		b, inB := before[k]
		a, inA := after[k]
		path := prefix + "/" + escapePointer(k)
		bm, bIsObj := b.(map[string]any)
		am, aIsObj := a.(map[string]any)
		switch {
		case inB && inA && bIsObj && aIsObj:
			diff(path, bm, am, out)
		case inB != inA || !reflect.DeepEqual(b, a):
			*out = append(*out, path)
		}
	}
}

// The cursor position is the last row's (at, id): at in microseconds, which
// is all PostgreSQL keeps.
func encodeHistoryPos(at time.Time, id uuid.UUID) []byte {
	return []byte(strconv.FormatInt(at.UnixMicro(), 10) + ":" + id.String())
}

func decodeHistoryPos(b []byte) (time.Time, uuid.UUID, error) {
	bad := errors.New("history cursor: bad position")
	us, rest, ok := strings.Cut(string(b), ":")
	if !ok {
		return time.Time{}, uuid.Nil, bad
	}
	n, err := strconv.ParseInt(us, 10, 64)
	if err != nil {
		return time.Time{}, uuid.Nil, bad
	}
	id, err := uuid.Parse(rest)
	if err != nil || id == uuid.Nil {
		return time.Time{}, uuid.Nil, bad
	}
	return time.UnixMicro(n).UTC(), id, nil
}
