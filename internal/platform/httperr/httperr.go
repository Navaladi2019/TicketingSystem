// Package httperr is the one place a failed request is reported: the error is
// recorded on the active span and logged with its text.
//
// The export filter (telemetry/filter.go) strips exception.message and the
// status description from spans, because error text can carry client values.
// The text therefore survives only in the log line, which the logging handler
// stamps with the trace_id, span_id and request_id of ctx. Handlers do not call
// span.RecordError themselves; a test enforces that.
package httperr

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TypeOf is the bounded error.type for err: its Go type, never its text.
func TypeOf(err error) string {
	return fmt.Sprintf("%T", err)
}

// Report marks the span of ctx as failed with the bounded errType and logs msg
// at ERROR level with err's text under the key "err", plus attrs. It never
// puts err's text on the span or into a response body.
func Report(ctx context.Context, log *slog.Logger, msg, errType string, err error, attrs ...slog.Attr) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("error.type", errType))
	span.RecordError(err)
	span.SetStatus(codes.Error, errType)
	attrs = append([]slog.Attr{slog.Any("err", err)}, attrs...)
	log.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}
