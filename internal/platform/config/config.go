// Package config loads nava-core configuration from the environment into a
// typed struct. There are no package-level settings: callers pass the Config
// they loaded to the constructors that need it.
package config

import (
	"errors"
	"fmt"
	"log/slog"
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

// HTTP configures the API listener.
type HTTP struct {
	Addr              string        // NAVA_HTTP_ADDR
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

// Telemetry configures OpenTelemetry. It uses the standard OTEL_* names, read
// here like every other variable, so the SDK never reads them behind our back.
type Telemetry struct {
	ServiceName  string  // OTEL_SERVICE_NAME
	OTLPEndpoint string  // OTEL_EXPORTER_OTLP_ENDPOINT; empty means no trace exporter
	Sampler      string  // OTEL_TRACES_SAMPLER
	SamplerArg   float64 // OTEL_TRACES_SAMPLER_ARG: the ratio for the *traceidratio samplers
}

// Trace samplers, as named by the OpenTelemetry environment specification.
const (
	SamplerAlwaysOn                = "always_on"
	SamplerAlwaysOff               = "always_off"
	SamplerTraceIDRatio            = "traceidratio"
	SamplerParentBasedAlwaysOn     = "parentbased_always_on"
	SamplerParentBasedAlwaysOff    = "parentbased_always_off"
	SamplerParentBasedTraceIDRatio = "parentbased_traceidratio"
)

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
	cfg := Config{
		HTTP: HTTP{
			Addr:              r.str("NAVA_HTTP_ADDR", ":8080"),
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
			ServiceName:  r.str("OTEL_SERVICE_NAME", "nava-core"),
			OTLPEndpoint: r.endpoint("OTEL_EXPORTER_OTLP_ENDPOINT"),
			Sampler: r.oneOf("OTEL_TRACES_SAMPLER", SamplerParentBasedTraceIDRatio,
				SamplerAlwaysOn, SamplerAlwaysOff, SamplerTraceIDRatio,
				SamplerParentBasedAlwaysOn, SamplerParentBasedAlwaysOff, SamplerParentBasedTraceIDRatio),
			SamplerArg: r.ratio("OTEL_TRACES_SAMPLER_ARG", 1.0),
		},
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

func (r *reader) str(key, def string) string {
	if v, ok := r.lookup(key); ok && v != "" {
		return v
	}
	return def
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

func (r *reader) ratio(key string, def float64) float64 {
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 || f > 1 {
		r.errs = append(r.errs, fmt.Errorf("%s: want a number from 0 to 1", key))
		return def
	}
	return f
}

// endpoint accepts an absolute http or https base URL, or nothing. It never
// echoes the value: an endpoint URL can carry credentials.
func (r *reader) endpoint(key string) string {
	v, ok := r.lookup(key)
	if !ok || v == "" {
		return ""
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		r.errs = append(r.errs, fmt.Errorf("%s: want an http or https base URL", key))
		return ""
	}
	return v
}
