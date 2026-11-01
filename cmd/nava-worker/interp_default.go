//go:build !interptest

package main

import (
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// interpreterActivities is nil in a production build: the interpreter's real
// DefinitionStore, RecordReader and RecordWriter do not exist yet (P3-41,
// P3-43), and the file-backed fakes are only in an interptest build
// (interp_interptest.go). TestProductionBuildLinksNoFakes keeps it so.
// TODO(TODO-TICK-367-01): wire the real ports here when P3-41 and P3-43 land.
func interpreterActivities(config.LookupFunc) (*interp.Activities, error) {
	return nil, nil //nolint:nilnil // nil Activities is "not registered", not an error
}
