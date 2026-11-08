package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/export"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// ExportAuditEntries serves GET /api/v1/audit/entries:export (ADR-004 R19-R21,
// P5-40). The audit.export gate, the export lease and the audit/export entry
// run in one nava_app transaction that commits before the first byte (R19,
// A1-7); the pages then stream from the reader pool, each its own short
// transaction.
func (h *handlers) ExportAuditEntries(ctx context.Context, req api.ExportAuditEntriesRequestObject) (api.ExportAuditEntriesResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil || h.exporter == nil {
		return nil, notImplemented()
	}
	prm := req.Params
	r := export.Request{Stubs: true}
	if prm.Stubs != nil {
		r.Stubs = *prm.Stubs
	}
	if prm.Format != nil {
		r.CSV = *prm.Format == api.Csv
	}
	f := &r.Filter
	f.From, f.To, f.AfterSeq, f.SubjectID, f.ChangeSetID = prm.From, prm.To, prm.AfterSeq, prm.SubjectId, prm.ChangeSetId
	if prm.Domain != nil {
		d := string(*prm.Domain)
		f.Domain = &d
	}
	if prm.SubjectType != nil {
		s := string(*prm.SubjectType)
		f.SubjectType = &s
	}

	var s export.Session
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		// The gate is first, so a caller without audit.export learns nothing,
		// not even whether its filter was well formed.
		if err := authz.RequireAuditPermission(ctx, tx, p, authz.PermAuditExport); err != nil {
			return err
		}
		if f.From != nil && f.To != nil && !f.From.Before(*f.To) {
			return &problemError{problem.ValidationFailed, []problem.FieldError{{Pointer: "/query/to", Rule: "range"}}, errors.New("from is not before to")}
		}
		var err error
		s, err = export.Begin(ctx, tx, p.TenantID, p.PrincipalID, r, h.exporter.Deadline())
		return err
	})
	var pe *problemError
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return nil, &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, export.ErrInProgress):
		return nil, &problemError{problem.ExportInProgress, nil, err}
	case errors.As(err, &pe):
		return nil, pe
	case err != nil:
		return nil, err
	}
	return &exportResponse{ctx: ctx, h: h, s: s}, nil
}

// exportResponse streams the export. The lease and the audit entry are
// already committed when it exists.
type exportResponse struct {
	ctx context.Context //nolint:containedctx // the request's context, carried to the streaming Visit
	h   *handlers
	s   export.Session
}

// VisitExportAuditEntriesResponse streams the file, then releases the lease.
// A failure after the first byte cannot change the status, so the connection
// is aborted: the client sees a broken transfer, and the file has no trailer.
func (e *exportResponse) VisitExportAuditEntriesResponse(w http.ResponseWriter) error {
	defer func() {
		if err := export.Release(e.ctx, e.h.pool, e.s.TenantID); err != nil {
			e.h.log.WarnContext(e.ctx, "audit export lease not released", e.attrs(slog.Any("error", err))...)
		}
	}()
	rc := http.NewResponseController(w)
	// The server's WriteTimeout would cut a long export; R21 caps it instead.
	_ = rc.SetWriteDeadline(time.Now().Add(e.h.exporter.Deadline() + time.Minute))
	ctype, ext, write := "application/x-ndjson", "jsonl", e.h.exporter.Write
	if e.s.Request.CSV {
		ctype, ext, write = "text/csv; charset=utf-8", "csv", e.h.exporter.WriteCSV
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Disposition", `attachment; filename="audit-export-`+e.s.ExportID.String()+`.`+ext+`"`)
	w.WriteHeader(http.StatusOK)
	flush := func() { _ = rc.Flush() }
	if err := write(e.ctx, w, flush, e.s); err != nil {
		e.h.log.ErrorContext(e.ctx, "audit export failed", e.attrs(slog.Any("error", err))...)
		panic(http.ErrAbortHandler)
	}
	return nil
}

func (e *exportResponse) attrs(extra ...any) []any {
	return append([]any{slog.String("tenant_id", e.s.TenantID.String()), slog.String("export_id", e.s.ExportID.String())}, extra...)
}

// newExporter builds the exporter over the reader; opts carries the test
// overrides.
func newExporter(src *read.Reader, o options) *export.Exporter {
	return &export.Exporter{Source: src, PageSize: o.exportPageSize, OnPage: o.exportOnPage}
}
