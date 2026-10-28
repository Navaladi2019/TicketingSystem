package isolation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/hba"
	"github.com/Navaladi2019/TicketingSystem/internal/purge"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TenantP is a provisioning tenant the onboarding paths create and remove.
const TenantP = "0d000000-0000-4000-8000-00000000000d"

// rolePaths are the CI-9 cases of T12 (TICK-71): each system-actor role
// reaches only what ADR-001 A.3 gives it. One path per bullet of the task's
// contract. Every DB path first proves its positive control, so a denial
// cannot come from a broken statement or connection.
func rolePaths() []Path {
	return []Path{
		{Name: "nava_relay cannot read records", Case: "CI-9", Role: "nava_relay", DB: relayCannotReadRecords},
		{Name: "nava_purge cannot read data columns or unexpired rows, and its purge deletes expired rows", Case: "CI-9", Role: "nava_purge", DB: purgeReach},
		{Name: "nava_onboard cannot write to a non-provisioning tenant", Case: "CI-9", Role: "nava_onboard", DB: onboardCannotWriteActive},
		{Name: "nava_backup is rejected for SQL sessions and logical walsenders (T08's pg_hba cases)", Case: "CI-9", Role: "nava_backup", DB: backupRejected},
		{Name: "nava_onboard cannot bind an active tenant", Case: "CI-9", Role: "nava_onboard", DB: onboardCannotBindActive},
		{Name: "nava_onboard cannot move an active tenant back to provisioning", Case: "CI-9", Role: "nava_onboard", DB: onboardCannotDeactivate},
	}
}

// errRollback ends a tenancy.InTenantTx callback so its writes are rolled back.
var errRollback = errors.New("rollback")

// sqlState returns err's SQLSTATE, or "" when it is not a server error.
func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// refused is nil when err is SQLSTATE 42501 (a grant or a policy said no).
func refused(what string, err error) error {
	switch {
	case err == nil:
		return fmt.Errorf("%s: succeeded, want SQLSTATE 42501", what)
	case sqlState(err) != "42501":
		return fmt.Errorf("%s: %w, want SQLSTATE 42501", what, err)
	}
	return nil
}

// inTenant runs fn in tenancy.InTenantTx(tenant) and rolls it back.
func inTenant(ctx context.Context, pool *pgxpool.Pool, tenant string, fn func(pgx.Tx) error) error {
	err := tenancy.InTenantTx(ctx, pool, uuid.MustParse(tenant), func(tx pgx.Tx) error {
		if err := fn(tx); err != nil {
			return err
		}
		return errRollback
	})
	if errors.Is(err, errRollback) {
		return nil
	}
	return err
}

// execOne runs q in its own savepoint, so a refusal does not end tx, and
// returns the rows it reached.
func execOne(ctx context.Context, tx pgx.Tx, q string, args ...any) (int64, error) {
	sp, err := tx.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = sp.Rollback(context.WithoutCancel(ctx)) }()
	tag, err := sp.Exec(ctx, q, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), sp.Commit(ctx)
}

// relayCannotReadRecords: the relay polls outbox across tenants, and nothing
// of records, by name, column or join, with or without a tenant set.
func relayCannotReadRecords(ctx context.Context, c Cluster) error {
	pool, err := c.Pool("nava_relay")
	if err != nil {
		return err
	}
	// Control: the same connection reads outbox.
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM outbox").Scan(&n); err != nil || n == 0 {
		return errors.Join(fmt.Errorf("control: relay reads outbox: %d rows", n), err)
	}
	queries := []string{
		"SELECT tenant_id FROM records",
		"SELECT id FROM records WHERE tenant_id = '" + TenantB + "'",
		"SELECT count(*) FROM records",
		"SELECT o.id FROM outbox o JOIN records r ON r.tenant_id = o.tenant_id",
		"SELECT 1 FROM outbox o WHERE EXISTS (SELECT 1 FROM records r WHERE r.tenant_id = o.tenant_id)",
	}
	var errs []error
	for _, q := range queries {
		_, err := pool.Exec(ctx, q)
		errs = append(errs, refused("no tenant: "+q, err))
		err = inTenant(ctx, pool, TenantB, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, q)
			return err
		})
		errs = append(errs, refused("tenant B set: "+q, err))
	}
	super, err := c.Super()
	if err != nil {
		return err
	}
	// Column grants, then table grants (TRUNCATE and the rest have no
	// column form).
	var column, table bool
	if err := super.QueryRow(ctx, `SELECT
		has_any_column_privilege('nava_relay', 'public.records', 'SELECT, INSERT, UPDATE, REFERENCES'),
		has_table_privilege('nava_relay', 'public.records', 'SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER')`).Scan(&column, &table); err != nil {
		return err
	}
	if column || table {
		errs = append(errs, fmt.Errorf("nava_relay holds a privilege on records (column %v, table %v)", column, table))
	}
	return errors.Join(errs...)
}

// purgeRow is one row purgeReach seeds, keyed as the purge selects it.
type purgeRow struct {
	table, key string
	expired    bool
}

// purgeReach seeds, for tenants A and B, a row past each purge window and one
// inside it, then: nava_purge sees exactly the expired rows and none of the
// data columns, and purge.Purger, the job's own code, deletes exactly the
// expired rows of both tenants.
func purgeReach(ctx context.Context, c Cluster) (err error) {
	super, err := c.Super()
	if err != nil {
		return err
	}
	rows, cleanup, err := seedPurgeRows(ctx, super)
	defer func() { err = errors.Join(err, cleanup()) }()
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	pool, err := c.Pool("nava_purge")
	if err != nil {
		return err
	}
	var errs []error

	// Only expired rows are visible, through the key columns.
	visible := map[string][]string{}
	for table, q := range map[string]string{
		purge.TableIdempotencyKeys: "SELECT key FROM idempotency_keys",
		purge.TableOutbox:          "SELECT id::text FROM outbox",
		purge.TableWorkflowRef:     "SELECT outbox_id::text FROM workflow_ref",
	} {
		got, err := column(ctx, pool, q)
		if err != nil {
			return fmt.Errorf("control: %s: %w", q, err)
		}
		visible[table] = got
	}
	for _, r := range rows {
		if seen := slices.Contains(visible[r.table], r.key); seen != r.expired {
			errs = append(errs, fmt.Errorf("%s %s (expired %v): visible to nava_purge = %v", r.table, r.key, r.expired, seen))
		}
	}
	for table, keys := range visible {
		for _, k := range keys {
			if !slices.ContainsFunc(rows, func(r purgeRow) bool { return r.table == table && r.key == k && r.expired }) {
				errs = append(errs, fmt.Errorf("%s %s: visible to nava_purge and not expired", table, k))
			}
		}
	}

	// No data column, even of an expired row.
	for _, q := range []string{
		"SELECT request_hash FROM idempotency_keys",
		"SELECT response_body FROM idempotency_keys",
		"SELECT state, response_status FROM idempotency_keys",
		"SELECT payload FROM outbox",
		"SELECT dedupe_key FROM outbox",
		"SELECT kind, target_key, last_error FROM outbox",
		"SELECT record_id FROM workflow_ref",
		"SELECT kind FROM workflow_ref",
		// Every column, so one added later cannot slip past the list above.
		"SELECT * FROM idempotency_keys",
		"SELECT * FROM outbox",
		"SELECT * FROM workflow_ref",
	} {
		_, err := pool.Exec(ctx, q)
		errs = append(errs, refused(q, err))
	}

	// The job deletes the expired rows of both tenants and nothing else.
	p, err := purge.New(pool, metricnoop.NewMeterProvider(), slog.New(slog.DiscardHandler), purge.Options{})
	if err != nil {
		return err
	}
	wantDeleted := map[string]int64{}
	for _, r := range rows {
		if r.expired {
			wantDeleted[r.table]++
		}
	}
	for _, res := range p.Once(ctx) {
		if res.Err != nil || res.Skipped || res.Deleted != wantDeleted[res.Table] {
			errs = append(errs, errors.Join(fmt.Errorf("purge %s: deleted %d (want %d), skipped %v",
				res.Table, res.Deleted, wantDeleted[res.Table], res.Skipped), res.Err))
		}
	}
	for _, r := range rows {
		var left bool
		if err := super.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM "+r.table+" WHERE "+keyColumn(r.table)+"::text = $1)", r.key).Scan(&left); err != nil {
			return err
		}
		if left == r.expired {
			errs = append(errs, fmt.Errorf("after purge, %s %s (expired %v): present = %v", r.table, r.key, r.expired, left))
		}
	}
	return errors.Join(errs...)
}

func keyColumn(table string) string {
	switch table {
	case purge.TableIdempotencyKeys:
		return "key"
	case purge.TableWorkflowRef:
		return "outbox_id"
	}
	return "id"
}

// column returns the first column of every row q returns.
func column(ctx context.Context, pool *pgxpool.Pool, q string) ([]string, error) {
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// seedPurgeRows adds, per tenant, one row past and one inside each window:
// idempotency keys expired an hour ago and expiring in an hour; outbox rows
// dispatched 8 and 6 days ago, dead 31 and 29 days ago; workflow refs
// closed 31 and 29 days ago. cleanup deletes whatever of them is left.
func seedPurgeRows(ctx context.Context, super *pgx.Conn) ([]purgeRow, func() error, error) {
	var rows []purgeRow
	cleanup := func() error {
		ctx := context.WithoutCancel(ctx)
		var errs []error
		for _, r := range rows {
			_, err := super.Exec(ctx, "DELETE FROM "+r.table+" WHERE "+keyColumn(r.table)+"::text = $1", r.key)
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}
	record := map[string]string{TenantA: RecordA1, TenantB: RecordB1}
	type ages struct{ idempotency, dispatched, dead, closed string }
	byExpiry := map[bool]ages{
		true:  {"-1 hour", "8 days", "31 days", "31 days"},
		false: {"1 hour", "6 days", "29 days", "29 days"},
	}
	n := 0
	id := func() string { n++; return fmt.Sprintf("0e000000-0000-4000-8000-%012d", n) }
	for _, tenant := range []string{TenantA, TenantB} {
		for _, expired := range []bool{true, false} {
			a := byExpiry[expired]
			k, dispatched, dead, ref := "ci9-"+id(), id(), id(), id()
			for _, s := range []struct {
				row  purgeRow
				sql  string
				args []any
			}{
				{purgeRow{purge.TableIdempotencyKeys, k, expired},
					"INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at) VALUES ($1, $1, $2, '\\x00', now() + $3::interval)",
					[]any{tenant, k, a.idempotency}},
				{purgeRow{purge.TableOutbox, dispatched, expired},
					"INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, dispatched_at) VALUES ($2::text::uuid, $1, 'k', $2::text, 't', 1, now() - $3::interval)",
					[]any{tenant, dispatched, a.dispatched}},
				{purgeRow{purge.TableOutbox, dead, expired},
					"INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, dead_at) VALUES ($2::text::uuid, $1, 'k', $2::text, 't', 1, now() - $3::interval)",
					[]any{tenant, dead, a.dead}},
				{purgeRow{purge.TableWorkflowRef, ref, expired},
					"INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind, closed_at) VALUES ($1, $2, $3, 'k', now() - $4::interval)",
					[]any{tenant, ref, record[tenant], a.closed}},
			} {
				rows = append(rows, s.row)
				if _, err := super.Exec(ctx, s.sql, s.args...); err != nil {
					return rows, cleanup, fmt.Errorf("%s: %w", s.row.table, err)
				}
			}
		}
	}
	return rows, cleanup, nil
}

// withProvisioning creates TenantP in 'provisioning' for fn and deletes it
// (and any binding of it) afterwards.
func withProvisioning(ctx context.Context, c Cluster, fn func(*pgxpool.Pool) error) (err error) {
	super, err := c.Super()
	if err != nil {
		return err
	}
	if _, err := super.Exec(ctx, "INSERT INTO tenants (id, state) VALUES ($1, 'provisioning')", TenantP); err != nil {
		return fmt.Errorf("seed tenant P: %w", err)
	}
	defer func() {
		ctx := context.WithoutCancel(ctx)
		_, e1 := super.Exec(ctx, "DELETE FROM tenant_identity_binding WHERE tenant_id = $1", TenantP)
		_, e3 := super.Exec(ctx, "DELETE FROM pending_invite WHERE tenant_id = $1", TenantP)
		_, e2 := super.Exec(ctx, "DELETE FROM tenants WHERE id = $1", TenantP)
		err = errors.Join(err, e1, e3, e2)
	}()
	pool, err := c.Pool("nava_onboard")
	if err != nil {
		return err
	}
	return fn(pool)
}

// onboardCannotWriteActive: nava_onboard activates its own provisioning
// tenant (control), and writes nothing to an active one: neither its own
// (tenant A) nor another (tenant B), nor tenant A from tenant P's
// transaction. That includes pending_invite (T252a): no invite, re-invite or
// extension; and tenant_email_domain (TICK-280): no release.
func onboardCannotWriteActive(ctx context.Context, c Cluster) error {
	return withProvisioning(ctx, c, func(pool *pgxpool.Pool) error {
		var errs []error
		err := inTenant(ctx, pool, TenantP, func(tx pgx.Tx) error {
			n, err := execOne(ctx, tx, "UPDATE tenants SET state = 'active' WHERE id = $1", TenantP)
			if err != nil || n != 1 {
				return errors.Join(fmt.Errorf("control: activate tenant P: %d rows", n), err)
			}
			return nil
		})
		errs = append(errs, err)
		errs = append(errs, onboardInvitesOnlyProvisioning(ctx, pool))
		for _, w := range []struct{ ctxTenant, target string }{
			{TenantA, TenantA}, {TenantA, TenantB}, {TenantP, TenantA},
		} {
			err := inTenant(ctx, pool, w.ctxTenant, func(tx pgx.Tx) error {
				var errs []error
				for _, q := range []string{
					"UPDATE tenants SET state = 'active' WHERE id = $1",
					"UPDATE tenants SET state = state WHERE id = $1",
					// T252a: tenant A has an invite for SharedInviteEmail, so
					// the guard, not a missing row, must stop the extension.
					extendInvites,
				} {
					n, err := execOne(ctx, tx, q, w.target)
					if n != 0 || (err != nil && sqlState(err) != "42501") {
						errs = append(errs, errors.Join(fmt.Errorf("in tenant %s, %s on %s: %d rows", w.ctxTenant, q, w.target, n), err))
					}
				}
				// A second row for an active tenant. As 'active',
				// tenants_onboard_insert refuses it whatever the tenant.
				_, err := execOne(ctx, tx, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", w.target)
				errs = append(errs, refused(fmt.Sprintf("in tenant %s, insert %s as active", w.ctxTenant, w.target), err))
				// As 'provisioning', the policy refuses another tenant; for
				// its own it passes and the primary key refuses the row.
				_, err = execOne(ctx, tx, "INSERT INTO tenants (id, state) VALUES ($1, 'provisioning')", w.target)
				what := fmt.Sprintf("in tenant %s, insert %s as provisioning", w.ctxTenant, w.target)
				if w.ctxTenant != w.target {
					errs = append(errs, refused(what, err))
				} else if code := sqlState(err); code != "23505" {
					errs = append(errs, errors.Join(fmt.Errorf("%s: SQLSTATE %q, want 23505 from the primary key", what, code), err))
				}
				// T252a: no invite, new or re-invited, for an active tenant.
				for _, q := range []string{inviteNew, reinviteShared} {
					_, err := execOne(ctx, tx, q, w.target)
					errs = append(errs, refused(fmt.Sprintf("in tenant %s, %s for %s", w.ctxTenant, q, w.target), err))
				}
				// TICK-280: no domain release for an active tenant. Tenants A
				// and B own seeded domains, so the guard, not a missing row,
				// must stop it.
				if n, err := execOne(ctx, tx, releaseDomains, w.target); n != 0 || err != nil {
					errs = append(errs, errors.Join(fmt.Errorf("in tenant %s, release %s's domains: %d rows", w.ctxTenant, w.target, n), err))
				}
				return errors.Join(errs...)
			})
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	})
}

// The pending_invite statements InviteUser runs (T252a), each with the
// target tenant as $1.
const (
	inviteNew      = "INSERT INTO pending_invite (tenant_id, email, expires_at) VALUES ($1, 'ci9-invite@example.test', now() + interval '7 days')"
	reinviteShared = `INSERT INTO pending_invite (tenant_id, email, expires_at) VALUES ($1, '` + SharedInviteEmail + `', now() + interval '30 days')
		ON CONFLICT (tenant_id, email) DO UPDATE SET expires_at = EXCLUDED.expires_at WHERE pending_invite.consumed_at IS NULL`
	extendInvites = "UPDATE pending_invite SET expires_at = now() + interval '30 days' WHERE tenant_id = $1"
	// TICK-280's domain release, the same shape.
	releaseDomains = "DELETE FROM tenant_email_domain WHERE tenant_id = $1"
)

// onboardInvitesOnlyProvisioning is the positive control for the
// pending_invite guard (security finding 1 on PR #164): in provisioning
// tenant P's own transaction, the insert, the re-invite upsert of that row
// and the expires_at update each reach one row. Activated in the same
// transaction, P takes none of them.
func onboardInvitesOnlyProvisioning(ctx context.Context, pool *pgxpool.Pool) error {
	return inTenant(ctx, pool, TenantP, func(tx pgx.Tx) error {
		var errs []error
		for _, q := range []string{inviteNew, reinviteShared, reinviteShared, extendInvites} {
			if n, err := execOne(ctx, tx, q, TenantP); err != nil || n < 1 {
				errs = append(errs, errors.Join(fmt.Errorf("control: in tenant P, %s: %d rows", q, n), err))
			}
		}
		if n, err := execOne(ctx, tx, "UPDATE tenants SET state = 'active' WHERE id = $1", TenantP); err != nil || n != 1 {
			return errors.Join(append(errs, fmt.Errorf("control: activate tenant P: %d rows", n), err)...)
		}
		for _, q := range []string{inviteNew, reinviteShared} {
			_, err := execOne(ctx, tx, q, TenantP)
			errs = append(errs, refused("in tenant P after activation, "+q, err))
		}
		if n, err := execOne(ctx, tx, extendInvites, TenantP); n != 0 || (err != nil && sqlState(err) != "42501") {
			errs = append(errs, errors.Join(fmt.Errorf("in tenant P after activation, %s: %d rows", extendInvites, n), err))
		}
		return errors.Join(errs...)
	})
}

// onboardCannotBindActive: nava_onboard binds its provisioning tenant
// (control) and cannot bind an active one, its own or another.
func onboardCannotBindActive(ctx context.Context, c Cluster) error {
	const ins = "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://idp.example/realms/ci9', $2)"
	return withProvisioning(ctx, c, func(pool *pgxpool.Pool) error {
		var errs []error
		errs = append(errs, inTenant(ctx, pool, TenantP, func(tx pgx.Tx) error {
			if _, err := execOne(ctx, tx, ins, TenantP, "org-p"); err != nil {
				return fmt.Errorf("control: bind tenant P: %w", err)
			}
			// Activated in the same transaction, it can be bound no more.
			if n, err := execOne(ctx, tx, "UPDATE tenants SET state = 'active' WHERE id = $1", TenantP); err != nil || n != 1 {
				return errors.Join(fmt.Errorf("control: activate tenant P: %d rows", n), err)
			}
			_, err := execOne(ctx, tx, ins, TenantP, "org-p2")
			return refused("bind tenant P after activation", err)
		}))
		for _, w := range []struct{ ctxTenant, target string }{
			{TenantA, TenantA}, {TenantA, TenantB}, {TenantP, TenantA},
		} {
			errs = append(errs, inTenant(ctx, pool, w.ctxTenant, func(tx pgx.Tx) error {
				_, err := execOne(ctx, tx, ins, w.target, "org-ci9")
				return refused(fmt.Sprintf("in tenant %s, bind %s", w.ctxTenant, w.target), err)
			}))
		}
		return errors.Join(errs...)
	})
}

// onboardCannotDeactivate: an active tenant stays active. Not tenant A from
// its own transaction, and not tenant P once activated in the same one. Each
// check runs in its own transaction, so one failing cannot skip another.
func onboardCannotDeactivate(ctx context.Context, c Cluster) error {
	const back = "UPDATE tenants SET state = 'provisioning' WHERE id = $1"
	stays := func(what string, n int64, err error) error {
		if n != 0 || (err != nil && sqlState(err) != "42501") {
			return errors.Join(fmt.Errorf("%s: %d rows", what, n), err)
		}
		return nil
	}
	return withProvisioning(ctx, c, func(pool *pgxpool.Pool) error {
		return errors.Join(
			inTenant(ctx, pool, TenantP, func(tx pgx.Tx) error {
				if n, err := execOne(ctx, tx, "UPDATE tenants SET state = 'active' WHERE id = $1", TenantP); err != nil || n != 1 {
					return errors.Join(fmt.Errorf("control: activate tenant P: %d rows", n), err)
				}
				n, err := execOne(ctx, tx, back, TenantP)
				return stays("tenant P active -> provisioning", n, err)
			}),
			inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
				n, err := execOne(ctx, tx, back, TenantA)
				return stays("tenant A active -> provisioning", n, err)
			}),
			// A provisioning tenant cannot be "updated" to provisioning
			// either: WITH CHECK allows only active.
			inTenant(ctx, pool, TenantP, func(tx pgx.Tx) error {
				_, err := execOne(ctx, tx, back, TenantP)
				return refused("tenant P provisioning -> provisioning", err)
			}),
		)
	})
}

// backupRejected runs T08's nava_backup cases (internal/db/hba, also run by
// db/pg_hba_test.go) against the embedded db/pg_hba.conf, then proves the
// grant layer underneath: even past pg_hba, nava_backup has no CONNECT on the
// platform database.
func backupRejected(ctx context.Context, c Cluster) error {
	lines, err := hba.Parse(db.HBA())
	if err != nil {
		return err
	}
	errs := []error{hba.Check(lines, hba.BackupCases())}
	pool, err := c.Pool("nava_backup")
	if err != nil {
		return err
	}
	errs = append(errs, refused("nava_backup SQL session on the platform database (no pg_hba here)", pool.Ping(ctx)))
	return errors.Join(errs...)
}
