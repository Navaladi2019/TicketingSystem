//go:build integration

// TICK-487: every tenant write route of the contract completes its idempotency
// key inside its own transaction (idempotency.CompleteInTx). The write-chain
// tripwire has no database and never reaches a handler, so this test, against
// real PostgreSQL, is what catches a route that skips the call and leaves the
// response to the after-commit fallback.

package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// idempotencyResult reads nava.idempotency.requests{result=<result>}.
func idempotencyResult(t *testing.T, tel *telemetry.Telemetry, result string) float64 {
	t.Helper()
	fams, err := tel.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var n float64
	for _, f := range fams {
		if !strings.HasPrefix(f.GetName(), "nava_idempotency_requests") {
			continue
		}
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "result" && l.GetValue() == result {
					n += m.GetCounter().GetValue()
				}
			}
		}
	}
	return n
}

// TestEveryTenantWriteRouteCompletesItsKeyInTheWrite sends one successful
// request to each tenant write route of the contract and checks that the key is
// completed the moment the handler returns and the fallback did not run.
// A route of the contract with no driver below fails the test, so a new write
// route must be added here, and so must be proven to call CompleteInTx.
func TestEveryTenantWriteRouteCompletesItsKeyInTheWrite(t *testing.T) {
	e := newPatchEnv(t)
	etCreateIncident(t, e)
	rec := e.mustCreateRec(t, "user-a", "job", "seed-rec", `{"short_description": "one"}`)

	drivers := map[string]func(key string) *httptest.ResponseRecorder{
		"POST /api/v1/entity-types": func(key string) *httptest.ResponseRecorder {
			return e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", key, etBody("widget", "WGT"))
		},
		"POST /api/v1/entity-types/{key}/versions": func(key string) *httptest.ResponseRecorder {
			body := etPublishBody(t, `"description": "a <b>&amp; tag"`, `"description": "changed"`)
			return e.do(t, http.MethodPost, etVersionsPath, "admin-a", key, body)
		},
		"POST /api/v1/records/{entity}": func(key string) *httptest.ResponseRecorder {
			return e.createRec(t, "user-a", "job", key, `{"short_description": "two"}`)
		},
		"PATCH /api/v1/records/{entity}/{id}": func(key string) *httptest.ResponseRecorder {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/records/job/"+rec.ID, strings.NewReader(`{"data": {"note": "n"}}`))
			req.Header.Set("Authorization", "Bearer user-a")
			req.Header.Set("Content-Type", "application/merge-patch+json")
			req.Header.Set("Idempotency-Key", key)
			req.Header.Set("If-Match", `W/"1"`)
			w := httptest.NewRecorder()
			e.h.ServeHTTP(w, req)
			return w
		},
		"PUT /api/v1/entity-types/{key}/access": func(key string) *httptest.ResponseRecorder {
			// Keeps the seeded principals able to use the types, as the test ACL did.
			body := `{"rules": [{"id": "k487", "ops": ["read", "write"], "who": {"users": ["` + e.a.admin.PrincipalID.String() + `", "` + e.a.user.PrincipalID.String() + `"]}}]}`
			return e.access(t, http.MethodPut, "/api/v1/entity-types/incident/access", "admin-a", key, `W/"1"`, body)
		},
		"POST /api/v1/number-formats": func(key string) *httptest.ResponseRecorder {
			return e.do(t, http.MethodPost, nfPath, "admin-a", key, nfMonthly)
		},
		"PUT /api/v1/number-formats/{id}": func(key string) *httptest.ResponseRecorder {
			seed := e.do(t, http.MethodPost, nfPath, "admin-a", key+"-seed", nfDefault)
			if seed.Code != http.StatusCreated {
				t.Fatalf("seed format = %d: %s", seed.Code, seed.Body)
			}
			return e.doIf(t, http.MethodPut, nfPath+"/"+nfID(t, seed.Body.Bytes()), "admin-a", key, `W/"1"`, nfDefault)
		},
		"POST /api/v1/audit/chain:reseed": func(key string) *httptest.ResponseRecorder {
			// A tenant with a dead chain head (R17: next_seal_at over 10 minutes late).
			ctx := t.Context()
			if _, err := e.super.Exec(ctx, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, $2) ON CONFLICT DO NOTHING", e.a.id, uuid.New()); err != nil {
				t.Fatal(err)
			}
			o := uuid.Must(uuid.NewV7())
			if _, err := e.super.Exec(ctx, `INSERT INTO outbox (id, tenant_id, kind, kind_version, dedupe_key, target_key, target_seq,
				causation_id, causation_depth, payload) VALUES ($1::uuid, $2::uuid, 'audit.seal', 1, 'audit.seal:' || $3::text, 'audit.seal:' || $3::text,
				(SELECT version FROM audit_chain_head WHERE tenant_id = $2::uuid), $1::uuid, 0, '{}')`, o, e.a.id, o.String()); err != nil {
				t.Fatal(err)
			}
			if _, err := e.super.Exec(ctx, "UPDATE audit_chain_head SET next_seal_outbox_id = $2, next_seal_at = now() - interval '11 minutes' WHERE tenant_id = $1", e.a.id, o); err != nil {
				t.Fatal(err)
			}
			return e.do(t, http.MethodPost, "/api/v1/audit/chain:reseed", "admin-a", key, "")
		},
	}

	_, tenant := contractWriteOps(t)
	var routes []string
	for _, op := range tenant {
		routes = append(routes, op.String())
		drive, ok := drivers[op.String()]
		if !ok {
			t.Errorf("%s: no driver in this test; add one that sends a successful request and proves the route calls idempotency.CompleteInTx", op)
			continue
		}
		t.Run(op.String(), func(t *testing.T) {
			key := fmt.Sprintf("k-487-%s-%s", op.Method, strings.NewReplacer("/", "-", "{", "", "}", "").Replace(op.Path))
			before := idempotencyResult(t, e.tel, "stored_after_commit")
			resp := drive(key)
			if resp.Code < 200 || resp.Code > 299 {
				t.Fatalf("%s = %d, want 2xx: %s", op, resp.Code, resp.Body)
			}
			var state string
			var status int
			if err := e.super.QueryRow(t.Context(),
				"SELECT state, response_status FROM idempotency_keys WHERE key = $1", key).Scan(&state, &status); err != nil {
				t.Fatal(err)
			}
			if state != "completed" || status != resp.Code {
				t.Errorf("key = %s/%d after the handler returned, want completed/%d (the route does not call CompleteInTx)", state, status, resp.Code)
			}
			if got := idempotencyResult(t, e.tel, "stored_after_commit"); got != before {
				t.Errorf("stored_after_commit moved by %v, want 0: the response was stored by the fallback", got-before)
			}
		})
	}
	for name := range drivers {
		if !slices.Contains(routes, name) {
			t.Errorf("driver %q matches no tenant write route of the contract", name)
		}
	}
}
