package recordports

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/schemabackfill"
)

// The job's own audit entries and its rewrites name one actor_system (ADR-009
// D2); the job cannot import the pipeline, so the two constants are pinned
// equal here.
func TestMaterializeActorSystem(t *testing.T) {
	if schemabackfill.ActorSystem != pipeline.ActorSchemaBackfill {
		t.Fatalf("schemabackfill.ActorSystem = %q, pipeline.ActorSchemaBackfill = %q", schemabackfill.ActorSystem, pipeline.ActorSchemaBackfill)
	}
}
