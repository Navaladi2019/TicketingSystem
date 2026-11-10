package tenancyconn_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Navaladi2019/TicketingSystem/tools/navalint/tenancyconn"
)

// TestOutsideTenancy: .Conn() on a pgx.Tx, called or as a method value, on a
// savepoint or in a closure, is reported outside internal/tenancy; a
// pgxpool.Conn's Conn() and another type's are not, and neither is a
// _test.go file (the `// want` comments in testdata/src/a).
func TestOutsideTenancy(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), tenancyconn.Analyzer, "a")
}

// TestTenancyIsAllowed: internal/tenancy, which owns the transaction, may
// call it.
func TestTenancyIsAllowed(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), tenancyconn.Analyzer, tenancyconn.TenancyPkg)
}

// TestTenancyHarnessIsAllowed: so may the packages under it, such as the
// CI-2 isolation harness, in its non-test files too.
func TestTenancyHarnessIsAllowed(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), tenancyconn.Analyzer, tenancyconn.TenancyPkg+"/isolation")
}

// TestPrefixIsNotAllowed: the exemption is by package path, not by string
// prefix: internal/tenancyother is reported.
func TestPrefixIsNotAllowed(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), tenancyconn.Analyzer,
		"github.com/Navaladi2019/TicketingSystem/internal/tenancyother")
}
