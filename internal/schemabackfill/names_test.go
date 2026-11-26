package schemabackfill

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

func TestWorkflowIDRoundTrip(t *testing.T) {
	for _, v := range []int{1, 2, math.MaxInt32} {
		j := Job{Tenant: uuid.New(), Definition: uuid.New(), Version: v}
		id := WorkflowID(j)
		got, err := ParseWorkflowID(id)
		if err != nil || got != j {
			t.Errorf("%s: %+v, %v", id, got, err)
		}
		if tenant, err := workerguard.TenantOf(id); err != nil || tenant != j.Tenant {
			t.Errorf("%s: TenantOf = %v, %v; want the job's tenant", id, tenant, err)
		}
	}
}

func TestParseWorkflowIDRejects(t *testing.T) {
	tenant, def := uuid.New(), uuid.New()
	base := "t:" + tenant.String() + ":schema.backfill:"
	for _, id := range []string{
		"", base, base + def.String(), base + def.String() + ":", base + def.String() + ":0", base + def.String() + ":-1",
		base + def.String() + ":01", base + def.String() + ":+1", base + def.String() + ":1:2",
		base + def.String() + ":" + strconv.FormatInt(math.MaxInt32+1, 10), base + uuid.Nil.String() + ":1",
		base + "X" + def.String()[1:] + ":1", "schema.backfill:" + def.String() + ":1",
		"t:" + tenant.String() + ":field_index.coordinator", "t:" + tenant.String() + ":schema.backfillx:" + def.String() + ":1",
	} {
		if j, err := ParseWorkflowID(id); err == nil {
			t.Errorf("%q parsed as %+v", id, j)
		}
	}
}

func TestPayloadMatches(t *testing.T) {
	j := Job{Tenant: uuid.New(), Definition: uuid.New(), Version: 2}
	ok := json.RawMessage(`{"entity_definition_id":"` + j.Definition.String() + `"}`)
	for _, p := range []json.RawMessage{nil, json.RawMessage(`{}`), ok} {
		if !payloadMatches(p, j) {
			t.Errorf("%s refused", p)
		}
	}
	for _, p := range []string{
		`{"entity_definition_id":"` + uuid.NewString() + `"}`,
		`{"entity_definition_id":"` + j.Definition.String() + `","tenant_id":"` + j.Tenant.String() + `"}`,
		`{"tenant_id":"` + j.Tenant.String() + `"}`, `[]`, `"x"`,
	} {
		if payloadMatches(json.RawMessage(p), j) {
			t.Errorf("%s accepted", p)
		}
	}
}

// Every activity argument and result type encodes to references only, the
// worker guard's rule (CI-7): a pointer set to the definition id, or nothing.
func TestActivityValuesAreReferencesOnly(t *testing.T) {
	id := uuid.New()
	for _, v := range []any{
		AdmitResult{}, AdmitResult{Admitted: &id}, AdmitResult{Ended: &id}, FillResult{Ended: &id},
		Cursor{}, Cursor{After: &id}, PageResult{Next: &id}, PageResult{Ended: &id}, EndResult{FailedRecords: &id},
	} {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]string
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Errorf("%T encodes as %s, not an object of strings", v, raw)
			continue
		}
		for k, s := range m {
			if u, err := uuid.Parse(s); err != nil || u.String() != s {
				t.Errorf("%T: %s = %q is not a canonical UUID", v, k, s)
			}
		}
	}
}

func TestJobDocHoldsNoRecordData(t *testing.T) {
	j := Job{Tenant: uuid.New(), Definition: uuid.New(), Version: 4}
	got := string(jobDoc(j, stateDone, &jobRow{rewritten: 3, skipped: 2, failed: 1}))
	want := `{"backfill":{"failed":1,"rewritten":3,"skipped":2,"state":"done","target_version":4}}`
	if got != want {
		t.Errorf("jobDoc = %s, want %s", got, want)
	}
	if got := string(jobDoc(j, stateFilling, nil)); got != `{"backfill":{"state":"filling","target_version":4}}` {
		t.Errorf("jobDoc without counts = %s", got)
	}
}
