package authz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// fakeStore has no path read at all: AdminGroupStore is the only way in.
type fakeStore struct {
	rows map[uuid.UUID]uuid.UUID
	err  error
}

func (f fakeStore) AdminGroupID(_ context.Context, t uuid.UUID) (uuid.UUID, error) {
	if f.err != nil {
		return uuid.Nil, f.err
	}
	g, ok := f.rows[t]
	if !ok {
		return uuid.Nil, ErrNoAdminGroup
	}
	return g, nil
}

func TestRequireTenantAdmin(t *testing.T) {
	ta, tb := uuid.New(), uuid.New()
	ga, gb := uuid.New(), uuid.New()
	st := fakeStore{rows: map[uuid.UUID]uuid.UUID{ta: ga, tb: gb}}
	boom := errors.New("db down")
	tests := []struct {
		name  string
		store AdminGroupStore
		p     identity.Principal
		want  error
	}{
		{"member passes", st, identity.Principal{TenantID: ta, Groups: []uuid.UUID{uuid.New(), ga}}, nil},
		{"non-member", st, identity.Principal{TenantID: ta, Groups: []uuid.UUID{uuid.New()}}, ErrForbidden},
		{"no groups", st, identity.Principal{TenantID: ta}, ErrForbidden},
		{"other tenant's admin group", st, identity.Principal{TenantID: ta, Groups: []uuid.UUID{gb}}, ErrForbidden},
		{"tenant without a row", fakeStore{}, identity.Principal{TenantID: ta, Groups: []uuid.UUID{ga}}, ErrForbidden},
		{"nil group id never matches", fakeStore{rows: map[uuid.UUID]uuid.UUID{ta: uuid.Nil}}, identity.Principal{TenantID: ta, Groups: []uuid.UUID{uuid.Nil}}, ErrForbidden},
		{"store error is not a 403", fakeStore{err: boom}, identity.Principal{TenantID: ta, Groups: []uuid.UUID{ga}}, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := RequireTenantAdmin(t.Context(), tc.store, tc.p)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if tc.want != nil && errors.Is(tc.want, boom) && errors.Is(err, ErrForbidden) {
				t.Fatal("a store failure must not read as forbidden")
			}
		})
	}
}
