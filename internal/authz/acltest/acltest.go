// Package acltest publishes record ACLs for tests (P3-31). Authorization
// fails closed, so a test that writes or reads records as a user must first
// publish a policy for the entity type. The onboarding default ACL is P3-35's;
// Publish here goes through authz.PolicyStore.Publish, as nava_app, in the caller's
// tenant transaction, without the route's validation.
package acltest

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

// Publish inserts rules, with no masks, as the entity type's next ACL
// version, in tx, and returns the version. The entity type must exist in tx's
// tenant.
func Publish(ctx context.Context, tx pgx.Tx, entityKey string, rules ...authz.Rule) (int, error) {
	return PublishPolicy(ctx, tx, entityKey, rules, nil)
}

// PublishPolicy is Publish with field masks (P3-33).
func PublishPolicy(ctx context.Context, tx pgx.Tx, entityKey string, rules []authz.Rule, masks []authz.Mask) (int, error) {
	store := authz.PolicyStore{}
	cur, err := store.Newest(ctx, tx, entityKey)
	if err != nil {
		return 0, fmt.Errorf("acltest: publish %s: %w", entityKey, err)
	}
	have := 0
	if cur != nil {
		have = cur.Version
	}
	// The same audited path the publish route uses: acl_versions has the
	// commit check (P3-35), so a bare INSERT would fail at COMMIT.
	pub, err := store.Publish(ctx, tx, entityKey, have, authz.Publisher{System: "acltest", Source: audit.SourceSystem}, rules, masks)
	if err != nil {
		return 0, fmt.Errorf("acltest: publish %s: %w", entityKey, err)
	}
	return pub.Version, nil
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
