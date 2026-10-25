// Package binarywrite holds calls named Write and Body that are not on a
// *problem.Registry, in a file that imports problem; the guard must pass them.
package binarywrite

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

var _ = problem.NotFound

type notRegistry struct{}

func (notRegistry) Write(a, b, c any)  {}
func (notRegistry) Body(a, b any) bool { return true }

func Encode(w io.Writer, v uint32) error {
	var n notRegistry
	n.Write(1, 2, 3)
	_ = n.Body(1, 2)
	_, _ = os.Stdout.Write([]byte("x"))
	return binary.Write(w, binary.BigEndian, v)
}
