package def

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// base is a valid definition; each rejection case mutates one part of it.
func base() map[string]any {
	return map[string]any{
		"format":  Format,
		"trigger": map[string]any{"on": "record.updated", "entity": "ticket"},
		"start":   "wait",
		"steps": []any{
			map[string]any{"key": "wait", "type": "timer", "duration": "PT5S", "next": "check"},
			map[string]any{"key": "check", "type": "condition", "field": "/priority", "op": "eq", "value": "high", "then": "route", "else": "done"},
			map[string]any{"key": "route", "type": "branch", "cases": []any{
				map[string]any{"field": "/team", "op": "in", "value": []any{"a", "b"}, "next": "approve"},
			}, "default": "done"},
			map[string]any{"key": "approve", "type": "wait_signal", "name": "approve", "timeout": "P1DT2H", "on_timeout": "done", "next": "apply"},
			map[string]any{"key": "apply", "type": "update_record", "set": map[string]any{"approved": true}, "next": "done"},
			map[string]any{"key": "done", "type": "end"},
		},
	}
}

func step(d map[string]any, i int) map[string]any { return d["steps"].([]any)[i].(map[string]any) }

func encode(t testing.TB, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseValid(t *testing.T) {
	d, err := Parse(encode(t, base()))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := len(d.Steps); got != 6 {
		t.Fatalf("steps = %d", got)
	}
	wait, _ := d.Step("wait")
	if wait.WaitSeconds() != 5 {
		t.Errorf("timer WaitSeconds = %d, want 5", wait.WaitSeconds())
	}
	approve, _ := d.Step("approve")
	if approve.WaitSeconds() != 93600 {
		t.Errorf("wait_signal WaitSeconds = %d, want 93600", approve.WaitSeconds())
	}
	if done, _ := d.Step("done"); len(done.Edges()) != 0 {
		t.Errorf("end has out-edges: %v", done.Edges())
	}
	if got := d.SignalNames(); !reflect.DeepEqual(got, []string{"approve"}) {
		t.Errorf("SignalNames = %v", got)
	}
}

func TestParseRejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(d map[string]any)
		raw    string // used instead of base() when set
		paths  []string
	}{
		{name: "not json", raw: `{"format":`, paths: []string{""}},
		{name: "trailing data", raw: `{} {}`, paths: []string{""}},
		{name: "wrong format", mutate: func(d map[string]any) { d["format"] = "nava.workflow/v2" }, paths: []string{"/format"}},
		{name: "unknown root field", mutate: func(d map[string]any) { d["expr"] = "x" }, paths: []string{"/expr"}},
		{name: "bad trigger", mutate: func(d map[string]any) { d["trigger"] = map[string]any{"on": "record.deleted", "entity": "ticket"} }, paths: []string{"/trigger/on"}},
		{name: "missing start", mutate: func(d map[string]any) { delete(d, "start") }, paths: []string{"/start"}},
		{name: "unknown start", mutate: func(d map[string]any) { d["start"] = "nowhere" }, paths: []string{"/start"}},
		{name: "no steps", mutate: func(d map[string]any) { d["steps"] = []any{} }, paths: []string{"/steps"}},
		{name: "duplicate key", mutate: func(d map[string]any) { step(d, 1)["key"] = "wait" }, paths: []string{"/steps/1/key"}},
		{name: "key upper case", mutate: func(d map[string]any) { step(d, 5)["key"] = "Done" }, paths: []string{"/steps/5/key"}},
		{name: "key starts with digit", mutate: func(d map[string]any) { step(d, 5)["key"] = "1done" }, paths: []string{"/steps/5/key"}},
		{name: "key too long", mutate: func(d map[string]any) { step(d, 5)["key"] = "a" + strings.Repeat("b", 40) }, paths: []string{"/steps/5/key"}},
		{name: "key with colon", mutate: func(d map[string]any) { step(d, 5)["key"] = "signal:x" }, paths: []string{"/steps/5/key"}},
		{name: "unknown type", mutate: func(d map[string]any) { step(d, 5)["type"] = "script" }, paths: []string{"/steps/5/type"}},
		{name: "unknown next", mutate: func(d map[string]any) { step(d, 0)["next"] = "nowhere" }, paths: []string{"/steps/0/next"}},
		{name: "unknown then", mutate: func(d map[string]any) { step(d, 1)["then"] = "nowhere" }, paths: []string{"/steps/1/then"}},
		{name: "unknown else", mutate: func(d map[string]any) { step(d, 1)["else"] = "nowhere" }, paths: []string{"/steps/1/else"}},
		{name: "unknown case next", mutate: func(d map[string]any) {
			step(d, 2)["cases"].([]any)[0].(map[string]any)["next"] = "nowhere"
		}, paths: []string{"/steps/2/cases/0/next"}},
		{name: "unknown default", mutate: func(d map[string]any) { step(d, 2)["default"] = "nowhere" }, paths: []string{"/steps/2/default"}},
		{name: "unknown on_timeout", mutate: func(d map[string]any) { step(d, 3)["on_timeout"] = "nowhere" }, paths: []string{"/steps/3/on_timeout"}},
		{name: "end with next", mutate: func(d map[string]any) { step(d, 5)["next"] = "wait" }, paths: []string{"/steps/5/next"}},
		{name: "cycle via next", mutate: func(d map[string]any) { step(d, 4)["next"] = "wait" }, paths: []string{"/steps/4/next"}},
		{name: "cycle via else", mutate: func(d map[string]any) { step(d, 1)["else"] = "wait" }, paths: []string{"/steps/1/else"}},
		{name: "cycle via on_timeout", mutate: func(d map[string]any) { step(d, 3)["on_timeout"] = "check" }, paths: []string{"/steps/3/on_timeout"}},
		{name: "cycle via case", mutate: func(d map[string]any) {
			step(d, 2)["cases"].([]any)[0].(map[string]any)["next"] = "route"
		}, paths: []string{"/steps/2/cases/0/next"}},
		{name: "self loop default", mutate: func(d map[string]any) { step(d, 2)["default"] = "route" }, paths: []string{"/steps/2/default"}},
		{name: "unreachable step", mutate: func(d map[string]any) {
			d["steps"] = append(d["steps"].([]any), map[string]any{"key": "orphan", "type": "end"})
		}, paths: []string{"/steps/6"}},
		{name: "unreachable when start skips", mutate: func(d map[string]any) { d["start"] = "check" }, paths: []string{"/steps/0"}},
		{name: "too many steps", mutate: func(d map[string]any) {
			steps := d["steps"].([]any)
			for i := len(steps); i <= MaxSteps; i++ {
				steps = append(steps, map[string]any{"key": fmt.Sprintf("e%d", i), "type": "end"})
			}
			d["steps"] = steps
		}, paths: []string{"/steps"}},
		{name: "timer without duration", mutate: func(d map[string]any) { delete(step(d, 0), "duration") }, paths: []string{"/steps/0/duration"}},
		{name: "duration field reference pointer", mutate: func(d map[string]any) { step(d, 0)["duration"] = "/wait_for" }, paths: []string{"/steps/0/duration"}},
		{name: "duration field reference object", mutate: func(d map[string]any) {
			step(d, 0)["duration"] = map[string]any{"field": "/wait_for"}
		}, paths: []string{"/steps/0/duration"}},
		{name: "duration template", mutate: func(d map[string]any) { step(d, 0)["duration"] = "PT{{record.wait}}S" }, paths: []string{"/steps/0/duration"}},
		{name: "duration years", mutate: func(d map[string]any) { step(d, 0)["duration"] = "P1Y" }, paths: []string{"/steps/0/duration"}},
		{name: "duration zero", mutate: func(d map[string]any) { step(d, 0)["duration"] = "PT0S" }, paths: []string{"/steps/0/duration"}},
		{name: "duration over 30d", mutate: func(d map[string]any) { step(d, 0)["duration"] = "P30DT1S" }, paths: []string{"/steps/0/duration"}},
		{name: "wait_signal without timeout", mutate: func(d map[string]any) { delete(step(d, 3), "timeout") }, paths: []string{"/steps/3/timeout"}},
		{name: "wait_signal without on_timeout", mutate: func(d map[string]any) { delete(step(d, 3), "on_timeout") }, paths: []string{"/steps/3/on_timeout"}},
		{name: "wait_signal timeout field reference", mutate: func(d map[string]any) { step(d, 3)["timeout"] = "/deadline" }, paths: []string{"/steps/3/timeout"}},
		{name: "wait_signal timeout over 30d", mutate: func(d map[string]any) { step(d, 3)["timeout"] = "P31D" }, paths: []string{"/steps/3/timeout"}},
		{name: "wait_signal bad name", mutate: func(d map[string]any) { step(d, 3)["name"] = "Approve!" }, paths: []string{"/steps/3/name"}},
		{name: "edge field reference", mutate: func(d map[string]any) { step(d, 0)["next"] = "/next_step" }, paths: []string{"/steps/0/next"}},
		{name: "edge object", mutate: func(d map[string]any) { step(d, 1)["then"] = map[string]any{"field": "/x"} }, paths: []string{"/steps/1/then"}},
		{name: "condition bad op", mutate: func(d map[string]any) { step(d, 1)["op"] = "gt" }, paths: []string{"/steps/1/op"}},
		{name: "condition field not a pointer", mutate: func(d map[string]any) { step(d, 1)["field"] = "priority" }, paths: []string{"/steps/1/field"}},
		{name: "condition in needs array", mutate: func(d map[string]any) { step(d, 1)["op"] = "in" }, paths: []string{"/steps/1/value"}},
		{name: "condition missing value", mutate: func(d map[string]any) { delete(step(d, 1), "value") }, paths: []string{"/steps/1/value"}},
		{name: "condition expression field", mutate: func(d map[string]any) { step(d, 1)["expr"] = "a > b" }, paths: []string{"/steps/1/expr"}},
		{name: "branch no cases", mutate: func(d map[string]any) { step(d, 2)["cases"] = []any{} }, paths: []string{"/steps/2/cases"}},
		{name: "branch missing default", mutate: func(d map[string]any) { delete(step(d, 2), "default") }, paths: []string{"/steps/2/default"}},
		{name: "update_record empty set", mutate: func(d map[string]any) { step(d, 4)["set"] = map[string]any{} }, paths: []string{"/steps/4/set"}},
		{name: "step id supplied", mutate: func(d map[string]any) { step(d, 5)["id"] = "8b82519a-0b33-4718-937d-972c8204d4ef" }, paths: []string{"/steps/5/id"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := []byte(tc.raw)
			if tc.raw == "" {
				d := base()
				tc.mutate(d)
				b = encode(t, d)
			}
			_, err := Parse(b)
			var inv *ErrInvalidDefinition
			if !errors.As(err, &inv) {
				t.Fatalf("Parse error = %v, want *ErrInvalidDefinition", err)
			}
			if !reflect.DeepEqual(inv.Paths, tc.paths) {
				t.Fatalf("Paths = %q, want %q", inv.Paths, tc.paths)
			}
		})
	}
}

// The error never echoes definition values.
func TestErrInvalidDefinitionCarriesNoValues(t *testing.T) {
	d := base()
	step(d, 0)["duration"] = "PT{{secret_value}}S"
	_, err := Parse(encode(t, d))
	if err == nil || strings.Contains(err.Error(), "secret_value") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseReportsEveryUnknownEdge(t *testing.T) {
	d := base()
	step(d, 0)["next"] = "x"
	step(d, 4)["next"] = "y"
	_, err := Parse(encode(t, d))
	var inv *ErrInvalidDefinition
	if !errors.As(err, &inv) || !reflect.DeepEqual(inv.Paths, []string{"/steps/0/next", "/steps/4/next"}) {
		t.Fatalf("err = %v", err)
	}
}

func TestParseMaxStepsAccepted(t *testing.T) {
	steps := make([]any, 0, MaxSteps)
	for i := 0; i < MaxSteps-1; i++ {
		steps = append(steps, map[string]any{"key": fmt.Sprintf("s%d", i), "type": "timer", "duration": "PT1S", "next": fmt.Sprintf("s%d", i+1)})
	}
	steps = append(steps, map[string]any{"key": fmt.Sprintf("s%d", MaxSteps-1), "type": "end"})
	d := map[string]any{"format": Format, "trigger": map[string]any{"on": "record.created", "entity": "ticket"}, "start": "s0", "steps": steps}
	if _, err := Parse(encode(t, d)); err != nil {
		t.Fatalf("Parse of %d steps: %v", MaxSteps, err)
	}
}

func TestFixturesParse(t *testing.T) {
	for _, name := range []string{"fixture-v1", "fixture-v2"} {
		b, err := os.ReadFile("testdata/" + name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		d, err := Parse(b)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var keys []StepType
		for cur := d.Start; ; {
			s, _ := d.Step(cur)
			keys = append(keys, s.Type)
			if s.Type == TypeEnd {
				break
			}
			cur = s.Edges()[0].To // then / next: the happy path
		}
		want := []StepType{TypeTimer, TypeCondition, TypeWaitSignal, TypeUpdateRecord, TypeEnd}
		if !reflect.DeepEqual(keys, want) {
			t.Errorf("%s path = %v, want %v", name, keys, want)
		}
		if s, _ := d.Step(d.Start); s.WaitSeconds() != 5 {
			t.Errorf("%s timer = %ds, want 5", name, s.WaitSeconds())
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, name := range []string{"fixture-v1", "fixture-v2"} {
		b, err := os.ReadFile("testdata/" + name + ".json")
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Add(encode(f, base()))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"format":"nava.workflow/v1","start":"a","steps":[{"key":"a","type":"timer","duration":"PT1S","next":"a"}]}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		d, err := Parse(b)
		if err != nil {
			var inv *ErrInvalidDefinition
			if !errors.As(err, &inv) {
				t.Fatalf("non-validation error: %v", err)
			}
			return
		}
		// An accepted definition is a DAG whose waits are in range.
		for i := range d.Steps {
			s := &d.Steps[i]
			if (s.Type == TypeTimer || s.Type == TypeWaitSignal) && (s.WaitSeconds() < MinWaitSeconds || s.WaitSeconds() > MaxWaitSeconds) {
				t.Fatalf("step %s wait %d out of range", s.Key, s.WaitSeconds())
			}
			for _, e := range s.Edges() {
				if _, ok := d.Step(e.To); !ok {
					t.Fatalf("step %s edge %s to unknown %s", s.Key, e.Field, e.To)
				}
			}
		}
		_ = AssignStepIDs(d, nil)
	})
}
