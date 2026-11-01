package pipeline_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// RFC 7396 appendix A cases, plus that the patch is never shared with the
// result.
func TestMergePatch(t *testing.T) {
	cases := []struct{ target, patch, want string }{
		{`{"a":"b"}`, `{"a":"c"}`, `{"a":"c"}`},
		{`{"a":"b"}`, `{"b":"c"}`, `{"a":"b","b":"c"}`},
		{`{"a":"b"}`, `{"a":null}`, `{}`},
		{`{"a":"b","b":"c"}`, `{"a":null}`, `{"b":"c"}`},
		{`{"a":["b"]}`, `{"a":"c"}`, `{"a":"c"}`},
		{`{"a":"c"}`, `{"a":["b"]}`, `{"a":["b"]}`},
		{`{"a":{"b":"c"}}`, `{"a":{"b":"d","c":null}}`, `{"a":{"b":"d"}}`},
		{`{"a":[{"b":"c"}]}`, `{"a":[1]}`, `{"a":[1]}`},
		{`{"e":null}`, `{"a":1}`, `{"e":null,"a":1}`},
		{`{}`, `{"a":{"bb":{"ccc":null}}}`, `{"a":{"bb":{}}}`},
		{`{"tags":["x","y","z"]}`, `{"tags":["q"]}`, `{"tags":["q"]}`},
	}
	for _, c := range cases {
		got := pipeline.MergePatch(decode(t, c.target), decode(t, c.patch))
		if want := decode(t, c.want); !reflect.DeepEqual(got, want) {
			t.Errorf("merge %s with %s = %v, want %v", c.target, c.patch, got, want)
		}
	}

	patch := decode(t, `{"tags":["a"],"obj":{"k":"v"}}`)
	got := pipeline.MergePatch(map[string]any{}, patch)
	got["tags"].([]any)[0] = "changed"
	got["obj"].(map[string]any)["k"] = "changed"
	if patch["tags"].([]any)[0] != "a" || patch["obj"].(map[string]any)["k"] != "v" {
		t.Fatal("the result shares values with the patch")
	}
}
