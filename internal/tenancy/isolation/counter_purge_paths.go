package isolation

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/purge"
)

// counterPurgeScope is one of the record_counters rows counterPurgeReach seeds
// for tenant A's and tenant B's ticket type, keyed by scope_key, with their
// period_end as an interval from now() ("" for NULL) and whether
// record_counters_purge may delete them (ADR-005 §4.3, §5).
type counterPurgeScope struct {
	scope, end string
	expired    bool
}

func counterPurgeScopes() []counterPurgeScope {
	return []counterPurgeScope{
		{"e1|d:ci9-expired", "-3 days", true},
		{"e1|d:ci9-grace", "-1 day -23 hours", false}, // inside the 2-day grace
		{"e1|d:ci9-open", "1 day", false},             // open period
		{"e1|f:ci9-field", "", false},                 // field-only: NULL
	}
}

// counterPurgePaths are the ADR-005 §5 rows of P6-10 (TICK-675): the
// record_counters purge is an ADR-001 A.3 allow-list entry, so nava_purge's
// reach on the table, and nava_app's lack of DELETE, are registered here.
func counterPurgePaths() []Path {
	return []Path{
		{Name: "nava_purge cannot see or delete open-period, in-grace or field-only counters, and its purge deletes expired ones",
			Case: "CI-9", Role: "nava_purge", DB: counterPurgeReach},
		{Name: "record_counters: nava_app deletes its own tenant's counters", Case: "CI-2", Role: "nava_app",
			SQL: &SQLCall{Query: "DELETE FROM record_counters WHERE tenant_id = $1", Args: []any{TenantA}, Expect: Denied}},
		{Name: "record_counters: nava_worker deletes its own tenant's counters", Case: "CI-2", Role: "nava_worker",
			SQL: &SQLCall{Query: "DELETE FROM record_counters WHERE tenant_id = $1", Args: []any{TenantA}, Expect: Denied}},
	}
}

// counterPurgeReach seeds counterPurgeScopes for tenants A and B, then:
// nava_purge sees exactly the expired rows of both tenants and never next;
// a DELETE naming a kept row deletes nothing; the job's own batch statement
// deletes exactly the expired rows.
func counterPurgeReach(ctx context.Context, c Cluster) (err error) {
	super, err := c.Super()
	if err != nil {
		return err
	}
	types := map[string]string{TenantA: definitionA1, TenantB: EntityDefinitionB1}
	cleanup := func() error {
		_, err := super.Exec(context.WithoutCancel(ctx), "DELETE FROM record_counters WHERE scope_key LIKE '%ci9-%'")
		return err
	}
	defer func() { err = errors.Join(err, cleanup()) }()
	for tenant, def := range types {
		for _, s := range counterPurgeScopes() {
			if _, err := super.Exec(ctx, `INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, period_end, next)
				VALUES ($1, $2, $3, now() + NULLIF($4, '')::interval, 5)`, tenant, def, s.scope, s.end); err != nil {
				return fmt.Errorf("seed %s %s: %w", tenant, s.scope, err)
			}
		}
	}
	pool, err := c.Pool("nava_purge")
	if err != nil {
		return err
	}
	var errs []error

	// Exactly the expired seeded rows are visible, in both tenants.
	rows, err := pool.Query(ctx, "SELECT tenant_id::text || ' ' || scope_key FROM record_counters WHERE scope_key LIKE '%ci9-%'")
	if err != nil {
		return fmt.Errorf("control: %w", err)
	}
	seen, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("control: %w", err)
	}
	var want []string
	for tenant := range types {
		for _, s := range counterPurgeScopes() {
			if s.expired {
				want = append(want, tenant+" "+s.scope)
			}
		}
	}
	slices.Sort(seen)
	slices.Sort(want)
	if !slices.Equal(seen, want) {
		errs = append(errs, fmt.Errorf("nava_purge sees %q, want %q", seen, want))
	}

	// Naming a kept row, of either tenant, deletes nothing.
	for tenant, def := range types {
		for _, s := range counterPurgeScopes() {
			if s.expired {
				continue
			}
			tag, err := pool.Exec(ctx, "DELETE FROM record_counters WHERE tenant_id = $1 AND entity_definition_id = $2 AND scope_key = $3",
				tenant, def, s.scope)
			switch {
			case err != nil:
				errs = append(errs, fmt.Errorf("nava_purge deletes %s %s: %w", tenant, s.scope, err))
			case tag.RowsAffected() != 0:
				errs = append(errs, fmt.Errorf("nava_purge deletes %s %s: %d rows, want 0", tenant, s.scope, tag.RowsAffected()))
			}
		}
	}

	// No counter value, no write, even of an expired row.
	for _, q := range []string{
		"SELECT next FROM record_counters",
		"SELECT * FROM record_counters",
		"UPDATE record_counters SET period_end = now() - interval '3 days'",
		"UPDATE record_counters SET next = 2",
	} {
		_, err := pool.Exec(ctx, q)
		errs = append(errs, refused(q, err))
	}

	// The job's batch deletes the expired rows of both tenants and nothing else.
	for _, s := range purge.Statements(purge.MaxBatch, purge.DefaultWindows()) {
		if s.Table != purge.TableRecordCounters {
			continue
		}
		tag, err := pool.Exec(ctx, s.SQL, s.Args...)
		if err != nil {
			return fmt.Errorf("purge batch: %w", err)
		}
		if tag.RowsAffected() != int64(len(want)) {
			errs = append(errs, fmt.Errorf("purge batch deleted %d rows, want %d", tag.RowsAffected(), len(want)))
		}
	}
	for tenant, def := range types {
		for _, s := range counterPurgeScopes() {
			var left bool
			if err := super.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM record_counters
				WHERE tenant_id = $1 AND entity_definition_id = $2 AND scope_key = $3)`, tenant, def, s.scope).Scan(&left); err != nil {
				return err
			}
			if left == s.expired {
				errs = append(errs, fmt.Errorf("after purge, %s %s (expired %v): present = %v", tenant, s.scope, s.expired, left))
			}
		}
	}
	return errors.Join(errs...)
}
