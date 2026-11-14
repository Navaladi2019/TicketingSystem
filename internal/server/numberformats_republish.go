package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/numberformats"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// "Re-publish linked types" (TICK-673, P6-08; ADR-005 §3.2, §3.7 C13): one
// independent definition publish per entity type that pins an older version
// of a format, each in its own transaction, each authorised for the caller on
// its own. There is no fan-out on a format publish; this route is the only
// one.

// TypePublishCheck decides whether p may publish the definition of the
// entity type key, inside the caller's InTenantTx. It returns
// authz.ErrForbidden to refuse. Definition publish is a tenant-admin right
// today, so the default is authz.RequireTenantAdminTx; the seam is where a
// per-type right plugs in (ADR-005 §3.7 C11 (5), C13).
type TypePublishCheck func(ctx context.Context, tx pgx.Tx, p identity.Principal, key string) error

func (h *handlers) canPublishType(ctx context.Context, tx pgx.Tx, p identity.Principal, key string) error {
	if h.typePublishCheck != nil {
		return h.typePublishCheck(ctx, tx, p, key)
	}
	return authz.RequireTenantAdminTx(ctx, tx, p)
}

// republishResult is one type's line of the report.
type republishResult struct {
	EntityType string               `json:"entity_type"`
	Status     string               `json:"status"`
	Version    *int                 `json:"version,omitempty"`
	Code       string               `json:"code,omitempty"`
	Errors     []problem.FieldError `json:"errors,omitempty"`
}

type republishReport struct {
	FormatID uuid.UUID         `json:"format_id"`
	Version  int               `json:"version"`
	Results  []republishResult `json:"results"`
}

type republishResponse republishReport

func (r republishResponse) VisitRepublishLinkedTypesResponse(w http.ResponseWriter) error {
	b, err := json.Marshal(republishReport(r))
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	return err
}

func (r *notFoundResponse) VisitRepublishLinkedTypesResponse(w http.ResponseWriter) error {
	return r.write(w)
}

var (
	_ api.RepublishLinkedTypesResponseObject = republishResponse{}
	_ api.RepublishLinkedTypesResponseObject = (*notFoundResponse)(nil)
)

// RepublishLinkedTypes serves POST /api/v1/number-formats/{id}/republish-linked.
// Check order: admin (403), format (404), body (422), target version (422).
func (h *handlers) RepublishLinkedTypes(ctx context.Context, req api.RepublishLinkedTypesRequestObject) (api.RepublishLinkedTypesResponseObject, error) {
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
	var (
		id     uuid.UUID
		target int
		keys   []string
	)
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) (err error) {
		if err := authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		if id, err = uuid.Parse(req.Id); err != nil {
			return numberformats.ErrNotFound // cannot exist, and must not reach the database
		}
		if _, err = h.nfStore.CurrentVersion(ctx, tx, id); err != nil {
			return err
		}
		var body struct {
			Version *int `json:"version"`
		}
		if json.Unmarshal(raw, &body) != nil || body.Version == nil || *body.Version < 1 {
			return formatProblem(problem.NumberFormatInvalid, "/version", "invalid")
		}
		target = *body.Version
		if _, err = h.nfStore.GetVersion(ctx, tx, id, target); errors.Is(err, numberformats.ErrNotFound) {
			// A new error, not err: the sentinel would read as the 404 below.
			return &problemError{problem.NumberFormatNotFound, []problem.FieldError{invalid("/version", "not_found")}, errors.New("number format version not found")}
		} else if err != nil {
			return err
		}
		keys, err = h.store.LinkedBelow(ctx, tx, id, target)
		return err
	})
	switch {
	case errors.Is(err, numberformats.ErrNotFound):
		return h.notFound(ctx), nil
	case err != nil:
		return nil, h.numberFormatError(err)
	}
	report := republishReport{FormatID: id, Version: target, Results: make([]republishResult, 0, len(keys))}
	for _, key := range keys {
		if r, ok := h.republishOne(ctx, p, key, id, target); ok {
			report.Results = append(report.Results, r)
		}
	}
	// Each type published in a transaction of its own, so no one transaction
	// covers the writes. The key completes, with the report, in a short closing
	// transaction. A run that dies before it can be repeated: a re-run finds only
	// the types still pinned below the target.
	if err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		return idempotency.CompleteInTx(ctx, tx, republishResponse(report).VisitRepublishLinkedTypesResponse)
	}); err != nil {
		return nil, err
	}
	h.log.InfoContext(ctx, "linked entity types re-published",
		slog.String("tenant_id", p.TenantID.String()), slog.String("format_id", id.String()),
		slog.Int("version", target), slog.Int("types", len(report.Results)))
	return republishResponse(report), nil
}

// republishOne publishes the type's current definition again, pinned to the
// target version of the format, in a transaction of its own. ok is false for
// a type that is no longer pinned below the target by the time its
// transaction runs (a concurrent publish moved it): it is not in the report.
func (h *handlers) republishOne(ctx context.Context, p identity.Principal, key string, format uuid.UUID, target int) (res republishResult, ok bool) {
	res = republishResult{EntityType: key}
	var published *entitytypes.Version
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := h.canPublishType(ctx, tx, p, key); err != nil {
			return err
		}
		cur, err := h.store.GetCurrent(ctx, tx, key)
		if err != nil {
			return err
		}
		if cur.NumberFormat == nil || cur.NumberFormat.ID != format || cur.NumberFormat.Version >= target {
			return errNoLongerLinked
		}
		published, err = h.store.Publish(ctx, tx, p.TenantID, entitytypes.PublishInput{
			Key: key, PublishedBy: p.PrincipalID,
			Definition:      metadata.Definition{Schema: cur.Schema, FixedColumns: cur.FixedColumns, States: cur.States},
			NumberFormat:    entitytypes.NumberFormatChange{Set: true, Ref: &entitytypes.FormatRef{ID: format, Version: target}},
			ExpectedVersion: []int{cur.Version}, // a concurrent publish of the type must not be overwritten
		})
		return err
	})
	var (
		overlap  *entitytypes.OverlapError
		conflict *entitytypes.ConflictError
	)
	switch {
	case err == nil:
		res.Status, res.Version = "published", &published.Version
		return res, true
	case errors.Is(err, errNoLongerLinked):
		return res, false
	case errors.Is(err, authz.ErrForbidden):
		res.Status, res.Code = "refused", string(problem.Forbidden)
	case errors.As(err, &overlap), errors.Is(err, entitytypes.ErrFormatNotFound):
		pe := formatLinkProblem(err)
		res.Status, res.Code, res.Errors = "refused", string(pe.code), pe.fields
	case errors.As(err, &conflict), errors.Is(err, entitytypes.ErrNotFound):
		res.Status, res.Code = "refused", string(problem.VersionConflict)
	default:
		// One type's failure does not hide the others' results. The detail
		// stays in the log, never in the report.
		h.log.ErrorContext(ctx, "re-publish of a linked entity type failed",
			slog.String("tenant_id", p.TenantID.String()), slog.String("entity_key", key), slog.Any("error", err))
		res.Status, res.Code = "refused", string(problem.InternalError)
	}
	return res, true
}

var errNoLongerLinked = errors.New("entity type is no longer linked below the target version")
