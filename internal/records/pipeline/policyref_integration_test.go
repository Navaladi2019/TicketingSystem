//go:build integration

// TICK-957 (ADR-010 D9): the workflow's named-policy test through the real
// PolicyTester, as nava_worker under RLS: current version, tenant isolation
// (same entity type key and policy name in two tenants with different rules),
// and a missing reference that denies, counts and audits.

package pipeline_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
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
		AND subject_type = 'acl_policy' AND domain = 'config' AND action = 'policy_ref_refused' AND actor_system = 'workflow'`, tenant)
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
	// A system write: a user create defaults requester_id to the caller, so a
	// NULL requester or assignee needs a writer that is not a user.
	res, err := write(context.Background(), e, newPipeline(e, nil), w.f.id, pipeline.Command{Op: pipeline.OpCreate, EntityKey: aclKey, Data: data, Actor: w.f.system(), Source: pipeline.SourceSystem})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	w.record = res.RecordID
	var reqNull, asgNull bool
	if err := e.super.QueryRow(context.Background(), `SELECT requester_id IS NULL, assignee_id IS NULL FROM records WHERE tenant_id = $1 AND id = $2`,
		w.f.id, w.record).Scan(&reqNull, &asgNull); err != nil {
		t.Fatal(err)
	}
	if reqNull != !requesterSet || asgNull != !assigneeSet {
		t.Fatalf("seeded requester NULL=%v assignee NULL=%v, want %v and %v", reqNull, asgNull, !requesterSet, !assigneeSet)
	}
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

// TICK-1114 (ADR-010 D9, D7): a workflow policy test resolves an under
// condition as a record read does. authz.TestPolicyRef runs as nava_worker
// under RLS over a real org-unit tree; the closure and org_unit_members reads
// are the grants db/grants.yaml gives the worker for this. An answer never
// counts a voided policy (authz_policy_invalid_total does not move).

// invalidTotal is authz_policy_invalid_total over every entity type.
func invalidTotal(t *testing.T) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := refMetrics().Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	var n int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != authz.PolicyInvalidMetric {
				continue
			}
			sum, _ := m.Data.(metricdata.Sum[int64])
			for _, dp := range sum.DataPoints {
				n += dp.Value
			}
		}
	}
	return n
}

// publishDoc publishes a format-2 document as the entity type's next version
// of the tenant, with the audit entry the commit check needs.
func publishDoc(t *testing.T, e env, tenant uuid.UUID, doc string) {
	t.Helper()
	inTx(t, e, tenant, func(tx pgx.Tx) error {
		ctx := context.Background()
		var def uuid.UUID
		var have int
		if err := tx.QueryRow(ctx, `SELECT d.id, coalesce(max(a.version), 0) FROM entity_definitions d
			LEFT JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
			WHERE d.tenant_id = $1 AND d.key = $2 GROUP BY d.id`, tenant, aclKey).Scan(&def, &have); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
			VALUES ($1, $2, $3, $4::text::jsonb, '[]')`, tenant, def, have+1, doc); err != nil {
			return err
		}
		sum := sha256.Sum256([]byte(doc))
		before, after := int64(have), int64(have+1)
		sys := "tick-1114"
		_, err := audit.Append(ctx, tx, audit.Entry{Domain: audit.DomainConfig, Action: audit.ActionPublish,
			SubjectType: audit.SubjectACLPolicy, SubjectID: def, VersionBefore: &before, VersionAfter: &after,
			Source: audit.SourceSystem, Actor: audit.Actor{System: &sys}, ContentSHA256: &sum})
		return err
	})
}

// onePolicy is a format-2 document with the one policy "edit" (read and
// write) under the condition when.
func onePolicy(when string) string {
	return `{"format":2,"conditions":[],"policies":[{"name":"edit","ops":["read","write"],"when":` + when + `}]}`
}

// setRequester makes who the requester of the records, past the audit commit
// check (replica role), as seedUnderWorld sets org_unit_id.
func setRequester(t *testing.T, e env, tenant, who uuid.UUID, records ...uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	for _, id := range records {
		if _, err := tx.Exec(ctx, `UPDATE records SET requester_id = $3 WHERE tenant_id = $1 AND id = $2`, tenant, id, who); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

// testRef is authz.TestPolicyRef as nava_worker in the tenant's transaction,
// and how much authz_policy_invalid_total moved.
func testRef(t *testing.T, e env, tenantTx, record uuid.UUID, op string) (got bool, invalid int64, err error) {
	t.Helper()
	pool := newWorker(t, e).pool
	before := invalidTotal(t)
	err = tenancy.InTenantTx(context.Background(), pool, tenantTx, func(tx pgx.Tx) (err error) {
		got, err = authz.TestPolicyRef(context.Background(), tx, authz.PolicyStore{},
			authz.PolicyTest{RecordID: record, EntityType: aclKey, Policy: "edit", Who: authz.WhoRequester, Op: op})
		return err
	})
	return got, invalidTotal(t) - before, err
}

func TestPolicyTest_UnderResolvesAsARecordReadDoes(t *testing.T) {
	e := migrated(t)
	refMetrics()
	w := seedUnderWorld(t, e) // u1 is a member of A and L; Lx is tombstoned below L
	setRequester(t, e, w.f.id, w.u1, w.records...)
	caller := `{"under":{"of":"caller"}}`
	requester := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	of := func(name string) string { return `{"under":{"of":["` + w.unit[name].String() + `"]}}` }

	// canRead is what ACL.CanRead says for the same record and principal under
	// a document with only this policy.
	canRead := func(record uuid.UUID) (ok bool) {
		inTx(t, e, w.f.id, func(tx pgx.Tx) error {
			row, err := pipelineGet(context.Background(), tx, w.f.id, record)
			if err != nil {
				return err
			}
			p := identity.Principal{TenantID: w.f.id, PrincipalID: w.u1}
			err = authz.ACL{Policies: authz.PolicyStore{}}.CanRead(context.Background(), tx, p, aclKey, row)
			ok = err == nil
			if err != nil && !errors.Is(err, authz.ErrNotVisible) {
				return err
			}
			return nil
		})
		return ok
	}

	for _, c := range []struct {
		name, when, unit string
		want             bool
	}{
		{"under caller: unit in the subtree", caller, "A1", true},
		{"under caller: unit outside it", caller, "R", false},
		{"under caller: unit tombstoned inside the subtree", caller, "Lx", false},
		{"under caller: NULL org_unit_id", caller, "none", false},
		{"any[is_requester, under caller]: requester, unit outside", `{"any":[` + requester + `,` + caller + `]}`, "R", true},
		{"under of [A]: unit in the subtree", of("A"), "A1", true},
		{"under of [A]: unit outside the subtree", of("A"), "R", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			publishDoc(t, e, w.f.id, onePolicy(c.when))
			got, invalid, err := testRef(t, e, w.f.id, w.record[c.unit], authz.OpWrite)
			if err != nil {
				t.Fatalf("TestPolicyRef: %v", err)
			}
			if got != c.want {
				t.Errorf("answer %v, want %v", got, c.want)
			}
			if invalid != 0 {
				t.Errorf("authz_policy_invalid_total moved by %d", invalid)
			}
		})
	}

	// inside is a literal: with r.org_unit_id dropped from policyTestRecordSQL
	// (the row's unit unset) not under is true for a record inside the unit,
	// an over-grant a comparison with ACL.CanRead alone would not name.
	for _, c := range []struct {
		unit   string
		inside bool
	}{{"A1", true}, {"R", false}, {"Lx", false}, {"none", false}} {
		unit := c.unit
		t.Run("not under caller: "+unit, func(t *testing.T) {
			publishDoc(t, e, w.f.id, onePolicy(`{"not":`+caller+`}`))
			want := canRead(w.record[unit])
			got, invalid, err := testRef(t, e, w.f.id, w.record[unit], authz.OpRead)
			if err != nil {
				t.Fatalf("TestPolicyRef: %v", err)
			}
			if got != want {
				t.Errorf("answer %v, ACL.CanRead says %v", got, want)
			}
			if c.inside && got {
				t.Error("not under caller allowed a record inside the caller's unit")
			}
			if invalid != 0 {
				t.Errorf("authz_policy_invalid_total moved by %d", invalid)
			}
		})
	}

	t.Run("transaction of tenant B, record of tenant A", func(t *testing.T) {
		publishDoc(t, e, w.f.id, onePolicy(caller))
		b := seedUnderWorld(t, e)
		_, _, err := testRef(t, e, b.f.id, w.record["A1"], authz.OpWrite)
		if !errors.Is(err, authz.ErrPolicyTestRecord) {
			t.Errorf("error %v, want ErrPolicyTestRecord", err)
		}
	})

	t.Run("another tenant's tree and membership do not grant", func(t *testing.T) {
		publishDoc(t, e, w.f.id, onePolicy(caller))
		// u3 is a principal of tenant A with no unit; the same principal id in
		// tenant B is a member of B's root, which has the same unit names.
		b := seedUnderWorld(t, e)
		u3 := uuid.New()
		for _, tenant := range []uuid.UUID{w.f.id, b.f.id} {
			mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, $2, $3, true, now())`,
				tenant, u3, "kc-"+tenant.String()+"-"+u3.String())
		}
		mustExec(t, e.super, `INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id) VALUES ($1, $2, $3)`, b.f.id, b.unit["R"], u3)
		setRequester(t, e, w.f.id, u3, w.record["A1"])
		got, invalid, err := testRef(t, e, w.f.id, w.record["A1"], authz.OpWrite)
		if err != nil {
			t.Fatalf("TestPolicyRef: %v", err)
		}
		if got {
			t.Error("allowed on the strength of another tenant's membership")
		}
		if invalid != 0 {
			t.Errorf("authz_policy_invalid_total moved by %d", invalid)
		}
	})
}
