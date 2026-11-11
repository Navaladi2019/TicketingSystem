//go:build integration

// P3-35 (TICK-620), contract amendment A4: on create, when the source is user
// or ai and the payload has no requester, the pipeline sets requester_id to
// the caller before the write check, so a requester can raise an incident
// under the default ACL (requester read and write where state is new).

package server_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// requesterOnlyACL lets a requester read and write its own tickets while new
// and nobody else do anything: the shape of the default incident ACL's
// requester rules.
func (e *etEnv) requesterOnlyACL(t *testing.T) {
	t.Helper()
	e.publishACL(t, e.a.id, "ticket", authz.Rule{
		ID: "requester-new", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Requester: true},
		Where: []authz.Cond{{Column: "state", Op: authz.CondEq, Values: []string{"new"}}},
	})
}

func (e *etEnv) requesterOf(t *testing.T, id string) *uuid.UUID {
	t.Helper()
	var r *uuid.UUID
	if err := e.super.QueryRow(t.Context(), "SELECT requester_id FROM records WHERE tenant_id = $1 AND id = $2", e.a.id, id).Scan(&r); err != nil {
		t.Fatal(err)
	}
	return r
}

// A user who sends no requester and no assignment group raises a ticket: 201,
// and requester is the caller, in the body and in the row.
func TestCreateWithoutARequesterDefaultsToTheCaller(t *testing.T) {
	e := newRecEnv(t)
	e.requesterOnlyACL(t)

	rec := e.createRec(t, "user-a", "ticket", "req-default", `{"short_description": "raised by a requester"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create with no requester and no group = %d, want 201: %s", rec.Code, rec.Body)
	}
	var b recBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if got := b.Data["requester"]; got != e.a.user.PrincipalID.String() {
		t.Errorf("body requester = %v, want the caller %s", got, e.a.user.PrincipalID)
	}
	if got := e.requesterOf(t, b.ID); got == nil || *got != e.a.user.PrincipalID {
		t.Errorf("records.requester_id = %v, want the caller", got)
	}
	// The same user reads it back (the requester rule), and another user does not.
	if r := e.do(t, http.MethodGet, "/api/v1/records/ticket/"+b.ID, "user-a", "", ""); r.Code != http.StatusOK {
		t.Errorf("the requester reads its own ticket = %d", r.Code)
	}
	if r := e.do(t, http.MethodGet, "/api/v1/records/ticket/"+b.ID, "lookalike-a", "", ""); r.Code != http.StatusNotFound {
		t.Errorf("another user reads it = %d, want 404", r.Code)
	}
}

// An explicit requester is kept as sent and goes through the write check like
// any value: a non-admin naming another user is refused, and nothing is written.
func TestCreateNamingAnotherRequesterIsForbiddenForANonAdmin(t *testing.T) {
	e := newRecEnv(t)
	e.requesterOnlyACL(t)
	before := e.recordCount(t, "a")

	rec := e.createRec(t, "user-a", "ticket", "req-other", `{"short_description": "for someone else", "requester": "`+e.a.lookalike.PrincipalID.String()+`"}`)
	wantProblem(t, rec, http.StatusForbidden, problem.Forbidden)
	if got := e.recordCount(t, "a"); got != before {
		t.Errorf("records %d -> %d: a refused create wrote", before, got)
	}
	// Naming itself is the same as the default.
	ok := e.createRec(t, "user-a", "ticket", "req-self", `{"short_description": "mine", "requester": "`+e.a.user.PrincipalID.String()+`"}`)
	if ok.Code != http.StatusCreated {
		t.Errorf("naming itself = %d, want 201: %s", ok.Code, ok.Body)
	}
}

// A system source has no principal to name: the requester stays null even when
// the command carries one.
func TestSystemCreateLeavesTheRequesterNull(t *testing.T) {
	e := newRecEnv(t)
	p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, pipeline.PoolRunner{Pool: e.pool}, pipeline.SystemClock{})
	for _, actor := range []identity.Principal{{TenantID: e.a.id}, e.a.user} {
		var res pipeline.Result
		if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
			var err error
			res, err = p.Write(t.Context(), tx, pipeline.Command{Op: pipeline.OpCreate, EntityKey: "ticket",
				Data: map[string]any{"short_description": "by the system"}, Actor: actor, Source: pipeline.SourceSystem})
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if got := e.requesterOf(t, res.RecordID.String()); got != nil {
			t.Errorf("system create (actor %v) has requester %v, want null", actor.PrincipalID, got)
		}
	}
}
