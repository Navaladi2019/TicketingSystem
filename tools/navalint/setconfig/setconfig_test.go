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

// TestPolicyTestsAreAllowed: the _test.go files of internal/db/dbtest may use
// them; its non-test files may not (the `// want` in testdata).
func TestPolicyTestsAreAllowed(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), setconfig.Analyzer,
		setconfig.PolicyTestPkg)
}

// TestIsolationTestsAreAllowed: the _test.go files of the CI-2 isolation
// suite may use them; its registry, a non-test file, may not.
func TestIsolationTestsAreAllowed(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), setconfig.Analyzer,
		setconfig.IsolationTestPkg)
}

// TestOtherDbtestIsReported: a _test.go file in another package named dbtest
// is reported; the exemption is by exact package path.
func TestOtherDbtestIsReported(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), setconfig.Analyzer, "other/dbtest")
}

// TestSQLPolicyIsNotChecked: a row-level security policy in a .sql file,
// embedded by the package as db/ does, reads current_setting('app.tenant_id')
// and is not reported. Only Go source is.
func TestSQLPolicyIsNotChecked(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), setconfig.Analyzer, "db")
}

// TestCatalogMayRead: internal/db/catalog and its tests may read app.tenant_id
// with current_setting, as the catalogue gate's predicate does (TICK-149);
// set_config and SET of app.tenant_id there are still reported.
func TestCatalogMayRead(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), setconfig.Analyzer,
		setconfig.CatalogPkg)
}

// TestOtherPackageMayNotRead: the same predicate in another package is
// reported, and so is SET LOCAL; the read exemption is by exact package path.
func TestOtherPackageMayNotRead(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), setconfig.Analyzer, "b")
}
