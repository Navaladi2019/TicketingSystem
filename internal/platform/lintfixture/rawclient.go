//go:build lintfixture

// Package lintfixture holds code the forbidigo rule must reject. It builds only
// with the lintfixture tag; `task lint:fixture` fails if the linter lets it pass.
package lintfixture

import "net/http"

// Raw uses a raw client outside internal/platform/telemetry.
func Raw(c *http.Client) *http.Client {
	_ = http.DefaultClient
	return c
}
