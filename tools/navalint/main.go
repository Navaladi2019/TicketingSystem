// Command navalint runs the platform's analyzers as a go vet tool:
//
//	go vet -vettool=bin/navalint ./...
//
// `make lint` builds and runs it: setconfig (T07) and idpclaims (T15, CI-11).
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/Navaladi2019/TicketingSystem/tools/navalint/idpclaims"
	"github.com/Navaladi2019/TicketingSystem/tools/navalint/setconfig"
)

func main() {
	unitchecker.Main(setconfig.Analyzer, idpclaims.Analyzer)
}
