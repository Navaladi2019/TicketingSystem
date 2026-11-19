package isolation

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// legacyACLInsert runs fn with acl_versions' tenant-id trigger (TICK-940)
// disabled, as the superuser, and always enables it again. It stores the
// rows the trigger now refuses but a database can still hold: rows published
// before 20261119100000, and ids whose group or user was tombstoned after
// the publish. The read path must grant nothing on them, and these paths
// prove it. The suite runs its paths one at a time.
//
// The DISABLE is committed (fn writes on another connection, which an open
// ALTER TABLE would block), so while fn runs the trigger is off for the whole
// cluster: the isolation suite's own, throwaway one. Only the table owner or
// a superuser can do it (TestRuntimeRolesCannotSkipTheTenantIDTrigger), and
// only this package's tests reach it (TestLegacyACLInsertIsTestOnly). A
// failed ENABLE is always returned, joined to fn's error, never dropped
// (TICK-943 finding F2).
func legacyACLInsert(ctx context.Context, super *pgx.Conn, fn func() error) (err error) {
	if _, err := super.Exec(ctx, "ALTER TABLE acl_versions DISABLE TRIGGER acl_versions_tenant_ids"); err != nil {
		return err
	}
	defer func() {
		if _, e := super.Exec(context.WithoutCancel(ctx), "ALTER TABLE acl_versions ENABLE TRIGGER acl_versions_tenant_ids"); e != nil {
			err = errors.Join(err, fmt.Errorf("legacyACLInsert: re-enable acl_versions_tenant_ids: %w", e))
		}
	}()
	return fn()
}

// aclPaths are the S10 authorization tables (P3-30, TICK-615): acl_versions
// and record_shares are per tenant, insert-only, and their composite foreign
// keys hold the tenant. The generic CI-2 checks cover them as every role; these
// name the lookups the evaluator (P3-31) and the list join (P3-34) make.
func aclPaths() []Path {
	sql := func(name, role string, expect Expect, query string, args ...any) Path {
		return Path{Name: name, Case: "CI-2", Role: role, SQL: &SQLCall{Query: query, Args: args, Expect: expect}}
	}
	return []Path{
		sql("acl_versions: read tenant B's ACL by its type", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM acl_versions WHERE entity_definition_id = $1", EntityDefinitionB1),
		sql("acl_versions: read every type's version 1", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM acl_versions WHERE version = 1"),
		sql("acl_versions: publish an ACL for tenant B", "nava_app", Denied,
			"INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks) VALUES ($1, $2, 3, '[]', '[]')",
			TenantB, EntityDefinitionB1),
		// The composite foreign key holds the tenant: tenant A cannot publish an
		// ACL for tenant B's type, though the policy passes the row. As
		// nava_app, the one role that publishes (ADR-010-A1 A1-3, TICK-940).
		sql("acl_versions: publish an ACL in tenant A for tenant B's type", "nava_app", Denied,
			"INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks) VALUES ($1, $2, 3, '[]', '[]')",
			TenantA, EntityDefinitionB1),
		// ADR-010-A1 A1-3 (TICK-940): the worker publishes nothing, even in its
		// own tenant for its own type. Denied also accepts class 23; the
		// database test TestTICK940WorkerCannotPublish pins 42501.
		sql("acl_versions: nava_worker publishes an ACL in its own tenant", "nava_worker", Denied,
			"INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks) VALUES ($1, $2, 3, '[]', '[]')",
			TenantA, definitionA1),
		sql("acl_versions: rewrite tenant B's ACL", "nava_app", Denied,
			"UPDATE acl_versions SET rules = '[]' WHERE tenant_id = $1", TenantB),
		sql("acl_versions: delete tenant B's ACL", "nava_worker", Denied,
			"DELETE FROM acl_versions WHERE tenant_id = $1", TenantB),
		// The list join (P3-34) by the caller's ids: tenant B's user and group
		// ids find none of tenant B's shares from tenant A.
		sql("record_shares: list the records shared with tenant B's user", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM record_shares WHERE principal_id = $1", PrincipalB1),
		sql("record_shares: list the records shared with tenant B's group", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM record_shares WHERE group_id = $1", GroupB1),
		sql("record_shares: read tenant B's shares by record", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM record_shares WHERE record_id = $1", RecordB2),
		sql("record_shares: share a record for tenant B", "nava_app", Denied,
			"INSERT INTO record_shares (tenant_id, record_id, user_id, principal_kind) VALUES ($1, $2, $3, 'user')",
			TenantB, RecordB2, PrincipalB1),
		// The composite foreign keys hold the tenant, though the policy passes
		// the row: tenant A cannot share tenant B's record, nor share its own
		// record with tenant B's user or group (TICK-625, AR-S10-01).
		sql("record_shares: share tenant B's record from tenant A", "nava_worker", Denied,
			"INSERT INTO record_shares (tenant_id, record_id, user_id, principal_kind) VALUES ($1, $2, $3, 'user')",
			TenantA, RecordB2, principalA1),
		sql("record_shares: share a tenant A record with tenant B's user", "nava_app", Denied,
			"INSERT INTO record_shares (tenant_id, record_id, user_id, principal_kind) VALUES ($1, $2, $3, 'user')",
			TenantA, RecordA2, PrincipalB1),
		sql("record_shares: share a tenant A record with tenant B's group", "nava_worker", Denied,
			"INSERT INTO record_shares (tenant_id, record_id, group_id, principal_kind) VALUES ($1, $2, $3, 'group')",
			TenantA, RecordA2, GroupB1),
		sql("record_shares: rewrite tenant B's shares", "nava_app", Denied,
			"UPDATE record_shares SET user_id = $2 WHERE tenant_id = $1", TenantB, NewID),
		sql("record_shares: delete tenant B's shares", "nava_worker", Denied,
			"DELETE FROM record_shares WHERE tenant_id = $1", TenantB),
	}
}
