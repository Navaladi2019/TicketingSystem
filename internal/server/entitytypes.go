package server

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Entity-type endpoints (TICK-343, P3-05a; publish is TICK-365, P3-05b).

const (
	listEntityTypesRoute = "listEntityTypes"
	defaultListLimit     = 50
	maxNameRunes         = 120
	// maxKeyBytes bounds a looked-up key: it cannot exist beyond it.
	maxKeyBytes = 64
)

var (
	keyPattern    = regexp.MustCompile(`^[a-z][a-z0-9_]{1,40}$`)
	prefixPattern = regexp.MustCompile(`^[A-Z]{2,6}$`)
)

var errNotConfigured = errors.New("entity types: database not configured")

func notImplemented() error {
	return &problemError{problem.NotImplemented, nil, errNotConfigured}
}

// ListEntityTypes serves GET /api/v1/entity-types.
func (h *handlers) ListEntityTypes(ctx context.Context, req api.ListEntityTypesRequestObject) (api.ListEntityTypesResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	if h.pool == nil || h.cursors == nil {
		return nil, notImplemented()
	}
	limit := defaultListLimit
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	// The listing has no filter, so the claims' query hash is that of no query.
	claims, err := cursor.ClaimsFor(ctx, listEntityTypesRoute, url.Values{})
	if err != nil {
		return nil, err
	}
	after := ""
	if req.Params.Cursor != nil {
		pos, err := h.cursors.Decode(claims, *req.Params.Cursor)
		if err != nil {
			return nil, &problemError{problem.InvalidCursor, nil, err}
		}
		after = string(pos)
	}
	var rows []entitytypes.Summary
	if err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) (err error) {
		rows, err = h.store.List(ctx, tx, after, limit+1)
		return err
	}); err != nil {
		return nil, err
	}
	out := api.EntityTypeList{Items: make([]api.EntityTypeSummary, 0, min(len(rows), limit))}
	for _, r := range rows[:min(len(rows), limit)] {
		out.Items = append(out.Items, api.EntityTypeSummary{
			Key: r.Key, Name: r.Name, NumberPrefix: r.NumberPrefix, CurrentVersion: r.CurrentVersion, UpdatedAt: r.UpdatedAt,
		})
	}
	if len(rows) > limit {
		next, err := h.cursors.Encode(claims, []byte(rows[limit-1].Key))
		if err != nil {
			return nil, err
		}
		out.NextCursor = &next
	}
	return api.ListEntityTypes200JSONResponse(out), nil
}

// GetEntityType serves GET /api/v1/entity-types/{key}.
func (h *handlers) GetEntityType(ctx context.Context, req api.GetEntityTypeRequestObject) (api.GetEntityTypeResponseObject, error) {
	r, err := h.readVersion(ctx, req.Key, 0)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// GetEntityTypeVersion serves GET /api/v1/entity-types/{key}/versions/{n}.
func (h *handlers) GetEntityTypeVersion(ctx context.Context, req api.GetEntityTypeVersionRequestObject) (api.GetEntityTypeVersionResponseObject, error) {
	if req.N < 1 {
		return h.notFound(ctx), nil
	}
	r, err := h.readVersion(ctx, req.Key, int(req.N))
	if err != nil {
		return nil, err
	}
	return r, nil
}

// versionReply is either the version or the 404.
type versionReply interface {
	api.GetEntityTypeResponseObject
	api.GetEntityTypeVersionResponseObject
}

// readVersion reads the current version (version 0) or the given one. An
// unknown key, another tenant's key and a missing version give the one 404.
func (h *handlers) readVersion(ctx context.Context, key string, version int) (versionReply, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	if len(key) > maxKeyBytes || !utf8.ValidString(key) || strings.IndexByte(key, 0) >= 0 {
		return h.notFound(ctx), nil // cannot exist, and must not reach the database
	}
	var v *entitytypes.Version
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) (err error) {
		if version == 0 {
			v, err = h.store.GetCurrent(ctx, tx, key)
		} else {
			v, err = h.store.GetVersion(ctx, tx, key, version)
		}
		return err
	})
	switch {
	case errors.Is(err, entitytypes.ErrNotFound):
		return h.notFound(ctx), nil
	case err != nil:
		return nil, err
	}
	return &entityTypeResponse{status: http.StatusOK, v: v}, nil
}

// createRequest is the body of POST /api/v1/entity-types, field by field.
type createRequest struct {
	key, name, prefix string
	def               metadata.Definition
}

// CreateEntityType serves POST /api/v1/entity-types. The admin check comes
// first, inside the create transaction; the body is read no further than as
// well-formed JSON before it.
func (h *handlers) CreateEntityType(ctx context.Context, req api.CreateEntityTypeRequestObject) (api.CreateEntityTypeResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	var raw json.RawMessage
	if req.Body != nil {
		raw = *req.Body
	}
	var v *entitytypes.Version
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		cr, perr := parseCreate(raw)
		if perr != nil {
			return perr
		}
		var err error
		v, err = h.store.Create(ctx, tx, p.TenantID, entitytypes.NewInput{Name: cr.name, PublishedBy: p.PrincipalID, Definition: cr.def})
		return err
	})
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return nil, &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, entitytypes.ErrKeyExists):
		return nil, &problemError{problem.EntityTypeExists, nil, err}
	case errors.Is(err, entitytypes.ErrPrefixTaken):
		return nil, &problemError{problem.NumberPrefixTaken, nil, err}
	case err != nil:
		return nil, err
	}
	h.log.InfoContext(ctx, "entity type created",
		slog.String("tenant_id", p.TenantID.String()), slog.String("entity_key", v.Key), slog.Int("version", v.Version))
	return &entityTypeResponse{status: http.StatusCreated, v: v}, nil
}

// PublishEntityTypeVersion serves POST /api/v1/entity-types/{key}/versions.
// Like create, the admin check is first and inside the one transaction that
// locks, classifies and writes. published_by is always the principal; a
// request with none is refused before any row is touched.
func (h *handlers) PublishEntityTypeVersion(ctx context.Context, req api.PublishEntityTypeVersionRequestObject) (api.PublishEntityTypeVersionResponseObject, error) {
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
	var v *entitytypes.Version
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := authz.RequireTenantAdminTx(ctx, tx, p); err != nil {
			return err
		}
		// The key shape is checked before the body, so an impossible key gets
		// the same 404 whatever the body is.
		if len(req.Key) > maxKeyBytes || !utf8.ValidString(req.Key) || strings.IndexByte(req.Key, 0) >= 0 {
			return entitytypes.ErrNotFound // cannot exist, and must not reach the database
		}
		def, perr := parsePublish(raw)
		if perr != nil {
			return perr
		}
		var err error
		v, err = h.store.Publish(ctx, tx, p.TenantID, entitytypes.PublishInput{Key: req.Key, PublishedBy: p.PrincipalID, Definition: def})
		return err
	})
	var (
		breaking *entitytypes.BreakingError
		invalidD *metadata.ErrInvalidDefinition
	)
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return nil, &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, entitytypes.ErrNotFound):
		return h.notFound(ctx), nil
	case errors.As(err, &breaking):
		fields := make([]problem.FieldError, len(breaking.Paths))
		for i, path := range breaking.Paths {
			fields[i] = invalid("/definition"+path, "breaking")
		}
		return nil, &problemError{problem.SchemaBreakingChange, fields, err}
	case errors.As(err, &invalidD):
		fields := make([]problem.FieldError, len(invalidD.Paths))
		for i, path := range invalidD.Paths {
			fields[i] = invalid("/definition"+path, "invalid")
		}
		return nil, unprocessable(fields...)
	case err != nil:
		return nil, err
	}
	h.log.InfoContext(ctx, "entity type version published",
		slog.String("tenant_id", p.TenantID.String()), slog.String("entity_key", v.Key),
		slog.Int("version", v.Version), slog.String("compat", v.Compat))
	return &entityTypeResponse{status: http.StatusCreated, v: v}, nil
}

// parsePublish validates the publish body: only `definition`, and never a
// number_prefix, which does not change after create. The definition itself is
// compiled by the store against the stored key and prefix.
func parsePublish(raw json.RawMessage) (metadata.Definition, *problemError) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return metadata.Definition{}, unprocessable(invalid("", "type"))
	}
	var bad []problem.FieldError
	if _, present := fields["number_prefix"]; present {
		bad = append(bad, invalid("/number_prefix", "immutable"))
	}
	var def struct {
		Schema       json.RawMessage     `json:"schema"`
		FixedColumns map[string]string   `json:"fixed_columns"`
		States       metadata.StateModel `json:"states"`
	}
	defRaw, present := fields["definition"]
	switch {
	case !present:
		bad = append(bad, invalid("/definition", "required"))
	case json.Unmarshal(defRaw, &def) != nil:
		bad = append(bad, invalid("/definition", "type"))
	}
	if len(bad) > 0 {
		return metadata.Definition{}, unprocessable(bad...)
	}
	return metadata.Definition{Schema: def.Schema, FixedColumns: def.FixedColumns, States: def.States}, nil
}

func invalid(pointer, rule string) problem.FieldError {
	return problem.FieldError{Pointer: pointer, Rule: rule}
}

// parseCreate validates the body and compiles the definition. Every failure
// is a 422 invalid_definition naming JSON pointers into the request.
func parseCreate(raw json.RawMessage) (createRequest, *problemError) {
	var cr createRequest
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return cr, unprocessable(invalid("", "type"))
	}
	var bad []problem.FieldError
	str := func(name string) (string, bool) {
		rawv, present := fields[name]
		if !present {
			bad = append(bad, invalid("/"+name, "required"))
			return "", false
		}
		var s string
		if err := json.Unmarshal(rawv, &s); err != nil {
			bad = append(bad, invalid("/"+name, "type"))
			return "", false
		}
		return s, true
	}
	if s, ok := str("key"); ok {
		cr.key = s
		if !keyPattern.MatchString(s) {
			bad = append(bad, invalid("/key", "pattern"))
		}
	}
	if s, ok := str("name"); ok {
		cr.name = s
		if n := utf8.RuneCountInString(s); n < 1 || n > maxNameRunes || !utf8.ValidString(s) {
			bad = append(bad, invalid("/name", "length"))
		}
	}
	if s, ok := str("number_prefix"); ok {
		cr.prefix = s
		if !prefixPattern.MatchString(s) {
			bad = append(bad, invalid("/number_prefix", "pattern"))
		}
	}
	var def struct {
		Schema       json.RawMessage     `json:"schema"`
		FixedColumns map[string]string   `json:"fixed_columns"`
		States       metadata.StateModel `json:"states"`
	}
	defRaw, present := fields["definition"]
	defOK := false
	switch {
	case !present:
		bad = append(bad, invalid("/definition", "required"))
	case json.Unmarshal(defRaw, &def) != nil:
		bad = append(bad, invalid("/definition", "type"))
	default:
		defOK = true
	}
	cr.def = metadata.Definition{Key: cr.key, Schema: def.Schema, NumberPrefix: cr.prefix, FixedColumns: def.FixedColumns, States: def.States}
	// Compile runs whenever the definition decoded, so one response lists the
	// field errors and the definition errors together.
	if defOK {
		if _, err := metadata.Compile(cr.def); err != nil {
			var ie *metadata.ErrInvalidDefinition
			if !errors.As(err, &ie) {
				return cr, &problemError{problem.InternalError, nil, err}
			}
			for _, path := range ie.Paths {
				if path != "/key" && path != "/number_prefix" {
					path = "/definition" + path
				}
				if !slices.ContainsFunc(bad, func(f problem.FieldError) bool { return f.Pointer == path }) {
					bad = append(bad, invalid(path, "invalid"))
				}
			}
		}
	}
	if len(bad) > 0 {
		return cr, unprocessable(bad...)
	}
	return cr, nil
}

func unprocessable(fields ...problem.FieldError) *problemError {
	return &problemError{problem.InvalidDefinition, fields, errors.New("invalid entity definition")}
}

// entityTypeResponse writes an EntityTypeVersion with the schema as the
// stored bytes. The generated response types encode through encoding/json,
// which compacts a json.RawMessage, so the body is assembled here.
type entityTypeResponse struct {
	status int
	v      *entitytypes.Version
}

func (r *entityTypeResponse) write(w http.ResponseWriter) error {
	v := r.v
	var b bytes.Buffer
	b.WriteByte('{')
	put := func(name string, val any) error {
		enc, err := json.Marshal(val)
		if err != nil {
			return err
		}
		if b.Len() > 1 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Quote(name))
		b.WriteByte(':')
		b.Write(enc)
		return nil
	}
	fixed := v.FixedColumns
	if fixed == nil {
		fixed = map[string]string{}
	}
	sum := hex.EncodeToString(v.SchemaSHA256[:])
	for _, f := range []struct {
		name string
		val  any
	}{
		{"key", v.Key}, {"name", v.Name}, {"number_prefix", v.NumberPrefix}, {"version", v.Version}, {"current", v.Current},
		{"schema", v.Schema}, {"schema_sha256", sum}, {"fixed_columns", fixed}, {"states", v.States},
		{"compat", v.Compat}, {"published_at", v.PublishedAt.UTC()},
	} {
		if f.name == "schema" {
			// Verbatim: json.Marshal would compact it.
			if b.Len() > 1 {
				b.WriteByte(',')
			}
			b.WriteString(`"schema":`)
			b.Write(v.Schema)
			continue
		}
		if err := put(f.name, f.val); err != nil {
			return err
		}
	}
	b.WriteByte('}')
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(r.status)
	_, err := b.WriteTo(w)
	return err
}

func (r *entityTypeResponse) VisitGetEntityTypeResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r *entityTypeResponse) VisitGetEntityTypeVersionResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r *entityTypeResponse) VisitPublishEntityTypeVersionResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r *entityTypeResponse) VisitCreateEntityTypeResponse(w http.ResponseWriter) error {
	return r.write(w)
}

var (
	_ api.GetEntityTypeResponseObject            = (*entityTypeResponse)(nil)
	_ api.GetEntityTypeVersionResponseObject     = (*entityTypeResponse)(nil)
	_ api.CreateEntityTypeResponseObject         = (*entityTypeResponse)(nil)
	_ api.PublishEntityTypeVersionResponseObject = (*entityTypeResponse)(nil)
)

// notFoundResponse is the 404 problem without `instance`: the same bytes for
// an unknown resource and another tenant's (D-3-10).
type notFoundResponse struct{ body []byte }

func (h *handlers) notFound(ctx context.Context) *notFoundResponse {
	b := h.reg.Body(ctx, problem.NotFound)
	b.Instance = ""
	out, _ := json.Marshal(b) // strings and ints cannot fail to marshal
	return &notFoundResponse{out}
}

func (r *notFoundResponse) write(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", problem.ContentType)
	w.WriteHeader(http.StatusNotFound)
	_, err := w.Write(slices.Clone(r.body))
	return err
}

func (r *notFoundResponse) VisitPublishEntityTypeVersionResponse(w http.ResponseWriter) error {
	return r.write(w)
}
func (r *notFoundResponse) VisitGetEntityTypeResponse(w http.ResponseWriter) error { return r.write(w) }
func (r *notFoundResponse) VisitGetEntityTypeVersionResponse(w http.ResponseWriter) error {
	return r.write(w)
}

var (
	_ api.GetEntityTypeResponseObject            = (*notFoundResponse)(nil)
	_ api.GetEntityTypeVersionResponseObject     = (*notFoundResponse)(nil)
	_ api.PublishEntityTypeVersionResponseObject = (*notFoundResponse)(nil)
)
