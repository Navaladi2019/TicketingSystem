package migrate

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgproto3"
)

// refusingServer answers every query of every connection with refusal, over
// the PostgreSQL wire protocol, and returns its address.
func refusingServer(t *testing.T, refusal *pgproto3.ErrorResponse) string {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				_ = refuse(conn, refusal)
			}()
		}
	}()
	return l.Addr().String()
}

func refuse(conn io.ReadWriter, refusal *pgproto3.ErrorResponse) error {
	b := pgproto3.NewBackend(conn, conn)
	if _, err := b.ReceiveStartupMessage(); err != nil {
		return err
	}
	b.Send(&pgproto3.AuthenticationOk{})
	b.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	for {
		if err := b.Flush(); err != nil {
			return err
		}
		m, err := b.Receive()
		if err != nil {
			return err
		}
		switch m.(type) {
		case *pgproto3.Parse:
			b.Send(refusal)
		case *pgproto3.Query:
			b.Send(refusal)
			b.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
		case *pgproto3.Sync:
			b.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
		}
	}
}

// wantFailedLine runs Up against addr, which must fail, and compares the one
// "migration failed" line it logs: the error's text and want, and nothing else.
func wantFailedLine(t *testing.T, addr string, want map[string]any) {
	t.Helper()
	cfg, err := pgx.ParseConfig("postgres://nava_migrator@" + addr + "/nava?sslmode=disable&connect_timeout=5")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	fsys := fstest.MapFS{"00001_a.sql": {Data: []byte("-- +goose Up\nSELECT 1;\n")}}
	_, err = Up(context.Background(), cfg, fsys, slog.New(slog.NewJSONHandler(&out, nil)), Options{})
	if err == nil {
		t.Fatal("Up succeeded")
	}
	want = maps.Clone(want)
	want["level"], want["msg"], want["error"] = "ERROR", "migration failed", err.Error()

	logged := out.String()
	n := 0
	for dec := json.NewDecoder(&out); dec.More(); {
		var line map[string]any
		if err := dec.Decode(&line); err != nil {
			t.Fatal(err)
		}
		if line["msg"] != "migration failed" {
			continue
		}
		n++
		delete(line, "time")
		if !maps.Equal(line, want) {
			t.Errorf("logged %v\nwant   %v", line, want)
		}
	}
	if n != 1 {
		t.Errorf("%d \"migration failed\" lines for %v, want 1; log:\n%s", n, err, logged)
	}
}

// The line carries the error, its SQLSTATE, constraint and HINT, and no other
// field of the error (TICK-1135): those can hold tenant data.
func TestUpLogsTheDatabaseErrorsHint(t *testing.T) {
	t.Parallel()
	wantFailedLine(t, refusingServer(t, &pgproto3.ErrorResponse{
		Severity: "ERROR", Code: "23514", Message: "refused",
		Detail: "secret", Hint: "Do not delete rows.", InternalQuery: "secret", Where: "secret",
		SchemaName: "secret", TableName: "secret", ColumnName: "secret", DataTypeName: "secret",
		ConstraintName: "t1135_check", File: "secret", Routine: "secret",
	}), map[string]any{"sqlstate": "23514", "constraint": "t1135_check", "hint": "Do not delete rows."})
}

// A field the error leaves empty is not logged.
func TestUpLogsOnlyTheFieldsTheDatabaseErrorSets(t *testing.T) {
	t.Parallel()
	wantFailedLine(t, refusingServer(t, &pgproto3.ErrorResponse{Severity: "ERROR", Code: "42501", Message: "denied"}),
		map[string]any{"sqlstate": "42501"})
}

// Nothing listens: the error is not the database's.
func TestUpLogsAFailureThatIsNotADatabaseError(t *testing.T) {
	t.Parallel()
	wantFailedLine(t, "127.0.0.1:1", map[string]any{})
}
