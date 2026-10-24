package httpobs

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"

	"github.com/felixge/httpsnoop"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/httperr"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
)

// serve runs next with w wrapped once, and returns what the response
// observed. A handler panic becomes a 500, so the request still gets its
// access line and HTTP metrics, and the server keeps serving. The panic is
// logged with its stack, on the request's context (trace and request id), and
// marks the span as an error.
//
// If the response was already committed, a 500 can no longer be sent: the
// panic is re-raised as http.ErrAbortHandler, so net/http drops the connection
// and the client cannot mistake the partial response for a complete one.
// A handler's own http.ErrAbortHandler is passed through untouched.
//
// It never replaces the *http.Request: ServeMux sets r.Pattern on the request
// it is given, and access reads it from the same one.
func serve(log *slog.Logger, next http.Handler, w http.ResponseWriter, r *http.Request) (resp response) {
	ww := resp.wrap(w)
	defer func() {
		v := recover()
		if v == nil {
			return
		}
		if v == http.ErrAbortHandler { //nolint:errorlint // net/http compares by identity
			panic(v)
		}
		ctx := r.Context()
		httperr.Report(ctx, log, "http handler panic", "panic", panicError(v),
			slog.Any("panic", v), slog.String("stack", string(debug.Stack())))
		if resp.committed {
			panic(http.ErrAbortHandler)
		}
		// Headers the handler set before panicking (Cache-Control, Set-Cookie,
		// Content-Encoding) describe a response that never happened.
		h := ww.Header()
		id := h.Get(requestid.Header)
		clear(h)
		h.Set(requestid.Header, id)
		problem.Internal(ww, r)
	}()
	next.ServeHTTP(ww, r)
	return resp
}

// response is what the wrapped writer observed: the status sent and whether
// the response is committed, so a 500 can no longer replace it.
type response struct {
	code      int
	committed bool
}

// status is the code the client got: net/http sends 200 when a handler
// returns without writing.
func (s *response) status() int {
	if s.code == 0 {
		return http.StatusOK
	}
	return s.code
}

// commit records code as the response status, once.
func (s *response) commit(code int) {
	if !s.committed {
		s.code, s.committed = code, true
	}
}

// wrap returns w with hooks that record the response. httpsnoop keeps every
// optional interface of w (Flusher, Hijacker, ReaderFrom), as otelhttp's own
// wrapper does.
func (s *response) wrap(w http.ResponseWriter) http.ResponseWriter {
	return httpsnoop.Wrap(w, httpsnoop.Hooks{
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(code int) {
				// 1xx responses are informational and commit nothing, except
				// 101: the connection has switched protocols.
				if code >= 200 || code == http.StatusSwitchingProtocols {
					s.commit(code)
				}
				next(code)
			}
		},
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) { s.commit(http.StatusOK); return next(b) }
		},
		WriteString: func(next httpsnoop.WriteStringFunc) httpsnoop.WriteStringFunc {
			return func(str string) (int, error) { s.commit(http.StatusOK); return next(str) }
		},
		ReadFrom: func(next httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
			return func(src io.Reader) (int64, error) { s.commit(http.StatusOK); return next(src) }
		},
		Flush: func(next httpsnoop.FlushFunc) httpsnoop.FlushFunc {
			return func() { s.commit(http.StatusOK); next() }
		},
		FlushError: func(next httpsnoop.FlushErrorFunc) httpsnoop.FlushErrorFunc {
			return func() error { s.commit(http.StatusOK); return next() }
		},
		Hijack: func(next httpsnoop.HijackFunc) httpsnoop.HijackFunc {
			return func() (net.Conn, *bufio.ReadWriter, error) { s.commit(http.StatusOK); return next() }
		},
	})
}

// panicError is the recovered value as an error, wrapping it when it is one.
func panicError(v any) error {
	if err, ok := v.(error); ok {
		return fmt.Errorf("panic: %w", err)
	}
	return fmt.Errorf("panic: %v", v)
}
