package metadata_test

import (
	"context"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// TestNoDatabaseOrNetworkImports keeps the package pure (P3-03): its own
// imports are an allow-list, and nothing in its dependency graph reaches a
// database driver, an HTTP or socket client, a process runner, or another
// platform package. The jsonschema library pulls in os, net/url and
// net/netip for parsing only; the deny-all loader keeps os unused for loads.
func TestNoDatabaseOrNetworkImports(t *testing.T) {
	direct := goList(t, "{{join .Imports \"\\n\"}}", ".")
	allowed := []string{
		"bytes", "cmp", "embed", "encoding/json", "errors", "fmt", "io", "io/fs", "maps",
		"path", "reflect", "regexp", "slices", "strconv", "strings",
		"github.com/santhosh-tekuri/jsonschema/v6", "github.com/santhosh-tekuri/jsonschema/v6/kind",
	}
	for _, imp := range direct {
		if !slices.Contains(allowed, imp) {
			t.Errorf("internal/metadata imports %q, which is not on the allow-list", imp)
		}
	}

	deps := goList(t, "{{.ImportPath}}", "-deps", ".")
	denied := []string{
		"database/", "net/http", "net/rpc", "net/smtp", "crypto/tls", "os/exec",
		"github.com/jackc/", "github.com/lib/pq", "google.golang.org/grpc", "go.temporal.io/",
		"github.com/Navaladi2019/TicketingSystem/",
	}
	for _, dep := range deps {
		if dep == "net" {
			t.Errorf("dependency graph contains package net (sockets)")
		}
		if dep == "github.com/Navaladi2019/TicketingSystem/internal/metadata" {
			continue
		}
		for _, d := range denied {
			if strings.HasPrefix(dep, d) {
				t.Errorf("dependency graph contains %q (denied prefix %q)", dep, d)
			}
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
