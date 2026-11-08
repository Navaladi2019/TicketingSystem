// Package outbox is the cross-package half of the TICK-581 fixture: it runs
// an exported SQL constant of the (fixture) seal package, which is allowed
// only in seal.go.
package outbox

import (
	"context"

	sealer "github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
)

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) error
}

func run(ctx context.Context, tx execer) error {
	if err := tx.Exec(ctx, sealer.SealSQL); err != nil { // want: update audit_log
		return err
	}
	return tx.Exec(ctx, sealer.SealSQL+" RETURNING id") // want: update audit_log
}
