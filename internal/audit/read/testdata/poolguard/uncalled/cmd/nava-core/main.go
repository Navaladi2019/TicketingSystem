package main

import "github.com/Navaladi2019/TicketingSystem/internal/audit/read"

// The one reference in main.go must be a call.
var open = read.OpenPool // want: openpool
