//go:build workerstack

// Package workerstack tests the nava-worker compose service (T31b, TICK-159)
// in the running stack: it is ready, its admin listener serves /metrics and
// the probes untraced and unpublished, and it runs the audit sealer's
// activities through the T31 guard and the nava_worker database role.
//
//	make stack-up
//	make stack-test-worker
//
// No test here polls nava-tq beside the always-on nava-worker; the
// interpreter cases stop it first (the nava-tq rule, TestWorkerServiceSealsThroughTheGuard).
package workerstack

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

// The namespace and the always-on worker's compose service.
const (
	namespace = "nava"
	service   = "nava-worker"
	// The guard's allow decisions on activities, as Prometheus names them.
	decisions = "nava_worker_guard_decisions_total"
)

// root is the repository root, from this package.
const root = "../.."

// compose runs `docker compose` from the repository root, so COMPOSE_FILE and
// COMPOSE_PROJECT_NAME from make apply as they do to the shell stack tests.
func compose(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	return string(out), err
}

// admin returns the raw HTTP response (status line, headers, body) for path
// on the worker's admin listener, from inside its container: the port is
// never published.
func admin(t *testing.T, path string) string {
	t.Helper()
	req := "GET " + path + " HTTP/1.0\r\nHost: localhost\r\n\r\n"
	out, err := compose(t, req, "exec", "-T", service, "nc", "127.0.0.1", "9090")
	if err != nil {
		t.Fatalf("GET %s on the %s admin listener: %v", path, service, err)
	}
	return strings.ReplaceAll(out, "\r", "")
}

func status(resp string) string {
	line, _, _ := strings.Cut(resp, "\n")
	f := strings.Fields(line)
	if len(f) < 2 {
		return ""
	}
	return f[1]
}

// awaitReady waits for /readyz 200: the service started its worker, and
// PostgreSQL and Temporal answer.
func awaitReady(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		resp := admin(t, "/readyz")
		if status(resp) == "200" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s /readyz not 200 within 2m:\n%s\n/healthz:\n%s", service, resp, admin(t, "/healthz"))
		}
		time.Sleep(time.Second)
	}
}

func TestWorkerServiceReadyAndAdminListener(t *testing.T) {
	awaitReady(t)

	health := admin(t, "/healthz")
	for _, want := range []string{`"postgres":"ok"`, `"temporal":"ok"`} {
		if !strings.Contains(health, want) {
			t.Errorf("/healthz has no %s:\n%s", want, health)
		}
	}

	metrics := admin(t, "/metrics")
	if status(metrics) != "200" {
		t.Fatalf("/metrics:\n%s", metrics)
	}
	head, _, _ := strings.Cut(metrics, "\n\n")
	// httpobs.Middleware sets X-Request-Id on every response; the admin
	// listener is not wrapped in it (TICK-94 item 1).
	for _, h := range []string{"x-request-id:", "traceparent:"} {
		if strings.Contains(strings.ToLower(head), h) {
			t.Errorf("/metrics answered with %s: the admin listener is wrapped in httpobs:\n%s", h, head)
		}
	}
	if !strings.Contains(metrics, "temporal_") {
		t.Error("/metrics has no Temporal SDK metric: the client's MetricsHandler is not telemetry.NewTemporal's")
	}

	// Nothing of the service is published to the host.
	if out, err := compose(t, "", "port", service, "9090"); err == nil && strings.TrimSpace(out) != "" {
		t.Errorf("%s publishes the admin port: %s", service, out)
	}
}

// activityDecisions sums the guard's decisions on activities from one /metrics
// scrape: allow, and every other outcome (a denial or an internal error).
func activityDecisions(t *testing.T) (allow, other float64) {
	t.Helper()
	for line := range strings.SplitSeq(admin(t, "/metrics"), "\n") {
		if !strings.HasPrefix(line, decisions+"{") || !strings.Contains(line, `guard="activity"`) {
			continue
		}
		f := strings.Fields(line)
		v, err := strconv.ParseFloat(f[len(f)-1], 64)
		if err != nil {
			t.Fatalf("%s line %q: %v", decisions, line, err)
		}
		if strings.Contains(line, `outcome="allow"`) {
			allow += v
		} else {
			other += v
		}
	}
	return allow, other
}

// The nava-tq rule (TICK-453, Technical Lead 2026-11-07): every worker that
// polls a task queue registers the same workflow types. nava-worker registers
// audit.seal there; nava-worker-interptest registers audit.seal and
// nava.interpreter, so the interpreter cases stop nava-worker while it runs
// (startInterpWorker). This test needs nava-worker up, so it starts no worker
// of its own on nava-tq: the service runs a workflow it registers, audit.seal,
// started by the stack's relay. A new case that brings its own worker to
// nava-tq stops nava-worker first, as startInterpWorker does.
//
// It replaces the closeRef body (T31b), which polled nava-tq as
// nava.interpreter beside the service. workflow_ref in the service image is
// still covered: TestInterpreterStartedByTheRelayCompletes ends in
// CloseWorkflowRef on nava-worker-interptest, the same Dockerfile.

// psqlSuper runs sql as the superuser over the postgres container's Unix
// socket (deploy/compose/test/lib.sh psql_super), unaligned, tuples only.
func psqlSuper(t *testing.T, sql string) string {
	t.Helper()
	out, err := compose(t, "", "exec", "-T", "postgres", "sh", "-c",
		`PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -t -A -v ON_ERROR_STOP=1 -U pgadmin -d nava -c "$0"`, sql)
	if err != nil {
		t.Fatalf("psql %q: %v (%s)", sql, err, out)
	}
	return strings.TrimSpace(out)
}

// seedSealLineage writes, as onboarding's SeedAuditChain does (A6.2), an
// active tenant, its audit_chain_head and the first audit.seal start row at
// head version 1, and returns the tenant and that row's id. The relay starts
// the row as t:<tenant>:audit.seal:<id>.
func seedSealLineage(t *testing.T) (tenant, first uuid.UUID) {
	t.Helper()
	tenant, first = uuid.New(), uuid.Must(uuid.NewV7())
	psqlSuper(t, fmt.Sprintf(`BEGIN;
INSERT INTO tenants (id, state) VALUES ('%[1]s', 'active');
INSERT INTO audit_chain_head (tenant_id, id, next_seal_outbox_id, next_seal_at) VALUES ('%[1]s', '%[3]s', '%[2]s', now());
INSERT INTO outbox (id, tenant_id, kind, kind_version, dedupe_key, target_key, target_seq, causation_id, causation_depth, payload)
VALUES ('%[2]s', '%[1]s', 'audit.seal', 1, 'audit.seal:1', 'audit.seal:%[2]s', 1, '%[2]s', 0, '{}');
COMMIT;`, tenant, first, uuid.Must(uuid.NewV7())))
	return tenant, first
}

// writeAuditEntry inserts one unsealed audit entry for tenant.
func writeAuditEntry(t *testing.T, tenant uuid.UUID) {
	t.Helper()
	psqlSuper(t, fmt.Sprintf(`INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
VALUES ('%s', '%s', 'audit', 'export', 'audit_export', gen_random_uuid(), 'system', 'system')`, tenant, uuid.Must(uuid.NewV7())))
}

// awaitSealed polls, bounded, until tenant has no unsealed entry and its head
// has sealed want entries and moved past the first run, and returns the head
// as "seq|next_seal_outbox_id".
func awaitSealed(t *testing.T, tenant, first uuid.UUID, want int) string {
	t.Helper()
	q := fmt.Sprintf(`SELECT (SELECT count(*) FROM audit_log WHERE tenant_id = '%[1]s' AND sealed_at IS NULL)
	|| '|' || h.seq || '|' || coalesce(h.next_seal_outbox_id::text, '') FROM audit_chain_head h WHERE h.tenant_id = '%[1]s'`, tenant)
	var got string
	for deadline := time.Now().Add(3 * time.Minute); time.Now().Before(deadline); time.Sleep(2 * time.Second) {
		got = psqlSuper(t, q)
		f := strings.Split(got, "|")
		if len(f) == 3 && f[0] == "0" && f[1] == strconv.Itoa(want) && f[2] != "" && f[2] != first.String() {
			return f[1] + "|" + f[2]
		}
	}
	t.Fatalf("tenant %s not sealed within 3m: unsealed|head seq|next run = %q, want 0|%d|<not %s>", tenant, got, want, first)
	return ""
}

// TestWorkerServiceSealsThroughTheGuard (TICK-453, CI-A16 on the stack): a
// relay-started audit.seal run of a seeded tenant runs in the nava-worker
// service. A1 and A2 pass the activity guard (counted on /metrics) and reach
// PostgreSQL as nava_worker under RLS: the entry is sealed at seq 1, the head
// points at the next run, the first start row is dispatched, and
// /metrics exports the sealer's audit_unsealed_oldest_seconds. The guard
// denies nothing meanwhile: its non-allow activity decisions do not move
// (TICK-589), so a denial cannot hide behind a later successful seal.
func TestWorkerServiceSealsThroughTheGuard(t *testing.T) {
	awaitReady(t)
	before, deniedBefore := activityDecisions(t)
	tenant, first := seedSealLineage(t)
	writeAuditEntry(t, tenant)

	awaitSealed(t, tenant, first, 1)
	if got := psqlSuper(t, fmt.Sprintf(`SELECT dispatched_at IS NOT NULL FROM outbox WHERE tenant_id = '%s' AND id = '%s'`, tenant, first)); got != "t" {
		t.Errorf("the first start row is not dispatched (%q)", got)
	}
	if got := psqlSuper(t, fmt.Sprintf(`SELECT count(*) FROM audit_log WHERE tenant_id = '%s' AND seq = 1 AND entry_hash IS NOT NULL`, tenant)); got != "1" {
		t.Errorf("%s entries at seq 1 with an entry_hash, want 1", got)
	}
	after, deniedAfter := activityDecisions(t)
	if after < before+2 {
		t.Errorf("%s{guard=activity,outcome=allow} went from %v to %v; want A1 and A2 counted", decisions, before, after)
	}
	if deniedAfter != deniedBefore {
		t.Errorf("%s{guard=activity,outcome!=allow} went from %v to %v; want no guard denial during the seal", decisions, deniedBefore, deniedAfter)
	}
	if m := admin(t, "/metrics"); !strings.Contains(m, "audit_unsealed_oldest_seconds") {
		t.Error("/metrics has no audit_unsealed_oldest_seconds")
	}
}

func certDir() string {
	if d := os.Getenv("NAVA_TEST_CERT_DIR"); d != "" {
		return d
	}
	return filepath.Join(root, "deploy", "compose", "certs")
}

func address() string {
	if a := os.Getenv("NAVA_TEST_TEMPORAL_ADDR"); a != "" {
		return a
	}
	if p := os.Getenv("TEMPORAL_TEST_PORT"); p != "" {
		return "127.0.0.1:" + p
	}
	return "127.0.0.1:17233"
}

// dial connects as the identity cn through the gateway compose.test.yaml
// publishes on loopback; the frontend certificate names `localhost`. It is
// the temporalstack helper (internal/temporalstack/stack_test.go).
func dial(t *testing.T, cn string, opts ...func(*client.Options)) client.Client {
	t.Helper()
	dir := certDir()
	pair, err := tls.LoadX509KeyPair(filepath.Join(dir, cn+".pem"), filepath.Join(dir, cn+".key"))
	if err != nil {
		t.Fatalf("%s: %v (run make dev-ca)", cn, err)
	}
	caPEM, err := os.ReadFile(filepath.Join(dir, "ca.pem"))
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	o := client.Options{
		HostPort:  address(),
		Namespace: namespace,
		ConnectionOptions: client.ConnectionOptions{TLS: &tls.Config{
			Certificates: []tls.Certificate{pair}, RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS12,
		}},
	}
	for _, opt := range opts {
		opt(&o)
	}
	c, err := client.Dial(o)
	if err != nil {
		t.Fatalf("%s: dial: %v", cn, err)
	}
	t.Cleanup(c.Close)
	return c
}
