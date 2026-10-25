// Package logging builds the process slog.Logger from configuration.
package logging

import (
	"context"
	"io"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
)

// New returns a structured logger writing to w. JSON is the container
// format; text is for local reading. Every record logged through a *Context
// method carries the request's trace_id, span_id and request_id, and, after
// authentication, its tenant_id and principal_id.
func New(w io.Writer, cfg config.Log) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.Level}
	if cfg.Format == config.FormatText {
		return slog.New(NewContextHandler(slog.NewTextHandler(w, opts)))
	}
	return slog.New(NewContextHandler(slog.NewJSONHandler(w, opts)))
}

// ContextHandler adds request correlation attributes from the record's
// context: trace_id and span_id from the active span, request_id, and the
// resolved tenant_id and principal_id (TICK-93). The two ids are log fields
// only, never metric attributes.
type ContextHandler struct {
	next slog.Handler
}

// NewContextHandler wraps next.
func NewContextHandler(next slog.Handler) *ContextHandler {
	return &ContextHandler{next: next}
}

// Enabled implements slog.Handler.
func (h *ContextHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.next.Enabled(ctx, l)
}

// Handle implements slog.Handler.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	if id := requestid.FromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	addPrincipal(ctx, &r)
	return h.next.Handle(ctx, r)
}

// addPrincipal adds the request's resolved tenant and principal ids, if any.
func addPrincipal(ctx context.Context, r *slog.Record) {
	if p, ok := identity.FromContext(ctx); ok {
		r.AddAttrs(slog.String("tenant_id", p.TenantID.String()), slog.String("principal_id", p.PrincipalID.String()))
	}
}

// WithAttrs implements slog.Handler.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{next: h.next.WithAttrs(attrs)}
}

// WithGroup implements slog.Handler. Correlation attributes added after a
// group is opened land inside that group (x.trace_id), which breaks log
// queries that join on trace_id. Do not call WithGroup on a logger used in
// request scope; group with slog.Group attributes instead. Revisit in T17.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{next: h.next.WithGroup(name)}
}

// WithExport returns a logger that also sends every record to export, the
// OTLP bridge, beside the handler of base. A nil export returns base itself,
// so stdout output is untouched when log export is off. The bridge takes the
// trace and span id from the record's context; request_id is added here
// because the bridge does not know it. The bridge enables every level, so
// export is gated to level, the configured NAVA_LOG_LEVEL. trace_id and
// span_id are not added as attributes: the exported record carries them as
// fields.
func WithExport(base *slog.Logger, export slog.Handler, level slog.Leveler) *slog.Logger {
	if export == nil {
		return base
	}
	return slog.New(slog.NewMultiHandler(base.Handler(), requestIDHandler{next: levelHandler{next: export, level: level}}))
}

// levelHandler enables a record only if it is at or above level.
type levelHandler struct {
	next  slog.Handler
	level slog.Leveler
}

func (h levelHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return l >= h.level.Level() && h.next.Enabled(ctx, l)
}

func (h levelHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.next.Handle(ctx, r)
}

func (h levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return levelHandler{next: h.next.WithAttrs(attrs), level: h.level}
}

func (h levelHandler) WithGroup(name string) slog.Handler {
	return levelHandler{next: h.next.WithGroup(name), level: h.level}
}

// requestIDHandler adds the request id, tenant id and principal id from the
// record's context.
type requestIDHandler struct {
	next slog.Handler
}

func (h requestIDHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.next.Enabled(ctx, l)
}

func (h requestIDHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := requestid.FromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	addPrincipal(ctx, &r)
	return h.next.Handle(ctx, r)
}

func (h requestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return requestIDHandler{next: h.next.WithAttrs(attrs)}
}

func (h requestIDHandler) WithGroup(name string) slog.Handler {
	return requestIDHandler{next: h.next.WithGroup(name)}
}
