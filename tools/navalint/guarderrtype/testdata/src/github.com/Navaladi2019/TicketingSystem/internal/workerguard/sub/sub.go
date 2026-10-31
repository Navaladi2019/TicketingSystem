package sub

import "github.com/Navaladi2019/TicketingSystem/internal/workerguard"

var t = workerguard.ErrTypeTenantGuard // want `workerguard.ErrTypeTenantGuard used outside internal/workerguard`
