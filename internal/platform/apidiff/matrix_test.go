package apidiff_test

import (
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/apidiff"
)

const head = "openapi: 3.0.3\ninfo: {title: t, version: '1'}\npaths:\n"

const other = "  /other:\n    get:\n      responses:\n        '204': {description: ok}\n"

// doc wraps the operations (YAML, indented under /items) in a minimal valid
// OpenAPI document.
func doc(ops string) string { return head + "  /items:\n" + ops + other }

const getOK = "    get:\n      responses:\n        '200': {description: ok}\n"

const created = "      responses:\n        '201': {description: ok}\n"

func getWith(params, resp string) string {
	return "    get:\n" + params + "      responses:\n        '200':\n          description: ok\n" + resp
}

func param(extra string) string {
	return "      parameters:\n        - {name: q, in: query, " + extra + "}\n"
}

func jsonResp(schema string) string {
	return "          content:\n            application/json: {schema: " + schema + "}\n"
}

func post(body string) string {
	return "    post:\n" + body + created
}

func reqBody(required bool, schema string) string {
	r := ""
	if required {
		r = "        required: true\n"
	}
	return "      requestBody:\n" + r + "        content:\n          application/json: {schema: " + schema + "}\n"
}

func TestBreakingChangeMatrix(t *testing.T) {
	t.Parallel()
	qStr := "schema: {type: string}"
	tests := []struct {
		name, oldD, newD, want string
	}{
		{"path removed", doc(getOK), head + other, "path /items removed"},
		{"operation removed", doc(getOK + post("")), doc(getOK), "POST /items removed"},
		{"new required parameter", doc(getOK), doc(getWith(param("required: true, "+qStr), "")), "new required parameter q"},
		{"parameter became required", doc(getWith(param(qStr), "")), doc(getWith(param("required: true, "+qStr), "")), "parameter q became required"},
		{"parameter removed", doc(getWith(param(qStr), "")), doc(getOK), "parameter q removed"},
		{"parameter type changed", doc(getWith(param(qStr), "")), doc(getWith(param("schema: {type: integer}"), "")), "parameter q"},
		{"request body removed", doc(post(reqBody(false, "{type: object}"))), doc(post("")), "request body removed"},
		{"new required request body", doc(post("")), doc(post(reqBody(true, "{type: object}"))), "new required request body"},
		{"request body became required", doc(post(reqBody(false, "{type: object}"))), doc(post(reqBody(true, "{type: object}"))), "request body became required"},
		{"response removed", doc(getOK), doc("    get:\n      responses:\n        '204': {description: ok}\n"), "response 200 removed"},
		{"media type removed", doc(getWith("", jsonResp("{type: object}"))),
			doc(getWith("", "          content:\n            text/plain: {schema: {type: string}}\n")), "media type application/json removed"},
		{"schema removed", doc(getWith("", jsonResp("{type: object}"))),
			doc(getWith("", "          content:\n            application/json: {}\n")), "schema removed"},
		{"type changed", doc(getWith("", jsonResp("{type: object}"))), doc(getWith("", jsonResp("{type: array, items: {type: string}}"))), "type changed"},
		{"format changed", doc(getWith("", jsonResp("{type: object, properties: {a: {type: string, format: uuid}}}"))),
			doc(getWith("", jsonResp("{type: object, properties: {a: {type: string, format: date}}}"))), "format changed"},
		{"property removed", doc(getWith("", jsonResp("{type: object, properties: {a: {type: string}}}"))),
			doc(getWith("", jsonResp("{type: object, properties: {b: {type: string}}}"))), "property a removed"},
		{"response property no longer always present",
			doc(getWith("", jsonResp("{type: object, required: [a], properties: {a: {type: string}}}"))),
			doc(getWith("", jsonResp("{type: object, properties: {a: {type: string}}}"))), "property a is no longer always present"},
		{"request property became required",
			doc(post(reqBody(false, "{type: object, properties: {a: {type: string}}}"))),
			doc(post(reqBody(false, "{type: object, required: [a], properties: {a: {type: string}}}"))), "property a became required"},
		{"request enum value removed", doc(post(reqBody(false, "{type: string, enum: [x, y]}"))),
			doc(post(reqBody(false, "{type: string, enum: [x]}"))), "enum value y removed"},
		{"request enum added", doc(post(reqBody(false, "{type: string}"))),
			doc(post(reqBody(false, "{type: string, enum: [x]}"))), "enum added"},
		{"array items type changed", doc(getWith("", jsonResp("{type: array, items: {type: string}}"))),
			doc(getWith("", jsonResp("{type: array, items: {type: integer}}"))), "[]: type changed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := check(t, tt.oldD, tt.newD)
			if !strings.Contains(strings.Join(out, "\n"), tt.want) {
				t.Fatalf("want a report containing %q, got %v", tt.want, out)
			}
		})
	}
}

func TestAdditiveChangesAndMovedParameter(t *testing.T) {
	t.Parallel()
	qStr := "schema: {type: string}"
	base := doc(getWith(param(qStr), ""))
	next := doc(getWith(param(qStr)+"        - {name: extra, in: query, "+qStr+"}\n", "") + post(""))
	if out := check(t, base, next); len(out) != 0 {
		t.Fatalf("adding an optional parameter and an operation is additive: %v", out)
	}
	// A parameter with the same name in another location is a different one.
	moved := doc(getWith("      parameters:\n        - {name: q, in: header, "+qStr+"}\n", ""))
	var removed bool
	for _, l := range check(t, base, moved) {
		removed = removed || strings.Contains(l, "parameter q removed")
	}
	if !removed {
		t.Fatal("moving a parameter must report its removal")
	}
}

func TestInvalidSpecsAreErrors(t *testing.T) {
	t.Parallel()
	good := []byte(doc(getOK))
	if _, err := apidiff.Check([]byte("{not json"), good); err == nil || !strings.Contains(err.Error(), "base spec") {
		t.Fatalf("bad base: %v", err)
	}
	if _, err := apidiff.Check(good, []byte("{not json")); err == nil || !strings.Contains(err.Error(), "new spec") {
		t.Fatalf("bad new: %v", err)
	}
	// Parses but fails validation: no info.
	if _, err := apidiff.Check(good, []byte(`{"openapi":"3.0.3","paths":{}}`)); err == nil {
		t.Fatal("an invalid document must be an error")
	}
}
