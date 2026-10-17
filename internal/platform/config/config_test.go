package config_test

import (
	"log/slog"
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
	if cfg.HTTP.Addr != ":8080" || cfg.HTTP.MetricsAddr != ":9090" {
		t.Errorf("Addr, MetricsAddr = %q, %q, want :8080, :9090", cfg.HTTP.Addr, cfg.HTTP.MetricsAddr)
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
		"NAVA_METRICS_ADDR":          "[::1]:0",
		"NAVA_HTTP_SHUTDOWN_TIMEOUT": "3s",
		"NAVA_LOG_LEVEL":             "debug",
		"NAVA_LOG_FORMAT":            "TEXT",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTP.Addr != "127.0.0.1:9000" || cfg.HTTP.MetricsAddr != "[::1]:0" || cfg.HTTP.ShutdownTimeout != 3*time.Second {
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
		for _, key := range []string{"NAVA_HTTP_ADDR", "NAVA_METRICS_ADDR"} {
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
		{"NAVA_METRICS_ADDR": ":8080"},
		{"NAVA_HTTP_ADDR": ":9090"},
		{"NAVA_HTTP_ADDR": "127.0.0.1:7000", "NAVA_METRICS_ADDR": "127.0.0.1:7000"},
	} {
		_, err := config.Load(env(m))
		if err == nil || !strings.Contains(err.Error(), "NAVA_METRICS_ADDR") {
			t.Errorf("%v: Load = %v, want an error naming NAVA_METRICS_ADDR", m, err)
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
	for _, key := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"} {
		cfg, err := config.Load(env(map[string]string{
			key:                               "http://collector:4318",
			"NAVA_TELEMETRY_SHUTDOWN_TIMEOUT": "2s",
		}))
		if err != nil {
			t.Fatalf("%s: Load: %v", key, err)
		}
		want := config.Telemetry{ExportTraces: true, ShutdownTimeout: 2 * time.Second}
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
