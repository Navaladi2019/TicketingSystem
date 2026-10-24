package idpclaims_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Navaladi2019/TicketingSystem/tools/navalint/idpclaims"
)

// TestOutsideAdapter: claim reads, json tags and admin API paths outside the
// adapter are reported (the `// want` comments in testdata/src/a).
func TestOutsideAdapter(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), idpclaims.Analyzer, "a")
}

// TestAdapterIsAllowed: internal/identity/keycloak and the packages below it
// may use them.
func TestAdapterIsAllowed(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), idpclaims.Analyzer,
		idpclaims.AdapterPkg, idpclaims.AdapterPkg+"/admin")
}

// TestStackTestsAreAllowed: the _test.go files of internal/identitystack may
// use them; its non-test files may not (the `// want` in testdata).
func TestStackTestsAreAllowed(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), idpclaims.Analyzer, idpclaims.StackTestPkg)
}
