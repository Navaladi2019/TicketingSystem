package config_test

import (
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func env(m map[string]string) config.LookupFunc {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTP.Addr != ":8080" || cfg.HTTP.AdminAddr != ":9090" {
		t.Errorf("Addr, MetricsAddr = %q, %q, want :8080, :9090", cfg.HTTP.Addr, cfg.HTTP.AdminAddr)
	}
	if cfg.HTTP.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 5s", cfg.HTTP.ReadHeaderTimeout)
	}
	if cfg.Log.Level != slog.LevelInfo || cfg.Log.Format != config.FormatJSON {
		t.Errorf("Log = %+v, want info/json", cfg.Log)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(map[string]string{
		"NAVA_HTTP_ADDR":             "127.0.0.1:9000",
		"NAVA_ADMIN_ADDR":            "[::1]:0",
		"NAVA_HTTP_SHUTDOWN_TIMEOUT": "3s",
		"NAVA_LOG_LEVEL":             "debug",
		"NAVA_LOG_FORMAT":            "TEXT",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTP.Addr != "127.0.0.1:9000" || cfg.HTTP.AdminAddr != "[::1]:0" || cfg.HTTP.ShutdownTimeout != 3*time.Second {
		t.Errorf("HTTP = %+v", cfg.HTTP)
	}
	if cfg.Log.Level != slog.LevelDebug || cfg.Log.Format != config.FormatText {
		t.Errorf("Log = %+v, want debug/text", cfg.Log)
	}
}

func TestLoadInvalidReportsEveryKeyWithoutValues(t *testing.T) {
	t.Parallel()
	const secretish = "s3cr3t-value"
	_, err := config.Load(env(map[string]string{
		"NAVA_HTTP_READ_TIMEOUT": secretish,
		"NAVA_HTTP_IDLE_TIMEOUT": "-1s",
		"NAVA_LOG_LEVEL":         "loud",
		"NAVA_LOG_FORMAT":        "xml",
	}))
	if err == nil {
		t.Fatal("Load: want error, got nil")
	}
	msg := err.Error()
	for _, key := range []string{"NAVA_HTTP_READ_TIMEOUT", "NAVA_HTTP_IDLE_TIMEOUT", "NAVA_LOG_LEVEL", "NAVA_LOG_FORMAT"} {
		if !strings.Contains(msg, key) {
			t.Errorf("error %q does not name %s", msg, key)
		}
	}
	if strings.Contains(msg, secretish) {
		t.Errorf("error echoes the invalid value: %q", msg)
	}
}

func TestLoadInvalidAddrs(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"9000", "localhost", ":http", ":65536", ":-1", "host:port:extra", "s3cr3t:pw@host"} {
		for _, key := range []string{"NAVA_HTTP_ADDR", "NAVA_ADMIN_ADDR"} {
			_, err := config.Load(env(map[string]string{key: v}))
			if err == nil {
				t.Errorf("%s=%q: Load accepted it", key, v)
				continue
			}
			if msg := err.Error(); !strings.Contains(msg, key) || strings.Contains(msg, v) {
				t.Errorf("%s=%q: error %q, want the key named and the value not echoed", key, v, msg)
			}
		}
	}
}

// The two listeners must not share an address: the internal one is what keeps
// /metrics off the public port.
func TestLoadRejectsSharedListenerAddr(t *testing.T) {
	t.Parallel()
	for _, m := range []map[string]string{
		{"NAVA_ADMIN_ADDR": ":8080"},
		{"NAVA_HTTP_ADDR": ":9090"},
		{"NAVA_HTTP_ADDR": "127.0.0.1:7000", "NAVA_ADMIN_ADDR": "127.0.0.1:7000"},
	} {
		_, err := config.Load(env(m))
		if err == nil || !strings.Contains(err.Error(), "NAVA_ADMIN_ADDR") {
			t.Errorf("%v: Load = %v, want an error naming NAVA_ADMIN_ADDR", m, err)
		}
	}
}

func TestLoadTelemetryDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := config.Telemetry{ExportTraces: false, ShutdownTimeout: 5 * time.Second}
	if cfg.Telemetry != want {
		t.Errorf("Telemetry = %+v, want %+v", cfg.Telemetry, want)
	}
}

func TestLoadTelemetryEitherEndpointExports(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"} {
		cfg, err := config.Load(env(map[string]string{
			key:                               "http://collector:4318",
			"NAVA_TELEMETRY_SHUTDOWN_TIMEOUT": "2s",
		}))
		if err != nil {
			t.Fatalf("%s: Load: %v", key, err)
		}
		// Only the base endpoint turns log export on, and the logs endpoint
		// alone does not turn trace export on.
		want := config.Telemetry{
			ExportTraces:    key != "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
			ExportLogs:      key == "OTEL_EXPORTER_OTLP_ENDPOINT",
			ShutdownTimeout: 2 * time.Second,
		}
		if cfg.Telemetry != want {
			t.Errorf("%s: Telemetry = %+v, want %+v", key, cfg.Telemetry, want)
		}
	}
}

func TestLoadTelemetryInvalidNeverEchoesValues(t *testing.T) {
	t.Parallel()
	const secretish = "user:s3cr3t@" // a credential pasted into the endpoint
	_, err := config.Load(env(map[string]string{
		// A valid signal endpoint must not hide an invalid base one.
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://collector:4318/v1/traces",
		"OTEL_EXPORTER_OTLP_ENDPOINT":        "ftp://" + secretish + "collector",
		"NAVA_TELEMETRY_SHUTDOWN_TIMEOUT":    "0s",
	}))
	if err == nil {
		t.Fatal("Load: want error, got nil")
	}
	msg := err.Error()
	for _, key := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "NAVA_TELEMETRY_SHUTDOWN_TIMEOUT"} {
		if !strings.Contains(msg, key) {
			t.Errorf("error %q does not name %s", msg, key)
		}
	}
	if strings.Contains(msg, "s3cr3t") {
		t.Errorf("error echoes the invalid value: %q", msg)
	}
}

func TestLoadOTLPHeadersInvalidNeverEchoesValues(t *testing.T) {
	t.Parallel()
	const secret = "s3cr3t-token"
	malformed := map[string]string{
		"missing =":  "Authorization: Bearer " + secret,
		"empty key":  "=Bearer " + secret,
		"bad key":    "Auth orization=Bearer " + secret,
		"bad escape": "Authorization=Bearer%zz" + secret,
		"second bad": "x-tenant=a,Authorization: Bearer " + secret,
	}
	for _, key := range []string{"OTEL_EXPORTER_OTLP_HEADERS", "OTEL_EXPORTER_OTLP_TRACES_HEADERS"} {
		for name, value := range malformed {
			_, err := config.Load(env(map[string]string{key: value}))
			if err == nil {
				t.Errorf("%s %s: Load: want error, got nil", key, name)
				continue
			}
			msg := err.Error()
			if !strings.Contains(msg, key) {
				t.Errorf("%s %s: error %q does not name the variable", key, name, msg)
			}
			if strings.Contains(msg, secret) || strings.Contains(msg, "Bearer") {
				t.Errorf("%s %s: error echoes the value: %q", key, name, msg)
			}
		}
	}
}

func TestLoadOTLPHeadersValid(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"Authorization=Bearer%20s3cr3t-token",
		" api-key = abc , x-tenant=a%2Cb ",
		"x-empty=",
		"   ", // blank counts as unset, as it does for the exporter
	} {
		_, err := config.Load(env(map[string]string{
			"OTEL_EXPORTER_OTLP_HEADERS":        value,
			"OTEL_EXPORTER_OTLP_TRACES_HEADERS": value,
		}))
		if err != nil {
			t.Errorf("%q: Load: %v", value, err)
		}
	}
}

func TestLoadOTLPTimeout(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"OTEL_EXPORTER_OTLP_TIMEOUT", "OTEL_EXPORTER_OTLP_TRACES_TIMEOUT"} {
		if _, err := config.Load(env(map[string]string{key: "10000"})); err != nil {
			t.Errorf("%s=10000: Load: %v", key, err)
		}
		for _, bad := range []string{"10s", "-1", "1.5"} {
			_, err := config.Load(env(map[string]string{key: bad}))
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Errorf("%s=%s: Load error = %v, want one naming the variable", key, bad, err)
				continue
			}
			if strings.Contains(err.Error(), bad) {
				t.Errorf("%s=%s: error echoes the value: %q", key, bad, err)
			}
		}
	}
}

func TestRejectsBadBadDependencyValuesByName(t *testing.T) {
	t.Parallel()
	for key, bad := range map[string]string{
		"NAVA_DATABASE_URL":            "mysql://u:hunter2@h/db",
		"NAVA_KEYCLOAK_MANAGEMENT_URL": "http://u:hunter2@keycloak:9000",
		"NAVA_KEYCLOAK_TOKEN_URL":      "http://u:hunter2@keycloak:8080/realms/nava/protocol/openid-connect/token",
		"NAVA_TEMPORAL_ADDRESS":        "temporal",
	} {
		_, err := config.Load(func(k string) (string, bool) { return bad, k == key })
		if err == nil || !strings.Contains(err.Error(), key) || strings.Contains(err.Error(), "hunter2") {
			t.Errorf("%s: error = %v, want one naming the key and not the value", key, err)
		}
	}
}

// NAVA_OIDC_FETCH_BASE_URL is a plain scheme://host[:port]: a path or query is
// refused (TICK-279), a trailing slash is trimmed.
func TestLoadFetchBaseURL(t *testing.T) {
	t.Parallel()
	load := func(v string) (config.Config, error) {
		return config.Load(func(k string) (string, bool) { return v, k == "NAVA_OIDC_FETCH_BASE_URL" })
	}
	cfg, err := load("http://keycloak:8080/")
	if err != nil || cfg.Deps.OIDCFetchBaseURL != "http://keycloak:8080" {
		t.Errorf("trailing slash: %q, %v", cfg.Deps.OIDCFetchBaseURL, err)
	}
	for _, bad := range []string{"http://keycloak:8080/realms/nava", "http://keycloak:8080?x=1", "http://keycloak:8080/realms/nava?x=1"} {
		if _, err := load(bad); err == nil || !strings.Contains(err.Error(), "NAVA_OIDC_FETCH_BASE_URL") {
			t.Errorf("NAVA_OIDC_FETCH_BASE_URL=%q: error = %v, want one naming the key", bad, err)
		}
	}
}

func TestDependencyDefaultsAreUnconfigured(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.Deps, config.Dependencies{}) {
		t.Errorf("Deps = %+v, want zero", cfg.Deps)
	}
}

func TestTemporalTLSFilesAreAllOrNone(t *testing.T) {
	t.Parallel()
	load := func(env map[string]string) (config.Config, error) {
		return config.Load(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	}
	cfg, err := load(map[string]string{
		"NAVA_TEMPORAL_TLS_CERT_FILE": "c.pem", "NAVA_TEMPORAL_TLS_KEY_FILE": "c.key",
		"NAVA_TEMPORAL_TLS_CA_FILE": "ca.pem", "NAVA_TEMPORAL_TLS_SERVER_NAME": "temporal",
	})
	if err != nil || !cfg.Deps.TemporalTLS.Enabled() || cfg.Deps.TemporalTLS.ServerName != "temporal" {
		t.Fatalf("full set: %+v, %v", cfg.Deps.TemporalTLS, err)
	}
	if cfg, err = load(nil); err != nil || cfg.Deps.TemporalTLS.Enabled() {
		t.Fatalf("none set: %+v, %v", cfg.Deps.TemporalTLS, err)
	}
	_, err = load(map[string]string{"NAVA_TEMPORAL_TLS_CERT_FILE": "c.pem"})
	if err == nil || !strings.Contains(err.Error(), "NAVA_TEMPORAL_TLS_CA_FILE") {
		t.Fatalf("partial set: %v, want an error naming the variables", err)
	}
}

func TestLoadRateLimit(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(env(nil))
	if err != nil || cfg.RateLimit.Rate != 50 || cfg.RateLimit.Burst != 100 || cfg.RateLimit.IdleTTL != 10*time.Minute {
		t.Fatalf("defaults = %+v, %v", cfg.RateLimit, err)
	}
	cfg, err = config.Load(env(map[string]string{
		"NAVA_RATE_LIMIT_RPS": "2.5", "NAVA_RATE_LIMIT_BURST": "7", "NAVA_RATE_LIMIT_IDLE_TTL": "30s",
	}))
	if err != nil || cfg.RateLimit.Rate != 2.5 || cfg.RateLimit.Burst != 7 || cfg.RateLimit.IdleTTL != 30*time.Second {
		t.Fatalf("overrides = %+v, %v", cfg.RateLimit, err)
	}
	for _, key := range []string{"NAVA_RATE_LIMIT_RPS", "NAVA_RATE_LIMIT_BURST", "NAVA_RATE_LIMIT_IDLE_TTL"} {
		for _, bad := range []string{"0", "-1", "abc"} {
			if _, err := config.Load(env(map[string]string{key: bad})); err == nil || !strings.Contains(err.Error(), key) || strings.Contains(err.Error(), bad+"\"") {
				t.Errorf("%s=%q: err = %v, want a named error", key, bad, err)
			}
		}
	}
}

func TestTemporalTLSRequiresServerName(t *testing.T) {
	t.Parallel()
	load := func(env map[string]string) (config.Config, error) {
		return config.Load(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	}
	files := map[string]string{
		"NAVA_TEMPORAL_TLS_CERT_FILE": "c.pem", "NAVA_TEMPORAL_TLS_KEY_FILE": "c.key",
		"NAVA_TEMPORAL_TLS_CA_FILE": "ca.pem",
	}
	_, err := load(files)
	if err == nil || !strings.Contains(err.Error(), "NAVA_TEMPORAL_TLS_SERVER_NAME") {
		t.Fatalf("TLS on, no server name: %v, want an error naming NAVA_TEMPORAL_TLS_SERVER_NAME", err)
	}
	files["NAVA_TEMPORAL_TLS_SERVER_NAME"] = "temporal"
	if cfg, err := load(files); err != nil || cfg.Deps.TemporalTLS.ServerName != "temporal" {
		t.Fatalf("TLS on with server name: %+v, %v", cfg.Deps.TemporalTLS, err)
	}
	if _, err := load(nil); err != nil {
		t.Fatalf("TLS off, nothing set: %v", err)
	}
	// A partial file set reports only the all-or-none error, not a second one
	// about the server name.
	_, err = load(map[string]string{"NAVA_TEMPORAL_TLS_CERT_FILE": "c.pem"})
	if err == nil || strings.Contains(err.Error(), "NAVA_TEMPORAL_TLS_SERVER_NAME") {
		t.Fatalf("partial set: %v, want only the all-or-none error", err)
	}
}
