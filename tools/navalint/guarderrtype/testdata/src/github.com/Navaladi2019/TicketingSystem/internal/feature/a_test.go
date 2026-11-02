package feature

import "github.com/Navaladi2019/TicketingSystem/internal/workerguard"

// Tests assert the guard's error type; they never build into a worker.
func isGuardInTest(typ string) bool { return typ == workerguard.ErrTypeTenantGuard }
