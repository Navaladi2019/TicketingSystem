//go:build temporalstack

package temporalstack

// Helpers for the CI-6 suite (ci6_isolation_test.go): raw gRPC connections with
// any certificate, the server's deny log and metric, and throwaway containers on
// the compose networks. Nothing here stubs the Authorizer: every call goes to the
// compose Temporal through the gateway.

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/mem"
	"google.golang.org/grpc/status"
)

// unlistedCN is the installation-CA certificate whose CN is no identity
// (`make dev-ca` writes it; nothing in the stack mounts it).
const unlistedCN = "nava-unknown"

func temporalImage() string {
	if i := os.Getenv("NAVA_TEST_TEMPORAL_IMAGE"); i != "" {
		return i
	}
	return "nava/temporal-server:dev"
}

// pairOf loads the certificate and key of cn from the dev CA directory.
func pairOf(t *testing.T, cn string) tls.Certificate {
	t.Helper()
	dir := certDir()
	pair, err := tls.LoadX509KeyPair(filepath.Join(dir, cn+".pem"), filepath.Join(dir, cn+".key"))
	if err != nil {
		t.Fatalf("%s: %v (run make dev-ca FORCE=1: the CI-6 suite needs %s.pem)", cn, err, cn)
	}
	return pair
}

func rootPool(t *testing.T) *x509.CertPool {
	t.Helper()
	caPEM, err := os.ReadFile(filepath.Join(certDir(), "ca.pem"))
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("ca.pem holds no certificate")
	}
	return pool
}

// clientTLS trusts the installation CA for the frontend and presents certs.
func clientTLS(t *testing.T, certs ...tls.Certificate) *tls.Config {
	return &tls.Config{Certificates: certs, RootCAs: rootPool(t), ServerName: "localhost", MinVersion: tls.VersionTLS12}
}

// rawConn is a plain gRPC connection to the gateway with cfg, for calls the SDK
// client does not expose (admin service, empty bodies, crafted task tokens).
func rawConn(t *testing.T, cfg *tls.Config) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(address(), grpc.WithTransportCredentials(credentials.NewTLS(cfg)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func rawConnAs(t *testing.T, cn string) *grpc.ClientConn {
	return rawConn(t, clientTLS(t, pairOf(t, cn)))
}

// rawCodec sends a pre-encoded request body. Its name is "proto", so the server
// decodes the bytes into the method's own request type.
type rawCodec struct{}

func (rawCodec) Marshal(v any) (mem.BufferSlice, error) {
	b, ok := v.(*[]byte)
	if !ok {
		return nil, fmt.Errorf("rawCodec: %T", v)
	}
	return mem.BufferSlice{mem.SliceBuffer(*b)}, nil
}

func (rawCodec) Unmarshal(data mem.BufferSlice, v any) error {
	b, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("rawCodec: %T", v)
	}
	*b = data.Materialize()
	return nil
}

func (rawCodec) Name() string { return "proto" }

// invokeRaw calls the full method name with body as the encoded request.
func invokeRaw(ctx context.Context, conn *grpc.ClientConn, method string, body []byte) error {
	var out []byte
	return conn.Invoke(ctx, method, &body, &out, grpc.ForceCodecV2(rawCodec{}))
}

// wantDenied fails unless err is the Authorizer's PermissionDenied.
func wantDenied(t *testing.T, what string, err error) {
	t.Helper()
	if status.Code(err) != codes.PermissionDenied {
		t.Errorf("%s: got %v (code %v), want PermissionDenied", what, err, status.Code(err))
	}
}

// rogueCertificate is a nava-relay client certificate from a fresh CA that is not
// the installation CA.
func rogueCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	newKey := func() *ecdsa.PrivateKey {
		k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		return k
	}
	caKey, leafKey := newKey(), newKey()
	caTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "ci6-rogue-ca"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, MaxPathLenZero: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "nava-relay"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: leafKey}
}

// handshakeAlert completes a TLS handshake with the gateway presenting certs and
// returns the error of the first read. TLS 1.3 verifies the client certificate
// after the client's side of the handshake is done, so a rejection arrives as an
// alert on the first read; a nil error means no alert within the deadline.
func handshakeAlert(t *testing.T, certs ...tls.Certificate) error {
	t.Helper()
	cfg := clientTLS(t, certs...)
	cfg.NextProtos = []string{"h2"} // gRPC-Go closes a connection that negotiated no ALPN
	d := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 10 * time.Second}, Config: cfg}
	conn, err := d.DialContext(context.Background(), "tcp", address())
	if err != nil {
		return err
	}
	defer conn.Close()
	// The HTTP/2 preface, so an accepted connection gets a SETTINGS frame back.
	if _, err := conn.Write([]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n\x00\x00\x00\x04\x00\x00\x00\x00\x00")); err != nil {
		return err
	}
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 9)
	_, err = conn.Read(buf)
	return err
}

// tenantPayload is the nava-tenant header value: a default-converter string.
func tenantPayload(t *testing.T, tenant string) *commonpb.Payload {
	t.Helper()
	p, err := converter.GetDefaultDataConverter().ToPayload(tenant)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func tenantHeader(t *testing.T, tenant string) *commonpb.Header {
	return &commonpb.Header{Fields: map[string]*commonpb.Payload{"nava-tenant": tenantPayload(t, tenant)}}
}

func uniqueID(tenant, what string) string {
	return fmt.Sprintf("t:%s:ci6-%s-%d", tenant, what, time.Now().UnixNano())
}

// startOnPlatformQueue starts an Echo workflow on nava-tq as c and terminates it
// (with the temporal-system certificate) when the test ends: a denied workflow
// task is retried until someone acts.
func startOnPlatformQueue(ctx context.Context, t *testing.T, c client.Client, id string, delay time.Duration, input string) client.WorkflowRun {
	t.Helper()
	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: id, TaskQueue: taskQueue, StartDelay: delay}, echoType, input)
	if err != nil {
		t.Fatalf("start %s: %v", id, err)
	}
	system := dial(t, "temporal-system")
	t.Cleanup(func() { _ = system.TerminateWorkflow(context.Background(), id, run.GetRunID(), "CI-6 cleanup") })
	return run
}

// pollTaskFor polls nava-tq as nava-worker until it receives a workflow task of
// workflowID. Tasks of other workflows (left over from an earlier run) are left
// to time out.
func pollTaskFor(ctx context.Context, t *testing.T, worker workflowservice.WorkflowServiceClient, workflowID string) *workflowservice.PollWorkflowTaskQueueResponse {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		pctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		resp, err := worker.PollWorkflowTaskQueue(pctx, &workflowservice.PollWorkflowTaskQueueRequest{
			Namespace: namespace, Identity: "ci6-attacker",
			TaskQueue: &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
		})
		cancel()
		if err != nil && status.Code(err) != codes.DeadlineExceeded {
			t.Fatalf("nava-worker PollWorkflowTaskQueue: %v", err)
		}
		if resp.GetWorkflowExecution().GetWorkflowId() == workflowID {
			return resp
		}
	}
	t.Fatalf("no workflow task for %s on %s within 90s", workflowID, taskQueue)
	return nil
}

// historyHas reports whether the workflow's history holds an event of type et.
func historyHas(ctx context.Context, t *testing.T, c client.Client, id string, et enumspb.EventType) bool {
	t.Helper()
	it := c.GetWorkflowHistory(ctx, id, "", false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for it.HasNext() {
		ev, err := it.Next()
		if err != nil {
			t.Fatalf("history of %s: %v", id, err)
		}
		if ev.GetEventType() == et {
			return true
		}
	}
	return false
}

// --- the server's deny log and metric -------------------------------------

// containerOf returns the id of the compose service's running container.
func containerOf(t *testing.T, service string) string {
	t.Helper()
	out := docker(t, "ps", "-q", "--filter", "label=com.docker.compose.project=nava",
		"--filter", "label=com.docker.compose.service="+service)
	id := strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
	if id == "" {
		t.Fatalf("no running %s container in compose project nava", service)
	}
	return id
}

// networkOf returns the id of the compose network (internal, temporal-backplane, ...).
func networkOf(t *testing.T, name string) string {
	t.Helper()
	out := docker(t, "network", "ls", "-q", "--filter", "label=com.docker.compose.project=nava",
		"--filter", "label=com.docker.compose.network="+name)
	id := strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
	if id == "" {
		t.Fatalf("no compose network %s in project nava", name)
	}
	return id
}

func docker(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.CommandContext(context.Background(), "docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// dockerRun runs a throwaway container and returns its output and exit error.
func dockerRun(args ...string) (string, error) {
	out, err := exec.CommandContext(context.Background(), "docker", append([]string{"run", "--rm"}, args...)...).CombinedOutput()
	return string(out), err
}

// logMark is a point in time to read the server log from. Docker Desktop's VM
// clock may lag the host's, so it starts a minute early; callers match on a
// unique workflow id or certificate subject.
func logMark() string { return strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10) }

// awaitDenyLog waits up to 20s for a JSON line of the temporal-server log, since
// mark, whose fields include every key/value of want.
func awaitDenyLog(t *testing.T, mark string, want map[string]string) map[string]any {
	t.Helper()
	srv := containerOf(t, "temporal-server")
	deadline := time.Now().Add(20 * time.Second)
	for {
		out, _ := exec.CommandContext(context.Background(), "docker", "logs", "--since", mark, srv).CombinedOutput()
		sc := bufio.NewScanner(bytes.NewReader(out))
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			var line map[string]any
			if json.Unmarshal(sc.Bytes(), &line) != nil {
				continue
			}
			if logMatches(line, want) {
				return line
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no temporal-server log line with %v", want)
		}
		time.Sleep(time.Second)
	}
}

// logMatches: a value starting with "~" is a substring match.
func logMatches(line map[string]any, want map[string]string) bool {
	for k, v := range want {
		got, ok := line[k].(string)
		if !ok {
			return false
		}
		if sub, isSub := strings.CutPrefix(v, "~"); isSub {
			if !strings.Contains(got, sub) {
				return false
			}
		} else if got != v {
			return false
		}
	}
	return true
}

// deniedCount sums nava_authz_denied over the series carrying every label of
// want, scraped from the server's metrics port on temporal-backplane (no platform
// network reaches it), with the nava-core image's busybox wget.
func deniedCount(t *testing.T, want map[string]string) float64 {
	t.Helper()
	out, err := dockerRun("--network", networkOf(t, "temporal-backplane"), "--entrypoint", "wget",
		"nava/nava-core:dev", "-q", "-O", "-", "http://temporal-server:9090/metrics")
	if err != nil {
		t.Fatalf("scrape temporal-server metrics: %v\n%s", err, out)
	}
	var sum float64
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "nava_authz_denied{") {
			continue
		}
		labels, value, ok := strings.Cut(line, "} ")
		if !ok {
			continue
		}
		match := true
		for k, v := range want {
			if !strings.Contains(labels, k+`="`+promValue(v)+`"`) {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err == nil {
			sum += f
		}
	}
	return sum
}

// promValue is a tag value as the server's Prometheus reporter writes it: every
// character outside [A-Za-z0-9_] becomes "_" (nava-worker is nava_worker).
func promValue(v string) string {
	return strings.Map(func(r rune) rune {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, v)
}

// awaitDeniedCount waits up to 20s for the counter to exceed before.
func awaitDeniedCount(t *testing.T, want map[string]string, before float64) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for {
		if got := deniedCount(t, want); got > before {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("nava_authz_denied%v did not rise above %v", want, before)
		}
		time.Sleep(time.Second)
	}
}

// expectCommandDeny polls the task of a fresh tenant-A workflow, responds with
// commands, and requires the Authorizer's deny, an alert=true error log line
// with the certificate, workflow id and command, and a nava_authz_denied rise.
// It returns the attacking workflow's id.
func expectCommandDeny(ctx context.Context, t *testing.T, reason, command string, commands func(attackID string) []*commandpb.Command) string {
	t.Helper()
	relay, workerConn := dial(t, "nava-relay"), rawConnAs(t, "nava-worker")
	worker := workflowservice.NewWorkflowServiceClient(workerConn)
	attackID := uniqueID("A", "attack")
	startOnPlatformQueue(ctx, t, relay, attackID, 0, "ci6")
	task := pollTaskFor(ctx, t, worker, attackID)

	labels := map[string]string{"identity": "nava-worker", "reason": reason, "command": command}
	before := deniedCount(t, labels)
	mark := logMark()
	_, err := worker.RespondWorkflowTaskCompleted(ctx, &workflowservice.RespondWorkflowTaskCompletedRequest{
		Namespace: namespace, TaskToken: task.GetTaskToken(), Identity: "ci6-attacker",
		Commands: commands(attackID),
	})
	wantDenied(t, "nava-worker RespondWorkflowTaskCompleted ("+reason+")", err)
	awaitDenyLog(t, mark, map[string]string{
		"level": "error", "alert": "true", "msg": "Temporal workflow command denied",
		"identity": "nava-worker", "api": "/temporal.api.workflowservice.v1.WorkflowService/RespondWorkflowTaskCompleted",
		"reason": reason, "command": command, "workflow_id": attackID, "cert_subject": "~CN=nava-worker",
	})
	awaitDeniedCount(t, labels, before)
	return attackID
}
