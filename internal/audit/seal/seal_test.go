package seal

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// R16: 10 s after a run that sealed anything, otherwise doubled up to 300 s,
// and never outside the column's CHECK bounds.
func TestNextInterval(t *testing.T) {
	cases := []struct {
		current int32
		sealed  bool
		want    int32
	}{
		{10, true, 10}, {300, true, 10}, {80, true, 10},
		{10, false, 20}, {20, false, 40}, {160, false, 300}, {299, false, 300}, {300, false, 300},
		{0, false, 10}, {5, false, 10}, {1 << 30, false, 300},
	}
	for _, c := range cases {
		if got := NextInterval(c.current, c.sealed); got != c.want {
			t.Errorf("NextInterval(%d, %v) = %d, want %d", c.current, c.sealed, got, c.want)
		}
	}
}

func TestRunOf(t *testing.T) {
	tenant, o := uuid.New(), uuid.Must(uuid.NewV7())
	got, err := RunOf("t:" + tenant.String() + ":audit.seal:" + o.String())
	if err != nil || got != o {
		t.Fatalf("RunOf = %s, %v; want %s", got, err, o)
	}
	for _, id := range []string{
		"",
		"audit.seal:" + o.String(),
		"t:" + tenant.String() + ":nava.interpreter:" + o.String(),
		"t:" + tenant.String() + ":audit.seal:",
		"t:" + tenant.String() + ":audit.seal:" + o.String() + ":x",
		"t:" + tenant.String() + ":audit.seal:" + uuid.Nil.String(),
		"t:" + tenant.String() + ":audit.seal:urn:uuid:" + o.String(),
		"t:" + tenant.String() + ":audit.seal:" + "{" + o.String() + "}",
		"t:" + tenant.String() + ":audit.sealx:" + o.String(),
	} {
		if _, err := RunOf(id); err == nil {
			t.Errorf("RunOf(%q) accepted", id)
		}
	}
}

// workerguard records only references: A1's result, stale or not, must pass
// the payload rule the guard applies (outbox.ValidatePayload).
func TestRescheduledIsReferencesOnly(t *testing.T) {
	next := uuid.Must(uuid.NewV7())
	for _, r := range []Rescheduled{{}, {NextOutboxID: &next}} {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := outbox.ValidatePayload(b); err != nil {
			t.Errorf("%s: %v", b, err)
		}
	}
}

// Without a verified tenant on the context neither activity opens a
// transaction (the pool is nil, so a query would panic).
func TestNoTenantNoTransaction(t *testing.T) {
	ctx := context.Background()
	if _, err := Reschedule(ctx, nil, uuid.New()); !errors.Is(err, tenancy.ErrNoTenant) {
		t.Errorf("Reschedule without a tenant: %v, want tenancy.ErrNoTenant", err)
	}
	if _, err := Seal(ctx, nil); !errors.Is(err, tenancy.ErrNoTenant) {
		t.Errorf("Seal without a tenant: %v, want tenancy.ErrNoTenant", err)
	}
}

// A digest column that is not 32 bytes (its CHECK forbids it) is unsealable,
// not truncated or padded.
func TestHash32(t *testing.T) {
	if h, err := hash32(nil); h != nil || err != nil {
		t.Errorf("hash32(nil) = %v, %v; want nil, nil", h, err)
	}
	b := make([]byte, 32)
	b[31] = 7
	if h, err := hash32(b); err != nil || h == nil || h[31] != 7 {
		t.Errorf("hash32(32 bytes) = %v, %v", h, err)
	}
	for _, n := range []int{0, 31, 33} {
		if _, err := hash32(make([]byte, n)); err == nil {
			t.Errorf("hash32(%d bytes) accepted", n)
		}
	}
}
