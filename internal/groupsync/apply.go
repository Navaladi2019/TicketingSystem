package groupsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// DefaultMaxOrgUnitDepth bounds every walk of the org-unit tree. A walk that
// reaches it fails closed.
const DefaultMaxOrgUnitDepth = 64

// Errors apply and CheckOrgUnitChains return. Match with errors.Is.
var (
	// ErrOrgUnitCycle: a parent change would make an org unit its own ancestor.
	ErrOrgUnitCycle = errors.New("groupsync: org unit parent cycle")
	// ErrOrgUnitDepth: a walk of the org-unit tree hit the depth bound.
	ErrOrgUnitDepth = errors.New("groupsync: org unit tree deeper than the bound")
	// ErrBadSnapshot: the directory snapshot contradicts itself.
	ErrBadSnapshot = errors.New("groupsync: inconsistent directory snapshot")
)

// stats counts the rows one apply wrote and removed.
type stats struct {
	PrincipalsWritten, PrincipalsTombstoned int
	GroupsWritten, GroupsDeleted            int
	// AdminGroupDeleted is true when a deleted group was the tenant's recorded
	// admins group (tenant_admin_group); the tenant then has no admin.
	AdminGroupDeleted                bool
	OrgUnitsWritten, OrgUnitsDeleted int
	// closureTouched holds the ids of the units added, moved, revived or
	// tombstoned: the roots of the closure delta.
	closureTouched []string
	// OrgMembersWritten counts org_unit_members rows inserted or whose
	// is_primary changed; OrgMembersRemoved those deleted.
	OrgMembersWritten, OrgMembersRemoved int
	// OrgSubtreeMax is the largest number of live units (the unit itself
	// included) under any one unit a user belongs to.
	OrgSubtreeMax int
	// ClosureRows is the tenant's org_unit_closure row count after the run;
	// ClosureRebuilt is true when the run wrote closure rows (a delta or a full rebuild).
	ClosureRows    int
	ClosureRebuilt bool
	// ClosureUnrooted is the number of live org units with no closure self
	// row after the run (TICK-1058); anything above 0 is reported, not failed.
	ClosureUnrooted int
	// Ignored is the number of snapshot groups and org units whose path is
	// outside the subtree they may be written to.
	Ignored int
}

// ---- path rules ------------------------------------------------------------

// isOrgUnitPath: strictly below /t/<tenant_id>/ou/.
func isOrgUnitPath(tenantID uuid.UUID, path string) bool {
	rest, ok := strings.CutPrefix(path, identity.TenantGroupPrefix(tenantID)+"ou/")
	return ok && rest != ""
}

// isGroupPath: below /t/<tenant_id>/ and not the ou container or below it.
func isGroupPath(tenantID uuid.UUID, path string) bool {
	rest, ok := strings.CutPrefix(path, identity.TenantGroupPrefix(tenantID))
	if !ok || rest == "" {
		return false
	}
	return rest != "ou" && !strings.HasPrefix(rest, "ou/")
}

// ---- apply -----------------------------------------------------------------

// apply makes the tenant's identity tables equal changes, in tx, which must be
// scoped to tenantID and hold the tenant's lock. The caller commits or rolls
// back.
func apply(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, changes identity.DirectoryChanges, now time.Time, maxDepth int) (stats, error) {
	var st stats
	tid := tenantID.String()

	// Keep only what the tenant's subtree allows; the snapshot is not trusted
	// to have filtered.
	var groups, units []identity.DirectoryGroup
	for _, g := range changes.Groups {
		if isGroupPath(tenantID, g.Path) {
			groups = append(groups, g)
		} else {
			st.Ignored++
		}
	}
	for _, g := range changes.OrgUnits {
		if isOrgUnitPath(tenantID, g.Path) {
			units = append(units, g)
		} else {
			st.Ignored++
		}
	}

	bound, err := boundAliases(ctx, tx, tid)
	if err != nil {
		return st, err
	}
	userIDs, err := applyPrincipals(ctx, tx, tid, changes.Users, bound, now, &st)
	if err != nil {
		return st, err
	}
	if err := applyGroups(ctx, tx, tid, groups, userIDs, &st); err != nil {
		return st, err
	}
	if err := applyOrgUnits(ctx, tx, tenantID, units, changes.Users, userIDs, maxDepth, &st); err != nil {
		return st, err
	}
	// Same transaction as the tree write: both commit or neither does. A run
	// that touched no unit changed no parent_id or deleted_at.
	st.ClosureRows, st.ClosureRebuilt, err = rebuildClosure(ctx, tx, tenantID, st.closureTouched)
	if err != nil {
		return st, err
	}
	if st.ClosureUnrooted, err = unrootedUnits(ctx, tx, tenantID); err != nil {
		return st, err
	}
	return st, nil
}

// send runs b and returns the first error.
func send(ctx context.Context, tx pgx.Tx, b *pgx.Batch) error {
	if b.Len() == 0 {
		return nil
	}
	br := tx.SendBatch(ctx, b)
	for range b.Len() {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return err
		}
	}
	return br.Close()
}

// linkJSON is one element of federated_links. UserID is set only for an alias
// the tenant binds; for any other alias it is nil and omitted: the row holds
// the foreign-link marker, never another IdP's external user id.
type linkJSON struct {
	IdentityProvider string  `json:"identityProvider"`
	UserID           *string `json:"userId,omitempty"`
}

// boundAliases is the set of IdP aliases the tenant binds. A tenant with no
// binding row has none, so no link is written for it (fail closed).
func boundAliases(ctx context.Context, tx pgx.Tx, tid string) (map[string]bool, error) {
	rows, err := tx.Query(ctx, `SELECT DISTINCT unnest(idp_aliases) FROM tenant_identity_binding WHERE tenant_id = $1::uuid`, tid)
	if err != nil {
		return nil, fmt.Errorf("read bound idps: %w", err)
	}
	defer rows.Close()
	bound := map[string]bool{}
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, fmt.Errorf("read bound idps: %w", err)
		}
		bound[a] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read bound idps: %w", err)
	}
	return bound, nil
}

// linksJSON is federated_links as the table documents it. A user with none
// gets '[]', never NULL: NULL means "never synced" and the resolver refuses it.
// A link to an IdP the tenant does not bind is kept as a marker: the alias
// only, without the external user id (that id is the other tenant's data). The
// resolver refuses a principal with any alias the tenant does not bind, so the
// marker must stay: dropping it would leave the resolver an empty list to accept.
func linksJSON(links []identity.FederatedLink, bound map[string]bool) ([]byte, error) {
	out := make([]linkJSON, 0, len(links))
	for _, l := range links {
		if !bound[l.IdPAlias] {
			out = append(out, linkJSON{IdentityProvider: l.IdPAlias})
			continue
		}
		uid := l.ExternalUserID
		out = append(out, linkJSON{IdentityProvider: l.IdPAlias, UserID: &uid})
	}
	return json.Marshal(out)
}

func sameJSON(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

type principalRow struct {
	id      uuid.UUID
	email   *string
	enabled bool
	links   []byte
	deleted bool // tombstoned: deleted_at is set
}

// applyPrincipals returns the Keycloak user id -> principal id map.
func applyPrincipals(ctx context.Context, tx pgx.Tx, tid string, users []identity.DirectoryUser, bound map[string]bool, now time.Time, st *stats) (map[string]uuid.UUID, error) {
	rows, err := tx.Query(ctx, `SELECT id, keycloak_user_id, email, enabled, federated_links, deleted_at IS NOT NULL FROM principals WHERE tenant_id = $1::uuid`, tid)
	if err != nil {
		return nil, fmt.Errorf("read principals: %w", err)
	}
	have := map[string]principalRow{}
	for rows.Next() {
		var kc string
		var r principalRow
		if err := rows.Scan(&r.id, &kc, &r.email, &r.enabled, &r.links, &r.deleted); err != nil {
			rows.Close()
			return nil, fmt.Errorf("read principals: %w", err)
		}
		have[kc] = r
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read principals: %w", err)
	}

	ids := make(map[string]uuid.UUID, len(users))
	want := make(map[string]bool, len(users))
	b := &pgx.Batch{}
	for _, u := range users {
		if u.KeycloakUserID == "" {
			return nil, fmt.Errorf("%w: user without an id", ErrBadSnapshot)
		}
		if want[u.KeycloakUserID] {
			return nil, fmt.Errorf("%w: user id appears twice", ErrBadSnapshot)
		}
		want[u.KeycloakUserID] = true
		links, err := linksJSON(u.FederatedLinks, bound)
		if err != nil {
			return nil, fmt.Errorf("federated links: %w", err)
		}
		var email *string
		if u.Email != "" {
			email = &u.Email
		}
		cur, ok := have[u.KeycloakUserID]
		switch {
		case !ok:
			id, err := uuid.NewV7()
			if err != nil {
				return nil, fmt.Errorf("principal id: %w", err)
			}
			ids[u.KeycloakUserID] = id
			b.Queue(`INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
				VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::jsonb, $7)`, tid, id.String(), u.KeycloakUserID, email, u.Enabled, string(links), now)
			st.PrincipalsWritten++
		default:
			ids[u.KeycloakUserID] = cur.id
			if cur.deleted {
				// A user who came back keeps the row and id; their
				// memberships are re-created by applyGroups.
				b.Queue(`UPDATE principals SET email = $3, enabled = $4, federated_links = $5::jsonb, deleted_at = NULL, synced_at = $6
					WHERE tenant_id = $1::uuid AND id = $2::uuid`, tid, cur.id.String(), email, u.Enabled, string(links), now)
				st.PrincipalsWritten++
				continue
			}
			if ptrEq(cur.email, email) && cur.enabled == u.Enabled && sameJSON(cur.links, links) {
				continue
			}
			b.Queue(`UPDATE principals SET email = $3, enabled = $4, federated_links = $5::jsonb, synced_at = $6
				WHERE tenant_id = $1::uuid AND id = $2::uuid`, tid, cur.id.String(), email, u.Enabled, string(links), now)
			st.PrincipalsWritten++
		}
	}
	// A principal that left the directory is tombstoned, never deleted, so a
	// foreign key to it cannot fail the sync. It is not in userIDs, so
	// applyGroups removes its memberships. Tombstoned rows are left alone.
	var gone []string
	for kc, r := range have {
		if !want[kc] && !r.deleted {
			gone = append(gone, r.id.String())
		}
	}
	if len(gone) > 0 {
		b.Queue(`UPDATE principals SET enabled = false, deleted_at = $3, synced_at = $3
			WHERE tenant_id = $1::uuid AND id = ANY($2::uuid[])`, tid, gone, now)
		st.PrincipalsTombstoned = len(gone)
	}
	if err := send(ctx, tx, b); err != nil {
		return nil, fmt.Errorf("write principals: %w", err)
	}
	return ids, nil
}

func ptrEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

type pair struct{ group, principal uuid.UUID }

// applyGroups writes groups and their memberships. Membership needs the user
// to be in the snapshot's directory: a member id with no principal is skipped.
func applyGroups(ctx context.Context, tx pgx.Tx, tid string, groups []identity.DirectoryGroup, userIDs map[string]uuid.UUID, st *stats) error {
	// Tombstoned rows are read too (D4): a group that came back is revived
	// on its own id, never inserted again (23505).
	rows, err := tx.Query(ctx, `SELECT id, keycloak_group_id, path, deleted_at IS NOT NULL FROM groups WHERE tenant_id = $1::uuid`, tid)
	if err != nil {
		return fmt.Errorf("read groups: %w", err)
	}
	type groupRow struct {
		id      uuid.UUID
		path    string
		deleted bool
	}
	have := map[string]groupRow{}
	for rows.Next() {
		var kc string
		var r groupRow
		if err := rows.Scan(&r.id, &kc, &r.path, &r.deleted); err != nil {
			rows.Close()
			return fmt.Errorf("read groups: %w", err)
		}
		have[kc] = r
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read groups: %w", err)
	}

	memRows, err := tx.Query(ctx, `SELECT group_id, principal_id FROM group_members WHERE tenant_id = $1::uuid`, tid)
	if err != nil {
		return fmt.Errorf("read group members: %w", err)
	}
	haveMembers := map[pair]bool{}
	for memRows.Next() {
		var p pair
		if err := memRows.Scan(&p.group, &p.principal); err != nil {
			memRows.Close()
			return fmt.Errorf("read group members: %w", err)
		}
		haveMembers[p] = true
	}
	memRows.Close()
	if err := memRows.Err(); err != nil {
		return fmt.Errorf("read group members: %w", err)
	}

	b := &pgx.Batch{}
	want := make(map[string]bool, len(groups))
	wantMembers := map[pair]bool{}
	for _, g := range groups {
		if g.KeycloakGroupID == "" {
			return fmt.Errorf("%w: group without an id", ErrBadSnapshot)
		}
		if want[g.KeycloakGroupID] {
			return fmt.Errorf("%w: group id appears twice", ErrBadSnapshot)
		}
		want[g.KeycloakGroupID] = true
		cur, ok := have[g.KeycloakGroupID]
		id := cur.id
		switch {
		case !ok:
			id, err = uuid.NewV7()
			if err != nil {
				return fmt.Errorf("group id: %w", err)
			}
			b.Queue(`INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, $2::uuid, $3, $4)`,
				tid, id.String(), g.KeycloakGroupID, g.Path)
			st.GroupsWritten++
		case cur.deleted:
			// Revive on the same id. Memberships are re-inserted in batch 2,
			// after this one, so the live FK accepts them.
			b.Queue(`UPDATE groups SET path = $3, deleted_at = NULL WHERE tenant_id = $1::uuid AND id = $2::uuid`, tid, id.String(), g.Path)
			st.GroupsWritten++
		case cur.path != g.Path:
			b.Queue(`UPDATE groups SET path = $3 WHERE tenant_id = $1::uuid AND id = $2::uuid`, tid, id.String(), g.Path)
			st.GroupsWritten++
		}
		for _, m := range g.MemberKeycloakUserIDs {
			if pid, ok := userIDs[m]; ok {
				wantMembers[pair{id, pid}] = true
			}
		}
	}
	// Inserts first, so a new group exists before its members; the stale
	// memberships and groups go afterwards.
	if err := send(ctx, tx, b); err != nil {
		return fmt.Errorf("write groups: %w", err)
	}

	// Batch 2 (I3): membership deletes first, so the tombstone passes the
	// live FK (D11); then the tombstone; then the pointer deletes, which the
	// policies allow only once the group is tombstoned. Inserts go last and
	// only target live groups.
	b = &pgx.Batch{}
	for p := range haveMembers {
		if !wantMembers[p] {
			b.Queue(`DELETE FROM group_members WHERE tenant_id = $1::uuid AND group_id = $2::uuid AND principal_id = $3::uuid`,
				tid, p.group.String(), p.principal.String())
		}
	}
	var gone []string
	for kc, r := range have {
		if !want[kc] && !r.deleted {
			gone = append(gone, r.id.String())
		}
	}
	if len(gone) > 0 {
		// Ask first whether the recorded admins group is going.
		var recorded bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tenant_admin_group WHERE tenant_id = $1::uuid AND group_id = ANY($2::uuid[]))`,
			tid, gone).Scan(&recorded); err != nil {
			return fmt.Errorf("read admin group: %w", err)
		}
		st.AdminGroupDeleted = recorded
		b.Queue(`UPDATE groups SET deleted_at = now() WHERE tenant_id = $1::uuid AND id = ANY($2::uuid[])`, tid, gone)
		b.Queue(`DELETE FROM tenant_admin_group WHERE tenant_id = $1::uuid AND group_id = ANY($2::uuid[])`, tid, gone)
		b.Queue(`DELETE FROM tenant_auditor_group WHERE tenant_id = $1::uuid AND group_id = ANY($2::uuid[])`, tid, gone)
		st.GroupsDeleted = len(gone)
	}
	for p := range wantMembers {
		if !haveMembers[p] {
			b.Queue(`INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1::uuid, $2::uuid, $3::uuid)`,
				tid, p.group.String(), p.principal.String())
		}
	}
	if err := send(ctx, tx, b); err != nil {
		return fmt.Errorf("write group members: %w", err)
	}
	return nil
}

type unitRow struct {
	id      uuid.UUID
	parent  *uuid.UUID
	name    string
	deleted bool // tombstoned: deleted_at is set
}

// applyOrgUnits writes the tree. New units go in as roots, then parents are
// set, then removed units are tombstoned (deleted_at; ADR-010 D7, TICK-947:
// nava_groupsync has no DELETE on org_units) and a returning one is revived on
// the same id. Membership follows: a vanishing unit's org_unit_members rows are
// deleted before the tombstone, which the live FK would otherwise refuse
// (23503), and the other rows and each user's primary unit are made equal to
// the snapshot. Before anything commits, a walk up from each
// moved unit checks that its chain to the root has no cycle and is no longer
// than maxDepth. It does not bound the depth of a moved unit's descendants.
func applyOrgUnits(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, units []identity.DirectoryGroup, users []identity.DirectoryUser, userIDs map[string]uuid.UUID, maxDepth int, st *stats) error {
	tid := tenantID.String()
	rows, err := tx.Query(ctx, `SELECT id, parent_id, keycloak_group_id, name, deleted_at IS NOT NULL FROM org_units WHERE tenant_id = $1::uuid`, tid)
	if err != nil {
		return fmt.Errorf("read org units: %w", err)
	}
	have := map[string]unitRow{}
	for rows.Next() {
		var kc string
		var r unitRow
		if err := rows.Scan(&r.id, &r.parent, &kc, &r.name, &r.deleted); err != nil {
			rows.Close()
			return fmt.Errorf("read org units: %w", err)
		}
		have[kc] = r
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read org units: %w", err)
	}

	// Ids first: a parent may come after its child in the snapshot.
	idOf := make(map[string]uuid.UUID, len(units))
	for _, u := range units {
		if u.KeycloakGroupID == "" {
			return fmt.Errorf("%w: org unit without an id", ErrBadSnapshot)
		}
		if _, dup := idOf[u.KeycloakGroupID]; dup {
			return fmt.Errorf("%w: org unit id appears twice", ErrBadSnapshot)
		}
		if cur, ok := have[u.KeycloakGroupID]; ok {
			idOf[u.KeycloakGroupID] = cur.id
		} else {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("org unit id: %w", err)
			}
			idOf[u.KeycloakGroupID] = id
		}
	}

	ins, upd := &pgx.Batch{}, &pgx.Batch{}
	var moved []string
	for _, u := range units {
		id := idOf[u.KeycloakGroupID]
		var parent *uuid.UUID
		if u.ParentKeycloakGroupID != "" {
			p, ok := idOf[u.ParentKeycloakGroupID]
			if !ok {
				// A parent that is not an org unit of this tenant: never
				// attach to it, and never guess a root.
				return fmt.Errorf("%w: org unit parent is not an org unit of the tenant", ErrBadSnapshot)
			}
			parent = &p
		}
		cur, ok := have[u.KeycloakGroupID]
		if !ok {
			ins.Queue(`INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1::uuid, $2::uuid, $3, $4)`,
				tid, id.String(), u.KeycloakGroupID, u.Name)
			st.OrgUnitsWritten++
			st.closureTouched = append(st.closureTouched, id.String())
			if parent != nil {
				upd.Queue(`UPDATE org_units SET parent_id = $3::uuid WHERE tenant_id = $1::uuid AND id = $2::uuid`, tid, id.String(), parent.String())
				moved = append(moved, id.String())
			}
			continue
		}
		parentChanged := !uuidPtrEq(cur.parent, parent)
		if !parentChanged && cur.name == u.Name && !cur.deleted {
			continue
		}
		var pArg *string
		if parent != nil {
			s := parent.String()
			pArg = &s
		}
		upd.Queue(`UPDATE org_units SET parent_id = $3::uuid, name = $4, deleted_at = NULL WHERE tenant_id = $1::uuid AND id = $2::uuid`, tid, id.String(), pArg, u.Name)
		st.OrgUnitsWritten++
		// A revive reattaches the unit, like a new one: its parent may have
		// moved deeper while it was tombstoned, so its chain is checked too.
		if parentChanged || cur.deleted {
			st.closureTouched = append(st.closureTouched, id.String())
			moved = append(moved, id.String())
		}
	}
	var gone []string
	for kc, r := range have {
		if _, ok := idOf[kc]; !ok && !r.deleted {
			gone = append(gone, r.id.String())
		}
	}
	mem, err := planOrgMembers(ctx, tx, tid, units, idOf, users, userIDs)
	if err != nil {
		return err
	}
	st.OrgMembersRemoved = mem.removed
	st.OrgMembersWritten = mem.written
	st.OrgSubtreeMax = orgSubtreeMax(units, mem.unitsOf, maxDepth)

	del := &pgx.Batch{}
	if len(gone) > 0 {
		del.Queue(`UPDATE org_units SET deleted_at = now() WHERE tenant_id = $1::uuid AND id = ANY($2::uuid[])`, tid, gone)
		st.OrgUnitsDeleted = len(gone)
		st.closureTouched = append(st.closureTouched, gone...)
	}
	// ins, upd; then the membership deletes (a vanishing unit's rows go before
	// its tombstone); then the tombstone; then the inserts, which only target
	// live units.
	for _, b := range []*pgx.Batch{ins, upd, mem.before, del, mem.after} {
		if err := send(ctx, tx, b); err != nil {
			return fmt.Errorf("write org units: %w", err)
		}
	}
	return CheckOrgUnitChains(ctx, tx, tenantID, moved, maxDepth)
}

type orgMemberPlan struct {
	before, after    *pgx.Batch
	removed, written int
	// unitsOf holds, per Keycloak user id, the Keycloak ids of the live units
	// the user belongs to.
	unitsOf map[string][]string
}

// planOrgMembers diffs org_unit_members against the snapshot. before holds the
// deletes and the is_primary clears (a user has one primary, so the old one is
// cleared before the new one is set); after holds the inserts and the sets.
// A member with no principal in the snapshot is skipped, as in applyGroups.
func planOrgMembers(ctx context.Context, tx pgx.Tx, tid string, units []identity.DirectoryGroup, idOf map[string]uuid.UUID, users []identity.DirectoryUser, userIDs map[string]uuid.UUID) (orgMemberPlan, error) {
	rows, err := tx.Query(ctx, `SELECT org_unit_id, principal_id, is_primary FROM org_unit_members WHERE tenant_id = $1::uuid`, tid)
	if err != nil {
		return orgMemberPlan{}, fmt.Errorf("read org unit members: %w", err)
	}
	have := map[pair]bool{} // value: is_primary
	for rows.Next() {
		var p pair
		var primary bool
		if err := rows.Scan(&p.group, &p.principal, &primary); err != nil {
			rows.Close()
			return orgMemberPlan{}, fmt.Errorf("read org unit members: %w", err)
		}
		have[p] = primary
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return orgMemberPlan{}, fmt.Errorf("read org unit members: %w", err)
	}
	return buildOrgMemberPlan(tid, have, units, idOf, users, userIDs), nil
}

// buildOrgMemberPlan is planOrgMembers without the read: have is the stored
// (unit, principal) -> is_primary.
func buildOrgMemberPlan(tid string, have map[pair]bool, units []identity.DirectoryGroup, idOf map[string]uuid.UUID, users []identity.DirectoryUser, userIDs map[string]uuid.UUID) orgMemberPlan {
	plan := orgMemberPlan{before: &pgx.Batch{}, after: &pgx.Batch{}, unitsOf: map[string][]string{}}
	// Want: each user's units, then the primary among them (D-11-6): the
	// attribute's unit if the user belongs to it, else the only unit, else none.
	pathOf := make(map[string]string, len(units))
	for _, u := range units {
		pathOf[u.KeycloakGroupID] = u.Path
	}
	for _, u := range units {
		seen := map[string]bool{}
		for _, m := range u.MemberKeycloakUserIDs {
			if _, ok := userIDs[m]; ok && !seen[m] {
				seen[m] = true
				plan.unitsOf[m] = append(plan.unitsOf[m], u.KeycloakGroupID)
			}
		}
	}
	want := map[pair]bool{} // value: is_primary
	for _, u := range users {
		kcUnits := plan.unitsOf[u.KeycloakUserID]
		if len(kcUnits) == 0 {
			continue
		}
		pid := userIDs[u.KeycloakUserID]
		var primary string
		if u.PrimaryOrgUnitPath != "" {
			for _, k := range kcUnits {
				if pathOf[k] == u.PrimaryOrgUnitPath {
					primary = k
				}
			}
		}
		if primary == "" && len(kcUnits) == 1 {
			primary = kcUnits[0]
		}
		for _, k := range kcUnits {
			want[pair{idOf[k], pid}] = k == primary
		}
	}

	for p, wasPrimary := range have {
		if _, ok := want[p]; !ok {
			plan.before.Queue(`DELETE FROM org_unit_members WHERE tenant_id = $1::uuid AND org_unit_id = $2::uuid AND principal_id = $3::uuid`,
				tid, p.group.String(), p.principal.String())
			plan.removed++
		} else if wasPrimary && !want[p] {
			plan.before.Queue(`UPDATE org_unit_members SET is_primary = false WHERE tenant_id = $1::uuid AND org_unit_id = $2::uuid AND principal_id = $3::uuid`,
				tid, p.group.String(), p.principal.String())
			plan.written++
		}
	}
	for p, primary := range want {
		wasPrimary, ok := have[p]
		switch {
		case !ok:
			plan.after.Queue(`INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id, is_primary) VALUES ($1::uuid, $2::uuid, $3::uuid, $4)`,
				tid, p.group.String(), p.principal.String(), primary)
			plan.written++
		case primary && !wasPrimary:
			plan.after.Queue(`UPDATE org_unit_members SET is_primary = true WHERE tenant_id = $1::uuid AND org_unit_id = $2::uuid AND principal_id = $3::uuid`,
				tid, p.group.String(), p.principal.String())
			plan.written++
		}
	}
	return plan
}

// orgSubtreeMax is the largest live-unit count under (and including) any unit
// some user belongs to, for a snapshot's units and its per-user unit lists.
func orgSubtreeMax(units []identity.DirectoryGroup, unitsOf map[string][]string, maxDepth int) int {
	parent := make(map[string]string, len(units))
	for _, u := range units {
		parent[u.KeycloakGroupID] = u.ParentKeycloakGroupID
	}
	var withMembers []string
	for _, ks := range unitsOf {
		withMembers = append(withMembers, ks...)
	}
	return subtreeMax(parent, withMembers, maxDepth)
}

// subtreeMax is the largest number of units under (and including) any of
// withMembers, where parent maps every live unit to its parent ("" for a
// root). The walk up from each unit stops at maxDepth, so a cycle cannot loop
// here; CheckOrgUnitChains refuses it.
func subtreeMax(parent map[string]string, withMembers []string, maxDepth int) int {
	if maxDepth <= 0 {
		maxDepth = DefaultMaxOrgUnitDepth
	}
	size := make(map[string]int, len(parent))
	for k0 := range parent {
		k := k0
		for i := 0; i <= maxDepth && k != ""; i++ {
			size[k]++
			k = parent[k]
		}
	}
	best := 0
	for _, k := range withMembers {
		best = max(best, size[k])
	}
	return best
}

// storedOrgSubtreeMax is orgSubtreeMax read from the tables, for a cycle whose
// snapshot is unchanged: a tenant near the cap must show even when its
// directory is quiet. It reads live units and the units that have members.
func storedOrgSubtreeMax(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, maxDepth int) (int, error) {
	tid := tenantID.String()
	rows, err := tx.Query(ctx, `SELECT id::text, coalesce(parent_id::text, '') FROM org_units WHERE tenant_id = $1::uuid AND deleted_at IS NULL`, tid)
	if err != nil {
		return 0, fmt.Errorf("read org units: %w", err)
	}
	parent := map[string]string{}
	for rows.Next() {
		var id, p string
		if err := rows.Scan(&id, &p); err != nil {
			rows.Close()
			return 0, fmt.Errorf("read org units: %w", err)
		}
		parent[id] = p
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("read org units: %w", err)
	}
	mrows, err := tx.Query(ctx, `SELECT DISTINCT org_unit_id::text FROM org_unit_members WHERE tenant_id = $1::uuid`, tid)
	if err != nil {
		return 0, fmt.Errorf("read org unit members: %w", err)
	}
	var withMembers []string
	for mrows.Next() {
		var id string
		if err := mrows.Scan(&id); err != nil {
			mrows.Close()
			return 0, fmt.Errorf("read org unit members: %w", err)
		}
		withMembers = append(withMembers, id)
	}
	mrows.Close()
	if err := mrows.Err(); err != nil {
		return 0, fmt.Errorf("read org unit members: %w", err)
	}
	return subtreeMax(parent, withMembers, maxDepth), nil
}

func uuidPtrEq(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// chainSQL walks up from each start unit, at most $3 + 1 levels, so a cycle
// ends at the bound instead of running forever.
const chainSQL = `WITH RECURSIVE up (start_id, id, parent_id, depth) AS (
    SELECT id, id, parent_id, 1 FROM org_units
    WHERE tenant_id = $1::uuid AND id = ANY($2::uuid[])
  UNION ALL
    SELECT up.start_id, o.id, o.parent_id, up.depth + 1
    FROM up JOIN org_units o ON o.tenant_id = $1::uuid AND o.id = up.parent_id
    WHERE up.depth <= $3::int
)
SELECT coalesce(bool_or(id = start_id AND depth > 1), false), coalesce(max(depth), 0) FROM up`

// CheckOrgUnitChains walks from each of ids up to its root in the tenant's
// current org_units and fails closed: ErrOrgUnitCycle when a unit is its own
// ancestor, ErrOrgUnitDepth when a chain is longer than maxDepth. Call it in
// the tenant's transaction after a parent change, so a refusal rolls it back.
func CheckOrgUnitChains(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, tenantID uuid.UUID, ids []string, maxDepth int) error {
	if len(ids) == 0 {
		return nil
	}
	if maxDepth <= 0 {
		maxDepth = DefaultMaxOrgUnitDepth
	}
	ids = slices.Clone(ids)
	var cycle bool
	var depth int
	if err := q.QueryRow(ctx, chainSQL, tenantID.String(), ids, maxDepth).Scan(&cycle, &depth); err != nil {
		return fmt.Errorf("walk org units: %w", err)
	}
	switch {
	case cycle:
		return ErrOrgUnitCycle
	case depth > maxDepth:
		return ErrOrgUnitDepth
	}
	return nil
}
