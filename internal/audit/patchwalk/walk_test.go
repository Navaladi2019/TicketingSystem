package patchwalk

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type visitRec struct {
	Index int
	Kind  Kind
	Field Field
	Ptr   Pointer
	V     string
}

func collect(t *testing.T, diff string) []visitRec {
	t.Helper()
	var got []visitRec
	err := Walk([]byte(diff), func(op Op, ptr Pointer, v json.RawMessage) error {
		got = append(got, visitRec{op.Index, op.Kind, op.Field, ptr, string(v)})
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	return got
}

func TestWalkVisitsPathFromAndEveryNestedValue(t *testing.T) {
	diff := `[
	  {"op":"add","path":"/a","value":{"b":[1,{"c":"x"}],"d~/e":true}},
	  {"op":"remove","path":"/r"},
	  {"op":"copy","from":"/s","path":"/t"},
	  {"op":"move","path":"/u","from":"/v"},
	  {"op":"test","path":"/w","value":"secret"},
	  {"op":"replace","path":"","value":null}
	]`
	want := []visitRec{
		{0, KindAdd, FieldPath, "/a", ""},
		{0, KindAdd, FieldValue, "/a", `{"b":[1,{"c":"x"}],"d~/e":true}`},
		{0, KindAdd, FieldValue, "/a/b", `[1,{"c":"x"}]`},
		{0, KindAdd, FieldValue, "/a/b/0", `1`},
		{0, KindAdd, FieldValue, "/a/b/1", `{"c":"x"}`},
		{0, KindAdd, FieldValue, "/a/b/1/c", `"x"`},
		{0, KindAdd, FieldValue, "/a/d~0~1e", `true`},
		{1, KindRemove, FieldPath, "/r", ""},
		{2, KindCopy, FieldPath, "/t", ""},
		{2, KindCopy, FieldFrom, "/s", ""},
		{3, KindMove, FieldPath, "/u", ""},
		{3, KindMove, FieldFrom, "/v", ""},
		{4, KindTest, FieldPath, "/w", ""},
		{4, KindTest, FieldValue, "/w", `"secret"`},
		{5, KindReplace, FieldPath, "", ""},
		{5, KindReplace, FieldValue, "", `null`},
	}
	got := collect(t, diff)
	if len(got) != len(want) {
		t.Fatalf("got %d visits, want %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("visit %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestWalkTestOpValueIsWalked(t *testing.T) {
	got := collect(t, `[{"op":"test","path":"/cfg","value":{"password":"canary"}}]`)
	var saw bool
	for _, v := range got {
		if v.Kind == KindTest && v.Field == FieldValue && v.Ptr == "/cfg/password" && v.V == `"canary"` {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("nested value of a test op was not visited: %v", got)
	}
}

func TestWalkRawKeepsOriginalBytes(t *testing.T) {
	got := collect(t, `[{"op":"add","path":"/a","value": { "k" : "\u0070w" , "n": 1.50e1 } }]`)
	if got[1].V != `{ "k" : "\u0070w" , "n": 1.50e1 }` {
		t.Errorf("raw value = %q", got[1].V)
	}
	if got[2].Ptr != "/a/k" || got[2].V != `"\u0070w"` {
		t.Errorf("nested = %+v", got[2])
	}
	if got[3].V != `1.50e1` {
		t.Errorf("number = %q", got[3].V)
	}
}

func TestWalkStopsOnCallbackError(t *testing.T) {
	stop := errors.New("stop")
	calls := 0
	err := Walk([]byte(`[{"op":"add","path":"/a","value":[1,2,3]}]`), func(Op, Pointer, json.RawMessage) error {
		calls++
		if calls == 3 {
			return stop
		}
		return nil
	})
	if !errors.Is(err, stop) || err.Error() != stop.Error() {
		t.Fatalf("err = %v, want the callback error unchanged", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestWalkRejectsMalformedWithoutLeakingValues(t *testing.T) {
	const canary = "CANARY-7f3a"
	cases := map[string]string{
		"not an array":            `{"op":"add","path":"/a","value":"` + canary + `"}`,
		"op not object":           `["` + canary + `"]`,
		"missing op":              `[{"path":"/a","value":"` + canary + `"}]`,
		"unknown op":              `[{"op":"` + canary + `","path":"/a"}]`,
		"op not string":           `[{"op":1,"path":"/a"}]`,
		"missing path":            `[{"op":"add","value":"` + canary + `"}]`,
		"path not string":         `[{"op":"add","path":1,"value":1}]`,
		"bad path":                `[{"op":"add","path":"` + canary + `","value":1}]`,
		"bad escape":              `[{"op":"add","path":"/a~2","value":1}]`,
		"missing value":           `[{"op":"add","path":"/a"}]`,
		"missing from":            `[{"op":"copy","path":"/a"}]`,
		"bad from":                `[{"op":"move","path":"/a","from":"` + canary + `"}]`,
		"value on remove":         `[{"op":"remove","path":"/a","value":"` + canary + `"}]`,
		"from on add":             `[{"op":"add","path":"/a","value":1,"from":"/` + canary + `"}]`,
		"case-folded Value":       `[{"op":"add","path":"/a","value":1,"Value":"` + canary + `"}]`,
		"unknown member":          `[{"op":"add","path":"/a","value":1,"` + canary + `":1}]`,
		"duplicate value":         `[{"op":"add","path":"/a","value":"` + canary + `","value":"x"}]`,
		"duplicate nested key":    `[{"op":"add","path":"/a","value":{"pw":"` + canary + `","pw":"x"}}]`,
		"duplicate escaped key":   `[{"op":"add","path":"/a","value":{"pw":"` + canary + `","\u0070w":"x"}}]`,
		"trailing data":           `[] "` + canary + `"`,
		"two arrays":              `[][]`,
		"syntax error":            `[{"op":"add","path":"/a","value":` + canary + `}]`,
		"unterminated":            `[{"op":"add","path":"/a","value":"` + canary,
		"empty":                   ``,
		"invalid utf-8":           "[{\"op\":\"add\",\"path\":\"/a\",\"value\":\"\xff" + canary + "\"}]",
		"too deep":                strings.Repeat("[", MaxDepth+1) + strings.Repeat("]", MaxDepth+1),
		"too deep in value":       `[{"op":"add","path":"/a","value":` + strings.Repeat("[", MaxDepth) + strings.Repeat("]", MaxDepth) + `}]`,
		"bad delimiter in object": `[{"op":"add","path":"/a","value":{"k" "` + canary + `"}}]`,
	}
	for name, diff := range cases {
		t.Run(name, func(t *testing.T) {
			calls := 0
			err := Walk([]byte(diff), func(Op, Pointer, json.RawMessage) error { calls++; return nil })
			if !errors.Is(err, ErrMalformed) {
				t.Fatalf("err = %v, want ErrMalformed", err)
			}
			if calls != 0 {
				t.Errorf("fn called %d times on a malformed diff", calls)
			}
			if strings.Contains(err.Error(), canary) {
				t.Errorf("error leaks the value: %v", err)
			}
		})
	}
}

func TestWalkEmptyPatch(t *testing.T) {
	if got := collect(t, ` [ ] `); len(got) != 0 {
		t.Fatalf("visits = %v", got)
	}
}

func TestPointer(t *testing.T) {
	for _, p := range []Pointer{"", "/", "/a", "/a~0b~1c", "/a/0/-"} {
		if err := p.Validate(); err != nil {
			t.Errorf("%q: %v", p, err)
		}
		if got := FromSegments(p.Segments()); got != p {
			t.Errorf("round trip %q = %q", p, got)
		}
	}
	for _, p := range []Pointer{"a", "/a~", "/a~2", "~0"} {
		if p.Validate() == nil {
			t.Errorf("%q: want invalid", p)
		}
	}
	prefix := []struct {
		p, prefix Pointer
		want      bool
	}{
		{"/a/b", "/a", true}, {"/a", "/a", true}, {"/a", "", true},
		{"/ab", "/a", false}, {"/a", "/a/b", false}, {"", "/a", false},
	}
	for _, c := range prefix {
		if got := c.p.HasPrefix(c.prefix); got != c.want {
			t.Errorf("%q.HasPrefix(%q) = %v", c.p, c.prefix, got)
		}
	}
}

// FuzzWalk: Walk never panics, and on a diff it accepts every op is visited
// exactly once for its path, copy and move once for their from, and every
// value visit sits at or below its op's path.
func FuzzWalk(f *testing.F) {
	for _, s := range []string{
		`[]`,
		`[{"op":"add","path":"/a","value":{"b":[1,{"c":"x"}]}}]`,
		`[{"op":"remove","path":"/a"},{"op":"copy","from":"/a","path":"/b"}]`,
		`[{"op":"test","path":"","value":[[[]]]}]`,
		`[{"op":"move","from":"/a~0~1","path":"/-"}]`,
		`[{"op":"replace","path":"/x","value":"https://u:p@h"}]`,
		`[{"op":"add","path":"/a","value":{"k":1,"k":2}}]`,
		`{"op":"add"}`,
		`[`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, diff []byte) {
		paths := map[int]int{}
		froms := map[int]int{}
		opPath := map[int]Pointer{}
		err := Walk(diff, func(op Op, ptr Pointer, v json.RawMessage) error {
			switch op.Field {
			case FieldPath:
				paths[op.Index]++
				opPath[op.Index] = ptr
				if v != nil {
					return fmt.Errorf("op %d: path visit has a value", op.Index)
				}
			case FieldFrom:
				froms[op.Index]++
				if op.Kind != KindCopy && op.Kind != KindMove {
					return fmt.Errorf("op %d: from visit on %s", op.Index, op.Kind)
				}
			case FieldValue:
				if !ptr.HasPrefix(opPath[op.Index]) {
					return fmt.Errorf("op %d: value at %q not under %q", op.Index, ptr, opPath[op.Index])
				}
				if !json.Valid(v) {
					return fmt.Errorf("op %d: value at %q is not valid JSON", op.Index, ptr)
				}
			}
			if err := ptr.Validate(); err != nil {
				return fmt.Errorf("op %d: visited an invalid pointer", op.Index)
			}
			return nil
		})
		if err != nil {
			if !errors.Is(err, ErrMalformed) {
				t.Fatalf("walk invariant broken: %v", err)
			}
			return
		}
		var ops []map[string]json.RawMessage
		if err := json.Unmarshal(diff, &ops); err != nil {
			t.Fatalf("Walk accepted a diff encoding/json rejects: %v", err)
		}
		if len(paths) != len(ops) {
			t.Fatalf("visited %d op paths, patch has %d ops", len(paths), len(ops))
		}
		for i := range ops {
			if paths[i] != 1 {
				t.Fatalf("op %d path visited %d times", i, paths[i])
			}
			var kind Kind
			_ = json.Unmarshal(ops[i]["op"], &kind)
			wantFrom := 0
			if kind == KindCopy || kind == KindMove {
				wantFrom = 1
			}
			if froms[i] != wantFrom {
				t.Fatalf("op %d (%s) from visited %d times", i, kind, froms[i])
			}
		}
	})
}
