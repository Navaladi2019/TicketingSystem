package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const listAuditEntriesRoute = "listAuditEntries"

// defaultAuditLimit is the page size when `limit` is absent; the contract
// caps it at 500.
const defaultAuditLimit = 100

// ListAuditEntries serves GET /api/v1/audit/entries (ADR-004 R7, R8). The
// audit.read gate is the first thing in the transaction, before the page
// token is looked at, so a caller without the permission learns nothing about
// the log, not even whether a token was well formed.
func (h *handlers) ListAuditEntries(ctx context.Context, req api.ListAuditEntriesRequestObject) (api.ListAuditEntriesResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil || h.cursors == nil || h.audit == nil || h.acl == nil {
		return nil, notImplemented()
	}
	prm := req.Params
	limit := defaultAuditLimit
	if prm.Limit != nil {
		limit = *prm.Limit
	}
	var f read.Filter
	q := url.Values{}
	if prm.From != nil {
		f.From = prm.From
		q.Set("from", prm.From.UTC().Format(time.RFC3339Nano))
	}
	if prm.To != nil {
		f.To = prm.To
		q.Set("to", prm.To.UTC().Format(time.RFC3339Nano))
	}
	if prm.Domain != nil {
		d := string(*prm.Domain)
		f.Domain = &d
		q.Set("domain", d)
	}
	if prm.SubjectType != nil {
		s := string(*prm.SubjectType)
		f.SubjectType = &s
		q.Set("subject_type", s)
	}
	if prm.SubjectId != nil {
		id := *prm.SubjectId
		f.SubjectID = &id
		q.Set("subject_id", id.String())
	}
	if prm.AfterSeq != nil {
		f.AfterSeq = prm.AfterSeq
		q.Set("after_seq", strconv.FormatInt(*prm.AfterSeq, 10))
	}
	claims, err := cursor.ClaimsFor(ctx, listAuditEntriesRoute, q)
	if err != nil {
		return nil, err
	}
	// Two transactions (ADR-004-A1 A1-7): the gate and the cursor on
	// nava_app, then the page on the reader's own pool, which alone may read
	// the diff and the salt, with the masker in the same transaction.
	// The caller's field masks (P3-33, PR #523 finding 4) are read here too,
	// on nava_app: the timeline is not ACL-filtered (audit.read), but a field
	// the record ACL hides from the caller is masked in every record entry's
	// diff, as a secret is.
	var (
		after *read.Position
		masks map[uuid.UUID][]string
	)
	err = tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := authz.RequireAuditPermission(ctx, tx, p, authz.PermAuditRead); err != nil {
			return err
		}
		var err error
		if masks, err = h.acl.ReadMasks(ctx, tx, p); err != nil {
			return err
		}
		if err := addOrgUnitMasks(ctx, tx, p.TenantID, masks); err != nil {
			return err
		}
		if prm.PageToken != nil {
			pos, err := h.cursors.Decode(claims, *prm.PageToken)
			if err != nil {
				return &problemError{problem.InvalidCursor, nil, err}
			}
			at, id, err := decodeHistoryPos(pos)
			if err != nil {
				return &problemError{problem.InvalidCursor, nil, err}
			}
			after = &read.Position{At: at, ID: id}
		}
		return nil
	})
	var pe *problemError
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return nil, &problemError{problem.Forbidden, nil, err}
	case errors.As(err, &pe):
		return nil, pe
	case err != nil:
		return nil, err
	}
	entries, err := h.audit.WithFieldMasks(masks).List(ctx, p.TenantID, f, after, limit+1)
	switch {
	case errors.Is(err, read.ErrNoDefinitionVersions):
		return nil, h.maskerFailed(ctx, err)
	case err != nil:
		return nil, err
	}
	out := api.AuditEntryList{Items: make([]api.AuditEntry, 0, min(len(entries), limit))}
	for _, e := range entries[:min(len(entries), limit)] {
		out.Items = append(out.Items, auditEntryBody(e))
	}
	if len(entries) > limit {
		last := entries[limit-1]
		next, err := h.cursors.Encode(claims, encodeHistoryPos(last.OccurredAt, last.ID))
		if err != nil {
			return nil, err
		}
		out.NextPageToken = &next
	}
	return api.ListAuditEntries200JSONResponse(out), nil
}

func auditEntryBody(e read.Entry) api.AuditEntry {
	b := api.AuditEntry{
		Id: e.ID, OccurredAt: e.OccurredAt.UTC(), Domain: api.AuditEntryDomain(e.Domain), Action: e.Action,
		SubjectType: e.SubjectType, SubjectId: e.SubjectID, EntityDefinitionId: e.EntityDefinitionID,
		VersionBefore: e.VersionBefore, VersionAfter: e.VersionAfter, ActorPrincipalId: e.ActorPrincipalID,
		ActorSystem: e.ActorSystem, Source: e.Source, SourceRef: e.SourceRef, ChangeSetId: e.ChangeSetID,
		CausationId: e.CausationID, CausationDepth: int(e.CausationDepth), RequestId: e.RequestID,
		Backfilled: e.Backfilled, DiffTruncated: e.DiffTruncated, DiffNulEscaped: e.DiffNULEscaped, DiffRedacted: e.DiffRedacted,
		Seq: e.Seq, SealedAt: e.SealedAt, ErasedAt: e.ErasedAt,
	}
	b.ContentSha256 = hexOf(e.ContentSHA256)
	b.DiffSha256 = hexOf(e.DiffSHA256)
	b.DiffSalt = hexOf(e.DiffSalt)
	if e.Diff != nil {
		d := e.Diff
		b.Diff = &d
	}
	return b
}

func hexOf(b []byte) *string {
	if len(b) == 0 {
		return nil
	}
	s := hex.EncodeToString(b)
	return &s
}

// maskerFailed maps a masker that found no visible version of an entry's
// definition to 500 internal (ADR-004-A1 A1-4). WriteError logs err once; its
// text names the definition and never holds a diff value, a pointer value or
// the salt.
func (h *handlers) maskerFailed(_ context.Context, err error) error {
	return &problemError{problem.InternalError, nil, err}
}

const fixedColumnsSQL = `
SELECT definition_id, fixed_columns
FROM entity_definition_versions
WHERE tenant_id = $1 AND definition_id = ANY($2::uuid[])`

// addOrgUnitMasks appends the org unit column to the masks of every
// definition that masks a property mapped to the requester column. The org
// unit is no schema property, so no mask names it, but it derives from the
// requester and must not outlive a requester mask in an audit diff. It reads
// fixed_columns on nava_app, which the audit reader cannot. It fails closed:
// a query or decode error, or a definition with no version row, is an error.
func addOrgUnitMasks(ctx context.Context, tx pgx.Tx, tenant uuid.UUID, masks map[uuid.UUID][]string) error {
	if len(masks) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(masks))
	idText := make([]string, 0, len(masks))
	for id := range masks {
		ids = append(ids, id)
		idText = append(idText, id.String())
	}
	rows, err := tx.Query(ctx, fixedColumnsSQL, tenant, idText)
	if err != nil {
		return fmt.Errorf("audit: read fixed columns: %w", err)
	}
	defer rows.Close()
	seen := make(map[uuid.UUID]bool, len(ids))
	hide := map[uuid.UUID]bool{}
	for rows.Next() {
		var (
			id  uuid.UUID
			raw []byte
		)
		if err := rows.Scan(&id, &raw); err != nil {
			return fmt.Errorf("audit: scan fixed columns: %w", err)
		}
		var fixed map[string]string
		if err := json.Unmarshal(raw, &fixed); err != nil {
			return fmt.Errorf("audit: decode fixed columns of %s: %w", id, err)
		}
		seen[id] = true
		for prop, col := range fixed {
			if col == metadata.ColumnRequesterID && slices.Contains(masks[id], prop) {
				hide[id] = true
			}
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("audit: read fixed columns: %w", err)
	}
	for _, id := range ids {
		if !seen[id] {
			return fmt.Errorf("audit: definition %s has no version row", id)
		}
	}
	for id := range hide {
		masks[id] = append(masks[id], metadata.ColumnOrgUnitID)
	}
	return nil
}
