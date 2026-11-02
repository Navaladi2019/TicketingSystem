package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

func env(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

const (
	tenant  = "00000000-0000-4000-8000-0000000000a1"
	record  = "1a2b3c4d-5e6f-4708-9192-a3b4c5d6e7f1"
	version = "51a11000-0000-4000-8000-000000000001"
)

// TestEntries: start and signal write the rows the interpreter reads, and
// both payloads pass the outbox's references-only check.
func TestEntries(t *testing.T) {
	base := map[string]string{"NAVA_DATABASE_URL": "postgres://x@y/z", "DEDUPE_KEY": "k", "TENANT_ID": tenant, "RECORD_ID": record}
	for _, tc := range []struct {
		action, idVar, kind string
		target              outbox.Target
		payload             map[string]string
	}{
		{"start", "VERSION_ID", interp.WorkflowType, outbox.TargetStart{Kind: interp.WorkflowType},
			map[string]string{"definition_version_id": version, "record_id": record}},
		{"signal", "SIGNAL_ID", interp.SignalName, outbox.TargetSignal{Kind: interp.WorkflowType, SignalName: interp.SignalName},
			map[string]string{"signal": version}},
	} {
		t.Run(tc.action, func(t *testing.T) {
			m := map[string]string{"DEMO_ACTION": tc.action, tc.idVar: version}
			for k, v := range base {
				m[k] = v
			}
			in, err := parse(env(m))
			if err != nil {
				t.Fatal(err)
			}
			e, err := in.entry()
			if err != nil {
				t.Fatal(err)
			}
			if e.Kind != tc.kind || e.Target != tc.target || e.RecordID.String() != record || e.DedupeKey != "k" {
				t.Errorf("entry %+v", e)
			}
			var got map[string]string
			if err := json.Unmarshal(e.Payload, &got); err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.payload) {
				t.Errorf("payload %v, want %v", got, tc.payload)
			}
			for k, v := range tc.payload {
				if got[k] != v {
					t.Errorf("payload %v, want %v", got, tc.payload)
				}
			}
			if err := outbox.ValidatePayload(e.Payload); err != nil {
				t.Errorf("payload refused: %v", err)
			}
		})
	}
}

func TestParseRefuses(t *testing.T) {
	for name, m := range map[string]map[string]string{
		"no action":  {"NAVA_DATABASE_URL": "x", "DEDUPE_KEY": "k"},
		"no key":     {"NAVA_DATABASE_URL": "x", "DEMO_ACTION": "start"},
		"long key":   {"NAVA_DATABASE_URL": "x", "DEMO_ACTION": "start", "DEDUPE_KEY": strings.Repeat("k", maxKeyLen+1)},
		"no dsn":     {"DEMO_ACTION": "start", "DEDUPE_KEY": "k", "TENANT_ID": tenant, "RECORD_ID": record, "VERSION_ID": version},
		"no version": {"NAVA_DATABASE_URL": "x", "DEMO_ACTION": "start", "DEDUPE_KEY": "k", "TENANT_ID": tenant, "RECORD_ID": record},
		"nil tenant": {"NAVA_DATABASE_URL": "x", "DEMO_ACTION": "signal", "DEDUPE_KEY": "k", "TENANT_ID": uuid.Nil.String(), "RECORD_ID": record, "SIGNAL_ID": version},
	} {
		if _, err := parse(env(m)); err == nil {
			t.Errorf("%s: parse accepted %v", name, m)
		}
	}
}

// TestRecordID: record-id needs no database, and a key names one record per tenant.
func TestRecordID(t *testing.T) {
	var out strings.Builder
	if err := run(t.Context(), env(map[string]string{"DEMO_ACTION": "record-id", "TENANT_ID": tenant, "DEDUPE_KEY": "k"}), &out); err != nil {
		t.Fatal(err)
	}
	a := uuid.MustParse(tenant)
	if got := strings.TrimSpace(out.String()); got != RecordID(a, "k").String() {
		t.Errorf("record-id printed %q, want %s", got, RecordID(a, "k"))
	}
	if RecordID(a, "k") == RecordID(uuid.MustParse("00000000-0000-4000-8000-0000000000b1"), "k") {
		t.Error("one key names the same record in two tenants")
	}
}
