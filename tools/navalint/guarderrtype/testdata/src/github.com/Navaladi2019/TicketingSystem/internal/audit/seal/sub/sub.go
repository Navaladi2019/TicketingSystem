package sub

import "github.com/Navaladi2019/TicketingSystem/internal/audit/seal"

// A package below internal/audit/seal is outside it.
var t = seal.ErrTypeUnsealable // want `seal.ErrTypeUnsealable used outside internal/workerguard and internal/audit/seal`
