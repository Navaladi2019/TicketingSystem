// Package logging builds the process slog.Logger from configuration.
package logging

import (
	"io"
	"log/slog"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// New returns a structured logger writing to w. JSON is the container
// format; text is for local reading.
func New(w io.Writer, cfg config.Log) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.Level}
	if cfg.Format == config.FormatText {
		return slog.New(slog.NewTextHandler(w, opts))
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}
