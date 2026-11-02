//go:build integration

package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// atVersion asserts err is an *ErrAtVersion naming want and the incident
// type's mapping, and returns it.
func atVersion(t *testing.T, err error, want int) *pipeline.ErrAtVersion {
	t.Helper()
	var av *pipeline.ErrAtVersion
	if !errors.As(err, &av) {
		t.Fatalf("err = %v, want *ErrAtVersion", err)
	}
	if av.SchemaVersion != want {
		t.Errorf("refused write reports v%d, want v%d", av.SchemaVersion, want)
	}
	if av.FixedColumns["requester"] != metadata.ColumnRequesterID || av.FixedColumns["status"] != metadata.ColumnState || av.StateProperty != "status" {
		t.Errorf("mapping = %v, state property %q; want the incident type's", av.FixedColumns, av.StateProperty)
	}
	return av
}

// AC (TICK-413): a refused write reports the definition version it validated
// against, on a validation failure as on a foreign key violation, so a caller
// never maps the refusal under a version it read itself.
func TestRefusedWriteReportsTheVersionItUsed(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	invalid := map[string]any{"priority": "urgent"}
	unknownRequester := map[string]any{"title": "x", "requester": uuid.NewString()}

	_, err := write(ctx, e, p, f.id, createCmd(f, invalid))
	atVersion(t, err, 1)
	var ve *pipeline.ErrValidation
	if !errors.As(err, &ve) {
		t.Errorf("err = %v, want it to still unwrap to *ErrValidation", err)
	}

	res := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	publishV2(t, e, f)

	_, err = write(ctx, e, p, f.id, createCmd(f, invalid))
	atVersion(t, err, 2)

	_, err = write(ctx, e, p, f.id, createCmd(f, unknownRequester))
	atVersion(t, err, 2)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "23503" {
		t.Errorf("err = %v, want it to still unwrap to the 23503", err)
	}

	// An update of a v1 record validates against the current version, v2.
	_, err = write(ctx, e, p, f.id, updateCmd(f, res.RecordID, 1, map[string]any{"title": ""}))
	atVersion(t, err, 2)
}

// A write refused before the definition is read names no version.
func TestWriteRefusedBeforeTheDefinitionHasNoVersion(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	cmd := createCmd(f, map[string]any{"title": "x"})
	cmd.EntityKey = "no_such_type"
	_, err := write(context.Background(), e, p, f.id, cmd)
	var av *pipeline.ErrAtVersion
	if !errors.Is(err, pipeline.ErrNotFound) || errors.As(err, &av) {
		t.Errorf("err = %v, want ErrNotFound without a version", err)
	}
}
