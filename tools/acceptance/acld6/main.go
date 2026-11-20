// Command acld6 is the S30 acceptance kit's D6 re-check (ADR-010 D6, PLAN-011
// P11-12, TICK-944) for every stored access document, format 1 or 2. It reads
// the newest acl_versions row of each (tenant, entity type), one per line as
// psql -At -F '|' prints
//
//	tenant_id|entity_key|version|rules
//
// decodes the rules with authz.Decode (format 1 is converted, as on every
// read) and counts each document's worst-case read branches with
// authz.ReadBranchBound, the counter of TICK-938 P11-06. It prints one line
// per document and a summary, and exits 1 when any document is above the
// SP-2 threshold or does not decode. For format 1 the bound equals the D6 SQL
// in docs/acceptance/S30/d6.sql.
//
// Usage (docs/acceptance/S30/run.sh does this):
//
//	psql -At -F '|' -f docs/acceptance/S30/d6-docs.sql | go run ./tools/acceptance/acld6
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

// spTwoThreshold is authz's maxReadBranches (internal/authz/acl.go, SP-2,
// docs/data/acl-format2-caps.md §3). It is unexported there; keep the two
// equal.
const spTwoThreshold = 16

func main() {
	threshold := flag.Int("threshold", spTwoThreshold, "the SP-2 total-branch threshold")
	flag.Parse()
	os.Exit(run(os.Stdin, os.Stdout, *threshold))
}

func run(in io.Reader, out io.Writer, threshold int) int {
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<24)
	tenants := map[string]bool{}
	docs, format2, worst, bad := 0, 0, 0, 0
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			fmt.Fprintf(out, "unreadable line: %q\n", line)
			bad++
			continue
		}
		tenant, key, version, rules := parts[0], parts[1], parts[2], parts[3]
		format := 1
		if strings.HasPrefix(strings.TrimSpace(rules), "{") {
			format = authz.FormatV2
			format2++
		}
		d, err := authz.Decode([]byte(rules))
		if err != nil {
			fmt.Fprintf(out, "%s %s v%s format %d: does not decode: %v\n", tenant, key, version, format, err)
			bad++
			continue
		}
		n := authz.ReadBranchBound(d)
		docs++
		tenants[tenant] = true
		worst = max(worst, n)
		fmt.Fprintf(out, "%s %-10s v%-3s format %d  read branches %d\n", tenant, key, version, format, n)
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintf(out, "read: %v\n", err)
		return 1
	}
	verdict := "PASS"
	if worst > threshold || bad > 0 {
		verdict = "FAIL"
	}
	fmt.Fprintf(out, "D6: %d documents (%d format 2) in %d tenants, max %d read branches, threshold %d: %s\n",
		docs, format2, len(tenants), worst, threshold, verdict)
	if verdict == "FAIL" {
		return 1
	}
	return 0
}
