package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// s22s29AttackPaths is the TICK-753 adversarial suite (S22 P7-13 and S29
// P10-11, which TICK-753 absorbed as TICK-926): field removal, the
// tenant_limits quota raise, the retired JSONB scan (ADR-006 D6, D11, D12,
// R3) and the schema-version backfill's tables, routes, filters and workflow
// start (ADR-009 D2, D10, D12, D15; RFC-021 §10 CI-3).
//
// Threat model: tenant A holds a valid token (or tenant A's worker task
// token), and asks for, writes, steers or learns from tenant B's field
// index, quota or backfill job. Every row has a positive control, so a dead
// route or a broken fixture fails instead of passing as a denial, and
// deletes what it writes.
//
// The rows were written from the contracts on 2026-11-26 (day 1 of the S22
// and S29 consolidation). A row whose code has not landed names its task in
// Pending; the PR that lands that task removes the Pending field (or the
// TICK-753 follow-up does), and the row must then pass. A row with no
// test body yet fails loudly if Pending is removed before the body is
// written (s22s29NotWritten). The broken variants each row was shown red
// against are listed in docs/reviews/tick-753-s22-s29-isolation-attacks.md.
//
// The Temporal halves (schema.backfill start-only, field_index.remove.run
// child-only) are temporal-server/authz/adversarial_s22_s29_test.go.
func s22s29AttackPaths() []Path {
	s21 := func(c, name string, fn func(context.Context, *etIso, *s21Fixture) error) Path {
		return Path{Name: "S22 attack " + name, Case: c, Role: "nava_app", DB: func(ctx context.Context, cl Cluster) error {
			return withS21Fixture(ctx, cl, fn)
		}}
	}
	return []Path{
		s21("CI-2", "1: removal's statements (flip, batched delete, NOT EXISTS, registry delete), run as nava_app and nava_worker in tenant A with tenant B's field id, reach no row; a publish dropping a flag on tenant B's key is the unknown key's 404", s22RemoveAcrossTenants),
		{Name: "S22 attack 1b: a field_index.remove outbox row of tenant A naming tenant B's field id starts a removal that deletes nothing of tenant B and leaves its registry row", Case: "CI-2", Role: "nava_worker",
			Pending: "TICK-750", DB: s22s29NotWritten("1b drives TICK-750's FieldIndexRemove through the coordinator, as row 3b drives FieldIndexBackfill")},
		s21("CI-2", "2: a field in removing takes no new side row from any records writer (every manifest role with INSERT or UPDATE on records), so the removal's NOT EXISTS holds and the registry delete passes the FK; tenant B's removal does not stop tenant A's indexing", s22RemovingTakesNoWrites),
		{Name: "S22 attack 2b: R3 race, writers of every records-writer role across the flip and the batched deletes of a real FieldIndexRemove leave 0 side rows and no registry row; with the barrier disabled the row can fail", Case: "CI-5", Role: "nava_worker",
			Pending: "TICK-750", DB: s22s29NotWritten("2b needs TICK-750's FieldIndexRemove and its barrier test seam")},
		{Name: "S22 attack 3: tenant_limits, no tenant role writes its own or another tenant's row; a tenant role reads its own row only, and nothing with no tenant set; only the manifest's operator role writes, and not across tenants", Case: "CI-2", Role: "nava_app",
			Pending: "TICK-916", DB: s22TenantLimits},
		{Name: "S22 attack 3b: the operator's tenant_limits change writes one audit entry naming the operator and the reason; a tenant admin's attempt is 403 and writes nothing", Case: "CI-2", Role: "nava_app",
			Pending: "TICK-916", DB: s22s29NotWritten("3b drives TICK-751's operator route, whose path TICK-916 fixes")},
		{Name: "S22 attack 4: with the legacy scan retired, no non-test Go code outside the allow-list evaluates a JSONB operator on records.data for a filter, and DataFilterTimeout and LegacyScanFields are gone", Case: "CI-2", Role: "nava_app",
			Pending: "TICK-752", Check: s22NoJSONBFilter},
		{Name: "S29 attack 1: the three backfill tables, as nava_app and nava_worker in tenant A, show no tenant B row, refuse a tenant B insert, and nava_app writes nothing; with no tenant set each reads zero rows; failures has no value column; tenant B's active job does not hold tenant A's admission (CI-3 (5), (6))", Case: "CI-2", Role: "nava_app",
			Pending: "TICK-916", DB: s29Tables},
		{Name: "S29 attack 2: getEntityType, .../backfill/failures and a default-adding publish on tenant B's key are the unknown key's 404 for tenant A and write nothing; a plain user's failures read is the same 404; tenant A's backfill and failures show tenant A's job only under a shared key; the publish's job and outbox rows are tenant A's, with no tenant in the payload (CI-3 (7))", Case: "CI-2", Role: "nava_app",
			Pending: "TICK-919", DB: func(ctx context.Context, cl Cluster) error { return withS29Fixture(ctx, cl, s29Routes) }},
		{Name: "S29 attack 3: with open backfill windows in both tenants under one key, a filter on the defaulted field never returns or counts a tenant B record, in the OR shape or the UNION ALL shape, and the UNION ALL arm never returns an ACL-hidden record", Case: "CI-8", Role: "nava_app",
			Pending: "TICK-919", DB: func(ctx context.Context, cl Cluster) error { return withS29Fixture(ctx, cl, s29Filters) }},
		{Name: "S29 attack 4: pipeline.Write(OpMaterialize) with a non-empty patch, a non-system source or another actor_system is refused with a typed code and writes nothing (CI-3 (4))", Case: "CI-2", Role: "nava_worker",
			Pending: "TICK-916", DB: s22s29NotWritten("4 needs TICK-917's OpMaterialize request type")},
		{Name: "S29 attack 5: a schema.backfill job t:A:… changes zero tenant B rows when tenant B has the same type key, its own job and pending rows (CI-3 (3)); its history carries no field value", Case: "CI-2", Role: "nava_worker",
			Pending: "TICK-920", DB: s22s29NotWritten("5 drives TICK-920's schema.backfill workflow on the Temporal test server, as row 3b does")},
		{Name: "S29 attack 6: the relay refuses a schema.backfill outbox row whose payload carries a tenant_id, and builds the workflow id from outbox.tenant_id only", Case: "CI-2", Role: "nava_relay",
			Pending: "TICK-920", DB: s22s29NotWritten("6 drives TICK-918's relay dispatch")},
	}
}

// s22s29NotWritten is the body of a row whose test needs Go code that has
// not landed: removing Pending without writing the row fails it.
func s22s29NotWritten(what string) func(context.Context, Cluster) error {
	return func(context.Context, Cluster) error {
		return errors.New("row not written yet: " + what + " (TICK-753)")
	}
}

// withS21Fixture runs fn on the S21 fixture (s21Setup: iso_et_s21 in both
// tenants under one key, every field flagged and ready, records in both),
// cleaning up before and after, as fieldIndexAttackPaths does.
func withS21Fixture(ctx context.Context, cl Cluster, fn func(context.Context, *etIso, *s21Fixture) error) error {
	e, err := newETIso(ctx, cl)
	if err != nil {
		return err
	}
	defer e.stop()
	clean := func() error {
		if err := s21Cleanup(ctx, e); err != nil {
			return err
		}
		return e.cleanup()
	}
	if err := clean(); err != nil {
		return err
	}
	fx, err := s21Setup(ctx, e)
	if err == nil {
		err = fn(ctx, e, fx)
	}
	if cerr := clean(); err == nil {
		err = cerr
	}
	return err
}

// s22SideCount counts tenant's side rows of field in table, past RLS.
func s22SideCount(ctx context.Context, e *etIso, table, tenant, field string) (int, error) {
	var n int
	err := e.super.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE tenant_id = $1 AND field_id = $2`, tenant, field).Scan(&n)
	return n, err
}

// s22Step is one statement of D11's removal: q with args for (tenant,
// field), the rows it must reach on tenant A's own field, and the rows (0)
// or the SQLSTATE class it must get on tenant B's.
type s22Step struct {
	name  string
	q     string
	own   int64
	class string // "" when tenant B's field must reach 0 rows
}

// Row 1. Removal (ADR-006 D11) run by tenant A against tenant B's field.
// First the publish: tenant A dropping a flag from tenant B's only key is the
// unknown key's 404 and moves no registry row. Then every D11 statement, in
// the order FieldIndexRemove runs them, as nava_app (the publish's flip) and
// nava_worker (the workflow), in tenant A's transaction: on tenant A's own
// note field (the control; rolled back) each reaches its rows, and on tenant
// B's note field each reaches none. NOT EXISTS on tenant B's field is true
// in tenant A (RLS hides tenant B's rows), which is exactly why the registry
// delete must also reach 0 rows: it does, because RLS hides tenant B's
// registry row too. Tenant B's registry and side rows do not move.
func s22RemoveAcrossTenants(ctx context.Context, e *etIso, _ *s21Fixture) error {
	unflag := fmt.Sprintf(`{"definition": %s}`, s21Definition(s21FlagsA[1:]...))
	if err := e.sameNotFound(ctx, http.MethodPost, unflag, "/api/v1/entity-types/incident/versions", "/api/v1/entity-types/iso_et_none/versions"); err != nil {
		return err
	}
	noteA, err := s21FieldID(ctx, e, TenantA, "note")
	if err != nil {
		return err
	}
	noteB, err := s21FieldID(ctx, e, TenantB, "note")
	if err != nil {
		return err
	}
	regB, err := s21Registry(ctx, e, TenantB)
	if err != nil {
		return err
	}
	sideB, err := s22SideCount(ctx, e, "rfv_text", TenantB, noteB)
	if err != nil || sideB != 2 {
		return fmt.Errorf("control: tenant B's note side rows = %d (%w), want 2", sideB, err)
	}
	// $1 is the tenant the statement names, $2 the field id. The own run
	// names tenant A and tenant A's field; the attack names tenant B's field,
	// once with tenant A (what a crafted payload can give a tenant A
	// workflow) and once with tenant B (a workflow that trusted a payload).
	type roleSteps struct {
		role  string
		steps []s22Step
	}
	plan := []roleSteps{
		{"nava_app", []s22Step{
			{"flip the field to removing", `UPDATE record_index_fields SET state = 'removing', updated_at = now() WHERE tenant_id = $1 AND id = $2`, 1, ""},
			{"flip by id alone", `UPDATE record_index_fields SET state = 'removing', updated_at = now() WHERE id = $2 AND $1::uuid IS NOT NULL`, 1, ""},
			{"delete the field's side rows", `DELETE FROM rfv_text WHERE tenant_id = $1 AND field_id = $2`, 2, ""},
		}},
		{"nava_worker", []s22Step{
			{"lock the registry row", `SELECT state FROM record_index_fields WHERE tenant_id = $1 AND id = $2 FOR UPDATE`, 1, ""},
			{"batched delete of 1,000", `DELETE FROM rfv_text WHERE (tenant_id, record_id, field_id, ord) IN
				(SELECT tenant_id, record_id, field_id, ord FROM rfv_text WHERE tenant_id = $1 AND field_id = $2 LIMIT 1000)`, 2, ""},
			{"batched delete by field id alone", `DELETE FROM rfv_text WHERE field_id = $2 AND $1::uuid IS NOT NULL`, 0, ""},
			{"NOT EXISTS on the field's side rows", `SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM rfv_text WHERE tenant_id = $1 AND field_id = $2)`, 1, ""},
			{"registry delete", `DELETE FROM record_index_fields WHERE tenant_id = $1 AND id = $2`, 1, ""},
		}},
	}
	for _, p := range plan {
		pool, err := e.cluster.Pool(p.role)
		if err != nil {
			return err
		}
		// Control: tenant A's own field, the statements in order in one
		// transaction, rolled back.
		if err := inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
			for _, s := range p.steps {
				n, err := execOne(ctx, tx, s.q, TenantA, noteA)
				if err != nil || n != s.own {
					return fmt.Errorf("control: %s %s on tenant A's own field: %d rows, %w; want %d", p.role, s.name, n, err, s.own)
				}
			}
			return nil
		}); err != nil {
			return err
		}
		for _, tenant := range []string{TenantA, TenantB} {
			if err := inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
				for _, s := range p.steps {
					n, err := execOne(ctx, tx, s.q, tenant, noteB)
					want := int64(0)
					if strings.HasPrefix(s.name, "NOT EXISTS") {
						want = 1 // true: tenant B's rows are invisible, see the registry delete
					}
					if err != nil || n != want {
						return fmt.Errorf("%s in tenant A naming tenant %s: %s on tenant B's field: %d rows, %w; want %d", p.role, tenant, s.name, n, err, want)
					}
				}
				return nil
			}); err != nil {
				return err
			}
		}
	}
	if got, err := s21Registry(ctx, e, TenantB); err != nil || got != regB {
		return fmt.Errorf("the removal statements moved tenant B's registry (%w):\n%s\n%s", err, regB, got)
	}
	if n, err := s22SideCount(ctx, e, "rfv_text", TenantB, noteB); err != nil || n != sideB {
		return fmt.Errorf("the removal statements moved tenant B's note side rows: %d -> %d (%w)", sideB, n, err)
	}
	return nil
}

// s22RecordWriters are the runtime roles the grant manifest lets INSERT or
// UPDATE records: every one of them fires the D4 trigger.
func s22RecordWriters() ([]string, error) {
	roles, err := s21RuntimeRoles()
	if err != nil {
		return nil, err
	}
	m, err := grants.Parse(db.Grants())
	if err != nil {
		return nil, err
	}
	var out []string
	for _, role := range roles {
		recs := m.Roles[role].Tables["records"]
		_, ins := recs["INSERT"]
		_, upd := recs["UPDATE"]
		if ins || upd {
			out = append(out, role)
		}
	}
	if !slices.Contains(out, "nava_app") || !slices.Contains(out, "nava_worker") {
		return nil, fmt.Errorf("control: the records writers %v miss nava_app or nava_worker", out)
	}
	return out, nil
}

// Row 2. The R3 orphan, DB half: once publish has flipped tenant A's note to
// removing (D11 step 1), no records writer adds a side row for it. Each
// writer role, in tenant A, rewrites a record's note and qty in one
// statement: the note (removing) gets no row with the new value, the qty
// (ready) does (the control: the trigger fired). In the same transaction
// D11 steps 3 to 5 then empty the note, NOT EXISTS holds, and the registry
// delete (nava_worker, the role that runs it) reaches the row instead of
// the D3 FK refusing it: nothing a writer did leaves an orphan. Then tenant
// B's note goes to removing and tenant A's back to ready: tenant A's writes
// are indexed again, so tenant B's removal neither stops nor steers tenant
// A's index. The race half (a writer whose transaction spans the flip) is
// row 2b, on the barrier.
func s22RemovingTakesNoWrites(ctx context.Context, e *etIso, fx *s21Fixture) error {
	writers, err := s22RecordWriters()
	if err != nil {
		return err
	}
	noteA, err := s21FieldID(ctx, e, TenantA, "note")
	if err != nil {
		return err
	}
	qtyA, err := s21FieldID(ctx, e, TenantA, "qty")
	if err != nil {
		return err
	}
	noteB, err := s21FieldID(ctx, e, TenantB, "note")
	if err != nil {
		return err
	}
	if err := s21SetState(ctx, e, TenantA, "note", "removing", 100); err != nil {
		return err
	}
	const write = `UPDATE records SET data = data || jsonb_build_object('note', $2::text, 'qty', $3::int) WHERE tenant_id = $1 AND id = $4`
	count := func(tx pgx.Tx, q string, args ...any) (int, error) {
		var n int
		err := tx.QueryRow(ctx, q, args...).Scan(&n)
		return n, err
	}
	for i, role := range writers {
		pool, err := e.cluster.Pool(role)
		if err != nil {
			return err
		}
		value, qty := "s22-removing-"+role, 7000+i
		if err := inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
			if n, err := execOne(ctx, tx, write, TenantA, value, qty, fx.desk); err != nil || n != 1 {
				return fmt.Errorf("control: %s rewriting tenant A's record: %d rows, %w", role, n, err)
			}
			if n, err := count(tx, `SELECT count(*) FROM rfv_num WHERE tenant_id = $1 AND field_id = $2 AND record_id = $3 AND value = $4`, TenantA, qtyA, fx.desk, qty); err != nil || n != 1 {
				return fmt.Errorf("control: %s's write indexed tenant A's ready qty %d times (%w), want 1", role, n, err)
			}
			if n, err := count(tx, `SELECT count(*) FROM rfv_text WHERE tenant_id = $1 AND field_id = $2 AND value = $3`, TenantA, noteA, value); err != nil || n != 0 {
				return fmt.Errorf("%s's write added %d side rows (%w) to tenant A's note while it is removing, want 0", role, n, err)
			}
			if role != "nava_worker" {
				return nil
			}
			// D11 steps 3 to 5, as the workflow's role, in the writer's
			// transaction.
			if n, err := execOne(ctx, tx, `DELETE FROM rfv_text WHERE tenant_id = $1 AND field_id = $2`, TenantA, noteA); err != nil || n == 0 {
				return fmt.Errorf("control: batched delete of tenant A's note: %d rows, %w", n, err)
			}
			if n, err := count(tx, `SELECT count(*) FROM rfv_text WHERE tenant_id = $1 AND field_id = $2`, TenantA, noteA); err != nil || n != 0 {
				return fmt.Errorf("orphan: %d side rows of tenant A's removing note after the batched delete (%w)", n, err)
			}
			if n, err := execOne(ctx, tx, `DELETE FROM record_index_fields WHERE tenant_id = $1 AND id = $2`, TenantA, noteA); err != nil || n != 1 {
				return fmt.Errorf("registry delete of tenant A's note after the batched delete: %d rows, %w; want 1 (an orphan makes the D3 FK refuse it)", n, err)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	// Tenant B removes its note; tenant A's is ready again.
	if err := s21SetState(ctx, e, TenantB, "note", "removing", 100); err != nil {
		return err
	}
	if err := s21SetState(ctx, e, TenantA, "note", "ready", 100); err != nil {
		return err
	}
	sideB, err := s22SideCount(ctx, e, "rfv_text", TenantB, noteB)
	if err != nil {
		return err
	}
	for _, role := range writers {
		pool, err := e.cluster.Pool(role)
		if err != nil {
			return err
		}
		value := "s22-ready-" + role
		if err := inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
			if n, err := execOne(ctx, tx, write, TenantA, value, 1, fx.desk); err != nil || n != 1 {
				return fmt.Errorf("control: %s rewriting tenant A's record: %d rows, %w", role, n, err)
			}
			if n, err := count(tx, `SELECT count(*) FROM rfv_text WHERE tenant_id = $1 AND field_id = $2 AND value = $3`, TenantA, noteA, value); err != nil || n != 1 {
				return fmt.Errorf("tenant B's removal changed tenant A's indexing: %s's write gave %d note rows (%w), want 1", role, n, err)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	if n, err := s22SideCount(ctx, e, "rfv_text", TenantB, noteB); err != nil || n != sideB {
		return fmt.Errorf("tenant A's writes moved tenant B's note side rows: %d -> %d (%w)", sideB, n, err)
	}
	return nil
}

// s22TenantRoles are the roles that serve a tenant's requests or jobs: none
// of them may write tenant_limits (ADR-006 D12: an operator change only).
func s22TenantRoles() []string { return []string{"nava_app", "nava_worker"} }

// Row 3. tenant_limits (TICK-751, absorbed into TICK-916): rows for tenant
// A (6) and tenant B (9) are seeded past the commit check. Every runtime
// role, in tenant A's transaction:
//   - a tenant role (nava_app, nava_worker) holds no write grant on the
//     table in the manifest, and INSERT, UPDATE and DELETE of its own row
//     and of tenant B's are refused (42501); its SELECT shows tenant A's row
//     only (control: it is there);
//   - any role the manifest does let write (the operator path) reaches no
//     tenant B row from tenant A's transaction.
//
// With no tenant set, a tenant role reads zero rows. Tenant B's row does
// not move.
func s22TenantLimits(ctx context.Context, c Cluster) error {
	super, err := c.Super()
	if err != nil {
		return err
	}
	seed := func(q string, args ...any) error {
		tx, err := super.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `SET LOCAL session_replication_role = replica`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, q, args...); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	const upsert = `INSERT INTO tenant_limits (tenant_id, filterable_fields_per_type, updated_at, updated_by) VALUES ($1, $2, now(), 'tick-753')
		ON CONFLICT (tenant_id) DO UPDATE SET filterable_fields_per_type = EXCLUDED.filterable_fields_per_type`
	for _, r := range []struct {
		tenant string
		n      int
	}{{TenantA, 6}, {TenantB, 9}} {
		if err := seed(upsert, r.tenant, r.n); err != nil {
			return fmt.Errorf("seed tenant_limits: %w", err)
		}
	}
	defer func() { _ = seed(`DELETE FROM tenant_limits WHERE tenant_id IN ($1, $2)`, TenantA, TenantB) }()
	readB := func() (int, error) {
		var n int
		err := super.QueryRow(ctx, `SELECT filterable_fields_per_type FROM tenant_limits WHERE tenant_id = $1`, TenantB).Scan(&n)
		return n, err
	}
	m, err := grants.Parse(db.Grants())
	if err != nil {
		return err
	}
	roles, err := s21RuntimeRoles()
	if err != nil {
		return err
	}
	for _, role := range roles {
		privs := m.Roles[role].Tables["tenant_limits"]
		_, sel := privs["SELECT"]
		writes := slices.ContainsFunc([]string{"INSERT", "UPDATE", "DELETE"}, func(p string) bool { _, ok := privs[p]; return ok })
		tenantRole := slices.Contains(s22TenantRoles(), role)
		if tenantRole && writes {
			return fmt.Errorf("db/grants.yaml lets tenant role %s write tenant_limits: a tenant could raise its own quota (ADR-006 D12)", role)
		}
		if tenantRole && !sel {
			return fmt.Errorf("control: tenant role %s cannot read tenant_limits, so PerTypeLimit cannot either", role)
		}
		pool, err := c.Pool(role)
		if err != nil {
			return err
		}
		if err := inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
			if sel {
				if n, err := execOne(ctx, tx, `SELECT 1 FROM tenant_limits WHERE tenant_id <> $1`, TenantA); err != nil || n != 0 {
					return fmt.Errorf("%s read %d tenant_limits rows of other tenants (%w)", role, n, err)
				}
				if n, err := execOne(ctx, tx, `SELECT 1 FROM tenant_limits WHERE tenant_id = $1`, TenantA); err != nil || n != 1 {
					return fmt.Errorf("control: %s reading tenant A's own limit: %d rows, %w", role, n, err)
				}
			}
			for _, w := range []struct {
				what, q string
				args    []any
			}{
				{"raise its own limit", `UPDATE tenant_limits SET filterable_fields_per_type = 10 WHERE tenant_id = $1`, []any{TenantA}},
				{"raise tenant B's limit", `UPDATE tenant_limits SET filterable_fields_per_type = 10 WHERE tenant_id = $1`, []any{TenantB}},
				{"insert a row for tenant B", `INSERT INTO tenant_limits (tenant_id, filterable_fields_per_type, updated_at, updated_by) VALUES ($1, 10, now(), 'x')`, []any{TenantB}},
				{"delete its own row (back to the default)", `DELETE FROM tenant_limits WHERE tenant_id = $1`, []any{TenantA}},
				{"delete tenant B's row", `DELETE FROM tenant_limits WHERE tenant_id = $1`, []any{TenantB}},
			} {
				n, err := execOne(ctx, tx, w.q, w.args...)
				if tenantRole {
					if err := refused(role+": "+w.what, err); err != nil {
						return err
					}
					continue
				}
				if !writes || w.args[0] != TenantB {
					continue
				}
				if err := s21Denied(role+" (manifest writer) in tenant A: "+w.what, n, err); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		if tenantRole {
			// No tenant set: zero rows, not every row.
			var n int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM tenant_limits`).Scan(&n); err != nil || n != 0 {
				return fmt.Errorf("%s with no tenant set read %d tenant_limits rows (%w), want 0", role, n, err)
			}
		}
	}
	if n, err := readB(); err != nil || n != 9 {
		return fmt.Errorf("tenant B's limit moved to %d (%w), want 9", n, err)
	}
	return nil
}

// s22JSONBOnData matches a JSONB read of a data column in a SQL string: an
// operator (containment, key existence, path or key extraction), also after
// a cast (data::jsonb ->>), or a jsonb_* function taking the column as its
// first argument (jsonb_extract_path_text(data, ...), jsonb_path_exists(...)).
var s22JSONBOnData = regexp.MustCompile(`\bdata\s*(::\s*jsonb\s*)?(@>|<@|->>|->|#>>|#>|\?\||\?&|\?|@\?|@@)|\bjsonb_[a-z_]+\s*\(\s*([a-z_]+\.)?data\b`)

// s22JSONBAllowed are the non-test files that may apply a JSONB operator to
// records.data, none of them a filter route:
//   - the field-index backfill and the schema backfill's fill read the
//     document to index or to queue records (ADR-006 D10, ADR-009 D2);
//   - the uniqueness probe (x-nava-unique) compares one property;
//   - the isolation suite's own adversarial SQL.
func s22JSONBAllowed(rel string) bool {
	for _, p := range []string{
		"internal/fieldindex/backfill/batch.go",
		"internal/records/store/unique.go",
		"internal/schemabackfill/",
		"internal/tenancy/isolation/",
	} {
		if rel == p || (strings.HasSuffix(p, "/") && strings.HasPrefix(rel, p)) {
			return true
		}
	}
	return false
}

// s22RepoRoot is the module root, from this file's path.
func s22RepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("no caller path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", ".."), nil
}

// Row 4. ADR-006 D6 (TICK-752): the legacy scan is gone, so a data filter can
// only be answered by the side tables under the D7 EXISTS with the row ACL.
// Every string literal in every non-test Go file of internal/ and cmd/ is
// read from the AST (comments do not count): one applying a JSONB operator
// to data outside s22JSONBAllowed fails, and so does any identifier named
// DataFilterTimeout, WithDataFilterTimeout or LegacyScanFields. The scan
// must see the list route (control), so a moved tree cannot pass it empty.
func s22NoJSONBFilter(context.Context) error {
	root, err := s22RepoRoot()
	if err != nil {
		return err
	}
	retired := []string{"DataFilterTimeout", "WithDataFilterTimeout", "LegacyScanFields"}
	var problems []string
	sawList := false
	for _, top := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			rel = filepath.ToSlash(rel)
			if rel == "internal/server/records_list.go" {
				sawList = true
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.BasicLit:
					if x.Kind != token.STRING || s22JSONBAllowed(rel) {
						return true
					}
					s, err := strconv.Unquote(x.Value)
					if err != nil {
						s = x.Value
					}
					if m := s22JSONBOnData.FindString(s); m != "" {
						problems = append(problems, fmt.Sprintf("%s: %q applies a JSONB operator to data", fset.Position(x.Pos()), m))
					}
				case *ast.Ident:
					if slices.Contains(retired, x.Name) {
						problems = append(problems, fmt.Sprintf("%s: %s is retired (ADR-006 D6)", fset.Position(x.Pos()), x.Name))
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			return err
		}
	}
	if !sawList {
		return errors.New("control: the scan did not reach internal/server/records_list.go")
	}
	if len(problems) > 0 {
		return fmt.Errorf("the legacy JSONB filter path is still there (%d sites):\n%s", len(problems), strings.Join(problems, "\n"))
	}
	return nil
}

// s29TableNames are ADR-009's three tables (contract C1).
func s29TableNames() []string {
	return []string{"schema_backfills", "schema_backfill_pending", "schema_backfill_failures"}
}

// s29Seed writes, past RLS and triggers, one job per tenant on a definition
// of that tenant (A pending, B running: B holds its one active slot), with
// one pending and one failure row each; s29Unseed deletes them.
func s29Seed(ctx context.Context, super *pgx.Conn) error {
	tx, err := super.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	stmts := []struct {
		q    string
		args []any
	}{
		{`SET LOCAL session_replication_role = replica`, nil},
		{`INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, state, default_keys, rewritten, skipped, failed, created_at)
			VALUES ($1, $2, 753, 'pending', '{iso_s29}', 0, 0, 0, now()), ($3, $4, 753, 'running', '{iso_s29}', 31337, 0, 1, now())`,
			[]any{TenantA, ticketDefA, TenantB, incidentDefB}},
		{`INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id) VALUES ($1, $2, 753, $3), ($4, $5, 753, $6)`,
			[]any{TenantA, ticketDefA, RecordA1, TenantB, incidentDefB, RecordB1}},
		{`INSERT INTO schema_backfill_failures (tenant_id, entity_definition_id, target_version, record_id, code, failed_at)
			VALUES ($1, $2, 753, $3, 'validation_failed', now()), ($4, $5, 753, $6, 'db_error', now())`,
			[]any{TenantA, ticketDefA, RecordA1, TenantB, incidentDefB, RecordB1}},
	}
	for _, s := range stmts {
		if _, err := tx.Exec(ctx, s.q, s.args...); err != nil {
			return fmt.Errorf("s29 seed: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func s29Unseed(ctx context.Context, super *pgx.Conn) error {
	_, err := super.Exec(ctx, `DELETE FROM schema_backfills WHERE target_version IN (753, 754) AND entity_definition_id IN ($1, $2)`, ticketDefA, incidentDefB)
	return err
}

// Row S29-1 (TICK-916; CI-3 (5), (6)). Tenant A and tenant B each hold a
// job, a pending row and a failure row. As nava_app and as nava_worker in
// tenant A, on each table: no row of another tenant, by tenant or by tenant
// B's definition id; tenant A's own rows are there (control); an INSERT
// naming tenant B is refused; nava_app's INSERT, UPDATE and DELETE are
// refused (42501) even on its own rows; nava_worker's UPDATE and DELETE of
// tenant B's rows reach none, of its own reach them (control). With no
// tenant set each reads zero rows. schema_backfill_failures has exactly
// the C1 columns: no value column. Admission: tenant B's running job holds
// tenant B's slot only; tenant A's job still moves to filling, and a second
// tenant A job is then refused (23505), the index's control.
func s29Tables(ctx context.Context, c Cluster) error {
	super, err := c.Super()
	if err != nil {
		return err
	}
	if err := s29Unseed(ctx, super); err != nil {
		return err
	}
	if err := s29Seed(ctx, super); err != nil {
		return err
	}
	defer func() { _ = s29Unseed(ctx, super) }()
	rows, err := super.Query(ctx, `SELECT attname::text FROM pg_attribute WHERE attrelid = 'schema_backfill_failures'::regclass AND attnum > 0 AND NOT attisdropped ORDER BY attname`)
	if err != nil {
		return err
	}
	cols, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if want := []string{"code", "entity_definition_id", "failed_at", "record_id", "target_version", "tenant_id"}; !slices.Equal(cols, want) {
		return fmt.Errorf("schema_backfill_failures columns = %v, want exactly %v (no value column, ever)", cols, want)
	}
	inserts := map[string]string{
		"schema_backfills": `INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, state, default_keys, created_at)
			VALUES ($1, $2, 754, 'pending', '{iso_s29}', now())`,
		"schema_backfill_pending":  `INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id) VALUES ($1, $2, 753, $3)`,
		"schema_backfill_failures": `INSERT INTO schema_backfill_failures (tenant_id, entity_definition_id, target_version, record_id, code, failed_at) VALUES ($1, $2, 753, $3, 'db_error', now())`,
	}
	insertArgs := func(table, tenant, def, record string) []any {
		if table == "schema_backfills" {
			return []any{tenant, def}
		}
		return []any{tenant, def, record}
	}
	for _, role := range []string{"nava_app", "nava_worker"} {
		pool, err := c.Pool(role)
		if err != nil {
			return err
		}
		worker := role == "nava_worker"
		for _, table := range s29TableNames() {
			if err := inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
				for _, q := range []struct {
					what string
					q    string
					arg  string
				}{
					{"rows of another tenant", `SELECT 1 FROM ` + table + ` WHERE tenant_id <> $1`, TenantA},
					{"rows of tenant B's definition", `SELECT 1 FROM ` + table + ` WHERE entity_definition_id = $1`, incidentDefB},
				} {
					if n, err := execOne(ctx, tx, q.q, q.arg); err != nil || n != 0 {
						return fmt.Errorf("%s: %s read %d %s (%w)", role, table, n, q.what, err)
					}
				}
				if n, err := execOne(ctx, tx, `SELECT 1 FROM `+table+` WHERE tenant_id = $1 AND entity_definition_id = $2`, TenantA, ticketDefA); err != nil || n != 1 {
					return fmt.Errorf("control: %s: %s read %d of tenant A's own rows (%w), want 1", role, table, n, err)
				}
				for _, bad := range []struct{ what, tenant, def, record string }{
					{"a row of tenant B", TenantB, incidentDefB, RecordB1},
					{"tenant A's row on tenant B's definition", TenantA, incidentDefB, RecordA1},
				} {
					n, err := execOne(ctx, tx, inserts[table], insertArgs(table, bad.tenant, bad.def, bad.record)...)
					if err := s21Denied(role+": "+table+": INSERT "+bad.what, n, err); err != nil {
						return err
					}
				}
				if !worker {
					n, err := execOne(ctx, tx, inserts[table], insertArgs(table, TenantA, ticketDefA, RecordA1)...)
					if err := s21Want(role+": "+table+": INSERT its own row", false, n, err, 0); err != nil {
						return err
					}
					_, err = execOne(ctx, tx, `UPDATE `+table+` SET target_version = target_version WHERE tenant_id = $1`, TenantA)
					if err := refused(role+": "+table+": UPDATE its own rows", err); err != nil {
						return err
					}
					_, err = execOne(ctx, tx, `DELETE FROM `+table+` WHERE tenant_id = $1`, TenantA)
					return refused(role+": "+table+": DELETE its own rows", err)
				}
				for _, q := range []string{
					`UPDATE ` + table + ` SET target_version = target_version WHERE tenant_id = $1`,
					`DELETE FROM ` + table + ` WHERE tenant_id = $1`,
				} {
					if n, err := execOne(ctx, tx, q, TenantB); err != nil || n != 0 {
						return fmt.Errorf("%s: %s reached %d of tenant B's rows (%w): %s", role, table, n, err, q)
					}
				}
				if n, err := execOne(ctx, tx, `UPDATE `+table+` SET target_version = target_version WHERE tenant_id = $1`, TenantA); err != nil || n != 1 {
					return fmt.Errorf("control: %s: %s UPDATE of its own row: %d rows, %w", role, table, n, err)
				}
				return nil
			}); err != nil {
				return err
			}
			var n int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&n); err != nil || n != 0 {
				return fmt.Errorf("%s with no tenant set read %d rows of %s (%w), want 0", role, n, table, err)
			}
		}
	}
	// Admission (D7): tenant B's running job does not hold tenant A's slot.
	worker, err := c.Pool("nava_worker")
	if err != nil {
		return err
	}
	return inTenant(ctx, worker, TenantA, func(tx pgx.Tx) error {
		const admit = `UPDATE schema_backfills SET state = 'filling', started_at = now() WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $3 AND state = 'pending'`
		if n, err := execOne(ctx, tx, admit, TenantA, ticketDefA, 753); err != nil || n != 1 {
			return fmt.Errorf("tenant A's admission with tenant B's job running: %d rows, %w; want 1 (another tenant's job must not hold tenant A's slot)", n, err)
		}
		if _, err := execOne(ctx, tx, inserts["schema_backfills"], TenantA, ticketDefA); err != nil {
			return fmt.Errorf("control: second tenant A job: %w", err)
		}
		if _, err := execOne(ctx, tx, admit, TenantA, ticketDefA, 754); sqlState(err) != "23505" {
			return fmt.Errorf("control: a second active tenant A job = %w, want 23505 on schema_backfills_one_active_per_tenant", err)
		}
		return nil
	})
}

// The S29 fixture: iso_et_s29 in both tenants under one key, created at v1
// without prio, then published at v2 adding prio (default "low",
// x-nava-filterable), so records created at v1 lack prio and fall in the
// backfill window. Tenant A's records are split by the S21 row ACL (desk
// group and admin group).
const (
	s29Key     = "iso_et_s29"
	s29Records = "/api/v1/records/" + s29Key
)

func s29Definition(withPrio bool) string {
	prio := ""
	if withPrio {
		prio = `"prio":{"type":"string","default":"low","x-nava-filterable":true},`
	}
	schema := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,` +
		`"required":["state"],"properties":{"state":{"type":"string","enum":["new","closed"],"default":"new"},` + prio +
		`"note":{"type":"string"},"group":{"type":"string","format":"uuid"}}}`
	return fmt.Sprintf(`{"schema": %s, "fixed_columns": {"state": "state", "group": "assignment_group_id"},
		"states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}`, schema)
}

// s29Fixture: a1 and a3 are tenant A's v1 records (no prio), a1 the desk's,
// a3 the admin's only; a2 is tenant A's v2 record with prio "high". b1 is
// tenant B's v1 record, b2 tenant B's v2 record with prio "low" explicitly.
type s29Fixture struct {
	a1, a2, a3, b1, b2 string
	defA, defB         string
}

func withS29Fixture(ctx context.Context, cl Cluster, fn func(context.Context, *etIso, *s29Fixture) error) error {
	e, err := newETIso(ctx, cl)
	if err != nil {
		return err
	}
	defer e.stop()
	clean := func() error {
		if _, err := e.super.Exec(ctx, `DELETE FROM schema_backfills WHERE entity_definition_id IN (SELECT id FROM entity_definitions WHERE key LIKE 'iso\_et\_%')`); err != nil {
			return err
		}
		if _, err := e.super.Exec(ctx, `DELETE FROM outbox WHERE kind = 'schema.backfill' AND (payload->>'entity_definition_id')::uuid IN (SELECT id FROM entity_definitions WHERE key LIKE 'iso\_et\_%')`); err != nil {
			return err
		}
		if err := s21Cleanup(ctx, e); err != nil {
			return err
		}
		return e.cleanup()
	}
	if err := clean(); err != nil {
		return err
	}
	fx, err := s29Setup(ctx, e)
	if err == nil {
		err = fn(ctx, e, fx)
	}
	if cerr := clean(); err == nil {
		err = cerr
	}
	return err
}

func s29Create(ctx context.Context, e *etIso, token, body string) (string, error) {
	r := e.ci8Do(ctx, e.h, http.MethodPost, s29Records, token, body, nil)
	if r.code != http.StatusCreated {
		return "", fmt.Errorf("control: %s creating a record = %d %s, want 201", token, r.code, r.body)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(r.body), &out); err != nil || out.ID == "" {
		return "", fmt.Errorf("control: the created record has no id: %w: %s", err, r.body)
	}
	return out.ID, nil
}

func s29Setup(ctx context.Context, e *etIso) (*s29Fixture, error) {
	fx := &s29Fixture{}
	for _, t := range []struct{ token, prefix string }{{s21Admin, "ISX"}, {etCallerBAdmin, "ISY"}} {
		body := fmt.Sprintf(`{"key": %q, "name": "Iso S29", "number_prefix": %q, "definition": %s}`, s29Key, t.prefix, s29Definition(false))
		if code, out := e.do(ctx, http.MethodPost, "/api/v1/entity-types", t.token, body, nil); code != http.StatusCreated {
			return nil, fmt.Errorf("control: %s creating %s = %d %s, want 201", t.token, s29Key, code, out)
		}
	}
	if err := tenancy.InTenantTx(ctx, e.pool, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(ctx, tx, s29Key, s21Rules(), nil)
		return err
	}); err != nil {
		return nil, fmt.Errorf("control: publishing tenant A's policy: %w", err)
	}
	var err error
	if fx.a1, err = s29Create(ctx, e, s21Admin, fmt.Sprintf(`{"data": {"note": "a1", "group": %q}}`, auditorGroupA)); err != nil {
		return nil, err
	}
	if fx.a3, err = s29Create(ctx, e, s21Admin, fmt.Sprintf(`{"data": {"note": "a3", "group": %q}}`, adminGroupA)); err != nil {
		return nil, err
	}
	if fx.b1, err = s29Create(ctx, e, auditCallerBAgent, `{"data": {"note": "b1"}}`); err != nil {
		return nil, err
	}
	// v2 adds prio with a default: the publish writes the job (TICK-919).
	v2 := fmt.Sprintf(`{"definition": %s}`, s29Definition(true))
	for _, token := range []string{s21Admin, etCallerBAdmin} {
		if code, out := e.do(ctx, http.MethodPost, "/api/v1/entity-types/"+s29Key+"/versions", token, v2, nil); code != http.StatusCreated {
			return nil, fmt.Errorf("control: %s publishing v2 with a default = %d %s, want 201", token, code, out)
		}
	}
	for _, d := range []struct {
		tenant string
		out    *string
	}{{TenantA, &fx.defA}, {TenantB, &fx.defB}} {
		if err := e.super.QueryRow(ctx, `SELECT id::text FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, d.tenant, s29Key).Scan(d.out); err != nil {
			return nil, err
		}
	}
	if _, err := e.super.Exec(ctx, `UPDATE record_index_fields f SET state = 'ready', progress = 100, updated_at = now()
		FROM entity_definitions d WHERE d.tenant_id = f.tenant_id AND d.id = f.entity_definition_id AND d.key = $1`, s29Key); err != nil {
		return nil, err
	}
	if fx.a2, err = s29Create(ctx, e, s21Admin, fmt.Sprintf(`{"data": {"note": "a2", "prio": "high", "group": %q}}`, adminGroupA)); err != nil {
		return nil, err
	}
	if fx.b2, err = s29Create(ctx, e, auditCallerBAgent, `{"data": {"note": "b2", "prio": "low"}}`); err != nil {
		return nil, err
	}
	// Control: the v1 records lack prio in storage.
	var lacking int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM records WHERE id = ANY($1::uuid[]) AND NOT (data ? 'prio')`, []string{fx.a1, fx.a3, fx.b1}).Scan(&lacking); err != nil || lacking != 3 {
		return nil, fmt.Errorf("control: %d of the v1 records lack prio (%w), want 3", lacking, err)
	}
	return fx, nil
}

// s29SetJob sets tenant's newest iso_et_s29 job (the one the v2 publish
// wrote) to state, with its pending rows exactly pending and its failure
// rows exactly failed, past RLS and triggers.
func s29SetJob(ctx context.Context, e *etIso, tenant, def, state string, pending, failed []string) error {
	tx, err := e.super.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var version int
	if err := tx.QueryRow(ctx, `SELECT max(target_version) FROM schema_backfills WHERE tenant_id = $1 AND entity_definition_id = $2`, tenant, def).Scan(&version); err != nil {
		return fmt.Errorf("control: tenant %s's v2 publish wrote no job: %w", tenant, err)
	}
	for _, s := range []struct {
		q    string
		args []any
	}{
		{`SET LOCAL session_replication_role = replica`, nil},
		{`DELETE FROM schema_backfill_pending WHERE tenant_id = $1 AND entity_definition_id = $2`, []any{tenant, def}},
		{`DELETE FROM schema_backfill_failures WHERE tenant_id = $1 AND entity_definition_id = $2`, []any{tenant, def}},
		{`UPDATE schema_backfills SET state = $3 WHERE tenant_id = $1 AND entity_definition_id = $2 AND target_version = $4`, []any{tenant, def, state, version}},
		{`INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id) SELECT $1, $2, $3, unnest($4::uuid[])`, []any{tenant, def, version, pending}},
		{`INSERT INTO schema_backfill_failures (tenant_id, entity_definition_id, target_version, record_id, code, failed_at) SELECT $1, $2, $3, unnest($4::uuid[]), 'validation_failed', now()`, []any{tenant, def, version, failed}},
	} {
		if _, err := tx.Exec(ctx, s.q, s.args...); err != nil {
			return fmt.Errorf("set tenant %s's job to %s: %w", tenant, state, err)
		}
	}
	return tx.Commit(ctx)
}

// s29ListHas is s21ListHas on iso_et_s29.
func s29ListHas(ctx context.Context, e *etIso, token, query string, want ...string) error {
	r := e.ci8Do(ctx, e.h, http.MethodGet, s29Records+"?"+query, token, "", nil)
	var p s10Page
	if r.code != http.StatusOK {
		return fmt.Errorf("%s ?%s = %d %s, want 200", token, query, r.code, r.body)
	}
	if err := json.Unmarshal([]byte(r.body), &p); err != nil || p.Total == nil {
		return fmt.Errorf("%s ?%s: not a page: %w: %s", token, query, err, r.body)
	}
	got := p.ids()
	slices.Sort(got)
	want = slices.Clone(want)
	slices.Sort(want)
	if !slices.Equal(got, want) || *p.Total != len(want) {
		return fmt.Errorf("%s's list ?%s holds %v with total %d, want %v", token, query, got, *p.Total, want)
	}
	return nil
}

// Row S29-3 (TICK-919, the D12 shapes of TICK-924). Tenant A's job goes
// through pending and filling (OR shape: a1 and a3 lack prio), running with
// two remaining (UNION ALL shape), then a1 pending and a3 failed in running
// and in failed_records (UNION ALL over pending ∪ failures); tenant B's job,
// under the same key and field key, goes through pending, running and
// failed_records at each step. For every
// pair of states: tenant A's admin's filter[prio]=low is exactly {a1, a3}
// with total 2 while the window is open; the desk's is exactly {a1} (a3 is
// the admin's: the ACL holds in the UNION ALL arm); the plain user's is
// empty; filter[prio]=high is {a2} for the admin; tenant B's admin sees only
// b1 and b2. No answer ever holds or counts a tenant B record.
func s29Filters(ctx context.Context, e *etIso, fx *s29Fixture) error {
	type jobState struct {
		state           string
		pending, failed []string
	}
	statesA := []jobState{
		{"pending", nil, nil},
		{"filling", nil, nil},
		{"running", []string{fx.a1, fx.a3}, nil},
		{"running", []string{fx.a1}, []string{fx.a3}},
		{"failed_records", []string{fx.a1}, []string{fx.a3}},
	}
	statesB := []jobState{
		{"pending", nil, nil},
		{"running", []string{fx.b1}, nil},
		{"failed_records", nil, []string{fx.b1}},
	}
	for _, a := range statesA {
		if err := s29SetJob(ctx, e, TenantA, fx.defA, a.state, a.pending, a.failed); err != nil {
			return err
		}
		for _, b := range statesB {
			if err := s29SetJob(ctx, e, TenantB, fx.defB, b.state, b.pending, b.failed); err != nil {
				return err
			}
			where := fmt.Sprintf("tenant A %s %d/%d, tenant B %s: ", a.state, len(a.pending), len(a.failed), b.state)
			for _, c := range []struct {
				token, query string
				want         []string
			}{
				{s21Admin, "filter[prio]=low", []string{fx.a1, fx.a3}},
				{s21Desk, "filter[prio]=low", []string{fx.a1}},
				{s21Plain, "filter[prio]=low", nil},
				{s21Admin, "filter[prio]=high", []string{fx.a2}},
				{s21Desk, "filter[prio]=high", nil},
				{s21Admin, "filter[prio][in]=low,high", []string{fx.a1, fx.a2, fx.a3}},
				{etCallerBAdmin, "filter[prio]=low", []string{fx.b1, fx.b2}},
			} {
				if err := s29ListHas(ctx, e, c.token, c.query, c.want...); err != nil {
					return errors.New(where + err.Error())
				}
			}
		}
	}
	return nil
}

// s29NotFoundAs requires both requests, as token, to be 404 with the same
// bytes.
func s29NotFoundAs(ctx context.Context, e *etIso, token, method, body, other, unknown string) error {
	oc, ob := e.do(ctx, method, other, token, body, nil)
	uc, ub := e.do(ctx, method, unknown, token, body, nil)
	switch {
	case uc != http.StatusNotFound:
		return fmt.Errorf("control: %s %s %s = %d %s, want 404", token, method, unknown, uc, ub)
	case oc != http.StatusNotFound:
		return fmt.Errorf("%s %s %s = %d %s, want 404", token, method, other, oc, ob)
	case ob != ub:
		return fmt.Errorf("%s %s %s: 404 differs from the unknown key's:\n%s\n%s", token, method, other, ob, ub)
	}
	return nil
}

// s29Jobs is every backfill job and schema.backfill outbox row of tenant,
// read past RLS, for a before/after compare.
func s29Jobs(ctx context.Context, e *etIso, tenant string) (string, error) {
	var s string
	err := e.super.QueryRow(ctx, `SELECT coalesce((SELECT string_agg(entity_definition_id::text || ':' || target_version::text || ':' || state, ',' ORDER BY 1)
		FROM schema_backfills WHERE tenant_id = $1), '') || '|' ||
		coalesce((SELECT string_agg(dedupe_key, ',' ORDER BY 1) FROM outbox WHERE tenant_id = $1 AND kind = 'schema.backfill'), '')`, tenant).Scan(&s)
	return s, err
}

// Row S29-2 (TICK-919, the routes of TICK-922; CI-3 (7) publish half).
//   - Tenant A on tenant B's only key (incident, which here holds a job and
//     a failure): getEntityType, .../backfill/failures and a publish adding a
//     default are the unknown key's 404, and tenant B's jobs and outbox rows
//     do not move.
//   - Tenant A's plain user on tenant A's own failures is the unknown key's
//     404 (control: the admin gets 200).
//   - Under the shared key, tenant A's getEntityType shows tenant A's job
//     (counters 0), never tenant B's (31337 rewritten), and tenant A's
//     failures list tenant A's record ids only.
//   - The v2 publishes of the fixture wrote one job and one outbox row per
//     tenant, each of its own tenant and its own definition, and no payload
//     names a tenant.
func s29Routes(ctx context.Context, e *etIso, fx *s29Fixture) error {
	if err := s29Unseed(ctx, e.super); err != nil {
		return err
	}
	if err := s29Seed(ctx, e.super); err != nil {
		return err
	}
	defer func() { _ = s29Unseed(ctx, e.super) }()
	jobsB, err := s29Jobs(ctx, e, TenantB)
	if err != nil {
		return err
	}
	for _, c := range [][2]string{
		{"/api/v1/entity-types/incident", "/api/v1/entity-types/iso_et_none"},
		{"/api/v1/entity-types/incident/backfill/failures", "/api/v1/entity-types/iso_et_none/backfill/failures"},
		{"/api/v1/entity-types/incident/backfill/failures?limit=100", "/api/v1/entity-types/iso_et_none/backfill/failures?limit=100"},
	} {
		if err := s29NotFoundAs(ctx, e, s21Admin, http.MethodGet, "", c[0], c[1]); err != nil {
			return err
		}
	}
	publish := fmt.Sprintf(`{"definition": %s}`, s29Definition(true))
	if err := s29NotFoundAs(ctx, e, s21Admin, http.MethodPost, publish, "/api/v1/entity-types/incident/versions", "/api/v1/entity-types/iso_et_none/versions"); err != nil {
		return err
	}
	if got, err := s29Jobs(ctx, e, TenantB); err != nil || got != jobsB {
		return fmt.Errorf("tenant A's requests on tenant B's key moved tenant B's jobs (%w): %s -> %s", err, jobsB, got)
	}
	if err := s29SetJob(ctx, e, TenantA, fx.defA, "failed_records", nil, []string{fx.a1}); err != nil {
		return err
	}
	if err := s29SetJob(ctx, e, TenantB, fx.defB, "failed_records", nil, []string{fx.b1}); err != nil {
		return err
	}
	if _, err := e.super.Exec(ctx, `UPDATE schema_backfills SET rewritten = 31337 WHERE tenant_id = $1 AND entity_definition_id = $2`, TenantB, fx.defB); err != nil {
		return err
	}
	own := "/api/v1/entity-types/" + s29Key + "/backfill/failures"
	{
		code, body := e.do(ctx, http.MethodGet, own, s21Admin, "", nil)
		if code != http.StatusOK {
			return fmt.Errorf("control: tenant A's admin on its own failures = %d %s, want 200", code, body)
		}
		var page struct {
			Items []struct {
				RecordID string `json:"record_id"`
				Code     string `json:"code"`
			} `json:"items"`
		}
		if err := json.Unmarshal([]byte(body), &page); err != nil {
			return fmt.Errorf("tenant A's failures: %w: %s", err, body)
		}
		if len(page.Items) != 1 || page.Items[0].RecordID != fx.a1 {
			return fmt.Errorf("tenant A's failures = %s, want exactly tenant A's %s", body, fx.a1)
		}
		if strings.Contains(body, fx.b1) {
			return fmt.Errorf("tenant A's failures name tenant B's record %s: %s", fx.b1, body)
		}
	}
	if err := s29NotFoundAs(ctx, e, s21Plain, http.MethodGet, "", own, "/api/v1/entity-types/iso_et_none/backfill/failures"); err != nil {
		return err
	}
	code, body := e.do(ctx, http.MethodGet, "/api/v1/entity-types/"+s29Key, s21Admin, "", nil)
	if code != http.StatusOK {
		return fmt.Errorf("control: tenant A's getEntityType = %d %s, want 200", code, body)
	}
	var et struct {
		Backfill *struct {
			State     string `json:"state"`
			Rewritten int    `json:"rewritten"`
			Failed    int    `json:"failed"`
		} `json:"backfill"`
	}
	if err := json.Unmarshal([]byte(body), &et); err != nil || et.Backfill == nil {
		return fmt.Errorf("control: tenant A's getEntityType has no backfill (%w): %s", err, body)
	}
	if et.Backfill.Rewritten == 31337 || strings.Contains(body, "31337") {
		return fmt.Errorf("tenant A's getEntityType shows tenant B's job: %s", body)
	}
	// The publishes' rows: own tenant, own definition, no tenant in payload.
	rows, err := e.super.Query(ctx, `SELECT o.tenant_id::text, coalesce(o.payload->>'entity_definition_id', ''), o.payload ? 'tenant_id', coalesce(d.tenant_id::text, '')
		FROM outbox o LEFT JOIN entity_definitions d ON d.id = (o.payload->>'entity_definition_id')::uuid
		WHERE o.kind = 'schema.backfill' AND (o.payload->>'entity_definition_id')::uuid IN ($1, $2)`, fx.defA, fx.defB)
	if err != nil {
		return err
	}
	type obRow struct {
		tenant, def string
		hasTenant   bool
		defTenant   string
	}
	obs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (obRow, error) {
		var o obRow
		err := r.Scan(&o.tenant, &o.def, &o.hasTenant, &o.defTenant)
		return o, err
	})
	if err != nil {
		return err
	}
	if len(obs) != 2 {
		return fmt.Errorf("control: the two v2 publishes wrote %d schema.backfill outbox rows, want 2", len(obs))
	}
	for _, o := range obs {
		if o.hasTenant || o.tenant != o.defTenant {
			return fmt.Errorf("schema.backfill outbox row of tenant %s names definition %s of tenant %s (payload tenant_id: %v)", o.tenant, o.def, o.defTenant, o.hasTenant)
		}
	}
	var mismatched int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM schema_backfills b WHERE NOT EXISTS
		(SELECT 1 FROM entity_definitions d WHERE d.tenant_id = b.tenant_id AND d.id = b.entity_definition_id)`).Scan(&mismatched); err != nil || mismatched != 0 {
		return fmt.Errorf("%d backfill jobs are not of their definition's tenant (%w)", mismatched, err)
	}
	return nil
}
