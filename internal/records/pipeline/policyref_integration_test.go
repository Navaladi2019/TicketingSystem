//go:build integration

// TICK-957 (ADR-010 D9): the workflow's named-policy test through the real
// PolicyTester, as nava_worker under RLS: current version, tenant isolation
// (same entity type key and policy name in two tenants with different rules),
// and a missing reference that denies, counts and audits.

package pipeline_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
)

//nolint:gochecknoglobals // one meter provider per test binary, as RegisterMetrics is per process
var refMetrics = sync.OnceValue(func() *sdkmetric.ManualReader {
	r := sdkmetric.NewManualReader()
	if err := authz.RegisterMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))); err != nil {
		panic(err)
	}
	return r
})

// refMissing is authz_policy_ref_missing_total for an entity type.
func refMissing(t *testing.T, entityType string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := refMetrics().Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != authz.PolicyRefMissingMetric {
				continue
			}
			sum, _ := m.Data.(metricdata.Sum[int64])
			for _, dp := range sum.DataPoints {
				if v, ok := dp.Attributes.Value("entity_type"); ok && v.AsString() == entityType {
					return dp.Value
				}
			}
		}
	}
	return 0
}

// refAudit counts the tenant's audit entries a missing reference wrote.
func refAudit(t *testing.T, e env, tenant uuid.UUID) int {
	t.Helper()
	return count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND source = 'workflow'
		AND subject_type = 'acl_policy' AND actor_system = 'workflow'`, tenant)
}

// refWorld is a tenant with the aclcase type, a record whose requester is u1
// and whose assignee is u2, and an open run on it.
type refWorld struct {
	f      tenantFixture
	u1, u2 uuid.UUID
	record uuid.UUID
	run    uuid.UUID
}

func seedRefWorld(t *testing.T, e env, requesterSet, assigneeSet bool, more ...metadata.Definition) refWorld {
	t.Helper()
	w := refWorld{f: seedTenantWith(t, e, append([]metadata.Definition{aclDefinition()}, more...)...), u2: uuid.New()}
	w.u1 = w.f.principal.PrincipalID
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, $2, $3, true, now())`,
		w.f.id, w.u2, "kc-"+w.u2.String())
	data := map[string]any{"title": "t"}
	if requesterSet {
		data["requester"] = w.u1.String()
	}
	if assigneeSet {
		data["assignee"] = w.u2.String()
	}
	res, err := write(context.Background(), e, newPipeline(e, nil), w.f.id, pipeline.Command{Op: pipeline.OpCreate, EntityKey: aclKey, Data: data, Actor: w.f.principal, Source: pipeline.SourceUser})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	w.record = res.RecordID
	w.run, _ = openRun(t, e, w.f, w.record, false)
	return w
}

// publish replaces the entity type's access with rules (a new version).
func (w refWorld) publish(t *testing.T, e env, rules ...authz.Rule) {
	t.Helper()
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		_, err := acltest.Publish(context.Background(), tx, aclKey, rules...)
		return err
	})
}

// ask runs the PolicyTester as the world's run.
func (w refWorld) ask(t *testing.T, tester recordports.PolicyTester, tenant, record uuid.UUID, entityType, policy, who string) (bool, error) {
	t.Helper()
	var got bool
	err := asRun(t, w.f.id, w.run, func(ctx context.Context) (err error) {
		got, err = tester.Test(ctx, tenant, interp.PolicyTest{RecordID: record, EntityType: entityType, Policy: policy, Who: who, Op: authz.OpWrite})
		return err
	})
	return got, err
}

// test asks about the world's own tenant and record.
func (w refWorld) test(t *testing.T, tester recordports.PolicyTester, entityType, policy, who string) bool {
	t.Helper()
	got, err := w.ask(t, tester, w.f.id, w.record, entityType, policy, who)
	if err != nil {
		t.Fatalf("Test(%s, %s, %s): %v", entityType, policy, who, err)
	}
	return got
}

func policyTester(t *testing.T, e env) recordports.PolicyTester {
	t.Helper()
	refMetrics()
	return recordports.PolicyTester{Pool: newWorker(t, e).pool, Policies: authz.PolicyStore{}}
}

//nolint:gochecknoglobals // read-only fixtures
var (
	editByRequester = authz.Rule{ID: "edit", Ops: []string{authz.OpWrite}, Who: authz.Subject{Requester: true}}
	editByAssignee  = authz.Rule{ID: "edit", Ops: []string{authz.OpWrite}, Who: authz.Subject{Assignee: true}}
)

// Two tenants define the same entity type key and policy name with different
// rules: each run sees only its own tenant's rule, for both who values.
func TestPolicyTest_SameNameInTwoTenantsSeesOnlyItsOwn(t *testing.T) {
	e := migrated(t)
	tester := policyTester(t, e)
	a, b := seedRefWorld(t, e, true, true), seedRefWorld(t, e, true, true)
	a.publish(t, e, editByRequester)
	b.publish(t, e, editByAssignee)
	for _, c := range []struct {
		name string
		w    refWorld
		who  string
		want bool
	}{
		{"A requester", a, "requester", true},
		{"A assignee", a, "assignee", false},
		{"B requester", b, "requester", false},
		{"B assignee", b, "assignee", true},
	} {
		if got := c.w.test(t, tester, aclKey, "edit", c.who); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	if n := refAudit(t, e, a.f.id) + refAudit(t, e, b.f.id); n != 0 {
		t.Errorf("%d missing-reference entries for references that resolved", n)
	}
}

// A run of tenant A naming an entity type key that only tenant B has, or a
// policy name the type lacks, or a type other than the record's: false,
// counted, audited in A, nothing in B.
func TestPolicyTest_MissingDeniesCountsAndAudits(t *testing.T) {
	e := migrated(t)
	tester := policyTester(t, e)
	onlyB := aclDefinition()
	onlyB.Key, onlyB.NumberPrefix = "onlyb", "OB"
	a, b := seedRefWorld(t, e, true, true), seedRefWorld(t, e, true, true, onlyB) // tenant B alone has the type onlyb
	a.publish(t, e, editByRequester)
	b.publish(t, e, editByRequester)

	for _, c := range []struct{ name, entity, policy string }{
		{"entity type only tenant B has", "onlyb", "edit"},
		{"policy the type lacks", aclKey, "gone"},
		{"another entity type than the record's", "other", "edit"},
	} {
		beforeN, beforeM := refAudit(t, e, a.f.id), refMissing(t, aclKey)
		if a.test(t, tester, c.entity, c.policy, "requester") {
			t.Errorf("%s: allowed", c.name)
		}
		if n := refAudit(t, e, a.f.id); n != beforeN+1 {
			t.Errorf("%s: %d audit entries, want %d", c.name, n, beforeN+1)
		}
		if m := refMissing(t, aclKey); m != beforeM+1 {
			t.Errorf("%s: authz_policy_ref_missing_total %d, want %d", c.name, m, beforeM+1)
		}
	}
	if n := refAudit(t, e, b.f.id); n != 0 {
		t.Errorf("tenant B has %d entries from tenant A's run", n)
	}
}

// A policy narrowed or removed after the run started applies at the next
// test: the current version is read every time, never a pinned one.
func TestPolicyTest_NarrowedPolicyAppliesAtTheNextTest(t *testing.T) {
	e := migrated(t)
	tester := policyTester(t, e)
	w := seedRefWorld(t, e, true, true)
	w.publish(t, e, editByRequester)
	if !w.test(t, tester, aclKey, "edit", "requester") {
		t.Fatal("denied before the policy was narrowed")
	}
	w.publish(t, e, editByAssignee) // narrowed: the requester no longer edits
	if w.test(t, tester, aclKey, "edit", "requester") {
		t.Error("allowed after the policy was narrowed")
	}
	w.publish(t, e, authz.Rule{ID: "other", Ops: []string{authz.OpWrite}, Who: authz.Subject{Requester: true}}) // removed
	if w.test(t, tester, aclKey, "edit", "requester") {
		t.Error("allowed after the policy was removed")
	}
	if n := refAudit(t, e, w.f.id); n != 1 {
		t.Errorf("%d missing-reference entries, want 1 (the removal)", n)
	}
}

// A NULL requester or assignee is false and is not a missing reference.
func TestPolicyTest_NullPrincipalIsFalse(t *testing.T) {
	e := migrated(t)
	tester := policyTester(t, e)
	w := seedRefWorld(t, e, false, false)
	w.publish(t, e, editByRequester)
	for _, who := range []string{"requester", "assignee"} {
		if w.test(t, tester, aclKey, "edit", who) {
			t.Errorf("%s NULL: allowed", who)
		}
	}
	if n := refAudit(t, e, w.f.id); n != 0 {
		t.Errorf("%d missing-reference entries for a NULL principal", n)
	}
}

// The activity's tenant must be the run's: a run of tenant A cannot ask about
// tenant B's record, nor under B's tenant id, and a run with no workflow_ref
// of the tenant is refused.
func TestPolicyTest_ForeignRecordAndTenantAreRefused(t *testing.T) {
	e := migrated(t)
	tester := policyTester(t, e)
	a, b := seedRefWorld(t, e, true, true), seedRefWorld(t, e, true, true)
	a.publish(t, e, editByRequester)
	b.publish(t, e, editByRequester)
	if _, err := a.ask(t, tester, a.f.id, b.record, aclKey, "edit", "requester"); !errors.Is(err, interp.ErrNotFound) {
		t.Errorf("tenant B's record from A's run: %v, want interp.ErrNotFound", err)
	}
	if _, err := a.ask(t, tester, b.f.id, b.record, aclKey, "edit", "requester"); !errors.Is(err, interp.ErrForbidden) {
		t.Errorf("tenant B's id from A's run: %v, want interp.ErrForbidden", err)
	}
	ghost := a
	ghost.run = uuid.Must(uuid.NewV7())
	if _, err := ghost.ask(t, tester, a.f.id, a.record, aclKey, "edit", "requester"); !errors.Is(err, interp.ErrForbidden) {
		t.Errorf("a run with no workflow_ref: %v, want interp.ErrForbidden", err)
	}
}
