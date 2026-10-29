// Package problem writes RFC 9457 application/problem+json error responses
// and holds the registry of error codes (api/errors.yaml).
//
// The body is built only from registry text and server-known values. Request
// input (a path value, a query value, a body field value) never goes into
// it: not in detail, not in instance, not in errors.
package problem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/httperr"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/strictyaml"
)

// ContentType is the Content-Type of every error response.
const ContentType = "application/problem+json"

// Code is a registered error code. Every constant here must be in
// api/errors.yaml, and the registry test fails if the two drift.
type Code string

// The registered codes.
const (
	TenantUnresolved    Code = "tenant_unresolved"
	InvalidToken        Code = "invalid_token"
	IdempotencyKeyReuse Code = "idempotency_key_reuse"
	IdempotencyInFlight Code = "idempotency_in_flight"
	Forbidden           Code = "forbidden"
	DomainClaimed       Code = "domain_claimed"
	InvalidIdPConfig    Code = "invalid_idp_config"
	OnboardingConflict  Code = "onboarding_conflict"
	InvalidCursor       Code = "invalid_cursor"
	RateLimited         Code = "rate_limited"
	NotFound            Code = "not_found"
	ValidationFailed    Code = "validation_failed"
	MethodNotAllowed    Code = "method_not_allowed"
	NotImplemented      Code = "not_implemented"
	InternalError       Code = "internal_error"
)

// Codes lists every Code constant.
func Codes() []Code {
	return []Code{
		TenantUnresolved, InvalidToken, IdempotencyKeyReuse, IdempotencyInFlight, Forbidden, DomainClaimed, InvalidIdPConfig, OnboardingConflict, InvalidCursor,
		RateLimited, NotFound, ValidationFailed, MethodNotAllowed,
		NotImplemented, InternalError,
	}
}

// Entry is one registry row.
type Entry struct {
	Code   Code   `yaml:"code"`
	Status int    `yaml:"status"`
	Title  string `yaml:"title"`
	Detail string `yaml:"detail"`
}

// FieldError names one failing field of a validation_failed response.
type FieldError struct {
	Pointer string `json:"pointer"`
	Rule    string `json:"rule,omitempty"`
}

// Registry maps a Code to its status, title and fixed detail text.
type Registry struct {
	entries map[Code]Entry
}

// LoadRegistry parses errors.yaml through the strict loader.
func LoadRegistry(data []byte) (*Registry, error) {
	var doc struct {
		Codes []Entry `yaml:"codes"`
	}
	if err := strictyaml.Decode(data, &doc); err != nil {
		return nil, fmt.Errorf("error registry: %w", err)
	}
	reg := &Registry{entries: make(map[Code]Entry, len(doc.Codes))}
	for _, e := range doc.Codes {
		if e.Code == "" || e.Title == "" || e.Detail == "" || e.Status < 400 || e.Status > 599 {
			return nil, fmt.Errorf("error registry: incomplete entry %q", e.Code)
		}
		if _, dup := reg.entries[e.Code]; dup {
			return nil, fmt.Errorf("error registry: duplicate code %q", e.Code)
		}
		reg.entries[e.Code] = e
	}
	for _, c := range Codes() {
		if _, ok := reg.entries[c]; !ok {
			return nil, fmt.Errorf("error registry: code %q is used in Go but not registered", c)
		}
	}
	return reg, nil
}

// Lookup returns the entry for code.
func (reg *Registry) Lookup(code Code) (Entry, bool) {
	e, ok := reg.entries[code]
	return e, ok
}

// Body is the wire shape, the Problem schema of api/openapi.yaml (api/openapi.bundled.yaml in the bundle).
type Body struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Code     Code         `json:"code"`
	Detail   string       `json:"detail"`
	Instance string       `json:"instance"`
	Errors   []FieldError `json:"errors,omitempty"`
}

// Write sends the problem for code, for a 4xx only: a 5xx goes through
// WriteError, which reports the cause (a test enforces it). The status comes
// from the registry, so a handler cannot send a code with a status the
// registry does not list. An unregistered code is a programming error and is
// sent as internal_error.
func (reg *Registry) Write(w http.ResponseWriter, r *http.Request, code Code, errs ...FieldError) {
	write(w, r, reg.Body(r.Context(), code, errs...))
}

// WriteError sends the problem for code like Write and, for a 5xx, first
// reports err through httperr.Report: the span is marked failed and the error
// text goes to the log only. The body never carries err's text.
func (reg *Registry) WriteError(w http.ResponseWriter, r *http.Request, log *slog.Logger, code Code, err error, attrs ...slog.Attr) {
	if e, ok := reg.Lookup(code); !ok || e.Status >= http.StatusInternalServerError {
		httperr.Report(r.Context(), log, "api error", httperr.TypeOf(err), err, attrs...)
	}
	reg.Write(w, r, code)
}

// Body builds the problem for code, with the request id of ctx as instance. An
// unregistered code is a programming error and is built as internal_error.
func (reg *Registry) Body(ctx context.Context, code Code, errs ...FieldError) Body {
	e, ok := reg.Lookup(code)
	if !ok {
		e, _ = reg.Lookup(InternalError)
		code = InternalError
	}
	return Body{
		Type:     "urn:nava:problem:" + string(code),
		Title:    e.Title,
		Status:   e.Status,
		Code:     code,
		Detail:   e.Detail,
		Instance: "urn:nava:request:" + requestid.FromContext(ctx),
		Errors:   errs,
	}
}

// Internal writes the internal_error problem without a registry, for the panic
// recovery path, which must not fail. The text is checked against the registry
// by the registry test.
func Internal(w http.ResponseWriter, r *http.Request) {
	write(w, r, Body{
		Type:     "urn:nava:problem:" + string(InternalError),
		Title:    "Internal error",
		Status:   http.StatusInternalServerError,
		Code:     InternalError,
		Detail:   "The server failed to handle the request. Quote the instance when reporting it.",
		Instance: "urn:nava:request:" + requestid.FromContext(r.Context()),
	})
}

func write(w http.ResponseWriter, _ *http.Request, b Body) {
	out, err := json.Marshal(b)
	if err != nil { // a Body of strings and ints cannot fail to marshal
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", ContentType)
	h.Set("Content-Length", fmt.Sprint(len(out)))
	w.WriteHeader(b.Status)
	_, _ = w.Write(out)
}
