//go:build integration

// TICK-963: ids minted when entity types are created and published are
// UUIDv7.

package server_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestEntityTypeCreateAndPublishMintV7Ids(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	if rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-v7", etPublishBody(t)); rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("publish = %d: %s", rec.Code, rec.Body)
	}
	rows, err := e.super.Query(t.Context(), `
SELECT id FROM entity_definitions WHERE tenant_id = $1
UNION ALL SELECT id FROM entity_definition_versions WHERE tenant_id = $1`, e.a.id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		n++
		if id.Version() != 7 {
			t.Errorf("id %s is version %d, want 7", id, id.Version())
		}
	}
	if n < 2 {
		t.Fatalf("checked %d ids, want a definition and a version", n)
	}
}

func TestWorkflowDefinitionPublishMintsV7Ids(t *testing.T) {
	e := newWFEnv(t)
	code, b := e.wfPublish(t, "admin-a", "triage", "v7-1", wfBody("incident", "short_description"))
	if code != http.StatusCreated {
		t.Fatalf("publish = %d: %s", code, b)
	}
	rows, err := e.super.Query(t.Context(), `
SELECT id FROM workflow_definitions WHERE tenant_id = $1
UNION ALL SELECT id FROM workflow_definition_versions WHERE tenant_id = $1`, e.a.id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		n++
		if id.Version() != 7 {
			t.Errorf("id %s is version %d, want 7", id, id.Version())
		}
	}
	if n != 2 {
		t.Fatalf("checked %d ids, want a definition and a version", n)
	}
	// Step ids are stored in the version row.
	for k, id := range wfParse(t, b).StepIDs {
		if id.Version() != 7 {
			t.Errorf("step %q id %s is version %d, want 7", k, id, id.Version())
		}
	}
}
