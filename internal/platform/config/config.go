// Package config loads nava-core configuration from the environment into a
// typed struct. There are no package-level settings: callers pass the Config
// they loaded to the constructors that need it.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Config is the full nava-core configuration.
type Config struct {
	HTTP HTTP
	Log  Log
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
