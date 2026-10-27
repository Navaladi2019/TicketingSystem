//go:build temporalstack

package temporalstack

// The S7 demo (make demo-outbox, TICK-275) must keep working: run twice with
// one dedupe key, it leaves exactly one workflow of the tenant, and that
// workflow's history holds the two signal rows' deliveries in commit order.
// The rows are written by cmd/demo-outbox through outbox.Enqueue and the
// stack's nava-relay service hands them to Temporal. The test's worker runs
// relayProbe as that type, which has no demo.note channel, so the signals stay
// unapplied and the history is the evidence. Run by
// make stack-test-demo_outbox only (NAVA_TEST_DEMO_OUTBOX=1), not by the
// whole-package stack-test-sdk run.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/converter"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// demoTenant is tenant-a, the seeded Keycloak organization.
const demoTenant = "00000000-0000-4000-8000-0000000000a1"

// runDemo runs make demo-outbox from the repository root and returns its
// "name: value" lines.
func runDemo(t *testing.T, key string) map[string]string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "make", "--no-print-directory", "demo-outbox", "DEDUPE_KEY="+key)
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make demo-outbox: %v\n%s", err, out)
	}
	got := map[string]string{}
	for line := range strings.Lines(string(out)) {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), ": "); ok {
			got[k] = v
		}
	}
	for _, k := range []string{"tenant", "start row", "signal row 1", "signal row 2", "workflow id"} {
		if got[k] == "" {
			t.Fatalf("make demo-outbox printed no %q line:\n%s", k, out)
		}
	}
	return got
}

func TestDemoOutboxTwiceWithOneKeyIsOneWorkflowWithTwoSignalsInOrder(t *testing.T) {
	if os.Getenv("NAVA_TEST_DEMO_OUTBOX") == "" {
		t.Skip("run by make stack-test-demo_outbox")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	s := newRelayStack(t)
	key := "demo-test-" + uuid.NewString()

	first := runDemo(t, key)
	second := runDemo(t, key)
	for k, v := range first {
		if second[k] != v {
			t.Errorf("second run printed %s %q, the first %q: the duplicate path wrote something", k, second[k], v)
		}
	}
	if first["tenant"] != demoTenant {
		t.Fatalf("tenant %q, want tenant-a %s", first["tenant"], demoTenant)
	}
	id := first["workflow id"]
	if want := "t:" + demoTenant + ":nava.smoke.relay:" + first["start row"]; id != want {
		t.Fatalf("workflow id %q, want %q", id, want)
	}
	s.terminateAtEnd(t, id)

	// Three rows for the key, however many runs: the duplicate wrote nothing.
	if n := psqlSuper(t, fmt.Sprintf(`SELECT count(*) FROM outbox WHERE tenant_id = '%s' AND dedupe_key LIKE '%s:%%';`, demoTenant, key)); n != "3" {
		t.Fatalf("%s outbox rows for the key after two runs, want 3", n)
	}
	// Every row settles dispatched (the relay's own work), the signals after the start.
	for _, k := range []string{"start row", "signal row 1", "signal row 2"} {
		rid, err := uuid.Parse(first[k])
		if err != nil {
			t.Fatalf("%s %q: %v", k, first[k], err)
		}
		if r := awaitSettled(t, rid); !r.dispatched || r.dead {
			t.Fatalf("%s not dispatched: %+v", k, r)
		}
	}

	s.awaitStarted(ctx, t, id)
	var got []string
	iter := s.core.GetWorkflowHistory(ctx, id, "", false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for iter.HasNext() {
		ev, err := iter.Next()
		if err != nil {
			t.Fatalf("history of %s: %v", id, err)
		}
		attrs := ev.GetWorkflowExecutionSignaledEventAttributes()
		if attrs == nil {
			continue
		}
		var d outbox.Delivery
		if err := converter.GetDefaultDataConverter().FromPayloads(attrs.GetInput(), &d); err != nil {
			t.Fatalf("signal input: %v", err)
		}
		got = append(got, attrs.GetSignalName()+" "+d.OutboxID.String())
	}
	want := []string{"demo.note " + first["signal row 1"], "demo.note " + first["signal row 2"]}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("signals in the history of %s:\n got  %v\n want %v", id, got, want)
	}
}
