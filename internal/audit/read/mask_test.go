package read

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
)

func mask(t *testing.T, diff string) (string, bool) {
	t.Helper()
	out, redacted, err := MaskDiff([]byte(diff), []patchwalk.Pointer{"/password"})
	if err != nil {
		t.Fatalf("MaskDiff: %v", err)
	}
	if !json.Valid(out) {
		t.Fatalf("masked diff is not JSON: %s", out)
	}
	return string(out), redacted
}

// CI-A12 (fixture masker): masks hold on a root add, a nested replace of a
// parent, copy/move from a masked path, and test.
func TestMaskDiffHolds(t *testing.T) {
	for _, tc := range []struct {
		name, diff, want string
	}{
		{"root add", `[{"op":"add","path":"","value":{"a":1,"password":"hunter2"}}]`,
			`[{"op":"add","path":"","value":{"a":1,"password":{"redacted":"mask"}}}]`},
		{"add at the path", `[{"op":"add","path":"/password","value":"hunter2"}]`,
			`[{"op":"add","path":"/password","value":{"redacted":"mask"}}]`},
		{"replace under the path", `[{"op":"replace","path":"/password/x","value":"hunter2"}]`,
			`[{"op":"replace","path":"/password/x","value":{"redacted":"mask"}}]`},
		{"nested replace of a parent", `[{"op":"replace","path":"","value":{"password":{"k":[1,2]},"n":1.50}}]`,
			`[{"op":"replace","path":"","value":{"n":1.50,"password":{"redacted":"mask"}}}]`},
		{"copy from a masked path", `[{"op":"copy","from":"/password","path":"/note"}]`, `[{"redacted":"op"}]`},
		{"move from a masked path", `[{"op":"move","from":"/password","path":"/note"}]`, `[{"redacted":"op"}]`},
		{"move from a parent of a masked path", `[{"op":"move","from":"","path":"/note"}]`, `[{"redacted":"op"}]`},
		{"test", `[{"op":"test","path":"/password","value":"hunter2"}]`,
			`[{"op":"test","path":"/password","value":{"redacted":"mask"}}]`},
		{"other ops stay", `[{"op":"add","path":"/a","value":1},{"op":"replace","path":"/password","value":"x"},{"op":"remove","path":"/b"}]`,
			`[{"op":"add","path":"/a","value":1},{"op":"replace","path":"/password","value":{"redacted":"mask"}},{"op":"remove","path":"/b"}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, redacted := mask(t, tc.diff)
			if !redacted {
				t.Error("redacted = false")
			}
			if tc.name == "nested replace of a parent" {
				// number text and key order are re-encoded; compare as values
				var g, w any
				_ = json.Unmarshal([]byte(got), &g)
				_ = json.Unmarshal([]byte(tc.want), &w)
				if a, b := mustJSON(g), mustJSON(w); a != b {
					t.Errorf("got %s want %s", got, tc.want)
				}
			} else if got != tc.want {
				t.Errorf("got %s\nwant %s", got, tc.want)
			}
			if strings.Contains(got, "hunter2") {
				t.Errorf("secret survives: %s", got)
			}
		})
	}
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func TestMaskDiffLeavesUntouchedBytes(t *testing.T) {
	diff := `[{"op":"add","path":"/a","value":1.0}]`
	got, redacted := mask(t, diff)
	if redacted || got != diff {
		t.Errorf("got %s redacted=%v, want the stored bytes", got, redacted)
	}
	// a masked pointer that the value does not hold changes nothing either
	diff = `[{"op":"add","path":"","value":{"a":1}}]`
	if got, redacted := mask(t, diff); redacted || got != diff {
		t.Errorf("absent member: got %s redacted=%v", got, redacted)
	}
	if out, red, _ := MaskDiff([]byte(diff), nil); red || string(out) != diff {
		t.Error("no masked paths must return the stored bytes")
	}
}

func TestMaskDiffMalformedFailsClosed(t *testing.T) {
	got, redacted := mask(t, `[{"op":"add","path":"/password","value":"x"`)
	if !redacted || got != `{"redacted":"diff"}` {
		t.Errorf("got %s redacted=%v", got, redacted)
	}
}

func TestReplayRebuildsAndStopsAtGaps(t *testing.T) {
	create := Entry{Action: "create", Diff: json.RawMessage(`[{"op":"add","path":"","value":{"a":1,"tags":["x"]}}]`)}
	upd := Entry{Action: "update", Diff: json.RawMessage(`[{"op":"replace","path":"/a","value":2},{"op":"add","path":"/tags/1","value":"y"}]`)}
	rm := Entry{Action: "update", Diff: json.RawMessage(`[{"op":"remove","path":"/tags/0"}]`)}
	erased := Entry{Action: "update"} // diff erased
	after := Entry{Action: "update", Diff: json.RawMessage(`[{"op":"replace","path":"/a","value":3}]`)}
	got := Replay([]Entry{create, upd, rm, erased, after})
	if !got[0].Known || got[0].Before != nil || mustJSON(got[0].After) != `{"a":1,"tags":["x"]}` {
		t.Errorf("create: %+v", got[0])
	}
	if mustJSON(got[1].Before) != `{"a":1,"tags":["x"]}` || mustJSON(got[1].After) != `{"a":2,"tags":["x","y"]}` {
		t.Errorf("update: %+v", got[1])
	}
	if mustJSON(got[2].After) != `{"a":2,"tags":["y"]}` {
		t.Errorf("remove: %+v", got[2])
	}
	if got[3].Known || got[4].Known {
		t.Errorf("an erased diff must make this and later steps unknown: %+v %+v", got[3], got[4])
	}
	if mustJSON(got[1].Before) != `{"a":1,"tags":["x"]}` {
		t.Error("replay mutated an earlier snapshot")
	}
}

func TestReplayCreateRestarts(t *testing.T) {
	bad := Entry{Action: "update", Diff: json.RawMessage(`[{"op":"replace","path":"/zzz","value":1}]`)}
	create := Entry{Action: "create", Diff: json.RawMessage(`[{"op":"add","path":"","value":{"a":1}}]`)}
	got := Replay([]Entry{bad, create})
	if got[0].Known || !got[1].Known {
		t.Errorf("got %+v", got)
	}
}
