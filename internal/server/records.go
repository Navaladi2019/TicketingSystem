package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Record endpoints (TICK-357, P3-14). Check order on every records route:
// authn, entity type (unknown is the one 404), record, authorizer, body.

// pgForeignKeyViolation is SQLSTATE 23503.
const pgForeignKeyViolation = "23503"

// refRule is the rule of a field error for a reference that does not resolve
// in the tenant. It is the same for an unknown id and another tenant's id.
const refRule = "reference"

// groupRemovedRule is the rule of the field error of a 422
// assignment_group_removed (ADR-004-E5 D6).
const groupRemovedRule = "removed"

// fkColumn maps the records foreign keys a payload can name to the fixed
// column they guard. records_created_by_fkey is the caller, not payload.
func fkColumn(constraint string) (string, bool) {
	switch constraint {
	case "records_requester_fkey":
		return metadata.ColumnRequesterID, true
	case "records_assignee_fkey":
		return metadata.ColumnAssigneeID, true
	case "records_assignment_group_fkey":
		return metadata.ColumnAssignmentGroupID, true
	}
	return "", false
}

func etag(version int64) string { return `W/"` + strconv.FormatInt(version, 10) + `"` }

// impossibleKey reports an entity key that cannot exist; it must not reach
// the database.
func impossibleKey(key string) bool {
	return len(key) > maxKeyBytes || !utf8.ValidString(key) || strings.IndexByte(key, 0) >= 0
}

// CreateRecord serves POST /api/v1/records/{entity}.
func (h *handlers) CreateRecord(ctx context.Context, req api.CreateRecordRequestObject) (api.CreateRecordResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil || h.pipe == nil {
		return nil, notImplemented()
	}
	if impossibleKey(req.Entity) {
		return h.notFound(ctx), nil
	}
	var raw json.RawMessage
	if req.Body != nil {
		raw = *req.Body
	}
	var (
		rec   *pipeline.Record
		fixed map[string]string // the mapping the row was written under
		resp  api.CreateRecord201JSONResponse
	)
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		def, curVersion, err := h.store.Current(ctx, tx, req.Entity)
		if err != nil {
			return err
		}
		// The authorizer runs inside the pipeline, after this envelope check
		// and before the schema check. S9's authorizer reads neither.
		data, perr := parseRecordBody(raw)
		if perr != nil {
			return perr
		}
		res, err := h.pipe.Write(ctx, tx, pipeline.Command{
			// Causation stays empty on purpose: origin and depth never come from
			// the request (TICK-88 security review).
			Op: pipeline.OpCreate, EntityKey: req.Entity, Data: data, Actor: p, Source: pipeline.SourceUser,
		})
		if err != nil {
			return err
		}
		if rec, err = pipeline.Get(ctx, tx, p.TenantID, res.RecordID); err != nil {
			return err
		}
		// The response is built from the version the row was written under,
		// which the pipeline read itself.
		fixed, err = h.fixedColumnsOf(ctx, tx, req.Entity, rec.SchemaVersion, curVersion, def)
		if err != nil {
			return err
		}
		resp = api.CreateRecord201JSONResponse{
			Body:    recordBody(rec, rec.Payload(fixed)),
			Headers: api.CreateRecord201ResponseHeaders{ETag: etag(rec.Version), Location: "/api/v1/records/" + rec.EntityKey + "/" + rec.ID.String()},
		}
		// The key completes in this transaction, with the write (TICK-482).
		return idempotency.CompleteInTx(ctx, tx, resp.VisitCreateRecordResponse)
	})
	if err != nil {
		if errors.Is(err, entitytypes.ErrNotFound) || errors.Is(err, pipeline.ErrNotFound) {
			return h.notFound(ctx), nil
		}
		// A refused write carries the version the pipeline validated against
		// (*pipeline.ErrAtVersion); recordError maps under that one.
		return nil, h.recordError(ctx, err, nil, "")
	}
	return resp, nil
}

// GetRecord serves GET /api/v1/records/{entity}/{id}.
func (h *handlers) GetRecord(ctx context.Context, req api.GetRecordRequestObject) (api.GetRecordResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	id, perr := uuid.Parse(req.Id)
	if impossibleKey(req.Entity) || perr != nil {
		return h.notFound(ctx), nil
	}
	var (
		rec  *pipeline.Record
		data map[string]any
	)
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		cur, curVersion, err := h.store.Current(ctx, tx, req.Entity)
		if err != nil {
			return err
		}
		if rec, err = pipeline.Get(ctx, tx, p.TenantID, id); err != nil {
			return err
		}
		if rec.EntityKey != req.Entity {
			return pipeline.ErrNotFound
		}
		// The row was split under its own version's mapping; defaults come
		// from the current version for what an older version never wrote.
		fixed, err := h.fixedColumnsOf(ctx, tx, req.Entity, rec.SchemaVersion, curVersion, cur)
		if err != nil {
			return err
		}
		data = cur.ApplyDefaults(rec.Payload(fixed))
		return nil
	})
	if errors.Is(err, pipeline.ErrNotFound) || errors.Is(err, entitytypes.ErrNotFound) {
		return h.notFound(ctx), nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetRecord200JSONResponse{Body: recordBody(rec, data), Headers: api.GetRecord200ResponseHeaders{ETag: etag(rec.Version)}}, nil
}

// fixedColumnsOf is the fixed-column mapping of the version a row was written
// under. It reads that version only when it is not the current one.
func (h *handlers) fixedColumnsOf(ctx context.Context, tx pgx.Tx, entity string, version, curVersion int, cur *metadata.Compiled) (map[string]string, error) {
	return fixedColumnsUnder(ctx, tx, h.store, entity, version, curVersion, cur)
}

func fixedColumnsUnder(ctx context.Context, tx pgx.Tx, defs entitytypes.DefinitionReader, entity string, version, curVersion int, cur *metadata.Compiled) (map[string]string, error) {
	if version == curVersion {
		return cur.FixedColumns(), nil
	}
	stored, err := defs.Version(ctx, tx, entity, version)
	if err != nil {
		return nil, fmt.Errorf("read stored definition: %w", err)
	}
	return stored.FixedColumns(), nil
}

func recordBody(r *pipeline.Record, data map[string]any) api.Record {
	body := api.Record{
		Id: r.ID, Number: r.Number, Version: r.Version, SchemaVersion: r.SchemaVersion, State: r.State,
		Data: data, CreatedAt: r.CreatedAt.UTC(), UpdatedAt: r.UpdatedAt.UTC(),
	}
	// D7: a record keeps its group after the group is tombstoned, and the
	// read says so.
	if r.AssignmentGroupID != nil {
		body.AssignmentGroup = &api.RecordAssignmentGroup{Id: *r.AssignmentGroupID, Removed: r.AssignmentGroupRemoved}
	}
	return body
}

// parseRecordBody reads `{data: {...}}`. Numbers stay exact.
func parseRecordBody(raw json.RawMessage) (map[string]any, *problemError) {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(raw, &env); err != nil || env == nil {
		return nil, recordInvalid(invalid("", "type"))
	}
	d, present := env["data"]
	if !present {
		return nil, recordInvalid(invalid("/data", "required"))
	}
	data, err := pipeline.DecodeObject(d)
	if string(bytes.TrimSpace(d)) == "null" {
		err = errors.New("data is null")
	}
	if err != nil || data == nil {
		return nil, recordInvalid(invalid("/data", "type"))
	}
	return data, nil
}

func recordInvalid(fields ...problem.FieldError) *problemError {
	return &problemError{problem.RecordInvalid, fields, errors.New("invalid record")}
}

// recordError maps a failed create or update transaction. A foreign key
// violation never shows its constraint, table or detail text: it is a field
// error naming the payload property whose fixed column the constraint guards.
// The constraint name is logged here, server side only.
//
// Pointers are built from the definition version the pipeline wrote under
// (*pipeline.ErrAtVersion), so a version published between the handler's own
// read and the pipeline's cannot move them (TICK-413). fixed and stateProp,
// from the handler's read, are used only for an error without that version.
func (h *handlers) recordError(ctx context.Context, err error, fixed map[string]string, stateProp string) error {
	if av := (*pipeline.ErrAtVersion)(nil); errors.As(err, &av) {
		fixed, stateProp = av.FixedColumns, av.StateProperty
	}
	var (
		ve *pipeline.ErrValidation
		it *pipeline.ErrIllegalTransition
		uv *pipeline.ErrUniqueViolation
		pe *problemError
		pg *pgconn.PgError
	)
	switch {
	case errors.As(err, &pe):
		return pe
	case errors.Is(err, authz.ErrForbidden):
		return &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, pipeline.ErrAssignmentGroupRemoved):
		// One body for a tombstoned group, another tenant's group and an
		// unknown id, on create and on update (ADR-004-E5 D6, N4).
		return &problemError{problem.AssignmentGroupRemoved, columnFields(fixed, metadata.ColumnAssignmentGroupID, groupRemovedRule), err}
	case errors.As(err, &it):
		return &problemError{problem.IllegalTransition, []problem.FieldError{invalid("/data/"+escapePointer(stateProp), "transition")}, err}
	case errors.As(err, &uv):
		return &problemError{problem.UniqueViolation, []problem.FieldError{invalid("/data/"+escapePointer(uv.Field), "unique")}, err}
	case errors.As(err, &ve):
		fields := make([]problem.FieldError, len(ve.Fields))
		for i, f := range ve.Fields {
			fields[i] = invalid("/data"+f.Path, f.Code)
		}
		return recordInvalid(fields...)
	case errors.As(err, &pg) && pg.Code == pgForeignKeyViolation:
		h.log.WarnContext(ctx, "record write refused by a foreign key", slog.String("constraint", pg.ConstraintName))
		col, ok := fkColumn(pg.ConstraintName)
		if !ok {
			return err
		}
		fields := columnFields(fixed, col, refRule)
		if len(fields) == 0 {
			return err
		}
		pr := recordInvalid(fields...)
		pr.err = errors.New("record references an unknown principal or group")
		return pr
	}
	return err
}

// columnFields is one field error, with rule, for each payload property
// mapped to the fixed column col, sorted by property.
func columnFields(fixed map[string]string, col, rule string) []problem.FieldError {
	var props []string
	for prop, c := range fixed {
		if c == col {
			props = append(props, prop)
		}
	}
	slices.Sort(props)
	fields := make([]problem.FieldError, len(props))
	for i, prop := range props {
		fields[i] = invalid("/data/"+escapePointer(prop), rule)
	}
	return fields
}

// escapePointer escapes one JSON pointer token (RFC 6901).
func escapePointer(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}

func (r *notFoundResponse) VisitPatchRecordResponse(w http.ResponseWriter) error { return r.write(w) }
func (r *notFoundResponse) VisitGetRecordHistoryResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r *notFoundResponse) VisitCreateRecordResponse(w http.ResponseWriter) error { return r.write(w) }
func (r *notFoundResponse) VisitGetRecordResponse(w http.ResponseWriter) error    { return r.write(w) }

var (
	_ api.CreateRecordResponseObject     = (*notFoundResponse)(nil)
	_ api.GetRecordResponseObject        = (*notFoundResponse)(nil)
	_ api.PatchRecordResponseObject      = (*notFoundResponse)(nil)
	_ api.GetRecordHistoryResponseObject = (*notFoundResponse)(nil)
)
