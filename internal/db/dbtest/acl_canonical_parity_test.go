//go:build integration

package dbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/acl"
)

// TICK-939 (ADR-010-A1 A1-2d; PLAN-011 D-11-12): acl.CanonicalSize(d) ==
// octet_length(d::jsonb::text), and the text itself is equal, over the
// fixtures, the A1 edge cases, fuzzed documents and the bench shapes, against
// real PostgreSQL with TICK-940's migrations applied.
func TestCanonicalSizeMatchesJSONBText(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)

	var corpus []string
	for _, name := range []string{"format2_valid.json", "format2_invalid.json"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "authz", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		var rows []struct {
			Doc json.RawMessage `json:"doc"`
		}
		if err := json.Unmarshal(b, &rows); err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			corpus = append(corpus, string(r.Doc))
		}
	}
	format1, err := filepath.Glob(filepath.Join("..", "..", "authz", "testdata", "format1", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range format1 {
		if strings.HasSuffix(p, ".golden.json") {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		corpus = append(corpus, string(b))
	}

	// The A1-2d edge cases.
	corpus = append(corpus,
		`"  "`, "\"  \"", `"<>&"`, `"<>&"`, `"a/b"`, `"a\/b"`,
		`"\b\f\n\r\t"`, `"\u0001\u0002\u001f\u007f"`, `"é日本語😀"`, `"😀"`, `"\"\\"`,
		`{"a\u0001":1,"é":2,"":3,"ab":4,"b":5,"aa":6}`,
		`[2, 2.0, 2.00, 1e2, 1E+2, 25e-1, 1e-3, 0.50, -0, -0.0, -12.50, 1.5e3, 12.345e-1, 0e5, 0.0e-3, 100, 1e0, 123456789012345678901234567890.123456789]`,
		`{"k":{"k":{"k":[[],{}]}}}`, `{"a":1,"a":2}`, `true`, `null`, `[]`, `{}`)

	// Fuzzed documents, deterministic.
	r := rand.New(rand.NewSource(939))
	for range 400 {
		corpus = append(corpus, fuzzJSON(r, 5))
	}
	// Bench shapes: a wide condition list, long names, many ids.
	var sb strings.Builder
	sb.WriteString(`{"format":2,"conditions":[`)
	for i := range 300 {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"name":"c%d","expr":{"member_of":{"groups":["%08x-0000-4000-8000-000000000000"]}}}`, i, i)
	}
	sb.WriteString(`],"policies":[]}`)
	corpus = append(corpus, sb.String())

	checked := 0
	for _, in := range corpus {
		got, err := acl.CanonicalJSON([]byte(in))
		if err != nil {
			continue // refused before measuring; jsonb refuses it too (below)
		}
		var text string
		var size int
		if err := super.QueryRow(context.Background(), `SELECT $1::text::jsonb::text, octet_length($1::text::jsonb::text)`, in).Scan(&text, &size); err != nil {
			t.Errorf("PostgreSQL refuses what CanonicalJSON accepts: %.80s: %v", in, err)
			continue
		}
		checked++
		if text != string(got) || size != len(got) {
			t.Errorf("CanonicalJSON(%.120s)\n  = %q (%d)\n  jsonb  %q (%d)", in, got, len(got), text, size)
		}
		if n, _ := acl.CanonicalSize([]byte(in)); n != size {
			t.Errorf("CanonicalSize = %d, octet_length = %d for %.80s", n, size, in)
		}
	}
	if checked < 300 {
		t.Fatalf("only %d documents compared", checked)
	}

	// What CanonicalJSON refuses, jsonb refuses too (or would store something
	// else): NUL and lone surrogate escapes.
	for _, in := range []string{`"a\u0000b"`, `"\ud800"`, `"\udc00"`, `"\ud800a"`} {
		if _, err := acl.CanonicalJSON([]byte(in)); err == nil {
			t.Errorf("CanonicalJSON accepted %s", in)
		}
		var s string
		if err := super.QueryRow(context.Background(), `SELECT $1::text::jsonb::text`, in).Scan(&s); err == nil {
			t.Errorf("PostgreSQL accepted %s as %s", in, s)
		}
	}
}

func fuzzJSON(r *rand.Rand, depth int) string {
	atoms := []string{`null`, `true`, `false`, `0`, `-1`, `1.50`, `1e2`, `7e-2`, `"x"`, `"a/b<>&"`, `" "`, `"é\n"`, `"\u0001"`, `""`, `"日本"`, `"\\\""`}
	if depth == 0 || r.Intn(3) == 0 {
		return atoms[r.Intn(len(atoms))]
	}
	n := r.Intn(5)
	parts := make([]string, 0, n)
	if r.Intn(2) == 0 {
		for range n {
			parts = append(parts, fuzzJSON(r, depth-1))
		}
		return "[" + strings.Join(parts, ",") + "]"
	}
	keys := []string{"a", "b", "ab", "name", "é", "a/b", "when", "", "zz", "a\n"}
	for range n {
		parts = append(parts, `"`+keys[r.Intn(len(keys))]+`":`+fuzzJSON(r, depth-1))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
