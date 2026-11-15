package main

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
	"github.com/Navaladi2019/TicketingSystem/internal/workflowdefs"
)

// productionActivities is the production interpreter: the DefinitionStore is
// the database one (P3-41, workflowdefs.Reader), the RecordReader and
// RecordWriter are recordports over pool (P3-43; the writer goes through the
// records pipeline as source=workflow, its TxRunner on pipePool). Every port
// reaches the database through workerguard.InTenantTx.
//
// pipePool is the pipeline's own pool: it opens a second connection while the
// write's transaction holds one of pool's (the cascade_too_deep audit entry
// is written on it so it survives the write's rollback), so on pool itself a
// burst of writes could hold every connection and wait for a second one
// (TICK-88 security review, availability).
//
// mp is the worker's meter provider, for the pipeline's uncoerced-value
// counter (ADR-006 R5); nil counts none.
//
// The production build always uses it (interp_default.go). An interptest
// build uses it only with NAVA_INTERPTEST_PORTS=db, for the P3-44 crash case
// against the real pipeline.Write (interp_interptest.go).
func productionActivities(pool, pipePool *pgxpool.Pool, mp metric.MeterProvider) (*interp.Activities, error) {
	if pool == nil || pipePool == nil {
		return nil, errors.New("config: the interpreter needs the database pool and the pipeline pool")
	}
	writer, err := recordports.NewWriter(pool, pipePool, mp)
	if err != nil {
		return nil, err
	}
	return &interp.Activities{
		Definitions: workflowdefs.Reader{Pool: pool},
		Records:     recordports.Reader{Pool: pool},
		Writer:      writer,
		Waits:       workerguard.WaitTracker{Pool: pool},
		Causes:      workerguard.CauseTracker{Pool: pool},
	}, nil
}
