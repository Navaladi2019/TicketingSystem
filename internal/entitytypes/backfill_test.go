package entitytypes

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

func compiledWith(t *testing.T, extra string) *metadata.Compiled {
	t.Helper()
	schema := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,
"required":["state"],"properties":{"state":{"type":"string","enum":["new","closed"],"default":"new"}` + extra + `}}`
	c, err := metadata.Compile(metadata.Definition{
		Key: "incident", NumberPrefix: "INC", Schema: []byte(schema),
		FixedColumns: map[string]string{"state": metadata.ColumnState},
		States: metadata.StateModel{Initial: "new", States: []string{"new", "closed"},
			Transitions: []metadata.Transition{{From: "new", To: "closed"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestAddsDefault(t *testing.T) {
	t.Parallel()
	base := compiledWith(t, "")
	withDefault := compiledWith(t, `,"severity":{"type":"string","default":"low"}`)
	noDefault := compiledWith(t, `,"note":{"type":"string"}`)
	for _, tc := range []struct {
		name       string
		prev, next *metadata.Compiled
		want       bool
	}{
		{"a new default", base, withDefault, true},
		{"no new default", base, noDefault, false},
		{"the same default", withDefault, withDefault, false},
	} {
		if got := addsDefault(tc.prev, tc.next); got != tc.want {
			t.Errorf("%s: addsDefault = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestQueueBackfillWritesNothingWithoutANewDefault(t *testing.T) {
	t.Parallel()
	c := compiledWith(t, "")
	tx := &fakeTx{t: t} // any database call fails the test
	if err := queueBackfill(context.Background(), tx, uuid.New(), uuid.New(), 2, uuid.New(), c, c, nil); err != nil {
		t.Fatalf("queueBackfill = %v, want nil", err)
	}
}

func TestQueueBackfillFailures(t *testing.T) {
	t.Parallel()
	prev := compiledWith(t, "")
	next := compiledWith(t, `,"severity":{"type":"string","default":"low"}`)
	t.Run("job insert fails", func(t *testing.T) {
		t.Parallel()
		tx := &fakeTx{t: t, steps: []step{{execErr: errDB}}}
		err := queueBackfill(context.Background(), tx, uuid.New(), uuid.New(), 2, uuid.New(), prev, next, nil)
		if !errors.Is(err, errDB) {
			t.Fatalf("err = %v, want the database error", err)
		}
	})
	t.Run("outbox row refused", func(t *testing.T) {
		t.Parallel()
		// Version 0 is out of range: the outbox helper refuses it before any call.
		tx := &fakeTx{t: t, steps: []step{{tag: "INSERT 0 1"}}}
		err := queueBackfill(context.Background(), tx, uuid.New(), uuid.New(), 0, uuid.New(), prev, next, nil)
		if err == nil {
			t.Fatal("err = nil, want the outbox refusal")
		}
	})
}
