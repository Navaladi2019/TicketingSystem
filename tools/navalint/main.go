// Command navalint runs the platform's analyzers as a go vet tool:
//
//	go vet -vettool=bin/navalint ./...
//
// `make lint` builds and runs it: setconfig (T07), idpclaims (T15, CI-11),
// guarderrtype (AR-044, P3-26, TICK-594), tenancyconn (TICK-698) and
// uuidv7 (TICK-963).
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/Navaladi2019/TicketingSystem/tools/navalint/guarderrtype"
	"github.com/Navaladi2019/TicketingSystem/tools/navalint/idpclaims"
	"github.com/Navaladi2019/TicketingSystem/tools/navalint/setconfig"
	"github.com/Navaladi2019/TicketingSystem/tools/navalint/tenancyconn"
	"github.com/Navaladi2019/TicketingSystem/tools/navalint/uuidv7"
)

func main() {
	unitchecker.Main(setconfig.Analyzer, idpclaims.Analyzer, guarderrtype.Analyzer, tenancyconn.Analyzer, uuidv7.Analyzer)
}
