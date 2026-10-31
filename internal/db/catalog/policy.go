package catalog

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// Policy is one row of pg_policies as the allow-list guard compares it: the
// command, the sorted roles and the deparsed USING and WITH CHECK ("" when
// absent).
type Policy struct{ Cmd, Roles, Qual, Check string }

// TenantPred is the deparsed tenant predicate on column col.
func TenantPred(col string) string {
	return "(" + col + " = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)"
}

// AllowList is ADR-001 A.3: the only policies that do not test app.tenant_id,
// keyed on schema.table.name and each pinned to its command, roles and exact
// predicates (T09, TICK-64). The key carries the table because PostgreSQL
// names policies per table: a copy of binding_lookup on records is not
// binding_lookup. Editing it needs CTO sign-off (PLAN-001 D-P10).
func AllowList() map[string]Policy {
	pending := "((dispatched_at IS NULL) AND (dead_at IS NULL))"
	outboxExpired := "((dispatched_at < (now() - '7 days'::interval)) OR (dead_at < (now() - '30 days'::interval)))"
	refExpired := "(closed_at < (now() - '30 days'::interval))"
	keyExpired := "(expires_at < now())"
	return map[string]Policy{
		"public.outbox.outbox_relay_select":                {"SELECT", "nava_relay", "true", ""},
		"public.outbox.outbox_relay_update":                {"UPDATE", "nava_relay", pending, "true"},
		"public.tenant_identity_binding.binding_lookup":    {"SELECT", "nava_app,nava_groupsync", "true", ""},
		"public.tenant_identity_binding.binding_onboard":   {"INSERT", "nava_onboard", "", "(tenant_id IN ( SELECT tenants.id\n   FROM tenants\n  WHERE (tenants.state = 'provisioning'::text)))"},
		"public.idempotency_keys.idempotency_purge":        {"DELETE", "nava_purge", keyExpired, ""},
		"public.idempotency_keys.idempotency_purge_select": {"SELECT", "nava_purge", keyExpired, ""},
		"public.outbox.outbox_purge":                       {"DELETE", "nava_purge", outboxExpired, ""},
		"public.outbox.outbox_purge_select":                {"SELECT", "nava_purge", outboxExpired, ""},
		"public.workflow_ref.workflow_ref_purge":           {"DELETE", "nava_purge", refExpired, ""},
		"public.workflow_ref.workflow_ref_purge_select":    {"SELECT", "nava_purge", refExpired, ""},
	}
}

// PinnedTenantPolicies are the tenant policies whose predicate adds a state
// test to the tenant predicate (the onboarding path), keyed as AllowList.
// email_domain_onboard (T18a) lets nava_onboard claim email domains only for
// its own tenant while it is provisioning, as binding_onboard binds one;
// email_domain_release (TICK-280) frees them under the same guard.
// pending_invite_insert and pending_invite_reinvite (T252a) do the same for
// the invites InviteUser writes (security finding 1 on PR #164).
// entity_definitions_onboard and entity_definition_versions_onboard (P3-02,
// TICK-340) let onboarding seed its own provisioning tenant's entity types.
func PinnedTenantPolicies() map[string]Policy {
	own := TenantPred("id")
	and := func(state string) string { return "(" + own + " AND (state = '" + state + "'::text))" }
	provisioning := "(tenant_id IN ( SELECT tenants.id\n   FROM tenants\n  WHERE (tenants.state = 'provisioning'::text)))"
	ownProvisioning := "(" + TenantPred("tenant_id") + " AND " + provisioning + ")"
	return map[string]Policy{
		"public.tenants.tenants_onboard_insert":                                {"INSERT", "nava_onboard", "", and("provisioning")},
		"public.tenants.tenants_activate":                                      {"UPDATE", "nava_onboard", and("provisioning"), and("active")},
		"public.tenant_email_domain.email_domain_onboard":                      {"INSERT", "nava_onboard", "", ownProvisioning},
		"public.tenant_email_domain.email_domain_release":                      {"DELETE", "nava_onboard", ownProvisioning, ""},
		"public.pending_invite.pending_invite_insert":                          {"INSERT", "nava_onboard", "", ownProvisioning},
		"public.pending_invite.pending_invite_reinvite":                        {"UPDATE", "nava_onboard", ownProvisioning, ownProvisioning},
		"public.groups.groups_insert_onboard":                                  {"INSERT", "nava_onboard", "", ownProvisioning},
		"public.groups.groups_select_onboard":                                  {"SELECT", "nava_onboard", ownProvisioning, ""},
		"public.tenant_admin_group.tenant_admin_group_insert":                  {"INSERT", "nava_onboard", "", ownProvisioning},
		"public.tenant_admin_group.tenant_admin_group_select_onboard":          {"SELECT", "nava_onboard", ownProvisioning, ""},
		"public.entity_definitions.entity_definitions_onboard":                 {"INSERT", "nava_onboard", "", ownProvisioning},
		"public.entity_definition_versions.entity_definition_versions_onboard": {"INSERT", "nava_onboard", "", ownProvisioning},
	}
}

// testsTenant reports whether p scopes every clause its command uses by
// exactly the tenant predicate on col: USING for SELECT and DELETE, WITH
// CHECK for INSERT, both for UPDATE. Other text, a missing clause or an extra
// one does not count, so neither "current_setting(...) IS NOT NULL" nor
// WITH CHECK (true) passes.
func testsTenant(p Policy, col string) bool {
	want := TenantPred(col)
	switch p.Cmd {
	case "SELECT", "DELETE":
		return p.Qual == want && p.Check == ""
	case "INSERT":
		return p.Qual == "" && p.Check == want
	case "UPDATE":
		return p.Qual == want && p.Check == want
	}
	return false
}

// PolicyProblems is the allow-list guard (CI-1, I1). It reports every policy
// in a platform schema that is FOR ALL or TO PUBLIC, a policy name used more
// than once, an allow-listed or pinned policy that differs from its pinned
// form or is missing, and any other policy whose predicate is not exactly the
// tenant predicate. Policies are matched on schema.table.name. untested lists,
// by that key, the policies that do not test app.tenant_id.
func PolicyProblems(ctx context.Context, q Querier) (problems, untested []string, err error) {
	rows, err := q.Query(ctx, `
		SELECT schemaname::text, tablename::text, policyname::text, cmd, roles::text[], coalesce(qual, ''), coalesce(with_check, '')
		FROM pg_catalog.pg_policies WHERE `+platformSchema("schemaname")+` ORDER BY policyname`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	allow, pinned := AllowList(), PinnedTenantPolicies()
	seen := map[string]bool{}
	tablesOf := map[string][]string{}
	for rows.Next() {
		var schema, table, name string
		var roles []string
		var p Policy
		if err := rows.Scan(&schema, &table, &name, &p.Cmd, &roles, &p.Qual, &p.Check); err != nil {
			return nil, nil, err
		}
		slices.Sort(roles)
		p.Roles = strings.Join(roles, ",")
		key := schema + "." + table + "." + name
		seen[key] = true
		tablesOf[name] = append(tablesOf[name], schema+"."+table)
		if p.Cmd == "ALL" {
			problems = append(problems, fmt.Sprintf("%s on %s is FOR ALL", name, table))
		}
		if slices.Contains(roles, "public") {
			problems = append(problems, fmt.Sprintf("%s on %s is TO PUBLIC", name, table))
		}
		if want, ok := allow[key]; ok {
			untested = append(untested, key)
			if p != want {
				problems = append(problems, fmt.Sprintf("%s (allow-listed) = %q, want %q", name, p, want))
			}
			continue
		}
		if want, ok := pinned[key]; ok {
			if p != want {
				problems = append(problems, fmt.Sprintf("%s = %q, want %q", name, p, want))
			}
			continue
		}
		col := tenantKey(schema, table)
		if !testsTenant(p, col) {
			untested = append(untested, key)
			problems = append(problems, fmt.Sprintf("%s on %s %q does not test exactly %s and is not on the allow-list",
				name, table, p, TenantPred(col)))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	for name, tables := range tablesOf {
		if len(tables) > 1 {
			slices.Sort(tables)
			problems = append(problems, fmt.Sprintf("%s is used by more than one policy, on %s", name, strings.Join(tables, ", ")))
		}
	}
	for _, m := range []map[string]Policy{allow, pinned} {
		for key := range m {
			if !seen[key] {
				problems = append(problems, key+" is missing")
			}
		}
	}
	slices.Sort(problems)
	slices.Sort(untested)
	return problems, untested, nil
}
