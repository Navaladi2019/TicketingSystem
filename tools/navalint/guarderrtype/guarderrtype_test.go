package guarderrtype_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Navaladi2019/TicketingSystem/tools/navalint/guarderrtype"
)

// featurePkg stands for feature code: workflows, activities, handlers.
const featurePkg = "github.com/Navaladi2019/TicketingSystem/internal/feature"

// TestOutsideGuard: references to the guard's failure types, and string
// constants equal to them, are reported outside internal/workerguard; test
// files are not (the `// want` comments in testdata/src/.../internal/feature).
func TestOutsideGuard(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), guarderrtype.Analyzer, featurePkg)
}

// TestGuardIsAllowed: internal/workerguard itself may use them; a package
// below it may not.
func TestGuardIsAllowed(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), guarderrtype.Analyzer,
		guarderrtype.GuardPkg, guarderrtype.GuardPkg+"/sub")
}
