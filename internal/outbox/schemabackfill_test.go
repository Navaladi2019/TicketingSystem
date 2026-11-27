package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
)

// TICK-920 (ADR-009 D1, contract C3): a schema.backfill row's payload is
// {entity_definition_id} and nothing else; its dedupe key names the
// definition and the target version; and its start goes through
// EnqueueSchemaBackfill only.
func TestSchemaBackfillPayloadAllowList(t *testing.T) {
	def := uuid.NewString()
	if err := ValidateKindPayload(SchemaBackfillKind, json.RawMessage(`{"entity_definition_id":"`+def+`"}`)); err != nil {
		t.Fatalf("the C3 payload is refused: %v", err)
	}
	for _, p := range []string{
		`{}`, `{"entity_definition_id":"` + def + `","tenant_id":"` + uuid.NewString() + `"}`,
		`{"entity_definition_id":"` + def + `","target_version":"` + uuid.NewString() + `"}`,
		`{"entity_definition_id":3}`,
	} {
		if err := ValidateKindPayload(SchemaBackfillKind, json.RawMessage(p)); !errors.Is(err, ErrInvalidEntry) {
			t.Errorf("%s: %v, want ErrInvalidEntry", p, err)
		}
	}
}

func TestSchemaBackfillDedupeKey(t *testing.T) {
	def := uuid.MustParse("0192bbbb-0000-7000-8000-0000000000d1")
	if got, want := SchemaBackfillDedupeKey(def, 7), "schema-backfill:0192bbbb-0000-7000-8000-0000000000d1:7"; got != want {
		t.Errorf("dedupe key %q, want %q", got, want)
	}
}

func TestEnqueueSchemaBackfillRefusesBeforeTheTransaction(t *testing.T) {
	for _, c := range []struct {
		def     uuid.UUID
		version int
	}{{uuid.Nil, 1}, {uuid.New(), 0}, {uuid.New(), math.MaxInt32 + 1}} {
		if _, err := EnqueueSchemaBackfill(context.Background(), nil, c.def, c.version); !errors.Is(err, ErrInvalidEntry) {
			t.Errorf("%v/%d: %v, want ErrInvalidEntry", c.def, c.version, err)
		}
	}
}

func TestSchemaBackfillStartHasItsOwnEntryPoint(t *testing.T) {
	e := Entry{Kind: SchemaBackfillKind, KindVersion: 1, DedupeKey: "k", RecordID: uuid.New(),
		Target: TargetStart{Kind: SchemaBackfillKind}, Causation: Causation{OriginID: uuid.New()}}
	if err := notOwnEntryPointStart(e); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("a schema.backfill start through EnqueueAtVersion: %v, want ErrInvalidEntry", err)
	}
}
