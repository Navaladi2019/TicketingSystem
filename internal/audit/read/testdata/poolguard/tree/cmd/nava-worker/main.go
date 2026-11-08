package main

import "github.com/Navaladi2019/TicketingSystem/internal/audit/read"

// A reference that is not a call still counts.
var open = read.OpenPool // want: openpool
