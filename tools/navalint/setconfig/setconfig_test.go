package setconfig_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Navaladi2019/TicketingSystem/tools/navalint/setconfig"
)

// TestOutsideTenancy: every way a Go file outside internal/tenancy spells the
// needles in a constant is reported (the `// want` comments in testdata/src/a).
func TestOutsideTenancy(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), setconfig.Analyzer, "a")
}

// TestTenancyIsAllowed: internal/tenancy and its external test package may
// use them.
func TestTenancyIsAllowed(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), setconfig.Analyzer,
		setconfig.TenancyPkg)
}

// TestSQLPolicyIsNotChecked: a row-level security policy in a .sql file,
// embedded by the package as db/ does, reads current_setting('app.tenant_id')
// and is not reported. Only Go source is.
func TestSQLPolicyIsNotChecked(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), setconfig.Analyzer, "db")
}
