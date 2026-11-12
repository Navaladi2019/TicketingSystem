package workflowdefs

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
)

func TestPublishRefusesBeforeTouchingTheDatabase(t *testing.T) {
	// A nil tx would panic if either check let the call through.
	if _, err := (Store{}).Publish(context.Background(), nil, uuid.New(), PublishInput{Key: "triage", Body: []byte(`{}`)}); !errors.Is(err, ErrNoPublisher) {
		t.Errorf("no publisher: err = %v, want ErrNoPublisher", err)
	}
	_, err := (Store{}).Publish(context.Background(), nil, uuid.New(), PublishInput{Key: "Bad Key", PublishedBy: uuid.New(), Body: []byte(`{}`)})
	var inv *InvalidError
	if !errors.As(err, &inv) || len(inv.Fields) != 1 || inv.Fields[0].Pointer != "/key" || inv.Fields[0].Rule != RuleKey {
		t.Errorf("bad key: err = %v, want InvalidError at /key", err)
	}
	if _, err := (Store{}).GetCurrent(context.Background(), nil, uuid.New(), "Bad Key"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetCurrent with a malformed key: err = %v, want ErrNotFound", err)
	}
}

func TestPublishBodyThatDoesNotParseIsInvalidAndNamesNoValue(t *testing.T) {
	_, err := (Store{}).Publish(context.Background(), nil, uuid.New(), PublishInput{Key: "triage", PublishedBy: uuid.New(), Body: []byte(`{"format":`)})
	var inv *InvalidError
	if !errors.As(err, &inv) || len(inv.Fields) == 0 || inv.Fields[0].Rule != RuleInvalid {
		t.Fatalf("err = %v, want InvalidError", err)
	}
}

const sample = `{"format":"nava.workflow/v1","trigger":{"on":"record.created","entity":"incident"},"start":"wait",
"steps":[{"key":"wait","type":"timer","duration":"PT1H","next":"mark"},
{"key":"mark","type":"update_record","set":{"b_field":1,"a_field":2,"ok":3},"next":"done"},
{"key":"done","type":"end"}]}`

func TestCheckSetsNamesStepAndFieldInOrder(t *testing.T) {
	d, err := def.Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	got := checkSets(d, map[string]string{"ok": "string"})
	want := []FieldError{
		{Pointer: "/steps/mark/set/a_field", Rule: RuleUnknownField},
		{Pointer: "/steps/mark/set/b_field", Rule: RuleUnknownField},
	}
	if !slices.Equal(got, want) {
		t.Errorf("checkSets = %v, want %v", got, want)
	}
	if got := checkSets(d, map[string]string{"ok": "", "b_field": "", "a_field": ""}); len(got) != 0 {
		t.Errorf("every field present: %v", got)
	}
}

func TestParseFieldsNamesTheStepByKeyWhenItHasOne(t *testing.T) {
	body := []byte(`{"steps":[{"key":"wait","x":1},{"key":"Bad Key!"},{"key":7},{}]}`)
	got := parseFields(body, []string{"/steps/0/x", "/steps/1/key", "/steps/2", "/steps/3/key", "/steps/9", "/start", "", "/steps/0"})
	want := []string{"/steps/wait/x", "/steps/1/key", "/steps/2", "/steps/3/key", "/steps/9", "/start", "", "/steps/wait"}
	for i, f := range got {
		if f.Pointer != want[i] || f.Rule != RuleInvalid {
			t.Errorf("field %d = %+v, want pointer %q", i, f, want[i])
		}
	}
}

func TestEscapeIsRFC6901(t *testing.T) {
	if got := escape("a/b~c"); got != "a~1b~0c" {
		t.Errorf("escape = %q", got)
	}
}
