package pgstore_test

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

// result is what the fake database answers to one statement.
type result struct {
	cols []uint32
	rows [][]*string // nil element is SQL NULL
	err  string      // non-empty: an ERROR response
}

func s(v string) *string { return &v }

// fakePG is a PostgreSQL wire-protocol server (simple query protocol) whose
// answers a test scripts per statement. It lets the tests drive pgstore's
// defensive branches (rows the schema forbids, such as a nil tenant or two
// bindings for one key) without a database that could be made to hold them.
type fakePG struct {
	mu      sync.Mutex
	handler func(sql string) result
	seen    []string
	addr    string
}

func (f *fakePG) statements() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.seen...)
}

func startFakePG(t *testing.T, handler func(sql string) result) *fakePG {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakePG{handler: handler, addr: ln.Addr().String()}
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

func (f *fakePG) serve(c net.Conn) {
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
			// The tenant scoping call that tenancy.InTenantTx sends. The fake answers it
			// blindly: these tests prove error mapping, not tenant scoping, which the
			// real-Postgres RLS tests (dbtest, identitystack, tenancy/isolation) prove.
			case strings.HasPrefix(lower, "select set_"):
				b.Send(&pgproto3.RowDescription{Fields: fields(oidText)})
				b.Send(&pgproto3.DataRow{Values: [][]byte{[]byte("")}})
				b.Send(&pgproto3.CommandComplete{CommandTag: []byte("SELECT 1")})
			default:
				r := f.handler(sql)
				if r.err != "" {
					b.Send(&pgproto3.ErrorResponse{Severity: "ERROR", Code: "XX000", Message: r.err})
					break
				}
				b.Send(&pgproto3.RowDescription{Fields: fields(r.cols...)})
				for _, row := range r.rows {
					vals := make([][]byte, len(row))
					for i, v := range row {
						if v != nil {
							vals[i] = []byte(*v)
						}
					}
					b.Send(&pgproto3.DataRow{Values: vals})
				}
				b.Send(&pgproto3.CommandComplete{CommandTag: []byte("SELECT")}) // INSERT/UPDATE report 0 rows too
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

func fields(oids ...uint32) []pgproto3.FieldDescription {
	out := make([]pgproto3.FieldDescription, len(oids))
	for i, o := range oids {
		out[i] = pgproto3.FieldDescription{Name: []byte("c"), DataTypeOID: o, DataTypeSize: -1, TypeModifier: -1}
	}
	return out
}

func poolOn(t *testing.T, f *fakePG) *pgxpool.Pool {
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
