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
	PrincipalsWritten, PrincipalsDeleted int
	GroupsWritten, GroupsDeleted         int
	OrgUnitsWritten, OrgUnitsDeleted     int
	// Ignored is the number of snapshot groups and org units whose path is
	// outside the subtree they may be written to.
	Ignored int
}

// ---- path rules ------------------------------------------------------------

func tenantPrefix(tenantID uuid.UUID) string { return "/t/" + tenantID.String() + "/" }

// isOrgUnitPath: strictly below /t/<tenant_id>/ou/.
func isOrgUnitPath(tenantID uuid.UUID, path string) bool {
	rest, ok := strings.CutPrefix(path, tenantPrefix(tenantID)+"ou/")
	return ok && rest != ""
}

// isGroupPath: below /t/<tenant_id>/ and not the ou container or below it.
func isGroupPath(tenantID uuid.UUID, path string) bool {
	rest, ok := strings.CutPrefix(path, tenantPrefix(tenantID))
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
	if err := applyOrgUnits(ctx, tx, tenantID, units, maxDepth, &st); err != nil {
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

type linkJSON struct {
	IdentityProvider string `json:"identityProvider"`
	UserID           string `json:"userId"`
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
// Only links whose IdP alias the tenant binds are kept: a link to another
// tenant's IdP carries that tenant's id and the external user id.
func linksJSON(links []identity.FederatedLink, bound map[string]bool) ([]byte, error) {
	out := make([]linkJSON, 0, len(links))
	for _, l := range links {
		if !bound[l.IdPAlias] {
			continue
		}
		out = append(out, linkJSON{IdentityProvider: l.IdPAlias, UserID: l.ExternalUserID})
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
}

// applyPrincipals returns the Keycloak user id -> principal id map.
func applyPrincipals(ctx context.Context, tx pgx.Tx, tid string, users []identity.DirectoryUser, bound map[string]bool, now time.Time, st *stats) (map[string]uuid.UUID, error) {
	rows, err := tx.Query(ctx, `SELECT id, keycloak_user_id, email, enabled, federated_links FROM principals WHERE tenant_id = $1::uuid`, tid)
	if err != nil {
		return nil, fmt.Errorf("read principals: %w", err)
	}
	have := map[string]principalRow{}
	for rows.Next() {
		var kc string
		var r principalRow
		if err := rows.Scan(&r.id, &kc, &r.email, &r.enabled, &r.links); err != nil {
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
			id := uuid.New()
			ids[u.KeycloakUserID] = id
			b.Queue(`INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
				VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::jsonb, $7)`, tid, id.String(), u.KeycloakUserID, email, u.Enabled, string(links), now)
			st.PrincipalsWritten++
		default:
			ids[u.KeycloakUserID] = cur.id
			if ptrEq(cur.email, email) && cur.enabled == u.Enabled && sameJSON(cur.links, links) {
				continue
			}
			b.Queue(`UPDATE principals SET email = $3, enabled = $4, federated_links = $5::jsonb, synced_at = $6
				WHERE tenant_id = $1::uuid AND id = $2::uuid`, tid, cur.id.String(), email, u.Enabled, string(links), now)
			st.PrincipalsWritten++
		}
	}
	// Removing a principal removes its memberships (ON DELETE CASCADE).
	var gone []string
	for kc, r := range have {
		if !want[kc] {
			gone = append(gone, r.id.String())
		}
	}
	if len(gone) > 0 {
		b.Queue(`DELETE FROM principals WHERE tenant_id = $1::uuid AND id = ANY($2::uuid[])`, tid, gone)
		st.PrincipalsDeleted = len(gone)
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
	rows, err := tx.Query(ctx, `SELECT id, keycloak_group_id, path FROM groups WHERE tenant_id = $1::uuid`, tid)
	if err != nil {
		return fmt.Errorf("read groups: %w", err)
	}
	type groupRow struct {
		id   uuid.UUID
		path string
	}
	have := map[string]groupRow{}
	for rows.Next() {
		var kc string
		var r groupRow
		if err := rows.Scan(&r.id, &kc, &r.path); err != nil {
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
		want[g.KeycloakGroupID] = true
		cur, ok := have[g.KeycloakGroupID]
		id := cur.id
		switch {
		case !ok:
			id = uuid.New()
			b.Queue(`INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, $2::uuid, $3, $4)`,
				tid, id.String(), g.KeycloakGroupID, g.Path)
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

	b = &pgx.Batch{}
	for p := range haveMembers {
		if !wantMembers[p] {
			b.Queue(`DELETE FROM group_members WHERE tenant_id = $1::uuid AND group_id = $2::uuid AND principal_id = $3::uuid`,
				tid, p.group.String(), p.principal.String())
		}
	}
	for p := range wantMembers {
		if !haveMembers[p] {
			b.Queue(`INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1::uuid, $2::uuid, $3::uuid)`,
				tid, p.group.String(), p.principal.String())
		}
	}
	var gone []string
	for kc, r := range have {
		if !want[kc] {
			gone = append(gone, r.id.String())
		}
	}
	if len(gone) > 0 {
		b.Queue(`DELETE FROM groups WHERE tenant_id = $1::uuid AND id = ANY($2::uuid[])`, tid, gone)
		st.GroupsDeleted = len(gone)
	}
	if err := send(ctx, tx, b); err != nil {
		return fmt.Errorf("write group members: %w", err)
	}
	return nil
}

type unitRow struct {
	id     uuid.UUID
	parent *uuid.UUID
	name   string
}

// applyOrgUnits writes the tree. New units go in as roots, then parents are
// set, then removed units go; the walk that proves the result has no cycle
// and is not deeper than maxDepth runs before anything commits.
func applyOrgUnits(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, units []identity.DirectoryGroup, maxDepth int, st *stats) error {
	tid := tenantID.String()
	rows, err := tx.Query(ctx, `SELECT id, parent_id, keycloak_group_id, name FROM org_units WHERE tenant_id = $1::uuid`, tid)
	if err != nil {
		return fmt.Errorf("read org units: %w", err)
	}
	have := map[string]unitRow{}
	for rows.Next() {
		var kc string
		var r unitRow
		if err := rows.Scan(&r.id, &r.parent, &kc, &r.name); err != nil {
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
		if cur, ok := have[u.KeycloakGroupID]; ok {
			idOf[u.KeycloakGroupID] = cur.id
		} else {
			idOf[u.KeycloakGroupID] = uuid.New()
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
			if parent != nil {
				upd.Queue(`UPDATE org_units SET parent_id = $3::uuid WHERE tenant_id = $1::uuid AND id = $2::uuid`, tid, id.String(), parent.String())
				moved = append(moved, id.String())
			}
			continue
		}
		parentChanged := !uuidPtrEq(cur.parent, parent)
		if !parentChanged && cur.name == u.Name {
			continue
		}
		var pArg *string
		if parent != nil {
			s := parent.String()
			pArg = &s
		}
		upd.Queue(`UPDATE org_units SET parent_id = $3::uuid, name = $4 WHERE tenant_id = $1::uuid AND id = $2::uuid`, tid, id.String(), pArg, u.Name)
		st.OrgUnitsWritten++
		if parentChanged {
			moved = append(moved, id.String())
		}
	}
	var gone []string
	for kc, r := range have {
		if _, ok := idOf[kc]; !ok {
			gone = append(gone, r.id.String())
		}
	}
	del := &pgx.Batch{}
	if len(gone) > 0 {
		// One statement: the self-FK is NO ACTION, checked at its end.
		del.Queue(`DELETE FROM org_units WHERE tenant_id = $1::uuid AND id = ANY($2::uuid[])`, tid, gone)
		st.OrgUnitsDeleted = len(gone)
	}
	for _, b := range []*pgx.Batch{ins, upd, del} {
		if err := send(ctx, tx, b); err != nil {
			return fmt.Errorf("write org units: %w", err)
		}
	}
	return CheckOrgUnitChains(ctx, tx, tenantID, moved, maxDepth)
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
