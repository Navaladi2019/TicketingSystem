package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"time"

	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/Navaladi2019/TicketingSystem/temporal-server/authz"
)

// runHealth dials the frontend over mTLS and checks that it serves. With
// -bootstrapped it also requires the nava namespace and NavaTenantId, so a
// healthy server container is a bootstrapped one. It prints one line and returns
// an error on any failure, including a failed handshake (which is how the stack
// tests prove a rogue-CA certificate is refused).
func runHealth(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	var f tlsFiles
	addr := fs.String("addr", "127.0.0.1:7233", "frontend host:port")
	fs.StringVar(&f.CA, "ca", "/etc/temporal/certs/ca.pem", "CA that signed the frontend certificate")
	fs.StringVar(&f.Cert, "cert", "/etc/temporal/certs/temporal-system.pem", "client certificate")
	fs.StringVar(&f.Key, "key", "/etc/temporal/certs/temporal-system.key", "client key")
	fs.StringVar(&f.ServerName, "server-name", "localhost", "expected name on the frontend certificate")
	bootstrapped := fs.Bool("bootstrapped", false, "also require the nava namespace and NavaTenantId")
	timeout := fs.Duration("timeout", 5*time.Second, "overall timeout")
	tcpOnly := fs.Bool("tcp-only", false, "only check that the port accepts connections (for the gateway, which holds no certificate)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tcpOnly {
		c, err := net.DialTimeout("tcp", *addr, *timeout)
		if err != nil {
			return err
		}
		_ = c.Close()
		_, err = fmt.Fprintln(out, "OPEN")
		return err
	}
	conn, err := f.dial(*addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resp, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("health check: %s", resp.Status)
	}
	if *bootstrapped {
		if _, err := workflowservice.NewWorkflowServiceClient(conn).DescribeNamespace(ctx,
			&workflowservice.DescribeNamespaceRequest{Namespace: authz.Namespace}); err != nil {
			return fmt.Errorf("namespace %s: %w", authz.Namespace, err)
		}
		sa, err := operatorservice.NewOperatorServiceClient(conn).ListSearchAttributes(ctx,
			&operatorservice.ListSearchAttributesRequest{Namespace: authz.Namespace})
		if err != nil {
			return fmt.Errorf("list search attributes: %w", err)
		}
		if _, ok := sa.CustomAttributes[TenantSearchAttribute]; !ok {
			return errors.New("search attribute " + TenantSearchAttribute + " is not registered")
		}
	}
	_, err = fmt.Fprintln(out, "SERVING")
	return err
}

// runProxy forwards TCP from -listen to -to, byte for byte (the TLS session is
// end to end, so the proxy holds no certificate). It is the only listener that
// reaches the server from the platform network.
func runProxy(args []string) error {
	fs := flag.NewFlagSet("proxy", flag.ContinueOnError)
	listen := fs.String("listen", ":7233", "address to listen on")
	to := fs.String("to", "temporal-server:7233", "address to forward to")
	if err := fs.Parse(args); err != nil {
		return err
	}
	l, err := net.Listen("tcp", *listen)
	if err != nil {
		return err
	}
	return serveProxy(l, *to)
}

// serveProxy accepts on l until it is closed.
func serveProxy(l net.Listener, to string) error {
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		go forward(c, to)
	}
}

func forward(c net.Conn, to string) {
	defer c.Close()
	u, err := net.DialTimeout("tcp", to, 5*time.Second)
	if err != nil {
		return
	}
	defer u.Close()
	done := make(chan struct{}, 2)
	pipe := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		if cw, ok := dst.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
		done <- struct{}{}
	}
	go pipe(u, c)
	go pipe(c, u)
	<-done
	<-done
}
