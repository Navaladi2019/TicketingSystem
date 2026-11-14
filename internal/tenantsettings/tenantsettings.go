// Package tenantsettings reads and writes the tenant-level settings that live
// as columns on tenants. Today that is time_zone (ADR-005 §3.6, TICK-672): the
// IANA zone the date parts of a record number are read in. A change affects
// future scope keys only.
//
// Every function takes the caller's transaction, opened by
// tenancy.InTenantTx, so row-level security lets it see and change its own
// tenant's row and no other. A change writes one config audit entry under the
// tenant_setting subject (TICK-854) in the same transaction.
package tenantsettings

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // zones validate the same on every host, with or without a zoneinfo database

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
)

var (
	// ErrNotFound means the caller's tenant row is not visible.
	ErrNotFound = errors.New("tenantsettings: tenant not found")
	// ErrInvalidTimeZone means the name is not an IANA zone this build knows.
	ErrInvalidTimeZone = errors.New("tenantsettings: not an IANA time zone")
	// ErrNoActor means a write named no principal: the audit actor is never
	// taken from the request.
	ErrNoActor = errors.New("tenantsettings: write needs a principal")
)

// maxTimeZoneBytes matches the shape CHECK on tenants.time_zone.
const maxTimeZoneBytes = 64

// CheckTimeZone accepts an IANA name time.LoadLocation knows. "" and "Local"
// are refused: neither names a zone, and "Local" is the server's, not a
// tenant's.
func CheckTimeZone(name string) error {
	if name == "" || name == "Local" || len(name) > maxTimeZoneBytes || strings.TrimSpace(name) != name {
		return ErrInvalidTimeZone
	}
	if _, err := time.LoadLocation(name); err != nil {
		return ErrInvalidTimeZone
	}
	return nil
}

// Store reads and writes tenant settings. It holds no state.
type Store struct{}

// TimeZone returns the tenant's time zone.
func (Store) TimeZone(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (string, error) {
	var tz string
	err := tx.QueryRow(ctx, `SELECT time_zone FROM tenants WHERE id = $1`, tenantID).Scan(&tz)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("tenantsettings: read time zone: %w", err)
	}
	return tz, nil
}

// SetTimeZone sets the tenant's time zone after CheckTimeZone, and audits the
// change. Setting the value it already has writes nothing and no audit entry.
// It returns the value now stored.
func (s Store) SetTimeZone(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, name string, by uuid.UUID) (string, error) {
	if by == uuid.Nil {
		return "", ErrNoActor
	}
	if err := CheckTimeZone(name); err != nil {
		return "", err
	}
	before, err := s.TimeZone(ctx, tx, tenantID)
	if err != nil {
		return "", err
	}
	if before == name {
		return name, nil
	}
	tag, err := tx.Exec(ctx, `UPDATE tenants SET time_zone = $2 WHERE id = $1`, tenantID, name)
	if err != nil {
		return "", fmt.Errorf("tenantsettings: set time zone: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return "", ErrNotFound
	}
	if err := auditTimeZone(ctx, tx, tenantID, before, name, by); err != nil {
		return "", err
	}
	return name, nil
}

// auditTimeZone writes the config update entry of a time-zone change. The
// diff is {"time_zone": before} to {"time_zone": after}; content_sha256 (R3)
// hashes the after document. No version moves, so both version fields are
// NULL. Actor and source come from the resolved principal.
func auditTimeZone(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, before, after string, by uuid.UUID) error {
	doc := func(tz string) (json.RawMessage, error) {
		b, err := json.Marshal(map[string]string{"time_zone": tz})
		return b, err
	}
	b, err := doc(before)
	if err != nil {
		return fmt.Errorf("tenantsettings: encode: %w", err)
	}
	a, err := doc(after)
	if err != nil {
		return fmt.Errorf("tenantsettings: encode: %w", err)
	}
	diff, err := jsondiff.Diff(b, a)
	if err != nil {
		return fmt.Errorf("tenantsettings: diff time zone: %w", err)
	}
	sum := sha256.Sum256(a)
	e := audit.Entry{
		Domain:        audit.DomainConfig,
		Action:        audit.ActionUpdate,
		SubjectType:   audit.SubjectTenantSetting,
		SubjectID:     tenantID,
		Actor:         audit.Actor{PrincipalID: &by},
		Source:        audit.SourceUser,
		ContentSHA256: &sum,
		Diff:          diff,
	}
	if id := requestid.FromContext(ctx); id != "" {
		e.RequestID = &id
	}
	if _, err := audit.Append(ctx, tx, e); err != nil {
		return fmt.Errorf("tenantsettings: audit time zone: %w", err)
	}
	return nil
}
