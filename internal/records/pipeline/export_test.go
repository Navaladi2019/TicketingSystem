package pipeline

import "context"

// SetBeforeReturn installs the test-only hook that runs after every statement
// of a successful write and before Write returns (the kill test).
func SetBeforeReturn(p *Pipeline, f func(context.Context)) { p.beforeReturn = f }

// MergePatch exposes mergePatch to the unit tests.
func MergePatch(target, patch map[string]any) map[string]any { return mergePatch(target, patch) }

// CanonicalValue exposes canonicalValue to the unit tests.
func CanonicalValue(v any, column bool) (string, any) { return canonicalValue(v, column) }
