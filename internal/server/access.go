package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Access endpoints (PLAN-003 S10, P3-35): read and publish an entity type's
// ACL, and ask why a user can or cannot read a record. All three are tenant
// admin only (authz.RequireTenantAdminTx, first, inside the one transaction)
// and see their own tenant only.

// accessBody is the JSON of an AccessPolicy.
type accessBody struct {
	Version     int             `json:"version"`
	Rules       json.RawMessage `json:"rules"`
	Masks       []authz.Mask    `json:"masks"`
	PublishedBy *uuid.UUID      `json:"published_by"`
	PublishedAt time.Time       `json:"published_at"`
	Warnings    []warningBody   `json:"warnings,omitempty"`
}

// warningBody is one lint of a publish that was accepted.
type warningBody struct {
	Pointer string `json:"pointer"`
	Code    string `json:"code"`
}

// accessResponse is a 200 AccessPolicy with its ETag. warnings are the lints
// of a publish; a read has none.
type accessResponse struct {
	pub      *authz.Published
	warnings []authz.Warning
}

func (r accessResponse) write(w http.ResponseWriter) error {
	// Rules are always format 2: a stored format-1 version is converted on read
	// (D12), and a format-2 version reads back as published.
	rules, err := authz.Encode(r.pub.Document)
	if err != nil {
		return err
	}
	masks := r.pub.Masks
	if masks == nil {
		masks = []authz.Mask{}
	}
	body := accessBody{Version: r.pub.Version, Rules: rules, Masks: masks, PublishedBy: r.pub.PublishedBy, PublishedAt: r.pub.PublishedAt.UTC()}
	for _, wn := range r.warnings {
		body.Warnings = append(body.Warnings, warningBody{wn.Pointer, wn.Code})
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", etag(int64(r.pub.Version)))
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	return err
}

func (r accessResponse) VisitGetEntityTypeAccessResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r accessResponse) VisitPutEntityTypeAccessResponse(w http.ResponseWriter) error {
	return r.write(w)
}

var (
	_ api.GetEntityTypeAccessResponseObject = accessResponse{}
	_ api.PutEntityTypeAccessResponseObject = accessResponse{}
)

func (r *notFoundResponse) VisitGetEntityTypeAccessResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r *notFoundResponse) VisitPutEntityTypeAccessResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r *notFoundResponse) VisitExplainAccessResponse(w http.ResponseWriter) error { return r.write(w) }

func (r *versionConflictResponse) VisitPutEntityTypeAccessResponse(w http.ResponseWriter) error {
	return r.VisitPatchRecordResponse(w)
}

var (
	_ api.GetEntityTypeAccessResponseObject = (*notFoundResponse)(nil)
	_ api.PutEntityTypeAccessResponseObject = (*notFoundResponse)(nil)
	_ api.PutEntityTypeAccessResponseObject = (*versionConflictResponse)(nil)
	_ api.ExplainAccessResponseObject       = (*notFoundResponse)(nil)
)

// GetEntityTypeAccess serves GET /api/v1/entity-types/{key}/access.
func (h *handlers) GetEntityTypeAccess(ctx context.Context, req api.GetEntityTypeAccessRequestObject) (api.GetEntityTypeAccessResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	var pub *authz.Published
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		if impossibleKey(req.Key) {
			return authz.ErrNoEntityType
		}
		var err error
		if pub, err = (authz.PolicyStore{}).Newest(ctx, tx, req.Key); err != nil {
			return err
		}
		if pub == nil {
			return authz.ErrNoEntityType // none published reads as missing
		}
		return nil
	})
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return nil, &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, authz.ErrNoEntityType):
		return h.notFound(ctx), nil
	case err != nil:
		return nil, err
	}
	return accessResponse{pub: pub}, nil
}

// aclIfMatch reads an If-Match header into the ACL versions it names. Unlike a
// record's, version 0 is a real value (the first publish is W/"0"), so a value
// that is not a version number names -1, which matches nothing. ok is false
// when the header is missing, empty or `*`.
func aclIfMatch(v *string) (versions []int, ok bool) {
	if v == nil || strings.TrimSpace(*v) == "" || strings.TrimSpace(*v) == "*" {
		return nil, false
	}
	for m := range strings.SplitSeq(*v, ",") {
		m = strings.Trim(strings.TrimPrefix(strings.TrimSpace(m), "W/"), `"`)
		n, err := strconv.Atoi(m)
		if err != nil || n < 0 || strconv.Itoa(n) != m {
			n = -1
		}
		versions = append(versions, n)
	}
	return versions, true
}

// accessPolicyError is a refused publish body.
type accessPolicyError struct{ problems []authz.Problem }

func (e *accessPolicyError) Error() string { return "invalid access policy" }

func (h *handlers) invalidPolicy(ctx context.Context, ps []authz.Problem) *problemError {
	fields := make([]problem.FieldError, len(ps))
	for i, pr := range ps {
		fields[i] = invalid(pr.Pointer, pr.Rule)
		if pr.Rule == authz.RuleDocumentTooLarge {
			fields[i].Detail = fmt.Sprintf("The %s exceed the platform's configured limit of %d bytes, measured on the stored form.", strings.TrimPrefix(pr.Pointer, "/"), h.aclLimit())
		}
	}
	// Pointers and rules only: the body never reaches the log.
	h.log.WarnContext(ctx, "access policy refused",
		slog.String("code", string(problem.InvalidAccessPolicy)), slog.Int("problems", len(ps)))
	return &problemError{problem.InvalidAccessPolicy, fields, errors.New("invalid access policy")}
}

// aclLimit is the byte limit on rules and on masks.
func (h *handlers) aclLimit() int {
	if h.aclMaxBytes > 0 {
		return h.aclMaxBytes
	}
	return authz.DefaultMaxDocumentBytes
}

// pgCheckProblem maps a 23514 from acl_versions to the 422 it stands for: the
// byte constraints to document_too_large, a check or trigger refusal with the
// JSON DETAIL of TICK-940 to its code and pointer. Any other 23514 is a 422
// with no pointer. The route checks first, so this is the backstop; it never
// answers 500.
func pgCheckProblem(err error) ([]authz.Problem, bool) {
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23514" {
		return nil, false
	}
	switch pe.ConstraintName {
	case "acl_versions_bytes_rules_check":
		return []authz.Problem{{Pointer: "/rules", Rule: authz.RuleDocumentTooLarge}}, true
	case "acl_versions_bytes_masks_check":
		return []authz.Problem{{Pointer: "/masks", Rule: authz.RuleDocumentTooLarge}}, true
	}
	var d struct {
		Column  string `json:"column"`
		Code    string `json:"code"`
		Pointer string `json:"pointer"`
	}
	if json.Unmarshal([]byte(pe.Detail), &d) == nil && d.Code != "" && (d.Column == "rules" || d.Column == "masks") {
		return []authz.Problem{{Pointer: "/" + d.Column + d.Pointer, Rule: d.Code}}, true
	}
	return []authz.Problem{{Pointer: "", Rule: authz.RuleValue}}, true
}

// PutEntityTypeAccess serves PUT /api/v1/entity-types/{key}/access.
func (h *handlers) PutEntityTypeAccess(ctx context.Context, req api.PutEntityTypeAccessRequestObject) (api.PutEntityTypeAccessResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	var raw json.RawMessage
	if req.Body != nil {
		raw = *req.Body
	}
	store := authz.PolicyStore{}
	// The limit and every check that needs no database come before the
	// transaction opens: a refused body costs no connection (A1-2c).
	if problems := authz.CheckSize(raw, h.aclLimit()); len(problems) > 0 {
		return nil, h.invalidPolicy(ctx, problems)
	}
	var prep *authz.Prepared
	var resp accessResponse
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		if impossibleKey(req.Key) {
			return authz.ErrNoEntityType
		}
		cur, err := store.Newest(ctx, tx, req.Key)
		if err != nil {
			return err
		}
		have := 0
		if cur != nil {
			have = cur.Version
		}
		versions, ok := aclIfMatch(req.Params.IfMatch)
		if !ok {
			return errIfMatchRequired
		}
		if !contains(versions, have) {
			return &authz.ConflictError{Current: have}
		}
		// The rest of the body checks, after the checks that come first in the
		// documented order (403, 404, 428, 412).
		var problems []authz.Problem
		if prep, problems = authz.Prepare(raw, h.aclLimit()); len(problems) > 0 {
			return &accessPolicyError{problems}
		}
		props, err := h.schemaProperties(ctx, tx, req.Key)
		if err != nil {
			return err
		}
		fieldProblems := authz.CheckMaskFields(prep.MaskList, props)
		scope, err := authz.NotInTenant(ctx, tx, prep.Refs)
		if err != nil {
			return err
		}
		if prep.Format == 2 {
			for i := range scope {
				scope[i].Rule = authz.RuleForeignID
			}
		}
		if all := append(fieldProblems, scope...); len(all) > 0 {
			return &accessPolicyError{all}
		}
		pub, err := store.PublishPrepared(ctx, tx, req.Key, have, authz.Publisher{Principal: &p.PrincipalID, Source: audit.SourceUser}, prep)
		if err != nil {
			return err
		}
		resp = accessResponse{pub: pub, warnings: prep.Warnings}
		// The key completes in this transaction, with the write (TICK-482).
		return idempotency.CompleteInTx(ctx, tx, resp.VisitPutEntityTypeAccessResponse)
	})
	var (
		conflict *authz.ConflictError
		bad      *accessPolicyError
	)
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return nil, &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, authz.ErrNoEntityType):
		return h.notFound(ctx), nil
	case errors.Is(err, errIfMatchRequired):
		return nil, &problemError{problem.IfMatchRequired, nil, err}
	case errors.As(err, &conflict):
		return h.versionConflict(ctx, int64(conflict.Current)), nil
	case errors.As(err, &bad):
		return nil, h.invalidPolicy(ctx, bad.problems)
	}
	if ps, ok := pgCheckProblem(err); ok {
		h.log.WarnContext(ctx, "access policy refused by the database backstop", slog.String("code", ps[0].Rule))
		return nil, h.invalidPolicy(ctx, ps)
	}
	if err != nil {
		return nil, err
	}
	for _, wn := range resp.warnings {
		h.log.WarnContext(ctx, "access policy lint", slog.String("code", wn.Code), slog.String("pointer", wn.Pointer),
			slog.String("tenant_id", p.TenantID.String()), slog.String("entity_key", req.Key))
	}
	h.log.InfoContext(ctx, "access policy published",
		slog.String("tenant_id", p.TenantID.String()), slog.String("entity_key", req.Key),
		slog.Int("version", resp.pub.Version), slog.Int("policies", len(resp.pub.Document.Policies)), slog.Int("masks", len(resp.pub.Masks)))
	return resp, nil
}

func contains(vs []int, v int) bool {
	return slices.Contains(vs, v)
}

// schemaProperties lists the top-level properties of the entity type's
// current schema, for the mask field check.
func (h *handlers) schemaProperties(ctx context.Context, tx pgx.Tx, key string) ([]string, error) {
	v, err := h.store.GetCurrent(ctx, tx, key)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(v.Schema, &doc); err != nil {
		return nil, fmt.Errorf("server: read schema properties: %w", err)
	}
	out := make([]string, 0, len(doc.Properties))
	for k := range doc.Properties {
		out = append(out, k)
	}
	return out, nil
}

// ExplainAccess serves GET /api/v1/access/explain.
func (h *handlers) ExplainAccess(ctx context.Context, req api.ExplainAccessRequestObject) (api.ExplainAccessResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	var ex authz.Explanation
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		recID, rerr := uuid.Parse(req.Params.Record)
		userID, uerr := uuid.Parse(req.Params.User)
		if impossibleKey(req.Params.Entity) || rerr != nil || uerr != nil {
			return pipeline.ErrNotFound
		}
		rec, err := pipeline.Get(ctx, tx, p.TenantID, recID)
		if err != nil {
			return err
		}
		if rec.EntityKey != req.Params.Entity {
			return pipeline.ErrNotFound
		}
		target, found, err := authz.PrincipalOf(ctx, tx, userID)
		if err != nil {
			return err
		}
		if !found {
			return pipeline.ErrNotFound
		}
		pol, err := (authz.PolicyStore{}).Current(ctx, tx, req.Params.Entity)
		if err != nil {
			return err
		}
		row, err := pipeline.RowOf(ctx, tx, target, rec)
		if err != nil {
			return err
		}
		// The target's units and the record's closure answer, as its own
		// read would bind them (TICK-1034).
		if pol, err = authz.ResolveUnder(ctx, tx, pol, target, row); err != nil {
			return err
		}
		ex = authz.Explain(pol, p, target, row)
		return nil
	})
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return nil, &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, pipeline.ErrNotFound):
		return h.notFound(ctx), nil
	case err != nil:
		return nil, err
	}
	out := api.AccessExplanation{AclVersion: ex.Version, Decision: api.AccessExplanationDecision("deny"), MaskedFields: ex.MaskedFields}
	if ex.Allow {
		out.Decision = "allow"
		out.RuleId = &ex.RuleID
		out.Policy = &ex.Policy
	}
	trace := make([]struct {
		Condition string `json:"condition"`
		Policy    string `json:"policy"`
		Value     bool   `json:"value"`
	}, len(ex.Trace))
	for i, t := range ex.Trace {
		trace[i].Condition, trace[i].Policy, trace[i].Value = t.Condition, t.Policy, t.Value
	}
	out.Trace = &trace
	invalid := make([]struct {
		Code    string `json:"code"`
		Pointer string `json:"pointer"`
		Policy  string `json:"policy"`
	}, len(ex.Invalid))
	for i, v := range ex.Invalid {
		invalid[i].Code, invalid[i].Pointer, invalid[i].Policy = v.Code, v.Pointer, v.Policy
	}
	out.Invalid = &invalid
	return api.ExplainAccess200JSONResponse(out), nil
}
