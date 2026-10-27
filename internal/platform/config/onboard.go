package config

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
)

// OperatorEntry is one operator allow-list entry.
type OperatorEntry struct{ Issuer, Subject string }

// Onboard is the nava-onboard configuration (T21): the process that serves
// the operator routes. It is the only process that reads the onboarding
// service account's secret and the nava_onboard database URL; nava-core reads
// neither (the request path never loads them).
type Onboard struct {
	Addr      string // NAVA_ONBOARD_ADDR: the operator API listener
	AdminAddr string // NAVA_ADMIN_ADDR: the admin listener (/metrics, /healthz, /readyz)
	Log       Log
	Telemetry Telemetry
	HTTP      HTTP

	// DatabaseURL is NAVA_DATABASE_URL, connecting as nava_app (tenant-token
	// refusal and the invite's registry). OnboardDatabaseURL is
	// NAVA_ONBOARD_DATABASE_URL, connecting as nava_onboard. Both hold
	// passwords: never logged, never echoed in an error.
	DatabaseURL        string
	OnboardDatabaseURL string
	// OIDCIssuers is NAVA_OIDC_ISSUERS; Issuer is NAVA_ONBOARD_ISSUER, one of
	// them, written to the tenant binding.
	OIDCIssuers []string
	Issuer      string
	// Operators is NAVA_OPERATORS: comma-separated "issuer|subject" pairs
	// (AR-030). Required: empty would refuse every call.
	Operators []OperatorEntry

	KeycloakURL   string // NAVA_KEYCLOAK_URL
	KeycloakRealm string // NAVA_KEYCLOAK_REALM
	// RuntimeClientID/Secret invite the first admin (NAVA_DIRECTORY_CLIENT_ID,
	// NAVA_DIRECTORY_CLIENT_SECRET). OnboardClientID/Secret are the onboarding
	// service account (NAVA_ONBOARD_CLIENT_ID, NAVA_ONBOARD_CLIENT_SECRET).
	// Secrets are never logged or echoed.
	RuntimeClientID     string
	RuntimeClientSecret string
	OnboardClientID     string
	OnboardClientSecret string
	// InviteTTL is how long a pending admin invite stays valid
	// (NAVA_INVITE_TTL, default 168h). An invitee who has not signed in by
	// then needs a new onboarding call.
	InviteTTL time.Duration
}

// DefaultInviteTTL and MaxInviteTTL bound NAVA_INVITE_TTL.
const (
	DefaultInviteTTL = 7 * 24 * time.Hour
	MaxInviteTTL     = 30 * 24 * time.Hour
)

// LoadOnboard reads the nava-onboard configuration through lookup, with the
// same rules as Load.
func LoadOnboard(lookup LookupFunc) (Onboard, error) {
	r := reader{lookup: lookup}
	tel := r.telemetry()
	cfg := Onboard{
		Addr:      r.addr("NAVA_ONBOARD_ADDR", ":8081"),
		AdminAddr: r.addr("NAVA_ADMIN_ADDR", ":9090"),
		Log: Log{
			Level:  r.level("NAVA_LOG_LEVEL", slog.LevelInfo),
			Format: r.oneOf("NAVA_LOG_FORMAT", FormatJSON, FormatJSON, FormatText),
		},
		Telemetry: tel,
		HTTP: HTTP{
			ReadHeaderTimeout: r.duration("NAVA_HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       r.duration("NAVA_HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      r.duration("NAVA_ONBOARD_WRITE_TIMEOUT", 120*time.Second),
			IdleTimeout:       r.duration("NAVA_HTTP_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout:   r.duration("NAVA_HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		DatabaseURL:         r.databaseURL("NAVA_DATABASE_URL"),
		OnboardDatabaseURL:  r.databaseURL("NAVA_ONBOARD_DATABASE_URL"),
		OIDCIssuers:         r.urlList("NAVA_OIDC_ISSUERS"),
		Issuer:              r.str("NAVA_ONBOARD_ISSUER"),
		Operators:           r.operators("NAVA_OPERATORS"),
		KeycloakURL:         r.baseURL("NAVA_KEYCLOAK_URL"),
		KeycloakRealm:       r.str("NAVA_KEYCLOAK_REALM"),
		RuntimeClientID:     r.str("NAVA_DIRECTORY_CLIENT_ID"),
		RuntimeClientSecret: r.str("NAVA_DIRECTORY_CLIENT_SECRET"),
		OnboardClientID:     r.str("NAVA_ONBOARD_CLIENT_ID"),
		OnboardClientSecret: r.str("NAVA_ONBOARD_CLIENT_SECRET"),
		InviteTTL:           r.duration("NAVA_INVITE_TTL", DefaultInviteTTL),
	}
	if cfg.KeycloakRealm == "" {
		cfg.KeycloakRealm = "nava"
	}
	for key, empty := range map[string]bool{
		"NAVA_DATABASE_URL":            cfg.DatabaseURL == "",
		"NAVA_ONBOARD_DATABASE_URL":    cfg.OnboardDatabaseURL == "",
		"NAVA_OIDC_ISSUERS":            len(cfg.OIDCIssuers) == 0,
		"NAVA_ONBOARD_ISSUER":          cfg.Issuer == "",
		"NAVA_OPERATORS":               len(cfg.Operators) == 0,
		"NAVA_KEYCLOAK_URL":            cfg.KeycloakURL == "",
		"NAVA_DIRECTORY_CLIENT_ID":     cfg.RuntimeClientID == "",
		"NAVA_DIRECTORY_CLIENT_SECRET": cfg.RuntimeClientSecret == "",
		"NAVA_ONBOARD_CLIENT_ID":       cfg.OnboardClientID == "",
		"NAVA_ONBOARD_CLIENT_SECRET":   cfg.OnboardClientSecret == "",
	} {
		if empty {
			r.errs = append(r.errs, fmt.Errorf("%s: required", key))
		}
	}
	if cfg.Issuer != "" && !slices.Contains(cfg.OIDCIssuers, cfg.Issuer) {
		r.errs = append(r.errs, errors.New("NAVA_ONBOARD_ISSUER: must be one of NAVA_OIDC_ISSUERS"))
	}
	if cfg.InviteTTL <= 0 || cfg.InviteTTL > MaxInviteTTL {
		r.errs = append(r.errs, fmt.Errorf("NAVA_INVITE_TTL: must be positive and at most %s", MaxInviteTTL))
	}
	if cfg.Addr == cfg.AdminAddr {
		r.errs = append(r.errs, errors.New("NAVA_ADMIN_ADDR: must differ from NAVA_ONBOARD_ADDR"))
	}
	if err := errors.Join(r.errs...); err != nil {
		return Onboard{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// operators reads "issuer|subject" pairs. A malformed entry is an error that
// does not echo the value.
func (r *reader) operators(key string) []OperatorEntry {
	v, _ := r.lookup(key)
	if strings.TrimSpace(v) == "" {
		return nil
	}
	var out []OperatorEntry
	for _, s := range strings.Split(v, ",") {
		iss, sub, ok := strings.Cut(strings.TrimSpace(s), "|")
		if !ok || !isBaseURL(iss) || strings.TrimSpace(sub) == "" {
			r.errs = append(r.errs, fmt.Errorf("%s: want comma-separated issuer|subject pairs", key))
			return nil
		}
		out = append(out, OperatorEntry{Issuer: iss, Subject: strings.TrimSpace(sub)})
	}
	return out
}
