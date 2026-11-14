package server

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// The generated router never lets a request reach a handler without a
// principal or a configured database; these call the handlers directly, to
// prove each one fails closed on its own if that wiring is ever wrong.

type call struct {
	name string
	fn   func(ctx context.Context, h *handlers) (any, error)
}

func tenantCalls() []call {
	return []call{
		{"ListEntityTypes", func(ctx context.Context, h *handlers) (any, error) {
			return h.ListEntityTypes(ctx, api.ListEntityTypesRequestObject{})
		}},
		{"GetEntityType", func(ctx context.Context, h *handlers) (any, error) {
			return h.GetEntityType(ctx, api.GetEntityTypeRequestObject{Key: "incident"})
		}},
		{"GetEntityTypeVersion", func(ctx context.Context, h *handlers) (any, error) {
			return h.GetEntityTypeVersion(ctx, api.GetEntityTypeVersionRequestObject{Key: "incident", N: 1})
		}},
		{"CreateEntityType", func(ctx context.Context, h *handlers) (any, error) {
			return h.CreateEntityType(ctx, api.CreateEntityTypeRequestObject{})
		}},
		{"PublishEntityTypeVersion", func(ctx context.Context, h *handlers) (any, error) {
			return h.PublishEntityTypeVersion(ctx, api.PublishEntityTypeVersionRequestObject{Key: "incident"})
		}},
		{"PublishWorkflowDefinitionVersion", func(ctx context.Context, h *handlers) (any, error) {
			return h.PublishWorkflowDefinitionVersion(ctx, api.PublishWorkflowDefinitionVersionRequestObject{Key: "triage"})
		}},
		{"GetWorkflowDefinition", func(ctx context.Context, h *handlers) (any, error) {
			return h.GetWorkflowDefinition(ctx, api.GetWorkflowDefinitionRequestObject{Key: "triage"})
		}},
		{"CreateRecord", func(ctx context.Context, h *handlers) (any, error) {
			return h.CreateRecord(ctx, api.CreateRecordRequestObject{Entity: "incident"})
		}},
		{"GetRecord", func(ctx context.Context, h *handlers) (any, error) {
			return h.GetRecord(ctx, api.GetRecordRequestObject{Entity: "incident", Id: uuid.NewString()})
		}},
		{"PatchRecord", func(ctx context.Context, h *handlers) (any, error) {
			return h.PatchRecord(ctx, api.PatchRecordRequestObject{Entity: "incident", Id: uuid.NewString()})
		}},
		{"GetRecordHistory", func(ctx context.Context, h *handlers) (any, error) {
			return h.GetRecordHistory(ctx, api.GetRecordHistoryRequestObject{Entity: "incident", Id: uuid.NewString()})
		}},
	}
}

func principalCtx(t *testing.T) context.Context {
	t.Helper()
	return identity.NewContext(t.Context(), identity.Principal{TenantID: uuid.New(), PrincipalID: uuid.New()})
}

func testHandlers(t *testing.T) *handlers {
	t.Helper()
	reg := mustRegistry(t)
	return &handlers{reg: reg, log: slog.New(slog.DiscardHandler)}
}

func mustRegistry(t *testing.T) *problem.Registry {
	t.Helper()
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestTenantHandlersRefuseWithoutAPrincipal(t *testing.T) {
	t.Parallel()
	for _, c := range tenantCalls() {
		for name, ctx := range map[string]context.Context{
			"no principal":  t.Context(),
			"nil principal": identity.NewContext(t.Context(), identity.Principal{TenantID: uuid.New()}),
		} {
			// The entity-type endpoints need only the tenant; a nil principal id
			// is refused by the record endpoints alone, and by the pool check
			// elsewhere. Both must still fail, never succeed.
			resp, err := c.fn(ctx, testHandlers(t))
			if err == nil {
				t.Errorf("%s (%s): answered %v with a nil error", c.name, name, resp)
				continue
			}
			if name == "no principal" && !errors.Is(err, errNoPrincipal) {
				t.Errorf("%s: err = %v, want errNoPrincipal", c.name, err)
			}
		}
	}
}

func TestTenantHandlersAnswerNotImplementedWithoutADatabase(t *testing.T) {
	t.Parallel()
	for _, c := range tenantCalls() {
		_, err := c.fn(principalCtx(t), testHandlers(t))
		var pe *problemError
		if !errors.As(err, &pe) || pe.code != problem.NotImplemented || !errors.Is(err, errNotConfigured) {
			t.Errorf("%s: err = %v, want the not_implemented problem", c.name, err)
		}
	}
}

func TestRecordHandlersAnswerNotFoundForImpossibleKeysAndIDs(t *testing.T) {
	t.Parallel()
	// A lazy pool: nothing connects until a handler runs a query, and these
	// answer before any query.
	pool, err := pgxpool.New(t.Context(), "postgres://u:pw@127.0.0.1:1/db?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	h := testHandlers(t)
	h.pool, h.pipe, h.cursors, h.audit, h.acl = pool, &pipeline.Pipeline{}, &cursor.Keyring{}, &read.Reader{}, authz.ACL{}
	ctx := principalCtx(t)
	long := strings.Repeat("a", maxKeyBytes+1)

	checks := map[string]func() (any, error){
		"create, key too long": func() (any, error) {
			return h.CreateRecord(ctx, api.CreateRecordRequestObject{Entity: long})
		},
		"create, NUL in key": func() (any, error) {
			return h.CreateRecord(ctx, api.CreateRecordRequestObject{Entity: "a\x00b"})
		},
		"get, key too long": func() (any, error) {
			return h.GetRecord(ctx, api.GetRecordRequestObject{Entity: long, Id: uuid.NewString()})
		},
		"get, bad id": func() (any, error) {
			return h.GetRecord(ctx, api.GetRecordRequestObject{Entity: "incident", Id: "not-a-uuid"})
		},
		"patch, bad id": func() (any, error) {
			return h.PatchRecord(ctx, api.PatchRecordRequestObject{Entity: "incident", Id: "not-a-uuid"})
		},
		"history, key too long": func() (any, error) {
			return h.GetRecordHistory(ctx, api.GetRecordHistoryRequestObject{Entity: long, Id: uuid.NewString()})
		},
		"history, bad id": func() (any, error) {
			return h.GetRecordHistory(ctx, api.GetRecordHistoryRequestObject{Entity: "incident", Id: "x"})
		},
	}
	for name, fn := range checks {
		resp, err := fn()
		if err != nil || resp == nil {
			t.Errorf("%s: resp=%v err=%v, want the 404 problem response", name, resp, err)
		}
	}
}

func TestStaleErrorNamesTheCurrentVersion(t *testing.T) {
	t.Parallel()
	if got := (&staleError{current: 7}).Error(); got != "record is at version 7" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestOnboardTenantRefusesWithoutOperatorContextOrBody(t *testing.T) {
	t.Parallel()
	h := testHandlers(t)
	if _, err := h.OnboardTenant(t.Context(), api.OnboardTenantRequestObject{}); !errors.Is(err, errNoPrincipal) {
		t.Fatalf("no operator context: err = %v", err)
	}
	h.onboarder = stubOnboarder{}
	ctx := context.WithValue(t.Context(), operatorCtxKey{}, operatorCall{})
	_, err := h.OnboardTenant(ctx, api.OnboardTenantRequestObject{})
	var pe *problemError
	if !errors.As(err, &pe) || pe.code != problem.ValidationFailed || len(pe.fields) != 1 || pe.fields[0].Pointer != "/body" {
		t.Fatalf("no body: err = %v, want validation_failed on /body", err)
	}
}

type stubOnboarder struct{ Onboarder }

func TestGetMeRefusesWithoutAPrincipal(t *testing.T) {
	t.Parallel()
	if _, err := testHandlers(t).GetMe(t.Context(), api.GetMeRequestObject{}); !errors.Is(err, errNoPrincipal) {
		t.Fatalf("err = %v, want errNoPrincipal", err)
	}
}

func TestHistoryRefusesCursorsThatDoNotDecode(t *testing.T) {
	t.Parallel()
	ring, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: []byte("0123456789abcdef0123456789abcdef")})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(t.Context(), "postgres://u:pw@127.0.0.1:1/db?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	h := testHandlers(t)
	h.pool, h.cursors, h.audit, h.acl = pool, ring, &read.Reader{}, authz.ACL{}
	ctx := principalCtx(t)
	id := uuid.New()

	// A cursor signed for this very request, but whose position is garbage.
	claims, err := cursor.ClaimsFor(ctx, getRecordHistoryRoute, url.Values{"entity": {"incident"}, "id": {id.String()}})
	if err != nil {
		t.Fatal(err)
	}
	signedGarbage, err := ring.Encode(claims, []byte("not-a-position"))
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]string{"forged": "AAAA.BBBB", "signed, bad position": signedGarbage} {
		_, err := h.GetRecordHistory(ctx, api.GetRecordHistoryRequestObject{
			Entity: "incident", Id: id.String(), Params: api.GetRecordHistoryParams{Cursor: &c},
		})
		var pe *problemError
		if !errors.As(err, &pe) || pe.code != problem.InvalidCursor {
			t.Errorf("%s: err = %v, want invalid_cursor", name, err)
		}
	}
}

func TestNewHandlerRefusesMissingDependencies(t *testing.T) {
	t.Parallel()
	tel := newTelemetryForTest(t)
	if _, err := NewHandler(slog.New(slog.DiscardHandler), tel, nil, nilPaths{}); err == nil {
		t.Fatal("a nil resolver must be refused")
	}
}

type nilPaths struct{ identity.GroupPathReader }

func newTelemetryForTest(t *testing.T) *telemetry.Telemetry {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	return tel
}

func TestCursorRoutesRefuseAPrincipalWithoutATenant(t *testing.T) {
	t.Parallel()
	ring, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: []byte("0123456789abcdef0123456789abcdef")})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(t.Context(), "postgres://u:pw@127.0.0.1:1/db?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	h := testHandlers(t)
	h.pool, h.cursors, h.audit, h.acl = pool, ring, &read.Reader{}, authz.ACL{}
	// No tenant: the cursor claims cannot be built, so no query may run.
	ctx := identity.NewContext(t.Context(), identity.Principal{PrincipalID: uuid.New()})

	if _, err := h.ListEntityTypes(ctx, api.ListEntityTypesRequestObject{}); !errors.Is(err, cursor.ErrNoPrincipal) {
		t.Errorf("ListEntityTypes: err = %v, want cursor.ErrNoPrincipal", err)
	}
	if _, err := h.GetRecordHistory(ctx, api.GetRecordHistoryRequestObject{Entity: "incident", Id: uuid.NewString()}); !errors.Is(err, cursor.ErrNoPrincipal) {
		t.Errorf("GetRecordHistory: err = %v, want cursor.ErrNoPrincipal", err)
	}
}
