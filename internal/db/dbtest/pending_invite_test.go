//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// T252a (TICK-254): pending_invite under FORCE RLS. nava_onboard invites and
// re-invites, nava_groupsync consumes, nobody deletes, and a consumed invite
// stays consumed.

const (
	inviteEmail  = "admin@tenant.test"
	insertInvite = "INSERT INTO pending_invite (tenant_id, email, expires_at) VALUES ($1, $2, now() + interval '7 days')"
	// The re-invite upsert TICK-252 runs: it extends a pending invite and
	// leaves a consumed one alone.
	reinvite = `INSERT INTO pending_invite (tenant_id, email, expires_at) VALUES ($1, $2, now() + interval '30 days')
		ON CONFLICT (tenant_id, email) DO UPDATE SET expires_at = EXCLUDED.expires_at
		WHERE pending_invite.consumed_at IS NULL`
	// The consume statement of condition 2d.
	consumeInvite = `UPDATE pending_invite SET consumed_at = now()
		WHERE tenant_id = $1 AND email = $2 AND consumed_at IS NULL AND expires_at > now()
		RETURNING tenant_id`
)

// seedInvites writes, as the superuser, one pending invite for inviteEmail
// in each tenant.
func seedInvites(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	for _, id := range tenants {
		exec(t, super, insertInvite, id, inviteEmail)
	}
}

// Security's negative test 10 (table part): with tenant B set, neither role
// reads, inserts, re-invites, extends or consumes tenant A's invite; with
// app.tenant_id unset or empty, every command sees and changes nothing.
func TestT252aInvitesInvisibleAcrossTenants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedInvites(t, super, tenantA, tenantB)

	for _, role := range []string{"nava_onboard", "nava_groupsync"} {
		conn := asRole(t, role)
		for _, mode := range []string{"B", "empty", "unset"} {
			var own, other int
			err := inTenantTx(conn, "", func(tx pgx.Tx) error {
				if mode != "unset" {
					v := map[string]string{"B": tenantB, "empty": ""}[mode]
					if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", v); err != nil {
						return err
					}
				}
				return tx.QueryRow(context.Background(),
					"SELECT count(*) FILTER (WHERE tenant_id = $1), count(*) FILTER (WHERE tenant_id <> $1) FROM pending_invite WHERE email = $2",
					tenantB, inviteEmail).Scan(&own, &other)
			})
			if err != nil {
				t.Fatalf("%s reads pending_invite (tenant %s): %v", role, mode, err)
			}
			wantOwn := 0
			if mode == "B" {
				wantOwn = 1
			}
			if own != wantOwn || other != 0 {
				t.Errorf("%s reads pending_invite with tenant %s: own %d, other tenants %d; want %d, 0", role, mode, own, other, wantOwn)
			}
		}
	}

	onboard, sync := asRole(t, "nava_onboard"), asRole(t, "nava_groupsync")
	for _, ctxTenant := range []string{tenantB, ""} {
		// Inserts and upserts aimed at A are refused by WITH CHECK.
		_, err := execIn(onboard, ctxTenant, insertInvite, tenantA, "new@tenant.test")
		wantSQLState(t, "nava_onboard in tenant "+ctxTenant+" invites for A", err, "42501")
		_, err = execIn(onboard, ctxTenant, reinvite, tenantA, inviteEmail)
		wantSQLState(t, "nava_onboard in tenant "+ctxTenant+" re-invites A's invitee", err, "42501")
		// Updates aimed at A reach no row.
		for _, u := range []struct {
			conn *pgx.Conn
			what string
			sql  string
		}{
			{onboard, "nava_onboard extends A's invites", "UPDATE pending_invite SET expires_at = now() + interval '30 days' WHERE tenant_id = $1 AND email = $2"},
			{sync, "nava_groupsync consumes A's invite", consumeInvite},
		} {
			n, err := execIn(u.conn, ctxTenant, u.sql, tenantA, inviteEmail)
			if err != nil || n != 0 {
				t.Errorf("%s from tenant %q: %d rows, %v; want 0, nil", u.what, ctxTenant, n, err)
			}
		}
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pending_invite
		WHERE tenant_id = $1 AND consumed_at IS NULL AND expires_at < now() + interval '8 days'`, tenantA); got != "1" {
		t.Errorf("tenant A's invite changed: %s unchanged pending rows, want 1", got)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM pending_invite"); got != "2" {
		t.Errorf("pending_invite holds %s rows, want the 2 seeded", got)
	}
}

// The lifecycle, in a provisioning tenant as onboarding runs it: invite,
// re-invite extends it, consume once, a second consume and a re-invite of
// the consumed invite change nothing, and an expired invite cannot be
// consumed. Once the tenant is active, nava_onboard can neither invite nor
// extend an invite there, and nava_groupsync still consumes.
func TestT252aInviteLifecycle(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantX})
	onboard, sync := asRole(t, "nava_onboard"), asRole(t, "nava_groupsync")

	if n, err := execIn(onboard, tenantX, reinvite, tenantX, inviteEmail); err != nil || n != 1 {
		t.Fatalf("first invite: %d rows, %v; want 1, nil", n, err)
	}
	if n, err := execIn(onboard, tenantX, reinvite, tenantX, inviteEmail); err != nil || n != 1 {
		t.Fatalf("re-invite a pending invite: %d rows, %v; want 1, nil", n, err)
	}
	if got := queryString(t, super, "SELECT (expires_at > now() + interval '29 days')::text FROM pending_invite WHERE tenant_id = $1", tenantX); got != "true" {
		t.Errorf("re-invite did not extend expires_at")
	}

	if n, err := execIn(sync, tenantX, consumeInvite, tenantX, inviteEmail); err != nil || n != 1 {
		t.Fatalf("consume: %d rows, %v; want 1, nil", n, err)
	}
	if n, err := execIn(sync, tenantX, consumeInvite, tenantX, inviteEmail); err != nil || n != 0 {
		t.Errorf("consume twice: %d rows, %v; want 0, nil", n, err)
	}
	if n, err := execIn(onboard, tenantX, reinvite, tenantX, inviteEmail); err != nil || n != 0 {
		t.Errorf("re-invite a consumed invite: %d rows, %v; want 0, nil", n, err)
	}
	if got := queryString(t, super, "SELECT (consumed_at IS NOT NULL)::text FROM pending_invite WHERE tenant_id = $1", tenantX); got != "true" {
		t.Errorf("the consumed invite was reopened")
	}

	exec(t, super, `INSERT INTO pending_invite (tenant_id, email, expires_at, created_at)
		VALUES ($1, 'late@tenant.test', now() - interval '1 day', now() - interval '8 days')`, tenantX)
	if n, err := execIn(sync, tenantX, consumeInvite, tenantX, "late@tenant.test"); err != nil || n != 0 {
		t.Errorf("consume an expired invite: %d rows, %v; want 0, nil", n, err)
	}

	// Activated: onboarding's writes stop, group sync's do not.
	exec(t, super, insertInvite, tenantX, "open@tenant.test")
	exec(t, super, "UPDATE tenants SET state = 'active' WHERE id = $1", tenantX)
	_, err := execIn(onboard, tenantX, insertInvite, tenantX, "new@tenant.test")
	wantSQLState(t, "invite in the active tenant", err, "42501")
	_, err = execIn(onboard, tenantX, reinvite, tenantX, "new@tenant.test")
	wantSQLState(t, "re-invite in the active tenant", err, "42501")
	if n, err := execIn(onboard, tenantX, "UPDATE pending_invite SET expires_at = now() + interval '30 days' WHERE tenant_id = $1", tenantX); err != nil || n != 0 {
		t.Errorf("extend invites in the active tenant: %d rows, %v; want 0, nil", n, err)
	}
	if n, err := execIn(sync, tenantX, consumeInvite, tenantX, "open@tenant.test"); err != nil || n != 1 {
		t.Errorf("consume in the active tenant: %d rows, %v; want 1, nil", n, err)
	}
}

// The grants are exactly db/grants.yaml's: no role deletes, nava_onboard
// cannot touch consumed_at (so a consumed invite cannot be reopened, even by
// the upsert form that names it), nava_groupsync writes only consumed_at,
// and no other role reaches the table.
func TestT252aGrantsAreNarrow(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedInvites(t, super, tenantA)
	exec(t, super, "UPDATE pending_invite SET consumed_at = now() WHERE tenant_id = $1", tenantA)

	onboard, sync := asRole(t, "nava_onboard"), asRole(t, "nava_groupsync")
	for _, d := range []struct {
		conn *pgx.Conn
		what string
		sql  string
	}{
		{onboard, "nava_onboard deletes", "DELETE FROM pending_invite WHERE tenant_id = $1"},
		{sync, "nava_groupsync deletes", "DELETE FROM pending_invite WHERE tenant_id = $1"},
		{onboard, "nava_onboard reopens a consumed invite", "UPDATE pending_invite SET consumed_at = NULL WHERE tenant_id = $1"},
		{onboard, "nava_onboard consumes an invite", "UPDATE pending_invite SET consumed_at = now() WHERE tenant_id = $1"},
		{onboard, "nava_onboard upserts with consumed_at = NULL",
			`INSERT INTO pending_invite (tenant_id, email, expires_at) VALUES ($1, 'admin@tenant.test', now() + interval '1 day')
			 ON CONFLICT (tenant_id, email) DO UPDATE SET expires_at = EXCLUDED.expires_at, consumed_at = NULL`},
		{onboard, "nava_onboard writes created_at",
			"INSERT INTO pending_invite (tenant_id, email, expires_at, created_at) VALUES ($1, 'x@tenant.test', now() + interval '1 day', now())"},
		{sync, "nava_groupsync invites", insertInvite},
		{sync, "nava_groupsync extends an invite", "UPDATE pending_invite SET expires_at = now() + interval '30 days' WHERE tenant_id = $1"},
		{sync, "nava_groupsync re-keys an invite", "UPDATE pending_invite SET email = 'y@tenant.test' WHERE tenant_id = $1"},
	} {
		args := []any{tenantA}
		if d.sql == insertInvite {
			args = append(args, "z@tenant.test")
		}
		_, err := execIn(d.conn, tenantA, d.sql, args...)
		wantSQLState(t, d.what, err, "42501")
	}
	for _, role := range []string{"nava_app", "nava_worker", "nava_relay", "nava_purge"} {
		_, err := execIn(asRole(t, role), tenantA, "SELECT count(*) FROM pending_invite")
		wantSQLState(t, role+" reads pending_invite", err, "42501")
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM pending_invite WHERE consumed_at IS NOT NULL"); got != "1" {
		t.Errorf("pending_invite holds %s consumed rows, want the 1 seeded", got)
	}
}

// The CHECKs refuse an email the normalized lookup could never match and an
// invite that expires before it is created.
func TestT252aInviteChecks(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantX})
	onboard := asRole(t, "nava_onboard")
	for _, e := range []string{"Admin@tenant.test", " admin@tenant.test", "admin@tenant.test ", "admin", ""} {
		_, err := execIn(onboard, tenantX, insertInvite, tenantX, e)
		wantSQLState(t, "invite "+e, err, "23514")
	}
	_, err := execIn(onboard, tenantX, "INSERT INTO pending_invite (tenant_id, email, expires_at) VALUES ($1, 'a@tenant.test', now())", tenantX)
	wantSQLState(t, "invite expiring at creation", err, "23514")
	if _, err := execIn(onboard, tenantX, insertInvite, tenantX, "admin@tenant.test"); err != nil {
		t.Errorf("control: invite admin@tenant.test: %v", err)
	}
}
