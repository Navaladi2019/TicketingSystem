package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workflowdefs"
)

// Workflow definition endpoints (P3-41, TICK-710). Both are tenant-admin
// only, and the admin check is the first statement of the one transaction
// that reads or writes.

// PublishWorkflowDefinitionVersion serves POST /api/v1/workflow-definitions/{key}/versions.
func (h *handlers) PublishWorkflowDefinitionVersion(ctx context.Context, req api.PublishWorkflowDefinitionVersionRequestObject) (api.PublishWorkflowDefinitionVersionResponseObject, error) {
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
	var v *workflowdefs.Version
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) (err error) {
		if err = authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		v, err = h.workflows.Publish(ctx, tx, p.TenantID, workflowdefs.PublishInput{Key: req.Key, PublishedBy: p.PrincipalID, Body: raw})
		if err != nil {
			return err
		}
		// The key completes in this transaction, with the write (TICK-482).
		return idempotency.CompleteInTx(ctx, tx, (&workflowDefResponse{status: http.StatusCreated, v: v}).VisitPublishWorkflowDefinitionVersionResponse)
	})
	var inv *workflowdefs.InvalidError
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return nil, &problemError{problem.Forbidden, nil, err}
	case errors.As(err, &inv):
		fields := make([]problem.FieldError, len(inv.Fields))
		for i, f := range inv.Fields {
			fields[i] = problem.FieldError{Pointer: f.Pointer, Rule: f.Rule}
		}
		return nil, &problemError{problem.WorkflowDefinitionInvalid, fields, err}
	case err != nil:
		return nil, err
	}
	h.log.InfoContext(ctx, "workflow definition version published",
		slog.String("tenant_id", p.TenantID.String()), slog.String("definition_key", v.Key), slog.Int("version", v.Version))
	return &workflowDefResponse{status: http.StatusCreated, v: v}, nil
}

// GetWorkflowDefinition serves GET /api/v1/workflow-definitions/{key}. An
// unknown key and another tenant's key give the one 404.
func (h *handlers) GetWorkflowDefinition(ctx context.Context, req api.GetWorkflowDefinitionRequestObject) (api.GetWorkflowDefinitionResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	var v *workflowdefs.Version
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) (err error) {
		if err = authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		v, err = h.workflows.GetCurrent(ctx, tx, req.Key)
		return err
	})
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return nil, &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, workflowdefs.ErrNotFound):
		return h.notFound(ctx), nil
	case err != nil:
		return nil, err
	}
	return &workflowDefResponse{status: http.StatusOK, v: v}, nil
}

// workflowDefResponse writes a WorkflowDefinitionVersion with the body as the
// stored bytes: the generated types encode through encoding/json, which would
// compact a json.RawMessage.
type workflowDefResponse struct {
	status int
	v      *workflowdefs.Version
}

func (r *workflowDefResponse) write(w http.ResponseWriter) error {
	ids, err := json.Marshal(r.v.StepIDs)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	for _, f := range []struct {
		name string
		val  any
	}{
		{"key", r.v.Key}, {"entity", r.v.Entity}, {"version", r.v.Version}, {"current", r.v.Current},
		{"version_id", r.v.ID}, {"body", r.v.Body}, {"body_sha256", r.v.SHA256Hex()},
		{"step_ids", json.RawMessage(ids)}, {"published_at", r.v.PublishedAt.UTC()},
	} {
		if b.Len() == 0 {
			b.WriteByte('{')
		} else {
			b.WriteByte(',')
		}
		name, _ := json.Marshal(f.name) // a string cannot fail to marshal
		b.Write(name)
		b.WriteByte(':')
		if raw, ok := f.val.(json.RawMessage); ok {
			b.Write(raw) // verbatim
			continue
		}
		val, err := json.Marshal(f.val)
		if err != nil {
			return err
		}
		b.Write(val)
	}
	b.WriteByte('}')
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(r.status)
	_, err = b.WriteTo(w)
	return err
}

func (r *workflowDefResponse) VisitGetWorkflowDefinitionResponse(w http.ResponseWriter) error {
	return r.write(w)
}

func (r *workflowDefResponse) VisitPublishWorkflowDefinitionVersionResponse(w http.ResponseWriter) error {
	return r.write(w)
}

func (r *notFoundResponse) VisitGetWorkflowDefinitionResponse(w http.ResponseWriter) error {
	return r.write(w)
}

var (
	_ api.GetWorkflowDefinitionResponseObject            = (*workflowDefResponse)(nil)
	_ api.PublishWorkflowDefinitionVersionResponseObject = (*workflowDefResponse)(nil)
	_ api.GetWorkflowDefinitionResponseObject            = (*notFoundResponse)(nil)
)
