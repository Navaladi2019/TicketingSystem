package read

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
)

// Field masks (P3-33) add "/<field>" to a record entry of a masked
// definition, on top of the base masker's secrets, and nothing to any other
// entry.
func TestFieldMasksApplyToRecordEntriesOfTheirDefinitionOnly(t *testing.T) {
	def, other := uuid.New(), uuid.New()
	m := FieldMasks{Base: StaticMasker{"/password"}, Fields: map[uuid.UUID][]string{def: {"cost", "password"}}}
	cases := []struct {
		name string
		s    Subject
		want []patchwalk.Pointer
	}{
		{"a record entry of the masked definition", Subject{Domain: "record", SubjectType: "record", EntityDefinitionID: &def}, []patchwalk.Pointer{"/password", "/cost"}},
		{"a record entry of another definition", Subject{Domain: "record", SubjectType: "record", EntityDefinitionID: &other}, []patchwalk.Pointer{"/password"}},
		{"a configuration entry of the definition", Subject{Domain: "config", SubjectType: "entity_definition", EntityDefinitionID: &def}, []patchwalk.Pointer{"/password"}},
		{"an entry with no definition", Subject{Domain: "record", SubjectType: "record"}, []patchwalk.Pointer{"/password"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := m.MaskedPaths(t.Context(), nil, c.s)
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Fatalf("MaskedPaths = %v, %v; want %v", got, err, c.want)
			}
		})
	}
	out, redacted, err := MaskDiff([]byte(`[{"op":"replace","path":"/cost","value":12}]`), []patchwalk.Pointer{"/password", "/cost"})
	if err != nil || !redacted || string(out) != `[{"op":"replace","path":"/cost","value":`+MaskedValue+`}]` {
		t.Fatalf("MaskDiff = %s, %v, %v", out, redacted, err)
	}
}

func TestWithFieldMasksLeavesTheReaderUnchanged(t *testing.T) {
	r := &Reader{Masker: StaticMasker{"/password"}, MaxRecordEntries: 7}
	c := r.WithFieldMasks(map[uuid.UUID][]string{uuid.New(): {"cost"}})
	if _, ok := r.Masker.(StaticMasker); !ok {
		t.Fatal("WithFieldMasks changed the shared reader's masker")
	}
	if _, ok := c.Masker.(FieldMasks); !ok || c.MaxRecordEntries != 7 {
		t.Fatalf("copy = %+v, want a FieldMasks masker and the same cap", c)
	}
}
