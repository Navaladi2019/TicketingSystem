package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

const effectiveSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object", "additionalProperties": false,
  "required": ["title", "state"],
  "properties": {
    "title": {"type": "string"},
    "state": {"type": "string", "enum": ["new", "closed"], "default": "new"},
    "severity": {"type": "string", "default": "low"}
  }
}`

func effectiveDef(t *testing.T, fixed map[string]string) *metadata.Compiled {
	t.Helper()
	c, err := metadata.Compile(metadata.Definition{
		Key: "job", NumberPrefix: "JOB", Schema: json.RawMessage(effectiveSchema), FixedColumns: fixed,
		States: metadata.StateModel{Initial: "new", States: []string{"new", "closed"}, Transitions: []metadata.Transition{{From: "new", To: "closed"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// fakeDefs serves one stored version, or an error.
type fakeDefs struct {
	def   *metadata.Compiled
	err   error
	calls int
}

func (f *fakeDefs) Version(context.Context, pgx.Tx, string, int) (*metadata.Compiled, error) {
	f.calls++
	return f.def, f.err
}

func (f *fakeDefs) Current(context.Context, pgx.Tx, string) (*metadata.Compiled, int, error) {
	return f.def, 2, f.err
}

var _ entitytypes.DefinitionReader = (*fakeDefs)(nil)

// A record stored without the defaulted key reads with it, the stored copy is
// untouched, and the fixed columns come from the mapping given.
func TestEffectiveAppliesDefaultsToACopy(t *testing.T) {
	cur := effectiveDef(t, map[string]string{"state": metadata.ColumnState})
	rec := &Record{State: "closed", Data: map[string]any{"title": "t"}}
	got := Effective(cur, rec, cur.FixedColumns())
	if got["severity"] != "low" || got["state"] != "closed" || got["title"] != "t" {
		t.Errorf("effective = %v, want title, state from the column and the default severity", got)
	}
	if _, has := rec.Data["severity"]; has {
		t.Errorf("Effective wrote the default into the stored data: %v", rec.Data)
	}
	// A stored value is never replaced by the default.
	rec.Data["severity"] = "high"
	if got := Effective(cur, rec, cur.FixedColumns()); got["severity"] != "high" {
		t.Errorf("effective severity = %v, want the stored high", got["severity"])
	}
}

func TestEffectiveUnderReadsTheStoredVersionOnlyWhenItIsNotCurrent(t *testing.T) {
	cur := effectiveDef(t, map[string]string{"state": metadata.ColumnState})
	older := effectiveDef(t, map[string]string{"state": metadata.ColumnState})
	defs := &fakeDefs{def: older}
	rec := &Record{SchemaVersion: 1, State: "closed", Data: map[string]any{"title": "t"}}

	got, fixed, err := EffectiveUnder(t.Context(), nil, defs, "job", rec, 2, cur)
	if err != nil || defs.calls != 1 || fixed[metadata.ColumnState] != "" && len(fixed) != 1 {
		t.Fatalf("older version: err %v, %d reads, fixed %v; want one read and the older mapping", err, defs.calls, fixed)
	}
	if got["state"] != "closed" || got["severity"] != "low" {
		t.Errorf("effective under the older mapping = %v", got)
	}

	rec.SchemaVersion = 2
	got, fixed, err = EffectiveUnder(t.Context(), nil, defs, "job", rec, 2, cur)
	if err != nil || defs.calls != 1 || len(fixed) != 1 {
		t.Fatalf("current version read the definition again: err %v, %d reads", err, defs.calls)
	}
	if got["state"] != "closed" {
		t.Errorf("effective under the current mapping = %v, want state from the column", got)
	}
}

func TestEffectiveUnderReturnsAStoredVersionReadError(t *testing.T) {
	cur := effectiveDef(t, map[string]string{"state": metadata.ColumnState})
	boom := errors.New("boom")
	rec := &Record{SchemaVersion: 1, Data: map[string]any{}}
	if _, _, err := EffectiveUnder(t.Context(), nil, &fakeDefs{err: boom}, "job", rec, 2, cur); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the read error wrapped", err)
	}
}

func TestDefaultsAreFreshAndHoldTheTopLevelDefaults(t *testing.T) {
	cur := effectiveDef(t, map[string]string{"state": metadata.ColumnState})
	a, b := Defaults(cur), Defaults(cur)
	if a["severity"] != "low" || a["state"] != "new" || len(a) != 2 {
		t.Errorf("defaults = %v, want severity and state", a)
	}
	a["severity"] = "changed"
	if b["severity"] != "low" {
		t.Error("two calls share one map")
	}
}
