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
	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.HTTP.Addr)
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
		"NAVA_HTTP_SHUTDOWN_TIMEOUT": "3s",
		"NAVA_LOG_LEVEL":             "debug",
		"NAVA_LOG_FORMAT":            "TEXT",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTP.Addr != "127.0.0.1:9000" || cfg.HTTP.ShutdownTimeout != 3*time.Second {
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
