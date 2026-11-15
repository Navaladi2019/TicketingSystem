//go:build !interptest

package main

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// interpreterActivities is the production interpreter (productionActivities,
// interp_ports.go); the file-backed fakes stay in an interptest build
// (interp_interptest.go; TestProductionBuildLinksNoFakes keeps it so).
//
// With workflows off (WorkflowsEnv=off, the activity-only worker) it returns
// nil, nil: that worker polls no workflow tasks and registers no interpreter,
// and must still start. The interptest build errors instead (newWorker).
func interpreterActivities(_ config.LookupFunc, pool, pipePool *pgxpool.Pool, workflows bool) (*interp.Activities, error) {
	if !workflows {
		return nil, nil //nolint:nilnil // nil Activities is "not registered", not an error
	}
	return productionActivities(pool, pipePool)
}
