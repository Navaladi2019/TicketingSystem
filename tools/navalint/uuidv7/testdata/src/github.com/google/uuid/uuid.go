// Package uuid is a stub of github.com/google/uuid for analyzer tests.
package uuid

type UUID [16]byte

func New() UUID                      { return UUID{} }
func NewString() string              { return "" }
func NewRandom() (UUID, error)       { return UUID{}, nil }
func NewV7() (UUID, error)           { return UUID{}, nil }
func NewSHA1(ns UUID, b []byte) UUID { return UUID{} }
func Must(u UUID, err error) UUID    { return u }
func MustParse(s string) UUID        { return UUID{} }
