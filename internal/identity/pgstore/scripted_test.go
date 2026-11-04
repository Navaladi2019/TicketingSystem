package pgstore_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
)

//nolint:gochecknoglobals // read-only test fixtures
var (
	tenantX = uuid.MustParse("00000000-0000-4000-8000-0000000000a1")
	groupG  = uuid.MustParse("00000000-0000-4000-8000-0000000000c3")
)

func bindingRow(tenant string) []*string { return []*string{s(tenant), s("{idp-a}")} }

func bindingHandler(bindingRows [][]*string, stateRows [][]*string, stateErr string) func(string) result {
	return func(sql string) result {
		switch {
		case strings.Contains(sql, "FROM tenant_identity_binding"):
			return result{cols: []uint32{oidUUID, oidTexts}, rows: bindingRows}
		case strings.Contains(sql, "FROM tenants"):
			return result{cols: []uint32{oidText}, rows: stateRows, err: stateErr}
		}
		return result{err: "unexpected statement: " + sql}
	}
}

func TestBindingDefensiveBranches(t *testing.T) {
	t.Parallel()
	one := [][]*string{bindingRow(tenantX.String())}
	active := [][]*string{{s("active")}}
	cases := []struct {
		name    string
		h       func(string) result
		wantErr string // substring; empty means success
		noBind  bool
	}{
		{"no binding", bindingHandler(nil, nil, ""), "", true},
		{"one binding", bindingHandler(one, active, ""), "", false},
		{"two bindings for one key are an error, not a choice", bindingHandler(append(one, bindingRow(uuid.NewString())), active, ""), "2 bindings", false},
		{"a binding with a nil tenant", bindingHandler([][]*string{bindingRow(uuid.Nil.String())}, active, ""), "nil tenant", false},
		{"the tenant row is gone", bindingHandler(one, nil, ""), "tenant row not found", false},
		{"the tenant state read fails", bindingHandler(one, nil, "boom"), "tenant state", false},
		{"an unreadable binding row", bindingHandler([][]*string{bindingRow("not-a-uuid")}, active, ""), "pgstore: binding", false},
		{"the binding read fails", func(string) result { return result{err: "boom"} }, "pgstore: binding", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			store := pgstore.New(poolOn(t, startFakePG(t, c.h)))
			b, err := store.Binding(t.Context(), "https://kc/realms/nava", "org-1")
			switch {
			case c.noBind:
				if !errors.Is(err, identity.ErrNoBinding) {
					t.Fatalf("err = %v, want ErrNoBinding", err)
				}
			case c.wantErr != "":
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, c.wantErr)
				}
				if errors.Is(err, identity.ErrNoBinding) || b.TenantID != uuid.Nil {
					t.Fatalf("a failure must not look like a binding: %+v, %v", b, err)
				}
			default:
				if err != nil || b.TenantID != tenantX || b.TenantState != "active" || len(b.IdPAliases) != 1 {
					t.Fatalf("binding = %+v, %v", b, err)
				}
			}
		})
	}
}

func principalHandler(row []*string, errMsg string) func(string) result {
	return func(sql string) result {
		if strings.Contains(sql, "FROM principals") {
			if errMsg != "" {
				return result{err: errMsg}
			}
			var rows [][]*string
			if row != nil {
				rows = [][]*string{row}
			}
			return result{cols: []uint32{oidUUID, oidBool, oidJSONB, oidUUIDs}, rows: rows}
		}
		return result{err: "unexpected statement: " + sql}
	}
}

func TestPrincipalDefensiveBranches(t *testing.T) {
	t.Parallel()
	id := uuid.NewString()
	run := func(t *testing.T, h func(string) result) (identity.PrincipalRecord, error) {
		t.Helper()
		return pgstore.New(poolOn(t, startFakePG(t, h))).Principal(t.Context(), tenantX, "kc-user")
	}

	t.Run("no row", func(t *testing.T) {
		t.Parallel()
		if _, err := run(t, principalHandler(nil, "")); !errors.Is(err, identity.ErrNoPrincipal) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("links that are not an array of links", func(t *testing.T) {
		t.Parallel()
		_, err := run(t, principalHandler([]*string{s(id), s("t"), s("[1]"), s("{}")}, ""))
		if err == nil || !strings.Contains(err.Error(), "federated_links") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("NULL links stay unknown and the groups are an empty list", func(t *testing.T) {
		t.Parallel()
		rec, err := run(t, principalHandler([]*string{s(id), s("t"), nil, s("{}")}, ""))
		if err != nil || rec.LinksKnown || rec.FederatedIdPs != nil || rec.Groups == nil || len(rec.Groups) != 0 || !rec.Enabled {
			t.Fatalf("rec = %+v, %v", rec, err)
		}
	})
	t.Run("links and groups are read", func(t *testing.T) {
		t.Parallel()
		g := groupG.String()
		rec, err := run(t, principalHandler([]*string{s(id), s("f"), s(`[{"identityProvider":"idp-a"},{"identityProvider":"idp-b"}]`), s("{" + g + "}")}, ""))
		if err != nil || !rec.LinksKnown || rec.Enabled || len(rec.FederatedIdPs) != 2 || rec.FederatedIdPs[1] != "idp-b" ||
			len(rec.Groups) != 1 || rec.Groups[0] != groupG {
			t.Fatalf("rec = %+v, %v", rec, err)
		}
	})
	t.Run("a database error is not absence", func(t *testing.T) {
		t.Parallel()
		if _, err := run(t, principalHandler(nil, "boom")); err == nil || errors.Is(err, identity.ErrNoPrincipal) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestTenantOwnsEmailDomainLowercasesAndWraps(t *testing.T) {
	t.Parallel()
	f := startFakePG(t, func(sql string) result {
		if strings.Contains(sql, "tenant_email_domain") {
			return result{cols: []uint32{oidBool}, rows: [][]*string{{s("t")}}}
		}
		return result{err: "unexpected statement"}
	})
	store := pgstore.New(poolOn(t, f))
	owns, err := store.TenantOwnsEmailDomain(t.Context(), tenantX, "Example.TEST")
	if err != nil || !owns {
		t.Fatalf("owns = %v, %v", owns, err)
	}
	var queried bool
	for _, q := range f.statements() {
		if strings.Contains(q, "tenant_email_domain") {
			queried = true
			if strings.Contains(q, "Example") || !strings.Contains(q, "example.test") {
				t.Errorf("the domain was not lower-cased before the query: %s", q)
			}
		}
	}
	if !queried {
		t.Fatal("no domain query was sent")
	}
}

func TestGroupPathsDefensiveBranches(t *testing.T) {
	t.Parallel()
	pid := uuid.New()
	run := func(t *testing.T, h func(string) result) ([]identity.GroupPath, error) {
		t.Helper()
		return pgstore.New(poolOn(t, startFakePG(t, h))).GroupPaths(t.Context(), tenantX, pid)
	}
	rows := func(rs ...[]*string) func(string) result {
		return func(sql string) result {
			if strings.Contains(sql, "FROM group_members") {
				return result{cols: []uint32{oidUUID, oidText}, rows: rs}
			}
			return result{err: "unexpected statement"}
		}
	}
	got, err := run(t, rows([]*string{s(groupG.String()), s("/t/x/devs")}))
	if err != nil || len(got) != 1 || got[0].ID != groupG || got[0].Path != "/t/x/devs" {
		t.Fatalf("paths = %+v, %v", got, err)
	}
	if got, err := run(t, rows()); err != nil || len(got) != 0 {
		t.Fatalf("no groups = %+v, %v", got, err)
	}
	if got, err := run(t, rows([]*string{s("not-a-uuid"), s("/t/x/devs")})); err == nil || got != nil {
		t.Fatalf("an unreadable row must fail the call: %+v, %v", got, err)
	}
	if got, err := run(t, func(string) result { return result{err: "boom"} }); err == nil || got != nil {
		t.Fatalf("a failed query must fail the call: %+v, %v", got, err)
	}
}
