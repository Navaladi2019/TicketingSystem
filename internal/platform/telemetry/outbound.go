package telemetry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Default outbound timeouts. A zero field in OutboundOptions selects them.
const (
	defaultOutboundTimeout = 10 * time.Second
	defaultOutboundDial    = 3 * time.Second
	defaultOutboundTLS     = 3 * time.Second
	outboundIdleConns      = 10
)

// OutboundOptions tunes NewOutboundClient. The zero value is valid.
type OutboundOptions struct {
	// Timeout bounds one whole request, redirects and body read included
	// (http.Client.Timeout). Default 10s.
	Timeout time.Duration
	// DialTimeout bounds connecting. Default 3s.
	DialTimeout time.Duration
	// TLSHandshakeTimeout bounds the TLS handshake. Default 3s.
	TLSHandshakeTimeout time.Duration
	// TLSConfig is the client TLS configuration (CA pool, client certificate).
	// Nil uses the system roots.
	TLSConfig *tls.Config
}

type routeKey struct{}

// WithRoute returns ctx marked with the route template of the outbound call,
// such as "/realms/{realm}/protocol/openid-connect/token". The template is
// all the path an error or span name will carry: real paths hold ids.
func WithRoute(ctx context.Context, template string) context.Context {
	return context.WithValue(ctx, routeKey{}, template)
}

func routeFrom(ctx context.Context) string {
	s, _ := ctx.Value(routeKey{}).(string)
	return s
}

type untracedKey struct{}

// WithoutTrace returns ctx marked so that an outbound call made with it is
// neither traced nor measured, nor carries trace headers. It is for periodic
// background calls such as dependency health probes: traced, each would be
// a root trace of its own every second, and every failing one would be kept
// by the Collector's tail sampler.
func WithoutTrace(ctx context.Context) context.Context {
	return context.WithValue(ctx, untracedKey{}, true)
}

func traced(r *http.Request) bool {
	off, _ := r.Context().Value(untracedKey{}).(bool)
	return !off
}

// OutboundClient is the one HTTP client for calls leaving the process
// (Keycloak, webhooks, LLM providers). Build it with NewOutboundClient; do not
// make an http.Client of your own.
//
// It traces and measures each call with the explicit providers, sets timeouts
// at every layer, and guarantees that no error it returns carries a query
// string, userinfo or real path: those hold emails, ids and tokens, and
// callers log errors.
type OutboundClient struct {
	hc *http.Client
}

// NewOutboundClient builds the client. It reports an error for a negative
// timeout.
func NewOutboundClient(tel *Telemetry, opts OutboundOptions) (*OutboundClient, error) {
	for _, d := range []time.Duration{opts.Timeout, opts.DialTimeout, opts.TLSHandshakeTimeout} {
		if d < 0 {
			return nil, errors.New("telemetry: outbound timeouts must not be negative")
		}
	}
	opts.Timeout = orDefault(opts.Timeout, defaultOutboundTimeout)
	opts.DialTimeout = orDefault(opts.DialTimeout, defaultOutboundDial)
	opts.TLSHandshakeTimeout = orDefault(opts.TLSHandshakeTimeout, defaultOutboundTLS)

	// A fresh transport, not http.DefaultTransport: no ambient proxy from the
	// environment and no sharing with code that changes the default.
	base := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: opts.DialTimeout, KeepAlive: 30 * time.Second}).DialContext,
		TLSClientConfig:       opts.TLSConfig,
		TLSHandshakeTimeout:   opts.TLSHandshakeTimeout,
		ResponseHeaderTimeout: opts.Timeout,
		ExpectContinueTimeout: time.Second,
		MaxIdleConns:          outboundIdleConns,
		MaxIdleConnsPerHost:   outboundIdleConns,
		IdleConnTimeout:       90 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	instrumented := otelhttp.NewTransport(base,
		otelhttp.WithFilter(traced),
		otelhttp.WithTracerProvider(tel.TracerProvider),
		otelhttp.WithMeterProvider(tel.MeterProvider),
		otelhttp.WithPropagators(tel.Propagator),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + routeFrom(r.Context())
		}),
	)
	return &OutboundClient{hc: &http.Client{Transport: redactingTransport{next: instrumented}, Timeout: opts.Timeout}}, nil
}

func orDefault(d, def time.Duration) time.Duration {
	if d == 0 {
		return def
	}
	return d
}

// Do sends req. Any error is a *url.Error whose URL is reduced to scheme, host
// and the route template from WithRoute (none: no path). http.Client wraps
// every transport error with the full request URL itself, so the reduction
// has to happen here, after it.
func (c *OutboundClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.hc.Do(req) //nolint:gosec // G704: the caller owns the URL; this client is the reviewed outbound path
	if err != nil {
		return nil, redactURLError(req.Context(), err)
	}
	return resp, nil
}

// redactingTransport reduces a *url.Error returned by the transport itself.
type redactingTransport struct{ next http.RoundTripper }

func (t redactingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return nil, redactURLError(req.Context(), err)
	}
	return resp, nil
}

// redactURLError rewrites the URL of a *url.Error and of any it wraps. The
// cause is kept only when its type is known to carry no request URL or remote
// text (net, context, TLS and x509 errors); anything else, such as an
// unparsable Location header that http.Client quotes verbatim, is replaced.
func redactURLError(ctx context.Context, err error) error {
	var ue *url.Error
	if !errors.As(err, &ue) {
		return err
	}
	return &url.Error{Op: ue.Op, URL: reduceURL(ue.URL, routeFrom(ctx)), Err: safeCause(ctx, ue.Err)}
}

var errOutboundFailed = errors.New("outbound: request failed")

func safeCause(ctx context.Context, err error) error {
	var (
		nested  *url.Error
		netErr  net.Error
		certErr *tls.CertificateVerificationError
		alert   tls.AlertError
		record  tls.RecordHeaderError
		unknown x509.UnknownAuthorityError
		host    x509.HostnameError
		invalid x509.CertificateInvalidError
	)
	switch {
	case errors.As(err, &nested):
		return redactURLError(ctx, err)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded),
		errors.As(err, &netErr), errors.As(err, &certErr), errors.As(err, &alert),
		errors.As(err, &record), errors.As(err, &unknown), errors.As(err, &host),
		errors.As(err, &invalid):
		return err
	}
	return errOutboundFailed
}

// reduceURL returns scheme://host plus route. A URL that does not parse
// yields only the route: nothing of it can be trusted.
func reduceURL(raw, route string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return route
	}
	return u.Scheme + "://" + u.Host + route
}
