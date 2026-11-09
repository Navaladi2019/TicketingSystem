package onboard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The default incident ACL (PLAN-003 S10, P3-35). Record authorization fails
// closed, so a tenant whose incident type has no published ACL version can
// read and write nothing. Onboarding publishes version 1 of this default for
// the seeded `incident` template; `nava-onboard reconcile-default-acl` gives
// the same version to a tenant that was onboarded or seeded before it
// existed. Both go through authz.PolicyStore.PublishIfNone, so a type that
// already has a version is never touched and a re-run changes nothing.

// IncidentKey is the entity type the default ACL is published for.
const IncidentKey = "incident"

// ServiceDeskGroupName is the name of the group the default ACL gives the
// service desk's access to, a child of the tenant's group tree.
const ServiceDeskGroupName = "service-desk"

// The ids of the default rules.
const (
	RuleAdmins      = "default-admins"
	RuleServiceDesk = "default-service-desk"
	RuleRequester   = "default-requester"
	// RuleRequesterNew lets a requester create and edit its own incident while
	// it is new (A3): without it the requester could read but never raise one.
	RuleRequesterNew = "default-requester-new"
)

// DefaultIncidentRules is the default incident ACL v1: the tenant admins read
// and write everything, the service-desk group reads and writes the incidents
// assigned to one of its own groups, a requester reads the incidents it raised
// and writes them while their state is new. An unassigned incident is
// triaged by the admins group (M1 has no null condition). There are no masks.
// A tenant with no service-desk group (serviceDesk
// empty) gets no service-desk rule: a rule that names no group matches no one.
func DefaultIncidentRules(admins uuid.UUID, serviceDesk []uuid.UUID) []authz.Rule {
	rules := []authz.Rule{{
		ID: RuleAdmins, Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: []uuid.UUID{admins}},
	}}
	if len(serviceDesk) > 0 {
		rules = append(rules, authz.Rule{
			ID: RuleServiceDesk, Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: serviceDesk},
			Where: []authz.Cond{{Column: metadata.ColumnAssignmentGroupID, Op: authz.CondIn, Caller: authz.CallerMyGroups}},
		})
	}
	return append(rules,
		authz.Rule{ID: RuleRequester, Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true}},
		authz.Rule{ID: RuleRequesterNew, Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Requester: true},
			Where: []authz.Cond{{Column: metadata.ColumnState, Op: authz.CondEq, Values: []string{"new"}}}})
}

// ACLPublisher publishes the default incident ACL of one tenant.
type ACLPublisher interface {
	// PublishDefaultIncident publishes version 1 when the tenant's incident
	// type has none, and reports whether it did. Idempotent.
	PublishDefaultIncident(ctx context.Context, tenantID uuid.UUID) (published bool, err error)
}

// ACLStore is the PostgreSQL ACLPublisher. Its pool logs in as nava_app, the
// role that writes acl_versions (db/audited-tables.yaml), and every statement
// runs in the tenant's own transaction, so row-level security scopes it.
type ACLStore struct{ Pool *pgxpool.Pool }

var _ ACLPublisher = ACLStore{}

const (
	adminGroupSQL = `SELECT group_id FROM tenant_admin_group WHERE tenant_id = $1`
	// A tombstoned group (deleted_at set) is not a group of the tenant any more.
	serviceDeskSQL = `SELECT id FROM groups WHERE tenant_id = $1 AND path = $2 AND deleted_at IS NULL`
)

// PublishDefaultIncident implements ACLPublisher. The admins group is read
// from tenant_admin_group (the group onboarding recorded, whichever id group
// sync gave it), the service-desk group from groups by its path. The version
// and its audit entry are written in one transaction.
func (s ACLStore) PublishDefaultIncident(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	var published bool
	err := tenancy.InTenantTx(ctx, s.Pool, tenantID, func(tx pgx.Tx) error {
		var admins uuid.UUID
		if err := tx.QueryRow(ctx, adminGroupSQL, tenantID).Scan(&admins); errors.Is(err, pgx.ErrNoRows) {
			return ErrNoTenantAdmin
		} else if err != nil {
			return err
		}
		var serviceDesk []uuid.UUID
		var id uuid.UUID
		err := tx.QueryRow(ctx, serviceDeskSQL, tenantID, identity.TenantGroupPrefix(tenantID)+ServiceDeskGroupName).Scan(&id)
		switch {
		case err == nil:
			serviceDesk = []uuid.UUID{id}
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		var err2 error
		published, err2 = authz.PolicyStore{}.PublishIfNone(ctx, tx, IncidentKey,
			authz.Publisher{System: "onboarding", Source: audit.SourceOnboarding},
			DefaultIncidentRules(admins, serviceDesk), nil)
		return err2
	})
	if err != nil {
		if errors.Is(err, ErrNoTenantAdmin) {
			return false, err
		}
		return false, fmt.Errorf("onboard: default acl: %w", err)
	}
	return published, nil
}

// ACLReconcileSummary counts the outcome of ReconcileDefaultACLs.
type ACLReconcileSummary struct{ Published, Unchanged, Failed int }

// ReconcileDefaultACLs gives each tenant's incident type the default ACL when
// it has none, one tenant at a time. One tenant's failure is logged and
// counted and does not stop the others; the returned error is non-nil when any
// failed, so the job exits non-zero.
func ReconcileDefaultACLs(ctx context.Context, pub ACLPublisher, log *slog.Logger, tenants []uuid.UUID) (ACLReconcileSummary, error) {
	var sum ACLReconcileSummary
	for _, id := range tenants {
		published, err := pub.PublishDefaultIncident(ctx, id)
		switch {
		case err != nil:
			sum.Failed++
			log.ErrorContext(ctx, "default acl reconcile failed",
				slog.String("tenant_id", id.String()), slog.Any("error", err))
		case published:
			sum.Published++
			log.InfoContext(ctx, "default acl published", slog.String("tenant_id", id.String()))
		default:
			sum.Unchanged++
		}
	}
	if sum.Failed > 0 {
		return sum, fmt.Errorf("onboard: default acl reconcile failed for %d of %d tenants", sum.Failed, len(tenants))
	}
	return sum, nil
}
