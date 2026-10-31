package def

import (
	"bytes"
	"encoding/json"
	"testing"
)

func record(t *testing.T, s string, useNumber bool) map[string]any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader([]byte(s)))
	if useNumber {
		dec.UseNumber()
	}
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestConditionHolds(t *testing.T) {
	const rec = `{"d":0.1,"de":1e-1,"neg":-2.5e3,"n":1,"big":9007199254740993,"s":"1","b":true,"z":null,"obj":{"a":[1,2]},"arr":["x","y"],"a/b":"slash","t~":"tilde"}`
	cases := []struct {
		field string
		op    Op
		value string
		want  bool
	}{
		{"/n", OpEq, `1`, true},
		{"/n", OpEq, `1.0`, true}, // decoded values: 1 equals 1.0
		{"/n", OpEq, `1e0`, true},
		{"/n", OpEq, `"1"`, false}, // no coercion
		{"/s", OpEq, `1`, false},
		{"/s", OpEq, `"1"`, true},
		{"/b", OpEq, `true`, true},
		{"/z", OpEq, `null`, true},
		{"/z", OpNe, `null`, false},
		{"/obj", OpEq, `{"a":[1.0,2]}`, true},
		{"/obj/a/1", OpEq, `2`, true},
		{"/arr/0", OpEq, `"x"`, true},
		{"/arr/01", OpEq, `"y"`, false}, // leading zero is not an index
		{"/arr/2", OpEq, `"y"`, false},
		{"/a~1b", OpEq, `"slash"`, true},
		{"/t~0", OpEq, `"tilde"`, true},
		{"/n", OpNe, `2`, true},
		{"/n", OpNe, `1`, false},
		{"/n", OpIn, `[3,1.0]`, true},
		{"/n", OpIn, `[3,"1"]`, false},
		{"/s", OpIn, `[]`, false},
		{"/n", OpEq, `1e999999999`, false}, // overflows float64: equals nothing, and is cheap
		// Decimals compare by decimal value for float64 and json.Number records.
		{"/d", OpEq, `0.1`, true},
		{"/d", OpEq, `1e-1`, true},
		{"/d", OpEq, `0.10`, true},
		{"/d", OpIn, `[0.1]`, true},
		{"/d", OpIn, `[2,1e-1]`, true},
		{"/d", OpNe, `0.1`, false},
		{"/d", OpEq, `0.2`, false},
		{"/de", OpEq, `0.1`, true},
		{"/de", OpIn, `[0.1]`, true},
		{"/neg", OpEq, `-2500`, true},
		{"/neg", OpEq, `-2500.0`, true},
		// A missing field equals nothing.
		{"/missing", OpEq, `null`, false},
		{"/missing", OpIn, `[null]`, false},
		{"/missing", OpNe, `1`, true},
		{"/n/deeper", OpEq, `1`, false},
	}
	for _, useNumber := range []bool{false, true} {
		r := record(t, rec, useNumber)
		for _, tc := range cases {
			c := Condition{Field: tc.field, Op: tc.op, Value: json.RawMessage(tc.value)}
			if got := c.Holds(r); got != tc.want {
				t.Errorf("useNumber=%v %s %s %s = %v, want %v", useNumber, tc.field, tc.op, tc.value, got, tc.want)
			}
		}
	}
	// With json.Number, integers beyond float64 precision compare exactly.
	r := record(t, rec, true)
	if (Condition{Field: "/big", Op: OpEq, Value: json.RawMessage(`9007199254740992`)}).Holds(r) {
		t.Error("9007199254740993 equals 9007199254740992 under json.Number")
	}
}

// A branch takes the first case that holds, in array order, else default.
func TestBranchCaseOrderContract(t *testing.T) {
	d := mustParse(t, encode(t, base()))
	route, _ := d.Step("route")
	route.Cases = append(route.Cases, Case{Condition: Condition{Field: "/team", Op: OpEq, Value: json.RawMessage(`"a"`)}, Next: "done"})
	pick := func(r map[string]any) string {
		for _, c := range route.Cases {
			if c.Holds(r) {
				return c.Next
			}
		}
		return route.Default
	}
	if got := pick(map[string]any{"team": "a"}); got != "approve" {
		t.Errorf("both cases hold: took %s, want the first case's approve", got)
	}
	if got := pick(map[string]any{"team": "z"}); got != "done" {
		t.Errorf("no case holds: took %s, want default done", got)
	}
}
