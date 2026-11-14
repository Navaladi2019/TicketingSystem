package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/numberformats"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Number-format endpoints (TICK-672, P6-07; ADR-005 §3.2, §4.2): create a
// format, publish its next version, list and read. Writes are tenant admin
// only (authz.RequireTenantAdminTx, first, inside the one transaction); the
// tenant is the token's and nothing in a body or path changes it (I2).

const listNumberFormatsRoute = "listNumberFormats"

// numberFormatBody is the JSON of a NumberFormatVersion. The document and
// the compiled form are the stored jsonb, written as is.
type numberFormatBody struct {
	ID             uuid.UUID       `json:"id"`
	Key            string          `json:"key"`
	Version        int             `json:"version"`
	CurrentVersion int             `json:"current_version"`
	Document       json.RawMessage `json:"document"`
	Compiled       json.RawMessage `json:"compiled"`
	CompiledSHA256 string          `json:"compiled_sha256"`
	CreatedAt      time.Time       `json:"created_at"`
	CreatedBy      uuid.UUID       `json:"created_by"`
}

// numberFormatResponse is a 200 or 201 NumberFormatVersion with its ETag.
type numberFormatResponse struct {
	status int
	v      *numberformats.Version
	// etagVersion is the version the ETag names: the new version on a write,
	// the format's current one on a read.
	etagVersion int
}

func (r numberFormatResponse) write(w http.ResponseWriter) error {
	v := r.v
	b, err := json.Marshal(numberFormatBody{v.ID, v.Key, v.Version, v.Current, v.Document, v.Compiled, v.CompiledSHA256, v.CreatedAt.UTC(), v.CreatedBy})
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", etag(int64(r.etagVersion)))
	w.WriteHeader(r.status)
	_, err = w.Write(b)
	return err
}

func (r numberFormatResponse) VisitCreateNumberFormatResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r numberFormatResponse) VisitGetNumberFormatResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r numberFormatResponse) VisitPublishNumberFormatVersionResponse(w http.ResponseWriter) error {
	return r.write(w)
}

func (r *notFoundResponse) VisitGetNumberFormatResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r *notFoundResponse) VisitPublishNumberFormatVersionResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r *versionConflictResponse) VisitPublishNumberFormatVersionResponse(w http.ResponseWriter) error {
	return r.VisitPatchRecordResponse(w)
}

var (
	_ api.CreateNumberFormatResponseObject         = numberFormatResponse{}
	_ api.GetNumberFormatResponseObject            = numberFormatResponse{}
	_ api.PublishNumberFormatVersionResponseObject = numberFormatResponse{}
	_ api.GetNumberFormatResponseObject            = (*notFoundResponse)(nil)
	_ api.PublishNumberFormatVersionResponseObject = (*notFoundResponse)(nil)
	_ api.PublishNumberFormatVersionResponseObject = (*versionConflictResponse)(nil)
)

// numberFormatList is the JSON of a NumberFormatList.
type numberFormatList struct {
	Items      []numberFormatSummary `json:"items"`
	NextCursor *string               `json:"next_cursor,omitempty"`
}

type numberFormatSummary struct {
	ID             uuid.UUID `json:"id"`
	Key            string    `json:"key"`
	CurrentVersion int       `json:"current_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type numberFormatListResponse numberFormatList

func (r numberFormatListResponse) VisitListNumberFormatsResponse(w http.ResponseWriter) error {
	b, err := json.Marshal(numberFormatList(r))
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	return err
}

var _ api.ListNumberFormatsResponseObject = numberFormatListResponse{}

// ListNumberFormats serves GET /api/v1/number-formats.
func (h *handlers) ListNumberFormats(ctx context.Context, req api.ListNumberFormatsRequestObject) (api.ListNumberFormatsResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	if h.pool == nil || h.cursors == nil {
		return nil, notImplemented()
	}
	limit := defaultListLimit
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	claims, err := cursor.ClaimsFor(ctx, listNumberFormatsRoute, url.Values{})
	if err != nil {
		return nil, err
	}
	after := ""
	if req.Params.Cursor != nil {
		pos, err := h.cursors.Decode(claims, *req.Params.Cursor)
		if err != nil {
			return nil, &problemError{problem.InvalidCursor, nil, err}
		}
		after = string(pos)
	}
	var rows []numberformats.Summary
	if err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) (err error) {
		rows, err = h.nfStore.List(ctx, tx, after, limit+1)
		return err
	}); err != nil {
		return nil, err
	}
	out := numberFormatList{Items: make([]numberFormatSummary, 0, min(len(rows), limit))}
	for _, r := range rows[:min(len(rows), limit)] {
		out.Items = append(out.Items, numberFormatSummary{r.ID, r.Key, r.CurrentVersion, r.CreatedAt.UTC(), r.UpdatedAt.UTC()})
	}
	if len(rows) > limit {
		next, err := h.cursors.Encode(claims, []byte(rows[limit-1].Key))
		if err != nil {
			return nil, err
		}
		out.NextCursor = &next
	}
	return numberFormatListResponse(out), nil
}

// GetNumberFormat serves GET /api/v1/number-formats/{id}. An id that is not
// a UUID, an unknown id, another tenant's id and a missing version give the
// one 404.
func (h *handlers) GetNumberFormat(ctx context.Context, req api.GetNumberFormatRequestObject) (api.GetNumberFormatResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return h.notFound(ctx), nil
	}
	version := 0
	if req.Params.Version != nil {
		version = *req.Params.Version
	}
	var v *numberformats.Version
	err = tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) (err error) {
		v, err = h.nfStore.GetVersion(ctx, tx, id, version)
		return err
	})
	switch {
	case errors.Is(err, numberformats.ErrNotFound):
		return h.notFound(ctx), nil
	case err != nil:
		return nil, err
	}
	return numberFormatResponse{status: http.StatusOK, v: v, etagVersion: v.Current}, nil
}

// numberFormatInput is a parsed create or publish body.
type numberFormatInput struct {
	key      string
	document json.RawMessage
	compiled numbering.Compiled
}

// parseNumberFormat reads a body: {key, document} for a create, {document}
// for a publish. Other fields, a forged tenant_id among them, are ignored.
// The document is checked against the number-format JSON Schema and
// compiled; a refusal is a 422 naming a JSON pointer, with the position in
// the template when the compiler gives one. No input is echoed.
func parseNumberFormat(raw json.RawMessage, create bool) (numberFormatInput, *problemError) {
	var in numberFormatInput
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return in, formatProblem(problem.NumberFormatInvalid, "", "type")
	}
	if create {
		rawKey, present := fields["key"]
		var key string
		switch {
		case !present:
			return in, formatProblem(problem.NumberFormatInvalid, "/key", "required")
		case json.Unmarshal(rawKey, &key) != nil:
			return in, formatProblem(problem.NumberFormatInvalid, "/key", "type")
		case !keyPattern.MatchString(key):
			return in, formatProblem(problem.NumberFormatInvalid, "/key", "pattern")
		}
		in.key = key
	}
	doc, present := fields["document"]
	if !present {
		return in, formatProblem(problem.NumberFormatInvalid, "/document", "required")
	}
	c, err := numbering.ParseFormat(doc)
	var fe *numbering.FormatError
	switch {
	case errors.As(err, &fe):
		return in, formatError(fe)
	case err != nil:
		return in, &problemError{problem.InternalError, nil, err}
	}
	in.document, in.compiled = doc, c
	return in, nil
}

func formatProblem(code problem.Code, pointer, rule string) *problemError {
	return &problemError{code, []problem.FieldError{invalid(pointer, rule)}, errors.New("invalid number format")}
}

// formatError maps a compiler refusal to its problem. The pointer is into the
// request; the rule is position:<offset> when the compiler names a position
// in the template, else invalid. Reason text is never sent.
func formatError(fe *numbering.FormatError) *problemError {
	pointer := "/document"
	switch {
	case strings.HasPrefix(fe.Field, "/"):
		pointer += strings.TrimSuffix(fe.Field, "/") // a JSON pointer into the document
	case fe.Field != "":
		pointer += "/" + fe.Field
	}
	rule := "invalid"
	if fe.Position >= 0 {
		rule = fmt.Sprintf("position:%d", fe.Position)
	}
	code := problem.NumberFormatInvalid
	if fe.Code == numbering.CodePrefixRequired {
		code = problem.NumberFormatPrefixRequired
	}
	return formatProblem(code, pointer, rule)
}

// gate refuses a format that is not the default while M3 has not run. The
// default is epoch 0; every other format compiles to EpochUnassigned.
func (h *handlers) gate(ctx context.Context, tx pgx.Tx, c numbering.Compiled) error {
	if c.Epoch == 0 {
		return nil
	}
	return h.nfGate.Check(ctx, tx)
}

// CreateNumberFormat serves POST /api/v1/number-formats.
func (h *handlers) CreateNumberFormat(ctx context.Context, req api.CreateNumberFormatRequestObject) (api.CreateNumberFormatResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	var raw json.RawMessage
	if req.Body != nil {
		raw = *req.Body
	}
	var v *numberformats.Version
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		in, perr := parseNumberFormat(raw, true)
		if perr != nil {
			return perr
		}
		if err := h.gate(ctx, tx, in.compiled); err != nil {
			return err
		}
		var err error
		if v, err = h.nfStore.Create(ctx, tx, p.TenantID, numberformats.NewInput{Key: in.key, Document: in.document, Compiled: in.compiled, CreatedBy: p.PrincipalID}); err != nil {
			return err
		}
		// The key completes in this transaction, with the write (TICK-482).
		return idempotency.CompleteInTx(ctx, tx, numberFormatResponse{http.StatusCreated, v, v.Version}.VisitCreateNumberFormatResponse)
	})
	if err != nil {
		return nil, h.numberFormatError(err)
	}
	h.log.InfoContext(ctx, "number format created",
		slog.String("tenant_id", p.TenantID.String()), slog.String("format_id", v.ID.String()), slog.String("format_key", v.Key))
	return numberFormatResponse{http.StatusCreated, v, v.Version}, nil
}

// PublishNumberFormatVersion serves PUT /api/v1/number-formats/{id}. Check
// order: admin, format (404), If-Match missing (428), stale (412), body (422),
// gate (422).
func (h *handlers) PublishNumberFormatVersion(ctx context.Context, req api.PublishNumberFormatVersionRequestObject) (api.PublishNumberFormatVersionResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	var raw json.RawMessage
	if req.Body != nil {
		raw = *req.Body
	}
	var v *numberformats.Version
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		id, err := uuid.Parse(req.Id)
		if err != nil {
			return numberformats.ErrNotFound // cannot exist, and must not reach the database
		}
		have, err := h.nfStore.CurrentVersion(ctx, tx, id)
		if err != nil {
			return err
		}
		versions, ok := ifMatchVersions(req.Params.IfMatch)
		if !ok {
			return errIfMatchRequired
		}
		match := make([]int, len(versions))
		for i, n := range versions {
			match[i] = int(n)
		}
		if !contains(match, have) {
			return &numberformats.ConflictError{Current: have}
		}
		in, perr := parseNumberFormat(raw, false)
		if perr != nil {
			return perr
		}
		if err := h.gate(ctx, tx, in.compiled); err != nil {
			return err
		}
		if v, err = h.nfStore.Publish(ctx, tx, p.TenantID, numberformats.PublishInput{ID: id, Document: in.document, Compiled: in.compiled, CreatedBy: p.PrincipalID, ExpectedVersion: match}); err != nil {
			return err
		}
		return idempotency.CompleteInTx(ctx, tx, numberFormatResponse{http.StatusOK, v, v.Version}.VisitPublishNumberFormatVersionResponse)
	})
	var conflict *numberformats.ConflictError
	switch {
	case errors.Is(err, numberformats.ErrNotFound):
		return h.notFound(ctx), nil
	case errors.As(err, &conflict):
		return h.versionConflict(ctx, int64(conflict.Current)), nil
	case err != nil:
		return nil, h.numberFormatError(err)
	}
	h.log.InfoContext(ctx, "number format version published",
		slog.String("tenant_id", p.TenantID.String()), slog.String("format_id", v.ID.String()), slog.Int("version", v.Version))
	return numberFormatResponse{http.StatusOK, v, v.Version}, nil
}

// numberFormatError maps a write's error to its problem; an error it does
// not know is returned as is (500).
func (h *handlers) numberFormatError(err error) error {
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, errIfMatchRequired):
		return &problemError{problem.IfMatchRequired, nil, err}
	case errors.Is(err, numberformats.ErrKeyExists):
		return &problemError{problem.NumberFormatExists, nil, err}
	case errors.Is(err, numberformats.ErrTemplatesUnavailable):
		return &problemError{problem.NumberingTemplatesUnavailable, nil, err}
	}
	return err
}
