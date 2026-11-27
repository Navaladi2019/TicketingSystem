package server

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// A limit outside 1 to 100 is refused before any database read (the pool here
// is never used), and a cursor that is not ours is invalid_cursor.
func TestListBackfillFailuresRefusesBadLimitAndCursorBeforeTheDatabase(t *testing.T) {
	kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: bytes.Repeat([]byte("s"), 32)})
	if err != nil {
		t.Fatal(err)
	}
	h := &handlers{pool: &pgxpool.Pool{}, cursors: kr}
	ctx := identity.NewContext(context.Background(), identity.Principal{TenantID: uuid.New(), PrincipalID: uuid.New()})
	for _, limit := range []int{0, -1, 101} {
		_, err := h.ListEntityTypeBackfillFailures(ctx, api.ListEntityTypeBackfillFailuresRequestObject{Key: "incident", Params: api.ListEntityTypeBackfillFailuresParams{Limit: &limit}})
		wantProblemCode(t, err, problem.ValidationFailed)
	}
	bad := "not-a-cursor"
	_, err = h.ListEntityTypeBackfillFailures(ctx, api.ListEntityTypeBackfillFailuresRequestObject{Key: "incident", Params: api.ListEntityTypeBackfillFailuresParams{Cursor: &bad}})
	wantProblemCode(t, err, problem.InvalidCursor)
}
