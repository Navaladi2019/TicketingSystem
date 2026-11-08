package read

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
)

// applyPatch applies every add, remove and replace the pipeline's diffs use,
// on objects and arrays, and refuses anything it cannot apply exactly, so
// Replay marks the change unknown instead of showing a wrong payload.
func TestApplyPatch(t *testing.T) {
	doc := func() map[string]any {
		return map[string]any{
			"s":   "text",
			"obj": map[string]any{"k": json.Number("1")},
			"arr": []any{"a", "b", map[string]any{"k": "v"}},
		}
	}
	for _, c := range []struct {
		name, diff string
		want       map[string]any // nil: the patch is refused
		wantErr    string
	}{
		{name: "add member", diff: `[{"op":"add","path":"/n","value":2}]`,
			want: map[string]any{"s": "text", "obj": map[string]any{"k": json.Number("1")}, "arr": []any{"a", "b", map[string]any{"k": "v"}}, "n": json.Number("2")}},
		{name: "add nested", diff: `[{"op":"add","path":"/obj/j","value":true}]`,
			want: map[string]any{"s": "text", "obj": map[string]any{"k": json.Number("1"), "j": true}, "arr": []any{"a", "b", map[string]any{"k": "v"}}}},
		{name: "replace member", diff: `[{"op":"replace","path":"/s","value":"new"}]`,
			want: map[string]any{"s": "new", "obj": map[string]any{"k": json.Number("1")}, "arr": []any{"a", "b", map[string]any{"k": "v"}}}},
		{name: "remove member", diff: `[{"op":"remove","path":"/s"}]`,
			want: map[string]any{"obj": map[string]any{"k": json.Number("1")}, "arr": []any{"a", "b", map[string]any{"k": "v"}}}},
		{name: "array insert", diff: `[{"op":"add","path":"/arr/1","value":"x"}]`,
			want: map[string]any{"s": "text", "obj": map[string]any{"k": json.Number("1")}, "arr": []any{"a", "x", "b", map[string]any{"k": "v"}}}},
		{name: "array append at length", diff: `[{"op":"add","path":"/arr/3","value":"z"}]`,
			want: map[string]any{"s": "text", "obj": map[string]any{"k": json.Number("1")}, "arr": []any{"a", "b", map[string]any{"k": "v"}, "z"}}},
		{name: "array replace", diff: `[{"op":"replace","path":"/arr/0","value":"A"}]`,
			want: map[string]any{"s": "text", "obj": map[string]any{"k": json.Number("1")}, "arr": []any{"A", "b", map[string]any{"k": "v"}}}},
		{name: "array remove", diff: `[{"op":"remove","path":"/arr/0"}]`,
			want: map[string]any{"s": "text", "obj": map[string]any{"k": json.Number("1")}, "arr": []any{"b", map[string]any{"k": "v"}}}},
		{name: "through an array element", diff: `[{"op":"replace","path":"/arr/2/k","value":"w"}]`,
			want: map[string]any{"s": "text", "obj": map[string]any{"k": json.Number("1")}, "arr": []any{"a", "b", map[string]any{"k": "w"}}}},
		{name: "replace the root with an object", diff: `[{"op":"replace","path":"","value":{"only":1}}]`,
			want: map[string]any{"only": json.Number("1")}},

		{name: "not a patch", diff: `{"op":"add"}`, wantErr: "patchwalk"},
		{name: "root not an object", diff: `[{"op":"replace","path":"","value":[1]}]`, wantErr: "result is not an object"},
		{name: "remove the root", diff: `[{"op":"remove","path":""}]`, wantErr: "remove of the root"},
		{name: "move", diff: `[{"op":"move","from":"/s","path":"/t"}]`, wantErr: "unsupported patch: move"},
		{name: "copy", diff: `[{"op":"copy","from":"/s","path":"/t"}]`, wantErr: "unsupported patch: copy"},
		{name: "test", diff: `[{"op":"test","path":"/s","value":"text"}]`, wantErr: "unsupported patch: test"},
		{name: "missing parent", diff: `[{"op":"add","path":"/none/k","value":1}]`, wantErr: "missing member"},
		{name: "missing grandparent", diff: `[{"op":"add","path":"/obj/none/k","value":1}]`, wantErr: "missing member"},
		{name: "replace a missing member", diff: `[{"op":"replace","path":"/none","value":1}]`, wantErr: "missing member"},
		{name: "remove a missing member", diff: `[{"op":"remove","path":"/none"}]`, wantErr: "missing member"},
		{name: "index not a number", diff: `[{"op":"replace","path":"/arr/x","value":1}]`, wantErr: "bad array index"},
		{name: "index with a leading zero", diff: `[{"op":"replace","path":"/arr/01","value":1}]`, wantErr: "bad array index"},
		{name: "index past the end, nested", diff: `[{"op":"add","path":"/arr/9/k","value":1}]`, wantErr: "index out of range"},
		{name: "error below an element", diff: `[{"op":"add","path":"/arr/2/none/k","value":1}]`, wantErr: "missing member"},
		{name: "add past the end", diff: `[{"op":"add","path":"/arr/4","value":1}]`, wantErr: "index out of range"},
		{name: "replace past the end", diff: `[{"op":"replace","path":"/arr/3","value":1}]`, wantErr: "index out of range"},
		{name: "remove past the end", diff: `[{"op":"remove","path":"/arr/3"}]`, wantErr: "index out of range"},
		{name: "path through a scalar", diff: `[{"op":"add","path":"/s/k","value":1}]`, wantErr: "path through a scalar"},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := doc()
			got, err := applyPatch(before, []byte(c.diff))
			if c.wantErr != "" {
				if err == nil || got != nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("applyPatch = %v, %v; want an error containing %q", got, err, c.wantErr)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Fatalf("applyPatch = %v, %v; want %v", got, err, c.want)
			}
			if !reflect.DeepEqual(before, doc()) {
				t.Errorf("the before payload changed: %v", before)
			}
		})
	}
}

// A step that cannot be applied makes it and every later step unknown, until
// a create restarts the replay.
func TestReplayStopsAtAnUnappliableStep(t *testing.T) {
	ch := Replay([]Entry{
		{Action: "create", Diff: []byte(`[{"op":"add","path":"","value":{"a":1}}]`)},
		{Action: "update", Diff: []byte(`[{"op":"remove","path":"/none"}]`)},
		{Action: "update", Diff: []byte(`[{"op":"replace","path":"/a","value":2}]`)},
		{Action: "create", Diff: []byte(`[{"op":"add","path":"","value":{"b":1}}]`)},
	})
	if got := []bool{ch[0].Known, ch[1].Known, ch[2].Known, ch[3].Known}; !reflect.DeepEqual(got, []bool{true, false, false, true}) {
		t.Fatalf("known = %v, want [true false false true]", got)
	}
	if ch[1].Before != nil || ch[1].After != nil || ch[3].After["b"] != json.Number("1") {
		t.Errorf("changes = %+v", ch)
	}
}

func TestStaticMaskerReturnsItsPointers(t *testing.T) {
	m := StaticMasker{"/a", "/b"}
	got, err := m.MaskedPaths(t.Context(), nil, Subject{})
	if err != nil || !reflect.DeepEqual(got, []patchwalk.Pointer{"/a", "/b"}) {
		t.Fatalf("MaskedPaths = %v, %v", got, err)
	}
}

// A diff the walker refuses touches every pointer and is redacted whole.
func TestMaskDiffRedactsAMalformedDiffWhole(t *testing.T) {
	out, redacted, err := MaskDiff([]byte(`[{"op":"add"}]`), []patchwalk.Pointer{"/password"})
	if err != nil || !redacted || string(out) != redactedDiff {
		t.Fatalf("MaskDiff = %s, %v, %v; want the whole-diff redaction", out, redacted, err)
	}
}

// A masked pointer below an op's path is cut out of the op's value, through
// objects and arrays; a pointer the value does not reach leaves it alone.
func TestMaskDiffCutsNestedMembers(t *testing.T) {
	diff := `[{"op":"add","path":"/a","value":{"b":{"c":"secret","d":1},"l":[{"c":"s2"}]}}]`
	out, redacted, err := MaskDiff([]byte(diff), []patchwalk.Pointer{"/a/b/c", "/a/l/0/c", "/a/none/c", "/a/l/9/c"})
	if err != nil || !redacted {
		t.Fatalf("MaskDiff = %s, %v, %v", out, redacted, err)
	}
	if s := string(out); strings.Contains(s, "secret") || strings.Contains(s, "s2") || !strings.Contains(s, `"d":1`) {
		t.Fatalf("masked diff %s", s)
	}
	// Only pointers the value does not hold: nothing changes.
	out, redacted, err = MaskDiff([]byte(diff), []patchwalk.Pointer{"/a/none/c"})
	if err != nil || redacted || string(out) != diff {
		t.Fatalf("unreached pointer: %s, %v, %v; want the stored bytes", out, redacted, err)
	}
}

// An entry with no diff carries no salt and asks the masker nothing.
func TestMaskDropsTheSaltOfAnEntryWithoutADiff(t *testing.T) {
	r := &Reader{Masker: StaticMasker{"/password"}}
	entries := []Entry{
		{Domain: "record", SubjectType: "record", DiffSalt: []byte{1}},
		{Domain: "record", SubjectType: "record", Diff: []byte(`[{"op":"add","path":"/n","value":1}]`), DiffSalt: []byte{2}},
	}
	if err := r.mask(t.Context(), nil, entries); err != nil {
		t.Fatal(err)
	}
	if entries[0].DiffSalt != nil || entries[0].Diff != nil {
		t.Errorf("entry without a diff = %+v, want no salt", entries[0])
	}
	if entries[1].DiffRedacted || len(entries[1].DiffSalt) != 1 {
		t.Errorf("untouched entry = %+v, want its salt kept", entries[1])
	}
}

var errBoom = errors.New("boom")

type queryErrTx struct{ pgx.Tx }

func (queryErrTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, errBoom }

// rowsTx answers with rows.
type rowsTx struct {
	pgx.Tx
	rows pgx.Rows
}

func (x rowsTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return x.rows, nil }

// badRows is fakeRows whose Scan or Err fails.
type badRows struct {
	fakeRows
	scanErr, err error
}

func (r *badRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	return r.fakeRows.Scan(dest...)
}

func (r *badRows) Err() error { return r.err }

// Every way DefinitionMasker cannot read a schema fails the read: it never
// answers "nothing is secret" on an error.
func TestDefinitionMaskerFailsOnReadErrors(t *testing.T) {
	def := uuid.New()
	s := Subject{Domain: "record", SubjectType: "record", EntityDefinitionID: &def}
	for name, c := range map[string]struct {
		tx   pgx.Tx
		want string
	}{
		"query":     {queryErrTx{}, "read definition schemas: boom"},
		"scan":      {rowsTx{rows: &badRows{fakeRows: fakeRows{schemas: []string{`{}`}, i: -1}, scanErr: errBoom}}, "boom"},
		"not JSON":  {rowsTx{rows: &badRows{fakeRows: fakeRows{schemas: []string{`{`}, i: -1}}}, "unexpected end of JSON"},
		"rows fail": {rowsTx{rows: &badRows{fakeRows: fakeRows{schemas: []string{`{}`}, i: -1}, err: errBoom}}, "boom"},
	} {
		t.Run(name, func(t *testing.T) {
			paths, err := DefinitionMasker{}.MaskedPaths(t.Context(), c.tx, s)
			if err == nil || paths != nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("MaskedPaths = %v, %v; want an error containing %q", paths, err, c.want)
			}
		})
	}
}
