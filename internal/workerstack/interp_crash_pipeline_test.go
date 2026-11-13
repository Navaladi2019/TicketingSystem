//go:build workerstack && interpcrash

package workerstack

// The S11b crash case (P3-44, TICK-713): P3-24's crash test, against the real
// records pipeline instead of the S11a fake writer.
//
// nava-worker-interptest runs with NAVA_INTERPTEST_PORTS=db: the production
// ports (the database DefinitionStore, recordports over the real
// pipeline.Write with the real EventSink), plus the interptest build's
// pause-after-commit hook. Tenant A's admin publishes, through nava-core, a
// definition that triggers on record.created of an incident and whose start
// step is update_record state=in_progress. Alice creates an incident through
// nava-core; the pipeline's sink enqueues the start, the stack's nava-relay
// dispatches it, and the worker's ApplyStep commits the write and blocks. The
// test SIGKILLs the container while it is blocked, restarts it, and Temporal
// retries the attempt the dead worker held. The retry calls pipeline.Write
// with the same key, which replays the stored result instead of writing again.
//
// Everything asserted is read outside the killed process: the database
// (records, audit_log, record_write_keys, workflow_ref, as the superuser
// through psql), the API (nava-core), and the Temporal history (attempt > 1).
//
// The definition stays published after the test, so the cleanup publishes a
// retiring version (a 24 h timer, then end) in its place: a later incident in
// tenant A (the S11b Bruno folder) starts no write from it.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"

	"github.com/Navaladi2019/TicketingSystem/internal/relay"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// The seeded tenant A (docs/acceptance/guide/tools/seed.sh) and its dev-only
// seed users (docs/acceptance/guide/bruno/environments/local.bru).
const (
	seedTenantA   = "00000000-0000-4000-8000-0000000000a1"
	seedAdminUser = "admin@tenant-a.test"
	seedAdminPass = "dev-only-admin-a"
	seedAliceUser = "alice@tenant-a.test"
	seedAlicePass = "dev-only-alice"
)

// apiURL is a published host port of the stack: the env variable a stack on
// other ports exports (make ci-isolated, the self-hosted runner), or the
// compose default.
func apiURL(env, def string) string {
	if p := os.Getenv(env); p != "" {
		return "http://127.0.0.1:" + p
	}
	return "http://127.0.0.1:" + def
}

// stackAPI calls nava-core with tokens from the stack's Keycloak.
type stackAPI struct {
	t              *testing.T
	core, keycloak string
	hc             *http.Client
}

func newStackAPI(t *testing.T) *stackAPI {
	return &stackAPI{t: t, core: apiURL("NAVA_CORE_PORT", "18080"), keycloak: apiURL("KEYCLOAK_PORT", "18081"),
		hc: &http.Client{Timeout: 30 * time.Second}}
}

// token is a password-grant access token of realm nava, client nava-web.
func (a *stackAPI) token(user, password string) string {
	a.t.Helper()
	form := url.Values{"grant_type": {"password"}, "client_id": {"nava-web"}, "scope": {"openid"},
		"username": {user}, "password": {password}}
	resp, err := a.hc.PostForm(a.keycloak+"/realms/nava/protocol/openid-connect/token", form)
	if err != nil {
		a.t.Fatalf("token for %s: %v", user, err)
	}
	defer resp.Body.Close()
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil || resp.StatusCode != http.StatusOK || tok.AccessToken == "" {
		a.t.Fatalf("token for %s: HTTP %d, %v", user, resp.StatusCode, err)
	}
	return tok.AccessToken
}

// call sends body (JSON, or none) to nava-core and decodes the answer into
// out. A status other than want fails the test with the body.
func (a *stackAPI) call(method, path, token string, header map[string]string, body any, want int, out any) {
	a.t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			a.t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	// Not t.Context(): the cleanup's retiring publish runs after it is
	// cancelled. The client's 30 s timeout bounds every call.
	req, err := http.NewRequestWithContext(context.Background(), method, a.core+path, rd)
	if err != nil {
		a.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := a.hc.Do(req)
	if err != nil {
		a.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		a.t.Fatalf("%s %s: HTTP %d, want %d: %s", method, path, resp.StatusCode, want, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			a.t.Fatalf("%s %s: %v: %s", method, path, err, raw)
		}
	}
}

// publishedVersion is the part of the publish answer the test reads.
type publishedVersion struct {
	Version   int                  `json:"version"`
	VersionID uuid.UUID            `json:"version_id"`
	StepIDs   map[string]uuid.UUID `json:"step_ids"`
}

// seedStack runs the acceptance seed (tenants A and B, the incident type, the
// admins group, one group sync): idempotent, so it is safe on a stack the
// S9 and S10 cases already seeded.
func seedStack(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "docs/acceptance/guide/tools/seed.sh")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tools/seed.sh: %v\n%s", err, out)
	}
}

// TestInterpreterCrashRealPipelineAppliesOnce: the worker is killed after the
// real pipeline.Write committed the update_record step and before ApplyStep
// completed, then restarted. The run completes; the incident is in_progress at
// version 2 with exactly one source=workflow audit entry and one
// record_write_keys row for the step's key; Temporal ran ApplyStep more than
// once.
func TestInterpreterCrashRealPipelineAppliesOnce(t *testing.T) {
	awaitReady(t)
	seedStack(t)
	api := newStackAPI(t)
	admin := api.token(seedAdminUser, seedAdminPass)
	run := fmt.Sprintf("%08x", rand.Uint32())
	key := "crash_" + run
	defs := "/api/v1/workflow-definitions/" + key + "/versions"

	// 1. The definition: on an incident's create, update_record at once.
	var v1 publishedVersion
	api.call(http.MethodPost, defs, admin, map[string]string{"Idempotency-Key": "tick713-crash-" + run + "-v1"},
		map[string]any{"format": "nava.workflow/v1", "trigger": map[string]any{"on": "record.created", "entity": "incident"},
			"start": "apply", "steps": []map[string]any{
				{"key": "apply", "type": "update_record", "set": map[string]any{"state": "in_progress"}, "next": "done"},
				{"key": "done", "type": "end"},
			}}, http.StatusCreated, &v1)
	if v1.StepIDs["apply"] == uuid.Nil {
		t.Fatalf("publish: no step id for apply: %+v", v1)
	}
	t.Cleanup(func() {
		// A fresh token: the admin's may have expired by now.
		api.call(http.MethodPost, defs, api.token(seedAdminUser, seedAdminPass),
			map[string]string{"Idempotency-Key": "tick713-crash-" + run + "-retire"},
			map[string]any{"format": "nava.workflow/v1", "trigger": map[string]any{"on": "record.created", "entity": "incident"},
				"start": "hold", "steps": []map[string]any{
					{"key": "hold", "type": "timer", "duration": "PT24H", "next": "done"},
					{"key": "done", "type": "end"},
				}}, http.StatusCreated, nil)
	})

	// 2. The interptest worker on the production ports, pause hook on, as the
	// only nava-tq poller (the always-on nava-worker is stopped).
	st := newCrashState()
	w := startInterpWorker(t, append(st.env(), "NAVA_INTERPTEST_PORTS=db")...)
	clients := newInterpClients(t)
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
	defer cancel()
	if out, err := w.oneShot(ctx, "0", "sh", "-c", fmt.Sprintf("mkdir -p %s && chown -R 10001:10001 %s", st.pause, st.base)); err != nil {
		t.Fatalf("create %s: %v\n%s", st.base, err, out)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if out, err := w.oneShot(ctx, "0", "rm", "-rf", st.base); err != nil {
			t.Errorf("remove %s: %v\n%s", st.base, err, out)
		}
	})

	// 3. Alice creates an incident: the real pipeline's sink enqueues the
	// start of v1 for it.
	alice := api.token(seedAliceUser, seedAlicePass)
	var rec struct {
		ID      uuid.UUID `json:"id"`
		Version int64     `json:"version"`
		State   string    `json:"state"`
	}
	begin := time.Now()
	api.call(http.MethodPost, "/api/v1/records/incident", alice, map[string]string{"Idempotency-Key": "tick713-crash-" + run + "-create"},
		map[string]any{"data": map[string]any{"short_description": "TICK-713 crash " + run}}, http.StatusCreated, &rec)
	if rec.Version != 1 || rec.State != "new" {
		t.Fatalf("created incident %+v, want version 1, state new", rec)
	}

	// The start row the sink enqueued for this record and v1.
	var start relay.Claimed
	deadline := time.Now().Add(30 * time.Second)
	for {
		out := psql(t, fmt.Sprintf(`SELECT id, target_key FROM outbox WHERE tenant_id = '%s' AND kind = '%s'
  AND payload->>'record_id' = '%s' AND payload->>'definition_version_id' = '%s';`, seedTenantA, interp.WorkflowType, rec.ID, v1.VersionID))
		if id, target, ok := strings.Cut(out, "|"); ok && !strings.Contains(target, "\n") {
			start = relay.Claimed{ID: uuid.MustParse(id), TenantID: uuid.MustParse(seedTenantA), Kind: interp.WorkflowType, TargetKey: target}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no single start row for record %s and version %s within 30 s: %q", rec.ID, v1.VersionID, out)
		}
		time.Sleep(250 * time.Millisecond)
	}
	wfID, ok := relay.WorkflowID(start)
	if !ok {
		t.Fatalf("relay.WorkflowID refused the start row %+v", start)
	}
	stepKey := interp.IdempotencyKey(wfID, v1.StepIDs["apply"])
	marker := containerName(st.pause, interp.PauseMarker, stepKey)

	// 4. ApplyStep committed through the real pipeline and is blocked.
	deadline = time.Now().Add(60 * time.Second)
	for {
		if _, err := composeIn(ctx, w.env, "exec", "-T", interpService, "test", "-f", marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			logs, _ := composeIn(ctx, w.env, "logs", "--no-color", "--tail", "100", interpService)
			t.Fatalf("no pause marker %s within 60 s: ApplyStep never reached the pause after commit\n%s", marker, logs)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if got := psql(t, fmt.Sprintf(`SELECT version || '|' || state FROM records WHERE tenant_id = '%s' AND id = '%s';`, seedTenantA, rec.ID)); got != "2|in_progress" {
		t.Fatalf("while paused after the commit: version|state %q, want 2|in_progress (the write committed)", got)
	}
	before := w.state(ctx)

	// 5. A real container kill while it is blocked.
	if out, err := composeIn(ctx, w.env, "kill", "-s", "SIGKILL", interpService); err != nil {
		t.Fatalf("compose kill %s: %v\n%s", interpService, err, out)
	}
	killed := w.state(ctx)
	if killed.running != "false" || killed.exitCode != "137" {
		t.Fatalf("after the kill: running %s exit %s, want false 137 (SIGKILL)", killed.running, killed.exitCode)
	}

	// 6. Release the hook for the retry, then restart the same container.
	release := containerName(st.pause, interp.ReleaseFile, stepKey)
	if out, err := w.oneShot(ctx, "10001", "touch", release); err != nil {
		t.Fatalf("create %s: %v\n%s", release, err, out)
	}
	if out, err := composeIn(ctx, w.env, "start", interpService); err != nil {
		t.Fatalf("compose start %s: %v\n%s", interpService, err, out)
	}
	restarted := w.state(ctx)
	for restarted.health != "healthy" {
		if ctx.Err() != nil {
			t.Fatalf("%s not healthy after the restart: %+v", interpService, restarted)
		}
		time.Sleep(time.Second)
		restarted = w.state(ctx)
	}
	if restarted.id != before.id || restarted.startedAt == before.startedAt {
		t.Fatalf("worker container before %+v, after %+v: want the same container started again", before, restarted)
	}

	// 7. The retry completes the run.
	if err := clients.result(t, wfID); err != nil {
		t.Fatalf("nava.interpreter %s after the restart: %v", wfID, err)
	}
	elapsed := time.Since(begin)

	// 8. Out of the killed process: Temporal, the database, the API.
	attempts := applyStepAttempts(ctx, t, clients, wfID)
	row := psql(t, fmt.Sprintf(`SELECT r.version || '|' || r.state
  || '|' || (SELECT count(*) FROM audit_log a WHERE a.tenant_id = r.tenant_id AND a.subject_id = r.id AND a.source = 'workflow')
  || '|' || (SELECT count(*) FROM record_write_keys k WHERE k.tenant_id = r.tenant_id AND k.idempotency_key = '%s')
  || '|' || (SELECT count(*) FROM record_write_keys k WHERE k.tenant_id = r.tenant_id AND k.record_id = r.id AND k.action = 'update')
  || '|' || (SELECT bool_and(closed_at IS NOT NULL) FROM workflow_ref f WHERE f.tenant_id = r.tenant_id AND f.outbox_id = '%s')
FROM records r WHERE r.tenant_id = '%s' AND r.id = '%s';`, stepKey, start.ID, seedTenantA, rec.ID))
	var got struct {
		Version int64  `json:"version"`
		State   string `json:"state"`
	}
	api.call(http.MethodGet, "/api/v1/records/incident/"+rec.ID.String(), api.token(seedAdminUser, seedAdminPass), nil, nil, http.StatusOK, &got)
	t.Logf("kill evidence: container %s exit %s, restarted (started %s, then %s); ApplyStep attempts %d; "+
		"version|state|workflow audit entries|keys for the step|update keys|ref closed = %s; API version %d state %s; %v from create to closed run",
		before.id[:12], killed.exitCode, before.startedAt, restarted.startedAt, attempts, row, got.Version, got.State, elapsed.Round(time.Second))

	if attempts < 2 {
		t.Errorf("ApplyStep attempts %d, want >= 2: the retry ran after the kill", attempts)
	}
	if row != "2|in_progress|1|1|1|true" {
		t.Errorf("version|state|workflow audit entries|keys for the step|update keys|ref closed = %q, want 2|in_progress|1|1|1|true: the effect applied once", row)
	}
	if got.Version != 2 || got.State != "in_progress" {
		t.Errorf("GET the incident: version %d state %s, want 2 in_progress", got.Version, got.State)
	}
	if elapsed > crashCaseBudget {
		t.Errorf("create to closed run took %v, want <= %v", elapsed.Round(time.Second), crashCaseBudget)
	}
}

// applyStepAttempts is the highest attempt of an ApplyStep activity in the
// run's history (the started events carry the attempt).
func applyStepAttempts(ctx context.Context, t *testing.T, c interpClients, wfID string) int32 {
	t.Helper()
	apply := map[int64]bool{}
	var most int32
	it := c.system.GetWorkflowHistory(ctx, wfID, "", false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for it.HasNext() {
		e, err := it.Next()
		if err != nil {
			t.Fatalf("history of %s: %v", wfID, err)
		}
		if a := e.GetActivityTaskScheduledEventAttributes(); a != nil && a.GetActivityType().GetName() == interp.ApplyStepActivity {
			apply[e.GetEventId()] = true
		}
		if a := e.GetActivityTaskStartedEventAttributes(); a != nil && apply[a.GetScheduledEventId()] {
			most = max(most, a.GetAttempt())
		}
	}
	return most
}
