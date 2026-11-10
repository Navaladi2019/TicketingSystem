package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/auth"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/ratelimit"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// baseURL is oapi-codegen's BaseURL and the validator's Options.Prefix. The
// contract's paths already start with /api/v1, so both are empty.
const baseURL = ""

// newAPI registers the /api/v1 routes of the embedded contract on mux and
// returns the handler to serve. Authentication (routeAuth) wraps the whole
// mux, outside the generated router: the generated wrapper binds path, query
// and header parameters before it runs a MiddlewareFunc, so an auth
// middleware inside it would answer 400 to an unauthenticated request with a
// bad or missing parameter. Behind authentication each route runs the
// validator. Only the spec and docs routes are public. Every other path under
// /api/v1, matched or not, is authenticated first (fail closed): an unknown
// path or wrong method with no token is 401, and 404/405 only behind a valid
// token. Paths outside /api/v1 get the 404 problem.
func newAPI(log *slog.Logger, tel *telemetry.Telemetry, resolver identity.IdentityResolver, paths identity.GroupPathReader, mux *http.ServeMux, opts options) (http.Handler, error) {
	specData, err := api.Files.ReadFile(api.SpecFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", api.SpecFile, err)
	}
	spec, err := apihttp.LoadSpec(specData)
	if err != nil {
		return nil, err
	}
	errData, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		return nil, fmt.Errorf("read errors.yaml: %w", err)
	}
	reg, err := problem.LoadRegistry(errData)
	if err != nil {
		return nil, err
	}
	authn, err := auth.New(resolver, reg, log, tel.MeterProvider)
	if err != nil {
		return nil, err
	}

	limit, err := ratelimit.New(opts.limiter(), reg, log)
	if err != nil {
		return nil, err
	}

	a := apihttp.New(spec, reg, log, baseURL)
	h := &handlers{reg: reg, log: log, onboarder: opts.onboarder, paths: paths, pool: opts.pool, cursors: opts.cursors, dataFilterTimeout: opts.dataFilterTimeout}
	if opts.pool != nil {
		// The S10 composition (PLAN-003): the record ACL reads and writes,
		// no events yet. S11b swaps the sink.
		var masker read.Masker = read.DefinitionMasker{}
		if opts.auditMasker != nil {
			masker = opts.auditMasker
		}
		if opts.auditPool != nil {
			h.audit = &read.Reader{Pool: opts.auditPool, Masker: masker, MaxRecordEntries: opts.historyCap, OnRecordRows: opts.historyRows}
		}
		if h.historyTooLong, err = tel.MeterProvider.Meter("github.com/Navaladi2019/TicketingSystem/internal/server").Int64Counter(
			"nava.records.history.too_long",
			metric.WithDescription("Record history requests refused because the record has more audit entries than the cap."),
			metric.WithUnit("{request}")); err != nil {
			return nil, err
		}
		h.acl = authz.ACL{Policies: authz.PolicyStore{}}
		runner := pipeline.PoolRunner{Pool: opts.pool}
		if opts.allocPool != nil {
			runner.Pool = opts.allocPool
		}
		h.pipe = pipeline.New(h.acl, h.store, pipeline.NoopSink{}, runner, pipeline.SystemClock{})
	}
	api.HandlerWithOptions(
		api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  a.StrictRequestError,
			ResponseErrorHandlerFunc: h.responseError(a),
		}),
		api.StdHTTPServerOptions{
			BaseURL:          baseURL,
			BaseRouter:       mux,
			Middlewares:      []api.MiddlewareFunc{withRawQuery, a.Middleware()},
			ErrorHandlerFunc: a.ErrorHandler,
		},
	)
	mux.Handle(specPattern, specHandler(specData))
	mux.Handle(docsPattern, docsHandler(spec))

	served := a.Wrap(mux)
	// A tenant request is authenticated (401) and then rate limited (429)
	// before the generated router binds parameters or the validator reads the
	// input. Operator routes carry no tenant, so they are not limited here.
	// Tenant writes then need an Idempotency-Key (authn -> ratelimit ->
	// idempotency, the R2 tripwire). Without a pool there is no key store, so a
	// tenant write answers 501 rather than run unprotected.
	// The operator-only process answers every tenant route 404, so it needs no
	// key store: a tenant POST there must stay 404, not 501.
	var idem func(http.Handler) http.Handler
	if opts.operatorOnly {
		idem = func(next http.Handler) http.Handler { return next }
	} else if opts.pool != nil {
		m, err := idempotency.New(opts.pool, reg, log, tel.MeterProvider)
		if err != nil {
			return nil, err
		}
		idem = m.Handler
	} else {
		idem = refuseWrites(reg, log)
	}
	tenant := func(next http.Handler) http.Handler {
		keyed := idem(next)
		// Only a route of the contract needs a key: an unknown path or a wrong
		// method (Pattern is empty) must still get its 404 or 405.
		gated := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Pattern == "" {
				next.ServeHTTP(w, r)
				return
			}
			keyed.ServeHTTP(w, r)
		})
		return authn.Middleware(limit.Handler(gated))
	}
	authed := routeAuth(reg, opts.operatorOnly, tenant, operatorAuth(reg, log, opts.operators, resolver))(served)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		if pattern == specPattern || pattern == docsPattern || (pattern == "" && !underAPI(r.URL.Path)) {
			served.ServeHTTP(w, r)
			return
		}
		r.Pattern = pattern // the authentication logs name the route
		authed.ServeHTTP(w, r)
	}), nil
}

// underAPI reports whether path is the contract's /api/v1 prefix or below it.
// The prefix is matched as written: a differently cased or encoded spelling
// cannot match a route either, so it gets 404 and no data.
func underAPI(path string) bool {
	return path == "/api/v1" || strings.HasPrefix(path, "/api/v1/")
}

const (
	specPattern = "GET /api/v1/openapi.yaml"
	docsPattern = "GET /api/v1/docs"
)

// handlers implements the generated strict server. Operations the contract
// declares but a later task implements answer 501 not_implemented.
type handlers struct {
	reg       *problem.Registry
	log       *slog.Logger
	onboarder Onboarder
	// paths is read by GetMe only (ADR-002).
	paths identity.GroupPathReader
	// pool (nava_app) and cursors serve the entity-type endpoints; nil answers
	// 501 not_implemented.
	pool              *pgxpool.Pool
	cursors           *cursor.Keyring
	dataFilterTimeout time.Duration
	store             entitytypes.Store
	// pipe serves the record endpoints; nil answers 501 not_implemented.
	pipe *pipeline.Pipeline
	// acl decides record reads (and, inside pipe, writes) and gives the
	// policy every record body and audit diff is shaped with (P3-33). A
	// record or audit route without it answers 501 not_implemented, never an
	// unchecked or unshaped read.
	acl recordACL
	// audit serves the audit timeline and a record's history (P5-17) on the
	// nava_audit_reader pool (ADR-004-A1); nil answers 501 not_implemented.
	audit *read.Reader
	// historyTooLong counts the histories refused for their length; nil
	// counts nothing.
	historyTooLong metric.Int64Counter
}

// errNoPrincipal means a handler ran outside the authentication middleware:
// a wiring bug, answered 500.
var errNoPrincipal = errors.New("server: no resolved principal in the request context")

// GetMe returns the caller as the middleware resolved it (T17).
func (h *handlers) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	groups := p.Groups
	if groups == nil {
		groups = []uuid.UUID{}
	}
	raw, err := h.paths.GroupPaths(ctx, p.TenantID, p.PrincipalID) //nolint:forbidigo // ADR-002: GetMe is the one allowed GroupPaths caller
	if err != nil {
		return nil, err
	}
	// Strip the tenant prefix; an entry without it is dropped and logged
	// without its path, so one bad row cannot take /me down (ADR-002 section 3).
	prefix := identity.TenantGroupPrefix(p.TenantID)
	paths := make([]api.GroupPath, 0, len(raw))
	for _, gp := range raw {
		rest, ok := strings.CutPrefix(gp.Path, prefix)
		if !ok {
			h.log.ErrorContext(ctx, "group path outside tenant prefix",
				slog.String("tenant_id", p.TenantID.String()), slog.String("group_id", gp.ID.String()))
			continue
		}
		paths = append(paths, api.GroupPath{Id: gp.ID, Path: "/" + rest})
	}
	return api.GetMe200JSONResponse{UserId: p.PrincipalID, TenantId: p.TenantID, Groups: groups, IdpAlias: p.IdPAlias, GroupPaths: paths}, nil
}

// responseError answers a handler's problemError with its registered code and
// hands every other error to the shared handler.
func (h *handlers) responseError(a *apihttp.API) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		var pe *problemError
		if errors.As(err, &pe) {
			// The 4xx codes that name fields. Each is a constant, so the
			// registry guard sees a client error; any other code, fields or
			// not, goes through WriteError.
			if len(pe.fields) > 0 {
				switch pe.code {
				case problem.ValidationFailed:
					h.reg.Write(w, r, problem.ValidationFailed, pe.fields...)
					return
				case problem.InvalidDefinition:
					h.reg.Write(w, r, problem.InvalidDefinition, pe.fields...)
					return
				case problem.SchemaBreakingChange:
					h.reg.Write(w, r, problem.SchemaBreakingChange, pe.fields...)
					return
				case problem.IllegalTransition:
					h.reg.Write(w, r, problem.IllegalTransition, pe.fields...)
					return
				case problem.UniqueViolation:
					h.reg.Write(w, r, problem.UniqueViolation, pe.fields...)
					return
				case problem.RecordInvalid:
					h.reg.Write(w, r, problem.RecordInvalid, pe.fields...)
					return
				case problem.ConfigSecretInContent:
					h.reg.Write(w, r, problem.ConfigSecretInContent, pe.fields...)
					return
				case problem.DefinitionSecretRequired:
					h.reg.Write(w, r, problem.DefinitionSecretRequired, pe.fields...)
					return
				case problem.AssignmentGroupRemoved:
					h.reg.Write(w, r, problem.AssignmentGroupRemoved, pe.fields...)
					return
				case problem.InvalidIdPConfig:
					h.reg.Write(w, r, problem.InvalidIdPConfig, pe.fields...)
					return
				case problem.FieldNotWritable:
					h.reg.Write(w, r, problem.FieldNotWritable, pe.fields...)
					return
				case problem.MaskedField:
					h.reg.Write(w, r, problem.MaskedField, pe.fields...)
					return
				case problem.InvalidAccessPolicy:
					h.reg.Write(w, r, problem.InvalidAccessPolicy, pe.fields...)
					return
				}
			}
			h.reg.WriteError(w, r, h.log, pe.code, pe.err)
			return
		}
		a.StrictResponseError(w, r, err)
	}
}

// refuseWrites answers 501 to a tenant write when no key store is configured.
func refuseWrites(reg *problem.Registry, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
				next.ServeHTTP(w, r)
			default:
				reg.WriteError(w, r, log, problem.NotImplemented, errors.New("idempotency store not configured"))
			}
		})
	}
}
