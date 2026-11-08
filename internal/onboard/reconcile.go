package onboard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// ErrReconcileNotActive means the tenant is unknown or not active. The upgrade
// reconcile (TICK-567) only touches tenants activated before S14; a
// provisioning tenant gets its auditors group from onboarding itself.
var ErrReconcileNotActive = errors.New("onboard: tenant is not active")

// ReconcileStore is the part of PGStore the upgrade reconcile uses. It adds
// no method to Store.
type ReconcileStore interface {
	Lock(ctx context.Context, tenantID uuid.UUID) (release func(), err error)
	// TenantState is the tenant's state, found false when nava_onboard sees no
	// row (unknown tenant).
	TenantState(ctx context.Context, tenantID uuid.UUID) (state string, found bool, err error)
	// HasAuditorGroup reports whether the tenant has its tenant_auditor_group row.
	HasAuditorGroup(ctx context.Context, tenantID uuid.UUID) (bool, error)
	RecordAuditorGroup(ctx context.Context, tenantID uuid.UUID, kcGroupID string) error
}

// AuditorDirectory is the part of Directory the reconcile uses.
type AuditorDirectory interface {
	EnsureAuditorGroup(ctx context.Context, tenantID uuid.UUID) (keycloakGroupID string, err error)
}

var _ ReconcileStore = (*PGStore)(nil)

// TenantState implements ReconcileStore, as nava_onboard in the tenant's own
// transaction.
func (s *PGStore) TenantState(ctx context.Context, tenantID uuid.UUID) (string, bool, error) {
	var state string
	found := true
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, "SELECT state FROM tenants WHERE id = $1", tenantID).Scan(&state)
		if errors.Is(err, pgx.ErrNoRows) {
			found = false
			return nil
		}
		return err
	})
	if err != nil {
		return "", false, fmt.Errorf("onboard: tenant state: %w", err)
	}
	return state, found, nil
}

// HasAuditorGroup implements ReconcileStore.
func (s *PGStore) HasAuditorGroup(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	var has bool
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM tenant_auditor_group WHERE tenant_id = $1)", tenantID).Scan(&has)
	})
	if err != nil {
		return false, fmt.Errorf("onboard: auditor group row: %w", err)
	}
	return has, nil
}

// ReconcileAuditorGroup gives one active tenant its auditors group, as
// onboarding gives it to a new tenant: the Keycloak group, the groups row and
// the tenant_auditor_group row (D-5-4). Idempotent: a tenant that already has
// the row is left alone, with no Keycloak call. It adds no member. Every
// statement runs in tenantID's own context, and only tenantID is touched.
// It returns whether it created anything.
func ReconcileAuditorGroup(ctx context.Context, store ReconcileStore, dir AuditorDirectory, tenantID uuid.UUID) (created bool, err error) {
	release, err := store.Lock(ctx, tenantID)
	if err != nil {
		return false, err
	}
	defer release()
	state, found, err := store.TenantState(ctx, tenantID)
	if err != nil {
		return false, err
	}
	if !found || state != "active" {
		return false, ErrReconcileNotActive
	}
	has, err := store.HasAuditorGroup(ctx, tenantID)
	if err != nil || has {
		return false, err
	}
	kcID, err := dir.EnsureAuditorGroup(ctx, tenantID)
	if err != nil {
		return false, fmt.Errorf("onboard: auditor group: %w", err)
	}
	if err := store.RecordAuditorGroup(ctx, tenantID, kcID); err != nil {
		return false, err
	}
	return true, nil
}

// ReconcileSummary counts the outcome of ReconcileAuditorGroups.
type ReconcileSummary struct{ Created, Unchanged, Failed int }

// ReconcileAuditorGroups reconciles each tenant on its own, in order. One
// tenant's failure is logged and counted and does not stop the others; the
// returned error is non-nil when any tenant failed, so the job exits non-zero.
func ReconcileAuditorGroups(ctx context.Context, store ReconcileStore, dir AuditorDirectory, log *slog.Logger, tenants []uuid.UUID) (ReconcileSummary, error) {
	var sum ReconcileSummary
	for _, id := range tenants {
		created, err := ReconcileAuditorGroup(ctx, store, dir, id)
		switch {
		case err != nil:
			sum.Failed++
			log.ErrorContext(ctx, "auditors group reconcile failed",
				slog.String("tenant_id", id.String()), slog.Any("error", err))
		case created:
			sum.Created++
			log.InfoContext(ctx, "auditors group created", slog.String("tenant_id", id.String()))
		default:
			sum.Unchanged++
		}
	}
	if sum.Failed > 0 {
		return sum, fmt.Errorf("onboard: auditors group reconcile failed for %d of %d tenants", sum.Failed, len(tenants))
	}
	return sum, nil
}
