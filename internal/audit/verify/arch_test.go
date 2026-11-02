package verify_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

const module = "github.com/Navaladi2019/TicketingSystem/"

// TestOfflineImports keeps the verifier offline (PLAN-005 D-5-6): neither
// cmd/nava nor internal/audit/verify reaches a database driver, an HTTP
// client or internal/db, and the only platform packages either links are
// internal/audit/chain and internal/audit/verify.
func TestOfflineImports(t *testing.T) {
	allowed := map[string]bool{
		module + "internal/audit/chain":  true,
		module + "internal/audit/verify": true,
		module + "cmd/nava":              true,
	}
	// database/sql/driver is allowed: github.com/google/uuid (via chain)
	// imports it for the Scanner and Valuer interfaces only; it holds no
	// client. database/sql itself, a driver or a socket client is denied.
	denied := []string{"net/http", "net/rpc", "net/smtp", "crypto/tls", "github.com/jackc/", "github.com/lib/pq", module + "internal/db"}
	for _, pkg := range []string{module + "cmd/nava", module + "internal/audit/verify"} {
		for _, dep := range goList(t, "{{.ImportPath}}", "-deps", pkg) {
			if dep == "database/sql" {
				t.Errorf("%s depends on database/sql", pkg)
			}
			for _, d := range denied {
				if strings.HasPrefix(dep, d) {
					t.Errorf("%s depends on %s (denied prefix %s)", pkg, dep, d)
				}
			}
			if strings.HasPrefix(dep, module) && !allowed[dep] {
				t.Errorf("%s depends on %s; it may link only internal/audit/chain and internal/audit/verify", pkg, dep)
			}
		}
	}
	direct := goList(t, `{{join .Imports "\n"}}`, module+"cmd/nava")
	for _, imp := range direct {
		if strings.HasPrefix(imp, module) && imp != module+"internal/audit/verify" && imp != module+"internal/audit/chain" {
			t.Errorf("cmd/nava imports %s", imp)
		}
	}
}

func goList(t *testing.T, format string, args ...string) []string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "go", append([]string{"list", "-f", format}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	return strings.Fields(string(out))
}
