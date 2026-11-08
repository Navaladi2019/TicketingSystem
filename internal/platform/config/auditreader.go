package config

import (
	"errors"
	"fmt"
)

// AuditReaderDatabaseURLKey is the audit reader's DSN key (ADR-004-A1 A1-6).
// Only nava-core reads it, through LoadAuditReader: config.Load, which
// nava-core, nava-worker and nava-relay share, never does, so the worker and
// the relay neither need nor see the raw-diff credential (PLAN-005-A1
// D-A1-6, AR-TICK-549-01).
const AuditReaderDatabaseURLKey = "NAVA_AUDIT_READER_DATABASE_URL"

// AuditReader is nava-core's audit reader configuration.
type AuditReader struct {
	// DatabaseURL is NAVA_AUDIT_READER_DATABASE_URL, connecting as
	// nava_audit_reader through PgBouncer. It is handed only to
	// read.OpenPool. It holds a password: it is never logged and never
	// echoed in an error.
	DatabaseURL string
}

// LoadAuditReader reads the audit reader configuration through lookup, with
// the same rules as Load. The DSN is required: nava-core does not start
// without it, because every audit read would otherwise fail.
func LoadAuditReader(lookup LookupFunc) (AuditReader, error) {
	r := reader{lookup: lookup}
	cfg := AuditReader{DatabaseURL: r.databaseURL(AuditReaderDatabaseURLKey)}
	if cfg.DatabaseURL == "" && len(r.errs) == 0 {
		r.errs = append(r.errs, errors.New(AuditReaderDatabaseURLKey+": required"))
	}
	if err := errors.Join(r.errs...); err != nil {
		return AuditReader{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
