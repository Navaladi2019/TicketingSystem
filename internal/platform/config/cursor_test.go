package config_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func loadErr(m map[string]string) (config.Config, error) { return config.Load(env(m)) }

func mustLoad(t *testing.T, m map[string]string) config.Config {
	t.Helper()
	cfg, err := loadErr(m)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func cursorKey(c string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat(c, 32)))
}

func TestLoadCursorKeyring(t *testing.T) {
	t.Parallel()
	cfg := mustLoad(t, map[string]string{
		"NAVA_CURSOR_KEYS":          "k1=" + cursorKey("a") + ", k2=" + cursorKey("b"),
		"NAVA_CURSOR_ACTIVE_KEY_ID": "k2",
	})
	if cfg.Cursor.ActiveKeyID != "k2" || len(cfg.Cursor.Keys) != 2 || cfg.Cursor.Keys[0].ID != "k1" ||
		string(cfg.Cursor.Keys[1].Secret) != strings.Repeat("b", 32) {
		t.Fatalf("Cursor = %+v", cfg.Cursor)
	}
}

func TestLoadCursorUnsetIsNotConfigured(t *testing.T) {
	t.Parallel()
	if cfg := mustLoad(t, nil); len(cfg.Cursor.Keys) != 0 || cfg.Cursor.ActiveKeyID != "" {
		t.Fatalf("Cursor = %+v", cfg.Cursor)
	}
}

func TestLoadCursorRefusals(t *testing.T) {
	t.Parallel()
	good := "k1=" + cursorKey("a")
	for name, m := range map[string]map[string]string{
		"no active":      {"NAVA_CURSOR_KEYS": good},
		"unknown active": {"NAVA_CURSOR_KEYS": good, "NAVA_CURSOR_ACTIVE_KEY_ID": "k9"},
		"active alone":   {"NAVA_CURSOR_ACTIVE_KEY_ID": "k1"},
		"short secret":   {"NAVA_CURSOR_KEYS": "k1=" + base64.RawURLEncoding.EncodeToString([]byte("short")), "NAVA_CURSOR_ACTIVE_KEY_ID": "k1"},
		"not base64":     {"NAVA_CURSOR_KEYS": "k1=!!!", "NAVA_CURSOR_ACTIVE_KEY_ID": "k1"},
		"no equals":      {"NAVA_CURSOR_KEYS": "k1", "NAVA_CURSOR_ACTIVE_KEY_ID": "k1"},
		"empty id":       {"NAVA_CURSOR_KEYS": "=" + cursorKey("a"), "NAVA_CURSOR_ACTIVE_KEY_ID": ""},
		"duplicate id":   {"NAVA_CURSOR_KEYS": good + ",k1=" + cursorKey("b"), "NAVA_CURSOR_ACTIVE_KEY_ID": "k1"},
	} {
		_, err := loadErr(m)
		if err == nil {
			t.Errorf("%s: accepted", name)
			continue
		}
		if strings.Contains(err.Error(), cursorKey("a")) || strings.Contains(err.Error(), "short") {
			t.Errorf("%s: error echoes a secret: %v", name, err)
		}
	}
}
