// Package config loads nava-core configuration from the environment into a
// typed struct. There are no package-level settings: callers pass the Config
// they loaded to the constructors that need it.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Config is the full nava-core configuration.
type Config struct {
	HTTP      HTTP
	Log       Log
	Telemetry Telemetry
	Deps      Dependencies
}

// HTTP configures the two listeners. Both use the same timeouts.
type HTTP struct {
	Addr              string        // NAVA_HTTP_ADDR: the public API listener
	AdminAddr         string        // NAVA_ADMIN_ADDR: the admin listener (/metrics, /healthz, /readyz)
	ReadHeaderTimeout time.Duration // NAVA_HTTP_READ_HEADER_TIMEOUT
	ReadTimeout       time.Duration // NAVA_HTTP_READ_TIMEOUT
	WriteTimeout      time.Duration // NAVA_HTTP_WRITE_TIMEOUT
	IdleTimeout       time.Duration // NAVA_HTTP_IDLE_TIMEOUT
	ShutdownTimeout   time.Duration // NAVA_HTTP_SHUTDOWN_TIMEOUT
}

// Dependencies locates what the health checks probe. An empty value means
// "not configured": the check reports it, and readiness fails closed for
// PostgreSQL.
type Dependencies struct {
	// DatabaseURL is NAVA_DATABASE_URL, a postgres:// or postgresql:// URL. It
	// holds a password: it is never logged and never echoed in an error.
	DatabaseURL string
	// KeycloakManagementURL is NAVA_KEYCLOAK_MANAGEMENT_URL, the base URL of
	// Keycloak's management port, such as http://keycloak:9000.
	KeycloakManagementURL string
	// TemporalAddress is NAVA_TEMPORAL_ADDRESS, the frontend host:port.
	TemporalAddress string
	// TemporalTLS is the client certificate nava-core dials the frontend with.
	TemporalTLS TemporalTLS
	// OIDCIssuers is NAVA_OIDC_ISSUERS, the comma-separated exact iss
	// allow-list of tenant access tokens, such as
	// http://keycloak:8080/realms/nava. Empty: every API request is refused
	// with 401 (fail closed).
	OIDCIssuers []string
	// OIDCFetchBaseURL is NAVA_OIDC_FETCH_BASE_URL, an optional scheme://host[:port]
	// to fetch discovery and JWKS from instead of the issuer's host, such as
	// http://keycloak:8080 when the pinned issuer is the browser URL. It never
	// widens the iss allow-list.
	OIDCFetchBaseURL string
}

// TemporalTLS locates nava-core's mTLS identity for the Temporal frontend (the
// `nava-core` certificate). The key is read from a file, never from a variable.
// CertFile, KeyFile and CAFile are set together or not at all.
type TemporalTLS struct {
	CertFile   string // NAVA_TEMPORAL_TLS_CERT_FILE
	KeyFile    string // NAVA_TEMPORAL_TLS_KEY_FILE
	CAFile     string // NAVA_TEMPORAL_TLS_CA_FILE, the CA that signed the frontend certificate
	ServerName string // NAVA_TEMPORAL_TLS_SERVER_NAME, the name on the frontend certificate
}

// Enabled reports whether the three files are configured.
func (t TemporalTLS) Enabled() bool { return t.CertFile != "" }

// Log configures the structured logger.
type Log struct {
	Level  slog.Level // NAVA_LOG_LEVEL: debug, info, warn, error
	Format string     // NAVA_LOG_FORMAT: json or text
}

// Telemetry configures OpenTelemetry. config reads only what decides whether
// a trace exporter exists. Every other standard OTEL_* variable (service name,
// resource attributes, sampler, OTLP headers, timeout and TLS) is read by the
// OTel SDK and exporter themselves, as the OTel specification defines. The
// OTLP headers and timeout are still checked here first (see Load).
type Telemetry struct {
	// ExportTraces is true when OTEL_EXPORTER_OTLP_TRACES_ENDPOINT or
	// OTEL_EXPORTER_OTLP_ENDPOINT is set. Neither set means no trace exporter.
	ExportTraces bool
	// ExportLogs is true when OTEL_EXPORTER_OTLP_ENDPOINT is set. The signal
	// specific OTEL_EXPORTER_OTLP_LOGS_ENDPOINT alone does not turn it on, so
	// stdout stays the only log destination until the base endpoint is set.
	ExportLogs bool
	// ShutdownTimeout is the flush budget after the HTTP server has drained,
	// separate from it. NAVA_TELEMETRY_SHUTDOWN_TIMEOUT.
	ShutdownTimeout time.Duration
}

// CascadeDepthLimit is the deepest causation depth an outbox row may carry
// (ADR-001 I9, PLAN-001 D-P7). It is platform configuration, the same for
// every tenant and never read from the environment; PLAN-002 confirms or
// changes it.
const CascadeDepthLimit = 8

// Log formats.
const (
	FormatJSON = "json"
	FormatText = "text"
)

// LookupFunc reads one environment variable. os.LookupEnv satisfies it; tests
// pass a map-backed function instead of mutating the process environment.
type LookupFunc func(key string) (string, bool)

// Load reads the configuration through lookup, applying defaults for unset
// variables. It reports every invalid variable at once, by name only: values
// are never echoed, because a misplaced secret must not reach the logs.
func Load(lookup LookupFunc) (Config, error) {
	r := reader{lookup: lookup}
	tel := r.telemetry()
	cfg := Config{
		HTTP: HTTP{
			Addr:              r.addr("NAVA_HTTP_ADDR", ":8080"),
			AdminAddr:         r.addr("NAVA_ADMIN_ADDR", ":9090"),
			ReadHeaderTimeout: r.duration("NAVA_HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       r.duration("NAVA_HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      r.duration("NAVA_HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:       r.duration("NAVA_HTTP_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout:   r.duration("NAVA_HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		Log: Log{
			Level:  r.level("NAVA_LOG_LEVEL", slog.LevelInfo),
			Format: r.oneOf("NAVA_LOG_FORMAT", FormatJSON, FormatJSON, FormatText),
		},
		Telemetry: tel,
	}
	cfg.Deps = Dependencies{
		DatabaseURL:           r.databaseURL("NAVA_DATABASE_URL"),
		KeycloakManagementURL: strings.TrimRight(r.baseURL("NAVA_KEYCLOAK_MANAGEMENT_URL"), "/"),
		TemporalAddress:       r.hostPort("NAVA_TEMPORAL_ADDRESS"),
		OIDCIssuers:           r.urlList("NAVA_OIDC_ISSUERS"),
		OIDCFetchBaseURL:      strings.TrimRight(r.baseURL("NAVA_OIDC_FETCH_BASE_URL"), "/"),
	}
	cfg.Deps.TemporalTLS = TemporalTLS{
		CertFile:   r.str("NAVA_TEMPORAL_TLS_CERT_FILE"),
		KeyFile:    r.str("NAVA_TEMPORAL_TLS_KEY_FILE"),
		CAFile:     r.str("NAVA_TEMPORAL_TLS_CA_FILE"),
		ServerName: r.str("NAVA_TEMPORAL_TLS_SERVER_NAME"),
	}
	if t := cfg.Deps.TemporalTLS; (t.CertFile != "" || t.KeyFile != "" || t.CAFile != "") &&
		(t.CertFile == "" || t.KeyFile == "" || t.CAFile == "") {
		r.errs = append(r.errs, errors.New("NAVA_TEMPORAL_TLS_CERT_FILE, NAVA_TEMPORAL_TLS_KEY_FILE and NAVA_TEMPORAL_TLS_CA_FILE: set all three or none"))
	}
	if cfg.HTTP.Addr == cfg.HTTP.AdminAddr {
		r.errs = append(r.errs, errors.New("NAVA_ADMIN_ADDR: must differ from NAVA_HTTP_ADDR"))
	}
	if err := errors.Join(r.errs...); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

type reader struct {
	lookup LookupFunc
	errs   []error
}

// telemetry reads the OTEL_* switches and NAVA_TELEMETRY_SHUTDOWN_TIMEOUT, and
// checks the OTLP values the exporter would otherwise log with their content.
func (r *reader) telemetry() Telemetry {
	// Both endpoints are validated, so a bad value fails here, by name, before
	// the exporter would report it with the value.
	tracesEndpoint := r.endpoint("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	baseEndpoint := r.endpoint("OTEL_EXPORTER_OTLP_ENDPOINT")
	// The exporter parses these itself and logs a bad entry or timeout with
	// its value, which for headers is a bearer token or API key. Check them
	// here, by name, so the exporter only ever sees values it accepts.
	r.otlpHeaders("OTEL_EXPORTER_OTLP_HEADERS")
	r.otlpHeaders("OTEL_EXPORTER_OTLP_TRACES_HEADERS")
	r.otlpHeaders("OTEL_EXPORTER_OTLP_LOGS_HEADERS")
	r.otlpTimeout("OTEL_EXPORTER_OTLP_TIMEOUT")
	r.otlpTimeout("OTEL_EXPORTER_OTLP_TRACES_TIMEOUT")
	r.otlpTimeout("OTEL_EXPORTER_OTLP_LOGS_TIMEOUT")
	r.endpoint("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT")
	return Telemetry{
		ExportTraces:    tracesEndpoint || baseEndpoint,
		ExportLogs:      baseEndpoint,
		ShutdownTimeout: r.duration("NAVA_TELEMETRY_SHUTDOWN_TIMEOUT", 5*time.Second),
	}
}

// addr checks key is a listen address, host:port with a numeric port (0 picks
// a free one). The host may be empty, meaning every interface.
func (r *reader) addr(key, def string) string {
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return def
	}
	_, port, err := net.SplitHostPort(v)
	if err == nil {
		_, err = strconv.ParseUint(port, 10, 16)
	}
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s: want host:port such as :8080", key))
		return def
	}
	return v
}

// databaseURL checks key is a postgres:// or postgresql:// URL with a host. It
// never echoes the value: it carries the password.
func (r *reader) databaseURL(key string) string {
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return ""
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
		r.errs = append(r.errs, fmt.Errorf("%s: want a postgres:// or postgresql:// URL", key))
		return ""
	}
	return v
}

// baseURL checks key is an absolute http or https URL with no userinfo, path,
// query or fragment. Empty is "not configured".
func (r *reader) baseURL(key string) string {
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return ""
	}
	if !isBaseURL(v) || !noPath(v) {
		r.errs = append(r.errs, fmt.Errorf("%s: want an http or https URL without credentials, path or query", key))
		return ""
	}
	return v
}

func isBaseURL(v string) bool {
	u, err := url.Parse(v)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" &&
		u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

// noPath reports whether v names only scheme://host[:port], with at most a
// trailing slash. Issuer URLs keep a path (/realms/nava), so isBaseURL leaves
// it alone and only baseURL requires this.
func noPath(v string) bool {
	u, err := url.Parse(v)
	return err == nil && (u.Path == "" || u.Path == "/")
}

// urlList reads a comma-separated list of URLs, each checked as baseURL
// checks one. Empty is "not configured".
func (r *reader) urlList(key string) []string {
	v, ok := r.lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(v, ",") {
		s = strings.TrimSpace(s)
		if !isBaseURL(s) {
			r.errs = append(r.errs, fmt.Errorf("%s: want comma-separated http or https URLs without credentials or query", key))
			return nil
		}
		out = append(out, s)
	}
	return out
}

// str returns key's value, empty when unset.
func (r *reader) str(key string) string {
	v, _ := r.lookup(key)
	return v
}

// hostPort checks key is host:port with a numeric port. Empty is "not
// configured".
func (r *reader) hostPort(key string) string {
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(v)
	if err == nil {
		_, err = strconv.ParseUint(port, 10, 16)
	}
	if err != nil || host == "" {
		r.errs = append(r.errs, fmt.Errorf("%s: want host:port such as temporal:7233", key))
		return ""
	}
	return v
}

func (r *reader) duration(key string, def time.Duration) time.Duration {
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		r.errs = append(r.errs, fmt.Errorf("%s: want a positive duration such as 15s", key))
		return def
	}
	return d
}

func (r *reader) level(key string, def slog.Level) slog.Level {
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return def
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(v)); err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s: want debug, info, warn or error", key))
		return def
	}
	return l
}

func (r *reader) oneOf(key, def string, allowed ...string) string {
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return def
	}
	v = strings.ToLower(v)
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	r.errs = append(r.errs, fmt.Errorf("%s: want one of %s", key, strings.Join(allowed, ", ")))
	return def
}

// endpoint reports whether key holds an absolute http or https URL. It never
// echoes the value: an endpoint URL can carry credentials.
func (r *reader) endpoint(key string) bool {
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return false
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		r.errs = append(r.errs, fmt.Errorf("%s: want an http or https URL", key))
		return false
	}
	return true
}

// otlpHeaders checks key against the OTLP exporter's header syntax: a
// comma-separated list of key=value entries, each key an HTTP token and each
// value percent-decodable. Like the exporter, a blank value counts as unset.
// It never echoes the value: it carries credentials.
func (r *reader) otlpHeaders(key string) {
	v, ok := r.lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return
	}
	for _, entry := range strings.Split(v, ",") {
		name, value, found := strings.Cut(entry, "=")
		if !found || !isHeaderToken(strings.TrimSpace(name)) {
			r.errs = append(r.errs, fmt.Errorf("%s: want comma-separated key=value entries with a valid header name", key))
			return
		}
		if _, err := url.PathUnescape(value); err != nil {
			r.errs = append(r.errs, fmt.Errorf("%s: a header value has an invalid percent-escape", key))
			return
		}
	}
}

// isHeaderToken reports whether s is a non-empty HTTP token (RFC 9110), the
// same rule the exporter applies to header names.
func isHeaderToken(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", c):
		default:
			return false
		}
	}
	return true
}

// otlpTimeout checks key is a whole number of milliseconds, as the exporter
// parses it. A blank value counts as unset, as it does for the exporter.
func (r *reader) otlpTimeout(key string) {
	v, ok := r.lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return
	}
	if ms, err := strconv.Atoi(strings.TrimSpace(v)); err != nil || ms < 0 {
		r.errs = append(r.errs, fmt.Errorf("%s: want a whole number of milliseconds such as 10000", key))
	}
}
