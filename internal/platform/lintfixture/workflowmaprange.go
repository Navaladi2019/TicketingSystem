//go:build lintfixture

package lintfixture

import "go.temporal.io/sdk/workflow"

// MapOrderWorkflow ranges over a map in workflow code, so its commands could
// come out in another order on replay. workflowcheck must report it (D-3-23).
func MapOrderWorkflow(ctx workflow.Context, steps map[string]int) (int, error) {
	total := 0
	for _, n := range steps {
		total = total*31 + n
	}
	return total, nil
}
