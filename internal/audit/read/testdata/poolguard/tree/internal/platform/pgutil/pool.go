package pgutil

import "context"

// OpenPool is unrelated to read.OpenPool.
func OpenPool(ctx context.Context) {}

func again(ctx context.Context) { OpenPool(ctx) }
