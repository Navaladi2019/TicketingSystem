package store_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

func fixedColumns() map[string]string {
	return map[string]string{
		"status":    "state",
		"requester": "requester_id",
		"assignee":  "assignee_id",
		"group":     "assignment_group_id",
	}
}

// A mapped property lives only in its column, and Payload merges it back.
func TestSplitAndPayload(t *testing.T) {
	fixed := fixedColumns()
	req, grp := uuid.New(), uuid.New()
	payload := map[string]any{
		"status": "open", "requester": req.String(), "group": grp.String(),
		"title": "printer", "tags": []any{"a"},
	}
	var r store.Record
	if err := r.Split(payload, fixed); err != nil {
		t.Fatal(err)
	}
	if want := map[string]any{"title": "printer", "tags": []any{"a"}}; !reflect.DeepEqual(r.Data, want) {
		t.Fatalf("data = %v, want %v", r.Data, want)
	}
	if r.State != "open" || *r.RequesterID != req || r.AssigneeID != nil || *r.AssignmentGroupID != grp {
		t.Fatalf("columns = %q %v %v %v", r.State, r.RequesterID, r.AssigneeID, r.AssignmentGroupID)
	}
	if _, ok := payload["status"]; !ok {
		t.Fatal("Split changed its input")
	}
	if got := r.Payload(fixed); !reflect.DeepEqual(got, payload) {
		t.Fatalf("payload = %v, want %v", got, payload)
	}

	// A stale copy in data never wins over the column.
	r.Data["status"] = "closed"
	if got := r.Payload(fixed)["status"]; got != "open" {
		t.Fatalf("payload status = %v, want the column value open", got)
	}
}

func TestSplitRefusesBadColumns(t *testing.T) {
	fixed := fixedColumns()
	for name, p := range map[string]map[string]any{
		"no state":         {"title": "x"},
		"state not string": {"status": json.Number("1")},
		"uuid not string":  {"status": "open", "requester": true},
		"not a uuid":       {"status": "open", "requester": "nope"},
	} {
		var r store.Record
		if err := r.Split(p, fixed); err == nil {
			t.Errorf("%s: Split accepted it", name)
		}
	}
}

func TestDecodeObjectKeepsNumbersExact(t *testing.T) {
	m, err := store.DecodeObject([]byte(`{"n": 12345678901234567890}`))
	if err != nil {
		t.Fatal(err)
	}
	if m["n"] != json.Number("12345678901234567890") {
		t.Fatalf("n = %#v", m["n"])
	}
	if m, err := store.DecodeObject(nil); err != nil || len(m) != 0 {
		t.Fatalf("empty input = %v, %v", m, err)
	}
}

// D-3-9: seven digits, zero padded, growing past seven.
func TestFormatNumber(t *testing.T) {
	for n, want := range map[int64]string{1: "INC0000001", 9999999: "INC9999999", 10000000: "INC10000000"} {
		if got := store.FormatNumber("INC", n); got != want {
			t.Errorf("FormatNumber(%d) = %q, want %q", n, got, want)
		}
	}
}
