package jsondiff_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
)

// goldenCases pin the output bytes. diff_sha256 hashes these bytes (ADR-004
// R13), so any change here is a change to the audit contract (P5-13b, P5-14)
// and needs Technical Lead review. Run with -count=20: the objects are wide
// enough that a map-order leak fails at least one run.
type goldenCase struct {
	name, before, after, want string
	nilBefore                 bool
}

func goldenCases() []goldenCase {
	return []goldenCase{
		{name: "create", nilBefore: true, after: `{"b":1, "a":[true,null,"x"]}`,
			want: `[{"op":"add","path":"","value":{"a":[true,null,"x"],"b":1}}]`},
		{name: "create scalar", nilBefore: true, after: `"hi"`,
			want: `[{"op":"add","path":"","value":"hi"}]`},
		{name: "equal objects in another member order", before: `{"a":1,"b":{"c":[1,2]}}`, after: `{ "b" : {"c":[1,2]}, "a":1 }`,
			want: `[]`},
		{name: "equal scalars", before: `null`, after: `null`, want: `[]`},
		{name: "root type change", before: `{"a":1}`, after: `[1]`,
			want: `[{"op":"replace","path":"","value":[1]}]`},
		{name: "root scalar change", before: `1`, after: `2`,
			want: `[{"op":"replace","path":"","value":2}]`},
		{name: "add remove replace in key order",
			before: `{"z":1,"m":2,"a":3,"k":{"x":1}}`,
			after:  `{"b":9,"m":2,"a":4,"k":{"x":1,"y":null}}`,
			want:   `[{"op":"replace","path":"/a","value":4},{"op":"add","path":"/b","value":9},{"op":"add","path":"/k/y","value":null},{"op":"remove","path":"/z"}]`},
		{name: "null transitions",
			before: `{"a":null,"b":1,"c":null}`,
			after:  `{"a":0,"b":null,"c":null}`,
			want:   `[{"op":"replace","path":"/a","value":0},{"op":"replace","path":"/b","value":null}]`},
		{name: "nested type changes",
			before: `{"a":{"x":1},"b":[1],"c":"s","d":true}`,
			after:  `{"a":[1],"b":{"x":1},"c":1,"d":"true"}`,
			want:   `[{"op":"replace","path":"/a","value":[1]},{"op":"replace","path":"/b","value":{"x":1}},{"op":"replace","path":"/c","value":1},{"op":"replace","path":"/d","value":"true"}]`},
		{name: "array grow", before: `[1,2]`, after: `[1,2,3,{"b":2,"a":1}]`,
			want: `[{"op":"add","path":"/2","value":3},{"op":"add","path":"/3","value":{"a":1,"b":2}}]`},
		{name: "array shrink removes from the end down", before: `[1,2,3,4]`, after: `[1]`,
			want: `[{"op":"remove","path":"/3"},{"op":"remove","path":"/2"},{"op":"remove","path":"/1"}]`},
		{name: "array reorder is index-wise replace", before: `[1,2,3]`, after: `[3,2,1]`,
			want: `[{"op":"replace","path":"/0","value":3},{"op":"replace","path":"/2","value":1}]`},
		{name: "array to empty", before: `{"l":[1,2]}`, after: `{"l":[]}`,
			want: `[{"op":"remove","path":"/l/1"},{"op":"remove","path":"/l/0"}]`},
		{name: "nested arrays of objects",
			before: `{"items":[{"id":1,"tags":["a"]},{"id":2}]}`,
			after:  `{"items":[{"id":1,"tags":["a","b"]},{"id":3}]}`,
			want:   `[{"op":"add","path":"/items/0/tags/1","value":"b"},{"op":"replace","path":"/items/1/id","value":3}]`},
		{name: "pointer escaping", before: `{"a/b":1,"m~n":1,"":1}`, after: `{"a/b":2,"m~n":2,"":2,"~1":0}`,
			want: `[{"op":"replace","path":"/","value":2},{"op":"replace","path":"/a~1b","value":2},{"op":"replace","path":"/m~0n","value":2},{"op":"add","path":"/~01","value":0}]`},
		{name: "string escaping is RFC 8785", before: `{"s":""}`, after: `{"s":"q\" b\\ \b\t\n\f\r \u0001\u001f <>&    é \/"}`,
			want: "[{\"op\":\"replace\",\"path\":\"/s\",\"value\":\"q\\\" b\\\\ \\b\\t\\n\\f\\r \\u0001\\u001f <>&    é /\"}]"},
		{name: "keys sort by UTF-16 code units", nilBefore: true, after: `{"｡":1,"😀":2,"a":3,"B":4}`,
			want: "[{\"op\":\"add\",\"path\":\"\",\"value\":{\"B\":4,\"a\":3,\"\U0001F600\":2,\"｡\":1}}]"},
		{name: "numbers keep their literal", before: `{"a":1,"b":1e2,"c":-0}`, after: `{"a":1.0,"b":1e2,"c":-0,"d":12345678901234567890123}`,
			want: `[{"op":"replace","path":"/a","value":1.0},{"op":"add","path":"/d","value":12345678901234567890123}]`},
		{name: "wide object", before: wideObject(0), after: wideObject(1),
			want: wideWant()},
	}
}

// wideObject has 64 members; seed 1 changes every even key's value.
func wideObject(seed int) string {
	var b strings.Builder
	b.WriteByte('{')
	for i := range 64 {
		if i > 0 {
			b.WriteByte(',')
		}
		v := i
		if seed == 1 && i%2 == 0 {
			v = -i - 1
		}
		fmt.Fprintf(&b, `"k%02d":%d`, 63-i, v)
	}
	b.WriteByte('}')
	return b.String()
}

func wideWant() string {
	var ops []string
	for key := range 64 {
		i := 63 - key
		if i%2 == 0 {
			ops = append(ops, fmt.Sprintf(`{"op":"replace","path":"/k%02d","value":%d}`, key, -i-1))
		}
	}
	return "[" + strings.Join(ops, ",") + "]"
}

func TestDiffGolden(t *testing.T) {
	for _, c := range goldenCases() {
		t.Run(c.name, func(t *testing.T) {
			var before json.RawMessage
			if !c.nilBefore {
				before = json.RawMessage(c.before)
			}
			got, err := jsondiff.Diff(before, json.RawMessage(c.after))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != c.want {
				t.Fatalf("diff bytes\n got: %s\nwant: %s", got, c.want)
			}
			checkApplies(t, before, json.RawMessage(c.after), got)
		})
	}
}

// TestDiffGoldenHash pins the SHA-256 of all golden outputs together, so a
// byte change is visible in review even if someone edits a want string.
func TestDiffGoldenHash(t *testing.T) {
	h := sha256.New()
	for _, c := range goldenCases() {
		var before json.RawMessage
		if !c.nilBefore {
			before = json.RawMessage(c.before)
		}
		got, err := jsondiff.Diff(before, json.RawMessage(c.after))
		if err != nil {
			t.Fatal(err)
		}
		h.Write(got)
		h.Write([]byte{'\n'})
	}
	const want = "d5830ad49e5fcc2c54dfbfedadf998d46f73a1eda17d124f850c75dc94010a9d"
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		t.Fatalf("golden hash = %s, want %s", got, want)
	}
}

func TestDiffErrors(t *testing.T) {
	cases := []struct {
		name          string
		before, after json.RawMessage
		want          error
	}{
		{"nil after", json.RawMessage(`{}`), nil, jsondiff.ErrNoAfter},
		{"both nil", nil, nil, jsondiff.ErrNoAfter},
		{"invalid after", json.RawMessage(`{}`), json.RawMessage(`{"a":`), jsondiff.ErrInvalidDocument},
		{"invalid before", json.RawMessage(`{"a"}`), json.RawMessage(`{}`), jsondiff.ErrInvalidDocument},
		{"trailing value", json.RawMessage(`{}`), json.RawMessage(`{} {}`), jsondiff.ErrInvalidDocument},
		{"trailing garbage", json.RawMessage(`1 x`), json.RawMessage(`{}`), jsondiff.ErrInvalidDocument},
		{"invalid UTF-8", json.RawMessage(`{}`), json.RawMessage("{\"a\":\"\xff\"}"), jsondiff.ErrInvalidDocument},
		{"whitespace only", json.RawMessage(`{}`), json.RawMessage(`  `), jsondiff.ErrInvalidDocument},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := jsondiff.Diff(c.before, c.after)
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if got != nil {
				t.Fatalf("output %s on error", got)
			}
		})
	}
}

// TestDiffProperty: for random related and unrelated document pairs (nested
// objects, arrays that grow, shrink and reorder, nulls, type changes),
// applying Diff(before, after) to before gives after, the output is
// deterministic and contains only whole add/remove/replace ops.
func TestDiffProperty(t *testing.T) {
	r := rand.New(rand.NewPCG(4510, 6902))
	for i := range 3000 {
		before := genValue(r, 4)
		var after any
		switch i % 3 {
		case 0:
			after = genValue(r, 4)
		default:
			after = mutate(r, deepCopy(before), 4)
		}
		bb, ab := mustJSON(t, before), mustJSON(t, after)
		got, err := jsondiff.Diff(bb, ab)
		if err != nil {
			t.Fatalf("case %d: %v\nbefore %s\nafter  %s", i, err, bb, ab)
		}
		checkApplies(t, bb, ab, got)
		again, err := jsondiff.Diff(bb, ab)
		if err != nil || !bytes.Equal(got, again) {
			t.Fatalf("case %d: not deterministic\n%s\n%s", i, got, again)
		}
		if bytes.Equal(bb, ab) && string(got) != "[]" {
			t.Fatalf("case %d: equal documents gave %s", i, got)
		}
	}
}

// checkApplies asserts the patch's shape and that it turns before into after.
func checkApplies(t *testing.T, before, after, patch json.RawMessage) {
	t.Helper()
	var raws []json.RawMessage
	if err := json.Unmarshal(patch, &raws); err != nil {
		t.Fatalf("patch is not a JSON array: %v\n%s", err, patch)
	}
	// Whole ops, compact: the array is exactly its ops joined by commas, so
	// P5-12b can cut it at an op boundary without re-encoding.
	rejoined := make([][]byte, len(raws))
	for i, r := range raws {
		rejoined[i] = r
		var op patchOp
		if err := json.Unmarshal(r, &op); err != nil {
			t.Fatal(err)
		}
		switch op.Op {
		case "add", "replace":
			if op.Value == nil {
				t.Fatalf("%s without value: %s", op.Op, r)
			}
		case "remove":
			if op.Value != nil {
				t.Fatalf("remove with value: %s", r)
			}
		default:
			t.Fatalf("op %q emitted: %s", op.Op, r)
		}
	}
	if j := "[" + string(bytes.Join(rejoined, []byte(","))) + "]"; j != string(patch) {
		t.Fatalf("patch is not compact whole ops\n got: %s\njoin: %s", patch, j)
	}

	var doc any
	if len(before) > 0 {
		var err error
		if doc, err = decodeDoc(before); err != nil {
			t.Fatal(err)
		}
	}
	res, err := apply(doc, patch)
	if err != nil {
		t.Fatalf("apply: %v\nbefore %s\nafter  %s\npatch  %s", err, before, after, patch)
	}
	want, err := decodeDoc(after)
	if err != nil {
		t.Fatal(err)
	}
	if !sameValue(res, want) {
		t.Fatalf("apply(before, diff) != after\nbefore %s\nafter  %s\npatch  %s\ngot    %s", before, after, patch, mustJSON(t, res))
	}
}

func FuzzDiff(f *testing.F) {
	for _, c := range goldenCases() {
		if !c.nilBefore {
			f.Add([]byte(c.before), []byte(c.after))
		}
	}
	f.Fuzz(func(t *testing.T, before, after []byte) {
		if len(before) == 0 || !json.Valid(before) || !json.Valid(after) {
			return
		}
		got, err := jsondiff.Diff(before, after)
		if err != nil {
			return // e.g. invalid UTF-8, which json.Valid accepts
		}
		checkApplies(t, before, after, got)
	})
}

func keyPool() []string {
	return []string{"a", "b", "c", "id", "x/y", "m~n", "", "é", "\U0001F600", "｡", "~1"}
}

func pickKey(r *rand.Rand) string {
	ks := keyPool()
	return ks[r.IntN(len(ks))]
}

func genValue(r *rand.Rand, depth int) any {
	k := r.IntN(8)
	if depth <= 0 {
		k = r.IntN(5)
	}
	switch k {
	case 0:
		return nil
	case 1:
		return r.IntN(2) == 0
	case 2:
		return json.Number(strconv.Itoa(r.IntN(5) - 2))
	case 3:
		return []string{"", "s", "t", "<&>", "\n"}[r.IntN(5)]
	case 4:
		return json.Number([]string{"1.0", "1e2", "-0", "0.5"}[r.IntN(4)])
	case 5, 6:
		m := map[string]any{}
		for range r.IntN(5) {
			m[pickKey(r)] = genValue(r, depth-1)
		}
		return m
	default:
		a := make([]any, r.IntN(5))
		for i := range a {
			a[i] = genValue(r, depth-1)
		}
		return a
	}
}

// mutate makes a related document: it edits members and elements, grows,
// shrinks and shuffles arrays, and sometimes changes a value's type.
func mutate(r *rand.Rand, v any, depth int) any {
	if r.IntN(6) == 0 {
		return genValue(r, depth)
	}
	switch t := v.(type) {
	case map[string]any:
		for k, e := range t {
			switch r.IntN(4) {
			case 0:
				delete(t, k)
			case 1:
				t[k] = mutate(r, e, depth-1)
			}
		}
		if r.IntN(2) == 0 {
			t[pickKey(r)] = genValue(r, depth-1)
		}
		return t
	case []any:
		switch r.IntN(4) {
		case 0:
			t = append(t, genValue(r, depth-1))
		case 1:
			if len(t) > 0 {
				t = t[:r.IntN(len(t))]
			}
		case 2:
			r.Shuffle(len(t), func(i, j int) { t[i], t[j] = t[j], t[i] })
		}
		for i := range t {
			if r.IntN(3) == 0 {
				t[i] = mutate(r, t[i], depth-1)
			}
		}
		return t
	default:
		return v
	}
}

func deepCopy(v any) any {
	switch t := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, e := range t {
			m[k] = deepCopy(e)
		}
		return m
	case []any:
		a := make([]any, len(t))
		for i, e := range t {
			a[i] = deepCopy(e)
		}
		return a
	default:
		return v
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
