package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/Navaladi2019/TicketingSystem/temporal-server/authz"
)

// TenantSearchAttribute is the Keyword search attribute that carries the tenant
// id on every workflow. It is for visibility only, never for access control.
const TenantSearchAttribute = "NavaTenantId"

// namespaceRetention is how long closed workflows stay in the nava namespace.
const namespaceRetention = 7 * 24 * time.Hour

// tlsFiles locates a client identity for dialing the frontend.
type tlsFiles struct {
	CA, Cert, Key, ServerName string
}

func (f tlsFiles) config() (*tls.Config, error) {
	caPEM, err := os.ReadFile(f.CA)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("no CA certificate in %s", f.CA)
	}
	tc := &tls.Config{RootCAs: pool, ServerName: f.ServerName, MinVersion: tls.VersionTLS12}
	if f.Cert != "" {
		pair, err := tls.LoadX509KeyPair(f.Cert, f.Key)
		if err != nil {
			return nil, err
		}
		tc.Certificates = []tls.Certificate{pair}
	}
	return tc, nil
}

func (f tlsFiles) dial(addr string) (*grpc.ClientConn, error) {
	tc, err := f.config()
	if err != nil {
		return nil, err
	}
	return grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tc)))
}

// bootstrap registers the nava namespace and the NavaTenantId search attribute
// as temporal-system, the only identity allowed RegisterNamespace. It waits for
// the frontend and is idempotent. The certificate comes from the server's own
// systemWorker config, so it never has to leave this container.
func bootstrap(ctx context.Context, cfg *config.Config, logger log.Logger) error {
	sw := cfg.Global.TLS.SystemWorker
	if len(sw.Client.RootCAFiles) == 0 || sw.CertFile == "" {
		return errors.New("bootstrap needs global.tls.systemWorker certFile and client.rootCaFiles")
	}
	f := tlsFiles{CA: sw.Client.RootCAFiles[0], Cert: sw.CertFile, Key: sw.KeyFile, ServerName: sw.Client.ServerName}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.Services["frontend"].RPC.GRPCPort))
	conn, err := f.dial(addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := ensureNamespace(ctx, conn); err != nil {
		return err
	}
	logger.Info("bootstrap complete", tag.NewStringTag("namespace", authz.Namespace),
		tag.NewStringTag("searchAttribute", TenantSearchAttribute))
	return nil
}

// ensureNamespace retries until the namespace and the search attribute exist.
func ensureNamespace(ctx context.Context, conn grpc.ClientConnInterface) error {
	wf := workflowservice.NewWorkflowServiceClient(conn)
	op := operatorservice.NewOperatorServiceClient(conn)
	step := func(ctx context.Context) error {
		if _, err := wf.GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{}); err != nil {
			return err
		}
		_, err := wf.RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{
			Namespace:                        authz.Namespace,
			WorkflowExecutionRetentionPeriod: durationpb.New(namespaceRetention),
		})
		if err != nil && status.Code(err) != codes.AlreadyExists {
			return err
		}
		_, err = op.AddSearchAttributes(ctx, &operatorservice.AddSearchAttributesRequest{
			Namespace:        authz.Namespace,
			SearchAttributes: map[string]enumspb.IndexedValueType{TenantSearchAttribute: enumspb.INDEXED_VALUE_TYPE_KEYWORD},
		})
		if err != nil && status.Code(err) != codes.AlreadyExists {
			return err
		}
		return nil
	}
	var last error
	for {
		sctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		last = step(sctx)
		cancel()
		if last == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("bootstrap: %w (last error: %v)", ctx.Err(), last)
		case <-time.After(2 * time.Second):
		}
	}
}
