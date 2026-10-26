package config

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Groupsync is the nava-groupsync configuration.
type Groupsync struct {
	AdminAddr string // NAVA_ADMIN_ADDR: the admin listener (/metrics, /healthz, /readyz)
	Log       Log
	Telemetry Telemetry
	// DatabaseURL is NAVA_DATABASE_URL, connecting as nava_groupsync. It holds
	// a password: it is never logged and never echoed in an error.
	DatabaseURL string
	// Interval between sync cycles. NAVA_GROUPSYNC_INTERVAL. At most
	// MaxGroupsyncInterval (D-P9).
	Interval time.Duration
	// KeycloakURL is NAVA_KEYCLOAK_URL, the Keycloak origin the admin API is
	// served from.
	KeycloakURL string
	// KeycloakRealm is NAVA_KEYCLOAK_REALM.
	KeycloakRealm string
	// ClientID and ClientSecret are the runtime service account
	// (NAVA_GROUPSYNC_CLIENT_ID, NAVA_GROUPSYNC_CLIENT_SECRET). The secret is
	// never logged and never echoed in an error.
	ClientID     string
	ClientSecret string
	// HTTP holds the admin listener's timeouts.
	HTTP HTTP
}

// MaxGroupsyncInterval is the longest sync interval (D-P9: every 5 minutes at
// most). A longer one is a configuration error, not a slower sync.
const MaxGroupsyncInterval = 5 * time.Minute

// LoadGroupsync reads the nava-groupsync configuration through lookup, with
// the same rules as Load.
func LoadGroupsync(lookup LookupFunc) (Groupsync, error) {
	r := reader{lookup: lookup}
	tel := r.telemetry()
	cfg := Groupsync{
		AdminAddr: r.addr("NAVA_ADMIN_ADDR", ":9090"),
		Log: Log{
			Level:  r.level("NAVA_LOG_LEVEL", slog.LevelInfo),
			Format: r.oneOf("NAVA_LOG_FORMAT", FormatJSON, FormatJSON, FormatText),
		},
		Telemetry:     tel,
		DatabaseURL:   r.databaseURL("NAVA_DATABASE_URL"),
		Interval:      r.duration("NAVA_GROUPSYNC_INTERVAL", MaxGroupsyncInterval),
		KeycloakURL:   r.baseURL("NAVA_KEYCLOAK_URL"),
		KeycloakRealm: r.str("NAVA_KEYCLOAK_REALM"),
		ClientID:      r.str("NAVA_GROUPSYNC_CLIENT_ID"),
		ClientSecret:  r.str("NAVA_GROUPSYNC_CLIENT_SECRET"),
		HTTP: HTTP{
			ReadHeaderTimeout: r.duration("NAVA_HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       r.duration("NAVA_HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      r.duration("NAVA_HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:       r.duration("NAVA_HTTP_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout:   r.duration("NAVA_HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
	}
	if cfg.KeycloakRealm == "" {
		cfg.KeycloakRealm = "nava"
	}
	for key, empty := range map[string]bool{
		"NAVA_DATABASE_URL":            cfg.DatabaseURL == "",
		"NAVA_KEYCLOAK_URL":            cfg.KeycloakURL == "",
		"NAVA_GROUPSYNC_CLIENT_ID":     cfg.ClientID == "",
		"NAVA_GROUPSYNC_CLIENT_SECRET": cfg.ClientSecret == "",
	} {
		if empty {
			r.errs = append(r.errs, fmt.Errorf("%s: required", key))
		}
	}
	if cfg.Interval > MaxGroupsyncInterval {
		r.errs = append(r.errs, fmt.Errorf("NAVA_GROUPSYNC_INTERVAL: must be at most %s", MaxGroupsyncInterval))
	}
	if err := errors.Join(r.errs...); err != nil {
		return Groupsync{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
