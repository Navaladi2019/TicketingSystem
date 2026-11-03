package onboard

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// column types used by the queries under test
const (
	oidBool  = 16
	oidText  = 25
	oidUUID  = 2950
	oidJSONB = 3802
	oidUUIDs = 2951
	oidTexts = 1009
)

// scripted is what the fake database answers to one statement.
type scripted struct {
	cols []uint32
	rows [][]*string // nil element is SQL NULL
	err  string      // non-empty: an ERROR response
	code string      // SQLSTATE of the error, default XX000
	cons string      // constraint name of the error
}

func sp(v string) *string { return &v }

// fakePGServer is a PostgreSQL wire-protocol server (simple query protocol) whose
// answers a test scripts per statement. It lets the tests drive pgstore's
// defensive branches (rows the schema forbids, such as a nil tenant or two
// bindings for one key) without a database that could be made to hold them.
type fakePGServer struct {
	mu      sync.Mutex
	handler func(sql string) scripted
	seen    []string
	addr    string
}

func (f *fakePGServer) statements() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.seen...)
}

func startFakePGServer(t *testing.T, handler func(sql string) scripted) *fakePGServer {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakePGServer{handler: handler, addr: ln.Addr().String()}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go f.serve(c)
		}
	}()
	return f
}

func (f *fakePGServer) serve(c net.Conn) {
	defer c.Close()
	b := pgproto3.NewBackend(c, c)
	for {
		msg, err := b.ReceiveStartupMessage()
		if err != nil {
			return
		}
		if _, ok := msg.(*pgproto3.SSLRequest); ok {
			if _, err := c.Write([]byte{'N'}); err != nil {
				return
			}
			continue
		}
		break
	}
	b.Send(&pgproto3.AuthenticationOk{})
	b.Send(&pgproto3.ParameterStatus{Name: "server_version", Value: "16.0"})
	b.Send(&pgproto3.ParameterStatus{Name: "client_encoding", Value: "UTF8"})
	b.Send(&pgproto3.ParameterStatus{Name: "standard_conforming_strings", Value: "on"})
	b.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	if b.Flush() != nil {
		return
	}
	status := byte('I')
	for {
		msg, err := b.Receive()
		if err != nil {
			return
		}
		switch m := msg.(type) {
		case *pgproto3.Terminate:
			return
		case *pgproto3.Query:
			sql := m.String
			f.mu.Lock()
			f.seen = append(f.seen, sql)
			f.mu.Unlock()
			lower := strings.ToLower(strings.TrimSpace(sql))
			switch {
			case strings.HasPrefix(lower, "begin"):
				status = 'T'
				b.Send(&pgproto3.CommandComplete{CommandTag: []byte("BEGIN")})
			case strings.HasPrefix(lower, "commit"):
				status = 'I'
				b.Send(&pgproto3.CommandComplete{CommandTag: []byte("COMMIT")})
			case strings.HasPrefix(lower, "rollback"):
				status = 'I'
				b.Send(&pgproto3.CommandComplete{CommandTag: []byte("ROLLBACK")})
			// The tenant scoping call that tenancy.InTenantTx sends.
			case strings.HasPrefix(lower, "select set_"):
				b.Send(&pgproto3.RowDescription{Fields: pgFields(oidText)})
				b.Send(&pgproto3.DataRow{Values: [][]byte{[]byte("")}})
				b.Send(&pgproto3.CommandComplete{CommandTag: []byte("SELECT 1")})
			default:
				r := f.handler(sql)
				if r.err != "" {
					b.Send(&pgproto3.ErrorResponse{Severity: "ERROR", Code: orDefault(r.code, "XX000"), ConstraintName: r.cons, Message: r.err})
					break
				}
				b.Send(&pgproto3.RowDescription{Fields: pgFields(r.cols...)})
				for _, row := range r.rows {
					vals := make([][]byte, len(row))
					for i, v := range row {
						if v != nil {
							vals[i] = []byte(*v)
						}
					}
					b.Send(&pgproto3.DataRow{Values: vals})
				}
				b.Send(&pgproto3.CommandComplete{CommandTag: []byte("SELECT")})
			}
			b.Send(&pgproto3.ReadyForQuery{TxStatus: status})
			if b.Flush() != nil {
				return
			}
		default:
			return
		}
	}
}

func pgFields(oids ...uint32) []pgproto3.FieldDescription {
	out := make([]pgproto3.FieldDescription, len(oids))
	for i, o := range oids {
		out[i] = pgproto3.FieldDescription{Name: []byte("c"), DataTypeOID: o, DataTypeSize: -1, TypeModifier: -1}
	}
	return out
}

func scriptedPool(t *testing.T, f *fakePGServer) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig("postgres://nava_app:x@" + f.addr + "/nava?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	cfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
