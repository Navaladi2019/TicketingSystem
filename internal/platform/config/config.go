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
}

// HTTP configures the two listeners. Both use the same timeouts.
type HTTP struct {
	Addr              string        // NAVA_HTTP_ADDR: the public API listener
	MetricsAddr       string        // NAVA_METRICS_ADDR: the internal listener (/metrics)
	ReadHeaderTimeout time.Duration // NAVA_HTTP_READ_HEADER_TIMEOUT
	ReadTimeout       time.Duration // NAVA_HTTP_READ_TIMEOUT
	WriteTimeout      time.Duration // NAVA_HTTP_WRITE_TIMEOUT
	IdleTimeout       time.Duration // NAVA_HTTP_IDLE_TIMEOUT
	ShutdownTimeout   time.Duration // NAVA_HTTP_SHUTDOWN_TIMEOUT
}

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
	// ShutdownTimeout is the flush budget after the HTTP server has drained,
	// separate from it. NAVA_TELEMETRY_SHUTDOWN_TIMEOUT.
	ShutdownTimeout time.Duration
}

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
	// Both endpoints are validated, so a bad value fails here, by name, before
	// the exporter would report it with the value.
	tracesEndpoint := r.endpoint("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	baseEndpoint := r.endpoint("OTEL_EXPORTER_OTLP_ENDPOINT")
	// The exporter parses these itself and logs a bad entry or timeout with
	// its value, which for headers is a bearer token or API key. Check them
	// here, by name, so the exporter only ever sees values it accepts.
	r.otlpHeaders("OTEL_EXPORTER_OTLP_HEADERS")
	r.otlpHeaders("OTEL_EXPORTER_OTLP_TRACES_HEADERS")
	r.otlpTimeout("OTEL_EXPORTER_OTLP_TIMEOUT")
	r.otlpTimeout("OTEL_EXPORTER_OTLP_TRACES_TIMEOUT")
	cfg := Config{
		HTTP: HTTP{
			Addr:              r.addr("NAVA_HTTP_ADDR", ":8080"),
			MetricsAddr:       r.addr("NAVA_METRICS_ADDR", ":9090"),
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
		Telemetry: Telemetry{
			ExportTraces:    tracesEndpoint || baseEndpoint,
			ShutdownTimeout: r.duration("NAVA_TELEMETRY_SHUTDOWN_TIMEOUT", 5*time.Second),
		},
	}
	if cfg.HTTP.Addr == cfg.HTTP.MetricsAddr {
		r.errs = append(r.errs, errors.New("NAVA_METRICS_ADDR: must differ from NAVA_HTTP_ADDR"))
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
