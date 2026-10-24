// Command navalint runs the platform's analyzers as a go vet tool:
//
//	go vet -vettool=bin/navalint ./...
//
// `make lint` builds and runs it. T07 adds setconfig; T15 adds the claim
// analyzers here.
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	"github.com/Navaladi2019/TicketingSystem/tools/navalint/setconfig"
)

func main() {
	unitchecker.Main(setconfig.Analyzer)
}
