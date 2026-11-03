package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/secrets"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

// The reasons of the two cascade entries' /cascade object (ADR-004-E1 R2a).
const (
	reasonCycle   = "ErrCascadeCycle"
	reasonTooDeep = "ErrCascadeTooDeep"
)

// recordEntry is the record-domain audit entry of one write (ADR-004 decision
// 1, PLAN-005 P5-13b): subject the record, its entity type, the versions, the
// writer and the cascade the write belongs to.
type recordEntry struct {
	id            uuid.UUID
	action        audit.Action
	rec           *store.Record
	versionBefore *int64
	// before and after are the payloads as PostgreSQL returns them
	// (store.PayloadJSON); before is nil on a create.
	before, after json.RawMessage
	secretProps   []string
}

// appendRecordEntry writes e for the command through audit.Append, in tx.
// After any Append error tx is aborted: the caller rolls it back and returns
// the error, and never retries in tx (P5-12b contract). A secret in the diff
// is refused as *ErrValidation with the code audit_secret_in_diff, naming the
// pointer and never the value (ADR-004 R6).
func appendRecordEntry(ctx context.Context, tx pgx.Tx, cmd Command, e recordEntry) error {
	diff, err := jsondiff.Diff(e.before, e.after)
	if err != nil {
		return fmt.Errorf("pipeline: audit diff: %w", err)
	}
	defID := e.rec.EntityDefinitionID
	va := e.rec.Version
	entry := audit.Entry{
		ID:                 e.id,
		Domain:             audit.DomainRecord,
		Action:             e.action,
		SubjectType:        audit.SubjectRecord,
		SubjectID:          e.rec.ID,
		EntityDefinitionID: &defID,
		VersionBefore:      e.versionBefore,
		VersionAfter:       &va,
		Actor:              actor(cmd.Actor.PrincipalID, cmd.Source),
		Source:             audit.Source(cmd.Source),
		SourceRef:          cmd.SourceRef,
		CausationID:        e.rec.CausationOriginID,
		CausationDepth:     e.rec.CausationDepth,
		RequestID:          requestID(ctx),
		Diff:               diff,
		SecretPaths:        secretPaths(e.secretProps),
	}
	if _, err := audit.Append(ctx, tx, entry); err != nil {
		return mapAppendError(err)
	}
	return nil
}

// AuditSuppressed writes the cascade_cycle entry of a start the cascade key
// suppressed (A5-4, ADR-004-E1 R2a), through audit.Append in tx: the
// triggering write's transaction, so it commits with the write and rolls
// back with it. One entry per suppressed attempt; the EventSink claims each
// definition at most once per write and deduplicates before calling it, as a
// second entry for the same (O, R, D) on one write is
// audit.ErrDuplicateEntry and aborts tx (ADR-004-E1, PG P5). c must be the
// unclaimed result of cascade.ClaimStart; ev is the write the EventSink was
// told about, and names the writer. The entry lives here, not in
// internal/records/cascade, because the pipeline writes the record domain
// (D-3-6).
//
// The entry's subject is the cascade key K(T, O, R, D); its version_after is
// R's version after the write, and its diff one add of /cascade with the
// origin, record, definition and reason, never record data.
func AuditSuppressed(ctx context.Context, tx pgx.Tx, c cascade.Claim, ev Event) error {
	if c.Claimed() || c.RecordID() == uuid.Nil {
		return fmt.Errorf("%w: only an unclaimed start is audited as suppressed", cascade.ErrInvalidKey)
	}
	origin, depth := c.Causation()
	diff, err := cascadeDiff(map[string]any{
		"origin_id":     origin.String(),
		"record_id":     c.RecordID().String(),
		"definition_id": c.DefinitionID().String(),
		"reason":        reasonCycle,
	})
	if err != nil {
		return err
	}
	defID, va := c.EntityDefinitionID(), c.Version()
	_, err = audit.Append(ctx, tx, audit.Entry{
		Domain:             audit.DomainRecord,
		Action:             audit.ActionCascadeCycle,
		SubjectType:        audit.SubjectCascadeKey,
		SubjectID:          audit.CascadeKey(c.TenantID(), origin, c.RecordID(), c.DefinitionID()),
		EntityDefinitionID: &defID,
		VersionAfter:       &va,
		Actor:              actor(ev.ActorID, ev.Source),
		Source:             audit.Source(ev.Source),
		SourceRef:          ev.SourceRef,
		CausationID:        &origin,
		CausationDepth:     depth,
		RequestID:          requestID(ctx),
		Diff:               diff,
	})
	if err != nil {
		return fmt.Errorf("pipeline: audit the suppressed start: %w", err)
	}
	return nil
}

// appendTooDeep writes the cascade_too_deep entry of a write refused for its
// depth (PLAN-003 D-3-7a, ADR-004-E1 R2a), in tx: the TxRunner's own
// transaction, one entry per runner transaction. Its subject is the record
// when tx reads it under RLS (an update of a record the tenant has), else the
// cascade key K(T, O, nil, nil) and the diff names no record.
func appendTooDeep(ctx context.Context, tx pgx.Tx, tenant uuid.UUID, cmd Command) error {
	origin := cmd.Causation.OriginID
	subjectType, subjectID := audit.SubjectCascadeKey, audit.CascadeKey(tenant, origin, uuid.Nil, uuid.Nil)
	detail := map[string]any{
		"origin_id": origin.String(),
		"depth":     cmd.Causation.Depth,
		"limit":     config.CascadeDepthLimit,
		"reason":    reasonTooDeep,
	}
	var defID *uuid.UUID
	var versionBefore *int64
	if cmd.Op == OpUpdate {
		v := cmd.ExpectedVersion
		versionBefore = &v
		switch rec, err := store.Get(ctx, tx, tenant, cmd.RecordID); {
		case err == nil:
			subjectType, subjectID = audit.SubjectRecord, rec.ID
			detail["record_id"] = rec.ID.String()
			id := rec.EntityDefinitionID
			defID = &id
		case !errors.Is(err, store.ErrNotFound):
			return err
		}
	}
	if defID == nil {
		switch id, err := store.DefinitionID(ctx, tx, tenant, cmd.EntityKey); {
		case err == nil:
			defID = &id
		case !errors.Is(err, store.ErrNotFound):
			return err
		}
	}
	diff, err := cascadeDiff(detail)
	if err != nil {
		return err
	}
	_, err = audit.Append(ctx, tx, audit.Entry{
		Domain:             audit.DomainRecord,
		Action:             audit.ActionCascadeTooDeep,
		SubjectType:        subjectType,
		SubjectID:          subjectID,
		EntityDefinitionID: defID,
		VersionBefore:      versionBefore,
		Actor:              actor(cmd.Actor.PrincipalID, cmd.Source),
		Source:             audit.Source(cmd.Source),
		SourceRef:          cmd.SourceRef,
		CausationID:        &origin,
		CausationDepth:     cmd.Causation.Depth,
		RequestID:          requestID(ctx),
		Diff:               diff,
	})
	return err
}

// cascadeDiff is the diff of a cascade entry: one add of /cascade whose value
// is v (ids, numbers and a reason; never record data). jsondiff writes it,
// so its bytes are canonical.
func cascadeDiff(v map[string]any) ([]byte, error) {
	after, err := json.Marshal(map[string]any{"cascade": v})
	if err != nil {
		return nil, fmt.Errorf("pipeline: encode cascade diff: %w", err)
	}
	diff, err := jsondiff.Diff(json.RawMessage(`{}`), after)
	if err != nil {
		return nil, fmt.Errorf("pipeline: cascade diff: %w", err)
	}
	return diff, nil
}

// actor is the entry's actor: the principal, or for a system writer with
// none the system actor named after its source (ADR-004 R1: exactly one).
func actor(principal uuid.UUID, source Source) audit.Actor {
	if principal != uuid.Nil {
		return audit.Actor{PrincipalID: &principal}
	}
	s := string(source)
	return audit.Actor{System: &s}
}

func requestID(ctx context.Context) *string {
	if id := requestid.FromContext(ctx); id != "" {
		return &id
	}
	return nil
}

// secretPaths are the payload pointers of the secret properties: "/<name>"
// each, as the markers are top-level only (metadata.Compiled.SecretProperties).
func secretPaths(props []string) []patchwalk.Pointer {
	out := make([]patchwalk.Pointer, 0, len(props))
	for _, p := range props {
		out = append(out, patchwalk.FromSegments([]string{p}))
	}
	return out
}

// unionSorted merges two sorted name lists.
func unionSorted(a, b []string) []string {
	out := append(slices.Clone(a), b...)
	slices.Sort(out)
	return slices.Compact(out)
}

// mapAppendError maps an audit.Append error of a record entry: a secret is
// refused as invalid input, anything else is wrapped unchanged
// (audit.ErrDuplicateEntry and database errors abort the transaction).
func mapAppendError(err error) error {
	var sec *secrets.ErrSecretInDiff
	if errors.As(err, &sec) {
		return &ErrValidation{Fields: []metadata.FieldError{{Path: string(sec.Path), Code: secrets.Code}}}
	}
	return fmt.Errorf("pipeline: audit: %w", err)
}
