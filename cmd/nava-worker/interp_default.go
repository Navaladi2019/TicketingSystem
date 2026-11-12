//go:build !interptest

package main

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflowdefs"
)

// interpreterActivities is the production interpreter: the DefinitionStore is
// the database one (P3-41, workflowdefs.Reader, in workerguard.InTenantTx), and
// the file-backed fakes stay in an interptest build (interp_interptest.go;
// TestProductionBuildLinksNoFakes keeps it so). The RecordReader and
// RecordWriter do not exist yet (P3-43): until they do, a step that needs one
// fails with a retryable error instead of reading or writing anything.
// TODO(TODO-TICK-367-01): wire the real RecordReader and RecordWriter here when P3-43 lands.
//
// With workflows off (WorkflowsEnv=off, the activity-only worker) it returns
// nil, nil: that worker polls no workflow tasks and registers no interpreter,
// and must still start. The interptest build errors instead (newWorker).
func interpreterActivities(_ config.LookupFunc, pool *pgxpool.Pool, workflows bool) (*interp.Activities, error) {
	if !workflows {
		return nil, nil //nolint:nilnil // nil Activities is "not registered", not an error
	}
	if pool == nil {
		return nil, errors.New("config: the interpreter needs the database pool")
	}
	return &interp.Activities{
		Definitions: workflowdefs.Reader{Pool: pool},
		Records:     noRecords{},
		Writer:      noRecords{},
	}, nil
}

// errNoRecordPorts is what a record port returns before P3-43.
var errNoRecordPorts = errors.New("interp: record ports are not wired yet (P3-43)")

type noRecords struct{}

func (noRecords) Read(context.Context, uuid.UUID, uuid.UUID) (map[string]any, error) {
	return nil, errNoRecordPorts
}

func (noRecords) Write(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, map[string]any) error {
	return errNoRecordPorts
}
