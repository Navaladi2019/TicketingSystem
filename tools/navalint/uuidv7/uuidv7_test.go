package uuidv7_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Navaladi2019/TicketingSystem/tools/navalint/uuidv7"
)

const mod = "github.com/Navaladi2019/TicketingSystem/"

// TestReportsV4 is the positive case: every v4 call under internal/ is
// reported, called or taken as a value; NewV7, NewSHA1 and MustParse are not,
// and a _test.go file in the same package is not.
func TestReportsV4(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), uuidv7.Analyzer, mod+"internal/svc")
}

// TestReportsCmd: cmd/ is in scope.
func TestReportsCmd(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), uuidv7.Analyzer, mod+"cmd/tool")
}

// TestExemptPaths is the negative case: test-helper packages and packages
// outside internal/ and cmd/ are not reported.
func TestExemptPaths(t *testing.T) {
	for _, p := range []string{"internal/tenancy/isolation", "internal/identity/contracttest", "pkg/other"} {
		analysistest.Run(t, analysistest.TestData(), uuidv7.Analyzer, mod+p)
	}
}
