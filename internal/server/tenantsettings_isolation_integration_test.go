//go:build integration

// Adversarial tenant-isolation guards for the tenant time-zone route
// (TICK-672, P6-07; Security & Multi-Tenancy review of PR #666). Threat
// model: an attacker holding a valid token for tenant B, aiming at tenant
// A's tenants row or its audit trail. Helpers are in
// tenantsettings_integration_test.go and entitytypes_integration_test.go.

package server_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Tenant B's admin reusing tenant A's Idempotency-Key string gets its own
// write, never a replay of A's stored response (which would disclose A's
// time zone), and A's value is untouched.
func TestTenantTimeZoneIsolationIdempotencyKeyIsPerTenant(t *testing.T) {
	e := newETEnv(t)
	a := e.do(t, http.MethodPut, tzPath, "admin-a", "shared-key", `{"time_zone":"Asia/Tokyo"}`)
	b := e.do(t, http.MethodPut, tzPath, "admin-b", "shared-key", `{"time_zone":"Europe/Berlin"}`)
	if a.Code != http.StatusOK || b.Code != http.StatusOK {
		t.Fatalf("puts = %d %d\n%s\n%s", a.Code, b.Code, a.Body, b.Body)
	}
	if got := tzBody(t, b); got != "Europe/Berlin" {
		t.Errorf("tenant B got %q, want its own Europe/Berlin (a replay of A's response?)", got)
	}
	if za, zb := tzOf(t, e, e.a.id), tzOf(t, e, e.b.id); za != "Asia/Tokyo" || zb != "Europe/Berlin" {
		t.Errorf("A = %q, B = %q; want Asia/Tokyo and Europe/Berlin", za, zb)
	}
	if tzAudit(t, e, e.a.id) != 1 || tzAudit(t, e, e.b.id) != 1 {
		t.Errorf("audit entries A = %d, B = %d; want 1 and 1", tzAudit(t, e, e.a.id), tzAudit(t, e, e.b.id))
	}
}

// C12: actor and source of the audit entry come from the resolved principal.
// A body naming another principal, a system actor or a source is ignored.
func TestTenantTimeZoneIsolationAuditActorFromPrincipalOnly(t *testing.T) {
	e := newETEnv(t)
	body := fmt.Sprintf(`{"time_zone":"America/New_York","tenant_id":%q,"actor":{"principal_id":%q,"system":"importer"},"actor_principal_id":%q,"source":"system"}`,
		e.b.id, e.b.admin.PrincipalID, e.a.user.PrincipalID)
	if r := e.do(t, http.MethodPut, tzPath, "admin-a", "c1", body); r.Code != http.StatusOK {
		t.Fatalf("put = %d %s", r.Code, r.Body)
	}
	var (
		actor     *string
		system    *string
		source    string
		subjectOK bool
	)
	if err := e.super.QueryRow(t.Context(), `SELECT actor_principal_id::text, actor_system, source, subject_id = tenant_id
		FROM audit_log WHERE tenant_id = $1 AND subject_type = 'tenant_setting'`, e.a.id).
		Scan(&actor, &system, &source, &subjectOK); err != nil {
		t.Fatal(err)
	}
	if actor == nil || *actor != e.a.admin.PrincipalID.String() || system != nil || source != "user" || !subjectOK {
		t.Errorf("audit actor=%v system=%v source=%s subject=tenant:%v; want principal %v, no system, user, true",
			actor, system, source, subjectOK, e.a.admin.PrincipalID)
	}
	if n := tzAudit(t, e, e.b.id); n != 0 || tzOf(t, e, e.b.id) != "UTC" {
		t.Errorf("tenant B touched: %d audit entries, zone %q", n, tzOf(t, e, e.b.id))
	}
}

// The lowest layer: inside tenant A's transaction, nava_app cannot change
// tenant B's row (the UPDATE policy hides it: 0 rows), and cannot change any
// tenants column but time_zone, its own row included (column grant: 42501).
// This holds whatever a future handler does.
func TestTenantTimeZoneIsolationDatabaseLayer(t *testing.T) {
	e := newETEnv(t)
	ctx := t.Context()

	err := tenancy.InTenantTx(ctx, e.pool, e.a.id, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE tenants SET time_zone = 'Asia/Tokyo' WHERE id = $1`, e.b.id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 0 {
			return fmt.Errorf("tenant A's transaction updated %d of tenant B's rows", tag.RowsAffected())
		}
		tag, err = tx.Exec(ctx, `UPDATE tenants SET time_zone = 'Asia/Tokyo'`)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("an unfiltered UPDATE touched %d rows, want only A's own", tag.RowsAffected())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if zb := tzOf(t, e, e.b.id); zb != "UTC" {
		t.Errorf("tenant B's zone = %q, want UTC", zb)
	}

	for _, stmt := range []string{
		`UPDATE tenants SET state = 'provisioning' WHERE id = $1`,
		`UPDATE tenants SET id = gen_random_uuid() WHERE id = $1`,
	} {
		err := tenancy.InTenantTx(ctx, e.pool, e.a.id, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, stmt, e.a.id)
			return err
		})
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "42501" {
			t.Errorf("%s: err = %v, want 42501 insufficient_privilege", stmt, err)
		}
	}
}

// A path naming tenant B answers byte for byte like a path naming no tenant:
// it does not tell A whether B exists.
func TestTenantTimeZoneIsolationTenantPathIsNoOracle(t *testing.T) {
	e := newETEnv(t)
	existing := e.do(t, http.MethodPut, "/api/v1/tenant/"+e.b.id.String()+"/settings/time-zone", "admin-a", "p1", `{"time_zone":"Asia/Tokyo"}`)
	unknown := e.do(t, http.MethodPut, "/api/v1/tenant/00000000-0000-4000-8000-000000000000/settings/time-zone", "admin-a", "p2", `{"time_zone":"Asia/Tokyo"}`)
	if existing.Code != http.StatusNotFound || existing.Code != unknown.Code {
		t.Fatalf("codes = %d, %d; want 404, 404", existing.Code, unknown.Code)
	}
	strip := func(b []byte) []byte { // the instance names the request, not the tenant
		if i := bytes.Index(b, []byte(`"instance"`)); i >= 0 {
			return b[:i]
		}
		return b
	}
	if !bytes.Equal(strip(existing.Body.Bytes()), strip(unknown.Body.Bytes())) {
		t.Errorf("bodies differ:\n%s\n%s", existing.Body, unknown.Body)
	}
	if zb := tzOf(t, e, e.b.id); zb != "UTC" {
		t.Errorf("tenant B's zone = %q, want UTC", zb)
	}
}
