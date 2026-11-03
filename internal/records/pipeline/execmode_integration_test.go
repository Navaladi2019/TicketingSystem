//go:build integration

package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// inExecMode returns e with its nava_app pool replaced by one in
// pgx.QueryExecModeExec, the mode nava-core's PgBouncer pool uses: no
// prepared statement, so no parameter has a declared type and pgx picks the
// wire format from the Go type alone. A []byte for a json or jsonb column
// then goes as bytea text and PostgreSQL refuses it (22P02, TICK-411).
func inExecMode(t *testing.T, e env) env {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(appDSN(t, e.platform))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	p, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	e.app = p
	return e
}

// enqueueSink enqueues a start row for every write with
// outbox.EnqueueAtVersion, as the S11b sink will (D-3-14), so the outbox
// insert runs in exec mode too.
type enqueueSink struct{ n int }

func (s *enqueueSink) RecordChanged(ctx context.Context, tx pgx.Tx, ev pipeline.Event) error {
	s.n++
	payload, err := json.Marshal(map[string]string{"record_id": ev.RecordID.String()})
	if err != nil {
		return err
	}
	_, err = outbox.EnqueueAtVersion(ctx, tx, outbox.Entry{Kind: "ticket.approval", KindVersion: 1,
		DedupeKey: ev.RecordID.String() + ":" + uuid.NewString(), RecordID: ev.RecordID,
		Target: outbox.TargetStart{Kind: "ticket.approval"}, Causation: ev.Causation, Payload: payload}, ev.Version)
	return err
}

// TestWritesInExecMode: every statement a write runs takes its arguments in
// QueryExecModeExec: the record insert and update (data jsonb), the audit
// row (before and after), the idempotent replay, the unique-value check
// ($4::jsonb), the cascade_rejected audit row on the runner, and the sink's
// outbox row (payload). A json argument sent as []byte fails each with 22P02.
func TestWritesInExecMode(t *testing.T) {
	e := migrated(t)
	f := seedTenantWith(t, e, assetDefinition())
	e = inExecMode(t, e)
	sink := &enqueueSink{}
	p := newPipeline(e, sink)
	ctx := context.Background()
	w := func(cmd pipeline.Command) (pipeline.Result, error) { return write(ctx, e, p, f.id, cmd) }

	created, err := w(assetCmd(f, map[string]any{"name": "one", "serial": "SN-1", "tag": 7}))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// The unique check ran and found the value, rather than failing.
	_, err = w(assetCmd(f, map[string]any{"name": "dup", "serial": "SN-1"}))
	wantUnique(t, err, "serial")
	updated, err := w(assetUpdate(f, created.RecordID, created.Version, map[string]any{"name": "renamed", "tag": 8}))
	if err != nil || updated.Version != 2 {
		t.Fatalf("update: %+v, %v; want version 2", updated, err)
	}

	// A system write with an idempotency key, then its replay.
	sys := pipeline.Command{Op: pipeline.OpCreate, EntityKey: "asset", Data: map[string]any{"name": "sys"},
		Actor: f.system(), Source: pipeline.SourceSystem, IdempotencyKey: "exec-mode-1"}
	first, err := w(sys)
	if err != nil {
		t.Fatalf("system create: %v", err)
	}
	again, err := w(sys)
	if err != nil || again != first {
		t.Fatalf("replay: %+v, %v; want %+v", again, err, first)
	}

	// Beyond the depth limit: refused, and the audit entry is written on the
	// runner, in exec mode too.
	deep := pipeline.Command{Op: pipeline.OpCreate, EntityKey: "asset", Data: map[string]any{"name": "deep"},
		Actor: f.system(), Source: pipeline.SourceSystem,
		Causation: outbox.Causation{OriginID: uuid.New(), Depth: config.CascadeDepthLimit + 1}}
	if _, err := w(deep); !errors.Is(err, outbox.ErrCascadeTooDeep) {
		t.Fatalf("too deep: %v, want ErrCascadeTooDeep", err)
	}

	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND domain = 'record'`, f.id); n != 4 {
		t.Errorf("%d audit entries, want 4 (create, update, system create, cascade_too_deep)", n)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE tenant_id = $1`, f.id); n != sink.n || n != 3 {
		t.Errorf("%d outbox rows for %d sink calls, want 3 each (the replay calls no sink)", n, sink.n)
	}
	if n := count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1 AND id = $2 AND data ->> 'name' = 'renamed'`,
		f.id, created.RecordID); n != 1 {
		t.Errorf("the update's data was not stored as jsonb")
	}
}
