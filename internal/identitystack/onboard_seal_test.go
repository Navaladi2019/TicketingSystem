//go:build identitystack

package identitystack

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestOnboardedTenantIsSealed (TICK-453, the check S15 acceptance relies
// on): a tenant onboarded through nava-onboard gets its audit chain sealed by
// the always-on nava-worker. Onboarding seeds the head and the first
// audit.seal start row (A6.2); the stack's relay starts that run, and its A1
// and A2 run in nava-worker. An entry written for the new tenant is sealed at
// seq 1, the head points past the first run (the lineage rescheduled itself
// through the outbox), and the first start row is dispatched.
//
// nava-worker must be up and be the only worker on nava-tq: every worker
// that polls a queue registers the same workflow types (the nava-tq rule,
// internal/temporalstack/main_test.go). This test starts no worker.
func TestOnboardedTenantIsSealed(t *testing.T) {
	key, tenant, domain, email := newOnboarding(t)
	mustOnboard(t, operatorToken(t), key, onboardBody(email, domain), http.StatusCreated, tenant)

	firstRaw := psqlSuper(t, fmt.Sprintf("SELECT next_seal_outbox_id FROM audit_chain_head WHERE tenant_id = '%s'", tenant))
	first, err := uuid.Parse(firstRaw)
	if err != nil {
		t.Fatalf("onboarding seeded no seal lineage for %s: head next_seal_outbox_id = %q", tenant, firstRaw)
	}
	psqlSuper(t, fmt.Sprintf(`INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
VALUES ('%s', '%s', 'audit', 'export', 'audit_export', gen_random_uuid(), 'system', 'system')`, tenant, uuid.Must(uuid.NewV7())))

	q := fmt.Sprintf(`SELECT (SELECT count(*) FROM audit_log WHERE tenant_id = '%[1]s' AND sealed_at IS NULL)
	|| '|' || h.seq || '|' || coalesce(h.next_seal_outbox_id::text, '') FROM audit_chain_head h WHERE h.tenant_id = '%[1]s'`, tenant)
	var got string
	sealed := false
	// The first run may have sealed before the entry was written; the next
	// one follows after seal_interval_s (10 s, doubling while idle), so 3
	// minutes covers several runs.
	for deadline := time.Now().Add(3 * time.Minute); time.Now().Before(deadline); time.Sleep(2 * time.Second) {
		got = psqlSuper(t, q)
		f := strings.Split(got, "|")
		if len(f) == 3 && f[0] == "0" && f[1] == "1" && f[2] != "" && f[2] != first.String() {
			sealed = true
			break
		}
	}
	if !sealed {
		t.Fatalf("onboarded tenant %s not sealed within 3m: unsealed|head seq|next run = %q, want 0|1|<not %s>", tenant, got, first)
	}
	if d := psqlSuper(t, fmt.Sprintf("SELECT dispatched_at IS NOT NULL FROM outbox WHERE tenant_id = '%s' AND id = '%s'", tenant, first)); d != "t" {
		t.Errorf("the first audit.seal start row is not dispatched (%q)", d)
	}
	if n := psqlSuper(t, fmt.Sprintf("SELECT count(*) FROM audit_log WHERE tenant_id = '%s' AND seq = 1 AND entry_hash IS NOT NULL", tenant)); n != "1" {
		t.Errorf("%s entries at seq 1 with an entry_hash, want 1", n)
	}
}
