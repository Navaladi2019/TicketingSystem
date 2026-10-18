// Command openapi-compat fails when api/openapi.yaml removes or narrows
// anything that api/openapi.yaml on the base branch offered: the additive-only
// rule for /api/v1.
//
// Usage: openapi-compat <base-spec> <new-spec>
package main

import (
	"fmt"
	"os"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/apidiff"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run returns the exit code: 0 additive, 1 breaking, 2 usage or unreadable input.
func run(args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: openapi-compat <base-spec> <new-spec>")
		return 2
	}
	base, err := os.ReadFile(args[0]) //nolint:gosec // a CLI reads the paths its caller names
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	next, err := os.ReadFile(args[1]) //nolint:gosec // a CLI reads the paths its caller names
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	problems, err := apidiff.Check(base, next)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "breaking:", p)
	}
	if len(problems) > 0 {
		return 1
	}
	return 0
}
