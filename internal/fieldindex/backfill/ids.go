// Package backfill is the field-index coordinator, FieldIndexBackfill and
// their activities (ADR-006 D9, D10; PLAN-007 D-7-12, D-7-15; TICK-745).
// Design note: docs/data/field-index-backfill.md.
//
// It is not internal/fieldindex itself: internal/metadata imports that leaf,
// and may import neither a database driver nor Temporal. The names the relay
// needs (workflow types, the signal, its body) live in the leaf.
package backfill

import (
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// errNotChildID is the error for a workflow id that is not a field-index
// child's: t:<tenant>:<type>:<field id>:<version>.
var errNotChildID = errors.New("workflow id is not t:<tenant>:<field index child type>:<field id>:<version>")

// maxVersion is the largest requested_version a child id may pin: the
// outbox's target_seq bound for EnqueueFieldIndexBackfill.
const maxVersion = 32767

// ChildID is the workflow id of the field-index child of type wt for field in
// tenant: t:<tenant>:<wt>:<field>:<version>. version is the
// requested_version the child is pinned to, or 0 for none (an item the
// coordinator carried across continue-as-new, which carries field ids only).
//
// The id is deterministic, so a coordinator that restarts an item starts the
// same child id, and the activities read the field and the version from it:
// nothing an activity acts on comes from an argument.
func ChildID(tenant uuid.UUID, wt string, field uuid.UUID, version int) string {
	return "t:" + tenant.String() + ":" + wt + ":" + field.String() + ":" + strconv.Itoa(version)
}

// child is a parsed ChildID.
type child struct {
	tenant  uuid.UUID
	field   uuid.UUID
	version int // 0: not pinned
}

// parseChildID parses a ChildID of type wt.
func parseChildID(workflowID, wt string) (child, error) {
	tenant, err := workerguard.TenantOf(workflowID)
	if err != nil {
		return child{}, errNotChildID
	}
	rest, ok := strings.CutPrefix(workflowID, "t:"+tenant.String()+":"+wt+":")
	if !ok {
		return child{}, errNotChildID
	}
	f, v, ok := strings.Cut(rest, ":")
	if !ok {
		return child{}, errNotChildID
	}
	field, err := uuid.Parse(f)
	if err != nil || field == uuid.Nil || field.String() != f {
		return child{}, errNotChildID
	}
	version, err := strconv.Atoi(v)
	if err != nil || version < 0 || version > maxVersion || strconv.Itoa(version) != v {
		return child{}, errNotChildID
	}
	return child{tenant: tenant, field: field, version: version}, nil
}

// parseAnyChildID parses the id of a backfill or a removal child.
func parseAnyChildID(workflowID string) (child, error) {
	if c, err := parseChildID(workflowID, fieldindex.BackfillWorkflowType); err == nil {
		return c, nil
	}
	return parseChildID(workflowID, fieldindex.RemoveWorkflowType)
}
