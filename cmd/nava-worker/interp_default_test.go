//go:build !interptest

package main

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/workflowdefs"
)

// TestProductionBuildWiresTheDatabaseDefinitionStore: the production worker
// registers the interpreter with the database DefinitionStore, and its
// record ports refuse until P3-43 (they never read or write anything).
func TestProductionBuildWiresTheDatabaseDefinitionStore(t *testing.T) {
	pool := &pgxpool.Pool{}
	acts, err := interpreterActivities(func(string) (string, bool) { return "/fixtures", true }, pool)
	if err != nil || acts == nil {
		t.Fatalf("interpreterActivities = (%v, %v), want the interpreter's ports", acts, err)
	}
	r, ok := acts.Definitions.(workflowdefs.Reader)
	if !ok || r.Pool != pool {
		t.Errorf("Definitions = %T, want workflowdefs.Reader over the worker's pool", acts.Definitions)
	}
	if _, err := acts.Records.Read(t.Context(), uuid.New(), uuid.New()); !errors.Is(err, errNoRecordPorts) {
		t.Errorf("Records.Read = %v, want errNoRecordPorts", err)
	}
	if err := acts.Writer.Write(t.Context(), uuid.New(), uuid.New(), uuid.New(), "k", nil); !errors.Is(err, errNoRecordPorts) {
		t.Errorf("Writer.Write = %v, want errNoRecordPorts", err)
	}
	reg := &recordingRegistry{}
	registerInterpreter(reg, acts)
	if len(reg.workflows) == 0 || len(reg.activities) == 0 {
		t.Errorf("registered workflows %v activities %v, want the interpreter's", reg.workflows, reg.activities)
	}
}

func TestProductionBuildInterpreterNeedsAPool(t *testing.T) {
	if _, err := interpreterActivities(func(string) (string, bool) { return "", false }, nil); err == nil {
		t.Error("interpreterActivities with no pool returned no error")
	}
}
