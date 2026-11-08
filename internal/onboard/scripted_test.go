package onboard

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
)

// pg answers by statement text: the longest key contained in the statement
// selects the answer, so the choice never depends on map order. No match is an
// error so a test fails loudly when the code sends something unexpected.
func pg(answers map[string]scripted) func(string) scripted {
	return func(sql string) scripted {
		best, found := "", false
		for k := range answers {
			if strings.Contains(sql, k) && (!found || len(k) > len(best)) {
				best, found = k, true
			}
		}
		if !found {
			return scripted{err: "unexpected statement: " + sql}
		}
		return answers[best]
	}
}

func scriptedStore(t *testing.T, answers map[string]scripted, b BindingReader) *PGStore {
	t.Helper()
	st, err := NewPGStore(scriptedPool(t, startFakePGServer(t, pg(answers))), b)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

type existing struct{ issuer, org string }

func (e existing) Binding(context.Context, uuid.UUID) (string, string, bool, error) {
	return e.issuer, e.org, true, nil
}

type unbound struct{}

func (unbound) Binding(context.Context, uuid.UUID) (string, string, bool, error) {
	return "", "", false, nil
}

func TestRegistryBoundIdPAliases(t *testing.T) {
	t.Parallel()
	run := func(t *testing.T, a scripted) ([]string, error) {
		t.Helper()
		r := NewRegistry(scriptedPool(t, startFakePGServer(t, pg(map[string]scripted{"idp_aliases": a}))))
		return r.BoundIdPAliases(t.Context(), uuid.New())
	}
	got, err := run(t, scripted{cols: []uint32{oidTexts}, rows: [][]*string{{sp("{idp-a,idp-b}")}, {sp("{idp-c}")}}})
	if err != nil || len(got) != 3 || got[0] != "idp-a" || got[2] != "idp-c" {
		t.Fatalf("aliases = %v, %v", got, err)
	}
	got, err = run(t, scripted{cols: []uint32{oidTexts}})
	if err != nil || len(got) != 0 {
		t.Fatalf("no binding = %v, %v", got, err)
	}
	// An unreadable row is an error, never a shorter alias list that would
	// make an IdP look unbound.
	if got, err := run(t, scripted{cols: []uint32{oidTexts}, rows: [][]*string{{sp("{unterminated")}}}); err == nil || len(got) != 0 {
		t.Fatalf("bad row = %v, %v", got, err)
	}
	if got, err := run(t, scripted{err: "boom"}); err == nil || len(got) != 0 {
		t.Fatalf("query error = %v, %v", got, err)
	}
}

func TestRegistryBinding(t *testing.T) {
	t.Parallel()
	run := func(t *testing.T, a scripted) (string, string, bool, error) {
		t.Helper()
		r := NewRegistry(scriptedPool(t, startFakePGServer(t, pg(map[string]scripted{"tenant_identity_binding": a}))))
		return r.Binding(t.Context(), uuid.New())
	}
	iss, org, found, err := run(t, scripted{cols: []uint32{oidText, oidText}, rows: [][]*string{{sp("https://kc/realms/nava"), sp("org-1")}}})
	if err != nil || !found || iss != "https://kc/realms/nava" || org != "org-1" {
		t.Fatalf("binding = %q %q %v %v", iss, org, found, err)
	}
	if _, _, found, err := run(t, scripted{cols: []uint32{oidText, oidText}}); err != nil || found {
		t.Fatalf("no row = %v, %v", found, err)
	}
	if _, _, found, err := run(t, scripted{err: "boom"}); err == nil || found {
		t.Fatalf("error = %v, %v", found, err)
	}
}

func TestPGStoreBeginBranches(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	noRow := scripted{cols: []uint32{oidText, oidText}}
	t.Run("insert tenant fails", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state, onboarding": noRow, "INSERT INTO tenants": {err: "boom"}}, noBindings{})
		if _, err := st.Begin(t.Context(), tenant, []byte("h"), []string{"a.test"}); err == nil || errors.Is(err, ErrDomainClaimed) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("another tenant owns the domain", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state, onboarding": noRow, "INSERT INTO tenants": {},
			"INSERT INTO tenant_email_domain": {err: "duplicate", code: "23505", cons: "tenant_email_domain_pkey"}}, noBindings{})
		if _, err := st.Begin(t.Context(), tenant, []byte("h"), []string{"a.test"}); !errors.Is(err, ErrDomainClaimed) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("another unique violation is not a domain claim", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state, onboarding": noRow, "INSERT INTO tenants": {},
			"INSERT INTO tenant_email_domain": {err: "duplicate", code: "23505", cons: "some_other_key"}}, noBindings{})
		if _, err := st.Begin(t.Context(), tenant, []byte("h"), []string{"a.test"}); err == nil || errors.Is(err, ErrDomainClaimed) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("domain insert fails", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state, onboarding": noRow, "INSERT INTO tenants": {},
			"INSERT INTO tenant_email_domain": {err: "boom"}}, noBindings{})
		if _, err := st.Begin(t.Context(), tenant, []byte("h"), []string{"a.test"}); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("same key, different body", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state, onboarding": {cols: []uint32{oidText, oidText},
			rows: [][]*string{{sp("provisioning"), sp(`\x6f74686572`)}}}}, noBindings{})
		if _, err := st.Begin(t.Context(), tenant, []byte("h"), nil); !errors.Is(err, ErrKeyReuse) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestPGStoreBindBranches(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	state := func(v string) scripted { return scripted{cols: []uint32{oidText}, rows: [][]*string{{sp(v)}}} }
	t.Run("state read fails", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state FROM tenants": {err: "boom"}}, noBindings{})
		if err := st.Bind(t.Context(), tenant, "iss", "org", "idp-x"); err == nil || errors.Is(err, ErrNotProvisioning) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("an active tenant is never bound", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state FROM tenants": state("active")}, noBindings{})
		if err := st.Bind(t.Context(), tenant, "iss", "org", "idp-x"); !errors.Is(err, ErrNotProvisioning) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("binding read fails", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state FROM tenants": state("provisioning")}, noBindings{})
		if err := st.Bind(t.Context(), tenant, "iss", "org", "idp-x"); err == nil || !strings.Contains(err.Error(), "binding read failed") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("bound to another org is a conflict, same pair is a retry", func(t *testing.T) {
		t.Parallel()
		answers := map[string]scripted{"SELECT state FROM tenants": state("provisioning")}
		if err := scriptedStore(t, answers, existing{"iss", "other"}).Bind(t.Context(), tenant, "iss", "org", "idp-x"); !errors.Is(err, ErrConflict) {
			t.Fatalf("conflict: %v", err)
		}
		if err := scriptedStore(t, answers, existing{"iss", "org"}).Bind(t.Context(), tenant, "iss", "org", "idp-x"); err != nil {
			t.Fatalf("retry: %v", err)
		}
	})
	t.Run("insert: unique violation is a conflict, other errors are errors", func(t *testing.T) {
		t.Parallel()
		answers := func(ins scripted) map[string]scripted {
			return map[string]scripted{"SELECT state FROM tenants": state("provisioning"), "INSERT INTO tenant_identity_binding": ins}
		}
		if err := scriptedStore(t, answers(scripted{err: "dup", code: "23505"}), unbound{}).Bind(t.Context(), tenant, "iss", "org", "idp-x"); !errors.Is(err, ErrConflict) {
			t.Fatalf("unique violation: %v", err)
		}
		if err := scriptedStore(t, answers(scripted{err: "boom"}), unbound{}).Bind(t.Context(), tenant, "iss", "org", "idp-x"); err == nil || errors.Is(err, ErrConflict) {
			t.Fatalf("other error: %v", err)
		}
		if err := scriptedStore(t, answers(scripted{}), unbound{}).Bind(t.Context(), tenant, "iss", "org", "idp-x"); err != nil {
			t.Fatalf("insert: %v", err)
		}
	})
}

func TestPGStoreRecordAdminGroupBranches(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	exists := func(v string) scripted { return scripted{cols: []uint32{oidBool}, rows: [][]*string{{sp(v)}}} }
	run := func(t *testing.T, answers map[string]scripted) error {
		t.Helper()
		return scriptedStore(t, answers, noBindings{}).RecordAdminGroup(t.Context(), tenant, "kc-g")
	}
	if err := run(t, map[string]scripted{"INSERT INTO groups": {err: "boom"}}); err == nil || errors.Is(err, ErrAdminGroupMissing) {
		t.Fatalf("groups insert: %v", err)
	}
	if err := run(t, map[string]scripted{"INSERT INTO groups": {}, "SELECT EXISTS": {err: "boom"}}); err == nil || errors.Is(err, ErrAdminGroupMissing) {
		t.Fatalf("exists read: %v", err)
	}
	// The stale-row case the runbook describes: no groups row carries the id,
	// and the tenant is not given an admin group that points nowhere.
	if err := run(t, map[string]scripted{"INSERT INTO groups": {}, "SELECT EXISTS": exists("f")}); !errors.Is(err, ErrAdminGroupMissing) {
		t.Fatalf("missing: %v", err)
	}
	if err := run(t, map[string]scripted{"INSERT INTO groups": {}, "SELECT EXISTS": exists("t"), "INSERT INTO tenant_admin_group": {}}); err != nil {
		t.Fatalf("recorded: %v", err)
	}
}

func TestPGStoreActivateBranches(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	state := func(v string) scripted { return scripted{cols: []uint32{oidText}, rows: [][]*string{{sp(v)}}} }
	run := func(t *testing.T, answers map[string]scripted) error {
		t.Helper()
		return scriptedStore(t, answers, noBindings{}).Activate(t.Context(), tenant)
	}
	// The fake reports 0 rows updated, so the state read tells the cases apart.
	upd := scripted{}
	if err := run(t, map[string]scripted{"UPDATE tenants": upd, "SELECT state FROM tenants": {cols: []uint32{oidText}}}); !errors.Is(err, ErrNotProvisioning) {
		t.Fatalf("no tenant row: %v", err)
	}
	if err := run(t, map[string]scripted{"UPDATE tenants": upd, "SELECT state FROM tenants": state("provisioning")}); !errors.Is(err, ErrNoTenantAdmin) {
		t.Fatalf("no admin group: %v", err)
	}
	if err := run(t, map[string]scripted{"UPDATE tenants": upd, "SELECT state FROM tenants": state("active")}); !errors.Is(err, ErrNotProvisioning) {
		t.Fatalf("already active: %v", err)
	}
	if err := run(t, map[string]scripted{"UPDATE tenants": upd, "SELECT state FROM tenants": {err: "boom"}}); err == nil ||
		errors.Is(err, ErrNotProvisioning) || errors.Is(err, ErrNoTenantAdmin) {
		t.Fatalf("state read: %v", err)
	}
	if err := run(t, map[string]scripted{"UPDATE tenants": {err: "boom"}}); err == nil || errors.Is(err, ErrNotProvisioning) {
		t.Fatalf("update: %v", err)
	}
}

func TestPGStoreRecordInviteConsumedStaysConsumed(t *testing.T) {
	t.Parallel()
	// The conflict update matches no row for a consumed invite: the command
	// tag says 0 rows and RecordInvite refuses rather than re-arm the admin.
	st := scriptedStore(t, map[string]scripted{"INSERT INTO pending_invite": {}}, noBindings{})
	err := st.RecordInvite(t.Context(), uuid.New(), "a@a.test", time.Now().Add(time.Hour))
	if !errors.Is(err, keycloak.ErrInviteConsumed) {
		t.Fatalf("err = %v", err)
	}
}
