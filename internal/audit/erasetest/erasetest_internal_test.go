package erasetest

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
)

func TestNormaliseRefusesBadRequests(t *testing.T) {
	ref := uuid.New()
	many := make([]uuid.UUID, audit.MaxErasedIDs+1)
	for i := range many {
		many[i] = uuid.New()
	}
	for name, c := range map[string]struct {
		ref uuid.UUID
		ids []uuid.UUID
	}{
		"nil ref":          {uuid.Nil, []uuid.UUID{uuid.New()}},
		"no ids":           {ref, nil},
		"over the E1 cap":  {ref, many},
		"a nil id in list": {ref, []uuid.UUID{uuid.New(), uuid.Nil}},
	} {
		if _, err := normalise(c.ref, c.ids); !errors.Is(err, ErrBadRequest) {
			t.Errorf("%s: err = %v, want ErrBadRequest", name, err)
		}
	}
	// The cap itself is allowed.
	if got, err := normalise(ref, many[:audit.MaxErasedIDs]); err != nil || len(got) != audit.MaxErasedIDs {
		t.Errorf("at the cap: %d ids, %v", len(got), err)
	}
}

// A duplicate id is erased once, so the row count matches the list.
func TestNormaliseSortsAndDedupes(t *testing.T) {
	a := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	b := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	got, err := normalise(uuid.New(), []uuid.UUID{b, a, b})
	if err != nil || len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("normalise = %v, %v; want [a b]", got, err)
	}
}

func TestEraseDiffIsTheVerifierShape(t *testing.T) {
	id := uuid.MustParse("0a000000-0000-4000-8000-0000000010a1")
	got, err := EraseDiff([]uuid.UUID{id})
	if err != nil {
		t.Fatal(err)
	}
	if want := `[{"op":"add","path":"/erased_ids","value":["0a000000-0000-4000-8000-0000000010a1"]}]`; string(got) != want {
		t.Errorf("diff = %s, want %s", got, want)
	}
}
