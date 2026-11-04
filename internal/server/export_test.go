package server

// OperatorPrefix is the prefix routeAuth sends to the operator authentication,
// so the write-chain tripwire classifies routes the way the handler does.
const OperatorPrefix = operatorPrefix

// WithHistoryCap sets the record-history cap and a hook that receives the
// number of audit rows each history read returned (P5-17-fu2, TICK-542).
func WithHistoryCap(limit int, rows func(int)) Option {
	return func(o *options) { o.historyCap, o.historyRows = limit, rows }
}
