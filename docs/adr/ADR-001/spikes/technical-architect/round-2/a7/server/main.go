// Spike A7: Temporal 1.28.4 custom Authorizer + ClaimMapper over mTLS.
// Does a per-certificate-identity API allow-list (RFC-001 rev1 §4.3) hold,
// and what does it NOT cover? Throwaway spike code; in-memory sqlite server.
//
// env: INSPECT=1 makes the authorizer inspect RespondWorkflowTaskCompleted
// commands for cross-tenant targets; EXTRA="Api1,Api2" adds APIs to every
// client identity (to find the minimal working list).
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/signal"
	"syscall"
	"sort"
	"strings"
	"sync"
	"time"

	commandpb "go.temporal.io/api/command/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/cluster"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	_ "go.temporal.io/server/common/persistence/sql/sqlplugin/sqlite"
	"go.temporal.io/server/common/primitives"
	"go.temporal.io/server/common/tasktoken"
	sqliteschema "go.temporal.io/server/schema/sqlite"
	"go.temporal.io/server/temporal"
)

const ns = "nava"

// ---------------- certificates ----------------
type kp struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pem  []byte
	kpem []byte
}

func mk(cn string, parent *kp, isCA, server bool) *kp {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	sn, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	t := &x509.Certificate{SerialNumber: sn, Subject: pkix.Name{CommonName: cn, Organization: []string{"nava"}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment}
	if isCA {
		t.IsCA, t.BasicConstraintsValid = true, true
		t.KeyUsage |= x509.KeyUsageCertSign
	}
	if server {
		t.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
		t.DNSNames = []string{"localhost"}
		t.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	} else if !isCA {
		t.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	signer, signKey := t, key
	if parent != nil {
		signer, signKey = parent.cert, parent.key
	}
	der, err := x509.CreateCertificate(rand.Reader, t, signer, &key.PublicKey, signKey)
	must(err)
	c, _ := x509.ParseCertificate(der)
	kb, _ := x509.MarshalECPrivateKey(key)
	return &kp{c, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb})}
}

// ---------------- claim mapper + authorizer ----------------
type certClaimMapper struct{}

func (certClaimMapper) GetClaims(ai *authorization.AuthInfo) (*authorization.Claims, error) {
	if ai == nil || ai.TLSSubject == nil {
		return &authorization.Claims{}, nil // no cert -> no identity -> denied
	}
	return &authorization.Claims{Subject: ai.TLSSubject.CommonName}, nil
}

var rfcAllow = map[string][]string{
	"nava-relay": {"StartWorkflowExecution", "SignalWorkflowExecution", "SignalWithStartWorkflowExecution"},
	"nava-core": {"RequestCancelWorkflowExecution", "UpdateWorkflowExecution", "QueryWorkflow",
		"DescribeWorkflowExecution", "GetWorkflowExecutionHistory"},
	"nava-worker": {"PollWorkflowTaskQueue", "PollActivityTaskQueue", "RespondWorkflowTask*", "RespondActivityTask*",
		"RecordActivityTaskHeartbeat", "RespondQueryTaskCompleted"},
}

type authz struct {
	mu      sync.Mutex
	seen    map[string]string // "cn api" -> decision
	inspect bool
	extra   []string
	ser     *tasktoken.Serializer
}

func method(api string) string { return api[strings.LastIndex(api, "/")+1:] }

func (a *authz) record(cn, api, dec string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	k := fmt.Sprintf("%-16s %-40s", cn, api)
	if _, ok := a.seen[k]; !ok {
		a.seen[k] = dec
	}
}

func allowed(list []string, m string) bool {
	for _, p := range list {
		if p == m || (strings.HasSuffix(p, "*") && strings.HasPrefix(m, strings.TrimSuffix(p, "*"))) {
			return true
		}
	}
	return false
}

func prefixTenant(id string) string {
	p := strings.SplitN(id, ":", 3)
	if len(p) < 3 || p[0] != "t" {
		return "<none>"
	}
	return p[1]
}

func (a *authz) Authorize(_ context.Context, c *authorization.Claims, t *authorization.CallTarget) (authorization.Result, error) {
	cn := "<none>"
	if c != nil && c.Subject != "" {
		cn = c.Subject
	}
	m := method(t.APIName)
	if strings.HasPrefix(t.APIName, "/grpc.health") {
		return authorization.Result{Decision: authorization.DecisionAllow}, nil
	}
	if cn == "temporal-system" { // server's own system worker
		return authorization.Result{Decision: authorization.DecisionAllow}, nil
	}
	ok := allowed(rfcAllow[cn], m) || (rfcAllow[cn] != nil && allowed(a.extra, m))
	reason := ""
	if ok && a.inspect && m == "RespondWorkflowTaskCompleted" {
		if req, isReq := t.Request.(*workflowservice.RespondWorkflowTaskCompletedRequest); isReq {
			tok, err := a.ser.Deserialize(req.TaskToken)
			if err != nil {
				ok, reason = false, "bad token"
			} else {
				own := prefixTenant(tok.WorkflowId)
				for _, cmd := range req.Commands {
					if tgt := commandTarget(cmd); tgt != "" && prefixTenant(tgt) != own {
						ok, reason = false, fmt.Sprintf("cross-tenant %s %s->%s", cmd.CommandType, tok.WorkflowId, tgt)
					}
				}
			}
		}
	}
	dec := "allow"
	if !ok {
		dec = "DENY " + reason
		fmt.Printf("AUTHZ DENY cn=%s api=%s ns=%s %s\n", cn, m, t.Namespace, reason)
	}
	a.record(cn, m, dec)
	if ok {
		return authorization.Result{Decision: authorization.DecisionAllow}, nil
	}
	return authorization.Result{Decision: authorization.DecisionDeny, Reason: reason}, nil
}

func commandTarget(cmd *commandpb.Command) string {
	switch cmd.CommandType {
	case enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION:
		return cmd.GetStartChildWorkflowExecutionCommandAttributes().GetWorkflowId()
	case enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION:
		return cmd.GetSignalExternalWorkflowExecutionCommandAttributes().GetExecution().GetWorkflowId()
	case enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION:
		return cmd.GetRequestCancelExternalWorkflowExecutionCommandAttributes().GetWorkflowId()
	}
	return ""
}

// ---------------- main ----------------
func must(err error) {
	if err != nil {
		panic(err)
	}
}

func write(dir, name string, b []byte) string {
	p := dir + "/" + name
	must(os.WriteFile(p, b, 0o600))
	return p
}

func main() {
	dir := "/certs"
	ca := mk("nava-internal-ca", nil, true, false)
	rogueCA := mk("rogue-ca", nil, true, false)
	srv := mk("localhost", ca, false, true)
	sys := mk("temporal-system", ca, false, true)
	caFile := write(dir, "ca.pem", ca.pem)
	for _, cn := range []string{"nava-relay", "nava-core", "nava-worker", "nava-unknown"} {
		k := mk(cn, ca, false, false)
		write(dir, cn+".pem", k.pem)
		write(dir, cn+".key", k.kpem)
	}
	r := mk("nava-relay", rogueCA, false, false)
	write(dir, "rogue.pem", r.pem)
	write(dir, "rogue.key", r.kpem)
	sqlCfg := &config.SQL{PluginName: "sqlite", DatabaseName: "a7", ConnectAddr: "localhost", ConnectProtocol: "tcp",
		ConnectAttributes: map[string]string{"mode": "memory", "cache": "shared"}, MaxConns: 1, MaxIdleConns: 1}
	nsCfg, err := sqliteschema.NewNamespaceConfig("active", ns, false, nil)
	must(err)
	must(sqliteschema.CreateNamespaces(sqlCfg, nsCfg))

	srvTLS := config.ServerTLS{CertFile: write(dir, "srv.pem", srv.pem), KeyFile: write(dir, "srv.key", srv.kpem),
		ClientCAFiles: []string{caFile}, RequireClientAuth: true}
	cliTLS := config.ClientTLS{ServerName: "localhost", RootCAFiles: []string{caFile}}
	svc := func(port int) config.Service {
		return config.Service{RPC: config.RPC{GRPCPort: port, MembershipPort: port + 100, BindOnIP: "127.0.0.1"}}
	}
	cfg := &config.Config{
		Global: config.Global{
			Membership: config.Membership{MaxJoinDuration: 30 * time.Second, BroadcastAddress: "127.0.0.1"},
			TLS: config.RootTLS{
				Frontend:     config.GroupTLS{Server: srvTLS, Client: cliTLS},
				SystemWorker: config.WorkerTLS{CertFile: write(dir, "sys.pem", sys.pem), KeyFile: write(dir, "sys.key", sys.kpem), Client: cliTLS},
			},
		},
		Persistence: config.Persistence{DefaultStore: "db", VisibilityStore: "db", NumHistoryShards: 1,
			DataStores: map[string]config.DataStore{"db": {SQL: sqlCfg}}},
		ClusterMetadata: &cluster.Config{EnableGlobalNamespace: false, FailoverVersionIncrement: 10,
			MasterClusterName: "active", CurrentClusterName: "active",
			ClusterInformation: map[string]cluster.ClusterInformation{"active": {Enabled: true, InitialFailoverVersion: 1, RPCAddress: "localhost:7233"}}},
		DCRedirectionPolicy: config.DCRedirectionPolicy{Policy: "noop"},
		Services: map[string]config.Service{"frontend": svc(7233), "history": svc(7234), "matching": svc(7235), "worker": svc(7239)},
		PublicClient: config.PublicClient{HostPort: "localhost:7233"},
	}

	az := &authz{seen: map[string]string{}, inspect: os.Getenv("INSPECT") == "1", ser: tasktoken.NewSerializer()}
	if e := os.Getenv("EXTRA"); e != "" {
		az.extra = strings.Split(e, ",")
	}
	s, err := temporal.NewServer(
		temporal.ForServices([]string{string(primitives.FrontendService), string(primitives.HistoryService),
			string(primitives.MatchingService), string(primitives.WorkerService)}),
		temporal.WithConfig(cfg),
		temporal.WithLogger(log.NewCLILogger()),
		temporal.WithDynamicConfigClient(dynamicconfig.StaticClient{}),
		temporal.WithAuthorizer(az),
		temporal.WithClaimMapper(func(*config.Config) authorization.ClaimMapper { return certClaimMapper{} }),
	)
	must(err)
	must(s.Start())
	defer s.Stop()
	time.Sleep(3 * time.Second)
	write(dir, "ready", []byte("1"))
	fmt.Printf("\n===== server up (INSPECT=%v EXTRA=%v)\n", az.inspect, az.extra)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)
	<-sig
	fmt.Println("\n== decisions seen (identity, API -> first decision)")
	az.mu.Lock()
	keys := make([]string, 0, len(az.seen))
	for k := range az.seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("SEEN %s %s\n", k, az.seen[k])
	}
	az.mu.Unlock()
}
