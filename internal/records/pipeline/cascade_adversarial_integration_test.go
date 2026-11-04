//go:build integration

package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Security review of P3-12 (TICK-88): tenant B sends a too-deep update that
// names tenant A's record id. The rejection path reads the record on the
// TxRunner to pick the cascade_too_deep entry's subject (ADR-004-E1), so it
// must stay inside tenant B:
//   - the error is the same ErrCascadeTooDeep that a random, nonexistent id
//     gets (no existence oracle for tenant A's ids);
//   - tenant B's entries have the cascade key as subject and name no record
//     in their diff, and tenant A gets no entry;
//   - tenant A's record is untouched.
func TestCI10_TooDeepUpdateOfAnotherTenantsRecordLeaksNothing(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	p := pipeline.New(authz.TenantMemberAuthorizer{}, entitytypes.Store{}, &countingSink{},
		pipeline.PoolRunner{Pool: e.app}, pipeline.SystemClock{})
	ctx := context.Background()
	victim := mustCreate(t, e, p, a, map[string]any{"title": "tenant A only"})
	auditA := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND domain = 'record'`, a.id)

	tooDeep := outbox.Causation{OriginID: uuid.New(), Depth: config.CascadeDepthLimit + 1}
	attempt := func(record uuid.UUID, step string) error {
		return tenancy.InTenantTx(ctx, e.app, b.id, func(tx pgx.Tx) error {
			_, err := p.Write(ctx, tx, workflowCmd(b, record, 1, tooDeep, step))
			return err
		})
	}

	errVictim := attempt(victim.RecordID, "cross-tenant")
	errRandom := attempt(uuid.New(), "nonexistent")
	if !errors.Is(errVictim, outbox.ErrCascadeTooDeep) || !errors.Is(errRandom, outbox.ErrCascadeTooDeep) {
		t.Fatalf("tenant A's id: %v; a random id: %v; want ErrCascadeTooDeep for both", errVictim, errRandom)
	}
	if errVictim.Error() != errRandom.Error() {
		t.Errorf("tenant A's id and a random id are told apart: %q vs %q", errVictim, errRandom)
	}

	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = 'cascade_too_deep'
		AND subject_type = 'cascade_key' AND diff::text NOT LIKE '%record_id%'`, b.id); n != 2 {
		t.Errorf("tenant B has %d cascade_too_deep entries with no record, want 2", n)
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id <> $2
		AND (subject_id = $1 OR diff::text LIKE '%' || $1::text || '%')`, victim.RecordID, a.id); n != 0 {
		t.Errorf("%d entries outside tenant A name tenant A's record, want 0", n)
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND domain = 'record'`, a.id); n != auditA {
		t.Errorf("tenant A's audit_log went from %d to %d entries", auditA, n)
	}
	if _, _, _, v, _ := storedData(t, e, victim.RecordID); v != victim.Version {
		t.Errorf("tenant A's record is at version %d, want %d", v, victim.Version)
	}
}
