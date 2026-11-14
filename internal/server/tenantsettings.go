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
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/tenantsettings"
)

// Tenant settings endpoints (TICK-672, P6-07; ADR-005 §3.6): read and set the
// tenant's numbering time zone. Both are tenant admin only
// (authz.RequireTenantAdminTx, first, inside the one transaction). The tenant
// is the token's: no path, query or body field names another one (I2).

// timeZoneResponse is a 200 TenantTimeZone.
type timeZoneResponse struct{ zone string }

func (r timeZoneResponse) write(w http.ResponseWriter) error {
	b, err := json.Marshal(struct {
		TimeZone string `json:"time_zone"`
	}{r.zone})
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	return err
}

func (r timeZoneResponse) VisitGetTenantTimeZoneResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r timeZoneResponse) VisitPutTenantTimeZoneResponse(w http.ResponseWriter) error {
	return r.write(w)
}

var (
	_ api.GetTenantTimeZoneResponseObject = timeZoneResponse{}
	_ api.PutTenantTimeZoneResponseObject = timeZoneResponse{}
)

// GetTenantTimeZone serves GET /api/v1/tenant/settings/time-zone.
func (h *handlers) GetTenantTimeZone(ctx context.Context, _ api.GetTenantTimeZoneRequestObject) (api.GetTenantTimeZoneResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	var zone string
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) (err error) {
		if err = authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		zone, err = h.tsStore.TimeZone(ctx, tx, p.TenantID)
		return err
	})
	if err != nil {
		return nil, tenantSettingsError(err)
	}
	return timeZoneResponse{zone}, nil
}

// PutTenantTimeZone serves PUT /api/v1/tenant/settings/time-zone.
func (h *handlers) PutTenantTimeZone(ctx context.Context, req api.PutTenantTimeZoneRequestObject) (api.PutTenantTimeZoneResponseObject, error) {
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
	var zone string
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		name, perr := parseTimeZone(raw)
		if perr != nil {
			return perr
		}
		var err error
		if zone, err = h.tsStore.SetTimeZone(ctx, tx, p.TenantID, name, p.PrincipalID); err != nil {
			return err
		}
		// The key completes in this transaction, with the write (TICK-482).
		return idempotency.CompleteInTx(ctx, tx, timeZoneResponse{zone}.VisitPutTenantTimeZoneResponse)
	})
	if err != nil {
		return nil, tenantSettingsError(err)
	}
	h.log.InfoContext(ctx, "tenant time zone set", slog.String("tenant_id", p.TenantID.String()))
	return timeZoneResponse{zone}, nil
}

// parseTimeZone reads {time_zone}. Other fields are ignored. A refusal is a
// 422 naming /time_zone; the value is never echoed.
func parseTimeZone(raw json.RawMessage) (string, *problemError) {
	bad := func(rule string) *problemError {
		return &problemError{problem.InvalidTimeZone, []problem.FieldError{invalid("/time_zone", rule)}, errors.New("invalid time zone")}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return "", &problemError{problem.InvalidTimeZone, []problem.FieldError{invalid("", "type")}, errors.New("invalid time zone")}
	}
	rawName, present := fields["time_zone"]
	if !present {
		return "", bad("required")
	}
	var name string
	if json.Unmarshal(rawName, &name) != nil {
		return "", bad("type")
	}
	if tenantsettings.CheckTimeZone(name) != nil {
		return "", bad("invalid")
	}
	return name, nil
}

// tenantSettingsError maps a settings error to its problem; an error it does
// not know is returned as is (500).
func tenantSettingsError(err error) error {
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, tenantsettings.ErrNotFound):
		return &problemError{problem.InternalError, nil, err} // the caller's own tenant row is missing
	}
	return err
}
