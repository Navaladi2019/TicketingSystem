// Command devcerts writes throwaway development certificates for the custom
// Temporal server (see internal/devcerts). Development only.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Navaladi2019/TicketingSystem/temporal-server/internal/devcerts"
)

func main() {
	out := flag.String("out", ".devcerts", "output directory")
	validFor := flag.Duration("valid-for", 30*24*time.Hour, "certificate lifetime")
	flag.Parse()
	if err := devcerts.Write(*out, *validFor); err != nil {
		fmt.Fprintln(os.Stderr, "devcerts:", err)
		os.Exit(1)
	}
	fmt.Println("wrote development certificates to", *out)
}
