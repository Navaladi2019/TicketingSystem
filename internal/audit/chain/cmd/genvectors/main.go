// Command genvectors rewrites the audit hash-chain golden vectors
// (internal/audit/chain/testdata/vectors.json). Run it from the module root:
//
//	go run ./internal/audit/chain/cmd/genvectors [-o path]
//
// The output is deterministic; TestVectorsFileIsCurrent fails when the
// committed file differs from it.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
)

func main() {
	out := flag.String("o", "internal/audit/chain/testdata/vectors.json", "output path")
	flag.Parse()
	b, err := vectors.Generate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "genvectors:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil { //nolint:gosec // G306: a committed, non-secret test fixture
		fmt.Fprintln(os.Stderr, "genvectors:", err)
		os.Exit(1)
	}
}
