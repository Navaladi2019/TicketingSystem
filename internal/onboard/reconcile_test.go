package onboard

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// recStore is a ReconcileStore over per-tenant maps that records the tenant id
// of every call, so a test can show a reconcile for A never names B.
type recStore struct {
	mu      sync.Mutex
	state   map[uuid.UUID]string
	pointer map[uuid.UUID]string
	touched []uuid.UUID
	locks   int
	failOn  uuid.UUID
}

func newRecStore(states map[uuid.UUID]string) *recStore {
	return &recStore{state: states, pointer: map[uuid.UUID]string{}}
}

func (s *recStore) touch(id uuid.UUID) {
	s.mu.Lock()
	s.touched = append(s.touched, id)
	s.mu.Unlock()
}

func (s *recStore) Lock(_ context.Context, id uuid.UUID) (func(), error) {
	s.touch(id)
	s.locks++
	return func() {}, nil
}

func (s *recStore) TenantState(_ context.Context, id uuid.UUID) (string, bool, error) {
	s.touch(id)
	st, ok := s.state[id]
	return st, ok, nil
}

func (s *recStore) HasAuditorGroup(_ context.Context, id uuid.UUID) (bool, error) {
	s.touch(id)
	_, ok := s.pointer[id]
	return ok, nil
}

func (s *recStore) RecordAuditorGroup(_ context.Context, id uuid.UUID, kc string) error {
	s.touch(id)
	if id == s.failOn {
		return errors.New("boom")
	}
	if _, ok := s.pointer[id]; !ok {
		s.pointer[id] = kc
	}
	return nil
}

type recDir struct {
	ensured []uuid.UUID
	err     error
}

func (d *recDir) EnsureAuditorGroup(_ context.Context, id uuid.UUID) (string, error) {
	d.ensured = append(d.ensured, id)
	return "kc-auditors-" + id.String(), d.err
}

func TestReconcileAuditorGroupCreatesThenIsANoOp(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	st, dir := newRecStore(map[uuid.UUID]string{a: "active"}), &recDir{}
	created, err := ReconcileAuditorGroup(t.Context(), st, dir, a)
	if err != nil || !created {
		t.Fatalf("first run = %v, %v; want created", created, err)
	}
	created, err = ReconcileAuditorGroup(t.Context(), st, dir, a)
	if err != nil || created {
		t.Fatalf("second run = %v, %v; want a no-op", created, err)
	}
	if len(dir.ensured) != 1 {
		t.Errorf("Keycloak ensured %d times, want 1 (the no-op makes no call)", len(dir.ensured))
	}
}

// A reconcile for tenant A never names tenant B, in the store or in Keycloak.
func TestReconcileAuditorGroupNeverTouchesAnotherTenant(t *testing.T) {
	t.Parallel()
	a, b := uuid.New(), uuid.New()
	st, dir := newRecStore(map[uuid.UUID]string{a: "active", b: "active"}), &recDir{}
	if _, err := ReconcileAuditorGroup(t.Context(), st, dir, a); err != nil {
		t.Fatal(err)
	}
	for _, id := range append(st.touched, dir.ensured...) {
		if id != a {
			t.Fatalf("reconcile for A touched %s", id)
		}
	}
	if _, ok := st.pointer[b]; ok {
		t.Error("B has a pointer after A's reconcile")
	}
}

func TestReconcileAuditorGroupRefusesNonActiveTenants(t *testing.T) {
	t.Parallel()
	prov, unknown := uuid.New(), uuid.New()
	st, dir := newRecStore(map[uuid.UUID]string{prov: "provisioning"}), &recDir{}
	for _, id := range []uuid.UUID{prov, unknown} {
		if _, err := ReconcileAuditorGroup(t.Context(), st, dir, id); !errors.Is(err, ErrReconcileNotActive) {
			t.Errorf("%s: err = %v, want ErrReconcileNotActive", id, err)
		}
	}
	if len(dir.ensured) != 0 {
		t.Error("Keycloak was called for a tenant that is not active")
	}
}

func TestReconcileAuditorGroupErrors(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	st := newRecStore(map[uuid.UUID]string{a: "active"})
	if _, err := ReconcileAuditorGroup(t.Context(), st, &recDir{err: errors.New("kc down")}, a); err == nil {
		t.Error("want the Keycloak error")
	}
	st.failOn = a
	if _, err := ReconcileAuditorGroup(t.Context(), st, &recDir{}, a); err == nil {
		t.Error("want the record error")
	}
}

func TestReconcileAuditorGroupsContinuesPastAFailure(t *testing.T) {
	t.Parallel()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	st := newRecStore(map[uuid.UUID]string{a: "active", b: "active", c: "active"})
	st.failOn = b
	st.pointer[c] = "kc-c"
	sum, err := ReconcileAuditorGroups(t.Context(), st, &recDir{}, slog.New(slog.DiscardHandler), []uuid.UUID{a, b, c})
	if err == nil {
		t.Fatal("want an error when one tenant failed")
	}
	if sum != (ReconcileSummary{Created: 1, Unchanged: 1, Failed: 1}) {
		t.Errorf("summary = %+v", sum)
	}
	if _, ok := st.pointer[b]; ok {
		t.Error("the failed tenant has a pointer")
	}
	sum, err = ReconcileAuditorGroups(t.Context(), newRecStore(nil), &recDir{}, slog.New(slog.DiscardHandler), nil)
	if err != nil || sum != (ReconcileSummary{}) {
		t.Errorf("empty list = %+v, %v", sum, err)
	}
}

func TestReconcileAuditorGroupLockError(t *testing.T) {
	t.Parallel()
	if _, err := ReconcileAuditorGroup(t.Context(), lockFails{}, &recDir{}, uuid.New()); err == nil {
		t.Error("want the lock error")
	}
}

type lockFails struct{ ReconcileStore }

func (lockFails) Lock(context.Context, uuid.UUID) (func(), error) { return nil, errors.New("lock") }

func TestPGStoreTenantStateAndHasAuditorGroup(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	t.Run("state found", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state FROM tenants": {cols: []uint32{oidText}, rows: [][]*string{{sp("active")}}}}, unbound{})
		if s, ok, err := st.TenantState(t.Context(), id); err != nil || !ok || s != "active" {
			t.Fatalf("= %q %v %v", s, ok, err)
		}
	})
	t.Run("state not found", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state FROM tenants": {cols: []uint32{oidText}}}, unbound{})
		if _, ok, err := st.TenantState(t.Context(), id); err != nil || ok {
			t.Fatalf("= %v %v", ok, err)
		}
	})
	t.Run("state error", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"SELECT state FROM tenants": {err: "boom"}}, unbound{})
		if _, _, err := st.TenantState(t.Context(), id); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("has row", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"FROM tenant_auditor_group": {cols: []uint32{16}, rows: [][]*string{{sp("t")}}}}, unbound{})
		if has, err := st.HasAuditorGroup(t.Context(), id); err != nil || !has {
			t.Fatalf("= %v %v", has, err)
		}
	})
	t.Run("has error", func(t *testing.T) {
		t.Parallel()
		st := scriptedStore(t, map[string]scripted{"FROM tenant_auditor_group": {err: "boom"}}, unbound{})
		if _, err := st.HasAuditorGroup(t.Context(), id); err == nil {
			t.Fatal("want an error")
		}
	})
}
