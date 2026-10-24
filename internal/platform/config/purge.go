package config

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Floors for the retention windows. The nava_purge policies in
// db/migrations/00002_platform_tables.sql fix these as their predicates, so a
// shorter window would silently delete nothing: LoadPurge rejects it instead.
const (
	MinOutboxDispatchedRetention = 7 * 24 * time.Hour
	MinOutboxDeadRetention       = 30 * 24 * time.Hour
	MinWorkflowRefRetention      = 30 * 24 * time.Hour
)

// Purge is the nava-purge configuration.
type Purge struct {
	AdminAddr string // NAVA_ADMIN_ADDR: the admin listener (/metrics, /healthz, /readyz)
	Log       Log
	Telemetry Telemetry
	// DatabaseURL is NAVA_DATABASE_URL, connecting as nava_purge. It holds a
	// password: it is never logged and never echoed in an error.
	DatabaseURL string
	// Interval between purge runs. NAVA_PURGE_INTERVAL.
	Interval time.Duration
	// OutboxDispatched, OutboxDead and WorkflowRef are how long a row is kept
	// after dispatched_at, dead_at and closed_at. idempotency_keys has no
	// window: a row goes once expires_at has passed.
	OutboxDispatched time.Duration // NAVA_PURGE_OUTBOX_DISPATCHED_RETENTION
	OutboxDead       time.Duration // NAVA_PURGE_OUTBOX_DEAD_RETENTION
	WorkflowRef      time.Duration // NAVA_PURGE_WORKFLOW_REF_RETENTION
	// HTTP holds the admin listener's timeouts.
	HTTP HTTP
}

// LoadPurge reads the nava-purge configuration through lookup, with the same
// rules as Load.
func LoadPurge(lookup LookupFunc) (Purge, error) {
	r := reader{lookup: lookup}
	tel := r.telemetry()
	cfg := Purge{
		AdminAddr: r.addr("NAVA_ADMIN_ADDR", ":9090"),
		Log: Log{
			Level:  r.level("NAVA_LOG_LEVEL", slog.LevelInfo),
			Format: r.oneOf("NAVA_LOG_FORMAT", FormatJSON, FormatJSON, FormatText),
		},
		Telemetry:        tel,
		DatabaseURL:      r.databaseURL("NAVA_DATABASE_URL"),
		Interval:         r.duration("NAVA_PURGE_INTERVAL", time.Hour),
		OutboxDispatched: r.duration("NAVA_PURGE_OUTBOX_DISPATCHED_RETENTION", MinOutboxDispatchedRetention),
		OutboxDead:       r.duration("NAVA_PURGE_OUTBOX_DEAD_RETENTION", MinOutboxDeadRetention),
		WorkflowRef:      r.duration("NAVA_PURGE_WORKFLOW_REF_RETENTION", MinWorkflowRefRetention),
		HTTP: HTTP{
			ReadHeaderTimeout: r.duration("NAVA_HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       r.duration("NAVA_HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      r.duration("NAVA_HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:       r.duration("NAVA_HTTP_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout:   r.duration("NAVA_HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
	}
	if cfg.DatabaseURL == "" {
		r.errs = append(r.errs, errors.New("NAVA_DATABASE_URL: required"))
	}
	for _, w := range []struct {
		key string
		got time.Duration
		min time.Duration
	}{
		{"NAVA_PURGE_OUTBOX_DISPATCHED_RETENTION", cfg.OutboxDispatched, MinOutboxDispatchedRetention},
		{"NAVA_PURGE_OUTBOX_DEAD_RETENTION", cfg.OutboxDead, MinOutboxDeadRetention},
		{"NAVA_PURGE_WORKFLOW_REF_RETENTION", cfg.WorkflowRef, MinWorkflowRefRetention},
	} {
		if w.got < w.min {
			r.errs = append(r.errs, fmt.Errorf("%s: must be at least %s, the window the nava_purge policy allows", w.key, w.min))
		}
	}
	if err := errors.Join(r.errs...); err != nil {
		return Purge{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
