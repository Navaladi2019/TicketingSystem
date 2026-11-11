package erasetest_test

import (
	"context"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

const (
	module   = "github.com/Navaladi2019/TicketingSystem/"
	helper   = module + "internal/audit/erasetest"
	cmdTree  = "./cmd/..."
	repoRoot = "../../.."
)

// TestNoBinaryLinksTheEraseHelper (TICK-449 grooming, D-5-10): the erase
// helper is test-only. No cmd/* package, the production binaries, may depend
// on it, directly or through another package.
//
// Control: the same scan, over a test binary that does use the helper (the
// integration tests of internal/server), finds it, so a scan that cannot see
// the import cannot pass.
func TestNoBinaryLinksTheEraseHelper(t *testing.T) {
	for _, line := range goList(t, `{{.ImportPath}}: {{join .Deps " "}}`, cmdTree) {
		pkg, deps, _ := strings.Cut(line, ": ")
		if slices.Contains(strings.Fields(deps), helper) {
			t.Errorf("%s depends on %s, a test-only helper (PLAN-005 D-5-10)", pkg, helper)
		}
	}
	var seen bool
	for _, dep := range goList(t, `{{join .Deps "\n"}}`, "-tags", "integration", "-test", "./internal/server") {
		seen = seen || dep == helper
	}
	if !seen {
		t.Fatal("control: the scan does not see the helper in internal/server's integration test binary; it proves nothing")
	}
}

func goList(t *testing.T, format string, args ...string) []string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "go", append([]string{"list", "-f", format}, args...)...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list %v: %v", args, err)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}
