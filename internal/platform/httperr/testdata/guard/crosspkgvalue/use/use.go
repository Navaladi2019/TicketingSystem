// Package use does not import problem. It takes Body from a registry another
// package returns and gives it that package's 5xx code; the guard must fail
// the method value.
package use

import (
	"context"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/httperr/testdata/guard/crosspkgvalue/server"
)

func Handler(ctx context.Context) int {
	f := server.Reg().Body
	return f(ctx, server.Code).Status
}
