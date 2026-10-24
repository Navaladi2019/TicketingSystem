package httperr_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/httperr"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

const secret = "alice@tenant-a.test"

type run struct {
	rec   *httptest.ResponseRecorder
	logs  []map[string]any
	spans tracetest.SpanStubs
}

// serve sends GET /x through the real telemetry and httpobs middleware to a
// handler that fails with an error carrying secret.
func serve(t *testing.T, code problem.Code) run {
	t.Helper()
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	exp := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), config.Telemetry{}, telemetry.WithSampler(sdktrace.AlwaysSample()), telemetry.WithSpanExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(logging.NewContextHandler(slog.NewJSONHandler(&logs, nil)))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /x", func(w http.ResponseWriter, r *http.Request) {
		reg.WriteError(w, r, log, code, errors.New("pq: invalid input for "+secret))
	})
	h := httpobs.Middleware(log, tel, mux)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", nil))
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })

	r := run{rec: rec, spans: exp.GetSpans()}
	for _, l := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("log line %q: %v", l, err)
		}
		r.logs = append(r.logs, m)
	}
	return r
}

func TestInternalErrorIsOnSpanWithoutTextAndInLogWithText(t *testing.T) {
	t.Parallel()
	r := serve(t, problem.InternalError)

	if len(r.spans) != 1 {
		t.Fatalf("%d spans, want 1", len(r.spans))
	}
	span := r.spans[0]
	if span.Status.Code != codes.Error {
		t.Errorf("span status = %v, want Error", span.Status)
	}
	var errType string
	for _, kv := range span.Attributes {
		if kv.Key == attribute.Key("error.type") {
			errType = kv.Value.AsString()
		}
	}
	if errType != "*errors.errorString" {
		t.Errorf("error.type = %q, want the Go type", errType)
	}
	dump, _ := json.Marshal(struct {
		Attrs  []attribute.KeyValue
		Status any
		Events any
	}{span.Attributes, span.Status, span.Events})
	for _, ev := range span.Events {
		for _, kv := range ev.Attributes {
			if kv.Key == "exception.message" {
				t.Errorf("exported event keeps exception.message %q", kv.Value.AsString())
			}
		}
	}
	if strings.Contains(string(dump), secret) || strings.Contains(span.Status.Description, secret) {
		t.Errorf("exported span carries the error text: %s", dump)
	}

	var errLines []map[string]any
	for _, l := range r.logs {
		if l["level"] == "ERROR" {
			errLines = append(errLines, l)
		}
	}
	if len(errLines) != 1 {
		t.Fatalf("%d ERROR lines, want exactly 1: %v", len(errLines), r.logs)
	}
	line := errLines[0]
	if s, _ := line["err"].(string); !strings.Contains(s, secret) {
		t.Errorf("ERROR line err = %v, want the error text", line["err"])
	}
	if line["trace_id"] != span.SpanContext.TraceID().String() || line["span_id"] == nil || line["request_id"] == nil {
		t.Errorf("ERROR line ids = %v, want trace_id %s", line, span.SpanContext.TraceID())
	}
}

func TestProblemBodyHasNoErrorText(t *testing.T) {
	t.Parallel()
	r := serve(t, problem.InternalError)
	if r.rec.Code != http.StatusInternalServerError || r.rec.Header().Get("Content-Type") != problem.ContentType {
		t.Errorf("response = %d %s", r.rec.Code, r.rec.Header().Get("Content-Type"))
	}
	if strings.Contains(r.rec.Body.String(), secret) || strings.Contains(r.rec.Body.String(), "pq:") {
		t.Errorf("body carries the error text: %s", r.rec.Body)
	}
}

func TestClientErrorIsNotReported(t *testing.T) {
	t.Parallel()
	r := serve(t, problem.NotFound)
	if r.rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", r.rec.Code)
	}
	for _, l := range r.logs {
		if l["level"] == "ERROR" {
			t.Errorf("4xx logged an ERROR line: %v", l)
		}
	}
	if len(r.spans) == 1 && r.spans[0].Status.Code == codes.Error {
		t.Errorf("4xx marked the span as Error")
	}
}

// TestRecordErrorOnlyInHelper: handlers report through httperr.Report, so the
// message cannot be recorded on a span and then lost for want of a log line.
func TestRecordErrorOnlyInHelper(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..") // internal/
	allowed := map[string]bool{
		filepath.Join(root, "platform", "httperr", "httperr.go"): true,
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || allowed[path] {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(src), ".RecordError(") {
			t.Errorf("%s calls RecordError; use httperr.Report", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTypeOfIsTheRootType(t *testing.T) {
	t.Parallel()
	root := errors.New("pq: " + secret)
	wrapped := fmt.Errorf("load item: %w", fmt.Errorf("query: %w", root))
	if got := httperr.TypeOf(wrapped); got != "*errors.errorString" {
		t.Errorf("TypeOf(wrapped) = %q, want the root type *errors.errorString", got)
	}
	if got := httperr.TypeOf(root); got != "*errors.errorString" {
		t.Errorf("TypeOf(root) = %q", got)
	}
}

// TestRegistryWriteIsClientErrorsOnly: Registry.Write and Body send a problem
// without reporting a cause, so they may only be given a constant code whose
// registry status is below 500. A 5xx goes through WriteError. The 501
// not_implemented placeholder is the one allowed exception.
func TestRegistryWriteIsClientErrorsOnly(t *testing.T) {
	t.Parallel()
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()

	// const name -> code value, from the problem package itself.
	pf, err := parser.ParseFile(fset, filepath.Join("..", "problem", "problem.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	consts := map[string]problem.Code{}
	for _, d := range pf.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.CONST {
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, n := range vs.Names {
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok {
						consts[n.Name] = problem.Code(strings.Trim(lit.Value, `"`))
					}
				}
			}
		}
	}

	allowed := map[string]bool{ // file -> NotImplemented placeholder
		filepath.Join("..", "..", "server", "api.go"): true,
	}
	root := filepath.Join("..", "..")
	calls := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
			filepath.Base(filepath.Dir(path)) == "problem" {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			idx := -1
			switch {
			case sel.Sel.Name == "Write" && len(call.Args) >= 3: // reg.Write(w, r, code, ...)
				idx = 2
			case sel.Sel.Name == "Body" && len(call.Args) >= 2: // reg.Body(ctx, code, ...)
				idx = 1
			default:
				return true
			}
			calls++
			pos := fset.Position(call.Pos())
			arg, ok := call.Args[idx].(*ast.SelectorExpr)
			if !ok {
				t.Errorf("%s: %s code is not a problem.<Const>; use WriteError for anything that can be a 5xx", pos, sel.Sel.Name)
				return true
			}
			code, known := consts[arg.Sel.Name]
			e, registered := reg.Lookup(code)
			switch {
			case !known || !registered:
				t.Errorf("%s: %s.%s is not a registered problem code", pos, sel.Sel.Name, arg.Sel.Name)
			case e.Status >= http.StatusInternalServerError && !(allowed[path] && code == problem.NotImplemented):
				t.Errorf("%s: %s sends %s (%d); use WriteError so the cause is reported", pos, sel.Sel.Name, code, e.Status)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls == 0 {
		t.Error("found no registry Write or Body calls; the scan is broken")
	}
}
