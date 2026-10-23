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
	installation := flag.Bool("installation", false, "write only the installation CA, internode CA and the five platform certificates (make dev-ca)")
	unlisted := flag.Bool("unlisted", false, "with -installation, also write the "+devcerts.UnlistedCN+" certificate (a CN that is no identity) for the CI-6 suite")
	rogue := flag.Bool("rogue", false, "write only a nava-relay certificate from an unrelated CA, for handshake tests")
	flag.Parse()
	var err error
	switch {
	case *rogue:
		err = devcerts.WriteRogue(*out, *validFor)
	case *installation && *unlisted:
		err = devcerts.WriteInstallationWithUnlisted(*out, *validFor)
	case *installation:
		err = devcerts.WriteInstallation(*out, *validFor)
	default:
		err = devcerts.Write(*out, *validFor)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "devcerts:", err)
		os.Exit(1)
	}
	fmt.Println("wrote development certificates to", *out)
}
