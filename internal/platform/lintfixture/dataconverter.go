//go:build lintfixture

package lintfixture

import (
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

// OwnConverter gives workflow code a converter of its own, so the history
// records an encoding the worker guard never checked. forbidigo must reject it
// (TICK-245).
func OwnConverter(ctx workflow.Context, dc converter.DataConverter) workflow.Context {
	return workflow.WithDataConverter(ctx, dc)
}
