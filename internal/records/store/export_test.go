package store

// VisibleCountStatementForTest is the statement VisibleAtMost runs for q and k.
func VisibleCountStatementForTest(q ListQuery, k int) string { return countStatement(q, q.Where, k+1) }
