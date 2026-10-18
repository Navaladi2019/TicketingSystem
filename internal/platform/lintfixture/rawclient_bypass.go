//go:build lintfixture

package lintfixture

import (
	"context"
	nethttp "net/http"
)

// Aliased imports a raw client under another package name. A text-only
// forbidigo pattern (^http\.Client$) does not see it.
func Aliased() *nethttp.Client {
	return new(nethttp.Client)
}

// RawTransport skips http.Client altogether: no client timeout, no URL
// redaction, no outbound span.
func RawTransport(ctx context.Context) (*nethttp.Response, error) {
	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, "http://idp.invalid/", nil)
	if err != nil {
		return nil, err
	}
	return (&nethttp.Transport{}).RoundTrip(req)
}
