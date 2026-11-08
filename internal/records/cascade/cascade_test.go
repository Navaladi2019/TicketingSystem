package cascade_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
)

// The claim against a real database (concurrency, RLS, the key) is tested
// in internal/records/pipeline's integration tests; these pin ClaimStart's
// refusals and error mapping on a scripted transaction.

var errBoom = errors.New("boom")

type rowFunc func(dest ...any) error

func (f rowFunc) Scan(dest ...any) error { return f(dest...) }

// claimTx answers the record lock with lock and the claim with claim.
type claimTx struct {
	pgx.Tx // nil: any method not overridden panics, which fails the test
	lock   rowFunc
	claim  rowFunc
	ran    []string
}

func (x *claimTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	// The lock names the transaction's tenant (TICK-480): tenancy.TenantOf
	// reads it first. It is not one of the statements under test.
	if strings.Contains(sql, "current_setting(") {
		return rowFunc(func(dest ...any) error { *dest[0].(*string) = txTenant; return nil })
	}
	x.ran = append(x.ran, sql)
	if strings.Contains(sql, "FOR UPDATE") {
		return x.lock
	}
	return x.claim
}

// txTenant is the tenant every claimTx is scoped to.
const txTenant = "0190a000-0000-7000-8000-00000000c0de"

func failWith(err error) rowFunc { return func(...any) error { return err } }

func recordRow(tenant uuid.UUID, origin *uuid.UUID, depth *int16) rowFunc {
	return func(dest ...any) error {
		*dest[0].(*uuid.UUID) = tenant
		*dest[1].(**uuid.UUID) = origin
		*dest[2].(**int16) = depth
		return nil
	}
}

func TestClaimStartRefusesAnIncompleteKey(t *testing.T) {
	for name, k := range map[string]cascade.Key{
		"no record":     {DefinitionID: uuid.New()},
		"no definition": {RecordID: uuid.New()},
	} {
		t.Run(name, func(t *testing.T) {
			// nil tx: a statement would panic.
			c, claimed, err := cascade.ClaimStart(t.Context(), nil, k)
			if !errors.Is(err, cascade.ErrInvalidKey) || claimed || c != (cascade.Claim{}) {
				t.Fatalf("ClaimStart = %+v, %v, %v; want ErrInvalidKey", c, claimed, err)
			}
		})
	}
}

func TestClaimStartErrors(t *testing.T) {
	tenant, origin := uuid.New(), uuid.New()
	depth := int16(2)
	k := cascade.Key{RecordID: uuid.New(), DefinitionID: uuid.New()}
	cases := []struct {
		name    string
		lock    rowFunc
		claim   rowFunc
		is      error
		message string
	}{
		// Another tenant's record is invisible under RLS: not found.
		{"record not found", failWith(pgx.ErrNoRows), nil, cascade.ErrRecordNotFound, "cascade: record not found"},
		{"lock fails", failWith(errBoom), nil, errBoom, "cascade: lock record: boom"},
		{"no origin", recordRow(tenant, nil, &depth), nil, cascade.ErrNoOrigin, "cascade: record has no causation origin"},
		{"nil origin", recordRow(tenant, &uuid.Nil, &depth), nil, cascade.ErrNoOrigin, "cascade: record has no causation origin"},
		{"no depth", recordRow(tenant, &origin, nil), nil, cascade.ErrNoOrigin, "cascade: record has no causation origin"},
		{"claim fails", recordRow(tenant, &origin, &depth), failWith(errBoom), errBoom, "cascade: claim start: boom"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tx := &claimTx{lock: c.lock, claim: c.claim}
			got, claimed, err := cascade.ClaimStart(t.Context(), tx, k)
			if !errors.Is(err, c.is) || err.Error() != c.message {
				t.Fatalf("err = %v, want %q", err, c.message)
			}
			if claimed || got != (cascade.Claim{}) {
				t.Errorf("claim = %+v, %v; want none", got, claimed)
			}
			if c.claim == nil && len(tx.ran) != 1 {
				t.Errorf("statements after the refusal: %q", tx.ran[1:])
			}
		})
	}
}

// The origin, depth and tenant come from the record row, never the caller.
func TestClaimStartReadsTheKeyFromTheRecord(t *testing.T) {
	tenant, origin, outboxID := uuid.New(), uuid.New(), uuid.New()
	depth := int16(3)
	k := cascade.Key{RecordID: uuid.New(), DefinitionID: uuid.New()}

	tx := &claimTx{lock: recordRow(tenant, &origin, &depth), claim: func(dest ...any) error {
		*dest[0].(*uuid.UUID) = outboxID
		return nil
	}}
	c, claimed, err := cascade.ClaimStart(t.Context(), tx, k)
	if err != nil || !claimed || !c.Claimed() || c.OutboxID() != outboxID {
		t.Fatalf("ClaimStart = %+v, %v, %v; want claimed with the returned outbox id", c, claimed, err)
	}
	if o, d := c.Causation(); o != origin || d != depth || c.TenantID() != tenant || c.RecordID() != k.RecordID || c.DefinitionID() != k.DefinitionID {
		t.Errorf("claim = %+v, want the record's tenant, origin and depth", c)
	}

	// A repeat: the key exists, nothing is returned.
	tx = &claimTx{lock: recordRow(tenant, &origin, &depth), claim: failWith(pgx.ErrNoRows)}
	c, claimed, err = cascade.ClaimStart(t.Context(), tx, k)
	if err != nil || claimed || c.Claimed() || c.OutboxID() != uuid.Nil || c.TenantID() != tenant {
		t.Fatalf("repeat = %+v, %v, %v; want unclaimed, naming the tenant", c, claimed, err)
	}
}
