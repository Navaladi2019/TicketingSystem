package interp_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// Two rules on the committed histories (QA, TICK-363).

// historiesRepoDir is historiesDir relative to the repository root.
const historiesRepoDir = "internal/workflow/testdata/histories"

// --- Append-only --------------------------------------------------------

// TestHistoriesAppendOnly: a history on main is never changed or deleted. A
// breaking interpreter change ships as a new GetVersion branch plus a new
// history, never as a re-recorded one, or the gate would stop proving that
// runs already in flight still replay.
//
// BASE names main's commit (origin/main in the ci workflow's unit job, for
// every event, as for TestNewMigrationsLaterThanBase). The reference is main,
// what has shipped: a story branch may re-record a history main does not have
// yet. Without BASE the test skips: a laptop's origin/main may be stale.
func TestHistoriesAppendOnly(t *testing.T) {
	base := os.Getenv("BASE")
	if base == "" {
		t.Skip("BASE not set: set BASE=origin/main (after git fetch origin main) to check the histories against main")
	}
	top, err := git(t, ".", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkAppendOnly(t, strings.TrimSpace(top), base, historiesRepoDir); err != nil {
		t.Fatal(err)
	}
}

// checkAppendOnly reports every file under dir in base that the work tree at
// repo has deleted or whose content differs (git hash-object against the
// blob base records).
func checkAppendOnly(t *testing.T, repo, base, dir string) error {
	t.Helper()
	out, err := git(t, repo, "ls-tree", "-r", "--full-tree", base, "--", dir)
	if err != nil {
		return fmt.Errorf("git ls-tree %s -- %s: %w (fetch main first; the checkout needs its history)", base, dir, err)
	}
	var problems []string
	for line := range strings.Lines(out) {
		meta, path, ok := strings.Cut(strings.TrimRight(line, "\r\n"), "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 || fields[1] != "blob" {
			continue
		}
		blob := fields[2]
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(path))); errors.Is(err, os.ErrNotExist) {
			problems = append(problems, path+" was deleted")
			continue
		}
		got, err := git(t, repo, "hash-object", "--", path)
		if err != nil {
			return err
		}
		if strings.TrimSpace(got) != blob {
			problems = append(problems, path+" was changed")
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("committed histories are append-only (%s is the reference):\n%s\n"+
			"Restore them. A breaking interpreter change ships as a workflow.GetVersion branch plus a NEW history "+
			"(docs/quality/ci-pipeline.md, \"Changing nava.interpreter\")", base, strings.Join(problems, "\n"))
	}
	return nil
}

func git(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

// TestAppendOnlyCheck runs checkAppendOnly on a scratch repository: a new
// history passes; an edited or a deleted one fails.
func TestAppendOnlyCheck(t *testing.T) {
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if _, err := git(t, repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	write := func(name, body string) {
		t.Helper()
		p := filepath.Join(repo, filepath.FromSlash(historiesRepoDir), name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q")
	run("config", "core.autocrlf", "false")
	write("a.json", `{"events":[1]}`+"\n")
	write("b.json", `{"events":[2]}`+"\n")
	run("add", ".")
	run("-c", "user.name=replay-gate", "-c", "user.email=replay-gate@example.invalid", "-c", "commit.gpgsign=false",
		"commit", "-q", "-m", "base")
	run("tag", "base")

	check := func() error { return checkAppendOnly(t, repo, "base", historiesRepoDir) }
	if err := check(); err != nil {
		t.Fatalf("unchanged: %v", err)
	}
	write("c.json", `{"events":[3]}`+"\n")
	if err := check(); err != nil {
		t.Fatalf("a new history must pass: %v", err)
	}
	write("a.json", `{"events":[1,4]}`+"\n")
	if err := os.Remove(filepath.Join(repo, filepath.FromSlash(historiesRepoDir), "b.json")); err != nil {
		t.Fatal(err)
	}
	err := check()
	if err == nil {
		t.Fatal("an edited and a deleted history passed")
	}
	for _, want := range []string{historiesRepoDir + "/a.json was changed", historiesRepoDir + "/b.json was deleted"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "c.json") {
		t.Errorf("error %q names the new history", err)
	}
}

// --- R1 walk ----------------------------------------------------------------

// TestHistoriesHoldReferencesOnly walks every payload of every committed
// history (ADR-001 R1, CI-7) and fails on any that is not:
//   - a start payload: the relay's outbox.Delivery whose payload is exactly
//     {definition_version_id, record_id}, or the continue-as-new
//     workerguard.RunInput carrying it with UUID-only refs;
//   - a signal delivery: an outbox.Delivery whose payload is exactly {signal};
//   - a workerguard.Directive;
//   - a UUID-only object (StepRef): every value a UUID or a list of UUIDs;
//   - an empty (binary/null) result.
//
// The one header the platform sets, nava-tenant, must be a UUID string; any
// other header fails too.
func TestHistoriesHoldReferencesOnly(t *testing.T) {
	for _, h := range loadHistories(t) {
		if errs := walkR1(h.h); len(errs) > 0 {
			t.Errorf("%s holds data that is not references only:\n%s", h.name, strings.Join(errs, "\n"))
		}
	}
}

// TestR1WalkCatchesRecordData plants a record field in a recorded history,
// in each kind of payload, and expects the walk to report it.
func TestR1WalkCatchesRecordData(t *testing.T) {
	plants := []struct {
		name string
		// pick returns the payload to overwrite in e, or nil.
		pick func(e *historypb.HistoryEvent) *commonpb.Payload
		data func(orig map[string]any) map[string]any
	}{
		{
			name: "directive result",
			pick: func(e *historypb.HistoryEvent) *commonpb.Payload {
				return first(e.GetActivityTaskCompletedEventAttributes().GetResult())
			},
			data: func(m map[string]any) map[string]any { m["priority"] = "high"; return m },
		},
		{
			name: "activity argument",
			pick: func(e *historypb.HistoryEvent) *commonpb.Payload {
				return first(e.GetActivityTaskScheduledEventAttributes().GetInput())
			},
			data: func(m map[string]any) map[string]any { m["title"] = "printer on fire"; return m },
		},
		{
			name: "start payload",
			pick: func(e *historypb.HistoryEvent) *commonpb.Payload {
				return first(e.GetWorkflowExecutionStartedEventAttributes().GetInput())
			},
			data: func(m map[string]any) map[string]any {
				m["payload"].(map[string]any)["subject"] = "printer on fire"
				return m
			},
		},
		{
			name: "signal delivery",
			pick: func(e *historypb.HistoryEvent) *commonpb.Payload {
				return first(e.GetWorkflowExecutionSignaledEventAttributes().GetInput())
			},
			data: func(m map[string]any) map[string]any {
				m["payload"].(map[string]any)["comment"] = "approved by the manager"
				return m
			},
		},
	}
	clean := readHistory(t, filepath.Join(historiesDir, "signal.json"))
	if errs := walkR1(clean.h); len(errs) > 0 {
		t.Fatalf("signal.json is not clean: %v", errs)
	}
	for _, p := range plants {
		t.Run(p.name, func(t *testing.T) {
			h := proto.Clone(clean.h).(*historypb.History)
			planted := false
			for _, e := range h.GetEvents() {
				pl := p.pick(e)
				if pl == nil {
					continue
				}
				var m map[string]any
				if err := json.Unmarshal(pl.GetData(), &m); err != nil {
					t.Fatal(err)
				}
				b, err := json.Marshal(p.data(m))
				if err != nil {
					t.Fatal(err)
				}
				pl.Data = b
				planted = true
				break
			}
			if !planted {
				t.Fatal("signal.json has no such payload to plant into")
			}
			if errs := walkR1(h); len(errs) == 0 {
				t.Fatal("the walk accepted a payload holding a record field")
			}
		})
	}
	t.Run("foreign header", func(t *testing.T) {
		h := proto.Clone(clean.h).(*historypb.History)
		start := h.GetEvents()[0].GetWorkflowExecutionStartedEventAttributes()
		p, err := converter.GetDefaultDataConverter().ToPayload("printer on fire")
		if err != nil {
			t.Fatal(err)
		}
		start.Header.Fields["subject"] = p
		if errs := walkR1(h); len(errs) == 0 {
			t.Fatal("the walk accepted a header other than nava-tenant")
		}
	})
}

func first(p *commonpb.Payloads) *commonpb.Payload {
	if len(p.GetPayloads()) == 0 {
		return nil
	}
	return p.GetPayloads()[0]
}

//nolint:gochecknoglobals // descriptors the walk matches on
var (
	payloadName = (&commonpb.Payload{}).ProtoReflect().Descriptor().FullName()
	headerName  = (&commonpb.Header{}).ProtoReflect().Descriptor().FullName()
)

// walkR1 returns one line per payload of h that is not references only. The
// lines name the event and the field path, never the payload's content.
func walkR1(h *historypb.History) []string {
	var errs []string
	for _, e := range h.GetEvents() {
		walkMessage(e.ProtoReflect(), "", func(path string, why string) {
			errs = append(errs, fmt.Sprintf("event %d (%s) %s: %s", e.GetEventId(), e.GetEventType(), path, why))
		})
	}
	return errs
}

func walkMessage(m protoreflect.Message, path string, report func(path, why string)) {
	switch m.Descriptor().FullName() {
	case payloadName:
		if why := payloadR1(m.Interface().(*commonpb.Payload)); why != "" {
			report(path, why)
		}
		return
	case headerName:
		for k, p := range m.Interface().(*commonpb.Header).GetFields() {
			if why := headerR1(k, p); why != "" {
				report(path+".fields["+k+"]", why)
			}
		}
		return
	}
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		p := path + "." + string(fd.Name())
		switch {
		case fd.IsMap():
			if fd.MapValue().Message() != nil {
				v.Map().Range(func(k protoreflect.MapKey, mv protoreflect.Value) bool {
					walkMessage(mv.Message(), p+"["+k.String()+"]", report)
					return true
				})
			}
		case fd.IsList():
			if fd.Message() != nil {
				for i := range v.List().Len() {
					walkMessage(v.List().Get(i).Message(), fmt.Sprintf("%s[%d]", p, i), report)
				}
			}
		case fd.Message() != nil:
			walkMessage(v.Message(), p, report)
		}
		return true
	})
}

func headerR1(key string, p *commonpb.Payload) string {
	if key != workerguard.TenantHeader {
		return "a header other than " + workerguard.TenantHeader
	}
	var s string
	if string(p.GetMetadata()[converter.MetadataEncoding]) != converter.MetadataEncodingJSON || json.Unmarshal(p.GetData(), &s) != nil {
		return "the tenant header is not a JSON string"
	}
	if id, err := uuid.Parse(s); err != nil || id.String() != s {
		return "the tenant header is not a canonical UUID"
	}
	return ""
}

// payloadR1 returns why p is not one of the allowed shapes, or "".
func payloadR1(p *commonpb.Payload) string {
	switch string(p.GetMetadata()[converter.MetadataEncoding]) {
	case converter.MetadataEncodingNil:
		if len(p.GetData()) != 0 {
			return "a binary/null payload with data"
		}
		return ""
	case converter.MetadataEncodingJSON:
	default:
		return "an encoding other than json/plain"
	}
	raw := json.RawMessage(p.GetData())
	switch {
	case isDirectiveJSON(raw), isDeliveryJSON(raw), isRunInputJSON(raw), isUUIDObject(raw):
		return ""
	}
	return "not a start payload, a signal delivery, a Directive or a UUID-only object"
}

// strict decodes raw into v, refusing unknown fields and trailing data.
func strict(raw json.RawMessage, v any) bool {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	return dec.Decode(v) == nil && !dec.More()
}

func isDirectiveJSON(raw json.RawMessage) bool {
	var d workerguard.Directive
	if !strict(raw, &d) || d.Step == uuid.Nil {
		return false
	}
	switch d.Kind {
	case workerguard.KindTimer, workerguard.KindWaitSignal, workerguard.KindUpdateRecord, workerguard.KindEnd:
		return true
	}
	return false
}

// deliveryPayloadOK: the relay's two payloads, exactly.
func deliveryPayloadOK(raw json.RawMessage) bool {
	var start struct {
		DefinitionVersionID uuid.UUID `json:"definition_version_id"`
		RecordID            uuid.UUID `json:"record_id"`
	}
	if strict(raw, &start) && start.DefinitionVersionID != uuid.Nil && start.RecordID != uuid.Nil {
		return true
	}
	var sig struct {
		Signal uuid.UUID `json:"signal"`
	}
	return strict(raw, &sig) && sig.Signal != uuid.Nil
}

func isDeliveryJSON(raw json.RawMessage) bool {
	var d outbox.Delivery
	return strict(raw, &d) && d.OutboxID != uuid.Nil && deliveryPayloadOK(d.Payload)
}

func isRunInputJSON(raw json.RawMessage) bool {
	var in workerguard.RunInput
	if !strict(raw, &in) || in.OutboxID == uuid.Nil || !deliveryPayloadOK(in.Payload) {
		return false
	}
	return len(in.Refs) == 0 || isUUIDObject(in.Refs)
}

// isUUIDObject: a JSON object whose every value is a canonical UUID string or
// a list of them.
func isUUIDObject(raw json.RawMessage) bool {
	var m map[string]json.RawMessage
	if !strict(raw, &m) || len(m) == 0 {
		return false
	}
	isUUID := func(r json.RawMessage) bool {
		var s string
		if json.Unmarshal(r, &s) != nil {
			return false
		}
		id, err := uuid.Parse(s)
		return err == nil && id.String() == s
	}
	for _, v := range m {
		if isUUID(v) {
			continue
		}
		var list []json.RawMessage
		if json.Unmarshal(v, &list) != nil {
			return false
		}
		for _, x := range list {
			if !isUUID(x) {
				return false
			}
		}
	}
	return true
}
