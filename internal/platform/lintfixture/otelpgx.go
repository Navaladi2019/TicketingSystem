//go:build lintfixture

package lintfixture

import "github.com/exaring/otelpgx"

// QueryParameters records every bind parameter, tenant data included, as a
// span attribute. forbidigo must reject it (TICK-94 item 4).
func QueryParameters() otelpgx.Option {
	return otelpgx.WithIncludeQueryParameters()
}
