package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
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
		cur, _, err := h.store.Current(ctx, tx, req.Entity)
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
		// An update writes under the current version, so its mapping and
		// defaults are the current ones.
		data = cur.ApplyDefaults(rec.Payload(cur.FixedColumns()))
		return nil
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
	if h.pool == nil || h.cursors == nil || h.audit == nil {
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
	// Two transactions (ADR-004-A1 A1-7). First nava_app reads the record in
	// the caller's tenant, so another tenant's record, an unreadable one and
	// a missing one are the same 404 before any audit row is read (Security
	// review of PR #432). Only then does the reader's pool read its entries.
	// READ COMMITTED gives each statement its own snapshot, so the split adds
	// no race.
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
		return nil
	})
	var changes []read.Change
	if err == nil {
		var entries []read.Entry
		if entries, err = h.audit.RecordEntries(ctx, p.TenantID, id); err == nil {
			changes = historyPage(read.Replay(entries), afterAt, afterID, limit+1)
		}
	}
	if errors.Is(err, pipeline.ErrNotFound) || errors.Is(err, entitytypes.ErrNotFound) {
		return h.notFound(ctx), nil
	}
	var tooLong *read.HistoryTooLongError
	if errors.As(err, &tooLong) {
		// The numbers only: no record data goes to the log or the metric.
		h.log.WarnContext(ctx, "record history longer than the cap",
			slog.String("code", string(problem.HistoryTooLong)), slog.String("entity", req.Entity),
			slog.String("record_id", id.String()), slog.Int64("entries", tooLong.Entries), slog.Int64("cap", tooLong.Max))
		if h.historyTooLong != nil {
			h.historyTooLong.Add(ctx, 1)
		}
		return nil, &problemError{problem.HistoryTooLong, nil, err}
	}
	if errors.Is(err, read.ErrNoDefinitionVersions) {
		return nil, h.maskerFailed(ctx, err)
	}
	if err != nil {
		return nil, err
	}
	out := api.RecordHistory{Items: make([]api.RecordChange, 0, min(len(changes), limit))}
	for _, c := range changes[:min(len(changes), limit)] {
		after := c.After
		if after == nil {
			after = map[string]any{}
		}
		action := c.Entry.Action
		if action == "state_change" {
			action = string(api.Update) // a state change is an update of the record
		}
		item := api.RecordChange{
			At: c.Entry.OccurredAt.UTC(), Source: c.Entry.Source, Action: api.RecordChangeAction(action),
			ActorId: c.Entry.ActorPrincipalID, Changed: []string{}, After: after,
		}
		if c.Known {
			item.Changed = ChangedPaths(c.Before, c.After)
		}
		if c.Before != nil {
			item.Before = &c.Before
		}
		out.Items = append(out.Items, item)
	}
	if len(changes) > limit {
		last := changes[limit-1].Entry
		next, err := h.cursors.Encode(claims, encodeHistoryPos(last.OccurredAt, last.ID))
		if err != nil {
			return nil, err
		}
		out.NextCursor = &next
	}
	return api.GetRecordHistory200JSONResponse(out), nil
}

// historyPage returns up to n changes, newest first, older than the position
// (afterAt, afterID) when afterID is not uuid.Nil. changes is oldest first.
func historyPage(changes []read.Change, afterAt time.Time, afterID uuid.UUID, n int) []read.Change {
	var out []read.Change
	for i := len(changes) - 1; i >= 0 && len(out) < n; i-- {
		e := changes[i].Entry
		if afterID != uuid.Nil {
			older := e.OccurredAt.Before(afterAt) || (e.OccurredAt.Equal(afterAt) && e.ID.String() < afterID.String())
			if !older {
				continue
			}
		}
		out = append(out, changes[i])
	}
	return out
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
