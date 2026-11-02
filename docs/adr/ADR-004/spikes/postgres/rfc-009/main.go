// Prints the cascade key for the RFC-009 spike vectors, computed the way
// internal/audit would: uuid.NewSHA1(ns, t[:] || o[:] || r[:] || d[:]).
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

func ckey(ns, t, o, r, d uuid.UUID) uuid.UUID {
	name := make([]byte, 0, 64)
	for _, id := range []uuid.UUID{t, o, r, d} {
		name = append(name, id[:]...)
	}
	return uuid.NewSHA1(ns, name)
}

func main() {
	ns := uuid.MustParse(os.Args[1])
	a := os.Args[2:]
	if len(a) == 0 { // stdin: "t o r d [anything]" per line
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			f := strings.Fields(sc.Text())
			a = append(a, f[:4]...)
		}
	}
	for i := 0; i+3 < len(a); i += 4 {
		fmt.Println(ckey(ns, uuid.MustParse(a[i]), uuid.MustParse(a[i+1]), uuid.MustParse(a[i+2]), uuid.MustParse(a[i+3])))
	}
}
