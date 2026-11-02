package server

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

func recFixedColumns() map[string]string {
	return map[string]string{"state": metadata.ColumnState, "owner": metadata.ColumnAssigneeID, "group": metadata.ColumnAssignmentGroupID}
}

// wrappedFK is what pipeline.Write returns for a foreign key violation: the
// pgx error under two %w wraps, with table and detail text in it.
func wrappedFK(constraint string) error {
	pg := &pgconn.PgError{
		Code: pgForeignKeyViolation, ConstraintName: constraint, TableName: "records",
		Message: `insert or update on table "records" violates foreign key constraint "` + constraint + `"`,
		Detail:  "Key (tenant_id, assignee_id)=(a, b) is not present in table \"principals\".",
	}
	return fmt.Errorf("pipeline: %w", fmt.Errorf("records/store: insert record: %w", pg))
}

func TestRecordErrorMapsAWrappedForeignKeyViolationToAFieldError(t *testing.T) {
	t.Parallel()
	h, logs := recHandler()
	ctx := t.Context()
	for constraint, pointer := range map[string]string{
		"records_assignee_fkey":         "/data/owner",
		"records_assignment_group_fkey": "/data/group",
	} {
		err := h.recordError(ctx, wrappedFK(constraint), recFixedColumns())
		var pe *problemError
		if !errors.As(err, &pe) || pe.code != problem.RecordInvalid {
			t.Fatalf("%s: err = %v, want a record_invalid problem", constraint, err)
		}
		if len(pe.fields) != 1 || pe.fields[0].Pointer != pointer || pe.fields[0].Rule != refRule {
			t.Errorf("%s: fields = %+v, want one %s error with rule %s", constraint, pe.fields, pointer, refRule)
		}
		if !strings.Contains(logs.String(), constraint) {
			t.Errorf("%s: the server log %q does not name the constraint", constraint, logs.String())
		}
		// Neither the error text nor the unwrap chain carries the database text.
		for _, s := range []string{err.Error(), fmt.Sprint(errors.Unwrap(err))} {
			if strings.Contains(s, "violates") || strings.Contains(s, "principals") || strings.Contains(s, constraint) {
				t.Errorf("%s: problem error text %q carries constraint text", constraint, s)
			}
		}
	}
}

func TestRecordErrorLeavesOtherForeignKeyViolationsAnInternalError(t *testing.T) {
	t.Parallel()
	h, _ := recHandler()
	ctx := t.Context()
	for name, tc := range map[string]struct {
		err   error
		fixed map[string]string
	}{
		"created_by is the caller, not the payload": {wrappedFK("records_created_by_fkey"), recFixedColumns()},
		"an unknown constraint":                     {wrappedFK("something_else_fkey"), recFixedColumns()},
		"no property maps to the column":            {wrappedFK("records_requester_fkey"), recFixedColumns()},
	} {
		got := h.recordError(ctx, tc.err, tc.fixed)
		var pe *problemError
		if errors.As(got, &pe) {
			t.Errorf("%s: mapped to the %s problem, want the raw error (a 500)", name, pe.code)
		}
	}
}

func TestRecordErrorMapsValidationForbiddenAndOthers(t *testing.T) {
	t.Parallel()
	h, _ := recHandler()
	ctx := t.Context()
	ve := fmt.Errorf("x: %w", &pipeline.ErrValidation{Fields: []metadata.FieldError{{Path: "/a~1b", Code: "required"}, {Path: "", Code: "type"}}})
	var pe *problemError
	if err := h.recordError(ctx, ve, recFixedColumns()); !errors.As(err, &pe) || pe.code != problem.RecordInvalid ||
		len(pe.fields) != 2 || pe.fields[0].Pointer != "/data/a~1b" || pe.fields[1].Pointer != "/data" {
		t.Errorf("validation = %v %+v", err, pe)
	}
	if err := h.recordError(ctx, authz.ErrForbidden, recFixedColumns()); !errors.As(err, &pe) || pe.code != problem.Forbidden {
		t.Errorf("forbidden = %v", err)
	}
	boom := errors.New("boom")
	if err := h.recordError(ctx, boom, recFixedColumns()); !errors.Is(err, boom) {
		t.Errorf("other = %v, want the error itself", err)
	}
}

func TestParseRecordBody(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct{ body, pointer string }{
		"not an object":    {`[]`, ""},
		"null":             {`null`, ""},
		"no data":          {`{}`, "/data"},
		"data not object":  {`{"data": 5}`, "/data"},
		"data null":        {`{"data": null}`, "/data"},
		"data is an array": {`{"data": []}`, "/data"},
	} {
		_, pe := parseRecordBody([]byte(tc.body))
		if pe == nil || pe.code != problem.RecordInvalid || len(pe.fields) != 1 || pe.fields[0].Pointer != tc.pointer {
			t.Errorf("%s: got %v, want one record_invalid error at %q", name, pe, tc.pointer)
		}
	}
	data, pe := parseRecordBody([]byte(`{"data": {"n": 12345678901234567890}}`))
	if pe != nil || fmt.Sprint(data["n"]) != "12345678901234567890" {
		t.Errorf("big number = %v %v, want it kept exact", data, pe)
	}
}

func recHandler() (*handlers, *bytes.Buffer) {
	var logs bytes.Buffer
	return &handlers{log: slog.New(slog.NewTextHandler(&logs, nil))}, &logs
}

// A request that reaches a records handler with no principal id is a wiring
// bug (errNoPrincipal), for reads as for writes.
func TestRecordHandlersRefuseAPrincipalWithoutAnID(t *testing.T) {
	t.Parallel()
	h, _ := recHandler()
	ctx := identity.NewContext(t.Context(), identity.Principal{TenantID: uuid.New()})
	if _, err := h.GetRecord(ctx, api.GetRecordRequestObject{Entity: "ticket", Id: uuid.NewString()}); !errors.Is(err, errNoPrincipal) {
		t.Errorf("GetRecord err = %v, want errNoPrincipal", err)
	}
	if _, err := h.CreateRecord(ctx, api.CreateRecordRequestObject{Entity: "ticket"}); !errors.Is(err, errNoPrincipal) {
		t.Errorf("CreateRecord err = %v, want errNoPrincipal", err)
	}
	if _, err := h.GetRecord(t.Context(), api.GetRecordRequestObject{}); !errors.Is(err, errNoPrincipal) {
		t.Errorf("GetRecord without a principal err = %v, want errNoPrincipal", err)
	}
}
