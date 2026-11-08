// Package acltest publishes record ACLs for tests (P3-31). Authorization
// fails closed, so a test that writes or reads records as a user must first
// publish a policy for the entity type. The onboarding default ACL is P3-35's;
// the publish route with its validation is P3-35's too. Publish here inserts
// the row directly, as nava_app, in the caller's tenant transaction.
package acltest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// publishSQL inserts the next ACL version of the tenant's entity type key.
const publishSQL = `
INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
SELECT d.tenant_id, d.id,
       coalesce((SELECT max(a.version) FROM acl_versions a WHERE a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id), 0) + 1,
       $3::text::jsonb, '[]'::jsonb
FROM entity_definitions d
WHERE d.tenant_id = $1 AND d.key = $2
RETURNING version`

// Publish inserts rules as the entity type's next ACL version, in tx, and
// returns the version. The entity type must exist in tx's tenant.
func Publish(ctx context.Context, tx pgx.Tx, entityKey string, rules ...authz.Rule) (int, error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return 0, err
	}
	if rules == nil {
		rules = []authz.Rule{}
	}
	b, err := json.Marshal(rules)
	if err != nil {
		return 0, err
	}
	var v int
	if err := tx.QueryRow(ctx, publishSQL, tenant, entityKey, string(b)).Scan(&v); err != nil {
		return 0, fmt.Errorf("acltest: publish %s: %w", entityKey, err)
	}
	return v, nil
}

// AllowUsers is a rule that lets the users read and write every record.
func AllowUsers(ids ...uuid.UUID) authz.Rule {
	return authz.Rule{ID: "test-allow-users", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: ids}}
}

// AllowGroups is a rule that lets the groups' members read and write every
// record.
func AllowGroups(ids ...uuid.UUID) authz.Rule {
	return authz.Rule{ID: "test-allow-groups", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: ids}}
}
