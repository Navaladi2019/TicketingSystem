package server

import (
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
)

// OperatorPrefix is the prefix routeAuth sends to the operator authentication,
// so the write-chain tripwire classifies routes the way the handler does.
const OperatorPrefix = operatorPrefix

// WithHistoryCap sets the record-history cap and a hook that receives the
// number of audit rows each history read returned (P5-17-fu2, TICK-542).
func WithHistoryCap(limit int, rows func(int)) Option {
	return func(o *options) { o.historyCap, o.historyRows = limit, rows }
}

// WithAuditMasker replaces the audit masker, so a test can make it fail.
func WithAuditMasker(m read.Masker) Option { return func(o *options) { o.auditMasker = m } }

// WithExportPaging sets the audit export's page size and a hook called inside
// each page's transaction with its start time and statement_timeout.
func WithExportPaging(pageSize int, onPage func(time.Time, string)) Option {
	return func(o *options) { o.exportPageSize, o.exportOnPage = pageSize, onPage }
}
