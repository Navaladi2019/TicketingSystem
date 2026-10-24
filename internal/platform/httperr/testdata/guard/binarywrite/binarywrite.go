// Package binarywrite holds calls named Write and Body that are not on a
// *problem.Registry; the guard must ignore them.
package binarywrite

import (
	"encoding/binary"
	"io"
)

type notRegistry struct{}

func (notRegistry) Write(a, b, c any)  {}
func (notRegistry) Body(a, b any) bool { return true }

func Encode(w io.Writer, v uint32) error {
	var n notRegistry
	n.Write(1, 2, 3)
	_ = n.Body(1, 2)
	return binary.Write(w, binary.BigEndian, v)
}
