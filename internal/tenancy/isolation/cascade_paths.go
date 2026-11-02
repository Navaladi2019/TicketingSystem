package isolation

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The cascade start key's seed (testdata/seed.sql, P3-12): tenant B also
// holds a record with tenant A's id RecordA1 (SharedRecord), both rows under
// the causation origin SharedOrigin, and tenant B alone has claimed
// SharedDefinition on it. A key read or claimed without its tenant would
// therefore meet tenant B's claim.
const (
	SharedRecord     = RecordA1
	SharedOrigin     = "0d000000-0000-4000-8000-0000000009d1"
	SharedDefinition = "0d000000-0000-4000-8000-0000000009d2"
)

// cascadePaths are ADR-001 CI-10 (f), as amended by A5: the start key
// (tenant, origin, record, definition) is per tenant. The pipeline cases
// (a)–(e) are integration tests in internal/records/pipeline (TestCI10_*),
// against the real pipeline and a test EventSink; (g) needs the interpreter
// and is P3-42's.
func cascadePaths() []Path {
	sql := func(name, role string, expect Expect, query string, args ...any) Path {
		return Path{Name: "cascade_start: " + name, Case: "CI-10", Role: role, SQL: &SQLCall{Query: query, Args: args, Expect: expect}}
	}
	return []Path{
		sql("look a shared origin up in every tenant", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM cascade_start WHERE causation_origin_id = $1", SharedOrigin),
		sql("read tenant B's claim by its record and definition", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM cascade_start WHERE record_id = $1 AND definition_id = $2", SharedRecord, SharedDefinition),
		// Tenant B cannot be made to claim a key from tenant A's transaction:
		// the insert policy holds the tenant.
		sql("claim a key for tenant B", "nava_worker", Denied,
			`INSERT INTO cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id)
			 VALUES ($1, $2, $3, $4, $5) ON CONFLICT ON CONSTRAINT cascade_start_pkey DO NOTHING RETURNING tenant_id`,
			TenantB, SharedOrigin, RecordB2, SharedDefinition, NewID),
		// The composite foreign key holds the tenant: tenant A cannot claim a
		// key on tenant B's record.
		sql("claim a key on tenant B's record from tenant A", "nava_app", Denied,
			`INSERT INTO cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id)
			 VALUES ($1, $2, $3, $4, $5)`, TenantA, SharedOrigin, RecordB2, SharedDefinition, NewID),
		sql("rewrite tenant B's claims", "nava_worker", Denied,
			"UPDATE cascade_start SET outbox_id = $2 WHERE tenant_id = $1", TenantB, NewID),
		sql("delete tenant B's claims", "nava_app", Denied,
			"DELETE FROM cascade_start WHERE tenant_id = $1", TenantB),
		{Name: "cascade_start: the same origin, record and definition in two tenants never collide", Case: "CI-10",
			Role: "nava_worker", DB: cascadeNoCollision},
	}
}

// cascadeNoCollision claims tenant B's seeded key, by the same uuids, from
// tenant A through cascade.ClaimStart, which must claim it for tenant A. The
// positive control: the same claim from tenant B finds B's seeded claim and
// is suppressed, so a check that cannot see the key cannot pass. Both
// transactions roll back.
func cascadeNoCollision(ctx context.Context, cl Cluster) error {
	pool, err := cl.Pool("nava_worker")
	if err != nil {
		return err
	}
	defer pool.Close()
	key := cascade.Key{RecordID: uuid.MustParse(SharedRecord), DefinitionID: uuid.MustParse(SharedDefinition)}
	claim := func(tenant string) (cascade.Claim, bool, error) {
		var (
			c       cascade.Claim
			claimed bool
		)
		err := tenancy.InTenantTx(ctx, pool, uuid.MustParse(tenant), func(tx pgx.Tx) error {
			var err error
			if c, claimed, err = cascade.ClaimStart(ctx, tx, key); err != nil {
				return err
			}
			return errRollback
		})
		if !errors.Is(err, errRollback) {
			return c, claimed, err
		}
		return c, claimed, nil
	}

	c, claimed, err := claim(TenantA)
	if err != nil {
		return fmt.Errorf("tenant A's claim: %w", err)
	}
	origin, _ := c.Causation()
	if !claimed || c.TenantID().String() != TenantA || origin.String() != SharedOrigin {
		return fmt.Errorf("tenant A's claim of tenant B's key: claimed %v, tenant %s, origin %s; want a claim of its own", claimed, c.TenantID(), origin)
	}
	c, claimed, err = claim(TenantB)
	if err != nil {
		return fmt.Errorf("control, tenant B's claim: %w", err)
	}
	if claimed || c.TenantID().String() != TenantB {
		return fmt.Errorf("control: tenant B's repeat claim of its own key was claimed %v, tenant %s; want suppressed", claimed, c.TenantID())
	}
	return nil
}
