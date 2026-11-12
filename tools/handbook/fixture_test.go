package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSha = "abcdef0123456789abcdef0123456789abcdef01"

// write creates root/rel with content.
func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

const fixtureSpec = `openapi: 3.1.0
paths:
  /api/v1/me:
    get:
      operationId: getMe
      summary: The caller
      responses:
        "200": {description: ok}
  /api/v1/audit/export:
    post:
      operationId: exportAudit
      summary: Export | pipe
      responses:
        "202": {description: accepted}
        "401": {description: no}
  /api/v1/audit/entries:
    get:
      operationId: listAuditEntries
      summary: List audit entries
      responses:
        "200": {description: ok}
`

const fixtureADR1 = `---
type: adr
id: ADR-001
title: Platform blueprint
status: ACCEPTED (approved 2026-10-15)
date: 2026-10-15
---

## 1. Invariants

| # | Invariant | Source |
|---|---|---|
| **I1** | **Tenant wall in the database.** Every tenant table has RLS. | P2 |
| **I2** | **The tenant comes only from a verified source.** Never the client. | P2 |

## 2. Decisions

x

## Errata

### E1 (2026-10-15): command inspection

Ruled by the CTO.

### E2 (2026-10-16): another

Status: SUPERSEDED
`

const fixtureSvg = `<svg xmlns="http://www.w3.org/2000/svg"><!-- payload-start -->QUJD<!-- payload-end --><rect width="1" height="1"/></svg>`

// newFixture lays out a minimal repository the generator can read.
func newFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "api/openapi.bundled.yaml", fixtureSpec)
	write(t, root, "docs/adr/ADR-001-platform-blueprint.md", fixtureADR1)
	write(t, root, "docs/adr/ADR-004-audit-log.md", "---\nid: ADR-004\ntitle: \"Audit: one log\"\nstatus: ACCEPTED\ndate: 2026-11-02\n---\nbody\n")
	write(t, root, "docs/adr/ADR-004-E1-cascade.md", "---\nid: ADR-004-E1\ntitle: Cascade entries\nstatus: ACCEPTED\ndate: 2026-11-03\n---\nbody\n")
	write(t, root, "docs/adr/ADR-004-explained.md", "---\nid: ADR-004-explained\ntitle: Explained\n---\nno status, but excluded\n")
	write(t, root, "docs/adr/ADR-004-audit-log.html", "<html></html>")
	write(t, root, "docs/adr/ADR-004/rfc/RFC-008.md", "no front matter, in a subfolder")
	write(t, root, "docs/budgets.md", "# Budgets\n\n## Record history\n\n| Budget | Measured as | Limit | Owner |\n|---|---|---|---|\n| History length | counted | 500 | Principal |\n")
	write(t, root, "db/migrations/00001_init.sql", "-- +goose Up\n-- First migration.\nCREATE TABLE audit_log (id int);\n")
	write(t, root, "internal/audit/audit_test.go", "package audit\n\nimport \"testing\"\n\nfunc TestChainSeals(t *testing.T) {}\n")
	write(t, root, "internal/audit/audit.go", "package audit\n")
	write(t, root, "docs/evidence/e.md", "evidence")
	write(t, root, "docs/handbook/assets/mermaid.min.js", "/* pinned */")
	write(t, root, "docs/handbook/src/diagrams/ok.excalidraw.svg", fixtureSvg)
	write(t, root, "docs/handbook/src/stories/stories.yaml", "stories:\n  - id: S14\n    issue: TICK-409\n    title: Audit log\n    status: delivered\n    evidence: [docs/evidence/e.md]\n")
	write(t, root, "docs/handbook/src/database/schema.json", `{"format":1,"postgres":"16","migration":"1","views":[],"tables":[
 {"schema":"public","name":"audit_log","kind":"partitioned","rls":{"enabled":true,"forced":true},
  "columns":[{"name":"id","type":"uuid","not_null":true,"default":"gen_random_uuid()"},{"name":"tenant_id","type":"uuid","not_null":true}],
  "constraints":[{"name":"audit_log_pkey","type":"primary_key","columns":["tenant_id","id"]},
                 {"name":"audit_log_tenant_fk","type":"foreign_key","columns":["tenant_id"],"references":{"table":"tenants","columns":["id"]}}],
  "indexes":[{"name":"audit_log_pk","method":"btree","unique":true,"keys":["tenant_id","id"]}],
  "policies":[{"name":"tenant","command":"ALL","roles":["nava_app"],"permissive":true,"using":"(tenant_id = x)","with_check":"(tenant_id = y)"}]},
 {"schema":"public","name":"tenants","kind":"table","columns":[{"name":"id","type":"uuid","not_null":true}],"constraints":[{"name":"tenants_pkey","type":"primary_key","columns":["id"]}]},
 {"schema":"archive","name":"old_log","kind":"table","columns":[{"name":"id","type":"uuid"}]}]}`)
	write(t, root, "docs/handbook/src/database/areas.yaml", "audit: [audit_log, tenants]\n")
	page(t, root, "index.md", "Home", "home", "delivered", "", "Welcome.\n")
	return root
}

// page writes a source page; section "home" is src/index.md.
func page(t *testing.T, root, rel, title, section, status, extraFront, body string) {
	t.Helper()
	front := "---\ntitle: \"" + title + "\"\nsection: " + section + "\nstatus: " + status + "\n" + extraFront + "---\n\n"
	write(t, root, "docs/handbook/src/"+rel, front+body)
}

func buildFixture(t *testing.T, root string) Output {
	t.Helper()
	out, err := Build(Config{Root: root, Sha: testSha, Date: "2026-11-11"})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	return out
}

func mustFail(t *testing.T, root string, wants ...string) {
	t.Helper()
	_, err := Build(Config{Root: root, Sha: testSha, Date: "2026-11-11"})
	if err == nil {
		t.Fatalf("build succeeded, want a failure naming %q", wants)
	}
	for _, w := range wants {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q does not name %q", err.Error(), w)
		}
	}
}
