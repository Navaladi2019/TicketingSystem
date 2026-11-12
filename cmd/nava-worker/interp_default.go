//go:build !interptest

package main

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
	"github.com/Navaladi2019/TicketingSystem/internal/workflowdefs"
)

// interpreterActivities is the production interpreter: the DefinitionStore is
// the database one (P3-41, workflowdefs.Reader), the RecordReader and
// RecordWriter are recordports over pool (P3-43; the writer goes through the
// records pipeline as source=workflow, its TxRunner on pipePool), and the
// file-backed fakes stay in an interptest build (interp_interptest.go;
// TestProductionBuildLinksNoFakes keeps it so). Every port reaches the
// database through workerguard.InTenantTx.
//
// pipePool is the pipeline's own pool: it opens a second connection while the
// write's transaction holds one of pool's (the cascade_too_deep audit entry
// is written on it so it survives the write's rollback), so on pool itself a
// burst of writes could hold every connection and wait for a second one
// (TICK-88 security review, availability).
//
// With workflows off (WorkflowsEnv=off, the activity-only worker) it returns
// nil, nil: that worker polls no workflow tasks and registers no interpreter,
// and must still start. The interptest build errors instead (newWorker).
func interpreterActivities(_ config.LookupFunc, pool, pipePool *pgxpool.Pool, workflows bool) (*interp.Activities, error) {
	if !workflows {
		return nil, nil //nolint:nilnil // nil Activities is "not registered", not an error
	}
	if pool == nil || pipePool == nil {
		return nil, errors.New("config: the interpreter needs the database pool and the pipeline pool")
	}
	return &interp.Activities{
		Definitions: workflowdefs.Reader{Pool: pool},
		Records:     recordports.Reader{Pool: pool},
		Writer:      recordports.NewWriter(pool, pipePool),
		Waits:       workerguard.WaitTracker{Pool: pool},
	}, nil
}
