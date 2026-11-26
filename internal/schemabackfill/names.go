// Package schemabackfill is the schema.backfill workflow and its activities
// (ADR-009 D2-D4, D7-D9; PLAN-010 P10-03, P10-05, P10-06). A publish that
// adds a default, or the rollout migration, queues a schema_backfills job and
// an outbox row; the relay starts this workflow from that row; the workflow
// writes the default into every record of the type that lacks it, one record
// per transaction, through pipeline.Write(OpMaterialize).
//
// Design note: docs/data/schema-backfill.md.
package schemabackfill

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The Temporal names (contract C3). The Authorizer lists WorkflowType in
// platformWorkflowTypes and startOnlyWorkflowTypes, and TaskQueue in
// platformTaskQueues (temporal-server/authz/policy.go): only nava-relay
// starts it, never a child or a continue-as-new.
const (
	// WorkflowType is the workflow type, and the outbox kind of its start row.
	WorkflowType = outbox.SchemaBackfillKind
	// TaskQueue is the queue the workflow and every activity of it run on, so
	// the worker that polls it caps how many pages run at once (D7, PLAN-010
	// D-10-4).
	TaskQueue = "schema-backfill"
)

// Job names one schema_backfills row: the tenant, the entity definition and
// the target version. Every activity reads it from the workflow id, never
// from an argument.
type Job struct {
	Tenant     uuid.UUID
	Definition uuid.UUID
	Version    int
}

// errNotJobID is the error for a workflow id that is not a job's.
var errNotJobID = errors.New("workflow id is not t:<tenant>:schema.backfill:<entity definition id>:<target version>")

// WorkflowID is the job's workflow id,
// t:<tenant>:schema.backfill:<entity_definition_id>:<target_version> (D2).
// The relay builds it from outbox.tenant_id, never from the payload.
func WorkflowID(j Job) string {
	return "t:" + j.Tenant.String() + ":" + WorkflowType + ":" + j.Definition.String() + ":" + strconv.Itoa(j.Version)
}

// ParseWorkflowID parses a WorkflowID. Every part must be in its one
// canonical form (lowercase UUIDs, a version from 1 with no sign or leading
// zero), so exactly one id names each job.
func ParseWorkflowID(id string) (Job, error) {
	tenant, err := workerguard.TenantOf(id)
	if err != nil {
		return Job{}, errNotJobID
	}
	rest, ok := strings.CutPrefix(id, "t:"+tenant.String()+":"+WorkflowType+":")
	if !ok {
		return Job{}, errNotJobID
	}
	d, v, ok := strings.Cut(rest, ":")
	if !ok {
		return Job{}, errNotJobID
	}
	def, err := uuid.Parse(d)
	if err != nil || def == uuid.Nil || def.String() != d {
		return Job{}, errNotJobID
	}
	version, err := strconv.Atoi(v)
	if err != nil || version < 1 || version > math.MaxInt32 || strconv.Itoa(version) != v {
		return Job{}, errNotJobID
	}
	return Job{Tenant: tenant, Definition: def, Version: version}, nil
}
