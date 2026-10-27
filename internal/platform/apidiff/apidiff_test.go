package apidiff_test

import (
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apidiff"
)

func spec(t *testing.T) string {
	t.Helper()
	data, err := api.Files.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func check(t *testing.T, oldS, newS string) []string {
	t.Helper()
	out, err := apidiff.Check([]byte(oldS), []byte(newS))
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestIdenticalSpecIsAdditive(t *testing.T) {
	t.Parallel()
	if out := check(t, spec(t), spec(t)); len(out) != 0 {
		t.Errorf("identical specs: %v", out)
	}
}

// TestRemovingAResponseFieldFails is the CI test of T20: dropping a field from
// a response of the real contract is reported.
func TestRemovingAResponseFieldFails(t *testing.T) {
	t.Parallel()
	base := spec(t)
	next := strings.Replace(base, "        tenant_id:\n          type: string\n          format: uuid\n", "", 1)
	next = strings.Replace(next, "required: [user_id, tenant_id, groups, idp_alias, group_paths]", "required: [user_id, groups, idp_alias, group_paths]", 1)
	if next == base {
		t.Fatal("test edit did not change the spec")
	}
	out := check(t, base, next)
	if !strings.Contains(strings.Join(out, "\n"), "property tenant_id removed") {
		t.Errorf("removal not reported: %v", out)
	}
}

func TestBreakingChanges(t *testing.T) {
	t.Parallel()
	base := spec(t)
	tests := []struct{ name, from, to, want string }{
		{"type changed", "        tenant_id:\n          type: string\n          format: uuid", "        tenant_id:\n          type: integer\n          format: uuid", "type changed"},
		{"format changed", "        user_id:\n          type: string\n          format: uuid", "        user_id:\n          type: string\n          format: date", "format changed"},
		{"response removed", "        \"501\":\n          $ref: \"#/components/responses/Problem\"\n", "", "response 501 removed"},
		{"required dropped", "required: [user_id, tenant_id, groups, idp_alias, group_paths]", "required: [user_id, groups, idp_alias, group_paths]", "no longer always present"},
		{"path removed", "  /api/v1/me:", "  /api/v1/other:", "path /api/v1/me removed"},
		{"operation removed", "    get:\n      operationId: getMe", "    head:\n      operationId: getMe", "GET /api/v1/me removed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			next := strings.Replace(base, tt.from, tt.to, 1)
			if next == base {
				t.Fatal("test edit did not change the spec")
			}
			out := strings.Join(check(t, base, next), "\n")
			if !strings.Contains(out, tt.want) {
				t.Errorf("want %q in:\n%s", tt.want, out)
			}
		})
	}
}

func TestAdditiveChangesPass(t *testing.T) {
	t.Parallel()
	base := spec(t)
	next := strings.Replace(base, "        tenant_id:\n          type: string\n          format: uuid\n",
		"        tenant_id:\n          type: string\n          format: uuid\n        display_name:\n          type: string\n", 1)
	next = strings.Replace(next, "        \"501\":", "        \"503\":\n          $ref: \"#/components/responses/Problem\"\n        \"501\":", 1)
	next = strings.Replace(next, "components:\n", "  /api/v1/ping:\n    get:\n      responses:\n        \"200\":\n          description: ok\ncomponents:\n", 1)
	if out := check(t, base, next); len(out) != 0 {
		t.Errorf("additive change reported: %v", out)
	}
}

func TestBadInputIsAnError(t *testing.T) {
	t.Parallel()
	if _, err := apidiff.Check([]byte("not: [openapi"), []byte(spec(t))); err == nil {
		t.Error("malformed base accepted")
	}
}
