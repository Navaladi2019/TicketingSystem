package server

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

func ptr[T any](v T) *T { return &v }

func TestIfMatchVersions(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		in   *string
		want []int64
		ok   bool
	}{
		"absent":       {nil, nil, false},
		"empty":        {ptr(""), nil, false},
		"star":         {ptr("*"), nil, false},
		"star padded":  {ptr(" * "), nil, false},
		"weak":         {ptr(`W/"3"`), []int64{3}, true},
		"strong":       {ptr(`"3"`), []int64{3}, true},
		"list":         {ptr(`W/"1", "2"`), []int64{1, 2}, true},
		"garbage":      {ptr(`W/"abc"`), []int64{0}, true},
		"zero":         {ptr(`"0"`), []int64{0}, true},
		"negative":     {ptr(`"-4"`), []int64{0}, true},
		"star in list": {ptr(`"1", *`), []int64{1, 0}, true},
	} {
		got, ok := ifMatchVersions(tc.in)
		if ok != tc.ok || !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %v %v, want %v %v", name, got, ok, tc.want, tc.ok)
		}
	}
}

func TestParsePatchBody(t *testing.T) {
	t.Parallel()
	good, bad := map[string]string{
		`{"data": {"a": 1, "state": "x"}}`:            "",
		`{"data": {"a": null}}`:                       "",
		`{"state": "closed"}`:                         "",
		`{"data": {"state": "x"}, "state": "closed"}`: "closed",
	}, map[string]string{
		`{}`:                       "/data",
		`null`:                     "",
		`[]`:                       "",
		`"x"`:                      "",
		`{"data": null}`:           "/data",
		`{"data": []}`:             "/data",
		`{"data": "x"}`:            "/data",
		`{"state": null}`:          "/state",
		`{"state": 3}`:             "/state",
		`{"data": {}, "state": 1}`: "/state",
	}
	for body, wantState := range good {
		patch, err := parsePatchBody(json.RawMessage(body), "state")
		if err != nil {
			t.Errorf("%s: %v", body, err)
			continue
		}
		if wantState != "" && patch["state"] != wantState {
			t.Errorf("%s: state = %v, want %s (top level wins)", body, patch["state"], wantState)
		}
	}
	for body, pointer := range bad {
		_, err := parsePatchBody(json.RawMessage(body), "state")
		if err == nil || err.code != problem.RecordInvalid || len(err.fields) != 1 || err.fields[0].Pointer != pointer {
			t.Errorf("%s: err = %+v, want record_invalid at %q", body, err, pointer)
		}
	}
}

func TestChangedPaths(t *testing.T) {
	t.Parallel()
	before := map[string]any{"a": "1", "gone": true, "tags": []any{"x"}, "same": 1, "o": map[string]any{"k": 1, "j": 2}, "a/b": 1}
	after := map[string]any{"a": "2", "new": 1, "tags": []any{"x", "y"}, "same": 1, "o": map[string]any{"k": 1, "j": 3}, "a/b": 1}
	got := ChangedPaths(before, after)
	want := []string{"/data/a", "/data/gone", "/data/new", "/data/o/j", "/data/tags"}
	if !slices.Equal(got, want) {
		t.Errorf("changed = %v, want %v", got, want)
	}
	if got := ChangedPaths(nil, map[string]any{"a/b": 1}); !slices.Equal(got, []string{"/data/a~1b"}) {
		t.Errorf("create changed = %v", got)
	}
	if got := ChangedPaths(map[string]any{}, map[string]any{}); got == nil || len(got) != 0 {
		t.Errorf("no change = %#v, want an empty non-nil list", got)
	}
}

func TestHistoryPositionRoundTrips(t *testing.T) {
	t.Parallel()
	at, id := time.Date(2026, 11, 2, 1, 2, 3, 456789000, time.UTC), uuid.New()
	gotAt, gotID, err := decodeHistoryPos(encodeHistoryPos(at, id))
	if err != nil || !gotAt.Equal(at) || gotID != id {
		t.Errorf("round trip = %v %v %v", gotAt, gotID, err)
	}
	for _, b := range []string{"", "x", "1:", "a:" + id.String(), "1:" + uuid.Nil.String(), "1:not-a-uuid"} {
		if _, _, err := decodeHistoryPos([]byte(b)); err == nil {
			t.Errorf("%q decoded", b)
		}
	}
}

func TestRecordErrorMapsTransitionAndUnique(t *testing.T) {
	t.Parallel()
	h, _ := recHandler()
	var pe *problemError
	err := h.recordError(t.Context(), errors.Join(errors.New("x"), &pipeline.ErrIllegalTransition{From: "closed", To: "new"}), nil, "state")
	if !errors.As(err, &pe) || pe.code != problem.IllegalTransition || len(pe.fields) != 1 || pe.fields[0].Pointer != "/data/state" {
		t.Errorf("transition = %v %+v", err, pe)
	}
	err = h.recordError(t.Context(), &pipeline.ErrUniqueViolation{Field: "a/b"}, nil, "state")
	if !errors.As(err, &pe) || pe.code != problem.UniqueViolation || pe.fields[0].Pointer != "/data/a~1b" {
		t.Errorf("unique = %v %+v", err, pe)
	}
}

func TestPatchAndHistoryRefuseAPrincipalWithoutAnID(t *testing.T) {
	t.Parallel()
	h, _ := recHandler()
	if _, err := h.PatchRecord(t.Context(), api.PatchRecordRequestObject{}); !errors.Is(err, errNoPrincipal) {
		t.Errorf("PatchRecord err = %v", err)
	}
	if _, err := h.GetRecordHistory(t.Context(), api.GetRecordHistoryRequestObject{}); !errors.Is(err, errNoPrincipal) {
		t.Errorf("GetRecordHistory err = %v", err)
	}
}
