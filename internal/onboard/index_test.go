package onboard

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// noIndex registers no field index.
type noIndex struct{}

func (noIndex) RegisterSeedFilterable(context.Context, uuid.UUID, string) error { return nil }

// fakeIndex records the (tenant, key) pairs it was asked to register and fails
// the ones in fail.
type fakeIndex struct {
	mu    sync.Mutex
	calls []string
	fail  map[string]error
	hook  func(uuid.UUID)
}

func (f *fakeIndex) RegisterSeedFilterable(_ context.Context, id uuid.UUID, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, key)
	if f.hook != nil {
		f.hook(id)
	}
	return f.fail[key]
}

func newSvcWithIndex(t *testing.T, idx SeedIndexer) (*Service, *fakeStore) {
	t.Helper()
	st := newStore()
	specs := NewSpecs()
	dir := &fakeDir{orgs: map[uuid.UUID]string{}, specs: specs}
	svc, err := New(Options{Store: st, Directory: dir, Finder: dir, Domains: st, ACL: &fakeACL{}, Index: idx, Specs: specs, Resolver: okResolver,
		Issuer: "https://kc/realms/nava", Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	return svc, st
}

// Onboarding registers every seed type's field index, in the tenant's
// provisioning window: after the seed, before Activate.
func TestOnboardRegistersSeedIndexesBeforeActivation(t *testing.T) {
	var st *fakeStore
	activeAtCall := true
	idx := &fakeIndex{hook: func(id uuid.UUID) { activeAtCall = st.active[id] }}
	var svc *Service
	svc, st = newSvcWithIndex(t, idx)
	if _, err := svc.Onboard(t.Context(), op1, "key-idx", []byte("h"), validReq()); err != nil {
		t.Fatal(err)
	}
	if want := metadata.TemplateKeys(); !slices.Equal(idx.calls, want) {
		t.Fatalf("registered %v, want every seed type %v", idx.calls, want)
	}
	if activeAtCall {
		t.Error("the field index was registered after the tenant was active")
	}
}

// A failed registration stops onboarding before Activate; the retry registers
// again (it is idempotent) and then activates.
func TestOnboardStopsBeforeActivationWhenIndexRegistrationFails(t *testing.T) {
	tenant := TenantID("op-1", "key-idx-fail")
	idx := &fakeIndex{fail: map[string]error{"request": errors.New("boom")}}
	svc, st := newSvcWithIndex(t, idx)
	if _, err := svc.Onboard(t.Context(), op1, "key-idx-fail", []byte("h"), validReq()); err == nil {
		t.Fatal("onboarding succeeded though the field index registration failed")
	}
	if st.active[tenant] {
		t.Fatal("the tenant was activated with an unregistered seed type")
	}
	delete(idx.fail, "request")
	if _, err := svc.Onboard(t.Context(), op1, "key-idx-fail", []byte("h"), validReq()); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !st.active[tenant] {
		t.Error("the retry did not activate the tenant")
	}
}

// The seed templates flag exactly the fields earlier kits filter on.
func TestSeedTemplatesFlagTheirFilteredFields(t *testing.T) {
	t.Parallel()
	want := map[string][]string{"incident": {"priority"}, "request": {"priority"}, "change": {"risk"}}
	for _, key := range metadata.TemplateKeys() {
		d, err := metadata.Template(key)
		if err != nil {
			t.Fatal(err)
		}
		c, err := metadata.Compile(d)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, f := range c.Filterable() {
			got = append(got, f.Key)
		}
		if !slices.Equal(got, want[key]) {
			t.Errorf("%s flags %v, want %v", key, got, want[key])
		}
	}
}
