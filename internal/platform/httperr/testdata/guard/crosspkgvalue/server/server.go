// Package server returns a registry and a 5xx code to other packages.
package server

import "github.com/Navaladi2019/TicketingSystem/internal/platform/problem"

const Code = problem.InternalError

func Reg() *problem.Registry { return nil }
