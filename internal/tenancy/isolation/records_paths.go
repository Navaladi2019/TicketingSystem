package isolation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// recordPaths are the record routes (P3-14, TICK-357): the entity-type
// harness (etIso), which runs the real nava-core handler over the suite's
// database as nava_app. Tenant A's admin creates the type iso_et_rec and a
// record of it; tenant B has no such type. A case deletes every row it writes
// (etIso.cleanup) before it returns.
func recordPaths() []Path {
	db := func(name string, fn func(context.Context, *etIso) error) Path {
		return Path{Name: "records: " + name, Case: "CI-8", Role: "nava_app", DB: func(ctx context.Context, cl Cluster) error {
			e, err := newETIso(ctx, cl)
			if err != nil {
				return err
			}
			defer e.stop()
			if err := e.cleanup(); err != nil { // rows a killed earlier run left
				return err
			}
			err = fn(ctx, e)
			if cerr := e.cleanup(); err == nil {
				err = cerr
			}
			return err
		}}
	}
	return []Path{
		db("tenant B reading tenant A's record gets the missing record's 404, byte for byte", recReadOtherTenants),
		db("tenant B creating under tenant A's type gets the unknown type's 404, byte for byte, and writes nothing", recCreateOtherTenantsType),
	}
}

// recCreate makes a record of iso_et_rec in tenant A and returns its id: the
// positive control that the type, the route and the token are live.
func recCreate(ctx context.Context, e *etIso) (string, error) {
	if err := e.create(ctx, "iso_et_rec", "ISC"); err != nil {
		return "", err
	}
	code, body := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_rec", etCallerA, `{"data": {"note": "mine"}}`, nil)
	if code != http.StatusCreated {
		return "", fmt.Errorf("control: tenant A creating a record = %d %s, want 201", code, body)
	}
	var r struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil || r.ID == "" {
		return "", fmt.Errorf("control: tenant A's record has no id: %w: %s", err, body)
	}
	if code, body := e.do(ctx, http.MethodGet, "/api/v1/records/iso_et_rec/"+r.ID, etCallerA, "", nil); code != http.StatusOK {
		return "", fmt.Errorf("control: tenant A reading its own record = %d %s, want 200", code, body)
	}
	return r.ID, nil
}

func recReadOtherTenants(ctx context.Context, e *etIso) error {
	id, err := recCreate(ctx, e)
	if err != nil {
		return err
	}
	const missing = "0194f2c0-0000-7000-8000-000000000000"
	// Tenant B names tenant A's type and record, and a type it does have
	// (incident) with tenant A's record id.
	for _, c := range [][2]string{
		{"/api/v1/records/iso_et_rec/" + id, "/api/v1/records/iso_et_rec/" + missing},
		{"/api/v1/records/incident/" + id, "/api/v1/records/incident/" + missing},
	} {
		oc, ob := e.do(ctx, http.MethodGet, c[0], etCallerB, "", nil)
		uc, ub := e.do(ctx, http.MethodGet, c[1], etCallerB, "", nil)
		switch {
		case uc != http.StatusNotFound:
			return fmt.Errorf("control: tenant B GET %s = %d %s, want 404", c[1], uc, ub)
		case oc != http.StatusNotFound:
			return fmt.Errorf("tenant B GET %s (tenant A's) = %d %s, want 404", c[0], oc, ob)
		case ob != ub:
			return fmt.Errorf("tenant B GET %s (tenant A's) 404 differs from the missing one's:\n%s\n%s", c[0], ob, ub)
		}
	}
	return nil
}

func recCreateOtherTenantsType(ctx context.Context, e *etIso) error {
	if _, err := recCreate(ctx, e); err != nil {
		return err
	}
	body := `{"data": {"note": "forged"}}`
	oc, ob := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_rec", etCallerB, body, nil)
	uc, ub := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_none", etCallerB, body, nil)
	switch {
	case uc != http.StatusNotFound:
		return fmt.Errorf("control: tenant B creating under an unknown type = %d %s, want 404", uc, ub)
	case oc != http.StatusNotFound:
		return fmt.Errorf("tenant B creating under tenant A's type = %d %s, want 404", oc, ob)
	case ob != ub:
		return fmt.Errorf("tenant B's 404 for tenant A's type differs from the unknown type's:\n%s\n%s", ob, ub)
	}
	var n int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
		WHERE d.key = 'iso_et_rec'`).Scan(&n); err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%d records of iso_et_rec exist, want only tenant A's one", n)
	}
	return nil
}
