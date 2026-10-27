package keycloak

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// Key-fetch limits (TICK-94 item 11).
const (
	// forceInterval: an unknown kid forces at most one JWKS fetch per issuer
	// in this window. Discovery and the first JWKS fetch count too.
	forceInterval = 30 * time.Second
	// minRefreshInterval is the floor of the background refresh.
	minRefreshInterval = 5 * time.Minute
	// fetchTimeout bounds one discovery or JWKS fetch, which also waits for
	// the cache's first fetch. The outbound client's own timeouts apply
	// inside it.
	fetchTimeout = 10 * time.Second
	// maxDiscoveryBytes bounds the discovery document read.
	maxDiscoveryBytes = 64 << 10
)

// Route templates of the outbound calls: all of the path a span name or an
// error carries (telemetry.WithRoute).
const (
	discoveryRoute = "/realms/{realm}/.well-known/openid-configuration"
	jwksRoute      = "/realms/{realm}/protocol/openid-connect/certs"
)

var (
	errKeysUnavailable = errors.New("keys unavailable")
	errUnknownKID      = errors.New("unknown kid")
)

// routedClient sends every request through the shared outbound client with a
// fixed route template, following no redirect. httprc builds its requests on
// its own context, so neither marker can come from the caller.
type routedClient struct {
	out   *telemetry.OutboundClient
	route string
}

func (c routedClient) Do(req *http.Request) (*http.Response, error) {
	return c.out.Do(req.WithContext(fetchContext(req.Context(), c.route)))
}

// fetchContext marks a discovery or JWKS fetch: route template for spans and
// errors, and no redirects, so the checked URL is the only one fetched.
func fetchContext(ctx context.Context, route string) context.Context {
	return telemetry.WithoutRedirects(telemetry.WithRoute(ctx, route))
}

// keySource holds the signing keys of each allowed issuer, each set fetched
// only from the jwks_uri of that issuer's own discovery document.
type keySource struct {
	out       *telemetry.OutboundClient
	cache     *jwk.Cache
	whitelist httprc.MapWhitelist
	issuers   map[string]*issuerKeys
	now       func() time.Time
	log       *slog.Logger
	fetches   metric.Int64Counter
	// fetchBase, when set, is the scheme and host discovery and JWKS are
	// fetched from instead of the issuer's own (back-channel). The checks
	// still run against the issuer.
	fetchBase *url.URL
}

// issuerKeys is one issuer's key set. gate serialises discovery and forced
// refreshes, so concurrent unknown-kid tokens share one fetch.
type issuerKeys struct {
	issuer string
	gate   chan struct{}
	set    atomic.Pointer[jwk.CachedSet]

	// Guarded by gate.
	jwksURL   string
	lastFetch time.Time
}

func newKeySource(ctx context.Context, out *telemetry.OutboundClient, issuers []string, fetchBase string, now func() time.Time, log *slog.Logger, mp metric.MeterProvider) (*keySource, error) {
	wl := httprc.NewMapWhitelist()
	jwksClient := routedClient{out: out, route: jwksRoute}
	cache, err := jwk.NewCache(ctx, httprc.NewClient(httprc.WithHTTPClient(jwksClient), httprc.WithWhitelist(wl)))
	if err != nil {
		return nil, fmt.Errorf("keycloak: start key cache: %w", err)
	}
	fetches, err := mp.Meter("nava-core/identity").Int64Counter("nava.identity.jwks.fetch",
		metric.WithDescription("Discovery and JWKS fetches by trigger and result; result=throttled is a fetch the 30 s gate skipped."))
	if err != nil {
		return nil, fmt.Errorf("keycloak: jwks fetch counter: %w", err)
	}
	s := &keySource{out: out, cache: cache, whitelist: wl, issuers: map[string]*issuerKeys{}, now: now, log: log, fetches: fetches}
	if fetchBase != "" {
		b, err := url.Parse(fetchBase)
		if err != nil || (b.Scheme != "http" && b.Scheme != "https") || b.Host == "" || b.User != nil || (b.Path != "" && b.Path != "/") || b.RawQuery != "" || b.Fragment != "" {
			return nil, errors.New("keycloak: invalid fetch base URL")
		}
		s.fetchBase = b
	}
	for _, iss := range issuers {
		s.issuers[iss] = &issuerKeys{issuer: iss, gate: make(chan struct{}, 1)}
	}
	return s, nil
}

func (s *keySource) close(ctx context.Context) error { return s.cache.Shutdown(ctx) }

// key returns the key with kid from iss's set. An unknown kid forces one
// refresh, unless the issuer was fetched within forceInterval, and then looks
// once more. iss must be on the allow-list.
func (s *keySource) key(ctx context.Context, iss, kid string) (jwk.Key, error) {
	ik, ok := s.issuers[iss]
	if !ok {
		return nil, errKeysUnavailable // unreachable: the verifier checks iss first
	}
	set := ik.set.Load()
	if set == nil {
		if err := s.fetch(ctx, ik, "initial"); err != nil {
			return nil, err
		}
		if set = ik.set.Load(); set == nil {
			return nil, errKeysUnavailable
		}
	} else if k, found := (*set).LookupKeyID(kid); found {
		return k, nil
	} else if err := s.fetch(ctx, ik, "unknown_kid"); err != nil && !errors.Is(err, errKeysUnavailable) {
		return nil, err
	}
	if k, found := (*ik.set.Load()).LookupKeyID(kid); found {
		return k, nil
	}
	return nil, errUnknownKID
}

// fetch discovers and registers the issuer's JWKS on first use, or forces a
// refresh. It returns at once, fetching nothing, when the issuer was fetched
// within forceInterval: a caller that waited on the gate then sees the keys
// the fetch it waited for brought.
func (s *keySource) fetch(ctx context.Context, ik *issuerKeys, trigger string) error {
	select {
	case ik.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-ik.gate }()

	if !ik.lastFetch.IsZero() && s.now().Sub(ik.lastFetch) < forceInterval {
		s.count(ctx, trigger, "throttled")
		if ik.set.Load() == nil {
			return errKeysUnavailable
		}
		return nil
	}
	ik.lastFetch = s.now()

	fctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	var err error
	if ik.set.Load() == nil {
		err = s.register(fctx, ik)
	} else {
		_, err = s.cache.Refresh(fctx, ik.jwksURL)
	}
	if err != nil {
		s.count(ctx, trigger, "error")
		// The cause can quote the JWKS URL and the response; the issuer is
		// from the allow-list, so it is safe to log.
		s.log.WarnContext(ctx, "identity: jwks fetch failed",
			slog.String("issuer", ik.issuer), slog.String("trigger", trigger), slog.String("error_type", fmt.Sprintf("%T", err)))
		return errKeysUnavailable
	}
	s.count(ctx, trigger, "ok")
	return nil
}

func (s *keySource) count(ctx context.Context, trigger, result string) {
	s.fetches.Add(ctx, 1, metric.WithAttributes(attribute.String("trigger", trigger), attribute.String("result", result)))
}

// register reads the issuer's discovery document, checks it, and adds its
// jwks_uri to the cache, waiting for the first fetch. Guarded by ik.gate.
func (s *keySource) register(ctx context.Context, ik *issuerKeys) error {
	if ik.jwksURL == "" {
		u, err := s.discover(ctx, ik.issuer)
		if err != nil {
			return err
		}
		ik.jwksURL = u
		s.whitelist.Add(u)
	}
	if s.cache.IsRegistered(ctx, ik.jwksURL) {
		// An earlier first fetch failed after registering.
		if _, err := s.cache.Refresh(ctx, ik.jwksURL); err != nil {
			return err
		}
	} else if err := s.cache.Register(ctx, ik.jwksURL,
		jwk.WithMinInterval(minRefreshInterval),
		jwk.WithHTTPClient(routedClient{out: s.out, route: jwksRoute})); err != nil {
		return err
	}
	set, err := s.cache.CachedSet(ik.jwksURL)
	if err != nil {
		return err
	}
	ik.set.Store(&set)
	return nil
}

// discover returns the jwks_uri of iss's discovery document. The document's
// issuer must equal iss exactly and the jwks_uri must lie under iss, so a
// document can never point key fetches at another host or realm.
func (s *keySource) discover(ctx context.Context, iss string) (string, error) {
	req, err := http.NewRequestWithContext(fetchContext(ctx, discoveryRoute), http.MethodGet, s.reach(iss, iss+"/.well-known/openid-configuration"), nil)
	if err != nil {
		return "", err
	}
	resp, err := s.out.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("discovery: status %d", resp.StatusCode)
	}
	var doc struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxDiscoveryBytes)).Decode(&doc); err != nil {
		return "", fmt.Errorf("discovery: decode: %w", err)
	}
	if doc.Issuer != iss {
		return "", errors.New("discovery: issuer mismatch")
	}
	jwks := doc.JWKSURI
	// With a fetch base, a back-channel discovery document names jwks_uri on
	// that base (Keycloak's dynamic back-channel hostname). Check it as the
	// same path under the issuer.
	if s.fetchBase != nil {
		if origin := s.fetchBase.Scheme + "://" + s.fetchBase.Host; strings.HasPrefix(jwks, origin+"/") {
			if is, err := url.Parse(iss); err == nil {
				jwks = is.Scheme + "://" + is.Host + strings.TrimPrefix(jwks, origin)
			}
		}
	}
	if !jwksUnderIssuer(iss, jwks) {
		return "", errors.New("discovery: jwks_uri outside the issuer")
	}
	return s.reach(iss, jwks), nil
}

// jwksUnderIssuer reports whether jwksURI names a resource under iss, checked
// both as text and as the URL the client will request: same scheme and host,
// no userinfo, query or fragment, no escaped path (RawPath set: %2e%2e,
// %2f), and a cleaned path still below the issuer's. The path after iss may
// hold no % at all and no //: a proxy that decodes twice (%252e) or merges
// slashes would otherwise see a path Go never sent.
func jwksUnderIssuer(iss, jwksURI string) bool {
	if !strings.HasPrefix(jwksURI, iss+"/") || strings.ContainsAny(jwksURI, "?#@\\") || strings.Contains(jwksURI, "..") {
		return false
	}
	if rest := jwksURI[len(iss):]; strings.Contains(rest, "%") || strings.Contains(rest, "//") {
		return false
	}
	is, err := url.Parse(iss)
	if err != nil {
		return false
	}
	u, err := url.Parse(jwksURI)
	if err != nil {
		return false
	}
	if u.Scheme != is.Scheme || u.Host != is.Host || u.User != nil || u.Opaque != "" ||
		u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return false
	}
	return strings.HasPrefix(path.Clean(u.Path), path.Clean(is.Path)+"/")
}

// reach is the URL to request for u, a URL under iss: u itself, or with the
// fetch base's scheme and host when one is set.
func (s *keySource) reach(iss, u string) string {
	if s.fetchBase == nil {
		return u
	}
	is, err := url.Parse(iss)
	if err != nil {
		return u
	}
	return s.fetchBase.Scheme + "://" + s.fetchBase.Host + strings.TrimPrefix(u, is.Scheme+"://"+is.Host)
}
